# ADR-007: Plugin Manifest and Permissions

- Status: Accepted
- Date: 2026-05-09

## Context

QueryBox previously discovered plugins by probing each binary with `info`. That gives weak guarantees around runtime selection, capability declaration, permissions, and limits.

Phase 1 needs a manifest-first contract that makes runtime metadata explicit and fail-fast.

## Decision

Each plugin ships a sidecar manifest named `<binary>.manifest.json`.

Manifest v1 requires:

- `id`
- `type`
- `name`
- `description`
- `version`
- `url`
- `author`
- `runtime`
- `capabilities`
- `permissions`
- `limits`
- `tags`
- `license`
- `icon_url`
- `contact`
- `metadata`
- `settings`

The current supported capability taxonomy is:

- `resource.graph`
- `query.execute`
- `stream.read`
- `connection.test`
- `schema.inspect`
- `query.explain`
- `row.mutate`
- `row.mutate.edit`
- `row.mutate.delete`

Discovery loads and validates the manifest before execution. The manifest is the source of truth for both runtime-sensitive fields and host-visible plugin metadata. `info` is no longer part of discovery.

## Consequences

- New plugins can declare runtime, security, and host-visible metadata without overloading `info`.
- Invalid or missing manifests fail fast during discovery instead of surfacing later at execution time.
- Build tooling must copy manifests beside built plugin binaries so bundled and user plugin directories stay consistent.
