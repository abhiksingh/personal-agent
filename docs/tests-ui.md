# PersonalAgent UI Manual Test Guide (Index)

This index keeps UI test context compact. Read only the sections relevant to the task.

## Quick Run

```bash
./tools/scripts/run_tests_ui.sh
```

For full baseline details from previous format, see [Full Guide](./tests-ui/full.md).

Temporary execution directive (2026-03-03): automation-driven UI tests (`app-host` / `XCUITest`) are paused. Use manual panel and checklist flows below.

## Decomposition Guardrail (U-255+)

Before starting additional `AppShellState` extraction tasks (`U-256` onward), run:

```bash
swift test --package-path source/apps/macos/app-host/Packages/PersonalAgentUI --scratch-path out/build/swiftpm/personal-agent-ui --filter AppShellStateDecompositionParityTests
swift test --package-path source/apps/macos/app-host/Packages/PersonalAgentUI --scratch-path out/build/swiftpm/personal-agent-ui --filter AppCommunicationsStoreTests
swift test --package-path source/apps/macos/app-host/Packages/PersonalAgentUI --scratch-path out/build/swiftpm/personal-agent-ui --filter AppConnectionConfigStoreTests
swift test --package-path source/apps/macos/app-host/Packages/PersonalAgentUI --scratch-path out/build/swiftpm/personal-agent-ui --filter AppModelsRouteStoreTests
swift test --package-path source/apps/macos/app-host/Packages/PersonalAgentUI --scratch-path out/build/swiftpm/personal-agent-ui --filter AppWorkflowQueueStoreTests
swift test --package-path source/apps/macos/app-host/Packages/PersonalAgentUI --scratch-path out/build/swiftpm/personal-agent-ui --filter AppInspectStoreTests
swift test --package-path source/apps/macos/app-host/Packages/PersonalAgentUI --scratch-path out/build/swiftpm/personal-agent-ui --filter ConfigurationDraftStoreTests
```

These parity suites lock selection/reducer/event behavior against extracted store contracts (`AppShellNavigationStore`, `AppRuntimeLifecycleStore`, `AppPanelProblemStore`, `ChatTurnExecutionStore`, `AppCommunicationsStore`, `AppConnectionConfigStore`, `AppModelsRouteStore`, `AppWorkflowQueueStore`, `AppInspectStore`, `ConfigurationDraftStore`) so shell behavior stays stable during decomposition.
For `Configuration` decomposition work (`U-262+`), verify `ConfigurationPanelView.swift` remains shell/router-only, mode-owned rendering remains bounded in dedicated `ConfigurationPanelView+*Mode.swift` files, draft orchestration continues through `ConfigurationDraftStore` helpers with deterministic validation/disabled reasons, and shared inventory/card primitives keep loading/empty/has-more behavior consistent across identity/trust/memory/retrieval diagnostics.

## Read Order (Default)

1. [Overview and Setup](./tests-ui/00-overview-and-setup.md)
2. [Taskbar and App Shell](./tests-ui/01-taskbar-and-shell.md)
3. Read panel-specific guides only for touched surfaces:
   - [Chat](./tests-ui/02-chat.md)
   - [Communications and Tasks](./tests-ui/03-communications-and-tasks.md)
   - [Inspect](./tests-ui/04-inspect.md)
   - [Channels](./tests-ui/05-channels.md)
   - [Connectors](./tests-ui/06-connectors.md)
   - [Configuration](./tests-ui/07-configuration.md)
   - [Models and Models-to-Chat E2E](./tests-ui/08-models-and-model-to-chat-e2e.md)
   - [Automation, Visual Accessibility, and Cleanup](./tests-ui/09-automation-visual-and-cleanup.md)
4. For lifecycle-parity validation while automation is paused:
   - [Cross-Channel Lifecycle Parity (Manual-Only)](./tests-ui/10-cross-channel-lifecycle-parity.md)
5. For auth/rate-limit/realtime resilience regression validation while automation is paused:
   - [Auth Scope + Rate Limit + Realtime Hardening (Manual-Only)](./tests-ui/11-auth-rate-limit-realtime-hardening.md)
6. For end-to-end autonomous unified-turn lifecycle regression validation while automation is paused:
   - [Autonomous Unified-Turn Workflows (Manual-Only)](./tests-ui/12-autonomous-unified-turn-workflows.md)
7. For clean-machine unsigned distribution validation (Gatekeeper override + daemon/TCC attribution):
   - [Unsigned DMG Install + Gatekeeper + TCC Attribution](./tests-ui/13-unsigned-dmg-install-gatekeeper-tcc.md)

## Orchestration-v2 Cutover Focus

When validating orchestration-v2 slices (`T-295` onward), run the Chat panel guide first and explicitly confirm:

- canonical typed timeline rendering (`user_message`, `assistant_message`, `tool_call`, `tool_result`, `approval_request`, `approval_decision`) without message-only fallback views;
- no legacy `Ask`/`Act` mode controls or mode-dependent behavior paths;
- route/planner/tool failures surface guided remediation actions (not raw daemon/provider error text);
- typed workflow-panel problems (`auth_scope`, `rate_limit_exceeded`) surface the shared remediation card with deterministic actions (`Open Configuration`, `Retry`, `Open Inspect`) and explicit disabled/in-flight reason copy;
- realtime/chat/ui-status payload updates continue rendering canonical UI states via typed contract decoding without transient raw decode errors when optional fields are omitted or reordered;
- websocket resilience states classify capacity rejection/stale-session/disconnect fallback transitions with deterministic chat copy and explicit `Retry Realtime Stream` recovery action;
- `Effective Workflow Context` includes deterministic `Route + Tool Explainability` loading/ready/empty/failure state transitions with remediation actions (`Refresh`, `Open Models`, `Open Inspect`);
- global `Do` entrypoint opens command palette with `do ` prefill and routes outcome commands (`Do: Send an Email`, `Do: Create a Task`, `Do: Review Approvals`, `Do: Inspect an Issue`) into canonical owner panels without parallel flows;
- chat-action flows remain deterministic across route-remediation, approval-required, and tool-failure scenarios.
- cross-channel continuity/handoff parity remains deterministic for canonical logical channels (`app|message|voice`) across `Communications` and drill-ins into `Chat`, `Approvals`, `Tasks`, and `Inspect` with source ribbons/chips.
- smoke-fixture lifecycle parity remains deterministic across canonical prompts/channels (`send_email -> awaiting approval`, `send_message -> completed`, `find_files -> blocked permission`, `browse_web -> completed`) after scrolling the timeline to newest visible entries.
- markdown transcript rendering remains spec-aligned (no newline hardcoding), with fenced code formatting and markdown image links rendered inline in chat rows.
- no legacy orchestration shims are assumed in UI assertions (request-text bridge semantics or synthetic execution-origin copy).
- execute manual lifecycle parity checks from `10-cross-channel-lifecycle-parity.md`; track automation follow-up under `U-234` until automation is explicitly re-enabled.
- execute manual auth/rate-limit/realtime hardening checks from `11-auth-rate-limit-realtime-hardening.md` as the canonical regression path while automation is paused.
- execute manual autonomous unified-turn lifecycle checks from `12-autonomous-unified-turn-workflows.md` for end-to-end in-flight/approval/blocked/success/recovery validation.

## Information Ownership and De-Dupe Focus

When validating UI dedupe work (`U-238` onward), confirm each screen keeps one canonical detailed owner for duplicated information classes:

- runtime/setup: `Configuration > Setup` is canonical; taskbar/shell/home/onboarding surfaces are summary + remediation only, and must not repeat paragraph-level diagnostics already visible in setup matrix rows.
  - workflow-panel guided-session ribbon appears only after setup readiness is complete; setup-incomplete workflow panels rely on onboarding/setup owners instead.
  - in `Configuration`, verify first-pass setup stays compact: `Setup Matrix Details` and `Runtime Details` are collapsed by default, and unavailable primary setup actions show deterministic guidance copy.
- chat route/workflow: `Effective Workflow Context` is canonical; header/composer and explainability route sections do not repeat identical provider/model/source route metadata.
- approvals: `Approvals` panel is canonical for full decision form/evidence; chat timeline approval rows remain compact/handoff-oriented.
- communications/channels/connectors/models: row/card top sections keep actionable state concise and avoid repeating the same values in multiple stacked blocks.
  - In `Communications`, channel-group headers own channel context; continuity/thread/event/attempt rows keep concise lifecycle summaries and reserve connector/persona/shaping diagnostics for `Details`.
  - In `Channels`/`Connectors`, each card keeps one helper-status owner (permission/diagnostic gating) and avoids stacking duplicate reason lines beneath action rows.
  - In `Models`, `Chat Model Route` plus one shared route-selection summary own provider/model/source context; simulation/explainability groups focus on decision details without repeating the same route block twice.
- empty states: if a panel header already displays the same status message, empty-state status text is suppressed and only unique remediation copy remains.
  - Suppression is implemented in shared empty-state infrastructure; verify parity in `Inspect`, `Channels`, `Connectors`, `Models`, `Tasks`, `Approvals`, and `Communications`.
- tasks/automation/inspect helper copy: keep one status/helper owner per panel (header or one supplementary line), avoid repeating the same status in empty-state/status stacks, and keep filter-bar empty copy to `Filters apply after ... are loaded`.

## Context-Loading Guidance

- Start with this index and load the minimum panel guide(s) needed.
- Avoid opening [Full Guide](./tests-ui/full.md) unless cross-panel reconciliation is required.
- When updating manual tests, edit the panel-specific file and only update the full guide if a full-document sync is explicitly needed.
