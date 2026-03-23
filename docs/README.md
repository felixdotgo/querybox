# QueryBox Docs

QueryBox is a desktop database client that delegates all database protocols to lightweight plugin executables, communicating over JSON stdin/stdout using protobuf-derived types.

> **BMAD note:** Architecture and planning artifacts live in `_bmad-output/planning-artifacts/`.
> This `docs/` folder is the AI coding agent knowledge base (`project_knowledge` in BMAD config).

## Reading Order

| # | Document | Purpose |
|---|----------|---------|
| 1 | [glossary.md](glossary.md) | Term definitions — read first to establish vocabulary |
| 2 | [data-model.md](data-model.md) | SQLite schemas, credential storage tiers |
| 3 | [features/01-connection-management.md](features/01-connection-management.md) | Connection CRUD, credential delegation |
| 4 | [features/02-plugin-system.md](features/02-plugin-system.md) | Plugin contract, CLI commands, authforms |
| 5 | [features/06-query-editor-autocomplete.md](features/06-query-editor-autocomplete.md) | Query editor suggestions powered by plugins and static keywords |
| 6 | [features/03-credential-management.md](features/03-credential-management.md) | CredManager 3-tier fallback, OS keyring |
| 7 | [features/04-event-system.md](features/04-event-system.md) | Event catalogue, naming conventions |
| 8 | [features/05-frontend-ui.md](features/05-frontend-ui.md) | Theme, layout, typography, icon system |
| 9 | [features/07-row-mutation.md](features/07-row-mutation.md) | Row insert / update / delete via plugin |
| 10 | [security.md](security.md) | Threat model, security properties |
| 11 | [ops.md](ops.md) | Build, dev workflow, runbook |

## Directory Structure

```
docs/                               ← project_knowledge (AI coding agent context)
  README.md                         ← this file
  glossary.md                       ← vocabulary reference
  data-model.md                     ← schemas & storage
  security.md                       ← threat model
  ops.md                            ← runbook & build
  features/
    01-connection-management.md     ← feature: connection CRUD
    02-plugin-system.md             ← feature: plugin contract
    03-credential-management.md     ← feature: credential storage
    04-event-system.md              ← feature: event bus
    05-frontend-ui.md               ← feature: UI guidelines
    06-query-editor-autocomplete.md ← feature: query editor auto-completion
    07-row-mutation.md              ← feature: row mutation

_bmad-output/planning-artifacts/    ← BMAD planning artifacts
  architecture.md                   ← system diagram, component map, data flows
```

## Adding a Feature Doc

1. Create `features/NN-feature-name.md` (next available number).
2. Add a row to the table above.
3. Follow the section template: **Overview → API / Contract → Implementation Notes → Edge Cases**.
