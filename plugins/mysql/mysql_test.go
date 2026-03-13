package main

import (
	"context"
	"strings"
	"testing"

	"github.com/felixdotgo/querybox/pkg/plugin"
)

func TestGetDatabaseFromConn(t *testing.T) {
    makeBlob := plugin.MakeTestBlob

    tests := []struct {
        name       string
        conn       map[string]string
        wantDB     string
    }{
        {"empty", map[string]string{}, ""},
        {"plain database", map[string]string{"database": "foo"}, ""},
        {"blob database", map[string]string{"credential_blob": makeBlob(map[string]string{"database": "bar"})}, ""},
        {"dsn with name", map[string]string{"dsn": "user:pass@tcp(localhost:3306)/baz"}, "baz"},
        {"blob dsn", map[string]string{"credential_blob": makeBlob(map[string]string{"dsn": "user:pass@tcp(localhost:3306)/qux"})}, "qux"},
        {"no db anywhere", map[string]string{"dsn": "user:pass@tcp(localhost:3306)/"}, ""},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := getDatabaseFromConn(tt.conn)
            if got != tt.wantDB {
                t.Fatalf("got %q, want %q", got, tt.wantDB)
            }
        })
    }
}

func TestBuildDSNTLSParam(t *testing.T) {
    conn := map[string]string{"credential_blob": plugin.MakeTestBlob(map[string]string{"host": "localhost", "database": "db1", "tls": "true"})}

    dsn, err := buildDSN(conn)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !strings.Contains(dsn, "tls=querybox") {
        t.Errorf("expected tls=querybox in dsn, got %q", dsn)
    }
}

func TestDescribeSchemaInvalid(t *testing.T) {
    m := &mysqlPlugin{}
    resp, err := m.DescribeSchema(context.Background(), &plugin.DescribeSchemaRequest{Connection: map[string]string{}})
    if err != nil {
        t.Fatalf("DescribeSchema error: %v", err)
    }
    if len(resp.Tables) != 0 {
        t.Errorf("expected no tables for invalid connection, got %d", len(resp.Tables))
    }
}

func TestBuildDSNDatabaseOverrideWithColon(t *testing.T) {
    // verify that an override containing a colon is used verbatim rather than
    // being mangled by the driver.  this guards against regressions if the
    // frontend ever passes a malformed value; the plugin should simply
    // forward it and allow the database to reject it.
    conn := map[string]string{"dsn": "user:pass@tcp(localhost:3306)/foo"}
    // override to a name containing a colon
    conn["database"] = "employees:employees"
    dsn, err := buildDSN(conn)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    // parsed DSN should end with the override value after the last '/'
    if !strings.Contains(dsn, "/employees:employees") {
        t.Errorf("override not applied, dsn=%q", dsn)
    }
}

func TestMutateRowEmptySource(t *testing.T) {
    m := &mysqlPlugin{}
    resp, err := m.MutateRow(context.Background(), &plugin.MutateRowRequest{
        Source:    "",
        Filter:    "id = 1",
        Operation: 2, // UPDATE
    })
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if resp.Success {
        t.Error("expected success=false for empty source")
    }
    if resp.Error == "" {
        t.Error("expected non-empty error message for empty source")
    }
}

func TestMutateRowEmptyFilter(t *testing.T) {
    m := &mysqlPlugin{}
    resp, err := m.MutateRow(context.Background(), &plugin.MutateRowRequest{
        Source:    "users",
        Filter:    "",
        Operation: 3, // DELETE
    })
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if resp.Success {
        t.Error("expected success=false for empty filter")
    }
    if resp.Error == "" {
        t.Error("expected non-empty error message for empty filter")
    }
}

func TestMutateRowUnsupportedOperation(t *testing.T) {
    m := &mysqlPlugin{}
    resp, err := m.MutateRow(context.Background(), &plugin.MutateRowRequest{
        Source:    "users",
        Filter:    "id = 1",
        Operation: 1, // INSERT – not yet supported
    })
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if resp.Success {
        t.Error("expected success=false for unsupported operation")
    }
    if resp.Error == "" {
        t.Error("expected non-empty error message for unsupported operation")
    }
}

func TestMutateRowUpdateEmptyValues(t *testing.T) {
    m := &mysqlPlugin{}
    resp, err := m.MutateRow(context.Background(), &plugin.MutateRowRequest{
        Source:    "users",
        Filter:    "id = 1",
        Operation: 2, // UPDATE
        Values:    map[string]string{},
    })
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if resp.Success {
        t.Error("expected success=false for UPDATE with no values")
    }
    if resp.Error == "" {
        t.Error("expected non-empty error message for UPDATE with no values")
    }
}
