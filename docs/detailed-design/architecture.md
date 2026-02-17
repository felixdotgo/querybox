# Architecture — detailed diagram

## Diagram

```mermaid
graph LR
    subgraph DriverPool ["Drivers (stateless)"]
        LAUNCH["Driver launcher<br/>(timeout, mem caps)"]
        DRV["Driver process<br/>(stateless, gRPC)"]
        DB[(Remote DB)]
    end

    subgraph Core ["Core (Go)"]
        CORE["Core Service<br/>(Wails bridge for UI)"]
        STORE[(Core backing store)]
        MK["Master key<br/>(env secret manager)"]
        KR["OS Keyring<br/>(go-keyring)"]
        AUD[Logs & telemetry]
    end

    subgraph Frontend ["Frontend (Wails)"]
        FE[Wails]
    end

    FE -->|store/retrieve password via Core (Wails bridge)| CORE
    FE -->|Execute request: session id| CORE

    CORE -->|load metadata| STORE
    CORE -->|decrypt creds - AES-256-GCM, in-memory only| MK
    CORE -->|access OS keyring via go-keyring| KR
    CORE -->|spawn driver - resource limits| LAUNCH
    LAUNCH -->|start: session id| DRV

    CORE -->|Execute gRPC: session id, creds, query| DRV
    DRV -->|open DB connection - uses creds| DB
    DRV -->|stream rows/errors - gRPC| CORE

    CORE -->|stream results to frontend| FE
    CORE -->|scrub creds & invalidate session| CORE
    DRV -->|exit after request| LAUNCH

    CORE -->|audit: session id only| AUD

```

---

## Query & credential flow (numbered)

1. Core stores per-user connection passwords in the OS keyring via `go-keyring` (macOS Keychain, Windows Credential Manager, Linux Secret Service). The Frontend MUST call Core (via the Wails bridge) to add or retrieve per-user credentials — the frontend never talks to the OS keyring directly.
2. User triggers an Execute in the Frontend; Frontend requests Core to execute a named connection and includes a generated session id.
3. Core resolves credentials (deployment-dependent):
   - Desktop / co-located Core: Core reads the per-user password from the OS keyring (via `go-keyring`) when Core and Frontend run on the same host.
   - Remote / server Core: Core reads the AES-256-GCM-encrypted credential blob from the Core backing store and decrypts it in-memory using the master key supplied by the environment/secret-manager. The Frontend MUST NOT write plaintext secrets to disk; if Core cannot access an OS keyring the Frontend should securely transmit credentials to Core over the protected channel or require admin provisioning.
   After resolving credentials, Core launches a stateless driver process with enforced time and memory limits.
4. Core calls the driver over gRPC with session id and decrypted credentials; driver opens the DB connection and streams rows/errors back over gRPC.
5. Core forwards streamed rows to the Frontend, scrubs credentials from memory, invalidates the session id, and records only the session id in logs for traceability.

---

## Notes & security callouts 🔐

- Core MUST use `go-keyring` for OS keyring access; the Frontend MUST call Core (via the Wails bridge) to add/retrieve per-user passwords. On desktop installs where Core and Frontend are co-located, Core may read/write the OS keyring on behalf of the user; otherwise Core uses the encrypted credential blob in its backing store.
- Core continues to persist only AES-256-GCM-encrypted credential blobs and never stores plaintext secrets on disk.
- Headless/server installs MUST use environment/secret-manager provisioning for Core master keys — document explicit admin opt-in for any disk-backed fallback.
- Drivers are stateless and never persist plaintext credentials; runtime protections (timeout, memory caps, non-root) apply to launched driver processes.
- Plugin manager: the host discovers on‑demand plugin executables under `./bin/plugins`, probes `plugin info` for metadata, and exposes `ListPlugins`, `Rescan`, and `ExecPlugin` via the Wails bridge. The canonical proto is `contracts/plugin/v1/plugin.proto` (generated `pluginpb`) and `pkg/plugin` provides `ServeCLI` helpers.
- Tests to add: keyring availability and permission-denied behavior (desktop), Core fallback when keyring unavailable (server/headless), and credential-migration flows into the OS keyring via Core (Wails bridge).
- Audit logs retain session ids only. Continue to require secure master-key provisioning and rotation for Core.

> Tip: diagram and flow reflect the MVP trust model — plan container isolation and further OS hardening for post-MVP.
