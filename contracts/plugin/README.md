Plugin contracts (proto)

- Source: `contracts/plugin/v1/plugin.proto`
- Generated Go package: `rpc/contracts/plugin/v1` (package `pluginpb`). `pkg/plugin` exposes convenience aliases and `ServeCLI` for plugin authors.

Notes
- Runtime uses JSON CLI for on‑demand plugins, but the `.proto` is the canonical spec for messages and the service interface.
- `AuthForms` (new) lets plugins describe structured authentication forms the UI can render.
  - Response shape: `AuthFormsResponse { forms: map<string, AuthForm> }` where an `AuthForm` contains `fields` (type, name, label, required, placeholder, options).
  - Example response:

```json
{
  "forms": {
    "basic": {
      "key": "basic",
      "name": "Basic",
      "fields": [
        { "type": "TEXT", "name": "host", "label": "Host", "required": true, "placeholder": "127.0.0.1" },
        { "type": "PASSWORD", "name": "password", "label": "Password" }
      ]
    }
  }
}
```

- `ExecResponse` was enhanced to return a typed result envelope (`sql`, `document`, or `kv`) instead of a raw string. SQL results now include `Column` metadata with names and optional types.
- To regenerate Go code after changing the proto, run `task proto:generate` (requires `protoc`, `protoc-gen-go`, `protoc-gen-go-grpc`).
- To implement CLI helpers in Go, use `pkg/plugin.ServeCLI` which now supports the `authforms` command.
- When adding fields/options, prefer conservative defaults so older UI versions still behave predictably.
