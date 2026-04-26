# Releasing

This document covers the full release process for the QueryBox application and its plugins.

---

## Overview

QueryBox has two independent release tracks:

| Track | Tag format | Triggers | Artifacts |
|---|---|---|---|
| **App** | `v1.2.3` | `.github/workflows/release.yml` | Linux tar.gz, macOS .zip, Windows installer |
| **Plugin** | `plugin-{name}-v1.2.3` | `.github/workflows/release-plugin.yml` | Per-platform plugin binaries |

These tracks are decoupled — a plugin can be updated without shipping a new app version.

---

## Versioning

Both the app and plugins follow [Semantic Versioning](https://semver.org/) (`MAJOR.MINOR.PATCH`):

| Change | Bump |
|---|---|
| Bug fixes, metadata changes, minor tweaks | PATCH (`0.1.0 → 0.1.1`) |
| New features, new capabilities, behavior changes | MINOR (`0.1.0 → 0.2.0`) |
| Breaking changes to the plugin contract or API | MAJOR (`0.1.0 → 1.0.0`) |

**Rule**: Any code change to a plugin requires a version bump in both the plugin's `Info()` function and in `plugins/registry.json` before pushing the release tag. See the [Plugin Version Bump rule in CLAUDE.md](../CLAUDE.md).

---

## Releasing the App

### Prerequisites
- A clean `main` branch with all intended changes merged
- All plugin changes already released (if shipping updated plugins with the app)

### Steps

1. **Decide the new version** based on changes since the last release (semver rules above).

2. **Update `plugins/registry.json`** if any bundled plugin versions changed since the last app release.

3. **Push the release tag:**
   ```bash
   git tag v1.2.3
   git push origin v1.2.3
   ```

4. **GitHub Actions builds automatically** on tag push:
   - Compiles the app for Linux (amd64), macOS (universal), and Windows (amd64)
   - Embeds the version string via `-X github.com/felixdotgo/querybox/pkg/version.Version=v1.2.3`
   - Packages artifacts and creates a GitHub Release

5. **Verify the release** at `https://github.com/felixdotgo/querybox/releases`:
   - All 3 platform artifacts are attached
   - Release notes are auto-generated from commit messages

### What gets embedded

The app version is injected at build time via linker flags. At runtime it is available as:
- `pkg/version.Version` (Go)
- `GetUpdateStatus().appCurrentVersion` (frontend, via updater service)
- The **About** dialog (`App.ShowAboutDialog`)

Local/dev builds use `"dev"` as the version.

---

## Releasing a Plugin

Plugins are released independently. Each plugin release creates a separate GitHub Release with per-platform binaries.

### Steps

1. **Make and commit your plugin changes.**

2. **Bump the version** in two places:

   a. In the plugin's `Info()` function (`plugins/{name}/main.go`):
   ```go
   Version: "0.1.2",
   ```

   b. In `plugins/registry.json`:
   ```json
   {
     "plugins": {
       "mysql": { "version": "0.1.2" }
     }
   }
   ```

3. **Commit both changes together:**
   ```bash
   git add plugins/mysql/main.go plugins/registry.json
   git commit -m "feat(plugin/mysql): ..."
   ```

4. **Push the plugin release tag:**
   ```bash
   git tag plugin-mysql-v0.1.2
   git push origin plugin-mysql-v0.1.2
   ```

5. **GitHub Actions builds automatically:**
   - Builds `mysql` binary for Linux (amd64), macOS (universal via lipo), and Windows (amd64)
   - Creates a GitHub Release named `Plugin mysql v0.1.2`
   - Attaches binaries: `mysql-linux-amd64`, `mysql-darwin-universal`, `mysql-windows-amd64.exe`

6. **Users with the app already installed** will see an update badge in the Plugins window on their next app startup (checked once per day, see [Update Check Mechanism](#update-check-mechanism)).

### Naming convention for plugin release tags

```
plugin-{plugin-name}-v{version}

Examples:
  plugin-mysql-v0.1.2
  plugin-postgresql-v0.2.0
  plugin-sqlite-v0.1.1
```

### Binary asset names

| Platform | Binary name |
|---|---|
| Linux amd64 | `{name}-linux-amd64` |
| macOS universal | `{name}-darwin-universal` |
| Windows amd64 | `{name}-windows-amd64.exe` |

---

## Update Check Mechanism

### How it works

On startup the app runs a background update check (non-blocking):

1. **Plugin registry** — fetches `plugins/registry.json` from the `main` branch via `raw.githubusercontent.com` (no rate limit, no auth required):
   ```
   https://raw.githubusercontent.com/felixdotgo/querybox/main/plugins/registry.json
   ```

2. **App latest release** — queries the GitHub Releases API once per day:
   ```
   https://api.github.com/repos/felixdotgo/querybox/releases/latest
   ```

3. Results are **cached for 24 hours** at:
   - Linux: `~/.config/querybox/update-cache.json`
   - macOS: `~/Library/Application Support/querybox/update-cache.json`
   - Windows: `%APPDATA%\querybox\update-cache.json`

4. The cache is **only written on success** — if both fetches fail (e.g. no network, registry not yet deployed), no cache is written and the check retries on next startup.

### What users see

| Condition | UI |
|---|---|
| App update available | Amber banner in Plugins window with "Download →" link |
| Plugin update available | Amber dot next to plugin name + notice in detail panel with "View release →" link |
| Dev build (`version = "dev"`) | Plugin registry is still fetched and cached; app update banner is suppressed (cannot meaningfully compare `dev` against a semver) |

### `plugins/registry.json` format

```json
{
  "plugins": {
    "mysql":      { "version": "0.1.1" },
    "postgresql": { "version": "0.1.1" },
    "sqlite":     { "version": "0.1.1" }
  }
}
```

Keys must match `plugin.name.toLowerCase()` as returned by the plugin's `Info()` response.

---

## Checklist

### App release checklist

- [ ] All plugin versions in `plugins/registry.json` are correct
- [ ] `git tag v1.2.3 && git push origin v1.2.3`
- [ ] CI passes on all 3 platforms
- [ ] GitHub Release has all 3 artifacts
- [ ] About dialog shows correct version on a test install

### Plugin release checklist

- [ ] Version bumped in `plugins/{name}/main.go` → `Info()` → `Version` field
- [ ] Version bumped in `plugins/registry.json`
- [ ] Both changes committed before tagging
- [ ] `git tag plugin-{name}-v{version} && git push origin plugin-{name}-v{version}`
- [ ] CI builds all 3 platform binaries successfully
- [ ] GitHub Release `plugin-{name}-v{version}` exists with correct assets
