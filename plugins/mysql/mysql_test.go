package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGetDatabaseFromConn(t *testing.T) {
    makeBlob := func(vals map[string]string) string {
        payload := struct {
            Form   string            `json:"form"`
            Values map[string]string `json:"values"`
        }{Form: "basic", Values: vals}
        b, _ := json.Marshal(payload)
        return string(b)
    }

    tests := []struct {
        name       string
        conn       map[string]string
        wantDB     string
    }{
        {"empty", map[string]string{}, ""},
        {"plain database", map[string]string{"database": "foo"}, "foo"},
        {"blob database", map[string]string{"credential_blob": makeBlob(map[string]string{"database": "bar"})}, "bar"},
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
    conn := map[string]string{"credential_blob": ""}
    // build a blob with host/database and tls parameter
    payload := struct {
        Form   string            `json:"form"`
        Values map[string]string `json:"values"`
    }{Form: "basic", Values: map[string]string{"host": "localhost", "database": "db1", "tls": "true"}}
    b, _ := json.Marshal(payload)
    conn["credential_blob"] = string(b)

    dsn, err := buildDSN(conn)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !strings.Contains(dsn, "tls=true") {
        t.Errorf("expected tls=true in dsn, got %q", dsn)
    }
}
