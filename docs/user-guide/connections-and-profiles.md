# Connections and Profiles

## What it is

Connections are local profiles that describe how QueryBox should reach a target system. Metadata is stored in `data/connections.db`; secrets are delegated to the credential manager.

## Current behavior

- Profiles are created through the frontend and persisted by `ConnectionService`.
- Secrets are stored separately from metadata.
- The frontend asks for credentials only when a plugin operation needs them.
- Connection lifecycle changes emit backend events for reactive UI updates.

## Credential storage model

Credentials use a three-tier fallback chain:

1. OS keyring
2. `data/credentials.db`
3. In-memory map for last-resort development/testing fallback

See [Credential management](../features/credential-management.md) and [Data and credential model](../architecture/data-and-credential-model.md).

## Related modules

- `services/connection.go`
- `services/credmanager/`
- `frontend/src/composables/usePlugins.ts`
- `features/connection-management.md`
