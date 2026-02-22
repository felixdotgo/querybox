# QueryBox (Current under development - Not released)

**QueryBox** is a lightweight database management tool for executing and managing queries across multiple database systems through a plugin-based architecture.

## Features

- **Multi-database support** via plugin system (MySQL, PostgreSQL)
- **Secure credential storage** using system keychain
- **Connection management** with SQLite-backed persistence
- **Cross-platform** support (Windows, macOS, Linux)
- **Plugin-based architecture** for extensibility

## Getting Started

### Prerequisites

- Go 1.26 or higher
- [Wails v3](https://v3alpha.wails.io/) framework
- [Taskfile](https://taskfile.dev/) for build automation
- protoc and protoc-gen-go for gRPC code generation

### Development

Navigate to the project directory and run:

```bash
wails3 dev
```

This starts the application in development mode with hot-reloading for both frontend and backend changes.

### Building

Build the application for production:

```bash
wails3 build
```

This creates a production-ready executable.

Build plugins separately:

```bash
task build:plugins
# or directly:
bash scripts/build-plugins.sh
```

Plugin executables are placed in `bin/plugins/` and automatically discovered at runtime on a 2-second scan interval.

## Project Structure

```
├── main.go                     # Application entry point
├── services/                   # Core services
│   ├── app.go                  # Window management (App service)
│   ├── connection.go           # ConnectionService (CRUD + SQLite + keyring)
│   ├── events.go               # LogEntry / emitLog shared helpers
│   ├── credmanager/            # CredManager (OS keyring + sqlite + in-memory)
│   └── pluginmgr/              # PluginManager (discovery + execution)
├── pkg/plugin/                 # Plugin SDK — type aliases, ServeCLI helper
├── plugins/                    # Database driver plugins
│   ├── mysql/                  # MySQL driver plugin
│   ├── postgresql/             # PostgreSQL driver plugin
│   └── template/               # Plugin template / starter
├── contracts/plugin/v1/        # Protobuf source definitions
├── rpc/contracts/plugin/v1/    # Generated Go (pluginpb) code
├── frontend/                   # Vue.js frontend (Wails)
│   ├── src/
│   │   ├── views/              # UI views (Home, Connections)
│   │   └── components/         # Reusable components
│   └── bindings/               # Auto-generated TypeScript bindings
├── docs/                       # Architecture and design docs
├── scripts/                    # Build helper scripts
└── build/                      # Build configuration and platform assets
```

## Architecture

### Services

QueryBox follows a service-oriented architecture:

- **App Service**: Window lifecycle management (maximize, minimize, fullscreen, connections window)
- **ConnectionService**: CRUD operations for database connections, plus SQLite persistence and credential retrieval. Exposes `CreateConnection`, `ListConnections`, `GetConnection`, `GetCredential`, `DeleteConnection`.
- **PluginManager**: On-demand plugin discovery and execution. Exposes `ListPlugins`, `Rescan`, `ExecPlugin`, `GetPluginAuthForms`, `GetConnectionTree`, `ExecTreeAction`.
- **CredentialManager**: Secure credential storage with 3-tier fallback — OS keyring (`go-keyring`) → persistent sqlite file (`data/credentials.db`) → in-memory map.
- **Event System**: `app:log` Wails event carries `LogEntry{Level, Message, Timestamp}` from all services to the frontend.  Registered in `main.go` for typed TypeScript bindings.

### Plugin System

Plugins are standalone executables implementing a simple CLI interface:

- `plugin info` — Returns metadata (name, version, description, type)
- `plugin exec` — Reads `{connection, query}` JSON from stdin; writes a typed `ExecResponse` (sql / document / kv payload) to stdout
- `plugin authforms` — Returns structured authentication form definitions
- `plugin connection-tree` — Reads `{connection}` JSON from stdin; returns `{nodes:[…]}` describing a hierarchical tree of database objects with optional `actions` arrays

Plugins communicate via JSON stdin/stdout using proto-derived types from `rpc/contracts/plugin/v1`. See [pkg/plugin/plugin.go](pkg/plugin/plugin.go) for the full contract and type aliases.

## Development Workflow

1. **Modify frontend**: Edit files in [frontend/src/](frontend/src/)
2. **Add backend logic**: Update services in [services/](services/)
3. **Create plugins**: Use [plugins/template/](plugins/template/) as a starting point
4. **See changes**: `wails3 dev` for hot-reload
5. **Build**: `wails3 build` for production executable

## References

- [Wails v3 Documentation](https://v3alpha.wails.io/)
- [Plugin Development Guide](docs/plugins.md)
- [Architecture Overview](docs/detailed-design/architecture.md)
- [go-keyring](https://github.com/zalando/go-keyring) for credential management
