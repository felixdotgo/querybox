package connection

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

// Connection represents a persisted connection record.
type Connection struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	DriverType      string `json:"driver_type"`
	CredentialBlob  string `json:"credential_blob"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

// ConnectionManager manages connection metadata persisted in SQLite.
// It is safe for concurrent use.
type ConnectionManager struct {
	db *sql.DB
}

// New creates a ConnectionManager and ensures the database schema exists.
// The database file is stored at `data/connections.db` relative to the working directory.
func New() *ConnectionManager {
	const dbPath = "data/connections.db"
	if err := os.MkdirAll("data", 0o755); err != nil {
		// If directory creation fails, return a manager that will return errors from ops.
		fmt.Printf("warning: unable to create data directory: %v\n", err)
		return &ConnectionManager{}
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		fmt.Printf("warning: unable to open sqlite db: %v\n", err)
		return &ConnectionManager{}
	}

	// Set reasonable connection pool defaults for a local embedded DB.
	db.SetMaxOpenConns(1)
	db.SetConnMaxLifetime(time.Minute * 5)

	create := `CREATE TABLE IF NOT EXISTS connections (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		driver_type TEXT NOT NULL,
		credential_blob BLOB,
		created_at DATETIME DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
		updated_at DATETIME DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
	);`
	if _, err := db.Exec(create); err != nil {
		fmt.Printf("warning: failed to create connections table: %v\n", err)
		_ = db.Close()
		return &ConnectionManager{}
	}

	return &ConnectionManager{db: db}
}

func (c *ConnectionManager) closeable() bool { return c.db != nil }

// List returns all stored connections ordered by creation time (newest first).
func (c *ConnectionManager) List(ctx context.Context) ([]Connection, error) {
	if !c.closeable() {
		return nil, errors.New("database not initialized")
	}
	rows, err := c.db.QueryContext(ctx, `SELECT id, name, driver_type, credential_blob, created_at, updated_at FROM connections ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("query connections: %w", err)
	}
	defer rows.Close()

	var out []Connection
	for rows.Next() {
		var r Connection
		var cred []byte
		if err := rows.Scan(&r.ID, &r.Name, &r.DriverType, &cred, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan connection: %w", err)
		}
		r.CredentialBlob = string(cred)
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
	var cred []byte
	row := c.db.QueryRowContext(ctx, `SELECT id, name, driver_type, credential_blob, created_at, updated_at FROM connections WHERE id = ?`, id)
	if err := row.Scan(&r.ID, &r.Name, &r.DriverType, &cred, &r.CreatedAt, &r.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Connection{}, fmt.Errorf("not found")
		}
		return Connection{}, fmt.Errorf("scan connection: %w", err)
	}
	r.CredentialBlob = string(cred)
	return r, nil
}

// Create inserts a new connection record and returns it.
func (c *ConnectionManager) Create(ctx context.Context, name, driverType, credential string) (Connection, error) {
	if name == "" || driverType == "" {
		return Connection{}, errors.New("name and driverType are required")
	}
	if !c.closeable() {
		return Connection{}, errors.New("database not initialized")
	}
	id := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := c.db.ExecContext(ctx, `INSERT INTO connections (id, name, driver_type, credential_blob, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`, id, name, driverType, []byte(credential), now, now); err != nil {
		return Connection{}, fmt.Errorf("insert connection: %w", err)
	}
	return Connection{
		ID:             id,
		Name:           name,
		DriverType:     driverType,
		CredentialBlob: credential,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

// Delete removes a connection by id.
func (c *ConnectionManager) Delete(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("empty id")
	}
	if !c.closeable() {
		return errors.New("database not initialized")
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
