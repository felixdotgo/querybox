# Plugin System

## What it is

QueryBox extends system support through out-of-process plugins. Each plugin is a compiled executable plus a sidecar manifest.

## Current behavior

- Discovery is manifest-first.
- Runtime execution is one subprocess per request.
- The active browse contract is `resource.graph`.
- Plugins may implement `info`, `authforms`, `exec`, `resource-graph`, `test-connection`, and optional metadata or mutation commands.

## Key components

- `services/pluginmgr/`
- `pkg/plugin`
- `contracts/plugin/v1/plugin.proto`
- `plugins/*`

## How it connects to other modules

- Drives resource browsing in the frontend.
- Supplies auth forms for connection creation.
- Provides query execution, schema metadata, and optional mutation features.
- Constrains security behavior through manifest permissions and limits.

## Limits and roadmap

- No persistent plugin sessions in the current baseline.
- No strong sandbox boundary yet.
- Streaming and remote runtime support remain roadmap work.
