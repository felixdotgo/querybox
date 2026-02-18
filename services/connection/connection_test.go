package connection

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestCreateStoresKeyAndReference(t *testing.T) {
	// Run test in a temporary working directory so we don't touch the repo data/ DB.
	d := t.TempDir()
	old, _ := os.Getwd()
	_ = os.Chdir(d)
	defer os.Chdir(old)

	mgr := New()
	ctx := context.Background()
	cred := `{"form":"basic","values":{"user":"u","password":"p"}}`
	conn, err := mgr.Create(ctx, "my-conn", "driver-x", cred)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if conn.ID == "" {
		t.Fatalf("expected id")
	}
	if conn.CredentialKey == "" {
		t.Fatalf("expected credential_key to be set")
	}

	// Retrieve and verify persisted reference
	r, err := mgr.Get(ctx, conn.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if r.CredentialKey != conn.CredentialKey {
		t.Fatalf("credential_key mismatch: got %q want %q", r.CredentialKey, conn.CredentialKey)
	}

	// Ensure the secret is available via the CredManager backing the manager
	secret, err := mgr.cred.Get(conn.CredentialKey)
	if err != nil {
		t.Fatalf("credmanager.Get failed: %v", err)
	}
	if !strings.Contains(secret, `"user":"u"`) {
		t.Fatalf("stored secret missing expected content: %q", secret)
	}
}
