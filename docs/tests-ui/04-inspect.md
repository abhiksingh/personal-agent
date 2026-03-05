# UI Tests: Inspect Panel

Source: [full guide](../tests-ui/full.md)

Regression anchor: run [Autonomous Unified-Turn Workflows (Manual-Only)](./12-autonomous-unified-turn-workflows.md) to validate inspect drill-in continuity and recovery context from unified turns.

Decomposition guardrail (`U-261+`): run `swift test --package-path source/apps/macos/app-host/Packages/PersonalAgentUI --filter AppInspectStoreTests` before/after inspect-store extraction changes.

## 8) Inspect Panel Checks

1. Select `Inspect`.
2. Confirm the view toggle defaults to `Activity`.
3. Click `Refresh` and verify activity cards render newest-first (LIFO) with status badge, human-readable event title, and concise summary text.
4. Toggle `Pause Tail` and verify live-tail status updates to paused while manual refresh remains available.
5. Toggle `Resume Tail` and verify live-tail resumes.
6. Change status filter (`All`, `Success`, `Running`, `Failure`) and verify rows are filtered accordingly in `Activity`.
7. In activity search, filter by one known identifier (`task`, `run`, `correlation`, provider, or model) and verify rows narrow deterministically.
8. Open one activity row with task/run context and click `Open Related Tasks`; verify navigation switches to `Tasks` and pre-fills identifier search.
9. Return to `Inspect` and click `Open Related Approvals` from one row with task/run/step context; verify navigation switches to `Approvals` and pre-fills identifier search.
10. Switch the mode toggle to `Trace` and verify metadata-focused controls appear (`Match` scope + `Group` picker).
11. In `Trace`, change metadata scope (`All Fields`, `Task`, `Run`, `Correlation`, `Provider`, `Model`) and verify matching behavior follows selected scope.
12. In `Trace`, change grouping (`No Grouping`, `Task`, `Run`, `Correlation`, `Provider`, `Model`) and verify grouped sections render while preserving newest-first rows inside each section.
13. Verify trace rows include input/output/metadata blocks and route/identifier context details.
14. With filters that match zero rows, verify deterministic no-match state appears and `Clear Filters` restores rows while keeping the current mode unchanged.
15. Leave panel open briefly and verify status text updates when live stream entries arrive (or timed-out polling remains stable).
16. Set non-default status/match/group/search filters in `Trace`, switch to `Approvals`, then return to `Inspect` and verify trace mode and filters are restored.
17. Switch back to `Activity` and verify trace-only `Match`/`Group` controls are hidden while status/search controls remain available.
18. Click `Open Gallery` and verify:
   - inspect renders a deterministic component gallery with status badges, action hierarchy examples, card surfaces, runtime banners, and remediation empty-state references.
   - status/search/live-tail controls are hidden while gallery mode is active.
19. Click `Back to Activity` and verify `Activity` rows + status/search/live-tail controls return.
20. Relaunch app in the same workspace and verify last selected inspect mode plus non-default filters (if set before quit) are restored.
21. In a workspace with no inspect logs, verify filter summary copy reads `Filters apply after inspect logs are loaded.`.
22. In the same empty state, verify the panel header owns status text and the empty-state body does not repeat the same status string.

Expected:

- Ordering remains newest-first.
- `Activity` is the default inspect mode and prioritizes user-readable event summaries.
- `Trace` mode preserves advanced metadata/group tooling and full debug detail rows.
- `Gallery` mode provides deterministic shared-component references and suppresses inspect filter/live-tail controls because gallery content is non-filterable.
- Status text reflects query/stream activity and errors clearly.
- Live-tail pause/resume controls do not break manual refresh.
- Status/metadata filtering narrows displayed rows without mutating underlying log snapshot ordering.
- Grouping preserves per-group newest-first ordering and does not break live-tail updates.
- Inspect rows with workflow identifiers provide one-click drill-ins to related `Tasks` and `Approvals` context.
- Inspect drill-ins open destination panels with context ribbon chips and a deterministic `Back to Inspect` affordance.
- Inspect no-data state provides direct remediation CTAs (`Refresh Logs` plus context navigation such as `Open Tasks`/`Open Configuration`).
- No crashes or UI freezes when daemon is offline or auth is invalid.
- On first section load with no cached rows, inspect body renders deterministic skeleton placeholders before transitioning to populated or empty/error states.
- Inspect filters persist workspace-scoped across section switches/relaunch and `Clear Filters` returns status/match/group/search context to defaults.
- Inspect shows a header-level active-filter summary (count + concise tokens) whenever non-default filters are active, with one-click clear parity to toolbar reset.
- Inspect status/helper copy keeps one canonical owner between header and empty-state surfaces (no duplicated status sentence in both places).
