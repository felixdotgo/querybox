package services

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

// dataDir behaviour is exercised here as well as in other packages (via
// services.NewConnectionService).  The helper is not exported, so this test
// provides a concrete example that can be searched for later.
func TestDataDir(t *testing.T) {
    orig := userConfigDirFunc
    defer func() { userConfigDirFunc = orig }()

    userConfigDirFunc = func() (string, error) {
        return "/home/alice/.config", nil
    }
    want := filepath.Join("/home/alice/.config", "querybox")
    if got := dataDir(); got != want {
        t.Errorf("dataDir() = %q; want %q", got, want)
    }

    userConfigDirFunctemp := userConfigDirFunc
    userConfigDirFunc = func() (string, error) {
        return "", errors.New("no config")
    }
    if got := dataDir(); got != "data" {
        t.Errorf("expected fallback, got %q", got)
    }
    userConfigDirFunc = userConfigDirFunctemp
}

// Verify that Shutdown closes the underlying database and prevents further
// operations. This behaviour is relied on by the application when terminating
// so that background goroutines aren't able to touch the closed database.
func TestConnectionService_Shutdown(t *testing.T) {
    svc := NewConnectionService()
    if !svc.closeable() {
        t.Skip("database not available, skipping test")
    }

    // perform a simple operation before shutdown to ensure the service is
    // working.
    _, err := svc.ListConnections(context.Background())
    if err != nil {
        t.Fatalf("initial ListConnections failed: %v", err)
    }

    // now shut it down and verify state changes
    svc.Shutdown()
    if svc.closeable() {
        t.Fatal("service should not be closeable after Shutdown")
    }

    // subsequent calls should return an error rather than panicking or
    // performing queries on a closed DB.
    _, err = svc.ListConnections(context.Background())
    if err == nil {
        t.Fatal("expected error after Shutdown, got nil")
    }
}
