# Security & Privacy

## Overview

QueryBox handles sensitive database credentials and implements multiple security layers to protect user data while maintaining cross-platform usability.

## Credential Management

### Storage Strategy

**Primary: OS Keyring Integration**
- Uses `go-keyring` library for platform-native secure storage
- Platforms:
  - **macOS**: Keychain Services
  - **Windows**: Credential Manager API
  - **Linux**: Secret Service (GNOME Keyring, KWallet, etc.)
- Benefits: OS-level encryption, per-user isolation, platform security policies applied

**Fallback: In-Memory Storage**
- Activated when OS keyring unavailable (server/headless/test environments)
- Thread-safe implementation using `sync.RWMutex`
- Credentials cleared on application restart
- No disk persistence in fallback mode
- Suitable for development/testing; production deployments should ensure keyring availability

### Data Flow

1. **Credential Creation**: Frontend sends credential JSON → ConnectionService → CredManager.Store()
2. **Storage Attempt**: CredManager tries OS keyring first (keyring.Set)
3. **Fallback**: If keyring fails, store in-memory map
4. **SQLite Persistence**: Only `credential_key` reference stored (e.g., "connection:uuid")
5. **Credential Retrieval**: CredManager.Get() → check keyring, then fallback
6. **Plugin Execution**: Credential passed via stdin (ephemeral, not logged)
7. **Cleanup**: Plugin process exits, credentials scrubbed from memory

### Security Properties

**No Plaintext on Disk**: SQLite contains only metadata + key references
**OS-Level Protection**: Keyring credentials encrypted by OS
**Per-User Isolation**: Credentials accessible only to owning user account
**Ephemeral Transit**: Plugins receive credentials via stdin (not environment/args)
**Short-Lived Processes**: Plugins exit after execution; no credential caching
**Migration Safety**: Old `credential_blob` automatically migrated to keyring
**Fallback Limitation**: In-memory storage cleared on restart (acceptable tradeoff for compatibility)

## Plugin Execution Security

### Resource Limits
- **Execution Timeout**: 30 seconds per plugin exec (context.WithTimeout)
- **Process Lifecycle**: Spawn → execute → exit (no persistent processes)
- **Info Probe Timeout**: 2 seconds for metadata queries
- **Environment Isolation**: Minimal environment variables passed

### Communication Channel
- **Stdin/Stdout**: JSON-based request/response (credentials sent via stdin)
- **No Network**: Plugins communicate only with target databases
- **stderr Capture**: Error output captured for debugging (credentials redacted)

### Trust Model
- Plugins run with same privileges as main application
- Plugin executables discovered under `bin/plugins/` (user-controlled directory)
- No code signing enforcement (future enhancement)
- Plugin author responsible for secure database connection handling

## Data Retention & Privacy

### Connection Metadata
- **Retention**: Indefinite until user deletes connection
- **Location**: `data/connections.db` (SQLite)
- **Backup**: User responsibility (copy SQLite file + export keyring separately)

### Credentials
- **Retention**: Persist in OS keyring until explicitly deleted
- **Deletion**: CredManager.Delete() removes from both keyring and fallback
- **Cross-Platform**: Credentials not portable (keyring format differs)

### Audit Logging
- **Current Status**: No audit logging implemented
- **Future**: Session IDs only (no credentials/queries in logs)
- **stderr Logs**: Plugin errors captured (should not contain credentials)

## Threat Model & Mitigations

| Threat | Mitigation Strategy | Status |
|--------|-------------------|--------|
| Credential theft from disk | OS keyring encryption; no plaintext in SQLite | ✅ Implemented |
| Credential exposure in logs | Credentials passed via stdin; no logging of secrets | ✅ Implemented |
| Malicious plugin execution | User controls plugin directory; 30s timeout | ⚠️ Partial (no sandboxing) |
| Memory dumps exposing credentials | Short-lived plugin processes; credentials scrubbed | ⚠️ Best-effort |
| Keyring unavail ability in production | Automatic fallback to in-memory (restart clears) | ⚠️ Acceptable tradeoff |
| Cross-user credential access | OS keyring per-user isolation | ✅ OS-dependent |
| Plugin resource exhaustion | Context timeout enforcement | ✅ Implemented |

## Compliance Considerations

### GDPR / Privacy
- User data (connection configs) stored locally only
- No telemetry or external data transmission
- User controls all credential storage and deletion
- Cross-border: Not applicable (local-only application)

### Recommendations
- Document keyring backup/restore procedures for users
- Advise against fallback mode for production environments
- Provide credential rotation guidance (delete + recreate connection)

## Future Security Enhancements

**Post-MVP Improvements**:
- Plugin sandboxing (seccomp, namespaces, or WebAssembly)
- Code signing for plugin verification
- Audit logging with session ID tracking
- Master key option for server deployments (encrypted blob fallback)
- Plugin permission model (which databases can be accessed)
- Automatic credential rotation support
- Query parameter redaction in logs
