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

## Release smoke

Before tagging a release candidate, run:

```bash
bash scripts/smoke-release.sh
```

After `wails3 build`, require an app artifact check:

```bash
bash scripts/smoke-release.sh --require-app-artifact
```

The smoke validates bundled plugin metadata, sidecar manifests, targeted plugin/runtime tests, focused frontend result-state specs, and the app artifact presence when requested.

See [Release process](../development/release-process.md) for versioning, release notes, and plugin release workflow.
