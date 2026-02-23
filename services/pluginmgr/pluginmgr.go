package pluginmgr

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/felixdotgo/querybox/pkg/plugin"
	pluginpb "github.com/felixdotgo/querybox/rpc/contracts/plugin/v1"
	"github.com/felixdotgo/querybox/services"
	"github.com/wailsapp/wails/v3/pkg/application"
	"google.golang.org/protobuf/encoding/protojson"
)

// PluginInfo holds metadata that the UI can display for each plugin.
type PluginInfo struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Running     bool   `json:"running"`        // always false in on-demand model
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
	app    *application.App
}

// SetApp injects the Wails application reference so the Manager can emit
// log events to the frontend. Call this after application.New returns.
func (m *Manager) SetApp(app *application.App) {
	m.app = app
}

// emitLog is a nil-safe helper that emits an app:log event on the Wails app.
func (m *Manager) emitLog(level, message string) {
	if m.app == nil {
		return
	}
	m.app.Event.Emit("app:log", services.LogEntry{
		Level:     services.LogLevel(level),
		Message:   message,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
	})
}

// exec request/response used for CLI JSON interchange with plugins.
// The CLI format mirrors the protobuf types so that authors can simply
// marshal the generated messages. We no longer use a plain string result;
// instead the envelope contains a typed ExecResult (see contracts/plugin/v1).
// For historical compatibility we still accept raw string output and wrap it
// in a key/value result.
type execRequest struct {
	Connection map[string]string `json:"connection"`
	Query      string            `json:"query"`
}

// We reuse the generated protobuf alias for the response so we stay in sync
// with any future changes.
//
// Note that plugin.ExecResponse is alias for pluginpb.PluginV1_ExecResponse.
// Using it here allows json.Unmarshal to correctly populate the nested
// ExecResult field.

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
	// decode in two steps because newer plugin binaries emit the enum
	// as a string (protojson) while older ones used a numeric value.  Doing
	// a straight unmarshal into an `int` would fail on the string case.
	var raw map[string]interface{}
	if err := json.Unmarshal(out, &raw); err != nil {
		return PluginInfo{}, fmt.Errorf("invalid info json: %w", err)
	}

	// interpret the type field from float64 or string enum
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
		Name        string `json:"name"`
		Version     string `json:"version"`
		Description string `json:"description"`
	}
	if b2, err2 := json.Marshal(raw); err2 == nil {
		_ = json.Unmarshal(b2, &resp)
	}

	return PluginInfo{Name: resp.Name, Type: typ, Version: resp.Version, Description: resp.Description}, nil
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
// payload on stdin. The method returns the structured `plugin.ExecResponse` or
// an error.  Historically this returned a raw string; callers may need to
// examine the `Result` field to access rows, documents, or key/value data.
func (m *Manager) ExecPlugin(name string, connection map[string]string, query string) (*plugin.ExecResponse, error) {
	m.mu.Lock()
	info, ok := m.plugins[name]
	m.mu.Unlock()
	if !ok {
		m.emitLog("error", fmt.Sprintf("ExecPlugin: plugin '%s' not found", name))
		return nil, fmt.Errorf("ExecPlugin: plugin %s not found\n", name)
	}
	full := info.Path
	if !isExecutable(full) {
		fmt.Printf("ExecPlugin: path %s not executable\n", full)
		m.emitLog("error", fmt.Sprintf("ExecPlugin: plugin '%s' is not executable", name))
		return nil, fmt.Errorf("ExecPlugin: plugin %s is not executable\n", name)
	}

	// Truncate long queries in log output to keep messages readable
	logQuery := query
	if len(logQuery) > 80 {
		logQuery = logQuery[:80] + "..."
	}
	m.emitLog("info", fmt.Sprintf("ExecPlugin: driver=%s query=%q", name, logQuery))

	req := execRequest{Connection: connection, Query: query}
	b, _ := json.Marshal(&req)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, full, "exec")
	cmd.Env = append(os.Environ(), "QUERYBOX_PLUGIN_NAME="+name)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		m.emitLog("error", fmt.Sprintf("ExecPlugin: stdin pipe error for plugin '%s': %v", name, err))
		return nil, fmt.Errorf("ExecPlugin: stdin pipe error: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		m.emitLog("error", fmt.Sprintf("ExecPlugin: stdout pipe error for plugin '%s': %v", name, err))
		return nil, fmt.Errorf("ExecPlugin: stdout pipe error: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		m.emitLog("error", fmt.Sprintf("ExecPlugin: stderr pipe error for plugin '%s': %v", name, err))
		return nil, fmt.Errorf("ExecPlugin: stderr pipe error: %w", err)
	}

	if err := cmd.Start(); err != nil {
		fmt.Printf("ExecPlugin: start error: %v\n", err)
		m.emitLog("error", fmt.Sprintf("ExecPlugin: failed to start plugin '%s': %v", name, err))
		return nil, fmt.Errorf("ExecPlugin: start error: %w", err)
	}

	// send request
	_, _ = stdin.Write(b)
	_ = stdin.Close()

	// read stdout
	outB, _ := io.ReadAll(stdout)
	errB, _ := io.ReadAll(stderr)

	if err := cmd.Wait(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			// the context will have killed the process after 30s
			m.emitLog("error", fmt.Sprintf("ExecPlugin: plugin '%s' timed out after 30s", name))
			return nil, fmt.Errorf("ExecPlugin: plugin timed out after 30s")
		}
		m.emitLog("error", fmt.Sprintf("ExecPlugin: plugin '%s' exited with error: %v", name, err))
		return nil, fmt.Errorf("ExecPlugin: plugin exited: %w - stderr: %s", err, string(errB))
	}

	// if the plugin didn't emit JSON we still want to return something useful
	// so wrap the raw output in a simple key/value result.  Older clients may
	// still just render the string.
	resp := &plugin.ExecResponse{}
	if len(outB) == 0 {
		return resp, nil
	}
	// protobuf structs are better parsed with protojson which correctly
	// handles oneof fields and enum names.  Older plugins that used
	// `encoding/json` to marshal a proto struct would emit a top-level
	// "Payload" field instead of the expected variant-specific name
	// (e.g. "sql", "kv").  When that happens protojson.Unmarshal complains
	// about an unknown field; we attempt to repair the JSON so the response
	// can still be interpreted.
	if err := protojson.Unmarshal(outB, resp); err != nil {
		// attempt to correct common mis-formatting
		if strings.Contains(err.Error(), "unknown field \"Payload\"") {
			var raw map[string]interface{}
			if jerr := json.Unmarshal(outB, &raw); jerr == nil {
				if r, ok := raw["result"].(map[string]interface{}); ok {
					if payload, ok2 := r["Payload"].(map[string]interface{}); ok2 {
						// move inner keys (should be one of sql/document/kv) up
						for k, v := range payload {
							// older JSON produced by encoding/json used Go struct field names (Sql, Kv, Document).
							// lowercase them so protojson will match the proto name.
							r[strings.ToLower(k)] = v
						}
						delete(r, "Payload")
						if fixed, merr := json.Marshal(raw); merr == nil {
							if perr := protojson.Unmarshal(fixed, resp); perr == nil {
								return resp, nil
							}
						}
					}
				}
			}
		}
		fmt.Printf("ExecPlugin: JSON unmarshal failed: %v\n", err)
		// fallback to embedding the raw output in a KV map under "_".
		return &plugin.ExecResponse{
			Result: &pluginpb.PluginV1_ExecResult{
				Payload: &pluginpb.PluginV1_ExecResult_Kv{
					Kv: &pluginpb.PluginV1_KeyValueResult{
						Data: map[string]string{"_": string(outB)},
					},
				},
			},
		}, nil
	}
	if resp.Error != "" {
		fmt.Printf("ExecPlugin: plugin returned error field: %s\n", resp.Error)
		m.emitLog("error", fmt.Sprintf("ExecPlugin: plugin '%s' returned error: %s", name, resp.Error))
		return resp, fmt.Errorf("ExecPlugin: plugin error: %s", resp.Error)
	}
	m.emitLog("info", fmt.Sprintf("ExecPlugin: driver=%s completed successfully", name))
	return resp, nil
}

// Rescan triggers an immediate directory scan.
func (m *Manager) Rescan() error {
	m.scanOnce()
	return nil
}

// GetConnectionTree asks the named plugin for its connection tree.  The
// request contains only the connection map; the plugin defines node structure
// and actions.  A timeout guards misbehaving plugins.
func (m *Manager) GetConnectionTree(name string, connection map[string]string) (*plugin.ConnectionTreeResponse, error) {
	m.mu.Lock()
	info, ok := m.plugins[name]
	m.mu.Unlock()
	if !ok {
		m.emitLog("error", fmt.Sprintf("GetConnectionTree: plugin '%s' not found", name))
		return nil, fmt.Errorf("GetConnectionTree: plugin %s not found", name)
	}
	full := info.Path
	if !isExecutable(full) {
		m.emitLog("error", fmt.Sprintf("GetConnectionTree: plugin '%s' is not executable", name))
		return nil, fmt.Errorf("GetConnectionTree: plugin %s is not executable", name)
	}
	m.emitLog("info", fmt.Sprintf("GetConnectionTree: fetching tree for driver=%s", name))

	req := plugin.ConnectionTreeRequest{Connection: connection}
	b, _ := json.Marshal(&req)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, full, "connection-tree")
	cmd.Env = append(os.Environ(), "QUERYBOX_PLUGIN_NAME="+name)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		m.emitLog("error", fmt.Sprintf("GetConnectionTree: stdin pipe error for plugin '%s': %v", name, err))
		return nil, fmt.Errorf("GetConnectionTree: stdin pipe error: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		m.emitLog("error", fmt.Sprintf("GetConnectionTree: stdout pipe error for plugin '%s': %v", name, err))
		return nil, fmt.Errorf("GetConnectionTree: stdout pipe error: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		m.emitLog("error", fmt.Sprintf("GetConnectionTree: stderr pipe error for plugin '%s': %v", name, err))
		return nil, fmt.Errorf("GetConnectionTree: stderr pipe error: %w", err)
	}

	if err := cmd.Start(); err != nil {
		m.emitLog("error", fmt.Sprintf("GetConnectionTree: failed to start plugin '%s': %v", name, err))
		return nil, fmt.Errorf("GetConnectionTree: start error: %w", err)
	}

	_, _ = stdin.Write(b)
	_ = stdin.Close()

	outB, _ := io.ReadAll(stdout)
	errB, _ := io.ReadAll(stderr)

	if err := cmd.Wait(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			m.emitLog("error", fmt.Sprintf("GetConnectionTree: plugin '%s' timed out after 30s", name))
			return nil, fmt.Errorf("GetConnectionTree: plugin timed out after 30s")
		}
		m.emitLog("error", fmt.Sprintf("GetConnectionTree: plugin '%s' exited with error: %v", name, err))
		return nil, fmt.Errorf("GetConnectionTree: plugin exited: %w - stderr: %s", err, string(errB))
	}

	resp := &plugin.ConnectionTreeResponse{}
	if len(outB) == 0 {
		m.emitLog("info", fmt.Sprintf("GetConnectionTree: driver=%s returned empty tree", name))
		return resp, nil
	}
	if err := protojson.Unmarshal(outB, resp); err != nil {
		m.emitLog("error", fmt.Sprintf("GetConnectionTree: invalid tree JSON from '%s': %v", name, err))
		return nil, fmt.Errorf("GetConnectionTree: invalid tree json: %w", err)
	}
	m.emitLog("info", fmt.Sprintf("GetConnectionTree: driver=%s returned %d node(s)", name, len(resp.Nodes)))
	return resp, nil
}

// ExecTreeAction is a convenience wrapper for executing the query payload
// attached to a tree node action.  It simply forwards to ExecPlugin.
func (m *Manager) ExecTreeAction(name string, connection map[string]string, actionQuery string) (*plugin.ExecResponse, error) {
	return m.ExecPlugin(name, connection, actionQuery)
}

// GetPluginAuthForms probes the plugin executable for supported authentication
// forms by invoking `plugin authforms` and decoding the JSON response. If the
// plugin doesn't implement the command or returns no forms an empty map is
// returned.
func (m *Manager) GetPluginAuthForms(name string) (map[string]*plugin.AuthForm, error) {
	m.mu.Lock()
	info, ok := m.plugins[name]
	m.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("GetPluginAuthForms: plugin %s not found", name)
	}
	full := info.Path
	if !isExecutable(full) {
		return nil, fmt.Errorf("GetPluginAuthForms: plugin %s is not executable", name)
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
	if err := protojson.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("GetPluginAuthForms: invalid authforms json: %w", err)
	}
	// convert to non-pointer map for convenience (pointer avoids lock copy)
	ret := make(map[string]*plugin.AuthForm)
	for k, v := range resp.Forms {
		if v == nil {
			continue
		}
		ret[k] = v
	}
	return ret, nil
}

// EnablePlugin is not applicable for on-demand execution model.
func (m *Manager) EnablePlugin(name string) error {
	return fmt.Errorf("EnablePlugin: enable/disable not supported for on-demand plugins")
}

// DisablePlugin is not applicable for on-demand execution model.
func (m *Manager) DisablePlugin(name string) error {
	return fmt.Errorf("DisablePlugin: enable/disable not supported for on-demand plugins")
}

// Shutdown stops background scanning.
func (m *Manager) Shutdown() {
	close(m.stopCh)
}
