# PersonalAgent UI Manual Test Guide

This guide validates the macOS SwiftUI surfaces with daemon-backed lifecycle, chat, inspect, channel, and connector status wiring.

Coverage in this guide:

1. Taskbar menu status + lifecycle controls
2. Main app shell layout/navigation behavior
3. Chat panel composer + daemon chat turn UX
4. Communications inbox panel (threads, events, call sessions, drill-ins)
5. Tasks panel run inventory, detail drill-ins, and run controls
6. Inspect panel LIFO log presentation
7. Channels and connectors card systems
8. Configuration panel runtime/token/advanced lifecycle controls
9. Models panel provider/model visibility and controls
10. Automation + workflow panel daemon wiring and runtime-state cohesion
11. Visual + interaction polish consistency checks
12. Global notification center + toast behavior
13. App icon + menu bar branding asset verification
14. First-10-minute external-user success walkthrough (setup, first send, first task, first approval, first recovery action)
15. Fixture-driven decode contract baseline (`control_auth`, task-run action defaults, channel/connector descriptors)

## Quick Run (Automated + Manual Handoff)

Run the saved UI manual-test runner from repo root:

```bash
./tools/scripts/run_tests_ui.sh
```

To run UI + CLI + daemon runners together, use:

```bash
./tools/scripts/run_tests_all.sh
```

Note: UI validation still includes manual handoff checks after automated steps complete.

Useful options:

```bash
# Skip xcodebuild compilation and run only lightweight checks
./tools/scripts/run_tests_ui.sh --skip-build

# Open the built app after checks complete
./tools/scripts/run_tests_ui.sh --open-app

# Run deterministic app-host XCUITest smoke journeys (opt-in)
./tools/scripts/run_tests_ui.sh --run-app-host-smoke

# Run deterministic app-host visual regression baselines (opt-in)
./tools/scripts/run_tests_ui.sh --run-visual-regression

# Refresh committed visual baselines, then re-run in assert mode
./tools/scripts/run_tests_ui.sh --run-visual-regression --update-visual-baselines

# Write to a specific log path
./tools/scripts/run_tests_ui.sh --log-file out/logs/manual-tests/tests-ui-manual.log
```

Use this from repository root:

```bash
cd /path/to/PersonalAgent
```

## Automated Smoke Coverage (App-Host XCUITest)

Opt-in smoke automation is available through the app-host UI test target and fixture-backed daemon responses:

```bash
./tools/scripts/run_tests_ui.sh --run-app-host-smoke
```

Current automated smoke coverage includes:

1. App launch with deterministic fixture data.
2. Primary navigation across high-frequency sections.
3. Chat composer input + send-state flow with fixture response.
4. Tasks and approvals drill-in affordances (`View Run Detail`, `Open Task Detail`).
5. Channels and connectors card expand/interact checks.
6. Onboarding gate visibility and setup-section accessibility (`Models` remains accessible while gated).
7. First-10-minute journey assertions covering setup recovery affordance visibility, first chat send, first task submit, and first approval decision submission.
8. Command palette natural-language query smoke assertion (`settings setup`) where `Enter` executes the ranked first enabled destination command (`Open Configuration`).
9. Chat-to-action smoke journey in default `Auto` mode covering representative action outcomes (approval-required email, successful text, blocked file lookup, successful browse) with deterministic typed timeline + remediation action assertions.
10. Recovery/navigation journeys for missing auth (`Open Configuration`), route-missing onboarding (`Open Models` with blocker ribbon continuity), and degraded runtime (`Command Palette -> Connectors` with degraded connector evidence + remediation control visibility).

Manual testing in this document remains the source for broader UX, visual-polish, accessibility, and platform-integration validation not covered by smoke automation.

## Automated Visual Regression Coverage (App-Host XCUITest)

Opt-in visual regression coverage is available via deterministic fixture-backed app-host snapshots:

```bash
./tools/scripts/run_tests_ui.sh --run-visual-regression
```

Current visual regression scope:

1. Full-window shell baseline in `Chat` state.
2. Full-window panel baselines for `Configuration`, `Communications`, `Automation`, `Tasks`, `Approvals`, `Inspect`, `Inspect Gallery`, `Channels`, `Connectors`, and `Models`.
3. Tolerance-based pixel diff assertion against baseline PNGs stored in a writable visual-baseline directory.
4. Baseline directory resolution order:
   - explicit `PA_UI_VISUAL_BASELINE_DIR` (recommended when you want project-local or shared storage),
   - otherwise auto-resolved Application Support path used by the app-host test runner.

Baseline update workflow:

1. Pick a baseline directory (optional but recommended for deterministic local workflow):
   ```bash
   export PA_UI_VISUAL_BASELINE_DIR="$PWD/out/artifacts/ui-visual-baselines"
   ```
2. Refresh snapshots:
   ```bash
   ./tools/scripts/run_tests_ui.sh --run-visual-regression --update-visual-baselines
   ```
3. Re-run assertion mode:
   ```bash
   ./tools/scripts/run_tests_ui.sh --run-visual-regression
   ```
4. Review changed PNG files in your configured baseline directory before commit.

## 1) Prerequisites

- Xcode installed.
- `xcodegen` installed (`brew install xcodegen` if missing).
- macOS desktop session (for app launch + menu bar checks).
- Local daemon running (default: `127.0.0.1:7071`) with a valid auth token for UI requests.

## 2) Build + Harness Baseline

```bash
cd /path/to/PersonalAgent/source/apps/macos/app-host
xcodegen generate

cd /path/to/PersonalAgent
xcodebuild \
  -project source/apps/macos/app-host/PersonalAgent.xcodeproj \
  -scheme PersonalAgent \
  -configuration Debug \
  -derivedDataPath out/build/xcode-derived-data \
  CODE_SIGNING_ALLOWED=NO \
  build

cd /path/to/PersonalAgent/source/apps/macos/app-host/Packages/PersonalAgentUI
export PA_UI_DEFAULTS_SUITE="${PA_UI_DEFAULTS_SUITE:-com.personalagent.app.tests}"
PA_UI_DEFAULTS_SUITE="$PA_UI_DEFAULTS_SUITE" swift test --scratch-path /path/to/PersonalAgent/out/build/swiftpm/personal-agent-ui

cd /path/to/PersonalAgent
./tools/scripts/check_harness.sh
```

Expected:

- `xcodegen generate` succeeds.
- `xcodebuild ... build` succeeds.
- `swift test` under `Packages/PersonalAgentUI` succeeds with `PA_UI_DEFAULTS_SUITE` isolation and repo-local scratch path output.
- Harness checks pass.
- Package decode tests consume canonical client-integration fixtures from `packages/contracts/control/fixtures/client-integration` for daemon lifecycle `control_auth`, task-run action availability defaults, and channel/connector descriptor metadata.

## 3) Launch App

Launch from Xcode or open the deterministic Debug build path used by scripts:

```bash
open out/build/xcode-derived-data/Build/Products/Debug/PersonalAgent.app
```

Expected:

- App launches and menu bar icon appears.
- Main window appears with sidebar + content panel.
- Dock/app switcher icon uses the branded Orbit Chat Core app icon asset (not a default placeholder).

## 3A) First-10-Minute External User Success Walkthrough

1. Start from a first-run/onboarding state.
2. Verify `Finish Setup` appears and `Models` remains directly accessible.
3. In `Models`, verify `Current Blocker` ribbon appears with a primary `Fix Next` action.
4. Transition to a setup-ready state.
5. In `Chat`, send one prompt and verify assistant response renders.
6. In `Tasks`, open `New Task`, set `Goal`, submit, and verify `Latest Submitted Task` appears.
7. In `Approvals`, use `Use Required Phrase`, submit `Approve and Continue`, and verify success status copy is shown.
8. Confirm one recovery affordance is visible and actionable during setup (`Fix Next`) or post-action remediation (`Open Related ...`, notification center next action).

Expected:

- First-run users see a deterministic setup path with visible recovery affordances before workflow actions.
- First successful chat send, task submit, and approval decision can be completed in one continuous flow.
- Recovery/remediation actions are surfaced inline at decision points without requiring hidden/debug-only paths.

## 4) Taskbar Menu Checks

1. Click the menu bar icon.
2. Verify the menu bar glyph uses the branded Orbit Chat Core template symbol (not the generic SF Symbol stack glyph).
3. Verify status region shows daemon + app connection labels.
4. On first launch before lifecycle status resolves, verify daemon/connection rows render neutral `Checking` values (not stale degraded/error copy).
5. Verify compact `Readiness` section appears with either `Ready` or setup hints.
6. For setup gaps, verify readiness actions are present and functional:
   - token/auth-state gap (`control_auth=missing`) -> `Open Config`
   - provider/model route gap -> `Open Models`
   - daemon gap -> contextual `Start`/`Install`/`Repair`
   - plugin worker degradation gap -> `Open Channels`
7. Verify daemon controls are present: `Start`, `Stop`, `Restart`.
8. Click one daemon lifecycle control (`Start`, `Stop`, or `Restart`) and verify a confirmation dialog appears with deterministic action copy plus `Cancel`/confirm actions.
9. For `Start` or `Stop`, confirm the action and verify a short-lived undo prompt appears in-app (`Undo Start`/`Undo Stop`).
10. Verify `Refresh` updates daemon state/detail text.
11. Verify main-window lifecycle control toggles label between `Open Window` and `Close Window`.
12. Verify `Quit` is present.

Expected:

- Menu structure uses native grouped form sections and remains minimal/readable.
- Menu bar icon rendering uses the dedicated template asset and remains legible in light and dark appearances.
- Menu window remains compact (reduced vertical spacing and concise labels).
- Runtime status rows use neutral `Checking` copy before first lifecycle status load completes.
- Readiness hints prioritize the highest-impact setup gaps and cap visible rows for compactness.
- Token readiness status follows daemon lifecycle `control_auth` metadata (`configured|missing`) when lifecycle status is loaded.
- Plugin-worker-only degradation in daemon lifecycle status appears as a dedicated readiness issue with `Open Channels` remediation (instead of generic setup-repair guidance), driven by daemon `health_classification` (`core_runtime_state=ready`, `plugin_runtime_state=degraded`).
- Controls are enabled/disabled according to daemon lifecycle `controls` payload.
- High-impact daemon lifecycle controls show deterministic confirmation copy before dispatch, and `Start`/`Stop` confirmations surface a short-lived undo affordance.

## 5) App Shell + Navigation Checks

1. In main window, confirm ready-state launch lands on `Home` and sidebar ordering is:
   - Top: `Configuration`
   - Workflow: `Home`, `Chat`, `Communications`, `Tasks`, `Approvals`, `Automation`
   - Advanced disclosure (collapsed by default): `Inspect`, `Channels`, `Connectors`, `Models`
2. In `Home`, verify a single primary next-action card is visible (`Next Best Action` or setup recovery equivalent) and that `Guided First Session` checklist rows are present with deterministic `Done`/action button states.
   - When first-session checklist is still incomplete, verify order is `Primary Next Action` -> `Guided First Session` -> `Quick Actions`.
   - When first-session checklist is complete, verify `Quick Actions` moves above `Guided First Session` (`Primary Next Action` -> `Quick Actions` -> `Guided First Session`).
   - While setup readiness is still incomplete, navigate to a workflow panel and verify the guided-session ribbon does not appear.
   - Once setup readiness is complete and first-session milestones are still incomplete, navigate to a workflow panel and verify a compact guided-session ribbon appears with `Step x of y` context.
   - If the ribbon points to another panel, click the guided primary action and verify it routes to the next milestone destination.
   - Click `Open Home Checklist` from the ribbon and verify navigation returns to `Home`.
   - Switch density mode to `Advanced` and verify `First-Success Funnel Diagnostics` appears with workspace completion summary plus per-milestone completion source/timestamp evidence (or explicit pending state).
3. Set one non-default filter in each persisted-filter section (`Communications`, `Tasks`, `Approvals`, `Inspect`) and verify corresponding sidebar rows show compact active-filter count badges.
   - Open `Communications`, `Tasks`, `Approvals`, and `Inspect` and verify each panel uses the same top scaffold order: header, optional active-filter summary, runtime banner, filter bar, divider, then content.
4. Hover each non-zero sidebar filter badge and verify help text shows concise persisted-filter summary tokens for that section.
5. Confirm sidebar footer shows daemon + connection status labels.
6. If onboarding is incomplete, select one non-setup section (for example `Chat`) and verify a single-path setup wizard is shown with progress (`x of y checks ready`), one current step, and one primary remediation action.
7. While onboarding is still incomplete, select `Configuration`, `Channels`, `Connectors`, and `Models` and verify each section opens directly instead of rendering the setup gate panel.
8. In `Channels`, `Connectors`, and `Models` (while readiness is incomplete), verify a compact `Current Blocker` ribbon appears above panel content and does not block normal panel interaction.
9. In that ribbon, click `Fix Next` and verify the highest-priority onboarding blocker remediation executes deterministically.
10. Verify the ribbon also shows one contextual secondary action (`Open Models`, `Open Configuration`, `Open Channels`, or `Refresh`) matching the active blocker context.
11. Click sidebar toggle button in the window toolbar.
12. Open `Channels`, edit one channel draft field without saving, then click a different sidebar section (for example `Connectors`).
13. Verify an unsaved-changes confirmation alert appears with section-specific summary text and `Stay`/`Discard Changes` actions.
14. Click `Stay` and verify current section and draft values remain unchanged.
15. Trigger the same navigation again, click `Discard Changes`, and verify navigation proceeds and the prior section draft resets.
16. Trigger one deterministic status action (for example `Channels > Refresh`) and verify a non-blocking toast appears in the top-right corner.
17. Click the toolbar `bell` button and verify a `Notification Center` sheet opens.
18. In the sheet, verify notifications are grouped by intent (`Needs Attention`, `Workflow Updates`, `Runtime and Setup`, `Diagnostics`, `General`) and rows remain newest-first within each group.
19. Verify rows include direct next-action affordances when available (for example `Open Channels`, `Open Tasks`) and clicking one marks the row read and navigates to the destination panel.
20. Use the search field and source filter, and verify rows narrow deterministically without breaking intent grouping.
21. Use `Mark All Read`, `Clear Read`, and `Clear All` actions and verify unread count and list state update immediately.
22. Click the toolbar `command.square` button and verify `Command Palette` opens with grouped `Navigation`, `Diagnostics`, `Workflow`, and `Runtime` command sections.
23. In command palette search, enter `inspect`, run `Open Inspect`, and verify section navigation switches to `Inspect`.
24. Re-open command palette and run `Refresh Current Panel`; verify active section refreshes without navigation drift.
25. Re-open command palette, search `inspect`, press `Enter`, and verify the first enabled match executes without pointer interaction.
26. Search a natural-language query (for example `start service`) and verify command palette ranks the intent-matching command (`Start Daemon`) first when enabled.
27. Search `open` and verify ties resolve deterministically in catalog order (`Open Configuration`, then `Open Chat`, then other open-navigation commands).
28. In command palette, run two different commands (for example `Open Chat` then `Open Inspect`), close/re-open the palette, and verify a `Recent` section appears with most-recent-first ordering.
29. Put daemon controls into an unavailable state (for example no `start` permission or lifecycle control in-flight), open command palette, and verify disabled runtime commands remain visible with deterministic disabled-reason copy.
29a. In command palette search, query a known concrete object from loaded state (for example task `run_id`, approval `id`, communication `thread id`, connector `id`, or `provider/model` key) and verify an `Objects` results section appears with deterministic ordering.
29b. Activate one object result and verify navigation opens the corresponding section with object context seeded (for example Tasks search seed, Approvals search seed, Communications thread filter, Connectors section status, or Models status focus).
30. Use keyboard shortcuts:
   - `⌘2` -> `Chat`
   - `⌘7` -> `Inspect`
   - `⌘8` -> `Channels`
   - `⌘9` -> `Connectors`
   - `⌘0` -> `Models`
   - `⇧⌘P` -> command palette
   - `⇧⌘N` -> notification center
31. While advanced disclosure starts collapsed, trigger one advanced shortcut (`⌘7`, `⌘8`, `⌘9`, or `⌘0`) and verify the advanced disclosure auto-expands with the selected destination highlighted.
32. Verify `Diagnostics` menu shortcuts (`⌥⌘I`, `⌥⌘C`, `⌥⌘K`) switch to `Inspect`, `Channels`, and `Connectors`.
33. Verify runtime command shortcuts (`⌥⌘S`, `⌥⌘.`, `⌥⌘R`) are disabled when daemon controls are unavailable/in-flight and enabled when daemon lifecycle control flags allow action.
34. In the toolbar, open the density control and verify both `Simple` and `Advanced` options are visible, with `Simple` selected by default for new workspace state.
35. Switch density to `Advanced`, open `Chat`, and verify operator trace metadata remains available only behind an explicit `Details` disclosure in workflow-context areas (no inline short-ID fragments in the default card body).
36. Open command palette, search for `density`, and verify `Set Density: Simple` and `Set Density: Advanced` actions are listed with deterministic enablement based on current mode.
37. Run `Set Density: Simple` from command palette and verify the same metadata rows collapse back to simplified user-facing copy.

Expected:

- Sidebar uses native split-view/list behavior and collapses/expands without layout breakage.
- Only one sidebar toggle affordance is visible in the toolbar.
- Main panel expands when sidebar is collapsed.
- Window title tracks selected section.
- Ready-state launch lands on `Home` first and shows one deterministic primary next action plus guided first-session milestone rows.
- `Home` ordering remains action-first and progressive: incomplete setup/session keeps checklist before quick actions, while completed first-session state promotes quick actions directly under primary action.
- In `Advanced` density mode, `Home` shows a `First-Success Funnel Diagnostics` card with aggregate completion metrics and per-milestone completion evidence (`source` + timestamp or pending state).
- While setup readiness is incomplete, workflow panels do not show the guided-session ribbon.
- Once setup readiness is complete and first-session milestones remain incomplete, workflow panels show a guided-session ribbon with `Step x of y`, deterministic next-step routing, and `Open Home Checklist`.
- Sidebar advanced destinations (`Inspect`, `Channels`, `Connectors`, `Models`) are progressively disclosed, collapsed by default, and auto-expand when selected via keyboard shortcut/command/deep-link navigation.
- `Communications`, `Tasks`, `Approvals`, and `Inspect` share consistent panel scaffold and filter-bar chrome for header/filter/loading/empty/action transitions.
- Sidebar rows for `Communications`, `Tasks`, `Approvals`, and `Inspect` display compact active-filter count badges when persisted non-default filters are active in the current workspace.
- Sidebar filter badges expose concise persisted-filter summary help text on hover.
- Sidebar runtime footer shows neutral `Checking` labels until first daemon lifecycle load resolves.
- Onboarding panel appears for non-setup workflow sections until token/daemon/provider/model-catalog/chat-route/channel-connector mapping criteria are met, while `Configuration`, `Channels`, `Connectors`, and `Models` remain directly accessible.
- Onboarding wizard always surfaces one deterministic highest-priority unresolved blocker in `Fix Next`, and the primary CTA performs one-click remediation (deep-link or lifecycle control) without extra navigation steps.
- Onboarding wizard keeps full checklist visibility available behind an optional collapsed disclosure while preserving single-path default guidance.
- Non-Configuration setup-accessible panels (`Channels`, `Connectors`, `Models`) show a compact `Current Blocker` ribbon while setup readiness is incomplete.
- `Current Blocker` ribbon includes one-click `Fix Next` plus one contextual secondary action (`Open Models`, `Open Configuration`, `Open Channels`, or `Refresh`) and does not block normal panel interaction.
- Cross-section navigation from `Channels`, `Connectors`, and `Models` shows deterministic discard confirmation when unsaved drafts exist.
- Major panel status updates surface as non-blocking toasts and persist to a global notification center activity log.
- Notification center groups rows by intent (`Needs Attention`, `Workflow Updates`, `Runtime and Setup`, `Diagnostics`, `General`), keeps newest-first ordering inside each group, and preserves deterministic read/clear plus source/query filtering controls.
- Notification rows expose deterministic next-action affordances (for example `Open Channels`, `Open Tasks`) that mark rows read and navigate to the mapped destination panel.
- Command palette exposes searchable grouped commands for section navigation, diagnostics, workflow actions, and runtime controls using one shared enablement contract.
- Command palette ranks natural-language query intent matches deterministically, with stable tie-break ordering for equal-score matches.
- Command palette search mode surfaces ranked concrete object matches (`Tasks`, `Approvals`, `Threads`, `Connectors`, `Models`) with deterministic ordering and one-click open behavior.
- Command palette executes first enabled search match on `Enter`, surfaces deterministic disabled reasons for unavailable actions, and prioritizes recently used commands in `Recent`.
- Keyboard shortcuts for section switching, command palette, notification center, diagnostics destinations, and runtime controls dispatch deterministic actions without bypassing guard rails.
- Workflow drill-ins (`Open Related ...`) present a destination context ribbon with source panel chips and a deterministic `Back to ...` return action.
- Toolbar includes a global information-density control with `Simple` default.
- Density mode command-palette actions remain in parity with toolbar state and expose deterministic enabled/disabled behavior.
- `Simple` mode prioritizes user-readable summaries while `Advanced` restores full operator metadata visibility.
- User-facing status paths in `Chat`, `Models`, `Channels`, `Connectors`, `Automation`, `Approvals`, and `Tasks` show remediation-first guidance and do not surface raw daemon transport/decode strings (for example `Failed to decode daemon payload` or `Daemon request failed (...)`) in default panel states.

## 6) Chat Panel Checks

1. Select `Chat`.
2. Confirm transcript starts without seeded assistant placeholder messages and shows explicit first-turn guidance.
3. Confirm large transcript area + multiline composer are visible, and composer includes an `Acting As` selector.
4. Confirm transcript renders typed timeline rows (user/assistant rows plus tool/approval/system rows when present), not a message-only fallback transcript path.
5. Confirm the composer exposes one autonomous `Send` path and no `Ask`/`Act` mode override controls.
6. Type multiline content using `Shift+Enter` for a newline.
7. Press `Enter` to send.
8. Observe realtime streaming state copy in the header/composer and token deltas appearing in transcript when daemon realtime is reachable.
9. While a chat turn is in flight, click `Interrupt`.
10. Verify chat request is cancelled and status copy updates deterministically (`Interrupt requested…` then `Chat interrupted.`).
11. Submit representative action prompts and verify deterministic autonomous outcome rendering for:
   - approval-required path
   - successful action path
   - blocked-readiness path
12. In multi-step tool workflows, verify tool/approval rows expose chain context labels (`Chain n`, `Step x of y`) and row status badges map to deterministic labels (`Pending`, `Running`, `Blocked`, `Complete`, `Failed`).
13. For tool and approval timeline rows, verify action buttons are deterministic (`Resume Turn`, `Retry Turn`, `Open Inspect`, `Open Approvals`, `Open Tasks`, `Cancel`) and failed/blocked tool rows include inline remediation actions (`Open Configuration`, `Open Connectors`, `Open Channels`) when applicable.
14. In one `approval_request` row, verify inline decision controls render (`Action`, `Decision By`, approve phrase helper `Use Required Phrase`, optional decision note), validation copy is deterministic, and submit buttons respect disabled reasons.
15. Trigger one remediation/timeline action and verify row-level action status copy updates inline and action buttons disable while that action is in flight.
16. Expand one timeline row `Details` disclosure and verify technical metadata appears; confirm disclosures are collapsed by default on first render.
17. In a workspace with no enabled ready chat model route, click `Send` and verify preflight blocks daemon turn submission with remediation copy.
18. Verify chat shows route-remediation actions (`Open Models`, `Check Again`) and does not append raw daemon 400 text as an assistant transcript message.
19. Click `Open Models`, configure/enable/select a valid chat route, return to `Chat`, then click `Check Again` and confirm remediation clears.
20. Trigger one chat turn with daemon realtime intentionally unavailable (for example by blocking websocket endpoint while keeping `/v1/chat/turn` reachable) and verify fallback status copy is shown.
21. Trigger one non-route chat failure condition (for example daemon connectivity or auth failure) and verify chat shows guided retry/remediation card actions (`Restore Prompt`, `Refresh Daemon`, `Open Configuration`).
22. Verify partial streamed output remains visible when daemon receipt fails after realtime lifecycle completion.
23. Use `Restore Prompt` and verify the last failed prompt is restored into the composer for user-controlled resend.
24. Verify chat header and composer footnote show resolved provider/model route context when available.
25. During an in-flight turn and after one successful turn, verify a single `Effective Workflow Context` card is visible and shows task class plus provider/model route context in the default body.
25a. In `Effective Workflow Context`, verify the default body renders summary-first reliability copy in order: `What happened`, `What needs action`, and `What next`.
26. Expand `Effective Workflow Context > Details` and verify technical metadata rows (route source, task/run state and IDs, correlation ID, approval/clarification hints when present) are available when daemon data exists; collapse by default on first render.
27. In `Effective Workflow Context`, click `Open Related Tasks` and verify navigation switches to `Tasks` with identifier search prefilled from run/task/correlation or route fallback context.
28. Return to `Chat`, click `Open Related Inspect` from the same context card, and verify navigation switches to `Inspect` with run filter or metadata search prefilled from chat context.

Expected:

- Transcript starts clean (no synthetic assistant bootstrap turn) with explicit empty-state guidance.
- Chat empty state surfaces direct remediation CTAs with deterministic visibility (`Open Configuration` when token missing, `Open Models`/`Check Again` when route remediation is active, runtime refresh when disconnected/degraded).
- `Shift+Enter` inserts newline.
- `Enter` sends.
- Chat uses a single autonomous `Send` path and does not expose ask/act mode overrides.
- Autonomous send supports tool/action progression via canonical typed timeline items without manual mode switching.
- Chat composer requires an explicit `Acting As` selection before submit and keeps the chosen actor visible.
- When selected actor is outside active identity scope, chat submit is blocked with deterministic delegation-safe validation copy.
- User message appears in transcript.
- Transcript renders typed timeline items and does not rely on legacy message-only fallback branches.
- Assistant tokens stream via realtime when available and reconcile to final daemon `chat.turn` response text.
- `Interrupt` is visible only while streaming, sends best-effort cancel signal, and cancels local in-flight request deterministically.
- Iterative tool workflows show chain context labels (`Chain n`, `Step x of y`) and deterministic row-state badges (`Pending`, `Running`, `Blocked`, `Complete`, `Failed`).
- Tool and approval timeline rows provide deterministic action affordances (including `Resume Turn`) with explicit disabled reasons, inline row-level action status copy, and default-collapsed technical details.
- Approval-request timeline rows provide inline guided decision controls (`Action`, `Decision By`, phrase helper, decision note) that reuse approval validation/submission semantics from the Approvals panel and still provide `Open Approvals` handoff when richer context is needed.
- Chat send performs a route preflight check and surfaces direct Models remediation when no enabled ready chat route is available.
- Route remediation does not inject raw daemon 400 route errors into transcript assistant messages.
- Non-route chat failures surface deterministic guided retry/remediation actions with direct runtime/setup navigation.
- Last failed prompt can be restored into composer for retry without manual retyping.
- When realtime is unavailable/disconnected, chat still completes via one-shot `/v1/chat/turn` with explicit fallback messaging.
- Realtime lifecycle completion/error events finalize chat turns deterministically, preserving partial streamed output when final HTTP receipt fails.
- Chat header/composer show resolved provider/model route context when available.
- Effective workflow-context card default copy is summary-first (`What happened`, `What needs action`, `What next`) and surfaces deterministic next-step/recovery guidance for approval-required, clarification-required, failed-step, and active-in-flight states.
- In default `Simple` mode, workflow-context summary copy stays plain-language and avoids identifier-heavy/operator phrasing; switching to `Advanced` restores operator wording/identifier detail where applicable.
- Successful turns surface daemon-provided `task_run_correlation` metadata plus unified-turn approval/clarification hints in `Effective Workflow Context > Details`, while default card copy stays workflow/provider-model focused.
- Chat workflow-context card provides direct `Open Related Tasks` and `Open Related Inspect` navigation affordances with context seeding.

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
6. In `Conversation Continuity`, verify row `Details` is collapsed by default and expanding reveals turn/correlation/thread/task/run metadata when available.
7. In `Conversations by Channel`, verify records are grouped under logical channel headers (`App`, `Message`, `Voice`) when data exists.
8. Under `Message`, verify connector-specific thread rows remain separate (for example `Messages` and `Twilio` do not merge into one thread row unless daemon returns one thread id).
9. Verify thread/event/call rows show connector attribution badges sourced from daemon `connector_id` when present.
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
- Communications triage states are deterministic and user-driven: thread/event triage controls support `Mark Handled`/`Reopen`, `Follow Up`/`Clear Follow Up`, and `Open Task Draft`.
- `Open Task Draft` from communications opens `Tasks` with a prefilled new-task draft sheet seeded from thread context.
- `Compact Scan` mode reduces row density for high-volume review while preserving channel/connector/thread clarity and action access.
- Delivery-attempt history is sourced from daemon `/v1/comm/attempts` with explicit thread context.
- Thread/event/call-session/attempt technical identifiers remain available behind row-level collapsed `Details` disclosures instead of inline short-ID labels.
- Panel supports deterministic loading, empty, no-match, and degraded error states.
- Communications no-data state includes deterministic one-click remediation CTAs (`Refresh Inbox` + section/dependency navigation such as `Open Channels` or `Open Configuration`).
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

Expected:

- Run-control button visibility and enablement is derived from daemon action metadata (`can_cancel`, `can_retry`, `can_requeue`) from `/v1/tasks/list`.
- Control actions use one shared confirmation contract before dispatch.
- Row action hierarchy keeps `View Run Detail` as the primary first action when available and renders `Cancel Run` with destructive styling.
- Disabled controls surface deterministic reason copy (missing run id, in-flight lock, or daemon-unavailable action state).
- In-flight state is per-run, blocks duplicate actions on that run, and presents explicit progress/status messaging.
- Successful/failed control outcomes update row/detail status copy consistently and do not desynchronize list vs detail surfaces.
- Task-submit draft continuity (sheet + `Goal`/`Details`/`Priority`/context overrides) persists workspace-scoped across section switches/relaunch and clears deterministically through destructive-confirmed `Reset Context`.
- `Simple` task summary/status wording remains action-first and plain-language in the default card body.
- `Advanced` mode restores operator wording for run-diagnostics context without removing default actions.

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

## 9) Channels Panel Checks

1. Select `Channels`.
2. Confirm cards exist for:
   - `App`
   - `Message`
   - `Voice`
3. Confirm each channel card is collapsed by default on first render.
4. Click `Refresh`.
5. Expand one card and verify daemon-declared diagnostics/remediation actions are functional when present.
6. In the `Message` channel card, if `Open Full Disk Access` (or equivalent `open_system_settings` remediation) is shown, click it and verify System Settings opens to Full Disk Access.
7. Click `Open Inspect Logs` on one card and verify navigation switches to `Inspect`.
8. In the `Message` card, verify mapped implementation/context fields include canonical logical channel IDs and mapped connector rollup details (health/reason/action summary) sourced from current daemon channel-connector bindings.
9. In one channel card `Configuration` section:
   - verify guided fields (when descriptors are present) render typed controls and metadata (`Required`, enum picker, bool toggle, secret/write-only/help text).
   - update one guided field value and click `Save Config`.
   - verify deterministic save status text is shown and the card refresh preserves the updated value.
10. Expand `Advanced` (default collapsed), add one raw key/value field that is not represented by guided descriptors, and verify it appears only in the advanced fallback list.
11. Click `Reset Draft` and verify draft values reset to last daemon-backed card values.
12. Click `Run Health Check` and verify inline result rendering includes:
   - status badge (`Healthy` or `Needs Attention`)
   - summary text
   - checked-at timestamp
   - structured detail rows when daemon returns detail payload.
13. In one channel card `Delivery Policy` section:
   - verify existing daemon policies render with default/non-default visibility.
   - click `Edit` on one policy and verify editor fields load (`endpoint pattern`, `primary channel`, `retry count`, `fallback channels`, `default toggle`).
   - update at least one field and click `Save Policy`; verify confirmation appears, then confirm and verify deterministic save status text plus refreshed policy data.
14. Click `New Policy`, enter values, and click `Save Policy`; verify confirmation appears, then confirm and verify the new or updated policy appears in the list.
15. Click `Reset Policy Draft` and verify draft values return to daemon-backed defaults.
16. In one channel card `Connector Mapping` section, verify mapped connector rows are visible with:
   - connector label + canonical connector id
   - enabled toggle
   - priority reorder controls (up/down)
   - capability summary text (when daemon returns capability metadata)
   - fallback policy label.
17. Toggle one mapping enabled state and move one mapping priority; verify draft-change behavior updates deterministically (`Save Mapping` becomes enabled and status copy reflects draft updates).
18. Click `Save Mapping` and verify deterministic save status text, then click `Refresh` and confirm mapping enabled/priority state persists.
19. In any workspace where `Message` has both `imessage` and `twilio` mappings enabled, change connector priority ordering in `Connector Mapping`, save, and verify the logical `Message` card summary/details track the mapping-priority-selected implementation after refresh.
20. In `Message` and `Voice` cards, verify constraint copy is explicit (`message -> imessage/twilio`, `voice -> twilio`) and capability-validation failures (if triggered) surface daemon error text inline in the mapping status row.
21. Collapse and expand each card.
22. If daemon returns no channel remediation actions for a card, verify the card shows deterministic unavailability copy, surfaces `Open Connectors`, and does not render generic fallback setup buttons.
23. If daemon returns zero channels, verify a deterministic empty-state view is shown instead of a blank panel.
24. Make at least one unsaved channel edit (config, delivery policy draft, or connector mapping) without clicking per-card save.
25. Verify Channels header shows `Unsaved changes` with enabled `Discard All` and `Save All` buttons, and edited card shows `Unsaved draft changes`.
26. Click `Discard All` and verify all channel drafts reset to daemon-backed values with deterministic section status copy.
27. Re-create at least one unsaved edit, click `Save All`, and verify all changed channel drafts persist with deterministic section status copy.
28. Expand at least one channel card, switch to another section, then return to `Channels` and verify expanded-card state restores for the active workspace.
29. Click `Refresh` and verify expanded/collapsed state remains stable after channel-status reload.
30. Trigger `Reset Context` from `Communications` or `Tasks`, return to `Channels`, and verify channel cards return to default-collapsed state for the active workspace.

Expected:

- Card status/detail data is populated from daemon `/v1/channels/status`.
- Channels render logical cards (`App`, `Message`, `Voice`) from canonical daemon channel IDs only.
- Logical channel mapping matrix remains stable across clean-install and upgrade-path payloads: `twilio` remains enabled under both `Message` and `Voice` when mapped.
- Guided channel configuration fields render from daemon descriptors (`config_field_descriptors`) with typed controls and metadata hints for required/enum/bool/secret/write-only/help states.
- Channel `Advanced` config disclosure is default-collapsed and retains raw key/value fallback editing only for non-descriptor keys.
- Configuration edits persist via daemon `/v1/channels/config/upsert` with deterministic save/reset status copy and no stale draft flicker after refresh.
- Channel health checks execute via daemon `/v1/channels/test` and render structured inline result summaries/details.
- Channel delivery-policy edits persist via daemon `/v1/comm/policy/set` and policy inventory refreshes via `/v1/comm/policy/list` with deterministic draft/save/reset behavior.
- Channel delivery-policy save actions require explicit confirmation before daemon mutation dispatch.
- Channel connector mapping inventory/actions are daemon-backed (`/v1/channels/mappings/list` and `/v1/channels/mappings/upsert`) with deterministic enable/disable and priority reorder behavior per logical channel.
- Mapping UX enforces explicit MVP constraints in copy (`app -> builtin.app`, `message -> imessage|twilio`, `voice -> twilio`) and surfaces daemon capability-validation failures without local override drift.
- Each card preserves status/details, shows mapped-connector rollups (health/reasons/actions) from daemon mapping bindings (not capability-name heuristics), and shows daemon-backed diagnostics actions when available.
- Dual-implementation `Message` primary-card selection follows mapping priority so connector-order changes produce deterministic logical-card summaries after refresh.
- No dead placeholder action buttons are shown in channel cards; unsupported actions are hidden and daemon-disabled actions remain visible with reason copy.
- Message-channel `open_system_settings` remediation routes to Full Disk Access when daemon emits iMessage ingest-remediation action metadata.
- Collapse/expand uses native disclosure affordances, defaults to collapsed on first render, and remains smooth/deterministic.
- Channel-card expansion state persists workspace-scoped across section switches/refresh/relaunch and clears under explicit continuity reset.
- On first section load with no cached channel inventory, channels body renders deterministic skeleton placeholders before transitioning to populated or empty/error states.
- Channels section exposes deterministic dirty-state affordances (`Unsaved changes`, `Discard All`, `Save All`) and card-level unsaved labels when drafts diverge.
- When channel inventory is empty, the panel shows explicit empty-state copy with status context.
- Channel empty-state copy includes direct remediation CTAs (`Refresh Channels` plus `Open Connectors`/`Open Configuration` based on setup readiness).

## 10) Connectors Panel Checks

1. Select `Connectors`.
2. Confirm connector cards exist and show permission status by default.
3. Verify UI shows one unified `Twilio` card (not separate SMS/Voice cards).
4. Verify `Twilio` card details include canonical connector identity (`twilio`), mapped connector IDs/capabilities, and summary reflects connector state.
5. Verify `iMessage` connector appears as its own connector-scoped card with canonical connector identity (`imessage`) in card details.
6. Confirm each connector card is collapsed by default on first render.
7. For a connector card where daemon diagnostics include a `request_permission` action and permission is missing:
   - Verify `Request Permission` is visible and enabled.
8. For a connector card where daemon diagnostics include a `request_permission` action and permission is granted:
   - Verify `Request Permission` stays visible but disabled.
9. For a connector card where daemon diagnostics do not include a `request_permission` action (for example cloudflared):
   - Verify no fallback `Request Permission` button is rendered and helper copy directs the user to `Open System Settings`/connector remediation actions.
10. Verify `Open System Settings` is visible and enabled on every card.
11. Validate connector remediation target matrix for `Open System Settings`:
   - `Mail` -> `Privacy & Security > Automation`
   - `Calendar` -> `Privacy & Security > Automation`
   - `Browser` -> `Privacy & Security > Automation`
   - `Finder` -> `Privacy & Security > Automation`
   - `Messages` (`Open System Settings`) -> `Privacy & Security > Automation`
   - `Messages` (`Open Full Disk Access`) -> `Privacy & Security > Full Disk Access`
12. Click `Open System Settings` on one connector card, then return to the app and verify the connector permission status re-check runs on app re-activation.
13. On at least one connector with non-granted permission, click `Request Permission`.
14. Verify the action is one-click (no second confirmation modal) and in-card request status messaging updates after completion.
15. For `Messages`, verify request-permission status copy references both Automation and Full Disk Access checks; for all permission-gated connectors, verify macOS permission attribution/prompt references `Personal Agent Daemon` (not the host process used to launch the app).
16. Expand one connector card and verify diagnostics/remediation action buttons are functional when shown (for example `Refresh Connector Status`, `Open Inspect Logs`, `Open System Settings`, and/or `Run Daemon Repair`).
17. In one connector card `Configuration` section:
   - verify guided fields (when descriptors are present) render typed controls and metadata (`Required`, enum picker, bool toggle, secret/write-only/help text).
   - update one guided field value and click `Save Config`.
   - verify deterministic save status text is shown and the card refresh preserves the updated value.
18. Expand `Advanced` (default collapsed), add one raw key/value field that is not represented by guided descriptors, and verify it appears only in the advanced fallback list.
19. Click `Reset Draft` and verify draft values reset to last daemon-backed card values.
20. Click `Run Health Check` and verify inline result rendering includes:
   - status badge (`Healthy` or `Needs Attention`)
   - summary text
   - checked-at timestamp
   - structured detail rows when daemon returns detail payload.
21. Click `Refresh` and verify connector status summaries update.
22. Collapse and expand cards.
23. If daemon returns zero connectors, verify a deterministic empty-state view is shown instead of a blank panel.
24. Make at least one unsaved connector configuration edit without clicking per-card save.
25. Verify Connectors header shows `Unsaved changes` with enabled `Discard All` and `Save All` buttons, and edited card shows `Unsaved draft changes`.
26. Click `Discard All` and verify connector drafts reset to daemon-backed values with deterministic section status copy.
27. Re-create at least one unsaved connector edit, click `Save All`, and verify changed connector drafts persist with deterministic section status copy.
28. Expand at least one connector card, switch to another section, then return to `Connectors` and verify expanded-card state restores for the active workspace.
29. Click `Refresh` and verify expanded/collapsed state remains stable after connector-status reload.
30. Trigger `Reset Context` from `Communications` or `Tasks`, return to `Connectors`, and verify connector cards return to default-collapsed state for the active workspace.

Expected:

- Connector status/detail data is populated from daemon `/v1/connectors/status`.
- Connector cards use canonical IDs (for example `twilio`, `imessage`) with deterministic primary-connector routing for actions/config.
- Guided connector configuration fields render from daemon descriptors (`config_field_descriptors`) with typed controls and metadata hints for required/enum/bool/secret/write-only/help states.
- Connector `Advanced` config disclosure is default-collapsed and retains raw key/value fallback editing only for non-descriptor keys.
- Configuration edits persist via daemon `/v1/connectors/config/upsert` with deterministic save/reset status copy and no stale draft flicker after refresh.
- Connector health checks execute via daemon `/v1/connectors/test` and render structured inline result summaries/details.
- Connector permission/health state reflects daemon metadata (`configuration.permission_state`, `configuration.status_reason`) with deterministic behavior.
- `Request Permission` is shown only when daemon declares a supported request action for that connector and is disabled when permission is already granted or daemon marks action unavailable.
- `Request Permission` routes through daemon API `/v1/connectors/permission/request` and performs daemon-managed native probe flows for supported connectors.
- `Request Permission` executes immediately (no extra confirmation modal before dispatch).
- `Messages` request-permission flow returns dual-permission guidance (Automation + Full Disk Access) with deterministic status copy.
- macOS permission attribution and/or privacy-pane entries for requested connector permissions reference `Personal Agent Daemon` instead of the parent host process.
- `Open System Settings` is always available and routes to connector-relevant privacy panes per matrix (`Mail/Calendar/Browser/Finder -> Automation`, `Messages -> Automation`) and `Messages` also exposes explicit `Open Full Disk Access` remediation for chat database access.
- Returning from System Settings triggers connector permission-state re-check on app activation, with deterministic status messaging.
- Diagnostics/remediation actions are daemon-backed when shown; dead placeholder actions are removed.
- Connector action ordering is deterministic with daemon-recommended actions prioritized and status-reason copy shown when available.
- Unsupported connector remediation actions are hidden; daemon-disabled actions stay visible with deterministic reason copy.
- Collapse/expand uses native disclosure affordances and defaults to collapsed on first render.
- Connector-card expansion state persists workspace-scoped across section switches/refresh/relaunch and clears under explicit continuity reset.
- On first section load with no cached connector inventory, connectors body renders deterministic skeleton placeholders before transitioning to populated or empty/error states.
- Connectors section exposes deterministic dirty-state affordances (`Unsaved changes`, `Discard All`, `Save All`) and card-level unsaved labels when drafts diverge.
- When connector inventory is empty, the panel shows explicit empty-state copy with status context.
- Connector empty-state copy includes direct remediation CTAs (`Refresh Connectors` plus `Open Channels`/`Open Configuration` based on setup readiness).

## 11) Configuration Panel Checks

1. Select `Configuration`.
2. Confirm a mode picker is visible with `Setup`, `Workspace`, `Integrations`, `Data`, and `Advanced`, and verify the default mode is `Setup`.
3. Confirm a `Setup Overview` section is visible with readiness badges (`Ready`, `Needs Attention`, optional `Checking`) and rows for `Daemon Lifecycle`, `Assistant Access Token`, `Provider Setup`, `Model Catalog`, and `Chat Route`.
4. In `Setup Overview`, verify quick actions are prioritized and compact:
   - `Fix Next` appears when onboarding has unresolved blockers.
   - one primary runtime remediation action is shown when needed.
   - `More Actions` appears for additional setup remediations.
   - `Refresh Checks` refreshes setup readiness/lifecycle/provider checks.
5. Click `Open Models` and verify navigation switches to `Models`.
6. Return to `Configuration` and click `Open Onboarding`; verify navigation switches to a non-configuration section where onboarding panel appears when setup is incomplete.
7. Return to `Configuration` and confirm runtime banner/detail copy does not flash stale degraded/disconnected state while lifecycle refresh is actively in flight.
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
16. Click `Test in Chat` and verify navigation switches to `Chat` with persona scope status guidance.
17. Return to `Configuration` and confirm `Delegation Rules` section is visible with:
   - `Grant Delegation` form (`From Actor`, `To Actor`, `Scope Type`, optional `Scope Key`, optional `Expires At`)
   - daemon-backed delegation inventory list (from/to/scope/status/created/expires)
   - per-rule `Revoke` action
   - `Refresh Delegation Rules` action with deterministic status text
18. In `Grant Delegation`, create one rule with different `From Actor`/`To Actor`, `Scope Type = EXECUTION`, and optional `Scope Key`; verify deterministic success/failure status copy and list refresh.
19. In delegation rule rows, verify `From` and `To` actors are display-name-first with explicit reveal/copy access to raw IDs.
20. Revoke the newly created rule (or any existing rule) and verify revoke confirmation appears before dispatch; confirm and verify in-flight indicator plus deterministic post-action status copy.
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
32. In a state where daemon control plane is reachable but one or more plugin workers fail, verify `Daemon Lifecycle` row uses degradation copy and includes `Open Channels` remediation; verify onboarding/setup summary does not show generic blocking `daemon setup needs repair` copy for this worker-only state.
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

Expected:

- Identity hub loads workspace/principal directory context from daemon identity APIs, including actor-handle mappings when available, and keeps deterministic fallback behavior when auth/context data is unavailable.
- Workspace/principal runtime summary rows reflect current app state (no hardcoded principal label).
- Workspace switching via Identity hub rehydrates app-shell workspace context and refreshes daemon-backed Configuration data without relaunch.
- Legacy workspace sentinel defaults (`default`) migrate to canonical `ws1` with persisted workspace-scoped UI context preserved.
- Identity refresh loading indicator is scoped to identity refresh actions only.
- Delegation section renders daemon-backed rule inventory with scope visibility (`scope_type`, optional `scope_key`) and deterministic loading/empty/error states.
- Delegation grant/revoke actions use daemon delegation APIs with deterministic validation and status copy (including self-delegation and scope-key constraints).
- Configuration mode navigation is built-in (`Setup`, `Workspace`, `Integrations`, `Data`, `Advanced`), defaults to `Setup`, and each mode hides unrelated sections to keep first-pass setup focused.
- Configuration `Workspace` mode includes daemon-backed chat persona policy controls with scope-specific load/save/reset/test behavior and default-collapsed advanced guardrails editing.
- Setup overview presents compact readiness badges and prioritized quick actions (`Fix Next`, primary remediation, `Refresh Checks`) while retaining deterministic one-click remediation behavior.
- Operator workflows in `Integrations`/`Data`/`Advanced` (`Capability Grants`, `Trust Receipts`, `Memory Browser`, `Retrieval Context Inspector`, `Runtime Supervisor Timeline`, `Panel Latency Budgets`) are visible as disclosure sections and default to collapsed until expanded by the user.
- Setup matrix deterministically reflects daemon lifecycle, token, provider setup, model catalog, and chat-route readiness and exposes direct remediation actions for unresolved checks.
- Worker-only plugin failures are classified as degraded runtime in setup matrix/taskbar context and route remediation to `Channels` diagnostics (`Open Channels`) rather than generic setup-repair blocking state; verify this is sourced from daemon lifecycle `health_classification` values.
- Configuration runtime banner/detail avoids stale degraded/disconnected flashes while lifecycle status refresh is in progress.
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
- `Install/Uninstall/Repair` actions are enabled/disabled from daemon lifecycle `controls` payload and are disabled while a lifecycle control call is in flight.
- Setup action status messaging is deterministic across in-progress, succeeded, and failed outcomes.
- High-impact configuration actions use explicit confirmation (`daemon lifecycle`, `retention`, and `revoke` flows), with short-lived undo affordances for reversible daemon start/stop actions.

## 12) Models Panel Checks

1. Select `Models`.
2. Confirm provider/model readiness section exists with:
   - provider cards for `OpenAI`, `Anthropic`, `Google`, and `Ollama`
   - each provider card supports expand/collapse and starts collapsed by default on first render
   - `Refresh Inventory` and `Run Checks` actions
   - route policy editor card with `Task Class`, `Provider`, and `Model` selectors plus `Save Route Policy`
   - per-provider setup controls (`Endpoint`, `Secret Name`, `API Key`, `Save Provider`, `Check`, `Reset Endpoint`)
   - chat model-route summary (or clear fallback status text)
   - provider-local model catalog entries showing model enablement + provider-ready state
   - provider-local routing-policy rows showing task-class -> provider/model mappings (or deterministic empty copy)
3. Expand/collapse at least two provider cards.
4. For one API-key provider (for example `OpenAI`):
   - set `Endpoint` to target endpoint.
   - set/update `Secret Name`.
   - enter `API Key` value.
   - click `Save Provider`.
5. Verify provider status copy transitions through deterministic save stages (keychain save, secret-ref registration, provider save) and then refreshes inventory.
6. Click per-card `Check` and verify readiness badge + status copy transitions to either `Healthy` or `Check Failed`.
7. Click `Reset Endpoint` and verify endpoint input resets to provider default, then click `Save Provider` to persist reset endpoint.
8. Verify API key field is write-only (value is not echoed back after save).
9. In one provider card, click `Discover` and verify discover status copy updates plus a `Discovered` subsection appears when models are returned.
10. In discovered rows:
   - click `Add to Catalog` for one model not already in catalog.
   - verify status copy confirms add/upsert and model appears in catalog list.
11. Use manual add:
   - enter a model key in `Add model key`.
   - click `Add Model`.
   - verify status copy confirms add and input clears.
12. In provider model rows, toggle at least one model from `Enabled` to `Disabled` (or vice versa) using per-row action button.
13. Verify per-row in-flight indicator and deterministic status copy (`enabled`/`disabled`) appears.
14. Click `Remove` on one catalog row and verify status copy confirms removal plus row disappears after refresh.
15. Verify chat route summary card refreshes after successful catalog/mutation actions without leaving the Models panel.
16. If chat route is unresolved, verify a `Route Readiness Checklist` card appears with rows for `Assistant Access Token`, `Daemon Reachability`, `Provider Setup`, `Model Catalog`, and `Chat Route`, each showing deterministic `Ready`/`Needs Attention`/`Checking` status badges.
17. In one provider model row, click `Set as Chat Route`; if the model is disabled, verify it is enabled first and then route policy is saved.
18. Verify `Set as Chat Route` updates deterministic status copy and the selected row shows a `Chat Route` badge.
19. Verify chat route summary card reflects the selected provider/model and updated source context without leaving Models.
20. In route policy editor, select `chat`, select a provider/model pair from available catalog options, then click `Save Route Policy`.
21. Verify route-policy confirmation appears with selected task-class/provider/model context; confirm save.
22. Verify route-policy save status copy reports success and the provider-local routing-policy row for `chat` updates immediately.
23. Click `Refresh Inventory` and then `Run Checks`.
24. Confirm a `Route Simulation + Explainability` card is visible with:
   - `Task Class` picker
   - optional `Principal Actor ID` input
   - `Use Active Principal`, `Pick Principal`, and `Clear Principal` helper actions
   - `Simulate Route`, `Explain Route`, and `Reset Output` actions
   - `Simulation Result` and `Explainability` result groups.
25. In the route-analysis card, set `Task Class` to one non-default class (for example `automation`) and optionally set a principal actor id, then click `Simulate Route`.
26. Verify simulation status text updates deterministically and `Simulation Result` renders:
   - selected provider/model/source summary
   - reason-code list
   - decision trace rows (`step`, `decision`, `reason_code`, optional provider/model/note)
   - fallback-chain rows with rank ordering and selected-candidate marker.
27. Click `Explain Route` for the same input and verify `Explainability` renders:
   - summary text
   - explanation bullet list
   - reason-code list
   - decision trace and fallback-chain context aligned to the explainability response.
28. Click `Reset Output` and verify both `Simulation Result` and `Explainability` return to deterministic empty guidance copy.
29. Make at least one unsaved provider setup edit (`Endpoint`, `Secret Name`, and/or write-only `API Key`) without clicking `Save Provider`.
30. Verify Models header shows `Unsaved changes` with enabled `Discard All` and `Save All` buttons, and edited provider card shows `Unsaved setup draft`.
31. Click `Discard All` and verify provider setup drafts reset to source values with deterministic section status copy.
32. Re-create at least one unsaved provider setup edit, click `Save All`, and verify changed provider drafts persist with deterministic section status copy.

Expected:

- Provider/model readiness reflects daemon inventory/check/resolve responses and remains readable when auth/config is missing.
- When provider inventory is empty, Models shows direct remediation CTAs (`Refresh Inventory` and setup-first actions such as `Open Configuration`/`Run Checks`) instead of blank space.
- Provider setup controls persist endpoint + secret reference using daemon mutation APIs.
- Provider-card actions (`Save`, `Check`, `Reset Endpoint`) are all functional and deterministic.
- Provider cards support daemon-backed model discovery plus explicit manual add/remove catalog management controls.
- When chat route is unresolved, Models surfaces a deterministic route-readiness checklist with actionable remediation buttons for unresolved setup blockers.
- Provider model rows expose one-click `Set as Chat Route` actions that can auto-enable disabled models before saving chat route policy.
- Route-policy editor only allows provider/model values that exist in current catalog entries and persists route updates through daemon model-select mutation.
- Route-policy saves require explicit confirmation before daemon mutation dispatch.
- Model rows expose daemon-backed `Enable`/`Disable` actions with deterministic in-flight/success/error status handling.
- Discover/add/remove actions produce deterministic provider-scoped status copy and keep provider-local catalog/discovered state coherent.
- Route-summary state updates after successful model toggle without requiring section re-entry.
- Provider readiness badges clearly differentiate `Setup Required`, `Configured`, `Healthy`, and `Check Failed`.
- Route simulation executes through daemon `/v1/models/route/simulate` with deterministic task-class/principal handling and renders reason-code decision/fallback context in-app.
- Route explainability executes through daemon `/v1/models/route/explain` and renders summary/explanations plus aligned reason-code decision/fallback context.
- API key values remain write-only in UI and are never rendered in panel copy.
- Model catalog/policy visibility reflects daemon `models.list` and `models.policy` responses with deterministic empty/error states and provider-local grouping.
- On first section load with no cached provider inventory, models provider area renders deterministic skeleton placeholders before transitioning to populated or empty/error states.
- Models section exposes deterministic dirty-state affordances (`Unsaved changes`, `Discard All`, `Save All`) and provider-level unsaved labels when setup drafts diverge.
- Runtime banner language remains consistent with other daemon-backed sections.

## 13) Automation + Runtime-State Cohesion Checks

1. Select `Automation` and click `Refresh`.
2. Verify no top-level banner reports `Failed to decode daemon payload`, and `Recent Trigger Evaluations` section is visible with deterministic loading/empty status copy.
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
18. Run `Simulate -> Run Schedule Simulation` and verify status text updates with a success/error summary.
19. Run `Simulate -> Run Comm Event Simulation` and verify status text updates with a success/error summary and seeded event metadata.
20. Verify fire-history section refreshes after simulation and recent rows include updated status/idempotency/timestamp linkage context.
21. Delete a trigger from its card and verify it is removed from list after refresh; repeat delete on the same trigger id path (if possible) and verify idempotent deletion messaging remains deterministic.
22. If workspace has zero triggers, verify deterministic empty-state copy is shown instead of a generic placeholder.
23. Select `Approvals` and verify daemon-backed pending/final approval rows render with risk/principal/decision metadata and route context (`task class`, provider/model, route source) when data exists (or deterministic empty/error copy when none exists).
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
- `Approvals` behaves as a daemon-backed inbox with deterministic pending/final grouping, summary-first card framing (`What happened`/`What needs action`/`What next`), collapsed-by-default details disclosure, route-context rendering, inline evidence disclosure (step input/output summaries + related artifacts/audit snippets), and guided in-app decision actions (`Action` selector, deterministic `Decision By`, approve phrase helper with `Use Required Phrase`, reject path without approval phrase requirement), plus direct task/list-detail/inspect drill-ins (including route-context fallback seeding when task/run linkage is absent), and persistent decision traceability/evidence expansion state after submit/refresh.
- `Approvals`, `Tasks`, `Communications`, and `Configuration` identity surfaces default to display-name-first labels; raw workspace/actor IDs are reachable only through explicit reveal/copy affordances.
- `Approvals` and `Tasks` both surface header-level active-filter indicators (count + concise summary tokens) when non-default filters/search are active, with one-click clear parity to toolbar reset actions.
- `Tasks` behaves as a daemon-backed task/run list with summary-first card framing (`What happened`/`What needs action`/`What next`), collapsed-by-default details disclosure, deterministic route-context metadata rendering, and a goal-first `New Task` submission flow (`Goal`, `Details`, `Priority`) with explicit `Override Context` controls (`task_class`, `requested_by`, `subject_principal`) plus correlation receipt feedback (`task_id`/`run_id`), row/detail drill-ins to related Inspect/Approvals context, run-detail drill-in (steps/artifacts/audit), filter/search controls, optional auto-refresh loop, and deterministic empty/no-match/error handling.
- Workflow-summary copy in `Simple` mode remains plain-language/action-first for Chat/Approvals/Tasks, and `Advanced` restores operator-level diagnostic phrasing without removing default workflow actions.
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

## 15) Models-to-Chat End-to-End Flow

1. Select `Models`.
2. In one API-key provider card (for example `OpenAI`), enter/update:
   - `Endpoint`
   - `Secret Name`
   - `API Key` (write-only)
3. Click `Save Provider` and verify deterministic success status copy.
4. In the same provider card, ensure at least one target chat model is `Enabled` (toggle if needed).
5. In `Route Policy Editor`, set:
   - `Task Class` = `chat`
   - `Provider` = chosen provider
   - `Model` = enabled target model
6. Click `Save Route Policy` and verify success status copy and updated route summary.
7. Navigate to `Chat`, enter a prompt, and click `Send`.
8. Verify chat turn completes successfully without route-remediation warning state.
9. Verify chat status copy reflects provider/model metadata from successful response.

Expected:

- User can complete provider setup, model enablement, and route selection entirely from `Models`.
- After setup, chat send succeeds without unresolved-route remediation errors.
- `Chat` and `Models` route context remain consistent for the selected provider/model pair.

## 16) Cleanup

- Quit the app from taskbar menu or `Cmd+Q`.
- If needed, remove temporary debug build artifacts manually from DerivedData.
