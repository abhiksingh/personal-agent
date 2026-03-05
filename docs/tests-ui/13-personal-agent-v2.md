# Personal Agent macOS v2 (Manual-Only)

## Scope

Use this guide for `source/apps/macos/app-host-v2` checkpoints while v2 workflow wiring is in progress.

## U-266 Baseline Checks (Daemon Transport/Auth Core)

1. Run v2 package tests:

```bash
swift test --package-path source/apps/macos/app-host-v2/Packages/PersonalAgentUIV2
```

2. Confirm these assertions pass:
- `typed daemon problem mapping for missing auth is actionable`
- `probe daemon invalid URL maps validation problem state`

3. Expected outcome:
- v2 daemon transport/auth contracts compile and decode.
- typed problem-state mapping produces deterministic remediation actions (`Open Get Started`, `Retry`, `Update Token Scope`) instead of raw decode/network dump copy.

## U-267 Baseline Checks (Session/Config Store + Startup Restore)

1. In `Get Started`, set:
   - daemon URL (`http://127.0.0.1:7071` or valid daemon endpoint)
   - workspace (`ws1` or test workspace)
   - principal (`default` or test principal)
   - density (`Simple` or `Advanced`)
2. Save a non-empty Assistant access token via `Save Token`.
3. Switch to another section (for example `Connectors & Models`), then quit/relaunch app.
4. Expected relaunch state:
   - last selected section restores.
   - daemon URL/workspace/principal restore.
   - density mode restores for the restored workspace.
   - token is not shown in plain text and reports as stored via setup chips.
5. Clear token (`Clear Token`) and verify:
   - setup chips/reporting show token missing.
   - high-impact actions in Replay detail and Connectors/Models show deterministic disabled reason copy directing to `Get Started`.

## U-268 Baseline Checks (Shared Panel State + Mutation Lifecycle)

1. Open each v2 workflow (`Get Started`, `Replay & Ask`, `Connectors & Models`) and confirm panel-state banner behavior uses one shared pattern:
   - `loading`: appears during `Verify Daemon` / mutation in-flight moments.
   - `degraded`: appears when setup prerequisites are missing (for example token missing).
   - `empty`: appears for replay when filters return no rows.
2. In `Replay & Ask`, apply a filter that returns zero rows; verify:
   - replay panel state becomes `empty`.
   - banner action includes `Clear Filters`.
3. In `Replay Detail`, run `Approve` on a pending row and verify:
   - mutation lifecycle transitions through in-flight to success (button briefly disabled, then success feedback).
4. In `Connectors & Models`, remove token in `Get Started` and return; verify:
   - connector/model mutation actions are disabled with deterministic setup remediation.
   - panel-state banner remains the owner of degraded-status summary.

## U-269 Baseline Checks (Live Get Started Readiness Projection)

1. Open `Get Started` with a valid daemon URL + token + workspace + principal, then click `Verify Daemon`.
2. Confirm checklist milestones now project live daemon signals (not seeded local assumptions):
   - `Daemon lifecycle is operational` reflects daemon lifecycle + worker health.
   - `Default model route is resolved` reflects `/v1/models/resolve`.
   - `At least one external connector is healthy` reflects `/v1/connectors/status`.
   - `Replay has live assistant activity` reflects replay availability from approvals/tasks/chat history probes.
3. Break one prerequisite (for example clear token or use an invalid daemon URL), then click `Verify Daemon`:
   - top panel state transitions to `degraded` with one deterministic next action.
   - failing checklist row shows explicit recovery guidance, not raw transport/decode text.
4. Restore valid setup and verify again:
   - checklist progress updates and returns to all-green when live prerequisites are healthy.
   - readiness chips in `Session Setup` update to match lifecycle/route/connector/replay status.

## U-270 Baseline Checks (Checklist Action Wiring + Cross-Panel Fix Next)

1. Leave at least one setup checkpoint unresolved (for example route missing or no healthy connector).
2. In `Replay & Ask` and `Connectors & Models`, confirm a `Current Blocker` ribbon appears:
   - primary CTA is `Fix Next`.
   - secondary CTA routes to `Open Get Started`.
3. Click `Fix Next` from each non-setup workflow and verify routing behavior is canonical:
   - route/model/connectors blockers send you to `Connectors & Models`.
   - replay-activity blocker sends you to `Replay & Ask`.
   - lifecycle blocker routes to `Get Started` and runs daemon verify.
4. In `Get Started`, click unresolved checklist-row actions and verify:
   - action status feedback appears inline under that row.
   - status text is deterministic (no blank/silent action outcomes).
5. Complete each blocker and verify:
   - blocker ribbon disappears from non-setup workflows.
   - checklist row inline action feedback clears once step is complete.

## U-271 Baseline Checks (Replay Feed Daemon Query/Filter/Pagination)

1. Configure valid session readiness in `Get Started` and open `Replay & Ask`.
2. Verify initial replay list behavior:
   - panel banner shows loading while live replay fetch is in progress.
   - replay rows populate from daemon-backed history/approvals/tasks (not only seeded placeholders).
3. Apply filters and confirm deterministic narrowing:
   - set status filter (`Needs Attention`, `Failed`, `Automated Safely`) and verify row set updates immediately.
   - set source filter (for example `Voice`) and verify list includes only matching source rows.
   - apply search query and verify matching rows only; clear filters resets to baseline view.
4. If `Load More` appears at the bottom of replay list:
   - click `Load More` and verify in-flight state (`Loading…`) then additional older replay rows append.
   - current selection remains valid if previously selected row still exists.
5. Trigger a replay fetch failure (for example invalid daemon URL), then open `Replay & Ask`:
   - panel state becomes `degraded` with deterministic remediation action (`Retry` / `Open Get Started`).
   - no raw transport/decode payload dump appears in primary banner copy.

## U-272 Baseline Checks (Replay Detail Evidence Loaders)

1. Select a replay row that has daemon-backed identifiers (run/task/correlation) and observe `Replay Detail`.
2. Verify detail evidence lifecycle:
   - detail pane briefly shows `Loading replay evidence…`.
   - on success, detail blocks (`What came in`, `What the assistant understood`, `What happened`) refresh from daemon evidence.
3. Expand disclosures and verify evidence enrichment:
   - `Source Context` includes daemon-linked fields when available (for example run/task IDs).
   - `Decision Trace` reflects inspect run steps/logs (not only seed placeholders).
4. For rows with no additional daemon evidence:
   - detail pane shows deterministic `empty` guidance and still renders baseline replay summary.
5. Trigger an evidence load failure (for example invalid daemon URL) and verify:
   - detail pane shows `failed` guidance with `Refresh Evidence` and `Open Maintenance`.
   - pressing `Refresh Evidence` retries; `Open Maintenance` routes to `Connectors & Models`.

## U-273 Baseline Checks (Replay Inline Actions via Daemon Mutations)

1. Open `Replay & Ask`, select a low-risk `Needs Approval` replay row with daemon-backed references.
2. Click `Approve` and verify deterministic lifecycle:
   - row enters optimistic in-flight state (`Approval submitted. Waiting for daemon confirmation.`).
   - action controls are temporarily disabled with deterministic in-flight copy.
   - replay feed refresh reconciles row state from daemon-backed data.
3. Select another approval row and click `Reject`:
   - row enters optimistic in-flight state.
   - on success, replay status reconciles to failed/rejected daemon-backed state.
4. Select a failed replay row and click `Retry Action`:
   - row transitions to optimistic running state.
   - replay refresh reconciles task/run state from daemon response.
5. Select a running replay row and click `Stop Run`:
   - action copy indicates stop request submission.
   - replay refresh reconciles cancelled/failed terminal state from daemon task control.
6. Force a mutation failure (invalid token/scope or daemon error) and verify:
   - optimistic row state rolls back to the pre-action snapshot.
   - mutation lifecycle reports failure with explicit remediation copy.
7. Select seeded/mock rows missing daemon references and verify:
   - action row shows deterministic disabled reason (`Missing approval reference...` or `Missing task reference...`).
   - action buttons remain disabled until replay sync refresh provides daemon locators.

## U-274 Baseline Checks (Ask Composer Live `/v1/chat/turn` Wiring)

1. Open `Replay & Ask` and expand `Ask`.
2. Enter a context question (for example, `Why was approval required on the last run?`) and click `Send Question`.
3. Verify send lifecycle behavior:
   - ask mutation enters in-flight with deterministic copy (`Sending question…`).
   - composer input and send action are disabled while in-flight.
4. Verify success reconciliation:
   - ask mutation transitions to success.
   - replay list shows a daemon-linked row for the submitted question (with correlation/task references when available).
   - `Replay Detail` renders evidence refresh against daemon-linked correlation where available.
5. Force `/v1/chat/turn` failure (invalid token/scope or daemon error) and verify:
   - ask mutation transitions to failed with actionable copy.
   - optimistic ask replay row is marked failed with recovery guidance.
   - original question text is restored into Ask draft for one-click retry.
6. Use `Reseed from Selection` and send again to verify:
   - seeded ask draft routes through the same live daemon path.
   - no local-only synthetic ask flow remains.

## U-275 Baseline Checks (Replay Realtime Ingestion + Reconciliation)

1. Open `Replay & Ask` with valid daemon readiness and confirm replay realtime connects automatically.
2. Trigger live daemon activity (for example approval decision, task retry/stop, or ask send) and verify:
   - replay list updates without manual refresh.
   - selected `Replay Detail` evidence refreshes/reconciles after live updates.
3. Force a realtime transport issue (disconnect/stale-session/capacity) and verify:
   - replay panel shows degraded realtime problem state with deterministic `Retry Realtime Stream` action.
   - user context/selection is preserved while degraded state is shown.
4. Click `Retry Realtime Stream` and verify:
   - reconnect attempt starts immediately with deterministic feedback.
   - on success, degraded banner clears and live replay reconciliation resumes.
5. Force auth-scope failure for realtime and verify:
   - realtime does not spin in infinite reconnect loop.
   - problem state routes remediation toward `Get Started` token/scope update path.

## U-276 Baseline Checks (Connectors Live Inventory + Mutations)

1. Open `Connectors & Models` with valid daemon readiness and verify:
   - connector cards load from live `/v1/connectors/status` inventory (not only seed toggles).
   - each connector card shows deterministic status/summary + permission chip state when available.
2. For a disconnected connector, click `Connect` and verify:
   - action enters deterministic in-flight state.
   - daemon config upsert is applied and connector status reconciles from refreshed inventory.
3. Expand `Configuration`, edit one draft field, then verify:
   - `Save Config` is enabled only when draft changes exist.
   - `Reset Draft` restores the last daemon baseline values.
   - successful save clears draft-dirty state and updates status copy.
4. Click `Run Check` on one connector and verify:
   - check operation runs through daemon `test-operation`.
   - last-check summary/timestamp update deterministically.
5. For connectors with missing permission/remediation actions:
   - `Request Permission` triggers daemon permission request and status copy updates.
   - secondary remediation actions route through deterministic fallback behavior (`refresh` / `open destination`).
6. Break setup readiness (clear token or invalid URL) and verify:
   - connector actions disable with deterministic setup remediation reasons.
   - panel banner remains canonical owner of degraded summary.

## U-277 Baseline Checks (Models Live Inventory + Route Explainability)

1. Open `Connectors & Models` with valid daemon readiness and verify:
   - model rows load from live `/v1/models/list` catalog (provider/model IDs, readiness, endpoint) instead of seed-only toggles.
   - route summary loads from live `/v1/models/resolve` and identifies current provider/model/source.
2. For a disabled but provider-ready model, click `Enable` and verify:
   - model mutation enters deterministic in-flight state, then succeeds.
   - post-mutation model inventory refresh reconciles row state from daemon.
3. Click `Set Primary` on a non-primary enabled model and verify:
   - daemon `/v1/models/select` is invoked and row action status confirms route update.
   - route summary updates after reconciliation and prior simulation/explainability evidence is reset with explicit rerun guidance.
4. Use route actions:
   - click `Simulate Route` and verify `/v1/models/route/simulate` output appears under `Route Simulation` disclosure.
   - click `Explain Route` and verify `/v1/models/route/explain` summary/explanations appear under `Route Explainability`.
5. Break setup readiness (clear token or invalid daemon URL) and verify:
   - model actions (`Enable/Disable`, `Set Primary`, `Simulate Route`, `Explain Route`) are disabled with deterministic remediation guidance.
   - panel banner remains canonical owner of degraded-state summary/recovery actions.
6. Trigger model-route API failure (scope/transport/server) and verify:
   - model route status copy remains actionable and avoids raw decode/network dumps.
   - retrying via panel action or `Refresh` re-attempts live route/catalog load deterministically.

## U-278 Baseline Checks (Cross-Workflow Coherence + Accessibility/Readability + Release Checklist)

1. Run v2 package tests and harness:

```bash
swift test --package-path source/apps/macos/app-host-v2/Packages/PersonalAgentUIV2
./tools/scripts/check_harness.sh
```

2. Cross-workflow coherence pass (`Get Started`, `Replay & Ask`, `Connectors & Models`):
   - panel-state banner remains the canonical owner of loading/degraded/empty/error summary in each workflow.
   - setup blocker routing remains deterministic (`Fix Next` and `Open Get Started` route to canonical owner panels).
   - summary strips/chips are compact signal only; detailed diagnostics remain inside owner cards/disclosures.
3. Keyboard-first replay path:
   - tab to replay filters (`Status`, `Source`, `Replay Search`) and confirm each control is focusable and operable from keyboard.
   - navigate replay rows via keyboard focus and open a selected row without pointer-only interaction.
   - in `Ask`, verify `⌘↩` submits `Send Question` when enabled.
4. VoiceOver/accessibility contract checks:
   - filter controls expose deterministic labels/hints (`Status Filter`, `Source Filter`, `Replay Search`).
   - replay list rows announce source/status/instruction summary and expose open-detail hint.
   - replay detail action controls expose stable accessibility identifiers (`Approve`, `Reject`, `Retry`, `Stop Run`, `Ask Why`, evidence actions).
5. Reduced-motion behavior:
   - enable macOS `Reduce Motion` and relaunch app.
   - verify section switches and major panel state transitions avoid abrupt decorative animation and remain deterministic.
6. Readability pass:
   - verify primary row copy remains legible at standard window sizes (no clipped route/status summaries in Replay and Connectors/Models).
   - verify helper/status copy remains subordinate (`caption`) while primary task/action copy is visually dominant.
7. End-to-end trust flow release checklist:
   - `Get Started`: save token + verify daemon + clear unresolved checklist blockers.
   - `Replay & Ask`: inspect a daemon-backed replay item, perform one inline action, then ask one question.
   - `Connectors & Models`: run one connector check, verify model route summary, run route simulation/explainability.
   - expected result: user can trace one instruction from intake -> decision -> action -> route explanation without ambiguous ownership or inaccessible controls.

## Notes

- Interactive v2 daemon-backed workflow wiring and cross-panel lifecycle states are tracked in `U-269` through `U-278`.
- Keep this guide updated as each v2 backlog task lands.
