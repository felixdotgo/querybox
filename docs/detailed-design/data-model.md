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

Credentials are NOT stored in SQLite. Instead, CredManager handles secret storage:

### Primary: OS Keyring (via go-keyring)
- **Service Name**: `"querybox"`
- **Key**: Value from `connections.credential_key` (e.g., `"connection:abc123"`)
- **Platforms**:
  - macOS: Keychain
  - Windows: Credential Manager
  - Linux: Secret Service (GNOME Keyring, KWallet, etc.)

### Fallback: In-Memory Map
- Used when OS keyring unavailable (server/headless/test environments)
- Thread-safe access via `sync.RWMutex`
- Cleared on application restart
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
- **Read**: Query by `id` or list all, fetch credential separately via CredManager
- **Update**: Modify row, optionally update credential in keyring
- **Delete**: Remove row, delete credential from keyring via CredManager

### Credential Operations
- **Store**: `CredManager.Store(key, secret)` → OS keyring or sqlite-file fallback
- **Retrieve**: `CredManager.Get(key)` → Check keyring first, then sqlite, then memory
- **Delete**: `CredManager.Delete(key)` → Remove from keyring, sqlite, and map

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
- No plaintext secrets in SQLite
- No encrypted blobs requiring master key management
- OS-level keyring security (platform-dependent)
- In-memory fallback cleared on restart (acceptable for development/testing)

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
