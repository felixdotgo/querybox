# QueryBox Copilot Onboarding

Trust this file first. Only search the repo when this file is incomplete or proven wrong.

## Repo summary
- QueryBox is a **Wails v3 desktop app** (Go backend + Vue 3 frontend) for querying multiple databases via **out-of-process plugin executables**.
- Plugin binaries live in `bin/plugins/`; source is in `plugins/*`.
- Main runtime path: Wails app (`main.go`) → services (`services/*`) → plugin subprocesses (`services/pluginmgr/*`).
- Workspace size observed: ~`35149` files (includes dependencies/artifacts).

## Stack and required toolchain
- Go `1.26` (`.go-version`, `go.mod`)
- Node.js `18+` (validated here: `v22.14.0`)
- npm (validated here: `10.9.2`)
- Task CLI (`task`) for canonical workflows
- Wails CLI (`wails3`) for dev/build/bindings
- Optional for proto edits: `protoc`, `protoc-gen-go`, `protoc-gen-go-grpc`

## Architecture quick map (edit targets)
- Entrypoint: `main.go`
- Connection CRUD + metadata DB: `services/connection.go`
- Credential storage fallback logic: `services/credmanager/credmanager.go`
- Plugin discovery/execution: `services/pluginmgr/pluginmgr.go`
- Plugin SDK: `pkg/plugin/plugin.go`
- Contract: `contracts/plugin/v1/plugin.proto` → generated `rpc/contracts/plugin/v1/*`
- Frontend app: `frontend/src/*` (Vue 3 + Vite + Naive UI + Tailwind)
- Build/task config: `Taskfile.yml`, `build/Taskfile.yml`, `build/{windows,linux,darwin,ios,android}/Taskfile.yml`

## CI/checks reality
- No `.github/workflows/*` are present in this snapshot.
- Validation is task/script driven; run the checklist below before finishing changes.

## Command validation (Windows Git Bash in this workspace)
### Confirmed working
- `go version` → `go1.26.0 windows/amd64`
- `node -v` → `v22.14.0`
- `npm -v` → `10.9.2`
- `command -v protoc` → found
- `cd frontend && npm ci` → succeeded (~52s, 390 packages, 0 vulnerabilities)
- `go install github.com/go-task/task/v3/cmd/task@latest` → succeeded
- `$(go env GOPATH)/bin/task --version` → `3.48.0`
- `go test ./...` ran (~18.9s) but failed in known suites (below)

### Observed failures/caveats
- `task --version` fails if Task is not on PATH (`bash: task: command not found`).
- `wails3` was not present by default (`command -v wails3` missing).
- `/usr/bin/time` may be missing in Git Bash on Windows; use shell builtin `time`.
- `go test ./...` failures observed:
  - `services/credmanager`: `TestBackend_Memory` expected `memory`, got `sqlite`
  - `services/pluginmgr`: discovery/population tests (`expected plugins, got 0`, copy fallback issues)
  - Keyring warnings are expected in headless/non-keyring environments.
- Some task/wails/frontend lint-build invocations were skipped in this run; treat them as not re-validated here.

## Always-use command order (fastest reliable flow)
1. Install tools once:
   - `go install github.com/go-task/task/v3/cmd/task@latest`
   - `go install github.com/wailsapp/wails/v3/cmd/wails3@latest`
   - Ensure on PATH: `task --version`, `wails3 version`
2. Install frontend deps first: `cd frontend && npm ci`
3. **Always build plugins before running app/tests that depend on drivers**:
   - Preferred: `task build:plugins`
   - Fallback: `bash ./scripts/build-plugins.sh`
   - Verify artifacts in `bin/plugins/` (`*.exe` on Windows)
4. Run focused tests first, then broad:
   - `go test ./pkg/...`
   - `go test ./plugins/<name>...` or `go test ./services/<subpkg>...`
   - `go test ./...`
5. Frontend checks:
   - `cd frontend && npm run lint`
   - `cd frontend && npm run build`
6. Run app in dev mode:
   - `wails3 dev -config ./build/config.yml -port 9245`
7. If `.proto` changed:
   - `task proto:generate`
   - rebuild plugins afterward

## High-value gotchas
- Build plugins first; plugin-dependent features break if binaries are stale/missing.
- Keep `frontend/vite.config.js` host at `127.0.0.1` (Windows localhost IPv4/IPv6 mismatch mitigation).
- Plugin JSON must match protobuf-style `protojson` output.
- Credential backend is environment-dependent (keyring → sqlite → memory).
- Build logic is delegated: when changing build behavior, update platform taskfiles under `build/*/Taskfile.yml`.

## Pre-PR validation checklist
1. `npm ci` (if frontend touched or uncertain)
2. `task build:plugins` (or script fallback)
3. Targeted `go test` for changed packages
4. `go test ./...` (note known env-sensitive failures separately)
5. `npm run lint` and `npm run build` in `frontend/`
6. `wails3 dev` smoke run for UI/behavior changes

If any step is unavailable (missing tool, skipped execution), state it explicitly in the PR summary.

## Root + next-level layout (quick nav)
- Root: `.github/`, `bin/`, `build/`, `contracts/`, `docs/`, `frontend/`, `pkg/`, `plugins/`, `rpc/`, `scripts/`, `services/`, plus `main.go`, `Taskfile.yml`, `go.mod`, `README.md`.
- Key subdirs:
  - `services/`: `app.go`, `connection.go`, `events.go`, `credmanager/`, `pluginmgr/`
  - `frontend/src/`: `App.vue`, `main.js`, `components/`, `composables/`, `views/`
  - `plugins/`: per-database plugin implementations + `template/`
