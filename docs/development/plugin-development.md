# Plugin Development

## Authoring model

Plugins are standalone executables. QueryBox invokes one subprocess per request, passes JSON on stdin, reads a proto-JSON response from stdout, and then the process exits.

## Required artifacts

1. `plugins/<name>/main.go`
2. `plugins/<name>/plugin.json`
3. Built binary under `bin/plugins/`
4. Sidecar manifest `<binary>.manifest.json`

## Current contract

- Discovery is manifest-first.
- `resource.graph` is the active browse contract.
- Capability names must follow the current taxonomy.
- Implement only the commands your manifest advertises.

## Minimum authoring flow

```bash
cp -r plugins/template plugins/<your-plugin-name>
task build:plugins
```

Then edit:

- `plugin.json`: id, runtime, capabilities, permissions, limits, metadata.
- `main.go`: command handlers and plugin service implementation.

## Guardrails

- Do not advertise capabilities that are not implemented.
- Do not swallow connection, auth, bootstrap, or query errors.
- Keep outputs bounded by timeout and output-size limits.
- Close rows, files, and handles; avoid hidden background work.

## Related docs

- [Plugin system](../features/plugin-system.md)
- [Plugin capabilities reference](../reference/plugin-capabilities.md)
- [Runtime and plugin lifecycle](../architecture/runtime-and-plugin-lifecycle.md)
