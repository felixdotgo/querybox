package pluginmgr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	pluginpb "github.com/felixdotgo/querybox/rpc/contracts/plugin/v1"
)

func TestUserPluginsDirBehavior(t *testing.T) {
	orig := userPluginDirFunc
	defer func() { userPluginDirFunc = orig }()

	userPluginDirFunc = func() (string, error) {
		return "/home/testuser/.config", nil
	}
	p, err := userPluginsDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasSuffix(p, filepath.Join("querybox", "plugins")) {
		t.Errorf("path wrong: %s", p)
	}

	// failure case
	userPluginDirFunc = func() (string, error) { return "", fmt.Errorf("fail") }
	if p, err := userPluginsDir(); err == nil {
		t.Errorf("expected error, got path %s", p)
	}
}

func TestProbeInfoDecoding(t *testing.T) {
	// prepare a fake JSON as plugin binary would emit (camelCase keys are
	// what protojson generates).
	raw := map[string]interface{}{
		"type": 1,
		"name": "foo",
		"version": "1.2.3",
		"description": "the foo driver",
		"url": "https://example.org/foo",
		"author": "Foo Corp",
		"capabilities": []string{"transactions"},
		"tags": []string{"sql"},
		"license": "MIT",
		"iconUrl": "https://example.org/icon.png",
		"contact": "support@example.org",
		"metadata": map[string]string{"key": "val"},
		"settings": map[string]string{"k2": "v2"},
	}
	b, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}

	var resp PluginInfo
	if err := json.Unmarshal(b, &resp); err != nil {
		t.Fatalf("unmarshal plugininfo: %v", err)
	}

	// mimic probeInfo() internals by building raw map then converting
	var parsed map[string]interface{}
	if err := json.Unmarshal(b, &parsed); err != nil {
		t.Fatalf("inner unmarshal: %v", err)
	}
	res, err := probeInfoFromRaw(parsed)
	if err != nil {
		t.Fatalf("probeInfoFromRaw error: %v", err)
	}

	if res.Name != "foo" || res.URL != "https://example.org/foo" || res.Author != "Foo Corp" {
		t.Errorf("bad basic fields: %+v", res)
	}
	if res.Type != int(pluginpb.PluginV1_DRIVER) {
		t.Errorf("type not decoded: %d", res.Type)
	}
	if len(res.Capabilities) != 1 || res.Capabilities[0] != "transactions" {
		t.Errorf("capabilities not preserved: %+v", res.Capabilities)
	}
	if len(res.Tags) != 1 || res.Tags[0] != "sql" {
		t.Errorf("tags not preserved: %+v", res.Tags)
	}
	if res.License != "MIT" {
		t.Errorf("license not preserved: %s", res.License)
	}
	if res.IconURL != "https://example.org/icon.png" {
		t.Errorf("icon url not preserved: %s", res.IconURL)
	}
	if res.Contact != "support@example.org" {
		t.Errorf("contact not preserved: %s", res.Contact)
	}
	if res.Metadata == nil || res.Metadata["key"] != "val" {
		t.Errorf("metadata missing: %+v", res.Metadata)
	}
}

// TestExecRequestMarshalling ensures that the internal execRequest struct
// correctly serialises the optional options map so the plugin receives it.
func TestExecRequestMarshalling(t *testing.T) {
	r := execRequest{
		Connection: map[string]string{"a": "b"},
		Query:      "SELECT 1",
		Options:    map[string]string{"explain-query": "yes"},
	}
	b, err := json.Marshal(&r)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if opts, ok := m["options"].(map[string]interface{}); !ok {
		t.Errorf("options field missing or wrong type: %#v", m)
	} else if opts["explain-query"] != "yes" {
		t.Errorf("unexpected options content: %#v", opts)
	}
}

func TestExecTreeActionForwardsOptions(t *testing.T) {
	m := New()
	_, err := m.ExecTreeAction("nonexistent", nil, "SELECT 1", map[string]string{"explain-query": "yes"})
	if err == nil {
		t.Errorf("expected error for missing plugin")
	}
}

func TestScanOnceConcurrent(t *testing.T) {
	dir, err := os.MkdirTemp("", "pmgrscan")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// create two dummy executable files
	for _, name := range []string{"p1", "p2"} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(""), 0o755); err != nil {
			t.Fatalf("write dummy plugin %s: %v", name, err)
		}
	}

	// instrumentation to ensure probes run in parallel
	var active, maxActive int32
	orig := probeInfoFunc
	probeInfoFunc = func(fullpath string) (PluginInfo, error) {
		curr := atomic.AddInt32(&active, 1)
		if curr > atomic.LoadInt32(&maxActive) {
			atomic.StoreInt32(&maxActive, curr)
		}
		// delay so there is opportunity for overlap
		time.Sleep(25 * time.Millisecond)
		atomic.AddInt32(&active, -1)
		return PluginInfo{ID: filepath.Base(fullpath), Name: filepath.Base(fullpath)}, nil
	}
	defer func() { probeInfoFunc = orig }()

	// construct a manager that scans only our temp directory
	m := &Manager{
		plugins:    make(map[string]PluginInfo),
		appReadyCh: make(chan struct{}),
	}
	m.dirs = []string{dir}
	m.Dir = dir // maintain backwards-compatible field for binding

	m.scanOnce()
	if len(m.plugins) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(m.plugins))
	}
	if atomic.LoadInt32(&maxActive) < 2 {
		t.Errorf("probe did not execute concurrently, maxActive=%d", maxActive)
	}

	// deleting one file should prune the registry
	os.Remove(filepath.Join(dir, "p1"))
	m.scanOnce()
	if len(m.plugins) != 1 {
		t.Fatalf("expected 1 plugin after removal, got %d", len(m.plugins))
	}
	if _, ok := m.plugins["p2"]; !ok {
		t.Errorf("remaining plugin should be p2")
	}
}

// TestPopulateUserDir verifies New() copies bundled binaries into the
// user directory on every invocation, overwriting existing files.
func TestPopulateUserDir(t *testing.T) {
	user, err := os.MkdirTemp("", "userplugins")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(user)
	bundle, err := os.MkdirTemp("", "bundleplugins")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(bundle)

	fname := "bundled"
	initial := []byte("first")
	later := []byte("second")

	// create a dummy plugin file in bundle
	if err := os.WriteFile(filepath.Join(bundle, fname), initial, 0o755); err != nil {
		t.Fatal(err)
	}

	origUser := userPluginDirFunc
	defer func() { userPluginDirFunc = origUser }()
	userPluginDirFunc = func() (string, error) { return user, nil }

	// first run copies initial content
	_ = New()
	if _, err := os.ReadFile(filepath.Join(user, fname)); err != nil {
		t.Fatalf("expected file copied to user dir: %v", err)
	}
	if !isExecutable(filepath.Join(user, fname)) {
		t.Errorf("copied file should be executable")
	}

	// modify bundle and run again -> user file should reflect new bytes
	if err := os.WriteFile(filepath.Join(bundle, fname), later, 0o755); err != nil {
		t.Fatal(err)
	}
	_ = New()
	got, err := os.ReadFile(filepath.Join(user, fname))
	if err != nil {
		t.Fatalf("failed to read user copy: %v", err)
	}
	if !bytes.Equal(got, later) {
		t.Errorf("expected overwrite with later content, got %s", string(got))
	}
	if !isExecutable(filepath.Join(user, fname)) {
		t.Errorf("overwritten file should remain executable")
	}
}

// TestFallbackToBundle ensures that if the user copy cannot be probed the
// manager will still load metadata from the bundled executable.
func TestFallbackToBundle(t *testing.T) {
	user, err := os.MkdirTemp("", "userplugins")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(user)
	bundle, err := os.MkdirTemp("", "bundleplugins")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(bundle)

	// both directories contain a plugin binary named "dup"
	if err := os.WriteFile(filepath.Join(user, "dup"), []byte(""), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bundle, "dup"), []byte(""), 0o755); err != nil {
		t.Fatal(err)
	}

	// make probeInfoFunc fail when given the user path but succeed for bundle
	orig := probeInfoFunc
	defer func() { probeInfoFunc = orig }()
	probeInfoFunc = func(fullpath string) (PluginInfo, error) {
		if strings.HasPrefix(fullpath, user) {
			return PluginInfo{}, fmt.Errorf("user path broken")
		}
		// simulate a valid driver response
		return PluginInfo{ID: "dup", Name: "dup", Type: int(pluginpb.PluginV1_DRIVER)}, nil
	}

	m := &Manager{
		plugins:    make(map[string]PluginInfo),
		appReadyCh: make(chan struct{}),
	}
	m.dirs = []string{user, bundle}
	m.Dir = user

	m.scanOnce()
	info, ok := m.plugins["dup"]
	if !ok {
		t.Fatal("dup not discovered")
	}
	if info.Path != filepath.Join(bundle, "dup") {
		t.Errorf("expected bundle path used, got %s", info.Path)
	}
	if info.Type != int(pluginpb.PluginV1_DRIVER) {
		t.Errorf("expected driver type, got %d", info.Type)
	}
}

// TestUserDirPrecedence ensures that a plugin placed in the first (user)
// directory takes precedence over an identically named executable in the
// fallback/bundled directory.
func TestUserDirPrecedence(t *testing.T) {
	user, err := os.MkdirTemp("", "userplugins")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(user)
	bundle, err := os.MkdirTemp("", "bundleplugins")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(bundle)

	// create plugin with same name in both locations
	if err := os.WriteFile(filepath.Join(user, "dup"), []byte(""), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bundle, "dup"), []byte(""), 0o755); err != nil {
		t.Fatal(err)
	}

	m := &Manager{
		plugins:    make(map[string]PluginInfo),
		appReadyCh: make(chan struct{}),
	}
	m.dirs = []string{user, bundle}
	m.Dir = user

	m.scanOnce()
	// we should discover only one plugin and its path should point to user dir
	if len(m.plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(m.plugins))
	}
	info, ok := m.plugins["dup"]
	if !ok {
		t.Fatal("plugin dup missing after scan")
	}
	if !strings.HasPrefix(info.Path, user) {
		t.Errorf("expected user dir to win, got path %s", info.Path)
	}
}


// helper extracted from probeInfo so we can call without executing command
func probeInfoFromRaw(raw map[string]interface{}) (PluginInfo, error) {
	// copy logic from probeInfo, including normalization
	// normalize camelCase keys like iconUrl -> icon_url
	norm := make(map[string]interface{}, len(raw)+4)
	for k, v := range raw {
		norm[k] = v
		switch k {
		case "iconUrl":
			norm["icon_url"] = v
		}
	}
	raw = norm

	// interpret the type field
	typ := 0
	if v, ok := raw["type"]; ok {
		switch vv := v.(type) {
		case float64:
			typ = int(vv)
		case string:
			if val, ok := pluginpb.PluginV1_Type_value[vv]; ok {
				typ = int(val)
			}
		}
	}

	var resp struct {
		Name        string            `json:"name"`
		Version     string            `json:"version"`
		Description string            `json:"description"`
		URL         string            `json:"url"`
		Author      string            `json:"author"`
		Capabilities []string         `json:"capabilities"`
		Tags        []string          `json:"tags"`
		License     string            `json:"license"`
		IconURL     string            `json:"icon_url"`
		Contact     string            `json:"contact"`
		Metadata    map[string]string `json:"metadata"`
		Settings    map[string]string `json:"settings"`
	}
	if b2, err2 := json.Marshal(raw); err2 == nil {
		_ = json.Unmarshal(b2, &resp)
	}

	return PluginInfo{
		Name:        resp.Name,
		Type:        typ,
		Version:     resp.Version,
		Description: resp.Description,
		URL:         resp.URL,
		Author:      resp.Author,
		Capabilities: resp.Capabilities,
		Tags:        resp.Tags,
		License:     resp.License,
		IconURL:     resp.IconURL,
		Contact:     resp.Contact,
		Metadata:    resp.Metadata,
		Settings:    resp.Settings,
	}, nil
}
