package pluginmgr

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/felixdotgo/querybox/pkg/driverid"
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
			}
		}
	}
}

// scanOnce updates the in-memory plugin registry by inspecting the folder. For
// newly discovered executables, it will attempt to probe `plugin info` for
// metadata. Failures are recorded in PluginInfo.LastError but do not prevent
// discovery.
func (m *Manager) scanOnce() {
	m.scanMu.Lock()
	defer m.scanMu.Unlock()

	// iterate through each configured directory in order; user directory
	// entries mask any identically named binaries in a later directory.
	found := map[string]struct{}{}
	type candidate struct {
		name   string
		full   string
		dirIdx int // index in m.dirs where this candidate came from
	}
	var toProbe []candidate

	m.mu.Lock()
	for idx, dir := range m.dirs {
		files, err := os.ReadDir(dir)
		if err != nil {
			continue // missing/ unreadable dirs are simply skipped
		}
		for _, f := range files {
			if f.IsDir() {
				continue
			}
			origName := f.Name()
			// normalize plugin identifier by stripping any filesystem extension
			name := driverid.Normalize(origName)
			if _, seen := found[name]; seen {
				// already discovered in a higher‑precedence directory
				continue
			}
			full := filepath.Join(dir, origName)
			if !isExecutable(full) {
				continue
			}
			found[name] = struct{}{}
			existing, exists := m.plugins[name]
			if !exists || existing.LastError != "" {
				toProbe = append(toProbe, candidate{name: name, full: full, dirIdx: idx})
			}
		}
	}
	m.mu.Unlock()

	// probe metadata concurrently (same as before)
	type result struct {
		name string
		info PluginInfo
	}
	resCh := make(chan result, len(toProbe))
	var wg sync.WaitGroup
	for _, cand := range toProbe {
		wg.Add(1)
		go func(c candidate) {
			defer wg.Done()
			// Use normalized `name` (no extension) for ID; keep the original
			// filename as a fallback for display if plugin metadata doesn't
			// provide a nicer human name.
			info := PluginInfo{ID: c.name, Name: c.name, Path: c.full, Running: false}
			meta, err := probeInfoFunc(c.full)
			if err != nil && c.dirIdx == 0 && len(m.dirs) > 1 {
				// primary directory probe failed; try fallback bundle entry if present
				alt := filepath.Join(m.dirs[len(m.dirs)-1], c.name)
				if alt != c.full && isExecutable(alt) {
					if meta2, err2 := probeInfoFunc(alt); err2 == nil {
						meta = meta2
						err = nil
						info.Path = alt // keep bundle path since user copy is bad
					}
				}
			}
			if err != nil {
				info.LastError = err.Error()
			} else {
				if meta.Name != "" {
					info.Name = meta.Name
				}
				info.Type = meta.Type
				info.Version = meta.Version
				info.Description = meta.Description
				info.URL = meta.URL
				info.Author = meta.Author
				info.Capabilities = meta.Capabilities
				info.Tags = meta.Tags
				info.License = meta.License
				info.IconURL = meta.IconURL
				info.Contact = meta.Contact
				// copy through whatever metadata the plugin provided.  The frontend
				// may look for specific hints such as `simple_icon` (a key that
				// indicates a simple-icons glyph to render for the driver) when
				// building the connection UI.
				info.Metadata = meta.Metadata
				info.Settings = meta.Settings
				info.LastError = ""
			}
			resCh <- result{name: c.name, info: info}
		}(cand)
	}
	wg.Wait()
	close(resCh)

	// update map and prune missing entries
	m.mu.Lock()
	for r := range resCh {
		m.plugins[r.name] = r.info
	}
	for name := range m.plugins {
		if _, ok := found[name]; !ok {
			delete(m.plugins, name)
		}
	}
	m.mu.Unlock()
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
		// Type is decoded as json.RawMessage to handle both numeric and string enum values.
		RawType     json.RawMessage   `json:"type"`
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

// Rescan clears the plugin registry and triggers a full re-probe of the
// plugins directory. This ensures that any metadata changes to existing
// plugins are picked up (e.g. after a plugin update).
func (m *Manager) Rescan() error {
	m.mu.Lock()
	m.plugins = make(map[string]PluginInfo)
	m.mu.Unlock()
	m.scanOnce()
	// after a manual rescan we also fire the ready event so listeners can
	// reload without needing a restart.  The event is synchronous here but
	// that's acceptable since Rescan is called from the UI with a spinner.
	m.emitPluginsReady()
	return nil
}
