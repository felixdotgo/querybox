package services

import (
	"context"
	"os"
	"testing"
)

func TestConnectionCredentialRoundtrip(t *testing.T) {
    // ensure a clean state by removing any existing data files
    _ = os.RemoveAll("data")
    svc := NewConnectionService()
    ctx := context.Background()

    // create a connection with a simple credential blob
    conn, err := svc.CreateConnection(ctx, "foo", "driver", "{\"host\":\"localhost\"}")
    if err != nil {
        t.Fatalf("CreateConnection failed: %v", err)
    }

    cred, err := svc.GetCredential(ctx, conn.ID)
    if err != nil {
        t.Fatalf("GetCredential error: %v", err)
    }
    if cred != "{\"host\":\"localhost\"}" {
        t.Fatalf("unexpected credential: %s", cred)
    }
}
