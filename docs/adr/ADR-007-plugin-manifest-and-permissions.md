# ADR-007: Plugin Manifest and Permissions

- Status: Accepted
- Date: 2026-05-09

## Context

QueryBox previously discovered plugins by probing each binary with `info`. That gives weak guarantees around runtime selection, capability declaration, permissions, and limits.

Phase 1 needs a manifest-first contract while preserving compatibility for old plugins.

## Decision

Each plugin may ship a sidecar manifest named `<binary>.manifest.json`.

Manifest v1 requires:

- `id`
- `version`
- `runtime`
- `capabilities`
- `permissions`
- `limits`

The current supported capability taxonomy is:

- `resource.graph`
- `query.execute`
- `stream.read`
- `connection.test`
- `schema.inspect`

Discovery loads and validates the manifest before execution. If no manifest exists, QueryBox falls back to the legacy `info` command so old plugins still work.

## Consequences

- New plugins can declare runtime and security-relevant metadata without overloading `info`.
- Invalid manifests fail fast during discovery instead of surfacing later at execution time.
- Build tooling must copy manifests beside built plugin binaries so bundled and user plugin directories stay consistent.
