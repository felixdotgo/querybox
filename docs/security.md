# Security

## Stored Secrets

| Layer | What is stored | Encrypted | Notes |
|-------|---------------|-----------|-------|
| `data/connections.db` | Metadata + `credential_key` references | N/A | No secrets |
| OS Keyring | Credential JSON | OS-level | Per-user isolation |
| `data/credentials.db` | Credential JSON (fallback) | ✗ | Restrict filesystem permissions |
| In-memory map | Credential JSON (last-resort) | ✗ | Cleared on restart |

**No plaintext secrets ever written to `data/connections.db`.**

---

## Plugin Execution Security

| Property | Detail |
|----------|--------|
| Timeout | Default 30s per `exec`/`resource-graph`/`tree-action`; 15s for `test-connection`; 5s for `info`. Manifest `limits.timeout_seconds` may tighten the execution ceiling per plugin. |
| Process lifecycle | `RuntimeManager` delegates to `LocalPluginHost`, which spawns one process per request and exits after response. No persistent plugin processes yet. |
| Credential transit | Passed via stdin (ephemeral, not logged, not in process env/args) |
| stderr capture | Captured for debugging; must not contain secrets |
| Trust model | Plugins run with the same OS privileges as the host app. Manifest permissions are declarative in Phase 1; they are not a strong sandbox boundary yet. |
| Sandboxing | None (future enhancement) |
| Plugin directory | per-user config path (`.../querybox/plugins`) and `bin/plugins/` — user-controlled; binaries and sidecar manifests are discovered together; no code signing enforced |

---

## Threat Model

| Threat | Mitigation | Status |
|--------|-----------|--------|
| Credentials stolen from disk | OS keyring encryption; only `credential_key` in SQLite | ✅ |
| Credentials in logs | Passed via stdin; no secret logging | ✅ |
| Malicious plugin | User-controlled directory; manifest validation + timeout limits | ⚠️ No sandboxing / declarative permissions only |
| Memory dump exposes credentials | Short-lived plugin processes | ⚠️ Best-effort |
| Cross-user credential access | OS keyring per-user isolation | ✅ OS-dependent |
| Plugin resource exhaustion | Context timeout enforcement + per-plugin manifest limits | ✅ |
| Keyring unavailable on server/CI | Automatic fallback to `data/credentials.db` | ✅ Acceptable tradeoff |

---

## Data Retention

| Asset | Retention | Deletion |
|-------|----------|---------|
| Connection metadata | Until user deletes | `ConnectionService.DeleteConnection` |
| Credentials (keyring) | Until user deletes | `CredManager.Delete` removes from all tiers |
| Credentials (sqlite fallback) | Until user deletes | Same as above |
| In-memory credentials | Cleared on restart | Automatic |

No audit logging. No telemetry. No external data transmission. All data is local-only.

---

## Backup & Cross-Platform Migration

- **Backup**: copy `data/connections.db` + export OS keyring (platform tool). For sqlite fallback environments, also copy `data/credentials.db`.
- **Cross-platform migration**: keyring formats differ — credentials must be re-entered after migrating to a new OS. Connection metadata (non-secret) can be copied directly.

---

## Future Enhancements

- Enforced permission model based on manifest declarations
- Plugin sandboxing (seccomp / namespaces / WebAssembly)
- Plugin code signing enforcement
- Audit logging (session IDs only — no credentials/queries)
- Master key option for server deployments (encrypted blob fallback)
- Query parameter redaction in stderr logs
- Plugin permission model
