package main

import (
	"encoding/json"
	"testing"
)

func TestGetRedisExplicitDB(t *testing.T) {
    makeBlob := func(vals map[string]string) string {
        payload := struct {
            Form   string            `json:"form"`
            Values map[string]string `json:"values"`
        }{Form: "basic", Values: vals}
        b, _ := json.Marshal(payload)
        return string(b)
    }

    tests := []struct {
        name     string
        conn     map[string]string
        wantIdx  int
        wantFlag bool
    }{
        {"none", map[string]string{}, 0, false},
        {"db field", map[string]string{"db": "3"}, 3, true},
        {"blob field", map[string]string{"credential_blob": makeBlob(map[string]string{"db": "7"})}, 7, true},
        {"invalid", map[string]string{"db": "abc"}, 0, false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            gotIdx, gotFlag := getRedisExplicitDB(tt.conn)
            if gotIdx != tt.wantIdx || gotFlag != tt.wantFlag {
                t.Fatalf("got (%d,%v), want (%d,%v)", gotIdx, gotFlag, tt.wantIdx, tt.wantFlag)
            }
        })
    }
}

func TestBuildClientTLS(t *testing.T) {
    // we only check that TLSConfig is set when tls=true
    conn := map[string]string{"credential_blob": "{}"}
    // first without tls
    conn["credential_blob"] = func() string {
        p := struct{ Form string `json:"form"`; Values map[string]string `json:"values"`}{Form: "basic", Values: map[string]string{"host": "127.0.0.1"}}
        b, _ := json.Marshal(p)
        return string(b)
    }()
    cli, err := buildClient(conn)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if cli.Options().TLSConfig != nil {
        t.Errorf("expected nil TLSConfig when not requested")
    }

    // now request TLS
    conn["credential_blob"] = func() string {
        p := struct{ Form string `json:"form"`; Values map[string]string `json:"values"`}{Form: "basic", Values: map[string]string{"host": "127.0.0.1", "tls": "true"}}
        b, _ := json.Marshal(p)
        return string(b)
    }()
    cli, err = buildClient(conn)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if cli.Options().TLSConfig == nil {
        t.Errorf("expected non-nil TLSConfig when tls=true")
    }
}
