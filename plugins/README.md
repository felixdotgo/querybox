# Plugins

Plugins are out-of-process executables placed under `bin/plugins/` and invoked on-demand by the host. QueryBox is manifest-first: a plugin is only discoverable when its compiled binary is accompanied by a valid sidecar manifest `<binary>.manifest.json`.

## Runtime Model

- Source layout: `plugins/<name>/` and `package main`.
- Build path: run `task build:plugins`.
- Runtime execution: one subprocess per request. There are no persistent plugin processes in Phase 1.
- Discovery paths:
  - bundled `bin/plugins`
  - user plugin directory under the OS config area

The manifest is the source of truth for:

- `id`
- `type`
- `version`
- `name`
- `description`
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

`plugin info` may still be implemented for diagnostics or direct inspection, but host discovery does not depend on it.

## CLI Contract

The host may invoke these commands:

- `plugin info`
- `plugin exec`
- `plugin authforms`
- `plugin resource-graph`
- `plugin test-connection`
- `plugin describe-schema`
- `plugin completion-fields`
- `plugin mutate-row`

All request/response payloads are defined by `contracts/plugin/v1/plugin.proto` and wired through `pkg/plugin.ServeCLI()`.

Bundled plugins and the template plugin should implement `resource-graph` directly. QueryBox no longer supports `connection-tree` as an active browse contract.

## Capability Taxonomy

Use only the current capability names:

- `resource.graph`
- `query.execute`
- `stream.read`
- `connection.test`
- `schema.inspect`
- `explain-query`
- `mutate-row`
- `mutate-row::edit`
- `mutate-row::delete`

Do not introduce or keep legacy names such as:

- `query`
- `describe-schema`

### Capability Rules

- Declare `resource.graph` if the plugin exposes browseable resources in the explorer.
- Declare `query.execute` if the plugin accepts `exec`.
- Declare `connection.test` if the plugin implements connectivity checks.
- Declare `schema.inspect` if the plugin implements `describe-schema` or completion/schema metadata.
- Declare `explain-query` only if `exec` understands `options["explain-query"]`.
- Declare `mutate-row` for editable results.
- Add `mutate-row::edit` and/or `mutate-row::delete` when mutation support is narrower than full edit+delete.

## Authoring Guardrails

### Error handling

- Do not swallow connection/auth/runtime errors.
- `test-connection` must return `ok=false` with a concrete message for invalid credentials, refused connections, TLS mismatch, auth failure, or catalog bootstrap failure.
- `resource-graph`, `exec`, and `mutate-row` must surface real failures instead of returning empty success payloads.
- Return an empty tree or empty result only when the underlying state is genuinely empty, not when the connection failed.
- Ignore unknown optional flags only when they are truly optional; do not silently discard required input.

### Resource safety

- Always close `rows`, files, sockets, and database handles.
- Avoid background goroutines, global caches, or retained buffers unless they are bounded and clearly owned.
- Do not hold entire large datasets in memory when a bounded sample is enough.
- Default query/action behavior should be bounded with `LIMIT`, sampling, paging, or catalog scoping.
- Respect manifest limits such as `timeout_seconds` and `max_output_bytes`; do not emit unbounded stdout.
- Prefer row-by-row scanning/formatting over building multiple full copies of the same result in memory.

### Result shape

- Return typed result payloads (`sql`, `document`, `kv`) that match actual data shape.
- Keep `resource-graph` nodes/action metadata small and stable.
- Put UI hints in `metadata` instead of overloading capability names.

## Writing a Plugin

1. Create `plugins/<name>/main.go`.
2. Create `plugins/<name>/plugin.json` with manifest v1 fields and the correct capability list.
3. Import `pkg/plugin` and call `plugin.ServeCLI()` in `main()`.
4. Implement only the commands the manifest advertises.
5. Ensure every command has bounded runtime/memory behavior and explicit failure paths.
6. Treat `0.x` releases as allowed breaking-change space: delete stale compatibility code instead of carrying parallel contracts unless a short-lived migration is explicitly required.
7. Build with `task build:plugins`.
8. Verify the build output contains both the binary and `<binary>.manifest.json`.

See `plugins/template/main.go` for a minimal starting point, but follow this README and `docs/features/02-plugin-system.md` as the current contract.
