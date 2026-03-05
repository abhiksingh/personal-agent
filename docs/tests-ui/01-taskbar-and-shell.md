# UI Tests: Taskbar and App Shell

Source: [full guide](../tests-ui/full.md)

Regression anchor: run [Auth Scope + Rate Limit + Realtime Hardening (Manual-Only)](./11-auth-rate-limit-realtime-hardening.md) for deterministic fallback/recovery validation while automation is paused.

## 4) Taskbar Menu Checks

1. Click the menu bar icon.
2. Verify the menu bar glyph uses the branded Orbit Chat Core template symbol (not the generic SF Symbol stack glyph).
3. Verify status region shows daemon + app connection labels.
3a. Verify the runtime status area remains compact (no extra paragraph-level diagnostics copy under daemon/connection rows).
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
12. With the main window open, click `Close Window` and verify Dock/app-switcher presence disappears while the menu bar icon remains; then click `Open Window` and verify the main window and Dock presence return.
13. With daemon running, click `Quit` and verify open main windows close and app exits from Dock/menu bar.

Expected:

- Menu structure uses native grouped form sections and remains minimal/readable.
- Menu bar icon rendering uses the dedicated template asset and remains legible in light and dark appearances.
- Menu window remains compact (reduced vertical spacing and concise labels).
- Runtime status rows use neutral `Checking` copy before first lifecycle status load completes.
- Taskbar runtime status rows do not duplicate paragraph-level runtime detail text.
- Readiness hints prioritize the highest-impact setup gaps and cap visible rows for compactness.
- Token readiness status follows daemon lifecycle `control_auth` metadata (`configured|missing`) when lifecycle status is loaded.
- Plugin-worker-only degradation in daemon lifecycle status appears as a dedicated readiness issue with `Open Channels` remediation (instead of generic setup-repair guidance), driven by daemon `health_classification` (`core_runtime_state=ready`, `plugin_runtime_state=degraded`).
- Controls are enabled/disabled according to daemon lifecycle `controls` payload.
- High-impact daemon lifecycle controls show deterministic confirmation copy before dispatch, and `Start`/`Stop` confirmations surface a short-lived undo affordance.
- `Close Window` transitions app to menu-bar-only mode (no Dock/app-switcher icon) until `Open Window` is used again.
- `Quit` closes open windows and exits app; quit path should issue best-effort daemon stop before termination.

## 5) App Shell + Navigation Checks

1. In main window, confirm ready-state launch lands on `Home` and sidebar ordering is:
   - Top: `Configuration`
   - Workflow: `Home`, `Chat`, `Communications`, `Tasks`, `Approvals`, `Automation`
   - Advanced disclosure (collapsed by default): `Inspect`, `Channels`, `Connectors`, `Models`
2. In `Home`, verify a single primary next-action card is visible (`Next Best Action` or setup recovery equivalent) and that `Guided First Session` checklist rows are present with deterministic `Done`/action button states.
   - When first-session checklist is still incomplete, verify order is `Primary Next Action` -> `Guided First Session` -> `Quick Actions`.
   - When first-session checklist is complete, verify `Quick Actions` moves above `Guided First Session` (`Primary Next Action` -> `Quick Actions` -> `Guided First Session`).
   - Verify `Home` does not render a separate runtime diagnostics banner/paragraph; setup/runtime detail ownership stays in `Configuration > Setup`.
   - While setup is incomplete, verify `Finish Setup` card includes explicit first-run trust guidance (`Open` via right-click/control-click and `Open Anyway` path) plus deterministic `Open Security Settings` and `Retry Setup Checks` actions.
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
   - Verify onboarding-gated surfaces do not also show shell-level blocker/guidance ribbons above the setup wizard.
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
22. Click the toolbar `sparkles` (`Do`) button and verify `Command Palette` opens prefilled with `do ` query and intent-oriented `Do:` actions.
23. From that `do ` query, run `Do: Send an Email` and verify navigation switches to `Chat` with a prefilled starter draft.
24. Click the toolbar `command.square` button and verify `Command Palette` opens with grouped `Navigation`, `Diagnostics`, `Workflow`, and `Runtime` command sections.
25. In command palette search, enter `inspect`, run `Open Inspect`, and verify section navigation switches to `Inspect`.
26. Re-open command palette and run `Refresh Current Panel`; verify active section refreshes without navigation drift.
27. Re-open command palette, search `inspect`, press `Enter`, and verify the first enabled match executes without pointer interaction.
28. Search a natural-language query (for example `start service`) and verify command palette ranks the intent-matching command (`Start Daemon`) first when enabled.
29. Search `open` and verify ties resolve deterministically in catalog order (`Open Configuration`, then `Open Chat`, then other open-navigation commands).
30. Search an outcome query (for example `send email to finance`) and verify a `Do:` action (`Do: Send an Email`) ranks first.
31. In command palette, run two different commands (for example `Open Chat` then `Open Inspect`), close/re-open the palette, and verify a `Recent` section appears with most-recent-first ordering.
32. Put daemon controls into an unavailable state (for example no `start` permission or lifecycle control in-flight), open command palette, and verify disabled runtime commands remain visible with deterministic disabled-reason copy.
33. Use keyboard shortcuts:
   - `⌘2` -> `Chat`
   - `⌘7` -> `Inspect`
   - `⌘8` -> `Channels`
   - `⌘9` -> `Connectors`
   - `⌘0` -> `Models`
   - `⇧⌘D` -> `Do` entrypoint (`do ` query prefilled)
   - `⇧⌘P` -> command palette
   - `⇧⌘N` -> notification center
34. While advanced disclosure starts collapsed, trigger one advanced shortcut (`⌘7`, `⌘8`, `⌘9`, or `⌘0`) and verify the advanced disclosure auto-expands with the selected destination highlighted.
35. Verify `Diagnostics` menu shortcuts (`⌥⌘I`, `⌥⌘C`, `⌥⌘K`) switch to `Inspect`, `Channels`, and `Connectors`.
36. Verify runtime command shortcuts (`⌥⌘S`, `⌥⌘.`, `⌥⌘R`) are disabled when daemon controls are unavailable/in-flight and enabled when daemon lifecycle control flags allow action.
37. In the toolbar, open the density control and verify both `Simple` and `Advanced` options are visible, with `Simple` selected by default for new workspace state.
38. Switch density to `Advanced`, open `Chat`, and verify operator trace metadata remains available only behind an explicit `Details` disclosure in workflow-context areas (no inline short-ID fragments in the default card body).
39. Open command palette, search for `density`, and verify `Set Density: Simple` and `Set Density: Advanced` actions are listed with deterministic enablement based on current mode.
40. Run `Set Density: Simple` from command palette and verify the same metadata rows collapse back to simplified user-facing copy.
41. From `Chat`, trigger a realtime fallback condition (capacity rejection or stale/disconnect), then verify sidebar/footer and runtime badges classify app connection as `Degraded` while fallback chat completion still succeeds.
42. Use `Chat > Actions > Retry Realtime Stream`, verify reconnect success, then confirm runtime connection state returns to `Connected`.

Expected:

- Sidebar uses native split-view/list behavior and collapses/expands without layout breakage.
- Only one sidebar toggle affordance is visible in the toolbar.
- Toolbar includes a global `Do` entrypoint (`sparkles`) that opens command palette in intent mode.
- Main panel expands when sidebar is collapsed.
- Window title tracks selected section.
- Ready-state launch lands on `Home` first and shows one deterministic primary next action plus guided first-session milestone rows.
- `Home` ordering remains action-first and progressive: incomplete setup/session keeps checklist before quick actions, while completed first-session state promotes quick actions directly under primary action.
- While setup is incomplete, `Home` setup recovery card includes explicit unsigned-build trust remediation guidance and deterministic `Open Security Settings` + `Retry Setup Checks` actions before daemon troubleshooting guidance.
- In `Advanced` density mode, `Home` shows a `First-Success Funnel Diagnostics` card with aggregate completion metrics and per-milestone completion evidence (`source` + timestamp or pending state).
- While setup readiness is incomplete, workflow panels do not show the guided-session ribbon.
- Once setup readiness is complete and first-session milestones remain incomplete, workflow panels show a guided-session ribbon with `Step x of y`, deterministic next-step routing, and `Open Home Checklist`.
- Sidebar advanced destinations (`Inspect`, `Channels`, `Connectors`, `Models`) are progressively disclosed, collapsed by default, and auto-expand when selected via keyboard shortcut/command/deep-link navigation.
- `Communications`, `Tasks`, `Approvals`, and `Inspect` share consistent panel scaffold and filter-bar chrome for header/filter/loading/empty/action transitions.
- Sidebar rows for `Communications`, `Tasks`, `Approvals`, and `Inspect` display compact active-filter count badges when persisted non-default filters are active in the current workspace.
- Sidebar filter badges expose concise persisted-filter summary help text on hover.
- Sidebar runtime footer shows neutral `Checking` labels until first daemon lifecycle load resolves.
- `Home` keeps checklist/next-action guidance as the setup owner and does not duplicate runtime diagnostics paragraphs.
- Onboarding panel appears for non-setup workflow sections until token/daemon/provider/model-catalog/chat-route/channel-connector mapping criteria are met, while `Configuration`, `Channels`, `Connectors`, and `Models` remain directly accessible.
- Onboarding wizard always surfaces one deterministic highest-priority unresolved blocker in `Fix Next`, and the primary CTA performs one-click remediation (deep-link or lifecycle control) without extra navigation steps.
- Onboarding wizard keeps full checklist visibility available behind an optional collapsed disclosure while preserving single-path default guidance.
- Onboarding-gated sections do not stack extra shell-level blocker/guidance ribbons above the setup wizard.
- Non-Configuration setup-accessible panels (`Channels`, `Connectors`, `Models`) show a compact `Current Blocker` ribbon while setup readiness is incomplete.
- `Current Blocker` ribbon includes one-click `Fix Next` plus one contextual secondary action (`Open Models`, `Open Configuration`, `Open Channels`, or `Refresh`) and does not block normal panel interaction.
- Cross-section navigation from `Channels`, `Connectors`, and `Models` shows deterministic discard confirmation when unsaved drafts exist.
- Major panel status updates surface as non-blocking toasts and persist to a global notification center activity log.
- Notification center groups rows by intent (`Needs Attention`, `Workflow Updates`, `Runtime and Setup`, `Diagnostics`, `General`), keeps newest-first ordering inside each group, and preserves deterministic read/clear plus source/query filtering controls.
- Notification rows expose deterministic next-action affordances (for example `Open Channels`, `Open Tasks`) that mark rows read and navigate to the mapped destination panel.
- Command palette exposes searchable grouped commands for section navigation, diagnostics, workflow actions, and runtime controls using one shared enablement contract.
- Command palette ranks natural-language query intent matches deterministically, with stable tie-break ordering for equal-score matches.
- Command palette executes first enabled search match on `Enter`, surfaces deterministic disabled reasons for unavailable actions, and prioritizes recently used commands in `Recent`.
- Outcome-style intent queries (for example `send email`, `create task`, `review approvals`, `inspect issue`) prioritize `Do:` commands and route into canonical owners (`Chat`, `Tasks`, `Approvals`, `Inspect`) with seeded context.
- Keyboard shortcuts for section switching, command palette, notification center, diagnostics destinations, and runtime controls dispatch deterministic actions without bypassing guard rails.
- Workflow drill-ins (`Open Related ...`) present a destination context ribbon with source panel chips and a deterministic `Back to ...` return action.
- Toolbar includes a global information-density control with `Simple` default.
- Density mode command-palette actions remain in parity with toolbar state and expose deterministic enabled/disabled behavior.
- `Simple` mode prioritizes user-readable summaries while `Advanced` restores full operator metadata visibility.
- Runtime shell status reflects realtime fallback resilience transitions (`Connected` -> `Degraded` on websocket fallback, then back to `Connected` on successful realtime retry).
- User-facing status paths in `Chat`, `Models`, `Channels`, `Connectors`, `Automation`, `Approvals`, and `Tasks` show remediation-first guidance and do not surface raw daemon transport/decode strings (for example `Failed to decode daemon payload` or `Daemon request failed (...)`) in default panel states.
