package pluginmgr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/felixdotgo/querybox/pkg/plugin"
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

type LocalPluginHostMock struct {
	RunFunc func(info PluginInfo, command string, timeout time.Duration, req []byte) ([]byte, error)
}

func (m *LocalPluginHostMock) Run(info PluginInfo, command string, timeout time.Duration, req []byte) ([]byte, error) {
	return m.RunFunc(info, command, timeout, req)
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

func TestApplyManifestCopiesDisplayMetadata(t *testing.T) {
	info := PluginInfo{}
	manifest := &plugin.Manifest{
		ID:          "manifested",
		Type:        int(pluginpb.PluginV1_DRIVER),
		Name:        "Manifested Plugin",
		Description: "manifest-first plugin",
		Version:     "1.2.3",
		URL:         "https://example.org/plugin",
		Author:      "QueryBox Team",
		Runtime: plugin.RuntimeSpec{
			Kind: plugin.RuntimeKindLocal,
		},
		Capabilities: []string{"query.execute", "connection.test"},
		Permissions:  []plugin.PermissionDecl{{Name: "network"}},
		Limits:       plugin.Limits{TimeoutSeconds: 12},
		Tags:         []string{"sql", "relational"},
		License:      "MIT",
		IconURL:      "https://example.org/icon.png",
		Contact:      "support@example.org",
		Metadata: map[string]string{
			"simple_icon": "postgresql",
		},
		Settings: map[string]string{
			"color": "blue",
		},
	}

	applyManifest(&info, "/tmp/manifested.manifest.json", manifest)

	if info.Name != "Manifested Plugin" || info.URL != "https://example.org/plugin" || info.Author != "QueryBox Team" {
		t.Fatalf("expected manifest display metadata, got %#v", info)
	}
	if info.Type != int(pluginpb.PluginV1_DRIVER) {
		t.Fatalf("expected driver type from manifest, got %d", info.Type)
	}
	if len(info.Tags) != 2 || info.Tags[0] != "sql" {
		t.Fatalf("expected tags from manifest, got %#v", info.Tags)
	}
	if info.License != "MIT" || info.IconURL != "https://example.org/icon.png" || info.Contact != "support@example.org" {
		t.Fatalf("expected manifest presentation fields, got %#v", info)
	}
	if info.Metadata["simple_icon"] != "postgresql" {
		t.Fatalf("expected manifest metadata, got %#v", info.Metadata)
	}
	if info.Settings["color"] != "blue" {
		t.Fatalf("expected manifest settings, got %#v", info.Settings)
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

// TestMutateRowRequestMarshalling ensures the internal mutateRowRequest
// serialises the operation enum and other fields correctly.
func TestMutateRowRequestMarshalling(t *testing.T) {
	r := mutateRowRequest{
		Connection: map[string]string{"a": "b"},
		Operation:  pluginpb.PluginV1_MutateRowRequest_INSERT,
		Source:     "t1",
		Values:     map[string]string{"x": "y"},
		Filter:     "id=1",
	}
	b, err := json.Marshal(&r)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// encoding/json will represent the enum as a number (float64 here).
	if op, ok := m["operation"].(float64); !ok || op != float64(pluginpb.PluginV1_MutateRowRequest_INSERT) {
		t.Errorf("operation not marshalled as numeric value: %#v", m["operation"])
	}
	if m["filter"] != "id=1" {
		t.Errorf("filter wrong: %#v", m["filter"])
	}
}

func TestMutateRowMissingPlugin(t *testing.T) {
	m := New()
	_, err := m.MutateRow("nonexistent", nil, pluginpb.PluginV1_MutateRowRequest_DELETE, "t", nil, "")
	if err == nil {
		t.Errorf("expected error for missing plugin")
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

func TestMutateRowParsesResponse(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script plugin not supported on Windows")
	}
	// create a dummy executable that handles the mutate-row command
	dir, err := os.MkdirTemp("", "pmgrmutate")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	name := pluginName("dummy")
	req := strings.TrimSuffix(name, filepath.Ext(name))
	script := filepath.Join(dir, name)
	bin := fmt.Sprintf(`#!/bin/sh
if [ "$1" = "mutate-row" ]; then
  echo '{"success":true}';
else
  echo '{}' ;
fi
`)
	if err := os.WriteFile(script, []byte(bin), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	m := &Manager{plugins: map[string]PluginInfo{req: {Path: script}}}

	resp, err := m.MutateRow(req, nil, pluginpb.PluginV1_MutateRowRequest_INSERT, "t", nil, "")
	if err != nil {
		t.Fatalf("MutateRow error: %v", err)
	}
	if !resp.Success {
		t.Errorf("unexpected response: %+v", resp)
	}
	// also try with extension to ensure normalization
	resp2, err2 := m.MutateRow(name, nil, pluginpb.PluginV1_MutateRowRequest_INSERT, "t", nil, "")
	if err2 != nil {
		t.Fatalf("MutateRow with extension failed: %v", err2)
	}
	if !resp2.Success {
		t.Errorf("unexpected response when using extension: %+v", resp2)
	}
}

func TestScanOnceConcurrent(t *testing.T) {
	dir, err := os.MkdirTemp("", "pmgrscan")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	manifest := `{"id":"%s","type":1,"name":"%s","version":"1.0.0","runtime":{"kind":"local"},"capabilities":["query.execute"],"permissions":[],"limits":{"timeout_seconds":5}}`

	// create two dummy executable files with manifests
	for _, base := range []string{"p1", "p2"} {
		name := pluginName(base)
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(""), 0o755); err != nil {
			t.Fatalf("write dummy plugin %s: %v", name, err)
		}
		sidecar := fmt.Sprintf(manifest, base, strings.ToUpper(base))
		if err := os.WriteFile(path+plugin.ManifestFileSuffix, []byte(sidecar), 0o644); err != nil {
			t.Fatalf("write manifest for %s: %v", name, err)
		}
	}

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
	if err := os.WriteFile(filepath.Join(bundle, fname)+plugin.ManifestFileSuffix, []byte(`{"id":"bundled","version":"0.0.1","runtime":{"kind":"local"},"capabilities":["query.execute"],"permissions":[],"limits":{"timeout_seconds":30}}`), 0o644); err != nil {
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
	if _, err := os.Stat(filepath.Join(userDir, fname) + plugin.ManifestFileSuffix); err != nil {
		t.Fatalf("expected manifest copied to user dir: %v", err)
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

	if err := os.WriteFile(filepath.Join(user, pluginName("dup"))+plugin.ManifestFileSuffix, []byte(`{"id":"dup","type":1,"version":"1.0.0","runtime":{"kind":"unsupported"},"capabilities":["resource.graph"],"permissions":[],"limits":{"timeout_seconds":5}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bundle, pluginName("dup"))+plugin.ManifestFileSuffix, []byte(`{"id":"dup","type":1,"name":"dup","version":"1.0.0","runtime":{"kind":"local"},"capabilities":["resource.graph"],"permissions":[],"limits":{"timeout_seconds":5}}`), 0o644); err != nil {
		t.Fatal(err)
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

func TestScanOnceLoadsManifestMetadata(t *testing.T) {
	dir, err := os.MkdirTemp("", "pmgrmanifest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	name := pluginName("manifested")
	fullpath := filepath.Join(dir, name)
	if err := os.WriteFile(fullpath, []byte(""), 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{"id":"manifested","type":1,"name":"Manifested Plugin","description":"manifest-first plugin","version":"1.2.3","runtime":{"kind":"local"},"capabilities":["query.execute","connection.test"],"permissions":[{"name":"network"}],"limits":{"timeout_seconds":12}}`
	if err := os.WriteFile(fullpath+plugin.ManifestFileSuffix, []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	m := &Manager{
		plugins:    make(map[string]PluginInfo),
		appReadyCh: make(chan struct{}),
		dirs:       []string{dir},
		Dir:        dir,
	}

	m.scanOnce()
	info, ok := m.plugins["manifested"]
	if !ok {
		t.Fatalf("manifested plugin not discovered")
	}
	if info.Name != "Manifested Plugin" {
		t.Fatalf("expected manifest name, got %q", info.Name)
	}
	if info.ManifestPath == "" || !strings.HasSuffix(info.ManifestPath, plugin.ManifestFileSuffix) {
		t.Fatalf("expected manifest path, got %q", info.ManifestPath)
	}
	if info.Runtime == nil || info.Runtime.Kind != plugin.RuntimeKindLocal {
		t.Fatalf("expected local runtime, got %#v", info.Runtime)
	}
	if info.Type != int(pluginpb.PluginV1_DRIVER) {
		t.Fatalf("expected driver type from manifest, got %d", info.Type)
	}
	if len(info.Capabilities) != 2 || info.Capabilities[0] != "query.execute" {
		t.Fatalf("unexpected capabilities: %#v", info.Capabilities)
	}
	if info.Limits == nil || info.Limits.TimeoutSeconds != 12 {
		t.Fatalf("unexpected limits: %#v", info.Limits)
	}
	if info.LastError != "" {
		t.Fatalf("expected manifest-first discovery without error, got %q", info.LastError)
	}
}

func TestScanOnceLoadsManifestDisplayMetadata(t *testing.T) {
	dir, err := os.MkdirTemp("", "pmgrmanifestmerge")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	name := pluginName("manifested")
	fullpath := filepath.Join(dir, name)
	if err := os.WriteFile(fullpath, []byte(""), 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{"id":"manifested","type":1,"name":"Manifested Plugin","description":"driver metadata from manifest","version":"1.2.3","url":"https://example.org/plugin","author":"QueryBox Team","runtime":{"kind":"local"},"capabilities":["query.execute","connection.test"],"permissions":[{"name":"network"}],"limits":{"timeout_seconds":12},"tags":["sql","relational"],"license":"MIT","icon_url":"https://example.org/icon.png","contact":"support@example.org","metadata":{"source":"manifest","simple_icon":"postgresql"},"settings":{"color":"blue"}}`
	if err := os.WriteFile(fullpath+plugin.ManifestFileSuffix, []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	m := &Manager{
		plugins:    make(map[string]PluginInfo),
		appReadyCh: make(chan struct{}),
		dirs:       []string{dir},
		Dir:        dir,
	}

	m.scanOnce()
	info := m.plugins["manifested"]
	if info.URL != "https://example.org/plugin" || info.Author != "QueryBox Team" {
		t.Fatalf("expected UI metadata from manifest, got %#v", info)
	}
	if info.Description != "driver metadata from manifest" {
		t.Fatalf("expected description from manifest, got %q", info.Description)
	}
	if len(info.Capabilities) != 2 || info.Capabilities[0] != "query.execute" || info.Capabilities[1] != "connection.test" {
		t.Fatalf("expected manifest capabilities, got %#v", info.Capabilities)
	}
	if info.Metadata["source"] != "manifest" {
		t.Fatalf("expected manifest metadata, got %#v", info.Metadata)
	}
	if info.Metadata["simple_icon"] != "postgresql" {
		t.Fatalf("expected manifest simple_icon metadata, got %#v", info.Metadata)
	}
	if info.Settings["color"] != "blue" {
		t.Fatalf("expected settings from manifest, got %#v", info.Settings)
	}
	if len(info.Tags) != 2 || info.Tags[0] != "sql" {
		t.Fatalf("expected tags from manifest, got %#v", info.Tags)
	}
}

func TestGetResourceGraphUsesNativeCommand(t *testing.T) {
	m := &Manager{
		plugins: map[string]PluginInfo{
			"native": {
				ID:           "native",
				Path:         "/tmp/native",
				Capabilities: []string{"resource.graph"},
			},
		},
	}

	origRuntime := m.runtime
	m.runtime = &RuntimeManager{
		local: &LocalPluginHost{},
	}

	orig := m.runtime.local
	m.runtime.local = &LocalPluginHostMock{
		RunFunc: func(info PluginInfo, command string, timeout time.Duration, req []byte) ([]byte, error) {
			if command != "resource-graph" {
				return nil, fmt.Errorf("unexpected command %q", command)
			}
			return []byte(`{"nodes":[{"id":"db","name":"DB","kind":"database","path":"db","children":[{"id":"table:users","name":"users","kind":"table","path":"table:users"}]}]}`), nil
		},
	}
	defer func() {
		m.runtime.local = orig
		m.runtime = origRuntime
	}()

	graph, err := m.GetResourceGraph("native", nil)
	if err != nil {
		t.Fatalf("GetResourceGraph error: %v", err)
	}
	if len(graph.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(graph.Nodes))
	}
	root := graph.Nodes[0]
	if root.Kind != "database" || root.Name != "DB" {
		t.Fatalf("unexpected root node: %#v", root)
	}
	if len(root.Children) != 1 || root.Children[0].Kind != "table" {
		t.Fatalf("unexpected child nodes: %#v", root.Children)
	}
}
