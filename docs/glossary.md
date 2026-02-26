# Glossary

| Term | Definition |
|------|-----------|
| **App Service** | Wails service (`services/app.go`) providing window lifecycle controls (maximize, minimize, fullscreen, close) for main and connections windows. |
| **Connection** | Persisted record of a named database endpoint: metadata (name, driver_type, created_at, updated_at) + a `credential_key` reference. No secrets stored inline. |
| **ConnectionService** | Go service (`services/connection.go`) that owns connection CRUD and credential delegation. Wails-bound. |
| **CredManager** | Go service (`services/credmanager/credmanager.go`) managing secure credential storage with a 3-tier fallback chain: OS keyring → sqlite file → in-memory map. |
| **credential_key** | TEXT column in `connections` table; a lookup key (`connection:<uuid>`) used by CredManager to retrieve the actual secret. Never the secret itself. |
| **Driver Type** | String identifier matching a plugin name (e.g. `"mysql"`, `"postgresql"`) that determines which plugin handles a connection. |
| **Event Bus** | Wails event system used by backend services to push domain events to the frontend. Backend produces; frontend only consumes. |
| **ExecResponse** | Proto-derived response from `plugin exec`: contains one of `sql`, `document`, or `kv` typed payloads. |
| **On-Demand Execution** | Plugin invocation model: one subprocess per request, exit after response. No persistent processes. |
| **OS Keyring** | Platform-native secure store accessed via `go-keyring` (macOS Keychain, Windows Credential Manager, Linux Secret Service). |
| **Plugin** | Standalone executable under `bin/plugins/` implementing the CLI JSON contract (`info`, `exec`, `authforms`, optionally `connection-tree`, `test-connection`). Language-agnostic. |
| **Plugin Capabilities** | Optional string array in `info` response advertising extra features a plugin supports (e.g. `"explain-query"`). |
| **PluginManager** | Go service (`services/pluginmgr/pluginmgr.go`) that discovers plugin executables, maintains an in-memory registry, and executes plugins on-demand with timeout enforcement. |
| **Plugin Registry** | In-memory map of discovered plugins keyed by name, containing full metadata. |
| **Plugin SDK** | `pkg/plugin` — minimal Go package providing `ServeCLI` helper and protobuf type aliases for plugin authors. |
| **Protobuf Contract** | Canonical API at `contracts/plugin/v1/plugin.proto` (generated Go package: `rpc/contracts/plugin/v1`, package `pluginpb`). |
| **Rescan** | Immediate synchronous plugin discovery triggered by `PluginManager.Rescan()`; background scanner also runs every 2 seconds. |
| **TestConnectionResponse** | Proto message `{ok: bool, message: string}` returned by `plugin test-connection`. |
| **Wails Bindings** | Auto-generated TypeScript interfaces enabling type-safe frontend calls to Go services. |
