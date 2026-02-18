package credmanager

import (
	"errors"
	"sync"

	keyring "github.com/zalando/go-keyring"
)

const serviceName = "querybox"

// CredManager provides a thin abstraction over the OS keyring. When the OS
// keyring is unavailable the manager falls back to an in-memory store so the
// application remains usable in headless/test environments.
type CredManager struct {
	fallbackMu sync.RWMutex
	fallback   map[string]string
}

// New constructs a credential manager instance.
func New() *CredManager {
	return &CredManager{fallback: make(map[string]string)}
}

// Store saves `secret` under `key`. Prefer the OS keyring and fall back to an
// in-memory store if the platform keyring isn't available.
func (c *CredManager) Store(key string, secret string) error {
	if key == "" {
		return errors.New("empty key")
	}
	if err := keyring.Set(serviceName, key, secret); err == nil {
		return nil
	}
	// fallback
	c.fallbackMu.Lock()
	c.fallback[key] = secret
	c.fallbackMu.Unlock()
	return nil
}

// Get retrieves a secret previously stored with Store.
func (c *CredManager) Get(key string) (string, error) {
	if key == "" {
		return "", errors.New("empty key")
	}
	if s, err := keyring.Get(serviceName, key); err == nil {
		return s, nil
	}
	c.fallbackMu.RLock()
	s, ok := c.fallback[key]
	c.fallbackMu.RUnlock()
	if ok {
		return s, nil
	}
	return "", errors.New("secret not found")
}

// Delete removes a secret (best-effort). If the OS keyring delete fails we
// clear the in-memory fallback entry if present.
func (c *CredManager) Delete(key string) error {
	if key == "" {
		return errors.New("empty key")
	}
	_ = keyring.Delete(serviceName, key) // ignore error; attempt fallback cleanup
	c.fallbackMu.Lock()
	delete(c.fallback, key)
	c.fallbackMu.Unlock()
	return nil
}
