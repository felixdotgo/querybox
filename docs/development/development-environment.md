# Development Environment

## Standard workflow

```bash
task build:plugins
wails3 dev
```

## Useful commands

```bash
task build:plugins
wails3 build
go test ./...
npm --prefix frontend run build
```

## Development expectations

- Build plugins before starting the app when plugin source or manifests changed.
- Keep plugin docs aligned with capability/runtime behavior in the same task.
- Prefer manifest-first runtime metadata; `info` is display metadata, not discovery authority.
- Do not preserve stale compatibility layers by default while the product is still in `0.x`.

## Validation at the end of docs work

- Run targeted searches to confirm docs links point to canonical paths.
- Recheck `README.md`, plugin docs, and comments that mention docs paths.
- Keep architecture pages explicit about baseline versus roadmap.
