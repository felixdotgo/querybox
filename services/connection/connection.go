package connection

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/felixdotgo/querybox/services/credmanager"
	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

// Connection represents a persisted connection record. NOTE: `CredentialKey`
// stores a key (not the secret) that the CredManager uses to fetch the secret
// from the OS keyring.
type Connection struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	DriverType    string `json:"driver_type"`
	CredentialKey string `json:"credential_key"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

// ConnectionManager manages connection metadata persisted in SQLite.
// It is safe for concurrent use.
type ConnectionManager struct {
	db   *sql.DB
	cred *credmanager.CredManager
}

// New creates a ConnectionManager and ensures the database schema exists.
// The database file is stored at `data/connections.db` relative to the working directory.
// Existing installations that previously stored `credential_blob` will be
// migrated: blobs are moved into the OS keyring (or in-memory fallback) and
// replaced by a `credential_key` reference.
func New() *ConnectionManager {
	const dbPath = "data/connections.db"
	if err := os.MkdirAll("data", 0o755); err != nil {
		// If directory creation fails, return a manager that will return errors from ops.
		fmt.Printf("warning: unable to create data directory: %v\n", err)
		return &ConnectionManager{cred: credmanager.New()}
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		fmt.Printf("warning: unable to open sqlite db: %v\n", err)
		return &ConnectionManager{cred: credmanager.New()}
	}

	// Set reasonable connection pool defaults for a local embedded DB.
	db.SetMaxOpenConns(1)
	db.SetConnMaxLifetime(time.Minute * 5)

	create := `CREATE TABLE IF NOT EXISTS connections (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		driver_type TEXT NOT NULL,
		credential_key TEXT,
		created_at DATETIME DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
		updated_at DATETIME DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
	);`
	if _, err := db.Exec(create); err != nil {
		fmt.Printf("warning: failed to create connections table: %v\n", err)
		_ = db.Close()
		return &ConnectionManager{cred: credmanager.New()}
	}

	mgr := &ConnectionManager{db: db, cred: credmanager.New()}

	// Migration: if old column `credential_blob` exists migrate its content into
	// the keyring and populate `credential_key` with a generated key.
	if has, _ := mgr.hasColumn("credential_blob"); has {
		// add the new column in case it wasn't present
		_, _ = db.Exec(`ALTER TABLE connections ADD COLUMN credential_key TEXT`)

		rows, err := db.Query(`SELECT id, credential_blob FROM connections WHERE credential_blob IS NOT NULL AND credential_blob != ''`)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var id string
				var blob []byte
				if err := rows.Scan(&id, &blob); err != nil {
					continue
				}
				key := "connection:" + id
				_ = mgr.cred.Store(key, string(blob))
				_, _ = db.Exec(`UPDATE connections SET credential_key = ? WHERE id = ?`, key, id)
				_, _ = db.Exec(`UPDATE connections SET credential_blob = NULL WHERE id = ?`, id)
			}
		}
	}

	return mgr
}

func (c *ConnectionManager) closeable() bool { return c.db != nil }

// hasColumn reports whether `table` contains a column named `col`.
func (c *ConnectionManager) hasColumn(col string) (bool, error) {
	if !c.closeable() {
		return false, errors.New("database not initialized")
	}
	rows, err := c.db.Query(`PRAGMA table_info(connections)`)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name string
		var ctype string
		var notnull int
		var dflt interface{}
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			continue
		}
		if name == col {
			return true, nil
		}
	}
	return false, nil
}

// List returns all stored connections ordered by creation time (newest first).
func (c *ConnectionManager) List(ctx context.Context) ([]Connection, error) {
	if !c.closeable() {
		return nil, errors.New("database not initialized")
	}
	rows, err := c.db.QueryContext(ctx, `SELECT id, name, driver_type, credential_key, created_at, updated_at FROM connections ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("query connections: %w", err)
	}
	defer rows.Close()

	var out []Connection
	for rows.Next() {
		var r Connection
		var credKey sql.NullString
		if err := rows.Scan(&r.ID, &r.Name, &r.DriverType, &credKey, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan connection: %w", err)
		}
		if credKey.Valid {
			r.CredentialKey = credKey.String
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate connections: %w", err)
	}
	return out, nil
}

// Get retrieves a single connection by id.
func (c *ConnectionManager) Get(ctx context.Context, id string) (Connection, error) {
	if id == "" {
		return Connection{}, errors.New("empty id")
	}
	if !c.closeable() {
		return Connection{}, errors.New("database not initialized")
	}
	var r Connection
	var credKey sql.NullString
	row := c.db.QueryRowContext(ctx, `SELECT id, name, driver_type, credential_key, created_at, updated_at FROM connections WHERE id = ?`, id)
	if err := row.Scan(&r.ID, &r.Name, &r.DriverType, &credKey, &r.CreatedAt, &r.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Connection{}, fmt.Errorf("not found")
		}
		return Connection{}, fmt.Errorf("scan connection: %w", err)
	}
	if credKey.Valid {
		r.CredentialKey = credKey.String
	}
	return r, nil
}

// Create inserts a new connection record and returns it.
// The provided `credential` (typically the frontend-serialized auth form) is
// stored in the OS keyring and the DB only keeps the key reference.
func (c *ConnectionManager) Create(ctx context.Context, name, driverType, credential string) (Connection, error) {
	if name == "" || driverType == "" {
		return Connection{}, errors.New("name and driverType are required")
	}
	if !c.closeable() {
		return Connection{}, errors.New("database not initialized")
	}
	id := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	key := "connection:" + id
	if err := c.cred.Store(key, credential); err != nil {
		return Connection{}, fmt.Errorf("store credential: %w", err)
	}
	if _, err := c.db.ExecContext(ctx, `INSERT INTO connections (id, name, driver_type, credential_key, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`, id, name, driverType, key, now, now); err != nil {
		return Connection{}, fmt.Errorf("insert connection: %w", err)
	}
	return Connection{
		ID:            id,
		Name:          name,
		DriverType:    driverType,
		CredentialKey: key,
		CreatedAt:     now,
		UpdatedAt:     now,
	}, nil
}

// Delete removes a connection by id and attempts to remove the associated
// secret from the keyring as a best-effort cleanup.
func (c *ConnectionManager) Delete(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("empty id")
	}
	if !c.closeable() {
		return errors.New("database not initialized")
	}
	// fetch credential_key (if any) so we can delete the secret from the keyring
	var credKey sql.NullString
	row := c.db.QueryRowContext(ctx, `SELECT credential_key FROM connections WHERE id = ?`, id)
	if err := row.Scan(&credKey); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("lookup connection before delete: %w", err)
	}
	if credKey.Valid && credKey.String != "" {
		_ = c.cred.Delete(credKey.String) // best-effort
	}
	res, err := c.db.ExecContext(ctx, `DELETE FROM connections WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete connection: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("not found")
	}
	return nil
}
