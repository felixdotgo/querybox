# Glossary

## Core terms

- `connection/profile`: a local record describing how QueryBox should reach a system.
- `resource graph`: the generic browse contract returned by plugins.
- `action`: a plugin-defined operation available from a resource or result context.
- `result`: a finite response payload such as SQL rows, documents, or key-value entries.
- `stream`: a future live or incremental result channel for logs, events, or queue consumption.
- `manifest-first discovery`: plugin discovery that treats the manifest as authoritative for runtime-sensitive metadata.
- `RuntimeManager`: the host abstraction that chooses how plugin operations execute.

## Terminology transition

- `connection-tree` -> historical term, no longer an active shared contract.
- `resource.graph` -> active browse contract.
- `plugin info` -> optional display metadata, not discovery authority.
