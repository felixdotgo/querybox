# QueryBox Runtime Evolution Proposal

**Version:** 0.1  
**Date:** 2026-05-08  
**Scope:** Technical proposal for evolving QueryBox from a plugin-based database client into a general-purpose operational data workspace without AI as a core product direction.

> Status note (2026-05-10): Phase 0 and Phase 1 items from this proposal have already been implemented in the repo. Section 9 below is the active roadmap rebased on the current baseline; earlier sections remain as design rationale.

---

## 1. Executive Summary

QueryBox currently has a strong foundation for a lightweight, extensible desktop data tool:

- Desktop application built with **Wails v3**, **Go backend**, and **Vue 3 frontend**.
- Plugin-based architecture where plugins are standalone executables.
- Plugin discovery from bundled and user plugin directories.
- Connection metadata stored locally, credentials stored through OS keyring with fallback layers.
- Current plugin workflow is command-oriented: `info`, `exec`, `authforms`, `resource-graph`, and `test-connection`, with some legacy commands still present on older plugins.
- Plugins are currently executed on-demand as subprocesses, using JSON stdin/stdout with protobuf-defined contracts.

This proposal recommends evolving QueryBox into a broader **operational runtime for databases, infrastructure, and distributed systems**, while keeping the product lightweight and non-AI-first.

The core idea is to move from:

```text
Plugin = database driver
```

To:

```text
Plugin = capability provider running inside a controlled runtime
```

This unlocks non-database integrations such as Redis, Kafka, S3, Kubernetes, CloudWatch Logs, Prometheus, REST APIs, GraphQL, gRPC reflection, Docker, and future remote/cloud execution.

The most important technical changes proposed are:

1. Introduce a **Capability Model**.
2. Replace database-specific assumptions with a **Universal Resource Graph**.
3. Evolve the plugin host into a **Runtime Manager**.
4. Add support for **long-running plugin sessions** where needed.
5. Add **streaming-first APIs** for logs, events, metrics, and message queues.
6. Define a **sandbox and permission model** before a third-party plugin ecosystem grows.
7. Introduce **Local / Remote / Cloud runtime modes** behind the same protocol.
8. Keep the current on-demand subprocess model for simple operations to preserve simplicity.

---

## 2. Current State Assessment

### 2.1 What QueryBox already does well

QueryBox already has several design choices that are good foundations for a future runtime platform.

#### Process-based plugin isolation

Plugins are standalone executables. This is a strong architectural decision because it provides:

- Crash isolation.
- Language independence.
- Easier plugin distribution.
- Simpler dependency isolation.
- Lower risk compared with in-process plugin loading.

#### On-demand execution

Current plugins are not always-on. They are spawned per operation, given input, return output, and terminate.

This works well for:

- `plugin info`
- `test-connection`
- schema/tree loading
- simple query execution
- stateless actions

Benefits:

- Low idle memory usage.
- No plugin lifecycle complexity.
- Easier cleanup on timeout.
- Easy failure containment.

#### Local-first credential management

Connection metadata is separated from credentials. Credentials are stored using OS keyring first, then fallback storage if needed.

This is a good baseline for a developer tool because it avoids forcing users into cloud sync early.

#### Clear desktop/backend/frontend separation

The current separation between frontend, Go services, and plugin manager is useful. It allows QueryBox to evolve the runtime without rewriting the entire UI.

---

### 2.2 Current limitations

The current architecture is good for database tools but will become limiting if QueryBox expands to infrastructure, logs, message queues, object storage, and remote execution.

#### Limitation 1: Plugin equals database connector

The current product model is still centered around database connections and database actions.

This becomes awkward for non-database systems:

- Kafka topics are not tables.
- S3 buckets are not schemas.
- Kubernetes pods are not database objects.
- Prometheus queries return time series, not rows.
- Logs are streams, not query result sets.

#### Limitation 2: Tree model is database-oriented

The current `connection-tree` command works for database browsing, but it needs to become more generic.

A future tree should represent arbitrary resources:

```text
postgres://prod
  ├── databases
  ├── schemas
  ├── tables

k8s://cluster-a
  ├── namespaces
  ├── pods
  ├── services

s3://company-data
  ├── buckets
  ├── prefixes
  ├── objects

kafka://prod
  ├── clusters
  ├── topics
  ├── consumer groups
```

#### Limitation 3: No first-class streaming model

For logs, Kafka, Redis Streams, NATS, metrics, and traces, QueryBox needs a streaming abstraction.

A synchronous `exec -> result` model is not enough for:

- tailing logs
- consuming Kafka messages
- watching Kubernetes pod events
- live metrics queries
- long-running operational views

#### Limitation 4: On-demand subprocesses are not always enough

On-demand execution is excellent for stateless calls, but inefficient for operations that need persistent state:

- SSH tunnels
- database connection pooling
- Kafka consumers
- log tailing
- Kubernetes watches
- long-running remote sessions

#### Limitation 5: Plugin security is currently coarse-grained

Running plugins as separate processes gives crash isolation, but not a full security boundary.

A plugin process can still potentially:

- read local files
- open arbitrary network connections
- spawn child processes
- consume CPU/memory
- access environment variables
- leak credentials if malicious or compromised

This may be acceptable for first-party plugins, but not for a marketplace or third-party ecosystem.

#### Limitation 6: Remote/cloud execution is not yet modeled

Today the desktop app appears to execute plugins locally. That is simple, but limits usage in production environments where data sources are often inside private networks.

Common problems:

- Developer laptop cannot access VPC resources.
- VPN setup is painful.
- Bastion hosts are inconsistent.
- SSH tunnels are duplicated across tools.
- Secrets should not always leave the private network.
- Some operations should run close to the data source for latency/security.

---

## 3. Product Direction

### 3.1 Proposed positioning

QueryBox should not be positioned only as a database GUI.

Recommended positioning:

> QueryBox is a lightweight operational workspace for databases, infrastructure, and distributed systems.

Alternative developer-facing wording:

> QueryBox is a local-first, plugin-based runtime for querying, browsing, and operating data systems.

This keeps the current database use case but expands the product surface to non-database systems.

---

### 3.2 What QueryBox should become

QueryBox should become a unified workspace where backend engineers can inspect and operate multiple systems from one interface:

```text
Operational Workspace
  ├── PostgreSQL query tab
  ├── Redis key inspector
  ├── Kafka topic viewer
  ├── Kubernetes pod logs
  ├── S3 object browser
  ├── Prometheus metrics panel
  └── Notes / saved investigation context
```

This is not a replacement for every specialized tool. Instead, QueryBox should focus on the common daily workflows of backend engineers:

- inspect data
- debug incidents
- trace production behavior
- query systems safely
- compare data across systems
- save operational context
- switch between systems without switching tools

---

## 4. Proposed Architecture

### 4.1 High-level target architecture

```text
┌─────────────────────────────────────────────┐
│                QueryBox UI                  │
│        Vue / Wails frontend layer           │
└──────────────────────┬──────────────────────┘
                       │
┌──────────────────────▼──────────────────────┐
│              QueryBox Core Services          │
│  ConnectionService / WorkspaceService / etc. │
└──────────────────────┬──────────────────────┘
                       │
┌──────────────────────▼──────────────────────┐
│              Runtime Manager                 │
│  local runtime / remote runtime / cloud      │
└───────────────┬───────────────┬─────────────┘
                │               │
     ┌──────────▼───────┐  ┌────▼────────────┐
     │ Local PluginHost │  │ Remote Agent    │
     └──────────┬───────┘  └────┬────────────┘
                │               │
     ┌──────────▼───────────────▼────────────┐
     │             Plugin Processes           │
     │ DB / Kafka / Redis / S3 / K8s / Logs   │
     └────────────────────────────────────────┘
```

The UI should not need to know whether a plugin is running:

- locally
- inside a remote agent
- inside a cloud runtime
- as native process
- as WASM module in the future

The Runtime Manager should abstract that away.

---

## 5. Important Technical Changes

## 5.1 Introduce a Capability Model

### Problem

The current plugin model is command-oriented and database-oriented. As QueryBox supports more systems, a plugin should not be forced into a database-shaped API.

### Proposal

Each plugin should declare capabilities in its manifest or `info` response.

Example:

```json
{
  "id": "querybox.plugin.kafka",
  "name": "Kafka",
  "version": "0.1.0",
  "runtime": "native-process",
  "capabilities": [
    "resource.graph",
    "stream.read",
    "message.publish",
    "connection.test"
  ],
  "permissions": [
    "network.outbound",
    "credential.read"
  ]
}
```

Possible capabilities:

```text
connection.test        Test whether a connection profile works
resource.graph         Return browsable resources
query.execute          Execute request/query and return finite result
stream.read            Subscribe to streaming results
stream.write           Publish/write streaming messages
file.read              Read object/file content
file.write             Write object/file content
command.execute        Execute plugin-defined action
schema.inspect         Return schema/introspection metadata
metric.query           Query time-series metrics
log.query              Query historical logs
log.tail               Tail live logs
transaction.run        Run transactional operation
migration.plan         Produce migration/diff plan
```

### Benefits

- Decouples plugin design from databases.
- Lets the UI adapt based on plugin capability.
- Makes third-party plugins easier to understand.
- Provides a foundation for permissions and sandboxing.
- Enables future marketplace filtering.

### Trade-offs

- Requires more upfront protocol design.
- Poorly designed capabilities can become hard to change later.
- Too many capabilities too early can overcomplicate the SDK.

### Recommendation

Start with a small stable set:

```text
resource.graph
query.execute
stream.read
connection.test
schema.inspect
```

Then add more as real plugins require them.

---

## 5.2 Replace Connection Tree with Universal Resource Graph

### Problem

`connection-tree` is useful for databases, but non-database resources need a more generic representation.

### Proposal

Replace or evolve `connection-tree` into `resource.graph`.

A resource node should be generic:

```json
{
  "id": "postgres://prod/public/users",
  "name": "users",
  "kind": "table",
  "path": ["prod", "public", "users"],
  "icon": "table",
  "children": [],
  "actions": ["open", "query", "describe", "export"],
  "metadata": {
    "schema": "public",
    "rowEstimate": 1200000
  }
}
```

For Kafka:

```json
{
  "id": "kafka://prod/topics/orders.created",
  "name": "orders.created",
  "kind": "topic",
  "actions": ["consume", "publish", "describe"],
  "metadata": {
    "partitions": 12,
    "replicationFactor": 3
  }
}
```

For Kubernetes:

```json
{
  "id": "k8s://prod/namespaces/default/pods/api-7f9d",
  "name": "api-7f9d",
  "kind": "pod",
  "actions": ["logs", "describe", "shell", "port-forward"],
  "metadata": {
    "namespace": "default",
    "status": "Running"
  }
}
```

### Benefits

- One explorer UI for all systems.
- Plugin-specific resources without hardcoding database concepts.
- Enables context menus/actions per resource.
- Helps future workspace/session features.

### Trade-offs

- Generic graph can become too abstract.
- UI may need plugin-provided hints for icons, grouping, and actions.
- Need stable resource IDs and path conventions.

### Recommendation

Phase 1 already standardized bundled plugins on `resource.graph` as the target browse contract.

```text
resource.graph = target contract
connection-tree = compatibility-only surface when a legacy plugin still needs it
```

Do not keep the adapter path as the center of the roadmap. Any remaining legacy support should be treated as bounded migration debt, not the destination architecture.

---

## 5.3 Split Plugin Execution into Stateless Operations and Stateful Sessions

### Problem

Current on-demand subprocess execution is simple and effective, but long-running operations need session state.

### Proposal

Support two execution modes:

```text
Mode A: Stateless Operation
  spawn plugin -> execute command -> return result -> exit

Mode B: Stateful Session
  start plugin session -> keep process alive -> multiplex requests/streams -> stop session
```

Use stateless mode for:

- plugin info
- auth forms
- test connection
- one-shot queries
- resource tree loading

Use stateful mode for:

- SSH tunnel
- Kubernetes watch
- Kafka consumer
- Redis monitor
- log tailing
- database connection pooling
- long-running export/import

### Benefits

- Preserves current simplicity.
- Adds efficient long-running workflows.
- Avoids forcing every plugin to become a daemon.
- Enables streaming and remote execution.

### Trade-offs

- Session lifecycle is harder than one-shot execution.
- Need cancellation, heartbeat, cleanup, and error propagation.
- Long-running plugins can leak memory or hold stale credentials.
- UI must handle reconnect/resume behavior.

### Recommendation

Do not replace on-demand plugin execution. Add sessions as an optional capability:

```text
session.start
session.stop
session.heartbeat
session.attach
```

A plugin declares whether it supports sessions.

---

## 5.4 Add Streaming-first Protocol

### Problem

Many important non-database systems are streaming systems.

Examples:

- Kafka messages
- NATS messages
- Redis streams
- Kubernetes events
- container logs
- CloudWatch live tail
- Prometheus range queries

A simple request/response protocol will not be enough.

### Proposal

Introduce a stream abstraction:

```text
StreamRequest
  connection_id
  resource_id
  stream_type
  filter
  cursor
  limit

StreamEvent
  stream_id
  sequence
  timestamp
  payload
  metadata
```

Stream types:

```text
log.tail
message.consume
metric.watch
resource.watch
data.export
```

The UI should treat streams as first-class panels.

### Benefits

- Enables logs, messages, metrics, events.
- Supports real-time operational workflows.
- Allows consistent pause/resume/cancel behavior.
- Works well with remote agents.

### Trade-offs

- More complex than request/response.
- Need backpressure and buffering strategy.
- Need clear limits to avoid memory growth.
- UI must handle high-volume output.

### Recommendation

Start with a minimal streaming MVP:

- start stream
- receive events
- cancel stream
- plugin-side limit
- UI-side pause
- max buffer size

Avoid complex replay semantics until there are real use cases.

---

## 5.5 Introduce Runtime Manager

### Problem

The current PluginManager is responsible for plugin discovery and execution. As QueryBox grows, it will need to manage more runtime concerns.

### Proposal

Evolve `PluginManager` into or complement it with `RuntimeManager`.

Responsibilities:

```text
Plugin discovery
Plugin metadata registry
Capability registry
Permission checks
Runtime selection
Process/session lifecycle
Timeouts and cancellation
Stream lifecycle
Remote agent routing
Crash recovery
Runtime logs
Resource limits
```

Suggested internal split:

```text
PluginRegistry
  - discover plugins
  - validate manifest/info
  - track versions

RuntimeManager
  - choose local/remote/cloud runtime
  - enforce permissions
  - create operation/session/stream

LocalPluginHost
  - spawn subprocess
  - manage stdin/stdout
  - monitor resource usage

RemoteRuntimeClient
  - talk to remote QueryBox Agent
  - forward operation/session/stream requests

PermissionManager
  - validate plugin permissions against user policy
```

### Benefits

- Avoids one huge PluginManager.
- Creates clean path toward remote/cloud execution.
- Makes sandboxing easier to reason about.
- Separates plugin metadata from plugin execution.

### Trade-offs

- More services and internal APIs.
- Requires careful migration from current PluginManager.
- Overengineering risk if implemented too early.

### Recommendation

Refactor incrementally:

1. Extract `PluginRegistry` from current plugin discovery logic.
2. Keep existing subprocess execution as `LocalPluginHost`.
3. Add `RuntimeManager` facade above existing execution path.
4. Add sessions/streams after the facade exists.

---

## 5.6 Add Plugin Manifest and Permission Model

### Problem

A plugin ecosystem requires trust boundaries. Current process isolation is useful but not enough.

### Proposal

Each plugin should ship with a manifest.

Example:

```yaml
id: querybox.plugin.postgres
name: PostgreSQL
version: 0.1.0
publisher: querybox-core
runtime:
  type: native-process
  entrypoint: querybox-plugin-postgres
capabilities:
  - connection.test
  - resource.graph
  - query.execute
  - schema.inspect
permissions:
  - credential.read
  - network.outbound
  - filesystem.none
limits:
  timeout_seconds: 30
  memory_mb: 256
```

Permission examples:

```text
credential.read           Read credentials explicitly passed by QueryBox
network.outbound          Open outbound network connections
network.localhost         Connect only to localhost
filesystem.read           Read selected files
filesystem.write          Write selected files
process.spawn             Spawn child processes
env.read                  Read environment variables
clipboard.read            Read clipboard
clipboard.write           Write clipboard
```

### Benefits

- Clear trust model.
- Better user visibility.
- Safer plugin installation.
- Foundation for marketplace review.
- Easier enterprise adoption.

### Trade-offs

- Native processes are hard to restrict cross-platform.
- Permissions can create false confidence if not enforced.
- Users may blindly approve permissions.
- Fine-grained security can slow development.

### Recommendation

Use staged enforcement:

```text
Phase 1: Declare permissions, show them to users, no hard sandbox.
Phase 2: Enforce coarse controls: timeout, working directory, environment, credential passing.
Phase 3: Enforce OS-level sandbox where possible.
Phase 4: Support WASM plugins for stronger sandboxing.
```

---

## 5.7 Add Sandboxing Levels

### Problem

Third-party plugins introduce risk.

### Proposal

Define multiple sandbox levels instead of trying to solve everything at once.

### Level 0: Trusted native plugin

Current behavior.

```text
Best for: first-party plugins, development mode
Security: low
Complexity: low
```

### Level 1: Managed native plugin

Host controls:

- timeout
- working directory
- environment variables
- credential injection
- stdout/stderr size
- max output size
- process cleanup

```text
Best for: bundled plugins
Security: medium-low
Complexity: low-medium
```

### Level 2: OS sandboxed plugin

Use platform-specific isolation.

Linux:

- cgroups
- namespaces
- seccomp
- AppArmor/SELinux where available

macOS:

- App Sandbox where feasible
- sandbox-exec only as limited/non-future-proof fallback

Windows:

- Job Objects
- restricted tokens
- AppContainer where feasible

```text
Best for: stronger native plugin control
Security: medium
Complexity: high
```

### Level 3: WASM plugin

Run plugin logic inside WASM runtime such as wazero or wasmtime.

```text
Best for: untrusted marketplace plugins, portable plugins
Security: high
Complexity: medium-high
```

### Trade-offs

Native sandboxing is powerful but inconsistent across OSes. WASM gives better portability and isolation but may not fit all integrations, especially those needing mature native client libraries.

### Recommendation

Use a hybrid model:

```text
First-party/core plugins: managed native process
Third-party simple plugins: WASM preferred
Advanced infrastructure plugins: native process with explicit trust warning
```

---

## 5.8 Support WASM Plugin Runtime

### Problem

Native executable plugins are flexible but hard to sandbox and distribute safely.

### Proposal

Add optional WASM plugin support.

Potential Go runtimes:

- `wazero`: pure Go runtime, easier embedding and cross-platform distribution.
- `wasmtime`: mature runtime, strong performance, but external dependency footprint may be larger.

WASM plugins would use a constrained host API:

```text
host.log(level, message)
host.getCredential(key)
host.httpRequest(request)
host.emitResource(node)
host.emitResult(result)
host.emitStreamEvent(event)
```

### Benefits

- Stronger sandboxing.
- Portable plugin artifacts.
- Safer third-party plugin distribution.
- Easier marketplace story.
- Lower risk of plugin reading arbitrary local files.

### Trade-offs

- Harder SDK design.
- Networking, TLS, database drivers, and native libraries are harder in WASM.
- Debugging experience may be weaker.
- Some plugins will still need native execution.

### Recommendation

Do not migrate existing database plugins to WASM immediately.

Use WASM first for:

- API integrations
- simple HTTP-based plugins
- data transformers
- formatters/renderers
- lightweight resource providers

Keep native plugins for:

- PostgreSQL/MySQL drivers
- Kubernetes
- Docker
- SSH/SSM tunnels
- heavy infrastructure integrations

---

## 5.9 Remote Runtime / QueryBox Agent

### Problem

Local-only execution breaks down when data sources are inside private networks.

### Proposal

Introduce a QueryBox Agent that can run close to the data source.

```text
Developer Laptop
  └── QueryBox Desktop
        └── secure connection
              └── QueryBox Agent
                    ├── Postgres plugin
                    ├── Redis plugin
                    ├── Kafka plugin
                    ├── K8s plugin
                    └── Logs plugin
```

Agent responsibilities:

- Run plugins in private network.
- Store or access credentials locally in that environment.
- Execute operations on behalf of the desktop client.
- Stream logs/events/results back to client.
- Enforce runtime policy.
- Provide audit logs.

Connection options:

```text
Desktop -> Agent over HTTPS/gRPC
Desktop -> Agent through SSH tunnel
Desktop -> Agent through reverse tunnel
Desktop -> Cloud broker -> Agent
```

### Benefits

- Avoids local VPN complexity.
- Keeps secrets close to infrastructure.
- Reduces latency to data sources.
- Enables team/shared runtime later.
- Opens path to cloud execution.

### Trade-offs

- Requires agent installation and upgrade flow.
- Adds auth/security complexity.
- Adds distributed system failure modes.
- Needs version compatibility between desktop and agent.
- Debugging becomes harder.

### Recommendation

Build agent support in three stages:

#### Stage 1: Experimental local agent

Agent runs on same machine but uses remote protocol.

Purpose:

- Validate protocol boundaries.
- Avoid real networking/security complexity initially.

#### Stage 2: Self-hosted remote agent

User deploys agent manually:

```bash
querybox-agent serve --config agent.yaml
```

Supports:

- static token auth
- TLS
- local plugin execution
- logs

#### Stage 3: Managed cloud broker

Optional future paid/hosted layer:

- agent registration
- reverse tunnel
- team access
- audit logs
- policy management

---

## 5.10 Cloud Execution

### Problem

Some operations should run even when the desktop app is closed.

Examples:

- scheduled query
- recurring health check
- continuous log watch
- background export
- alert condition
- shared workspace refresh

### Proposal

Cloud execution should be treated as a runtime target, not a separate product bolted on later.

```text
RuntimeTarget:
  - local
  - remote-agent
  - cloud
```

Cloud runtime capabilities:

```text
scheduled operation
background stream
shared credentials/policies
team workspace state
audit logs
notification hooks
```

### Benefits

- Monetization path.
- Team workflows.
- Production monitoring workflows.
- Long-running operations without desktop dependency.

### Trade-offs

- Highest security burden.
- Requires account system, billing, auth, secrets, tenancy.
- Changes QueryBox from local tool to cloud platform.
- May distract from open-source core too early.

### Recommendation

Do not build cloud execution first.

Design the runtime interface so cloud execution is possible later, but prioritize:

1. Local runtime.
2. Local session/stream model.
3. Self-hosted remote agent.
4. Cloud broker/runtime only after real demand.

---

## 5.11 Workspace and Session Model

### Problem

Operational debugging often spans multiple resources and tabs.

A user may open:

- PostgreSQL query
- Redis key
- Kafka topic
- Kubernetes logs
- notes
- exported result

Without workspace state, context is lost.

### Proposal

Introduce workspace files or local workspace records.

```yaml
workspace:
  id: incident-checkout-latency
  name: Checkout Latency Investigation
  resources:
    - postgres://prod/public/orders
    - kafka://prod/topics/orders.created
    - k8s://prod/default/deployments/checkout-api
  tabs:
    - type: query
      connection: postgres-prod
      content: SELECT ...
    - type: stream
      connection: kafka-prod
      topic: orders.created
    - type: logs
      connection: k8s-prod
      selector: app=checkout-api
  notes:
    path: notes.md
```

### Benefits

- Strong developer workflow.
- Useful without AI.
- Can be local-first and Git-friendly.
- Becomes foundation for collaboration later.

### Trade-offs

- More state management.
- Need stable resource identifiers.
- Sensitive data may accidentally enter workspace files.

### Recommendation

Start local-only:

- saved tabs
- saved query snippets
- saved resource references
- optional notes

Avoid team sync until local workflow is polished.

---

## 6. Non-Database Plugin Roadmap

### 6.1 Recommended plugin categories

#### Tier 1: Very aligned with backend engineers

```text
Redis
Kafka
S3 / MinIO
Docker
Kubernetes
Prometheus
Loki
Elasticsearch / OpenSearch
```

#### Tier 2: Useful but more specialized

```text
NATS
RabbitMQ
SQS
CloudWatch Logs
BigQuery
ClickHouse
DynamoDB
GraphQL
REST collections
```

#### Tier 3: Ecosystem/marketplace candidates

```text
Stripe
Shopify
Supabase
Firebase
Notion
Airtable
HubSpot
Linear
GitHub Issues
```

---

### 6.2 Suggested first non-database plugins

#### Redis

Why first:

- Already close to database mental model.
- Simple resource graph: databases, keys, streams, pubsub.
- Useful to backend engineers.

Capabilities:

```text
resource.graph
query.execute
stream.read
command.execute
```

#### S3 / MinIO

Why second:

- Object storage is common.
- Resource graph maps naturally to buckets/prefixes/objects.
- Useful for SaaS/backend debugging.

Capabilities:

```text
resource.graph
file.read
file.write
query.execute
```

#### Kafka

Why third:

- Strong differentiator.
- Forces streaming architecture.
- Valuable operational workflow.

Capabilities:

```text
resource.graph
stream.read
stream.write
schema.inspect
```

#### Kubernetes

Why later:

- Very powerful but more complex.
- Requires permissions, streaming, logs, watches, port-forwarding.

Capabilities:

```text
resource.graph
log.tail
resource.watch
command.execute
tunnel.open
```

---

## 7. Key Trade-offs

## 7.1 Simplicity vs Platform Capability

### Option A: Stay simple database GUI

Pros:

- Easier roadmap.
- Easier UX.
- Faster to polish.
- Lower security burden.

Cons:

- Competes directly with mature tools.
- Smaller differentiation.
- Plugin system may be underused.

### Option B: Become operational runtime

Pros:

- Strong differentiation.
- Better fit for backend engineers.
- Opens future remote/cloud/team features.
- Plugin ecosystem becomes meaningful.

Cons:

- More architecture complexity.
- Harder UX design.
- More security concerns.
- Longer time to maturity.

### Recommendation

Choose Option B, but implement it incrementally and keep the default user experience simple.

---

## 7.2 Native Plugins vs WASM Plugins

### Native plugins

Pros:

- Maximum compatibility.
- Easy to use existing Go/database/cloud SDKs.
- Better for infrastructure integrations.
- Easier for current architecture.

Cons:

- Harder to sandbox.
- Cross-platform packaging complexity.
- Higher security risk.

### WASM plugins

Pros:

- Better sandboxing.
- Portable artifact.
- Better marketplace safety.
- Controlled host API.

Cons:

- Harder SDK.
- Limited native library support.
- Harder networking and TLS cases.
- Not ideal for every plugin.

### Recommendation

Use both:

```text
Native = trusted/complex integrations
WASM = untrusted/simple integrations
```

---

## 7.3 On-demand Execution vs Long-running Runtime

### On-demand subprocess

Pros:

- Simple.
- Safe cleanup.
- Low idle memory.
- Good for one-shot commands.

Cons:

- Inefficient for repeated calls.
- No connection pooling.
- Poor fit for streams/tunnels.

### Long-running session

Pros:

- Supports streams.
- Supports connection reuse.
- Supports tunnels/watchers.
- Lower latency for repeated operations.

Cons:

- Lifecycle complexity.
- More failure modes.
- Resource leak risk.

### Recommendation

Do not replace on-demand execution. Add long-running sessions only for plugins/capabilities that need them.

---

## 7.4 Local-first vs Cloud-first

### Local-first

Pros:

- Trustworthy for developers.
- Easier open-source adoption.
- Lower infrastructure cost.
- No account required.

Cons:

- Harder team collaboration.
- Harder remote production access.
- No always-on jobs.

### Cloud-first

Pros:

- Collaboration.
- Scheduled jobs.
- Managed remote access.
- Monetization.

Cons:

- Much higher security and compliance burden.
- Slower to build.
- Could alienate open-source users.

### Recommendation

Stay local-first. Design runtime interfaces so remote/cloud can be added without rewriting core abstractions.

---

## 8. Existing and Future Problems

## 8.1 Existing problems likely present today

### Problem: Plugin protocol may be too narrow

Current plugin commands are database-oriented. This can slow expansion into infrastructure and event systems.

Mitigation:

- Introduce capability model.
- Keep old commands via compatibility adapter.

---

### Problem: JSON stdin/stdout may become limiting

JSON stdin/stdout is simple and good for one-shot commands, but it can become painful for:

- streaming
- large result sets
- cancellation
- multiplexing
- binary payloads
- long-running sessions

Mitigation:

- Keep JSON mode for simple plugins.
- Add gRPC or framed protobuf transport for session/stream mode.
- Add output size limits and pagination.

---

### Problem: Large query results can hurt memory/UI performance

If plugins return large results at once, the host and frontend may become slow or crash.

Mitigation:

- Introduce result pagination.
- Add row limits by default.
- Stream large results.
- Support lazy table rendering.
- Add export jobs for large data.

---

### Problem: Credential fallback storage may create security concerns

Fallback from OS keyring to SQLite/memory is useful, but users should know when credentials are not stored in the OS keyring.

Mitigation:

- Show credential storage status in settings.
- Warn when using SQLite fallback.
- Allow disabling persistent fallback.
- Add master password option only if necessary.

---

### Problem: Plugin trust is unclear

Users may not know whether a plugin is first-party, local, modified, or third-party.

Mitigation:

- Plugin manifest with publisher field.
- Show plugin source path.
- Show permission list.
- Add signature/checksum support later.

---

## 8.2 Problems likely to appear as QueryBox grows

### Problem: Protocol versioning

As plugin contracts evolve, older plugins may break.

Mitigation:

```text
plugin_api_version: v1
minimum_host_version: 0.4.0
supported_transports: [json-stdio, grpc-session]
```

Runtime should reject incompatible plugins with clear errors.

---

### Problem: Plugin marketplace supply chain risk

If users install third-party plugins, malicious plugins become a major risk.

Mitigation:

- Permission declarations.
- Plugin signing.
- Verified publisher labels.
- Sandboxed WASM runtime.
- User warnings for native plugins.
- Marketplace review pipeline.

---

### Problem: Remote agent becomes a security-sensitive component

A remote agent may have access to production databases and infrastructure.

Mitigation:

- Mutual TLS or strong token auth.
- Short-lived tokens.
- Least privilege plugin permissions.
- Audit logs.
- Explicit allowlist of plugins.
- No arbitrary shell execution by default.
- Separate read-only and write-capable modes.

---

### Problem: Remote/cloud runtime creates distributed failure modes

Possible failures:

- agent unreachable
- version mismatch
- stream interrupted
- partial result delivered
- operation timed out remotely but UI still waiting
- cloud broker connected but agent disconnected

Mitigation:

- Operation IDs.
- Heartbeats.
- Retry semantics.
- Explicit operation states.
- Idempotency keys for write operations.
- Clear UI error states.

---

### Problem: Generic UI can become confusing

A universal resource graph can make everything look the same, losing system-specific usability.

Mitigation:

- Generic core model.
- Plugin-provided UI hints.
- Resource-specific actions.
- Custom renderers for selected resource kinds.
- Avoid fully generic lowest-common-denominator UI.

---

### Problem: Too many integrations can dilute product quality

Supporting many systems can make QueryBox broad but shallow.

Mitigation:

- Choose a small number of high-quality first-party plugins.
- Define plugin quality checklist.
- Let community build long-tail integrations.
- Keep core runtime stable.

---

## 9. Recommended Roadmap

Current baseline already in repo:

- Product positioning and local-first runtime direction are locked by `ADR-001`.
- `PluginRegistry`, `RuntimeManager`, manifest-first discovery, and `resource.graph` are already in place for bundled plugins.
- Active roadmap should therefore start from cleanup and product validation work, not from redoing Phase 0 or Phase 1.

---

## Phase 1.5: Foundation Cleanup

Goal:

- Reconcile the capability model with the code/docs that already shipped in Phase 1.

Deliverables:

- `ADR-002: Plugin Capability Model`.
- Clear split between core runtime capabilities and feature-specific extension capabilities.
- Removal plan for stale capability and browse compatibility paths that no longer fit the `0.x` contract direction.

Success criteria:

- `pkg/plugin`, manifests, and docs describe the same taxonomy.
- Discovery is manifest-only for both runtime-sensitive and host-visible metadata.
- Active backlog no longer treats capability work as a greenfield design exercise.

---

## Phase 2: First Non-Database Workflow

Goal:

- Prove QueryBox can support one useful workflow outside the database-only path.

Recommended starting point:

1. Redis key inspection
2. S3/MinIO object browsing
3. Kafka read-only MVP after the first workflow is stable

Deliverables:

- One selected non-database plugin implemented against the current manifest/runtime/resource model.
- Resource-specific actions and result handlers for that workflow.
- Acceptance criteria for browse -> open -> inspect -> act.

Success criteria:

- A user can complete one realistic investigation flow that crosses the database mindset boundary.

---

## Phase 3: Streaming and Sessions

Goal:

- Add long-running runtime primitives only after the first non-database workflow proves where they are actually needed.

Deliverables:

- Stateful session lifecycle API.
- Stream protocol and host-side control model.
- UI validation path for at least one real stream workflow.

Recommended validation:

- Kafka consume stream
- Redis stream read
- Kubernetes or Docker log tailing

Success criteria:

- Long-running operations no longer need to be forced through one-shot subprocess semantics.

---

## Phase 4: Sandboxing and Plugin Trust

Goal:

- Turn declarative manifest metadata into visible and partially enforced trust boundaries.

Deliverables:

- Permission declaration UI.
- Coarse runtime enforcement for timeout, environment allowlist, workdir, and output limits.
- Sandbox level model and optional WASM evaluation.

Success criteria:

- Users can see what a plugin is asking for, and the host can enforce basic runtime controls before a marketplace grows.

---

## Phase 5: Workspace Layer

Goal:

- Make the local-first operational workspace visible in product behavior, not only in positioning.

Deliverables:

- Saved workspace model.
- Reopenable tabs/resources/notes state.
- Safe restore behavior when plugins or targets are unavailable.

Success criteria:

- A user can suspend and resume an investigation context locally without rebuilding the workspace from scratch.

---

## Phase 6: Remote Agent

Goal:

- Extend the runtime to private infrastructure only after the local-first workspace and plugin model are stable.

Deliverables:

- `querybox-agent` boundary prototype.
- Remote runtime protocol.
- Self-hosted agent MVP with basic auth/TLS and stream forwarding.

Success criteria:

- Desktop can browse or query a resource reachable only through the agent without changing the core UI model.

---

## Phase 7: Cloud Runtime / Collaboration

Goal:

- Keep cloud and collaboration as a future backlog, not an active implementation commitment.

Deliverables:

- `RuntimeTarget` abstraction that leaves room for cloud later.
- Explicit backlog for shared workspaces, audit logs, scheduled jobs, and policy management.

Success criteria:

- Core local-first runtime does not take a dependency on cloud features before remote-agent demand is proven.

---

## 10. Design Principles

### 10.1 Keep simple things simple

Do not force every plugin to implement sessions, streams, or complex permissions.

A simple plugin should still be easy to write:

```text
read stdin -> do work -> write stdout -> exit
```

### 10.2 Make advanced things possible

The architecture should allow:

- streaming
- remote runtime
- sandboxing
- long-running sessions
- workspace state

without requiring them for every plugin.

### 10.3 Prefer local-first by default

QueryBox should remain useful without an account, server, or cloud dependency.

### 10.4 Treat credentials as toxic data

Credentials should be passed only when needed, never logged, and never implicitly exposed to plugins.

### 10.5 Avoid becoming a generic admin console too early

Focus on backend engineer workflows, not every possible administrative feature.

### 10.6 Stable protocol over fast feature growth

The plugin protocol is the foundation. Changing it carelessly will create ecosystem pain.

---

## 11. Suggested Technical ADRs

Already accepted and assumed by the active roadmap:

1. `ADR-001: QueryBox as Operational Runtime`
2. `ADR-003: Universal Resource Graph`
3. `ADR-006: Runtime Manager Abstraction`
4. `ADR-007: Plugin Manifest and Permissions`

Still pending and worth writing as the roadmap advances:

5. `ADR-002: Plugin Capability Model`
6. `ADR-004: Stateless Operations and Stateful Sessions`
7. `ADR-005: Streaming Protocol`
8. `ADR-008: Native vs WASM Plugin Runtime`
9. `ADR-009: Remote Agent Runtime`
10. `ADR-010: Local-first Workspace Model`

---

## 12. Example Target Protocol Sketch

### 12.1 Plugin info

```json
{
  "plugin_id": "querybox.plugin.redis",
  "name": "Redis",
  "version": "0.1.0",
  "api_version": "v1",
  "runtime": {
    "type": "native-process",
    "supports_sessions": true,
    "supports_streams": true
  },
  "capabilities": [
    "connection.test",
    "resource.graph",
    "query.execute",
    "stream.read"
  ],
  "permissions": [
    "credential.read",
    "network.outbound"
  ]
}
```

### 12.2 Resource graph request

```json
{
  "operation": "resource.graph",
  "connection_id": "conn_redis_prod",
  "resource_id": null,
  "depth": 2
}
```

### 12.3 Resource graph response

```json
{
  "nodes": [
    {
      "id": "redis://prod/db/0",
      "name": "DB 0",
      "kind": "redis.database",
      "actions": ["open", "scan"]
    },
    {
      "id": "redis://prod/db/0/key/session:123",
      "name": "session:123",
      "kind": "redis.key.string",
      "actions": ["view", "edit", "delete"]
    }
  ]
}
```

### 12.4 Stream request

```json
{
  "operation": "stream.start",
  "connection_id": "conn_kafka_prod",
  "resource_id": "kafka://prod/topics/orders.created",
  "options": {
    "from": "latest",
    "limit": 1000
  }
}
```

### 12.5 Stream event

```json
{
  "stream_id": "stream_abc123",
  "sequence": 42,
  "timestamp": "2026-05-08T10:00:00Z",
  "event_type": "message",
  "payload": {
    "key": "order_123",
    "value": "{...}",
    "partition": 3,
    "offset": 991827
  }
}
```

---

## 13. MVP Recommendation

The best next technical milestone is not cloud execution yet.

The recommended MVP is:

> Capability model + universal resource graph + one non-database plugin.

Suggested MVP scope:

1. Add plugin manifest v1.
2. Add capability declaration.
3. Add `resource.graph` API.
4. Create compatibility adapter for existing database tree.
5. Implement Redis or S3 plugin using the new model.
6. Add resource-specific actions in UI.

Why this MVP:

- Proves QueryBox can move beyond SQL databases.
- Does not require remote runtime yet.
- Does not require full sandboxing yet.
- Keeps scope manageable.
- Creates architectural foundation for everything else.

---

## 14. Final Recommendation

QueryBox should evolve gradually from a plugin-based database app into a general operational runtime.

The strongest technical direction is:

```text
Capability-based plugin system
+ Universal resource graph
+ Optional stateful sessions
+ Streaming-first protocol
+ Runtime manager abstraction
+ Progressive sandboxing
+ Future remote/cloud runtime
```

Avoid trying to build everything at once. The priority should be to make the plugin contract future-proof enough for non-database systems while preserving the current simplicity that makes QueryBox approachable.

The most important near-term decision is to stop modeling everything as a database connection and start modeling everything as:

```text
Connection/Profile -> Resource Graph -> Actions -> Results/Streams
```

That single conceptual shift will make Redis, Kafka, S3, Kubernetes, logs, metrics, and remote runtime much easier to support later.

---

## Appendix A: Source Notes

This proposal was written based on public QueryBox repository and documentation available as of 2026-05-08, including:

- QueryBox repository README and file structure.
- QueryBox development setup documentation.
- QueryBox connection management documentation.
- QueryBox architecture overview documentation.

Key observed facts:

- QueryBox is described as a plugin-based database app for focused data work.
- The stack includes Go, Wails v3, Vue 3, Naive UI, Tailwind CSS, Vite, SQLite, OS keyring, and protobuf contracts.
- Plugins are standalone executables discovered from bundled and user plugin directories.
- Current plugin commands include `info`, `exec`, `authforms`, `connection-tree`, and `test-connection`.
- Current plugin execution is on-demand subprocess execution using JSON stdin/stdout.
- Credentials are stored with a three-tier fallback: OS keyring, SQLite fallback, and memory fallback.
