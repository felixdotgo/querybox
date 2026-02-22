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
- Connection tree nodes accept action objects where the `type` field is a machine name such as `"select"` or `"describe"`. Go authors can now use the constants defined in `pkg/plugin` (e.g. `plugin.ConnectionTreeAction_SELECT`) instead of hardcoding the strings.
- `pkg/plugin.ServeCLI` now marshals responses using `protojson` instead of `encoding/json`. After upgrading the SDK, **rebuild any existing plugin binaries** (e.g. `bash scripts/build-plugins.sh`) so they output correctly formatted JSON. The host side will also repair legacy output (missing or uppercase `Payload` fields) when decoding.
- To regenerate Go code after changing the proto, run `task proto:generate` (requires `protoc`, `protoc-gen-go`, `protoc-gen-go-grpc`).
- To implement CLI helpers in Go, use `pkg/plugin.ServeCLI` which now supports the `authforms` command.  The helper emits protobuf-style JSON using `protojson` so that oneof payloads are correctly named (older plugins which accidentally used `encoding/json` produced a  `"Payload"` field which the host could not parse).
- When adding fields/options, prefer conservative defaults so older UI versions still behave predictably.
