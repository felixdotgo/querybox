# Glossary

| Term | Definition |
|------|-----------|
| **App Service** | Wails service (`services/app.go`) providing window lifecycle controls (maximize, minimize, fullscreen, close) for the main window, connections window, and plugins window. |
| **Connection / Profile** | Persisted access profile for a target system. The current product still uses the UI label **Connection**, but the broader runtime direction treats it as a reusable profile for a plugin-backed system plus credential reference. |
| **ConnectionService** | Go service (`services/connection.go`) that owns connection CRUD and credential delegation. Wails-bound. |
| **CredManager** | Go service (`services/credmanager/credmanager.go`) managing secure credential storage with a 3-tier fallback chain: OS keyring → sqlite file → in-memory map. |
| **credential_key** | TEXT column in `connections` table; a lookup key (`connection:<uuid>`) used by CredManager to retrieve the actual secret. Never the secret itself. |
| **Driver Type** | String identifier matching a plugin name (e.g. `"mysql"`, `"postgresql"`) that determines which plugin handles a connection/profile. |
| **Event Bus** | Wails event system used by backend services to push domain events to the frontend. Backend produces; frontend only consumes. |
| **ExecResponse** | Proto-derived response from `plugin exec`: contains one of `sql`, `document`, or `kv` typed payloads. |
| **Operational Workspace** | The target product shape for QueryBox: a local-first desktop workspace where backend engineers inspect, query, debug, and operate data systems. |
| **On-Demand Execution** | Plugin invocation model: one subprocess per request, exit after response. No persistent processes. |
| **OS Keyring** | Platform-native secure store accessed via `go-keyring` (macOS Keychain, Windows Credential Manager, Linux Secret Service). |
| **Plugin** | Standalone executable under `bin/plugins/` (or the per-user plugin directory) that implements the CLI JSON contract. In the current runtime it is expected to ship a sidecar manifest `<binary>.manifest.json` declaring type, runtime, capabilities, permissions, and limits, and to expose browse data via `resource-graph`. |
| **Plugin Capabilities** | The supported manifest capability taxonomy in Phase 1: `resource.graph`, `query.execute`, `stream.read`, `connection.test`, `schema.inspect`. Legacy `info.capabilities` strings such as `explain-query` still exist for older feature flags and UI hints. |
| **PluginManager** | Go service (`services/pluginmgr/pluginmgr.go`) that exposes the public plugin API to the app layer. The frontend explorer now consumes browse data through `GetResourceGraph`, while execution is delegated to `RuntimeManager`. |
| **Plugin Registry** | Discovery/metadata component that scans plugin directories, loads and validates manifests, and caches plugin metadata in memory. Bundled plugins are expected to ship valid manifests; there is no runtime fallback to legacy discovery. |
| **RuntimeManager** | Execution facade that chooses how a plugin runs. Phase 1 ships only `LocalPluginHost`, but the abstraction is intended to support future remote or sandboxed runtimes. |
| **Plugin SDK** | `pkg/plugin` — minimal Go package providing `ServeCLI` helper and protobuf type aliases for plugin authors. |
| **Protobuf Contract** | Canonical API at `contracts/plugin/v1/plugin.proto` (generated Go package: `rpc/contracts/plugin/v1`, package `pluginpb`). |
| **Resource Graph** | The runtime-neutral browse model returned by `resource-graph`. It generalizes the database-specific `connection-tree` into plugin-defined resources, relationships, actions, and metadata. |
| **Resource Action** | A user-triggered operation attached to a resource or workspace context, such as inspect, query, open, export, or stream. |
| **Results / Streams** | Output surfaces for actions. Bounded responses render as results; unbounded or long-lived outputs render as streams. |
| **Rescan** | Immediate synchronous plugin discovery triggered manually via `PluginManager.Rescan()` or the Rescan button in the Plugins window. Discovery also runs asynchronously once at application startup. Plugin binary changes require a restart or manual Rescan to take effect. |
| **TestConnectionResponse** | Proto message `{ok: bool, message: string}` returned by `plugin test-connection`. |
| **Wails Bindings** | Auto-generated TypeScript interfaces enabling type-safe frontend calls to Go services. |
| **Auto‑completion** | In‑editor feature that suggests keywords, commands, or database field names while typing; driven by frontend logic and optional plugin metadata. |
| **Completion Fields** | Metadata returned by the `completion-fields` RPC; a list of field/column names (and optional types) sampled from the target database/collection. |
