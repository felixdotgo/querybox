# QueryBox Documentation

QueryBox is a local-first operational workspace for backend engineers. The current product is strongest on database workflows, but the architecture and roadmap are intentionally broader: resources, actions, results, and streams across multiple system types.

Use this model when reading the docs:

```text
connection/profile -> resource graph -> actions -> results/streams
```

## Reading Paths

### Product user path

1. [Product overview](user-guide/overview.md)
2. [Prerequisites](installation/prerequisites.md)
3. [Local setup](installation/local-setup.md)
4. [Connections and profiles](user-guide/connections-and-profiles.md)
5. [Browsing resources and actions](user-guide/browsing-resources-and-actions.md)
6. [Querying, results, and mutations](user-guide/querying-results-and-mutations.md)
7. [Troubleshooting](user-guide/troubleshooting.md)

### Contributor path

1. [Development environment](development/development-environment.md)
2. [Repository structure](development/repo-structure.md)
3. [System overview](architecture/system-overview.md)
4. [Runtime and plugin lifecycle](architecture/runtime-and-plugin-lifecycle.md)
5. [Plugin development](development/plugin-development.md)
6. [Plugin system feature](features/plugin-system.md)
7. [Build and package](installation/build-and-package.md)
8. [Release process](development/release-process.md)
9. [ADRs](adr/ADR-001-querybox-as-operational-runtime.md)

## Documentation Map

- `installation/`: machine setup, local run, packaging.
- `user-guide/`: user-facing workflows and troubleshooting.
- `development/`: contributor setup, repo layout, plugin authoring, release flow.
- `architecture/`: current baseline, accepted architecture, roadmap, and future direction.
- `features/`: detailed module and workflow pages for shipped behavior.
- `reference/`: glossary, protocol sketch, and capability reference.
- `adr/`: accepted architecture decisions.

## Current Status Labels

The architecture and roadmap pages use these labels consistently:

- `Current baseline`: implemented and expected in the repo today.
- `Accepted architecture`: approved direction that already shapes code and docs.
- `Roadmap`: planned work not fully shipped yet.
- `Future / not implemented`: exploratory or deferred direction.

## Key References

- [System overview](architecture/system-overview.md)
- [Runtime evolution roadmap](architecture/runtime-evolution-roadmap.md)
- [Plugin capabilities reference](reference/plugin-capabilities.md)
- [Glossary](reference/glossary.md)
