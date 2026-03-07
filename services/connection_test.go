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

func TestConnectionService_UpdateConnection(t *testing.T) {
	svc := NewConnectionService()
	if !svc.closeable() {
		t.Skip("database not available, skipping test")
	}
	defer svc.Shutdown()

	ctx := context.Background()

	// Create a connection to update.
	created, err := svc.CreateConnection(ctx, "original", "postgresql", `{"form":"basic","values":{"host":"localhost"}}`)
	if err != nil {
		t.Fatalf("CreateConnection failed: %v", err)
	}

	// Update name and credential.
	updated, err := svc.UpdateConnection(ctx, created.ID, "renamed", `{"form":"basic","values":{"host":"remotehost"}}`)
	if err != nil {
		t.Fatalf("UpdateConnection failed: %v", err)
	}
	if updated.Name != "renamed" {
		t.Errorf("expected updated name 'renamed', got %q", updated.Name)
	}
	if updated.ID != created.ID {
		t.Errorf("expected same ID %q, got %q", created.ID, updated.ID)
	}
	if updated.DriverType != created.DriverType {
		t.Errorf("expected driver_type preserved, got %q", updated.DriverType)
	}
	if updated.CredentialKey != created.CredentialKey {
		t.Errorf("expected credential_key preserved, got %q", updated.CredentialKey)
	}

	// Verify the stored credential was updated.
	cred, err := svc.GetCredential(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetCredential failed: %v", err)
	}
	want := `{"form":"basic","values":{"host":"remotehost"}}`
	if cred != want {
		t.Errorf("expected credential %q, got %q", want, cred)
	}
}

func TestConnectionService_UpdateConnection_UnknownID(t *testing.T) {
	svc := NewConnectionService()
	if !svc.closeable() {
		t.Skip("database not available, skipping test")
	}
	defer svc.Shutdown()

	_, err := svc.UpdateConnection(context.Background(), "does-not-exist", "newname", "cred")
	if err == nil {
		t.Fatal("expected error for unknown connection ID, got nil")
	}
}
