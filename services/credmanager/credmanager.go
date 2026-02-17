package credmanager

import "errors"

// CredManager stores and retrieves credentials for external connections.
// This is a small, internal skeleton — expand with secure storage (OS keystore,
// encryption) when needed.
type CredManager struct{}

// New returns a new credential manager instance.
func New() *CredManager { return &CredManager{} }

// Store saves a secret for a key (placeholder — implement secure storage).
func (c *CredManager) Store(key string, secret string) error {
	if key == "" {
		return errors.New("empty key")
	}
	return nil
}

// Get retrieves a secret (placeholder).
func (c *CredManager) Get(key string) (string, error) {
	if key == "" {
		return "", errors.New("empty key")
	}
	return "", nil
}
