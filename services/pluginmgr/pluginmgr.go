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
	pluginpb "github.com/felixdotgo/querybox/rpc/contracts/plugin/v1"
	"google.golang.org/protobuf/encoding/protojson"
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
// payload on stdin. The method returns the structured `plugin.ExecResponse` or
// an error.  Historically this returned a raw string; callers may need to
// examine the `Result` field to access rows, documents, or key/value data.
func (m *Manager) ExecPlugin(name string, connection map[string]string, query string) (*plugin.ExecResponse, error) {
	fmt.Printf("ExecPlugin called name=%s query=%s connection=%v\n", name, query, connection)
	m.mu.Lock()
	info, ok := m.plugins[name]
	m.mu.Unlock()
	if !ok {
		fmt.Printf("ExecPlugin: plugin %s not found\n", name)
		return nil, errors.New("plugin not found")
	}
	full := info.Path
	if !isExecutable(full) {
		fmt.Printf("ExecPlugin: path %s not executable\n", full)
		return nil, errors.New("plugin is not executable")
	}

	req := execRequest{Connection: connection, Query: query}
	b, _ := json.Marshal(&req)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, full, "exec")
	cmd.Env = append(os.Environ(), "QUERYBOX_PLUGIN_NAME="+name)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		fmt.Printf("ExecPlugin: start error: %v\n", err)
		return nil, err
	}

	// send request
	fmt.Printf("ExecPlugin: sending request to plugin %s: %s\n", name, string(b))
	_, _ = stdin.Write(b)
	_ = stdin.Close()

	// read stdout
	outB, _ := io.ReadAll(stdout)
	errB, _ := io.ReadAll(stderr)

	fmt.Printf("ExecPlugin: stdout=%s stderr=%s\n", string(outB), string(errB))

	if err := cmd.Wait(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			// the context will have killed the process after 30s
			fmt.Printf("ExecPlugin: plugin timed out after 30s\n")
			return nil, fmt.Errorf("plugin timed out after 30s")
		}
		fmt.Printf("ExecPlugin: command wait error: %v\n", err)
		return nil, fmt.Errorf("plugin exited: %w - stderr: %s", err, string(errB))
	}

	// if the plugin didn't emit JSON we still want to return something useful
	// so wrap the raw output in a simple key/value result.  Older clients may
	// still just render the string.
	resp := &plugin.ExecResponse{}
	if len(outB) == 0 {
		return resp, nil
	}
	// protobuf structs are better parsed with protojson which correctly
	// handles oneof fields and enum names.
	if err := protojson.Unmarshal(outB, resp); err != nil {
		fmt.Printf("ExecPlugin: JSON unmarshal failed: %v\n", err)
		// fallback to embedding the raw output in a KV map under "_".
		return &plugin.ExecResponse{Result: &pluginpb.PluginV1_ExecResult{Payload: &pluginpb.PluginV1_ExecResult_Kv{Kv: &pluginpb.PluginV1_KeyValueResult{Data: map[string]string{"_": string(outB)}}}}}, nil
	}
	if resp.Error != "" {
		fmt.Printf("ExecPlugin: plugin returned error field: %s\n", resp.Error)
		return resp, errors.New(resp.Error)
	}
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
	fmt.Printf("GetConnectionTree called name=%s connection=%v\n", name, connection)
	m.mu.Lock()
	info, ok := m.plugins[name]
	m.mu.Unlock()
	if !ok {
		fmt.Printf("GetConnectionTree: plugin %s not found\n", name)
		return nil, errors.New("plugin not found")
	}
	full := info.Path
	if !isExecutable(full) {
		return nil, errors.New("plugin is not executable")
	}

	req := plugin.ConnectionTreeRequest{Connection: connection}
	b, _ := json.Marshal(&req)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, full, "connection-tree")
	cmd.Env = append(os.Environ(), "QUERYBOX_PLUGIN_NAME="+name)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	_, _ = stdin.Write(b)
	_ = stdin.Close()

	outB, _ := io.ReadAll(stdout)
	errB, _ := io.ReadAll(stderr)

	if err := cmd.Wait(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("plugin timed out after 30s")
		}
		return nil, fmt.Errorf("plugin exited: %w - stderr: %s", err, string(errB))
	}

	resp := &plugin.ConnectionTreeResponse{}
	if len(outB) == 0 {
		return resp, nil
	}
	if err := protojson.Unmarshal(outB, resp); err != nil {
		return nil, fmt.Errorf("invalid tree json: %w", err)
	}
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
	if err := protojson.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("invalid authforms json: %w", err)
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
