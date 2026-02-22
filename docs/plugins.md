# Plugin model (on-demand CLI)

Overview
- Plugins are single-shot executables discovered in `./bin/plugins`.
- The host does NOT keep plugin processes running. Instead it invokes plugins on-demand when the user requests an action.
- This enables adding/removing/updating plugin binaries while the app is running without restarting.

Developer flow
1. Create plugin under `plugins/<name>` (package `main`).
2. Build: `task build:plugins` (binary appears in `bin/plugins/<name>`).
3. Drop the built binary into `bin/plugins`; the host will discover it automatically.

Contract (CLI)
- `plugin info` → plugin prints JSON: `{ name, version, description }`.
- `plugin exec` → plugin reads JSON `{ connection, query }` from stdin and writes JSON `{ result, error }` to stdout.  `connection` may be a simple DSN string or a credential blob JSON; arbitrary extra query parameters (including `tls` settings for SSL) are allowed and appended by the host. `result` is now a structured object containing one of `sql`, `document`, or `kv` payloads; older plugins may still return a raw string which will be wrapped in a `kv` map by the host.
- `plugin connection-tree` (or simply `plugin tree`) → plugin reads JSON `{ connection }` and returns `{ nodes: [...]}` describing a hierarchical browse structure.  Each node may include an `actions` array describing what the core should do when the user activates that node.

Contract (proto)
- `contracts/plugin/v1/plugin.proto` defines `Info`, `Exec`, `AuthForms`, and `ConnectionTree` messages — the canonical proto for plugins (generated Go package: `rpc/contracts/plugin/v1`, `package pluginpb`).
- `pkg/plugin` provides a small Go shim (`ServeCLI`), type aliases to `pluginpb`, and the `FormatSQLValue(v interface{}) string` utility for safely converting `database/sql` scan values to human-readable strings (handles `[]byte` → UTF-8 string or hex).

Auth forms
- Plugins can now expose structured authentication forms via `authforms` (CLI) / `AuthForms` (proto).
- The host will call `GetPluginAuthForms(pluginName)` and render one or more tabs for the plugin's supported forms.
- Credential storage: the UI serializes the selected form values as a JSON string and sends it to Core (previously stored in `credential_blob`). Core now stores the sensitive secret in the OS keyring and persists only a `credential_key` reference in SQLite; plugins *still* receive the credential JSON in the execution request.
- Backwards compatibility: plugins that don't implement `authforms` will continue to work; the UI falls back to the single DSN/credential input.
Runtime contract
- The host still uses the JSON CLI interchange for on‑demand plugins (stdin/stdout); the `.proto` is available if you need a gRPC/shim.

Host-side
- `services/pluginmgr` discovers available executables and invokes them on-demand using the CLI contract.
- `ListPlugins`, `Rescan`, `ExecPlugin`, `GetConnectionTree`, `ExecTreeAction`, and `GetPluginAuthForms` are available from the manager for UI integration.
- `GetConnectionTree(name, connection)` — spawns `plugin connection-tree`; returns `ConnectionTreeResponse` (list of nodes with optional actions).
- `ExecTreeAction(name, connection, actionQuery)` — convenience wrapper; delegates to `ExecPlugin` with the action’s query string.

Notes
- On-demand model = simpler lifecycle, easier hot-swap, and predictable resource usage.
- Use the proto messages for formalizing the API; runtime uses the CLI JSON interchange.
