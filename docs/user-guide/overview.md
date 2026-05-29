# Product Overview

## What QueryBox is

QueryBox is a local-first operational workspace for backend engineers who need to inspect, query, debug, and operate data systems from one desktop app.

## Current baseline

- Desktop app built with Wails v3, Go, and Vue 3.
- Plugin-based runtime with out-of-process executables.
- Strongest shipped workflows today: SQL databases plus a Redis MVP.
- Resource browsing is centered on `resource.graph`, not database-only tree terminology.
- Credentials stay local by default.

## What QueryBox is not

- Not a broad infrastructure provisioning console.
- Not an AI-first product surface.
- Not a mandatory cloud service or hosted control plane.

## Core workflow

1. Create or open a connection profile.
2. Authenticate through plugin-provided auth forms or a credential payload.
3. Browse resources via the resource graph.
4. Open an action or query path.
5. Inspect bounded results and, when supported, mutate rows or records.

For the deeper product and architecture framing, see [System overview](../architecture/system-overview.md).
