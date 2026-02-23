# QueryBox Basic Design

**Version**: 0.0.1 (Draft)
**Updated**: February 22, 2026

## 1. Architecture

### 1.1 Overview
- QueryBox Core orchestrates query execution and credential management.
- Plugins are on-demand executables discovered under `bin/plugins` that implement database-specific behavior.
- Frontend initiates operations through Wails service bindings to Core and receives execution results.
- **ConnectionService** embeds all persistence and credential-delegation logic (no separate `ConnectionManager` struct). Exposes `CreateConnection`, `ListConnections`, `GetConnection`, `GetCredential`, and `DeleteConnection` via Wails bindings.
- **PluginManager** exposes `ListPlugins`, `Rescan`, `ExecPlugin`, `GetPluginAuthForms`, `GetConnectionTree`, and `ExecTreeAction` via Wails bindings. `pkg/plugin` provides a CLI helper (`ServeCLI`) and the canonical proto is at `contracts/plugin/v1/plugin.proto` (generated package `pluginpb`).
- **Event System**: QueryBox follows a **backend-emits / frontend-listens** contract. All domain events are emitted exclusively by Go services; the frontend only subscribes and reacts — it never calls `Events.Emit` for domain topics.
  - `app:log` → structured `LogEntry{Level, Message, Timestamp}` emitted by every service for observability.
  - `connection:created` → emitted by `ConnectionService` after a successful `CreateConnection`; payload is the full `Connection` object.
  - `connection:deleted` → emitted by `ConnectionService` after a successful `DeleteConnection`; payload carries the connection `id`.
  - Event constants are declared in `services/events.go`. See `docs/detailed-design/architecture.md` § *Event-Driven Architecture Rules* for the full contract and event catalogue.

### 1.2 Core Concepts
- **Connection Service**: stores connection metadata in SQLite (including a `credential_key` reference), delegates secret storage to CredManager, and exposes `GetCredential` to the frontend for building plugin requests.
- **Credential Manager**: 3-tier fallback storage — **primary**: OS keyring via `go-keyring`; **secondary**: persistent sqlite file at `data/credentials.db`; **last-resort**: in-memory map (ephemeral, cleared on restart).
- **Stateless Plugins**: spawned per request, receive connection parameters and query via JSON stdin/stdout, execute queries, and return results as JSON.
- **On-Demand Execution**: plugins are CLI executables invoked when needed; no long-running processes or gRPC communication.
- **Connection Tree**: plugins may implement `connection-tree` to expose a hierarchical browse structure (databases → tables → columns). The Core forwards tree-node actions back to the plugin via `ExecTreeAction`.
- **Separation of Knowledge**: Core never implements database protocols; plugins never persist credentials or connection metadata.

## 2. Connection & Credential Management

### 2.1 Storage
- Connections persist in SQLite (`data/connections.db`) with metadata plus a `credential_key` (TEXT) reference.
- Actual credential secrets are stored by CredManager using a **3-tier fallback**:
  1. **Primary**: OS keyring via `go-keyring` (macOS Keychain, Windows Credential Manager, Linux Secret Service)
  2. **Secondary fallback**: Persistent sqlite file (`data/credentials.db`) — used when keyring is unavailable (headless/server/CI). Survives app restarts.
  3. **Last-resort fallback**: In-memory map — used only if the sqlite fallback file cannot be opened. Cleared on restart.
- When creating connections, the frontend serializes plugin AuthForms to JSON and sends to ConnectionService; Core stores the secret via CredManager and persists only the `credential_key` reference in SQLite.
- `ConnectionService.GetCredential(id)` allows the frontend to retrieve the stored credential by connection ID when building plugin execution requests.
- Schema includes `created_at` and `updated_at` timestamps for audit tracking.

### 2.2 Execution Flow

**Query execution:**
1. Frontend calls `PluginManager.ExecPlugin` with plugin name, connection parameters, and query. The returned `ExecResponse` includes a typed `result` field (one of `sql`, `document`, or `kv` payload).
2. PluginManager looks up the plugin executable in its registry (scanned from `bin/plugins`).
3. Manager spawns the plugin as a subprocess: `plugin exec` with 30-second timeout.
4. Plugin request is sent as JSON via stdin: `{"connection": {...}, "query": "..."}`.
5. Plugin executes the query and writes a proto-JSON response to stdout.
6. PluginManager reads stdout/stderr, unmarshals the response (via `protojson`), and returns results to frontend.
7. Plugin process exits after completing the request; no persistent connections maintained.

**Connection tree browsing:**
1. Frontend calls `PluginManager.GetConnectionTree` with plugin name and connection parameters.
2. Manager spawns `plugin connection-tree` with 30-second timeout, sends `{"connection": {...}}` via stdin.
3. Plugin returns `{"nodes": [...]}` — a hierarchical tree with optional `actions` per node.
4. Frontend renders the tree; when a node action is selected it calls `PluginManager.ExecTreeAction` which delegates to `ExecPlugin` with the action's query string.

### 2.3 Security Posture
- **Current Implementation**:
  - Credentials stored in OS keyring via `go-keyring` (preferred).
  - Automatic fallback chain: sqlite file (`data/credentials.db`) → in-memory map when keyring unavailable.
  - Only `credential_key` references persisted in SQLite; no plaintext secrets on disk.
  - Plugin execution timeout: 30 seconds per request (exec, connection-tree, tree actions).
  - Plugins receive connection parameters via stdin (ephemeral, not logged).
- **Runtime Protections**:
  - Plugins spawned per-request with context timeout enforcement.
  - No long-running plugin processes to manage.
  - Credential retrieval isolated in CredManager with concurrent-safe access (`sync.RWMutex`).
- **Migration Support**: Automatic migration from old `credential_blob` column to `credential_key` + keyring storage on startup.

## 3. MVP Implementation (0.0.1)

### 3.1 Technology Stack
- **Backend**: Go 1.26, Wails v3 framework for desktop UI integration.
- **Storage**: SQLite via `modernc.org/sqlite` for connection metadata (`data/connections.db`) and credential fallback (`data/credentials.db`).
- **Credentials**: 3-tier — `go-keyring` (OS keyring) → sqlite file → in-memory map.
- **Plugins**: Standalone Go executables using CLI JSON interchange (stdin/stdout) with proto-derived types from `rpc/contracts/plugin/v1`.
- **Reference Plugins**: MySQL (`go-sql-driver/mysql`), PostgreSQL (`github.com/lib/pq`), and SQLite (`modernc.org/sqlite`); MySQL and PostgreSQL support arbitrary connection parameters (tls/settings) with a built-in dialing timeout. All three implement `connection-tree`: MySQL/PostgreSQL return schemas → tables, SQLite returns the flat table list from `sqlite_master`.
- **Frontend**: Vue 3 + Naive UI components, Tailwind CSS for styling, TypeScript bindings auto-generated from Go services.

### 3.2 Current Implementation Status
- ConnectionService with SQLite persistence, credential_key references, and `GetCredential` method.
- CredManager with 3-tier fallback: OS keyring → sqlite file (`data/credentials.db`) → in-memory map.
- PluginManager with on-demand discovery, scanning, and CLI-based execution (`ExecPlugin`, `GetConnectionTree`, `ExecTreeAction`).
- MySQL plugin implementing `info`, `exec`, `authforms`, and `connection-tree` commands (TLS/query-parameter support; built-in connection timeout).
- PostgreSQL plugin implementing `info`, `exec`, `authforms`, and `connection-tree` commands.
- SQLite plugin implementing `info`, `exec`, `authforms`, and `connection-tree` commands (file-path based connection via `credential_blob`; tables sourced from `sqlite_master`).
- Plugin SDK (`pkg/plugin`) with ServeCLI helper, protobuf type aliases, and `FormatSQLValue` utility.
- Structured event system: all services emit `app:log` / `LogEntry`; `ConnectionService` emits `connection:created` and `connection:deleted` domain events. Event constants defined in `services/events.go`. Frontend only subscribes — never emits domain events.
- Frontend Wails bindings for ConnectionService and PluginManager.
- Automatic migration from old credential_blob schema to credential_key model.

### 3.3 Operational Considerations
- **Plugin Discovery**: PluginManager scans `bin/plugins/` every 2 seconds for new/removed executables; `Rescan()` triggers an immediate scan.
- **Credential Migration**: Existing installations automatically migrate from `credential_blob` to keyring on startup.
- **Error Handling**: Plugin failures captured with stderr output; graceful fallback when commands not implemented. `ExecPlugin` degrades to wrapping raw text output in a `kv` result.
- **Platform Support**: Cross-platform builds via Taskfile (Windows, macOS, Linux, iOS, Android).
- **Development Workflow**: `wails3 dev` for hot reload, `task build:plugins` / `scripts/build-plugins.sh` for plugin compilation.

### 3.4 Frontend UI / Theme
- Use Tailwind's default *light* theme for the entire UI — do not hardcode a global dark background or form colors in `public/style.css`.
- Do not use inline `style="..."` attributes in components; prefer Tailwind utility classes for layout and visual styling.
- Component system: `Naive UI` (Vue 3, themeable). Use Naive UI for form controls and interactive components; pair with Tailwind for layout and utility styling.
- Prefer Tailwind utility classes and component-level classes (e.g. `btn-tw`, `input-tw`) for styling; avoid global color overrides that conflict with Tailwind.
- Inputs and forms should rely on `input-tw` (light background / dark text) and primary actions may use `btn-tw`.
- Document any deliberate deviations from the default Tailwind palette in design docs and PR descriptions.

### 3.5 Icon System

**Library**: `@vicons/ionicons5` wrapped in Naive UI's `<n-icon>` component.

**Golden rule**: Never import icon components directly from `@vicons/ionicons5` in Vue SFCs or render
functions.  Always import from `frontend/src/lib/icons.js`.  Swapping the icon library only requires
changing that one file.

**Usage in templates:**

```vue
<script setup>
import { TrashOutline } from "@/lib/icons"
</script>

<template>
  <n-icon :size="16"><TrashOutline /></n-icon>
</template>
```

**Usage in render functions** (e.g. `renderPrefix` / `renderLabel` in ConnectionsPanel):

```js
import { NIcon } from "naive-ui"
import { LayersOutline } from "@/lib/icons"

function renderPrefix({ option }) {
  return h(NIcon, { size: 14 }, { default: () => h(LayersOutline) })
}
```

**Icon sizes:**
| Context | Size |
|---|---|
| Toolbar / action buttons | 16 |
| Tree node prefix icons | 14 |
| Inline / footer chevrons | 12 |

**Icon registry** (`frontend/src/lib/icons.js`):

| Export name | Use case |
|---|---|
| `LayersOutline` | Driver group node in connection tree |
| `ServerOutline` | Connection node in connection tree |
| `LibraryOutline` | Tree node with `node_type === "database"` |
| `GridOutline` | Tree node with `node_type === "table"` |
| `CodeSlashOutline` | Tree node with `node_type === "column"` |
| `DocumentOutline` | Unknown / generic tree node fallback |
| `ChevronDownOutline` | Footer collapse toggle (rotated −90° when collapsed) |
| `ArrowDownOutline` | LogsPanel auto-scroll enable toggle |
| `AddCircleOutline` | "New connection" toolbar button |
| `FlashOutline` | "Connect" action on connection row |
| `RefreshOutline` | "Refresh" action on connection row |
| `TrashOutline` | "Delete" action (also in LogsPanel clear) |

**Tree node type resolution** — plugin nodes carry an optional `node_type` string field (defined in
`contracts/plugin/v1/plugin.proto`, field 5 on `ConnectionTreeNode`).  Recognised values and their
icons are tracked in `nodeTypeIconMap` exported from `icons.js`.  Unknown or empty values fall back
to `nodeTypeFallbackIcon` (`DocumentOutline`).  Plugins should set `NodeType` on every node they
return so the frontend can render the correct icon without depth heuristics.

---
