# Runtime Evolution Roadmap

## Current baseline

The active backlog starts after the completed foundation work:

- operational-workspace positioning is accepted
- manifest-first discovery is shipped
- `resource.graph` is the only active browse contract
- `RuntimeManager` exists with the local runtime path
- Redis is the first shipped non-database MVP slice
- release scripts exist for app and bundled-plugin release preparation
- release smoke validation covers bundled plugin metadata, plugin builds, targeted runtime tests, and optional app artifact presence

## Roadmap

### Phase 2.1: Redis workflow hardening

- strengthen the Redis validation slice around `browse -> open -> inspect -> action`
- continue generalizing result handlers and resource actions beyond SQL-only assumptions
- cover Redis-style key/value and document result routing with focused frontend tests
- keep stream/session design out of this phase unless a Redis validation gap proves it is required

### Phase 2.2: product quality and release confidence

- continue polishing the primary user journeys around connection setup, resource browsing, result inspection, and edit/delete actions
- continue making loading, empty, and error states concrete across connection, plugin, resource, result, and mutation failures
- keep Redis and other non-SQL workflows from feeling secondary to table-result workflows
- keep release smoke validation current as app startup, bundled plugin discovery, basic connection tests, and packaged artifact checks evolve
- keep release docs, artifact names, plugin metadata, and platform prerequisites aligned

### Phase 3: sessions and streaming

- design explicit session lifecycle APIs
- add stream primitives instead of forcing live workflows into one-shot `exec`
- validate Redis stream read as the first real stream workflow before broadening to Kafka or log tailing

### Phase 4: plugin trust and sandboxing

- surface permission declarations in install, enable, or first-use UI
- enforce coarse runtime controls
- define trust levels for native, managed, sandboxed, and possible WASM runtimes

### Phase 5: workspace layer

- persist local investigation context
- support reopen and resume flows across tabs, resources, and later stream panels

### Phase 6: remote agent

- validate a remote-runtime protocol boundary
- introduce a self-hosted agent only after the local-first path is stable

### Future / not implemented

- cloud execution
- collaboration features
- richer multi-user policy layers

## Planning guardrails

- every phase must prove user-facing value, not only add protocol surface
- major runtime primitives need a release smoke path and at least one polished end-user workflow before shipping
- docs must distinguish shipped baseline from roadmap intent
- compatibility cleanup can delete stale wrappers while the product remains in `0.x`
