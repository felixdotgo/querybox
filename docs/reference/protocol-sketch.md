# Protocol Sketch

## Current baseline

Plugins are invoked as CLI-like subprocesses with JSON requests over stdin and proto-JSON responses on stdout.

## Resource graph sketch

```json
{
  "nodes": [
    {
      "id": "db:main",
      "name": "main",
      "kind": "database",
      "path": "db:main",
      "children": [],
      "actions": [
        { "id": "open", "kind": "select", "title": "Open", "new_tab": true }
      ],
      "metadata": {}
    }
  ]
}
```

## Result sketch

`exec` returns one of:

- `sql`
- `document`
- `kv`

## Future / not implemented

The original proposal also sketched session and stream request envelopes. Those remain roadmap material until Phase 3 design work is accepted and implemented.
