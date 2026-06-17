# Plugin Capabilities Reference

## Current taxonomy

- `resource.graph`
- `query.execute`
- `stream.read`
- `connection.test`
- `schema.inspect`
- `query.explain`
- `row.mutate`
- `row.mutate.edit`
- `row.mutate.delete`

## Usage rules

- Declare only capabilities that the plugin actually implements.
- Do not use legacy names such as `query` or `describe-schema`.
- Use `resource.graph` for browseable plugins.
- Use `schema.inspect` for schema and completion metadata flows.
- Use `query.explain` only when the plugin understands the `options["explain-query"]` execution option.
- Use `row.mutate.*` when support is narrower than full edit plus delete behavior.

## Relationship to manifests

Capability declarations belong in the manifest and must match implementation behavior. `info` can enrich display metadata, but it must not redefine runtime-sensitive capability truth.
