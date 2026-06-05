# Redis Workflow

## What it is

Redis is the first shipped non-database validation slice for QueryBox's operational-workspace direction.

## Current behavior

- Provides manifest-first plugin discovery like the SQL plugins.
- Supports `connection.test`, `resource.graph`, and query/action flows needed for a bounded Redis MVP.
- Focuses on `browse -> open -> inspect -> action` without requiring streaming or stateful sessions.
- When browse results are scan-limited, the resource graph exposes a next-page inspect action that runs `SCAN` and renders returned keys as document results.

## Key components

- `plugins/redis/main.go`
- `plugins/redis/plugin.json`
- frontend resource actions and result handling for the Redis slice

## How it connects to other modules

- proves that the resource graph and capability model can support non-tabular workflows
- exercises generic resource actions outside the SQL table mindset
- informs the roadmap for more generalized non-database result handlers

## Limits and roadmap

- This is still a bounded MVP, not a full Redis operations console.
- Scan continuation is inspect-only; it does not yet merge additional keys back into the resource graph.
- Streams, consumers, and long-lived subscriptions belong to later session and streaming phases.
