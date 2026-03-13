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

// makeConn builds the connection map that MutateRow / DescribeSchema expect.
func makeConn(t *testing.T, fname string) map[string]string {
    t.Helper()
    payload := struct {
        Form   string            `json:"form"`
        Values map[string]string `json:"values"`
    }{Form: "basic", Values: map[string]string{"file": fname}}
    b, _ := json.Marshal(payload)
    return map[string]string{"credential_blob": string(b)}
}

func TestMutateRowUpdate(t *testing.T) {
    fname, cleanup := prepareDB(t)
    defer cleanup()

    // seed a row
    db, err := sql.Open("sqlite", fname)
    if err != nil {
        t.Fatalf("open db: %v", err)
    }
    if _, err := db.Exec(`INSERT INTO users(id, name, age) VALUES (1, 'Alice', 30)`); err != nil {
        db.Close()
        t.Fatalf("seed: %v", err)
    }
    db.Close()

    p := &sqlitePlugin{}
    resp, err := p.MutateRow(context.Background(), &pluginpb.PluginV1_MutateRowRequest{
        Connection: makeConn(t, fname),
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

    // verify the change persisted
    db2, _ := sql.Open("sqlite", fname)
    defer db2.Close()
    var name string
    var age int
    if err := db2.QueryRow(`SELECT name, age FROM users WHERE id = 1`).Scan(&name, &age); err != nil {
        t.Fatalf("select: %v", err)
    }
    if name != "Bob" || age != 25 {
        t.Errorf("expected Bob/25, got %s/%d", name, age)
    }
}

func TestMutateRowDelete(t *testing.T) {
    fname, cleanup := prepareDB(t)
    defer cleanup()

    // seed two rows
    db, err := sql.Open("sqlite", fname)
    if err != nil {
        t.Fatalf("open db: %v", err)
    }
    if _, err := db.Exec(`INSERT INTO users(id, name, age) VALUES (1, 'Alice', 30), (2, 'Bob', 25)`); err != nil {
        db.Close()
        t.Fatalf("seed: %v", err)
    }
    db.Close()

    p := &sqlitePlugin{}
    resp, err := p.MutateRow(context.Background(), &pluginpb.PluginV1_MutateRowRequest{
        Connection: makeConn(t, fname),
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

    // verify only one row remains
    db2, _ := sql.Open("sqlite", fname)
    defer db2.Close()
    var count int
    if err := db2.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
        t.Fatalf("count: %v", err)
    }
    if count != 1 {
        t.Errorf("expected 1 row after delete, got %d", count)
    }
}

func TestMutateRowMissingSource(t *testing.T) {
    p := &sqlitePlugin{}
    resp, err := p.MutateRow(context.Background(), &pluginpb.PluginV1_MutateRowRequest{
        Connection: map[string]string{},
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

func TestMutateRowMissingFilter(t *testing.T) {
    p := &sqlitePlugin{}
    resp, err := p.MutateRow(context.Background(), &pluginpb.PluginV1_MutateRowRequest{
        Connection: map[string]string{},
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
