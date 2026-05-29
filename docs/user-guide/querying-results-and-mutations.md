# Querying, Results, and Mutations

## Result model

`exec` returns one typed result payload:

- `sql`
- `document`
- `kv`

The frontend renders the result according to its payload shape and the action that produced it.

## Current behavior

- SQL drivers support query execution, explain-query flows, and row mutation.
- The Redis MVP supports bounded browse and inspect flows without requiring sessions or streaming.
- When a plugin advertises `mutate-row`, the UI can expose edit and delete affordances for returned records.

## Important limits

- QueryBox currently optimizes for bounded, finite results.
- Streaming protocols and stateful sessions are roadmap work, not baseline behavior.
- Plugins must surface real errors rather than hiding them behind empty trees or fake success payloads.

## Related docs

- [Row mutation](../features/row-mutation.md)
- [Query editor autocomplete](../features/query-editor-autocomplete.md)
- [Runtime evolution roadmap](../architecture/runtime-evolution-roadmap.md)
