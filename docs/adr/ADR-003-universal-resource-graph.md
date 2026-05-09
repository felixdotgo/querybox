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

The host keeps a compatibility adapter from legacy `connection-tree` payloads into `resource.graph` so database plugins continue working during migration.

## Consequences

- Frontend explorer logic should prefer `kind`, `actions`, and `metadata` instead of hardcoded database nouns.
- New plugins can implement `resource.graph` directly without inheriting legacy tree constraints.
- Existing database plugins remain supported through host-side adaptation until they ship native graph support.
