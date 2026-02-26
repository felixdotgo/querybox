# Data Model

## Storage Overview

| Store | File | Purpose |
|-------|------|---------|
| `data/connections.db` | SQLite | Connection metadata + `credential_key` references |
| `data/credentials.db` | SQLite | Credential secrets (CredManager tier-2 fallback) |
| OS Keyring | Platform | Credential secrets (CredManager tier-1 primary) |
| In-Memory Map | Runtime | Credential secrets (CredManager tier-3 last-resort, ephemeral) |

SQLite driver: `modernc.org/sqlite`. Pool: max 1 open connection, 5-minute lifetime. Schema created/migrated automatically on startup.

---

## connections (data/connections.db)

```sql
CREATE TABLE IF NOT EXISTS connections (
    id           TEXT PRIMARY KEY,
    name         TEXT NOT NULL,
    driver_type  TEXT NOT NULL,
    credential_key TEXT,
    created_at   DATETIME DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    updated_at   DATETIME DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
```

| Column | Type | Description |
|--------|------|-------------|
| `id` | TEXT PK | UUID — `github.com/google/uuid` |
| `name` | TEXT NOT NULL | User-friendly label |
| `driver_type` | TEXT NOT NULL | Plugin identifier e.g. `"mysql"` |
| `credential_key` | TEXT | CredManager lookup key: `"connection:<uuid>"`. Never the secret. |
| `created_at` | DATETIME | ISO8601, UTC |
| `updated_at` | DATETIME | ISO8601, UTC |

**No secrets, no encrypted blobs stored here.** `credential_blob` column was removed after keyring migration.

### Go Struct

```go
type Connection struct {
    ID            string `json:"id"`
    Name          string `json:"name"`
    DriverType    string `json:"driver_type"`
    CredentialKey string `json:"credential_key"`
    CreatedAt     string `json:"created_at"`
    UpdatedAt     string `json:"updated_at"`
}
```

---

## credentials (data/credentials.db) — Tier-2 Fallback

```sql
CREATE TABLE IF NOT EXISTS credentials (
    key    TEXT PRIMARY KEY,
    secret TEXT NOT NULL
);
```

Used only when OS keyring is unavailable. Secrets stored unencrypted — restrict file permissions in production. Survives restarts.

---

## Credential Storage Tiers

| Tier | Backend | Persistent | Notes |
|------|---------|-----------|-------|
| 1 (primary) | OS Keyring via `go-keyring` | ✓ | macOS Keychain / Windows Credential Manager / Linux Secret Service |
| 2 (fallback) | `data/credentials.db` SQLite | ✓ | Server/headless/CI environments; no OS keyring required |
| 3 (last-resort) | In-memory `sync.RWMutex` map | ✗ | Cleared on restart; acceptable only for dev/testing |

CredManager tries tiers in order for every Store/Get/Delete operation, stopping at first success.

---

## Data Access Patterns

```go
// Create
id := uuid.New().String()
key := "connection:" + id
credManager.Store(key, `{"host":"localhost","user":"admin","password":"secret"}`)
db.Exec(`INSERT INTO connections (id, name, driver_type, credential_key) VALUES (?,?,?,?)`, id, "MyDB", "mysql", key)

// Read credential for plugin execution
conn := getConnectionRow(id)
credential, _ := credManager.Get(conn.CredentialKey)

// Delete
credManager.Delete(conn.CredentialKey)
db.Exec(`DELETE FROM connections WHERE id = ?`, id)
```

---

## Indexes & Future Considerations

Current: primary key only (small dataset, single-user desktop app).

Future:
- Index on `driver_type` for driver-specific filtering
- Unique constraint on `name`
- Index on `updated_at` for "recently used" sort
