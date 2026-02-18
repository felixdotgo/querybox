package pluginmgr

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestScanOnceSetsType verifies that scanOnce records the plugin `Type` field
// based on the metadata returned by the plugin's `info` command.
func TestScanOnceSetsType(t *testing.T) {
	d := t.TempDir()
	bin := filepath.Join(d, "sh-mock")
	content := `#!/bin/sh
if [ "$1" = "info" ]; then
  echo '{"type":1,"name":"sh-mock","version":"0.1.0","description":"mock"}'
  exit 0
fi
# noop for exec
cat > /dev/null
`
	if err := os.WriteFile(bin, []byte(content), 0o755); err != nil {
		t.Fatalf("write mock plugin: %v", err)
	}

	m := &Manager{
		Dir:          d,
		scanInterval: 10 * time.Millisecond,
		plugins:      make(map[string]PluginInfo),
		stopCh:       make(chan struct{}),
	}

	// run scanOnce and validate
	m.scanOnce()
	list := m.ListPlugins()
	if len(list) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(list))
	}
	p := list[0]
	if p.Type != 1 {
		t.Fatalf("expected Type=1, got %d", p.Type)
	}
	if p.Version != "0.1.0" {
		t.Fatalf("unexpected version: %s", p.Version)
	}
	if p.Name != "sh-mock" {
		t.Fatalf("unexpected name (filename preserved): %s", p.Name)
	}
}

func TestProbeAuthForms(t *testing.T) {
	d := t.TempDir()
	bin := filepath.Join(d, "sh-mock-auth")
	content := `#!/bin/sh
if [ "$1" = "info" ]; then
  echo '{"type":1,"name":"sh-mock-auth","version":"0.1.0","description":"mock auth"}'
  exit 0
fi
if [ "$1" = "authforms" ]; then
  cat <<EOF
{"forms":{"basic":{"key":"basic","name":"Basic","fields":[{"type":"TEXT","name":"host","label":"Host","required":true}]}}}
EOF
  exit 0
fi
# noop for exec
cat > /dev/null
`
	if err := os.WriteFile(bin, []byte(content), 0o755); err != nil {
		t.Fatalf("write mock plugin: %v", err)
	}

	m := &Manager{
		Dir:          d,
		scanInterval: 10 * time.Millisecond,
		plugins:      make(map[string]PluginInfo),
		stopCh:       make(chan struct{}),
	}

	m.scanOnce()
	list := m.ListPlugins()
	if len(list) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(list))
	}

	forms, err := m.GetPluginAuthForms("sh-mock-auth")
	if err != nil {
		t.Fatalf("GetPluginAuthForms error: %v", err)
	}
	if len(forms) != 1 {
		t.Fatalf("expected 1 form, got %d", len(forms))
	}
	f, ok := forms["basic"]
	if !ok {
		t.Fatalf("missing basic form")
	}
	if f.Name != "Basic" {
		t.Fatalf("unexpected name: %s", f.Name)
	}
}
