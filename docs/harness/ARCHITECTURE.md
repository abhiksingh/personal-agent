# Architecture

Top-level architecture map for the daemon-first MVP.

## Domains

1. `ui`: `Personal Agent` SwiftUI menu bar app for chat, approvals, settings, traces.
2. `daemon`: `Personal Agent Daemon` (Go) for task planning, policy, scheduling, orchestration, and plugin worker supervision.
3. `cli`: `Personal Agent CLI` (Go) thin client for daemon interaction and capability testing without UI.
4. `channels`: pluggable channel adapters executed as daemon-managed user-space worker processes.
5. `connectors`: pluggable connector adapters (Mail, Calendar, Browser, Finder in MVP) executed as daemon-managed user-space worker processes.
6. `persistence`: SQLite storage with serialized single-writer boundary (queue or equivalent gate), retention jobs.
7. `memory`: context retrieval, compaction, budgeted prompt assembly.
8. `transport`: platform-agnostic HTTP/JSON control APIs + WebSocket realtime streams, localhost TCP default, optional Unix socket/Windows named-pipe bindings.
9. `secrets`: write-only secret-value ingestion in client processes; daemon stores/resolves `SecretRef` metadata only and never exposes secret values in APIs.

## Running Processes

```mermaid
flowchart LR
  subgraph CLI["Process: personal-agent (CLI)"]
    CLI_PARSE["Command parsing + request shaping"]
    CLI_RENDER["JSON/stream rendering"]
    CLI_SECRET["Local secure-store writes (secret set/delete only)"]
  end

  subgraph APP["Process: Personal Agent (Swift app)"]
    APP_UI["Chat + approvals + settings UI"]
    APP_SECRET["Local secure-store writes from UI onboarding"]
  end

  subgraph DAEMON["Process: personal-agent-daemon (Go)"]
    API["Transport API (HTTP/JSON + WebSocket)"]
    CORE["Core services: provider/model/chat, agent/delegation, comm, automation, inspect, retention, context"]
    SECRETREF["SecretRef registry + resolver"]
    SUP["Plugin supervisor + worker dispatch"]
    DB["SQLite + single-writer gate"]
  end

  subgraph CHANNEL_WORKERS["Processes: channel workers"]
    TWILIO_WORKER["Twilio worker"]
  end

  subgraph CONNECTOR_WORKERS["Processes: connector workers"]
    MAIL_WORKER["Mail worker"]
    CAL_WORKER["Calendar worker"]
    BROWSER_WORKER["Browser worker"]
    FINDER_WORKER["Finder worker"]
  end

  CLI_PARSE --> API
  CLI_RENDER <-- API
  APP_UI --> API
  APP_UI <-- API
  API --> CORE
  CORE --> SECRETREF
  CORE --> DB
  CORE --> SUP
  SUP --> TWILIO_WORKER
  SUP --> MAIL_WORKER
  SUP --> CAL_WORKER
  SUP --> BROWSER_WORKER
  SUP --> FINDER_WORKER
```

## Data Flow

```mermaid
sequenceDiagram
  participant User
  participant Client as CLI/UI Client
  participant Store as Local Secure Store
  participant Daemon as Personal Agent Daemon
  participant Worker as Channel/Connector Worker
  participant DB as SQLite

  User->>Client: Set secret value
  Client->>Store: Write secret material
  Client->>Daemon: Register SecretRef metadata
  Daemon->>DB: Persist SecretRef + config

  User->>Client: Run chat/agent/channel command
  Client->>Daemon: Authenticated API request
  Daemon->>DB: Load policy, model, state
  Daemon->>Store: Resolve SecretRef when needed
  Daemon->>Worker: Dispatch channel/connector operation
  Worker-->>Daemon: Structured result/events
  Daemon->>DB: Persist transcript/audit/attempts
  Daemon-->>Client: Response + optional WS stream
```

## Layering Rule

Within each domain, depend only in this direction:

`types -> config -> contract -> repository -> service -> runtime -> interface`

Cross-domain dependencies must go through explicit interfaces/contracts.

## Lint Guardrails

- `tools/scripts/check_architecture_security_lint.sh` enforces package-layer import boundaries and fails when hotspot modules exceed configured line limits.
- `tools/scripts/check_architecture_security_lint.sh` also guards `source/services/daemon-go/internal/transport/types_payload_helpers.go` against direct `int(...)` coercion in risky `readAnyIntPointer` branches (`uint32`, `uint64`, `float32`, `float64`, `json.Number`), requiring checked conversion helpers instead.
- Current hotspot guards cover transport server/core backend paths, transport daemon-ops route modules, daemonruntime channel/connector worker-dispatch modules, daemonruntime comm inbox query modules, daemonruntime UI-status/identity modules, unified-turn orchestration modules, provider-model-chat modules, Twilio webhook runtime modules, Twilio voice persistence adapter modules, agent intent parsing modules, high-traffic OpenAPI adapter modules, queued-task/control-backend runtime modules, plugin supervisor modules, CLI main entrypoint, CLI quickstart/doctor modules, split CLI root-registry domain files, and split CLI command-discovery modules.

## Extension Rule

- Channels and connectors are selected through a registry keyed by declared capabilities.
- New adapters must implement shared contracts and register without planner/policy internal edits.
- Daemon-supervised plugin workers must register capabilities via startup handshake before routing.
- Adapter failures must return structured errors preserving retry/idempotency semantics.

## Invariant Focus

- Boundary inputs are validated.
- Writes are idempotent where retries are possible.
- Audit and trace events are append-only.
- Policy checks happen before side effects.
- Daemon startup/shutdown executes in deterministic phases with rollback on startup failure and ordered runtime drain during shutdown.

## Canonical Companion Docs

- `docs/spec/spec.md`
  - Canonical runtime behavior and product policy.
- `docs/spec/data-model.md`
  - Canonical schema and persistence invariants.
