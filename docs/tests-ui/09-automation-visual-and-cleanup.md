# UI Tests: Automation, Visual Accessibility, and Cleanup

Source: [full guide](../tests-ui/full.md)

Regression anchors:
- run [Autonomous Unified-Turn Workflows (Manual-Only)](./12-autonomous-unified-turn-workflows.md) for approval-to-chat lifecycle continuity checks.
- run [Auth Scope + Rate Limit + Realtime Hardening (Manual-Only)](./11-auth-rate-limit-realtime-hardening.md) for shared panel-problem remediation checks on automation/approvals queries.

## 13) Automation + Runtime-State Cohesion Checks

1. Select `Automation` and click `Refresh`.
2. Verify no top-level banner reports `Failed to decode daemon payload`, and `Recent Trigger Evaluations` section is visible with deterministic loading/empty status copy.
2b. If no triggers/fire-history rows exist, verify the empty-state body shows at most one supplementary status line (no repeated management/simulation/fire-history defaults stacked together).
2a. Trigger typed panel-problem responses (`auth_scope` and `rate_limit_exceeded`) for automation queries and verify the shared remediation card appears with deterministic actions (`Open Configuration`, `Retry`, `Open Inspect`) and explicit retry disabled reason while automation refresh is in flight.
3. If fire-history rows exist, verify each row shows:
   - status badge
   - idempotency signal
   - fired timestamp
   - linked `Task ID` / `Run ID` values when present
   - route metadata (`Task Class`, `Provider`, `Model`, `Route Source`) when present
4. Use section-level `Open Tasks` and `Open Inspect` actions in fire-history and confirm navigation switches sections without UI errors.
5. For one fire-history row with task/run linkage, click `Open Related Tasks` and verify navigation switches to `Tasks` with search prefilled to linked run/task id, plus a drill-in context ribbon showing source (`Automation`) chips and a working `Back to Automation` action.
6. Return to `Automation`, click `Open Related Inspect` on a row with `Run ID`, and verify Inspect opens with a visible run filter badge and scoped logs.
7. For one fire-history row without `Task ID`/`Run ID` but with provider/model route metadata, click `Open Related Tasks` and verify `Tasks` opens with search prefilled to route model/provider context.
8. In `Inspect`, click `Clear Run Filter` and verify unscoped inspect logs can be loaded again.
9. Click `New Trigger` and create one `SCHEDULE` trigger with:
   - acting-as principal selected from picker
   - title/instruction
   - interval seconds
   - enabled toggle
10. Verify create action closes with deterministic status copy and row refreshes with the new trigger.
11. Edit the new trigger (`Edit`) and change title/instruction/cooldown.
12. Verify edit action refreshes row content and status copy; repeat submit without changes and verify idempotent/no-op wording.
13. Toggle trigger enablement (`Enable`/`Disable`) and verify row badge/state updates after refresh.
14. For one `ON_COMM_EVENT` trigger (create if needed), open edit form and verify:
   - source defaults render (`Event Type=MESSAGE`, `Direction=INBOUND`, `Assistant Emitted=false`).
   - guided token editors render for channels/principal/sender/thread/keyword filters with add/remove behavior.
   - `Advanced Raw Filter JSON` disclosure exists and is collapsed by default.
15. In the same `ON_COMM_EVENT` editor:
   - add duplicated/whitespace-heavy tokens in at least one guided filter editor.
   - verify inline validation hints explain lowercase normalization and duplicate collapse behavior.
   - verify preview summary and normalized JSON preview update as guided fields change.
16. In `Advanced Raw Filter JSON`, toggle `Use raw JSON override when saving`, enter invalid JSON, and verify submission is blocked with deterministic validation messaging; then enter valid JSON and verify save re-enables.
17. Save the edited trigger and reopen it; verify normalized values persist in guided fields and trigger update status copy is deterministic.
17a. In trigger cards, verify the top-row `Enabled/Disabled` badge is the canonical directive status owner and the details list does not repeat a second `Directive Status` row.
18. Run `Simulate -> Run Schedule Simulation` and verify status text updates with a success/error summary.
19. Run `Simulate -> Run Comm Event Simulation` and verify status text updates with a success/error summary and seeded event metadata.
20. Verify fire-history section refreshes after simulation and recent rows include updated status/idempotency/timestamp linkage context.
21. Delete a trigger from its card and verify it is removed from list after refresh; repeat delete on the same trigger id path (if possible) and verify idempotent deletion messaging remains deterministic.
22. If workspace has zero triggers, verify deterministic empty-state copy is shown instead of a generic placeholder.
23. Select `Approvals` and verify daemon-backed pending/final approval rows render with risk/principal/decision metadata and route context (`task class`, provider/model, route source) when data exists (or deterministic empty/error copy when none exists).
23b. Trigger typed panel-problem responses (`auth_scope` and `rate_limit_exceeded`) for approval queries and verify the shared remediation card appears with deterministic actions (`Open Configuration`, `Retry`, `Open Inspect`) and explicit retry disabled reason while approvals refresh is in flight.
23c. From a chat approval-request row, click `Open Approvals` and verify full decision controls remain available in `Approvals` (`Action`, `Decision By`, required-phrase helper, decision note, evidence).
23d. For a `policy` risk approval surfaced in chat, verify optional bounded inline fast-path controls (`Low-Risk Inline Decision`) do not replace Approvals ownership and still keep `Open Approvals` available for full evidence and decision rationale.
23e. In an empty approvals state, verify empty-state body does not repeat the same status sentence already shown in the Approvals header subtitle.
23. In at least one approval row, verify `Requested By`, `Subject`, `Acting As`, and `Decision By` (when present) default to display-name-first labels, and raw actor IDs appear only after explicit reveal/copy action.
23a. In the same approval row, verify a summary block renders `What happened`, `What needs action`, and `What next` before dense metadata, and `Details` starts collapsed.
24. In `Approvals`, enter a non-default search term and verify a header-level active-filter indicator appears with count + summary token, then use `Clear Filters` in that indicator and confirm search reset.
25. For one approval row with `Run ID`, click `Open Task Detail` and verify navigation switches to `Tasks`, opens run detail, and keeps the linked run in summary metadata.
26. Return to `Approvals`, click `Open Related Tasks` and verify navigation switches to `Tasks` with search prefilled to linked run/task id (or provider/model/task-class fallback when run/task ids are absent but route context exists).
27. Return to `Approvals`, click `Open Related Inspect` for one row:
   - if `Run ID` exists, verify Inspect opens with run-filter context.
   - if no `Run ID`, verify Inspect metadata search is prefilled from task or route fallback context.
28. Return to `Approvals` and expand the `Evidence` disclosure for one row with `Run ID`; verify:
   - a deterministic loading state appears on first expand.
   - step context renders inline (step/status/capability + updated timestamp).
   - `Step Input` and `Step Output` summaries render (or deterministic fallback copy when payload fields are unavailable).
   - related artifacts and audit snippets render without leaving `Approvals`.
   - `Reload Evidence` triggers refresh without collapsing the card.
29. Return to `Approvals`; for at least one pending approval row:
   - keep `Evidence` expanded during decision entry.
   - verify `Action` defaults to `Approve` and `Decision By` defaults deterministically (acting-as, requested-by, then selected principal fallback).
   - with `Approve` selected, set an incorrect phrase and verify submission is blocked with deterministic required-phrase guidance.
   - click `Use Required Phrase`, then submit `Approve and Continue`; verify in-flight progress and post-submit refresh.
   - switch `Action` to `Reject` and verify `Submit Rejection` succeeds without requiring the approval phrase.
29a. For one approval that originated from a chat tool outcome, verify card summary/details keep request provenance (approval request id and related run/task context) and still allow full inline decision flow without requiring navigation back to `Chat`.
30. After decision refresh, verify decision metadata, latest action-status copy, and `Evidence` disclosure expansion state remain visible on the approval card without manual id correlation.
31. Select `Tasks` and verify daemon-backed task/run rows render with state, priority, timestamp, principal metadata, and route context (`task class`, provider/model, route source) when available.
32. In `Tasks`, verify `Requested By`, `Subject`, and `Acting As` rows default to display-name-first labels, and raw actor IDs appear only after explicit reveal/copy action.
32a. In the same task row, verify a summary block renders `What happened`, `What needs action`, and `What next` before dense metadata, and `Details` starts collapsed.
33. In `Tasks`, click `New Task` and verify the submission sheet opens with goal-first defaults: `Goal`, `Details`, and `Priority`.
34. Expand `Override Context` and verify principal pickers use display-name-first labels while preserving raw actor IDs as selection values, and task class remains editable there.
35. In `New Task`, verify submit is disabled until `Goal` is non-empty, then submit one task and verify status copy reports accepted submission.
36. Verify the sheet auto-closes on success and `Latest Submitted Task` card appears with `Task ID`, `Run ID`, optional `Correlation`, and submitted timestamp.
37. In the `Latest Submitted Task` card, click `Filter to Run` and verify task search pre-fills to the submitted `Run ID`.
38. In `Tasks`, click `Open Related Inspect` on one row and verify navigation switches to `Inspect` with run focus when run id exists.
39. Return to `Tasks`, click `Open Related Approvals` on one row and verify navigation switches to `Approvals` with identifier search prefilled.
40. Return to `Tasks`; for a row with a run id, click `View Run Detail` and verify the detail sheet renders summary, route metadata, steps, artifacts, and audit entries.
41. In run detail, click `Open Related Inspect` and verify navigation switches to `Inspect` with the detail run id in focus.
42. Return to `Tasks` and reopen run detail, click `Open Related Approvals`, and verify navigation switches to `Approvals` with the detail run id prefilled.
43. Return to `Tasks`; in filter/search controls:
   - select one non-default `State` filter and verify rows narrow accordingly.
   - select one non-default `Priority` filter and verify rows narrow accordingly.
   - select one non-default `Principal` filter and verify rows narrow accordingly.
44. Verify a header-level active-filter indicator appears in `Tasks` with count + summary tokens for current non-default filters.
45. Enter a known `Task ID`, `Run ID`, provider, model, task-class fragment, or principal display-name fragment in search and verify list narrows to matching rows; then enter a non-matching value and verify deterministic no-match empty state appears.
46. Enable `Auto-refresh`, choose interval (`15s`, `30s`, or `60s`), and verify auto-refresh status text and loading indicator behavior update as refresh runs occur.
47. Use `Clear Filters` from the header indicator (or toolbar reset) and verify task filters/search return to defaults.
48. Put runtime into a non-healthy state (for example stop daemon or use an invalid token), then open `Chat`, `Inspect`, `Channels`, `Connectors`, `Models`, `Configuration`, `Automation`, and `Approvals`.
49. Verify runtime notice language remains consistent across those daemon-backed sections.
50. Trigger one deep-link flow twice in succession (for example `Approvals -> Open Related Tasks` or `Automation -> Open Related Inspect`) and verify destination section refresh/status updates occur on both invocations, including when destination was already selected.

Expected:

- Automation behaves as a daemon-backed inventory + management panel with stable empty/error states.
- Automation no-activity empty state includes direct remediation CTAs (`Create Trigger` when setup is ready, otherwise setup-first actions such as `Open Configuration`, plus refresh/navigation helpers).
- Automation fire-history section shows recent trigger evaluations/fires with deterministic status/idempotency/timestamp plus route-context (`task class`, provider/model, route source) copy and clear empty/error states.
- Fire-history rows provide direct drill-ins to related `Tasks` and run-scoped `Inspect` context, with deterministic provider/model fallback drill-in behavior when task/run linkage is absent.
- Drill-in destinations from Automation/Approvals/Tasks/Inspect/Communications show consistent context ribbons (`Opened from ...`) with bounded chips and a working `Back to ...` action.
- Automation create/edit/delete/toggle actions persist via daemon APIs and surface deterministic validation, success, and idempotent/no-op status copy.
- Automation ON_COMM_EVENT create/edit uses guided token editors with inline normalization hints, normalized payload preview, and an explicit advanced raw-JSON override path with deterministic validation messaging.
- Automation create/edit forms include `Acting As` selection and block out-of-scope actors with deterministic delegation-safe validation copy.
- Automation simulation actions surface deterministic daemon success/failure summaries.
- Automation typed-problem failures (`auth_scope`, `rate_limit_exceeded`) use the shared panel remediation card/actions contract with deterministic retry disablement messaging during in-flight refresh.
- Automation status/helper copy keeps one canonical supplementary status line (or none) instead of repeating management/simulation/fire-history status defaults across stacked text blocks.
- Trigger-card details do not duplicate directive status when enabled/disabled state is already shown in row badges.
- `Approvals` behaves as a daemon-backed inbox with deterministic pending/final grouping, summary-first card framing (`What happened`/`What needs action`/`What next`), collapsed-by-default details disclosure, route-context rendering, inline evidence disclosure (step input/output summaries + related artifacts/audit snippets), and guided in-app decision actions (`Action` selector, deterministic `Decision By`, approve phrase helper with `Use Required Phrase`, reject path without approval phrase requirement), plus direct task/list-detail/inspect drill-ins (including route-context fallback seeding when task/run linkage is absent), and persistent decision traceability/evidence expansion state after submit/refresh.
- Approvals typed-problem failures (`auth_scope`, `rate_limit_exceeded`) use the shared panel remediation card/actions contract with deterministic retry disablement messaging during in-flight refresh.
- Approvals empty-state status copy is suppressed when it matches the header subtitle, leaving one canonical status owner.
- Chat approval-request rows remain compact handoff/status surfaces and keep `Approvals` as canonical full decision/evidence owner; bounded low-risk inline fast-path controls are optional accelerators, not replacements.
- Chat-origin approvals preserve request provenance and remain fully actionable inline in `Approvals` without context loss.
- `Approvals`, `Tasks`, `Communications`, and `Configuration` identity surfaces default to display-name-first labels; raw workspace/actor IDs are reachable only through explicit reveal/copy affordances.
- `Approvals` and `Tasks` both surface header-level active-filter indicators (count + concise summary tokens) when non-default filters/search are active, with one-click clear parity to toolbar reset actions.
- `Tasks` behaves as a daemon-backed task/run list with summary-first card framing (`What happened`/`What needs action`/`What next`), collapsed-by-default details disclosure, deterministic route-context metadata rendering, and a goal-first `New Task` submission flow (`Goal`, `Details`, `Priority`) with explicit `Override Context` controls (`task_class`, `requested_by`, `subject_principal`) plus correlation receipt feedback (`task_id`/`run_id`), row/detail drill-ins to related Inspect/Approvals context, run-detail drill-in (steps/artifacts/audit), filter/search controls, optional auto-refresh loop, and deterministic empty/no-match/error handling.
- On first section load with no cached inventory, `Automation`, `Approvals`, and `Tasks` render deterministic skeleton placeholders before transitioning to populated or empty/error states.
- `Approvals` and `Tasks` filter/search context persists workspace-scoped across section switches/relaunch and explicit reset controls clear both visible and persisted context.
- Runtime notices are consistent for disconnected/degraded/missing/broken/stopped states.
- Cross-view deep links trigger one deterministic destination refresh even when the destination section is already selected.

## 14) Visual + Accessibility Checks

1. Traverse `Chat`, `Inspect`, `Channels`, `Connectors`, `Models`, and `Configuration`.
2. Verify consistent card surfaces, spacing rhythm, and typography hierarchy.
3. Verify controls are primarily native-styled (`.bordered`/system text fields/menus) and not over-customized.
4. Verify app shell uses native split-view/list styling with smooth sidebar transitions.
5. Hover over sidebar rows and action buttons; verify hover and pressed states are visible and consistent.
6. Toggle macOS **Reduce Motion** accessibility preference and trigger at least one toast + undo prompt; verify motion is minimized (no large slide transitions) while state changes remain clear.
7. Toggle macOS **Increase Contrast** and verify card/badge/toast chrome becomes more pronounced (stronger border/shadow separation) without clipping or layout drift.
8. Enable VoiceOver and verify icon-only controls announce descriptive labels/hints:
   - undo prompt dismiss (`Dismiss undo prompt`)
   - notification toast dismiss (`Dismiss notification toast`)
   - channel mapping reorder controls (`Move <connector> up/down`)
   - advanced config field remove buttons (`Remove advanced field <key>`)
   - workflow drill-in ribbon dismiss (`Dismiss drill-in context`) with hint about clearing source chips while staying in the current panel
9. Use keyboard-only navigation (tab, shift-tab, space/enter activation plus app shortcuts like `⇧⌘P` and `⇧⌘N`) and verify all primary flows remain operable without pointer interaction.
10. Open Command Palette and verify the search field is immediately focused (typing starts query input without extra click), then switch to `Communications`, `Approvals`, and `Tasks` and confirm each search field exposes deterministic accessibility IDs for automation (`communications-search-field`, `approvals-search-field`, `tasks-search-field`).
11. Trigger at least one successful high-frequency action in each panel category below and verify subtle success reinforcement appears without disruptive motion:
   - `Chat` send (`Turn Sent` badge and send-button pulse)
   - `Approvals` decision submit (`Saved` badge and action-button pulse)
   - `Tasks` submit or run control (`Send`/run-control pulse)
   - `Channels`, `Connectors`, or `Models` `Save All` / provider/route save button pulse

Expected:

- UI appears cohesive and macOS-native.
- App shell and taskbar menu present a flatter Tahoe glass treatment (lighter materials/separators, reduced shadow weight).
- No clipped content or overlap at typical window sizes.
- Interaction feedback is clear without relying on heavy motion.
- Increased-contrast mode strengthens visual separation for cards/badges/toasts while preserving Tahoe styling.
- VoiceOver announces actionable icon-only controls with deterministic, task-specific labels/hints.
- Primary workflows remain keyboard-operable end to end.
- Command palette opens into keyboard-ready search state, and core workflow filter/search fields remain directly discoverable through deterministic accessibility identifiers.
- High-frequency success actions show restrained and deterministic reinforcement cues, and those cues are reduced-motion safe.

## 16) Cleanup

- Quit the app from taskbar menu or `Cmd+Q`.
- If needed, remove temporary debug build artifacts manually from DerivedData.
