# Reliability Guardrails (MVP)

Operational reliability rules derived from the canonical spec.

## Delivery Semantics

- Outbound messaging is at-least-once delivery.
- Duplicates are minimized through per-destination idempotency keys.
- Default policy is `iMessage -> retry once -> SMS`.
- Response routing defaults to the originating channel unless fallback policy is triggered.
- Delivery behavior must be consistent whether initiated from the Swift app or Personal Agent CLI.

## Trigger Semantics

- `ON_COMM_EVENT` task creation uses idempotency key `(workspace_id, trigger_id, source_event_id)`.
- Default trigger cooldown is disabled; rate limits are configurable per trigger.
- Trigger processing must be retry-safe.

## Transport and Interface Semantics

- Daemon IPC uses a platform-agnostic hybrid transport contract: HTTP/JSON control APIs plus WebSocket realtime streams.
- Control API schemas must be versioned and published via OpenAPI.
- WebSocket stream events must use stable typed envelopes and monotonic sequence IDs for deterministic client ordering.
- Default binding is localhost TCP in MVP so identical flows work on macOS, Windows, and Linux/Raspberry Pi targets.
- Optional platform bindings (Unix sockets, Windows named pipes) must preserve the same service contract and behavior.
- SSE server-stream fallback may be exposed for one-way streaming clients using the same event envelope schema.
- Transport requests must include correlation IDs for trace continuity across app/CLI/daemon.
- Transport retries and reconnects must be safe for idempotent operations.
- Channel and connector adapters must expose stable capability declarations for deterministic selection.

## Persistence Semantics

- SQLite writes flow through one writer queue in MVP.
- Queue behavior must support crash recovery and replay.
- Write handlers must be idempotent.

## Approval and Safety

- Destructive/non-reversible actions always require approval.
- Low-confidence risk classification defaults to approval-required.
- Voice-originated destructive requests require in-app approval handoff.

## Must-Pass Reliability Checks

1. Trigger replay does not create duplicate tasks for same idempotency key.
2. Delivery retry/replay does not produce duplicate sends for same idempotency key.
3. Writer queue restart replays pending writes without corruption.
4. Approval-gated steps never execute before valid approval.
5. CLI and app issue equivalent daemon operations with equivalent outcomes for same inputs.
6. Adapter registry resolution remains deterministic when multiple adapters advertise overlapping capabilities.
