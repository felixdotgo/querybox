# Connection Management

## What it is

`ConnectionService` owns connection lifecycle operations. It persists connection metadata and delegates secret storage to `CredManager`.

## Current behavior

- `ListConnections` returns profiles ordered for the UI.
- `CreateConnection` stores credentials, writes metadata, and emits `connection:created`.
- `GetConnection` and `GetCredential` separate metadata retrieval from secret retrieval.
- `DeleteConnection` removes both the credential reference and metadata, then emits `connection:deleted`.

## Key components

- `services/connection.go`
- `services/credmanager/`
- Wails-bound backend methods
- frontend listeners for connection events

## How it connects to other modules

- Uses the credential manager for secret storage.
- Feeds plugin execution with credential payloads on demand.
- Produces events consumed by the frontend.

## Limits and roadmap

- Connection profiles are still single-user local records.
- Future workspace persistence may add richer resume context on top of connections, but not replace them.
