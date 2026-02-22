# QueryBox Basic Design

**Version**: 0.0.1 (Draft)
**Updated**: February 17, 2026

## 1. Architecture

### 1.1 Overview
- QueryBox Core orchestrates query execution and credential management.
- Plugins are on-demand executables discovered under `bin/plugins` that implement database-specific behavior.
- Frontend initiates operations through Wails service bindings to Core and receives execution results.
- Plugin Manager exposes `ListPlugins`, `Rescan`, `ExecPlugin`, and `GetPluginAuthForms` via Wails bindings. `pkg/plugin` provides a CLI helper (`ServeCLI`) and the canonical proto is at `contracts/plugin/v1/plugin.proto` (generated package `pluginpb`).

### 1.2 Core Concepts
- **Connection Service**: stores connection metadata in SQLite (including a `credential_key` reference) and delegates credential storage to CredManager.
- **Credential Manager**: stores secrets in OS keyring (via `go-keyring`) with automatic fallback to a local sqlite file when keyring unavailable (server/headless/test environments).
- **Stateless Plugins**: spawned per request, receive connection parameters and query via JSON stdin/stdout, execute queries, and return results as JSON.
- **On-Demand Execution**: plugins are CLI executables invoked when needed; no long-running processes or gRPC communication.
- **Separation of Knowledge**: Core never implements database protocols; plugins never persist credentials or connection metadata.

## 2. Connection & Credential Management

### 2.1 Storage
- Connections persist in SQLite (`data/connections.db`) with metadata plus a `credential_key` (TEXT) reference.
- Actual credential secrets are stored by CredManager:
  - **Primary**: OS keyring via `go-keyring` (macOS Keychain, Windows Credential Manager, Linux Secret Service)
  - **Fallback**: Persistent sqlite file (`data/credentials.db`) when keyring unavailable; only uses in-memory map if the sqlite file cannot be opened.
- When creating/updating connections, the frontend serializes plugin AuthForms to JSON and sends to ConnectionService; Core stores the secret via CredManager and persists only the `credential_key` reference in SQLite.
- Schema includes `created_at` and `updated_at` timestamps for audit tracking.

### 2.2 Execution Flow
1. Frontend calls `PluginManager.ExecPlugin` with plugin name, connection parameters, and query. The returned ExecResponse includes a `result` field holding one of several payload types (sql, document, or kv) so the UI can handle them generically.
2. PluginManager looks up the plugin executable in its registry (scanned from `bin/plugins`).
3. Manager spawns the plugin as a subprocess: `plugin exec` with 30-second timeout.
4. Plugin request is sent as JSON via stdin: `{"connection": {...}, "query": "..."}`.
5. Plugin executes the query against the database and writes JSON response to stdout: `{"result": "...", "error": "..."}` or returns plaintext results.
6. PluginManager reads stdout/stderr, parses the response, and returns results to frontend.
7. Plugin process exits after completing the request; no persistent connections maintained.

### 2.3 Security Posture
- **Current Implementation**:
  - Credentials stored in OS keyring via `go-keyring` (preferred).
  - Automatic fallback to in-memory storage when keyring unavailable.
  - Only `credential_key` references persisted in SQLite; no plaintext secrets on disk.
  - Plugin execution timeout: 30 seconds per request.
  - Plugins receive connection parameters via stdin (ephemeral, not logged).
- **Runtime Protections**:
  - Plugins spawned per-request with context timeout enforcement.
  - No long-running plugin processes to manage.
  - Credential retrieval isolated in CredManager with concurrent-safe access.
- **Migration Support**: Automatic migration from old `credential_blob` column to `credential_key` + keyring storage on startup.

## 3. MVP Implementation (0.0.1)

### 3.1 Technology Stack
- **Backend**: Go 1.26, Wails v3 framework for desktop UI integration.
- **Storage**: SQLite via `modernc.org/sqlite` for connection metadata.
- **Credentials**: `go-keyring` (github.com/zalando/go-keyring) for OS keyring access with sqlite-file fallback (persisted in `data/credentials.db`).
- **Plugins**: Standalone Go executables using CLI JSON interchange (stdin/stdout).
- **Reference Plugins**: MySQL (`go-sql-driver/mysql`) and PostgreSQL (`github.com/lib/pq`); both drivers now support arbitrary connection parameters (tls/settings) with a built-in dialing timeout for robustness.
  - MySQL plugin accepts arbitrary connection parameters (e.g. `tls=skip-verify`) and enforces a default 5s dialing timeout to avoid hanging requests.
- **Frontend**: Vue 3 + Naive UI components, Tailwind CSS for styling, TypeScript bindings auto-generated from Go services.

### 3.2 Current Implementation Status
- ✅ ConnectionService with SQLite persistence and credential_key references.
- ✅ CredManager with OS keyring (go-keyring) + in-memory fallback.
- ✅ PluginManager with on-demand discovery, scanning, and CLI-based execution.
- ✅ MySQL plugin implementing info, exec, and authforms commands (now with TLS/query-parameter support and built‑in connection timeout).
- ✅ Plugin SDK (`pkg/plugin`) with ServeCLI helper and protobuf contracts.
- ✅ Frontend Wails bindings for ConnectionService and PluginManager.
- ✅ Automatic migration from old credential_blob schema to credential_key model.

### 3.3 Operational Considerations
- **Plugin Discovery**: PluginManager scans `bin/plugins/` every 2 seconds for new/removed executables.
- **Credential Migration**: Existing installations automatically migrate from `credential_blob` to keyring on startup.
- **Error Handling**: Plugin failures captured with stderr output; graceful fallback when plugins not implemented.
- **Platform Support**: Cross-platform builds via Taskfile (Windows, macOS, Linux, iOS, Android).
- **Development Workflow**: `wails3 dev` for hot reload, `scripts/build-plugins.sh` for plugin compilation.

### 3.4 Frontend UI / Theme
- Use Tailwind's default *light* theme for the entire UI — do not hardcode a global dark background or form colors in `public/style.css`.
- Do not use inline `style="..."` attributes in components; prefer Tailwind utility classes for layout and visual styling.
- Component system: `Naive UI` (Vue 3, themeable). Use Naive UI for form controls and interactive components; pair with Tailwind for layout and utility styling.
- Prefer Tailwind utility classes and component-level classes (e.g. `btn-tw`, `input-tw`) for styling; avoid global color overrides that conflict with Tailwind.
- Inputs and forms should rely on `input-tw` (light background / dark text) and primary actions may use `btn-tw`.
- Document any deliberate deviations from the default Tailwind palette in design docs and PR descriptions.

---
