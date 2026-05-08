# QueryBox Docs

These docs cover both the current implementation and the target direction for QueryBox as a local-first operational workspace for backend engineers. The shipping product is still database-heavy in several places, but the intended scope is broader than a desktop database client.

Use this mental model when reading the docs:

```text
connection/profile -> resource graph -> actions -> results/streams
```

> **BMAD note:** Architecture and planning artifacts live in `_bmad-output/planning-artifacts/`.
> This `docs/` folder is the AI coding agent knowledge base (`project_knowledge` in BMAD config).

## Reading Order

| # | Document | Purpose |
|---|----------|---------|
| 1 | [adr/ADR-001-querybox-as-operational-runtime.md](adr/ADR-001-querybox-as-operational-runtime.md) | Product direction, scope boundaries, and local-first runtime principles |
| 2 | [glossary.md](glossary.md) | Term definitions and terminology transition |
| 3 | [data-model.md](data-model.md) | SQLite schemas, credential storage tiers |
| 4 | [features/01-connection-management.md](features/01-connection-management.md) | Current connection/profile CRUD and credential delegation flow |
| 5 | [features/02-plugin-system.md](features/02-plugin-system.md) | Current plugin contract, CLI commands, authforms |
| 6 | [features/06-query-editor-autocomplete.md](features/06-query-editor-autocomplete.md) | Query editor suggestions powered by plugins and static keywords |
| 7 | [features/03-credential-management.md](features/03-credential-management.md) | CredManager 3-tier fallback, OS keyring |
| 8 | [features/04-event-system.md](features/04-event-system.md) | Event catalogue, naming conventions |
| 9 | [features/05-frontend-ui.md](features/05-frontend-ui.md) | Theme, layout, typography, icon system |
| 10 | [features/07-row-mutation.md](features/07-row-mutation.md) | Row insert / update / delete via plugin |
| 11 | [security.md](security.md) | Threat model, security properties |
| 12 | [ops.md](ops.md) | Build, dev workflow, runbook |

## Directory Structure

```
docs/                               ← project_knowledge (AI coding agent context)
  README.md                         ← this file
  adr/
    ADR-001-querybox-as-operational-runtime.md ← product/runtime direction
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
```

## Adding a Feature Doc

1. Create `features/NN-feature-name.md` (next available number).
2. Add a row to the table above.
3. Follow the section template: **Overview → API / Contract → Implementation Notes → Edge Cases**.
