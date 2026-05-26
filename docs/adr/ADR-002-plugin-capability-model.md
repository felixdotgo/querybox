# ADR-002: Plugin Capability Model

- Status: Accepted
- Date: 2026-05-26

## Context

Phase 1 introduced manifest-first discovery and `resource.graph`, but the repo
still carried mixed signals about capability scope, browse compatibility, and
whether `info` remained part of plugin discovery.

Because QueryBox is still in `0.x`, the team does not need to preserve old
plugin contracts purely for backward compatibility. Keeping parallel capability
or browse models would increase complexity precisely when the runtime surface is
still being shaped.

## Decision

QueryBox standardizes on one manifest-declared capability taxonomy:

### Core runtime capabilities

- `resource.graph`
- `query.execute`
- `stream.read`
- `connection.test`
- `schema.inspect`

### Feature and UI extension capabilities

- `explain-query`
- `mutate-row`
- `mutate-row::edit`
- `mutate-row::delete`

Rules:

- The manifest is the only source of truth for capability declaration.
- `resource.graph` is the only supported browse capability for active plugins.
- Legacy names such as `query` and `describe-schema` are invalid in manifests.
- Plugin implementations must match declared capabilities; the host should not infer capabilities from optional commands or legacy behavior.
- Since QueryBox is in `0.x`, stale compatibility layers may be removed once built-in plugins and docs are moved onto the new contract.

## Consequences

- Plugin discovery, validation, docs, and built-in manifests now share one capability vocabulary.
- New plugin work should target `resource.graph` directly and avoid reusing database-only browse terminology as a contract surface.
- Future phases can build non-database workflows, permission enforcement, and runtime evolution on a smaller, clearer contract.
