# PersonalAgent UI Spec (Canonical UI Surface)

This document is the single source of truth for product UI decisions.
It complements `spec.md` and `data-model.md`; if a cross-cutting product contract changes, update the canonical docs in the same change.

## 1) Product Definition

Build a macOS-first SwiftUI client (menu bar app + main window) that makes PersonalAgent usable end-to-end for chat, approvals, communication workflows, automation control, settings, and runtime inspection.

## 2) MVP UI Scope (Working Baseline)

- Platform: macOS-first SwiftUI app.
- App project structure: hybrid (Xcode app host + Swift package modules).
- UX model: taskbar menu + full app window for primary workflows.
- Runtime model: thin client over daemon APIs (`/v1/*`) and realtime stream.
- Current execution mode for UI workstream: placeholder-first visual iteration, then API wiring as daemon endpoints stabilize.
- Primary main-window sections in MVP:
  - Configuration
  - Home
  - Chat
  - Communications
  - Automation
  - Approvals
  - Tasks
  - Inspect
  - Channels
  - Connectors
  - Models

## 3) Core UI Principles

- Thin client: UI does not own business side effects; daemon does.
- Explainability: every run/step/approval has visible rationale and status.
- Realtime first: streaming output and progress updates are default behavior.
- Safety clarity: destructive actions are explicit, interruptible, and auditable.
- Multi-principal clarity: requested-by / subject / acting-as are visible when relevant.
- Identity readability first: user-facing identity labels default to display-name-first wording; raw IDs are available only through explicit reveal/copy affordances.
- Predictable recovery: clear offline/degraded/retry states for daemon and transport failures.
- Problem-details UX mapping: default workflow panels (`Chat`, `Models`, `Channels`, `Connectors`, `Tasks`, `Approvals`, `Automation`) must map daemon typed problem metadata (`error.code`, `error.details`) into remediation-first copy and avoid raw transport/decode dumps, while `Inspect` retains raw diagnostics.
- Typed `auth_scope` and `rate_limit_exceeded` problem states must use one shared remediation card contract across workflow panels with deterministic actions (`Open Configuration`, `Retry`, `Open Inspect`) and explicit disabled/in-flight reason copy.
- Typed payload contract decoding: realtime-event payloads, chat turn-item metadata, and ui-status configuration/test-operation details must be decoded via typed models with explicit extension buckets (not ad hoc dictionary-key parsing in panel logic).
- UX writing consistency: user-facing default copy should use plain-language, action-first wording (`message`, `conversation`, `task`, `approval`) and avoid transport-path jargon or identifier-heavy phrasing outside explicit advanced/details disclosures.
- Runtime-state cohesion: daemon/auth connectivity notices should use consistent language and severity mapping across core panels.
- Information ownership: each high-value status class (runtime, setup, route, approval, continuity) has one canonical detailed owner surface; all other surfaces must show summary + remediation only and avoid reprinting the same detail text.
- Loading continuity: before first data load resolves, runtime status and panel bodies should render stable `Checking`/skeleton placeholders rather than flashing degraded or empty-state copy.
- Distribution trust clarity: when running unsigned/ad-hoc local/internal builds, setup surfaces must explicitly explain expected Gatekeeper override steps and avoid framing host trust blocks as daemon runtime defects.
- Perceived-performance budgets: panel `initial render`, `refresh`, and `transition` latency samples should be captured with deterministic per-panel budgets and explicit regression summaries visible in advanced diagnostics.
- Keyboard-first operability: primary navigation and high-frequency controls should be reachable via deterministic shortcuts and a searchable command palette.
- Workspace continuity: workspace-scoped context should persist across section switches/relaunch for Communications/Tasks/Approvals/Inspect filters plus in-progress work (`Communications` compose drafts, triage state, `Tasks` submit drafts, and `Channels`/`Connectors` expanded cards) until explicitly reset.
- Workspace canonicalization: UI workspace selection and workspace-scoped persisted state keys must use canonical workspace IDs exactly as provided by daemon context APIs (no legacy alias rewriting).
- Filter-state visibility: persisted panel filters should be visibly summarized near panel headers with one-click clear to avoid hidden stale context.
- macOS-native product language: visual behavior should closely match Apple system-app conventions for macOS Tahoe (spacing, typography hierarchy, controls, motion, and interaction patterns).
- Native-component-first implementation: prefer built-in SwiftUI/AppKit controls and platform styles; add custom wrappers only when required for product-specific behavior.
- Minimal default chrome: keep taskbar and navigation surfaces simple, calm, and high-signal.
- Collapsible-card default policy: all collapsible cards/disclosure surfaces default to collapsed unless an explicit, documented exception is approved.
- Information density control: provide a workspace-scoped `Simple` (default) vs `Advanced` mode so external-user default views avoid low-value internal IDs/trace fragments while operator detail remains one click away.
- Visual completeness before backend completeness: panel layouts and control affordances should look production-ready even when some actions are temporarily non-functional.
- Interaction fidelity: primary controls should provide clear hover/pressed/disabled feedback and respect reduced-motion accessibility preferences.
- Success-state reinforcement: high-frequency workflow controls (`Send`, `Approve/Reject`, `Save`, task run controls) should provide subtle deterministic success feedback (for example symbol effects + compact success badges) without introducing heavy animation.
- Empty-state utility: major panel empty/error states should include direct remediation CTAs (for example `Open Models`, `Refresh`, `Open Configuration`) with deterministic visibility rules.

### 3.1 UI/UX Execution Playbook (Mandatory for UI Tasks)

This playbook defines the execution standard for all UI changes.
Any UI task implementation must satisfy these checks before moving to `review` or `done`.

1. Information architecture (IA)
   - Place new functionality in an existing canonical section first (`Configuration`, `Home`, `Chat`, `Communications`, `Tasks`, `Approvals`, `Automation`, `Inspect`, `Channels`, `Connectors`, `Models`).
   - Add new top-level navigation destinations only when no existing section can contain the workflow without coupling unrelated actions.
   - Keep progressive disclosure boundaries explicit: default surfaces stay scan-first; advanced/operator detail goes in collapsed-by-default disclosures.
   - Preserve deep-link and return-path consistency (`Open Related ...` + `Back to ...`) when adding cross-panel actions.
2. Action affordance hierarchy
   - Each panel/card/sheet exposes one clear primary action, with secondary/destructive actions visually and spatially distinct.
   - Row-level workflow action bars should place the primary action first, keep secondary actions grouped after it, and render destructive actions with distinct destructive styling.
   - High-impact actions use shared confirmation patterns before dispatch and show deterministic in-flight/complete/failure states.
   - Disabled controls must show deterministic reason copy sourced from runtime/state contracts (no silent disable).
3. UX copy contract
   - `Simple` mode copy is action-first, plain-language, and user-outcome oriented.
   - Technical/internal identifiers (raw IDs, transport names, trace fragments) must stay in explicit advanced/details disclosures.
   - Error and empty states must include actionable remediation language with direct CTA labels (`Fix Next`, `Open ...`, `Refresh`).
4. Density-mode behavior
   - `Simple` is default per workspace and must suppress low-value operator metadata in primary reading paths.
   - `Advanced` restores full operator/debug details without requiring alternate endpoints or hidden controls.
   - New UI elements must declare what is shown in `Simple` vs `Advanced` when metadata-rich content is introduced.
5. Interaction patterns
   - Use native SwiftUI/AppKit controls and established repo patterns before introducing custom components.
   - Preserve deterministic keyboard and command-palette parity for high-frequency actions where applicable.
   - Provide complete state handling for loading, empty, degraded, in-flight, and success/failure transitions.
6. Verification evidence
   - Update `docs/tests-ui.md` (index) and relevant `docs/tests-ui/*.md` panel files for user-testable UI flow changes.
   - Update `tools/scripts/run_tests_ui.sh` when manual UI test steps change.
   - Ensure the task plan includes before/after UX intent and validation evidence for non-trivial UI changes.

## 4) Information Architecture

### 4.1 Global App Shell

1. Taskbar menu for daemon and app lifecycle actions.
2. Main window with sidebar navigation and workspace content.
3. Sidebar footer status for daemon health and client connection state.
4. First-run onboarding gate for workflow panels when daemon/token readiness is not yet complete.
5. Global notification center for non-blocking action outcomes and cross-panel activity visibility.
6. Global command palette for searchable workflow/diagnostic/runtime actions plus concrete object search (`Tasks`, `Approvals`, `Threads`, `Connectors`, `Models`).
7. Global information-density mode control in toolbar and command palette.

### 4.2 Taskbar Menu Contract

1. Must show daemon process status.
2. Must provide primary day-to-day daemon controls:
   - start
   - stop
   - optional restart
3. Must provide main app window open/close action.
4. Must provide app exit action.
5. Must remain minimal and macOS-native, with no dense or non-system-styled controls in MVP.
6. Taskbar icon should use a dedicated app-provided template glyph asset (not a generic SF Symbol) so menu bar identity remains consistent with app branding in light/dark appearances.
7. Daemon lifecycle actions should be initiated directly from UI controls; when privileged operations are required, the app should use native macOS authorization prompts and display clear in-menu progress/failure states.
8. `Install/Uninstall` daemon actions should live in `Configuration > Advanced` as the default location.
9. Taskbar menu may surface contextual setup/repair actions (for example install/reinstall) only when daemon setup is missing, broken, or otherwise not runnable.
10. Taskbar menu should include compact readiness hints for auth-token/provider/model setup gaps with direct remediation actions (`Open Configuration`, `Open Models`, contextual daemon `Start`/`Install`/`Repair`) without adding dense chrome.
11. When daemon control plane is healthy but one or more plugin workers fail, taskbar readiness should classify this as plugin degradation and route remediation to `Channels` diagnostics instead of generic setup-repair copy.
12. High-impact daemon lifecycle actions from taskbar controls (`Start`/`Stop`/`Restart`) must require explicit confirmation before request dispatch; `Start`/`Stop` confirmations should surface a short-lived in-app undo affordance.
13. Closing the main window (taskbar action or window chrome close) should transition the app to menu-bar-only mode (`.accessory`) so Dock/app-switcher presence is removed until reopened.
14. Taskbar `Quit` should close any open app window and issue a best-effort daemon stop request before app termination.

### 4.3 Main Window Layout Contract

1. Two-area split layout:
   - left sidebar for navigation
   - right main panel for active section content
2. Main window shell should use native macOS split-view primitives (`NavigationSplitView` + sidebar `List`) by default.
3. Sidebar supports collapse.
4. Main panel expands to full window width when sidebar is collapsed.
5. Sidebar ordering:
   - top: `Configuration`
   - workflow: `Home`, `Chat`, `Communications`, `Tasks`, `Approvals`, `Automation`
   - advanced (progressively disclosed, collapsed by default): `Inspect`, `Channels`, `Connectors`, `Models`
   - advanced group auto-expands when an advanced destination is selected by shortcut, command palette, or deep-link action.
   - bottom footer: daemon status + app-to-daemon connection status
   - while lifecycle bootstrap is unresolved, footer status should show neutral `Checking` labels for daemon and connection.
6. Main-window toolbar should include a global outcome-first `Do` affordance (`sparkles` icon) that opens command palette in intent mode (prefilled `do ` query).
7. Main-window toolbar should include a notification-center affordance (`bell` icon) that opens a searchable activity sheet.
8. Main-window toolbar should include a command-palette affordance (`command.square`) that opens searchable action execution.
9. Major workflow panels (`Communications`, `Tasks`, `Approvals`, `Inspect`) should reuse one shared scaffold order (`Header`, `Active Filters`, `Runtime Banner`, `Filter Bar`, `Divider`, `Content`) and shared filter-bar card styling so loading/empty/action transitions remain visually consistent across sections; explicit modal exceptions (for example `Inspect` `Gallery` mode) may hide filter controls when that mode is non-filterable by design.
10. In major workflow panel filter bars, `Clear Filters` is the canonical filter-reset label, and `Reset Context` actions must be presented as destructive and require explicit confirmation before clearing workspace continuity state.
11. App shell landmarks and icon-only controls must expose deterministic VoiceOver labels/hints (`Sidebar`, panel landmark labels, and drill-in dismiss controls) to keep core workflow navigation unambiguous.

### 4.4 Primary Navigation

1. `Configuration`: workspace/principal/channel/retention/transport settings and daemon lifecycle controls.
2. `Home`: default workflow landing panel with one primary next action, first-session milestone guidance, and direct workflow remediation routing.
3. `Chat`: conversation-driven agent interaction with streaming responses.
4. `Communications`: inbox timeline for threads, message events, and call-session context.
5. `Tasks`: task/run list and details.
6. `Approvals`: approval inbox and decision actions.
7. `Automation`: trigger and directive management.
8. `Inspect`: traces, audit logs, plugin lifecycle, diagnostics.
9. `Channels`: per-channel cards with status and configuration.
10. `Connectors`: per-connector cards with status, configuration, and permission controls.
11. `Models`: provider readiness, model route resolution, catalog visibility, and routing-policy visibility.
12. Cross-panel deep-link actions should route through one navigation path that triggers exactly one section refresh even when the destination section is already selected.
13. Cross-section navigation from editable draft sections (`Channels`, `Connectors`, `Models`) must prompt before losing unsaved drafts and support explicit `Stay` or `Discard Changes` outcomes.
14. Workflow drill-ins between `Communications`, `Tasks`, `Approvals`, `Automation`, and `Inspect` should show consistent `Open Related ...` action labels, preserve source context chips at the destination, and provide one-click `Back to ...` return behavior.

### 4.5 Keyboard and Command Palette

1. App must expose deterministic keyboard shortcuts for section switching (`⌘1...⌘0`), refresh current panel (`⌘R`), notification center (`⇧⌘N`), command palette (`⇧⌘P`), and outcome-first `Do` entrypoint (`⇧⌘D`).
   - Section-switch shortcuts must remain active even when advanced sidebar destinations are currently collapsed.
2. App must expose diagnostics destination shortcuts (`⌥⌘I`, `⌥⌘C`, `⌥⌘K`) for `Inspect`, `Channels`, and `Connectors`.
3. Command palette should use one centralized command contract for action enablement/disablement and dispatch parity with menu shortcuts.
4. Runtime command shortcuts (`Start`, `Stop`, `Restart`) must respect daemon control availability and in-flight disablement.
5. Command palette search submit (`Enter`) should execute the first enabled match without requiring pointer selection.
6. Command palette should prioritize recently executed actions in a dedicated `Recent` section and keep deterministic ordering for non-recent items.
7. Disabled command rows should surface deterministic reason copy sourced from the centralized command-enable contract.
8. Opening Command Palette should place keyboard focus in its search field, and workflow panel search bars should expose deterministic accessibility labels/identifiers for keyboard-first traversal and UI automation parity.
9. `Do` outcome actions must route to existing canonical workflow owners (for example `Chat`, `Tasks`, `Approvals`, `Inspect`) and may seed context/drafts/status, but must not create parallel non-canonical execution surfaces.

### 4.6 Information Density Mode

1. App shell toolbar must provide a built-in control to switch information density between `Simple` and `Advanced`.
2. Command palette must expose matching density commands (`Set Density: Simple`, `Set Density: Advanced`) with deterministic enablement/disabled reasons.
3. `Simple` is the default mode for each workspace and should suppress low-value internal metadata in primary reading paths (for example raw IDs, route-source internals, trace/debug fragments) while preserving core actions.
4. `Advanced` must restore full operator detail for troubleshooting workflows without requiring additional navigation.
5. Density preference must persist per workspace and reapply on workspace switch/relaunch.

### 4.7 Information Ownership and De-Duplication Contract

1. Runtime lifecycle detail ownership:
   - Canonical detailed owner: `Configuration > Setup`.
   - Summary-only surfaces: sidebar footer, taskbar runtime rows, panel runtime banners.
   - Summary surfaces must not duplicate the same paragraph-level runtime detail already shown in the panel header/body.
2. Setup readiness ownership:
   - Canonical detailed owner: `Configuration > Setup Matrix`.
   - Summary-only surfaces: `Current Blocker` ribbon, `Home` primary setup card, taskbar readiness hints, onboarding progress.
   - Summary surfaces should show one compact blocker summary + one primary remediation CTA and must not repeat checklist-level detail paragraphs.
3. Guided first-session ownership:
   - Canonical owner: `Home` checklist and next-step card.
   - Secondary ribbons outside `Home` should only show compact step progress + deep-link CTA, without duplicating checklist detail text, and should appear only after setup readiness is complete.
4. Chat route and workflow context ownership:
   - Canonical owner: `Chat` `Effective Workflow Context` card.
   - Header/composer/explainability sections should avoid repeating the same provider/model/source strings unless needed for a distinct action.
5. Approval decision ownership:
   - Canonical owner: `Approvals` pending-card decision form and evidence disclosures.
   - `Chat` approval timeline rows should stay compact and decision-oriented (status + handoff/quick action) without duplicating full decision form fields.
6. Communications continuity metadata ownership:
   - Canonical row-level owner: continuity/thread/event primary card row for actionable state.
   - Non-essential duplicated metadata must live in one collapsed `Details` disclosure.
7. Empty-state status ownership:
   - If panel header subtitle already renders the same status string, empty-state status text should be suppressed and only unique remediation copy should remain visible.

## 5) Screen Contracts (MVP)

### 5.1 Home

- `Home` is the default landing panel whenever onboarding-gate prerequisites are satisfied.
- `Home` must surface one deterministic primary next action (`Next Best Action`) and route it through existing section navigation contracts.
- `Home` must render a guided first-session checklist with these milestones:
  - send first chat message
  - send one communication
  - submit one task
  - review approvals
- `Home` section order should remain progressive-disclosure and action-first:
  - always render `Primary Next Action` first
  - when setup is complete and first-session checklist is complete, show `Quick Actions` immediately after the primary card
  - otherwise keep `Guided First Session` before `Quick Actions` so checklist guidance remains primary
  - keep diagnostics (`First-Success Funnel Diagnostics`) subordinate in `Advanced` density mode
- Each incomplete milestone must expose an explicit CTA button that navigates to the owning workflow section.
- Completed milestones must show deterministic completion state (`Done`) with no ambiguous copy.
- Outside `Home`, once setup readiness is complete, incomplete first-session milestones must remain discoverable through a compact guided-session ribbon in workflow panels (`Chat`, `Communications`, `Tasks`, `Approvals`, `Automation`, `Inspect`) showing `Step x of y`, one deterministic next-step CTA, and an `Open Home Checklist` fallback action.

### 5.2 Configuration

- Must support workspace/principal context selection.
- Configuration should use built-in mode navigation with user-facing groups (`Setup`, `Workspace`, `Integrations`, `Data`, `Advanced`) and default to `Setup`.
- Within `Configuration` modes, section order should remain action-first:
  - `Setup`: `Setup Overview` -> `Assistant Access Token` -> collapsed `Runtime Details`
  - `Workspace`: `Identity Hub` -> `Chat Persona Policy` -> collapsed `Delegation Rules` -> collapsed `Identity Devices and Sessions`
  - `Advanced`: collapsed `Daemon Lifecycle Controls` -> collapsed `Runtime Supervisor Timeline` -> collapsed `Panel Latency Budgets`
- Operator-heavy Configuration workflows (`Runtime Supervisor Timeline`, `Capability Grants`, `Trust Receipts`, `Memory Browser`, `Retrieval Context Inspector`) should render in non-default modes as disclosure sections that default to collapsed.
- `Configuration > Advanced` should include a collapsed-by-default `Panel Latency Budgets` diagnostics disclosure with budget/regression summary, latest per-panel samples, and clear/capture controls.
- Must include an Identity hub in `Configuration` with:
  - workspace picker
  - active context metadata (workspace/principal source and resolution)
  - workspace directory inventory
  - principal directory inventory with actor-handle mappings
- Configuration identity surfaces should render workspace/principal labels display-name-first and keep raw IDs behind explicit reveal/copy affordances.
- Configuration Identity hub should include daemon-backed device and session inventory controls (`POST /v1/identity/devices/list`, `POST /v1/identity/sessions/list`) with filterable workspace-scoped status metadata and deterministic loading/empty/error copy.
- Configuration Identity hub should include per-session revoke controls (`POST /v1/identity/sessions/revoke`) with explicit in-flight, idempotent, and post-refresh status handling.
- Workspace switching in the Identity hub must rehydrate app-shell context (workspace label and daemon-backed section refreshes) without requiring app relaunch.
- Must include a Delegation management surface in `Configuration` with:
  - daemon-backed rule inventory (`from`/`to`, scope type/key, status, created/expires metadata)
  - scoped grant creation (`EXECUTION`/`APPROVAL`/`ALL`) with deterministic validation copy
  - revoke action per rule with deterministic in-flight/result status handling
- `Configuration > Workspace` must include a chat persona policy editor backed by daemon APIs (`POST /v1/chat/persona/get`, `POST /v1/chat/persona/set`) with:
  - scope selector (`Workspace Default`, `Principal`, `Channel`, `Principal + Channel`)
  - principal/channel pickers shown only when the selected scope requires them
  - required simple style-prompt editor
  - default-collapsed advanced guardrails editor (one guardrail per line)
  - deterministic `Refresh Scope`, `Save Policy`, `Reset Draft`, and `Test in Chat` actions with explicit disabled reasons/status copy
  - explicit `Test in Chat` response-shaping context badges (`Test Channel`, `Profile`) and mismatch guidance when persona scope channel differs from app-channel chat validation path
- Must preserve write-only secret behavior from UI surfaces.
- Must expose retention and context-budget controls through daemon-backed APIs.
- Principal option inventory may use daemon delegation APIs (`POST /v1/delegation/list`) to surface known acting-as identities, while preserving a `default` fallback option.
- Retention operations should use daemon retention APIs (`POST /v1/retention/purge`, `POST /v1/retention/compact-memory`) and show deterministic success/error summaries.
- Context budget operations should use daemon context APIs (`POST /v1/context/samples`, `POST /v1/context/tune`) with explicit task-class and sample-limit controls.
- Must include an `Advanced` area for daemon install/uninstall and related maintenance actions.
- Must read daemon lifecycle/setup state from daemon lifecycle status APIs; `start`/`stop`/`restart`/`uninstall` actions use daemon lifecycle control APIs, while `install`/`repair` run from bundled helper assets in app-host setup flow.
- Current auth-token strategy for UI/daemon integration starts with assistant access token usage.
- First-run onboarding should provide explicit completion checks for access token setup, daemon reachability, provider setup, model catalog enablement, chat-route readiness, and channel/connector mapping readiness before non-setup workflow panels are fully available; `Configuration`, `Channels`, `Connectors`, and `Models` must remain directly accessible during onboarding.
- Onboarding should use a single-path setup wizard that always highlights one deterministic highest-priority unresolved blocker (`Fix Next`), one primary remediation action, clear progress (`x of y checks ready`), and a completion handoff into primary workflow sections.
- Onboarding should keep setup guidance summary-first; full setup diagnostics/check matrices remain canonical in `Configuration > Setup`.
- In the current non-Developer-ID distribution path, onboarding/setup guidance must include explicit host trust remediation copy for blocked first launch (`Open Anyway`/right-click `Open`) before daemon lifecycle troubleshooting steps.
- Non-Configuration panels should surface a compact `Current Blocker` ribbon whenever setup readiness is incomplete, with a primary `Fix Next` action plus one contextual secondary action (`Open Models`, `Open Configuration`, `Open Channels`, or `Refresh`) without blocking normal panel interaction in setup-accessible sections; ribbon copy should remain summary-only and defer detailed diagnostics to setup owners.
- Configuration should include an explicit setup matrix for daemon lifecycle, access token, provider setup, model catalog, chat-route readiness, and production-profile transport security readiness with direct remediation actions (`Start`/`Install`/`Repair` daemon lifecycle plus `Open Models`, `Open Onboarding`, and transport-security remediation destinations as needed).
- Setup-matrix `Chat Route` readiness should render resolved provider/model/source context when available and keep deterministic `Open Models` remediation when route resolution is missing or blocked.
- `Setup` mode should present a compact readiness overview (ready/attention/checking counts), a prioritized quick-action row (`Fix Next`, primary runtime remediation, `Refresh Checks`), and overflow setup actions under built-in menu controls.
- `Setup` mode should keep `Setup Matrix Details` collapsed by default and keep `Runtime Details` behind a collapsed disclosure so first-pass setup remains outcome-first.
- `Setup` quick-action guidance should render deterministic disabled-reason copy whenever `Fix Next` or primary remediation actions are temporarily unavailable.
- `Workspace` mode should keep `Identity Devices and Sessions` and `Delegation Rules` in collapsed-by-default disclosures while preserving one-click access.
- `Advanced` mode should keep `Daemon Lifecycle Controls` behind a collapsed disclosure by default, alongside existing collapsed diagnostics disclosures.
- On Configuration open, runtime warning banners/details should not briefly show stale degraded/disconnected content while lifecycle refresh is still in-flight.
- Plugin-worker-only failures (worker `failed > 0` with daemon database/control plane ready) should be shown as degraded runtime state with direct `Open Channels` diagnostics remediation, not as generic setup-repair blocking guidance.
- Configuration should include capability-grant governance controls backed by daemon capability-grant APIs (`POST /v1/delegation/capability-grants/list`, `POST /v1/delegation/capability-grants/upsert`) with actor/capability/status filters plus create/update/revoke status management.
- Capability-grant upsert authoring should default to a guided scope editor (key/value entries) with deterministic validation and provide an explicit advanced raw JSON override path for non-guided scope payloads.
- Configuration should include communication trust-receipt inventories for webhook and ingest receipts (`POST /v1/comm/webhook-receipts/list`, `POST /v1/comm/ingest-receipts/list`) with trust-state/audit-link context and inspect drill-ins seeded from event/audit identifiers.
- Configuration should include a daemon-backed Runtime Supervisor Timeline (`POST /v1/daemon/lifecycle/plugins/history`) with filter controls (`plugin_id`, kind/state/event type, limit), per-plugin health trend summaries, and lifecycle-event rows that drill into related `Inspect` context and plugin-kind diagnostics destinations (`Channels` for channel workers, `Connectors` for connector workers).
- Configuration should include a daemon-backed Memory Browser and Retrieval Context Inspector (`POST /v1/context/memory/inventory`, `POST /v1/context/memory/compaction-candidates`, `POST /v1/context/retrieval/documents`, `POST /v1/context/retrieval/chunks`) with principal/source filters, deterministic loading/empty/error states, and explicit document-selection flow before chunk inspection.
- High-impact configuration actions (`Install`/`Uninstall`/`Repair`/`Start`/`Stop`/`Restart`, retention purge/compaction apply, and revoke actions) must use the shared confirmation contract before dispatch; reversible lifecycle actions should expose short-lived undo affordances.

### 5.3 Chat

- Must provide a large chat history/view area.
- Chat history must render a unified typed timeline (`user_message`, `assistant_message`, `tool_call`, `tool_result`, `approval_request`, `approval_decision`) instead of a message-only transcript model.
- Iterative tool execution in one turn must render coherent tool-chain context (`Chain n`, `Step x of y`) across tool and approval rows.
- Tool/approval row state badges must map deterministically to `pending`, `running`, `blocked`, `completed`, and `failed`.
- Timeline technical metadata must stay in collapsed-by-default `Details` disclosures on each item.
- Tool and approval timeline rows must expose deterministic primary/secondary/destructive actions with explicit disabled-reason copy.
- Approval-request timeline rows should present compact lifecycle status and direct handoff/quick actions (`Open Approvals`, related drill-ins, optional quick-continue), while full guided decision controls and evidence remain canonical in `Approvals`.
- Chat may expose a bounded inline fast-path decision for low-risk pending approvals (`policy` risk only) with deterministic confirm + draft-undo + validation states; destructive or unknown-risk approvals must remain Approvals-only.
- Must provide a multiline input box for user entry.
- `Enter` must submit/send.
- `Shift+Enter` must insert newline.
- Must support streaming assistant output and execution progress.
- Chat composer must provide one autonomous `Send` path by default.
- Chat composer must not require or expose ask/act submission-mode overrides for normal workflow execution.
- Must expose task/run identity and current state.
- Must support cancel/interrupt controls while runs are active.
- Must preflight chat route readiness before send and block turn submission when no enabled ready model route exists.
- When preflight detects unresolved chat route, chat must surface one primary `Fix and Continue` remediation action with direct `Models` handoff (plus explicit fallback controls) instead of raw daemon 400/error text in transcript/status.
- `Effective Workflow Context` is the canonical route/provider/model/source owner in Chat; header and composer should keep concise state guidance and avoid reprinting the same route metadata.
- Chat send failures must present one primary `Fix and Continue` recovery action and explicit fallback controls (for example `Restore Prompt`, runtime refresh, setup navigation) with deterministic in-flight/result status copy.
- If `Fix and Continue` requires external remediation in `Models` or `Configuration`, Chat must preserve pending draft intent and auto-resume route checks/send handoff when user returns to Chat.
- Chat default copy should refer to user actions as `message` sends and reserve endpoint/transport wording for advanced troubleshooting surfaces only.
- If realtime stream setup fails or disconnects mid-turn, chat must fall back to one-shot `/v1/chat/turn` completion with explicit status messaging.
- Chat lifecycle state should finalize from realtime `chat_completed` and `chat_error` events when available, preserve partial streamed output on receipt/connectivity failures, and avoid false daemon-unreachable banners when completion already arrived over realtime.
- Chat timeline should consume canonical realtime turn-item lifecycle events when present (`turn_item_started`, `turn_item_delta`, `turn_item_completed`, `tool_call_started`, `tool_call_output`, `tool_call_completed`), without retaining legacy fallback rendering branches.
- Autonomous chat-to-action flows should preserve lifecycle parity for canonical tool outcomes (approval-required, completed, blocked, failed) with deterministic row-state and assistant-summary rendering, independent of viewport position or section re-entry.
- Tool timeline rows in failed/blocked states must surface inline remediation deep-links (`Open Configuration`, `Open Connectors`, `Open Channels`) when diagnostics indicate the relevant blocker domain.
- Chat composer must keep `Acting As` under a collapsed-by-default progressive disclosure (`Advanced Override`) and auto-reveal when non-default delegation context is selected or validation requires correction.
- Chat `Acting As` options must be sourced from workspace principal options with deterministic delegation-safe validation copy when the selected actor is outside the active identity directory.
- Chat-turn request payloads should persist the selected actor in `acting_as_actor_id` for daemon-side delegation context when supported.
- Chat traceability metadata should use `chat.turn` response correlation fields (`task_run_correlation.task_id|run_id|task_state|run_state|source`) when available.
- Chat should render a single effective workflow-context card for active/recent turns (task class + provider/model by default), expose direct `Open Related Tasks`/`Open Related Inspect` drill-ins, and keep technical trace metadata (route source, task/run IDs, correlation) inside a collapsed-by-default `Details` disclosure.
- Effective workflow-context card should surface response-shaping channel/profile/persona-source badges when present and include shaping guardrail/instruction counts in `Details`.
- Effective workflow-context card default body should present a scan-first reliability summary triplet (`What happened`, `What needs action`, `What next`) with deterministic recovery/next-step guidance before technical metadata rows.
- In `Simple`, workflow-context summary copy should stay action-first and plain-language (for example `Something went wrong`, `Open Approvals`) and avoid identifier-heavy/operator phrasing (`workflow step`, approval request ids) in default body text.
- In `Advanced`, technical/operator phrasing may be restored (for example explicit workflow-step wording and approval-request identifier context) while details remain discoverable through the same summary/actions contract.
- Interrupt action should issue a best-effort realtime cancel signal and cancel the in-flight local request path.
- Must link to task/run trace details.

### 5.4 Automation

- Must support create/edit/list for `SCHEDULE` and `ON_COMM_EVENT`.
- Must expose in-panel trigger management actions (`New Trigger`, per-card `Edit`, per-card enable/disable toggle, per-card `Delete`).
- Trigger create/edit forms must use typed ON_COMM_EVENT source/filter controls (not raw JSON editing), validate required fields with deterministic user-readable copy, and provide inline normalization/compatibility hints.
- ON_COMM_EVENT filter authoring should default to guided token editors (channels, actor IDs, senders, threads, keyword sets) with direct add/remove behavior and provide an explicit advanced raw JSON override path when users need non-guided payload control.
- Trigger update/delete responses should surface explicit idempotent/no-op status language so repeat actions remain explainable.
- Must support trigger simulation/testing and fire-history inspection.
- Trigger create/edit forms must include an `Acting As` selector sourced from workspace principal options, enforce deterministic delegation-safe validation copy, and persist the selected actor into daemon mutation payloads.
- Must show trigger inventory and status context using daemon automation list APIs (`POST /v1/automation/list`).
- Must show fire-history timeline context using daemon fire-history query API (`POST /v1/automation/fire-history`).
- Fire-history rows should surface route metadata (`task_class`, provider, model, route source) when available and expose direct drill-ins to related `Tasks`/`Inspect` context; when a run id is available, Inspect drill-in should scope to that run with an explicit clear/reset affordance.
- ON_COMM_EVENT authoring should surface source defaults (`MESSAGE`, `INBOUND`, `assistant_emitted=false`) and a normalized payload preview summary before submit.
- Must persist trigger create/update/delete actions via daemon automation APIs (`POST /v1/automation/create`, `POST /v1/automation/update`, `POST /v1/automation/delete`).
- Must expose in-app trigger simulation actions using daemon automation run APIs (`POST /v1/automation/run/schedule`, `POST /v1/automation/run/comm-event`).
- Should preserve deterministic empty/error states when no triggers are configured or daemon access fails.

### 5.5 Approvals

- Must list pending approval requests with risk rationale and acting-as context.
- Approvals loading/empty/filter copy should use decision-oriented wording and avoid exposing raw internal state-key names in default summaries.
- Approval cards should render identity context (`requested_by`, `subject`, `acting_as`, `decision_by`) display-name-first with explicit reveal/copy access to raw IDs.
- Must enforce exact phrase `GO AHEAD` for destructive approvals.
- Pending approval cards should use a guided decision form (`Action`, `Decision By`, `Decision note`) with deterministic actor defaults instead of raw actor-id text entry.
- Approve path should surface explicit required-phrase guidance with one-click phrase insertion (`Use Required Phrase`); reject path should not require the approval phrase.
- Approvals panel is the canonical full decision/evidence surface for approval actions; any quick actions in other panels must avoid duplicating full decision form fields and phrase-entry UI.
- Must show final decision state and audit linkage.
- Approval cards must present a scan-first summary triplet (`What happened`, `What needs action`, `What next`) before metadata rows.
- Approval metadata-heavy fields should remain available in a collapsed-by-default `Details` disclosure to preserve fast queue scanning.
- Approval cards should surface route metadata (`task_class`, provider, model, route source) when daemon route context is available.
- In `Simple`, approval summary copy should stay decision-first and plain-language (`Approval needed`, `Review details`) while avoiding audit/operator wording in default summary text.
- In `Advanced`, approval summary text may include operator-oriented evidence/audit phrasing where helpful, without moving core decision actions out of the primary card body.
- Approval cards should expose direct drill-ins to related `Tasks` list/detail and `Inspect` context using task/run linkage first, then deterministic route-context fallbacks when linkage is absent.
- Post-submit decision traceability (decision actor/outcome/rationale and latest action status copy) should remain visible after refresh/state transitions.
- Approval cards should include an inline expandable evidence panel that loads run detail on demand and surfaces step context, step input/output summaries (best effort from audit payloads), and related artifacts/audit snippets without leaving `Approvals`.
- Approvals identifier search context should persist workspace-scoped across panel switches/relaunch until explicit `Reset Filters`.
- Approvals inbox data source should use daemon approval query API (`POST /v1/approvals/list`).
- Approval decision submission should use daemon agent-approval API (`POST /v1/agent/approve`) with `workspace_id`, `approval_request_id`, `decision_by_actor_id`, and phrase fields.
- Must preserve deterministic pending/final grouping plus empty/error states when daemon approval data is unavailable.

### 5.6 Tasks

- Must list tasks/runs with state, priority, timestamps, and principal context.
- Task row and run-detail identity context (`requested_by`, `subject`, `acting_as`) should render display-name-first with explicit reveal/copy access to raw IDs.
- Must provide run detail including steps, artifacts, and policy decisions.
- Task cards must present a scan-first summary triplet (`What happened`, `What needs action`, `What next`) before metadata rows.
- Task metadata-heavy fields should remain available in a collapsed-by-default `Details` disclosure to preserve fast queue scanning.
- Must provide in-panel filtering by workflow state, priority band, and principal context.
- Must provide text search for task/run identifiers with deterministic no-match empty-state behavior.
- Tasks state/priority/principal/search filters should persist workspace-scoped across panel switches/relaunch until explicit `Reset Filters`.
- `New Task` should default to a goal-first submission flow with `Goal`, `Details`, and `Priority` visible first.
- `New Task` should auto-resolve principal context from the active workspace identity and keep `task_class`, `requested_by`, and `subject_principal` edits behind an explicit `Override Context` disclosure.
- Until daemon task-submit adds a first-class priority field, UI priority selection should be encoded into the submitted description payload in a deterministic backward-compatible format.
- Tasks `New Task` sheet continuity (presented state + draft fields) should persist workspace-scoped across section switches/relaunch and be cleared by explicit `Reset Context`.
- Must provide user-controlled auto-refresh mode (interval-based) with explicit in-flight/last-refresh visibility.
- Must surface route metadata per run (`task_class`, provider, model, route source) when daemon route context is available.
- In `Simple`, task summary and run-control status copy should be task-oriented and plain-language (for example `Task is paused`, `Retry started`) without run-id-heavy operator phrasing in default state text.
- In `Advanced`, task/run status copy may include run-specific operator wording and identifiers for diagnostics context.
- Must expose direct one-click drill-ins from task rows/details to related `Inspect` and `Approvals` context.
- Task rows and run-detail summary must expose daemon-backed run controls (`Cancel`, `Retry`, `Requeue`) with shared confirmation UX, deterministic action-availability enablement from daemon metadata, and per-run in-flight/result status copy.
- Task/run list data source should use daemon task query API (`POST /v1/tasks/list`).
- Run-detail drill-in may use daemon inspect-run API (`POST /v1/inspect/run`) when task status APIs do not include full step/artifact/audit detail.
- Must preserve deterministic empty/error states when daemon task/run data is unavailable.

### 5.7 Inspect

- Must default to an `Activity` view that prioritizes user-readable timeline summaries.
- Must expose a distinct `Trace` view for advanced diagnostics and correlation-heavy workflows.
- Must expose a non-filterable `Gallery` view for deterministic shared-component references used by visual regression baselines.
- `Activity` view should keep event summaries concise while preserving direct workflow drill-ins (`Open Tasks`, `Open Approvals`).
- `Trace` view should preserve full trace timeline and audit-event visibility.
- `Gallery` view should render stable reference treatments for shared status badges, action-role controls, card surfaces, runtime banners, remediation empty states, and filter-bar chrome.
- Must render logs in LIFO order (newest first).
- Must show success/failure status, input payload summary, output/result summary, and key debug metadata per log item.
- Must support live-tail streaming controls (`Resume Tail` / `Pause Tail`) without breaking manual refresh behavior.
- Must support status/severity filtering in-panel with deterministic no-match empty-state behavior.
- Must support metadata-aware filtering/search by task id, run id, correlation id, provider, and model context.
- Must support optional grouping by task id, run id, correlation id, provider, and model without mutating newest-first ordering semantics.
- Inspect status/metadata scope/grouping/search filters should persist workspace-scoped across panel switches/relaunch until explicit `Reset Filters`.
- Inspect log data source should use daemon inspect log query/stream APIs (`POST /v1/inspect/logs/query`, `POST /v1/inspect/logs/stream`).
- Inspect rows with task/run/step context should expose one-click navigation to related `Tasks` and `Approvals` context.
- Must expose plugin worker lifecycle events and health transitions.
- Must support operational debugging without direct database access.
- During placeholder phase, panel may ship with an explicit empty state and/or preview rows while daemon inspect APIs are being finalized.

### 5.8 Channels

- Must render logical channel cards for `App`, `Message`, and `Voice`.
- Each logical channel card must display status/configuration context for the selected primary mapped channel plus mapping visibility for all mapped channel implementations.
- Channel card data source should use daemon channel status/config summary API (`POST /v1/channels/status`).
- Channel cards must expose connector-mapping controls backed by daemon mapping APIs:
  - list (`POST /v1/channels/mappings/list`)
  - upsert (`POST /v1/channels/mappings/upsert`)
- Connector-mapping controls must support enable/disable and priority ordering per logical channel, surface connector capability metadata when available, and display deterministic channel-to-connector MVP constraints (`app -> builtin.app`, `message -> imessage|twilio`, `voice -> twilio`).
- Logical channel connector rollups (`Mapped Connectors`, connector health/reason/action summaries, and primary implementation selection) must be derived from daemon mapping metadata (`/v1/channels/mappings/list` bindings + priority/enabled state), not connector capability/name heuristics.
- Channel cards must expose editable configuration controls that persist via daemon upsert API (`POST /v1/channels/config/upsert`) with deterministic save/reset status copy.
- Channel configuration editors should prefer daemon descriptor-driven guided controls (`config_field_descriptors`) with typed rendering for enums/booleans/secrets/required fields; when descriptors are absent, guided controls should be synthesized from daemon editable-config metadata so primary setup paths are never raw-only, while preserving a default-collapsed `Advanced` raw key-value fallback for non-guided fields.
- Channel cards must expose explicit health test actions backed by daemon test API (`POST /v1/channels/test`) and render structured test summary/details inline.
- Channel cards must expose delivery-policy controls backed by daemon comm policy APIs:
  - list (`POST /v1/comm/policy/list`)
  - create/update (`POST /v1/comm/policy/set`)
- Delivery-policy controls must support primary channel, retry count, fallback channels, default-policy semantics, and endpoint-pattern editing.
- Delivery-policy save actions must use the shared high-impact confirmation contract before mutation dispatch.
- Channel diagnostics actions should use daemon channel diagnostics summary API (`POST /v1/channels/diagnostics`) for worker-health snapshots and remediation action metadata.
- Channels must render daemon-declared remediation actions directly; actions with unsupported intent/destination contracts should be hidden deterministically, while daemon-disabled actions remain visible with reason copy.
- Logical channel cards must surface mapped-connector rollups (health, reasons, available connector remediation actions) and provide direct navigation to `Connectors` when channel-level actions are unavailable.
- Daemon channel payloads are canonical (`app|message|voice`); UI should render exactly one logical card per channel kind with deterministic mapping rollups and diagnostics action visibility/copy.
- Channel cards should support per-card collapse/expand to reduce clutter in long views and should default to collapsed on first render.
- Channel-card expanded/collapsed state should persist workspace-scoped across section switches/relaunch and clear under explicit continuity reset actions.
- Channels must expose panel-level dirty-state visibility with `Save All`/`Discard All` controls for channel config, delivery-policy, and connector-mapping drafts.
- Channels should show per-card unsaved-draft indicators when configuration/policy/mapping edits diverge from daemon-backed source values.
- UI should feel complete even if some channel actions are temporarily unavailable; unavailable actions should be visibly disabled with explanatory copy.

### 5.9 Connectors

- Must render each connector as a dedicated card.
- Connector cards should be connector-semantic (for example one unified `Twilio` card with SMS+voice capability context and one `Messages` card for iMessage integration).
- Each connector card must display relevant status and configuration fields.
- Connector card data source should use daemon connector status/config summary API (`POST /v1/connectors/status`).
- Logical connector card IDs should use canonical daemon connector identities (`twilio`, `imessage`, `builtin.app`, etc.) with one logical card per canonical connector ID.
- Connector cards must expose editable configuration controls that persist via daemon upsert API (`POST /v1/connectors/config/upsert`) with deterministic save/reset status copy.
- Connector configuration editors should prefer daemon descriptor-driven guided controls (`config_field_descriptors`) with typed rendering for enums/booleans/secrets/required fields; when descriptors are absent, guided controls should be synthesized from daemon editable-config metadata so primary setup paths are never raw-only, while preserving a default-collapsed `Advanced` raw key-value fallback for non-guided fields.
- Unified connector cards should route mutable operations (save/test/remediation) through a deterministic primary connector ID.
- Connector cards must expose explicit health test actions backed by daemon test API (`POST /v1/connectors/test`) and render structured test summary/details inline.
- Connector diagnostics actions should use daemon connector diagnostics summary API (`POST /v1/connectors/diagnostics`) for worker-health snapshots and remediation action metadata.
- Connectors must render daemon-declared remediation actions directly; unsupported action contracts should be hidden deterministically, while daemon-disabled actions remain visible with reason copy.
- Connector cards must provide Apple TCC permission management controls for connector-required permissions.
- On connector-view open, each card must show current permission status by default.
- Connector permission and health state rendering should prioritize daemon connector metadata (`configuration.permission_state`, `configuration.status_reason`) with local probes used only when daemon metadata is unavailable.
- `Request Permission` action should render only when daemon declares a `request_permission` remediation action for that connector; it should remain disabled when permission is already granted or when daemon marks the action disabled.
- `Open System Settings` action should always be visible and enabled.
- Opening connector System Settings from card or diagnostics actions must register a pending refresh and re-check permission state when the app becomes active again.
- Supported permission prompts should route through daemon permission request API (`POST /v1/connectors/permission/request`) so TCC attribution targets `Personal Agent Daemon`.
- For permission classes that cannot be granted programmatically, connector cards should route users directly to the correct System Settings surface.
- Connector remediation actions should be presented in deterministic priority order with daemon-recommended actions first and status-reason-specific operator copy when available.
- Connector cards should support per-card collapse/expand and should default to collapsed on first render.
- Connector-card expanded/collapsed state should persist workspace-scoped across section switches/relaunch and clear under explicit continuity reset actions.
- Connectors must expose panel-level dirty-state visibility with `Save All`/`Discard All` controls for connector configuration drafts.
- Connectors should show per-card unsaved-draft indicators when editable values diverge from daemon-backed source values.

### 5.10 Models

- Must show provider inventory/readiness state in-app using daemon APIs (`POST /v1/providers/list`, `POST /v1/providers/check`).
- Must expose per-provider setup controls for endpoint and API-key secret reference management using daemon mutation APIs (`POST /v1/providers/set`, `POST /v1/secrets/refs`).
- Must show current chat-route model resolution context from daemon model-routing APIs (`POST /v1/models/resolve`).
- Must show model catalog readiness and task-class policy records in-app using daemon model APIs (`POST /v1/models/list`, `POST /v1/models/policy`).
- Must support provider-scoped model catalog management via daemon model APIs:
  - discovery (`POST /v1/models/discover`)
  - manual add/upsert (`POST /v1/models/add`)
  - remove (`POST /v1/models/remove`)
- Must render each provider as a dedicated card (for example `OpenAI`, `Anthropic`, `Google`, `Ollama`) with provider-scoped model catalog and routing-policy details.
- Model catalog rows must expose per-model enable/disable controls backed by daemon mutation APIs (`POST /v1/models/enable`, `POST /v1/models/disable`).
- Provider cards must expose:
  - `Discover` action for provider-backed model inventory
  - manual `Add Model` control for explicit model-key entry
  - per-row `Remove` action for catalog entries
- Provider cards should support collapse/expand to keep the Models view scannable when provider/model lists grow and should default to collapsed on first render.
- Provider setup controls must expose explicit `Save`, `Check`, and `Reset Endpoint` actions per provider card.
- Provider readiness states must clearly distinguish `Setup Required`, `Configured`, `Healthy`, and `Check Failed`.
- API-key values entered in Models setup controls must remain write-only and be stored via local secure storage with daemon secret-reference registration before provider save.
- Successful model toggles should refresh chat-route candidacy context in-panel without requiring a full Models panel reload.
- Successful discover/add/remove actions should refresh model catalog + policy context in-panel without requiring section re-entry.
- Must include a route-policy editor in `Models` that allows selecting task class (at minimum `chat`) and choosing provider/model only from currently valid catalog entries.
- Route-policy editor save must persist through daemon model-select API (`POST /v1/models/select`) and update in-panel routing-policy rows plus chat route summary source/context immediately after success.
- Route-policy save actions must use the shared high-impact confirmation contract before mutation dispatch.
- Models must include a quickstart wizard card (default scan-first owner above advanced controls) with deterministic step flow: `Connect Provider` -> `Choose and Enable Model` -> `Set Chat Route` -> `Test in Chat`.
- Quickstart wizard must expose exactly one primary current-step action at a time and reuse existing provider/catalog/route mutations rather than introducing parallel setup owners.
- Quickstart `Test in Chat` action must deep-link to `Chat` with route context chips and an optional seeded validation draft when the composer is empty.
- Must include route simulation and explainability controls in `Models` backed by daemon route-analysis APIs (`POST /v1/models/route/simulate`, `POST /v1/models/route/explain`) with task-class and optional principal context inputs.
- `Models` section order should prioritize setup-before-policy:
  - quickstart + provider readiness header first
  - provider cards (setup/catalog/policy controls) before route fine-tuning controls
  - route summary/readiness and route-policy editor after provider setup controls
  - route simulation/explainability remains advanced and subordinate to setup/policy controls
- Route simulation output must show selected provider/model/source, machine-readable reason codes, step-level decision trace, and fallback-chain ranking (including selected candidate marker).
- Route explainability output must show summary/explanations plus reason-code, decision-trace, and fallback-chain context aligned to the same task-class/principal input.
- Models must expose panel-level dirty-state visibility with `Save All`/`Discard All` controls for provider setup drafts and show per-provider unsaved indicators.
- Must preserve deterministic empty/error states when provider/model APIs are unavailable.

### 5.11 Communications

- Must provide a dedicated communications inbox panel for daemon-backed thread, event, and call-session visibility.
- Communications inbox data sources should use daemon comm query APIs:
  - thread list (`POST /v1/comm/threads/list`)
  - event timeline (`POST /v1/comm/events/list`)
  - call-session list (`POST /v1/comm/call-sessions/list`)
  - delivery-attempt history (`POST /v1/comm/attempts`) using operation-scoped filters with optional context linkage (`thread_id`, `task_id`, `run_id`, `step_id`) when present
- Communications presentation should group records by logical channel (`app`, `message`, `voice`) while preserving connector-scoped thread separation within `message`.
- Communications query/filter and row attribution should consume first-class daemon `connector_id` metadata for threads/events/call sessions instead of inferring connector identity from `external_ref` parsing.
- Communications should include a `Conversation Continuity` section backed by `/v1/chat/history`, grouped by logical channel, showing latest turn-item summary per turn/correlation with deterministic `Open Chat`, `Open Related Tasks`, and `Open Related Inspect` drill-ins.
- Conversation continuity rows should preserve cross-channel lifecycle parity (`awaiting_approval`, `completed`, `failed`) for canonical `app|message|voice` orchestration paths and keep lifecycle status visible after refresh/re-entry.
- Conversation continuity rows should show response-shaping profile/persona-source badges when available and keep shaping metadata/counts in default-collapsed `Details`.
- Communications channel/direction/thread/search filters should persist workspace-scoped across panel switches/relaunch until explicit `Reset Filters`.
- Communications must provide a persisted `Compact Scan` mode optimized for high-volume triage while preserving thread title, channel, connector attribution, and direction clarity.
- Communications compose draft continuity (sheet presented state + source/thread/connector/destination/message fields) should persist workspace-scoped across section switches/relaunch with stale-thread invalidation when thread context disappears.
- Communications toolbar should expose `Reset Context` to clear persisted communications continuity (filters, compact mode, triage state, and compose draft) for the active workspace.
- Communications should surface current workspace/principal identity context display-name-first with explicit reveal/copy access to raw IDs.
- Must render sender/recipient metadata for event rows when available, including channel/type direction context.
- Must include deterministic loading, empty, no-match, and degraded error states that do not depend on seeded preview payloads.
- Communications must expose outbound compose flows for `New Message`, `Reply`, and `Start Call`, all routed through daemon send API (`POST /v1/comm/send`).
- Communications compose should default to a `Quick Send` form showing only `Recipient` and `Message`, with `Source Channel`, `Conversation ID`, and `Connector Hint` behind an explicit default-collapsed `Advanced` disclosure.
- Reply/start-call compose entry points should prefill thread-context hints (`thread_id`, connector hint, and destination suggestion when deterministic) while keeping destination editable.
- Compose submit path must surface deterministic validation/in-flight/result status copy and refresh communications inbox + delivery-attempt context after send completion.
- Communications thread/event rows must surface deterministic triage indicators (`Unread`, `New`, `Follow Up`, `Handled`) and provide one-click triage actions (`Mark Handled`/`Reopen`, `Follow Up`/`Clear Follow Up`, `Open Task Draft`) without requiring panel navigation.
- Communications technical identifiers/trace metadata (thread/event/session/attempt IDs, idempotency/operation/task/run linkage) should stay available behind collapsed-by-default `Details` disclosures on row cards instead of inline short-ID labels.
- Must provide in-panel filters for free-text search and channel/direction/thread scoping.
- Delivery-attempt timeline rows must surface retry/fallback evidence (`route_phase`, `retry_ordinal`, `fallback_from_channel`) and status badges.
- Rows with related workflow identifiers should expose one-click drill-ins to `Inspect` and channel context.
- Must keep sort behavior deterministic (newest-first timeline semantics where timestamp metadata is available).

## 6) Identity and Access UX

- UI must clearly represent:
  - `requested_by`
  - `subject_principal`
  - `acting_as`
- Cross-principal execution attempts should display delegation status and failure reason when denied.

## 7) Communication UX Model

- Thread and channel-linked event visibility lives primarily in `Communications`, with contextual drill-ins across `Chat`, `Tasks`, and `Channels`.
- Event-level sender/recipient metadata should remain visible where context requires reply/audit correctness.
- Channel-level capability limits and fallback behavior should be visible in channel cards and relevant run details.

## 8) Automation UX Model

- Directives and triggers are user-manageable objects with explicit scope.
- Automation evaluation outcomes should be inspectable with idempotency signals.
- ON_COMM_EVENT authoring should be typed and preview-first, with source defaults and filter normalization visible before save.

## 9) Model and Provider UX

- Users can inspect selected provider/model policy outcome for a run.
- Connectivity/health state for configured providers should be visible in `Models`.
- `Models` should expose daemon-backed model catalog availability and routing-policy records for key task classes.

## 10) Safety and Approval UX

- Destructive/non-reversible actions require explicit in-app approval UX.
- Ambiguous classification or low confidence should surface clarifying prompts or safe alternatives.
- Voice-originated destructive requests require in-app approval handoff representation.
- High-impact app actions should use one shared confirmation system (deterministic title/message/confirm label semantics) and provide undo affordances for reversible actions where feasible.

## 11) Execution and Concurrency UX

- Multiple runs may be visible concurrently.
- State machine progression must be represented clearly:
  - `queued`
  - `planning`
  - `awaiting_approval`
  - `running`
  - `blocked`
  - `completed|failed|cancelled`

## 12) Delivery and Reliability UX

- Delivery attempts and idempotency/replay outcomes should be visible where user-relevant.
- Transport or daemon failures should include actionable recovery guidance.
- First-load panel transitions should preserve layout continuity with deterministic skeleton placeholders until each section completes its first data query.

## 13) Voice UX Requirements

- Voice-linked events and transcript turns should appear in comm/inspect surfaces.
- Inbound/outbound call session status should be visible with lifecycle transitions.

## 14) Storage, Secrets, and Retention UX

- Secret values are never displayed or retrievable via app UI.
- Retention controls are visible and explain default policy behavior.
- Memory/context controls expose tuning settings and last known telemetry summaries.

## 15) MVP UI Acceptance Criteria

1. Taskbar menu shows daemon status and supports day-to-day daemon start/stop controls.
2. Taskbar menu supports open/close main app window and app exit; closing the main window moves the app to menu-bar-only mode (no Dock/app-switcher icon) until reopened from taskbar.
3. Taskbar `Quit` closes open app windows and issues a best-effort daemon stop request before app termination.
4. Taskbar lifecycle actions execute directly from UI and handle native authorization + action progress states.
4. `Install/Uninstall` daemon actions are available in `Configuration > Advanced`; `Install`/`Repair` run from bundled helper setup flow with deterministic remediation copy, and taskbar menu surfaces setup/repair actions contextually when daemon setup is missing/broken.
5. Taskbar menu includes concise readiness hints for token/provider/model setup gaps with direct remediation actions (`Open Configuration`, `Open Models`, contextual daemon setup/start/repair).
6. Main window uses sidebar + main panel split with collapsible sidebar behavior.
7. Sidebar navigation includes `Configuration`, workflow destinations (`Home`, `Chat`, `Communications`, `Tasks`, `Approvals`, `Automation`), and an advanced disclosure group (`Inspect`, `Channels`, `Connectors`, `Models`) that is collapsed by default but auto-expands when an advanced destination is selected.
8. Ready-state app launch lands on `Home` first and exposes one deterministic next action plus first-session milestone checklist before users branch into operator-heavy sections.
8. Sidebar footer always shows daemon and app-to-daemon connection status.
9. Chat panel supports large conversation view + multiline input (`Enter` send, `Shift+Enter` newline).
10. Inspect view uses LIFO logs with clear success/failure and input/output debugging summaries.
11. Channels view provides logical cards for `App`, `Message`, and `Voice`, with per-card collapse/expand, first-render collapsed defaults, mapped connector health/reason/action rollups driven by daemon channel-to-connector bindings, and mapping-priority-aware primary implementation selection.
12. First-run users see a dedicated onboarding panel for workflow sections until token, daemon, provider, catalog, chat-route, and channel/connector mapping checks are satisfied, while `Configuration`, `Channels`, `Connectors`, and `Models` remain directly accessible for setup/remediation actions through prioritized `Fix Next`; non-Configuration panels also show a compact `Current Blocker` ribbon during incomplete readiness with deterministic contextual secondary remediation action.
13. Connectors view shows connector-semantic cards keyed by canonical connector IDs (for example `twilio`, `imessage`) with status/configuration and TCC management controls.
14. Cross-panel deep-link actions (`Open Models`, `Open Tasks`, `Open Inspect`, `Open Configuration`) trigger one deterministic refresh in the destination section, including same-section re-entry.
15. Connector cards show permission status by default from daemon metadata, render `Request Permission` only when daemon declares it for the connector and keep it disabled when already granted/daemon-disabled, and always enable `Open System Settings`.
16. UI never exposes plaintext secret values and remains thin-client with daemon-owned side effects.
17. Approvals and Tasks render daemon-backed workflow inbox/list rows with risk/principal/decision metadata (`/v1/approvals/list`) and task/run state/priority/timestamp/principal metadata (`/v1/tasks/list`), with deterministic empty/error handling.
18. A dedicated UI manual test guide index exists (`docs/tests-ui.md`) with panel-specific manuals under `docs/tests-ui/*.md`, and both are maintained for user-testable app flows.
19. Core UI surfaces (shell, taskbar menu, chat, inspect, channels, connectors, models, configuration) use a cohesive Tahoe-style visual system with reduced-motion-safe interaction behavior.
20. Runtime status notices for disconnected/degraded/missing/broken/stopped states are consistent across `Chat`, `Inspect`, `Channels`, `Connectors`, `Models`, and `Configuration`.
21. Models shows provider/model readiness context using daemon-backed provider inventory/check and model-route resolution APIs.
22. Models supports daemon-backed model discovery/add/remove management flows per provider (`/v1/models/discover`, `/v1/models/add`, `/v1/models/remove`) with deterministic status messaging.
23. Models shows daemon-backed model catalog entries and routing-policy records with readable provider readiness context.
24. Automation section renders daemon-backed trigger inventory with status metadata and deterministic empty/error copy.
25. Automation section includes daemon-backed schedule and comm-event simulation actions with clear success/error summaries.
26. App window and taskbar menu use a flatter Tahoe glass baseline with lighter separators/materials and reduced decorative shadows.
27. Core controls (buttons, menus, text fields) use built-in SwiftUI/AppKit component styles by default, with minimal custom chrome.
28. Main-window navigation uses native split-view/list selection behavior instead of a custom chrome container.
29. Taskbar menu uses native form sections, and Channels/Connectors cards use native disclosure affordances with first-render collapsed defaults.
30. Models view renders provider-scoped cards with collapse/expand behavior, first-render collapsed defaults, and provider-local model catalog/policy context.
31. Startup surfaces avoid synthetic preview runtime data for inspect/channels/connectors; runtime warning banners appear only after the first daemon lifecycle probe completes.
32. Chat section uses daemon realtime stream (`/v1/realtime/ws`) for token deltas when available, provides in-flight interrupt control, and shows deterministic fallback copy when realtime is unavailable.
33. Automation section supports daemon-backed trigger creation/edit/deletion flows with deterministic validation and idempotent status messaging.
34. Inspect section supports metadata-aware filter/group controls (task/run/correlation/provider/model) and per-row direct navigation into related Tasks/Approvals context.
35. Tasks section renders daemon route metadata per run and provides direct drill-ins from rows/detail into related Inspect/Approvals context without manual identifier copy/paste.
36. Approvals section renders daemon route metadata and provides direct drill-ins to related Tasks list/detail and Inspect context while preserving decision traceability state after refreshes.
37. Plugin-worker-only lifecycle degradation is surfaced with direct Channels diagnostics remediation in Configuration/taskbar surfaces and does not block onboarding with generic daemon setup-repair messaging.
38. Approvals section includes an inline expandable evidence panel with step input/output summaries and related artifacts/audit snippets, and keeps disclosure state stable across approvals refresh and decision submissions.
39. Configuration includes a daemon-backed identity hub (workspace picker + workspace/principal directories with actor-handle mappings) and workspace switching refreshes app-shell context deterministically.
40. Chat and Automation authoring flows expose `Acting As` controls, block submits with deterministic validation copy when actor selection is out of active identity scope, and persist selected actor values in outgoing daemon request payloads; Chat keeps the control collapsed by default under progressive disclosure unless explicit delegation correction/selection is required.
41. Configuration includes daemon-backed delegation rule management (list/create/revoke) with scope visibility and deterministic validation/status behavior for scope and actor constraints.
42. Communications section renders daemon-backed thread/event/call-session data grouped by logical channel (`app|message|voice`) with connector-attributed rows/filters (from first-class `connector_id` fields), deterministic filter/search behavior, empty/no-match/error states, and direct Inspect/Channels drill-ins.
43. Channels and Connectors cards support daemon-backed configuration editing (`/v1/channels/config/upsert`, `/v1/connectors/config/upsert`) using descriptor-driven guided fields with default-collapsed `Advanced` raw fallback, plus explicit health test actions (`/v1/channels/test`, `/v1/connectors/test`) and inline result rendering.
44. Channels section includes daemon-backed delivery-policy list/edit controls (`/v1/comm/policy/list`, `/v1/comm/policy/set`) with deterministic draft/save/reset status behavior.
45. Communications section includes a delivery-attempt timeline backed by `/v1/comm/attempts` with retry/fallback/status visibility and direct workflow drill-ins where attempt context is available.
46. Automation ON_COMM_EVENT create/edit flows use guided source/filter controls with inline normalization hints and normalized payload preview, while keeping an explicit advanced raw-JSON override path.
47. Configuration includes a runtime supervisor timeline backed by `/v1/daemon/lifecycle/plugins/history` with filterable plugin lifecycle events, per-plugin trend summaries, and direct drill-ins to related `Inspect`, `Channels`, or `Connectors` diagnostics context.
48. Configuration includes daemon-backed memory/retrieval diagnostics surfaces with principal/source-scoped memory inventory browsing, compaction-candidate preview visibility, retrieval document selection, and document-scoped retrieval chunk inspection with deterministic status/empty/error behavior.
49. Configuration includes capability-grant governance controls (upsert/list/revoke with filterable inventory) and communication trust-receipt governance inventory (webhook + ingest filters with inspect/audit-link drill-ins) backed by daemon APIs.
50. Models includes daemon-backed route simulation and explainability controls (`/v1/models/route/simulate`, `/v1/models/route/explain`) with task-class/optional-principal inputs plus selected-route/reason-code/decision/fallback rendering.
51. Configuration includes daemon-backed identity device/session inventory visibility and per-session revoke controls with deterministic filters/status summaries.
52. `Channels`, `Connectors`, and `Models` show unsaved-draft indicators, support section-level `Save All`/`Discard All`, and prompt before cross-section navigation discards unsaved drafts.
53. App shell surfaces non-blocking notification toasts for major panel status outcomes, and a notification center sheet persists those outcomes as an actionable inbox grouped by user intent (`Needs Attention`, `Workflow Updates`, `Runtime and Setup`, `Diagnostics`, `General`) with newest-first ordering inside each group, direct next-action buttons, read/clear controls, and search/source filters.
54. Configuration uses built-in mode navigation with `Setup` as first/default mode and groups advanced/operator content outside the default setup flow.
55. Configuration `Setup` mode provides compact readiness summary badges and prioritized setup quick actions (`Fix Next`, primary remediation, `Refresh Checks`) while preserving deterministic one-click remediation behavior.
56. Operator-heavy Configuration workflows in `Integrations`, `Data`, and `Advanced` modes render as default-collapsed disclosure sections that preserve deterministic loading/empty/error behavior when expanded.
54. Empty/error states in `Chat`, `Communications`, `Tasks`, `Approvals`, `Automation`, `Inspect`, `Channels`, `Connectors`, and `Models` surface concise explanation plus deterministic one-click remediation CTAs aligned to setup/runtime context.
55. Command palette should surface intent-ranked natural-language command results and ranked concrete-object matches (`Tasks`, `Approvals`, `Threads`, `Connectors`, `Models`) with deterministic tie-break ordering, preserve deterministic disabled-reason explanations for unavailable runtime/setup actions, and execute the first enabled query match on `Enter`.
56. Communications, Tasks, Approvals, and Inspect should surface concise active-filter indicators near panel headers (count + summary) and provide one-click clear controls that reset local + persisted filter context.
57. Sidebar navigation rows for Communications, Tasks, Approvals, and Inspect should show compact active-filter count badges when persisted non-default filters are active in the current workspace.
58. Sidebar active-filter count badges should expose concise summary help text derived from persisted filter tokens so users can inspect narrowed context without opening each panel.
59. App shell provides a workspace-scoped information-density mode control with `Simple` as default and deterministic command-palette parity for density toggles.
60. `Simple` mode suppresses low-value internal metadata in major workflow panels (`Chat`, `Communications`, `Tasks`, `Approvals`, `Automation`, `Inspect`, `Channels`, `Connectors`, `Models`), while `Advanced` restores full operator detail.
61. `Models` includes a deterministic quickstart wizard (`Connect Provider`, `Choose and Enable Model`, `Set Chat Route`, `Test in Chat`) that reuses existing provider/model/route mutation flows and deep-links into `Chat` for one-click route validation.
61. Cross-panel workflow drill-ins among `Communications`, `Tasks`, `Approvals`, `Automation`, and `Inspect` use consistent `Open Related ...` labels, show destination context chips, and provide deterministic `Back to ...` return behavior.
62. Capability-grant upsert forms default to guided scope key/value editing and keep an explicit advanced raw-JSON override path with deterministic object-validation messaging.
63. UI validation contracts must include explicit first-10-minute external-user success coverage (setup gating/recovery affordances, first chat send, first task submit, and first approval decision) across app-host smoke automation and manual UI checks.
64. Configuration `Workspace` mode includes daemon-backed chat persona policy controls (scope/principal/channel/simple-prompt/advanced-guardrails) with deterministic refresh/save/reset/test behavior and default-collapsed advanced guardrails disclosure.
65. Core workflow accessibility contract requires deterministic sidebar/panel landmarks, labeled icon-only dismiss controls for drill-in ribbons, and search-field accessibility identifiers for command palette + workflow filter bars (`Communications`, `Approvals`, `Tasks`).
66. App-host smoke coverage includes deterministic recovery journeys for missing auth, route-missing onboarding, and degraded runtime with explicit remediation/navigation assertions and no legacy fallback assertions.
67. `Home` in `Advanced` density mode includes first-success funnel diagnostics (message/communication/task/approval) with aggregate completion metrics plus per-milestone first-completion source/timestamp evidence (or explicit pending state).
68. Cross-panel status messaging follows the information-ownership contract: one canonical detailed owner per status class, with summary/remediation-only rendering elsewhere and no duplicated status paragraphs between header/empty-state/body surfaces.
69. For unsigned/ad-hoc local/internal macOS builds, setup surfaces render deterministic host-trust remediation guidance (Gatekeeper override actions) and keep daemon/runtime diagnostics copy scoped to post-launch setup issues.

## 16) Data Model Touchpoints

Primary objects consumed by UI include:

- `Task`, `TaskRun`, `TaskStep`
- `ApprovalRequest`
- `CommThread`, `CommEvent`, `DeliveryAttempt`, `CommProviderMessage`, `CommCallSession`
- `Directive`, `AutomationTrigger`, `TriggerFire`
- `CapabilityGrant`
- `UserDevice`, `DeviceSession`
- `AuditLogEntry`
- `CommWebhookReceipt`, `CommIngestReceipt`
- `RuntimePlugin`, `RuntimePluginProcess`
- `SecretRef` (metadata only)
- `ContextBudgetSample`, `ContextBudgetTuningProfile`

## 17) Decision Log

Use this section to record explicit UI decisions as they are made.

| Date | Area | Decision | Rationale | Status |
|---|---|---|---|---|
| 2026-03-03 | Cross-Screen Information Ownership + De-Dupe Contract | Standardized canonical owner surfaces for runtime, setup, route, approval, communications continuity, and empty-state status copy so non-owner surfaces remain summary/remediation-only. | Prevents repeated status text and conflicting guidance across shell/panel surfaces, improving scan speed and action clarity for both first-session users and operators. | accepted |
| 2026-03-03 | Orchestration-v2 Cross-Channel Lifecycle Contract | Clarified canonical UI contract that autonomous chat outcomes and communications continuity rows must preserve deterministic lifecycle parity (`awaiting_approval`, `completed`, `failed`) across `app|message|voice`, and setup-matrix chat-route rows must show resolved provider/model context with deterministic `Open Models` remediation when unresolved. | Keeps Chat/Communications/Configuration behavior and manual validation aligned with orchestration-v2 cutover expectations while preventing legacy or viewport-dependent interpretation drift. | accepted |
| 2026-03-02 | Inspect Component Gallery Baseline | Added a dedicated `Inspect` `Gallery` mode with deterministic shared-component references and a visual-regression snapshot anchor (`window-inspect-gallery`). | Establishes one in-app source of truth for reusable UI pattern treatments and reduces panel-level visual drift during ongoing UI iteration. | accepted |
| 2026-03-02 | Accessibility Navigation Second Pass | Standardized core accessibility landmarks + control metadata: sidebar/panel landmark labels, labeled drill-in dismiss icon control, command-palette auto-focus, and deterministic search-field accessibility IDs/labels in `Communications`/`Approvals`/`Tasks`. | Keeps keyboard-first and VoiceOver-first workflow traversal deterministic across the highest-frequency panels and prevents unlabeled icon controls from re-entering core routes. | accepted |
| 2026-02-27 | Command Palette Intent Ranking | Added intent-ranked natural-language query ordering for command palette results with deterministic tie-break behavior, while preserving disabled-reason visibility and first-enabled `Enter` execution. | Improves keyboard-first command discovery for non-exact phrasing without sacrificing deterministic action ordering and guard-rail clarity. | accepted |
| 2026-02-27 | First-10-Minute Validation Baseline | Added explicit first-10-minute success validation contract across app-host smoke + manual UI checks covering setup recovery affordances, first chat send, first task submit, and first approval decision. | Keeps early-user journey quality measurable and prevents regressions that make initial setup/actions look “done” but unusable in sequence. | accepted |
| 2026-02-27 | Notification Center Actionable Inbox | Reframed Notification Center as an intent-grouped workflow inbox with deterministic section ordering, newest-first rows per intent, and row-level `Open ...` next-action buttons for direct navigation/remediation. | Moves the notification surface from passive status history to task-oriented triage while preserving searchable/read-clearable history and predictable ordering. | accepted |
| 2026-02-27 | Guided Editors with Raw Fallback (`Automation` + Capability Governance) | Switched ON_COMM_EVENT filter authoring and capability-grant scope authoring to guided editors by default, and kept explicit advanced raw JSON override flows (`Use raw JSON override when saving`) for non-guided payload control. | Reduces operator-heavy raw editing in common flows while preserving full payload flexibility and deterministic validation messaging for advanced users. | accepted |
| 2026-02-27 | Cross-Panel Drill-In Context + Return | Standardized workflow drill-ins to `Open Related ...` labels and added a destination drill-in ribbon showing source context chips plus one-click `Back to ...` return behavior. | Keeps cross-panel diagnostics/workflow navigation consistent, reduces “where did this context come from?” ambiguity, and shortens return-path friction after detail inspection. | accepted |
| 2026-02-27 | Inspect Activity/Trace Split | Split Inspect into `Activity` (default) and `Trace` (advanced) modes, persisting mode selection per workspace while keeping trace-only filter/group tooling under explicit `Trace` selection. | Makes default inspect UX readable for end users while preserving full operator diagnostics without removing existing workflows. | accepted |
| 2026-02-27 | Workspace Sentinel Migration | Standardized UI workspace canonicalization so env/defaults/daemon-sourced `default` workspace values are normalized to `ws1`, with idempotent migration of workspace-scoped persisted UI state keys (`filters`, `triage`, `continuity`, `density`). | Prevents mixed-workspace restore/query drift across app upgrades and restart/bootstrap paths while preserving existing user context. | accepted |
| 2026-02-27 | Micro-Interaction Success Feedback | Added source-scoped success pulse feedback for high-frequency controls and reduced-motion-safe symbol effects across chat/approvals/tasks/save actions, plus compact success reinforcement badges where action completion is most visible. | Improves action confidence and perceived responsiveness without adding heavy motion or introducing non-deterministic animation behavior. | accepted |
| 2026-02-27 | Panel Latency Budget Diagnostics | Added deterministic panel latency instrumentation categories (`initial render`, `refresh`, `transition`) with per-section budgets and surfaced latest/regression summaries in `Configuration > Advanced` under `Panel Latency Budgets`. | Makes perceived-performance regressions observable and testable without external profiling tooling, while keeping diagnostics in a deliberate operator-only surface. | accepted |
| 2026-02-26 | Workspace Continuity Beyond Filters | Extended workspace continuity to include `Communications` compose draft/triage context, `Tasks` new-task draft context, and `Channels`/`Connectors` expanded-card state, with explicit `Reset Context` affordances to clear persisted continuity safely. | Preserves in-progress user intent across section switches/relaunch while preventing hidden stale state through deterministic reset controls. | accepted |
| 2026-02-26 | Information Density Mode | Added a workspace-scoped `Simple`/`Advanced` information-density control in app shell + command palette, defaulting each workspace to `Simple` while preserving full operator metadata in `Advanced`. | Keeps default UX focused on user-readable outcomes without removing troubleshooting depth needed for advanced operators. | accepted |
| 2026-02-26 | Channel/Connector Guided Config Forms | Standardized channel/connector configuration UX on daemon `config_field_descriptors` for typed guided controls (enum/bool/secret/required/help) and moved raw key-value editing to a default-collapsed `Advanced` fallback section for non-descriptor fields. | Keeps setup flows aligned to backend metadata with less operator guesswork while preserving escape-hatch editing for compatibility fields and unknown keys. | accepted |
| 2026-02-26 | Sidebar Filter Badge Summaries | Added concise hover/help summaries to sidebar active-filter count badges for Communications/Tasks/Approvals/Inspect using persisted filter-token text. | Gives users quick filter-context visibility directly in navigation and reduces hidden-state confusion without adding sidebar clutter. | accepted |
| 2026-02-26 | Sidebar Filter Count Badges | Added compact sidebar count badges for Communications/Tasks/Approvals/Inspect that reflect persisted active-filter totals in the current workspace. | Improves discoverability of hidden filter context before opening a panel and reduces confusion when persisted filters narrow data unexpectedly. | accepted |
| 2026-02-26 | Active Filter Indicator Baseline | Added compact header-level active-filter indicators for Communications/Tasks/Approvals/Inspect with count + summary tokens and one-click clear actions tied to persisted filter context resets. | Prevents hidden stale filters after panel switches/relaunch while keeping reset affordance immediately visible without opening full filter controls. | accepted |
| 2026-02-26 | Command Palette Ergonomics | Added recent-command prioritization, first-enabled-match execution on `Enter`, and deterministic disabled-reason copy for unavailable commands inside command palette rows. | Speeds repeat operator workflows while preserving explicit guard-rail explainability when setup/runtime actions are unavailable. | accepted |
| 2026-02-26 | Workspace-Scoped Filter Context Persistence | Standardized workspace-scoped persistence for Communications, Tasks, Approvals, and Inspect search/filter context across panel switches and app relaunch, with explicit per-panel `Reset Filters` actions clearing local + persisted context together. | Keeps operator workflow continuity during cross-panel diagnostics while preserving deterministic reset semantics and preventing stale filter confusion across workspace boundaries. | accepted |
| 2026-02-26 | Keyboard-First Command Access | Added scene-level keyboard shortcuts + searchable command palette backed by one centralized command enablement/dispatch contract for navigation, diagnostics destinations, refresh, notification center, onboarding fix-next, and runtime controls. | Improves operator speed and discoverability for primary workflows while preventing menu/palette behavior drift through one shared action contract. | accepted |
| 2026-02-26 | Bootstrap Loading Continuity | Added neutral runtime `Checking` status during initial lifecycle bootstrap and standardized skeleton placeholders for first-load panel states (Inspect, Communications, Automation, Approvals, Tasks, Channels, Connectors, Models). | Prevents startup degraded/empty-state flicker, preserves visual continuity, and keeps status transitions deterministic while daemon queries initialize. | accepted |
| 2026-02-26 | Draft Save/Discard Guarding | Added section-level dirty indicators and `Save All`/`Discard All` controls for Channels/Connectors/Models, plus cross-section navigation discard confirmation when unsaved drafts exist. | Prevents accidental config loss during multi-panel setup/remediation while keeping draft editing behavior explicit and consistent across configurable panels. | accepted |
| 2026-02-26 | High-Impact Action Safety Rails | Standardized confirmation-first UX for high-impact daemon lifecycle/retention/revoke/permission/policy-save actions, with short-lived undo affordances for reversible daemon start/stop actions. | Reduces accidental high-impact mutations while keeping workflows fast and recoverable for common reversible lifecycle operations. | accepted |
| 2026-02-26 | Connector Canonical ID UI Contract | Standardized logical connector card identity on canonical daemon connector IDs (`twilio`, `imessage`, `builtin.app`, etc.) and constrained legacy aliases to compatibility-only merging into those canonical cards. | Prevents duplicate connector cards during migration and keeps action/config/test routing deterministic by anchoring card identity to stable connector contracts. | accepted |
| 2026-02-26 | Channel Mapping Source Of Truth | Standardized logical channel connector rollups and primary legacy implementation selection to be mapping-driven from daemon channel-connector bindings (`/v1/channels/mappings/list`) instead of connector capability/name heuristics. | Keeps channel cards aligned to workspace-scoped routing intent and avoids connector attribution drift when multiple compatible connectors are present. | accepted |
| 2026-02-26 | Communications Logical Grouping + Connector Attribution | Standardized Communications inbox presentation on logical channels (`App`, `Message`, `Voice`) and required connector attribution to use daemon-provided `connector_id` fields (no `external_ref` inference). | Keeps channel UX aligned to canonical logical model while preserving connector-separated operational context under `Message`, reducing attribution drift and parsing ambiguity. | accepted |
| 2026-02-26 | Communications Outbound Compose Flows | Added in-panel outbound compose actions (`New Message`, `Reply`, `Start Call`) using daemon `/v1/comm/send`, with thread-context prefill hints and deterministic send-status/refresh behavior. | Closes the inbox-to-action loop so users can execute outbound communication directly from communications context without leaving the panel. | accepted |
| 2026-02-26 | Communications Triage + Compact Scan | Added deterministic communications triage states (`Unread`, `New`, `Follow Up`, `Handled`), one-click triage actions (`Mark Handled`/`Reopen`, `Follow Up`/`Clear Follow Up`, `Open Task Draft`), and workspace-scoped `Compact Scan` mode for high-volume inbox review. | Improves high-frequency inbox throughput while preserving connector/thread clarity and enabling direct follow-up task drafting without leaving communications context. | accepted |
| 2026-02-26 | Logical Channel Card UX | Standardized `Channels` UI to logical cards (`App`, `Message`, `Voice`) with compatibility mapping from legacy daemon channel IDs plus mapped-connector health/reason/action rollups and `Open Connectors` remediation fallback. | Removes provider-specific card sprawl from core channel UX while preserving compatibility during daemon migration and keeps connector-level remediation discoverable from channel context. | accepted |
| 2026-02-25 | Collapsible Card Defaults | Standardized all collapsible cards/disclosure surfaces to load collapsed by default (`Channels`, `Connectors`, `Models`, and future card-style disclosures) unless an explicit exception is documented. | Reduces visual clutter on first load, improves scanability in dense operational panels, and gives one consistent interaction baseline across current and future collapsible surfaces. | accepted |
| 2026-02-25 | Identity Devices/Sessions Governance | Added daemon-backed Configuration inventory surfaces for identity devices and sessions with per-session revoke controls and deterministic filter/status semantics. | Keeps workspace access hygiene visible in-app and gives operators direct session revocation controls without CLI-only flows. | accepted |
| 2026-02-24 | UI Spec Setup | Created canonical UI spec doc using `spec.md`-style structure and MVP app-surface contracts. | Establishes a single place to capture and evolve UI decisions before implementation. | accepted |
| 2026-02-24 | Taskbar Menu | Added daemon lifecycle controls (install/uninstall/start/stop), app window open/close, and app exit actions as required taskbar-menu contract. | Makes daemon operations and app lifecycle accessible from a minimal always-available menu surface. | accepted |
| 2026-02-24 | Window Layout + Nav | Locked main-window split layout with collapsible sidebar, explicit section ordering (`Configuration`, `Chat`, `Automation`, `Approvals`, `Tasks`, `Inspect`, `Channels`, `Connectors`, `Models`), and sidebar footer status indicators. | Defines canonical app-shell behavior and navigation information architecture before UI build. | accepted |
| 2026-02-24 | Panel Behavior | Locked chat input behavior (`Enter` send, `Shift+Enter` newline), inspect LIFO log formatting, channel/connector card model, and connector-level Apple TCC permission management controls. | Sets implementation-level UX contracts for core high-frequency workflows and operability. | accepted |
| 2026-02-24 | Lifecycle + Cards + TCC States | Confirmed direct UI daemon lifecycle initiation with native authorization handling, initial channel-card baseline (later superseded by logical channel cards), and connector permission action-state behavior (`Request Permission` only when needed, `Open System Settings` always enabled). | Closes implementation ambiguity for taskbar operations, clutter control, and connector permission UX. | accepted |
| 2026-02-24 | macOS-style Lifecycle Placement | Adopted Apple-like flow where taskbar menu remains minimal for daily actions (`start/stop/open/quit`) and daemon `install/uninstall` lives in `Configuration > Advanced`, with contextual setup/repair actions shown in menu only when needed. | Aligns with native macOS app conventions while preserving direct UI control over daemon lifecycle. | accepted |
| 2026-02-24 | Placeholder-First UI Iteration | Approved placeholder-first UI execution: ship production-grade layout/visuals first, wire to daemon endpoints as backend migration tasks complete. | Enables fast look/feel iteration without blocking on daemon API completion. | accepted |
| 2026-02-24 | UI Integration Defaults | Confirmed initial UI/daemon auth strategy uses assistant access token; inspect tab can ship as empty-state panel until inspect APIs are ready. | Reduces early integration friction while preserving explicit roadmap for later hardening. | accepted |
| 2026-02-24 | Direct Permission UX | Confirmed connector permission UX requests permissions directly when supported and always provides a one-click path to System Settings. | Aligns with native macOS permission flow and minimizes user friction. | accepted |
| 2026-02-24 | Project Structure | Confirmed hybrid app structure (Xcode app host + Swift package modules) for stable install identity and modular UI/runtime code organization. | Balances production-grade macOS packaging/signing needs with maintainable modular code structure. | accepted |
| 2026-02-24 | Tahoe Visual + Interaction Modernization | Finalized modern Tahoe-style visual system across shell/menu/panels with shared materials/tokens, stronger hierarchy, and reduced-motion-safe hover/pressed interaction states. | Aligns UI quality with current macOS design language while preserving existing daemon-backed functionality. | accepted |
| 2026-02-24 | Placeholder + Runtime State Cohesion | Standardized section-specific placeholder surfaces for `Automation`/`Approvals`/`Tasks` and unified runtime-state messaging across core panels using one shared banner contract. | Preserves a complete-feel UI during phased backend wiring and avoids conflicting offline/degraded guidance across sections. | accepted |
| 2026-02-24 | Provider/Model + Automation Daemon Wiring | Wired provider/model readiness to daemon provider/model APIs and promoted `Automation` from placeholder to daemon trigger-inventory panel while retaining deterministic empty/error UX. | Aligns high-impact settings/automation surfaces with available backend contracts (including Anthropic/Google parity) without regressing Tahoe UI coherence. | accepted |
| 2026-02-24 | Model Catalog + Policy Visibility | Added explicit model catalog and routing-policy visibility requirements using daemon model APIs. | Makes provider parity actionable by surfacing model availability and task-class route policy state directly in-app for operator debugging. | accepted |
| 2026-02-24 | Models IA Refactor | Moved provider/model readiness and catalog/policy surfaces from `Configuration` into a dedicated `Models` section positioned after `Connectors`. | Separates runtime/auth/daemon controls from model-operations visibility and improves navigation clarity as model controls expand. | accepted |
| 2026-02-24 | Models Provider Card Layout | Refactored Models to provider-scoped expandable cards so each provider owns its model catalog/policy details inside one card surface. | Matches established Channels/Connectors card patterns and improves readability as provider-specific data grows. | accepted |
| 2026-02-24 | Automation Simulation Controls | Added explicit UI requirements for schedule and comm-event simulation actions backed by daemon automation run APIs with deterministic status summaries. | Enables operators to validate automation evaluation paths directly in the app without external CLI tooling. | accepted |
| 2026-02-25 | Communications Inbox Section | Added a dedicated `Communications` sidebar destination and panel contract for daemon-backed thread/event/call-session timelines with sender/recipient metadata, filters, and cross-panel drill-ins. | Separates communication observability from workflow/automation views and gives operators a first-class inbox for message/call context without overloading Chat/Inspect. | accepted |
| 2026-02-24 | Approvals Inbox Daemon Wiring | Replaced `Approvals` placeholder scaffolding with daemon-backed pending/recent inbox rendering via `/v1/approvals/list`, including risk rationale, principal context, and decision metadata. | Closes a core workflow visibility gap and aligns approvals UX with persisted daemon-backed control-plane data. | accepted |
| 2026-02-24 | Tasks List Daemon Wiring | Replaced `Tasks` placeholder scaffolding with daemon-backed task/run list rendering via `/v1/tasks/list`, including workflow state, priority, timestamps, principal context, and last-error metadata. | Completes workflow execution visibility in-app and aligns task queue monitoring with persisted daemon run data. | accepted |
| 2026-02-24 | Startup Reliability Hardening | Removed seeded preview runtime data from inspect/channels/connectors, added empty/loading states for those sections, and delayed runtime warning banners until the first lifecycle probe resolves. | Reduces false-negative first-launch UX, keeps status messaging trustworthy, and avoids decode-drift startup regressions from partially populated daemon lifecycle payloads. | accepted |
| 2026-02-24 | Approvals Action Wiring | Added in-card pending approval decision controls (actor, phrase, rationale) and wired submit flow to daemon approval decision API with explicit `GO AHEAD` enforcement for approve actions. | Closes a core workflow loop by allowing approval execution from UI while preserving destructive-action phrase policy and deterministic per-row submit/error feedback. | accepted |
| 2026-02-24 | Tasks Run-Detail Drill-In | Added `Tasks` run-detail modal with step/artifact/audit rendering and wired detail fetch to daemon inspect-run API for rows with run ids. | Restores explainability for execution workflows directly from task list rows without requiring manual inspect-tab correlation. | accepted |
| 2026-02-24 | Native Connector Permission Requests | Replaced connector permission toggles with native request flows (Calendar via EventKit, Browser via Accessibility prompt, Mail/Finder via Apple Events automation prompt) plus connector-specific System Settings deep links. | Aligns connector card behavior with macOS permission UX conventions and removes false-positive local toggle states. | accepted |
| 2026-02-24 | Configuration Retention/Context + Principal Controls | Added Configuration controls for principal selection, retention purge/compaction, and context sample/tune actions backed by daemon delegation/retention/context APIs with deterministic status copy. | Makes runtime maintenance/tuning user-accessible in-app and removes remaining placeholder-only behavior from Configuration core workflows. | accepted |
| 2026-02-24 | Chat Realtime Stream + Interrupt Fallback UX | Added chat realtime token-streaming contract (`/v1/realtime/ws`) with in-panel interrupt control and deterministic fallback messaging when realtime is unavailable/disconnected. | Preserves realtime-first chat UX while preventing blank/ambiguous states on transport degradation and keeping cancel affordance explicit. | accepted |
| 2026-02-24 | Automation Trigger Management UX | Added in-app trigger create/edit/delete management surfaces with deterministic form validation and explicit idempotent status messaging for update/delete actions. | Completes the automation panel contract by allowing end-to-end trigger lifecycle control in UI while preserving daemon-owned persistence semantics. | accepted |
| 2026-02-25 | Plugin-Worker Degradation Classification | Classified daemon states with failed plugin workers (while control plane/database stay ready) as degraded runtime for UI messaging, with direct `Open Channels` remediation in setup surfaces/taskbar and no generic setup-repair onboarding block. | Prevents false-negative “daemon setup needs repair” UX when the daemon is reachable but optional workers are degraded, while preserving actionable diagnostics navigation. | accepted |
| 2026-02-25 | Inline Approval Evidence Disclosure | Added lazy-loaded approval evidence disclosure that surfaces step context, step input/output summaries (best effort from audit payloads), and related artifacts/audit snippets directly in approval cards. | Improves approval decision confidence and reduces context switching by keeping core evidence visible in the inbox workflow. | accepted |
| 2026-02-25 | Configuration Identity Hub | Added a daemon-backed identity hub in Configuration with workspace switching, active-context metadata, workspace/principal directory inventory, and actor-handle mappings. | Aligns UI context controls with canonical identity objects and gives operators direct visibility into workspace/principal boundaries without CLI context switches. | accepted |
| 2026-02-25 | Chat + Automation Acting-As Selector | Added in-flow `Acting As` selectors for Chat and Automation authoring, with identity-directory-backed validation copy and payload persistence of selected actor IDs for delegation-safe execution context. | Makes principal delegation intent explicit at submission time and reduces hidden-context execution mistakes. | accepted |
| 2026-02-25 | Configuration Delegation Rule Management | Added daemon-backed delegation inventory with scoped grant creation and per-rule revoke controls in Configuration. | Keeps principal delegation governance in-app with clear scope visibility and deterministic rule lifecycle feedback. | accepted |
| 2026-02-25 | Channel/Connector Config + Test Controls | Added in-card editable configuration and explicit health test workflows for Channels and Connectors, backed by daemon config upsert and test-operation APIs. | Gives operators direct remediation and validation loops from the same card surface without leaving runtime status context. | accepted |
| 2026-02-25 | Channel Delivery Policy + Attempt Timeline UX | Added channel delivery-policy list/edit controls and communications delivery-attempt timeline visibility wired to daemon comm policy/attempt APIs with deterministic thread-scoped query behavior. | Exposes fallback/retry routing outcomes and policy tuning in-app so operators can validate delivery behavior without CLI workflows. | accepted |
| 2026-02-25 | Typed ON_COMM_EVENT Trigger Builder | Replaced raw ON_COMM_EVENT filter JSON editing with typed source/filter controls, inline normalization hints, and normalized payload preview in Automation create/edit flows. | Reduces authoring errors, makes trigger semantics readable before submit, and aligns automation UX with typed daemon trigger contracts. | accepted |
| 2026-02-25 | Runtime Supervisor Timeline Diagnostics | Added a daemon-backed runtime timeline in Configuration for plugin lifecycle history, trend summaries, and direct diagnostics drill-ins to Inspect plus plugin-kind destinations. | Makes plugin restart/failure cadence visible in one place and reduces operator time-to-diagnosis for degraded worker states. | accepted |
| 2026-02-25 | Memory Browser + Retrieval Context Inspector | Added daemon-backed Configuration diagnostics surfaces for memory inventory, compaction-candidate previews, retrieval document browsing, and document-scoped chunk inspection with principal/source filters. | Gives operators first-class visibility into context memory and retrieval artifacts in-app, reducing CLI-only troubleshooting for context quality and compaction readiness. | accepted |
| 2026-02-25 | Capability Grants + Trust Receipt Governance | Added Configuration governance sections for capability-grant lifecycle controls and communication trust-receipt inventories (webhook + ingest) with filterable lists, deterministic status handling, and direct Inspect drill-ins from receipt audit context. | Keeps security/trust governance visible in-app with actionable audit evidence and without CLI-only workflows. | accepted |
| 2026-02-25 | Models Route Simulation + Explainability | Added `Models` route-analysis controls for task-class + optional principal simulation and explainability queries, rendering selected-route summaries, reason-code traces, and fallback-chain context from daemon route-analysis APIs. | Gives operators in-app routing transparency and policy-debug context without relying on daemon/CLI-only traces. | accepted |

## 18) Status

This document is the canonical UI specification to build from for app-surface behavior and UX flows.
