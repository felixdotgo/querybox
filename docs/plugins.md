# Plugin model (on-demand CLI)

Overview
- Plugins are single-shot executables discovered in `./bin/plugins`.
- Communication with plugins happens over **stdin/stdout using protobuf‑defined JSON**; because the format is language-agnostic any executable can implement the contract (Go, Python, Rust, etc).
- The host does NOT keep plugin processes running. Instead it invokes plugins on-demand when the user requests an action.
- This enables adding/removing/updating plugin binaries while the app is running without restarting.

Developer flow
1. Create plugin under `plugins/<name>` (package `main`).
2. Build: `task build:plugins` (binary appears in `bin/plugins/<name>`; Windows builds get a `.exe` extension).
3. Drop the built binary into `bin/plugins`; the host will discover it automatically.

Contract (CLI)
- `plugin info` → plugin prints JSON: `{ name, version, description, type }` plus optional metadata fields.  The standard keys now include:
  - `type` (PluginV1_Type enum).
  - `name`, `version`, `description` (existing fields).
  - **optional** extras: `url`, `author`, `capabilities` (string array), `tags` (string array),
    `license`, `icon_url`, `contact`, plus two free-form maps `metadata` and
    `settings`.  Hosts ignore unknown keys so older plugins continue working.
    The UI renders a fixed set of metadata rows (type, version, author, license,
    URL, contact, path) and substitutes a `—` placeholder when a value is
    absent; plugins need not fill every field.  When the response includes an
    `icon_url`/`iconUrl` value, the UI will render the icon adjacent to the
    plugin name.  See `plugins/template` for an example that demonstrates all
    of them.
  Older plugins emitted the numeric value while new ones produce the enum name
  (e.g. `"DRIVER"`); hosts post‑0.0.1 parse either form transparently.
- `plugin exec` → plugin reads JSON `{ connection, query }` from stdin and writes JSON `{ result, error }` to stdout.  `connection` may be a simple DSN string or a credential blob JSON; arbitrary extra query parameters (including `tls` settings for SSL) are allowed and appended by the host. `result` is now a structured object containing one of `sql`, `document`, or `kv` payloads; older plugins may still return a raw string which will be wrapped in a `kv` map by the host.
- `plugin connection-tree` (or simply `plugin tree`) → plugin reads JSON `{ connection }` and returns `{ nodes: [...]}` describing a hierarchical browse structure.  Each node may include an `actions` array describing what the core should do when the user activates that node.
- `plugin test-connection` → plugin reads JSON `{ connection }` from stdin and writes JSON `{ ok: bool, message: string }` to stdout.  The plugin must attempt to open and verify connectivity (e.g. `db.Open` + `db.Ping()` for SQL drivers) without persisting any state.  Plugins that cannot meaningfully probe connectivity should return `{ ok: true, message: "..." }`.  The host uses a **15-second** timeout for this command.

Contract (proto)
- `contracts/plugin/v1/plugin.proto` defines `Info`, `Exec`, `AuthForms`, `ConnectionTree`, and `TestConnection` messages — the canonical proto for plugins (generated Go package: `rpc/contracts/plugin/v1`, `package pluginpb`).
- `TestConnectionRequest` carries `map<string, string> connection` (same format as `ExecRequest.connection`).  `TestConnectionResponse` carries `bool ok` and `string message`.
- `pkg/plugin` is deliberately minimal: it exposes `ServeCLI` (a helper that speaks the protobuf‑defined CLI protocol over stdin/stdout) and a few convenience aliases/constants plus `FormatSQLValue`. Plugins may import `pluginpb` directly if they prefer; the SDK is small so language‑specific implementations need not depend on Go at all.

Auth forms
- Plugins can now expose structured authentication forms via `authforms` (CLI) / `AuthForms` (proto).
- The host will call `GetPluginAuthForms(pluginName)` and render one or more tabs for the plugin's supported forms.
- Credential storage: the UI serializes the selected form values as a JSON string and sends it to Core (previously stored in `credential_blob`). Core now stores the sensitive secret in the OS keyring and persists only a `credential_key` reference in SQLite; plugins *still* receive the credential JSON in the execution request.
- Backwards compatibility: plugins that don't implement `authforms` will continue to work; the UI falls back to the single DSN/credential input.
Runtime contract
- The host still uses the JSON CLI interchange for on‑demand plugins (stdin/stdout); the `.proto` is available if you need a gRPC/shim.

Host-side
- `services/pluginmgr` discovers available executables and invokes them on-demand using the CLI contract.
- `ListPlugins`, `Rescan`, `ExecPlugin`, `GetConnectionTree`, `ExecTreeAction`, `GetPluginAuthForms`, and `TestConnection` are available from the manager for UI integration.
- `GetConnectionTree(name, connection)` — spawns `plugin connection-tree`; returns `ConnectionTreeResponse` (list of nodes with optional actions).
- `ExecTreeAction(name, connection, actionQuery)` — convenience wrapper; delegates to `ExecPlugin` with the action's query string.
- `TestConnection(name, connection)` — spawns `plugin test-connection` (15s timeout); returns `*TestConnectionResponse{Ok bool, Message string}`.  Does not persist any state.  Used by the New Connection form's **Test Connection** button.

Notes
- On-demand model = simpler lifecycle, easier hot-swap, and predictable resource usage.
- Use the proto messages for formalizing the API; runtime uses the CLI JSON interchange.
