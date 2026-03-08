package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/felixdotgo/querybox/pkg/certs"
	"github.com/felixdotgo/querybox/pkg/plugin"
	pluginpb "github.com/felixdotgo/querybox/rpc/contracts/plugin/v1"
)

func TestBuildConnStringTLS(t *testing.T) {
    conn := map[string]string{"credential_blob": makeBlob(map[string]string{"host": "localhost", "database": "db1", "tls": "require"})}
    dsn, err := buildConnString(conn)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !strings.Contains(dsn, "sslmode=require") {
        t.Errorf("expected sslmode=require in conn string, got %q", dsn)
    }
}

func TestBuildConnStringDisable(t *testing.T) {
    conn := map[string]string{"credential_blob": makeBlob(map[string]string{"host": "localhost", "database": "db1", "tls": "disable"})}
    dsn, err := buildConnString(conn)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !strings.Contains(dsn, "sslmode=disable") {
        t.Errorf("expected sslmode=disable in conn string, got %q", dsn)
    }
}

func TestBuildConnStringDefaultDisable(t *testing.T) {
    // tls field missing should still default to disable
    conn := map[string]string{"credential_blob": makeBlob(map[string]string{"host": "localhost", "database": "db1"})}
    dsn, err := buildConnString(conn)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !strings.Contains(dsn, "sslmode=disable") {
        t.Errorf("expected default sslmode=disable in conn string, got %q", dsn)
    }
}

// verify that leaving the database name blank doesn't cause the sslmode
// token to be parsed as the database name (user-reported bug).
func TestBuildConnStringEmptyDatabase(t *testing.T) {
    conn := map[string]string{"credential_blob": makeBlob(map[string]string{"host": "localhost", "tls": "disable"})}
    dsn, err := buildConnString(conn)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if strings.Contains(dsn, "dbname=") {
        t.Errorf("expected no dbname parameter when database blank, got %q", dsn)
    }
    if !strings.Contains(dsn, "sslmode=disable") {
        t.Errorf("expected sslmode=disable in conn string, got %q", dsn)
    }
}

func TestBuildConnStringBlobDSN(t *testing.T) {
    // user provided a DSN inside credential_blob without sslmode
    raw := "postgres://user@localhost/db"
    conn := map[string]string{"credential_blob": makeBlob(map[string]string{"dsn": raw})}
    dsn, err := buildConnString(conn)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !strings.Contains(dsn, "sslmode=disable") {
        t.Errorf("expected sslmode=disable added to blob DSN, got %q", dsn)
    }
}

// Helpers for constructing blobs used across multiple tests.
func makeBlob(vals map[string]string) string {
    payload := struct {
        Form   string            `json:"form"`
        Values map[string]string `json:"values"`
    }{Form: "basic", Values: vals}
    b, _ := json.Marshal(payload)
    return string(b)
}

func TestEnsureSSLModeDefaults(t *testing.T) {
    // keyword style without sslmode should get disable appended
    raw := "host=foo port=5432 user=bar"
    got := ensureSSLMode(raw)
    if !strings.Contains(got, "sslmode=disable") {
        t.Errorf("keyword DSN missing default sslmode: %s", got)
    }

    // url style should also receive param
    rawURL := "postgres://user@localhost/dbname"
    gotURL := ensureSSLMode(rawURL)
    if !strings.Contains(gotURL, "sslmode=disable") {
        t.Errorf("url DSN missing default sslmode: %s", gotURL)
    }
}

func TestEnsureSSLModePreserve(t *testing.T) {
    with := "host=foo sslmode=require"
    if ensureSSLMode(with) != with {
        t.Errorf("explicit sslmode modified: %s", ensureSSLMode(with))
    }
    urlWith := "postgres://foo@bar/baz?sslmode=verify-full"
    out := ensureSSLMode(urlWith)
    if !strings.Contains(out, "sslmode=verify-full") {
        t.Errorf("sslmode was altered for url: %s", out)
    }
}

func TestEnsureSSLModeRootCert(t *testing.T) {
    // ensure we actually can create a certificate file first; if bundle
    // fails to load we'll skip the remainder.
    path, err := certs.RootCertPath()
    if err != nil || path == "" {
        t.Skipf("cannot write root cert file: %v", err)
    }

    // keyword form verify-full should get sslrootcert appended
    out := ensureSSLMode("host=foo sslmode=verify-full")
    if !strings.Contains(out, "sslrootcert=") {
        t.Errorf("expected sslrootcert, got %s", out)
    }

    // URL form verify-ca also should gain root cert
    out2 := ensureSSLMode("postgres://foo@bar/baz?sslmode=verify-ca")
    t.Logf("ensureSSLMode output for url: %s", out2)
    if !strings.Contains(out2, "sslrootcert=") {
        t.Errorf("expected sslrootcert in url, got %s", out2)
    }
}

func TestBuildConnStringVerifyCert(t *testing.T) {
    conn := map[string]string{"credential_blob": makeBlob(map[string]string{"host": "localhost", "database": "db1", "tls": "verify-full"})}
    dsn, err := buildConnString(conn)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !strings.Contains(dsn, "sslrootcert=") {
        t.Errorf("expected sslrootcert in constructed dsn, got %q", dsn)
    }
}

func TestBuildConnStringDirectDSN(t *testing.T) {
    conn := map[string]string{"dsn": "host=foo sslmode=verify-full"}
    dsn, err := buildConnString(conn)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !strings.Contains(dsn, "sslrootcert=") {
        t.Errorf("expected sslrootcert appended to direct dsn, got %q", dsn)
    }
}

func TestBuildConnStringDSNDatabaseOverride(t *testing.T) {
    // keyword-style DSN should have its dbname replaced
    conn := map[string]string{"dsn": "host=foo dbname=orig sslmode=disable", "database": "newdb"}
    dsn, err := buildConnString(conn)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !strings.Contains(dsn, "dbname=newdb") || strings.Contains(dsn, "dbname=orig") {
        t.Errorf("expected override to newdb got %q", dsn)
    }

    // URL-style DSN should update the path and/or query param
    conn2 := map[string]string{"dsn": "postgres://user@localhost/orig?sslmode=disable", "database": "other"}
    dsn2, err := buildConnString(conn2)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !strings.Contains(dsn2, "/other") && !strings.Contains(dsn2, "dbname=other") {
        t.Errorf("expected url override to other got %q", dsn2)
    }
}

func TestBuildConnStringBlobDatabaseOverride(t *testing.T) {
    // ConnectionTree injects connection["database"] = dbname when opening
    // each non-current database.  buildConnString must honour this override
    // even when the credentials are carried in credential_blob (the common
    // path for QueryBox connections).

    // Case 1: separate-fields blob + database override
    blobConn := map[string]string{
        "credential_blob": makeBlob(map[string]string{"host": "localhost", "database": "original", "tls": "disable"}),
        "database":        "overridden",
    }
    dsn, err := buildConnString(blobConn)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !strings.Contains(dsn, "dbname=overridden") {
        t.Errorf("separate-fields blob: expected dbname=overridden, got %q", dsn)
    }
    if strings.Contains(dsn, "dbname=original") {
        t.Errorf("separate-fields blob: original dbname should be replaced, got %q", dsn)
    }

    // Case 2: blob carrying an embedded DSN + database override
    blobDSNConn := map[string]string{
        "credential_blob": makeBlob(map[string]string{"dsn": "postgres://user@localhost/original?sslmode=disable"}),
        "database":        "overridden",
    }
    dsn2, err := buildConnString(blobDSNConn)
    if err != nil {
        t.Fatalf("unexpected error (dsn blob): %v", err)
    }
    if !strings.Contains(dsn2, "overridden") {
        t.Errorf("blob-DSN: expected overridden database in result, got %q", dsn2)
    }
    if strings.Contains(dsn2, "/original") {
        t.Errorf("blob-DSN: original database path should be replaced, got %q", dsn2)
    }
}

func TestDSNTLSOverride(t *testing.T) {
    // DSN specifies require but TLS field disables it
    conn := map[string]string{"dsn": "host=foo sslmode=require", "tls": "disable"}
    dsn, err := buildConnString(conn)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if strings.Contains(dsn, "sslmode=require") {
        t.Errorf("expected require removed, got %q", dsn)
    }
    if !strings.Contains(dsn, "sslmode=disable") {
        t.Errorf("expected disable applied, got %q", dsn)
    }
}

func TestFormatPingError(t *testing.T) {
    err := fmt.Errorf("SSL is not enabled on the server")
    msg := formatPingError(err)
    if !strings.Contains(msg, "hint:") {
        t.Errorf("expected hint in message, got %q", msg)
    }
}

func TestDescribeSchemaInvalid(t *testing.T) {
    m := &postgresqlPlugin{}
    resp, err := m.DescribeSchema(context.Background(), &plugin.DescribeSchemaRequest{Connection: map[string]string{}})
    if err != nil {
        t.Fatalf("DescribeSchema error: %v", err)
    }
    if len(resp.Tables) != 0 {
        t.Errorf("expected no tables for invalid connection, got %d", len(resp.Tables))
    }
}

func TestGetDatabaseFromConn(t *testing.T) {
    // explicit field
    if got := getDatabaseFromConn(map[string]string{"database": "foo"}); got != "foo" {
        t.Errorf("expected foo, got %s", got)
    }
    // credential blob
    blob := makeBlob(map[string]string{"database": "bar"})
    if got := getDatabaseFromConn(map[string]string{"credential_blob": blob}); got != "bar" {
        t.Errorf("expected bar, got %s", got)
    }
    // keyword DSN
    if got := getDatabaseFromConn(map[string]string{"dsn": "host=localhost dbname=baz"}); got != "baz" {
        t.Errorf("expected baz, got %s", got)
    }
    // URL DSN
    if got := getDatabaseFromConn(map[string]string{"dsn": "postgres://user@localhost/qux"}); got != "qux" {
        t.Errorf("expected qux, got %s", got)
    }
}

func TestConnectionTreeListsDatabases(t *testing.T) {
    orig := openPostgresDB
    defer func() { openPostgresDB = orig }()

    db, mock, err := sqlmock.New()
    if err != nil {
        t.Fatalf("failed to create mock: %v", err)
    }
    var seenDSNs []string
    openPostgresDB = func(dsn string) (*sql.DB, error) {
        seenDSNs = append(seenDSNs, dsn)
        return db, nil
    }

    p := &postgresqlPlugin{}
    ctx := context.Background()

    // expectations for initial connection
    mock.ExpectQuery("SELECT current_database\\(\\)").WillReturnRows(sqlmock.NewRows([]string{"current_database"}).AddRow("db1"))
    mock.ExpectQuery("SELECT datname FROM pg_database").WillReturnRows(sqlmock.NewRows([]string{"datname"}).AddRow("db1").AddRow("db2"))
    mock.ExpectQuery("SELECT schema_name").WillReturnRows(sqlmock.NewRows([]string{"schema_name"}).AddRow("public"))
    mock.ExpectQuery("SELECT\\s+c\\.relname").WithArgs("public").WillReturnRows(sqlmock.NewRows([]string{"relname", "type"}).AddRow("users", "table"))
    // second database schemas/tables
    mock.ExpectQuery("SELECT schema_name").WillReturnRows(sqlmock.NewRows([]string{"schema_name"}).AddRow("public"))
    mock.ExpectQuery("SELECT\\s+c\\.relname").WithArgs("public").WillReturnRows(sqlmock.NewRows([]string{"relname", "type"}).AddRow("items", "table"))

    resp, err := p.ConnectionTree(ctx, &pluginpb.PluginV1_ConnectionTreeRequest{Connection: map[string]string{"dsn": "postgres://foo"}})
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    if len(seenDSNs) != 2 {
        t.Fatalf("expected 2 open calls, got %d", len(seenDSNs))
    }
    if !strings.Contains(seenDSNs[1], "db2") {
        t.Errorf("second dsn should reference db2, got %q", seenDSNs[1])
    }

    if len(resp.Nodes) != 3 {
        t.Fatalf("expected 3 nodes, got %d", len(resp.Nodes))
    }
    if resp.Nodes[1].Label != "db1" {
        t.Errorf("first db label wrong: %s", resp.Nodes[1].Label)
    }
    if resp.Nodes[2].Label != "db2" {
        t.Errorf("second db label wrong: %s", resp.Nodes[2].Label)
    }
    if len(resp.Nodes[1].Children) != 1 {
        t.Errorf("db1 should have 1 schema")
    }
    if len(resp.Nodes[1].Children[0].Children) != 1 {
        t.Errorf("db1.public should have 1 table")
    }
    if resp.Nodes[1].Children[0].Children[0].Label != "users" {
        t.Errorf("db1 table name mismatch")
    }
    if len(resp.Nodes[2].Children) != 1 {
        t.Errorf("db2 should have 1 schema")
    }
    if resp.Nodes[2].Children[0].Children[0].Label != "items" {
        t.Errorf("db2 table name mismatch")
    }

    if err := mock.ExpectationsWereMet(); err != nil {
        t.Errorf("unmet expectations: %v", err)
    }
}

func TestConnectionTreeFilterDatabase(t *testing.T) {
    orig := openPostgresDB
    defer func() { openPostgresDB = orig }()

    db, mock, err := sqlmock.New()
    if err != nil {
        t.Fatalf("failed to create mock: %v", err)
    }
    var seenDSNs []string
    openPostgresDB = func(dsn string) (*sql.DB, error) {
        seenDSNs = append(seenDSNs, dsn)
        return db, nil
    }

    p := &postgresqlPlugin{}
    ctx := context.Background()

    // initial expectations: current db and list
    mock.ExpectQuery("SELECT current_database\\(\\)").WillReturnRows(sqlmock.NewRows([]string{"current_database"}).AddRow("db1"))
    mock.ExpectQuery("SELECT datname FROM pg_database").WillReturnRows(sqlmock.NewRows([]string{"datname"}).AddRow("db1").AddRow("db2"))
    // only schema/table queries for db2 because filter should remove db1
    mock.ExpectQuery("SELECT schema_name").WillReturnRows(sqlmock.NewRows([]string{"schema_name"}).AddRow("public"))
    mock.ExpectQuery("SELECT\\s+c\\.relname").WithArgs("public").WillReturnRows(sqlmock.NewRows([]string{"relname", "type"}).AddRow("things", "table"))

    resp, err := p.ConnectionTree(ctx, &pluginpb.PluginV1_ConnectionTreeRequest{Connection: map[string]string{"dsn": "postgres://foo", "database": "db2"}})
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    if len(seenDSNs) != 2 {
        t.Fatalf("expected 2 open calls, got %d", len(seenDSNs))
    }
    if !strings.Contains(seenDSNs[1], "db2") {
        t.Errorf("expected override for db2 in second dsn, got %q", seenDSNs[1])
    }

    if len(resp.Nodes) != 2 {
        t.Fatalf("expected 2 nodes (create + db2), got %d", len(resp.Nodes))
    }
    if resp.Nodes[1].Label != "db2" {
        t.Errorf("expected only db2 node, got %s", resp.Nodes[1].Label)
    }
    if len(resp.Nodes[1].Children) != 1 {
        t.Errorf("db2 should have 1 schema")
    }
    if resp.Nodes[1].Children[0].Children[0].Label != "things" {
        t.Errorf("db2 table name mismatch")
    }

    if err := mock.ExpectationsWereMet(); err != nil {
        t.Errorf("unmet expectations: %v", err)
    }
}
