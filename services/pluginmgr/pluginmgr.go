package pluginmgr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/felixdotgo/querybox/pkg/plugin"
)

// PluginInfo holds metadata that the UI can display for each plugin.
type PluginInfo struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Running     bool   `json:"running"` // always false in on-demand model
	Type        int    `json:"type,omitempty"` // follows PluginV1.Type enum (DRIVER = 1)
	Version     string `json:"version,omitempty"`
	Description string `json:"description,omitempty"`
	LastError   string `json:"lastError,omitempty"`
}

// Manager discovers executables under ./bin/plugins and invokes them on-demand.
// It does NOT manage long-running plugin processes.
type Manager struct {
	Dir          string
	scanInterval time.Duration

	mu      sync.Mutex
	plugins map[string]PluginInfo

	stopCh chan struct{}
}

// exec request/response used for CLI JSON interchange with plugins.
type execRequest struct {
	Connection map[string]string `json:"connection"`
	Query      string            `json:"query"`
}

type execResponse struct {
	Result string `json:"result"`
	Error  string `json:"error"`
}

// New creates a Manager and starts a background scanner for the plugins folder.
func New() *Manager {
	m := &Manager{
		Dir:          filepath.Join(".", "bin", "plugins"),
		scanInterval: 2 * time.Second,
		plugins:      make(map[string]PluginInfo),
		stopCh:       make(chan struct{}),
	}
	_ = os.MkdirAll(m.Dir, 0o755)
	// Perform an initial synchronous scan so callers (UI) get immediate results on first ListPlugins()
	m.scanOnce()
	go m.run()
	return m
}

func (m *Manager) run() {
	ticker := time.NewTicker(m.scanInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			m.scanOnce()
		case <-m.stopCh:
			return
		}
	}
}

// scanOnce updates the in-memory plugin registry by inspecting the folder. For
// newly discovered executables, it will attempt to probe `plugin info` for
// metadata. Failures are recorded in PluginInfo.LastError but do not prevent
// discovery.
func (m *Manager) scanOnce() {
	files, err := os.ReadDir(m.Dir)
	if err != nil {
		return
	}
	found := map[string]struct{}{}
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		name := f.Name()
		full := filepath.Join(m.Dir, name)
		if !isExecutable(full) {
			continue
		}
		found[name] = struct{}{}
		m.mu.Lock()
		if _, ok := m.plugins[name]; !ok {
			// probe metadata
			info := PluginInfo{Name: name, Path: full, Running: false}
			meta, err := probeInfo(full)
			if err != nil {
				info.LastError = err.Error()
			} else {
				// Preserve filename as the displayed name/key but copy important
				// metadata (type/version/description) returned by the plugin.
				info.Type = meta.Type
				info.Version = meta.Version
				info.Description = meta.Description
				info.LastError = ""
			}
			m.plugins[name] = info
		}
		m.mu.Unlock()
	}

	// remove entries no longer present
	m.mu.Lock()
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
func probeInfo(fullpath string) (PluginInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, fullpath, "info")
	out, err := cmd.Output()
	if err != nil {
		return PluginInfo{}, fmt.Errorf("probe info failed: %w", err)
	}
	var resp struct {
		Type        int    `json:"type,omitempty"`
		Name        string `json:"name"`
		Version     string `json:"version"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return PluginInfo{}, fmt.Errorf("invalid info json: %w", err)
	}
	return PluginInfo{Name: resp.Name, Type: resp.Type, Version: resp.Version, Description: resp.Description}, nil
}

// ListPlugins returns the discovered plugins (does not start them).
func (m *Manager) ListPlugins() []PluginInfo {
	m.mu.Lock()
	defer m.mu.Unlock()
	ret := make([]PluginInfo, 0, len(m.plugins))
	for _, p := range m.plugins {
		ret = append(ret, p)
	}
	return ret
}

// ExecPlugin runs the named plugin with the provided connection info and query.
// The plugin is invoked as an executable: `plugin exec` and receives a JSON
// payload on stdin. The method returns the plugin's result or an error.
func (m *Manager) ExecPlugin(name string, connection map[string]string, query string) (string, error) {
	m.mu.Lock()
	info, ok := m.plugins[name]
	m.mu.Unlock()
	if !ok {
		return "", errors.New("plugin not found")
	}
	full := info.Path
	if !isExecutable(full) {
		return "", errors.New("plugin is not executable")
	}

	req := execRequest{Connection: connection, Query: query}
	b, _ := json.Marshal(&req)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, full, "exec")
	cmd.Env = append(os.Environ(), "QUERYBOX_PLUGIN_NAME="+name)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}

	if err := cmd.Start(); err != nil {
		return "", err
	}

	// send request
	_, _ = stdin.Write(b)
	_ = stdin.Close()

	// read stdout
	outB, _ := io.ReadAll(stdout)
	errB, _ := io.ReadAll(stderr)

	if err := cmd.Wait(); err != nil {
		return "", fmt.Errorf("plugin exited: %w - stderr: %s", err, string(errB))
	}

	var resp execResponse
	if len(outB) == 0 {
		// allow plain text responses
		return string(outB), nil
	}
	if err := json.Unmarshal(outB, &resp); err != nil {
		// if not JSON, return raw output
		return string(outB), nil
	}
	if resp.Error != "" {
		return resp.Result, errors.New(resp.Error)
	}
	return resp.Result, nil
}

// Rescan triggers an immediate directory scan.
func (m *Manager) Rescan() error {
	m.scanOnce()
	return nil
}

// GetPluginAuthForms probes the plugin executable for supported authentication
// forms by invoking `plugin authforms` and decoding the JSON response. If the
// plugin doesn't implement the command or returns no forms an empty map is
// returned.
func (m *Manager) GetPluginAuthForms(name string) (map[string]plugin.AuthForm, error) {
	m.mu.Lock()
	info, ok := m.plugins[name]
	m.mu.Unlock()
	if !ok {
		return nil, errors.New("plugin not found")
	}
	full := info.Path
	if !isExecutable(full) {
		return nil, errors.New("plugin is not executable")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, full, "authforms")
	out, err := cmd.Output()
	if err != nil {
		// treat as not implemented gracefully
		return nil, nil
	}
	if len(out) == 0 {
		return nil, nil
	}
	var resp plugin.AuthFormsResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("invalid authforms json: %w", err)
	}
	// convert to non-pointer map for convenience
	ret := make(map[string]plugin.AuthForm)
	for k, v := range resp.Forms {
		if v == nil {
			continue
		}
		ret[k] = *v
	}
	return ret, nil
}

// EnablePlugin is not applicable for on-demand execution model.
func (m *Manager) EnablePlugin(name string) error {
	return errors.New("enable/disable not supported for on-demand plugins")
}

// DisablePlugin is not applicable for on-demand execution model.
func (m *Manager) DisablePlugin(name string) error {
	return errors.New("enable/disable not supported for on-demand plugins")
}

// Shutdown stops background scanning.
func (m *Manager) Shutdown() {
	close(m.stopCh)
}
