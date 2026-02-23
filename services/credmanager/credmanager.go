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
	probeKey      = "__availability_probe__"
	probeValue    = "__probe__"
	defaultDBDir  = "data"
	defaultDBFile = "credentials.db"
)

// CredManager provides a credential store backed by the OS keyring when
// available (Keychain on macOS, Credential Manager on Windows, libsecret /
// KWallet on Linux). When the keyring is not usable – headless servers,
// containers, CI environments – it falls back to a persistent SQLite file,
// and finally to an in-memory map if even the database cannot be opened.
type CredManager struct {
	useKeyring bool
	mu         sync.RWMutex // guards fallback map
	fallback   map[string]string
	// db holds the sqlite connection for persistent fallback storage. Only
	// opened when the keyring probe fails. May be nil if initialisation
	// failed; operations fall through to the in-memory map in that case.
	db *sql.DB
}

// probeKeyring checks whether the OS keyring daemon / service is actually
// reachable by writing, reading, and deleting a sentinel key. It uses the
// same function variables as the rest of the package so that tests can inject
// fakes.
func probeKeyring() bool {
	if err := keyringSet(serviceName, probeKey, probeValue); err != nil {
		fmt.Printf("warning: OS keyring probe failed: %v\n", err)
		return false
	}
	_, err := keyringGet(serviceName, probeKey)
	if err != nil {
		fmt.Printf("warning: OS keyring probe failed: %v\n", err)
	}
	_ = keyringDelete(serviceName, probeKey)
	return err == nil
}

// New constructs a credential manager using the default database path
// (`data/credentials.db`).
func New() *CredManager {
	path := filepath.Join(defaultDBDir, defaultDBFile)
	return NewWithPath(path)
}

// NewWithPath constructs a credential manager. If the OS keyring probe
// succeeds the manager uses the keyring exclusively and the SQLite file is
// never opened. If the probe fails the manager operates entirely through
// SQLite (or in-memory if the database cannot be opened either).
func NewWithPath(dbPath string) *CredManager {
	c := &CredManager{fallback: make(map[string]string)}

	if probeKeyring() {
		c.useKeyring = true
		return c
	}

	fmt.Printf("warning: OS keyring unavailable, falling back to SQLite at %s\n", dbPath)

	// Keyring unavailable – initialise the SQLite fallback.
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

// Store saves secret under key. Uses the OS keyring when available, otherwise
// the SQLite fallback, and finally the in-memory map.
func (c *CredManager) Store(key string, secret string) error {
	if key == "" {
		return errors.New("empty key")
	}
	if c.useKeyring {
		return keyringSet(serviceName, key, secret)
	}
	if c.db != nil {
		_, err := c.db.Exec(`INSERT OR REPLACE INTO credentials (key, secret) VALUES (?, ?)`, key, secret)
		if err == nil {
			return nil
		}
		// fall through to in-memory if db write fails
	}
	c.mu.Lock()
	c.fallback[key] = secret
	c.mu.Unlock()
	return nil
}

// Get retrieves a secret previously stored with Store.
func (c *CredManager) Get(key string) (string, error) {
	if key == "" {
		return "", errors.New("empty key")
	}
	if c.useKeyring {
		return keyringGet(serviceName, key)
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

// Delete removes a secret. Only the active backend is consulted.
func (c *CredManager) Delete(key string) error {
	if key == "" {
		return errors.New("empty key")
	}
	if c.useKeyring {
		return keyringDelete(serviceName, key)
	}
	if c.db != nil {
		_, _ = c.db.Exec(`DELETE FROM credentials WHERE key = ?`, key)
	}
	c.mu.Lock()
	delete(c.fallback, key)
	c.mu.Unlock()
	return nil
}

// Backend returns a human-readable label for the active credential backend.
// Useful for logging and diagnostics.
func (c *CredManager) Backend() string {
	if c.useKeyring {
		return "keyring"
	}
	if c.db != nil {
		return "sqlite"
	}
	return "memory"
}

// Close shuts down the underlying database if one is open. It is safe to call
// multiple times.
func (c *CredManager) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}
