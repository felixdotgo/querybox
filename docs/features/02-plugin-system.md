# Feature: Plugin System

## Overview

Plugins are single-shot executables under `bin/plugins/`. The host spawns one subprocess per request, sends a JSON request via stdin, reads a proto-JSON response from stdout, and the plugin exits. No persistent processes. Language-agnostic — any executable can implement the contract.

**Host service**: `services/pluginmgr/pluginmgr.go`
**Plugin SDK**: `pkg/plugin` (`ServeCLI` helper + protobuf aliases)
**Proto contract**: `contracts/plugin/v1/plugin.proto` → generated `rpc/contracts/plugin/v1` (`pluginpb`)

---

## CLI Commands

| Command | Stdin | Stdout | Timeout | Required |
|---------|-------|--------|---------|---------|
| `info` | — | `{name, version, description, type, ...}` | 2s | ✓ |
| `exec` | `{connection, query, options?}` | `{result, error}` | 30s | ✓ |
| `authforms` | — | Auth form definitions | 30s | ✓ |
| `connection-tree` | `{connection}` | `{nodes: [...]}` | 30s | optional |
| `test-connection` | `{connection}` | `{ok: bool, message: string}` | 15s | optional |

### exec — result payloads

`result` contains exactly one of:

| Field | Type | Use |
|-------|------|-----|
| `sql` | `SqlResult{columns, rows}` | Query results with column names |
| `document` | `DocumentResult{documents}` | JSON document store results |
| `kv` | `KvResult{entries}` | Key-value results (also used as raw-text wrapper) |

Plugins that return a raw string are wrapped in `kv` by the host.

### info — optional metadata fields

```json
{
  "name": "mysql",
  "version": "1.0.0",
  "description": "MySQL / MariaDB driver",
  "type": "DRIVER",
  "url": "https://...",
  "author": "...",
  "license": "MIT",
  "icon_url": "...",
  "capabilities": ["explain-query"],
  "tags": ["sql", "relational"],
  "contact": "...",
  "metadata": {},
  "settings": {}
}
```

Hosts ignore unknown fields; older plugins emitting a numeric `type` are also accepted.

---

## Auth Forms

`plugin authforms` returns structured form definitions. The host calls `GetPluginAuthForms(pluginName)` and renders one tab per form. On submit, the frontend serializes form values as JSON and calls `CreateConnection` with the credential string.

Plugins that do not implement `authforms` fall back to a single DSN/credential text input.

---

## Connection Tree

`plugin connection-tree` returns a hierarchical browse structure (e.g. databases → schemas → tables → columns):

```json
{
  "nodes": [
    {
      "id": "db:mydb",
      "label": "mydb",
      "type": "database",
      "children": [...],
      "actions": [
        { "label": "Show Tables", "query": "SHOW TABLES" }
      ]
    }
  ]
}
```

When the user activates a node action, the frontend calls `ExecTreeAction(name, conn, actionQuery, options)` which delegates to `ExecPlugin`.

---

## Explain-Query Capability

If a plugin advertises `"explain-query"` in its `capabilities` array, the host renders an **Explain** button in the result workspace. Clicking it reruns the current query with `options: {"explain-query": "yes"}`. The plugin is responsible for interpreting the flag (e.g. prepending `EXPLAIN`). The host renders the result in a separate **Explain** tab.

---

## Reference Plugins

| Plugin | Commands | Capabilities | Notes |
|--------|----------|-------------|-------|
| `mysql` | exec, authforms, connection-tree, test-connection | explain-query | TLS support |
| `postgresql` | exec, authforms, connection-tree, test-connection | explain-query | |
| `sqlite` | exec, authforms, connection-tree, test-connection | explain-query | Two auth forms: local file (`modernc.org/sqlite`) + Turso Cloud (`go-libsql`) |
| `redis` | exec, authforms | — | Two auth forms: basic (host/port/password/db) + URL string |
| `arangodb` | exec, authforms | — | Multi-model (documents, graphs); basic auth form |

---

## Plugin Discovery

At runtime the host looks in two locations for plugins. The first path is a
user-writable directory under the operating system's config area (`$XDG_CONFIG_HOME/querybox/plugins` on Linux, `%APPDATA%\querybox\plugins` on Windows, `~/Library/Application Support/querybox/plugins` on macOS). Each startup the application copies whatever binaries exist in the bundled `bin/plugins` directory into this user folder, overwriting any existing files; this keeps the user directory in sync with the shipped bundle while still allowing extra drivers to be added. The user directory takes precedence over the bundle when names conflict.

The second path is the traditional `bin/plugins` directory next to the
executable (inside `.app` bundles, installers, or a `wails3 dev` working
directory). This fallback keeps the built-in drivers available even when the
user folder is populated later.

PluginManager scans the configured directories **once at startup**. For each
executable found it probes `plugin info` (2s timeout) and caches the result
in memory for the lifetime of the process. There is no background re-scan;
adding, removing, or replacing a plugin binary requires **restarting the
application** to take effect. `Rescan()` (exposed as a button in the Plugins
window) triggers an immediate synchronous re-probe if a manual refresh is
needed without a full restart.

---

## Writing a Plugin

1. Create `plugins/<name>/main.go` (package `main`).
2. Import `pkg/plugin` and call `plugin.ServeCLI()` in `main()`.
3. Implement handler functions for each command (`exec`, `authforms`, etc.).
4. Build: `task build:plugins` → binary lands in `bin/plugins/<name>` (`.exe` on Windows).
5. Drop binary into `bin/plugins/`; the host discovers it automatically.

See `plugins/template/main.go` for a minimal example with all optional fields.
