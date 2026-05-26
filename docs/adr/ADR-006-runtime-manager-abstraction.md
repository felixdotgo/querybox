# ADR-006: Runtime Manager Abstraction

- Status: Accepted
- Date: 2026-05-09

## Context

`PluginManager` currently mixes discovery, metadata caching, and subprocess execution in one place. That coupling blocks runtime evolution toward local, remote-agent, and cloud-hosted execution targets.

## Decision

Introduce a `RuntimeManager` facade that owns plugin execution and delegates the current subprocess path to `LocalPluginHost`.

Phase 1 keeps only one concrete runtime target:

- `RuntimeManager` selects the runtime path from plugin metadata.
- `LocalPluginHost` executes the local binary and applies timeout limits.
- `PluginManager` keeps the public API surface but delegates execution to the runtime layer.

## Consequences

- Discovery and execution can now evolve independently.
- Future remote or sandboxed runtimes can be added behind `RuntimeManager` without rewriting every manager method.
- Existing callers still use `PluginManager`, so rollout risk stays low.
