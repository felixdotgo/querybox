# Runtime and Plugin Lifecycle

## Current baseline

- `RuntimeManager` owns execution.
- Phase 1 keeps a local native-process host only.
- One plugin subprocess is spawned per request.
- Discovery is manifest-first and cached in memory until startup or explicit rescan.

## Request lifecycle

1. QueryBox resolves the plugin from registry metadata.
2. The runtime manager chooses the local plugin host.
3. The host spawns the plugin executable.
4. The host sends a command payload over stdin.
5. The host enforces timeout and output limits.
6. The plugin returns a response and exits.

## Active command surface

- `info`
- `authforms`
- `exec`
- `resource-graph`
- `test-connection`
- optional metadata and mutation commands such as `completion-fields`, `describe-schema`, and `mutate-row`

## Accepted architecture

- Manifest metadata is authoritative for runtime-sensitive behavior.
- `resource.graph` replaced the old browse contract as the single active model.
- UI behavior should adapt to capabilities and resource/action metadata instead of assuming SQL tables everywhere.

## Roadmap

- Stateful sessions for workflows that need long-lived context.
- Streaming-first APIs for logs, queues, and live operational views.
- Remote runtime targets behind the same runtime manager abstraction.

See [Runtime evolution roadmap](runtime-evolution-roadmap.md) and [Protocol sketch](../reference/protocol-sketch.md).
