package pluginmgr

import (
	"os"
	"sync"
	"time"

	"github.com/felixdotgo/querybox/pkg/plugin"
	"github.com/felixdotgo/querybox/services"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// On Windows the helper hideWindow (implemented in platform-specific files)
// will configure subprocesses so they do not show a console window. This
// keeps the application from flashing plugin binaries during background
// scans or executions.
//
// PluginInfo holds metadata that the UI can display for each plugin.
//
// The `ID` field is used as the canonical driver identifier (also stored
// in connection.driver_type).  It is always normalized to strip any
// filesystem extension such as ".exe" so that the same value appears on all
// OSes.
type PluginInfo struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Path        string            `json:"path"`
	Running     bool              `json:"running"`        // always false in on-demand model
	Type        int               `json:"type,omitempty"` // follows PluginV1.Type enum (DRIVER = 1)
	Version     string            `json:"version,omitempty"`
	Description string            `json:"description,omitempty"`
	URL         string            `json:"url,omitempty"`
	Author      string            `json:"author,omitempty"`
	Capabilities []string         `json:"capabilities,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	License     string            `json:"license,omitempty"`
	IconURL     string            `json:"icon_url,omitempty"`
	Contact     string            `json:"contact,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Settings    map[string]string `json:"settings,omitempty"`
	LastError   string            `json:"lastError,omitempty"`
}

// Manager discovers executables under one or more plugin directories and
// invokes them on-demand. By default the first scan location is a per-user
// configuration directory (writable by the current user); if that path is
// unavailable the bundled `bin/plugins` directory next to the executable is
// used instead. Plugins found in an earlier directory mask identical names in
// later directories. The Manager does NOT manage long-running plugin processes.
type Manager struct {
    // Dir is the directory that should be treated as the canonical plugin
    // location; it is kept for backwards compatibility and exported bindings.
    // In practice this will equal the first element of dirs (usually the
    // per-user config directory when available).
    Dir string

    // dirs holds the ordered list of directories that will be scanned when
    // looking for plugins. The first entry has precedence in the event of
    // name collisions. The slice may contain one or two elements depending on
    // whether a user directory could be computed.
    dirs []string

    // fallbackDir holds the bundled path, primarily for tests and logging.
    // It is equal to bundledPluginsDir() and may be empty if the user dir
    // took precedence and the bundled path is not present.
    fallbackDir string

	mu      sync.Mutex
	scanMu  sync.Mutex // serializes scanOnce calls so concurrent Rescan/init don't interleave
	plugins map[string]PluginInfo

	emitter    services.EventEmitter
	appReadyCh chan struct{} // closed by SetApp once the Wails app is available

	// onPluginsReady, if non-nil, is invoked whenever a plugins:ready event is
	// emitted. This is useful for tests that don't run a full Wails application.
	onPluginsReady func()
}

// SetApp injects the Wails application reference so the Manager can emit
// log events to the frontend. Call this after application.New returns.
func (m *Manager) SetApp(app *application.App) {
	m.emitter = &services.WailsEmitter{App: app}
	close(m.appReadyCh)
}

// emitLog is a nil-safe helper that emits an app:log event via the EventEmitter.
func (m *Manager) emitLog(level services.LogLevel, message string) {
	if m.emitter == nil {
		return
	}
	m.emitter.EmitEvent(services.EventAppLog, services.LogEntry{
		Level:     level,
		Message:   message,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
	})
}

// Plugin command timeout constants.
const (
	defaultPluginTimeout = 30 * time.Second
	fastPluginTimeout    = 15 * time.Second
)

// exec request/response used for CLI JSON interchange with plugins.
// The CLI format mirrors the protobuf types so that authors can simply
// marshal the generated messages. We no longer use a plain string result;
// instead the envelope contains a typed ExecResult (see contracts/plugin/v1).
// For historical compatibility we still accept raw string output and wrap it
// in a key/value result.
type execRequest struct {
	Connection map[string]string `json:"connection"`
	Query      string            `json:"query"`
	// opaque options forwarded from the frontend; currently used for
	// explain-query=yes requests.  This mirrors the protobuf ExecRequest
	// `options` field and allows the host to signal driver-specific flags.
	Options    map[string]string `json:"options,omitempty"`
}

// mutateRowRequest mirrors the protobuf MutateRowRequest but uses simple
// Go types for CLI JSON encoding.  The `Operation` field reuses the
// alias defined in pkg/plugin so the enum names are consistent.
type mutateRowRequest struct {
	Connection map[string]string        `json:"connection"`
	Operation  plugin.OperationType     `json:"operation"`
	Source     string                   `json:"source"`
	Values     map[string]string        `json:"values"`
	Filter     string                   `json:"filter"`
}

// We reuse the generated protobuf alias for the response so we stay in sync
// with any future changes.
//
// Note that plugin.ExecResponse is alias for pluginpb.PluginV1_ExecResponse.
// Using it here allows json.Unmarshal to correctly populate the nested
// ExecResult field.

// New creates a Manager, performs a single plugin scan at startup, and
// emits "plugins:ready" when done. Plugins are not re-scanned at runtime;
// the user must restart the application to pick up added or removed plugins.
// It prefers a writable per-user directory but will fall back to the bundled
// location beside the executable. The returned Manager populates Dir, dirs,
// and fallbackDir accordingly.
func New() *Manager {
    userDir, err := userPluginsDir()
    bundle := bundledPluginsDirFunc()

    m := &Manager{
        plugins:    make(map[string]PluginInfo),
        appReadyCh: make(chan struct{}),
        fallbackDir: bundle,
    }

    if err == nil && userDir != "" {
        // if the user directory exists or can be created, use it as primary
        // and copy bundled plugins into it every run. This keeps the user
        // directory in sync with whatever shipped in the bundle; bundle files
        // will replace any existing copies.
        if err2 := os.MkdirAll(userDir, 0o755); err2 == nil {
            populateUserDir(userDir, bundle)
        }
        m.dirs = append(m.dirs, userDir)
        m.Dir = userDir
    }

    if bundle != "" {
        // always include bundle location as fallback so that built-in plugins
        // remain usable even if the user directory is populated later.
        m.dirs = append(m.dirs, bundle)
        if m.Dir == "" {
            // if no user dir, make bundle the canonical Dir
            m.Dir = bundle
        }
    }

    // ensure we at least have something to scan
    if m.Dir == "" {
        // last resort: use old behaviour
        m.Dir = bundle
        m.dirs = []string{bundle}
        _ = os.MkdirAll(m.Dir, 0o755)
    }

	// Probing each plugin binary can take up to 2 seconds (timeout), and with
	// several plugins this adds up before Wails even initialises its windows.
	// emitPluginsReady fires a "plugins:ready" event once the scan completes so
	// the frontend can reload its plugin list without polling.
	go func() {
		m.scanOnce()
		m.emitPluginsReady()
	}()
	return m
}

// emitPluginsReady emits the EventPluginsReady event to inform the frontend
// that the initial plugin scan has completed and ListPlugins() is populated.
// It waits for SetApp() to provide the Wails app reference before emitting.
func (m *Manager) emitPluginsReady() {
	select {
	case <-m.appReadyCh:
		// app is ready, emit the event
	case <-time.After(10 * time.Second):
		// give up if SetApp is never called (e.g. in tests)
		return
	}
	// notify any test hook before sending the Wails event
	if m.onPluginsReady != nil {
		m.onPluginsReady()
	}
	if m.emitter != nil {
		m.emitter.EmitEvent(services.EventPluginsReady, nil)
	}
}

// Shutdown is a no-op; there is no background scanner to stop.
// It is kept so Wails can still call the lifecycle method without error.
func (m *Manager) Shutdown() {}

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
