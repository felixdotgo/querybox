# Data and Credential Model

## Current baseline

| Store | File or backend | Purpose |
|------|------------------|---------|
| Connection metadata | `data/connections.db` | profile metadata plus `credential_key` references |
| Fallback secret store | `data/credentials.db` | tier-2 secret storage |
| Primary secret store | OS keyring | tier-1 secret storage |
| Last-resort secret store | in-memory map | tier-3 fallback |

## Important rule

`data/connections.db` stores metadata and references only. It should not contain plaintext secrets or encrypted secret blobs.

## Connection record shape

- `id`
- `name`
- `driver_type`
- `credential_key`
- `created_at`
- `updated_at`

## Access pattern

1. `ConnectionService` creates a connection ID and derived `credential_key`.
2. `CredManager` stores the secret in the best available tier.
3. Metadata is written to `data/connections.db`.
4. At execution time, QueryBox resolves the credential through `CredManager` and forwards it to the plugin request.

## Related docs

- [Connections and profiles](../user-guide/connections-and-profiles.md)
- [Credential management](../features/credential-management.md)
- [Security model](security-model.md)
