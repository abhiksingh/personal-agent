# UI Tests: Overview and Setup

Source: [full guide](../tests-ui/full.md)
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

Temporary directive (2026-03-03): do not execute automation-driven UI tests for active task validation. Keep this section as reference only and run manual flows instead.

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

Manual testing in this document remains the source for broader UX, visual-polish, accessibility, and platform-integration validation not covered by smoke automation.

## Automated Visual Regression Coverage (App-Host XCUITest)

Temporary directive (2026-03-03): visual/app-host automation is paused for active task validation. Use manual visual checks in section 14 instead.

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
5. With setup now ready and first-session milestones still incomplete, open a workflow panel (for example `Chat`) and verify the compact guided-session ribbon appears.
6. In `Chat`, send one prompt and verify assistant response renders.
7. In `Tasks`, open `New Task`, set `Goal`, submit, and verify `Latest Submitted Task` appears.
8. In `Approvals`, use `Use Required Phrase`, submit `Approve and Continue`, and verify success status copy is shown.
9. Confirm one recovery affordance is visible and actionable during setup (`Fix Next`) or post-action remediation (`Open Related ...`, notification center next action).

Expected:

- First-run users see a deterministic setup path with visible recovery affordances before workflow actions.
- Guided first-session ribboning appears only after setup readiness is complete.
- First successful chat send, task submit, and approval decision can be completed in one continuous flow.
- Recovery/remediation actions are surfaced inline at decision points without requiring hidden/debug-only paths.
