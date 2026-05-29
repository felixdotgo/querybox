# Repository Structure

## Top-level layout

```text
main.go                 app entrypoint
services/               backend application services
pkg/plugin/             plugin SDK helpers and shared types
plugins/                bundled plugin source trees
contracts/plugin/v1/    protobuf contract
frontend/               Vue 3 application
docs/                   canonical documentation
scripts/                build and release helpers
build/                  platform packaging configuration
```

## Important subsystems

- `services/pluginmgr/`: discovery, execution, runtime lifecycle, and resource graph access.
- `services/credmanager/`: secret storage fallback chain.
- `services/connection.go`: connection CRUD and event emission.
- `frontend/src/composables/`: UI behavior around plugins, resource browsing, query results, and mutations.
- `plugins/template/`: starting point for new bundled plugins.

## Contributor guidance

- Treat plugin manifests as the runtime source of truth.
- Keep user-facing and contributor-facing docs in `docs/`, not only in inline comments or local README files.
- When module behavior changes, update the corresponding feature page and any affected architecture/reference page in the same task.
