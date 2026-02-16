# Core service — component design

## Responsibility
- Store connection metadata and encrypted credential blobs.
- Orchestrate driver processes and route query results to clients.
- Provide OS keyring integration and master-key based decryption.

## Public interfaces
- gRPC: `Execute`, `GetSchema`, `ListConnections`, `ManageConnection`
- Frontend bridge: Wails bridge (UI ↔ Core)

## State & storage
- Backing store for `connections`, `audit_logs`, schema versions.
- Master key provided via environment/secret-manager.

## Failure modes
- Keyring unavailable → server fallback to encrypted blob using master key.
- Driver start failure → return specific error codes and retry policy.

## Operational concerns
- Limits for spawned drivers (timeout, memory).
- Audit logging with session identifiers (no secrets in logs).

## Diagram
- Reference `architecture.md` and `diagrams/architecture.svg` for end-to-end flows.
