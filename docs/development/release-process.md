# Release Process

## App release

- Run the local release smoke before tagging:

```bash
bash scripts/smoke-release.sh
```

- Build the desktop application with `wails3 build`.
- For a release-candidate artifact check after the app build, run:

```bash
bash scripts/smoke-release.sh --require-app-artifact
```

- Ensure version information is embedded correctly before packaging.
- Generate release notes against the previous `v*` app tag, not an arbitrary git range.

## Plugin release

- Plugins are versioned and shipped with their sidecar manifests.
- Release scripts under `scripts/` cover app and plugin packaging flows.
- Keep plugin contract documentation aligned with any runtime or manifest changes before release.

## Important files

- `scripts/smoke-release.sh`
- `scripts/release-app.sh`
- `scripts/release-plugin.sh`
- `scripts/check-plugin-release.sh`
- `.github/workflows/release.yml`

## Verification checklist

1. Run `bash scripts/smoke-release.sh`.
2. Build the app with `wails3 build`.
3. Run `bash scripts/smoke-release.sh --require-app-artifact`.
4. Verify release-note boundaries use the correct previous app tag.
5. Recheck docs links if release behavior or artifact layout changed.
