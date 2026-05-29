# Troubleshooting

## Plugin not discovered

1. Confirm the plugin binary exists under `bin/plugins/` or the per-user plugin directory.
2. Confirm the sidecar manifest `<binary>.manifest.json` exists and is valid.
3. Restart the app or trigger a plugin rescan.
4. Check plugin metadata and last error details in the Plugins UI.

## Credential not found

1. Check whether the OS keyring is available on the current machine.
2. If not, confirm `data/credentials.db` is writable.
3. If neither persistent tier works, the in-memory fallback clears on restart.

## Connection metadata or secrets after migration

- Connection metadata lives in `data/connections.db`.
- Secrets should not be in `data/connections.db`; only `credential_key` references belong there.
- If the keyring is unavailable, fallback secrets move to `data/credentials.db`.

## SQLite corruption or local data loss

- Restore `data/connections.db` from backup for metadata.
- Restore OS keyring contents or `data/credentials.db` for secrets.
- If secrets are lost, recreate the connection or re-enter credentials.

## Related docs

- [Security model](../architecture/security-model.md)
- [Data and credential model](../architecture/data-and-credential-model.md)
- [Release process](../development/release-process.md)
