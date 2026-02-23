package credmanager

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// fakeKeyring is an in-process keyring backed by a plain map. It is used to
// inject controllable behaviour via the keyringSet/keyringGet/keyringDelete
// package-level variables.
type fakeKeyring struct {
	data      map[string]string
	available bool // when false every call returns an error
}

func newFake(available bool) *fakeKeyring {
	return &fakeKeyring{data: make(map[string]string), available: available}
}

var errUnavailable = errors.New("keyring: service unavailable")

func (f *fakeKeyring) set(service, key, secret string) error {
	if !f.available {
		return errUnavailable
	}
	f.data[service+"/"+key] = secret
	return nil
}

func (f *fakeKeyring) get(service, key string) (string, error) {
	if !f.available {
		return "", errUnavailable
	}
	v, ok := f.data[service+"/"+key]
	if !ok {
		return "", errors.New("not found")
	}
	return v, nil
}

func (f *fakeKeyring) del(service, key string) error {
	if !f.available {
		return errUnavailable
	}
	delete(f.data, service+"/"+key)
	return nil
}

// installFake replaces the package-level keyring functions with the fake and
// returns a restore function.
func installFake(f *fakeKeyring) func() {
	origSet := keyringSet
	origGet := keyringGet
	origDel := keyringDelete
	keyringSet = f.set
	keyringGet = f.get
	keyringDelete = f.del
	return func() {
		keyringSet = origSet
		keyringGet = origGet
		keyringDelete = origDel
	}
}

// tempDB returns a writable temp path for a SQLite database and a cleanup fn.
func tempDB(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return filepath.Join(dir, "creds.db")
}

// ---------------------------------------------------------------------------
// Tests: keyring available
// ---------------------------------------------------------------------------

func TestBackend_Keyring(t *testing.T) {
	fake := newFake(true)
	restore := installFake(fake)
	defer restore()

	cm := NewWithPath(tempDB(t))
	defer cm.Close()

	if cm.Backend() != "keyring" {
		t.Fatalf("expected backend=keyring, got %q", cm.Backend())
	}
	if cm.db != nil {
		t.Fatal("sqlite db should not be opened when keyring is available")
	}
}

func TestStoreGetDelete_Keyring(t *testing.T) {
	fake := newFake(true)
	restore := installFake(fake)
	defer restore()

	cm := NewWithPath(tempDB(t))
	defer cm.Close()

	if err := cm.Store("conn1", "s3cr3t"); err != nil {
		t.Fatalf("Store: %v", err)
	}
	got, err := cm.Get("conn1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "s3cr3t" {
		t.Fatalf("Get returned %q; want %q", got, "s3cr3t")
	}
	if err := cm.Delete("conn1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := cm.Get("conn1"); err == nil {
		t.Fatal("expected error after Delete, got nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: keyring unavailable → SQLite fallback
// ---------------------------------------------------------------------------

func TestBackend_SQLite(t *testing.T) {
	fake := newFake(false) // keyring unavailable
	restore := installFake(fake)
	defer restore()

	cm := NewWithPath(tempDB(t))
	defer cm.Close()

	if cm.Backend() != "sqlite" {
		t.Fatalf("expected backend=sqlite, got %q", cm.Backend())
	}
	if cm.useKeyring {
		t.Fatal("useKeyring should be false when probe fails")
	}
}

func TestStoreGetDelete_SQLite(t *testing.T) {
	fake := newFake(false)
	restore := installFake(fake)
	defer restore()

	cm := NewWithPath(tempDB(t))
	defer cm.Close()

	if err := cm.Store("conn2", "pass123"); err != nil {
		t.Fatalf("Store: %v", err)
	}
	got, err := cm.Get("conn2")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "pass123" {
		t.Fatalf("Get returned %q; want %q", got, "pass123")
	}
	if err := cm.Delete("conn2"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := cm.Get("conn2"); err == nil {
		t.Fatal("expected error after Delete, got nil")
	}
}

// SQLite persistence: data survives Close+reopen.
func TestSQLite_Persistence(t *testing.T) {
	fake := newFake(false)
	restore := installFake(fake)
	defer restore()

	dbPath := tempDB(t)

	cm := NewWithPath(dbPath)
	if err := cm.Store("persist-key", "persist-val"); err != nil {
		t.Fatalf("Store: %v", err)
	}
	_ = cm.Close()

	cm2 := NewWithPath(dbPath)
	defer cm2.Close()

	got, err := cm2.Get("persist-key")
	if err != nil {
		t.Fatalf("Get after reopen: %v", err)
	}
	if got != "persist-val" {
		t.Fatalf("Get returned %q; want %q", got, "persist-val")
	}
}

// ---------------------------------------------------------------------------
// Tests: keyring unavailable + SQLite cannot be opened → in-memory fallback
// ---------------------------------------------------------------------------

func TestBackend_Memory(t *testing.T) {
	fake := newFake(false)
	restore := installFake(fake)
	defer restore()

	// Point to a path we cannot write (root-owned directory).
	cm := NewWithPath("/proc/impossible/path/creds.db")
	defer cm.Close()

	if cm.Backend() != "memory" {
		t.Fatalf("expected backend=memory, got %q", cm.Backend())
	}
}

func TestStoreGetDelete_Memory(t *testing.T) {
	fake := newFake(false)
	restore := installFake(fake)
	defer restore()

	cm := NewWithPath("/proc/impossible/path/creds.db")
	defer cm.Close()

	if err := cm.Store("mk", "mv"); err != nil {
		t.Fatalf("Store: %v", err)
	}
	got, err := cm.Get("mk")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "mv" {
		t.Fatalf("Get returned %q; want %q", got, "mv")
	}
	if err := cm.Delete("mk"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := cm.Get("mk"); err == nil {
		t.Fatal("expected error after Delete, got nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: input validation
// ---------------------------------------------------------------------------

func TestEmptyKey(t *testing.T) {
	fake := newFake(true)
	restore := installFake(fake)
	defer restore()

	cm := NewWithPath(tempDB(t))
	defer cm.Close()

	if err := cm.Store("", "v"); err == nil {
		t.Fatal("Store with empty key should return error")
	}
	if _, err := cm.Get(""); err == nil {
		t.Fatal("Get with empty key should return error")
	}
	if err := cm.Delete(""); err == nil {
		t.Fatal("Delete with empty key should return error")
	}
}

// ---------------------------------------------------------------------------
// Tests: probeKeyring uses the injected functions
// ---------------------------------------------------------------------------

func TestProbeKeyring_Available(t *testing.T) {
	fake := newFake(true)
	restore := installFake(fake)
	defer restore()

	if !probeKeyring() {
		t.Fatal("probeKeyring should return true when fake is available")
	}
	// Probe sentinel key must be cleaned up.
	if _, ok := fake.data[serviceName+"/"+probeKey]; ok {
		t.Fatal("probe sentinel key was not cleaned up")
	}
}

func TestProbeKeyring_Unavailable(t *testing.T) {
	fake := newFake(false)
	restore := installFake(fake)
	defer restore()

	if probeKeyring() {
		t.Fatal("probeKeyring should return false when keyring is unavailable")
	}
}

// ---------------------------------------------------------------------------
// Tests: Close is idempotent
// ---------------------------------------------------------------------------

func TestClose_Idempotent(t *testing.T) {
	fake := newFake(false)
	restore := installFake(fake)
	defer restore()

	cm := NewWithPath(tempDB(t))
	if err := cm.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := cm.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

// Ensure no file was created for the keyring-backed manager.
func TestNoDBFile_WhenKeyringAvailable(t *testing.T) {
	fake := newFake(true)
	restore := installFake(fake)
	defer restore()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "creds.db")

	cm := NewWithPath(dbPath)
	defer cm.Close()

	if _, err := os.Stat(dbPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("SQLite file should not exist when keyring is available")
	}
}
