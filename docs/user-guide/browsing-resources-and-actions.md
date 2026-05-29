# Browsing Resources and Actions

## What it is

QueryBox presents connected systems through a generic resource graph. A node may represent a database, schema, table, keyspace, Redis key, or another system-specific resource.

## Current behavior

- The host calls plugin `resource-graph` and normalizes the response for the explorer UI.
- Nodes carry `kind`, `metadata`, and optional `actions`.
- The UI should adapt from plugin metadata instead of assuming every resource is SQL-shaped.

## Current baseline

- `resource.graph` is the only active browse contract.
- Legacy `connection-tree` is no longer part of the shared SDK/proto surface.
- Bundled plugins are expected to emit resource-graph-native nodes and actions directly.

## Related docs

- [Plugin system](../features/plugin-system.md)
- [Runtime and plugin lifecycle](../architecture/runtime-and-plugin-lifecycle.md)
- [Redis workflow](../features/redis-workflow.md)
