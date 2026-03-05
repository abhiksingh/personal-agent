# PersonalAgent Bootstrap Spec

Purpose: provide a compact, stable starting context before loading full specs.

For canonical detail, see:
- [Product Spec](./spec.md)
- [Data Model](./data-model.md)
- [UI Spec](./spec-ui.md)
- [Connector Authoring Kit](./connector-authoring-kit/README.md)

## 1) Product Snapshot

PersonalAgent is an autonomous assistant that executes end-to-end user workflows across channels and connectors with deterministic auditability.

MVP focus:
- Unified chat turn orchestration with typed turn items.
- Deterministic tool/approval lifecycle and traceability.
- Canonical logical channels (`app`, `message`, `voice`) mapped to connectors.
- Human-usable remediation flows across setup/runtime/approval failures.

## 2) Non-Negotiable Runtime Invariants

1. Idempotency and auditability are required for side-effecting operations.
2. Canonical logical channel IDs are `app|message|voice`; aliases normalize to canonical IDs.
3. Connector identities are canonicalized (for example `builtin.app`, `imessage`, `twilio`) and may map from legacy aliases.
4. Unified turn contracts are typed (`assistant_message`, `tool_call`, `tool_result`, `approval_request`, `approval_decision`, etc.).
5. Approval-gated actions must remain explicit, resumable, and trace-linked.
6. Routing/model decisions must be explainable and observable.
7. Delivery/retry/fallback behavior must be deterministic and inspectable.
8. Secret handling stays out of plain-text UI surfaces by default.
9. Context/memory retention and compaction are bounded by explicit policy.
10. Backward-incompatible cleanup is allowed when it improves canonical behavior.

## 3) Data Model Snapshot

Core object families:
- Identity: workspaces, principals, sessions/devices, acting-as context.
- Communications: threads/events/attempts/call sessions.
- Execution: tasks, task runs, steps, policy decisions, trace events.
- Automation: triggers, runs, fire history.
- Integrations: channels, connectors, permissions/config descriptors.
- Governance: capability grants, approvals, trust receipts.
- Context: memory inventory, candidates, retrieval docs/chunks.

Reference invariants and index priorities:
- [Data Model Invariants](./data-model.md#8-required-invariants)
- [Initial Index Priorities](./data-model.md#9-initial-index-priorities)

## 4) UI/Operator Contract Snapshot

Primary app surfaces:
- `Home`, `Chat`, `Communications`, `Automation`, `Approvals`, `Tasks`, `Inspect`, `Channels`, `Connectors`, `Models`, `Configuration`.

UI principles:
- Action-first defaults in normal views, technical detail behind explicit disclosure.
- Deterministic status/disabled reason/recovery actions.
- Reuse shared interaction patterns over bespoke one-off UI behavior.

## 5) Public Docs and Validation Snapshot

- Contributor entry points:
  - `README.md`
  - `CONTRIBUTING.md`
  - `SUPPORT.md`
  - `SECURITY.md`
- Manual test guides:
  - `docs/tests-cli.md`
  - `docs/tests-daemon.md`
  - `docs/tests-ui.md`
- Repository validation entry points:
  - `tools/scripts/check_harness.sh`
  - `tools/scripts/run_tests_all.sh`

## 6) What To Load Next (Selective)

Load minimally by task type:
- Runtime/core contract changes: `spec.md` + `data-model.md` + relevant sections only.
- New connector implementation/evolution: `connector-authoring-kit/README.md` + kit templates/checklist + targeted runtime contracts.
- UI behavior changes: `spec-ui.md` + `tests-ui.md` index + touched panel test file(s).
- Contribution workflow questions: `README.md` + `CONTRIBUTING.md` + relevant ops/test docs only.
- Schema/index/invariant work: `data-model.md` first, then targeted product sections.

If something is unclear, defer to full canonical docs listed at the top.

## 7) Context-Efficient Workflow Defaults

1. Diff-first scope:
   - `git diff --name-only`
   - `git diff --cached --name-only`
   - `git ls-files --others --exclude-standard`
2. Load only touched files plus direct canonical contracts first.
3. Keep matching manual guides and runner scripts aligned when behavior changes.
4. For long command output, write logs to file and inspect failures first:
   - `tools/scripts/parse_failure_lines.sh <log-file>`
