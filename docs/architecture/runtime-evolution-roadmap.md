# Runtime Evolution Roadmap

## Current baseline

The active backlog starts after the completed foundation work:

- operational-workspace positioning is accepted
- manifest-first discovery is shipped
- `resource.graph` is the only active browse contract
- `RuntimeManager` exists with the local runtime path
- Redis is the first shipped non-database MVP slice

## Roadmap

### Phase 2: deepen the first non-database workflow

- strengthen the Redis validation slice around `browse -> open -> inspect -> action`
- continue generalizing result handlers and resource actions beyond SQL-only assumptions

### Phase 3: sessions and streaming

- design explicit session lifecycle APIs
- add stream primitives instead of forcing live workflows into one-shot `exec`
- validate one real stream workflow end to end before broadening scope

### Phase 4: plugin trust and sandboxing

- surface permission declarations in the UI
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
- docs must distinguish shipped baseline from roadmap intent
- compatibility cleanup can delete stale wrappers while the product remains in `0.x`
