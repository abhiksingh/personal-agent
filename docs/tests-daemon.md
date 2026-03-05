# PersonalAgent Daemon Manual Test Guide

This guide validates daemon transport/control/realtime behavior independently from local-only CLI workflows.

Coverage in this guide:

1. Daemon startup and authenticated health checks
2. Production-mode auth-token hardening guardrail
3. Daemon lifecycle status/control APIs for UI integration (`start|stop|restart`)
4. Channel/connector status, config-mutation, test-operation, and diagnostics APIs for UI cards/actions (including Cloudflared connector control-plane routes)
5. Connector happy-path runs (Mail/Calendar/Browser/Finder)
6. Approval inbox query/list API with pending/final metadata
7. Voice destructive-action handoff gate enforcement
8. Inspect log query/stream APIs with LIFO semantics
9. Control API capability smoke + metadata discovery + task lifecycle checks
10. Task/run list query API with principal/timestamp/error metadata
11. Realtime stream stability checks (including chat lifecycle event metadata)
12. Twilio channel control-plane checks (`ingest`, `serve`, `replay`)
13. Messages/Mail/Calendar/Browser local-source ingest checks (Messages `chat.db` poll + Mail rule + Calendar change + Safari extension handoff, replay-safe cursor/receipt writes)
14. Daemon automation auto-evaluation checks (`SCHEDULE`, `ON_COMM_EVENT`)
15. Transport mode checks (`tcp`, `unix`, `named_pipe`)
16. Platform service install/start-on-boot script validation (`launchd`, `systemd --user`, Windows scheduled task)
17. Communications inbox and delivery-policy/attempt-history APIs for thread inventory, event timeline, voice call-session summaries, workspace-scoped iMessage visibility stability, fallback policy updates, and context-filtered retry/fallback attempt visibility
18. Context memory/retrieval query APIs for principal/source-scoped browsing and deterministic paging
19. Delegation capability-grant APIs and communications trust-receipt query APIs with audit-link metadata
20. Model-route simulation and explainability APIs with fallback-chain decision traces
21. Identity device/session inventory and session-revoke APIs with last-seen/session-health metadata and idempotent revoke behavior
22. Daemon auth-scope startup policy enforcement (`--auth-token-scopes`) for privileged-read deny behavior
23. Principal-aware control-plane rate limiting with typed deterministic `429` retry metadata
24. Realtime websocket session guardrails (read bounds, heartbeat liveness, and capacity caps)

## Quick Run (Automated)

Run the saved daemon manual-test runner from repo root:

```bash
./tools/scripts/run_tests_daemon.sh
```

To run daemon + CLI + UI runners together, use:

```bash
./tools/scripts/run_tests_all.sh
```

The runner exits with process code `0` only when all steps pass; any recorded step failure returns non-zero.

Useful options:

```bash
# Skip the optional regression section
./tools/scripts/run_tests_daemon.sh --skip-regression

# Write to a specific log path
./tools/scripts/run_tests_daemon.sh --log-file out/logs/manual-tests/tests-daemon-manual.log
```

Use this from repository root:

```bash
cd <repo-root>
```

## 1) Prerequisites

- Go toolchain installed.
- `curl` installed.
- `jq` installed.

## 2) Common Setup

```bash
export TEST_RUNTIME_ROOT="$PWD/out/test-state/daemon"
export WORKSPACE="${WORKSPACE:-test-ws1}"
export INSPECT_WORKSPACE="${INSPECT_WORKSPACE:-daemon}"
export DAEMON_AUTH_TOKEN="daemon-test-token"
export DAEMON_DB_PATH="${DAEMON_DB_PATH:-$TEST_RUNTIME_ROOT/manual-test-daemon.db}"
export DAEMON_TCP_ADDR="127.0.0.1:7071"
export DAEMON_UNIX_SOCKET="/tmp/personal-agent-daemon.sock"
export DAEMON_AUTH_TOKEN_FILE="${DAEMON_AUTH_TOKEN_FILE:-$TEST_RUNTIME_ROOT/manual-test-daemon.control.token}"
export PA_RUNTIME_PROFILE="${PA_RUNTIME_PROFILE:-test}"
export PA_RUNTIME_ROOT_DIR="${PA_RUNTIME_ROOT_DIR:-$TEST_RUNTIME_ROOT/runtime-root}"
export DAEMON_LIFECYCLE_HOST_OPS_MODE="${DAEMON_LIFECYCLE_HOST_OPS_MODE:-dry-run}" # use apply for real host install/uninstall/repair operations
export PA_MESSAGES_SEND_DRY_RUN="${PA_MESSAGES_SEND_DRY_RUN:-1}" # keeps iMessage send checks deterministic on non-macOS/TCC-restricted hosts
export PA_MAIL_AUTOMATION_DRY_RUN="1" # forced for mail-test safety (prevents real sends)
export PA_CALENDAR_AUTOMATION_DRY_RUN="${PA_CALENDAR_AUTOMATION_DRY_RUN:-1}" # keeps Calendar connector checks deterministic on non-macOS/TCC-restricted hosts
export PA_BROWSER_AUTOMATION_DRY_RUN="${PA_BROWSER_AUTOMATION_DRY_RUN:-1}" # keeps Safari connector checks deterministic on non-macOS/TCC-restricted hosts
export PA_CLOUDFLARED_DRY_RUN="${PA_CLOUDFLARED_DRY_RUN:-1}" # keeps cloudflared connector checks deterministic/offline-friendly
export PA_INBOUND_WATCHER_WORKSPACE_ID="$WORKSPACE" # daemon watcher default workspace
export PA_INBOUND_WATCHER_INBOX_DIR="${PA_INBOUND_WATCHER_INBOX_DIR:-$TEST_RUNTIME_ROOT/inbound-watcher-inbox}" # Mail/Calendar/Browser handoff inbox root
export PA_INBOUND_WATCHER_POLL_INTERVAL="${PA_INBOUND_WATCHER_POLL_INTERVAL:-150ms}" # shorter poll for manual validation
export PA_INBOUND_WATCHER_MESSAGES_SOURCE_SCOPE="${PA_INBOUND_WATCHER_MESSAGES_SOURCE_SCOPE:-daemon-fixture-scope}"
export PA_INBOUND_WATCHER_MESSAGES_SOURCE_DB_PATH="${PA_INBOUND_WATCHER_MESSAGES_SOURCE_DB_PATH:-$TEST_RUNTIME_ROOT/messages-ingest-fixture.db}"
export PA_INBOUND_WATCHER_MESSAGES_LIMIT="${PA_INBOUND_WATCHER_MESSAGES_LIMIT:-10}"
mkdir -p "$TEST_RUNTIME_ROOT" "$PA_RUNTIME_ROOT_DIR"
rm -f "$DAEMON_DB_PATH"

pa_tcp() {
  (cd source/clients/cli-go && go run ./cmd/personal-agent --mode tcp --address "$DAEMON_TCP_ADDR" --auth-token "$DAEMON_AUTH_TOKEN" "$@")
}

pa_unix() {
  (cd source/clients/cli-go && go run ./cmd/personal-agent --mode unix --address "$DAEMON_UNIX_SOCKET" --auth-token "$DAEMON_AUTH_TOKEN" "$@")
}
```

## 3) TCP Mode Validation

Run daemon (terminal A):

```bash
PA_RUNTIME_PROFILE="$PA_RUNTIME_PROFILE" PA_RUNTIME_ROOT_DIR="$PA_RUNTIME_ROOT_DIR" go -C source/services/daemon-go run ./cmd/personal-agent-daemon \
  --listen-mode tcp \
  --listen-address "$DAEMON_TCP_ADDR" \
  --auth-token "$DAEMON_AUTH_TOKEN" \
  --db "$DAEMON_DB_PATH" \
  --lifecycle-host-ops-mode "$DAEMON_LIFECYCLE_HOST_OPS_MODE"
```

Run checks (terminal B).

### 3.0 Auth token and local-bind guardrails

```bash
# Daemon guardrail: auth token is required
go -C source/services/daemon-go run ./cmd/personal-agent-daemon \
  --listen-mode tcp \
  --listen-address "$DAEMON_TCP_ADDR" \
  --runtime-profile prod \
  --auth-token ""

# CLI guardrail
go -C source/clients/cli-go run ./cmd/personal-agent \
  --mode tcp \
  --address "$DAEMON_TCP_ADDR" \
  --runtime-profile prod \
  smoke

# Local-bind default guardrail
go -C source/services/daemon-go run ./cmd/personal-agent-daemon \
  --listen-mode tcp \
  --listen-address "0.0.0.0:notaport" \
  --auth-token "$DAEMON_AUTH_TOKEN"

# Bootstrap + rotate production auth token file (no plaintext token output)
rm -f "$DAEMON_AUTH_TOKEN_FILE"
go -C source/clients/cli-go run ./cmd/personal-agent auth bootstrap --file "$DAEMON_AUTH_TOKEN_FILE"
go -C source/clients/cli-go run ./cmd/personal-agent auth rotate --file "$DAEMON_AUTH_TOKEN_FILE"

# Production guardrail: auth-token-file alone is insufficient without TLS+mTLS
go -C source/services/daemon-go run ./cmd/personal-agent-daemon \
  --listen-mode tcp \
  --listen-address "$DAEMON_TCP_ADDR" \
  --runtime-profile prod \
  --auth-token-file "$DAEMON_AUTH_TOKEN_FILE" \
  --db "$DAEMON_DB_PATH" \
  --lifecycle-host-ops-mode "$DAEMON_LIFECYCLE_HOST_OPS_MODE"

# Start daemon in local profile with auth-token-file
PA_RUNTIME_PROFILE="$PA_RUNTIME_PROFILE" PA_RUNTIME_ROOT_DIR="$PA_RUNTIME_ROOT_DIR" go -C source/services/daemon-go run ./cmd/personal-agent-daemon \
  --listen-mode tcp \
  --listen-address "$DAEMON_TCP_ADDR" \
  --auth-token-file "$DAEMON_AUTH_TOKEN_FILE" \
  --db "$DAEMON_DB_PATH" \
  --lifecycle-host-ops-mode "$DAEMON_LIFECYCLE_HOST_OPS_MODE" &
DAEMON_PROD_PID=$!

# CLI smoke using the same auth-token-file
go -C source/clients/cli-go run ./cmd/personal-agent \
  --mode tcp \
  --address "$DAEMON_TCP_ADDR" \
  --auth-token-file "$DAEMON_AUTH_TOKEN_FILE" \
  smoke

kill "$DAEMON_PROD_PID"
wait "$DAEMON_PROD_PID" 2>/dev/null || true
```

Expected:

- all commands fail before making/serving control-plane requests.
- daemon missing-auth command includes `--auth-token is required`.
- CLI command without auth token includes `--auth-token is required`.
- non-local bind command includes `non-local; use --allow-non-local-bind`.
- `auth bootstrap`/`auth rotate` emit metadata-only JSON (includes `token_file` and `token_sha256`, no raw token).
- production daemon startup without TLS material is rejected (`--runtime-profile=prod requires --tls-cert-file and --tls-key-file`).
- daemon and CLI smoke succeed in local profile when both use the same `--auth-token-file`.

### 3.1 Auth + health enforcement

```bash
curl -s -o /tmp/pa-smoke-noauth.json -w "%{http_code}\n" "http://$DAEMON_TCP_ADDR/v1/capabilities/smoke"
curl -s -o /tmp/pa-smoke-badauth.json -w "%{http_code}\n" -H "Authorization: Bearer wrong-token" "http://$DAEMON_TCP_ADDR/v1/capabilities/smoke"
curl -s -o /tmp/pa-smoke-auth.json -w "%{http_code}\n" -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" "http://$DAEMON_TCP_ADDR/v1/capabilities/smoke"

cat /tmp/pa-smoke-noauth.json
cat /tmp/pa-smoke-auth.json | jq .
```

Expected:

- no-auth HTTP status is `401`.
- bad-auth HTTP status is `401`.
- valid-auth HTTP status is `200`.
- authenticated payload contains `healthy=true`.

### 3.1.1 Auth-scope startup policy enforcement

Start daemon with explicit task-read-only scopes (terminal A):

```bash
PA_RUNTIME_PROFILE="$PA_RUNTIME_PROFILE" PA_RUNTIME_ROOT_DIR="$PA_RUNTIME_ROOT_DIR" go -C source/services/daemon-go run ./cmd/personal-agent-daemon \
  --listen-mode tcp \
  --listen-address "$DAEMON_TCP_ADDR" \
  --auth-token "$DAEMON_AUTH_TOKEN" \
  --auth-token-scopes "tasks:read" \
  --db "$DAEMON_DB_PATH" \
  --lifecycle-host-ops-mode "$DAEMON_LIFECYCLE_HOST_OPS_MODE"
```

Run checks (terminal B):

```bash
# Privileged daemon lifecycle read should be denied under tasks-only scope
curl -s -o /tmp/pa-scope-daemon-status.json -w "%{http_code}\n" \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  "http://$DAEMON_TCP_ADDR/v1/daemon/lifecycle/status"

# Task status read remains scope-authorized (status may be 200/404 based on fixture state, but not 403)
curl -s -o /tmp/pa-scope-task-status.json -w "%{http_code}\n" \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  "http://$DAEMON_TCP_ADDR/v1/tasks/task-scope-check"

cat /tmp/pa-scope-daemon-status.json | jq .
```

Expected:

- daemon lifecycle status request returns HTTP `403`.
- forbidden payload uses typed authorization failure (`error.code=auth_forbidden`).
- forbidden details include `required_scopes[0]=daemon:read` and `granted_scopes` includes `tasks:read`.
- task status route does not return `403` under `tasks:read` scope.

### 3.2 Transport-backed capability smoke

```bash
pa_tcp smoke
```

Expected:

- command succeeds and returns daemon capability JSON (`healthy=true`, channels/connectors list present).

### 3.2.1 Daemon runtime discovery metadata API

```bash
curl -sS -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  "http://$DAEMON_TCP_ADDR/v1/meta/capabilities" | jq .

pa_tcp --output json-compact meta capabilities
```

Expected:

- authenticated metadata route returns HTTP `200`.
- payload includes:
  - `api_version` (`v1`)
  - `route_groups[]`
  - `realtime_event_types[]`
  - `client_signal_types[]`
  - `protocol_modes[]`
  - `transport_listener_modes[]`
- `client_signal_types[]` includes `cancel`.
- `protocol_modes[]` includes `http_json` and `websocket_json`.
- `realtime_event_types[]` includes `turn_item_delta`, `chat_completed`, and `chat_error` in addition to task lifecycle events.

### 3.2.2 Daemon worker-runtime execution/error regressions

Run focused worker-runtime regressions for channel, connector, and cloudflared entrypoints:

```bash
go -C source/services/daemon-go test ./cmd/personal-agent-daemon \
  -run 'Test(ChannelWorkerRuntimeUnsupportedOperation|DecodeChannelWorkerPayloadRequiresPayload|WriteChannelWorkerErrorEnvelope|ExecuteTwilioConnectorWorkerOperationUnsupported|ExecuteTwilioConnectorWorkerOperationDecodeFailure|DecodeChannelConnectorPayloadRequiresPayload|ExecuteMessagesConnectorWorkerOperationUnsupported|WriteWorkerErrorEnvelope|CloudflaredWorkerStateExecute.*|LoadDaemonPluginWorkers.*|ResolveDaemonPluginWorkersManifestPath.*)' \
  -count=1
```

Expected:

- test run passes.
- channel-worker runtime coverage verifies unsupported operations, payload decode failure handling, and structured JSON error envelopes.
- connector-worker runtime coverage verifies payload decode failure handling, unsupported operations, and structured JSON error envelopes.
- cloudflared worker runtime coverage verifies unsupported operation handling, decode failures, and deterministic dry-run `version`/`exec` execution paths.
- plugin-worker bootstrap coverage verifies manifest loading, env/flag manifest-path resolution, connector-vs-channel argument shaping, and duplicate/invalid manifest guardrails.

### 3.3 Daemon lifecycle status + control APIs

```bash
curl -sS -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  "http://$DAEMON_TCP_ADDR/v1/daemon/lifecycle/status" | jq .

curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"action":"start","reason":"manual ensure running"}' \
  "http://$DAEMON_TCP_ADDR/v1/daemon/lifecycle/control" | jq .

curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"action":"restart","reason":"manual restart validation"}' \
  "http://$DAEMON_TCP_ADDR/v1/daemon/lifecycle/control" | jq .

curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"action":"install","reason":"manual install validation"}' \
  "http://$DAEMON_TCP_ADDR/v1/daemon/lifecycle/control" | jq .

curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"action":"repair","reason":"manual repair validation"}' \
  "http://$DAEMON_TCP_ADDR/v1/daemon/lifecycle/control" | jq .

curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"action":"uninstall","reason":"manual uninstall validation"}' \
  "http://$DAEMON_TCP_ADDR/v1/daemon/lifecycle/control" | jq .
```

Expected:

- status payload includes `lifecycle_state`, `setup_state`, `install_state`, `controls`, `control_operation`, typed `health_classification` fields (`overall_state`, `core_runtime_state`, `plugin_runtime_state`, `blocking`), and typed `control_auth` metadata (`state`, `source`, `remediation_hints`).
- when daemon starts with `--auth-token "$DAEMON_AUTH_TOKEN"` and `$DAEMON_AUTH_TOKEN=daemon-test-token`, status reports `control_auth.state=configured` and `control_auth.source=auth_token_flag`.
- `start` control is idempotent (`accepted=true`, `idempotent=true`, `operation_state=succeeded`) when daemon is already running.

### 3.3.a Single-writer execution transaction regression

Run focused `agentexec` regression coverage to ensure message dispatch can perform SQLite round-trips without deadlocking when the DB pool is single-writer (`MaxOpenConns=1`):

```bash
go -C source/services/daemon-go test ./internal/core/service/agentexec \
  -run 'Test(ExecuteUsesTypedNativeActionForSingleBrowserToolOperation|PlanStepsMailUnreadSummaryUsesCanonicalCapability|ExecuteMessageDispatchCanReenterSQLiteWithSingleWriterPool)' \
  -count=1
```

Expected:

- suite passes.
- `ExecuteMessageDispatchCanReenterSQLiteWithSingleWriterPool` passes, proving message dispatch DB access occurs outside long-lived execution transactions.
- no `context deadline exceeded` starvation when `message_send` workflows execute under single-writer SQLite settings.

### 3.3.0 Principal-aware control-plane rate-limit regressions

Run focused transport regressions for per-route principal-aware limiter behavior and typed `429` metadata:

```bash
go -C source/services/daemon-go test ./internal/transport \
  -run 'TestTransportControlRateLimit(ReturnsTyped429AndResetsAfterWindow|IsPrincipalAwarePerEndpoint|CoversHighRiskMutatingRoutes)' \
  -count=1
```

Expected:

- suite passes.
- limiter keying is principal-aware (same endpoint + different principals do not share exhausted buckets).
- high-risk mutating control routes (`tasks`, `agent`, `chat`, `automation`, `daemon lifecycle control`) consistently emit typed `429` once limits are exceeded.
- typed `429` details include deterministic retry metadata (`retry_after_seconds`, `reset_at`) and bucket context (`endpoint`, `scope_type`, `scope_key`, `bucket_key`).

### 3.3.1 Unified-turn chat transport + realtime lifecycle regression suite

Run focused transport regressions for canonical unified-turn routing, replay/persona APIs, and realtime lifecycle publication:

```bash
go -C source/services/daemon-go test ./internal/transport \
  -run 'TestTransport(ChatTurn(RouteIncludesTaskRunCorrelationMetadata|UsesCanonicalTurnItemsContract|HistoryRoute|PublishesRealtimeLifecycleEvents|PublishesRealtimeLifecycleEventsForMultipleToolCalls|FailurePublishesRealtimeChatError)|ChatPersonaPolicyRoutes)' \
  -count=1
```

Expected:

- suite passes.
- `/v1/chat/turn` uses canonical typed `items[]` contract (no ask/act mode field).
- `/v1/chat/history` and `/v1/chat/persona/{get,set}` routes return typed responses and forward request scopes deterministically.
- successful chat turns publish realtime lifecycle completion events with typed turn/tool item metadata.
- streamed `turn_item_delta` payloads preserve raw whitespace/newline tokens (no trim-induced word concatenation).
- successful multi-tool chat turns publish repeated typed tool lifecycle events (`tool_call_started`/`tool_call_output`/`tool_call_completed`) for each tool call.
- failed chat turns publish realtime lifecycle error events with correlation metadata.

### 3.3.1a Unified-turn model-only streaming callback regression suite

Run focused daemonruntime regression for planner-direct model-only replies with streaming callback enabled:

```bash
go -C source/services/daemon-go test ./internal/daemonruntime \
  -run 'TestUnifiedTurnServiceChatTurnStreamsModelOnlyResponseWhenTokenCallbackProvided' \
  -count=1
```

Expected:

- suite passes.
- planner-direct model-only replies perform planner + response-synthesis model calls when callback streaming is enabled.
- token callback receives streamed deltas for the synthesized assistant response (driving realtime `turn_item_delta` publication on `/v1/chat/turn` server path).

### 3.3.2 Unified-turn capability-driven tool registry regression suite

Run focused daemonruntime regressions for capability-derived unified-turn tool exposure:

```bash
go -C source/services/daemon-go test ./internal/daemonruntime \
  -run 'TestUnifiedTurnServiceResolveAvailableTools(SupportsDotCapabilityKeys|SkipsBlockedConnectorCapabilities|IncludesExpandedConnectorCapabilities|DeduplicatesAliasCapabilities)' \
  -count=1
```

Expected:

- suite passes.
- planner tool inventory is derived from ready connector capabilities.
- blocked connectors do not contribute planner tools.
- capability aliases (`mail.send` vs `mail_send`) deduplicate to one canonical tool entry.
- expanded ready capabilities expose corresponding tools without editing a static hardcoded tool list.

### 3.3.3 Unified-turn iterative orchestration regression suite

Run focused daemonruntime regressions for iterative `model -> tool* -> model` turn behavior:

```bash
go -C source/services/daemon-go test ./internal/daemonruntime \
  -run 'TestUnifiedTurnService(ChatTurnSupportsIterativeToolLoop|ChatTurnStopsAtToolCallLimit)' \
  -count=1
```

Expected:

- suite passes.
- one chat turn can execute multiple sequential tool calls before assistant completion.
- iterative loop enforces deterministic tool-call cap and records stop metadata (`stop_reason=tool_call_limit_reached`).
- approval or no-action planner outputs stop the loop deterministically and continue with safe assistant synthesis.

### 3.3.3a Typed tool payload bridge regression suite

Run focused regressions for typed `native_action` tool execution payloads and adapter `step.input` contracts:

```bash
go -C source/services/daemon-go test ./internal/daemonruntime \
  -run 'Test(UnifiedTurnServiceChatTurn(ModelToolModelSuccess|SupportsIterativeToolLoop)|BuildMailNativeActionUnreadSummarySupportsOptionalLimit)' \
  -count=1
go -C source/services/daemon-go test ./internal/core/service/agentexec \
  -run 'TestExecuteUsesTypedNativeActionForSingleBrowserToolOperation' \
  -count=1
go -C source/services/daemon-go test ./internal/connectors/adapters/mail ./internal/connectors/adapters/calendar ./internal/connectors/adapters/browser ./internal/connectors/adapters/finder \
  -count=1
```

Expected:

- suite passes.
- unified-turn tool calls invoke agent runtime with typed `native_action` payloads (no synthetic `request_text` bridge).
- mail unread-summary tool maps typed `limit` input to `native_action.mail.operation=summarize_unread`.
- execution engine plans/persists typed step input payloads and executes single-operation native actions deterministically.
- mail/calendar/browser/finder adapters consume core fields from `step.input` payloads (recipient/title/url/path/query/root_path) for execution.
- mail adapter unread-summary execution reads persisted inbound mail events and returns auditable unread counts/item summaries.
- no legacy request-text bridge payloads are required for tool execution.

### 3.3.3b Chat-action reliability cutover regression suite

Run focused regressions for execution-origin mapping, planner repair/fallback, end-to-end tool chains, and provider-native tool-calling:

```bash
go -C source/services/daemon-go test ./internal/daemonruntime \
  -run 'TestUnifiedTurnServiceChatTurn(ModelToolModelSuccess|PolicyRequireApprovalFlow|RepairsMalformedPlannerOutputBeforeToolExecution|FallsBackAfterPlannerRepairRetriesExhausted|NaturalLanguageEmailAndBrowserToolChain|NaturalLanguageEmailFailureEdge|NaturalLanguageBrowserApprovalEdge|ReturnsModelRouteRemediationOnPlannerRouteFailure|ToolExecutionFailure)' \
  -count=1
go -C source/services/daemon-go test ./internal/daemonruntime \
  -run 'TestModelRoute(SimulationAndExplainabilityTaskPolicySelected|SelectRejectsNonActionCapableChatPolicy|SimulationChatPolicyMisconfiguredFallsBackToActionCapableCandidate|ResolveChatFailsWithoutActionCapableCandidates)' \
  -count=1
go -C source/services/daemon-go test ./internal/chatruntime \
  -run 'TestStreamAssistantResponse(OpenAINativeToolCalling|OllamaNativeToolCalling|UnsupportedProviderToolFallback)' \
  -count=1
go -C source/services/daemon-go test ./internal/daemonruntime \
  -run 'Test(PlannerToolSpecsFromPromptParsesUnifiedPrompt|ProviderModelChatServiceChatTurnPlannerPromptUsesNativeToolCallingForOllama)' \
  -count=1
```

Expected:

- suite passes.
- tool execution origin mapping is deterministic: app/message-channel turns execute with origin `app`, voice turns execute with origin `voice`.
- malformed planner output triggers bounded repair attempts before fallback; exhaustion records deterministic metadata (`planner_repair_attempts`, `stop_reason=planner_output_invalid`) with remediation hints.
- natural-language email/browser tool chains produce canonical `tool_call` and `tool_result` item progression without dropping to message-only paths.
- chat route simulation/select rejects non-action-capable chat policies and falls back to action-capable candidates when available.
- provider-native tool-calling paths are exercised for OpenAI/Ollama planner turns and unsupported providers deterministically fall back without runtime failure.
- model-route and tool-execution failure paths include actionable remediation metadata for client guidance.
- no legacy chat-action shims are accepted (`ask/act` mode fields, request-text bridge payloads, synthetic origin aliases).

### 3.3.4 Action-readiness classification regression suite

Run focused daemonruntime regressions for channel/connector action-readiness classification:

```bash
go -C source/services/daemon-go test ./internal/daemonruntime \
  -run 'Test(ChannelActionReadinessClassification|ConnectorActionReadinessClassification)' \
  -count=1
```

Expected:

- suite passes.
- readiness classification resolves `ready`, `blocked`, and `degraded` deterministically for channel/connector cards.
- blocker codes include canonical reasons (`config_incomplete`, `credentials_missing`, `permission_missing`, `worker_unavailable`).
- `restart` control returns accepted response and daemon becomes reachable again at the same endpoint shortly after.
- after restart completes, channel status for logical `message` reports `worker.plugin_id=messages.daemon` and does not return sticky `worker.state=failed` with `worker.last_error=manual restart requested`.
- after restart completes, configured core logical channels (`app`, `message`) report `status=ready` with expected worker plugin IDs; when Twilio is configured with credentials, logical `voice` also returns `status=ready`.
- `health_classification` semantics distinguish core runtime readiness (`install_required`, `database_unavailable`, `control_plane_unavailable`) from plugin runtime degradation (`plugin_runtime_state=degraded`) so UI can map severity without inferring from legacy fields.
- `install`, `repair`, and `uninstall` controls return accepted in-progress responses (`operation_state=in_progress`) and lifecycle status reflects terminal completion (`control_operation.state=succeeded|failed`) for the corresponding action.
- default daemon mode for host setup actions is `unsupported`; for this guide use `DAEMON_LIFECYCLE_HOST_OPS_MODE=dry-run` (non-destructive) or `apply` (real host operations).

### 3.3.0 Lifecycle host-ops/startup regression tests

Run focused lifecycle host-op and lifecycle-action-state regressions:

```bash
go -C source/services/daemon-go test ./cmd/personal-agent-daemon -run 'TestLifecycleHost' -count=1
go -C source/services/daemon-go test ./internal/daemonruntime -run 'TestDaemonLifecycleControl(InstallTracksInProgressAndSuccess|RepairReflectsFailure|UninstallTracksInProgressAndSuccess)' -count=1
```

Expected:

- host-op regressions pass for render/execution paths across macOS/Linux/Windows, including dry-run install/repair/uninstall behavior and unsupported-platform fallback handling.
- lifecycle action-state regressions pass and verify deterministic control semantics: first `install|uninstall` control request reports `operation_state=in_progress`, replay during in-flight execution is idempotent, and terminal status transitions settle to `succeeded|failed` with stable action metadata.

### 3.3.1 Daemon plugin lifecycle history query API

```bash
curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"workspace_id":"daemon","limit":1}' \
  "http://$DAEMON_TCP_ADDR/v1/daemon/lifecycle/plugins/history" | jq .

LIFECYCLE_PLUGIN_HISTORY_CURSOR_CREATED_AT="$(curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"workspace_id":"daemon","limit":1}' \
  "http://$DAEMON_TCP_ADDR/v1/daemon/lifecycle/plugins/history" | jq -r '.next_cursor_created_at')"

LIFECYCLE_PLUGIN_HISTORY_CURSOR_ID="$(curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"workspace_id":"daemon","limit":1}' \
  "http://$DAEMON_TCP_ADDR/v1/daemon/lifecycle/plugins/history" | jq -r '.next_cursor_id')"

curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"daemon\",\"cursor_created_at\":\"$LIFECYCLE_PLUGIN_HISTORY_CURSOR_CREATED_AT\",\"cursor_id\":\"$LIFECYCLE_PLUGIN_HISTORY_CURSOR_ID\",\"limit\":1}" \
  "http://$DAEMON_TCP_ADDR/v1/daemon/lifecycle/plugins/history" | jq .
```

Expected:

- response envelope is deterministic: `{workspace_id, items[], has_more, next_cursor_created_at, next_cursor_id}`.
- `items[]` rows include lifecycle context fields: `plugin_id`, `kind`, `state`, `event_type`, `process_id`, `restart_count`, `reason`, `error`, `restart_event`, `failure_event`, `recovery_event`, and `occurred_at`.
- ordering is stable and descending (`occurred_at DESC`, `audit_id DESC`); cursor pagination returns the next older slice without duplicates.
- filters (`plugin_id`, `kind`, `state`, `event_type`) constrain returned rows deterministically while preserving ordering and pagination behavior.
- restart/failure/recovery classification is explicit via `restart_event`, `failure_event`, `recovery_event`, with `reason` conveying normalized lifecycle cause (`health_timeout`, `restart_after_error`, `worker_recovered`, etc.).

### 3.3.2 Context memory + retrieval query APIs

Seed deterministic context fixtures:

```bash
cat > "$TEST_RUNTIME_ROOT/context-query-fixture.seed.go" <<'EOF'
package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	if len(os.Args) != 3 {
		panic("usage: context-query-fixture.seed.go <db-path> <workspace-id>")
	}
	dbPath := os.Args[1]
	workspaceID := os.Args[2]

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	stmts := []string{
		fmt.Sprintf(`INSERT INTO actors(id, workspace_id, actor_type, display_name, status, created_at, updated_at) VALUES ('actor.context.a', '%s', 'human', 'Context A', 'ACTIVE', '2026-02-25T00:00:00Z', '2026-02-25T00:00:00Z') ON CONFLICT(id) DO NOTHING`, workspaceID),
		fmt.Sprintf(`INSERT INTO actors(id, workspace_id, actor_type, display_name, status, created_at, updated_at) VALUES ('actor.context.b', '%s', 'human', 'Context B', 'ACTIVE', '2026-02-25T00:00:00Z', '2026-02-25T00:00:00Z') ON CONFLICT(id) DO NOTHING`, workspaceID),
		fmt.Sprintf(`INSERT INTO memory_items(id, workspace_id, owner_principal_actor_id, scope_type, key, value_json, status, source_summary, created_at, updated_at) VALUES ('manual-mem-1', '%s', 'actor.context.a', 'conversation', 'manual-key-1', '{"kind":"summary","token_estimate":9,"content":"manual one"}', 'ACTIVE', 'event://manual-1', '2026-02-25T00:00:01Z', '2026-02-25T00:00:01Z') ON CONFLICT(id) DO NOTHING`, workspaceID),
		fmt.Sprintf(`INSERT INTO memory_items(id, workspace_id, owner_principal_actor_id, scope_type, key, value_json, status, source_summary, created_at, updated_at) VALUES ('manual-mem-2', '%s', 'actor.context.a', 'conversation', 'manual-key-2', '{"kind":"summary","token_estimate":11,"content":"manual two"}', 'ACTIVE', 'event://manual-2', '2026-02-25T00:00:02Z', '2026-02-25T00:00:02Z') ON CONFLICT(id) DO NOTHING`, workspaceID),
		`INSERT INTO memory_sources(id, memory_item_id, source_type, source_ref, created_at) VALUES ('manual-src-1', 'manual-mem-1', 'comm_event', 'event://manual-1', '2026-02-25T00:00:01Z') ON CONFLICT(id) DO NOTHING`,
		`INSERT INTO memory_sources(id, memory_item_id, source_type, source_ref, created_at) VALUES ('manual-src-2', 'manual-mem-2', 'comm_event', 'event://manual-2', '2026-02-25T00:00:02Z') ON CONFLICT(id) DO NOTHING`,
		fmt.Sprintf(`INSERT INTO memory_candidates(id, workspace_id, owner_principal_actor_id, candidate_json, score, status, created_at) VALUES ('manual-cand-1', '%s', 'actor.context.a', '{"kind":"summary","token_estimate":20,"source_ids":["manual-mem-1","manual-mem-2"],"source_refs":["event://manual-1","event://manual-2"]}', 0.95, 'PENDING', '2026-02-25T00:00:03Z') ON CONFLICT(id) DO NOTHING`, workspaceID),
		fmt.Sprintf(`INSERT INTO context_documents(id, workspace_id, owner_principal_actor_id, source_uri, checksum, created_at) VALUES ('manual-doc-1', '%s', 'actor.context.a', 'memory://manual/doc-1', 'manual-checksum-1', '2026-02-25T00:00:01Z') ON CONFLICT(id) DO NOTHING`, workspaceID),
		`INSERT INTO context_chunks(id, document_id, chunk_index, text_body, token_count, created_at) VALUES ('manual-chunk-1', 'manual-doc-1', 0, 'manual retrieval chunk', 7, '2026-02-25T00:00:01Z') ON CONFLICT(id) DO NOTHING`,
	}

	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			panic(err)
		}
	}
}
EOF
go -C source/services/daemon-go run "$TEST_RUNTIME_ROOT/context-query-fixture.seed.go" "$DAEMON_DB_PATH" "$WORKSPACE"
rm -f "$TEST_RUNTIME_ROOT/context-query-fixture.seed.go"
```

Query context APIs:

```bash
CONTEXT_MEMORY_PAGE1="$(curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"owner_actor_id\":\"actor.context.a\",\"source_type\":\"comm_event\",\"limit\":1}" \
  "http://$DAEMON_TCP_ADDR/v1/context/memory/inventory")"
echo "$CONTEXT_MEMORY_PAGE1" | jq .

CONTEXT_MEMORY_CURSOR_UPDATED_AT="$(echo "$CONTEXT_MEMORY_PAGE1" | jq -r '.next_cursor_updated_at')"
CONTEXT_MEMORY_CURSOR_ID="$(echo "$CONTEXT_MEMORY_PAGE1" | jq -r '.next_cursor_id')"

curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"owner_actor_id\":\"actor.context.a\",\"source_type\":\"comm_event\",\"cursor_updated_at\":\"$CONTEXT_MEMORY_CURSOR_UPDATED_AT\",\"cursor_id\":\"$CONTEXT_MEMORY_CURSOR_ID\",\"limit\":1}" \
  "http://$DAEMON_TCP_ADDR/v1/context/memory/inventory" | jq .

curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"owner_actor_id\":\"actor.context.a\",\"status\":\"PENDING\",\"limit\":20}" \
  "http://$DAEMON_TCP_ADDR/v1/context/memory/compaction-candidates" | jq .

curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"owner_actor_id\":\"actor.context.a\",\"source_uri_query\":\"memory://manual\",\"limit\":20}" \
  "http://$DAEMON_TCP_ADDR/v1/context/retrieval/documents" | jq .

curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"document_id\":\"manual-doc-1\",\"chunk_text_query\":\"manual retrieval\",\"limit\":20}" \
  "http://$DAEMON_TCP_ADDR/v1/context/retrieval/chunks" | jq .
```

Expected:

- memory inventory response is deterministic: `{workspace_id, items[], has_more, next_cursor_updated_at, next_cursor_id}`.
- memory inventory filters (`owner_actor_id`, `source_type`, `source_ref_query`, `status`) constrain rows while preserving `updated_at DESC, memory_id DESC` paging order.
- memory inventory items include parsed metadata (`kind`, `is_canonical`, `token_estimate`) and expanded `sources[]`.
- memory candidate response is deterministic: `{workspace_id, items[], has_more, next_cursor_created_at, next_cursor_id}` with parsed candidate preview metadata (`candidate_kind`, `token_estimate`, `source_ids[]`, `source_refs[]`).
- retrieval document and chunk responses are deterministic and cursor-paginated (`created_at DESC, id DESC`) with principal/source/document filters.

For stop validation (run when you are done with current daemon session):

```bash
curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"action":"stop","reason":"manual shutdown validation"}' \
  "http://$DAEMON_TCP_ADDR/v1/daemon/lifecycle/control" | jq .
```

### 3.4 Channel and connector status/config summary APIs

```bash
curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\"}" \
  "http://$DAEMON_TCP_ADDR/v1/channels/status" | jq .

curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\"}" \
  "http://$DAEMON_TCP_ADDR/v1/connectors/status" | jq .
```

Expected:

- channel payload includes logical cards: `app`, `message`, `voice`.
- connector payload includes canonical cards for `builtin.app`, `imessage`, `twilio`, `mail`, `calendar`, `browser`, `finder`, and `cloudflared`.
- each card includes stable `status`, `configured`, worker summary fields (when available), and `remediation_actions[]`.
- each card includes action-level readiness metadata:
  - `action_readiness` (`ready|blocked|degraded`)
  - `action_blockers[]` with `code`, `message`, and optional remediation metadata.
- each status card includes `config_field_descriptors[]` entries with typed form metadata keys:
  - `key`, `label`, `type`, `required`, `editable`, `enum_options`, `secret`, `write_only`, `help_text`.
  - default values are deterministic when fields are not explicitly set: `enum_options=[]`, `secret=false`, `write_only=false`, `help_text=""`.
- `twilio` connector descriptors include write-only secret inputs (for example `auth_token_value`) and SecretRef name fields.
- each connector card includes `configuration.status_reason` so diagnostics can distinguish runtime failures, permission blockers, and cloudflared binary availability.
- when `imessage` ingest failure is a permissions error (for example `operation not permitted` reading `chat.db`), connector status surfaces `configuration.status_reason=permission_missing` and `configuration.permission_state=missing`.
- when calendar Automation access is denied (`-1743`) during connector execute-probe checks, connector status surfaces `configuration.status_reason=permission_missing`, `action_readiness=blocked`, and a `permission_missing` action blocker.
- when `cloudflared` reports `configuration.status_reason == "cloudflared_binary_missing"`, remediation actions include `install_cloudflared_connector`.
- each status-card remediation action includes typed fields: `identifier`, `label`, `intent`, optional `destination`, optional `parameters`, plus `enabled`/`recommended`.

### 3.4.0 Channel and connector config mutation + test-operation APIs

```bash
curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"channel_id\":\"app\",\"configuration\":{\"enabled\":true,\"transport\":\"daemon_realtime\"},\"merge\":true}" \
  "http://$DAEMON_TCP_ADDR/v1/channels/config/upsert" | jq .

pa_tcp secret set --workspace "$WORKSPACE" --name TWILIO_ACCOUNT_SID_UI_UPSERT --value "ACDAEMONUIUPSERT"
pa_tcp secret set --workspace "$WORKSPACE" --name TWILIO_AUTH_TOKEN_UI_UPSERT --value "daemon-ui-upsert-token"

curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"connector_id\":\"twilio\",\"configuration\":{\"account_sid_secret_name\":\"TWILIO_ACCOUNT_SID_UI_UPSERT\",\"auth_token_secret_name\":\"TWILIO_AUTH_TOKEN_UI_UPSERT\",\"number\":\"+15555550009\",\"endpoint\":\"https://api.twilio.com\"},\"merge\":true}" \
  "http://$DAEMON_TCP_ADDR/v1/connectors/config/upsert" | jq .

pa_tcp connector twilio get --workspace "$WORKSPACE"

curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"connector_id\":\"mail\",\"configuration\":{\"scope\":\"inbox\"},\"merge\":true}" \
  "http://$DAEMON_TCP_ADDR/v1/connectors/config/upsert" | jq .

curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"channel_id\":\"app\",\"operation\":\"health\"}" \
  "http://$DAEMON_TCP_ADDR/v1/channels/test" | jq .

curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"connector_id\":\"mail\",\"operation\":\"health\"}" \
  "http://$DAEMON_TCP_ADDR/v1/connectors/test" | jq .
```

Expected:

- channel config upsert response is deterministic: `{workspace_id, channel_id, configuration, updated_at}`.
- Twilio connector config upsert with registered secret refs canonicalizes to Twilio runtime fields (`account_sid_secret_name`, `auth_token_secret_name`, `sms_number`, `voice_number`) and returns `credentials_configured=true`.
- `pa_tcp connector twilio get` returns canonical Twilio config matching the UI upserted secret names and number values.
- connector config upsert response is deterministic: `{workspace_id, connector_id, configuration, updated_at}`.
- channel test response is deterministic: `{workspace_id, channel_id, operation, success, status, summary, checked_at, details}`.
- connector test response is deterministic: `{workspace_id, connector_id, operation, success, status, summary, checked_at, details}`.
- unsupported operations return a validation error (`operation must be health`) without process-level transport failure.

### 3.4.0.1 Channel-mapping migration matrix (clean install + upgrade aliases + Twilio config-once parity)

```bash
pa_tcp channel mapping list --workspace "$WORKSPACE" --channel message
pa_tcp channel mapping list --workspace "$WORKSPACE" --channel voice

pa_tcp channel mapping disable --workspace "$WORKSPACE" --channel message --connector twilio
pa_tcp channel mapping prioritize --workspace "$WORKSPACE" --channel message --connector twilio --priority 1
pa_tcp channel mapping enable --workspace "$WORKSPACE" --channel message --connector twilio --priority 1
pa_tcp channel mapping list --workspace "$WORKSPACE" --channel message

pa_tcp channel mapping disable --workspace "$WORKSPACE" --channel voice --connector twilio
pa_tcp channel mapping enable --workspace "$WORKSPACE" --channel voice --connector twilio --priority 1
pa_tcp channel mapping list --workspace "$WORKSPACE" --channel voice

pa_tcp connector twilio get --workspace "$WORKSPACE"
pa_tcp channel mapping list --workspace "$WORKSPACE" --channel message
pa_tcp channel mapping list --workspace "$WORKSPACE" --channel voice
```

Expected:

- clean-install defaults include logical-channel bindings:
  - `message` has both `imessage` and `twilio`.
  - `voice` has `twilio`.
- enable/disable flows are deterministic for both `message` and `voice` channels (`connector_id=twilio`).
- unified Twilio config-once parity:
  - one Twilio config record (`connector twilio get`) returns both `sms_number` and `voice_number`.
  - `twilio` remains enabled in both `message` and `voice` channel mappings after Twilio config mutation.

### 3.4.1 Channel and connector diagnostics summary APIs

```bash
curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\"}" \
  "http://$DAEMON_TCP_ADDR/v1/channels/diagnostics" | jq .

curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\"}" \
  "http://$DAEMON_TCP_ADDR/v1/connectors/diagnostics" | jq .
```

Expected:

- channel diagnostics envelope is deterministic: `{workspace_id, diagnostics[]}`.
- connector diagnostics envelope is deterministic: `{workspace_id, diagnostics[]}`.
- each diagnostics item includes `worker_health` (`registered` plus optional `worker` snapshot).
- each diagnostics item includes `remediation_actions[]` with typed fields (`identifier`, `label`, `intent`, optional `destination`, optional `parameters`, `enabled`, `recommended`).
- filtering by `channel_id` or `connector_id` returns only the requested diagnostics item when present.

### 3.4.1.0 TCC remediation destination matrix (connector/channel)

Use the previously captured diagnostics payload variables from section `3.4.1`.

```bash
echo "$CONNECTOR_DIAGNOSTICS_JSON" | jq '
  [
    .diagnostics[]
    | select(.connector_id == "mail" or .connector_id == "calendar" or .connector_id == "browser" or .connector_id == "finder")
    | {
        connector_id,
        destination: (
          [.remediation_actions[]? | select(.identifier == "open_connector_system_settings") | .destination][0]
        )
      }
  ]'

echo "$CONNECTOR_DIAGNOSTICS_JSON" | jq '
  [
    .diagnostics[]
    | select(.connector_id == "imessage")
    | {
        connector_id,
        destination: (
          [.remediation_actions[]? | select(.identifier == "open_connector_system_settings") | .destination][0]
        )
      }
  ]'

echo "$CONNECTOR_DIAGNOSTICS_JSON" | jq '
  [
    .diagnostics[]
    | select(.connector_id == "imessage")
    | {
        connector_id,
        destination: (
          [.remediation_actions[]? | select(.identifier == "open_imessage_system_settings") | .destination][0]
        )
      }
  ]'

echo "$CHANNEL_DIAGNOSTICS_JSON" | jq '
  [
    .diagnostics[]
    | select(.channel_id == "message")
    | {
        channel_id,
        destination: (
          [.remediation_actions[]? | select(.identifier == "open_channel_system_settings") | .destination][0]
        )
      }
  ]'
```

Expected:

- connector matrix maps to system-settings destinations:
  - `mail` -> `ui://system-settings/privacy/automation`
  - `calendar` -> `ui://system-settings/privacy/automation`
  - `browser` -> `ui://system-settings/privacy/automation`
  - `finder` -> `ui://system-settings/privacy/automation`
  - `imessage` (`open_connector_system_settings`) -> `ui://system-settings/privacy/automation`
  - `imessage` (`open_imessage_system_settings`) -> `ui://system-settings/privacy/full-disk-access`
- channel matrix:
  - `message` emits `open_channel_system_settings` with destination `ui://system-settings/privacy/full-disk-access` when ingest failure remediation is active.

### 3.4.1.1 Connector permission request API (daemon-owned prompt path)

```bash
curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"connector_id\":\"mail\"}" \
  "http://$DAEMON_TCP_ADDR/v1/connectors/permission/request" | jq .

curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"connector_id\":\"calendar\"}" \
  "http://$DAEMON_TCP_ADDR/v1/connectors/permission/request" | jq .

curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"connector_id\":\"imessage\"}" \
  "http://$DAEMON_TCP_ADDR/v1/connectors/permission/request" | jq .
```

Expected:

- response envelope is deterministic: `{workspace_id, connector_id, permission_state, message}`.
- `permission_state` is deterministic (`granted|missing|unknown`) and native-tooling absence returns a typed payload instead of transport failure.
- `calendar` permission request retries after warm-launch when Calendar is not running and should return deterministic automation remediation (`allow Personal Agent Daemon`) instead of `application unavailable` messaging.
- `imessage` request path is daemon-owned and returns explicit dual-permission remediation/state guidance for Automation + Full Disk Access checks.
- endpoint is daemon-owned so UI permission actions can route through daemon identity.

### 3.4.1.2 Model catalog discover/add/remove APIs

```bash
pa_tcp provider set --workspace "$WORKSPACE" --provider ollama --endpoint "http://127.0.0.1:11434"

curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"provider\":\"ollama\"}" \
  "http://$DAEMON_TCP_ADDR/v1/models/discover" | jq .

curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"provider\":\"ollama\",\"model_key\":\"llama3.2-custom\",\"enabled\":true}" \
  "http://$DAEMON_TCP_ADDR/v1/models/add" | jq .

curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"provider\":\"ollama\",\"model_key\":\"llama3.2-custom\"}" \
  "http://$DAEMON_TCP_ADDR/v1/models/remove" | jq .
```

Expected:

- discover response envelope is deterministic: `{workspace_id, results[]}`.
- each `results[]` row includes `provider`, `success`, and `models[]` entries.
- add response returns catalog record fields including `provider`, `model_key`, `enabled`, and `updated_at`.
- remove response returns deterministic removal fields: `provider`, `model_key`, `removed=true`, and `removed_at`.

### 3.4.1.2.1 Model-route simulation and explainability APIs

```bash
MODEL_ROUTE_SIMULATE_JSON="$(curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"task_class\":\"chat\"}" \
  "http://$DAEMON_TCP_ADDR/v1/models/route/simulate")"
echo "$MODEL_ROUTE_SIMULATE_JSON" | jq .

MODEL_ROUTE_PROVIDER="$(echo "$MODEL_ROUTE_SIMULATE_JSON" | jq -r '.selected_provider')"
MODEL_ROUTE_MODEL="$(echo "$MODEL_ROUTE_SIMULATE_JSON" | jq -r '.selected_model_key')"
MODEL_ROUTE_SOURCE="$(echo "$MODEL_ROUTE_SIMULATE_JSON" | jq -r '.selected_source')"
test -n "$MODEL_ROUTE_PROVIDER" && [ "$MODEL_ROUTE_PROVIDER" != "null" ]
test -n "$MODEL_ROUTE_MODEL" && [ "$MODEL_ROUTE_MODEL" != "null" ]
test -n "$MODEL_ROUTE_SOURCE" && [ "$MODEL_ROUTE_SOURCE" != "null" ]

MODEL_ROUTE_EXPLAIN_JSON="$(curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"task_class\":\"chat\"}" \
  "http://$DAEMON_TCP_ADDR/v1/models/route/explain")"
echo "$MODEL_ROUTE_EXPLAIN_JSON" | jq .
```

Expected:

- simulate response is deterministic and includes selected route fields: `selected_provider`, `selected_model_key`, `selected_source`.
- chat-route simulation/select policy for `task_class=chat` must only resolve action-capable models; non-action-capable selections are rejected deterministically.
- simulate response includes machine-readable decision trace fields:
  - `reason_codes[]`
  - `decisions[]` (`step`, `decision`, `reason_code`, optional provider/model/note)
  - `fallback_chain[]` (`rank`, provider/model, `selected`, `reason_code`)
- exactly one `fallback_chain[]` row is selected.
- explain response includes selected route parity with simulation and adds non-empty explainability text fields: `summary` and `explanations[]`.
- explain response includes the same route-decision metadata envelopes (`reason_codes[]`, `decisions[]`, `fallback_chain[]`) for deterministic client rendering.

### 3.4.1.3 Workspace and principal directory identity APIs

Seed principal directory fixture rows (workspace + actors/principals) with a temporary delegation grant:

```bash
IDENTITY_SEED_GRANT_JSON="$(pa_tcp delegation grant \
  --workspace "$WORKSPACE" \
  --from actor.requester \
  --to actor.approver \
  --scope-type EXECUTION)"
echo "$IDENTITY_SEED_GRANT_JSON" | jq .

IDENTITY_SEED_RULE_ID="$(echo "$IDENTITY_SEED_GRANT_JSON" | jq -r '.id')"
```

Query identity directory APIs:

```bash
curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"include_inactive":true}' \
  "http://$DAEMON_TCP_ADDR/v1/identity/workspaces" | jq .

curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\"}" \
  "http://$DAEMON_TCP_ADDR/v1/identity/principals" | jq .

curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\"}" \
  "http://$DAEMON_TCP_ADDR/v1/identity/context" | jq .

curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"principal_actor_id\":\"actor.approver\",\"source\":\"manual-runner\"}" \
  "http://$DAEMON_TCP_ADDR/v1/identity/context/select-workspace" | jq .
```

Cleanup temporary fixture rule:

```bash
pa_tcp delegation revoke --workspace "$WORKSPACE" --rule-id "$IDENTITY_SEED_RULE_ID"
```

Expected:

- workspace directory response is deterministic: `{active_context, workspaces[]}` with `workspace_id`, aggregate counts (`principal_count`, `actor_count`, `handle_count`), and `is_active`.
- principal directory response is deterministic: `{workspace_id, active_context, principals[]}` and every principal includes `handles[]` mapping rows.
- active context response is deterministic: `{active_context}` including `workspace_id`, `workspace_resolved`, and typed source metadata.
- workspace selection mutation returns updated `active_context` with `workspace_source=selected`; when `principal_actor_id` is provided and valid, `principal_source=selected`.

### 3.4.2 Cloudflared connector control-plane APIs

```bash
pa_tcp connector cloudflared version --workspace "$WORKSPACE"
pa_tcp connector cloudflared exec --workspace "$WORKSPACE" --arg version
```

Expected:

- `version` response includes `available=true`, `binary_path`, and `dry_run=true` when `PA_CLOUDFLARED_DRY_RUN=1`.
- `exec` response includes `success=true`, requested `args`, and deterministic dry-run output.

### 3.4.3 Messages worker outbound send path (iMessage)

```bash
IMESSAGE_SEND_JSON="$(pa_tcp comm send \
  --workspace "$WORKSPACE" \
  --operation-id op-daemon-imessage-direct \
  --source-channel message \
  --destination +15555550123 \
  --message 'daemon imessage bridge transport')"
echo "$IMESSAGE_SEND_JSON" | jq .

IMESSAGE_ATTEMPTS_JSON="$(pa_tcp comm attempts \
  --workspace "$WORKSPACE" \
  --operation-id op-daemon-imessage-direct)"
echo "$IMESSAGE_ATTEMPTS_JSON" | jq .
```

Expected:

- send response includes `success=true`, `result.Channel=imessage`, and at least one attempt in `result.Attempts`.
- attempts response contains at least one row with `channel=imessage` and `status=sent`.

### 3.4.4 Daemon-managed local-source watcher auto-ingest (Messages + Mail + Calendar + Browser)

Create four `ON_COMM_EVENT` automations (one per channel) so you can observe watcher-driven ingestion without running manual `ingest` commands:

```bash
AUTO_WATCH_MESSAGES_JSON="$(pa_tcp automation create --workspace "$WORKSPACE" --subject actor.requester --trigger-type ON_COMM_EVENT --title "Watcher messages auto" --instruction "watcher messages auto" --filter '{"channels":["imessage"],"keywords":{"contains_any":["daemon watcher messages token"]}}')"
AUTO_WATCH_MESSAGES_DIRECTIVE_ID="$(echo "$AUTO_WATCH_MESSAGES_JSON" | jq -r '.directive_id')"

AUTO_WATCH_MAIL_JSON="$(pa_tcp automation create --workspace "$WORKSPACE" --subject actor.requester --trigger-type ON_COMM_EVENT --title "Watcher mail auto" --instruction "watcher mail auto" --filter '{"channels":["mail"],"keywords":{"contains_any":["daemon watcher mail token"]}}')"
AUTO_WATCH_MAIL_DIRECTIVE_ID="$(echo "$AUTO_WATCH_MAIL_JSON" | jq -r '.directive_id')"

AUTO_WATCH_CALENDAR_JSON="$(pa_tcp automation create --workspace "$WORKSPACE" --subject actor.requester --trigger-type ON_COMM_EVENT --title "Watcher calendar auto" --instruction "watcher calendar auto" --filter '{"channels":["calendar"],"keywords":{"contains_any":["daemon watcher calendar token"]}}')"
AUTO_WATCH_CALENDAR_DIRECTIVE_ID="$(echo "$AUTO_WATCH_CALENDAR_JSON" | jq -r '.directive_id')"

AUTO_WATCH_BROWSER_JSON="$(pa_tcp automation create --workspace "$WORKSPACE" --subject actor.requester --trigger-type ON_COMM_EVENT --title "Watcher browser auto" --instruction "watcher browser auto" --filter '{"channels":["browser"],"keywords":{"contains_any":["daemon watcher browser token"]}}')"
AUTO_WATCH_BROWSER_DIRECTIVE_ID="$(echo "$AUTO_WATCH_BROWSER_JSON" | jq -r '.directive_id')"
```

Seed source fixtures using bridge helper commands:

```bash
MESSAGES_FIXTURE_DB="$PWD/out/messages-ingest-fixture.db"
rm -rf "$PA_INBOUND_WATCHER_INBOX_DIR"
rm -f "$MESSAGES_FIXTURE_DB"

cat > "$MESSAGES_FIXTURE_DB.seed.go" <<EOF
package main

import (
  "database/sql"
  "os"
  _ "modernc.org/sqlite"
)

func main() {
  target := os.Args[1]
  _ = os.Remove(target)
  db, err := sql.Open("sqlite", target)
  if err != nil { panic(err) }
  defer db.Close()
  stmts := []string{
    "CREATE TABLE message (ROWID INTEGER PRIMARY KEY, guid TEXT, text TEXT, date INTEGER, is_from_me INTEGER, handle_id INTEGER, service TEXT);",
    "CREATE TABLE chat (ROWID INTEGER PRIMARY KEY, guid TEXT);",
    "CREATE TABLE chat_message_join (chat_id INTEGER, message_id INTEGER);",
    "CREATE TABLE handle (ROWID INTEGER PRIMARY KEY, id TEXT);",
    "INSERT INTO handle(ROWID, id) VALUES (1, '+15555550100');",
    "INSERT INTO chat(ROWID, guid) VALUES (1, 'chat-guid-daemon-1');",
    "INSERT INTO message(ROWID, guid, text, date, is_from_me, handle_id, service) VALUES (1001, 'imessage-guid-daemon-1', 'daemon watcher messages token', 1000000000, 0, 1, 'iMessage');",
    "INSERT INTO chat_message_join(chat_id, message_id) VALUES (1, 1001);",
  }
  for _, stmt := range stmts {
    if _, err := db.Exec(stmt); err != nil { panic(err) }
  }
}
EOF
go -C source/services/daemon-go run "$MESSAGES_FIXTURE_DB.seed.go" "$MESSAGES_FIXTURE_DB"
rm -f "$MESSAGES_FIXTURE_DB.seed.go"

pa_tcp connector bridge setup --workspace "$WORKSPACE" | jq .

pa_tcp connector mail handoff \
  --workspace "$WORKSPACE" \
  --source-scope "mailbox://daemon-inbox" \
  --source-event-id "mail-daemon-event-1" \
  --source-cursor "9101" \
  --message-id "<mail-daemon-event-1@example.com>" \
  --thread-ref "mail-daemon-thread-1" \
  --in-reply-to "<mail-root@example.com>" \
  --references-header "<mail-root@example.com>" \
  --from "sender@example.com" \
  --to "recipient@example.com" \
  --subject "Daemon mail watcher fixture" \
  --body "daemon watcher mail token" \
  --occurred-at "2026-02-24T11:00:00Z" | jq .

pa_tcp connector calendar handoff \
  --workspace "$WORKSPACE" \
  --source-scope "calendar://daemon-primary" \
  --source-event-id "calendar-daemon-event-1" \
  --source-cursor "9102" \
  --calendar-id "calendar-daemon-primary" \
  --calendar-name "Primary" \
  --event-uid "calendar-daemon-event-uid-1" \
  --change-type "updated" \
  --title "Daemon calendar watcher fixture" \
  --notes "daemon watcher calendar token" \
  --location "Room 9" \
  --starts-at "2026-02-24T11:30:00Z" \
  --ends-at "2026-02-24T12:00:00Z" \
  --occurred-at "2026-02-24T11:05:00Z" | jq .

pa_tcp connector browser handoff \
  --workspace "$WORKSPACE" \
  --source-scope "safari://window/daemon-1" \
  --source-event-id "browser-daemon-event-1" \
  --source-cursor "9103" \
  --window-id "window-daemon-1" \
  --tab-id "tab-daemon-1" \
  --page-url "https://example.com" \
  --page-title "Example Domain" \
  --event-type "navigation" \
  --payload "daemon watcher browser token" \
  --occurred-at "2026-02-24T11:10:00Z" | jq .
```

Poll task runs to observe watcher-driven automations (best effort):

```bash
for _ in $(seq 1 80); do
  AUTO_WATCH_TASKS_JSON="$(pa_tcp task runs --workspace "$WORKSPACE" --limit 200 2>/dev/null || true)"
  if echo "$AUTO_WATCH_TASKS_JSON" | jq -e ".items | map(select(.title == \"ON_COMM_EVENT ${AUTO_WATCH_MESSAGES_DIRECTIVE_ID}\")) | length > 0" >/dev/null \
    && echo "$AUTO_WATCH_TASKS_JSON" | jq -e ".items | map(select(.title == \"ON_COMM_EVENT ${AUTO_WATCH_MAIL_DIRECTIVE_ID}\")) | length > 0" >/dev/null \
    && echo "$AUTO_WATCH_TASKS_JSON" | jq -e ".items | map(select(.title == \"ON_COMM_EVENT ${AUTO_WATCH_CALENDAR_DIRECTIVE_ID}\")) | length > 0" >/dev/null \
    && echo "$AUTO_WATCH_TASKS_JSON" | jq -e ".items | map(select(.title == \"ON_COMM_EVENT ${AUTO_WATCH_BROWSER_DIRECTIVE_ID}\")) | length > 0" >/dev/null; then
    break
  fi
  sleep 0.25
done
echo "$AUTO_WATCH_TASKS_JSON" | jq .
```

Expected:

- tasks payload is returned and can include any/all titles: `ON_COMM_EVENT <messages/mail/calendar/browser directive id>`.
- no manual `channel messages ingest` / `connector <mail|calendar|browser> ingest` invocation is needed.
- no hand-crafted `mkdir`/`cat > .../pending/*.json` queue file setup is needed; bridge helper commands perform setup/handoff.
- watcher inbox queue counts are present as numeric values; `failed` remains `0` for each source, and processed payloads (when present) are archived under `$PA_INBOUND_WATCHER_INBOX_DIR/<source>/processed`.

### 3.4.5 Communications inbox query APIs (`threads`, `events`)

Query thread inventory with cursor pagination:

```bash
COMM_THREADS_PAGE1_JSON="$(curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"limit\":1}" \
  "http://$DAEMON_TCP_ADDR/v1/comm/threads/list")"
echo "$COMM_THREADS_PAGE1_JSON" | jq .
COMM_THREADS_HAS_MORE="$(echo "$COMM_THREADS_PAGE1_JSON" | jq -r '.has_more')"
COMM_THREAD_CURSOR="$(echo "$COMM_THREADS_PAGE1_JSON" | jq -r '.next_cursor')"
COMM_THREAD_ID="$(echo "$COMM_THREADS_PAGE1_JSON" | jq -r '.items[0].thread_id')"
test -n "$COMM_THREAD_ID" && [ "$COMM_THREAD_ID" != "null" ]
if [[ "$COMM_THREADS_HAS_MORE" == "true" ]]; then
  test -n "$COMM_THREAD_CURSOR" && [ "$COMM_THREAD_CURSOR" != "null" ]
fi

if [[ "$COMM_THREADS_HAS_MORE" == "true" && -n "$COMM_THREAD_CURSOR" && "$COMM_THREAD_CURSOR" != "null" ]]; then
  curl -sS -X POST \
    -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"workspace_id\":\"$WORKSPACE\",\"limit\":1,\"cursor\":\"$COMM_THREAD_CURSOR\"}" \
    "http://$DAEMON_TCP_ADDR/v1/comm/threads/list" | jq .
fi

curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"channel\":\"mail\",\"connector_id\":\"mail\",\"query\":\"watcher\",\"limit\":20}" \
  "http://$DAEMON_TCP_ADDR/v1/comm/threads/list" | jq .
```

Validate thread-aware reply send and connector-targeted dispatch hints:

```bash
COMM_MAIL_THREAD_ID="$(curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"channel\":\"mail\",\"connector_id\":\"mail\",\"query\":\"watcher\",\"limit\":20}" \
  "http://$DAEMON_TCP_ADDR/v1/comm/threads/list" | jq -r '.items[0].thread_id')"
COMM_REPLY_CONNECTOR_ID="mail"
if [[ -z "$COMM_MAIL_THREAD_ID" || "$COMM_MAIL_THREAD_ID" == "null" ]]; then
  COMM_MAIL_THREAD_ID="$COMM_THREAD_ID"
  COMM_REPLY_CONNECTOR_ID="$(echo "$COMM_THREADS_PAGE1_JSON" | jq -r '.items[0].connector_id')"
fi

curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"thread_id\":\"$COMM_MAIL_THREAD_ID\",\"connector_id\":\"$COMM_REPLY_CONNECTOR_ID\",\"message\":\"thread-aware reply test\"}" \
  "http://$DAEMON_TCP_ADDR/v1/comm/send" | jq .

curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"source_channel\":\"message\",\"connector_id\":\"twilio\",\"destination\":\"+15550004444\",\"message\":\"connector-hint route test\"}" \
  "http://$DAEMON_TCP_ADDR/v1/comm/send" | jq .
```

Query event timeline with cursor pagination:

```bash
COMM_EVENTS_PAGE1_JSON="$(curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"limit\":1}" \
  "http://$DAEMON_TCP_ADDR/v1/comm/events/list")"
echo "$COMM_EVENTS_PAGE1_JSON" | jq .
COMM_EVENTS_HAS_MORE="$(echo "$COMM_EVENTS_PAGE1_JSON" | jq -r '.has_more')"
COMM_EVENT_CURSOR="$(echo "$COMM_EVENTS_PAGE1_JSON" | jq -r '.next_cursor')"
if [[ "$COMM_EVENTS_HAS_MORE" == "true" ]]; then
  test -n "$COMM_EVENT_CURSOR" && [ "$COMM_EVENT_CURSOR" != "null" ]
fi

if [[ "$COMM_EVENTS_HAS_MORE" == "true" && -n "$COMM_EVENT_CURSOR" && "$COMM_EVENT_CURSOR" != "null" ]]; then
  curl -sS -X POST \
    -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"workspace_id\":\"$WORKSPACE\",\"limit\":1,\"cursor\":\"$COMM_EVENT_CURSOR\"}" \
    "http://$DAEMON_TCP_ADDR/v1/comm/events/list" | jq .
fi

curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"thread_id\":\"$COMM_THREAD_ID\",\"limit\":20}" \
  "http://$DAEMON_TCP_ADDR/v1/comm/events/list" | jq .
```

Expected:

- thread query returns deterministic envelope `{workspace_id, items[], has_more, next_cursor}`.
- first thread page returns one row with participant/event metadata and connector attribution (`connector_id`).
- if `has_more=true`, `next_cursor` is non-empty and page-2 via cursor advances to a different thread row.
- channel/query/connector filtered thread query returns only matching `channel` + `connector_id` rows.
- `/v1/comm/send` accepts optional `thread_id` and `connector_id` hints:
  thread-aware call without explicit `destination` resolves `resolved_destination` and `resolved_source_channel` from thread context.
  connector-targeted call with `connector_id=twilio` resolves `resolved_source_channel=twilio` for message sends.
- event query returns deterministic envelope `{workspace_id, thread_id?, items[], has_more, next_cursor}`.
- first event page returns one row with populated `addresses[]` and connector attribution (`connector_id`).
- if `has_more=true`, `next_cursor` is non-empty and page-2 via cursor advances to a different event row.
- thread-filtered event query returns only events for the requested `thread_id` and includes `connector_id` per item.

### 3.4.5.1 Workspace-scoped iMessage visibility stability across section refreshes

Capture iMessage baseline snapshots in the primary workspace:

```bash
IMESSAGE_BASE_THREADS_JSON="$(curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"channel\":\"message\",\"connector_id\":\"imessage\",\"limit\":50}" \
  "http://$DAEMON_TCP_ADDR/v1/comm/threads/list")"
echo "$IMESSAGE_BASE_THREADS_JSON" | jq .

IMESSAGE_BASE_EVENTS_JSON="$(curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"channel\":\"message\",\"connector_id\":\"imessage\",\"limit\":50}" \
  "http://$DAEMON_TCP_ADDR/v1/comm/events/list")"
echo "$IMESSAGE_BASE_EVENTS_JSON" | jq .

IMESSAGE_THREAD_COUNT="$(echo "$IMESSAGE_BASE_THREADS_JSON" | jq -r '.items | length')"
IMESSAGE_THREAD_FIRST_ID="$(echo "$IMESSAGE_BASE_THREADS_JSON" | jq -r '.items[0].thread_id')"
IMESSAGE_EVENT_COUNT="$(echo "$IMESSAGE_BASE_EVENTS_JSON" | jq -r '.items | length')"
IMESSAGE_EVENT_FIRST_ID="$(echo "$IMESSAGE_BASE_EVENTS_JSON" | jq -r '.items[0].event_id')"
test "$IMESSAGE_THREAD_COUNT" -ge 1
test "$IMESSAGE_EVENT_COUNT" -ge 1
```

Simulate section navigation by refreshing channel status in primary and alternate workspaces:

```bash
ALT_WORKSPACE="${WORKSPACE}-alt"
curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$ALT_WORKSPACE\",\"channel_id\":\"app\",\"configuration\":{\"enabled\":true,\"transport\":\"daemon_realtime\"},\"merge\":true}" \
  "http://$DAEMON_TCP_ADDR/v1/channels/config/upsert" | jq .

for _ in $(seq 1 3); do
  curl -sS -X POST \
    -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"workspace_id\":\"$WORKSPACE\"}" \
    "http://$DAEMON_TCP_ADDR/v1/channels/status" >/dev/null
  curl -sS -X POST \
    -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"workspace_id\":\"$ALT_WORKSPACE\"}" \
    "http://$DAEMON_TCP_ADDR/v1/channels/status" >/dev/null
done
```

Re-read iMessage snapshots and verify they match baseline visibility:

```bash
IMESSAGE_AFTER_THREADS_JSON="$(curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"channel\":\"message\",\"connector_id\":\"imessage\",\"limit\":50}" \
  "http://$DAEMON_TCP_ADDR/v1/comm/threads/list")"
echo "$IMESSAGE_AFTER_THREADS_JSON" | jq .

IMESSAGE_AFTER_EVENTS_JSON="$(curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"channel\":\"message\",\"connector_id\":\"imessage\",\"limit\":50}" \
  "http://$DAEMON_TCP_ADDR/v1/comm/events/list")"
echo "$IMESSAGE_AFTER_EVENTS_JSON" | jq .

test "$(echo "$IMESSAGE_AFTER_THREADS_JSON" | jq -r '.items | length')" -eq "$IMESSAGE_THREAD_COUNT"
test "$(echo "$IMESSAGE_AFTER_THREADS_JSON" | jq -r '.items[0].thread_id')" = "$IMESSAGE_THREAD_FIRST_ID"
test "$(echo "$IMESSAGE_AFTER_EVENTS_JSON" | jq -r '.items | length')" -eq "$IMESSAGE_EVENT_COUNT"
test "$(echo "$IMESSAGE_AFTER_EVENTS_JSON" | jq -r '.items[0].event_id')" = "$IMESSAGE_EVENT_FIRST_ID"
```

Expected:

- iMessage thread/event queries in the primary workspace (`channel=message`, `connector_id=imessage`) always return non-empty snapshots.
- repeated `channels/status` refreshes (including alternate workspace refreshes) do not change primary workspace iMessage visibility.
- baseline and post-refresh snapshots preserve count + top-most IDs, confirming no intermittent hide/show drift.

### 3.4.6 Identity device/session inventory + revoke APIs

Seed deterministic fixture rows (users/devices/sessions):

```bash
cat > "$TEST_RUNTIME_ROOT/identity-device-session-fixture.seed.go" <<'EOF'
package main

import (
  "database/sql"
  "os"

  _ "github.com/mattn/go-sqlite3"
)

func mustExec(db *sql.DB, query string, args ...any) {
  if _, err := db.Exec(query, args...); err != nil {
    panic(err)
  }
}

func main() {
  if len(os.Args) != 3 {
    panic("usage: identity-device-session-fixture.seed.go <db-path> <workspace-id>")
  }
  dbPath := os.Args[1]
  workspaceID := os.Args[2]

  db, err := sql.Open("sqlite3", dbPath)
  if err != nil {
    panic(err)
  }
  defer db.Close()

  mustExec(db, "INSERT INTO users(id, email, display_name, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT(id) DO NOTHING", "user.alpha", "alpha@example.com", "User Alpha", "ACTIVE", "2026-02-25T00:00:00Z", "2026-02-25T00:00:00Z")
  mustExec(db, "INSERT INTO users(id, email, display_name, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT(id) DO NOTHING", "user.beta", "beta@example.com", "User Beta", "ACTIVE", "2026-02-25T00:00:00Z", "2026-02-25T00:00:00Z")
  mustExec(db, "INSERT INTO user_devices(id, workspace_id, user_id, device_type, platform, label, last_seen_at, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(id) DO NOTHING", "device-alpha", workspaceID, "user.alpha", "phone", "ios", "Alpha iPhone", "2026-02-25T00:10:00Z", "2026-02-25T00:00:00Z")
  mustExec(db, "INSERT INTO user_devices(id, workspace_id, user_id, device_type, platform, label, last_seen_at, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(id) DO NOTHING", "device-beta", workspaceID, "user.beta", "desktop", "macos", "Beta Mac", "2026-02-25T00:20:00Z", "2026-02-25T00:05:00Z")
  mustExec(db, "INSERT INTO device_sessions(id, workspace_id, device_id, session_token_hash, started_at, expires_at, revoked_at) VALUES (?, ?, ?, ?, ?, ?, ?) ON CONFLICT(id) DO NOTHING", "session-alpha-active", workspaceID, "device-alpha", "hash-active", "2026-02-25T00:01:00Z", "2099-01-01T00:00:00Z", nil)
  mustExec(db, "INSERT INTO device_sessions(id, workspace_id, device_id, session_token_hash, started_at, expires_at, revoked_at) VALUES (?, ?, ?, ?, ?, ?, ?) ON CONFLICT(id) DO NOTHING", "session-alpha-expired", workspaceID, "device-alpha", "hash-expired", "2026-02-24T00:01:00Z", "2026-02-24T01:00:00Z", nil)
  mustExec(db, "INSERT INTO device_sessions(id, workspace_id, device_id, session_token_hash, started_at, expires_at, revoked_at) VALUES (?, ?, ?, ?, ?, ?, ?) ON CONFLICT(id) DO NOTHING", "session-alpha-revoked", workspaceID, "device-alpha", "hash-revoked", "2026-02-23T00:01:00Z", "2099-01-01T00:00:00Z", "2026-02-23T02:00:00Z")
  mustExec(db, "INSERT INTO device_sessions(id, workspace_id, device_id, session_token_hash, started_at, expires_at, revoked_at) VALUES (?, ?, ?, ?, ?, ?, ?) ON CONFLICT(id) DO NOTHING", "session-beta-active", workspaceID, "device-beta", "hash-beta-active", "2026-02-25T00:06:00Z", "2099-01-01T00:00:00Z", nil)
}
EOF
go -C source/services/daemon-go run "$TEST_RUNTIME_ROOT/identity-device-session-fixture.seed.go" "$DAEMON_DB_PATH" "$WORKSPACE"
rm -f "$TEST_RUNTIME_ROOT/identity-device-session-fixture.seed.go"
```

Query device/session inventory and revoke:

```bash
IDENTITY_DEVICES_PAGE1_JSON="$(curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"limit\":1}" \
  "http://$DAEMON_TCP_ADDR/v1/identity/devices/list")"
echo "$IDENTITY_DEVICES_PAGE1_JSON" | jq .
IDENTITY_DEVICE_CURSOR_CREATED_AT="$(echo "$IDENTITY_DEVICES_PAGE1_JSON" | jq -r '.next_cursor_created_at')"
IDENTITY_DEVICE_CURSOR_ID="$(echo "$IDENTITY_DEVICES_PAGE1_JSON" | jq -r '.next_cursor_id')"
test -n "$IDENTITY_DEVICE_CURSOR_CREATED_AT" && [ "$IDENTITY_DEVICE_CURSOR_CREATED_AT" != "null" ]
test -n "$IDENTITY_DEVICE_CURSOR_ID" && [ "$IDENTITY_DEVICE_CURSOR_ID" != "null" ]

curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"cursor_created_at\":\"$IDENTITY_DEVICE_CURSOR_CREATED_AT\",\"cursor_id\":\"$IDENTITY_DEVICE_CURSOR_ID\",\"limit\":2}" \
  "http://$DAEMON_TCP_ADDR/v1/identity/devices/list" | jq .

curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"session_health\":\"revoked\",\"limit\":20}" \
  "http://$DAEMON_TCP_ADDR/v1/identity/sessions/list" | jq .

IDENTITY_SESSION_REVOKE_JSON="$(curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"session_id\":\"session-alpha-active\"}" \
  "http://$DAEMON_TCP_ADDR/v1/identity/sessions/revoke")"
echo "$IDENTITY_SESSION_REVOKE_JSON" | jq .

curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"session_id\":\"session-alpha-active\"}" \
  "http://$DAEMON_TCP_ADDR/v1/identity/sessions/revoke" | jq .
```

Expected:

- device inventory envelope is deterministic: `{workspace_id, items[], has_more, next_cursor_created_at, next_cursor_id}`.
- device rows include `last_seen_at` and session-health aggregates (`session_total`, `session_active_count`, `session_expired_count`, `session_revoked_count`).
- session inventory envelope is deterministic: `{workspace_id, items[], has_more, next_cursor_started_at, next_cursor_id}` with `session_health` (`active|expired|revoked`) and `device_last_seen_at`.
- revoke endpoint is idempotent: first call returns `idempotent=false` and sets `revoked_at`; replay returns `idempotent=true` with unchanged `revoked_at`.

### 3.4.7 Delegation capability-grant APIs

```bash
CAPABILITY_GRANT_UPSERT_JSON="$(curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"actor_id\":\"actor.requester\",\"capability_key\":\"messages.send\",\"scope_json\":\"{\\\"channels\\\":[\\\"message\\\"],\\\"allow\\\":true}\",\"status\":\"active\"}" \
  "http://$DAEMON_TCP_ADDR/v1/delegation/capability-grants/upsert")"
echo "$CAPABILITY_GRANT_UPSERT_JSON" | jq .
CAPABILITY_GRANT_ID="$(echo "$CAPABILITY_GRANT_UPSERT_JSON" | jq -r '.grant_id')"
test -n "$CAPABILITY_GRANT_ID" && [ "$CAPABILITY_GRANT_ID" != "null" ]

curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"actor_id\":\"actor.approver\",\"capability_key\":\"messages.send\",\"scope_json\":\"{\\\"channels\\\":[\\\"imessage\\\"],\\\"allow\\\":true}\",\"status\":\"active\"}" \
  "http://$DAEMON_TCP_ADDR/v1/delegation/capability-grants/upsert" | jq .

curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"grant_id\":\"$CAPABILITY_GRANT_ID\",\"status\":\"revoked\",\"expires_at\":\"2030-01-01T00:00:00Z\"}" \
  "http://$DAEMON_TCP_ADDR/v1/delegation/capability-grants/upsert" | jq .

CAPABILITY_GRANT_LIST_PAGE1_JSON="$(curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"capability_key\":\"messages.send\",\"limit\":1}" \
  "http://$DAEMON_TCP_ADDR/v1/delegation/capability-grants/list")"
echo "$CAPABILITY_GRANT_LIST_PAGE1_JSON" | jq .
CAPABILITY_GRANT_CURSOR_CREATED_AT="$(echo "$CAPABILITY_GRANT_LIST_PAGE1_JSON" | jq -r '.next_cursor_created_at')"
CAPABILITY_GRANT_CURSOR_ID="$(echo "$CAPABILITY_GRANT_LIST_PAGE1_JSON" | jq -r '.next_cursor_id')"
test -n "$CAPABILITY_GRANT_CURSOR_CREATED_AT" && [ "$CAPABILITY_GRANT_CURSOR_CREATED_AT" != "null" ]
test -n "$CAPABILITY_GRANT_CURSOR_ID" && [ "$CAPABILITY_GRANT_CURSOR_ID" != "null" ]

curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"capability_key\":\"messages.send\",\"cursor_created_at\":\"$CAPABILITY_GRANT_CURSOR_CREATED_AT\",\"cursor_id\":\"$CAPABILITY_GRANT_CURSOR_ID\",\"limit\":2}" \
  "http://$DAEMON_TCP_ADDR/v1/delegation/capability-grants/list" | jq .

curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"actor_id\":\"actor.requester\",\"status\":\"revoked\",\"limit\":20}" \
  "http://$DAEMON_TCP_ADDR/v1/delegation/capability-grants/list" | jq .
```

Expected:

- `capability-grants/upsert` supports both create and update semantics (update by `grant_id`).
- records preserve deterministic fields (`grant_id`, `workspace_id`, `actor_id`, `capability_key`, `scope_json`, `status`, `created_at`, `expires_at`).
- status normalization is deterministic (`ACTIVE`/`REVOKED`) and `expires_at` accepts RFC3339 values.
- list API supports cursor pagination via (`next_cursor_created_at`, `next_cursor_id`) and filtered queries by `actor_id`, `capability_key`, and `status`.

### 3.5 Agent happy-path runs (Mail/Calendar/Browser + resolved Messages channel)

```bash
MAIL_RUN_JSON="$(pa_tcp agent run --workspace "$WORKSPACE" --request 'send an email to recipient@example.com saying "daemon update"')"
echo "$MAIL_RUN_JSON" | jq .

CALENDAR_RUN_JSON="$(pa_tcp agent run --workspace "$WORKSPACE" --request 'schedule "daemon calendar fixture"')"
echo "$CALENDAR_RUN_JSON" | jq .

CALENDAR_EVENT_ID="$(echo "$CALENDAR_RUN_JSON" | jq -r '.step_states[0].evidence.event_id')"
test -n "$CALENDAR_EVENT_ID" && [ "$CALENDAR_EVENT_ID" != "null" ]

CALENDAR_UPDATE_RUN_JSON="$(pa_tcp agent run --workspace "$WORKSPACE" --request "reschedule calendar event id $CALENDAR_EVENT_ID to \"daemon calendar fixture updated\"")"
echo "$CALENDAR_UPDATE_RUN_JSON" | jq .

CALENDAR_CANCEL_RUN_JSON="$(pa_tcp agent run --workspace "$WORKSPACE" --approval-phrase 'GO AHEAD' --request "cancel calendar event id $CALENDAR_EVENT_ID")"
echo "$CALENDAR_CANCEL_RUN_JSON" | jq .

BROWSER_RUN_JSON="$(pa_tcp agent run --workspace "$WORKSPACE" --request 'open https://example.com and summarize')"
echo "$BROWSER_RUN_JSON" | jq .

MESSAGES_SMS_RUN_JSON="$(pa_tcp agent run --workspace "$WORKSPACE" --request 'send a message via twilio to +15550001111 saying "hello"' 2>&1 || true)"
printf "%s\n" "$MESSAGES_SMS_RUN_JSON"
```

Expected:

- mail run: `workflow=mail`, `run_state=completed`, at least one `mail_*` step capability is present, and provider evidence includes a mail automation provider marker.
- calendar create/update/cancel runs: create emits stable `event_id`; update and cancel complete only when the same `event_id` is supplied explicitly, with provider evidence `apple-calendar-dry-run` (or `apple-calendar` when not in dry-run mode).
- browser run: `workflow=browser`, `run_state=completed`, step capabilities include `browser_open`, `browser_extract`, `browser_close`; extract evidence includes deep-content fields (`content_chars`, `content_preview`) and query-grounded output (`query_answer`), with provider evidence `safari-automation-dry-run` (or `safari-automation` when not in dry-run mode).
- messages run: explicit Twilio message request may execute (`clarification_required=false`, `run_state=completed`, step capability `messages_send_sms`, destination `+15550001111`) or return deterministic clarification (`clarification_required=true`, `missing_slots` contains `message_channel`) or actionable text error output (`request failed` + `what failed`) when intent resolution does not map to an executable messages action in that environment.
- Finder destructive connector flow is covered in approval + voice-gate sections below (`delete file` requests).

### 3.5.0 Delivery policy update + attempt-history context APIs

Use the messages run output from section 3.5 and communications IDs from section 3.4.5:

```bash
if echo "$MESSAGES_SMS_RUN_JSON" | jq . >/dev/null 2>&1; then
  MESSAGES_SMS_TASK_ID="$(echo "$MESSAGES_SMS_RUN_JSON" | jq -r '.task_id // empty')"
  MESSAGES_SMS_RUN_ID="$(echo "$MESSAGES_SMS_RUN_JSON" | jq -r '.run_id // empty')"
  MESSAGES_SMS_STEP_ID="$(echo "$MESSAGES_SMS_RUN_JSON" | jq -r '.step_states[0].step_id // empty')"
else
  MESSAGES_SMS_TASK_ID=""
  MESSAGES_SMS_RUN_ID=""
  MESSAGES_SMS_STEP_ID=""
fi
CONTEXT_STEP_ID="$MESSAGES_SMS_STEP_ID"
if [[ -z "$CONTEXT_STEP_ID" || "$CONTEXT_STEP_ID" == "null" ]]; then
  CONTEXT_STEP_ID="manual-step-context"
fi
CONTEXT_EVENT_ID="$COMM_EVENT_ID"
if [[ -z "$CONTEXT_EVENT_ID" || "$CONTEXT_EVENT_ID" == "null" ]]; then
  CONTEXT_EVENT_ID="manual-event-context"
fi
test -n "$CONTEXT_STEP_ID" && [ "$CONTEXT_STEP_ID" != "null" ]
test -n "$COMM_THREAD_ID" && [ "$COMM_THREAD_ID" != "null" ]
test -n "$COMM_EVENT_ID" && [ "$COMM_EVENT_ID" != "null" ]

COMM_POLICY_CREATE_JSON="$(curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"source_channel\":\"message\",\"endpoint_pattern\":\"+1555%\",\"primary_channel\":\"imessage\",\"retry_count\":1,\"fallback_channels\":[\"sms\"],\"is_default\":true}" \
  "http://$DAEMON_TCP_ADDR/v1/comm/policy/set")"
echo "$COMM_POLICY_CREATE_JSON" | jq .
COMM_POLICY_ID="$(echo "$COMM_POLICY_CREATE_JSON" | jq -r '.id')"
test -n "$COMM_POLICY_ID" && [ "$COMM_POLICY_ID" != "null" ]

COMM_POLICY_UPDATE_JSON="$(curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"policy_id\":\"$COMM_POLICY_ID\",\"workspace_id\":\"$WORKSPACE\",\"source_channel\":\"message\",\"endpoint_pattern\":\"+1555%\",\"primary_channel\":\"twilio\",\"retry_count\":0,\"fallback_channels\":[],\"is_default\":true}" \
  "http://$DAEMON_TCP_ADDR/v1/comm/policy/set")"
echo "$COMM_POLICY_UPDATE_JSON" | jq .

curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"source_channel\":\"message\"}" \
  "http://$DAEMON_TCP_ADDR/v1/comm/policy/list" | jq .

COMM_CONTEXT_SEND_JSON="$(curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"operation_id\":\"op-daemon-context-history\",\"source_channel\":\"message\",\"destination\":\"+15550001234\",\"message\":\"daemon context history\",\"step_id\":\"$CONTEXT_STEP_ID\",\"event_id\":\"$CONTEXT_EVENT_ID\",\"imessage_failures\":2}" \
  "http://$DAEMON_TCP_ADDR/v1/comm/send")"
echo "$COMM_CONTEXT_SEND_JSON" | jq .

COMM_CONTEXT_HISTORY_PAGE1_JSON="$(curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"operation_id\":\"op-daemon-context-history\",\"limit\":1}" \
  "http://$DAEMON_TCP_ADDR/v1/comm/attempts")"
echo "$COMM_CONTEXT_HISTORY_PAGE1_JSON" | jq .
COMM_CONTEXT_HISTORY_CURSOR="$(echo "$COMM_CONTEXT_HISTORY_PAGE1_JSON" | jq -r '.next_cursor')"
test -n "$COMM_CONTEXT_HISTORY_CURSOR" && [ "$COMM_CONTEXT_HISTORY_CURSOR" != "null" ]

curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"operation_id\":\"op-daemon-context-history\",\"cursor\":\"$COMM_CONTEXT_HISTORY_CURSOR\",\"limit\":2}" \
  "http://$DAEMON_TCP_ADDR/v1/comm/attempts" | jq .
```

Expected:

- `comm/policy/set` supports both create and update semantics; update path reuses `policy_id`.
- policy list returns the updated row with `policy.primary_channel=twilio`, `policy.retry_count=0`, and empty fallback list.
- context-linked send returns deterministic attempt records (success or failure is explicit; no silent drop).
- attempt-history query supports operation-scoped filtering plus cursor pagination.
- attempt rows include route metadata fields (`route_phase`, `retry_ordinal`, `fallback_from_channel`) and preserve `operation_id`.

### 3.5.1 Clarification loop for missing slots

Trigger an under-specified Finder request:

```bash
FINDER_CLARIFY_JSON="$(pa_tcp agent run --workspace "$WORKSPACE" --request 'delete file now')"
echo "$FINDER_CLARIFY_JSON" | jq .
```

Expected:

- `clarification_required=true`
- `task_state=clarification_required` and `run_state=clarification_required`
- `missing_slots` includes `finder_query`
- no `task_id`/`run_id` is created

Trigger an under-specified Messages request:

```bash
MESSAGES_CLARIFY_JSON="$(pa_tcp agent run --workspace "$WORKSPACE" --request 'send a text to +15550001111: "hello"')"
echo "$MESSAGES_CLARIFY_JSON" | jq .
```

Expected:

- `workflow=messages`
- `clarification_required=true`
- `missing_slots` includes `message_channel`
- `native_action.messages.recipient` and `native_action.messages.body` are populated

### 3.6 Approval inbox query/list API

Trigger a destructive-request approval so the inbox has a pending row:

```bash
APPROVAL_RUN_JSON="$(pa_tcp agent run \
  --workspace "$WORKSPACE" \
  --requested-by actor.requester \
  --subject actor.requester \
  --acting-as actor.requester \
  --request 'delete file /tmp/personal-agent-daemon-approval.txt')"
echo "$APPROVAL_RUN_JSON" | jq .
APPROVAL_ID="$(echo "$APPROVAL_RUN_JSON" | jq -r '.approval_request_id')"
test -n "$APPROVAL_ID" && [ "$APPROVAL_ID" != "null" ]
```

Query pending approvals:

```bash
curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"state\":\"pending\",\"limit\":20}" \
  "http://$DAEMON_TCP_ADDR/v1/approvals/list" | jq .
```

Verify unauthorized approver is denied:

```bash
pa_tcp agent approve --workspace "$WORKSPACE" --approval-id "$APPROVAL_ID" --actor-id actor.approver --phrase "GO AHEAD"
```

Expected: command fails with `approval denied`.

Grant approval-scope delegation, approve, and verify final-state rows:

```bash
APPROVAL_GRANT_JSON="$(pa_tcp delegation grant --workspace "$WORKSPACE" --from actor.requester --to actor.approver --scope-type APPROVAL)"
echo "$APPROVAL_GRANT_JSON" | jq .
APPROVAL_RULE_ID="$(echo "$APPROVAL_GRANT_JSON" | jq -r '.id')"
test -n "$APPROVAL_RULE_ID" && [ "$APPROVAL_RULE_ID" != "null" ]
pa_tcp agent approve --workspace "$WORKSPACE" --approval-id "$APPROVAL_ID" --actor-id actor.approver --phrase "GO AHEAD"
pa_tcp delegation revoke --workspace "$WORKSPACE" --rule-id "$APPROVAL_RULE_ID"
pa_tcp delegation check --workspace "$WORKSPACE" --requested-by actor.requester --acting-as actor.approver --scope-type APPROVAL | jq .

curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"state\":\"final\",\"include_final\":true,\"limit\":20}" \
  "http://$DAEMON_TCP_ADDR/v1/approvals/list" | jq .
```

Expected:

- pending query returns the new approval row with `state=pending`.
- row includes risk metadata (`risk_level`, `risk_rationale`) and principal context (`requested_by_actor_id`, `subject_principal_actor_id`, `acting_as_actor_id`).
- row includes route metadata envelope (`route.task_class`, `route.task_class_source`, `route.route_source`) and provider/model fields when route resolution is available.
- post-revoke delegation check returns `allowed=false` with `reason_code=missing_delegation_rule`.
- final query includes the same approval row with `state=final` and `decision=approved`.

### 3.6.1 Voice destructive-action handoff gate

Run a voice-origin destructive request without in-app confirmation:

```bash
VOICE_DESTRUCTIVE_JSON="$(pa_tcp agent run \
  --workspace "$WORKSPACE" \
  --request 'delete file /tmp/personal-agent-daemon-voice-handoff.txt' \
  --origin voice \
  --approval-phrase 'GO AHEAD')"
echo "$VOICE_DESTRUCTIVE_JSON" | jq .
```

Expected:

- `approval_required=true`
- `run_state=awaiting_approval`
- at least one step summary contains `in-app approval handoff`

Approve the pending voice request and verify resume:

```bash
VOICE_APPROVAL_ID="$(echo "$VOICE_DESTRUCTIVE_JSON" | jq -r '.approval_request_id')"
test -n "$VOICE_APPROVAL_ID" && [ "$VOICE_APPROVAL_ID" != "null" ]
pa_tcp agent approve --workspace "$WORKSPACE" --approval-id "$VOICE_APPROVAL_ID" --actor-id actor.requester --phrase "GO AHEAD"
```

Run a voice-origin destructive request with in-app confirmation:

```bash
VOICE_CONFIRMED_JSON="$(pa_tcp agent run \
  --workspace "$WORKSPACE" \
  --request 'delete file /tmp/personal-agent-daemon-voice-handoff-confirmed.txt' \
  --origin voice \
  --in-app-approval-confirmed=true)"
echo "$VOICE_CONFIRMED_JSON" | jq .
```

Expected:

- `approval_required=false`
- `run_state=completed`

### 3.7 Inspect logs query/stream APIs

Query newest-first inspect logs:

```bash
INSPECT_QUERY="$(curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$INSPECT_WORKSPACE\",\"limit\":10}" \
  "http://$DAEMON_TCP_ADDR/v1/inspect/logs/query")"
echo "$INSPECT_QUERY" | jq .
```

Poll inspect stream from the most recent cursor:

```bash
CURSOR_CREATED_AT="$(echo "$INSPECT_QUERY" | jq -r '.logs[0].created_at // empty')"
CURSOR_ID="$(echo "$INSPECT_QUERY" | jq -r '.logs[0].log_id // empty')"

curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$INSPECT_WORKSPACE\",\"cursor_created_at\":\"$CURSOR_CREATED_AT\",\"cursor_id\":\"$CURSOR_ID\",\"timeout_ms\":500,\"poll_interval_ms\":100,\"limit\":10}" \
  "http://$DAEMON_TCP_ADDR/v1/inspect/logs/stream" | jq .
```

Expected:

- query response returns logs in LIFO order (`created_at` newest-first).
- each log entry contains `status`, `input_summary`, and `output_summary` fields for UI inspect rows.
- each log entry includes deterministic route metadata fields (`route.task_class`, `route.task_class_source`, `route.route_source`) and provider/model when resolvable.
- stream response returns `timed_out` boolean and log entries matching the same structured schema.

### 3.8 Task submit/status/cancel/retry/requeue queued lifecycle over daemon control API

```bash
QUEUE_STREAM_LOG="/tmp/pa-daemon-queue-stream.jsonl"
rm -f "$QUEUE_STREAM_LOG"
pa_tcp stream --duration 5s >"$QUEUE_STREAM_LOG" &
QUEUE_STREAM_PID=$!

TASK_CANCEL_JSON="$(pa_tcp task submit --workspace "$WORKSPACE" --requested-by actor.requester --subject actor.requester --title 'Daemon transport cancel task' --description 'send an email update' --task-class chat)"
echo "$TASK_CANCEL_JSON"
TASK_CANCEL_ID="$(echo "$TASK_CANCEL_JSON" | jq -r '.task_id')"
TASK_CANCEL_RUN_ID="$(echo "$TASK_CANCEL_JSON" | jq -r '.run_id')"
test -n "$TASK_CANCEL_ID" && [ "$TASK_CANCEL_ID" != "null" ]
test -n "$TASK_CANCEL_RUN_ID" && [ "$TASK_CANCEL_RUN_ID" != "null" ]
pa_tcp task cancel --run-id "$TASK_CANCEL_RUN_ID" --reason "manual daemon cancellation"
pa_tcp task status --task-id "$TASK_CANCEL_ID" | jq -e '.state == "cancelled"'

TASK_RETRY_JSON="$(pa_tcp task retry --run-id "$TASK_CANCEL_RUN_ID" --reason "manual daemon retry")"
echo "$TASK_RETRY_JSON"
TASK_RETRY_RUN_ID="$(echo "$TASK_RETRY_JSON" | jq -r '.run_id')"
test -n "$TASK_RETRY_RUN_ID" && [ "$TASK_RETRY_RUN_ID" != "null" ] && [ "$TASK_RETRY_RUN_ID" != "$TASK_CANCEL_RUN_ID" ]
pa_tcp task status --task-id "$TASK_CANCEL_ID" | jq -e ".state == \"queued\" and .run_id == \"$TASK_RETRY_RUN_ID\" and .actions.can_cancel == true and .actions.can_retry == false and .actions.can_requeue == true"

TASK_REQUEUE_JSON="$(pa_tcp task requeue --run-id "$TASK_RETRY_RUN_ID" --reason "manual daemon requeue")"
echo "$TASK_REQUEUE_JSON"
TASK_REQUEUE_RUN_ID="$(echo "$TASK_REQUEUE_JSON" | jq -r '.run_id')"
test -n "$TASK_REQUEUE_RUN_ID" && [ "$TASK_REQUEUE_RUN_ID" != "null" ] && [ "$TASK_REQUEUE_RUN_ID" != "$TASK_RETRY_RUN_ID" ]
pa_tcp task status --task-id "$TASK_CANCEL_ID" | jq -e ".state == \"queued\" and .run_id == \"$TASK_REQUEUE_RUN_ID\" and .actions.can_cancel == true and .actions.can_retry == false and .actions.can_requeue == true"

TASK_JSON="$(pa_tcp task submit --workspace "$WORKSPACE" --requested-by actor.requester --subject actor.requester --title 'Daemon transport queued task' --description 'send an email update' --task-class chat)"
echo "$TASK_JSON"
TASK_ID="$(echo "$TASK_JSON" | jq -r '.task_id')"
test -n "$TASK_ID" && [ "$TASK_ID" != "null" ]
TASK_STATUS_JSON="$(pa_tcp task status --task-id "$TASK_ID")"
echo "$TASK_STATUS_JSON" | jq .
echo "$TASK_STATUS_JSON" | jq -e ".task_id == \"$TASK_ID\" and (.actions.can_cancel|type)==\"boolean\" and (.actions.can_retry|type)==\"boolean\" and (.actions.can_requeue|type)==\"boolean\""

TASK_LIST_JSON="$(curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"limit\":100}" \
  "http://$DAEMON_TCP_ADDR/v1/tasks/list")"
echo "$TASK_LIST_JSON" | jq -e "
  ([.items[] | select(.task_id == \"$TASK_ID\")][0]) as \$row
  | \$row != null
  | (\$row.actions.can_cancel|type) == \"boolean\"
  | (\$row.actions.can_retry|type) == \"boolean\"
  | (\$row.actions.can_requeue|type) == \"boolean\"
  | if (\$row.run_state | ascii_downcase) == \"failed\"
    then ((\$row.last_error // \"\") | length) > 0
    else true
    end
"

wait "$QUEUE_STREAM_PID"
jq -c 'select(.event_type=="task_run_lifecycle") | {event_type, correlation_id, lifecycle_state: .payload.lifecycle_state, task_id: .payload.task_id, run_id: .payload.run_id}' "$QUEUE_STREAM_LOG"
```

Expected:

- submit returns `task_id` and `run_id`.
- `task cancel --run-id <id>` transitions the selected run to `cancelled` with persisted task/run cancellation state.
- `task retry --run-id <cancelled|failed-run>` creates a fresh queued run (`run_id` changes) for the same task.
- `task requeue --run-id <queued|planning|awaiting_approval|blocked-run>` creates a fresh queued run (`run_id` changes) and supersedes the previous run.
- `task status` and `tasks/list` include deterministic action availability metadata (`actions.can_cancel`, `actions.can_retry`, `actions.can_requeue`).
- when final run state is `failed`, `tasks/list` returns a non-empty `last_error` for actionable remediation context.
- realtime stream contains `task_run_lifecycle` events for submitted runs with correlation IDs and lifecycle progression including cancellation (`cancelled`), retry (`queued` with `control_backend_retry`), and requeue (`cancelled` + `queued` with `control_backend_requeue`); terminal auto-drain progression may depend on local executor configuration.

### 3.9 Canonical agent approve route over daemon control API

```bash
APPROVAL_DECIDE_RUN_JSON="$(pa_tcp agent run \
  --workspace "$WORKSPACE" \
  --requested-by actor.requester \
  --subject actor.requester \
  --acting-as actor.requester \
  --request 'delete file /tmp/personal-agent-daemon-approval-decide.txt')"
echo "$APPROVAL_DECIDE_RUN_JSON" | jq .
APPROVAL_DECIDE_ID="$(echo "$APPROVAL_DECIDE_RUN_JSON" | jq -r '.approval_request_id')"
test -n "$APPROVAL_DECIDE_ID" && [ "$APPROVAL_DECIDE_ID" != "null" ]

pa_tcp agent approve --workspace "$WORKSPACE" --approval-id "$APPROVAL_DECIDE_ID" --phrase "GO AHEAD" --actor-id actor.requester
```

Expected:

- `agent approve` returns `task_state=completed` and `run_state=completed`.
- approval state transition is persisted by daemon workflow runtime (not in-memory route state).

### 3.10 Task/run list query API

```bash
curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"limit\":20}" \
  "http://$DAEMON_TCP_ADDR/v1/tasks/list" | jq .

curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"state\":\"completed\",\"limit\":20}" \
  "http://$DAEMON_TCP_ADDR/v1/tasks/list" | jq .
```

Expected:

- list response includes `items[]` rows with `task_id`, `run_id`, `task_state`, `run_state`, `priority`, principal context, and timestamp/error metadata.
- each item includes route metadata envelope (`route.task_class`, `route.task_class_source`, `route.route_source`) and provider/model when route resolution is available.
- `state` filter returns only rows matching the effective run/task state.

### 3.10.1 Chat turn task/run correlation metadata contract

```bash
go -C source/services/daemon-go test ./internal/transport -run 'TestTransportChatTurnRouteIncludesTaskRunCorrelationMetadata|TestTransportChatTurnExplainRoute' -count=1
```

Expected:

- test passes and verifies `chat.turn` responses include `task_run_correlation` metadata fields.
- `chat.turn` response envelope includes explicit contract metadata fields (`contract_version`, `turn_item_schema_version`, `realtime_event_contract_version`) for deterministic parser gating.
- correlation envelope includes deterministic availability/source semantics plus task/run identifiers when correlation exists.
- `/v1/chat/turn/explain` returns selected route context plus turn-context `tool_catalog[]` schemas and per-tool `policy_decisions[]` for the provided actor/channel scope.

### 3.10.2 Inspect run route metadata envelope

```bash
curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"run_id\":\"$TASK_RUN_ID\"}" \
  "http://$DAEMON_TCP_ADDR/v1/inspect/run" | jq .
```

Expected:

- response includes task/run/steps/artifacts/audit envelopes for the requested run.
- top-level `route` metadata is present with deterministic fields (`task_class`, `task_class_source`, `route_source`) and provider/model when resolvable.

### 3.11 Realtime stream stability

```bash
pa_tcp stream --duration 3s

SIGNAL_TASK_JSON="$(pa_tcp task submit --workspace "$WORKSPACE" --requested-by actor.requester --subject actor.requester --title 'Realtime cancel signal task' --description 'send an email update' --task-class chat)"
SIGNAL_RUN_ID="$(echo "$SIGNAL_TASK_JSON" | jq -r '.run_id')"
test -n "$SIGNAL_RUN_ID" && [ "$SIGNAL_RUN_ID" != "null" ]

SIGNAL_STREAM_LOG="/tmp/pa-daemon-signal-stream.jsonl"
rm -f "$SIGNAL_STREAM_LOG"
pa_tcp stream --duration 2s --signal-type cancel --run-id "$SIGNAL_RUN_ID" --reason "manual signal cancel" >"$SIGNAL_STREAM_LOG"
cat "$SIGNAL_STREAM_LOG"
jq -e 'select(.event_type=="client_signal_ack" and .payload.signal_type=="cancel" and .payload.accepted==true)' "$SIGNAL_STREAM_LOG" >/dev/null
```

Expected:

- command exits successfully after duration.
- no panic/stack trace is printed.
- signal-driven cancel emits `client_signal_ack` with `accepted=true` and `signal_type=cancel`.

### 3.11.1 Realtime websocket guardrail regressions

Run focused realtime transport guardrail regressions:

```bash
go -C source/services/daemon-go test ./internal/transport \
  -run 'TestTransportRealtime(RejectsWhenConnectionCapExceeded|RejectsWhenSubscriptionCapExceeded|ClosesConnectionOnOversizedSignalPayload|ClosesStaleClientWithoutPong)' \
  -count=1
```

Expected:

- suite passes.
- websocket handshake rejects over-capacity sessions with deterministic typed `429` payloads.
- oversized realtime client signals are disconnected under configured read limits.
- stale clients that do not maintain ping/pong liveness are disconnected deterministically.

### 3.12 Daemon schedule automation loop (no manual run command)

```bash
AUTO_SCHEDULE_JSON="$(pa_tcp automation create \
  --workspace "$WORKSPACE" \
  --subject actor.requester \
  --trigger-type SCHEDULE \
  --title 'Daemon auto schedule' \
  --instruction 'daemon automation loop check' \
  --interval-seconds 1)"
echo "$AUTO_SCHEDULE_JSON" | jq .
AUTO_SCHEDULE_DIRECTIVE_ID="$(echo "$AUTO_SCHEDULE_JSON" | jq -r '.directive_id')"
test -n "$AUTO_SCHEDULE_DIRECTIVE_ID" && [ "$AUTO_SCHEDULE_DIRECTIVE_ID" != "null" ]

AUTO_SCHEDULE_DUP_JSON="$(pa_tcp automation create \
  --workspace "$WORKSPACE" \
  --subject actor.requester \
  --trigger-type SCHEDULE \
  --title 'Daemon auto schedule' \
  --instruction 'daemon automation loop check' \
  --interval-seconds 1)"
echo "$AUTO_SCHEDULE_DUP_JSON" | jq .
AUTO_SCHEDULE_DUP_DIRECTIVE_ID="$(echo "$AUTO_SCHEDULE_DUP_JSON" | jq -r '.directive_id')"
test -n "$AUTO_SCHEDULE_DUP_DIRECTIVE_ID" && [ "$AUTO_SCHEDULE_DUP_DIRECTIVE_ID" != "null" ]
test "$AUTO_SCHEDULE_DUP_DIRECTIVE_ID" != "$AUTO_SCHEDULE_DIRECTIVE_ID"

for _ in $(seq 1 25); do
  AUTO_SCHEDULE_TASKS_JSON="$(curl -sS -X POST \
    -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"workspace_id\":\"$WORKSPACE\",\"limit\":50}" \
    "http://$DAEMON_TCP_ADDR/v1/tasks/list")"
  if echo "$AUTO_SCHEDULE_TASKS_JSON" | jq -e ".items | map(select(.title == \"Scheduled directive ${AUTO_SCHEDULE_DIRECTIVE_ID}\")) | length > 0" >/dev/null; then
    break
  fi
  sleep 0.2
done
echo "$AUTO_SCHEDULE_TASKS_JSON" | jq .
echo "$AUTO_SCHEDULE_TASKS_JSON" | jq -e ".items | map(select(.title == \"Scheduled directive ${AUTO_SCHEDULE_DUP_DIRECTIVE_ID}\")) | length == 0" >/dev/null
```

Expected:

- scheduled task appears without running `automation run schedule`.
- generated task title follows `Scheduled directive <directive_id>`.
- duplicate-equivalent schedule trigger is suppressed by runtime dedupe and does not create a second parallel scheduled task stream.

### 3.12.1 Automation fire-history query API

```bash
curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"limit\":20}" \
  "http://$DAEMON_TCP_ADDR/v1/automation/fire-history" | jq .

curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"status\":\"created_task\",\"limit\":20}" \
  "http://$DAEMON_TCP_ADDR/v1/automation/fire-history" | jq .
```

Expected:

- response envelope is deterministic: `{workspace_id, fires[]}` (empty array when no records).
- each `fires[]` item includes `status`, `idempotency_key`, `idempotency_signal`, `fired_at`, and trigger identifiers.
- each `fires[]` item includes route metadata envelope (`route.task_class`, `route.task_class_source`, `route.route_source`) and provider/model when route resolution is available.
- when linked task/run records exist, `task_id` and `run_id` are populated.
- status filtering returns only the requested normalized status (`pending|created_task|failed`).

### 3.12.2 Typed ON_COMM_EVENT metadata and validation APIs

```bash
curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\"}" \
  "http://$DAEMON_TCP_ADDR/v1/automation/comm-trigger/metadata" | jq .

curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"subject_actor_id\":\"actor.approver\",\"filter\":{\"channels\":[\" Twilio_SMS \",\"twilio_sms\"],\"principal_actor_ids\":[\"actor.requester\"],\"sender_allowlist\":[],\"thread_ids\":[],\"keywords\":{\"contains_any\":[\" Hello \",\"hello\"],\"contains_all\":[],\"exact_phrases\":[]}}}" \
  "http://$DAEMON_TCP_ADDR/v1/automation/comm-trigger/validate" | jq .

curl -sS -o /tmp/pa-auto-comm-legacy.json -w "%{http_code}\n" -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"subject_actor_id\":\"actor.requester\",\"filter_json\":\"{\\\"channels\\\":[\\\"twilio_sms\\\"]}\"}" \
  "http://$DAEMON_TCP_ADDR/v1/automation/comm-trigger/validate"
cat /tmp/pa-auto-comm-legacy.json | jq .
```

Expected:

- metadata response exposes typed ON_COMM_EVENT contract fields: `trigger_type`, `required_defaults`, `idempotency_key_fields`, `filter_defaults`, `filter_schema[]`, and `compatibility`.
- `required_defaults` are deterministic and match runtime evaluator defaults: `event_type=MESSAGE`, `direction=INBOUND`, `assistant_emitted=false`.
- validation response accepts typed `filter` and returns deterministic normalization in both `normalized_filter` and `normalized_filter_json`.
- normalization is deterministic and case-folded/deduplicated for list terms (for example `Twilio_SMS` and `twilio_sms` normalize to one `twilio_sms` value).
- compatibility envelope reports subject/principal compatibility only (`compatible`, `subject_actor_id`, `subject_matches_principal_rule`).
- legacy `filter_json` payload input is rejected at transport decode with HTTP `400`.

## 4) Twilio Webhook Control-Plane Validation (Daemon-Backed)

Use daemon-backed CLI commands to run direct ingest + webhook serve/replay through daemon APIs.

```bash
pa_tcp connector twilio set \
  --workspace "$WORKSPACE" \
  --account-sid "ACDAEMONTEST" \
  --auth-token "daemon-test-token" \
  --sms-number "+15555550001" \
  --voice-number "+15555550002"
```

### 4.1 Direct unified `sms-chat` turn orchestration

```bash
pa_tcp connector twilio sms-chat \
  --workspace "$WORKSPACE" \
  --to "+15555550997" \
  --message "daemon direct sms-chat turn" \
  --operation-id "daemon-direct-turn-001"

pa_tcp connector twilio sms-chat \
  --workspace "$WORKSPACE" \
  --to "+15555550997" \
  --message "daemon direct sms-chat turn" \
  --operation-id "daemon-direct-turn-001"
```

Expected:

- first call returns one turn with `success=true`, `idempotent_replay=false`, and a non-empty `thread_id`.
- first call returns assistant metadata deterministically:
  - either `assistant_reply` (with `assistant_operation_id`) is populated when model routing is configured, or
  - `assistant_error` is populated when assistant generation/delivery is unavailable.
- first call must not return top-level turn `error` unless ingest is rejected.
- second call with the same `operation_id` returns one turn with `success=true` and `idempotent_replay=true` (no duplicate assistant delivery attempt).

### 4.2 Direct ingest APIs (`ingest-sms`, `ingest-voice`)

```bash
pa_tcp connector twilio ingest-sms \
  --workspace "$WORKSPACE" \
  --skip-signature=true \
  --from "+15555550999" \
  --to "+15555550001" \
  --body "daemon ingest test" \
  --message-sid "SMDAEMONINGEST1" \
  --account-sid "ACDAEMONTEST"

pa_tcp connector twilio ingest-voice \
  --workspace "$WORKSPACE" \
  --skip-signature=true \
  --provider-event-id "voice-daemon-ingest-1" \
  --call-sid "CADAEMONINGEST1" \
  --account-sid "ACDAEMONTEST" \
  --from "+15555550002" \
  --to "+15555550999" \
  --direction outbound-api \
  --call-status in-progress \
  --transcript "daemon voice ingest transcript" \
  --transcript-direction INBOUND

pa_tcp connector twilio ingest-voice \
  --workspace "$WORKSPACE" \
  --skip-signature=true \
  --provider-event-id "voice-daemon-ingest-2" \
  --call-sid "CADAEMONINGEST2" \
  --account-sid "ACDAEMONTEST" \
  --from "+15555550002" \
  --to "+15555550998" \
  --direction outbound-api \
  --call-status completed \
  --transcript "daemon voice ingest transcript completed" \
  --transcript-direction OUTBOUND
```

Expected:

- both commands succeed with `accepted=true`.
- repeating either command with the same idempotency key (`message-sid` or `provider-event-id`) returns `replayed=true`.

Query communications call-session summaries with cursor pagination:

```bash
COMM_CALLS_PAGE1_JSON="$(curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"limit\":1}" \
  "http://$DAEMON_TCP_ADDR/v1/comm/call-sessions/list")"
echo "$COMM_CALLS_PAGE1_JSON" | jq .
COMM_CALL_CURSOR="$(echo "$COMM_CALLS_PAGE1_JSON" | jq -r '.next_cursor')"
COMM_CALL_SESSION_ID="$(echo "$COMM_CALLS_PAGE1_JSON" | jq -r '.items[0].session_id')"
test -n "$COMM_CALL_CURSOR" && [ "$COMM_CALL_CURSOR" != "null" ]
test -n "$COMM_CALL_SESSION_ID" && [ "$COMM_CALL_SESSION_ID" != "null" ]

curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"limit\":1,\"cursor\":\"$COMM_CALL_CURSOR\"}" \
  "http://$DAEMON_TCP_ADDR/v1/comm/call-sessions/list" | jq .

curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"connector_id\":\"twilio\",\"status\":\"completed\",\"limit\":20}" \
  "http://$DAEMON_TCP_ADDR/v1/comm/call-sessions/list" | jq .
```

Expected:

- call-session query returns deterministic envelope `{workspace_id, items[], has_more, next_cursor}`.
- first call-session page returns one row with connector attribution (`connector_id`) and non-empty `next_cursor`.
- second page via `cursor` advances to a different `session_id`.
- `connector_id=twilio` + `status=completed` filters return only Twilio completed call-session rows.

### 4.2.1 Comm trust-receipt query APIs (`webhook-receipts`, `ingest-receipts`)

```bash
COMM_WEBHOOK_RECEIPTS_PAGE1_JSON="$(curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"provider\":\"twilio\",\"limit\":1}" \
  "http://$DAEMON_TCP_ADDR/v1/comm/webhook-receipts/list")"
echo "$COMM_WEBHOOK_RECEIPTS_PAGE1_JSON" | jq .
COMM_WEBHOOK_CURSOR_CREATED_AT="$(echo "$COMM_WEBHOOK_RECEIPTS_PAGE1_JSON" | jq -r '.next_cursor_created_at')"
COMM_WEBHOOK_CURSOR_ID="$(echo "$COMM_WEBHOOK_RECEIPTS_PAGE1_JSON" | jq -r '.next_cursor_id')"
test -n "$COMM_WEBHOOK_CURSOR_CREATED_AT" && [ "$COMM_WEBHOOK_CURSOR_CREATED_AT" != "null" ]
test -n "$COMM_WEBHOOK_CURSOR_ID" && [ "$COMM_WEBHOOK_CURSOR_ID" != "null" ]

curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"provider\":\"twilio\",\"cursor_created_at\":\"$COMM_WEBHOOK_CURSOR_CREATED_AT\",\"cursor_id\":\"$COMM_WEBHOOK_CURSOR_ID\",\"limit\":2}" \
  "http://$DAEMON_TCP_ADDR/v1/comm/webhook-receipts/list" | jq .

curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"provider\":\"twilio\",\"provider_event_query\":\"daemon\",\"limit\":20}" \
  "http://$DAEMON_TCP_ADDR/v1/comm/webhook-receipts/list" | jq .

COMM_INGEST_RECEIPTS_PAGE1_JSON="$(curl -sS -X POST \
  -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"workspace_id\":\"$WORKSPACE\",\"limit\":1}" \
  "http://$DAEMON_TCP_ADDR/v1/comm/ingest-receipts/list")"
echo "$COMM_INGEST_RECEIPTS_PAGE1_JSON" | jq .
COMM_INGEST_HAS_MORE="$(echo "$COMM_INGEST_RECEIPTS_PAGE1_JSON" | jq -r '.has_more')"
COMM_INGEST_CURSOR_CREATED_AT="$(echo "$COMM_INGEST_RECEIPTS_PAGE1_JSON" | jq -r '.next_cursor_created_at')"
COMM_INGEST_CURSOR_ID="$(echo "$COMM_INGEST_RECEIPTS_PAGE1_JSON" | jq -r '.next_cursor_id')"
COMM_INGEST_SOURCE="$(echo "$COMM_INGEST_RECEIPTS_PAGE1_JSON" | jq -r '.items[0].source // empty')"
COMM_INGEST_SOURCE_SCOPE="$(echo "$COMM_INGEST_RECEIPTS_PAGE1_JSON" | jq -r '.items[0].source_scope // empty')"
if [[ "$COMM_INGEST_HAS_MORE" == "true" ]]; then
  test -n "$COMM_INGEST_CURSOR_CREATED_AT" && [ "$COMM_INGEST_CURSOR_CREATED_AT" != "null" ]
  test -n "$COMM_INGEST_CURSOR_ID" && [ "$COMM_INGEST_CURSOR_ID" != "null" ]
fi

if [[ "$COMM_INGEST_HAS_MORE" == "true" && -n "$COMM_INGEST_CURSOR_CREATED_AT" && "$COMM_INGEST_CURSOR_CREATED_AT" != "null" && -n "$COMM_INGEST_CURSOR_ID" && "$COMM_INGEST_CURSOR_ID" != "null" ]]; then
  curl -sS -X POST \
    -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"workspace_id\":\"$WORKSPACE\",\"cursor_created_at\":\"$COMM_INGEST_CURSOR_CREATED_AT\",\"cursor_id\":\"$COMM_INGEST_CURSOR_ID\",\"limit\":2}" \
    "http://$DAEMON_TCP_ADDR/v1/comm/ingest-receipts/list" | jq .
fi

if [[ -n "$COMM_INGEST_SOURCE" && "$COMM_INGEST_SOURCE" != "null" ]]; then
  curl -sS -X POST \
    -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"workspace_id\":\"$WORKSPACE\",\"source\":\"$COMM_INGEST_SOURCE\",\"source_scope\":\"$COMM_INGEST_SOURCE_SCOPE\",\"limit\":20}" \
    "http://$DAEMON_TCP_ADDR/v1/comm/ingest-receipts/list" | jq .
fi
```

Expected:

- webhook receipt query returns deterministic envelope `{workspace_id, provider?, items[], has_more, next_cursor_created_at, next_cursor_id}`.
- webhook receipt rows include trust metadata (`trust_state`, `signature_valid`, `signature_value_present`), linked event/thread IDs (when available), and `audit_links[]` entries.
- ingest receipt query returns deterministic envelope `{workspace_id, source?, source_scope?, items[], has_more, next_cursor_created_at, next_cursor_id}`.
- ingest receipt rows include source trust metadata (`source`, `source_scope`, `source_event_id`, `trust_state`) plus typed `audit_links[]` metadata (array may be empty).
- both APIs support cursor pagination and filter combinations (`provider*`, `source*`, `event_id`, `trust_state`).

### 4.3 `ON_COMM_EVENT` automation auto-evaluation via inbound Twilio SMS

```bash
AUTO_COMM_JSON="$(pa_tcp automation create \
  --workspace "$WORKSPACE" \
  --subject actor.requester \
  --trigger-type ON_COMM_EVENT \
  --title 'Daemon auto comm event' \
  --instruction 'auto evaluate inbound comm events' \
  --filter '{"channels":["twilio_sms"],"keywords":{"contains_any":["daemon auto comm event trigger"]}}')"
echo "$AUTO_COMM_JSON" | jq .
AUTO_COMM_DIRECTIVE_ID="$(echo "$AUTO_COMM_JSON" | jq -r '.directive_id')"
test -n "$AUTO_COMM_DIRECTIVE_ID" && [ "$AUTO_COMM_DIRECTIVE_ID" != "null" ]

AUTO_COMM_DUP_JSON="$(pa_tcp automation create \
  --workspace "$WORKSPACE" \
  --subject actor.requester \
  --trigger-type ON_COMM_EVENT \
  --title 'Daemon auto comm event' \
  --instruction 'auto evaluate inbound comm events' \
  --filter '{"channels":["twilio_sms"],"keywords":{"contains_any":["daemon auto comm event trigger"]}}')"
echo "$AUTO_COMM_DUP_JSON" | jq .
AUTO_COMM_DUP_DIRECTIVE_ID="$(echo "$AUTO_COMM_DUP_JSON" | jq -r '.directive_id')"
test -n "$AUTO_COMM_DUP_DIRECTIVE_ID" && [ "$AUTO_COMM_DUP_DIRECTIVE_ID" != "null" ]
test "$AUTO_COMM_DUP_DIRECTIVE_ID" != "$AUTO_COMM_DIRECTIVE_ID"

pa_tcp connector twilio ingest-sms \
  --workspace "$WORKSPACE" \
  --skip-signature=true \
  --from "+15555550998" \
  --to "+15555550001" \
  --body "daemon auto comm event trigger" \
  --message-sid "SMDAEMONAUTOCOMM1" \
  --account-sid "ACDAEMONTEST"

for _ in $(seq 1 25); do
  AUTO_COMM_TASKS_JSON="$(curl -sS -X POST \
    -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"workspace_id\":\"$WORKSPACE\",\"limit\":50}" \
    "http://$DAEMON_TCP_ADDR/v1/tasks/list")"
  if echo "$AUTO_COMM_TASKS_JSON" | jq -e ".items | map(select(.title == \"ON_COMM_EVENT ${AUTO_COMM_DIRECTIVE_ID}\")) | length > 0" >/dev/null; then
    break
  fi
  sleep 0.2
done
echo "$AUTO_COMM_TASKS_JSON" | jq .
echo "$AUTO_COMM_TASKS_JSON" | jq -e ".items | map(select(.title == \"ON_COMM_EVENT ${AUTO_COMM_DUP_DIRECTIVE_ID}\" or .title == \"Scheduled directive ${AUTO_COMM_DUP_DIRECTIVE_ID}\")) | length == 0" >/dev/null
```

Expected:

- inbound Twilio SMS ingestion is accepted.
- automation-created task appears without running `automation run comm-event`.
- generated task title follows `ON_COMM_EVENT <directive_id>`.
- duplicate-equivalent `ON_COMM_EVENT` trigger is suppressed by runtime dedupe and does not create a second parallel task stream for the same inbound event.

### 4.4 Webhook serve + replay APIs

Create a fixture:

```bash
cat >/tmp/pa-daemon-webhook-sms.json <<'JSON'
{
  "kind": "sms",
  "params": {
    "From": "+15555550999",
    "To": "+15555550001",
    "Body": "daemon webhook replay",
    "MessageSid": "SMDAEMONWEBHOOK1",
    "AccountSid": "ACDAEMONTEST"
  }
}
JSON
```

Start serve (terminal B):

```bash
pa_tcp connector twilio webhook serve \
  --workspace "$WORKSPACE" \
  --listen "127.0.0.1:18088" \
  --signature-mode bypass \
  --cloudflared-mode auto \
  --run-for 8s
```

Replay from terminal C while serve is running:

```bash
pa_tcp connector twilio webhook replay \
  --workspace "$WORKSPACE" \
  --fixture /tmp/pa-daemon-webhook-sms.json \
  --base-url "http://127.0.0.1:18088" \
  --signature-mode bypass
```

Expected:

- `serve` command succeeds and returns webhook URLs.
- serve URLs use versioned Twilio connector webhook paths (`/<project-name>/v1/connector/twilio/{sms|voice}`), where `<project-name>` defaults from daemon process name (or `PA_PROJECT_NAME` when set), and include cloudflared warning details if tunnel bootstrap is unavailable.
- `replay` command succeeds with `status_code=200`.
- webhook ingestion output indicates accepted request (`accepted=true` in response body).

## 5) Unix Socket Mode Validation (macOS/Linux)

Stop TCP daemon, then run daemon in Unix mode (terminal A):

```bash
rm -f "$DAEMON_UNIX_SOCKET"
go -C source/services/daemon-go run ./cmd/personal-agent-daemon \
  --listen-mode unix \
  --listen-address "$DAEMON_UNIX_SOCKET" \
  --auth-token "$DAEMON_AUTH_TOKEN"
```

Run checks (terminal B):

```bash
pa_unix smoke

TASK_JSON="$(pa_unix task submit --workspace "$WORKSPACE" --requested-by actor.requester --subject actor.requester --title 'Unix queued transport task' --description 'send an email update' --task-class chat)"
echo "$TASK_JSON"
TASK_ID="$(echo "$TASK_JSON" | jq -r '.task_id')"
test -n "$TASK_ID" && [ "$TASK_ID" != "null" ]
TASK_STATUS_JSON="$(pa_unix task status --task-id "$TASK_ID" 2>/dev/null || true)"
echo "$TASK_STATUS_JSON" | jq .
echo "$TASK_STATUS_JSON" | jq -e ".task_id == \"$TASK_ID\" and (.actions.can_cancel|type)==\"boolean\" and (.actions.can_retry|type)==\"boolean\" and (.actions.can_requeue|type)==\"boolean\""
```

Expected:

- all commands succeed over Unix socket transport.
- queued task status contract remains deterministic with action metadata (terminal auto-drain may depend on local executor configuration).

## 6) Named-Pipe Mode Validation (Windows)

On Windows, run daemon and CLI in named-pipe mode.

Daemon (terminal A):

```bash
go -C source/services/daemon-go run ./cmd/personal-agent-daemon \
  --listen-mode named_pipe \
  --listen-address '\\.\pipe\personal-agent' \
  --auth-token "$DAEMON_AUTH_TOKEN"
```

CLI checks (terminal B):

```bash
go -C source/clients/cli-go run ./cmd/personal-agent \
  --mode named_pipe \
  --address '\\.\pipe\personal-agent' \
  --auth-token "$DAEMON_AUTH_TOKEN" \
  smoke
```

Expected on Windows:

- command succeeds and returns capability JSON.

Expected on non-Windows hosts:

- clear unsupported-mode error (for example, named-pipe mode is only supported on Windows).

## 7) Platform Service Install Script Validation (Dry Run)

These checks validate generated install artifacts without requiring privileged installation.

macOS daemon app packaging (stable TCC identity):

```bash
./tools/scripts/package_daemon_app_macos.sh \
  --output-app "$PWD/out/dist/Personal Agent Daemon.app" \
  --skip-sign
```

macOS app-host release packaging (unsigned/ad-hoc local/internal drag-install DMG):

```bash
./tools/scripts/package_macos_app_release.sh \
  --output-dir "$PWD/out/dist/macos-release"
```

macOS LaunchAgent script dry-run (packaged daemon app path):

```bash
./tools/scripts/install_daemon_service_macos.sh \
  --daemon-app "$PWD/out/dist/Personal Agent Daemon.app" \
  --auth-token-file "$DAEMON_AUTH_TOKEN_FILE" \
  --db-path "$DAEMON_DB_PATH" \
  --output "/tmp/com.personalagent.daemon.test.plist" \
  --dry-run
```

macOS LaunchAgent script dry-run (explicit daemon binary override):

```bash
./tools/scripts/install_daemon_service_macos.sh \
  --daemon-binary "$PWD/source/services/daemon-go/bin/personal-agent-daemon" \
  --auth-token-file "$DAEMON_AUTH_TOKEN_FILE" \
  --db-path "$DAEMON_DB_PATH" \
  --output "/tmp/com.personalagent.daemon.test.plist" \
  --dry-run
```

Linux systemd user-service script dry-run:

```bash
./tools/scripts/install_daemon_service_linux.sh \
  --daemon-binary "$PWD/source/services/daemon-go/bin/personal-agent-daemon" \
  --auth-token-file "$DAEMON_AUTH_TOKEN_FILE" \
  --db-path "$DAEMON_DB_PATH" \
  --output "/tmp/personal-agent-daemon.test.service" \
  --dry-run
```

Windows scheduled-task script dry-run:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\install_daemon_service_windows.ps1 `
  -DaemonBinary "$PWD\runtime\go\bin\personal-agent-daemon.exe" `
  -AuthTokenFile "$PWD\runtime\test-state\daemon\manual-test-daemon.control.token" `
  -DbPath "$PWD\runtime\test-state\daemon\manual-test-daemon.db" `
  -DryRun
```

Expected:

- macOS packaging command prints packaged app and executable paths under `.../Personal Agent Daemon.app/Contents/MacOS/personal-agent-daemon`.
- macOS app-host release packaging command prints packaged app path, embedded daemon helper path, DMG path, and generated checksum/manifest paths under `out/dist/macos-release`.
- macOS LaunchAgent dry-run using `--daemon-app` prints generated plist content with daemon flags and packaged executable path.
- macOS and Linux dry-runs print generated artifact content including daemon flags (`--listen-mode`, `--listen-address`, `--auth-token-file`, `--db`).
- Windows dry-run prints scheduled-task command configuration with daemon arguments.

For full macOS packaged-daemon/TCC guidance, use:

- `docs/ops/macos-daemon-packaging.md`

Unified local launch script dry-run help:

```bash
./tools/scripts/launch_personal_agent.sh --help
```

Expected:

- help text describes daemon check/start behavior, app launch behavior, and Ctrl+C cleanup semantics.
- help text includes `--daemon-start-mode auto|direct|launchctl`.
- help text documents `--daemon-auth-token` default resolution as stored app token from Keychain first (service `personalagent.ui.local_dev_token.v1` / account `daemon_auth_token`), then fallback token.
- help text includes `--stop-daemon-on-exit` and documents that launcher exits leave daemon running by default.
- on macOS, `auto` mode prefers launchctl startup; if launchctl startup cannot make daemon reachable, launcher falls back to direct spawn with explicit log message.

## 8) Clean-Machine Unsigned DMG Install + Gatekeeper/TCC Attribution (macOS Manual)

Use this matrix on a clean macOS profile (or clean machine) when validating unsigned/ad-hoc local/internal app-host distribution.

1. Build unsigned release artifacts:

```bash
./tools/scripts/package_macos_app_release.sh --output-dir "$PWD/out/dist/macos-release"
```

2. Move `out/dist/macos-release/PersonalAgent-unsigned.dmg` to `~/Downloads`, mount it in Finder, verify the styled drag-to-Applications background/instructions render, drag `PersonalAgent.app` into `/Applications`, then eject DMG.
3. Launch `/Applications/PersonalAgent.app`.
4. If blocked by Gatekeeper, override once using:
   - Control-click/Right-click `PersonalAgent.app` -> `Open`, or
   - `System Settings > Privacy & Security > Open Anyway` for `PersonalAgent`.
5. In app, complete setup token step and run `Configuration > Advanced > Install` (or `Repair`).
6. Verify launch agent identity resolves to packaged daemon helper path:

```bash
launchctl print "gui/$(id -u)/com.personalagent.daemon" | rg -n "program|Program|personal-agent-daemon|Personal Agent Daemon.app"
test -d "$HOME/Library/Application Support/personal-agent/daemon/Personal Agent Daemon.app"
```

7. Trigger connector permission prompts from app (`Connectors` panel) for a connector that prompts macOS (for example Calendar or automation-backed connectors).
8. Verify the prompt attribution target is `Personal Agent Daemon` and refresh connector status in app after responding.
9. If permission prompts were dismissed/denied and no longer appear, reset and retry:

```bash
tccutil reset AppleEvents com.personalagent.daemon || true
tccutil reset Calendar com.personalagent.daemon || true
```

Expected:

- unsigned DMG drag-install flow works from `/Applications`.
- mounted DMG shows styled drag-to-Applications guidance (background + deterministic icon placement) instead of a plain two-file folder presentation.
- setup surfaces provide explicit Gatekeeper override guidance before daemon troubleshooting.
- install/repair actions succeed from bundled helper assets and deterministic `/Applications` remediation appears when app placement is wrong.
- launch agent/executable identity resolves to `Personal Agent Daemon.app/Contents/MacOS/personal-agent-daemon`.
- connector permission prompts and post-prompt status attribution target `Personal Agent Daemon`.

## 9) Optional Regression Checks

Run these after any transport/runtime change:

```bash
cd source/services/daemon-go && go test ./internal/transport
cd source/services/daemon-go && go test ./internal/daemonruntime -run 'TestDaemonLifecycleStatusTreatsBusySingleWriterConnectionAsReady' -count=1
cd ../..
./tools/scripts/check_harness.sh
```

Expected: all checks pass.

## 10) Cleanup

- Stop daemon process in terminal A (`Ctrl+C`).
- Remove temporary Unix socket file if present:

```bash
rm -f "$DAEMON_UNIX_SOCKET"
```
