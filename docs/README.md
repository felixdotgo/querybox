# QueryBox Docs

QueryBox is a desktop database client that delegates all database protocols to lightweight plugin executables, communicating over JSON stdin/stdout using protobuf-derived types.

## Reading Order

| # | Document | Purpose |
|---|----------|---------|
| 1 | [glossary.md](glossary.md) | Term definitions — read first to establish vocabulary |
| 2 | [architecture.md](architecture.md) | System diagram, component map, data flows |
| 3 | [data-model.md](data-model.md) | SQLite schemas, credential storage tiers |
| 4 | [features/01-connection-management.md](features/01-connection-management.md) | Connection CRUD, credential delegation |
| 5 | [features/02-plugin-system.md](features/02-plugin-system.md) | Plugin contract, CLI commands, authforms |
| 6 | [features/03-credential-management.md](features/03-credential-management.md) | CredManager 3-tier fallback, OS keyring |
| 7 | [features/04-event-system.md](features/04-event-system.md) | Event catalogue, naming conventions |
| 8 | [features/05-frontend-ui.md](features/05-frontend-ui.md) | Theme, layout, typography, icon system |
| 9 | [security.md](security.md) | Threat model, security properties |
| 10 | [ops.md](ops.md) | Build, dev workflow, runbook |

## Directory Structure

```
docs/
  README.md                         ← this file
  glossary.md                       ← vocabulary reference
  architecture.md                   ← system overview & flows
  data-model.md                     ← schemas & storage
  security.md                       ← threat model
  ops.md                            ← runbook & build
  features/
    01-connection-management.md     ← feature: connection CRUD
    02-plugin-system.md             ← feature: plugin contract
    03-credential-management.md     ← feature: credential storage
    04-event-system.md              ← feature: event bus
    05-frontend-ui.md               ← feature: UI guidelines
```

## Adding a Feature Doc

1. Create `features/NN-feature-name.md` (next available number).
2. Add a row to the table above.
3. Follow the section template: **Overview → API / Contract → Implementation Notes → Edge Cases**.
