# PersonalAgent Data Model (Canonical)

This document defines the persistence model aligned to the product contracts in `spec.md`.

## 1) Identity

- `Workspace`
- `User`
- `WorkspaceMember`
- `Actor`
- `WorkspacePrincipal`
- `ActorHandle`
- optional `UserActorLink`

## 2) Communications

- `CommThread` (logical channel + connector-attributed thread identity; connector-isolated by default)
- `CommEvent` (inherits logical channel and connector attribution for each event)
- `CommEventAddress`
- optional `CommAttachment`
- optional `EmailEventMeta`
- `CommProviderMessage` (provider envelope for SMS/voice metadata such as provider SIDs)
- `CommCallSession` (voice call lifecycle and linkage to comm threads/events with connector attribution)
- `CommWebhookReceipt` (webhook trust/idempotency record for provider callbacks)
- `CommIngestReceipt` (local/remote ingress trust + idempotency receipt keyed by source/source_event identity before side effects)
- `CommIngestCursor` (high-water mark cursor per ingress source/scope for poll-based local watchers)
- optional derived indexes: `ThreadParticipant`, `ThreadLocalIdentity`

## 3) Task and Execution

- `Task` (canonical; maps prior `WorkItem`)
- `TaskRun` (canonical; maps prior `WorkRun`)
- `TaskPlan` (canonical; maps prior `ExecutionPlan`)
- `TaskStep` (canonical; maps prior `ExecutionStep`)
- optional `TaskStepRecipient` for communication steps
- `DeliveryAttempt` (channel delivery attempt log with idempotency key and provider receipt)
- `RunArtifact`
- `ApprovalRequest`
- `AuditLogEntry` (append-only)
- `ChatTurnItem` (typed unified-turn ledger persisted in `chat_turn_items`)
- `ChatPersonaPolicy` (workspace/principal/channel style contract persisted in `chat_persona_policies`)

## 4) Automation

- `Directive`
- `AutomationTrigger`
- `TriggerFire` (trigger evaluation/execution record with idempotency key)
- `AutomationSourceSubscription` (subscription metadata for inbound local source feeds such as Mail rules, Calendar store notifications, iMessage watchers, Safari extension channels)

## 5) Devices, Capabilities, Connectors

- `UserDevice`
- `DeviceSession`
- `CapabilityGrant`
- `ChannelConnectorBinding` (workspace-scoped logical channel to connector mapping with enabled flag and priority order)
- `SecretRef`
- `RuntimePlugin` (registered channel/connector plugin metadata and declared capabilities)
- `RuntimePluginProcess` (daemon-supervised plugin process lifecycle/health state)

## 6) Policy and Delegation

- `DelegationRule` (explicit per-principal delegation scopes, including execution and approval scopes)
- `ChannelDeliveryPolicy` (retry/fallback routing per channel/endpoint)

## 7) Memory and Context

- `MemoryItem`
- `MemorySource`
- recommended `MemoryCandidate`
- optional retrieval index objects: `ContextDocument`, `ContextChunk`
- `ContextBudgetSample` (prompt/context budget telemetry by task class)
- `ContextBudgetTuningProfile` (per-task-class retrieval budget multipliers and utilization summaries)

## 8) Required Invariants

- Logical channel IDs are constrained to `app`, `message`, and `voice`.
- Connector IDs are stable machine IDs (for example `builtin.app`, `imessage`, `twilio`) and are immutable once introduced.
- Runtime capability keys used for model-tool exposure must canonicalize deterministically (normalize `.`/`-` to `_`) before policy/tool matching.
- `ChannelConnectorBinding` enforces uniqueness on `(workspace_id, channel_id, connector_id)` and ordering uniqueness on `(workspace_id, channel_id, priority)` for enabled bindings.
- `CommThread.connector_id` must reference a connector that is mapped to `CommThread.channel` via `ChannelConnectorBinding` in the same workspace.
- `CommEvent.connector_id` must match the parent thread connector attribution.
- `CommCallSession.connector_id` must be present for connector-backed voice call flows.
- Communication APIs must expose first-class `connector_id` fields; connector attribution must not require parsing provider-specific external refs.
- Message channel threads must remain connector-isolated by default; same-contact cross-connector events must not auto-merge into one thread without explicit merge policy.
- `TaskRun.acting_as_actor_id` must reference an actor in `WorkspacePrincipal`.
- `TaskStep.input_json` persists canonical typed step input payloads used for adapter execution and approval-resume replay.
- `TaskStep` core execution fields (for example recipient/title/url/path) must be sourced from `input_json` payloads, not parsed from display `name` text.
- Finder task-step contracts use canonical capability keys (`finder_find`, `finder_list`, `finder_preview`, `finder_delete`) and typed `input_json` fields (`path`, `query`, optional `root_path`) for deterministic target resolution.
- Finder destructive query execution must only proceed when query resolution is unambiguous; ambiguous delete queries must fail with guardrail-denied status.
- If `TaskRun.acting_as_actor_id != Task.requested_by_actor_id`, a valid `DelegationRule` must exist for that scope.
- `Directive.subject_principal_actor_id` must reference an actor in `WorkspacePrincipal`.
- Outbound comm events must have valid address-role structure (`FROM` + recipients).
- `CommWebhookReceipt` must enforce uniqueness for `(workspace_id, provider, provider_event_id)` to guarantee webhook replay idempotency.
- `CommIngestReceipt` must enforce uniqueness for `(workspace_id, source, source_event_id)` so local-source and non-webhook replay paths are idempotent.
- `CommIngestReceipt.trust_state` must be recorded before ingest side effects (`accepted|rejected`) and remain immutable after write.
- `CommIngestCursor` must enforce uniqueness for `(workspace_id, source, source_scope)` and monotonically advance after accepted ingest writes.
- For Twilio provider events, invalid signature receipts must not create side-effecting `CommEvent`/`TriggerFire` records unless explicit local development bypass mode is enabled.
- `CommCallSession.provider_call_id` (for example Twilio Call SID) must be unique per workspace when present.
- `CommCallSession.status` transitions are monotonic across provider callbacks (`initiated -> ringing -> in_progress -> terminal`).
- Outbound Twilio SMS/voice provider IDs (message SID/call SID) must be persisted in `DeliveryAttempt.provider_receipt` and/or `CommProviderMessage`.
- `ApprovalRequest` references exactly one of `{run_id, step_id}`.
- `ApprovalRequest.decision_by_actor_id` must be either the `acting_as` principal or a delegated approver.
- `TriggerFire` enforces uniqueness for `(workspace_id, trigger_id, source_event_id)`.
- `DeliveryAttempt` enforces uniqueness on delivery idempotency key per destination endpoint.
- `DeliveryAttempt` must persist both logical channel and connector attribution for audit/replay visibility.
- `AuditLogEntry` is immutable and correlation-linked.
- `SecretRef` must not store plaintext secret material; only secure-store identifiers/metadata may be persisted.
- API surfaces returning secret records must not include raw secret values.
- `RuntimePluginProcess.plugin_id` must reference a valid `RuntimePlugin` registration.
- `RuntimePluginProcess` lifecycle transitions (`STARTED`, handshake accepted, `HEALTH_TIMEOUT`, `RESTARTING`, `STOPPED`/`FAILED`) must be represented by append-only `AuditLogEntry` records.
- `ChatTurnItem` enforces uniqueness on `(turn_id, item_index)` to preserve deterministic replay ordering.
- `ChatTurnItem.channel_id` is constrained to canonical logical channels (`app`, `message`, `voice`).
- `ChatTurnItem.item_type` must be one of canonical typed turn items (`user_message`, `assistant_message`, `tool_call`, `tool_result`, `approval_request`, `approval_decision`).
- `ChatPersonaPolicy` enforces uniqueness on `(workspace_id, principal_actor_id, channel_id)` so persona resolution is deterministic by scope.

## 9) Initial Index Priorities

- `CommThread(workspace_id, channel, connector_id, updated_at)`
- `CommEvent(thread_id, occurred_at)`
- `CommEvent(thread_id, connector_id, occurred_at)`
- `CommEventAddress(event_id, address_role)`
- `CommWebhookReceipt(workspace_id, provider, provider_event_id)` unique index
- `CommIngestReceipt(workspace_id, source, source_event_id)` unique index
- `CommIngestCursor(workspace_id, source, source_scope)` unique index
- `CommProviderMessage(workspace_id, provider, provider_message_id)` unique index
- `CommCallSession(workspace_id, status, updated_at)`
- `Task(workspace_id, state, created_at)`
- `TaskRun(task_id, state, started_at)`
- `TaskStep(run_id, step_index, status)`
- `ChannelConnectorBinding(workspace_id, channel_id, enabled, priority)`
- `RuntimePlugin(workspace_id, kind, status)`
- `RuntimePluginProcess(workspace_id, state, updated_at)`
- `ChatTurnItem(turn_id, item_index)` unique index
- `ChatTurnItem(workspace_id, channel_id, thread_id, created_at, item_index)`
- `ChatTurnItem(workspace_id, correlation_id, created_at, item_index)`
- `ChatPersonaPolicy(workspace_id, principal_actor_id, channel_id, updated_at)`
- `MemoryItem(owner_principal_actor_id, scope_type, key, status)`
- `TriggerFire(workspace_id, trigger_id, source_event_id)` unique index
- `DeliveryAttempt(destination_endpoint, idempotency_key)` unique index
