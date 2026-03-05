# UI Tests: Configuration Panel

Source: [full guide](../tests-ui/full.md)
## 11) Configuration Panel Checks

1. Select `Configuration`.
2. Confirm a mode picker is visible with `Setup`, `Workspace`, `Integrations`, `Data`, and `Advanced`, and verify the default mode is `Setup`.
3. Confirm a `Setup Overview` section is visible with readiness badges (`Ready`, `Needs Attention`, optional `Checking`) and rows for `Daemon Lifecycle`, `Assistant Access Token`, `Provider Setup`, `Model Catalog`, and `Chat Route`.
3a. In `Chat Route`, verify resolved provider/model/source context appears when route resolution is healthy; when unresolved/blocked, verify deterministic remediation copy with `Open Models`.
3b. In `Setup` mode, verify vertical section order is `Setup Overview`, then `Assistant Access Token`, then collapsed `Runtime Details`.
4. In `Setup Overview`, verify quick actions are prioritized and compact:
   - `Fix Next` appears when onboarding has unresolved blockers.
   - one primary runtime remediation action is shown when needed.
   - `More Actions` appears for additional setup remediations.
   - `Refresh Checks` refreshes setup readiness/lifecycle/provider checks.
4a. In `Setup Overview`, verify `Setup Matrix Details` is collapsed by default and can be expanded on demand.
4b. If `Fix Next` or the primary remediation action is disabled, verify deterministic guidance copy explains why it is unavailable.
4c. Verify `First-Run Trust Guidance (Unsigned Build)` is visible and includes:
   - explicit first-launch override steps (`Control-click/Right-click Open`, `Open Anyway` in System Settings)
   - expected permission-prompt guidance for `Personal Agent Daemon`
   - deterministic actions: `Open Security Settings` and `Retry Setup Checks`
5. Click `Open Models` and verify navigation switches to `Models`.
6. Return to `Configuration` and click `Open Onboarding`; verify navigation switches to a non-configuration section where onboarding panel appears when setup is incomplete.
7. Return to `Configuration` in `Setup` mode and confirm runtime/setup detail is owned by `Setup Overview` + `Setup Matrix` rows (no standalone runtime banner above the mode content).
7a. In `Setup` mode, verify `Runtime Details` is collapsed by default and expands only when explicitly opened.
7b. Switch to a non-`Setup` mode (for example `Workspace`) and confirm runtime messaging remains summary-only and points users back to `Setup Matrix` for detailed readiness/remediation.
8. Switch mode to `Workspace` and confirm `Identity Hub` section is visible with:
   - `Workspace` picker
   - `Active Principal` picker
   - `Active Context` metadata (`workspace/principal source`, resolution, last updated when present)
   - `Workspace Directory` list
   - `Principal Directory` list with actor-handle mappings
   - `Refresh Identity Directory` action and deterministic status text
9. In `Identity Hub`, verify `Active Principal` picker and `Active Context` workspace/principal rows use display-name-first labels, with raw IDs only behind explicit reveal/copy controls.
10. Click `Refresh Identity Directory` and verify in-row loading appears only for identity refresh activity.
11. Change `Workspace` selection to a different workspace id (when available) and verify:
   - workspace context switch status copy is shown.
   - active context/workspace label updates to the selected workspace.
   - directory lists re-render for the selected workspace.
   - setup/runtime data refreshes without requiring app relaunch.
   - if testing an upgraded defaults profile that previously stored workspace `default`, verify active workspace resolves to canonical `ws1` and prior workspace-scoped UI context (filters/drafts) remains available.
12. In `Workspace` mode, confirm `Chat Persona Policy` section is visible with:
   - `Scope` picker (`Workspace Default`, `Principal`, `Channel`, `Principal + Channel`)
   - conditional principal/channel pickers based on selected scope
   - required `Style prompt` editor
   - default-collapsed `Advanced Guardrails` disclosure with one-guardrail-per-line editing
   - deterministic actions (`Refresh Scope`, `Save Policy`, `Reset Draft`, `Test in Chat`) and status text
13. In `Chat Persona Policy`, switch scope at least once and verify scope refresh status text updates for the selected principal/channel context.
14. Enter a style prompt and two guardrail lines, click `Save Policy`, and verify deterministic save status plus source/updated badges.
15. Modify style prompt without saving, click `Reset Draft`, and verify values revert to last loaded policy.
15a. In `Chat Persona Policy`, verify response-shaping badges are visible for `Test Channel` and `Profile`.
15b. Select a channel-scoped persona where channel is not app; verify helper copy explains `Test in Chat` validates app-channel shaping and points to Communications for channel-specific validation.
16. Click `Test in Chat` and verify navigation switches to `Chat` with persona scope status guidance.
17. Return to `Configuration` and confirm `Delegation Rules` section is visible with:
   - `Grant Delegation` form (`From Actor`, `To Actor`, `Scope Type`, optional `Scope Key`, optional `Expires At`)
   - daemon-backed delegation inventory list (from/to/scope/status/created/expires)
   - per-rule `Revoke` action
   - `Refresh Delegation Rules` action with deterministic status text
18. In `Grant Delegation`, create one rule with different `From Actor`/`To Actor`, `Scope Type = EXECUTION`, and optional `Scope Key`; verify deterministic success/failure status copy and list refresh.
19. In delegation rule rows, verify `From` and `To` actors are display-name-first with explicit reveal/copy access to raw IDs.
20. Revoke the newly created rule (or any existing rule) and verify revoke confirmation appears before dispatch; confirm and verify in-flight indicator plus deterministic post-action status copy.
20a. In `Workspace` mode, verify disclosure order after `Chat Persona Policy` is collapsed `Delegation Rules` followed by collapsed `Identity Devices and Sessions`.
14. Switch mode to `Data` and confirm retention control section is visible with:
   - `Trace Days`, `Transcript Days`, and `Memory Days` steppers
   - memory compaction controls (`token threshold`, `stale age`, `scan limit`, `apply` toggle)
   - `Run Retention Purge` and `Run/Preview Memory Compaction` actions
15. Trigger one retention action and verify confirmation appears before dispatch; confirm and verify status text updates with success/error summary copy.
16. Confirm context budget section is visible with:
   - `Task Class` picker
   - `Sample Limit` stepper
   - `Load Context Samples` and `Tune Context Profile` actions
17. Trigger one context action and verify status text updates with success/error summary copy.
18. Confirm `Memory Browser` disclosure is visible and collapsed by default, then expand it and verify:
   - memory inventory filter controls (`Owner Actor ID`, `Scope Type`, `Status`, `Source Type`, `Source Ref Query`, `Limit`)
   - `Refresh Inventory` and `Reset Filters` actions
   - memory inventory list region with deterministic loading/empty states
   - compaction candidate filter controls (`Owner Actor ID`, `Status`, `Limit`) plus refresh/reset actions
   - compaction candidate list region with deterministic loading/empty states
19. Click `Refresh Inventory` and verify status copy updates with deterministic filter summary text (`owner/scope/status/source`) and list rows render when data exists.
20. Apply at least one memory inventory filter (for example `Status = ACTIVE` or `Source Ref Query = memory://manual`), refresh, and verify rows narrow to matching records or deterministic no-match copy.
21. Click `Refresh Candidates` and verify compaction candidate status copy updates with deterministic owner/status summary plus candidate rows when available.
22. Confirm a `Retrieval Context Inspector` disclosure is visible and collapsed by default, then expand it and verify:
   - document filter controls (`Owner Actor ID`, `Source URI Query`, `Limit`) and refresh/reset actions
   - retrieval documents list with deterministic loading/empty states
   - selected-document indicator + chunk query controls (`Chunk Text Query`, `Limit`) and `Load Chunks` action
   - retrieval chunk list with deterministic loading/empty states
23. Click `Refresh Documents` and verify status copy updates with deterministic owner/source summary and document rows render when data exists.
24. Select one retrieval document via `Inspect Chunks` and verify selected state is shown, then verify chunk query status updates for the selected document id.
25. Enter a chunk text query, click `Load Chunks`, and verify chunk rows narrow accordingly (or deterministic no-match copy) while preserving selected-document context.
26. Switch mode to `Setup` and confirm assistant access token section includes manual `Save Token`/`Clear Stored Token` actions plus `Bootstrap from CLI` controls (`Copy Command`, `Run Bootstrap`).
27. Click `Copy Command` and verify clipboard text includes `auth bootstrap-local-dev` plus current `--workspace` and daemon `--address` values.
28. Click `Run Bootstrap` (with `personal-agent` CLI available) and verify status transitions from running to completion, then token readiness transitions through `Checking` to `Configured` without revealing token value.
29. (Optional contract check) run daemon without auth token and confirm token status shows missing-auth remediation guidance.
30. Switch mode to `Advanced` and confirm advanced daemon controls exist:
   - `Install`
   - `Uninstall`
   - `Repair`
   - `Start Daemon`
   - `Stop Daemon`
   - `Restart Daemon`
31. Trigger one setup action (`Install`, `Uninstall`, or `Repair`) when enabled and verify confirmation appears before dispatch; confirm and verify in-flight progress/status copy appears, then refresh lifecycle status to confirm terminal result copy.
31a. With app launched outside `/Applications` (for example from `Downloads`), trigger `Install` or `Repair` and verify deterministic remediation copy: `Move PersonalAgent.app to /Applications before running daemon install or repair.`
32. In a state where daemon control plane is reachable but one or more plugin workers fail, verify `Daemon Lifecycle` row uses degradation copy and includes `Open Channels` remediation; verify onboarding/setup summary does not show generic blocking `daemon setup needs repair` copy for this worker-only state.
32a. In `Advanced` mode, verify disclosure order is collapsed `Daemon Lifecycle Controls`, then `Runtime Supervisor Timeline`, then `Panel Latency Budgets`.
33. Confirm a `Runtime Supervisor Timeline` disclosure is visible and collapsed by default, then expand it and verify:
   - `Filters` group
   - `Plugin ID (optional)` text field
   - `Kind`, `State`, and `Event Type` picker controls
   - `Limit` stepper
   - `Refresh Timeline` and `Reset Filters` actions
34. Click `Refresh Timeline` and verify loading feedback appears (progress indicator or status copy) and status text updates with deterministic query summary language.
35. Apply at least one filter (for example `Kind = Channel` or set a specific `Plugin ID`), click `Refresh Timeline`, and verify event/trend results narrow to the selected scope with deterministic empty-state copy when no events match.
36. For any visible lifecycle event row, click `Open Inspect` and verify navigation switches to `Inspect` with search seeded to the selected plugin id.
37. For any visible lifecycle event row with kind `channel` or `connector`, click the diagnostics destination action (`Open Channels` or `Open Connectors`) and verify navigation switches to the expected destination section.
38. Click `Reset Filters` and verify all timeline filters return to defaults (`all` for pickers, empty plugin id, default limit) before re-query.
39. Confirm a `Panel Latency Budgets` disclosure is visible and collapsed by default, then expand it and verify:
   - status summary text is present
   - sample and regression badges are visible
   - `Latest by Section` region shows either deterministic empty copy or per-section sample rows
   - `Capture Current Panel` and `Clear Samples` actions are present
40. Click `Capture Current Panel` and verify sample count increases and `Latest by Section` includes/updates the current section row with `duration ms / budget ms` visibility.
41. Click `Clear Samples` and verify sample rows clear and status text returns to `No panel latency samples captured yet.`
42. Switch mode to `Integrations` and confirm a `Capability Grants Governance` disclosure is visible and collapsed by default, then expand it and verify:
   - `Upsert Capability Grant` form (`Grant ID`, `Actor`, `Capability Key`, `Status`, guided `Scope` editor, `Expires At`)
   - inventory filters (`Actor ID`, `Capability Key`, `Status`, `Limit`)
   - `Refresh Grants` and `Reset Filters` actions
   - grant inventory rows with `Load into Form`, `Revoke` (when not already revoked), and `Open Inspect`.
39. In `Upsert Capability Grant`, add at least one guided scope entry (`key` + `value`), click `Sync to Raw JSON`, and verify the advanced raw scope editor shows a deterministic JSON object.
40. In the same form, open `Advanced Raw Scope JSON`, toggle `Use raw JSON override when saving`, enter invalid JSON, and verify save is blocked with deterministic validation messaging; then enter valid JSON and verify save re-enables.
41. Disable raw override, click `Save Capability Grant`, and verify deterministic success/error status copy and refreshed inventory summary.
42. In inventory filters, apply at least one filter (for example `Status = ACTIVE` or a specific actor id), click `Refresh Grants`, and verify rows narrow or deterministic no-match copy appears.
43. On one grant row, click `Load into Form` and verify draft controls hydrate with row values; click `Open Inspect` and verify navigation switches to `Inspect` with grant/capability seed context.
44. On one non-revoked grant row, click `Revoke` and verify in-flight indicator plus deterministic post-action status copy in the row and section summary.
45. Confirm a `Communication Trust Receipts` disclosure is visible and collapsed by default, then expand it and verify a segmented inventory control (`Webhook` / `Ingest`), filter controls, inventory lists, and `Open Inspect` affordances.
46. In `Webhook` inventory:
   - apply one filter (for example `Provider` or `Provider Event ID`) and click `Refresh Webhook Receipts`.
   - verify deterministic summary status copy and row metadata (provider event id, trust badge/signature fields, payload hash, optional event/thread ids).
   - click one row `Open Inspect` and verify Inspect opens with receipt/event seed context.
   - when audit links are visible, click one audit-link id and verify Inspect search seeds to that audit context.
47. Switch to `Ingest` inventory:
   - apply one filter (for example `Trust State = accepted` or source/source-event query) and click `Refresh Ingest Receipts`.
   - verify deterministic summary status copy and row metadata (source/source-scope/source-event id, trust badge, payload hash, optional event/thread ids).
   - click one row `Open Inspect` and verify Inspect opens with ingest receipt/event seed context.
46. Use `Reset Filters` in both webhook and ingest inventories and verify controls return to defaults before re-query.
47. Switch mode back to `Workspace` and confirm an `Identity Devices and Sessions` section is visible with:
   - `Device Inventory Filters` (`User ID`, `Device Type`, `Platform`, `Limit`) plus `Refresh Devices` and `Reset Filters`.
   - `Session Inventory Filters` (`Device ID`, `User ID`, `Session Health`, `Limit`) plus `Refresh Sessions` and `Reset Filters`.
   - per-session `Revoke Session` action (only enabled for active sessions).
48. In `Device Inventory Filters`, set at least one filter and click `Refresh Devices`; verify deterministic summary status copy and row metadata (device label/id, user, platform/type, active/expired/revoked session counts).
49. In `Session Inventory Filters`, set at least one filter and click `Refresh Sessions`; verify deterministic summary status copy and row metadata (session id, device/user context, health badge, started/expires/revoked timestamps).
50. For one active session row, click `Revoke Session` and verify:
   - row-level in-flight indicator appears.
   - deterministic revoke status copy is shown.
   - session inventory refreshes and the session transitions to revoked state (or idempotent already-revoked copy is shown).
50a. In `Workspace`, verify `Identity Devices and Sessions` and `Delegation Rules` disclosures are collapsed by default on first render.
50b. In `Advanced`, verify `Daemon Lifecycle Controls` remains collapsed by default alongside other diagnostics disclosures.
50c. In `Workspace`/`Integrations`, verify delegation and capability-grant draft forms preserve deterministic behavior after principal-option refreshes: actor pickers auto-seed valid values, invalid grant combinations remain disabled, and guided/raw scope edits stay synchronized when toggling raw override.
50d. Across `Identity`, `Trust Receipts`, `Memory Browser`, and `Retrieval Context Inspector`, verify inventory cards use consistent loading/empty/has-more behavior and row-card spacing/actions remain consistent when switching modes.

Expected:

- Identity hub loads workspace/principal directory context from daemon identity APIs, including actor-handle mappings when available, and keeps deterministic fallback behavior when auth/context data is unavailable.
- Workspace/principal runtime summary rows reflect current app state (no hardcoded principal label).
- Workspace switching via Identity hub rehydrates app-shell workspace context and refreshes daemon-backed Configuration data without relaunch.
- Legacy workspace sentinel defaults (`default`) migrate to canonical `ws1` with persisted workspace-scoped UI context preserved.
- Identity refresh loading indicator is scoped to identity refresh actions only.
- Delegation section renders daemon-backed rule inventory with scope visibility (`scope_type`, optional `scope_key`) and deterministic loading/empty/error states.
- Delegation grant/revoke actions use daemon delegation APIs with deterministic validation and status copy (including self-delegation and scope-key constraints).
- Configuration mode navigation is built-in (`Setup`, `Workspace`, `Integrations`, `Data`, `Advanced`), defaults to `Setup`, and each mode hides unrelated sections to keep first-pass setup focused.
- Configuration rendering ownership stays mode-bounded (`Setup`, `Workspace`, `Integrations`, `Data`, `Advanced`) with the parent panel acting only as shell/router; no cross-mode section drift.
- Configuration draft/edit orchestration for delegation and capability grants is centralized in `ConfigurationDraftStore`, preserving deterministic seeding, validation, and mutation-input shaping behavior.
- Configuration identity/trust/memory/retrieval inventories use shared row/list primitives so loading/empty/has-more states remain behaviorally consistent across modes without changing action ownership.
- Configuration `Workspace` mode includes daemon-backed chat persona policy controls with scope-specific load/save/reset/test behavior and default-collapsed advanced guardrails editing.
- Configuration persona policy section shows deterministic response-shaping test context badges (`Test Channel`, `Profile`) and channel-mismatch guidance for `Test in Chat`.
- Setup overview presents compact readiness badges and prioritized quick actions (`Fix Next`, primary remediation, `Refresh Checks`) while retaining deterministic one-click remediation behavior.
- Setup overview includes explicit unsigned-build host-trust remediation guidance with deterministic first-launch override steps (`Open`/`Open Anyway`), expected permission-prompt guidance, and deterministic `Open Security Settings` + `Retry Setup Checks` actions.
- Setup overview keeps `Setup Matrix Details` collapsed by default and renders deterministic disabled-reason guidance when primary setup actions are unavailable.
- Setup `Runtime Details`, workspace `Identity Devices and Sessions`/`Delegation Rules`, and advanced `Daemon Lifecycle Controls` remain collapsed by default while preserving one-click access.
- Operator workflows in `Integrations`/`Data`/`Advanced` (`Capability Grants`, `Trust Receipts`, `Memory Browser`, `Retrieval Context Inspector`, `Runtime Supervisor Timeline`, `Panel Latency Budgets`) are visible as disclosure sections and default to collapsed until expanded by the user.
- Setup matrix deterministically reflects daemon lifecycle, token, provider setup, model catalog, and chat-route readiness, exposes resolved route context when available, and provides direct remediation actions (`Open Models` when route is unresolved/blocked).
- Worker-only plugin failures are classified as degraded runtime in setup matrix/taskbar context and route remediation to `Channels` diagnostics (`Open Channels`) rather than generic setup-repair blocking state; verify this is sourced from daemon lifecycle `health_classification` values.
- `Configuration > Setup` keeps detailed runtime/setup ownership in setup-matrix rows; non-setup modes keep summary-only runtime copy that points back to setup matrix detail.
- Runtime Supervisor Timeline queries daemon plugin lifecycle history with deterministic filters/status copy, renders per-plugin trend summaries plus lifecycle event rows, and supports direct drill-ins to `Inspect` and plugin-kind diagnostics destinations (`Channels`/`Connectors`).
- Panel Latency Budgets diagnostics shows deterministic latency summary, latest per-section samples with budget comparison, and clear/capture controls for manual regression checks.
- Memory Browser uses daemon context query APIs for principal/source-scoped inventory and compaction candidate preview visibility with deterministic filter summary, loading, empty, and error behavior.
- Retrieval Context Inspector uses daemon retrieval document/chunk query APIs with explicit selected-document flow before chunk inspection and deterministic status/empty/error messaging.
- Capability Grants Governance uses daemon capability-grant APIs with guided scope authoring by default (key/value entries), explicit advanced raw scope JSON override validation, deterministic upsert/list/revoke status copy, filterable inventory behavior, and row-level Inspect drill-ins.
- Communication Trust Receipts governance uses daemon webhook/ingest receipt APIs with deterministic filter/summary status messaging, trust-state/audit-link visibility, and Inspect drill-ins seeded from receipt/event/audit identifiers.
- Identity device/session inventories use daemon identity APIs with deterministic filter summary status copy, and session revocation uses daemon revoke API with clear in-flight/idempotent/post-refresh behavior.
- Retention and context controls remain visible and produce deterministic daemon-backed success/error summaries.
- Token UX remains write-only (no plaintext token echo).
- `Start/Stop/Restart` state reflects daemon lifecycle API controls.
- `Install`/`Repair` actions are enabled when Assistant Access Token is configured and no lifecycle action is already in progress; they run from bundled helper artifacts and surface deterministic remediation copy for `/Applications` placement and launchctl setup failures.
- `Uninstall` action remains enabled/disabled from daemon lifecycle `controls` payload.
- Setup action status messaging is deterministic across in-progress, succeeded, and failed outcomes.
- High-impact configuration actions use explicit confirmation (`daemon lifecycle`, `retention`, and `revoke` flows), with short-lived undo affordances for reversible daemon start/stop actions.
