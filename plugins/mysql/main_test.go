package main

import (
	"strings"
	"testing"

	"github.com/felixdotgo/querybox/pkg/plugin"
)

func TestBuildDSNEmpty(t *testing.T) {
    dsn, err := buildDSN(nil)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if dsn != "" {
        t.Fatalf("expected empty dsn for nil connection, got %q", dsn)
    }
}

func TestBuildDSNFromBlob(t *testing.T) {
    blob := `{"form":"basic","values":{"host":"127.0.0.1","user":"u","password":"p","port":"3306","database":"db"}}`
    dsn, err := buildDSN(map[string]string{"credential_blob": blob})
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if dsn == "" {
        t.Fatal("expected non-empty dsn")
    }
}

func TestBuildDSNWithParams(t *testing.T) {
    blob := `{"form":"basic","values":{"host":"localhost","user":"u","password":"p","port":"3306","database":"db","tls":"skip-verify","foo":"bar"}}`
    dsn, err := buildDSN(map[string]string{"credential_blob": blob})
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !strings.Contains(dsn, "tls=skip-verify") || !strings.Contains(dsn, "foo=bar") {
        t.Fatalf("expected extra params in dsn, got %q", dsn)
    }
    if !strings.Contains(dsn, "timeout=5s") {
        t.Fatalf("expected default timeout in dsn, got %q", dsn)
    }
}

func TestConnectionTreeNoConnection(t *testing.T) {
    m := &mysqlPlugin{}
    res, err := m.ConnectionTree(plugin.ConnectionTreeRequest{})
    if err != nil {
        t.Fatalf("ConnectionTree returned error: %v", err)
    }
    if len(res.Nodes) != 0 {
        t.Fatalf("expected no nodes, got %d", len(res.Nodes))
    }
}
