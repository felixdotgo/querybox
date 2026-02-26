# Feature: Credential Management

## Overview

`CredManager` (`services/credmanager/credmanager.go`) manages secret storage with a 3-tier fallback chain. It is used exclusively by `ConnectionService` — the frontend never calls CredManager directly.

---

## Storage Tiers

```
Store(key, secret)
  1. OS Keyring (go-keyring)            → keyring.Set("querybox", key, secret)
     If fails (keyring unavailable):
  2. SQLite file (data/credentials.db)  → INSERT OR REPLACE INTO credentials
     If fails (file cannot open):
  3. In-memory map (sync.RWMutex)       → map[key]secret  ← ephemeral, cleared on restart

Get(key) / Delete(key) follow the same cascade.
```

| Tier | Backend | Persistent | Availability |
|------|---------|-----------|-------------|
| 1 | OS Keyring via `go-keyring` | ✓ | macOS Keychain, Windows Credential Manager, Linux Secret Service |
| 2 | `data/credentials.db` (SQLite) | ✓ | Server/CI/headless — OS keyring not available |
| 3 | In-memory `sync.RWMutex` map | ✗ | Last resort; dev/testing only |

---

## API

```go
type CredManager interface {
    Store(key, secret string) error
    Get(key string) (string, error)
    Delete(key string) error
}
```

- **key format**: `"connection:<uuid>"` (set by `ConnectionService` on creation).
- **secret**: credential JSON string (opaque to CredManager).
- Concurrent-safe via `sync.RWMutex` for the in-memory tier; SQLite uses `database/sql` which is goroutine-safe.

---

## Integration Points

| Caller | Operation | When |
|--------|-----------|------|
| `ConnectionService.CreateConnection` | `Store` | After DB insert |
| `ConnectionService.GetCredential` | `Get` | On frontend request before plugin exec |
| `ConnectionService.DeleteConnection` | `Delete` | Before DB row removal |

---

## Implementation Notes

- CredManager stops at the first tier that succeeds; it does not re-sync across tiers if a higher tier later becomes available.
- Delete is best-effort on tier 1 (keyring errors are logged but not fatal).
- `data/credentials.db` secrets are stored unencrypted — restrict filesystem permissions in production.
- In-memory fallback is acceptable for CI test runs where neither keyring nor writable disk is available.
