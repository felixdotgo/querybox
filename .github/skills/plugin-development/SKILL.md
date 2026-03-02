---
name: plugin-development
description: Guidance for building or updating a QueryBox plugin, including project structure, command implementation, and testing.
---

# Plugin Development Guide

This skill provides guidance for building or updating a QueryBox plugin, including project structure, command implementation, and testing.

## Overview

Plugins are small command-line programs invoked by the host. They must implement a JSON-over-stdin protocol defined in `contracts/plugin/v1/plugin.proto` and are placed in `bin/plugins` at runtime. The host calls individual commands (`info`, `exec`, `authforms`, etc.) and expects a single response before the process exits.

## When to use this skill
- Building a new plugin for a data source or service
- Adding features to an existing plugin (e.g. new commands, auth methods)
- Updating plugin dependencies or build configuration
- Writing tests for plugin functionality
- Debugging plugin issues (e.g. connection problems, query errors)

## Development Workflow

- Always reference existing examples under `plugins/` and the template directory.
- Suggest copying `plugins/template` and editing `main.go` with handler functions for the required commands.
- Build plugins with `task build:plugins` and verify binaries appear in `bin/plugins` (with `.exe` on Windows).
- Encourage writing tests under `plugins/<name>/*_test.go` and running `go test ./plugins/<name>`.
- Explain capabilities like `explain-query` and optional commands (`connection-tree`, `test-connection`).
- Remind that the host probes plugins on startup; a restart or manual rescan is needed after rebuilding.
- When adding new features to a plugin, update the metadata returned by `info` and document any new `options` flags.
- Keep the documentation up to date with any changes to the plugin contract or new commands added.
