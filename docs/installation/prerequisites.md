# Prerequisites

## Runtime prerequisites

| Tool | Version | Purpose |
|------|---------|---------|
| Go | `>=1.26.1` | Backend and plugin builds |
| Node.js | `>=24` | Frontend tooling |
| Wails v3 CLI | current v3 alpha used by the repo | Desktop runtime and packaging |
| Task | latest | Build automation |

## Optional code generation prerequisites

You only need these when editing `contracts/plugin/v1/plugin.proto` or generated bindings:

| Tool | Version | Purpose |
|------|---------|---------|
| `protoc` | repo-compatible | Protocol buffer compiler |
| `protoc-gen-go` | `v1.36.10` | Go protobuf codegen |
| `protoc-gen-go-grpc` | `v1.6.1` | gRPC Go codegen |

## Platform notes

- QueryBox is developed as a local-first desktop app; Linux, macOS, and Windows packaging are supported through the `build/` tree.
- Linux release builds use the Wails v3 GTK3/WebKit2GTK 4.1 stack for the pinned Wails alpha: install `gcc`, `libgtk-3-dev`, `libwebkit2gtk-4.1-dev`, and `libsoup-3.0-dev` on Ubuntu/Debian build hosts.
- Plugin binaries are compiled for the host OS by default and bundled under `bin/plugins/`.
- OS keyring support affects the primary credential storage tier. When keyring support is unavailable, QueryBox falls back to local SQLite storage for secrets.
