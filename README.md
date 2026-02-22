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
```

Plugin executables are placed in `bin/plugins/` and automatically discovered at runtime.

## Project Structure

```
├── main.go                # Application entry point
├── services/              # Application services/features
├── pkg/plugin/            # Plugin SDK and contracts
├── plugins/               # Database driver plugins
│   ├── mysql/             # MySQL driver plugin
│   ├── postgresql/        # PostgreSQL driver plugin
│   └── template/          # Plugin template
├── contracts/plugin/      # Protobuf definitions
├── rpc/contracts/plugin/  # Generated gRPC code
├── frontend/              # Vue.js frontend
│   ├── src/
│   │   ├── views/         # UI views (Home, Connections)
│   │   └── components/    # Reusable components
│   └── bindings/          # Auto-generated TypeScript bindings
├── docs/                  # Architecture and design docs
└── build/                 # Build configuration and assets
```

## Architecture

### Services

QueryBox follows a service-oriented architecture:

- **App Service**: Window lifecycle management (maximize, minimize, fullscreen)
- **ConnectionService**: CRUD operations for database connections
- **ConnectionManager**: SQLite-backed connection persistence
- **PluginManager**: On-demand plugin discovery and execution
- **CredentialManager**: Secure credential storage via `go-keyring` with sqlite-file fallback

### Plugin System

Plugins are standalone executables implementing a simple CLI interface:

- `plugin info` - Returns metadata (name, version, description)
- `plugin exec` - Executes query against database
- `plugin authforms` - Provides authentication form definitions

Plugins communicate via JSON stdin/stdout. See [pkg/plugin/plugin.go](pkg/plugin/plugin.go) for the contract.

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
