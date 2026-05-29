# Local Setup

## Install dependencies

```bash
go install github.com/go-task/task/v3/cmd/task@latest
go install github.com/wailsapp/wails/v3/cmd/wails3@latest
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.10
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.6.1
```

Install `protoc` separately only if you will modify `.proto` files.

## Clone and start development

```bash
git clone https://github.com/your-username/querybox.git
cd querybox
task build:plugins
wails3 dev
```

## Local development flow

1. Build bundled plugins with `task build:plugins`.
2. Start the desktop app with `wails3 dev`.
3. Iterate on frontend and backend code with hot reload.
4. Rebuild plugins when plugin source or manifests change.

## Build outputs

- App binary: produced by `wails3 build`, placed under `bin/`.
- Plugin binaries: produced by `task build:plugins`, placed under `bin/plugins/`.
- Plugin manifests: copied beside each plugin binary as `<binary>.manifest.json`.
