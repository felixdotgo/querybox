package main

import (
	"context"
	"database/sql"
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
var makeBlob = plugin.MakeTestBlob

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

// Verify that DescribeSchema filters out partition child tables by adding
// a NOT EXISTS clause against pg_inherits.  The mock returns one parent and
// one partition row; only the parent should be observed.
func TestDescribeSchemaFiltersPartitions(t *testing.T) {
    orig := openPostgresDB
    defer func() { openPostgresDB = orig }()

    db, mock, err := sqlmock.New()
    if err != nil {
        t.Fatalf("failed to create mock: %v", err)
    }
    openPostgresDB = func(dsn string) (*sql.DB, error) {
        return db, nil
    }

    // Expect the base query including our filter
    mock.ExpectQuery(`FROM information_schema.tables t[\s\S]*pg_inherits`).WillReturnRows(sqlmock.NewRows([]string{"table_schema", "table_name"}).AddRow("public", "parent"))

    m := &postgresqlPlugin{}
    resp, err := m.DescribeSchema(context.Background(), &plugin.DescribeSchemaRequest{Connection: map[string]string{"dsn": "foo"}})
    if err != nil {
        t.Fatalf("DescribeSchema error: %v", err)
    }
    if len(resp.Tables) != 1 || resp.Tables[0].Name != "public.parent" {
        t.Errorf("expected only parent table, got %+v", resp.Tables)
    }

    if err := mock.ExpectationsWereMet(); err != nil {
        t.Errorf("unmet expectations: %v", err)
    }
}

// TestDescribeSchemaWithSchemaFilter verifies that passing Database= (a
// postgres schema name like "public") appends a $1-style predicate against
// t.table_schema, not table_catalog.
func TestDescribeSchemaWithSchemaFilter(t *testing.T) {
    orig := openPostgresDB
    defer func() { openPostgresDB = orig }()

    db, mock, err := sqlmock.New()
    if err != nil {
        t.Fatalf("failed to create mock: %v", err)
    }
    openPostgresDB = func(dsn string) (*sql.DB, error) { return db, nil }

    // The query must contain "table_schema = $1" – not table_catalog or "?"
    mock.ExpectQuery(`(?i)table_schema\s*=\s*\$1`).
        WithArgs("public").
        WillReturnRows(sqlmock.NewRows([]string{"table_schema", "table_name"}).
            AddRow("public", "users"))
    // column query for "public"."users"
    mock.ExpectQuery(`(?i)information_schema\.columns`).
        WithArgs("public", "users").
        WillReturnRows(sqlmock.NewRows([]string{"column_name", "data_type", "is_nullable", "ordinal_position", "column_default"}).
            AddRow("id", "integer", "NO", 1, nil))
    // index query
    mock.ExpectQuery(`(?i)pg_indexes`).
        WithArgs("public", "users").
        WillReturnRows(sqlmock.NewRows([]string{"indexname", "indexdef"}))

    m := &postgresqlPlugin{}
    resp, err := m.DescribeSchema(context.Background(), &plugin.DescribeSchemaRequest{
        Connection: map[string]string{"dsn": "postgres://localhost/test?sslmode=disable"},
        Database:   "public",
    })
    if err != nil {
        t.Fatalf("DescribeSchema error: %v", err)
    }
    if len(resp.Tables) != 1 || resp.Tables[0].Name != "public.users" {
        t.Errorf("expected public.users, got %+v", resp.Tables)
    }
    if len(resp.Tables[0].Columns) != 1 || resp.Tables[0].Columns[0].Name != "id" {
        t.Errorf("expected column id, got %+v", resp.Tables[0].Columns)
    }
    if err := mock.ExpectationsWereMet(); err != nil {
        t.Errorf("unmet expectations: %v", err)
    }
}

// TestDescribeSchemaWithTableFilter verifies that passing both Database and
// Table appends two numbered $1/$2 predicates.
func TestDescribeSchemaWithTableFilter(t *testing.T) {
    orig := openPostgresDB
    defer func() { openPostgresDB = orig }()

    db, mock, err := sqlmock.New()
    if err != nil {
        t.Fatalf("failed to create mock: %v", err)
    }
    openPostgresDB = func(dsn string) (*sql.DB, error) { return db, nil }

    // Expect both $1 (schema) and $2 (table) predicates
    mock.ExpectQuery(`(?i)table_schema\s*=\s*\$1[\s\S]*table_name\s*=\s*\$2`).
        WithArgs("public", "orders").
        WillReturnRows(sqlmock.NewRows([]string{"table_schema", "table_name"}).
            AddRow("public", "orders"))
    mock.ExpectQuery(`(?i)information_schema\.columns`).
        WithArgs("public", "orders").
        WillReturnRows(sqlmock.NewRows([]string{"column_name", "data_type", "is_nullable", "ordinal_position", "column_default"}))
    mock.ExpectQuery(`(?i)pg_indexes`).
        WithArgs("public", "orders").
        WillReturnRows(sqlmock.NewRows([]string{"indexname", "indexdef"}))

    m := &postgresqlPlugin{}
    resp, err := m.DescribeSchema(context.Background(), &plugin.DescribeSchemaRequest{
        Connection: map[string]string{"dsn": "postgres://localhost/test?sslmode=disable"},
        Database:   "public",
        Table:      "orders",
    })
    if err != nil {
        t.Fatalf("DescribeSchema error: %v", err)
    }
    if len(resp.Tables) != 1 || resp.Tables[0].Name != "public.orders" {
        t.Errorf("expected public.orders, got %+v", resp.Tables)
    }
    if err := mock.ExpectationsWereMet(); err != nil {
        t.Errorf("unmet expectations: %v", err)
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

    // db1 (current db) — 7 queries for schema "public"
    mock.ExpectQuery("SELECT current_database\\(\\)").WillReturnRows(sqlmock.NewRows([]string{"current_database"}).AddRow("db1"))
    mock.ExpectQuery("SELECT datname FROM pg_database").WillReturnRows(sqlmock.NewRows([]string{"datname"}).AddRow("db1").AddRow("db2"))
    // the query contains newlines and filters; allow spaces/newlines before schema_name
    mock.ExpectQuery("(?s)SELECT\\s+schema_name").WillReturnRows(sqlmock.NewRows([]string{"schema_name"}).AddRow("public"))
    mock.ExpectQuery("(?s)relkind IN.*pg_inherits").WithArgs("public").WillReturnRows(sqlmock.NewRows([]string{"relname"}).AddRow("users"))


    // debug: what would our plugin produce for a db2 override?
    tmp := map[string]string{"dsn": "postgres://foo", "database": "db2"}
    if dsn2, err := buildConnString(tmp); err == nil {
        t.Logf("debug dsn2 calculated as %q", dsn2)
    }

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
        t.Fatalf("expected 3 nodes (create + db1 + db2), got %d", len(resp.Nodes))
    }
    if resp.Nodes[1].Label != "db1" {
        t.Errorf("first db label wrong: %s", resp.Nodes[1].Label)
    }
    if resp.Nodes[2].Label != "db2" {
        t.Errorf("second db label wrong: %s", resp.Nodes[2].Label)
    }

    t.Logf("debug seenDSNs: %v", seenDSNs)
    // db1.public should only expose the Tables group (others are disabled)
    if len(resp.Nodes[1].Children) == 0 {
        t.Fatalf("no schemas returned for db1: %+v", resp)
    }
    db1Schema := resp.Nodes[1].Children[0]
    if len(db1Schema.Children) != 1 {
        t.Errorf("db1.public should have 1 category group, got %d", len(db1Schema.Children))
    }
    tablesGroup := db1Schema.Children[0]
    if tablesGroup.Label != "Tables" {
        t.Errorf("first category should be Tables, got %s", tablesGroup.Label)
    }
    if len(tablesGroup.Children) != 1 || tablesGroup.Children[0].Label != "users" {
        t.Errorf("db1 Tables group should contain 'users'")
    }

    // note: we don't require any specific schema or tables for db2 in this
    // test. its presence in resp.Nodes is enough, and earlier debug output will
    // show if any categories were returned.
    if len(resp.Nodes[2].Children) > 0 {
        t.Logf("db2 had schemas: %+v", resp.Nodes[2].Children)
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

    // initial connection — current db is db1, list returns db1+db2, filter=db2
    mock.ExpectQuery("SELECT current_database\\(\\)").WillReturnRows(sqlmock.NewRows([]string{"current_database"}).AddRow("db1"))
    mock.ExpectQuery("SELECT datname FROM pg_database").WillReturnRows(sqlmock.NewRows([]string{"datname"}).AddRow("db1").AddRow("db2"))
    // only db2 gets schema/table queries (filter removes db1, opens new conn for db2)
    mock.ExpectQuery("SELECT schema_name").WillReturnRows(sqlmock.NewRows([]string{"schema_name"}).AddRow("public"))
    mock.ExpectQuery("(?s)relkind IN.*pg_inherits").WithArgs("public").WillReturnRows(sqlmock.NewRows([]string{"relname"}).AddRow("things"))

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

    db2Schema := resp.Nodes[1].Children[0]
    if len(db2Schema.Children) != 1 {
        t.Errorf("db2.public should have 1 category group, got %d", len(db2Schema.Children))
    }
    tablesGroup := db2Schema.Children[0]
    if tablesGroup.Label != "Tables" {
        t.Errorf("first category should be Tables, got %s", tablesGroup.Label)
    }
    if len(tablesGroup.Children) != 1 || tablesGroup.Children[0].Label != "things" {
        t.Errorf("db2 Tables group should contain 'things'")
    }

    if err := mock.ExpectationsWereMet(); err != nil {
        t.Errorf("unmet expectations: %v", err)
    }
}

func TestConnectionTreeSchemaGroups(t *testing.T) {
    // verify override with unusual database name does not get mangled
    if dsn, err := buildConnString(map[string]string{"dsn": "postgres://foo", "database": "phonedb:public:public"}); err == nil {
        if !strings.Contains(dsn, "phonedb:public:public") {
            t.Errorf("colon-containing override was altered: %s", dsn)
        }
    }

    orig := openPostgresDB
    defer func() { openPostgresDB = orig }()

    db, mock, err := sqlmock.New()
    if err != nil {
        t.Fatalf("failed to create mock: %v", err)
    }
    openPostgresDB = func(dsn string) (*sql.DB, error) { return db, nil }

    p := &postgresqlPlugin{}
    ctx := context.Background()

    mock.ExpectQuery("SELECT current_database\\(\\)").WillReturnRows(sqlmock.NewRows([]string{"current_database"}).AddRow("mydb"))
    mock.ExpectQuery("SELECT datname FROM pg_database").WillReturnRows(sqlmock.NewRows([]string{"datname"}).AddRow("mydb"))
    mock.ExpectQuery("SELECT schema_name").WillReturnRows(sqlmock.NewRows([]string{"schema_name"}).AddRow("app"))
    // tables only (other object types are not currently fetched)
    mock.ExpectQuery("(?s)relkind IN.*pg_inherits").WithArgs("app").WillReturnRows(sqlmock.NewRows([]string{"relname"}).AddRow("orders").AddRow("users"))

    resp, err := p.ConnectionTree(ctx, &pluginpb.PluginV1_ConnectionTreeRequest{Connection: map[string]string{"dsn": "postgres://foo"}})
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    // structure: create-node + mydb
    if len(resp.Nodes) != 2 {
        t.Fatalf("expected 2 top-level nodes, got %d", len(resp.Nodes))
    }
    dbNode := resp.Nodes[1]
    if dbNode.Label != "mydb" {
        t.Fatalf("expected mydb, got %s", dbNode.Label)
    }
    if len(dbNode.Children) != 1 {
        t.Fatalf("expected 1 schema, got %d", len(dbNode.Children))
    }
    schemaNode := dbNode.Children[0]
    if schemaNode.Label != "app" {
        t.Errorf("expected schema 'app', got %s", schemaNode.Label)
    }

    // schema node should have no direct create-table action (moved to Tables group)
    for _, a := range schemaNode.Actions {
        if a.Type == plugin.ConnectionTreeActionCreateTable {
            t.Errorf("create-table action should be on Tables group, not schema node")
        }
    }

    // only one category group currently exists
    if len(schemaNode.Children) != 1 {
        t.Fatalf("expected 1 category group, got %d", len(schemaNode.Children))
    }

    tablesGroup := schemaNode.Children[0]
    // Tables group should have create-table action
    hasCreateTable := false
    for _, a := range tablesGroup.Actions {
        if a.Type == plugin.ConnectionTreeActionCreateTable {
            hasCreateTable = true
        }
    }
    if !hasCreateTable {
        t.Errorf("Tables group should have create-table action")
    }
    if len(tablesGroup.Children) != 2 {
        t.Errorf("Tables group should have 2 tables, got %d", len(tablesGroup.Children))
    }

    if err := mock.ExpectationsWereMet(); err != nil {
        t.Errorf("unmet expectations: %v", err)
    }
}

// --- MutateRow tests ---

func TestMutateRowPGMissingSource(t *testing.T) {
    p := &postgresqlPlugin{}
    resp, err := p.MutateRow(context.Background(), &pluginpb.PluginV1_MutateRowRequest{
        Connection: map[string]string{"dsn": "host=localhost sslmode=disable"},
        Operation:  pluginpb.PluginV1_MutateRowRequest_DELETE,
        Filter:     "id = 1",
    })
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if resp.Success {
        t.Error("expected failure when source is empty")
    }
}

func TestMutateRowPGMissingFilter(t *testing.T) {
    p := &postgresqlPlugin{}
    resp, err := p.MutateRow(context.Background(), &pluginpb.PluginV1_MutateRowRequest{
        Connection: map[string]string{"dsn": "host=localhost sslmode=disable"},
        Operation:  pluginpb.PluginV1_MutateRowRequest_DELETE,
        Source:     "users",
    })
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if resp.Success {
        t.Error("expected failure when filter is empty")
    }
}

func TestMutateRowPGUnsupportedOperation(t *testing.T) {
    orig := openPostgresDB
    defer func() { openPostgresDB = orig }()

    db, _, err := sqlmock.New()
    if err != nil {
        t.Fatalf("failed to create mock: %v", err)
    }
    openPostgresDB = func(dsn string) (*sql.DB, error) { return db, nil }

    p := &postgresqlPlugin{}
    resp, err := p.MutateRow(context.Background(), &pluginpb.PluginV1_MutateRowRequest{
        Connection: map[string]string{"dsn": "host=localhost sslmode=disable"},
        Operation:  pluginpb.PluginV1_MutateRowRequest_INSERT,
        Source:     "users",
        Filter:     "id = 1",
    })
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if resp.Success {
        t.Error("expected failure for INSERT (unsupported)")
    }
}

func TestMutateRowPGUpdate(t *testing.T) {
    orig := openPostgresDB
    defer func() { openPostgresDB = orig }()

    db, mock, err := sqlmock.New()
    if err != nil {
        t.Fatalf("failed to create mock: %v", err)
    }
    openPostgresDB = func(dsn string) (*sql.DB, error) { return db, nil }

    // keys are sorted alphabetically: age then name
    mock.ExpectExec(`UPDATE "users" SET "age"=\$1, "name"=\$2 WHERE id = 1`).
        WithArgs("25", "Bob").
        WillReturnResult(sqlmock.NewResult(1, 1))

    p := &postgresqlPlugin{}
    resp, err := p.MutateRow(context.Background(), &pluginpb.PluginV1_MutateRowRequest{
        Connection: map[string]string{"dsn": "host=localhost sslmode=disable"},
        Operation:  pluginpb.PluginV1_MutateRowRequest_UPDATE,
        Source:     "users",
        Values:     map[string]string{"name": "Bob", "age": "25"},
        Filter:     "id = 1",
    })
    if err != nil {
        t.Fatalf("MutateRow error: %v", err)
    }
    if !resp.Success {
        t.Fatalf("expected success, got error: %s", resp.Error)
    }
    if err := mock.ExpectationsWereMet(); err != nil {
        t.Errorf("unmet expectations: %v", err)
    }
}

func TestMutateRowPGDelete(t *testing.T) {
    orig := openPostgresDB
    defer func() { openPostgresDB = orig }()

    db, mock, err := sqlmock.New()
    if err != nil {
        t.Fatalf("failed to create mock: %v", err)
    }
    openPostgresDB = func(dsn string) (*sql.DB, error) { return db, nil }

    mock.ExpectExec(`DELETE FROM "users" WHERE id = 1`).
        WillReturnResult(sqlmock.NewResult(1, 1))

    p := &postgresqlPlugin{}
    resp, err := p.MutateRow(context.Background(), &pluginpb.PluginV1_MutateRowRequest{
        Connection: map[string]string{"dsn": "host=localhost sslmode=disable"},
        Operation:  pluginpb.PluginV1_MutateRowRequest_DELETE,
        Source:     "users",
        Filter:     "id = 1",
    })
    if err != nil {
        t.Fatalf("MutateRow error: %v", err)
    }
    if !resp.Success {
        t.Fatalf("expected success, got error: %s", resp.Error)
    }
    if err := mock.ExpectationsWereMet(); err != nil {
        t.Errorf("unmet expectations: %v", err)
    }
}

func TestQuoteSourcePG(t *testing.T) {
    cases := []struct {
        input string
        want  string
    }{
        {"users", `"users"`},
        {"public.users", `"public"."users"`},
        {`has"quote`, `"has""quote"`},
    }
    for _, c := range cases {
        got := quoteSourcePG(c.input)
        if got != c.want {
            t.Errorf("quoteSourcePG(%q) = %q, want %q", c.input, got, c.want)
        }
    }
}
