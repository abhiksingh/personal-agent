# UI Tests: Communications and Tasks Panels

Source: [full guide](../tests-ui/full.md)

Regression anchors:
- run [Autonomous Unified-Turn Workflows (Manual-Only)](./12-autonomous-unified-turn-workflows.md) for cross-panel lifecycle drill-in continuity checks.
- run [Auth Scope + Rate Limit + Realtime Hardening (Manual-Only)](./11-auth-rate-limit-realtime-hardening.md) to validate shared remediation-card behavior for `Tasks`.
- for communications-store decomposition regressions (`U-257` onward), run `swift test --package-path source/apps/macos/app-host/Packages/PersonalAgentUI --scratch-path out/build/swiftpm/personal-agent-ui --filter "AppCommunicationsStoreTests|AppShellStateCommunicationsTests|AppShellStateDecompositionParityTests"` before manual checks.

## 7) Communications Panel Checks

1. Select `Communications`.
2. Confirm the panel header shows `Communications` and includes `New Message`, `Start Call`, and `Refresh` actions.
3. Verify the identity context bar shows `Workspace` and `Principal` display-name-first labels, and raw IDs appear only after explicit reveal/copy action.
4. Verify three data regions are visible:
   - `Conversation Continuity`
   - `Conversations by Channel`
   - `Event Timeline by Channel`
   - `Call Sessions`
5. In `Conversation Continuity`, verify rows are grouped under logical channel headers (`App`, `Message`, `Voice`) and each row includes `Open Chat`, `Open Related Tasks`, and `Open Related Inspect`.
5a. Verify channel-group headers display only channel + count, and connector attribution is not duplicated in the group header text.
6. In `Conversation Continuity`, verify row `Details` is collapsed by default and expanding reveals turn/correlation/thread/task/run metadata when available.
6a. Verify continuity primary rows keep a concise surface (`title`, lifecycle/type badges, summary, actions) and move connector/persona/shaping metadata into `Details`.
6b. In `Conversation Continuity`, verify shaping metadata/count rows are available in `Details`.
6c. In ready smoke-fixture mode, verify cross-channel lifecycle parity in `Conversation Continuity`:
   - `App` row shows awaiting-approval/blocked lifecycle state.
   - `Message` row shows completed lifecycle state.
   - `Voice` row shows failed lifecycle state.
6d. From those continuity rows, verify deterministic drill-ins remain stable:
   - `App` -> `Open Related Tasks` navigates to `Tasks` with source ribbon/chips.
   - `Message` -> `Open Related Inspect` navigates to `Inspect` with source ribbon/chips.
   - `Voice` -> `Open Chat` navigates to `Chat` with source ribbon/chips.
7. In `Conversations by Channel`, verify records are grouped under logical channel headers (`App`, `Message`, `Voice`) when data exists.
8. Under `Message`, verify connector-specific thread rows remain separate (for example `Messages` and `Twilio` do not merge into one thread row unless daemon returns one thread id).
9. Verify thread/event/call rows show connector attribution badges sourced from daemon `connector_id` when present.
9a. Verify grouped thread/event/attempt rows do not re-render logical channel badges inside each row (channel context is owned by the channel-group header).
10. Click `Refresh` and verify loading state appears without freezing sidebar interactions.
8. Verify thread rows surface triage badges when applicable (`Unread`, `New`, `Follow Up`, `Handled`).
9. On one thread, click `Follow Up`; verify a `Follow Up` badge appears.
10. On the same thread, click `Mark Handled`; verify follow-up clears and row state updates to handled.
11. Click `Reopen`; verify handled state clears and inbound thread status can return to `Unread`.
12. Click `Open Task Draft` from a thread row; verify navigation switches to `Tasks` and the `New Task` sheet opens with prefilled title/description.
13. Return to `Communications`; click `Follow Up` and `Open Task Draft` from an event row and verify identical triage/task-draft behavior via parent thread context.
14. Toggle `Compact Scan` on; verify rows collapse to denser summaries while preserving thread title, channel, connector, and key triage status.
15. With `Compact Scan` on, verify secondary actions are still reachable through row menus (`More`/`Actions`) and remain functional.
16. Toggle `Compact Scan` off and verify full-detail row rendering returns.
17. Click `New Message` and verify compose sheet opens as `Quick Send` with editable `Recipient` and multiline `Message` fields by default.
18. In compose sheet, verify `Advanced` is collapsed by default and expands to `Source Channel`, `Conversation ID`, and `Connector Hint` controls.
19. In compose sheet, submit with empty message and verify deterministic validation (`Message body is required.`).
20. In compose sheet, submit with no destination and no thread ID and verify deterministic validation (`Destination is required unless a thread context is selected.`).
21. Enter destination + message and submit; verify in-flight state, deterministic success/error status copy, and post-submit inbox refresh.
22. In `Conversations by Channel`, click `Reply` on one thread and verify compose sheet opens with thread-context prefill (visible through `Advanced`: `Conversation ID`, connector hint, optional recipient suggestion).
23. In reply compose sheet, clear recipient while keeping conversation context + message and submit; verify daemon thread-aware send path either succeeds or returns deterministic derivation error copy without crashing.
24. In `Conversations by Channel`, click `Start Call` and verify compose sheet opens with voice routing context prefilled (visible through `Advanced`).
25. In `Call Sessions`, click `Start Call` and verify destination/thread context prefill uses call-session metadata when present.
26. In `Event Timeline`, verify rows show sender/recipient metadata when available and include channel/type/direction context.
27. Use search input to filter by a known identifier (`thread_id`, `connector_id`, provider message id, or call id) and verify all three regions narrow consistently.
28. Set a `Channel` filter and verify non-matching logical-channel rows are hidden.
29. Set a `Direction` filter and verify timeline/call rows narrow deterministically.
30. Click a thread-level action that opens inspect context (for example `Open Related Inspect`) and verify navigation switches to `Inspect` with seeded search/filter context.
31. Click a channel navigation action (for example `Open Channel`) and verify navigation switches to `Channels` with logical channel status copy (`app|message|voice`).
32. Set `Thread` filter to one specific thread and verify `Delivery Attempts` section loads daemon attempt history for that context.
33. In `Delivery Attempts`, verify rows show retry/fallback/status evidence (`route phase`, `retry ordinal`, optional `fallback from channel`) and include `Open Related Tasks`, `Open Related Inspect`, and `Open Related Channels` actions.
34. In thread/event/call-session/delivery-attempt rows, verify `Details` is collapsed by default and expanding it reveals technical identifiers (thread/event/session/attempt IDs, operation/task/run linkage, idempotency/provider receipt where present).
35. Set `Thread` back to `All Threads` and verify delivery-attempt section shows deterministic guidance to select a thread context.
36. Clear filters and verify full lists return.
37. In a no-data workspace, verify deterministic empty-state copy is shown for each section (not a blank panel).
37a. In that no-data state, verify communications empty-state body does not repeat the same status sentence already shown in the panel header subtitle.
38. In degraded/offline scenarios, verify deterministic error/degraded copy is shown without generic decode-placeholder drift.
39. Set non-default search/channel/direction/thread filters and `Compact Scan`, switch to `Tasks`, then return to `Communications` and verify the same context is restored.
40. Verify a header-level active-filter indicator appears with count + summary tokens for active communications filters (including compact density mode when enabled).
41. Click `Clear Filters` in that header indicator and verify search/channel/direction/thread + compact mode all return to defaults.
42. Re-apply non-default filters, click toolbar `Clear Filters`, and verify full reset parity with header clear.
43. Relaunch app in the same workspace and verify last non-default communications filters (if set before quit) are restored.
44. In the same workspace, verify previously selected thread triage states (`Handled`/`Follow Up`) persist after relaunch.
45. Open `New Message`, enter draft content (for example destination + message), close the sheet without sending, switch to `Tasks`, then return to `Communications` and verify compose draft state/fields restore.
46. Click `Reset Context` in the communications toolbar and verify a destructive confirmation appears with `Cancel` and `Reset Context`.
47. Confirm `Reset Context` and verify filters/compact mode/triage/compose draft all reset to defaults for the active workspace.
48. Switch to a different workspace and verify previous workspace communications continuity state does not leak (compose draft is clean, triage/follow-up markers are not reused, filters follow the new workspace context).
49. Switch back to the original workspace and verify its prior persisted continuity state remains isolated.

Expected:

- Communications data is sourced from daemon comm inbox APIs (`/v1/comm/threads/list`, `/v1/comm/events/list`, `/v1/comm/call-sessions/list`) plus conversation continuity replay (`/v1/chat/history`).
- Communications outbound compose actions dispatch through daemon `/v1/comm/send` with deterministic validation and in-flight status messaging.
- `New Message`, `Reply`, and `Start Call` flows share one `Quick Send` compose surface with default `Recipient` + `Message` fields and a default-collapsed `Advanced` disclosure for `Source Channel`, `Conversation ID`, and `Connector Hint`.
- Reply/start-call entry points preserve thread-context prefill (`Conversation ID`, connector hint, optional recipient suggestion) even when `Advanced` remains collapsed.
- Reply compose flow supports destination-optional thread-aware sends; daemon success/failure copy is surfaced deterministically in-panel without UI interruption.
- Successful send updates render a `Latest Outbound Send` receipt card and trigger inbox/attempt refresh.
- Threads/events/call sessions consume first-class daemon `connector_id` attribution for row badges/search/filter context without parsing `external_ref`.
- Communications identity context bar renders workspace/principal display-name-first labels with explicit reveal/copy raw-ID affordances.
- Logical channel grouping (`App`/`Message`/`Voice`) is deterministic and does not merge connector-specific threads under `Message`.
- Conversation continuity rows are grouped by logical channel and expose deterministic drill-ins to `Chat`, `Tasks`, and `Inspect` with context chips at destination.
- Channel-group headers keep concise ownership (`channel + count`) and avoid duplicating connector metadata already present in row-level state or details.
- Conversation continuity preserves canonical cross-channel lifecycle parity (`App` awaiting approval/blocked, `Message` completed, `Voice` failed) after refresh and panel re-entry.
- Conversation continuity shaping/persona metadata is preserved in row `Details` (with collapsed-by-default disclosure) without duplicating those values in top-row badges.
- Conversation continuity primary rows keep scan-first lifecycle context and actions; connector/shaping/persona metadata is available in one collapsed `Details` disclosure.
- Communications triage states are deterministic and user-driven: thread/event triage controls support `Mark Handled`/`Reopen`, `Follow Up`/`Clear Follow Up`, and `Open Task Draft`.
- `Open Task Draft` from communications opens `Tasks` with a prefilled new-task draft sheet seeded from thread context.
- `Compact Scan` mode reduces row density for high-volume review while preserving channel/connector/thread clarity and action access.
- Delivery-attempt history is sourced from daemon `/v1/comm/attempts` with explicit thread context.
- Thread/event/call-session/attempt technical identifiers remain available behind row-level collapsed `Details` disclosures instead of inline short-ID labels.
- Panel supports deterministic loading, empty, no-match, and degraded error states.
- Communications no-data state includes deterministic one-click remediation CTAs (`Refresh Inbox` + section/dependency navigation such as `Open Channels` or `Open Configuration`).
- Communications empty-state status copy is suppressed when it matches the header subtitle, leaving one canonical status owner.
- Event rows preserve sender/recipient metadata visibility and channel/type context when daemon data is present.
- Search/channel/direction/thread filters narrow data deterministically and are resettable.
- Delivery-attempt rows expose retry/fallback/status metadata and workflow drill-ins when context identifiers are present.
- Cross-panel drill-ins to `Inspect` and `Channels` provide context-seeded navigation without duplicate refresh glitches.
- On first section load with empty cached inbox state, communications body renders deterministic skeleton placeholders before transitioning to populated or empty/error states.
- Communications filters (including compact mode) persist workspace-scoped across section switches/relaunch and `Clear Filters` clears both visible and persisted context.
- Communications triage state persists workspace-scoped for handled/follow-up thread intent.
- Communications compose draft state (sheet + fields) persists workspace-scoped across section switches/relaunch, invalidates stale thread context safely, and is cleared by `Reset Context`.
- `Reset Context` is destructive-confirmed and clears communications continuity state (filters, compact mode, triage, compose draft) deterministically for the active workspace.
- Communications shows a header-level active-filter summary (count + concise tokens) whenever non-default filters are active, with one-click clear parity to toolbar reset.

## 7A) Tasks Panel Checks

1. Select `Tasks`.
2. Verify task rows render run controls (`Cancel Run`, `Retry Run`, `Requeue Run`) in each row where `run_id` is present.
3. For a run where daemon metadata marks one action unavailable (for example `Retry` disabled), verify that action is disabled and a deterministic reason message is visible.
4. For a run where at least one action is available, click one control and verify confirmation appears with action-specific title/button copy.
5. Cancel the confirmation and verify no daemon action dispatch occurs and row status remains unchanged.
6. Trigger one confirmed run-control action and verify per-run in-flight status appears (`Cancelling task…`, `Retrying task…`, or `Requeueing task…`) plus spinner while request is active.
7. After completion, verify deterministic result status appears on the row and section header status updates.
8. Open `View Run Detail` for the same run and verify the same run-control actions/status are present in the detail `Summary` group.
9. In task rows with a run id, verify `View Run Detail` appears as the primary first action in the row action strip.
10. Trigger a control from detail view and verify list/detail status remains synchronized after refresh.
11. For a task row without `run_id`, verify run controls are disabled and show the deterministic missing-run-id reason.
11a. Trigger typed panel-problem responses (`auth_scope` and `rate_limit_exceeded`) for task/run list queries and verify the shared remediation card appears with deterministic actions (`Open Configuration`, `Retry`, `Open Inspect`) plus explicit retry disabled reason while list refresh is in flight.
12. Click `New Task`, verify the sheet opens as a goal-first flow with `Goal`, `Details`, and `Priority` visible by default.
13. Expand `Override Context` and verify `Task Class`, `Requested By`, and `Subject Principal` controls are available; collapse it and verify summary context remains visible.
14. Enter draft values (`Goal`, optional `Details`, and a non-default `Priority`), close the sheet without submitting, switch to `Chat`, then return to `Tasks` and verify task-submit draft state/fields restore.
15. With a draft still present, click `Reset Context` in `Tasks` and verify a destructive confirmation appears with `Cancel` and `Reset Context`.
16. Confirm `Reset Context` and verify the task-submit sheet closes and draft fields reset to defaults.
17. Re-open `New Task` after reset and verify Goal/Details are empty, Priority resets to `Medium`, and principal/task-class defaults are auto-resolved.
18. Switch to a different workspace and verify prior workspace task-submit draft state does not leak into the new workspace.
19. Switch back to the original workspace and verify task-submit continuity remains isolated per workspace.
20. In default `Simple` mode, verify task card summary copy uses plain-language task phrasing (`Task ...`) and avoids operator-heavy row summary wording.
21. Switch density to `Advanced` and verify operator phrasing returns in task summary copy (`Run ...`) while technical route metadata stays available in details.
22. In a workspace with no loaded tasks/runs, verify filter summary copy reads `Filters apply after tasks are loaded.` and does not revert to raw placeholder phrasing.
23. In the same no-task state, verify empty-state remediation copy appears once and does not duplicate the same panel-header status text inside the empty-state body.

Expected:

- Run-control button visibility and enablement is derived from daemon action metadata (`can_cancel`, `can_retry`, `can_requeue`) from `/v1/tasks/list`.
- Control actions use one shared confirmation contract before dispatch.
- Row action hierarchy keeps `View Run Detail` as the primary first action when available and renders `Cancel Run` with destructive styling.
- Disabled controls surface deterministic reason copy (missing run id, in-flight lock, or daemon-unavailable action state).
- In-flight state is per-run, blocks duplicate actions on that run, and presents explicit progress/status messaging.
- Successful/failed control outcomes update row/detail status copy consistently and do not desynchronize list vs detail surfaces.
- Tasks typed-problem failures (`auth_scope`, `rate_limit_exceeded`) use the shared panel remediation card/actions contract with deterministic retry disablement messaging during in-flight refresh.
- Task-submit draft continuity (sheet + `Goal`/`Details`/`Priority`/context overrides) persists workspace-scoped across section switches/relaunch and clears deterministically through destructive-confirmed `Reset Context`.
- `Simple` task summary/status wording remains action-first and plain-language in the default card body.
- `Advanced` mode restores operator wording for run-diagnostics context without removing default actions.
- Tasks panel empty/filter helper copy keeps one canonical status owner (header or remediation surface) without repeated duplicate status strings.
