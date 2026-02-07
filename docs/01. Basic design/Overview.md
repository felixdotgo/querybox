# QueryBox Basic Design

**Version**: 0.0.1 (Draft)
**Updated**: February 8, 2026

## 1. Architecture

### 1.1 Overview
- QueryBox Core orchestrates query execution and credential management.
- Driver processes implement database-specific behavior and run out of process.
- Frontend initiates operations through Core APIs and streams results supplied by Core.

### 1.2 Core Concepts
- Orchestrator Core: stores connection metadata, encrypts credentials, and routes queries to the right driver.
- Stateless Drivers: launched per request (or short-lived pool), contain protocol logic, and stream results over gRPC.
- Session Envelope: every execution uses an ephemeral session identifier to bind Core and driver communication.
- Separation of Knowledge: Core never implements database protocols; drivers never persist credentials or metadata.

## 2. Connection & Credential Management

### 2.1 Storage
- Connections persist in the Core backing store (SQLite) with metadata plus an encrypted credential blob.
- AES-256-GCM with per-record nonce protects credentials; the master key is supplied via environment secret management.
- Audit fields capture creation and last access timestamps to support monitoring and rotation.

### 2.2 Execution Flow
1. Frontend asks Core to execute a query on a named connection.
2. Core loads connection metadata, decrypts credentials in memory, and selects the driver binary by driver_type.
3. Core spawns the driver process with predefined timeout and resource limits.
4. Core issues an Execute gRPC call containing the session id, decrypted credentials, and query payload.
5. Driver opens the database connection, streams rows or errors back via gRPC, and never persists credentials.
6. Core forwards streamed results to the frontend, scrubs credentials from memory, and invalidates the session id.
7. Driver process exits; logs retain the session id for traceability without exposing secrets.

### 2.3 Security Posture
- Trust-based driver model for MVP: official drivers are open source, and installation assumes manual review.
- Runtime protections: per-request timeout (30 s), memory cap (512 MB), non-root execution, and parent death signal enforcement.
- Future hardening (containers, seccomp, network namespaces) is tracked for post-MVP evaluation.

## 3. MVP Implementation (0.0.1)

### 3.1 Technology Stack
- Core service: Go 1.22, gRPC, SQLite (mattn/go-sqlite3) for metadata and credential blob storage.
- Drivers: Go-based reference PostgreSQL driver built with pgx/v5 for protocol handling.
- Encryption: Go standard crypto/aes, crypto/cipher, and crypto/rand packages for AES-256-GCM.
- Frontend: existing Electron client consuming Core gRPC gateway; no driver-facing changes required in MVP.

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

---
