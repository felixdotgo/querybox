package pluginmgr

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/felixdotgo/querybox/pkg/plugin"
	pluginpb "github.com/felixdotgo/querybox/rpc/contracts/plugin/v1"
)

// userPluginDirFunc is a test hook that returns the base configuration
// directory for the current user. In production this is os.UserConfigDir.
// Tests override it to control the value without hitting the real filesystem.
var userPluginDirFunc = os.UserConfigDir

// userPluginsDir returns a location under the per-user config area where
// plugins may be stored. It mirrors services.dataDir() behaviour but is
// specific to the plugin subsystem. When UserConfigDir fails or returns an
// empty string we return an empty path.
func userPluginsDir() (string, error) {
	if dir, err := userPluginDirFunc(); err == nil && dir != "" {
		return filepath.Join(dir, "querybox", "plugins"), nil
	}
	return "", fmt.Errorf("user config dir unavailable")
}

// bundledPluginsDir returns the location of the built-in plugins that were
// shipped alongside the executable. This is essentially the old
// defaultPluginsDir implementation. It may point inside an .app bundle on
// macOS or simply ./bin/plugins when running in development.
// bundledPluginsDirFunc is a variable so tests can override where the
// code looks for built-in plugins. Production code assigns the real
// bundledPluginsDir implementation, but tests may substitute a temporary
// directory.
var bundledPluginsDirFunc = bundledPluginsDir

func bundledPluginsDir() string {
	if exe, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}
		dir := filepath.Join(filepath.Dir(exe), "bin", "plugins")
		if _, err := os.Stat(dir); err == nil {
			return dir
		}
	}
	return filepath.Join(".", "bin", "plugins")
}

// populateUserDir copies executable files from the bundled directory into the
// user directory every time New() is called. Existing files will be overwritten
// with the bundle version, ensuring that the on-disk listing mirrors what the
// application shipped with. If the bundle path is empty or unreadable the
// function does nothing.
func populateUserDir(userDir, bundle string) {
	if bundle == "" || userDir == "" {
		return
	}
	entries, err := os.ReadDir(bundle)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		src := filepath.Join(bundle, e.Name())
		dst := filepath.Join(userDir, e.Name())
		srcInfo, err := os.Stat(src)
		if err != nil || !isExecutable(src) {
			continue
		}
		// Skip overwrite when the destination already matches the source by
		// size and modification time. This avoids replacing a plugin binary
		// that may be in active use by another instance of the application.
		if dstInfo, err := os.Stat(dst); err == nil {
			if dstInfo.Size() == srcInfo.Size() && !dstInfo.ModTime().Before(srcInfo.ModTime()) {
				continue
			}
		}
		// read and write bytes; then explicitly chmod to ensure mode isn't
		// stripped by the process umask (common issue on Unix).
		if b, err := os.ReadFile(src); err == nil {
			tmp := dst + ".tmp"
			if werr := os.WriteFile(tmp, b, srcInfo.Mode()); werr == nil {
				_ = os.Chmod(tmp, srcInfo.Mode())
				// rename into place; on Windows this will replace existing file only
				_ = os.Rename(tmp, dst)
				copyManifestSidecar(src, dst)
			}
		}
	}
}

// scanOnce delegates discovery to PluginRegistry and mirrors the result onto
// the legacy Manager.plugins field so existing bindings and tests keep
// functioning while discovery/validation are owned by the registry.
func (m *Manager) scanOnce() {
	m.scanMu.Lock()
	defer m.scanMu.Unlock()

	registry := m.ensureRegistry()
	m.setPlugins(registry.Scan())
}

// isExecutable checks whether the given path looks like an executable file.
func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	mode := info.Mode()
	// On Unix, check any executable bit. On Windows, rely on extension.
	if mode.IsDir() {
		return false
	}
	if filepath.Ext(path) == ".exe" {
		return true
	}
	return mode&0111 != 0
}

// probeInfo executes `binary info` and decodes the JSON InfoResponse. If the
// plugin doesn't implement `info` the call will error and we return that error.
//
// For testability we expose a variable pointing at the real implementation;
// tests may override probeInfoFunc to avoid spawning real binaries.
var probeInfoFunc = probeInfo

func probeInfo(fullpath string) (PluginInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, fullpath, "info")
	hideWindow(cmd)
	out, err := cmd.Output()
	if err != nil {
		return PluginInfo{}, fmt.Errorf("probe info failed: %w", err)
	}

	// Unmarshal known string fields directly. The "type" field needs
	// special handling because newer plugins emit it as a string enum
	// (via protojson) while older ones used a numeric value.
	var resp struct {
		Name         string            `json:"name"`
		Version      string            `json:"version"`
		Description  string            `json:"description"`
		URL          string            `json:"url"`
		Author       string            `json:"author"`
		Capabilities []string          `json:"capabilities"`
		Tags         []string          `json:"tags"`
		License      string            `json:"license"`
		IconURL      string            `json:"icon_url"`
		Contact      string            `json:"contact"`
		Metadata     map[string]string `json:"metadata"`
		Settings     map[string]string `json:"settings"`
		// Type is decoded as json.RawMessage to handle both numeric and string enum values.
		RawType json.RawMessage `json:"type"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return PluginInfo{}, fmt.Errorf("invalid info json: %w", err)
	}

	// interpret the type field from numeric or string enum
	typ := 0
	if len(resp.RawType) > 0 {
		var numVal float64
		if json.Unmarshal(resp.RawType, &numVal) == nil {
			typ = int(numVal)
		} else {
			var strVal string
			if json.Unmarshal(resp.RawType, &strVal) == nil {
				if val, ok := pluginpb.PluginV1_Type_value[strVal]; ok {
					typ = int(val)
				}
			}
		}
	}

	return PluginInfo{
		Name:         resp.Name,
		Type:         typ,
		Version:      resp.Version,
		Description:  resp.Description,
		URL:          resp.URL,
		Author:       resp.Author,
		Capabilities: resp.Capabilities,
		Tags:         resp.Tags,
		License:      resp.License,
		IconURL:      resp.IconURL,
		Contact:      resp.Contact,
		Metadata:     resp.Metadata,
		Settings:     resp.Settings,
	}, nil
}

// Rescan clears the plugin registry and triggers a full re-probe of the
// plugins directory. This ensures that any metadata changes to existing
// plugins are picked up (e.g. after a plugin update).
func (m *Manager) Rescan() error {
	if registry := m.ensureRegistry(); registry != nil {
		registry.Reset()
	}
	m.setPlugins(map[string]PluginInfo{})
	m.scanOnce()
	// after a manual rescan we also fire the ready event so listeners can
	// reload without needing a restart.  The event is synchronous here but
	// that's acceptable since Rescan is called from the UI with a spinner.
	m.emitPluginsReady()
	return nil
}

func copyManifestSidecar(src, dst string) {
	manifestSrc := src + plugin.ManifestFileSuffix
	info, err := os.Stat(manifestSrc)
	if err != nil || info.IsDir() {
		return
	}
	manifestDst := dst + plugin.ManifestFileSuffix
	if dstInfo, err := os.Stat(manifestDst); err == nil {
		if dstInfo.Size() == info.Size() && !dstInfo.ModTime().Before(info.ModTime()) {
			return
		}
	}
	if data, err := os.ReadFile(manifestSrc); err == nil {
		_ = os.WriteFile(manifestDst, data, 0o644)
	}
}
