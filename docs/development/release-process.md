# Release Process

## App release

- Build the desktop application with `wails3 build`.
- Ensure version information is embedded correctly before packaging.
- Generate release notes against the previous `v*` app tag, not an arbitrary git range.

## Plugin release

- Plugins are versioned and shipped with their sidecar manifests.
- Release scripts under `scripts/` cover app and plugin packaging flows.
- Keep plugin contract documentation aligned with any runtime or manifest changes before release.

## Important files

- `scripts/release-app.sh`
- `scripts/release-plugin.sh`
- `scripts/check-plugin-release.sh`
- `.github/workflows/release.yml`

## Verification checklist

1. Build app and bundled plugins.
2. Verify manifests exist beside plugin binaries.
3. Verify release-note boundaries use the correct previous app tag.
4. Recheck docs links if release behavior or artifact layout changed.
