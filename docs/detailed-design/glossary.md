# Glossary

**App Service**: Wails service providing window lifecycle management (maximize, minimize, fullscreen, close) for main and connections windows.

**Connection**: A persisted database connection configuration containing metadata (name, driver_type) and a reference to stored credentials.

**ConnectionService**: Application-facing service that embeds all connection persistence and credential-delegation logic. Exposes `CreateConnection`, `ListConnections`, `GetConnection`, `GetCredential`, and `DeleteConnection` to the frontend via Wails bindings. Located at `services/connection.go`.

**CredManager** (Credential Manager): Service managing secure credential storage using OS keyring (primary) with a persistent sqlite-file fallback; in-memory map used only when sqlite is unavailable.

**credential_key**: TEXT column in SQLite storing a keyring lookup key (format: `"connection:<uuid>"`); references the actual credential stored via CredManager.

**Driver Type**: Identifier matching a plugin name (e.g., "mysql", "postgresql") used to select which plugin handles a connection.

**Fallback Storage**: Persistent sqlite file (`data/credentials.db`) used by CredManager when the OS keyring is unavailable; an in-memory map is only used if the sqlite file itself cannot be opened.

**On-Demand Execution**: Plugin invocation model where executables are spawned per-request, execute, and exit (no persistent processes).

**OS Keyring**: Platform-native secure credential storage (macOS Keychain, Windows Credential Manager, Linux Secret Service) accessed via `go-keyring` library.

**Plugin**: Standalone executable implementing the CLI-based database driver contract (`info`, `exec`, `authforms`, `connection-tree` commands) with JSON stdin/stdout communication, using protobuf-derived types.

**PluginManager**: Service (`services/pluginmgr/pluginmgr.go`) discovering plugin executables under `bin/plugins/`, managing an in-memory registry, and executing plugins on-demand with timeout enforcement.

**Plugin Registry**: In-memory map of discovered plugins with metadata (name, path, type, version, description, lastError).

**Plugin SDK**: `pkg/plugin` package providing `ServeCLI` helper and type aliases to protobuf contracts for plugin authors.

**Protobuf Contract**: Canonical API definition in `contracts/plugin/v1/plugin.proto` (generated Go package: `rpc/contracts/plugin/v1`, package `pluginpb`).

**Rescan**: Manual refresh of plugin discovery (`PluginManager.Rescan()`); also runs automatically every 2 seconds.

**Wails Bindings**: Auto-generated TypeScript interfaces enabling type-safe frontend calls to Go services (ConnectionService, PluginManager, App).
