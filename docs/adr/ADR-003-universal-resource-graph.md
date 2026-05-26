# ADR-003: Universal Resource Graph

- Status: Accepted
- Date: 2026-05-09

## Context

The current plugin and frontend model is centered on `connection-tree`, which works well for relational databases but leaks database-only assumptions into the runtime and UI.

Phase 1 needs a resource model that can represent databases, buckets, topics, pods, keys, streams, and future operational targets without rewriting the explorer again.

## Decision

QueryBox standardizes on `resource.graph` as the runtime-neutral browse contract.

The host-side model contains:

- `ResourceGraphRequest` with `connection`, `resource_id`, and `depth`
- `ResourceGraphResponse` with top-level `nodes`
- `ResourceNode` with `id`, `name`, `kind`, `path`, `actions`, `children`, and `metadata`
- `ResourceAction` with `id`, `kind`, `title`, `query`, `new_tab`, `fields`, and `metadata`

The host now treats `resource.graph` as the only browse contract for bundled plugins. Compatibility adapters from `connection-tree` are no longer part of the runtime path.

## Consequences

- Frontend explorer logic should prefer `kind`, `actions`, and `metadata` instead of hardcoded database nouns.
- New plugins can implement `resource.graph` directly without inheriting legacy tree constraints.
- Built-in plugins should move to `resource.graph` as the only active browse contract; stale compatibility adapters should be removed once no longer needed.
