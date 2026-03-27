package credmanager

// CredentialStore defines the interface for credential storage backends.
// It abstracts over the OS keyring, SQLite fallback, and in-memory store
// so that consumers (e.g. ConnectionService) can be tested with mocks.
type CredentialStore interface {
	Store(key, secret string) error
	Get(key string) (string, error)
	Delete(key string) error
}

// Verify CredManager implements CredentialStore at compile time.
var _ CredentialStore = (*CredManager)(nil)
