# Runbook — QueryBox Desktop App

## Daily Development Checks
- Run `task build:plugins` to rebuild plugin binaries after any changes under `plugins/`.
- Confirm `bin/plugins/` contains the expected executables before starting the app (`ls bin/plugins/`).
- Run `wails3 dev` for hot-reload development; frontend changes reflect immediately, Go changes trigger rebuild.

## Build & Release

1. Build plugins first: `task build:plugins` (or `scripts/build-plugins.sh`).
2. Build the desktop app for the target platform via the relevant `build/*/Taskfile.yml`.
3. Verify the produced binary launches and can connect to a test database.
4. Tag the release and update `VERSION` / changelog.

## Common Issues

### Plugin Not Discovered
- Confirm the binary is under `bin/plugins/` and is executable (`chmod +x`).
- Run `Rescan()` from the UI or restart the app; the scanner runs every 2 seconds.
- Check `PluginInfo.LastError` in the UI — this captures `plugin info` probe failures.

### Credential Not Found
- The app uses a 3-tier fallback: OS keyring → `data/credentials.db` → in-memory map.
- On Linux, ensure a Secret Service provider is running (GNOME Keyring or KWallet).
- If the keyring is unavailable, credentials fall back to `data/credentials.db`; confirm that file is writable.
- In-memory fallback is cleared on restart — re-enter credentials if the app was restarted without a working keyring or sqlite fallback.

### Connection Migration (Old Schema)
- On first launch after upgrading from a version that stored `credential_blob`, the app automatically migrates blobs to the keyring.
- If migration fails (keyring unavailable), credentials fall back to `data/credentials.db`.
- Verify by inspecting `data/connections.db`: `credential_key` column should be populated, `credential_blob` should be NULL.

### SQLite Corruption
- Replace `data/connections.db` with a backup copy; connection metadata will be restored without needing to re-enter credentials if the keyring still holds the secrets.
- If `data/credentials.db` is also lost and the keyring was unavailable, re-create each connection manually.

## Backup & Recovery

| Asset | Backup Method | Recovery |
|-------|--------------|----------|
| Connection metadata | Copy `data/connections.db` | Replace file; restart app |
| Credentials (keyring) | Platform keyring export tool | Re-import or re-enter via UI |
| Credentials (sqlite fallback) | Copy `data/credentials.db` | Replace file; restart app |

> Cross-platform credential migration is not supported — keyring formats differ between platforms. Re-enter credentials after migrating to a new OS.

## Rollback

- Replace the app binary with the previous version.
- The SQLite schema is forwards-compatible for minor versions; `credential_blob` migration is non-destructive.
- If a plugin binary is broken, replace it in `bin/plugins/`; the manager picks up the new executable within 2 seconds.
