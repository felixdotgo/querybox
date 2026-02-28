package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/felixdotgo/querybox/pkg/certs"
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
