package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/felixdotgo/querybox/services/credmanager"
	"github.com/google/uuid"
	"github.com/wailsapp/wails/v3/pkg/application"
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

// ConnectionService is the application-facing service that exposes connection
// management APIs to the frontend. The service now embeds the persistence and
// credential-storage logic (previously in connection.ConnectionManager). It is
// safe for concurrent use.
type ConnectionService struct {
	db   *sql.DB
	cred *credmanager.CredManager
	app  *application.App
}

// SetApp injects the Wails application reference so the service can emit
// log events to the frontend. Call this after application.New returns.
func (s *ConnectionService) SetApp(app *application.App) {
	s.app = app
}

// dataDir returns the directory where application data (e.g. the SQLite DB)
// should be stored.  Its behaviour is intentionally simple so callers can
// reason about backups, migrations, and runtime diagnostics.  The path is
// built from whatever `os.UserConfigDir()` reports, joined with the fixed
// subdirectory `querybox`.
//
// Platform specifics:
//   * macOS   → ~/Library/Application Support/querybox (same for dev runs or
//                bundled .app).
//   * Windows → %APPDATA%\querybox (e.g. C:\Users\You\AppData\Roaming\querybox).
//   * Linux   → ${XDG_CONFIG_HOME:-$HOME/.config}/querybox.  This is also the
//                directory used when running an AppImage; the host session
//                determines $XDG_CONFIG_HOME.
//
// If `os.UserConfigDir()` returns an error (which can happen in headless
// containers or when $HOME is unset) we fall back to a simple relative
// "data" directory beneath the current working directory.  That behaviour is
// exercised by unit tests and makes the binary behave sensibly when run from
// a build agent or inside a temporary folder.
//
// The helper is unexported, but its behaviour is recorded in tests so you can
// grep for `dataDir` when you need to know where production data lands.
var userConfigDirFunc = os.UserConfigDir

func dataDir() string {
	if dir, err := userConfigDirFunc(); err == nil {
		return filepath.Join(dir, "querybox")
	}
	return "data"
}

// NewConnectionService constructs a ConnectionService and initializes the
// underlying SQLite database and credential manager. It performs the same
// migrations and schema setup that existed previously in the manager.
func NewConnectionService() *ConnectionService {
	dir := dataDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return &ConnectionService{cred: credmanager.New()}
	}
	dbPath := filepath.Join(dir, "connections.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return &ConnectionService{cred: credmanager.New()}
	}

	// Embedded DB is local — limit connections and lifetime.
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
		_ = db.Close()
		return &ConnectionService{cred: credmanager.New()}
	}

	svc := &ConnectionService{db: db, cred: credmanager.New()}

	// Migration: move any legacy `credential_blob` into the keyring and set
	// `credential_key` to the generated key.
	if has, _ := svc.hasColumn("credential_blob"); has {
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
				_ = svc.cred.Store(key, string(blob))
				_, _ = db.Exec(`UPDATE connections SET credential_key = ? WHERE id = ?`, key, id)
				_, _ = db.Exec(`UPDATE connections SET credential_blob = NULL WHERE id = ?`, id)
			}
		}
	}

	return svc
}

func (s *ConnectionService) closeable() bool { return s.db != nil }

// Shutdown releases resources held by the service. It is invoked by Wails when
// the application is quitting.
func (s *ConnectionService) Shutdown() {
	if s.db != nil {
		_ = s.db.Close()
		s.db = nil
	}
}

// hasColumn reports whether the `connections` table contains a column named
// `col`.
func (s *ConnectionService) hasColumn(col string) (bool, error) {
	if !s.closeable() {
		return false, errors.New("connections database not initialized")
	}
	rows, err := s.db.Query(`PRAGMA table_info(connections)`)
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

// ListConnections returns all stored connections ordered by creation time
// (newest first).
func (s *ConnectionService) ListConnections(ctx context.Context) ([]Connection, error) {
	if !s.closeable() {
		return nil, errors.New("connections database not initialized")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, driver_type, credential_key, created_at, updated_at FROM connections ORDER BY created_at DESC`)
	if err != nil {
		emitLog(s.app, LogLevelError, fmt.Sprintf("ListConnections: query failed: %v", err))
		return nil, fmt.Errorf("query connections: %w", err)
	}
	defer rows.Close()

	var out []Connection
	for rows.Next() {
		var r Connection
		var credKey sql.NullString
		if err := rows.Scan(&r.ID, &r.Name, &r.DriverType, &credKey, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan connections: %w", err)
		}
		if credKey.Valid {
			r.CredentialKey = credKey.String
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate connections: %w", err)
	}
	emitLog(s.app, LogLevelInfo, fmt.Sprintf("ListConnections: found %d connection(s)", len(out)))
	return out, nil
}

// GetConnection retrieves a single connection by id.
func (s *ConnectionService) GetConnection(ctx context.Context, id string) (Connection, error) {
	if id == "" {
		return Connection{}, errors.New("empty database connection id")
	}
	if !s.closeable() {
		return Connection{}, errors.New("connections database not initialized")
	}
	var r Connection
	var credKey sql.NullString
	row := s.db.QueryRowContext(ctx, `SELECT id, name, driver_type, credential_key, created_at, updated_at FROM connections WHERE id = ?`, id)
	if err := row.Scan(&r.ID, &r.Name, &r.DriverType, &credKey, &r.CreatedAt, &r.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Connection{}, fmt.Errorf("database connection not found")
		}
		return Connection{}, fmt.Errorf("scan connections: %w", err)
	}
	if credKey.Valid {
		r.CredentialKey = credKey.String
	}
	return r, nil
}

// CreateConnection inserts a new connection record and returns it. The
// provided `credential` (typically the frontend-serialized auth form) is
// stored in the OS keyring and the DB only keeps the key reference.
func (s *ConnectionService) CreateConnection(ctx context.Context, name, driverType, credential string) (Connection, error) {
	if name == "" || driverType == "" {
		return Connection{}, errors.New("name and driverType are required")
	}
	if !s.closeable() {
		return Connection{}, errors.New("connections database not initialized")
	}
	emitLog(s.app, LogLevelInfo, fmt.Sprintf("CreateConnection: creating '%s' (driver: %s)", name, driverType))
	id := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	key := "connection:" + id
	if err := s.cred.Store(key, credential); err != nil {
		emitLog(s.app, LogLevelError, fmt.Sprintf("CreateConnection: failed to store credential for '%s': %v", name, err))
		return Connection{}, fmt.Errorf("store credential: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, `INSERT INTO connections (id, name, driver_type, credential_key, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`, id, name, driverType, key, now, now); err != nil {
		emitLog(s.app, LogLevelError, fmt.Sprintf("CreateConnection: failed to insert connection '%s': %v", name, err))
		return Connection{}, fmt.Errorf("insert database connection: %w", err)
	}
	emitLog(s.app, LogLevelInfo, fmt.Sprintf("CreateConnection: '%s' created successfully (id: %s)", name, id))
	conn := Connection{
		ID:            id,
		Name:          name,
		DriverType:    driverType,
		CredentialKey: key,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	emitConnectionCreated(s.app, conn)
	return conn, nil
}

// GetCredential retrieves the raw credential blob associated with the
// connection.  This is used by the frontend when it needs to establish a
// plugin connection (e.g. building a tree or executing a query). The value was
// originally supplied when the connection was created and is stored via
// CredManager.  Returning the credential to the caller is considered a
// security-sensitive operation, but the frontend already has full access to a
// saved connection (it can execute arbitrary queries), so this method simply
// fetches and returns whatever string is stored under the connection's key.
func (s *ConnectionService) GetCredential(ctx context.Context, id string) (string, error) {
	if id == "" {
		return "", errors.New("empty id")
	}
	if !s.closeable() {
		return "", errors.New("connections database not initialized")
	}
	emitLog(s.app, LogLevelInfo, fmt.Sprintf("GetCredential: fetching credential for connection '%s'", id))
	conn, err := s.GetConnection(ctx, id)
	if err != nil {
		emitLog(s.app, LogLevelError, fmt.Sprintf("GetCredential: connection '%s' not found: %v", id, err))
		return "", err
	}
	if conn.CredentialKey == "" {
		return "", errors.New("no credential stored")
	}
	cred, err := s.cred.Get(conn.CredentialKey)
	if err != nil {
		emitLog(s.app, LogLevelError, fmt.Sprintf("GetCredential: keyring lookup failed for '%s': %v", id, err))
		return "", fmt.Errorf("fetch credential: %w", err)
	}
	return cred, nil
}

// DeleteConnection removes a connection by id and attempts to remove the
// associated secret from the keyring as a best-effort cleanup.
func (s *ConnectionService) DeleteConnection(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("empty id")
	}
	if !s.closeable() {
		return errors.New("connections database not initialized")
	}
	emitLog(s.app, LogLevelInfo, fmt.Sprintf("DeleteConnection: deleting connection '%s'", id))
	// fetch credential_key (if any) so we can delete the secret from the keyring
	var credKey sql.NullString
	row := s.db.QueryRowContext(ctx, `SELECT credential_key FROM connections WHERE id = ?`, id)
	if err := row.Scan(&credKey); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("lookup database connection before delete: %w", err)
	}
	if credKey.Valid && credKey.String != "" {
		_ = s.cred.Delete(credKey.String) // best-effort
	}
	res, err := s.db.ExecContext(ctx, `DELETE FROM connections WHERE id = ?`, id)
	if err != nil {
		emitLog(s.app, LogLevelError, fmt.Sprintf("DeleteConnection: failed to delete connection '%s': %v", id, err))
		return fmt.Errorf("delete database connection: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		emitLog(s.app, LogLevelWarn, fmt.Sprintf("DeleteConnection: connection '%s' not found", id))
		return fmt.Errorf("database connection not found")
	}
	emitLog(s.app, LogLevelInfo, fmt.Sprintf("DeleteConnection: connection '%s' deleted successfully", id))
	emitConnectionDeleted(s.app, id)
	return nil
}
