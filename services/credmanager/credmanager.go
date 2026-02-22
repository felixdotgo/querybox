package credmanager

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	keyring "github.com/zalando/go-keyring"
	_ "modernc.org/sqlite"
)

// allow tests to override the underlying keyring calls
var (
	keyringSet    = keyring.Set
	keyringGet    = keyring.Get
	keyringDelete = keyring.Delete
)

const (
	serviceName   = "querybox"
	defaultDBDir  = "data"
	defaultDBFile = "credentials.db"
)

// CredManager provides a thin abstraction over the OS keyring. When the OS
// keyring isn't available the manager falls back to a persistent SQLite file
// (and if that cannot be opened, an in-memory map) so the application remains
// usable in headless/test environments.
type CredManager struct {
	mu       sync.RWMutex // guards fallback map
	fallback map[string]string
	// db holds the sqlite connection for persistent fallback storage.  May be
	// nil if initialization failed; code will still operate using the in-memory
	// map in that case.
	db *sql.DB
}

// New constructs a credential manager using the default database path
// (`data/credentials.db`).
func New() *CredManager {
	path := filepath.Join(defaultDBDir, defaultDBFile)
	return NewWithPath(path)
}

// NewWithPath constructs a credential manager that persists fallback secrets
// in the sqlite file at dbPath. If the directory cannot be created or the
// database fails to open the manager will continue operating with an in-memory
// map and will print a warning.  This helper exists primarily for testing.
func NewWithPath(dbPath string) *CredManager {
	c := &CredManager{fallback: make(map[string]string)}
	// ensure parent directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fmt.Printf("warning: unable to create credential db directory: %v\n", err)
		return c
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		fmt.Printf("warning: unable to open credential db: %v\n", err)
		return c
	}
	// keep it simple for a local embedded file
	db.SetMaxOpenConns(1)
	db.SetConnMaxLifetime(0)

	create := `CREATE TABLE IF NOT EXISTS credentials (
		key TEXT PRIMARY KEY,
		secret TEXT NOT NULL
	);`
	if _, err := db.Exec(create); err != nil {
		fmt.Printf("warning: failed to create credentials table: %v\n", err)
		_ = db.Close()
		return c
	}
	c.db = db
	return c
}

// Store saves `secret` under `key`. Prefer the OS keyring and fall back to
// a persistent sqlite file if the keyring call fails. The map is only used if
// the database itself cannot be opened.
func (c *CredManager) Store(key string, secret string) error {
	if key == "" {
		return errors.New("empty key")
	}
	if err := keyringSet(serviceName, key, secret); err == nil {
		return nil
	}
	// keyring unavailable; attempt db fallback
	if c.db != nil {
		_, err := c.db.Exec(`INSERT OR REPLACE INTO credentials (key, secret) VALUES (?, ?)`, key, secret)
		if err == nil {
			return nil
		}
		// fall through to in-memory if db write fails
	}
	// in-memory fallback
	c.mu.Lock()
	c.fallback[key] = secret
	c.mu.Unlock()
	return nil
}

// Get retrieves a secret previously stored with Store. The method prefers the
// OS keyring; if that fails it checks the sqlite fallback and finally the
// in-memory map.
func (c *CredManager) Get(key string) (string, error) {
	if key == "" {
		return "", errors.New("empty key")
	}
	if s, err := keyringGet(serviceName, key); err == nil {
		return s, nil
	}
	if c.db != nil {
		var secret string
		row := c.db.QueryRow(`SELECT secret FROM credentials WHERE key = ?`, key)
		if err := row.Scan(&secret); err == nil {
			return secret, nil
		}
	}
	c.mu.RLock()
	s, ok := c.fallback[key]
	c.mu.RUnlock()
	if ok {
		return s, nil
	}
	return "", errors.New("secret not found")
}

// Delete removes a secret (best-effort). If the OS keyring delete fails we
// attempt to remove it from the sqlite fallback and/or in-memory map.
func (c *CredManager) Delete(key string) error {
	if key == "" {
		return errors.New("empty key")
	}
	_ = keyringDelete(serviceName, key) // ignore error
	if c.db != nil {
		_, _ = c.db.Exec(`DELETE FROM credentials WHERE key = ?`, key)
	}
	c.mu.Lock()
	delete(c.fallback, key)
	c.mu.Unlock()
	return nil
}
// Close shuts down the underlying database if one is open. It is safe to call
// multiple times.
func (c *CredManager) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}
