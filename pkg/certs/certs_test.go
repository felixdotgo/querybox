package certs

import (
	"os"
	"testing"
)

func TestRootCertPool(t *testing.T) {
    p, err := RootCertPool()
    if err != nil {
        t.Fatalf("root pool error: %v", err)
    }
    if p == nil {
        t.Fatalf("root pool is nil")
    }
    // pool may be empty if the embedded bundle failed to parse; that's not a
    // fatal error since callers can still proceed without it.
}

func TestRootCertPath(t *testing.T) {
    path, err := RootCertPath()
    if err != nil {
        t.Fatalf("root path error: %v", err)
    }
    if path == "" {
        t.Fatal("empty path returned")
    }
    fi, err := os.Stat(path)
    if err != nil {
        t.Fatalf("stat error: %v", err)
    }
    if fi.Size() == 0 {
        t.Fatal("certificate file is empty")
    }
}
