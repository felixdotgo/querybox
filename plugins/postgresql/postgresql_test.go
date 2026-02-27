package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildConnStringTLS(t *testing.T) {
    makeBlob := func(vals map[string]string) string {
        payload := struct {
            Form   string            `json:"form"`
            Values map[string]string `json:"values"`
        }{Form: "basic", Values: vals}
        b, _ := json.Marshal(payload)
        return string(b)
    }
    conn := map[string]string{"credential_blob": makeBlob(map[string]string{"host": "localhost", "database": "db1", "tls": "require"})}
    dsn, err := buildConnString(conn)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !strings.Contains(dsn, "sslmode=require") {
        t.Errorf("expected sslmode=require in conn string, got %q", dsn)
    }
}
