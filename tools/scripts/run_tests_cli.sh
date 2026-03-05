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
LIVE_MODE="skip" # skip|strict|off

while [[ $# -gt 0 ]]; do
  case "$1" in
    --log-file)
      LOG_FILE="${2:-}"
      shift 2
      ;;
    --live-mode)
      LIVE_MODE="${2:-}"
      shift 2
      ;;
    *)
      echo "unknown argument: $1" >&2
      echo "usage: $0 [--log-file <path>] [--live-mode skip|strict|off]" >&2
      exit 2
      ;;
  esac
done

if [[ "$LIVE_MODE" != "skip" && "$LIVE_MODE" != "strict" && "$LIVE_MODE" != "off" ]]; then
  echo "invalid --live-mode: $LIVE_MODE (expected skip|strict|off)" >&2
  exit 2
fi

LOG_DIR="$ROOT/out/logs/manual-tests"
mkdir -p "$LOG_DIR"
if [[ -z "$LOG_FILE" ]]; then
  STAMP="$(date +%Y%m%d-%H%M%S)"
  LOG_FILE="$LOG_DIR/tests-cli-$STAMP.log"
fi

exec > >(tee -a "$LOG_FILE") 2>&1

echo "CLI manual tests started at $(date -u +'%Y-%m-%dT%H:%M:%SZ')"
echo "Repository root: $ROOT"
echo "Log file: $LOG_FILE"
echo "Live Twilio mode: $LIVE_MODE"
echo "Tip: run ./tools/scripts/run_tests_all.sh to execute CLI + daemon + UI runners together."

normalize_project_name() {
  local raw="$1"
  local candidate
  candidate="$(printf '%s' "$raw" | tr '[:upper:]' '[:lower:]')"
  candidate="${candidate%.*}"
  candidate="${candidate%-daemon}"
  candidate="${candidate%_daemon}"
  candidate="${candidate%daemon}"
  candidate="$(printf '%s' "$candidate" | tr -cd 'a-z0-9')"
  if [[ -z "$candidate" ]]; then
    candidate="personalagent"
  fi
  printf '%s' "$candidate"
}

FAILURES=0
STEPS=0
TEST_RUNTIME_ROOT="${PA_TEST_RUNTIME_ROOT:-$ROOT/out/test-state/cli}"
WORKSPACE="${WORKSPACE:-test-ws1}"
DB_PATH="${DB_PATH:-$TEST_RUNTIME_ROOT/manual-test-cli.db}"
CONTROL_AUTH_TOKEN_FILE="${CONTROL_AUTH_TOKEN_FILE:-$TEST_RUNTIME_ROOT/manual-test-cli.control.token}"
PROFILE_AUTH_TOKEN_FILE="${PROFILE_AUTH_TOKEN_FILE:-$TEST_RUNTIME_ROOT/manual-test-cli.profile.token}"
LOCAL_DEV_BOOTSTRAP_TOKEN_FILE="${LOCAL_DEV_BOOTSTRAP_TOKEN_FILE:-$TEST_RUNTIME_ROOT/manual-test-cli.local-dev.token}"
QUICKSTART_TOKEN_FILE="${QUICKSTART_TOKEN_FILE:-$TEST_RUNTIME_ROOT/manual-test-cli.quickstart.token}"
DAEMON_PORT="${PA_CLI_TEST_DAEMON_PORT:-$((21000 + (RANDOM % 2000)))}"
DAEMON_ADDR="${DAEMON_ADDR:-127.0.0.1:${DAEMON_PORT}}"
DAEMON_AUTH_TOKEN="cli-test-token"
PA_RUNTIME_PROFILE="${PA_RUNTIME_PROFILE:-test}"
PA_RUNTIME_ROOT_DIR="${PA_RUNTIME_ROOT_DIR:-$TEST_RUNTIME_ROOT/runtime-root}"
PROJECT_NAME="$(normalize_project_name "${PA_PROJECT_NAME:-personalagent}")"
WEBHOOK_API_VERSION="v1"
TWILIO_SMS_WEBHOOK_PATH="/${PROJECT_NAME}/${WEBHOOK_API_VERSION}/connector/twilio/sms"
TWILIO_VOICE_WEBHOOK_PATH="/${PROJECT_NAME}/${WEBHOOK_API_VERSION}/connector/twilio/voice"
OPENAI_MOCK_PORT="${PA_CLI_TEST_OPENAI_MOCK_PORT:-$((23000 + (RANDOM % 2000)))}"
OPENAI_MOCK_ADDR="${PA_CLI_TEST_OPENAI_MOCK_ADDR:-127.0.0.1:${OPENAI_MOCK_PORT}}"
OPENAI_MOCK_ENDPOINT="http://${OPENAI_MOCK_ADDR}"
TWILIO_MOCK_PORT="${PA_CLI_TEST_TWILIO_MOCK_PORT:-$((25000 + (RANDOM % 2000)))}"
TWILIO_MOCK_ADDR="${PA_CLI_TEST_TWILIO_MOCK_ADDR:-127.0.0.1:${TWILIO_MOCK_PORT}}"
TWILIO_MOCK_ENDPOINT="http://${TWILIO_MOCK_ADDR}"
WEBHOOK_LISTEN_PORT="${PA_CLI_TEST_WEBHOOK_PORT:-$((27000 + (RANDOM % 2000)))}"
WEBHOOK_LISTEN_ADDR="${PA_CLI_TEST_WEBHOOK_ADDR:-127.0.0.1:${WEBHOOK_LISTEN_PORT}}"
export PA_PROJECT_NAME="$PROJECT_NAME"
export PA_RUNTIME_PROFILE
export PA_RUNTIME_ROOT_DIR
mkdir -p "$TEST_RUNTIME_ROOT" "$PA_RUNTIME_ROOT_DIR"
MESSAGES_SEND_DRY_RUN="${PA_MESSAGES_SEND_DRY_RUN:-1}"
MAIL_AUTOMATION_DRY_RUN="1"
CALENDAR_AUTOMATION_DRY_RUN="${PA_CALENDAR_AUTOMATION_DRY_RUN:-1}"
BROWSER_AUTOMATION_DRY_RUN="${PA_BROWSER_AUTOMATION_DRY_RUN:-1}"
CLOUDFLARED_DRY_RUN="${PA_CLOUDFLARED_DRY_RUN:-1}"
MOCKS_PID=""
WEBHOOK_PID=""
DAEMON_PID=""
MESSAGES_FIXTURE_DB="${MESSAGES_FIXTURE_DB:-$TEST_RUNTIME_ROOT/messages-cli-fixture.db}"
DESTRUCTIVE_JSON=""
APPROVAL_ID=""
APPROVAL_DELEGATION_JSON=""
APPROVAL_RULE_ID=""
CALENDAR_DESTRUCTIVE_JSON=""
CALENDAR_EVENT_ID=""
CALENDAR_APPROVAL_ID=""
VOICE_DESTRUCTIVE_JSON=""
VOICE_APPROVAL_ID=""
VOICE_CONFIRMED_JSON=""
GRANT_JSON=""
RULE_ID=""
RUN_JSON=""
RUN_ID=""
WEBHOOK_SERVE_LOG=""
MOCKS_LOG=""
DAEMON_LOG=""
TASK_JSON=""
TASK_ID=""
TASK_CANCEL_JSON=""
TASK_CANCEL_ID=""
TASK_CANCEL_RUN_ID=""
TASK_RETRY_JSON=""
TASK_RETRY_RUN_ID=""
TASK_REQUEUE_JSON=""
TASK_REQUEUE_RUN_ID=""
APPROVAL_DECIDE_RUN_JSON=""
APPROVAL_DECIDE_ID=""
APPROVAL_DECIDE_JSON=""
START_CALL_JSON=""
CALL_SID=""
AUTH_BOOTSTRAP_JSON=""
AUTH_ROTATE_JSON=""
AUTH_LOCAL_DEV_BOOTSTRAP_JSON=""
PROFILE_SET_JSON=""
PROFILE_LIST_JSON=""
PROFILE_GET_JSON=""
PROFILE_LOCAL_DEV_GET_JSON=""
PROFILE_USE_JSON=""
PROFILE_ACTIVE_JSON=""
PROFILE_RENAME_JSON=""
PROFILE_GET_RENAMED_JSON=""
PROFILE_DELETE_JSON=""
PROFILE_ACTIVE_AFTER_DELETE_JSON=""
PROFILE_SMOKE_JSON=""
MACHINE_SMOKE_JSON=""
HELP_USAGE_TEXT=""
HELP_TASK_USAGE_TEXT=""
HELP_TASK_FLAG_USAGE_TEXT=""
HELP_TASK_SUBMIT_FLAG_USAGE_TEXT=""
META_SCHEMA_JSON=""
META_CAPABILITIES_JSON=""
VERSION_JSON=""
VERSION_TEXT_OUTPUT=""
DOCTOR_JSON=""
DOCTOR_QUICK_JSON=""
DOCTOR_TEXT_OUTPUT=""
QUICKSTART_JSON=""
QUICKSTART_FAILURE_JSON=""
ASSISTANT_TASK_JSON=""
ASSISTANT_CANCEL_JSON=""
ACTIONABLE_ERROR_TEXT=""
PROFILE_PROVIDER_LIST_JSON=""
PROVIDER_LIST_TEXT_OUTPUT=""
MODEL_LIST_TEXT_OUTPUT=""
TASK_STATUS_TEXT_OUTPUT=""
MESSAGES_INGEST_JSON=""
MAIL_INGEST_JSON=""
CALENDAR_INGEST_JSON=""
BROWSER_INGEST_JSON=""
MODEL_DISCOVER_JSON=""
MODEL_ADD_JSON=""
MODEL_REMOVE_JSON=""
CLOUDFLARED_VERSION_JSON=""
CLOUDFLARED_EXEC_JSON=""
CLOUDFLARED_TUNNEL_START_JSON=""
CLOUDFLARED_TUNNEL_STATUS_JSON=""
CLOUDFLARED_TUNNEL_STOP_JSON=""
CLOUDFLARED_TUNNEL_ID=""
IDENTITY_WORKSPACES_JSON=""
IDENTITY_CONTEXT_JSON=""
IDENTITY_PRINCIPALS_JSON=""
IDENTITY_SELECT_JSON=""
CHANNEL_MAPPING_INITIAL_JSON=""
CHANNEL_MAPPING_DISABLE_JSON=""
CHANNEL_MAPPING_PRIORITIZE_JSON=""
CHANNEL_MAPPING_ENABLE_JSON=""
CHANNEL_MAPPING_FINAL_JSON=""
CHANNEL_MAPPING_VOICE_INITIAL_JSON=""
CHANNEL_MAPPING_VOICE_DISABLE_JSON=""
CHANNEL_MAPPING_VOICE_ENABLE_JSON=""
CHANNEL_MAPPING_VOICE_FINAL_JSON=""
CHANNEL_MAPPING_LEGACY_MESSAGE_JSON=""
CHANNEL_MAPPING_LEGACY_VOICE_JSON=""
CHANNEL_MAPPING_POST_TWILIO_MESSAGE_JSON=""
CHANNEL_MAPPING_POST_TWILIO_VOICE_JSON=""
TWILIO_GET_JSON=""
IDENTITY_BOOTSTRAP_FIRST_JSON=""
IDENTITY_BOOTSTRAP_SECOND_JSON=""
IDENTITY_TARGET_WORKSPACE="$WORKSPACE"
IDENTITY_DEVICES_PAGE1_JSON=""
IDENTITY_DEVICES_PAGE2_JSON=""
IDENTITY_SESSIONS_JSON=""
IDENTITY_REVOKE_FIRST_JSON=""
IDENTITY_REVOKE_SECOND_JSON=""
IDENTITY_ACTIVE_SESSION_ID=""

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

capture_cmd() {
  local out_var="$1"
  local label="$2"
  shift 2
  STEPS=$((STEPS + 1))
  echo
  echo "================================================================"
  echo "STEP $STEPS: $label"
  echo "CMD: $*"
  echo "----------------------------------------------------------------"
  local output
  output="$("$@" 2>&1)"
  local rc=$?
  printf '%s\n' "$output"
  printf -v "$out_var" '%s' "$output"
  echo "RESULT: exit_code=$rc"
  if [[ "$rc" -ne 0 ]]; then
    FAILURES=$((FAILURES + 1))
  fi
  return 0
}

wait_for_http() {
  local url="$1"
  local retries="${2:-300}"
  local mode="${3:-strict}"
  local i
  for i in $(seq 1 "$retries"); do
    if [[ "$mode" == "strict" ]]; then
      if curl -sSf "$url" >/dev/null 2>&1; then
        echo "ready: $url"
        return 0
      fi
    else
      if curl -sS --max-time 2 "$url" >/dev/null 2>&1; then
        echo "ready: $url"
        return 0
      fi
    fi
    sleep 0.2
  done
  echo "timeout waiting for: $url"
  return 1
}

wait_for_http_status() {
  local url="$1"
  local expected_status="$2"
  local auth_token="$3"
  local retries="${4:-200}"
  local i
  local status
  for i in $(seq 1 "$retries"); do
    if [[ -n "$auth_token" ]]; then
      status="$(curl -s -o /dev/null -w '%{http_code}' -H "Authorization: Bearer $auth_token" "$url" || true)"
    else
      status="$(curl -s -o /dev/null -w '%{http_code}' "$url" || true)"
    fi
    if [[ "$status" == "$expected_status" ]]; then
      echo "ready: $url status=$status"
      return 0
    fi
    sleep 0.2
  done
  echo "timeout waiting for $url status=$expected_status"
  return 1
}

pa() {
  (
    cd "$ROOT/source/clients/cli-go"
    go run ./cmd/personal-agent --db "$DB_PATH" "$@"
  )
}

pa_daemon() {
  pa --mode tcp --address "$DAEMON_ADDR" --auth-token "$DAEMON_AUTH_TOKEN" "$@"
}

cleanup_background() {
  if [[ -n "$WEBHOOK_PID" ]] && kill -0 "$WEBHOOK_PID" >/dev/null 2>&1; then
    kill "$WEBHOOK_PID" >/dev/null 2>&1 || true
    wait "$WEBHOOK_PID" >/dev/null 2>&1 || true
    WEBHOOK_PID=""
  fi
  if [[ -n "$DAEMON_PID" ]] && kill -0 "$DAEMON_PID" >/dev/null 2>&1; then
    kill "$DAEMON_PID" >/dev/null 2>&1 || true
    wait "$DAEMON_PID" >/dev/null 2>&1 || true
    DAEMON_PID=""
  fi
  if [[ -n "$MOCKS_PID" ]] && kill -0 "$MOCKS_PID" >/dev/null 2>&1; then
    kill "$MOCKS_PID" >/dev/null 2>&1 || true
    wait "$MOCKS_PID" >/dev/null 2>&1 || true
    MOCKS_PID=""
  fi
  rm -f "$CONTROL_AUTH_TOKEN_FILE"
  rm -f "$PROFILE_AUTH_TOKEN_FILE"
  rm -f "$LOCAL_DEV_BOOTSTRAP_TOKEN_FILE"
  rm -f "$QUICKSTART_TOKEN_FILE"
  rm -f "$MESSAGES_FIXTURE_DB"
  if [[ "$PA_RUNTIME_ROOT_DIR" == "$TEST_RUNTIME_ROOT/runtime-root" ]]; then
    rm -rf "$PA_RUNTIME_ROOT_DIR"
  fi
}
trap cleanup_background EXIT

# 1) Prerequisites
run_cmd "Check Go toolchain" go version
run_cmd "Check curl" curl --version
run_cmd "Check jq" jq --version

# 2) Test environment setup
run_cmd "Remove existing CLI manual test DB" rm -f "$DB_PATH"
run_cmd "Remove existing CLI control auth token file" rm -f "$CONTROL_AUTH_TOKEN_FILE"
run_cmd "Remove existing CLI profile auth token file" rm -f "$PROFILE_AUTH_TOKEN_FILE"
run_cmd "Remove existing local-dev bootstrap token file" rm -f "$LOCAL_DEV_BOOTSTRAP_TOKEN_FILE"
run_cmd "Remove existing quickstart token file" rm -f "$QUICKSTART_TOKEN_FILE"
run_eval "Show isolated runtime defaults" 'echo "WORKSPACE=$WORKSPACE"; echo "DB_PATH=$DB_PATH"; echo "PA_RUNTIME_PROFILE=$PA_RUNTIME_PROFILE"; echo "PA_RUNTIME_ROOT_DIR=$PA_RUNTIME_ROOT_DIR"'
run_eval "Show endpoint defaults" 'echo "PA_PROJECT_NAME=$PA_PROJECT_NAME"; echo "TWILIO_SMS_WEBHOOK_PATH=$TWILIO_SMS_WEBHOOK_PATH"; echo "TWILIO_VOICE_WEBHOOK_PATH=$TWILIO_VOICE_WEBHOOK_PATH"; echo "DAEMON_ADDR=$DAEMON_ADDR"; echo "OPENAI_MOCK_ENDPOINT=$OPENAI_MOCK_ENDPOINT"; echo "TWILIO_MOCK_ENDPOINT=$TWILIO_MOCK_ENDPOINT"; echo "WEBHOOK_LISTEN_ADDR=$WEBHOOK_LISTEN_ADDR"'
capture_cmd AUTH_BOOTSTRAP_JSON "Bootstrap control auth token file" pa auth bootstrap --file "$CONTROL_AUTH_TOKEN_FILE"
capture_cmd AUTH_ROTATE_JSON "Rotate control auth token file" pa auth rotate --file "$CONTROL_AUTH_TOKEN_FILE"
run_eval "Validate control auth token bootstrap/rotate outputs" '
TOKEN_VALUE="$(tr -d "[:space:]" < "$CONTROL_AUTH_TOKEN_FILE")"
test -n "$TOKEN_VALUE"
echo "$AUTH_BOOTSTRAP_JSON" | jq -e ".operation == \"bootstrap\" and .token_file == \"$CONTROL_AUTH_TOKEN_FILE\" and (.token_sha256|type) == \"string\"" >/dev/null
echo "$AUTH_ROTATE_JSON" | jq -e ".operation == \"rotate\" and .token_file == \"$CONTROL_AUTH_TOKEN_FILE\" and .rotated == true and (.token_sha256|type) == \"string\"" >/dev/null
! printf "%s\n" "$AUTH_BOOTSTRAP_JSON" | grep -F -- "$TOKEN_VALUE" >/dev/null
! printf "%s\n" "$AUTH_ROTATE_JSON" | grep -F -- "$TOKEN_VALUE" >/dev/null
'
capture_cmd AUTH_LOCAL_DEV_BOOTSTRAP_JSON "Bootstrap local-dev token/profile in one command" pa auth bootstrap-local-dev --profile local-dev-bootstrap --mode tcp --address "$DAEMON_ADDR" --workspace "$WORKSPACE" --token-file "$LOCAL_DEV_BOOTSTRAP_TOKEN_FILE" --rotate-token
capture_cmd PROFILE_LOCAL_DEV_GET_JSON "Get local-dev bootstrap profile" pa profile get --name local-dev-bootstrap
run_eval "Validate one-command local-dev bootstrap output" '
LOCAL_DEV_TOKEN_VALUE="$(tr -d "[:space:]" < "$LOCAL_DEV_BOOTSTRAP_TOKEN_FILE")"
test -n "$LOCAL_DEV_TOKEN_VALUE"
echo "$AUTH_LOCAL_DEV_BOOTSTRAP_JSON" | jq -e ".operation == \"bootstrap_local_dev\" and .token_file == \"$LOCAL_DEV_BOOTSTRAP_TOKEN_FILE\" and (.token_sha256|type) == \"string\" and ((.token_created == true) or (.token_rotated == true)) and .profile.name == \"local-dev-bootstrap\" and .profile.workspace_id == \"$WORKSPACE\" and .profile.address == \"$DAEMON_ADDR\" and .profile.auth_token_file == \"$LOCAL_DEV_BOOTSTRAP_TOKEN_FILE\" and .active_profile == \"local-dev-bootstrap\" and .defaults.workspace.value == \"$WORKSPACE\" and .defaults.workspace.source == \"explicit\" and .defaults.workspace.override_flag == \"--workspace\" and .defaults.profile.value == \"local-dev-bootstrap\" and .defaults.profile.source == \"explicit\" and .defaults.profile.override_flag == \"--profile\" and .defaults.token_file.value == \"$LOCAL_DEV_BOOTSTRAP_TOKEN_FILE\" and .defaults.token_file.source == \"explicit\" and .defaults.token_file.override_flag == \"--token-file\" and ((.defaults.override_hints // []) | length) >= 3 and ((.defaults.override_hints // []) | map(test(\"--workspace\")) | any) and ((.defaults.override_hints // []) | map(test(\"--profile\")) | any) and ((.defaults.override_hints // []) | map(test(\"--token-file\")) | any)" >/dev/null
echo "$PROFILE_LOCAL_DEV_GET_JSON" | jq -e ".profile.name == \"local-dev-bootstrap\" and .profile.address == \"$DAEMON_ADDR\" and .profile.workspace_id == \"$WORKSPACE\" and .profile.auth_token_file == \"$LOCAL_DEV_BOOTSTRAP_TOKEN_FILE\"" >/dev/null
! printf "%s\n" "$AUTH_LOCAL_DEV_BOOTSTRAP_JSON" | grep -F -- "$LOCAL_DEV_TOKEN_VALUE" >/dev/null
'
run_eval "Validate workspace resolution defaults and explicit overrides" '
WORKSPACE_CONTEXT_JSON="$(PERSONAL_AGENT_WORKSPACE_ID=ws-cli-context pa connector bridge status)"
echo "$WORKSPACE_CONTEXT_JSON"
echo "$WORKSPACE_CONTEXT_JSON" | jq -e ".workspace_id == \"ws-cli-context\"" >/dev/null
WORKSPACE_EMPTY_ENV_JSON="$(PERSONAL_AGENT_WORKSPACE_ID= pa connector bridge status)"
echo "$WORKSPACE_EMPTY_ENV_JSON"
echo "$WORKSPACE_EMPTY_ENV_JSON" | jq -e ".workspace_id == \"ws1\"" >/dev/null
WORKSPACE_EXPLICIT_DEFAULT_JSON="$(PERSONAL_AGENT_WORKSPACE_ID=ws-cli-context pa connector bridge status --workspace default)"
echo "$WORKSPACE_EXPLICIT_DEFAULT_JSON"
echo "$WORKSPACE_EXPLICIT_DEFAULT_JSON" | jq -e ".workspace_id == \"default\"" >/dev/null
'

# 3) Baseline health checks
run_eval "Run Go test suite" 'cd "$ROOT/source/services/daemon-go" && go test ./...'
run_eval "Run harness checks" 'cd "$ROOT" && ./tools/scripts/check_harness.sh'

# 4) Start local mock providers
run_eval "Start local OpenAI+Twilio mocks" '
MOCKS_LOG="$TEST_RUNTIME_ROOT/mock-providers-cli.log"
rm -f "$MOCKS_LOG"
go -C source/services/daemon-go run ./cmd/personal-agent-mock --mode all --openai-listen "$OPENAI_MOCK_ADDR" --twilio-listen "$TWILIO_MOCK_ADDR" >"$MOCKS_LOG" 2>&1 &
MOCKS_PID=$!
echo "MOCKS_PID=$MOCKS_PID"
echo "MOCKS_LOG=$MOCKS_LOG"
'
run_cmd "Wait for OpenAI mock endpoint" wait_for_http_status "$OPENAI_MOCK_ENDPOINT/v1/models" "200" "" 300
run_cmd "Wait for Twilio mock endpoint" wait_for_http_status "$TWILIO_MOCK_ENDPOINT/2010-04-01/Accounts/AC123.json" "200" "" 300

# 5.0) Start daemon for secret workflow
run_eval "Start daemon for secret/provider/model/chat workflows" '
DAEMON_LOG="$TEST_RUNTIME_ROOT/daemon-cli.log"
rm -f "$DAEMON_LOG"
PA_PROJECT_NAME="$PA_PROJECT_NAME" \
PA_RUNTIME_PROFILE="$PA_RUNTIME_PROFILE" \
PA_RUNTIME_ROOT_DIR="$PA_RUNTIME_ROOT_DIR" \
PA_MESSAGES_SEND_DRY_RUN="$MESSAGES_SEND_DRY_RUN" \
PA_MAIL_AUTOMATION_DRY_RUN="$MAIL_AUTOMATION_DRY_RUN" \
PA_CALENDAR_AUTOMATION_DRY_RUN="$CALENDAR_AUTOMATION_DRY_RUN" \
PA_BROWSER_AUTOMATION_DRY_RUN="$BROWSER_AUTOMATION_DRY_RUN" \
PA_CLOUDFLARED_DRY_RUN="$CLOUDFLARED_DRY_RUN" \
go -C source/services/daemon-go run ./cmd/personal-agent-daemon \
  --listen-mode tcp \
  --listen-address "$DAEMON_ADDR" \
  --auth-token "$DAEMON_AUTH_TOKEN" \
  --db "$DB_PATH" >"$DAEMON_LOG" 2>&1 &
DAEMON_PID=$!
echo "DAEMON_PID=$DAEMON_PID"
echo "DAEMON_LOG=$DAEMON_LOG"
'
run_cmd "Wait for daemon endpoint" wait_for_http_status "http://$DAEMON_ADDR/v1/capabilities/smoke" "200" "$DAEMON_AUTH_TOKEN" 300

# 5.0.1) CLI profile defaults (endpoint/auth/workspace)
run_eval "Write profile auth token file for daemon defaults" '
printf "%s\n" "$DAEMON_AUTH_TOKEN" > "$PROFILE_AUTH_TOKEN_FILE"
chmod 600 "$PROFILE_AUTH_TOKEN_FILE"
'
capture_cmd PROFILE_SET_JSON "Set CLI profile defaults" pa profile set --name local-daemon --mode tcp --address "$DAEMON_ADDR" --workspace "$WORKSPACE" --auth-token-file "$PROFILE_AUTH_TOKEN_FILE"
capture_cmd PROFILE_LIST_JSON "List CLI profiles" pa profile list
capture_cmd PROFILE_GET_JSON "Get active CLI profile" pa profile get
capture_cmd PROFILE_USE_JSON "Select active CLI profile" pa profile use --name local-daemon
capture_cmd PROFILE_ACTIVE_JSON "Inspect active CLI profile metadata" pa profile active
capture_cmd PROFILE_RENAME_JSON "Rename CLI profile local-daemon -> local-daemon-renamed" pa profile rename --name local-daemon --to local-daemon-renamed
capture_cmd PROFILE_GET_RENAMED_JSON "Get renamed CLI profile" pa profile get --name local-daemon-renamed
capture_cmd PROFILE_DELETE_JSON "Delete renamed CLI profile and trigger active fallback" pa profile delete --name local-daemon-renamed
capture_cmd PROFILE_ACTIVE_AFTER_DELETE_JSON "Inspect active CLI profile metadata after delete fallback" pa profile active
capture_cmd PROFILE_FALLBACK_SYNC_JSON "Sync fallback local-dev-bootstrap profile to daemon auth token defaults" pa profile set --name local-dev-bootstrap --mode tcp --address "$DAEMON_ADDR" --workspace "$WORKSPACE" --auth-token-file "$PROFILE_AUTH_TOKEN_FILE"
capture_cmd PROFILE_ACTIVE_AFTER_SYNC_JSON "Inspect active CLI profile metadata after fallback token sync" pa profile active
capture_cmd PROFILE_SMOKE_JSON "Run smoke with profile defaults (no daemon flags)" pa smoke
run_eval "Validate CLI profile default flow" '
set -e
echo "$PROFILE_SET_JSON" | jq -e ".profile.name == \"local-daemon\" and .active_profile == \"local-dev-bootstrap\" and .active == false and .profile.workspace_id == \"$WORKSPACE\" and .profile.address == \"$DAEMON_ADDR\" and .profile.auth_token_file == \"$PROFILE_AUTH_TOKEN_FILE\"" >/dev/null
echo "$PROFILE_LIST_JSON" | jq -e ".active_profile == \"local-dev-bootstrap\" and ((.profiles // []) | length) >= 2 and (((.profiles // []) | map(.name)) | index(\"local-daemon\")) != null and (((.profiles // []) | map(.name)) | index(\"local-dev-bootstrap\")) != null" >/dev/null
echo "$PROFILE_GET_JSON" | jq -e ".profile.name == \"local-dev-bootstrap\" and .active == true and .profile.auth_token_file == \"$LOCAL_DEV_BOOTSTRAP_TOKEN_FILE\"" >/dev/null
echo "$PROFILE_USE_JSON" | jq -e ".profile.name == \"local-daemon\" and .active == true" >/dev/null
echo "$PROFILE_ACTIVE_JSON" | jq -e ".active_profile == \"local-daemon\" and .profile_exists == true and .profile.name == \"local-daemon\"" >/dev/null
echo "$PROFILE_RENAME_JSON" | jq -e ".previous_name == \"local-daemon\" and .profile.name == \"local-daemon-renamed\" and .active_profile == \"local-daemon-renamed\" and .active == true" >/dev/null
echo "$PROFILE_GET_RENAMED_JSON" | jq -e ".profile.name == \"local-daemon-renamed\" and .active == true and .profile.address == \"$DAEMON_ADDR\"" >/dev/null
echo "$PROFILE_DELETE_JSON" | jq -e ".deleted_profile == \"local-daemon-renamed\" and .active_profile_changed == true and .active_profile == \"local-dev-bootstrap\"" >/dev/null
echo "$PROFILE_ACTIVE_AFTER_DELETE_JSON" | jq -e ".active_profile == \"local-dev-bootstrap\" and .profile_exists == true and .profile.name == \"local-dev-bootstrap\"" >/dev/null
echo "$PROFILE_FALLBACK_SYNC_JSON" | jq -e ".profile.name == \"local-dev-bootstrap\" and .active == true and .profile.workspace_id == \"$WORKSPACE\" and .profile.address == \"$DAEMON_ADDR\" and .profile.auth_token_file == \"$PROFILE_AUTH_TOKEN_FILE\"" >/dev/null
echo "$PROFILE_ACTIVE_AFTER_SYNC_JSON" | jq -e ".active_profile == \"local-dev-bootstrap\" and .profile_exists == true and .profile.name == \"local-dev-bootstrap\" and .profile.auth_token_file == \"$PROFILE_AUTH_TOKEN_FILE\"" >/dev/null
echo "$PROFILE_SMOKE_JSON" | jq -e ".healthy == true and ((.daemon_version // \"\") | length) > 0" >/dev/null
set +e
'
run_eval "Validate CLI exit-code semantics for usage vs runtime failures" '
set +e
PROFILE_SET_USAGE_TEXT="$(pa profile set 2>&1 >/dev/null)"
PROFILE_SET_USAGE_RC=$?
AUTH_ROTATE_USAGE_TEXT="$(pa auth rotate 2>&1 >/dev/null)"
AUTH_ROTATE_USAGE_RC=$?
set -e
printf "%s\n" "$PROFILE_SET_USAGE_TEXT"
printf "%s\n" "$AUTH_ROTATE_USAGE_TEXT"
test "$PROFILE_SET_USAGE_RC" -ne 0
test "$AUTH_ROTATE_USAGE_RC" -ne 0
printf "%s\n" "$PROFILE_SET_USAGE_TEXT" | rg -q -- "--name is required"
printf "%s\n" "$AUTH_ROTATE_USAGE_TEXT" | rg -q -- "--file is required"
printf "%s\n" "$PROFILE_SET_USAGE_TEXT" | rg -q -- "exit status 2"
printf "%s\n" "$AUTH_ROTATE_USAGE_TEXT" | rg -q -- "exit status 2"
set +e
'
run_eval "Write quickstart token file matching daemon auth token" '
printf "%s\n" "$DAEMON_AUTH_TOKEN" > "$QUICKSTART_TOKEN_FILE"
chmod 600 "$QUICKSTART_TOKEN_FILE"
'
capture_cmd QUICKSTART_JSON "Run guided quickstart setup flow" pa quickstart --workspace "$WORKSPACE" --profile quickstart-manual --mode tcp --address "$DAEMON_ADDR" --token-file "$QUICKSTART_TOKEN_FILE" --provider openai --endpoint "$OPENAI_MOCK_ENDPOINT/v1" --api-key "sk-local-quickstart-mock" --model gpt-4.1-mini --task-class chat --skip-doctor=true
run_eval "Validate quickstart setup flow payload and secrecy guarantees" '
echo "$QUICKSTART_JSON" | jq -e ".schema_version == \"1.0.0\" and .workspace_id == \"$WORKSPACE\" and .overall_status == \"pass\" and .success == true and .defaults.workspace.value == \"$WORKSPACE\" and .defaults.workspace.source == \"explicit\" and .defaults.workspace.override_flag == \"--workspace\" and .defaults.profile.value == \"quickstart-manual\" and .defaults.profile.source == \"explicit\" and .defaults.profile.override_flag == \"--profile\" and .defaults.token_file.value == \"$QUICKSTART_TOKEN_FILE\" and .defaults.token_file.source == \"explicit\" and .defaults.token_file.override_flag == \"--token-file\" and ((.defaults.override_hints // []) | length) >= 3 and ((.steps // []) | map(select(.id==\"auth.bootstrap\" and .status==\"pass\")) | length) == 1 and ((.steps // []) | map(select(.id==\"daemon.connectivity\" and .status==\"pass\")) | length) == 1 and ((.steps // []) | map(select(.id==\"provider.configure\" and .status==\"pass\")) | length) == 1 and ((.steps // []) | map(select(.id==\"model.route\" and .status==\"pass\")) | length) == 1 and ((.steps // []) | map(select(.id==\"readiness.doctor\" and .status==\"skipped\")) | length) == 1" >/dev/null
! printf "%s\n" "$QUICKSTART_JSON" | grep -F -- "sk-local-quickstart-mock" >/dev/null
'
run_eval "Run quickstart connectivity failure path for remediation coverage" '
set +e
QUICKSTART_FAILURE_JSON="$(pa quickstart --workspace "$WORKSPACE" --profile quickstart-remediation --activate=false --mode tcp --address 127.0.0.1:17999 --token-file "$QUICKSTART_TOKEN_FILE" --provider openai --skip-provider-setup=true --skip-model-route=true --skip-doctor=true 2>&1 | sed -E "/^exit status [0-9]+$/d")"
QUICKSTART_FAILURE_RC=$?
set -e
printf "%s\n" "$QUICKSTART_FAILURE_JSON"
test "$QUICKSTART_FAILURE_RC" -eq 1
set +e
'
run_eval "Validate quickstart failure remediation payload" '
echo "$QUICKSTART_FAILURE_JSON" | jq -e ".schema_version == \"1.0.0\" and .workspace_id == \"$WORKSPACE\" and .overall_status == \"fail\" and .success == false and .defaults.workspace.value == \"$WORKSPACE\" and .defaults.workspace.source == \"explicit\" and .defaults.profile.value == \"quickstart-remediation\" and .defaults.profile.source == \"explicit\" and .defaults.token_file.value == \"$QUICKSTART_TOKEN_FILE\" and .defaults.token_file.source == \"explicit\" and ((.defaults.override_hints // []) | length) >= 3 and ((.steps // []) | map(select(.id==\"auth.bootstrap\" and .status==\"pass\")) | length) == 1 and ((.steps // []) | map(select(.id==\"daemon.connectivity\" and .status==\"fail\")) | length) == 1 and ((.remediation.human_summary // \"\") | length) > 0 and ((.remediation.next_steps // []) | length) > 0" >/dev/null
echo "$QUICKSTART_FAILURE_JSON" | jq -e "any((.remediation.next_steps // [])[]?; contains(\"personal-agent-daemon --listen-mode\") and contains(\"--listen-address\") and contains(\"127.0.0.1:17999\") and contains(\"--auth-token-file\"))" >/dev/null
echo "$QUICKSTART_FAILURE_JSON" | jq -e "any((.remediation.next_steps // [])[]?; contains(\"personal-agent profile use --name\") and contains(\"quickstart-remediation\"))" >/dev/null
echo "$QUICKSTART_FAILURE_JSON" | jq -e "any((.remediation.next_steps // [])[]?; contains(\"personal-agent --mode\") and contains(\"--address\") and contains(\"127.0.0.1:17999\") and contains(\"--auth-token-file\"))" >/dev/null
'
capture_cmd HELP_USAGE_TEXT "Render goal-oriented CLI help output" pa help
capture_cmd HELP_TASK_USAGE_TEXT "Render task scoped usage via help command" pa help task
capture_cmd HELP_TASK_FLAG_USAGE_TEXT "Render task scoped usage via --help without daemon auth config" pa --auth-token "" task --help
capture_cmd HELP_TASK_SUBMIT_FLAG_USAGE_TEXT "Render task submit scoped usage via --help without daemon auth config" pa --auth-token "" task submit --help
run_eval "Validate goal-oriented CLI help headings/examples" '
printf "%s\n" "$HELP_USAGE_TEXT" | rg -q -- "Quickstart workflows \\(copy/paste\\):"
printf "%s\n" "$HELP_USAGE_TEXT" | rg -q -- "Skim command groups:"
printf "%s\n" "$HELP_USAGE_TEXT" | rg -q -- "Full command reference \\(generated from schema\\):"
printf "%s\n" "$HELP_USAGE_TEXT" | rg -q -- "personal-agent quickstart"
printf "%s\n" "$HELP_USAGE_TEXT" | rg -q -- "help"
printf "%s\n" "$HELP_USAGE_TEXT" | rg -q -- "doctor"
printf "%s\n" "$HELP_USAGE_TEXT" | rg -q -- "task"
printf "%s\n" "$HELP_USAGE_TEXT" | rg -q -- "agent"
printf "%s\n" "$HELP_USAGE_TEXT" | rg -q -- "comm"
printf "%s\n" "$HELP_TASK_USAGE_TEXT" | rg -q -- "Usage: personal-agent task <subcommand> \\[flags\\]"
printf "%s\n" "$HELP_TASK_USAGE_TEXT" | rg -q -- "Subcommands:"
printf "%s\n" "$HELP_TASK_FLAG_USAGE_TEXT" | rg -q -- "Usage: personal-agent task <subcommand> \\[flags\\]"
printf "%s\n" "$HELP_TASK_SUBMIT_FLAG_USAGE_TEXT" | rg -q -- "Usage: personal-agent task submit \\[flags\\]"
printf "%s\n" "$HELP_TASK_SUBMIT_FLAG_USAGE_TEXT" | rg -q -- "Required flags:"
printf "%s\n" "$HELP_TASK_SUBMIT_FLAG_USAGE_TEXT" | rg -q -- "--workspace"
printf "%s\n" "$HELP_TASK_SUBMIT_FLAG_USAGE_TEXT" | rg -q -- "--requested-by"
printf "%s\n" "$HELP_TASK_SUBMIT_FLAG_USAGE_TEXT" | rg -q -- "--subject"
printf "%s\n" "$HELP_TASK_SUBMIT_FLAG_USAGE_TEXT" | rg -q -- "--title"
'
capture_cmd COMPLETION_BASH_SCRIPT "Render bash completion script" pa completion --shell bash
capture_cmd COMPLETION_FISH_SCRIPT "Render fish completion script" pa completion --shell fish
capture_cmd COMPLETION_ZSH_SCRIPT "Render zsh completion script" pa completion zsh
run_eval "Validate completion script generation output" '
printf "%s\n" "$COMPLETION_BASH_SCRIPT" | rg -q -- "_personal_agent_completion\\(\\)"
printf "%s\n" "$COMPLETION_BASH_SCRIPT" | rg -q -- "_pa_subcommands_for_path\\(\\)"
printf "%s\n" "$COMPLETION_BASH_SCRIPT" | rg -q -- "_pa_flags_for_path\\(\\)"
printf "%s\n" "$COMPLETION_BASH_SCRIPT" | rg -q -- "complete -F _personal_agent_completion personal-agent"
printf "%s\n" "$COMPLETION_BASH_SCRIPT" | rg -q -- "\"connector twilio webhook\"\\)"
printf "%s\n" "$COMPLETION_BASH_SCRIPT" | rg -q -- "replay serve"
printf "%s\n" "$COMPLETION_BASH_SCRIPT" | rg -q -- "--requested-by"
printf "%s\n" "$COMPLETION_BASH_SCRIPT" | rg -q -- "--primary-channel"
printf "%s\n" "$COMPLETION_BASH_SCRIPT" | rg -q -- "--signature-mode"
printf "%s\n" "$COMPLETION_BASH_SCRIPT" | rg -q -- "--shell"
printf "%s\n" "$COMPLETION_FISH_SCRIPT" | rg -q -- "function __pa_dynamic_completions"
printf "%s\n" "$COMPLETION_FISH_SCRIPT" | rg -q -- "function __pa_subcommands_for_path"
printf "%s\n" "$COMPLETION_FISH_SCRIPT" | rg -q -- "function __pa_flags_for_path"
printf "%s\n" "$COMPLETION_FISH_SCRIPT" | rg -q -- "complete -c personal-agent -f -a"
printf "%s\n" "$COMPLETION_FISH_SCRIPT" | rg -q -- "case \"connector twilio webhook\""
printf "%s\n" "$COMPLETION_FISH_SCRIPT" | rg -q -- "--requested-by"
printf "%s\n" "$COMPLETION_FISH_SCRIPT" | rg -q -- "--primary-channel"
printf "%s\n" "$COMPLETION_FISH_SCRIPT" | rg -q -- "--signature-mode"
printf "%s\n" "$COMPLETION_FISH_SCRIPT" | rg -q -- "--shell"
printf "%s\n" "$COMPLETION_ZSH_SCRIPT" | rg -q -- "#compdef personal-agent"
printf "%s\n" "$COMPLETION_ZSH_SCRIPT" | rg -q -- "_pa_subcommands_for_path\\(\\)"
printf "%s\n" "$COMPLETION_ZSH_SCRIPT" | rg -q -- "_pa_flags_for_path\\(\\)"
printf "%s\n" "$COMPLETION_ZSH_SCRIPT" | rg -q -- "compdef _personal_agent personal-agent"
printf "%s\n" "$COMPLETION_ZSH_SCRIPT" | rg -q -- "\"connector twilio webhook\"\\)"
printf "%s\n" "$COMPLETION_ZSH_SCRIPT" | rg -q -- "--requested-by"
printf "%s\n" "$COMPLETION_ZSH_SCRIPT" | rg -q -- "--primary-channel"
printf "%s\n" "$COMPLETION_ZSH_SCRIPT" | rg -q -- "--signature-mode"
printf "%s\n" "$COMPLETION_ZSH_SCRIPT" | rg -q -- "--shell"
'
run_eval "Validate unknown-command suggestion hint" '
set +e
UNKNOWN_COMMAND_TEXT="$(pa provder 2>&1 >/dev/null)"
UNKNOWN_COMMAND_RC=$?
set -e
printf "%s\n" "$UNKNOWN_COMMAND_TEXT"
test "$UNKNOWN_COMMAND_RC" -ne 0
printf "%s\n" "$UNKNOWN_COMMAND_TEXT" | rg -q -- "unknown command \"provder\""
printf "%s\n" "$UNKNOWN_COMMAND_TEXT" | rg -q -- "did you mean \"provider\"\\?"
printf "%s\n" "$UNKNOWN_COMMAND_TEXT" | rg -q -- "run \`personal-agent help\` to view available commands"
printf "%s\n" "$UNKNOWN_COMMAND_TEXT" | rg -q -- "exit status 2"
set +e
'
run_eval "Validate unknown-subcommand suggestion hint" '
set +e
UNKNOWN_SUBCOMMAND_TEXT="$(pa meta schem 2>&1 >/dev/null)"
UNKNOWN_SUBCOMMAND_RC=$?
set -e
printf "%s\n" "$UNKNOWN_SUBCOMMAND_TEXT"
test "$UNKNOWN_SUBCOMMAND_RC" -ne 0
printf "%s\n" "$UNKNOWN_SUBCOMMAND_TEXT" | rg -q -- "unknown meta subcommand \"schem\""
printf "%s\n" "$UNKNOWN_SUBCOMMAND_TEXT" | rg -q -- "did you mean \"schema\"\\?"
printf "%s\n" "$UNKNOWN_SUBCOMMAND_TEXT" | rg -q -- "run \`personal-agent help meta\` to view available subcommands"
printf "%s\n" "$UNKNOWN_SUBCOMMAND_TEXT" | rg -q -- "exit status 2"
set +e
'
run_eval "Run assistant task-submit flow with back navigation" '
ASSISTANT_TASK_JSON="$(printf "actor.requester\nback\nactor.requester\nactor.requester\nAssistant task title\nAssistant description\nchat\n" | pa assistant --workspace "$WORKSPACE" --flow task_submit)"
printf "%s\n" "$ASSISTANT_TASK_JSON"
'
run_eval "Validate assistant task-submit flow payload" '
echo "$ASSISTANT_TASK_JSON" | jq -e ".schema_version == \"1.0.0\" and .flow == \"task_submit\" and .workspace_id == \"$WORKSPACE\" and .success == true and .cancelled == false and ((.backtracks // 0) >= 1) and ((.result.task_id // \"\") | length) > 0 and ((.result.run_id // \"\") | length) > 0" >/dev/null
'
run_eval "Run assistant comm-send flow cancellation" '
ASSISTANT_CANCEL_JSON="$(printf "cancel\n" | pa assistant --workspace "$WORKSPACE" --flow comm_send)"
printf "%s\n" "$ASSISTANT_CANCEL_JSON"
'
run_eval "Validate assistant cancellation payload" '
echo "$ASSISTANT_CANCEL_JSON" | jq -e ".schema_version == \"1.0.0\" and .flow == \"comm_send\" and .workspace_id == \"$WORKSPACE\" and .cancelled == true and .success == false" >/dev/null
'
capture_cmd MACHINE_SMOKE_JSON "Run compact machine-output smoke check" pa --output json-compact smoke
run_eval "Validate compact machine-output smoke payload" '
echo "$MACHINE_SMOKE_JSON" | jq -e ".healthy == true and ((.daemon_version // \"\") | length) > 0" >/dev/null
[[ "$MACHINE_SMOKE_JSON" != *$'"'"'\n'"'"'* ]]
'
run_eval "Validate structured machine-error output contract" '
MACHINE_ERROR_JSON="$(pa --output json-compact --error-output json task status --task-id task-does-not-exist 2>&1 | sed -E "/^exit status [0-9]+$/d")"
MACHINE_ERROR_RC=$?
printf "%s\n" "$MACHINE_ERROR_JSON"
test "$MACHINE_ERROR_RC" -eq 1
echo "$MACHINE_ERROR_JSON" | jq -e ".error.code == \"resource_not_found\" and ((.error.message // \"\") | length) > 0 and .error.status_code == 404 and ((.error.correlation_id // \"\") | length) > 0" >/dev/null
'
run_eval "Capture actionable default text error output contract" '
set +e
ACTIONABLE_ERROR_TEXT="$(pa task status --task-id task-does-not-exist 2>&1 >/dev/null)"
ACTIONABLE_ERROR_RC=$?
set -e
printf "%s\n" "$ACTIONABLE_ERROR_TEXT"
test "$ACTIONABLE_ERROR_RC" -eq 1
set +e
'
run_eval "Validate actionable default text error output contract" '
printf "%s\n" "$ACTIONABLE_ERROR_TEXT" | rg -q -- "^request failed$"
printf "%s\n" "$ACTIONABLE_ERROR_TEXT" | rg -q -- "^what failed:"
printf "%s\n" "$ACTIONABLE_ERROR_TEXT" | rg -q -- "^why:"
printf "%s\n" "$ACTIONABLE_ERROR_TEXT" | rg -q -- "^do next:"
printf "%s\n" "$ACTIONABLE_ERROR_TEXT" | rg -q -- "--error-output json"
! printf "%s\n" "$ACTIONABLE_ERROR_TEXT" | rg -q -- "status="
! printf "%s\n" "$ACTIONABLE_ERROR_TEXT" | rg -q -- "code="
'
capture_cmd VERSION_JSON "Emit CLI build/version metadata JSON" pa --output json-compact version
run_eval "Validate CLI build/version metadata JSON payload" '
echo "$VERSION_JSON" | jq -e ".schema_version == \"1.0.0\" and .program == \"personal-agent\" and ((.version // \"\") | length) > 0 and ((.go_version // \"\") | length) > 0 and ((.platform // \"\") | test(\".+/.+\"))" >/dev/null
'
run_eval "Validate CLI build/version metadata text output mode" '
VERSION_TEXT_OUTPUT="$(pa --output text version)"
printf "%s\n" "$VERSION_TEXT_OUTPUT"
printf "%s\n" "$VERSION_TEXT_OUTPUT" | rg -q -- "^personal-agent version$"
printf "%s\n" "$VERSION_TEXT_OUTPUT" | rg -q -- "^version:"
printf "%s\n" "$VERSION_TEXT_OUTPUT" | rg -q -- "^platform:"
! printf "%s\n" "$VERSION_TEXT_OUTPUT" | jq . >/dev/null 2>&1
'
capture_cmd META_SCHEMA_JSON "Emit CLI machine-readable schema manifest" pa --output json-compact meta schema
run_eval "Validate CLI machine-readable schema manifest payload" '
echo "$META_SCHEMA_JSON" | jq -e ".schema_version == \"1.0.0\" and .program == \"personal-agent\" and ((.output_modes // []) | index(\"json-compact\")) != null and ((.output_modes // []) | index(\"text\")) != null and ((.error_output_modes // []) | index(\"json\")) != null and ((.global_flags // []) | map(.name) | index(\"--output\")) != null and ((.global_flags // []) | map(.name) | index(\"--error-output\")) != null and ((.commands // []) | map(.name) | index(\"meta\")) != null and ((.commands // []) | map(.name) | index(\"help\")) != null and ((.commands // []) | map(select(.name==\"help\") | .requires_daemon) | flatten | index(false)) != null and ((.commands // []) | map(.name) | index(\"completion\")) != null and ((.commands // []) | map(select(.name==\"completion\") | .subcommands[] | select(.name==\"bash\") | .requires_daemon) | flatten | index(false)) != null and ((.commands // []) | map(select(.name==\"completion\") | .subcommands[] | select(.name==\"fish\") | .requires_daemon) | flatten | index(false)) != null and ((.commands // []) | map(select(.name==\"completion\") | .subcommands[] | select(.name==\"zsh\") | .requires_daemon) | flatten | index(false)) != null and ((.commands // []) | map(.name) | index(\"quickstart\")) != null and ((.commands // []) | map(select(.name==\"quickstart\") | .requires_daemon) | flatten | index(false)) != null and ((.commands // []) | map(.name) | index(\"version\")) != null and ((.commands // []) | map(select(.name==\"version\") | .requires_daemon) | flatten | index(false)) != null and ((.commands // []) | map(.name) | index(\"assistant\")) != null and ((.commands // []) | map(select(.name==\"assistant\") | .requires_daemon) | flatten | index(true)) != null and ((.commands // []) | map(select(.name==\"task\") | .subcommands[] | select(.name==\"submit\") | .required_flags // []) | flatten | index(\"--workspace\")) != null and ((.commands // []) | map(select(.name==\"meta\") | .subcommands[] | select(.name==\"capabilities\") | .requires_daemon) | flatten | index(true)) != null" >/dev/null
'
run_eval "Validate CLI schema nested command discoverability and streaming capability" '
echo "$META_SCHEMA_JSON" | jq -e "((.commands // []) | map(.name) | unique | length) == ((.commands // []) | length) and ((.commands // []) | map(select(.name==\"stream\") | .supports_streaming) | flatten | index(true)) != null and ((.commands // []) | map(select(.name==\"chat\") | .supports_streaming) | flatten | index(true)) != null and ((.commands // []) | map(select(.name==\"chat\") | .machine_output_safe) | flatten | index(false)) != null and ((.commands // []) | map(select(.name==\"connector\") | .subcommands[]? | select(.name==\"twilio\") | .subcommands[]? | select(.name==\"webhook\") | .subcommands[]? | .name) | flatten | index(\"serve\")) != null and ((.commands // []) | map(select(.name==\"channel\") | .subcommands[]? | select(.name==\"mapping\") | .subcommands[]? | .name) | flatten | index(\"enable\")) != null and ((.commands // []) | map(select(.name==\"automation\") | .subcommands[]? | select(.name==\"run\") | .subcommands[]? | .name) | flatten | index(\"comm-event\")) != null and ((.commands // []) | map(select(.name==\"comm\") | .subcommands[]? | select(.name==\"policy\") | .subcommands[]? | .name) | flatten | index(\"set\")) != null" >/dev/null
'
capture_cmd META_CAPABILITIES_JSON "Emit daemon runtime capabilities metadata" pa --output json-compact meta capabilities
run_eval "Validate daemon runtime capabilities metadata payload" '
echo "$META_CAPABILITIES_JSON" | jq -e ".api_version == \"v1\" and ((.route_groups // []) | length) > 0 and ((.realtime_event_types // []) | index(\"task_run_lifecycle\")) != null and ((.client_signal_types // []) | index(\"cancel\")) != null and ((.protocol_modes // []) | index(\"http_json\")) != null and ((.transport_listener_modes // []) | index(\"tcp\")) != null" >/dev/null
'
run_eval "Run workspace readiness doctor diagnostics (allow pass/fail)" '
DOCTOR_JSON="$(pa --output json-compact doctor --workspace "$WORKSPACE")"
DOCTOR_RC=$?
printf "%s\n" "$DOCTOR_JSON"
[ "$DOCTOR_RC" -eq 0 ] || [ "$DOCTOR_RC" -eq 1 ]
'
run_eval "Validate workspace readiness doctor payload schema/check IDs" '
echo "$DOCTOR_JSON" | jq -e ".schema_version == \"1.0.0\" and (.generated_at | type) == \"string\" and .workspace_id == \"$WORKSPACE\" and ((.overall_status // \"\") | length) > 0 and ((.summary.pass // 0) >= 0) and ((.summary.warn // 0) >= 0) and ((.summary.fail // 0) >= 0) and ((.summary.skipped // 0) >= 0) and ((.checks // []) | map(.id) | index(\"daemon.connectivity\")) != null and ((.checks // []) | map(.id) | index(\"daemon.lifecycle\")) != null and ((.checks // []) | map(.id) | index(\"workspace.context\")) != null and ((.checks // []) | map(.id) | index(\"providers.readiness\")) != null and ((.checks // []) | map(.id) | index(\"models.route_readiness\")) != null and ((.checks // []) | map(.id) | index(\"channels.mappings\")) != null and ((.checks // []) | map(.id) | index(\"secrets.references\")) != null and ((.checks // []) | map(.id) | index(\"plugins.health\")) != null and ((.checks // []) | map(.id) | index(\"tooling.optional\")) != null and (((.checks // []) | map(select(.id==\"daemon.lifecycle\") | .details.control_auth.state // \"\") | first) as \$authState | (\$authState == \"configured\" or \$authState == \"missing\")) and (((.checks // []) | map(select(.id==\"daemon.lifecycle\") | .details.control_auth.source // \"\") | first) as \$authSource | (\$authSource == \"auth_token_flag\" or \$authSource == \"auth_token_file\" or \$authSource == \"unknown\"))" >/dev/null
'
run_eval "Run workspace readiness doctor quick diagnostics (allow pass/fail)" '
DOCTOR_QUICK_JSON="$(pa --output json-compact doctor --workspace "$WORKSPACE" --quick)"
DOCTOR_QUICK_RC=$?
printf "%s\n" "$DOCTOR_QUICK_JSON"
[ "$DOCTOR_QUICK_RC" -eq 0 ] || [ "$DOCTOR_QUICK_RC" -eq 1 ]
'
run_eval "Validate doctor quick-mode skipped deep checks" '
echo "$DOCTOR_QUICK_JSON" | jq -e ".schema_version == \"1.0.0\" and .workspace_id == \"$WORKSPACE\" and ((.checks // []) | map(select(.id==\"providers.readiness\") | .status) | first) == \"skipped\" and ((.checks // []) | map(select(.id==\"models.route_readiness\") | .status) | first) == \"skipped\" and ((.checks // []) | map(select(.id==\"channels.mappings\") | .status) | first) == \"skipped\" and ((.checks // []) | map(select(.id==\"secrets.references\") | .status) | first) == \"skipped\" and ((.checks // []) | map(select(.id==\"plugins.health\") | .status) | first) == \"skipped\" and ((.checks // []) | map(select(.id==\"tooling.optional\") | .status) | first) == \"skipped\" and ((.checks // []) | map(select(.id==\"daemon.connectivity\") | .status) | first) != \"skipped\" and ((.checks // []) | map(select(.id==\"daemon.lifecycle\") | .status) | first) != \"skipped\" and ((.checks // []) | map(select(.id==\"workspace.context\") | .status) | first) != \"skipped\"" >/dev/null
'
run_eval "Validate human-readable doctor text output mode" '
DOCTOR_TEXT_OUTPUT="$(pa --output text doctor --workspace "$WORKSPACE" 2>/dev/null)"
DOCTOR_TEXT_RC=$?
printf "%s\n" "$DOCTOR_TEXT_OUTPUT"
[ "$DOCTOR_TEXT_RC" -eq 0 ] || [ "$DOCTOR_TEXT_RC" -eq 1 ]
printf "%s\n" "$DOCTOR_TEXT_OUTPUT" | rg -q -- "^doctor report$"
printf "%s\n" "$DOCTOR_TEXT_OUTPUT" | rg -q -- "^overall_status:"
printf "%s\n" "$DOCTOR_TEXT_OUTPUT" | rg -q -- "daemon.connectivity"
! printf "%s\n" "$DOCTOR_TEXT_OUTPUT" | jq . >/dev/null 2>&1
'

# 5.1) Identity context and workspace selection
capture_cmd IDENTITY_BOOTSTRAP_FIRST_JSON "Bootstrap identity workspace/principal/handle (first run)" pa_daemon identity bootstrap --workspace "$WORKSPACE" --workspace-name "Manual Test Workspace" --principal actor.bootstrap.cli --display-name "CLI Bootstrap User" --actor-type human --principal-status ACTIVE --handle-channel app --handle-value "bootstrap-${WORKSPACE}" --handle-primary=true
run_eval "Validate first identity bootstrap response" '
echo "$IDENTITY_BOOTSTRAP_FIRST_JSON" | jq -e --arg ws "$WORKSPACE" ".workspace_id == \$ws and .principal_actor_id == \"actor.bootstrap.cli\" and .principal_linked == true and (.workspace_created|type)==\"boolean\" and (.principal_created|type)==\"boolean\" and .idempotent == false" >/dev/null
'
capture_cmd IDENTITY_BOOTSTRAP_SECOND_JSON "Bootstrap identity workspace/principal/handle (idempotent replay)" pa_daemon identity bootstrap --workspace "$WORKSPACE" --workspace-name "Manual Test Workspace" --principal actor.bootstrap.cli --display-name "CLI Bootstrap User" --actor-type human --principal-status ACTIVE --handle-channel app --handle-value "bootstrap-${WORKSPACE}" --handle-primary=true
run_eval "Validate idempotent identity bootstrap replay response" '
echo "$IDENTITY_BOOTSTRAP_SECOND_JSON" | jq -e ".workspace_created == false and .principal_created == false and .principal_linked == false and .handle_created == false and .idempotent == true" >/dev/null
'
capture_cmd IDENTITY_WORKSPACES_JSON "List identity workspaces" pa_daemon identity workspaces --include-inactive=true
run_eval "Resolve identity workspace target" '
IDENTITY_TARGET_WORKSPACE="$(echo "$IDENTITY_WORKSPACES_JSON" | jq -r --arg fallback "$WORKSPACE" ".active_context.workspace_id // empty | select(length > 0) // (.workspaces[0].workspace_id // empty) // \$fallback")"
test -n "$IDENTITY_TARGET_WORKSPACE"
echo "IDENTITY_TARGET_WORKSPACE=$IDENTITY_TARGET_WORKSPACE"
'
capture_cmd IDENTITY_CONTEXT_JSON "Show active identity context" pa_daemon identity context
run_eval "Validate identity context payload" '
echo "$IDENTITY_CONTEXT_JSON" | jq -e "(.active_context.workspace_id | type) == \"string\" and (.active_context.mutation_source | type) == \"string\" and (.active_context.mutation_reason | type) == \"string\" and ((.active_context.selection_version // 0) | tonumber) >= 0" >/dev/null
'
capture_cmd IDENTITY_PRINCIPALS_JSON "List identity principals for target workspace" pa_daemon identity principals --workspace "$IDENTITY_TARGET_WORKSPACE"
run_eval "Validate identity principals payload" '
echo "$IDENTITY_PRINCIPALS_JSON" | jq -e --arg ws "$IDENTITY_TARGET_WORKSPACE" ".workspace_id == \$ws and (.principals | type) == \"array\"" >/dev/null
'
capture_cmd IDENTITY_SELECT_JSON "Select identity workspace context" pa_daemon identity select-workspace --workspace "$IDENTITY_TARGET_WORKSPACE"
run_eval "Validate identity select-workspace payload" '
echo "$IDENTITY_SELECT_JSON" | jq -e --arg ws "$IDENTITY_TARGET_WORKSPACE" ".active_context.workspace_id == \$ws and (.active_context.workspace_source | type) == \"string\" and ((.active_context.selection_version // 0) | tonumber) > 0 and .active_context.mutation_reason == \"explicit_select_workspace\"" >/dev/null
'
capture_cmd CHANNEL_MAPPING_INITIAL_JSON "List message channel connector mappings" pa_daemon channel mapping list --workspace "$WORKSPACE" --channel message
run_eval "Validate initial channel mapping payload" '
echo "$CHANNEL_MAPPING_INITIAL_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .channel_id == \"message\" and .fallback_policy == \"priority_order\" and ((.bindings // []) | length) >= 2 and (((.bindings // []) | map(.connector_id)) | index(\"twilio\")) != null and (((.bindings // []) | map(.connector_id)) | index(\"imessage\")) != null" >/dev/null
'
capture_cmd CHANNEL_MAPPING_VOICE_INITIAL_JSON "List voice channel connector mappings" pa_daemon channel mapping list --workspace "$WORKSPACE" --channel voice
run_eval "Validate initial voice channel mapping payload" '
echo "$CHANNEL_MAPPING_VOICE_INITIAL_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .channel_id == \"voice\" and .fallback_policy == \"priority_order\" and ((.bindings // []) | length) >= 1 and (((.bindings // []) | map(.connector_id)) | index(\"twilio\")) != null and (((.bindings // []) | map(select(.connector_id==\"twilio\") | .enabled)) | first) == true" >/dev/null
'
capture_cmd CHANNEL_MAPPING_DISABLE_JSON "Disable Twilio mapping on message channel" pa_daemon channel mapping disable --workspace "$WORKSPACE" --channel message --connector twilio
run_eval "Validate twilio mapping disabled" '
echo "$CHANNEL_MAPPING_DISABLE_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .channel_id == \"message\" and .connector_id == \"twilio\" and .enabled == false" >/dev/null
'
capture_cmd CHANNEL_MAPPING_PRIORITIZE_JSON "Prioritize Twilio mapping on message channel" pa_daemon channel mapping prioritize --workspace "$WORKSPACE" --channel message --connector twilio --priority 1
run_eval "Validate twilio mapping priority update" '
echo "$CHANNEL_MAPPING_PRIORITIZE_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .channel_id == \"message\" and .connector_id == \"twilio\" and .priority == 1" >/dev/null
'
capture_cmd CHANNEL_MAPPING_ENABLE_JSON "Enable Twilio mapping on message channel" pa_daemon channel mapping enable --workspace "$WORKSPACE" --channel message --connector twilio --priority 1
run_eval "Validate twilio mapping enabled" '
echo "$CHANNEL_MAPPING_ENABLE_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .channel_id == \"message\" and .connector_id == \"twilio\" and .enabled == true and .priority == 1" >/dev/null
'
capture_cmd CHANNEL_MAPPING_FINAL_JSON "List message channel connector mappings after mutation" pa_daemon channel mapping list --workspace "$WORKSPACE" --channel message
run_eval "Validate final channel mapping state" '
echo "$CHANNEL_MAPPING_FINAL_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .channel_id == \"message\" and (((.bindings // []) | map(select(.connector_id==\"twilio\") | .enabled)) | first) == true and (((.bindings // []) | map(select(.connector_id==\"twilio\") | .priority)) | first) == 1" >/dev/null
'
capture_cmd CHANNEL_MAPPING_VOICE_DISABLE_JSON "Disable Twilio mapping on voice channel" pa_daemon channel mapping disable --workspace "$WORKSPACE" --channel voice --connector twilio
run_eval "Validate twilio voice mapping disabled" '
echo "$CHANNEL_MAPPING_VOICE_DISABLE_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .channel_id == \"voice\" and .connector_id == \"twilio\" and .enabled == false" >/dev/null
'
capture_cmd CHANNEL_MAPPING_VOICE_ENABLE_JSON "Enable Twilio mapping on voice channel" pa_daemon channel mapping enable --workspace "$WORKSPACE" --channel voice --connector twilio --priority 1
run_eval "Validate twilio voice mapping enabled" '
echo "$CHANNEL_MAPPING_VOICE_ENABLE_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .channel_id == \"voice\" and .connector_id == \"twilio\" and .enabled == true and .priority == 1" >/dev/null
'
capture_cmd CHANNEL_MAPPING_VOICE_FINAL_JSON "List voice channel connector mappings after mutation" pa_daemon channel mapping list --workspace "$WORKSPACE" --channel voice
run_eval "Validate final voice channel mapping state" '
echo "$CHANNEL_MAPPING_VOICE_FINAL_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .channel_id == \"voice\" and (((.bindings // []) | map(select(.connector_id==\"twilio\") | .enabled)) | first) == true and (((.bindings // []) | map(select(.connector_id==\"twilio\") | .priority)) | first) == 1" >/dev/null
'
run_eval "Validate canonical channel mapping parity after message/voice mutations" '
echo "$CHANNEL_MAPPING_FINAL_JSON" | jq -e "(((.bindings // []) | map(select(.connector_id==\"twilio\") | .enabled)) | first) == true" >/dev/null
echo "$CHANNEL_MAPPING_VOICE_FINAL_JSON" | jq -e "(((.bindings // []) | map(select(.connector_id==\"twilio\") | .enabled)) | first) == true" >/dev/null
'
run_eval "Seed identity device/session fixtures for inventory checks" '
cat > "$TEST_RUNTIME_ROOT/seed_identity_inventory.go" <<'"'"'EOF'"'"'
package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "modernc.org/sqlite"
)

func mustExec(db *sql.DB, query string, args ...any) {
	if _, err := db.Exec(query, args...); err != nil {
		panic(fmt.Sprintf("%v :: %s", err, query))
	}
}

func main() {
	if len(os.Args) < 3 {
		panic("usage: seed_identity_inventory <db-path> <workspace>")
	}
	dbPath := os.Args[1]
	workspace := os.Args[2]
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	now := "2026-02-26T06:00:00Z"
	mustExec(db, "INSERT OR REPLACE INTO workspaces(id, name, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?)", workspace, "Identity Fixture Workspace", "ACTIVE", now, now)
	mustExec(db, "INSERT OR REPLACE INTO users(id, email, display_name, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)", "user.identity.one", "one@example.com", "Identity One", "ACTIVE", now, now)
	mustExec(db, "INSERT OR REPLACE INTO users(id, email, display_name, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)", "user.identity.two", "two@example.com", "Identity Two", "ACTIVE", now, now)
	mustExec(db, "INSERT OR REPLACE INTO user_devices(id, workspace_id, user_id, device_type, platform, label, last_seen_at, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)", "device.identity.one", workspace, "user.identity.one", "phone", "ios", "Identity Phone", "2026-02-26T06:02:00Z", "2026-02-26T06:00:00Z")
	mustExec(db, "INSERT OR REPLACE INTO user_devices(id, workspace_id, user_id, device_type, platform, label, last_seen_at, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)", "device.identity.two", workspace, "user.identity.two", "desktop", "macos", "Identity Desktop", "2026-02-26T06:03:00Z", "2026-02-26T06:01:00Z")
	mustExec(db, "INSERT OR REPLACE INTO device_sessions(id, workspace_id, device_id, session_token_hash, started_at, expires_at, revoked_at) VALUES (?, ?, ?, ?, ?, ?, ?)", "session.identity.active", workspace, "device.identity.one", "hash-active", "2026-02-26T06:00:10Z", "2099-01-01T00:00:00Z", nil)
	mustExec(db, "INSERT OR REPLACE INTO device_sessions(id, workspace_id, device_id, session_token_hash, started_at, expires_at, revoked_at) VALUES (?, ?, ?, ?, ?, ?, ?)", "session.identity.expired", workspace, "device.identity.one", "hash-expired", "2026-02-25T05:00:00Z", "2026-02-25T05:30:00Z", nil)
	mustExec(db, "INSERT OR REPLACE INTO device_sessions(id, workspace_id, device_id, session_token_hash, started_at, expires_at, revoked_at) VALUES (?, ?, ?, ?, ?, ?, ?)", "session.identity.revoked", workspace, "device.identity.two", "hash-revoked", "2026-02-26T05:00:00Z", "2099-01-01T00:00:00Z", "2026-02-26T05:30:00Z")
}
EOF
go -C "$ROOT/source/services/daemon-go" run "$TEST_RUNTIME_ROOT/seed_identity_inventory.go" "$DB_PATH" "$IDENTITY_TARGET_WORKSPACE"
SEED_IDENTITY_RC=$?
rm -f "$TEST_RUNTIME_ROOT/seed_identity_inventory.go"
test "$SEED_IDENTITY_RC" -eq 0
'
capture_cmd IDENTITY_DEVICES_PAGE1_JSON "List identity devices (page 1)" pa_daemon identity devices --workspace "$IDENTITY_TARGET_WORKSPACE" --limit 1
run_eval "Validate identity devices page 1 payload" '
echo "$IDENTITY_DEVICES_PAGE1_JSON" | jq -e --arg ws "$IDENTITY_TARGET_WORKSPACE" ".workspace_id == \$ws and (.items | length) >= 1 and (.has_more|type)==\"boolean\"" >/dev/null
'
run_eval "List identity devices (page 2 via cursor when available)" '
IDENTITY_CURSOR_CREATED_AT="$(echo "$IDENTITY_DEVICES_PAGE1_JSON" | jq -r ".next_cursor_created_at // empty")"
IDENTITY_CURSOR_ID="$(echo "$IDENTITY_DEVICES_PAGE1_JSON" | jq -r ".next_cursor_id // empty")"
if [[ -n "$IDENTITY_CURSOR_CREATED_AT" && -n "$IDENTITY_CURSOR_ID" ]]; then
  IDENTITY_DEVICES_PAGE2_JSON="$(pa_daemon identity devices --workspace "$IDENTITY_TARGET_WORKSPACE" --cursor-created-at "$IDENTITY_CURSOR_CREATED_AT" --cursor-id "$IDENTITY_CURSOR_ID" --limit 5)"
else
  IDENTITY_DEVICES_PAGE2_JSON="$IDENTITY_DEVICES_PAGE1_JSON"
fi
printf "%s\n" "$IDENTITY_DEVICES_PAGE2_JSON"
'
run_eval "Validate identity devices page 2 payload" '
echo "$IDENTITY_DEVICES_PAGE2_JSON" | jq -e --arg ws "$IDENTITY_TARGET_WORKSPACE" ".workspace_id == \$ws and (.items | length) >= 1" >/dev/null
'
capture_cmd IDENTITY_SESSIONS_JSON "List active identity sessions for seeded device" pa_daemon identity sessions --workspace "$IDENTITY_TARGET_WORKSPACE" --device-id device.identity.one --session-health active --limit 5
run_eval "Validate identity sessions payload and capture session id" '
echo "$IDENTITY_SESSIONS_JSON" | jq -e --arg ws "$IDENTITY_TARGET_WORKSPACE" ".workspace_id == \$ws and (.items | length) >= 1 and .items[0].session_health == \"active\"" >/dev/null
IDENTITY_ACTIVE_SESSION_ID="$(echo "$IDENTITY_SESSIONS_JSON" | jq -r ".items[0].session_id // empty")"
echo "IDENTITY_ACTIVE_SESSION_ID=$IDENTITY_ACTIVE_SESSION_ID"
test -n "$IDENTITY_ACTIVE_SESSION_ID"
'
capture_cmd IDENTITY_REVOKE_FIRST_JSON "Revoke identity session (first call)" pa_daemon identity revoke-session --workspace "$IDENTITY_TARGET_WORKSPACE" --session-id "$IDENTITY_ACTIVE_SESSION_ID"
run_eval "Validate first identity revoke-session response" '
echo "$IDENTITY_REVOKE_FIRST_JSON" | jq -e --arg ws "$IDENTITY_TARGET_WORKSPACE" --arg sid "$IDENTITY_ACTIVE_SESSION_ID" ".workspace_id == \$ws and .session_id == \$sid and .session_health == \"revoked\" and .idempotent == false" >/dev/null
'
capture_cmd IDENTITY_REVOKE_SECOND_JSON "Revoke identity session (idempotent replay)" pa_daemon identity revoke-session --workspace "$IDENTITY_TARGET_WORKSPACE" --session-id "$IDENTITY_ACTIVE_SESSION_ID"
run_eval "Validate idempotent identity revoke-session replay response" '
echo "$IDENTITY_REVOKE_SECOND_JSON" | jq -e --arg ws "$IDENTITY_TARGET_WORKSPACE" --arg sid "$IDENTITY_ACTIVE_SESSION_ID" ".workspace_id == \$ws and .session_id == \$sid and .session_health == \"revoked\" and .idempotent == true" >/dev/null
'

# 5.2) Secret storage
run_cmd "Set OPENAI_API_KEY secret (write-only value + daemon SecretRef registration)" pa_daemon secret set --workspace "$WORKSPACE" --name OPENAI_API_KEY --value "sk-local-mock"
run_cmd "Get OPENAI_API_KEY secret metadata" pa_daemon secret get --workspace "$WORKSPACE" --name OPENAI_API_KEY

# 5.3) Provider and model setup
run_cmd "Set OpenAI provider" pa_daemon provider set --workspace "$WORKSPACE" --provider openai --endpoint "$OPENAI_MOCK_ENDPOINT/v1" --api-key-secret OPENAI_API_KEY
run_cmd "Set Ollama provider" pa_daemon provider set --workspace "$WORKSPACE" --provider ollama --endpoint "http://127.0.0.1:11434"
run_cmd "List providers" pa_daemon provider list --workspace "$WORKSPACE"
capture_cmd PROVIDER_LIST_TEXT_OUTPUT "List providers in text output mode" pa_daemon --output text provider list --workspace "$WORKSPACE"
run_eval "Validate provider list text output mode" '
printf "%s\n" "$PROVIDER_LIST_TEXT_OUTPUT" | rg -q -- "^provider list$"
printf "%s\n" "$PROVIDER_LIST_TEXT_OUTPUT" | rg -q -- "^workspace: $WORKSPACE$"
printf "%s\n" "$PROVIDER_LIST_TEXT_OUTPUT" | rg -q -- "provider=openai"
! printf "%s\n" "$PROVIDER_LIST_TEXT_OUTPUT" | jq . >/dev/null 2>&1
'
run_cmd "Check OpenAI provider" pa_daemon provider check --workspace "$WORKSPACE" --provider openai
run_cmd "List models" pa_daemon model list --workspace "$WORKSPACE"
capture_cmd MODEL_LIST_TEXT_OUTPUT "List models in text output mode" pa_daemon --output text model list --workspace "$WORKSPACE"
run_eval "Validate model list text output mode" '
printf "%s\n" "$MODEL_LIST_TEXT_OUTPUT" | rg -q -- "^model list$"
printf "%s\n" "$MODEL_LIST_TEXT_OUTPUT" | rg -q -- "^workspace: $WORKSPACE$"
printf "%s\n" "$MODEL_LIST_TEXT_OUTPUT" | rg -q -- "provider=openai"
! printf "%s\n" "$MODEL_LIST_TEXT_OUTPUT" | jq . >/dev/null 2>&1
'
capture_cmd MODEL_DISCOVER_JSON "Discover OpenAI models" pa_daemon model discover --workspace "$WORKSPACE" --provider openai
run_eval "Validate model discover response" '
echo "$MODEL_DISCOVER_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and ((.results // []) | length) >= 1 and ((.results[0].provider // \"\") | ascii_downcase) == \"openai\"" >/dev/null
'
capture_cmd MODEL_ADD_JSON "Add custom OpenAI model catalog entry" pa_daemon model add --workspace "$WORKSPACE" --provider openai --model gpt-5-codex --enabled=true
run_eval "Validate model add response" '
echo "$MODEL_ADD_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and ((.provider // \"\") | ascii_downcase) == \"openai\" and .model_key == \"gpt-5-codex\" and .enabled == true" >/dev/null
'
capture_cmd MODEL_REMOVE_JSON "Remove custom OpenAI model catalog entry" pa_daemon model remove --workspace "$WORKSPACE" --provider openai --model gpt-5-codex
run_eval "Validate model remove response" '
echo "$MODEL_REMOVE_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and ((.provider // \"\") | ascii_downcase) == \"openai\" and .model_key == \"gpt-5-codex\" and .removed == true" >/dev/null
'
run_cmd "Select chat model" pa_daemon model select --workspace "$WORKSPACE" --task-class chat --provider openai --model gpt-4.1-mini
run_cmd "Resolve chat model" pa_daemon model resolve --workspace "$WORKSPACE" --task-class chat
capture_cmd PROFILE_PROVIDER_LIST_JSON "List providers using profile defaults (no daemon flags/workspace flag)" pa provider list
run_eval "Validate profile-default provider list response" '
echo "$PROFILE_PROVIDER_LIST_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and ((.providers // []) | map((.provider // \"\") | ascii_downcase) | index(\"openai\")) != null and ((.providers // []) | map((.provider // \"\") | ascii_downcase) | index(\"ollama\")) != null" >/dev/null
'

# 5.3.1) Cloudflared connector control-plane checks
capture_cmd CLOUDFLARED_VERSION_JSON "Cloudflared connector version check" pa_daemon connector cloudflared version --workspace "$WORKSPACE"
run_eval "Validate cloudflared version response" '
echo "$CLOUDFLARED_VERSION_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .available == true and ((.binary_path // \"\") | length) > 0 and .dry_run == true" >/dev/null
'
capture_cmd CLOUDFLARED_EXEC_JSON "Cloudflared connector exec check" pa_daemon connector cloudflared exec --workspace "$WORKSPACE" --arg version
run_eval "Validate cloudflared exec response" '
echo "$CLOUDFLARED_EXEC_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .success == true and ((.args // []) | length) == 1 and .args[0] == \"version\" and .dry_run == true" >/dev/null
'

# 5.4) Chat streaming
run_cmd "Chat streaming" pa_daemon chat --workspace "$WORKSPACE" --task-class chat --message "Say hello for manual test"

# 5.5) Agent execution (non-destructive connector happy paths)
capture_cmd BROWSER_RUN_JSON "Agent run browser request" pa_daemon agent run --workspace "$WORKSPACE" --request "open https://example.com"
run_eval "Validate browser run response" 'echo "$BROWSER_RUN_JSON" | jq -e ".workflow == \"browser\" and .run_state == \"completed\"" >/dev/null'
capture_cmd MAIL_RUN_JSON "Agent run mail request" pa_daemon agent run --workspace "$WORKSPACE" --request 'send an email to recipient@example.com saying "cli update"'
run_eval "Validate mail run response" 'echo "$MAIL_RUN_JSON" | jq -e ".workflow == \"mail\" and .run_state == \"completed\"" >/dev/null'
capture_cmd CALENDAR_RUN_JSON "Agent run calendar request with explicit GO AHEAD phrase" pa_daemon agent run --workspace "$WORKSPACE" --request "schedule event with the team" --approval-phrase "GO AHEAD"
run_eval "Validate calendar run response" 'echo "$CALENDAR_RUN_JSON" | jq -e ".workflow == \"calendar\" and .run_state == \"completed\"" >/dev/null'
run_eval "Agent run messages request with explicit SMS channel" '
set +e
MESSAGES_SMS_RUN_JSON="$(pa_daemon agent run --workspace "$WORKSPACE" --request '"'"'send an sms to +15550001111: "hello"'"'"' 2>&1)"
MESSAGES_SMS_RUN_RC=$?
set -e
printf "%s\n" "$MESSAGES_SMS_RUN_JSON"
[ "$MESSAGES_SMS_RUN_RC" -eq 0 ] || [ "$MESSAGES_SMS_RUN_RC" -eq 1 ]
set +e
'
run_eval "Validate executable messages run response or actionable failure contract" '
if echo "$MESSAGES_SMS_RUN_JSON" | jq -e ".workflow == \"messages\"" >/dev/null 2>&1; then
  echo "$MESSAGES_SMS_RUN_JSON" | jq -e "((.clarification_required // false) == false) and .run_state == \"completed\" and ((.step_states // []) | length) == 1 and ((.step_states // [])[0].capability_key == \"messages_send_sms\") and (((.step_states // [])[0].evidence.channel // \"\") == \"sms\") and (((.step_states // [])[0].evidence.destination // \"\") == \"+15550001111\")" >/dev/null
else
  printf "%s\n" "$MESSAGES_SMS_RUN_JSON" | rg -q -- "^request failed$"
  printf "%s\n" "$MESSAGES_SMS_RUN_JSON" | rg -q -- "^what failed:"
  printf "%s\n" "$MESSAGES_SMS_RUN_JSON" | rg -q -- "unsupported source channel \"sms\"|unable to determine intent"
fi
'
capture_cmd FINDER_CLARIFY_JSON "Agent run clarification request missing finder target path" pa_daemon agent run --workspace "$WORKSPACE" --request "delete file now"
run_eval "Validate finder clarification response" '
echo "$FINDER_CLARIFY_JSON" | jq -e ".workflow == \"finder\" and .clarification_required == true and .task_state == \"clarification_required\" and .run_state == \"clarification_required\" and ((.missing_slots // []) | index(\"finder_query\")) != null and ((.task_id // \"\") == \"\") and ((.run_id // \"\") == \"\")" >/dev/null
'
capture_cmd MESSAGES_CLARIFY_JSON "Agent run clarification request for messages channel selection" pa_daemon agent run --workspace "$WORKSPACE" --request 'send a text to +15550001111: "hello"'
run_eval "Validate messages clarification response" '
echo "$MESSAGES_CLARIFY_JSON" | jq -e ".workflow == \"messages\" and .clarification_required == true and ((.missing_slots // []) | index(\"message_channel\")) != null and ((.native_action.messages.recipient // \"\") == \"+15550001111\") and ((.native_action.messages.body // \"\") == \"hello\")" >/dev/null
'

# 5.6) Agent destructive-action approval flow
capture_cmd DESTRUCTIVE_JSON "Agent run destructive request" pa_daemon agent run --workspace "$WORKSPACE" --request "delete file /tmp/manual-test.txt"
run_eval "Echo destructive run JSON" 'printf "%s\n" "$DESTRUCTIVE_JSON"'
run_eval "Extract approval_request_id" 'APPROVAL_ID="$(printf "%s\n" "$DESTRUCTIVE_JSON" | jq -r ".approval_request_id")"; echo "APPROVAL_ID=$APPROVAL_ID"; test -n "$APPROVAL_ID" && [ "$APPROVAL_ID" != "null" ]'
run_eval "Verify unauthorized approver is rejected" '
UNAUTHORIZED_APPROVE_OUTPUT="$(pa_daemon agent approve --workspace "$WORKSPACE" --approval-id "$APPROVAL_ID" --actor-id actor.approver --phrase "GO AHEAD" 2>&1 || true)"
printf "%s\n" "$UNAUTHORIZED_APPROVE_OUTPUT"
printf "%s\n" "$UNAUTHORIZED_APPROVE_OUTPUT" | grep -F "approval denied" >/dev/null
'
capture_cmd APPROVAL_DELEGATION_JSON "Grant approval-scope delegation" pa_daemon delegation grant --workspace "$WORKSPACE" --from actor.requester --to actor.approver --scope-type APPROVAL
run_eval "Extract approval delegation rule ID" 'APPROVAL_RULE_ID="$(printf "%s\n" "$APPROVAL_DELEGATION_JSON" | jq -r ".id")"; echo "APPROVAL_RULE_ID=$APPROVAL_RULE_ID"; test -n "$APPROVAL_RULE_ID" && [ "$APPROVAL_RULE_ID" != "null" ]'
run_cmd "Approve destructive run with delegated approver" pa_daemon agent approve --workspace "$WORKSPACE" --approval-id "$APPROVAL_ID" --actor-id actor.approver --phrase "GO AHEAD"
run_cmd "Revoke approval-scope delegation" pa_daemon delegation revoke --workspace "$WORKSPACE" --rule-id "$APPROVAL_RULE_ID"
run_eval "Extract calendar event_id from calendar create run" '
CALENDAR_EVENT_ID="$(echo "$CALENDAR_RUN_JSON" | jq -r "(.step_states // []) | map(select((.capability_key // \"\") == \"calendar_create\") | .evidence.event_id // empty) | first // empty")"
echo "CALENDAR_EVENT_ID=$CALENDAR_EVENT_ID"
test -n "$CALENDAR_EVENT_ID"
'
capture_cmd CALENDAR_DESTRUCTIVE_JSON "Agent run explicit calendar cancel request without approval phrase (calendar_cancel gate)" pa_daemon agent run --workspace "$WORKSPACE" --request "cancel calendar event id $CALENDAR_EVENT_ID"
run_eval "Validate calendar destructive approval gate response" '
echo "$CALENDAR_DESTRUCTIVE_JSON" | jq -e ".workflow == \"calendar\" and .approval_required == true and .run_state == \"awaiting_approval\" and ((.step_states // []) | map(select(.status == \"pending\" and .capability_key == \"calendar_cancel\")) | length) >= 1" >/dev/null
'
run_eval "Extract calendar approval_request_id" 'CALENDAR_APPROVAL_ID="$(printf "%s\n" "$CALENDAR_DESTRUCTIVE_JSON" | jq -r ".approval_request_id")"; echo "CALENDAR_APPROVAL_ID=$CALENDAR_APPROVAL_ID"; test -n "$CALENDAR_APPROVAL_ID" && [ "$CALENDAR_APPROVAL_ID" != "null" ]'
run_cmd "Approve calendar destructive request with acting_as principal" pa_daemon agent approve --workspace "$WORKSPACE" --approval-id "$CALENDAR_APPROVAL_ID" --actor-id actor.requester --phrase "GO AHEAD"

# 5.6.1) Voice destructive-action handoff gate
capture_cmd VOICE_DESTRUCTIVE_JSON "Voice-origin destructive run without in-app handoff confirmation" pa_daemon agent run --workspace "$WORKSPACE" --request "delete file /tmp/manual-voice-handoff.txt" --origin voice --approval-phrase "GO AHEAD"
run_eval "Validate voice-origin unconfirmed run is blocked for handoff" '
echo "$VOICE_DESTRUCTIVE_JSON" | jq -e ".approval_required == true and .run_state == \"awaiting_approval\" and ((.step_states // []) | map((.summary // \"\") | ascii_downcase | contains(\"in-app approval handoff\")) | any)" >/dev/null
'
run_eval "Extract voice handoff approval_request_id" 'VOICE_APPROVAL_ID="$(printf "%s\n" "$VOICE_DESTRUCTIVE_JSON" | jq -r ".approval_request_id")"; echo "VOICE_APPROVAL_ID=$VOICE_APPROVAL_ID"; test -n "$VOICE_APPROVAL_ID" && [ "$VOICE_APPROVAL_ID" != "null" ]'
run_cmd "Approve voice-origin pending request after handoff" pa_daemon agent approve --workspace "$WORKSPACE" --approval-id "$VOICE_APPROVAL_ID" --actor-id actor.requester --phrase "GO AHEAD"
capture_cmd VOICE_CONFIRMED_JSON "Voice-origin destructive run with in-app handoff confirmation" pa_daemon agent run --workspace "$WORKSPACE" --request "delete file /tmp/manual-voice-handoff-confirmed.txt" --origin voice --in-app-approval-confirmed=true
run_eval "Validate voice-origin confirmed run executes without pending approval" '
echo "$VOICE_CONFIRMED_JSON" | jq -e "((.approval_required // false) == false) and .run_state == \"completed\"" >/dev/null
'

# 5.7) Delegation allow/deny
run_cmd "Delegation check before grant" pa_daemon delegation check --workspace "$WORKSPACE" --requested-by actor.alice --acting-as actor.bob --scope-type EXECUTION
capture_cmd GRANT_JSON "Grant delegation" pa_daemon delegation grant --workspace "$WORKSPACE" --from actor.alice --to actor.bob --scope-type EXECUTION
run_eval "Echo delegation grant JSON" 'printf "%s\n" "$GRANT_JSON"'
run_eval "Extract delegation rule ID" 'RULE_ID="$(printf "%s\n" "$GRANT_JSON" | jq -r ".id")"; echo "RULE_ID=$RULE_ID"; test -n "$RULE_ID" && [ "$RULE_ID" != "null" ]'
run_cmd "Delegation check after grant" pa_daemon delegation check --workspace "$WORKSPACE" --requested-by actor.alice --acting-as actor.bob --scope-type EXECUTION
run_cmd "Revoke delegation rule" pa_daemon delegation revoke --workspace "$WORKSPACE" --rule-id "$RULE_ID"
run_cmd "Delegation check after revoke" pa_daemon delegation check --workspace "$WORKSPACE" --requested-by actor.alice --acting-as actor.bob --scope-type EXECUTION

# 5.8) Task submit/status/cancel/retry/requeue + canonical agent approve
capture_cmd TASK_CANCEL_JSON "Submit daemon-backed task for cancellation" pa_daemon task submit --workspace "$WORKSPACE" --requested-by actor.requester --subject actor.requester --title "manual cancel task" --description "send an email update"
run_eval "Extract task_id/run_id from cancellable task submit response" '
TASK_CANCEL_ID="$(printf "%s\n" "$TASK_CANCEL_JSON" | jq -r ".task_id")"
TASK_CANCEL_RUN_ID="$(printf "%s\n" "$TASK_CANCEL_JSON" | jq -r ".run_id")"
echo "TASK_CANCEL_ID=$TASK_CANCEL_ID"
echo "TASK_CANCEL_RUN_ID=$TASK_CANCEL_RUN_ID"
test -n "$TASK_CANCEL_ID" && [ "$TASK_CANCEL_ID" != "null" ]
test -n "$TASK_CANCEL_RUN_ID" && [ "$TASK_CANCEL_RUN_ID" != "null" ]
'
run_cmd "Cancel queued task run via task cancel command" pa_daemon task cancel --run-id "$TASK_CANCEL_RUN_ID" --reason "manual cli cancellation"
run_eval "Validate cancelled task status" '
CANCELLED_STATUS_JSON="$(pa_daemon task status --task-id "$TASK_CANCEL_ID")"
echo "$CANCELLED_STATUS_JSON" | jq -e ".task_id == \"$TASK_CANCEL_ID\" and .state == \"cancelled\" and .actions.can_cancel == false and .actions.can_retry == true and .actions.can_requeue == false" >/dev/null
'
capture_cmd TASK_STATUS_TEXT_OUTPUT "Render task status in text output mode" pa_daemon --output text task status --task-id "$TASK_CANCEL_ID"
run_eval "Validate task status text output mode" '
printf "%s\n" "$TASK_STATUS_TEXT_OUTPUT" | rg -q -- "^task status$"
printf "%s\n" "$TASK_STATUS_TEXT_OUTPUT" | rg -q -- "^task_id: $TASK_CANCEL_ID$"
printf "%s\n" "$TASK_STATUS_TEXT_OUTPUT" | rg -q -- "^actions:"
! printf "%s\n" "$TASK_STATUS_TEXT_OUTPUT" | jq . >/dev/null 2>&1
'
capture_cmd TASK_RETRY_JSON "Retry cancelled task run via task retry command" pa_daemon task retry --run-id "$TASK_CANCEL_RUN_ID" --reason "manual cli retry"
run_eval "Validate retry response and queued task status action metadata" '
TASK_RETRY_RUN_ID="$(printf "%s\n" "$TASK_RETRY_JSON" | jq -r ".run_id")"
echo "TASK_RETRY_RUN_ID=$TASK_RETRY_RUN_ID"
test -n "$TASK_RETRY_RUN_ID" && [ "$TASK_RETRY_RUN_ID" != "null" ] && [ "$TASK_RETRY_RUN_ID" != "$TASK_CANCEL_RUN_ID" ]
echo "$TASK_RETRY_JSON" | jq -e ".retried == true and .previous_run_id == \"$TASK_CANCEL_RUN_ID\" and .task_state == \"queued\" and .run_state == \"queued\" and .actions.can_cancel == true and .actions.can_retry == false and .actions.can_requeue == true" >/dev/null
RETRY_STATUS_JSON="$(pa_daemon task status --task-id "$TASK_CANCEL_ID")"
echo "$RETRY_STATUS_JSON" | jq -e ".state == \"queued\" and .run_id == \"$TASK_RETRY_RUN_ID\" and .actions.can_cancel == true and .actions.can_retry == false and .actions.can_requeue == true" >/dev/null
'
capture_cmd TASK_REQUEUE_JSON "Requeue queued task run via task requeue command" pa_daemon task requeue --run-id "$TASK_RETRY_RUN_ID" --reason "manual cli requeue"
run_eval "Validate requeue response and queued task status action metadata" '
TASK_REQUEUE_RUN_ID="$(printf "%s\n" "$TASK_REQUEUE_JSON" | jq -r ".run_id")"
echo "TASK_REQUEUE_RUN_ID=$TASK_REQUEUE_RUN_ID"
test -n "$TASK_REQUEUE_RUN_ID" && [ "$TASK_REQUEUE_RUN_ID" != "null" ] && [ "$TASK_REQUEUE_RUN_ID" != "$TASK_RETRY_RUN_ID" ]
echo "$TASK_REQUEUE_JSON" | jq -e ".requeued == true and .previous_run_id == \"$TASK_RETRY_RUN_ID\" and .task_state == \"queued\" and .run_state == \"queued\" and .actions.can_cancel == true and .actions.can_retry == false and .actions.can_requeue == true" >/dev/null
REQUEUE_STATUS_JSON="$(pa_daemon task status --task-id "$TASK_CANCEL_ID")"
echo "$REQUEUE_STATUS_JSON" | jq -e ".state == \"queued\" and .run_id == \"$TASK_REQUEUE_RUN_ID\" and .actions.can_cancel == true and .actions.can_retry == false and .actions.can_requeue == true" >/dev/null
'
capture_cmd TASK_JSON "Submit daemon-backed queued task" pa_daemon task submit --workspace "$WORKSPACE" --requested-by actor.requester --subject actor.requester --title "manual persisted task" --description "send an email update"
run_eval "Extract task_id from task submit response" 'TASK_ID="$(printf "%s\n" "$TASK_JSON" | jq -r ".task_id")"; echo "TASK_ID=$TASK_ID"; test -n "$TASK_ID" && [ "$TASK_ID" != "null" ]'
run_eval "Validate queued task status contract and actionable controls" '
TASK_STATUS_JSON="$(pa_daemon task status --task-id "$TASK_ID" 2>/dev/null || true)"
echo "$TASK_STATUS_JSON" | jq .
echo "$TASK_STATUS_JSON" | jq -e ".task_id == \"$TASK_ID\" and ((.state // \"\")|type)==\"string\" and ((.run_state // \"\")|type)==\"string\" and (.actions.can_cancel|type)==\"boolean\" and (.actions.can_retry|type)==\"boolean\" and (.actions.can_requeue|type)==\"boolean\"" >/dev/null
TASK_FINAL_STATE="$(echo "$TASK_STATUS_JSON" | jq -r ".state // empty")"
case "$TASK_FINAL_STATE" in
  queued|running|completed|failed|awaiting_approval|blocked|cancelled) true ;;
  *)
    echo "unexpected task lifecycle state after queued submit: state=$TASK_FINAL_STATE" >&2
    false
    ;;
esac
'
capture_cmd APPROVAL_DECIDE_RUN_JSON "Agent run for canonical agent approve route" pa_daemon agent run --workspace "$WORKSPACE" --request "delete file /tmp/manual-approval-decide.txt"
run_eval "Extract approval_request_id for canonical agent approve route" 'APPROVAL_DECIDE_ID="$(printf "%s\n" "$APPROVAL_DECIDE_RUN_JSON" | jq -r ".approval_request_id")"; echo "APPROVAL_DECIDE_ID=$APPROVAL_DECIDE_ID"; test -n "$APPROVAL_DECIDE_ID" && [ "$APPROVAL_DECIDE_ID" != "null" ]'
capture_cmd APPROVAL_DECIDE_JSON "Approve via canonical agent approve route" pa_daemon agent approve --workspace "$WORKSPACE" --approval-id "$APPROVAL_DECIDE_ID" --phrase "GO AHEAD" --actor-id actor.requester
run_eval "Validate canonical agent approve response" 'echo "$APPROVAL_DECIDE_JSON" | jq -e "((.approval_required // false) == false) and .task_state == \"completed\" and .run_state == \"completed\" and (.run_id|type)==\"string\" and (.run_id|length)>0" >/dev/null'

# 5.9) Communication fallback + idempotency
capture_cmd IMESSAGE_DIRECT_JSON "Comm send over Messages channel worker transport" pa_daemon comm send --workspace "$WORKSPACE" --operation-id op-manual-imessage-direct --source-channel message --destination +15555550123 --message "manual imessage direct test"
run_eval "Validate Messages direct-send payload" '
echo "$IMESSAGE_DIRECT_JSON" | jq -e ".success == true and ((.result.Channel // \"\") | length) > 0 and ((.result.Channel == \"imessage\") or (.result.Channel == \"twilio\")) and ((.result.Attempts | length) == 1)" >/dev/null
'
capture_cmd IMESSAGE_DIRECT_ATTEMPTS_JSON "Comm attempts for direct Messages send operation" pa_daemon comm attempts --workspace "$WORKSPACE" --operation-id op-manual-imessage-direct
run_eval "Validate Messages direct-send attempt ledger" '
echo "$IMESSAGE_DIRECT_ATTEMPTS_JSON" | jq -e "(.attempts | length) == 1 and ((.attempts[0].channel == \"imessage\") or (.attempts[0].channel == \"twilio\")) and (.attempts[0].status == \"sent\") and (.attempts[0].route_index == 0)" >/dev/null
'
run_cmd "Comm send with iMessage failures (fallback path)" pa_daemon comm send --workspace "$WORKSPACE" --operation-id op-manual-001 --source-channel message --destination +15555550123 --message "manual comm test" --imessage-failures 2
run_cmd "Comm attempts for operation" pa_daemon comm attempts --workspace "$WORKSPACE" --operation-id op-manual-001
run_cmd "Comm send replay with same operation ID" pa_daemon comm send --workspace "$WORKSPACE" --operation-id op-manual-001 --source-channel message --destination +15555550123 --message "manual comm test" --imessage-failures 2
run_eval "Create local Messages chat.db fixture for CLI ingest test" '
cat > "$MESSAGES_FIXTURE_DB.seed.go" <<'"'"'EOF'"'"'
package main

import (
	"database/sql"
	"os"

	_ "modernc.org/sqlite"
)

func main() {
	target := os.Args[1]
	_ = os.Remove(target)
	db, err := sql.Open("sqlite", target)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	if _, err := db.Exec("CREATE TABLE message (ROWID INTEGER PRIMARY KEY, guid TEXT, text TEXT, date INTEGER, is_from_me INTEGER, handle_id INTEGER, service TEXT);"); err != nil {
		panic(err)
	}
	if _, err := db.Exec("CREATE TABLE chat (ROWID INTEGER PRIMARY KEY, guid TEXT);"); err != nil {
		panic(err)
	}
	if _, err := db.Exec("CREATE TABLE chat_message_join (chat_id INTEGER, message_id INTEGER);"); err != nil {
		panic(err)
	}
	if _, err := db.Exec("CREATE TABLE handle (ROWID INTEGER PRIMARY KEY, id TEXT);"); err != nil {
		panic(err)
	}
	if _, err := db.Exec("INSERT INTO handle(ROWID, id) VALUES (?, ?)", 1, "+15555550100"); err != nil {
		panic(err)
	}
	if _, err := db.Exec("INSERT INTO chat(ROWID, guid) VALUES (?, ?)", 1, "chat-guid-cli-1"); err != nil {
		panic(err)
	}
	if _, err := db.Exec("INSERT INTO message(ROWID, guid, text, date, is_from_me, handle_id, service) VALUES (?, ?, ?, ?, ?, ?, ?)", 2001, "imessage-guid-cli-1", "cli inbound fixture", 1000000000, 0, 1, "iMessage"); err != nil {
		panic(err)
	}
	if _, err := db.Exec("INSERT INTO chat_message_join(chat_id, message_id) VALUES (?, ?)", 1, 2001); err != nil {
		panic(err)
	}
}
EOF
go -C "$ROOT/source/services/daemon-go" run "$MESSAGES_FIXTURE_DB.seed.go" "$MESSAGES_FIXTURE_DB"
SEED_MESSAGES_RC=$?
rm -f "$MESSAGES_FIXTURE_DB.seed.go"
test "$SEED_MESSAGES_RC" -eq 0
'
capture_cmd MESSAGES_INGEST_JSON "Ingest inbound Messages events via daemon-managed channel worker" pa_daemon channel messages ingest --workspace "$WORKSPACE" --source-scope cli-fixture-scope --source-db-path "$MESSAGES_FIXTURE_DB" --limit 10
run_eval "Validate Messages ingest response payload" '
echo "$MESSAGES_INGEST_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .source == \"apple_messages_chatdb\" and .source_scope == \"cli-fixture-scope\" and ((.polled // 0) >= 0) and ((.accepted // 0) >= 0) and ((.replayed // 0) >= 0)" >/dev/null
'
capture_cmd MAIL_INGEST_JSON "Ingest inbound Mail rule event via daemon API" pa_daemon connector mail ingest --workspace "$WORKSPACE" --source-scope "mailbox://inbox" --source-event-id "mail-cli-event-1" --source-cursor "7001" --message-id "<mail-cli-event-1@example.com>" --thread-ref "mail-cli-thread-1" --in-reply-to "<mail-root@example.com>" --references-header "<mail-root@example.com>" --from "sender@example.com" --to "recipient@example.com" --subject "CLI mail ingest fixture" --body "CLI mail ingest body" --occurred-at "2026-02-24T10:00:00Z"
run_eval "Validate Mail ingest response payload" '
echo "$MAIL_INGEST_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .source == \"apple_mail_rule\" and .source_scope == \"mailbox://inbox\" and .source_event_id == \"mail-cli-event-1\" and .accepted == true and .replayed == false and (.event_id|type)==\"string\" and (.event_id|length)>0 and (.thread_id|type)==\"string\" and (.thread_id|length)>0" >/dev/null
'
capture_cmd CALENDAR_INGEST_JSON "Ingest calendar change event via daemon API" pa_daemon connector calendar ingest --workspace "$WORKSPACE" --source-scope "calendar://primary" --source-event-id "calendar-cli-event-1" --source-cursor "7002" --calendar-id "calendar-primary" --calendar-name "Primary" --event-uid "calendar-event-uid-1" --change-type "updated" --title "CLI calendar ingest fixture" --notes "CLI calendar ingest body" --location "Room 42" --starts-at "2026-02-24T10:30:00Z" --ends-at "2026-02-24T11:00:00Z" --occurred-at "2026-02-24T10:10:00Z"
run_eval "Validate calendar ingest response payload" '
echo "$CALENDAR_INGEST_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .source == \"apple_calendar_eventkit\" and .source_scope == \"calendar://primary\" and .source_event_id == \"calendar-cli-event-1\" and .accepted == true and .replayed == false and .change_type == \"updated\" and (.event_id|type)==\"string\" and (.event_id|length)>0 and (.thread_id|type)==\"string\" and (.thread_id|length)>0" >/dev/null
'
capture_cmd BROWSER_INGEST_JSON "Ingest browser extension event via daemon API" pa_daemon connector browser ingest --workspace "$WORKSPACE" --source-scope "safari://window/cli-1" --source-event-id "browser-cli-event-1" --source-cursor "7003" --window-id "window-cli-1" --tab-id "tab-cli-1" --page-url "https://example.com" --page-title "Example Domain" --event-type "navigation" --payload "CLI browser ingest payload" --occurred-at "2026-02-24T10:20:00Z"
run_eval "Validate browser ingest response payload" '
echo "$BROWSER_INGEST_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .source == \"apple_safari_extension\" and .source_scope == \"safari://window/cli-1\" and .source_event_id == \"browser-cli-event-1\" and .accepted == true and .replayed == false and .event_type == \"navigation\" and (.event_id|type)==\"string\" and (.event_id|length)>0 and (.thread_id|type)==\"string\" and (.thread_id|length)>0" >/dev/null
'
capture_cmd BRIDGE_STATUS_JSON "Read local ingest bridge readiness status" pa_daemon connector bridge status --workspace "$WORKSPACE"
run_eval "Validate local ingest bridge status payload" '
echo "$BRIDGE_STATUS_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and ((.status.sources | length) == 3) and (.status.inbox_root|type)==\"string\" and (.status.ready|type)==\"boolean\"" >/dev/null
'
capture_cmd BRIDGE_SETUP_JSON "Ensure local ingest bridge queue paths" pa_daemon connector bridge setup --workspace "$WORKSPACE"
run_eval "Validate local ingest bridge setup payload" '
echo "$BRIDGE_SETUP_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .ensure_applied == true and .status.ready == true and ((.status.sources | length) == 3)" >/dev/null
'
capture_cmd MAIL_HANDOFF_JSON "Queue mail watcher handoff via local ingest bridge helper" pa_daemon connector mail handoff --workspace "$WORKSPACE" --source-scope "mailbox://inbox" --source-event-id "mail-cli-handoff-1" --source-cursor "7101" --message-id "<mail-cli-handoff-1@example.com>" --thread-ref "mail-cli-handoff-thread-1" --from "sender@example.com" --to "recipient@example.com" --subject "CLI mail handoff fixture" --body "CLI mail handoff body" --occurred-at "2026-02-24T10:40:00Z"
run_eval "Validate mail handoff queue response payload" '
echo "$MAIL_HANDOFF_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .source == \"mail\" and .source_event_id == \"mail-cli-handoff-1\" and .queued == true and ((.file_path // \"\")|length)>0" >/dev/null
'
capture_cmd CALENDAR_HANDOFF_JSON "Queue calendar watcher handoff via local ingest bridge helper" pa_daemon connector calendar handoff --workspace "$WORKSPACE" --source-scope "calendar://primary" --source-event-id "calendar-cli-handoff-1" --source-cursor "7102" --calendar-id "calendar-primary" --calendar-name "Primary" --event-uid "calendar-handoff-uid-1" --change-type "updated" --title "CLI calendar handoff fixture" --notes "CLI calendar handoff body" --location "Room 43" --starts-at "2026-02-24T10:45:00Z" --ends-at "2026-02-24T11:15:00Z" --occurred-at "2026-02-24T10:41:00Z"
run_eval "Validate calendar handoff queue response payload" '
echo "$CALENDAR_HANDOFF_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .source == \"calendar\" and .source_event_id == \"calendar-cli-handoff-1\" and .queued == true and ((.file_path // \"\")|length)>0" >/dev/null
'
capture_cmd BROWSER_HANDOFF_JSON "Queue browser watcher handoff via local ingest bridge helper" pa_daemon connector browser handoff --workspace "$WORKSPACE" --source-scope "safari://window/cli-2" --source-event-id "browser-cli-handoff-1" --source-cursor "7103" --window-id "window-cli-2" --tab-id "tab-cli-2" --page-url "https://example.com" --page-title "Example Domain" --event-type "navigation" --payload "CLI browser handoff payload" --occurred-at "2026-02-24T10:42:00Z"
run_eval "Validate browser handoff queue response payload" '
echo "$BROWSER_HANDOFF_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .source == \"browser\" and .source_event_id == \"browser-cli-handoff-1\" and .queued == true and ((.file_path // \"\")|length)>0" >/dev/null
'

# 5.10) Automation create/list/run
run_cmd "Create ON_COMM_EVENT automation" pa_daemon automation create --workspace "$WORKSPACE" --subject actor.requester --trigger-type ON_COMM_EVENT --filter '{"channels":["imessage"]}'
run_cmd "List automations" pa_daemon automation list --workspace "$WORKSPACE"
run_cmd "Run comm-event automation (first)" pa_daemon automation run comm-event --workspace "$WORKSPACE" --event-id manual-evt-1 --channel imessage --body "hello" --sender sender@example.com
run_cmd "Run comm-event automation (replay)" pa_daemon automation run comm-event --workspace "$WORKSPACE" --event-id manual-evt-1 --channel imessage --body "hello" --sender sender@example.com

# 5.11) Inspect + retention + context
capture_cmd RUN_JSON "Agent run for inspect flow" pa_daemon agent run --workspace "$WORKSPACE" --request 'send an email to recipient@example.com saying "inspect flow update"'
run_eval "Echo inspect run JSON" 'printf "%s\n" "$RUN_JSON"'
run_eval "Extract inspect run_id" 'RUN_ID="$(printf "%s\n" "$RUN_JSON" | jq -r ".run_id")"; echo "RUN_ID=$RUN_ID"; test -n "$RUN_ID" && [ "$RUN_ID" != "null" ]'
run_cmd "Inspect run" pa_daemon inspect run --run-id "$RUN_ID"
run_cmd "Inspect transcript" pa_daemon inspect transcript --workspace "$WORKSPACE" --limit 20
run_cmd "Inspect memory" pa_daemon inspect memory --workspace "$WORKSPACE" --limit 20
run_cmd "Retention purge" pa_daemon retention purge --trace-days 7 --transcript-days 7 --memory-days 7
run_cmd "Retention compact memory" pa_daemon retention compact-memory --workspace "$WORKSPACE" --owner actor.requester --apply
run_cmd "Context samples" pa_daemon context samples --workspace "$WORKSPACE" --task-class chat --limit 20
run_cmd "Context tune" pa_daemon context tune --workspace "$WORKSPACE" --task-class chat

# 6.1) Twilio setup/check
run_cmd "Twilio channel set" pa_daemon connector twilio set --workspace "$WORKSPACE" --account-sid AC123 --auth-token twilio-local-token --sms-number +15555550001 --voice-number +15555550002 --endpoint "$TWILIO_MOCK_ENDPOINT"
capture_cmd TWILIO_GET_JSON "Twilio channel get" pa_daemon connector twilio get --workspace "$WORKSPACE"
run_eval "Validate unified Twilio config-once payload" '
echo "$TWILIO_GET_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and ((.account_sid_secret_name // \"\") | length) > 0 and ((.auth_token_secret_name // \"\") | length) > 0 and ((.sms_number // \"\") == \"+15555550001\") and ((.voice_number // \"\") == \"+15555550002\") and ((.endpoint // \"\") == \"$TWILIO_MOCK_ENDPOINT\") and .credentials_configured == true" >/dev/null
'
run_cmd "Twilio channel check" pa_daemon connector twilio check --workspace "$WORKSPACE"
capture_cmd CHANNEL_MAPPING_POST_TWILIO_MESSAGE_JSON "List message channel mappings after Twilio config-once setup" pa_daemon channel mapping list --workspace "$WORKSPACE" --channel message
capture_cmd CHANNEL_MAPPING_POST_TWILIO_VOICE_JSON "List voice channel mappings after Twilio config-once setup" pa_daemon channel mapping list --workspace "$WORKSPACE" --channel voice
run_eval "Validate Twilio mapping parity after unified Twilio setup" '
echo "$CHANNEL_MAPPING_POST_TWILIO_MESSAGE_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .channel_id == \"message\" and (((.bindings // []) | map(select(.connector_id==\"twilio\") | .enabled)) | first) == true" >/dev/null
echo "$CHANNEL_MAPPING_POST_TWILIO_VOICE_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .channel_id == \"voice\" and (((.bindings // []) | map(select(.connector_id==\"twilio\") | .enabled)) | first) == true" >/dev/null
'

# 6.2) SMS workflow + replay-safe ingest
run_cmd "Twilio sms-chat" pa_daemon connector twilio sms-chat --workspace "$WORKSPACE" --to +15555550999 --message "twilio sms chat test"
run_cmd "Twilio ingest-sms first" pa_daemon connector twilio ingest-sms --workspace "$WORKSPACE" --skip-signature=true --from +15555550999 --to +15555550001 --body "inbound sms one" --message-sid SMINMANUAL1 --account-sid AC123
run_cmd "Twilio ingest-sms replay" pa_daemon connector twilio ingest-sms --workspace "$WORKSPACE" --skip-signature=true --from +15555550999 --to +15555550001 --body "inbound sms one" --message-sid SMINMANUAL1 --account-sid AC123

# 6.3) Voice workflow + replay-safe ingest
capture_cmd START_CALL_JSON "Twilio start-call" pa_daemon connector twilio start-call --workspace "$WORKSPACE" --to +15555550999 --twiml-url https://agent.local/twiml/voice
run_eval "Echo start-call JSON" 'printf "%s\n" "$START_CALL_JSON"'
run_eval "Extract call_sid" 'CALL_SID="$(printf "%s\n" "$START_CALL_JSON" | jq -r ".call_sid")"; echo "CALL_SID=$CALL_SID"; test -n "$CALL_SID" && [ "$CALL_SID" != "null" ]'
run_cmd "Twilio ingest-voice first" pa_daemon connector twilio ingest-voice --workspace "$WORKSPACE" --skip-signature=true --provider-event-id voice-manual-1 --call-sid "$CALL_SID" --account-sid AC123 --from +15555550002 --to +15555550999 --direction outbound-api --call-status in-progress --transcript "voice transcript one" --transcript-direction INBOUND
run_cmd "Twilio ingest-voice replay" pa_daemon connector twilio ingest-voice --workspace "$WORKSPACE" --skip-signature=true --provider-event-id voice-manual-1 --call-sid "$CALL_SID" --account-sid AC123 --from +15555550002 --to +15555550999 --direction outbound-api --call-status in-progress --transcript "voice transcript one" --transcript-direction INBOUND
run_cmd "Twilio call-status" pa_daemon connector twilio call-status --workspace "$WORKSPACE" --call-sid "$CALL_SID"
run_cmd "Twilio transcript for call" pa_daemon connector twilio transcript --workspace "$WORKSPACE" --call-sid "$CALL_SID" --limit 20

# 7) Conversational webhook runtime
run_eval "Start conversational webhook runtime in background" 'WEBHOOK_SERVE_LOG="$TEST_RUNTIME_ROOT/webhook-serve-cli.log"; rm -f "$WEBHOOK_SERVE_LOG"; pa_daemon --timeout 70s connector twilio webhook serve --workspace "$WORKSPACE" --listen "$WEBHOOK_LISTEN_ADDR" --signature-mode bypass --cloudflared-mode auto --assistant-replies=true --assistant-task-class chat --voice-response-mode twiml --run-for 60s >"$WEBHOOK_SERVE_LOG" 2>&1 & WEBHOOK_PID=$!; echo "WEBHOOK_PID=$WEBHOOK_PID"; echo "WEBHOOK_SERVE_LOG=$WEBHOOK_SERVE_LOG"'
run_cmd "Wait for webhook runtime" wait_for_http "http://$WEBHOOK_LISTEN_ADDR/" 300 any
run_cmd "Webhook inbound SMS curl" curl -s -X POST "http://$WEBHOOK_LISTEN_ADDR${TWILIO_SMS_WEBHOOK_PATH}" --data-urlencode "From=+15555550999" --data-urlencode "To=+15555550001" --data-urlencode "Body=Please reply from webhook mode" --data-urlencode "MessageSid=SMWEBHOOKMANUAL1" --data-urlencode "AccountSid=AC123"
run_cmd "Webhook inbound voice curl" curl -s -X POST "http://$WEBHOOK_LISTEN_ADDR${TWILIO_VOICE_WEBHOOK_PATH}" --data-urlencode "CallSid=CAWEBHOOKMANUAL1" --data-urlencode "AccountSid=AC123" --data-urlencode "From=+15555550999" --data-urlencode "To=+15555550002" --data-urlencode "Direction=inbound" --data-urlencode "CallStatus=in-progress" --data-urlencode "SpeechResult=Hello from voice webhook"
run_cmd "Post-webhook transcript check" pa_daemon connector twilio transcript --workspace "$WORKSPACE" --limit 30
run_cmd "Post-webhook call-status check" pa_daemon connector twilio call-status --workspace "$WORKSPACE" --limit 20
run_eval "Wait for webhook runtime exit" '
if [[ -n "$WEBHOOK_PID" ]]; then
  wait "$WEBHOOK_PID"
  rc=$?
  echo "webhook runtime exit=$rc"
  if [[ -n "$WEBHOOK_SERVE_LOG" && -f "$WEBHOOK_SERVE_LOG" ]]; then
    tail -n 30 "$WEBHOOK_SERVE_LOG" || true
  fi
  WEBHOOK_PID=""
  [ "$rc" -eq 0 ] || [ "$rc" -eq 1 ]
else
  true
fi
'

# 8) Daemon guide reference
run_eval "Skip daemon tests in CLI runner" 'echo "daemon transport validation is covered by ./tools/scripts/run_tests_daemon.sh"'

# 9) Live Twilio/carrier validation
if [[ "$LIVE_MODE" == "strict" ]]; then
  run_eval "Live Twilio smoke script (strict)" 'cd "$ROOT" && ./tools/scripts/twilio_live_cli_smoke.sh all'
elif [[ "$LIVE_MODE" == "skip" ]]; then
  run_eval "Live Twilio smoke script (skip missing env)" 'cd "$ROOT" && ./tools/scripts/twilio_live_cli_smoke.sh all --skip-missing-env'
else
  run_eval "Skip live Twilio smoke" 'echo "live Twilio validation skipped by --live-mode off"'
fi

# 10) Cleanup
run_eval "Stop daemon process" 'if [[ -n "$DAEMON_PID" ]] && kill -0 "$DAEMON_PID" >/dev/null 2>&1; then kill "$DAEMON_PID" 2>/dev/null || true; wait "$DAEMON_PID" >/dev/null 2>&1 || true; DAEMON_PID=""; fi'
run_eval "Stop mock servers" 'if [[ -n "$MOCKS_PID" ]] && kill -0 "$MOCKS_PID" >/dev/null 2>&1; then kill "$MOCKS_PID" 2>/dev/null || true; wait "$MOCKS_PID" >/dev/null 2>&1 || true; MOCKS_PID=""; fi'
run_cmd "Remove CLI manual test DB" rm -f "$DB_PATH"

echo
echo "================================================================"
echo "CLI manual tests completed at $(date -u +'%Y-%m-%dT%H:%M:%SZ')"
echo "Log file: $LOG_FILE"
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
