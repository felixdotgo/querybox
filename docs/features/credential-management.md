# Credential Management

## What it is

`CredManager` stores secrets using a fallback chain so QueryBox remains usable across desktop, CI, and headless environments.

## Current behavior

Storage order:

1. OS keyring
2. `data/credentials.db`
3. in-memory map

The frontend never calls `CredManager` directly; `ConnectionService` mediates access.

## Key components

- `services/credmanager/credmanager.go`
- `services/connection.go`
- `data/credentials.db`

## How it connects to other modules

- Supplies connection secrets to plugin execution.
- Works with the data model that keeps `credential_key` references in `data/connections.db`.
- Supports local-first security behavior without forcing a cloud secret service.

## Limits and roadmap

- SQLite fallback is unencrypted and depends on local filesystem permissions.
- No cross-tier reconciliation occurs if a stronger tier becomes available later.
