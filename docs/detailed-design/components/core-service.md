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
│ - ConnectionManager (SQLite)        │
│ - CredManager (OS keyring)          │
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

**Location**: `services/connection_service.go`

### Responsibility
Application-facing service exposing connection CRUD operations to frontend. Thin wrapper over ConnectionManager to keep Wails bindings focused and minimal.

### Dependencies
- `ConnectionManager`: Internal persistence layer

### Public Methods (Frontend-exposed)

| Method | Signature | Description |
|--------|-----------|-------------|
| `ListConnections` | `(ctx) → ([]Connection, error)` | Retrieve all connections |
| `CreateConnection` | `(ctx, name, driverType, credential) → (Connection, error)` | Create new connection with credential storage |
| `DeleteConnection` | `(ctx, id) → error` | Delete connection and associated credentials |
| `GetConnection` | `(ctx, id) → (Connection, error)` | Retrieve single connection by ID |

### Data Model (Frontend-visible)

```go
type Connection struct {
    ID            string `json:"id"`             // UUID
    Name          string `json:"name"`           // User-friendly name
    DriverType    string `json:"driver_type"`    // Plugin identifier
    CredentialKey string `json:"credential_key"` // Keyring reference
    CreatedAt     string `json:"created_at"`     // ISO8601 timestamp
    UpdatedAt     string `json:"updated_at"`     // ISO8601 timestamp
}
```

## 3. ConnectionManager

**Location**: `services/connection/connection.go`

### Responsibility
Internal persistence layer managing connection metadata in SQLite and delegating credential storage to CredManager. Not directly exposed to frontend.

### State
- `db`: `*sql.DB` (SQLite connection to `data/connections.db`)
- `cred`: `*credmanager.CredManager` (credential storage delegate)

### Key Methods

| Method | Description |
|--------|-------------|
| `New()` | Constructor: creates SQLite DB, runs migrations, initializes CredManager |
| `List(ctx)` | Query all connections from SQLite |
| `Create(ctx, name, driverType, credential)` | Generate UUID, store credential in keyring, persist metadata |
| `Delete(ctx, id)` | Remove from SQLite and delete credential from keyring |
| `Get(ctx, id)` | Retrieve single connection by ID |

### Schema Management
- Automatic table creation if not exists
- Automatic migration from old `credential_blob` column to `credential_key` model
- SQLite connection pool: max 1 connection, 5-minute lifetime

### Migration Logic (Startup)
1. Check for `credential_blob` column existence
2. If present, add `credential_key` column
3. For each row with non-empty blob:
   - Generate key: `"connection:<id>"`
   - Store blob in keyring: `CredManager.Store(key, blob)`
   - Update row: set `credential_key = key`, clear `credential_blob`

## 4. CredManager

**Location**: `services/credmanager/credmanager.go`

### Responsibility
Secure credential storage abstraction with OS keyring (primary) and in-memory fallback.

### State
- `fallback`: `map[string]string` (in-memory credential storage)
- `fallbackMu`: `sync.RWMutex` (concurrent access protection)

### Constants
- **Service Name**: `"querybox"` (used for keyring storage)

### Public Methods

| Method | Behavior |
|--------|----------|
| `Store(key, secret)` | Try `keyring.Set()` → fallback to in-memory on failure |
| `Get(key)` | Try `keyring.Get()` → fallback to in-memory → error if not found |
| `Delete(key)` | Best-effort `keyring.Delete()` + remove from in-memory fallback |

### Platform Support
- **macOS**: Keychain Services
- **Windows**: Credential Manager
- **Linux**: Secret Service (GNOME Keyring, KWallet)
- **Fallback**: In-memory map (server/headless/test environments)

### Error Handling
- Keyring failures trigger automatic fallback (silent)
- Empty key returns error
- Missing credential returns error after checking both locations

## 5. PluginManager

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
| `ExecPlugin` | `(name, connParams, query) → (ExecResponse, error)` | Execute plugin with 30s timeout; `ExecResponse` carries a typed result (sql/document/kv) that the UI can render generically |
| `GetPluginAuthForms` | `(name) → (map[string]AuthForm, error)` | Probe plugin for auth form definitions |
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
1. Lookup plugin in registry by name
2. Validate executable permissions
3. Create context with 30s timeout
4. Spawn subprocess: `plugin exec`
5. Marshal request to JSON and write to stdin: `{"connection": {...}, "query": "..."}`
6. Read stdout (response) and stderr (errors)
7. Wait for process exit
8. Parse JSON response or return raw text
9. Return results or error

### Plugin Probing
- **Info Probe**: Execute `plugin info` with 2s timeout
- **AuthForms Probe**: Execute `plugin authforms` with 2s timeout
- Graceful failure handling (treat as not implemented if error/empty)

## Service Initialization (main.go)

Services are registered with Wails application in `main.go`:

```go
application.New(application.Options{
    Services: []application.Service{
        application.NewService(services.NewConnectionService()),
        application.NewService(pluginmgr.New()),
        application.NewService(app), // App struct with window references
    },
})
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
| SQLite DB creation fails | ConnectionManager returns empty manager; operations return errors |
| Keyring unavailable | Automatic fallback to in-memory storage |
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
- ConnectionManager: SQLite handles concurrency
- CredManager: `sync.RWMutex` protects fallback map
- PluginManager: `sync.Mutex` protects plugin registry

## Testing Considerations

### Unit Tests
- `services/connection/connection_test.go`: ConnectionManager CRUD operations
- `services/pluginmgr/pluginmgr_test.go`: Plugin discovery and execution
- `pkg/plugin/plugin_test.go`: Plugin SDK helpers

### Integration Tests
- Test credential migration from old schema
- Test keyring fallback behavior
- Test plugin timeout enforcement
- Test concurrent connection operations

### Manual Testing
- Verify keyring integration on each platform
- Test plugin hot-swap (add/remove while running)
- Verify window management on each OS
- Test connection creation/deletion flow

## Future Enhancements

- **ConnectionService**: Add `UpdateConnection` method
- **PluginManager**: Add plugin enable/disable support
- **Audit Service**: Track connection access and modifications
- **ConnectionManager**: Add connection usage statistics
- **CredManager**: Add credential rotation support
- **PluginManager**: Add plugin permission model and sandboxing
