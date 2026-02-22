package credmanager

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/zalando/go-keyring"
)

// The tests in this file verify that the fallback sqlite database is used when
// the OS keyring is unavailable, and that entries persist across manager
// instances. We override the keyring helpers defined in credmanager.go so we
// don't depend on the real platform keyring during unit tests.

func TestSqliteFallbackPersistence(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "creds.db")

	// Make the keyring functions always fail so we hit the sqlite branch.
	keyringSet = func(service, user, secret string) error { return errors.New("fail") }
	keyringGet = func(service, user string) (string, error) { return "", errors.New("fail") }
	keyringDelete = func(service, user string) error { return errors.New("fail") }
	defer func() {
		// restore to original behavior after test
		keyringSet = keyring.Set
		keyringGet = keyring.Get
		keyringDelete = keyring.Delete
	}()

	mgr := NewWithPath(path)
	if mgr == nil {
		t.Fatal("expected manager instance")
	}

	// store a secret and verify retrieval
	if err := mgr.Store("foo", "bar"); err != nil {
		t.Fatalf("store failed: %v", err)
	}

	val, err := mgr.Get("foo")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if val != "bar" {
		t.Fatalf("unexpected value: %s", val)
	}

	// close and reopen manager to confirm persistence
	_ = mgr.Close()
	mgr2 := NewWithPath(path)
	if v2, err := mgr2.Get("foo"); err != nil || v2 != "bar" {
		t.Fatalf("persisted value not found after reopen: %v %s", err, v2)
	}

	// delete the key and ensure it is gone
	if err := mgr2.Delete("foo"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if _, err := mgr2.Get("foo"); err == nil {
		t.Fatal("expected error after deletion")
	}
}

func TestInMemoryFallbackWhenDBUnusable(t *testing.T) {
	// force the db open to fail by using a path in a non-existent directory with
	// insufficient permissions. On most systems "/root" cannot be written by
	// regular users.
	mgr := NewWithPath("/root/invalid.db")
	// manager should exist and fallback to memory
	if mgr == nil {
		t.Fatal("expected manager even if db creation failed")
	}

	// store and retrieve
	if err := mgr.Store("a", "b"); err != nil {
		t.Fatalf("store failed: %v", err)
	}
	if v, err := mgr.Get("a"); err != nil || v != "b" {
		t.Fatalf("unexpected in-memory value %v %s", err, v)
	}
}
