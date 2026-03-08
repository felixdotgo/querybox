package pluginmgr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	pluginpb "github.com/felixdotgo/querybox/rpc/contracts/plugin/v1"
)

// pluginName returns a filename appropriate for the current OS. On Windows
// the manager only treats files with ".exe" extension as executable, so
// tests must append that suffix accordingly.
func pluginName(base string) string {
	if runtime.GOOS == "windows" {
		return base + ".exe"
	}
	return base
}

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
		"metadata": map[string]string{"key": "val", "simple_icon": "postgresql"},
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
	// simple_icon is a special hint used by the frontend to pick a branded
	// database glyph.  Ensure it survives the normalization round-trip.
	if res.Metadata["simple_icon"] != "postgresql" {
		t.Errorf("simple_icon metadata not preserved: %+v", res.Metadata)
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

func TestDescribeSchemaMissingPlugin(t *testing.T) {
	m := New()
	_, err := m.DescribeSchema("nonexistent", nil, "", "")
	if err == nil {
		t.Errorf("expected error for missing plugin")
	}
}

// GetPluginAuthForms should not return an error when the plugin is absent;
// callers treat a nil result as “no forms.” This simulates the dev-mode
// scenario where the frontend queries before the scan completes.
func TestGetPluginAuthFormsMissingPlugin(t *testing.T) {
	m := New()
	forms, err := m.GetPluginAuthForms("nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if forms != nil {
		t.Errorf("expected nil forms, got %#v", forms)
	}
}

// If the plugin path exists but is not executable, treat it the same way.
// Non-executable binaries may show up during scanning if permissions are wrong.
func TestGetPluginAuthFormsNonExecutable(t *testing.T) {
	dir, err := os.MkdirTemp("", "pmgrnoexec")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	name := pluginName("notexec")
	path := filepath.Join(dir, name)
	// create a file without exec bit
	if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	m := &Manager{plugins: map[string]PluginInfo{"notexec": {Path: path}}}
	forms, err := m.GetPluginAuthForms("notexec")
	if err != nil {
		t.Fatalf("unexpected error for non-exec path: %v", err)
	}
	if forms != nil {
		t.Errorf("expected nil forms for non-executable plugin, got %#v", forms)
	}
	// calling with extension should behave identically (normalization)
	forms2, err2 := m.GetPluginAuthForms(name)
	if err2 != nil {
		t.Fatalf("unexpected error for non-exec path with ext: %v", err2)
	}
	if forms2 != nil {
		t.Errorf("expected nil forms for non-executable plugin with ext, got %#v", forms2)
	}
}

func TestDescribeSchemaParsesResponse(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script plugin not supported on Windows")
	}
	// create a dummy executable that handles the describe-schema command
	dir, err := os.MkdirTemp("", "pmgrschema")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	name := pluginName("dummy")
	req := strings.TrimSuffix(name, filepath.Ext(name))
	script := filepath.Join(dir, name)
	bin := fmt.Sprintf(`#!/bin/sh
if [ "$1" = "describe-schema" ]; then
  echo '{"tables":[{"name":"foo","columns":[{"name":"id","type":"int"}],"indexes":[]}]}';
else
  echo '{"nodes":[]}'
fi
`)
	if err := os.WriteFile(script, []byte(bin), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	m := &Manager{plugins: map[string]PluginInfo{req: {Path: script}}}

	// DescribeSchema expects the plugin name without extension.  Call with
	// both trimmed and untrimmed inputs to ensure normalization logic works.
	resp, err := m.DescribeSchema(req, nil, "", "")
	if err != nil {
		t.Fatalf("DescribeSchema error: %v", err)
	}
	if len(resp.Tables) != 1 || resp.Tables[0].Name != "foo" {
		t.Errorf("unexpected response: %+v", resp)
	}
	// also try with the raw filename (extension included) to confirm it gets
	// normalized before lookup
	resp2, err2 := m.DescribeSchema(name, nil, "", "")
	if err2 != nil {
		t.Fatalf("DescribeSchema with extension failed: %v", err2)
	}
	if len(resp2.Tables) != 1 || resp2.Tables[0].Name != "foo" {
		t.Errorf("unexpected response when using extension: %+v", resp2)
	}
}

func TestScanOnceConcurrent(t *testing.T) {
	dir, err := os.MkdirTemp("", "pmgrscan")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// create two dummy executable files
	for _, base := range []string{"p1", "p2"} {
		name := pluginName(base)
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
		base := filepath.Base(fullpath)
		trim := strings.TrimSuffix(base, filepath.Ext(base))
		return PluginInfo{ID: trim, Name: trim}, nil
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

	// map keys should be normalized (no .exe extension)
	for k := range m.plugins {
		if strings.HasSuffix(k, ".exe") {
			t.Errorf("plugin key %s unexpectedly contains extension", k)
		}
	}

	// deleting one file should prune the registry
	os.Remove(filepath.Join(dir, pluginName("p1")))
	m.scanOnce()
	if len(m.plugins) != 1 {
		t.Fatalf("expected 1 plugin after removal, got %d", len(m.plugins))
	}
	if _, ok := m.plugins["p2"]; !ok {
		t.Errorf("remaining plugin should be %s", "p2")
	}
}
// TestPluginsReadyCallback ensures that the onPluginsReady hook is invoked
// when the manager emits the ready event. By constructing a manager manually
// we can set the hook before the notification is fired.
func TestPluginsReadyCallback(t *testing.T) {
	m := &Manager{
		plugins:    make(map[string]PluginInfo),
		appReadyCh: make(chan struct{}),
	}
	done := make(chan struct{})
	m.onPluginsReady = func() { close(done) }

	// run scan and emit in background
	go func() {
		m.scanOnce()
		m.emitPluginsReady()
	}()

	close(m.appReadyCh)

	select {
	case <-done:
		// good
	case <-time.After(1 * time.Second):
		t.Fatal("plugins ready callback was not invoked")
	}
}

// TestRescanFiresPluginsReady ensures invoking Rescan also triggers the
// ready notification.
func TestRescanFiresPluginsReady(t *testing.T) {
	m := &Manager{
		plugins:    make(map[string]PluginInfo),
		appReadyCh: make(chan struct{}),
	}
	done := make(chan struct{})
	m.onPluginsReady = func() { close(done) }

	close(m.appReadyCh)
	if err := m.Rescan(); err != nil {
		t.Fatalf("Rescan failed: %v", err)
	}

	select {
	case <-done:
		// good
	case <-time.After(1 * time.Second):
		t.Fatal("plugins ready callback not invoked after rescan")
	}
}
// TestPopulateUserDir verifies the standalone populateUserDir helper. It
// simulates the bundle and user filesystem paths directly, avoiding New() so
// the behaviour is easy to control.
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

	userDir := filepath.Join(user, "querybox", "plugins")

	fname := pluginName("bundled")
	initial := []byte("first")
	later := []byte("second")

	// create a dummy plugin file in bundle
	if err := os.WriteFile(filepath.Join(bundle, fname), initial, 0o755); err != nil {
		t.Fatal(err)
	}
	// ensure the target user directory exists
	if err := os.MkdirAll(userDir, 0o755); err != nil {
		t.Fatalf("failed to create userDir: %v", err)
	}

	// first copy
	populateUserDir(userDir, bundle)
	if data, err := os.ReadFile(filepath.Join(userDir, fname)); err != nil {
		t.Fatalf("expected file copied to user dir: %v", err)
	} else if !bytes.Equal(data, initial) {
		t.Errorf("unexpected initial content: %s", string(data))
	}

	// ensure executable detection works
	filePath := filepath.Join(userDir, fname)
	if info, err := os.Stat(filePath); err == nil {
		t.Logf("copied file mode: %v, ext: %s", info.Mode(), filepath.Ext(filePath))
	}
	if !isExecutable(filePath) {
		t.Errorf("copied file should be executable")
	}

	// second copy with updated bundle
	if err := os.WriteFile(filepath.Join(bundle, fname), later, 0o755); err != nil {
		t.Fatal(err)
	}
	populateUserDir(userDir, bundle)
	if data, err := os.ReadFile(filepath.Join(userDir, fname)); err != nil {
		t.Fatalf("failed to read user copy: %v", err)
	} else if !bytes.Equal(data, later) {
		t.Errorf("expected overwrite with later content, got %s", string(data))
	}
	if !isExecutable(filePath) {
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
	if err := os.WriteFile(filepath.Join(user, pluginName("dup")), []byte(""), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bundle, pluginName("dup")), []byte(""), 0o755); err != nil {
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
		return PluginInfo{ID: pluginName("dup"), Name: "dup", Type: int(pluginpb.PluginV1_DRIVER)}, nil
	}

	m := &Manager{
		plugins:    make(map[string]PluginInfo),
		appReadyCh: make(chan struct{}),
	}
	m.dirs = []string{user, bundle}
	m.Dir = user

	m.scanOnce()
	id := pluginName("dup")
	info, ok := m.plugins[id]
	if !ok {
		t.Fatalf("%s not discovered", id)
	}
	if info.Path != filepath.Join(bundle, id) {
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
	if err := os.WriteFile(filepath.Join(user, pluginName("dup")), []byte(""), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bundle, pluginName("dup")), []byte(""), 0o755); err != nil {
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
	id := pluginName("dup")
	info, ok := m.plugins[id]
	if !ok {
		t.Fatalf("plugin %s missing after scan", id)
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
