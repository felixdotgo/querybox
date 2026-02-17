Plugin contracts (proto)

- Source: `contracts/plugin/v1/plugin.proto`
- Generated Go package: `rpc/contracts/plugin/v1` (package `pluginpb`). `pkg/plugin` exposes convenience aliases and `ServeCLI` for plugin authors.

Notes
- Runtime uses JSON CLI for on‑demand plugins, but the `.proto` is the canonical spec for messages and the service interface.
- To produce canonical generated Go code with `protoc`, run `task proto:generate` (requires `protoc`, `protoc-gen-go`, `protoc-gen-go-grpc`).
