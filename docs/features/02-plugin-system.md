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
| `info` | — | `{name, version, description, type, ...}` | 5s | optional |
| `exec` | `{connection, query, options?}` | `{result, error}` | 30s | ✓ |
| `authforms` | — | Auth form definitions | 2s | ✓ |
| `resource-graph` | `{connection, resource_id?, depth?}` | `{nodes: [...]}` | 30s | ✓ for browseable plugins |
| `test-connection` | `{connection}` | `{ok: bool, message: string}` | 15s | optional |
| `describe-schema` | `{connection, database?, table?}` | `{tables: [{name, columns, indexes}]}` | 30s | optional |
| `completion-fields` | `{connection, database?, collection?}` | `{fields: [{name, type?}]}` | 5s | optional |
| `mutate-row` | `{connection, operation, source, values?, filter?}` | `{success: bool, error?: string}` | 30s | optional |

### exec — result payloads

### completion-fields — editor metadata
`completion-fields` is used by the frontend query editor to obtain field/column names for the currently selected database and collection (or table). The request is best-effort; schemaless plugins may sample recent documents or inspect a limited catalog. The response should contain zero or more `fields` with `name` and optional `type`. Plugins that cannot provide metadata should return an empty response. This RPC is OPTIONAL and behaviour is equivalent to an empty response if the plugin simply exits without writing anything.


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

`metadata` is an arbitrary string→string map.  The host currently recognises an
optional `simple_icon` key whose value should match a name exported by the
[`simple-icons`](https://www.npmjs.com/package/simple-icons) npm package; when
present the UI will render that logo for connections associated with the
plugin. Hosts must ignore unknown metadata keys. Discovery and runtime policy
must not depend on `info`.

### manifest — runtime metadata and limits

Phase 1 introduces a manifest-first discovery path. Each plugin source folder
may ship `plugin.json`; the build copies it beside the binary as
`<binary>.manifest.json`.

Manifest v1 requires:

- `id`
- `type`
- `name`
- `description`
- `version`
- `url`
- `author`
- `runtime`
- `capabilities`
- `permissions`
- `limits`
- `tags`
- `license`
- `icon_url`
- `contact`
- `metadata`
- `settings`

The currently supported capability taxonomy is:

- `resource.graph`
- `query.execute`
- `stream.read`
- `connection.test`
- `schema.inspect`
- `explain-query`
- `mutate-row`
- `mutate-row::edit`
- `mutate-row::delete`

Do not use legacy capability names such as `query` or `describe-schema` in new
or updated plugins.

Bundled plugins are expected to ship a manifest. Discovery fails fast when the
manifest is missing or invalid.

QueryBox is still in `0.x`, so plugin/runtime contract changes may be breaking.
Prefer deleting stale compatibility paths over keeping multiple browse/capability
contracts alive in parallel.

---

## Auth Forms

`plugin authforms` returns structured form definitions. The host calls `GetPluginAuthForms(pluginName)` and renders one tab per form. On submit, the frontend serializes form values as JSON and calls `CreateConnection` with the credential string.

The host method is intentionally permissive: if the named plugin cannot be found (e.g. during a dev-mode backend restart) or is not currently executable, `GetPluginAuthForms` returns `nil` rather than an error. Clients should treat a nil result as “no forms”; this is equivalent to the plugin not implementing the `authforms` command.

Plugins that do not implement `authforms` fall back to a single DSN/credential text input.

---

## Resource Graph

`plugin resource-graph` is the browse contract. It returns a generic
resource tree that is not tied to database-only nouns:

```json
{
  "nodes": [
    {
      "id": "db:mydb",
      "name": "mydb",
      "kind": "database",
      "path": "db:mydb",
      "children": [],
      "actions": [
        { "id": "select", "kind": "select", "title": "Open", "query": "SELECT 1", "new_tab": true }
      ],
      "metadata": {}
    }
  ]
}
```

The host/frontend normalize this graph into the existing explorer model. Node
rendering should prefer `kind`, `actions`, and `metadata`.

On the frontend, the explorer now fetches browse data through
`PluginManager.GetResourceGraph` and normalizes the returned `ResourceNode`
objects in `useResourceGraph.ts`.

## Mutate‑Row Capability

Plugins that support in‑place row/document editing advertise the
`"mutate-row"` string in their `capabilities` list. The frontend uses this to
show pencil/delete icons on query results; missing the capability or a
failure/empty response from the RPC causes the UI to remain read-only. The
RPC itself is optional and, if implemented, should accept the same JSON
structure described earlier and return `success`/`error`.

---

## Explain-Query Capability

If a plugin advertises `"explain-query"` in its `capabilities` array, the host renders an **Explain** button in the result workspace. Clicking it reruns the current query with `options: {"explain-query": "yes"}`. The plugin is responsible for interpreting the flag (e.g. prepending `EXPLAIN`). The host renders the result in a separate **Explain** tab.

---

## Exec Options Convention

The `exec` command accepts an optional `options` map (`map<string, string>`) to pass feature flags. Plugins should check for known keys and ignore unknown ones.

| Option | Value | Description |
|---|---|---|
| `explain-query` | `"yes"` | Plugin prepends `EXPLAIN` to the query |
| `sort-column` | column name string | Plugin appends `ORDER BY <col>` with dialect-specific identifier quoting |
| `sort-direction` | `"asc"` or `"desc"` | Sort direction to use with `sort-column` (default: `"asc"`) |

**Dialect quoting for `sort-column`:**
| Plugin | Quote char | Example |
|---|---|---|
| `mysql` | `` ` `` (backtick) | `` ORDER BY `name` ASC `` |
| `postgresql` | `"` (double-quote) | `ORDER BY "name" ASC` |
| `sqlite` | `"` (double-quote) | `ORDER BY "name" ASC` |

Plugins strip any existing `ORDER BY` clause before appending the new one (using `strings.LastIndex`).

---

## Reference Plugins

| Plugin | Commands | Capabilities | Notes |
|--------|----------|-------------|-------|
| `mysql` | info, exec, authforms, resource-graph, test-connection, describe-schema, completion-fields, mutate-row | `resource.graph`, `query.execute`, `connection.test`, `schema.inspect`, `explain-query`, `mutate-row` | Manifest is the discovery/source-of-truth contract |
| `postgresql` | info, exec, authforms, resource-graph, test-connection, describe-schema, completion-fields, mutate-row | `resource.graph`, `query.execute`, `connection.test`, `schema.inspect`, `explain-query`, `mutate-row` | Manifest is the discovery/source-of-truth contract |
| `redis` | info, exec, authforms, resource-graph, test-connection | `resource.graph`, `query.execute`, `connection.test` | Phase 2 Redis MVP: bounded SCAN-based key inspector over the selected logical DB |
| `sqlite` | info, exec, authforms, resource-graph, test-connection, describe-schema, completion-fields, mutate-row | `resource.graph`, `query.execute`, `connection.test`, `schema.inspect`, `explain-query`, `mutate-row` | Manifest is the discovery/source-of-truth contract |
| `template` | info, exec, authforms, resource-graph, test-connection, mutate-row | `resource.graph`, `connection.test`, `query.execute` | Minimal sample plugin used as authoring reference |

---

## Plugin Discovery

At runtime the host looks in two locations for plugins. The first path is a
user-writable directory under the operating system's config area (`$XDG_CONFIG_HOME/querybox/plugins` on Linux, `%APPDATA%\querybox\plugins` on Windows, `~/Library/Application Support/querybox/plugins` on macOS). Each startup the application copies whatever binaries exist in the bundled `bin/plugins` directory into this user folder, overwriting any existing files; this keeps the user directory in sync with the shipped bundle while still allowing extra drivers to be added. The user directory takes precedence over the bundle when names conflict.

The second path is the traditional `bin/plugins` directory next to the
executable (inside `.app` bundles, installers, or a `wails3 dev` working
directory). This fallback keeps the built-in drivers available even when the
user folder is populated later.

`PluginRegistry` scans the configured directories **once at startup** and again
when `Rescan()` is called. For each executable it loads
`<binary>.manifest.json`, validates supported capabilities/runtime/limits, and
rejects plugins whose manifest is missing or invalid.

Discovery results are cached in memory for the lifetime of the process.
Replacing a plugin binary still requires a restart or a manual `Rescan()` to
take effect.

`RuntimeManager` owns execution. Phase 1 keeps only `LocalPluginHost`, which
spawns the plugin binary on-demand and applies timeout limits before returning
results to `PluginManager`.

---

## Plugin Implementation Guardrails

### Capability declaration

- Manifest capabilities must match real implemented behavior.
- `resource.graph` is required for browseable plugins.
- `query.execute` is required for `exec`.
- `connection.test` is required for explicit connectivity checks.
- `schema.inspect` is required for schema and completion metadata flows.
- `explain-query` is allowed only when the plugin understands `options["explain-query"]`.
- `mutate-row` is required for editable results; add `mutate-row::edit` and/or `mutate-row::delete` when support is narrower.

### Error handling

- Do not swallow connection, auth, TLS, bootstrap, or query errors.
- `test-connection` should return `ok=false` with a concrete message for real failures.
- `resource-graph`, `exec`, and `mutate-row` should return an error or failure payload when the underlying operation fails.
- Empty trees or empty result sets should represent genuinely empty state, not hidden connection failure.

### Memory and process safety

- One request should produce one bounded result; avoid loading unbounded catalogs or datasets into memory.
- Apply `LIMIT`, sampling, paging, or catalog scoping by default where full scans are risky.
- Close rows, files, DB handles, and any other acquired resources.
- Avoid long-lived goroutines, hidden background polling, or unbounded caches in plugin processes.
- Respect manifest limits such as timeout and output size; do not print huge stdout payloads that risk host OOM.

---

## Writing a Plugin

1. Create `plugins/<name>/main.go` (package `main`).
2. Create `plugins/<name>/plugin.json` with manifest v1 fields.
3. Import `pkg/plugin` and call `plugin.ServeCLI()` in `main()`.
4. Implement handler functions for each command (`exec`, `authforms`, `resource-graph`, etc.).
5. Match manifest capabilities to the commands actually implemented; do not keep legacy capability names.
6. Make failure paths explicit and keep result size/memory use bounded.
7. Build: `task build:plugins` → binary lands in `bin/plugins/<name>` (`.exe` on Windows) and the manifest is copied as `<binary>.manifest.json`.
8. Drop the built plugin into `bin/plugins/` or the user plugin directory; the host discovers it automatically at startup or on manual `Rescan()`.

See `plugins/template/main.go` for a minimal example with all optional fields.
