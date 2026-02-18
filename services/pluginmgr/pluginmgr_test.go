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
