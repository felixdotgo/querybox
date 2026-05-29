# Build and Package

## Build the desktop application

```bash
wails3 build
```

## Build bundled plugins

```bash
task build:plugins
```

Cross-compilation follows standard Go environment variables:

```bash
GOOS=windows GOARCH=amd64 task build:plugins
```

## Packaging model

- The app bundle carries built-in plugin binaries under `bin/plugins/`.
- On startup, QueryBox copies bundled plugins into the per-user plugin directory and then scans both the user directory and bundled directory.
- The user directory takes precedence when the same plugin name exists in both locations.

See [Release process](../development/release-process.md) for versioning, release notes, and plugin release workflow.
