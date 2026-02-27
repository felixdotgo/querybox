package main

import (
	"encoding/json"
	"testing"
)

func TestExplicitDatabase(t *testing.T) {
    makeBlob := func(vals map[string]string) string {
        payload := struct {
            Form   string            `json:"form"`
            Values map[string]string `json:"values"`
        }{Form: "basic", Values: vals}
        b, _ := json.Marshal(payload)
        return string(b)
    }

    tests := []struct {
        name   string
        conn   map[string]string
        want   string
    }{
        {"none", map[string]string{}, ""},
        {"plain", map[string]string{"database": "foo"}, "foo"},
        {"blob", map[string]string{"credential_blob": makeBlob(map[string]string{"database": "bar"})}, "bar"},
        {"blob empty", map[string]string{"credential_blob": makeBlob(map[string]string{})}, ""},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := explicitDatabase(tt.conn)
            if got != tt.want {
                t.Fatalf("got %q, want %q", got, tt.want)
            }
        })
    }
}

func TestParseConnParamsTLS(t *testing.T) {
    blob := func(vals map[string]string) string {
        payload := struct {
            Form   string            `json:"form"`
            Values map[string]string `json:"values"`
        }{Form: "basic", Values: vals}
        b, _ := json.Marshal(payload)
        return string(b)
    }

    tests := []struct {
        name string
        conn map[string]string
        want bool
    }{
        {"no tls", map[string]string{"credential_blob": blob(map[string]string{})}, false},
        {"tls true", map[string]string{"credential_blob": blob(map[string]string{"tls": "true"})}, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            p, err := parseConnParams(tt.conn)
            if err != nil {
                t.Fatalf("parse error: %v", err)
            }
            if p.tls != tt.want {
                t.Fatalf("tls = %v, want %v", p.tls, tt.want)
            }
        })
    }
}
