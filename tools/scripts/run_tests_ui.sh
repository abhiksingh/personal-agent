#!/usr/bin/env bash
set -u
set -o pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
if [[ ! -f "${ROOT}/AGENTS.md" && -f "${SCRIPT_DIR}/../../AGENTS.md" ]]; then
  ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
fi
cd "$ROOT"

LOG_FILE=""
SKIP_BUILD=false
OPEN_APP=false
RUN_APP_HOST_SMOKE=false
RUN_VISUAL_REGRESSION=false
UPDATE_VISUAL_BASELINES=false
XCODE_PROJECT="$ROOT/source/apps/macos/app-host/PersonalAgent.xcodeproj"
XCODE_SCHEME="PersonalAgent"
APP_NAME="PersonalAgent.app"
APP_DERIVED_DATA_PATH="${APP_DERIVED_DATA_PATH:-$ROOT/out/build/xcode-derived-data}"
UI_SWIFTPM_SCRATCH_PATH="${UI_SWIFTPM_SCRATCH_PATH:-$ROOT/out/build/swiftpm/personal-agent-ui}"
UI_TEST_DEFAULTS_SUITE="${PA_UI_DEFAULTS_SUITE:-com.personalagent.app.tests}"
VISUAL_BASELINE_DIR_OVERRIDE="${PA_UI_VISUAL_BASELINE_DIR:-}"
VISUAL_UPDATE_MARKER_FILE="${PA_UI_VISUAL_UPDATE_MARKER_FILE:-/tmp/personalagent-ui-update-baselines.flag}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --log-file)
      LOG_FILE="${2:-}"
      shift 2
      ;;
    --skip-build)
      SKIP_BUILD=true
      shift
      ;;
    --open-app)
      OPEN_APP=true
      shift
      ;;
    --run-app-host-smoke)
      RUN_APP_HOST_SMOKE=true
      shift
      ;;
    --run-visual-regression)
      RUN_VISUAL_REGRESSION=true
      shift
      ;;
    --update-visual-baselines)
      UPDATE_VISUAL_BASELINES=true
      shift
      ;;
    *)
      echo "unknown argument: $1" >&2
      echo "usage: $0 [--log-file <path>] [--skip-build] [--open-app] [--run-app-host-smoke] [--run-visual-regression] [--update-visual-baselines]" >&2
      exit 2
      ;;
  esac
done

if [[ "$UPDATE_VISUAL_BASELINES" == "true" && "$RUN_VISUAL_REGRESSION" == "false" ]]; then
  RUN_VISUAL_REGRESSION=true
fi

LOG_DIR="$ROOT/out/logs/manual-tests"
mkdir -p "$LOG_DIR"
if [[ -z "$LOG_FILE" ]]; then
  STAMP="$(date +%Y%m%d-%H%M%S)"
  LOG_FILE="$LOG_DIR/tests-ui-$STAMP.log"
fi

exec > >(tee -a "$LOG_FILE") 2>&1

echo "UI manual tests started at $(date -u +'%Y-%m-%dT%H:%M:%SZ')"
echo "Repository root: $ROOT"
echo "Log file: $LOG_FILE"
echo "Skip build: $SKIP_BUILD"
echo "Open app: $OPEN_APP"
echo "Run app-host smoke tests: $RUN_APP_HOST_SMOKE"
echo "Run visual regression tests: $RUN_VISUAL_REGRESSION"
echo "Update visual baselines: $UPDATE_VISUAL_BASELINES"
if [[ -n "$VISUAL_BASELINE_DIR_OVERRIDE" ]]; then
  echo "Visual baseline dir override: $VISUAL_BASELINE_DIR_OVERRIDE"
else
  echo "Visual baseline dir override: <auto application-support path>"
fi
echo "Visual baseline update marker: $VISUAL_UPDATE_MARKER_FILE"
echo "UI defaults suite: $UI_TEST_DEFAULTS_SUITE"
echo "SwiftPM scratch path: $UI_SWIFTPM_SCRATCH_PATH"
echo "Tip: run ./tools/scripts/run_tests_all.sh to execute UI + CLI + daemon runners together."

FAILURES=0
STEPS=0

run_cmd() {
  local label="$1"
  shift
  STEPS=$((STEPS + 1))
  echo
  echo "================================================================"
  echo "STEP $STEPS: $label"
  echo "CMD: $*"
  echo "----------------------------------------------------------------"
  "$@"
  local rc=$?
  echo "RESULT: exit_code=$rc"
  if [[ "$rc" -ne 0 ]]; then
    FAILURES=$((FAILURES + 1))
  fi
  return 0
}

run_eval() {
  local label="$1"
  shift
  local cmd="$*"
  local start_dir="$PWD"
  STEPS=$((STEPS + 1))
  echo
  echo "================================================================"
  echo "STEP $STEPS: $label"
  echo "CMD: $cmd"
  echo "----------------------------------------------------------------"
  eval "$cmd"
  local rc=$?
  cd "$start_dir" >/dev/null 2>&1 || true
  echo "RESULT: exit_code=$rc"
  if [[ "$rc" -ne 0 ]]; then
    FAILURES=$((FAILURES + 1))
  fi
  return 0
}

resolve_debug_app_bundle() {
  local bundle="$APP_DERIVED_DATA_PATH/Build/Products/Debug/$APP_NAME"
  if [[ -d "$bundle" ]]; then
    printf "%s\n" "$bundle"
    return 0
  fi
  return 1
}

# 1) Prerequisites
run_cmd "Check macOS environment" sw_vers
run_cmd "Check Xcode toolchain" xcodebuild -version
run_cmd "Check xcodegen" xcodegen --version

# 2) Build + harness baseline
if [[ "$SKIP_BUILD" == "true" ]]; then
  run_eval "Skip project generation + build" 'echo "build skipped by --skip-build"'
else
  run_eval "Generate Xcode project" 'cd "$ROOT/source/apps/macos/app-host" && xcodegen generate'
  run_cmd "Build PersonalAgent Debug app bundle" \
    xcodebuild \
      -project "$XCODE_PROJECT" \
      -scheme "$XCODE_SCHEME" \
      -configuration Debug \
      -derivedDataPath "$APP_DERIVED_DATA_PATH" \
      CODE_SIGNING_ALLOWED=NO \
      build
fi
run_eval "Run PersonalAgentUI package tests (includes client-integration fixture contract decode checks)" 'mkdir -p "$UI_SWIFTPM_SCRATCH_PATH" && cd "$ROOT/source/apps/macos/app-host/Packages/PersonalAgentUI" && PA_UI_DEFAULTS_SUITE="$UI_TEST_DEFAULTS_SUITE" swift test --scratch-path "$UI_SWIFTPM_SCRATCH_PATH"'
if [[ "$RUN_APP_HOST_SMOKE" == "true" ]]; then
  run_cmd "Run app-host XCUITest smoke suite (includes command-palette, autonomous chat-to-action, and cross-channel drill-in parity coverage)" \
    xcodebuild \
      -project "$XCODE_PROJECT" \
      -scheme "$XCODE_SCHEME" \
      -configuration Debug \
      -derivedDataPath "$APP_DERIVED_DATA_PATH" \
      CODE_SIGNING_ALLOWED=YES \
      CODE_SIGN_IDENTITY=- \
      -only-testing:PersonalAgentHostUITests \
      test
else
  run_eval "Skip app-host XCUITest smoke suite" 'echo "app-host smoke tests skipped (use --run-app-host-smoke to enable)"'
fi
if [[ "$RUN_VISUAL_REGRESSION" == "true" ]]; then
  if [[ "$UPDATE_VISUAL_BASELINES" == "true" ]]; then
    run_eval "Enable visual baseline update marker" 'mkdir -p "$(dirname "$VISUAL_UPDATE_MARKER_FILE")" && touch "$VISUAL_UPDATE_MARKER_FILE"'
    run_cmd "Run app-host visual regression suite (update baselines)" \
      env PA_UI_UPDATE_VISUAL_BASELINES=1 \
      PA_UI_VISUAL_BASELINE_DIR="$VISUAL_BASELINE_DIR_OVERRIDE" \
      PA_UI_VISUAL_UPDATE_MARKER_FILE="$VISUAL_UPDATE_MARKER_FILE" \
      xcodebuild \
        -project "$XCODE_PROJECT" \
        -scheme "$XCODE_SCHEME" \
        -configuration Debug \
        -derivedDataPath "$APP_DERIVED_DATA_PATH" \
        CODE_SIGNING_ALLOWED=YES \
        CODE_SIGN_IDENTITY=- \
        -only-testing:PersonalAgentHostUITests/PersonalAgentHostVisualRegressionTests \
        test
    run_eval "Disable visual baseline update marker" 'rm -f "$VISUAL_UPDATE_MARKER_FILE"'
  else
    run_eval "Clear stale visual baseline update marker" 'rm -f "$VISUAL_UPDATE_MARKER_FILE"'
    run_cmd "Run app-host visual regression suite" \
      env PA_UI_VISUAL_BASELINE_DIR="$VISUAL_BASELINE_DIR_OVERRIDE" \
      PA_UI_VISUAL_UPDATE_MARKER_FILE="$VISUAL_UPDATE_MARKER_FILE" \
      xcodebuild \
        -project "$XCODE_PROJECT" \
        -scheme "$XCODE_SCHEME" \
        -configuration Debug \
        -derivedDataPath "$APP_DERIVED_DATA_PATH" \
        CODE_SIGNING_ALLOWED=YES \
        CODE_SIGN_IDENTITY=- \
        -only-testing:PersonalAgentHostUITests/PersonalAgentHostVisualRegressionTests \
        test
  fi
else
  run_eval "Skip app-host visual regression suite" 'echo "visual regression tests skipped (use --run-visual-regression to enable)"'
fi
run_eval "Run harness checks" 'cd "$ROOT" && ./tools/scripts/check_harness.sh'

# 3) Optional app launch
if [[ "$OPEN_APP" == "true" ]]; then
  run_eval "Locate Debug app bundle (repo-local derived data)" '
APP_BUNDLE_PATH="$(resolve_debug_app_bundle)"
echo "APP_BUNDLE_PATH=$APP_BUNDLE_PATH"
test -n "$APP_BUNDLE_PATH"
test -f "$APP_BUNDLE_PATH/Contents/Resources/Assets.car"
test -f "$APP_BUNDLE_PATH/Contents/Resources/AppIcon.icns"
'
  run_eval "Open PersonalAgent.app" 'open "$APP_BUNDLE_PATH"'
else
  run_eval "Skip app launch" 'echo "app launch skipped (use --open-app to launch the latest Debug build)"'
fi

# 4) Manual handoff guidance
run_eval "Print manual test handoff" '
cat <<'"'"'EOF'"'"'
Next: follow docs/tests-ui.md for the compact index and load only relevant panel files:
- docs/tests-ui/00-overview-and-setup.md
- docs/tests-ui/01-taskbar-and-shell.md
- docs/tests-ui/02-chat.md
- docs/tests-ui/03-communications-and-tasks.md
- docs/tests-ui/04-inspect.md
- docs/tests-ui/05-channels.md
- docs/tests-ui/06-connectors.md
- docs/tests-ui/07-configuration.md
- docs/tests-ui/08-models-and-model-to-chat-e2e.md
- docs/tests-ui/09-automation-visual-and-cleanup.md
- docs/tests-ui/10-cross-channel-lifecycle-parity.md
- docs/tests-ui/11-auth-rate-limit-realtime-hardening.md
- docs/tests-ui/12-autonomous-unified-turn-workflows.md
- docs/tests-ui/13-unsigned-dmg-install-gatekeeper-tcc.md

Temporary directive (2026-03-03): automation-driven UI tests (app-host/XCUITest) are paused for active task validation. Execute manual checklist flows instead.

Use docs/tests-ui/full.md only when you need cross-panel reconciliation against the full baseline guide.
For orchestration-v2 cutover checks, validate canonical typed timeline behavior, no Ask/Act controls, guided remediation actions (not raw daemon/provider error text), deterministic `Route + Tool Explainability` loading/ready/failure states in `Effective Workflow Context`, and no legacy orchestration shims.
For typed payload-contract regressions, validate realtime/chat/ui-status updates continue rendering canonical states when optional payload fields are omitted/reordered (no transient raw decode-error fallback copy).
For typed panel-problem checks, validate `auth_scope` and `rate_limit_exceeded` states in Chat/Models/Channels/Connectors/Automation/Approvals/Tasks render the shared remediation card with `Open Configuration`, `Retry`, and `Open Inspect`, including explicit retry disabled reasons when refresh is already in flight.
For AppShellState decomposition tasks, run `swift test --package-path source/apps/macos/app-host/Packages/PersonalAgentUI --scratch-path out/build/swiftpm/personal-agent-ui --filter AppShellStateDecompositionParityTests` plus `swift test --package-path source/apps/macos/app-host/Packages/PersonalAgentUI --scratch-path out/build/swiftpm/personal-agent-ui --filter AppCommunicationsStoreTests`, `swift test --package-path source/apps/macos/app-host/Packages/PersonalAgentUI --scratch-path out/build/swiftpm/personal-agent-ui --filter AppConnectionConfigStoreTests`, `swift test --package-path source/apps/macos/app-host/Packages/PersonalAgentUI --scratch-path out/build/swiftpm/personal-agent-ui --filter AppModelsRouteStoreTests`, `swift test --package-path source/apps/macos/app-host/Packages/PersonalAgentUI --scratch-path out/build/swiftpm/personal-agent-ui --filter AppWorkflowQueueStoreTests`, `swift test --package-path source/apps/macos/app-host/Packages/PersonalAgentUI --scratch-path out/build/swiftpm/personal-agent-ui --filter AppInspectStoreTests`, and `swift test --package-path source/apps/macos/app-host/Packages/PersonalAgentUI --scratch-path out/build/swiftpm/personal-agent-ui --filter ConfigurationDraftStoreTests` before additional extraction work to lock selection/reducer/event parity against extracted stores.
For Configuration decomposition tasks (`U-262+`), confirm `ConfigurationPanelView.swift` remains shell/router-only, mode-owned rendering stays bounded in dedicated `ConfigurationPanelView+*Mode.swift` files, delegation/capability draft orchestration continues through `ConfigurationDraftStore` with deterministic validation + disabled-reason behavior, and shared inventory/card primitives keep identity/trust/memory/retrieval loading/empty/has-more behavior consistent.
For runtime/setup de-dupe checks, validate `Configuration > Setup` owns detailed readiness/remediation while taskbar, shell ribbons, `Home`, and onboarding-gated surfaces stay summary/CTA-only without repeated diagnostics paragraphs, and verify guided-session ribbons in workflow panels appear only after setup readiness is complete.
For taskbar lifecycle checks, validate `Close Window` moves app to menu-bar-only mode (no Dock/app-switcher presence) until `Open Window`, and `Quit` closes open windows while issuing a best-effort daemon stop request before app termination.
For unsigned distribution trust checks, validate `Home` and `Configuration > Setup` show explicit Gatekeeper override guidance (`Control-click/Right-click Open`, `Open Anyway`) plus deterministic `Open Security Settings` and `Retry Setup Checks` remediation actions before daemon troubleshooting.
For Configuration essential-first checks, validate `Setup Matrix Details` and `Runtime Details` are collapsed by default in `Setup`, `Identity Devices and Sessions`/`Delegation Rules` are collapsed in `Workspace`, `Daemon Lifecycle Controls` is collapsed in `Advanced`, and unavailable primary setup actions surface deterministic guidance copy.
For unsigned local/internal install-path checks, validate `Configuration > Advanced` `Install`/`Repair` actions execute bundled helper setup and that running outside `/Applications` surfaces deterministic remediation copy to move `PersonalAgent.app` into `/Applications`.
For chat route/workflow de-dupe checks, validate provider/model/source route metadata is owned by `Chat > Effective Workflow Context`, while header/composer/explainability sections avoid repeating identical route strings.
For chat remediation flow checks, validate route/failure cards expose one primary `Fix and Continue` action, keep explicit fallback controls, and auto-resume pending chat intent when returning from `Models`/`Configuration`.
For approval ownership de-dupe checks, validate `Approvals` owns the full decision form/evidence workflow and `Chat` approval rows keep `Open Approvals` as the primary decision path; low-risk (`policy`) rows may expose bounded inline fast-path controls with explicit confirm + undo-draft states.
For chat acting-as progressive disclosure checks, validate composer `Advanced Override` is collapsed by default with `Auto` summary and auto-reveals when a non-default actor is selected or delegation validation requires correction.
For communications row de-dupe checks, validate channel-group headers own channel context (`channel + count`) while continuity/thread/event/attempt rows avoid repeating channel/connector/persona metadata across badges, pills, and `Details`.
For tasks/automation/inspect de-dupe checks, validate each panel keeps one canonical status/helper owner (header or one supplementary line), avoids repeated empty-state/header status sentences, and uses `Filters apply after ... are loaded` copy when inventories are empty.
For channels/connectors de-dupe checks, validate each card renders one helper-status owner under action rows (no stacked duplicate reason lines), and keep connector reason/action diagnostics owned by details surfaces rather than repeated helper paragraphs.
For models route de-dupe checks, validate `Chat Model Route` plus one shared route-selection summary own provider/model/source context, and avoid repeating the same route block inside both simulation and explainability result groups.
For models quickstart checks, validate `Connect Provider -> Choose and Enable Model -> Set Chat Route -> Test in Chat` progression shows one current-step primary action, reuses existing provider/catalog/route mutations, and deep-links to Chat with route chip plus seeded draft only when composer is empty.
For cross-panel empty-state de-dupe checks, validate `Inspect`, `Channels`, `Connectors`, `Models`, `Tasks`, `Approvals`, and `Communications` suppress empty-state status text when it duplicates the panel/section status owner subtitle.
For realtime resilience checks, validate chat/websocket fallback copy distinguishes capacity rejection, stale-session/disconnect paths, and verify `Chat > Actions > Retry Realtime Stream` recovers connection status from `Degraded` back to `Connected` when available.
For chat transcript ergonomics checks, validate auto-follow only while new timeline items are appended, manual scrolling remains free after turn activity settles, user-message bubbles hug content (no fixed wide expansion for short prompts), assistant rows expand to available transcript width for wide markdown blocks (tables/code), markdown rendering stays spec-aligned, fenced code (including json/yaml) appears in monospaced code blocks with language labels, markdown image links render inline with deterministic loading/failure placeholders, and long-running healthy turns do not end in timeout-driven truncated assistant text.
For chat transport-blip recovery checks, validate completed turns reconcile from `/v1/chat/history` on transient receipt interruptions and avoid false `Could not reach daemon while loading Chat` terminal copy when history recovery succeeds.
For cross-channel lifecycle parity checks, validate smoke-fixture outcomes across canonical workflows (`send_email` awaiting approval, `send_message` completed, `find_files` blocked permission, `browse_web` completed) plus continuity drill-ins for `app|message|voice` (`Open Related Tasks`/`Open Related Inspect`/`Open Chat`) with source ribbons.
Use `docs/tests-ui/10-cross-channel-lifecycle-parity.md` as the canonical manual replacement for cross-channel lifecycle parity checks while automation remains paused; track automation stabilization under `U-234`.
Use `docs/tests-ui/11-auth-rate-limit-realtime-hardening.md` as the canonical manual replacement for auth-scope/rate-limit/realtime-hardening regressions while automation remains paused.
Use `docs/tests-ui/12-autonomous-unified-turn-workflows.md` as the canonical manual replacement for autonomous unified-turn lifecycle regression checks while automation remains paused.
Use `docs/tests-ui/13-unsigned-dmg-install-gatekeeper-tcc.md` as the canonical clean-machine validation path for unsigned DMG install, Gatekeeper override, daemon launch identity, and `Personal Agent Daemon` TCC attribution checks.
For persona/channel response-shaping checks, validate `Effective Workflow Context` shaping badges/details and Configuration `Chat Persona Policy` test-profile guidance (`Test Channel`/`Profile` plus channel-mismatch helper copy).
For density-mode regressions, explicitly validate `Simple` plain-language workflow summary copy and `Advanced` operator wording parity in Chat/Approvals/Tasks.
For `Home` advanced-density coverage, validate `First-Success Funnel Diagnostics` shows aggregate completion summary plus per-milestone source/timestamp evidence (or pending state).
For `Inspect` regression coverage, validate `Open Gallery` renders deterministic shared-component references and `Back to Activity` restores status/search/live-tail controls.
For command-palette regressions, validate global `Do` entrypoint (`do ` prefill), `Do:` outcome ranking for natural-language intent queries, and object search/open behavior for task/approval/thread/connector/model queries.
For accessibility regressions, validate command-palette auto-focus, deterministic search-field accessibility IDs (`communications-search-field`, `approvals-search-field`, `tasks-search-field`), and the drill-in dismiss VoiceOver label/hint contract.
EOF
'

echo
echo "================================================================"
echo "UI manual tests completed at $(date -u +'%Y-%m-%dT%H:%M:%SZ')"
echo "Log file: $LOG_FILE"
echo "Total steps: $STEPS"
echo "Failures: $FAILURES"
if [[ "$FAILURES" -eq 0 ]]; then
  echo "STATUS: PASS"
  echo "================================================================"
  exit 0
else
  echo "STATUS: FAIL"
  echo "================================================================"
  exit 1
fi
