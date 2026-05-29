# System Overview

## Accepted architecture

QueryBox is positioned as a local-first operational workspace for backend engineers. It should help users inspect, query, debug, and operate data systems without requiring a hosted control plane.

## Current baseline

- Desktop app with Wails v3, Go backend, and Vue 3 frontend.
- Plugin-based runtime using out-of-process executables.
- Local-first credential model with OS keyring primary storage.
- Manifest-first discovery and `resource.graph` as the active browse contract.
- Redis is the first shipped non-database validation slice beyond the SQL core.

## High-level architecture

```text
UI (Vue / Wails)
  -> Core services
  -> Runtime manager
  -> Plugin processes
```

The frontend should not need to care whether a capability comes from a SQL plugin, Redis plugin, or future runtime target. The host should normalize the contract around resources, actions, and result shapes.

## Product boundaries

QueryBox is intended for:

- browsing system resources
- executing bounded queries and actions
- inspecting results
- switching between systems in one workspace

QueryBox is not intended to become:

- a full infrastructure control plane
- a mandatory cloud collaboration surface
- an AI-first operator cockpit

## Relationship to the old proposal

The original `plan.md` proposal mixed implemented baseline, accepted direction, and future roadmap in one place. This docs tree splits those concerns:

- current system behavior lives in feature and architecture baseline pages
- accepted decisions live in ADRs
- planned expansion lives in the roadmap pages
