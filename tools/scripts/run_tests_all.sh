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
LOG_DIR="$ROOT/out/logs/manual-tests"
CLI_LIVE_MODE="skip" # skip|strict|off
DAEMON_SKIP_REGRESSION=false
UI_SKIP_BUILD=false
UI_OPEN_APP=false
RUN_CLI=true
RUN_DAEMON=true
RUN_UI=true
RUN_HARNESS=true
FAIL_FAST=false

usage() {
  cat <<'EOF'
Usage: tools/scripts/run_tests_all.sh [options]

Options:
  --log-file <path>          Consolidated suite log path.
  --log-dir <dir>            Directory for per-runner logs.
  --cli-live-mode <mode>     CLI Twilio mode: skip|strict|off (default: skip).
  --daemon-skip-regression   Pass --skip-regression to daemon runner.
  --ui-skip-build            Pass --skip-build to UI runner.
  --ui-open-app              Pass --open-app to UI runner.
  --skip-cli                 Skip CLI runner.
  --skip-daemon              Skip daemon runner.
  --skip-ui                  Skip UI runner.
  --skip-harness             Skip final harness check.
  --fail-fast                Stop immediately on first failure.
  -h, --help                 Show this help text.
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --log-file)
      LOG_FILE="${2:-}"
      shift 2
      ;;
    --log-dir)
      LOG_DIR="${2:-}"
      shift 2
      ;;
    --cli-live-mode)
      CLI_LIVE_MODE="${2:-}"
      shift 2
      ;;
    --daemon-skip-regression)
      DAEMON_SKIP_REGRESSION=true
      shift
      ;;
    --ui-skip-build)
      UI_SKIP_BUILD=true
      shift
      ;;
    --ui-open-app)
      UI_OPEN_APP=true
      shift
      ;;
    --skip-cli)
      RUN_CLI=false
      shift
      ;;
    --skip-daemon)
      RUN_DAEMON=false
      shift
      ;;
    --skip-ui)
      RUN_UI=false
      shift
      ;;
    --skip-harness)
      RUN_HARNESS=false
      shift
      ;;
    --fail-fast)
      FAIL_FAST=true
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if [[ "$CLI_LIVE_MODE" != "skip" && "$CLI_LIVE_MODE" != "strict" && "$CLI_LIVE_MODE" != "off" ]]; then
  echo "invalid --cli-live-mode: $CLI_LIVE_MODE (expected skip|strict|off)" >&2
  exit 2
fi

if [[ "$RUN_CLI" == "false" && "$RUN_DAEMON" == "false" && "$RUN_UI" == "false" && "$RUN_HARNESS" == "false" ]]; then
  echo "no checks selected; remove one or more --skip-* flags" >&2
  exit 2
fi

STAMP="$(date +%Y%m%d-%H%M%S)"
mkdir -p "$LOG_DIR"
if [[ -z "$LOG_FILE" ]]; then
  LOG_FILE="$LOG_DIR/tests-all-$STAMP.log"
fi
RUNNER_LOG_DIR="$LOG_DIR/tests-all-$STAMP-runners"
mkdir -p "$RUNNER_LOG_DIR"

exec > >(tee -a "$LOG_FILE") 2>&1

echo "Full test suite started at $(date -u +'%Y-%m-%dT%H:%M:%SZ')"
echo "Repository root: $ROOT"
echo "Suite log file: $LOG_FILE"
echo "Runner logs dir: $RUNNER_LOG_DIR"
echo "CLI live mode: $CLI_LIVE_MODE"
echo "Daemon skip regression: $DAEMON_SKIP_REGRESSION"
echo "UI skip build: $UI_SKIP_BUILD"
echo "UI open app: $UI_OPEN_APP"
echo "Fail fast: $FAIL_FAST"

FAILURES=0
STEPS=0
STOP_EARLY=false

CLI_STATUS="skipped"
DAEMON_STATUS="skipped"
UI_STATUS="skipped"
HARNESS_STATUS="skipped"

CLI_LOG_PATH="$RUNNER_LOG_DIR/tests-cli.log"
DAEMON_LOG_PATH="$RUNNER_LOG_DIR/tests-daemon.log"
UI_LOG_PATH="$RUNNER_LOG_DIR/tests-ui.log"
HARNESS_LOG_PATH="$RUNNER_LOG_DIR/check-harness.log"

run_command_step() {
  local label="$1"
  local status_var="$2"
  shift 2

  STEPS=$((STEPS + 1))
  echo
  echo "================================================================"
  echo "STEP $STEPS: $label"
  echo "CMD: $*"
  echo "----------------------------------------------------------------"

  "$@"
  local rc=$?
  echo "RESULT: exit_code=$rc"

  if [[ "$rc" -eq 0 ]]; then
    printf -v "$status_var" '%s' "pass"
  else
    printf -v "$status_var" '%s' "fail"
    FAILURES=$((FAILURES + 1))
    if [[ "$FAIL_FAST" == "true" ]]; then
      return 1
    fi
  fi
  return 0
}

run_piped_step() {
  local label="$1"
  local status_var="$2"
  local log_path="$3"
  shift 3

  STEPS=$((STEPS + 1))
  echo
  echo "================================================================"
  echo "STEP $STEPS: $label"
  echo "CMD: $* | tee $log_path"
  echo "----------------------------------------------------------------"

  "$@" 2>&1 | tee "$log_path"
  local rc=${PIPESTATUS[0]}
  echo "RESULT: exit_code=$rc"

  if [[ "$rc" -eq 0 ]]; then
    printf -v "$status_var" '%s' "pass"
  else
    printf -v "$status_var" '%s' "fail"
    FAILURES=$((FAILURES + 1))
    if [[ "$FAIL_FAST" == "true" ]]; then
      return 1
    fi
  fi
  return 0
}

if [[ "$RUN_CLI" == "true" ]]; then
  if ! run_command_step \
    "Run CLI suite" \
    CLI_STATUS \
    ./tools/scripts/run_tests_cli.sh \
    --log-file "$CLI_LOG_PATH" \
    --live-mode "$CLI_LIVE_MODE"; then
    STOP_EARLY=true
  fi
else
  echo "CLI suite skipped by --skip-cli"
fi

if [[ "$STOP_EARLY" == "true" ]]; then
  echo "Daemon suite skipped due to --fail-fast after prior failure"
elif [[ "$RUN_DAEMON" == "true" ]]; then
  daemon_cmd=(./tools/scripts/run_tests_daemon.sh --log-file "$DAEMON_LOG_PATH")
  if [[ "$DAEMON_SKIP_REGRESSION" == "true" ]]; then
    daemon_cmd+=(--skip-regression)
  fi
  if ! run_command_step \
    "Run daemon suite" \
    DAEMON_STATUS \
    "${daemon_cmd[@]}"; then
    STOP_EARLY=true
  fi
else
  echo "Daemon suite skipped by --skip-daemon"
fi

if [[ "$STOP_EARLY" == "true" ]]; then
  echo "UI suite skipped due to --fail-fast after prior failure"
elif [[ "$RUN_UI" == "true" ]]; then
  ui_cmd=(./tools/scripts/run_tests_ui.sh --log-file "$UI_LOG_PATH")
  if [[ "$UI_SKIP_BUILD" == "true" ]]; then
    ui_cmd+=(--skip-build)
  fi
  if [[ "$UI_OPEN_APP" == "true" ]]; then
    ui_cmd+=(--open-app)
  fi
  if ! run_command_step \
    "Run UI suite" \
    UI_STATUS \
    "${ui_cmd[@]}"; then
    STOP_EARLY=true
  fi
else
  echo "UI suite skipped by --skip-ui"
fi

if [[ "$STOP_EARLY" == "true" ]]; then
  echo "Harness check skipped due to --fail-fast after prior failure"
elif [[ "$RUN_HARNESS" == "true" ]]; then
  if ! run_piped_step \
    "Run harness check" \
    HARNESS_STATUS \
    "$HARNESS_LOG_PATH" \
    ./tools/scripts/check_harness.sh; then
    STOP_EARLY=true
  fi
else
  echo "Harness check skipped by --skip-harness"
fi

echo
echo "================================================================"
echo "Full test suite completed at $(date -u +'%Y-%m-%dT%H:%M:%SZ')"
echo "Suite log file: $LOG_FILE"
echo "Runner logs:"
echo "- CLI: $CLI_STATUS ($CLI_LOG_PATH)"
echo "- Daemon: $DAEMON_STATUS ($DAEMON_LOG_PATH)"
echo "- UI: $UI_STATUS ($UI_LOG_PATH)"
echo "- Harness: $HARNESS_STATUS ($HARNESS_LOG_PATH)"
echo
echo "Notes:"
echo "- UI runner still includes manual handoff verification from docs/tests-ui.md."
echo "- Use --fail-fast to stop on first failing suite."
echo
echo "Total steps: $STEPS"
echo "Failures: $FAILURES"

if [[ "$FAILURES" -eq 0 ]]; then
  echo "STATUS: PASS"
  echo "Exit code: 0"
  echo "================================================================"
  exit 0
else
  echo "STATUS: FAIL"
  echo "Exit code: 1"
  echo "================================================================"
  exit 1
fi
