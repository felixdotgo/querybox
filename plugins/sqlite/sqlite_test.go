package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"testing"

	pluginpb "github.com/felixdotgo/querybox/rpc/contracts/plugin/v1"
	_ "modernc.org/sqlite"
)

// helper that creates a temporary sqlite database with a single table.
func prepareDB(t *testing.T) (string, func()) {
    t.Helper()
    f, err := os.CreateTemp("", "qbtest-*.db")
    if err != nil {
        t.Fatalf("create temp file: %v", err)
    }
    fname := f.Name()
    f.Close()

    db, err := sql.Open("sqlite", fname)
    if err != nil {
        t.Fatalf("open db: %v", err)
    }
    defer db.Close()

    _, err = db.Exec(`CREATE TABLE users (
        id INTEGER PRIMARY KEY,
        name TEXT NOT NULL,
        age INTEGER
    );`)
    if err != nil {
        t.Fatalf("create table: %v", err)
    }

    cleanup := func() { os.Remove(fname) }
    return fname, cleanup
}

func TestDescribeSchema(t *testing.T) {
    fname, cleanup := prepareDB(t)
    defer cleanup()

    // build a connection map using the basic form
    conn := map[string]string{"credential_blob": ``}
    payload := struct {
        Form   string            `json:"form"`
        Values map[string]string `json:"values"`
    }{Form: "basic", Values: map[string]string{"file": fname}}
    b, _ := json.Marshal(payload)
    conn["credential_blob"] = string(b)

    plugin := &sqlitePlugin{}
    resp, err := plugin.DescribeSchema(context.Background(), &pluginpb.PluginV1_DescribeSchemaRequest{
        Connection: conn,
    })
    if err != nil {
        t.Fatalf("DescribeSchema returned error: %v", err)
    }
    if len(resp.GetTables()) != 1 {
        t.Fatalf("expected one table, got %d", len(resp.GetTables()))
    }
    tbl := resp.GetTables()[0]
    if tbl.GetName() != "users" {
        t.Errorf("unexpected table name %q", tbl.GetName())
    }
    cols := tbl.GetColumns()
    if len(cols) != 3 {
        t.Errorf("expected 3 columns, got %d", len(cols))
    }
    // find id column
    var idCol *pluginpb.PluginV1_ColumnSchema
    for _, c := range cols {
        if c.GetName() == "id" {
            idCol = c
            break
        }
    }
    if idCol == nil {
        t.Fatalf("id column missing")
    }
    if !idCol.GetPrimaryKey() {
        t.Errorf("id column should be marked primary")
    }
}

func TestMutateRowStub(t *testing.T) {
    plugin := &sqlitePlugin{}
    req := &pluginpb.PluginV1_MutateRowRequest{
        Connection: map[string]string{},
        Operation:  pluginpb.PluginV1_MutateRowRequest_DELETE,
        Source:     "users",
        Values:     map[string]string{"id": "1"},
        Filter:     "id=1",
    }
    resp, err := plugin.MutateRow(context.Background(), req)
    if err != nil {
        t.Fatalf("MutateRow error: %v", err)
    }
    if !resp.Success {
        t.Errorf("expected success response, got %+v", resp)
    }
}
