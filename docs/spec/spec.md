# PersonalAgent Unified Spec (Canonical)

This is the single source of truth for the product specification.
The data model is maintained separately for independent iteration.

## 1) Product Definition

Build a macOS-first assistant OS that behaves like a human assistant operating on-device, communicates via app chat, iMessage/SMS, and voice, executes parallel tasks, and asks for approval only when risk requires it.

## 2) MVP Scope (Locked)

- Platform: macOS-first.
- Core runtime in MVP: `Personal Agent Daemon` implemented in Go for cross-platform portability, while product UX remains macOS-first.
- CLI in MVP: `Personal Agent CLI` implemented in Go for parity with daemon-supported platforms.
- User model in MVP: multi-principal workspace from day one.
- Channels in MVP are logical communication methods: `app`, `message`, and `voice`.
- Message-channel connector mappings in MVP: `imessage` (iMessage) and `twilio` (SMS capability).
- Voice-channel connector mappings in MVP: `twilio` (voice capability).
- App-channel connector mapping in MVP: `builtin.app` (internal app chat transport).
- Email in MVP is connector-only (Mail connector), not a standalone channel service.
- Connectors in MVP (hard requirement): `builtin.app`, `imessage`, `twilio`, Mail, Calendar, Browser, Finder.
- For current macOS-first connector/channel execution, prefer native macOS automation surfaces (Apple Events scripting dictionaries, EventKit, Safari extension messaging, and local user-space data watchers where necessary) over direct third-party provider APIs.
- Current macOS distribution posture for app/daemon artifacts is developer-style local/internal install without Apple Developer ID signing or notarization; first launch may require explicit Gatekeeper override actions.
- Proactive triggers in MVP: `SCHEDULE` and `ON_COMM_EVENT`.
- Storage in MVP: SQLite for operational data + Keychain for secrets.

## 3) Core Principles

- Product/API naming is Task/Step first.
- Multi-principal identity is explicit and first-class.
- Connectors-first execution, UI automation fallback.
- Approval required only for destructive/non-reversible actions.
- Auditability and step trace are mandatory.
- Memory is principal-scoped, retention-configurable, and compacted regularly.
- Secret material is write-only through product surfaces (CLI/app/daemon APIs never return raw secret values).
- Daemon owns lifecycle supervision of channel and connector plugin worker processes.

## 4) High-Level Architecture

1. Personal Agent (SwiftUI menu bar app): chat UI, approvals, settings, trace viewer.
2. Personal Agent Daemon (Go): planner, scheduler, policy engine, model router, worker pool, and plugin process supervisor.
3. Personal Agent CLI (Go): thin command-line client for daemon interaction and capability testing without the Swift app.
4. Channel Runtime: logical channel orchestration (routing/policy) over stable channel IDs (`app`, `message`, `voice`), with no provider-specific channel IDs.
5. Connector Runtime: pluggable connector adapters running as daemon-managed user-space plugin worker processes where applicable (`imessage`, `twilio`, Mail, Calendar, Browser, Finder in MVP), plus internal built-in connector surfaces (`builtin.app`) where process boundaries are not required.
6. Storage: SQLite + Keychain.
7. Client/daemon transport: platform-agnostic transport abstraction between app/CLI and daemon.
8. Daemon/plugin transport: internal daemon-to-plugin control/execution transport for channel and connector worker processes.
9. MVP control protocol contract: HTTP/JSON APIs with versioned OpenAPI definitions.
10. MVP realtime protocol contract: WebSocket event streams for token streaming, progress updates, interrupts/cancel, and approval/status push events.
11. Optional server-stream fallback: SSE endpoints using the same event envelope schema when bidirectional WebSocket control is unavailable.
12. Queued execution lifecycle transitions must emit deterministic realtime `task_run_lifecycle` events with correlation IDs; queue/control transitions (`queued`, `cancelled`, retry/requeue queued transitions) are required, while executor-driven terminal progression (`running -> awaiting_approval|completed|failed`) is configuration-dependent.
13. MVP default local binding: authenticated localhost TCP for cross-platform support (macOS, Windows, Linux including Raspberry Pi).
14. Production runtime profile requires explicit transport security material (`--tls-cert-file` + `--tls-key-file`, plus mTLS inputs when enabled by policy); auth token material alone is not sufficient for startup.
15. Optional platform bindings (for the same protocol contract): Unix domain sockets on Unix-like hosts and named pipes on Windows.
16. Apple bridge note: XPC may be used inside Apple-specific bridge components when required by platform APIs, but XPC is not the core runtime IPC contract.

### 4.1 Extensible Integration Contracts (Locked)

- Channel and connector implementations must conform to explicit adapter interfaces and register through a runtime registry.
- Channel IDs are stable logical identifiers (`app`, `message`, `voice`) and must not encode provider-specific transport identity.
- Connector IDs are stable machine identifiers (for example `builtin.app`, `imessage`, `twilio`) and must remain immutable once released.
- Adapter contracts must include stable identity, declared capabilities, health checks, execution entrypoints, and structured result/error envelopes.
- Adding a new channel/connector implementation must not require changing planner/policy core logic when contract requirements are met.
- MVP ships built-in adapters for required channels/connectors, but architecture must support extension loading and capability discovery.
- Daemon must own plugin process lifecycle management (spawn, health-check, restart, stop) for channel and connector workers.
- Daemon plugin-worker bootstrap must be manifest/config-driven (embedded default manifest plus optional override path); adding a worker must not require editing daemon main bootstrap code.
- Plugin workers must complete a structured handshake with declared metadata/capabilities before being marked runnable by daemon supervision.
- Daemon must emit append-only audit events for plugin lifecycle transitions (start, handshake, health timeout, restart, stop/exit).
- Channel/connector plugin processes run as normal user-space processes under the invoking user context; no privileged escalation is required for baseline operation.

### 4.2 Daemon API Group Baseline (Migration Contract)

Daemon transport surfaces are grouped under versioned `/v1` paths.
During daemon-first migration, CLI/app may adopt breaking command-level changes as long as docs/tests are updated in the same slice.

Core groups:

1. `/v1/tasks`, `/v1/approvals`, `/v1/capabilities`, `/v1/realtime`
2. `/v1/daemon` (lifecycle status/control surfaces for client UX integration)
3. `/v1/secrets`
4. `/v1/providers`
5. `/v1/models`
6. `/v1/chat`
7. `/v1/agent`
8. `/v1/delegation`
9. `/v1/comm`
10. `/v1/automation`
11. `/v1/inspect`
12. `/v1/retention`
13. `/v1/context`
14. `/v1/channels` (status/config summaries and channel-specific operations)
15. `/v1/connectors` (status/config summaries for connector cards)

Retention purge consistency contract:
- `POST /v1/retention/purge` uses explicit `partial_success` consistency semantics under statement-level failures.
- Response payload always includes typed result status (`completed|partial_failure`) and deterministic failure metadata (`stage`, `code`) when a mid-sequence purge statement fails after earlier deletions committed.

### 4.3 macOS Native Automation Strategy (Locked for Current Connector/Channel Path)

This strategy defines the default production path for macOS-native integrations in the current workstream.

1. Mail connector:
   - Execute outbound actions (draft/send/reply) and inbox unread-summary queries through Mail connector surfaces.
   - Ingest inbound "new mail" events through Mail rule script handoff into daemon ingress APIs.
   - Unread-summary semantics are deterministic: inbound mail events with no newer assistant-emitted outbound mail event in the same thread.
2. Calendar connector:
   - Execute create/update/cancel/read operations through EventKit-backed automation worker logic.
   - `calendar_create` must emit a stable `event_id` in evidence/output.
   - `calendar_update` and `calendar_cancel` must require explicit `event_id` targeting; title-derived fallback targeting is not allowed.
   - Calendar event automation payloads may include `title` and `notes`; update must carry at least one mutable field (`title` or `notes`) in addition to `event_id`.
   - Ingest calendar changes through EventKit store-change notifications with deterministic cursoring.
3. Messages connector (`imessage`):
   - Treat iMessage as a first-class message-channel connector path (not fallback-only).
   - Execute outbound sends through Messages automation surfaces.
   - Ingest inbound iMessage events through user-space local-source watcher ingestion with explicit idempotency/trust records.
4. Safari browser connector:
   - Execute open/extract/close through Safari-native automation.
   - `browser_extract` returns structured extraction payloads (`title`, `url`, `content_text`, `content_preview`) and may include a query-grounded `query_answer` when a query is provided.
   - Ingest browser-side updates/events through Safari extension/app messaging channels.
5. Finder connector:
   - Execute canonical finder actions `finder_find`, `finder_list`, `finder_preview`, and `finder_delete`.
   - Finder list/preview/delete actions accept either an absolute `path` or a semantic `query` (+ optional `root_path`) resolved by deterministic ranking.
   - `finder_find` must return deterministic `selected_path` and bounded candidate match metadata.
   - `finder_delete` with query-based resolution must enforce destructive guardrails: ambiguous multi-match queries are denied until the target is uniquely specified.
6. Trust boundary and permissions:
   - All macOS automation workers run as daemon-supervised user-space processes under the invoking user context.
   - TCC/automation permission requirements must be explicit and surfaced in status/config APIs.
7. Fallback behavior:
   - If native automation permission or transport is unavailable, operations fail with structured, user-actionable remediation (no silent synthetic success paths).

### 4.4 Canonical Channel/Connector IDs and Mapping (Locked)

- Logical channel IDs are fixed as `app`, `message`, and `voice`.
- Connector IDs are fixed machine identifiers; display names are independent presentation metadata.
- Initial canonical connector IDs in MVP:
  - `builtin.app`
  - `imessage`
  - `twilio`
  - `mail`
  - `calendar`
  - `browser`
  - `finder`
- Channel-to-connector mappings are explicit workspace-scoped bindings with enable/disable state and priority ordering.
- Default MVP mapping:
  - `app -> builtin.app`
  - `message -> imessage, twilio`
  - `voice -> twilio`
- Message-channel threading must preserve connector isolation by default; cross-connector threads are not auto-merged.

### 4.5 macOS Distribution Trust Model (Current Local/Internal Path)

1. Current app/daemon packaging targets developer/operator local and internal use; it is not a Developer ID signed/notarized production-distribution channel.
2. Gatekeeper override is expected for first launch of downloaded/transferred artifacts in this mode (for example right-click `Open` or System Settings `Open Anyway`).
3. Documentation and UI setup flows must explicitly disclose this trust posture and provide deterministic remediation guidance when launch is blocked by host trust controls.
4. TCC attribution requirements remain unchanged in this mode:
   - Automation/permission prompts should be initiated by `Personal Agent Daemon` identity, not transient terminal/Codex parent process chains.
   - Daemon identity must remain stable (`com.personalagent.daemon`) across local/internal packaging iterations to minimize avoidable permission re-prompts.
5. Distribution-mode limitations must be explicit: absence of Developer ID signatures/notarization means host-level trust prompts and stricter user override steps are expected behavior, not runtime defects.

## 5) Canonical Runtime Contracts

### 5.1 Task

A user/system objective with priority, deadline, channel context, principal context, and risk classification.

### 5.2 TaskRun

A concrete execution attempt of a Task, including `acting_as` principal identity and execution state.

### 5.3 TaskStep

An atomic executable unit with capability requirements, timeout/retry policy, interaction level, recipients (when communication step), and evidence outputs.
Each step carries canonical typed `input` payload fields that adapters consume directly for execution; core execution fields must not be derived by parsing display-oriented step names.

### 5.4 PolicyDecision

`ALLOW | REQUIRE_CONFIRM | DENY`, with rationale and matched policy/rule references.

### 5.5 TraceEvent

Immutable execution/audit event containing timestamp, actor, acting-as identity, action, result, and correlation IDs.

### 5.6 TurnItem

Canonical chat-turn unit persisted and streamed by type:

- `user_message`
- `assistant_message`
- `tool_call`
- `tool_result`
- `approval_request`
- `approval_decision`

Each turn item carries deterministic `item_id`, type-specific payload, and optional status metadata.

### 5.7 ChatTurn

`/v1/chat/turn` is a unified orchestrated turn contract:

- Request includes canonical `items[]` and optional channel/context metadata.
- Response returns canonical generated `items[]` for the completed turn.
- Response must include explicit contract version fields (`contract_version`, `turn_item_schema_version`, `realtime_event_contract_version`) so clients can bind deterministic parser behavior.
- Legacy ask/act mode branching fields are not part of the canonical contract.
- Tool execution and approvals are represented as typed turn items, not mode-specific envelopes.
- Persisted turn-item replay is exposed through `/v1/chat/history` for diagnostics/UI continuity.
- Persona/style policy read+write APIs are exposed through `/v1/chat/persona/get` and `/v1/chat/persona/set`.
- Planner-visible tool registry is generated per turn from ready connector capability inventory (not a static hardcoded execution list).
- Capability key normalization (`.` and `_`) must resolve to one canonical matching key so tool exposure remains deterministic across connector metadata variants.
- Unified-turn orchestration supports iterative tool execution in one turn (`model -> tool* -> model`) with deterministic bounded tool-call limits.
- Iterative loops must stop deterministically on approval pause (`awaiting_approval`), planner no-action output, or max tool-call limit exhaustion.
- Planner structured-output handling must run bounded repair retries before model-only fallback; exhausted retries must emit deterministic remediation metadata (`planner_output_invalid`) instead of silently dropping tool intent.
- Chat tool execution origin is channel-canonical: `app` for app/message-channel turns and `voice` for voice-channel turns; legacy synthetic origins (for example `chat_unified_tool`) are not part of the canonical contract.
- Unified-turn tool execution must pass typed `native_action` payloads to agent runtime; synthetic request-text bridges are not part of the canonical execution path.
- For provider chat runtimes that support native tool-calling protocols (currently OpenAI and Ollama), planner turns must use provider-native tool payloads and convert tool-call responses back into canonical turn-item directives; unsupported providers must deterministically fall back to canonical non-native planner parsing.
- Model-route and tool/planner failure paths must emit actionable remediation metadata so app/UI layers can render deterministic recovery actions without parsing raw provider/runtime error strings.
- Turn explainability is exposed through `/v1/chat/turn/explain` and must include selected model route context, turn-context tool catalog schema, and per-tool policy decisions for the provided actor/channel scope.
- No legacy compatibility shims are allowed for chat-action orchestration cutover contracts (ask/act fields, synthetic request-text bridges, or execution-origin aliases).

### 5.8 Chat Realtime Lifecycle

Realtime turn streams include typed lifecycle events for turn items and tools:

- `turn_item_started`
- `turn_item_delta`
- `turn_item_completed`
- `tool_call_started`
- `tool_call_output`
- `tool_call_completed`
- Chat lifecycle events must include explicit realtime contract metadata (`contract_version`, `lifecycle_schema_version`) for deterministic turn/tool event parsing.

## 6) Identity and Access Model (Locked)

Use the workspace/actor/principal model as canonical.

- `Workspace`: tenant boundary.
- `User`: login/auth identity.
- `WorkspaceMember`: user membership and role.
- `Actor`: human/service/assistant identity in a workspace.
- `WorkspacePrincipal`: actors the assistant may act as.
- `ActorHandle`: channel address identity (email/phone/etc.).

Required identity semantics:

- `requested_by`: who asked for work.
- `subject_principal`: whose context the work concerns.
- `acting_as`: which principal identity is used to execute.

Delegation rule (MVP):

- Cross-principal `acting_as` requires explicit per-principal delegation.
- Without delegation, a principal can only execute as themselves.

### 6.1 Active Context Ownership and Sync Semantics (Locked)

- Daemon is the source of truth for active workspace/principal context across clients.
- Context mutation is accepted only through explicit selection APIs (for example `identity/context/select-workspace` and bootstrap flows that call select), not through passive read/list calls.
- Deterministic conflict rule: last accepted explicit selection write wins; daemon assigns a monotonic `selection_version` to each accepted mutation.
- Active-context query payloads include `mutation_source` (`app|cli|daemon`), `mutation_reason` (`explicit_select_workspace|request_override|derived_resolution|default_resolution`), and `selection_version` so clients can surface why context changed and ignore stale updates.
- `identity/context` with explicit `workspace_id` is a request-scoped read override (`mutation_reason=request_override`) and must not mutate daemon-selected context.
- `identity/context` and passive identity directory reads without explicit workspace override must resolve workspace deterministically in this order: selected workspace (if still resolvable) -> canonical default `ws1` (when present) -> first non-reserved workspace fallback -> unresolved canonical default response. These read-time fallback results must not mutate selected context state or increment `selection_version`.

## 7) Communication Model

- `CommThread`: logical channel thread with explicit connector attribution.
- `CommEvent`: message/call/system event.
- `CommEventAddress`: source of truth for sender/recipients (`FROM/TO/CC/BCC/REPLY_TO` as applicable).
- `CommAttachment`: optional for MVP if attachment processing is deferred.

Rules:

- Event-level addresses are authoritative for reply-all/group correctness.
- Group membership indexes are derived and non-authoritative.
- Communication records must persist both logical channel identity and connector identity; clients must not infer connector from provider-specific `external_ref` parsing.
- Message channel histories are connector-isolated by default (for example iMessage thread history is separate from Twilio SMS thread history even for the same user/contact).

### 7.2 iMessage Connector Ingestion Semantics (Locked for macOS Path)

1. Inbound iMessage events must carry deterministic source identity (`source`, `source_scope`, `source_event_id`) before side effects are applied.
2. Inbound iMessage ingestion must persist trust + replay receipts before creating `CommEvent`, trigger fires, or delivery side effects.
3. Outbound iMessage attempts must persist delivery attempt evidence and route/fallback decisions the same way as other connector-attributed paths.

### 7.1 Twilio Connector Semantics (Locked for Twilio Path)

- Twilio SMS inbound webhooks map to `CommEvent` with `event_type=MESSAGE`, `direction=INBOUND`, and Twilio provider message SID metadata.
- Twilio SMS outbound messages record provider receipt/SID for delivery evidence and idempotent replay behavior.
- Twilio SMS webhook runtime supports optional conversational assistant-reply mode that generates model-backed replies and delivers outbound SMS responses on accepted, non-replayed inbound events.
- Direct control-plane `connector twilio sms-chat` turns must execute the same inbound-ingest + thread-context assistant-reply orchestration as webhook conversational mode, with deterministic replay keying derived from `operation_id` and response metadata fields (`assistant_reply`, `assistant_operation_id`, `assistant_error`).
- Twilio voice webhooks map call lifecycle to call/session events and transcript events in `CommEvent`.
- Outbound voice calls initiated by agent must create a call session record before provider dispatch and update status via provider callbacks.
- Twilio voice webhook runtime supports optional TwiML conversational loop responses (`Gather` + `Say`) for real-time call-turn interaction while preserving call-session/transcript persistence.
- Twilio configuration is connector-scoped and shared for both SMS and voice capabilities (set once, reused by both channels that map to Twilio capabilities).
- Default webhook callback paths are versioned under `/<project-name>/v1/connector/twilio/{sms|voice}` (overridable by explicit path arguments). The daemon resolves `<project-name>` from its process/binary name and supports `PA_PROJECT_NAME` override when an explicit project-name mapping is required.
- Daemon webhook serve may auto-bootstrap a cloudflared quick tunnel (`cloudflared-mode=auto`) and return public callback URLs; when cloudflared is unavailable it must gracefully fall back to local-only URLs with explicit warning metadata.

## 8) Proactive and Automation Model (Locked)

- Use `Directive` for durable intent.
- Use `AutomationTrigger` for execution triggers.
- MVP trigger types required: `SCHEDULE` and `ON_COMM_EVENT`.
- Recurrence in v1 is not restricted to only daily/weekly.

`ON_COMM_EVENT` trigger contract (MVP):

- Required default filter behavior:
1. `event_type=MESSAGE`
2. `direction=INBOUND`
3. source event must not be assistant-emitted
- Configurable filters:
1. channel include list
2. principal scope
3. sender actor/handle allowlist
4. thread include list
5. keyword match rules
- Keyword match mode in MVP:
1. case-insensitive `contains` and exact-phrase matching
2. optional any/all term grouping
3. regex matching deferred until post-MVP
- Rate limiting is configurable per trigger; default is no cooldown.
- Trigger idempotency key is `(workspace_id, trigger_id, source_event_id)`.

## 9) Model Runtime and Routing

Providers: OpenAI, Anthropic, Google, Ollama.

Routing inputs:

- task type
- risk
- complexity/context size
- privacy mode
- tool/connector need
- historical success/corrections/latency/cost

Privacy modes:

- `local_only`
- `hybrid`
- `cloud_ok`

Routing outputs:

- selected model
- tool strategy
- fallback chain

## 10) Safety and Approval Policy (Locked)

- Reversible/non-destructive actions: auto-execute unless denied by explicit user policy.
- Destructive/non-reversible actions: mandatory approval gate.
- Action risk classification is model-assisted:
1. model proposes reversible vs destructive classification with rationale and confidence
2. policy engine enforces deterministic guardrails and capability constraints
3. low-confidence or ambiguous classification defaults to approval-required
- Approval UX for destructive actions is exact phrase `GO AHEAD`.
- For voice-originated destructive actions, approval requires in-app handoff and in-app confirmation before execution.
- Approvers for destructive actions: acting-as principal and explicitly delegated approvers.
- Approval is not required for every action.
- When uncertain, ask a clarifying question and offer dry-run or stop.

## 11) Execution, Concurrency, and Conflicts

Task state machine:

`queued -> planning -> awaiting_approval -> running -> blocked -> completed|failed|cancelled`

Runtime requirements:

- Parallel task execution via worker pool.
- Priority/deadline-aware scheduling and retry/timeout controls.
- Conflict detection across shared resources.
- Conflict preference can be persisted after first explicit user resolution.
- Channel/connector execution must route through pluggable adapter contracts with capability-based selection.
- Channel/connector execution dispatch is daemon-mediated and targets supervised out-of-process plugin workers; CLI/app clients do not execute adapter logic in-process.

## 12) Channel Delivery and Reliability (Locked)

Messaging delivery policy in v1:

- default message-channel path: `imessage` (iMessage) retry once on failure, then `twilio` SMS capability fallback
- fallback policies must be configurable so future channel/endpoint policies can be added without architecture changes
- Response routing default: send responses on the originating inbound connector within the originating logical channel.
- Channel change is allowed only when an explicit delivery policy fallback is triggered.
- Message-channel connector threads must remain separate by default; fallback between connectors does not imply automatic thread merge.

### 12.1 Twilio Webhook Trust and Idempotency (Locked for Twilio Path)

- Inbound Twilio webhooks require signature validation (`X-Twilio-Signature`) against workspace-scoped Twilio auth token, except explicit local-dev bypass mode.
- Invalid signatures must be rejected and logged as audit events; no task/comm side effects may be created.
- Webhook ingestion idempotency key must include provider identity and provider event/message/call SID so webhook retries do not duplicate `CommEvent` or trigger execution.
- Twilio webhook ingest persistence is daemon-owned; connector workers do not expose webhook-ingest write operations for this path.
- Twilio provider receipts/SIDs must be persisted for outbound SMS/voice attempts to support at-least-once delivery evidence and replay safety.

### 12.2 macOS Local Ingress Trust and Idempotency (Locked for Mail/Calendar/Messages/Safari Paths)

1. Local-source ingress (Mail rules, EventKit change feed, iMessage watcher, Safari extension feed) must persist deterministic receipt records keyed by `(workspace_id, source, source_event_id)` before side effects.
2. Replayed local-source events must be accepted as no-op replays (no duplicate `CommEvent`, `TriggerFire`, or task side effects).
3. Ingress trust state (`accepted`/`rejected`) must be recorded for every local-source event to preserve auditability and debugging.
4. Poll-based ingesters (for example local message-store watchers) must persist source cursor state so restarts do not lose events or duplicate historical events.

## 13) Voice Requirements

- Voice is included in MVP.
- Quick acknowledgment on inbound voice requests.
- Progress updates during long-running tasks without excessive chatter.
- Voice transcripts stored by default (retention policy below).
- Calling is behind provider abstraction (Continuity/Twilio implementations can sit behind same interface).
- Twilio connector voice capability must support:
1. inbound call to agent-owned number with conversation routed into agent runtime
2. outbound call initiated by agent to target number
3. call status updates (initiated/ringing/in_progress/completed/failed/no_answer/busy/canceled)
4. transcript event persistence for call turns suitable for `ON_COMM_EVENT` automation evaluation where applicable

## 14) Storage, Secrets, and Retention

- SQLite is the MVP operational store (tasks, traces, comm metadata, memory metadata).
- Keychain stores credentials/tokens/secrets.
- Retention default is 7 days for traces, transcripts, and memory; all are configurable.
- Runtime write path uses a single-writer queue boundary for SQLite in MVP (or documented equivalent serialized single-writer SQLite gate), with architecture prepared for a future multi-writer model.

### Secret Handling (Locked)

- Secret values may be set from clients (CLI/app) only as write operations into the secure store.
- Secret values are never returned by daemon/client APIs; product surfaces expose only metadata/masked status and references.
- Daemon receives/persists only secret references (`SecretRef`) and resolves values from secure storage at execution time.
- User-facing secret value inspection/management occurs in the underlying secure-store tooling (for example Keychain app), not through Personal Agent read APIs.

### Memory Retention and Compaction (Locked)

- Memory retention follows the same 7-day default unless configured otherwise.
- Memory remains principal-scoped with explicit status (`ACTIVE`, `DISABLED`, `DELETED`).
- Compaction is threshold-based to keep retrieval context bounded.
- Retrieval must enforce strict context budgets using model-context limits to minimize token use while preserving relevant context.

Compaction policy (MVP practical baseline):

1. Keep canonical structured memory facts/rules.
2. Summarize stale conversational memory into compact summaries.
3. Preserve source links for auditability.
4. Exclude disabled/deleted memory from retrieval.

### LLM Context Budget Defaults (MVP)

Use a dynamic token budget per request class, based on the selected model and provider limits.

1. Compute `W` as the selected model context window.
2. Reserve output tokens as `max(1024, 15% of W)`, capped by the model output limit.
3. Reserve system/tools budget as `max(1500, 10% of W)`.
4. Reserve safety headroom as `max(512, 5% of W)` to avoid hard-limit overruns.
5. The remaining budget is available for recent thread history + retrieved memory/context.
6. Retrieval target defaults to `min(24000, remaining budget)` and may expand for explicit deep-analysis tasks.
7. Always run provider token-count estimation before final prompt assembly when available.

### Context Budget Telemetry and Tuning (MVP)

1. Runtime records prompt/context usage telemetry per task class, including retrieval target usage and token spend.
2. Tuning profiles are maintained per task class with deterministic bounded multipliers for retrieval target sizing.
3. Tuning must never violate model context safety constraints; final retrieval target remains capped by remaining budget.

## 15) MVP Build Acceptance Criteria

1. Task/TaskRun/TaskStep interfaces frozen and implemented.
2. Multi-principal identity model operational (`requested_by`, `subject_principal`, `acting_as`).
3. Approval gate implemented for destructive actions using exact `GO AHEAD` phrase.
4. Non-destructive actions execute without mandatory approval prompts.
5. End-to-end logical channel support: `app`, `message`, and `voice`.
6. Hard MVP connectors pass at least one E2E flow each: Mail, Calendar, Browser, Finder.
7. Proactive automations work for `SCHEDULE` and `ON_COMM_EVENT` triggers.
8. Messaging failure path validates connector-priority routing in message channel (iMessage retry once, then Twilio SMS fallback by default).
9. Trace/transcript 7-day retention default works with configuration override.
10. Memory retention uses 7-day default with configuration override and threshold-based compaction.
11. Destructive actions initiated from voice require successful in-app approval handoff before execution.
12. Cross-principal execution fails without explicit delegation and succeeds with valid delegation.
13. Connector acceptance starts with happy-path E2E coverage for Mail, Calendar, Browser, and Finder.
14. `ON_COMM_EVENT` keyword filters pass with case-insensitive contains/exact phrase matching.
15. Outbound delivery achieves at-least-once semantics with idempotency safeguards to minimize duplicates.
16. Personal Agent Daemon can be exercised via Go-based Personal Agent CLI for task submission/status and capability smoke tests without the Swift app.
17. Channel and connector runtimes support pluggable adapter registration with capability discovery, enabling extension without core planner/policy rewrites.
18. Core app/daemon communication uses the hybrid platform-agnostic contract (HTTP/JSON control APIs + WebSocket realtime stream) with localhost TCP default binding in MVP; any XPC usage is confined to Apple-specific bridge components.
19. Realtime clients can receive streaming LLM output and step progress over WebSocket and can issue interactive control signals (for example cancel) on the same session.
20. Twilio connector SMS capability supports inbound/outbound chat workflows with validated webhooks, idempotent retry handling, and persisted provider receipts.
21. Twilio connector voice capability supports inbound and outbound call workflows with persisted call status transitions and transcript events.
22. Personal Agent CLI can exercise Twilio connector SMS/voice workflows without requiring the Swift app.
23. Twilio webhook runtime can run in conversational mode: SMS inbound turns can trigger assistant auto-replies and voice inbound turns can return TwiML loop responses for continued conversation.
24. Secret management is write-only for secret value material: CLI/app/daemon APIs never return plaintext secret values after set.
25. Daemon supervises channel/connector plugin worker lifecycles and routes channel/connector execution through those user-space processes, with handshake-based capability registration and audit events for lifecycle transitions.
26. CLI/app act as thin clients over daemon APIs for core workflows; breaking CLI surface changes are acceptable during daemon-first refactor if docs/tests are updated.
27. Mail, Calendar, Apple Messages, and Safari integrations support two-way operation on macOS: command execution plus inbound change/event ingestion into daemon automation/comm evaluation paths.
28. iMessage is operational as a first-class message-channel connector path (outbound + inbound ingest evidence), not only as a policy fallback hop.
29. Canonical product/docs contracts explicitly describe the current non-Apple-account macOS distribution trust model (unsigned/ad-hoc local/internal artifacts, Gatekeeper override requirements, and daemon-owned TCC attribution expectations).

## 16) Data Model

The canonical data model is in:

- [`docs/spec/data-model.md`](./data-model.md)

## 17) Status

This document is the canonical spec to build from.
