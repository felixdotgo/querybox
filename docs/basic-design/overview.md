# QueryBox Basic Design

**Version**: 0.0.1 (Draft)
**Updated**: February 17, 2026

## 1. Architecture

### 1.1 Overview
- QueryBox Core orchestrates query execution and credential management.
- Driver processes implement database-specific behavior and run out of process.
- Frontend initiates operations through the Wails bridge to Core and streams results supplied by Core.
- Plugins: on‑demand executables discovered under `bin/plugins`; the host exposes `ListPlugins`, `Rescan`, and `ExecPlugin` via the plugin manager. `pkg/plugin` provides a CLI shim and the canonical proto is at `contracts/plugin/v1` (generated package `pluginpb`).

### 1.2 Core Concepts
- Orchestrator Core: stores connection metadata (including a `credential_key` reference) and routes queries to the right driver. Credential secrets are stored in the OS keyring (via `go-keyring`) or an encrypted blob for server/headless deployments.
- Stateless Drivers: launched per request (or short-lived pool), contain protocol logic, and stream results over gRPC.
- Session Envelope: every execution uses an ephemeral session identifier to bind Core and driver communication.
- Separation of Knowledge: Core never implements database protocols; drivers never persist credentials or metadata.

## 2. Connection & Credential Management

### 2.1 Storage
- Connections persist in the Core backing store with metadata plus a `credential_key` (TEXT). The actual credential secret is stored in the OS keyring (desktop) or in an encrypted credential blob (server/headless).
- MVP behaviour: the frontend serializes plugin AuthForms into JSON and sends that string to Core when creating/updating a connection; Core stores the secret in the OS keyring and persists only a `credential_key` reference in SQLite.
- Audit fields capture creation and last access timestamps to support monitoring and rotation.

### 2.2 Execution Flow
1. Frontend asks Core to execute a query on a named connection.
2. Core loads connection metadata (including `credential_blob`) and selects the driver binary by `driver_type`.
3. For plugin-backed connections the `credential_blob` commonly contains JSON: `{"form":"<key>","values":{...}}`; drivers are responsible for interpreting that blob (plugins accept DSN or `credential_blob` JSON today).
4. Core spawns the driver process with predefined timeout and resource limits and provides the connection metadata (including `credential_blob`) to the driver.
5. Driver opens the database connection using the provided credentials, streams rows/errors back to Core, and never persists plaintext secrets.
6. Core forwards streamed results to the frontend and invalidates the session id.

### 2.3 Security posture (current vs planned)
- Current MVP: `credential_blob` is persisted as-is in the local backing store (SQLite). There is no OS keyring or master-key encryption in the current implementation.
- Planned improvements (post-MVP): encrypted credential blobs (AES-256-GCM), OS keyring integration for desktop installs, and master-key provisioning for headless/server deployments (with migration tooling and UX).
- Runtime protections remain in place: per-request timeout, memory caps, non-root execution, and parent-death enforcement.

## 3. MVP Implementation (0.0.1)

### 3.1 Technology Stack
- Core service: Go 1.22, gRPC, core backing store for metadata and credential blob storage.
- Drivers: Go-based reference PostgreSQL driver built with pgx/v5 for protocol handling.
- Encryption (planned): use Go standard crypto/aes, crypto/cipher, and crypto/rand packages for AES-256-GCM when implementing encrypted credential blobs.
- Frontend: existing Wails client communicating with Core via the Wails bridge; no driver-facing changes required in MVP.

### 3.2 Must-Have Deliverables
- Core connection manager with encrypted credential store, schema migrations, and key rotation hooks.
- Shared gRPC contract (Execute, GetSchema) and generated clients for Core and drivers.
- Driver launcher enforcing resource and timeout limits plus session lifecycle.
- PostgreSQL driver implementing Execute and GetSchema using pgx connection pools.
- Frontend wiring to trigger Execute, stream results, and display driver errors.

### 3.3 Operational Tasks
- Secure master key loading (environment file plus documented rotation procedure).
- Structured logging with session identifiers and configurable query redaction.
- Basic telemetry (success/failure counters, duration histogram) exposed via Prometheus endpoint.
- Documentation for driver trust assumptions, installation flow, and credential handling practices.

### 3.4 Frontend UI / Theme
- Use Tailwind's default *light* theme for the entire UI — do not hardcode a global dark background or form colors in `public/style.css`.
- Do not use inline `style="..."` attributes in components; prefer Tailwind utility classes for layout and visual styling.
- Component system: `Naive UI` (Vue 3, themeable). Use Naive UI for form controls and interactive components; pair with Tailwind for layout and utility styling.
- Prefer Tailwind utility classes and component-level classes (e.g. `btn-tw`, `input-tw`) for styling; avoid global color overrides that conflict with Tailwind.
- Inputs and forms should rely on `input-tw` (light background / dark text) and primary actions may use `btn-tw`.
- Document any deliberate deviations from the default Tailwind palette in design docs and PR descriptions.

---
