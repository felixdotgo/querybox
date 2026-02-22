package main

import (
	"strings"
	"testing"

	"github.com/felixdotgo/querybox/pkg/plugin"
)

func TestBuildConnStringEmpty(t *testing.T) {
    dsn, err := buildConnString(nil)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if dsn != "" {
        t.Fatalf("expected empty dsn for nil connection, got %q", dsn)
    }
}

func TestBuildConnStringWithParams(t *testing.T) {
    blob := `{"form":"basic","values":{"host":"localhost","user":"u","password":"p","port":"5432","database":"db","sslmode":"disable","foo":"bar"}}`
    dsn, err := buildConnString(map[string]string{"credential_blob": blob})
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !strings.Contains(dsn, "foo=bar") || !strings.Contains(dsn, "connect_timeout=5") {
        t.Fatalf("expected extra params and default timeout in dsn, got %q", dsn)
    }
}

func TestBuildConnStringFromBlob(t *testing.T) {
    blob := `{"form":"basic","values":{"host":"127.0.0.1","user":"u","password":"p","port":"5432","database":"db"}}`
    dsn, err := buildConnString(map[string]string{"credential_blob": blob})
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if dsn == "" {
        t.Fatal("expected non-empty connection string")
    }
}

func TestConnectionTreeNoConnection(t *testing.T) {
    p := &postgresqlPlugin{}
    res, err := p.ConnectionTree(&plugin.ConnectionTreeRequest{})
    if err != nil {
        t.Fatalf("ConnectionTree returned error: %v", err)
    }
    if res == nil || len(res.Nodes) != 0 {
        t.Fatalf("expected no nodes, got %+v", res)
    }
}
