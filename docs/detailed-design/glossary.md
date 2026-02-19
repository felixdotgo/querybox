# Glossary

**App Service**: Wails service providing window lifecycle management (maximize, minimize, fullscreen, close) for main and connections windows.

**Connection**: A persisted database connection configuration containing metadata (name, driver_type) and a reference to stored credentials.

**ConnectionManager**: SQLite-backed persistence layer managing connection metadata and delegating credential storage to CredManager.

**ConnectionService**: Application-facing service exposing connection CRUD operations to the frontend via Wails bindings.

**CredManager** (Credential Manager): Service managing secure credential storage using OS keyring (primary) with in-memory fallback (headless/test environments).

**credential_key**: TEXT column in SQLite storing a keyring lookup key (format: `"connection:<uuid>"`); references the actual credential stored via CredManager.

**Driver Type**: Identifier matching a plugin name (e.g., "mysql", "postgresql") used to select which plugin handles a connection.

**Fallback Storage**: In-memory credential map (`sync.RWMutex`-protected) used when OS keyring unavailable; cleared on application restart.

**On-Demand Execution**: Plugin invocation model where executables are spawned per-request, execute, and exit (no persistent processes).

**OS Keyring**: Platform-native secure credential storage (macOS Keychain, Windows Credential Manager, Linux Secret Service) accessed via `go-keyring` library.

**Plugin**: Standalone executable implementing CLI-based database driver contract (`info`, `exec`, `authforms` commands) with JSON stdin/stdout communication.

**PluginManager**: Service discovering plugin executables under `bin/plugins/`, managing registry, and executing plugins on-demand with timeout enforcement.

**Plugin Registry**: In-memory map of discovered plugins with metadata (name, path, type, version, description, lastError).

**Plugin SDK**: `pkg/plugin` package providing `ServeCLI` helper and type aliases to protobuf contracts for plugin authors.

**Protobuf Contract**: Canonical API definition in `contracts/plugin/v1/plugin.proto` (generated Go package: `rpc/contracts/plugin/v1`, package `pluginpb`).

**Rescan**: Manual refresh of plugin discovery (`PluginManager.Rescan()`); also runs automatically every 2 seconds.

**Wails Bindings**: Auto-generated TypeScript interfaces enabling type-safe frontend calls to Go services (ConnectionService, PluginManager, App).
