# Connector Implementation Checklist

Use this checklist to gate completion for connector work.

## 1) Contract Setup

- [ ] Connector ID is canonical and stable.
- [ ] Plugin ID and worker type are defined and consistent.
- [ ] Capability keys are unique, deterministic, and documented.
- [ ] Any channel-mapping implications (`app|message|voice`) are explicit.

## 2) Adapter and Worker Runtime

- [ ] Adapter implements metadata, health check, and execute-step behavior.
- [ ] Unsupported capabilities return deterministic failed status + reason.
- [ ] Worker runtime is wired for the new worker type.
- [ ] Worker handshake emits valid metadata (`id`, `kind`, `capabilities`, runtime address).
- [ ] Worker execute endpoint enforces daemon-issued bearer auth.

## 3) Manifest and Dispatch Wiring

- [ ] `plugin_workers_manifest.json` includes the plugin worker entry.
- [ ] Connector registry registration is added where required for agent execution.
- [ ] Supervisor dispatch can resolve and execute connector capabilities.

## 4) Status, Config, Diagnostics, and Test Operations

- [ ] Connector appears in `/v1/connectors/status`.
- [ ] Config descriptors are present for editable/runtime fields.
- [ ] Diagnostics actions are deterministic and actionable.
- [ ] `/v1/connectors/test` behavior is implemented and truthful.
- [ ] Permission-state/status-reason fields are set when applicable.

## 5) Ingest and Persistence (If Applicable)

- [ ] Ingest path enforces idempotency via receipts/event IDs.
- [ ] `connector_id` attribution is persisted for threads/events/sessions.
- [ ] Cursor/subscription state updates are deterministic.

## 6) Security and Reliability

- [ ] Outbound HTTP clients use explicit timeout-configured clients.
- [ ] Side effects are idempotent per step/operation identity.
- [ ] Retry semantics are bounded and deterministic.
- [ ] Secret values are never returned in status/evidence/log payloads.

## 7) Tests and Docs

- [ ] Unit tests added/updated for adapter input/output edge cases.
- [ ] Worker runtime tests added/updated for payload/auth/error paths.
- [ ] Status/diagnostics/test-operation tests updated.
- [ ] Manual test docs updated when user-testable behavior changed.
- [ ] Related runner scripts updated when manual test steps changed.
- [ ] Relevant harness checks were run and results recorded.
