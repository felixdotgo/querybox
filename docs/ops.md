# Ops & Runbook

## Development Workflow

```bash
# 1. Build plugin binaries (required before running the app)
task build:plugins
# or: scripts/build-plugins.sh

# 2. Verify plugins are in place
# listing the current plugin directory

ls bin/plugins/
# if you have per-user plugins they live under the OS config path, e.g.
#   $XDG_CONFIG_HOME/querybox/plugins  (Linux)
#   %APPDATA%\querybox\plugins      (Windows)
#   ~/Library/Application Support/querybox/plugins (macOS)

# 3. Start the app with hot-reload
wails3 dev
# Frontend changes reflect immediately; Go changes trigger rebuild
```

---

## Build & Release

```bash
# 1. Build plugins
task build:plugins

# 2. Build desktop app for target platform
# (platform-specific Taskfile in build/<platform>/Taskfile.yml)
task -d build/windows   # Windows
task -d build/darwin    # macOS
task -d build/linux     # Linux

# 3. Verify the binary launches and can connect to a test database
# 4. Tag release + update VERSION and changelog
```

Cross-platform builds are documented in `build/*/Taskfile.yml`.

---

## Common Issues

### Plugin not discovered
1. Confirm binary is under `bin/plugins/` (or the per-user plugins path above) and is executable (`chmod +x` on Unix).
2. Plugins are scanned **once at startup**. Restart the app to pick up newly added or removed binaries.
3. Alternatively, use the **Rescan** button in the Plugins window to re-probe without a full restart.
4. Check `PluginInfo.LastError` in the Plugins window — captures `plugin info` probe failures.

### Credential not found
1. Check which tier is active: OS keyring → `data/credentials.db` → in-memory map.
2. On Linux, ensure a Secret Service provider is running (GNOME Keyring or KWallet).
3. If keyring is unavailable and `data/credentials.db` fallback is used, confirm the file is writable.
4. In-memory map is cleared on restart — re-enter credentials if the app was restarted without a working keyring or sqlite fallback.

### Connection metadata missing after schema change
- On first launch after upgrading from old `credential_blob` schema, the app auto-migrates blobs to the keyring.
- Verify: inspect `data/connections.db` — `credential_key` should be populated.
- If migration failed (keyring unavailable), check `data/credentials.db` for the fallback secrets.

### SQLite corruption
- Replace `data/connections.db` with a backup; credentials remain intact in keyring.
- If both `data/credentials.db` and the keyring are lost, re-create each connection manually.

---

## Backup & Recovery

| Asset | Backup | Recovery |
|-------|--------|---------|
| Connection metadata | Copy `data/connections.db` | Replace file, restart app |
| Credentials (keyring) | Platform keyring export tool | Re-import or re-enter via UI |
| Credentials (sqlite fallback) | Copy `data/credentials.db` | Replace file, restart app |

> Cross-platform credential migration is not supported — keyring formats differ. Re-enter credentials after migrating OS.

---

## Rollback

1. Replace app binary with the previous version.
2. SQLite schema is forwards-compatible for minor versions; `credential_blob` migration is non-destructive.
3. For a broken plugin binary: drop the replacement into `bin/plugins/` — PluginManager picks it up within 2 seconds.
