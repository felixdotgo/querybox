# Core Services — Component Design

## Overview

QueryBox Core consists of multiple Go services that work together to provide connection management, credential storage, plugin execution, and UI window management. Services are bound to the Wails application and exposed to the frontend via auto-generated TypeScript bindings.

## Service Architecture

```
Frontend (Vue + TypeScript)
    ↓ (Wails bindings)
┌─────────────────────────────────────┐
│         Core Services (Go)          │
├─────────────────────────────────────┤
│ - App Service                       │
│ - ConnectionService                 │
│ - PluginManager                     │
└─────────────────────────────────────┘
    ↓
┌─────────────────────────────────────┐
│      Internal Components            │
├─────────────────────────────────────┤
│ - CredManager (OS keyring + sqlite) │
│ - Plugin executables (CLI)          │
└─────────────────────────────────────┘
```

## 1. App Service

**Location**: `services/app.go`

### Responsibility
Manages Wails application windows and provides window lifecycle controls to the frontend.

### State
- `App`: Reference to `*application.App` (Wails application instance)
- `MainWindow`: Reference to main application window
- `ConnectionsWindow`: Reference to connections management window (hidden by default)

### Public Methods (Frontend-exposed)

| Method | Description |
|--------|-------------|
| `MaximiseMainWindow()` | Maximize main window to full screen |
| `MinimiseMainWindow()` | Minimize main window |
| `CloseMainWindow()` | Close main window (triggers app exit) |
| `ToggleFullScreenMainWindow()` | Toggle fullscreen mode |
| `ShowConnectionsWindow()` | Show connections window and bring to front (always-on-top) |
| `CloseConnectionsWindow()` | Hide connections window (removes always-on-top, hides instead of closing) |

### Implementation Notes
- Connection window is hidden (not destroyed) to avoid WebKit assertion failures on reopen
- Window controls are nil-safe (check before operating)
- Main window maximized by default on startup

## 2. ConnectionService

**Location**: `services/connection.go`

### Responsibility
Application-facing service that **embeds all persistence and credential-delegation logic** (there is no separate `ConnectionManager` struct). Exposes connection CRUD operations to the frontend via Wails bindings. Safe for concurrent use.

### State
- `db`: `*sql.DB` — SQLite connection to `data/connections.db`
- `cred`: `*credmanager.CredManager` — credential storage delegate
- `app`: `*application.App` — Wails app reference for emitting events

### Public Methods (Frontend-exposed)

| Method | Signature | Description |
|--------|-----------|-------------|
| `ListConnections` | `(ctx) → ([]Connection, error)` | Retrieve all connections, newest first |
| `CreateConnection` | `(ctx, name, driverType, credential) → (Connection, error)` | Create new connection — stores credential via CredManager, persists metadata in SQLite, emits `connection:created` |
| `GetConnection` | `(ctx, id) → (Connection, error)` | Retrieve single connection by ID |
| `GetCredential` | `(ctx, id) → (string, error)` | Retrieve raw credential blob for use by plugins |
| `DeleteConnection` | `(ctx, id) → error` | Delete connection + credential, emits `connection:deleted` |

### Data Model (Frontend-visible)

```go
type Connection struct {
    ID            string `json:"id"`             // UUID
    Name          string `json:"name"`           // User-friendly name
    DriverType    string `json:"driver_type"`    // Plugin identifier
    CredentialKey string `json:"credential_key"` // Keyring reference (not the secret)
    CreatedAt     string `json:"created_at"`     // ISO8601 timestamp
    UpdatedAt     string `json:"updated_at"`     // ISO8601 timestamp
}
```

### Schema Management
- Automatic table creation if not exists.
- Automatic migration from old `credential_blob` column to `credential_key` + keyring model on startup.
- SQLite connection pool: max 1 connection, 5-minute lifetime.

### Migration Logic (Startup)
1. Check for `credential_blob` column existence via `PRAGMA table_info`.
2. If present, add `credential_key` column.
3. For each row with non-empty blob:
   - Generate key: `"connection:<id>"`
   - Store blob in CredManager: `CredManager.Store(key, blob)`
   - Update row: set `credential_key = key`, clear `credential_blob = NULL`

## 3. CredManager

**Location**: `services/credmanager/credmanager.go`

### Responsibility
Secure credential storage abstraction with OS keyring (primary) and local sqlite-file fallback (in-memory only if sqlite cannot be opened).

### State
- `mu`: `sync.RWMutex` — guards in-memory fallback map
- `fallback`: `map[string]string` — in-memory credential storage (last-resort)
- `db`: `*sql.DB` — SQLite connection to `data/credentials.db` for persistent fallback

### Constants
- **Service Name**: `"querybox"` (used for keyring storage)

### Public Methods

| Method | Behavior |
|--------|----------|
| `Store(key, secret)` | Try `keyring.Set()` → sqlite `INSERT OR REPLACE` on failure → in-memory map if db error |
| `Get(key)` | Try `keyring.Get()` → sqlite `SELECT` → in-memory map → error if not found |
| `Delete(key)` | Best-effort `keyring.Delete()` + sqlite `DELETE` + in-memory `delete` |
| `Close()` | Close the underlying sqlite connection (safe to call multiple times) |

### Platform Support
- **macOS**: Keychain Services
- **Windows**: Credential Manager
- **Linux**: Secret Service (GNOME Keyring, KWallet)
- **Fallback**: Persistent sqlite file (`data/credentials.db`) when keyring unavailable; in-memory map only when sqlite file cannot be opened.

### Error Handling
- Empty key returns an error immediately.
- Keyring failures trigger automatic fallback (silent).
- Missing credential returns error after exhausting all three tiers.

## 4. PluginManager

**Location**: `services/pluginmgr/pluginmgr.go`

### Responsibility
Discover plugin executables, manage plugin registry, execute plugins on-demand with timeout enforcement, and provide auth form metadata.

### State
- `Dir`: Plugin directory path (`./bin/plugins`)
- `scanInterval`: Background scan frequency (2 seconds)
- `plugins`: `map[string]PluginInfo` (discovered plugins registry)
- `mu`: `sync.Mutex` (registry access protection)
- `stopCh`: Shutdown signal channel

### Public Methods (Frontend-exposed)

| Method | Signature | Description |
|--------|-----------|-------------|
| `ListPlugins` | `() → []PluginInfo` | Return discovered plugins (does not spawn processes) |
| `ExecPlugin` | `(name, connParams, query, options?) → (*ExecResponse, error)` | Execute plugin with 30s timeout; `ExecResponse` carries a typed result (sql/document/kv). `options` map is forwarded to the plugin (e.g. explain-query). |
| `GetPluginAuthForms` | `(name) → (map[string]*AuthForm, error)` | Probe plugin for auth form definitions (2s timeout); returns nil on unimplemented |
| `GetConnectionTree` | `(name, connParams) → (*ConnectionTreeResponse, error)` | Retrieve driver-defined tree; nodes may include `select` / `describe` actions |
| `ExecTreeAction` | `(name, connParams, query, options?) → (*ExecResponse, error)` | Convenience wrapper — delegates to `ExecPlugin` with the action query string and optional options map |
| `Rescan` | `() → error` | Manual trigger for directory scan |

### Data Model

```go
type PluginInfo struct {
    Name        string `json:"name"`        // Filename (registry key)
    Path        string `json:"path"`        // Absolute path to executable
    Running     bool   `json:"running"`     // Always false (on-demand model)
    Type        int    `json:"type"`        // Plugin type (1 = DRIVER)
    Version     string `json:"version"`     // From plugin info response
    Description string `json:"description"` // From plugin info response
    LastError   string `json:"lastError"`   // Probe or exec error message
}
```

### Background Operations
- **Scanner**: Goroutine running every 2s calling `scanOnce()`
- **Discovery**: Detect new executables, probe `plugin info`, update registry
- **Cleanup**: Remove entries for deleted files
- **Graceful Shutdown**: Via `Shutdown()` closing `stopCh`

### Plugin Execution Flow
1. Lookup plugin in registry by name.
2. Validate executable permissions.
3. Create context with 30s timeout.
4. Spawn subprocess: `plugin exec` with `QUERYBOX_PLUGIN_NAME` env var set.
5. Marshal request to JSON and write to stdin: `{"connection": {...}, "query": "..."}`.
6. Read stdout (response) and stderr (errors).
7. Wait for process exit.
8. Unmarshal proto-JSON response via `protojson`; degrade to `kv` wrapper on parse failure.
9. Return `*ExecResponse` or error.

### Plugin Probing
- **Info Probe**: Execute `plugin info` with 2s timeout
- **AuthForms Probe**: Execute `plugin authforms` with 2s timeout
- Graceful failure handling (treat as not implemented if error/empty)

## Service Initialization (main.go)

Services are registered with the Wails application in `main.go`:

```go
connSvc := services.NewConnectionService()
mgr := pluginmgr.New()

app.App = application.New(application.Options{
    Services: []application.Service{
        application.NewService(connSvc),
        application.NewService(mgr),
        application.NewService(app),
    },
})

connSvc.SetApp(app.App)
mgr.SetApp(app.App)
```

## Frontend Integration

Auto-generated TypeScript bindings provide type-safe access:

```typescript
// Example frontend usage
import { CreateConnection, ListConnections } from '@/bindings/...'
import { ExecPlugin, ListPlugins } from '@/bindings/...'

// Create connection
const conn = await CreateConnection("My DB", "mysql", credentialJSON)

// Execute query via plugin
const res = await ExecPlugin("mysql", connParams, "SELECT 1")
// `res.result` will contain one of `{ sql: {...} } | { document: {...} } | { kv: {...} }` depending on plugin
```

## Operational Concerns

### Failure Modes

| Failure | Handling Strategy |
|---------|------------------|
| SQLite DB creation fails | `ConnectionService` returns a degraded instance; operations return errors |
| Keyring unavailable | CredManager automatically falls back to sqlite file then in-memory map |
| Plugin returns unknown JSON | Wrapped in `kv` result with `"_"` key |
| Plugin not found | Return "plugin not found" error |
| Plugin timeout (30s) | Context cancellation, return timeout error |
| Plugin exec error | Capture stderr and return as error message |
| Plugin probe error | Store in `PluginInfo.LastError`; plugin still listed |

### Resource Management
- SQLite connection pool: max 1 connection
- Plugin execution timeout: 30 seconds
- Plugin info probe timeout: 2 seconds
- Plugin scanner interval: 2 seconds
- In-memory fallback: cleared on restart

### Concurrency Safety
- `ConnectionService`: SQLite handles concurrent queries via single-connection pool.
- CredManager: `sync.RWMutex` protects fallback map
- PluginManager: `sync.Mutex` protects plugin registry

## Testing Considerations

### Unit Tests
- `pkg/plugin/plugin_test.go`: Plugin SDK helpers (`FormatSQLValue`, type aliases).

### Integration Tests
- Test credential migration from old schema
- Test keyring fallback behavior
- Test plugin timeout enforcement
- Test concurrent connection operations

### Manual Testing
- Verify keyring integration on each platform (macOS Keychain, Windows Credential Manager, Linux Secret Service).
- Test plugin hot-swap (add/remove binary while app is running).
- Verify window management behaviour on each OS.
- Test connection creation → tree browse → query execution end-to-end.

## Future Enhancements

- **ConnectionService**: Add `UpdateConnection` method.
- **PluginManager**: Add plugin sandboxing (seccomp / WebAssembly).
- **PluginManager**: Add plugin code-signing verification.
- **CredManager**: Add credential rotation support.
- **Audit Service**: Track connection access and modifications.
