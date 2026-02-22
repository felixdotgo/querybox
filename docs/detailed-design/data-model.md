# Data Model

## Purpose
Describes the persistent data structures used by QueryBox Core services, focusing on connection metadata storage and credential reference management.

## Storage Backend
- **Database**: SQLite via `modernc.org/sqlite`
- **Location**: `data/connections.db` (relative to working directory)
- **Connection Pool**: Max 1 open connection, 5-minute connection lifetime
- **Schema Management**: Automatic schema creation and migration on startup

## Tables

### connections

Stores connection metadata with references to credentials stored in OS keyring.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | TEXT | PRIMARY KEY | UUID (generated via `github.com/google/uuid`) |
| `name` | TEXT | NOT NULL | User-friendly connection name |
| `driver_type` | TEXT | NOT NULL | Plugin/driver identifier (e.g., "mysql", "postgresql") |
| `credential_key` | TEXT | NULL | Key for retrieving secret from CredManager (format: `"connection:<uuid>"`) |
| `created_at` | DATETIME | DEFAULT now() | ISO8601 timestamp of creation |
| `updated_at` | DATETIME | DEFAULT now() | ISO8601 timestamp of last update |

**Notes**:
- `credential_key` references a secret stored via CredManager (OS keyring with sqlite-file fallback).
- No plaintext credentials or encrypted blobs stored in this table.
- Legacy `credential_blob` column was removed after migration to keyring model.

## Credential Storage (CredManager)

Credentials are NOT stored in the connections SQLite. Instead, CredManager handles secret storage with a **3-tier fallback chain**:

### Tier 1: OS Keyring (via go-keyring) — Primary
- **Service Name**: `"querybox"`
- **Key**: Value from `connections.credential_key` (e.g., `"connection:abc123"`)
- **Platforms**:
  - macOS: Keychain
  - Windows: Credential Manager
  - Linux: Secret Service (GNOME Keyring, KWallet, etc.)

### Tier 2: Persistent SQLite File — Secondary Fallback
- **Location**: `data/credentials.db`
- **Used when**: OS keyring is unavailable (server/CI/headless environments)
- **Schema**: single `credentials` table with `key` (PRIMARY KEY) and `secret` columns
- **Persistence**: survives application restarts

### Tier 3: In-Memory Map — Last-Resort Fallback
- Used only when both OS keyring and the sqlite fallback file cannot be used
- Thread-safe access via `sync.RWMutex`
- Cleared on application restart — ephemeral
- No disk persistence

## Migration Path

### From credential_blob to credential_key + Keyring

On startup, ConnectionManager automatically migrates old schemas:

1. Check if `credential_blob` column exists
2. If yes, add `credential_key` column (if not present)
3. For each row with non-empty `credential_blob`:
   - Generate key: `"connection:<id>"`
   - Store blob content in keyring via `CredManager.Store(key, blob)`
   - Update row: set `credential_key = key`
   - Clear old blob: set `credential_blob = NULL`

## Schema Creation SQL

```sql
CREATE TABLE IF NOT EXISTS connections (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    driver_type TEXT NOT NULL,
    credential_key TEXT,
    created_at DATETIME DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    updated_at DATETIME DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
```

## Indexes & Constraints

**Current**: No additional indexes (small dataset, single-user desktop application).

**Future Considerations**:
- Index on `driver_type` for driver-specific filtering
- Unique constraint on `name` per user for multi-user scenarios
- Index on `updated_at` for "recently used" queries

## Data Access Patterns

### Connection CRUD
- **Create**: Insert row with generated UUID, store credential via CredManager
- **Read**: Query by `id` or list all; fetch credential via `ConnectionService.GetCredential(id)` which calls `CredManager.Get`
- **Update**: Modify row, optionally update credential in keyring
- **Delete**: Remove row, delete credential from keyring via CredManager

### Credential Operations
- **Store**: `CredManager.Store(key, secret)` → OS keyring → sqlite fallback → in-memory map
- **Retrieve**: `CredManager.Get(key)` → Check keyring first, then sqlite file, then in-memory map
- **Delete**: `CredManager.Delete(key)` → Remove from keyring (best-effort), sqlite fallback, and in-memory map

## Operational Notes

### Backup & Recovery
- **SQLite backup**: Copy `data/connections.db` (contains only metadata + keys)
- **Credential backup**: OS keyring-specific tools required (e.g., macOS Keychain export)
- **Cross-platform migration**: Credentials must be re-entered (keyring format differs)

### Data Retention
- No automatic deletion policy
- No audit trail for CRUD operations (future enhancement)
- Credentials persist in keyring until explicitly deleted

### Security Properties
- No plaintext secrets in `data/connections.db` (metadata only)
- `data/credentials.db` is an additional fallback for environments without OS keyring; stored secrets are unencrypted in this file — access should be restricted via filesystem permissions
- OS-level keyring security (platform-dependent)
- In-memory map is ephemeral — acceptable for development/testing; cleared on restart

## Example Data Flow

```go
// Create connection
id := uuid.New().String()
credKey := "connection:" + id
credManager.Store(credKey, `{"host":"localhost","user":"admin","password":"secret"}`)
db.Exec(`INSERT INTO connections (id, name, driver_type, credential_key)
         VALUES (?, ?, ?, ?)`, id, "My DB", "mysql", credKey)

// Retrieve connection
var conn Connection
db.QueryRow(`SELECT id, name, driver_type, credential_key FROM connections WHERE id = ?`, id).Scan(...)
credential, _ := credManager.Get(conn.CredentialKey)

// Delete connection
credManager.Delete(conn.CredentialKey)
db.Exec(`DELETE FROM connections WHERE id = ?`, id)
```
