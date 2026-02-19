# Plugins

Plugins are out-of-process executables placed under `bin/plugins/` and invoked on-demand by the host.

- Development: put plugin source under `plugins/<name>/` (must be `package main`) and run `task build:plugins`.
- Runtime: host discovers executables in `./bin/plugins` (no persistent plugin processes). The host may invoke a plugin binary whenever the user requests an operation.
- Contract (CLI): the plugin should implement two subcommands:
  - `plugin info` → prints JSON `{name, version, description}`
  - `plugin exec` → reads JSON `{connection, query}` from stdin and writes JSON `{result, error}` to stdout

See `plugins/template` for a minimal example that follows the on-demand contract.
