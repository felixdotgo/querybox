# Glossary

- Core: the orchestrator service that manages connections and drivers.
- Driver: a stateless process that implements a DB protocol and executes queries.
- Session id: ephemeral identifier for a single execution lifecycle.
- Credential blob: AES-256-GCM encrypted blob persisted in Core backing store.
- OS keyring: platform keyring (macOS Keychain, Windows Credential Manager, Linux Secret Service).
