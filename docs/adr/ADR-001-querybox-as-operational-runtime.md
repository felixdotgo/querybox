# ADR-001: QueryBox as Operational Runtime

- Status: Accepted
- Date: 2026-05-08

## Context

QueryBox is currently implemented and documented primarily as a plugin-based database application. The current host contract is centered on concepts such as connection creation, query execution, and `connection-tree` browsing.

That framing is now too narrow for the next product phases. The roadmap already assumes support for non-database systems and workflows such as key inspection, object browsing, stream consumption, and operational debugging across multiple systems.

If QueryBox keeps modeling the product as a database GUI, the plugin contract, frontend language, and runtime abstractions will continue to overfit schemas, tables, and one-shot query execution. That would make the next phases much harder than necessary.

## Decision

QueryBox is positioned as a local-first operational workspace for backend engineers.

The product is optimized for these workflows:

- inspect
- query
- debug
- operate

The default runtime model is local-first:

- core usage must remain useful without an account, hosted control plane, or cloud dependency
- plugins execute on the user's machine by default
- credentials stay local by default and are passed only when needed

QueryBox is not positioned as:

- a general-purpose admin console
- a broad infrastructure provisioning surface
- an AI-first operator cockpit

The conceptual model for future product and protocol work is:

```text
connection/profile -> resource graph -> actions -> results/streams
```

This terminology does not require an immediate breaking change. Existing connection-oriented and database-oriented flows remain supported during the transition.

## Product Boundaries

QueryBox should focus on operational workflows that benefit from a shared desktop workspace:

- browsing system resources
- executing bounded queries and actions
- inspecting returned data
- reading live or incremental streams when needed
- switching across multiple systems without changing tools

QueryBox should avoid expanding prematurely into:

- full administrative control planes
- large-scale infrastructure management
- mandatory remote collaboration services
- feature sprawl that is unrelated to inspect/query/debug/operate workflows

## Consequences

- Documentation and UI copy should stop describing QueryBox as only a database client.
- New architecture work should prefer resource-oriented and capability-oriented language over database-only nouns.
- Existing database plugins should continue working through compatibility layers while the runtime model evolves.
- Follow-up ADRs should define the capability model, universal resource graph, sessions, streaming, runtime management, and sandboxing.
