# Feature: Connection Management

## Overview

`ConnectionService` owns all connection lifecycle operations. It stores metadata in SQLite and delegates secret storage to `CredManager`. All mutations emit Wails events for reactive frontend state updates.

**Location**: `services/connection.go`

---

## API (Wails-bound)

| Method | Signature | Description |
|--------|-----------|-------------|
| `ListConnections` | `(ctx) → ([]Connection, error)` | All connections, newest first |
| `CreateConnection` | `(ctx, name, driverType, credential) → (Connection, error)` | Store secret via CredManager; persist metadata; emit `connection:created` |
| `GetConnection` | `(ctx, id) → (Connection, error)` | Fetch single connection by UUID |
| `GetCredential` | `(ctx, id) → (string, error)` | Raw credential JSON for building plugin requests |
| `DeleteConnection` | `(ctx, id) → error` | Remove metadata + credential; emit `connection:deleted` |

---

## Connection Struct

```go
type Connection struct {
    ID            string `json:"id"`
    Name          string `json:"name"`
    DriverType    string `json:"driver_type"`
    CredentialKey string `json:"credential_key"` // keyring reference, not the secret
    CreatedAt     string `json:"created_at"`
    UpdatedAt     string `json:"updated_at"`
}
```

---

## Create Flow

```
Frontend: CreateConnection(name, driver, credentialJSON)
  → ConnectionService: generate UUID, derive credential_key = "connection:<uuid>"
  → CredManager.Store(credential_key, credentialJSON)     // keyring → sqlite → memory
  → INSERT INTO connections (id, name, driver_type, credential_key, ...)
  → emit "connection:created" { Connection }              // frontend appends to list
```

## Delete Flow

```
Frontend: DeleteConnection(id)
  → ConnectionService: SELECT credential_key WHERE id = ?
  → CredManager.Delete(credential_key)                    // remove from all tiers
  → DELETE FROM connections WHERE id = ?
  → emit "connection:deleted" { id }                      // frontend removes from list
```

## Credential Retrieval (for plugin execution)

```
Frontend: GetCredential(id)
  → ConnectionService: SELECT credential_key WHERE id = ?
  → CredManager.Get(credential_key)                       // keyring → sqlite → memory
  → return credential JSON string
```

Frontend passes the returned JSON as the `connection` parameter in plugin requests.

---

## Implementation Notes

- `credential_key` format: `"connection:<uuid>"`. Never store the secret in `connections.db`.
- SQLite pool: max 1 connection, 5-minute lifetime. Schema auto-created on startup.
- Events emitted strictly **after** successful DB write — never speculatively.
- `GetCredential` is intentionally a separate call so the frontend can defer credential fetch until plugin execution time.

### Branding icons via plugin metadata

Connections are rendered with a logo hinted by the driver plugin.  Drivers
may supply a `simple_icon` key in their `metadata` map (a string corresponding
to a key in the `simple-icons` npm package, e.g. `"postgresql"`).  When present
and recognised the frontend will show that glyph; otherwise the generic server
icon is used.  This makes the connection list more user-friendly without
requiring frontend changes for every new driver.
