package services

import (
	"context"
	"testing"
)

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
