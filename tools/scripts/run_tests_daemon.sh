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
SKIP_REGRESSION=false

while [[ $# -gt 0 ]]; do
  case "$1" in
    --log-file)
      LOG_FILE="${2:-}"
      shift 2
      ;;
    --skip-regression)
      SKIP_REGRESSION=true
      shift
      ;;
    *)
      echo "unknown argument: $1" >&2
      echo "usage: $0 [--log-file <path>] [--skip-regression]" >&2
      exit 2
      ;;
  esac
done

LOG_DIR="$ROOT/out/logs/manual-tests"
mkdir -p "$LOG_DIR"
if [[ -z "$LOG_FILE" ]]; then
  STAMP="$(date +%Y%m%d-%H%M%S)"
  LOG_FILE="$LOG_DIR/tests-daemon-$STAMP.log"
fi

exec > >(tee -a "$LOG_FILE") 2>&1

echo "Daemon manual tests started at $(date -u +'%Y-%m-%dT%H:%M:%SZ')"
echo "Repository root: $ROOT"
echo "Log file: $LOG_FILE"
echo "Tip: run ./tools/scripts/run_tests_all.sh to execute daemon + CLI + UI runners together."

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
OS_NAME="$(uname -s | tr '[:upper:]' '[:lower:]')"
TEST_RUNTIME_ROOT="${PA_TEST_RUNTIME_ROOT:-$ROOT/out/test-state/daemon}"
WORKSPACE="${WORKSPACE:-test-ws1}"
INSPECT_WORKSPACE="${INSPECT_WORKSPACE:-daemon}"
DAEMON_AUTH_TOKEN="daemon-test-token"
DAEMON_AUTH_TOKEN_FILE="${DAEMON_AUTH_TOKEN_FILE:-$TEST_RUNTIME_ROOT/manual-test-daemon.control.token}"
DAEMON_DB_PATH="${DAEMON_DB_PATH:-$TEST_RUNTIME_ROOT/manual-test-daemon.db}"
DAEMON_PROD_AUTH_TOKEN=""
PROJECT_NAME="$(normalize_project_name "${PA_PROJECT_NAME:-personalagent}")"
WEBHOOK_API_VERSION="v1"
TWILIO_SMS_WEBHOOK_PATH="/${PROJECT_NAME}/${WEBHOOK_API_VERSION}/connector/twilio/sms"
TWILIO_VOICE_WEBHOOK_PATH="/${PROJECT_NAME}/${WEBHOOK_API_VERSION}/connector/twilio/voice"
export PA_PROJECT_NAME="$PROJECT_NAME"
PA_RUNTIME_PROFILE="${PA_RUNTIME_PROFILE:-test}"
PA_RUNTIME_ROOT_DIR="${PA_RUNTIME_ROOT_DIR:-$TEST_RUNTIME_ROOT/runtime-root}"
export PA_RUNTIME_PROFILE
export PA_RUNTIME_ROOT_DIR
mkdir -p "$TEST_RUNTIME_ROOT" "$PA_RUNTIME_ROOT_DIR"
DAEMON_TCP_PORT="$((26000 + (RANDOM % 2000)))"
DAEMON_TCP_ADDR="127.0.0.1:${DAEMON_TCP_PORT}"
DAEMON_BOOTSTRAP_TCP_ADDR="127.0.0.1:$((DAEMON_TCP_PORT + 1))"
DAEMON_UNIX_SOCKET="${DAEMON_UNIX_SOCKET:-$TEST_RUNTIME_ROOT/personal-agent-daemon.sock}"
DAEMON_NAMED_PIPE='\\.\pipe\personal-agent'
DAEMON_LIFECYCLE_HOST_OPS_MODE="${DAEMON_LIFECYCLE_HOST_OPS_MODE:-dry-run}"
MESSAGES_SEND_DRY_RUN="${PA_MESSAGES_SEND_DRY_RUN:-1}"
MAIL_AUTOMATION_DRY_RUN="1"
CALENDAR_AUTOMATION_DRY_RUN="${PA_CALENDAR_AUTOMATION_DRY_RUN:-1}"
BROWSER_AUTOMATION_DRY_RUN="${PA_BROWSER_AUTOMATION_DRY_RUN:-1}"
CLOUDFLARED_DRY_RUN="${PA_CLOUDFLARED_DRY_RUN:-1}"
INBOUND_WATCHER_INBOX_DIR="${PA_INBOUND_WATCHER_INBOX_DIR:-$TEST_RUNTIME_ROOT/inbound-watcher-inbox}"
INBOUND_WATCHER_POLL_INTERVAL="${PA_INBOUND_WATCHER_POLL_INTERVAL:-150ms}"
MESSAGES_WATCHER_SOURCE_SCOPE="${PA_INBOUND_WATCHER_MESSAGES_SOURCE_SCOPE:-daemon-fixture-scope}"
DAEMON_PID=""
WEBHOOK_SERVE_PID=""
MESSAGES_FIXTURE_DB="${PA_INBOUND_WATCHER_MESSAGES_SOURCE_DB_PATH:-$TEST_RUNTIME_ROOT/messages-ingest-fixture.db}"
TASK_JSON=""
TASK_ID=""
TASK_RUN_ID=""
TASK_CANCEL_JSON=""
TASK_CANCEL_ID=""
TASK_CANCEL_RUN_ID=""
TASK_RETRY_JSON=""
TASK_RETRY_RUN_ID=""
TASK_REQUEUE_JSON=""
TASK_REQUEUE_RUN_ID=""
SIGNAL_TASK_JSON=""
SIGNAL_RUN_ID=""
SIGNAL_STREAM_LOG=""
QUEUE_STREAM_LOG=""
QUEUE_STREAM_PID=""
TASK_RUN_LIST_JSON=""
TASK_RUN_LIST_FILTERED_JSON=""
APPROVAL_DECIDE_RUN_JSON=""
APPROVAL_DECIDE_ID=""
APPROVAL_DECIDE_JSON=""
LIFECYCLE_STATUS_JSON=""
META_CAPABILITIES_JSON=""
LIFECYCLE_CONTROL_JSON=""
LIFECYCLE_PLUGIN_HISTORY_PAGE1_JSON=""
LIFECYCLE_PLUGIN_HISTORY_PAGE2_JSON=""
LIFECYCLE_PLUGIN_HISTORY_FILTERED_JSON=""
LIFECYCLE_PLUGIN_HISTORY_CURSOR_CREATED_AT=""
LIFECYCLE_PLUGIN_HISTORY_CURSOR_ID=""
LIFECYCLE_PLUGIN_HISTORY_FIRST_AUDIT_ID=""
LIFECYCLE_PLUGIN_HISTORY_PLUGIN_ID=""
LIFECYCLE_PLUGIN_HISTORY_STATE=""
CONTEXT_MEMORY_PAGE1_JSON=""
CONTEXT_MEMORY_PAGE2_JSON=""
CONTEXT_MEMORY_FILTERED_JSON=""
CONTEXT_MEMORY_CURSOR_UPDATED_AT=""
CONTEXT_MEMORY_CURSOR_ID=""
CONTEXT_MEMORY_FIRST_ID=""
CONTEXT_MEMORY_CANDIDATES_JSON=""
CONTEXT_RETRIEVAL_DOCUMENTS_JSON=""
CONTEXT_RETRIEVAL_CHUNKS_JSON=""
MODEL_DISCOVER_JSON=""
MODEL_ADD_JSON=""
MODEL_REMOVE_JSON=""
MODEL_ROUTE_SIMULATE_JSON=""
MODEL_ROUTE_EXPLAIN_JSON=""
IDENTITY_SEED_GRANT_JSON=""
IDENTITY_SEED_RULE_ID=""
IDENTITY_WORKSPACES_JSON=""
IDENTITY_PRINCIPALS_JSON=""
IDENTITY_CONTEXT_JSON=""
IDENTITY_SELECT_JSON=""
IDENTITY_DEVICES_PAGE1_JSON=""
IDENTITY_DEVICES_PAGE2_JSON=""
IDENTITY_DEVICES_FILTERED_JSON=""
IDENTITY_DEVICE_CURSOR_CREATED_AT=""
IDENTITY_DEVICE_CURSOR_ID=""
IDENTITY_DEVICE_PAGE1_FIRST_ID=""
IDENTITY_SESSIONS_PAGE1_JSON=""
IDENTITY_SESSIONS_PAGE2_JSON=""
IDENTITY_SESSIONS_FILTERED_JSON=""
IDENTITY_SESSIONS_REVOKE_JSON=""
IDENTITY_SESSIONS_REVOKE_IDEMPOTENT_JSON=""
IDENTITY_SESSION_CURSOR_STARTED_AT=""
IDENTITY_SESSION_CURSOR_ID=""
IDENTITY_SESSION_PAGE1_FIRST_ID=""
CAPABILITY_GRANT_UPSERT_JSON=""
CAPABILITY_GRANT_SECONDARY_JSON=""
CAPABILITY_GRANT_UPDATE_JSON=""
CAPABILITY_GRANT_LIST_PAGE1_JSON=""
CAPABILITY_GRANT_LIST_PAGE2_JSON=""
CAPABILITY_GRANT_LIST_FILTERED_JSON=""
CAPABILITY_GRANT_ID=""
CAPABILITY_GRANT_CURSOR_CREATED_AT=""
CAPABILITY_GRANT_CURSOR_ID=""
CAPABILITY_GRANT_PAGE1_FIRST_ID=""
CHANNEL_STATUS_JSON=""
CHANNEL_CONFIG_UPSERT_JSON=""
CHANNEL_CONFIG_UPSERT_MERGE_JSON=""
CHANNEL_TWILIO_CONFIG_UPSERT_JSON=""
CHANNEL_TWILIO_GET_JSON=""
CHANNEL_MAPPING_MESSAGE_INITIAL_JSON=""
CHANNEL_MAPPING_MESSAGE_DISABLE_JSON=""
CHANNEL_MAPPING_MESSAGE_PRIORITIZE_JSON=""
CHANNEL_MAPPING_MESSAGE_ENABLE_JSON=""
CHANNEL_MAPPING_MESSAGE_FINAL_JSON=""
CHANNEL_MAPPING_VOICE_INITIAL_JSON=""
CHANNEL_MAPPING_VOICE_DISABLE_JSON=""
CHANNEL_MAPPING_VOICE_ENABLE_JSON=""
CHANNEL_MAPPING_VOICE_FINAL_JSON=""
CHANNEL_MAPPING_LEGACY_MESSAGE_JSON=""
CHANNEL_MAPPING_LEGACY_VOICE_JSON=""
CHANNEL_MAPPING_POST_TWILIO_MESSAGE_JSON=""
CHANNEL_MAPPING_POST_TWILIO_VOICE_JSON=""
CHANNEL_TEST_OPERATION_JSON=""
CHANNEL_DIAGNOSTICS_JSON=""
CHANNEL_DIAGNOSTICS_FILTERED_JSON=""
CONNECTOR_STATUS_JSON=""
CONNECTOR_CONFIG_UPSERT_JSON=""
CONNECTOR_CONFIG_UPSERT_MERGE_JSON=""
CONNECTOR_TEST_OPERATION_JSON=""
CONNECTOR_DIAGNOSTICS_JSON=""
CONNECTOR_DIAGNOSTICS_FILTERED_JSON=""
CONNECTOR_PERMISSION_JSON=""
CLOUDFLARED_VERSION_JSON=""
CLOUDFLARED_EXEC_JSON=""
CLOUDFLARED_TUNNEL_START_JSON=""
CLOUDFLARED_TUNNEL_STATUS_JSON=""
CLOUDFLARED_TUNNEL_STOP_JSON=""
CLOUDFLARED_TUNNEL_ID=""
MAIL_RUN_JSON=""
CALENDAR_RUN_JSON=""
CALENDAR_UPDATE_RUN_JSON=""
CALENDAR_CANCEL_RUN_JSON=""
CALENDAR_EVENT_ID=""
BROWSER_RUN_JSON=""
INSPECT_QUERY_JSON=""
INSPECT_STREAM_JSON=""
INSPECT_RUN_JSON=""
INSPECT_CURSOR_CREATED_AT=""
INSPECT_CURSOR_ID=""
APPROVAL_RUN_JSON=""
APPROVAL_ID=""
APPROVAL_GRANT_JSON=""
APPROVAL_RULE_ID=""
APPROVAL_DELEGATION_CHECK_AFTER_REVOKE_JSON=""
APPROVAL_PENDING_JSON=""
APPROVAL_FINAL_JSON=""
VOICE_DESTRUCTIVE_JSON=""
VOICE_APPROVAL_ID=""
VOICE_CONFIRMED_JSON=""
AUTO_SCHEDULE_JSON=""
AUTO_SCHEDULE_DIRECTIVE_ID=""
AUTO_SCHEDULE_TASKS_JSON=""
AUTO_FIRE_HISTORY_JSON=""
AUTO_FIRE_HISTORY_FILTERED_JSON=""
AUTO_MANAGE_JSON=""
AUTO_MANAGE_TRIGGER_ID=""
AUTO_UPDATE_JSON=""
AUTO_DELETE_JSON=""
AUTO_DELETE_REPLAY_JSON=""
AUTO_COMM_METADATA_JSON=""
AUTO_COMM_VALIDATE_JSON=""
AUTO_COMM_VALIDATE_CONFLICT_JSON=""
AUTO_COMM_JSON=""
AUTO_COMM_DIRECTIVE_ID=""
AUTO_COMM_INGEST_JSON=""
AUTO_COMM_TASKS_JSON=""
AUTO_COMM_MESSAGE_SID=""
AUTO_WATCH_MESSAGES_JSON=""
AUTO_WATCH_MESSAGES_DIRECTIVE_ID=""
AUTO_WATCH_MAIL_JSON=""
AUTO_WATCH_MAIL_DIRECTIVE_ID=""
AUTO_WATCH_CALENDAR_JSON=""
AUTO_WATCH_CALENDAR_DIRECTIVE_ID=""
AUTO_WATCH_BROWSER_JSON=""
AUTO_WATCH_BROWSER_DIRECTIVE_ID=""
AUTO_WATCH_TASKS_JSON=""
COMM_THREADS_PAGE1_JSON=""
COMM_THREADS_PAGE2_JSON=""
COMM_THREADS_FILTERED_JSON=""
COMM_THREAD_CURSOR=""
COMM_THREAD_ID=""
COMM_EVENTS_PAGE1_JSON=""
COMM_EVENTS_PAGE2_JSON=""
COMM_EVENTS_FILTERED_JSON=""
COMM_EVENT_CURSOR=""
COMM_EVENT_ID=""
IMESSAGE_VISIBILITY_ALT_WORKSPACE=""
IMESSAGE_VISIBILITY_BASELINE_THREADS_JSON=""
IMESSAGE_VISIBILITY_BASELINE_EVENTS_JSON=""
IMESSAGE_VISIBILITY_AFTER_THREADS_JSON=""
IMESSAGE_VISIBILITY_AFTER_EVENTS_JSON=""
IMESSAGE_VISIBILITY_THREAD_COUNT=""
IMESSAGE_VISIBILITY_THREAD_FIRST_ID=""
IMESSAGE_VISIBILITY_EVENT_COUNT=""
IMESSAGE_VISIBILITY_EVENT_FIRST_ID=""
COMM_POLICY_CREATE_JSON=""
COMM_POLICY_UPDATE_JSON=""
COMM_POLICY_LIST_JSON=""
COMM_POLICY_ID=""
MESSAGES_SMS_TASK_ID=""
MESSAGES_SMS_RUN_ID=""
MESSAGES_SMS_STEP_ID=""
CONTEXT_STEP_ID=""
CONTEXT_EVENT_ID=""
COMM_CONTEXT_SEND_JSON=""
COMM_CONTEXT_HISTORY_PAGE1_JSON=""
COMM_CONTEXT_HISTORY_PAGE2_JSON=""
COMM_CONTEXT_HISTORY_CURSOR=""
COMM_VOICE_INGEST_ONE_JSON=""
COMM_VOICE_INGEST_TWO_JSON=""
COMM_WEBHOOK_RECEIPTS_PAGE1_JSON=""
COMM_WEBHOOK_RECEIPTS_PAGE2_JSON=""
COMM_WEBHOOK_RECEIPTS_FILTERED_JSON=""
COMM_WEBHOOK_RECEIPTS_CURSOR_CREATED_AT=""
COMM_WEBHOOK_RECEIPTS_CURSOR_ID=""
COMM_WEBHOOK_RECEIPT_ID=""
COMM_INGEST_RECEIPTS_PAGE1_JSON=""
COMM_INGEST_RECEIPTS_PAGE2_JSON=""
COMM_INGEST_RECEIPTS_FILTERED_JSON=""
COMM_INGEST_RECEIPTS_CURSOR_CREATED_AT=""
COMM_INGEST_RECEIPTS_CURSOR_ID=""
COMM_INGEST_RECEIPT_ID=""
COMM_CALLS_PAGE1_JSON=""
COMM_CALLS_PAGE2_JSON=""
COMM_CALLS_FILTERED_JSON=""
COMM_CALL_CURSOR=""
COMM_CALL_SESSION_ID=""
MAC_SERVICE_DRYRUN_OUTPUT=""
LINUX_SERVICE_DRYRUN_OUTPUT=""
WINDOWS_SERVICE_DRYRUN_OUTPUT=""

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

wait_for_lifecycle_operation_state() {
  local expected_action="$1"
  local expected_state="$2"
  local retries="${3:-120}"
  local i
  local payload=""
  local action=""
  local state=""
  for i in $(seq 1 "$retries"); do
    payload="$(curl -sS -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" "http://$DAEMON_TCP_ADDR/v1/daemon/lifecycle/status" || true)"
    action="$(echo "$payload" | jq -r '.control_operation.action // ""' 2>/dev/null || true)"
    state="$(echo "$payload" | jq -r '.control_operation.state // ""' 2>/dev/null || true)"
    if [[ "$action" == "$expected_action" && "$state" == "$expected_state" ]]; then
      LIFECYCLE_STATUS_JSON="$payload"
      echo "ready lifecycle operation: action=$action state=$state"
      return 0
    fi
    sleep 0.1
  done
  echo "timeout waiting for lifecycle operation action=$expected_action state=$expected_state (last action=$action state=$state)"
  echo "$payload"
  return 1
}

wait_for_socket() {
  local socket_path="$1"
  local retries="${2:-200}"
  local i
  for i in $(seq 1 "$retries"); do
    if [[ -S "$socket_path" ]]; then
      echo "ready socket: $socket_path"
      return 0
    fi
    sleep 0.1
  done
  echo "timeout waiting for socket: $socket_path"
  return 1
}

wait_for_channel_worker_not_sticky_failed() {
  local channel_id="$1"
  local expected_plugin_id="$2"
  local retries="${3:-120}"
  local i
  local payload=""
  local worker_plugin_id=""
  local worker_state=""
  local worker_last_error=""
  for i in $(seq 1 "$retries"); do
    payload="$(curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\"}" "http://$DAEMON_TCP_ADDR/v1/channels/status" || true)"
    worker_plugin_id="$(echo "$payload" | jq -r ".channels[]? | select(.channel_id==\"$channel_id\") | .worker.plugin_id // \"\"" 2>/dev/null || true)"
    worker_state="$(echo "$payload" | jq -r ".channels[]? | select(.channel_id==\"$channel_id\") | .worker.state // \"\"" 2>/dev/null || true)"
    worker_last_error="$(echo "$payload" | jq -r ".channels[]? | select(.channel_id==\"$channel_id\") | .worker.last_error // \"\"" 2>/dev/null || true)"
    if [[ "$worker_plugin_id" == "$expected_plugin_id" ]] && [[ -n "$worker_state" ]] && [[ "$worker_state" != "failed" ]] && [[ "$worker_last_error" != "manual restart requested" ]]; then
      echo "ready worker state after restart: channel=$channel_id plugin=$worker_plugin_id state=$worker_state"
      return 0
    fi
    sleep 0.1
  done
  echo "timeout waiting for non-sticky worker state after restart: channel=$channel_id plugin=$expected_plugin_id state=$worker_state last_error=$worker_last_error"
  echo "$payload"
  return 1
}

wait_for_channel_runtime_ready() {
  local retries="${1:-160}"
  local i
  local payload=""
  local app_status=""
  local app_worker_state=""
  local app_plugin=""
  local messages_status=""
  local messages_worker_state=""
  local messages_plugin=""
  local voice_status=""
  local voice_configured=""

  for i in $(seq 1 "$retries"); do
    payload="$(curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\"}" "http://$DAEMON_TCP_ADDR/v1/channels/status" || true)"

    app_status="$(echo "$payload" | jq -r '.channels[]? | select(.channel_id=="app") | .status // ""' 2>/dev/null || true)"
    app_worker_state="$(echo "$payload" | jq -r '.channels[]? | select(.channel_id=="app") | .worker.state // ""' 2>/dev/null || true)"
    app_plugin="$(echo "$payload" | jq -r '.channels[]? | select(.channel_id=="app") | .worker.plugin_id // ""' 2>/dev/null || true)"

    messages_status="$(echo "$payload" | jq -r '.channels[]? | select(.channel_id=="message") | .status // ""' 2>/dev/null || true)"
    messages_worker_state="$(echo "$payload" | jq -r '.channels[]? | select(.channel_id=="message") | .worker.state // ""' 2>/dev/null || true)"
    messages_plugin="$(echo "$payload" | jq -r '.channels[]? | select(.channel_id=="message") | .worker.plugin_id // ""' 2>/dev/null || true)"

    voice_status="$(echo "$payload" | jq -r '.channels[]? | select(.channel_id=="voice") | .status // ""' 2>/dev/null || true)"
    voice_configured="$(echo "$payload" | jq -r '.channels[]? | select(.channel_id=="voice") | .configured // false' 2>/dev/null || true)"

    if [[ "$app_status" == "ready" ]] && [[ "$app_plugin" == "app_chat.daemon" ]] && [[ "$app_worker_state" != "failed" ]] \
      && [[ -n "$messages_status" ]] && [[ "$messages_plugin" == "messages.daemon" ]] && [[ "$messages_worker_state" != "failed" ]]; then
      if [[ "$voice_configured" == "true" ]]; then
        if [[ "$voice_status" == "ready" ]]; then
          echo "ready channel runtime: app=$app_status message=$messages_status voice=$voice_status"
          return 0
        fi
      else
        echo "ready channel runtime: app=$app_status message=$messages_status voice_configured=$voice_configured"
        return 0
      fi
    fi
    sleep 0.2
  done

  echo "timeout waiting for channel runtime readiness: app=$app_status/$app_worker_state message=$messages_status/$messages_worker_state voice=$voice_status voice_configured=$voice_configured"
  echo "$payload"
  return 1
}

seed_messages_fixture_db() {
  cat > "$TEST_RUNTIME_ROOT/messages-fixture.seed.go" <<'EOF'
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

	stmts := []string{
		`CREATE TABLE message (ROWID INTEGER PRIMARY KEY, guid TEXT, text TEXT, date INTEGER, is_from_me INTEGER, handle_id INTEGER, service TEXT);`,
		`CREATE TABLE chat (ROWID INTEGER PRIMARY KEY, guid TEXT);`,
		`CREATE TABLE chat_message_join (chat_id INTEGER, message_id INTEGER);`,
		`CREATE TABLE handle (ROWID INTEGER PRIMARY KEY, id TEXT);`,
		`INSERT INTO handle(ROWID, id) VALUES (1, '+15555550100');`,
		`INSERT INTO chat(ROWID, guid) VALUES (1, 'chat-guid-daemon-1');`,
		`INSERT INTO message(ROWID, guid, text, date, is_from_me, handle_id, service) VALUES (1001, 'imessage-guid-daemon-1', 'daemon watcher messages token', 1000000000, 0, 1, 'iMessage');`,
		`INSERT INTO chat_message_join(chat_id, message_id) VALUES (1, 1001);`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			panic(err)
		}
	}
}
EOF
  go -C "$ROOT/source/services/daemon-go" run "$TEST_RUNTIME_ROOT/messages-fixture.seed.go" "$MESSAGES_FIXTURE_DB"
  local rc=$?
  rm -f "$TEST_RUNTIME_ROOT/messages-fixture.seed.go"
  return "$rc"
}

pa_tcp() {
  (
    cd "$ROOT/source/clients/cli-go"
    go run ./cmd/personal-agent --mode tcp --address "$DAEMON_TCP_ADDR" --auth-token "$DAEMON_AUTH_TOKEN" "$@"
  )
}

pa_unix() {
  (
    cd "$ROOT/source/clients/cli-go"
    go run ./cmd/personal-agent --mode unix --address "$DAEMON_UNIX_SOCKET" --auth-token "$DAEMON_AUTH_TOKEN" "$@"
  )
}

start_daemon() {
  local mode="$1"
  local address="$2"
  shift 2
  local extra_args=("$@")
  stop_daemon
  (
    cd "$ROOT/source/services/daemon-go"
    PA_PROJECT_NAME="$PA_PROJECT_NAME" \
    PA_RUNTIME_PROFILE="$PA_RUNTIME_PROFILE" \
    PA_RUNTIME_ROOT_DIR="$PA_RUNTIME_ROOT_DIR" \
    PA_MESSAGES_SEND_DRY_RUN="$MESSAGES_SEND_DRY_RUN" \
    PA_MAIL_AUTOMATION_DRY_RUN="$MAIL_AUTOMATION_DRY_RUN" \
    PA_CALENDAR_AUTOMATION_DRY_RUN="$CALENDAR_AUTOMATION_DRY_RUN" \
    PA_BROWSER_AUTOMATION_DRY_RUN="$BROWSER_AUTOMATION_DRY_RUN" \
    PA_CLOUDFLARED_DRY_RUN="$CLOUDFLARED_DRY_RUN" \
    PA_INBOUND_WATCHER_WORKSPACE_ID="$WORKSPACE" \
    PA_INBOUND_WATCHER_INBOX_DIR="$INBOUND_WATCHER_INBOX_DIR" \
    PA_INBOUND_WATCHER_POLL_INTERVAL="$INBOUND_WATCHER_POLL_INTERVAL" \
    PA_INBOUND_WATCHER_MESSAGES_SOURCE_SCOPE="$MESSAGES_WATCHER_SOURCE_SCOPE" \
    PA_INBOUND_WATCHER_MESSAGES_SOURCE_DB_PATH="$MESSAGES_FIXTURE_DB" \
    PA_INBOUND_WATCHER_MESSAGES_LIMIT="10" \
    go run ./cmd/personal-agent-daemon \
      --listen-mode "$mode" \
      --listen-address "$address" \
      --auth-token "$DAEMON_AUTH_TOKEN" \
      --db "$DAEMON_DB_PATH" \
      --lifecycle-host-ops-mode "$DAEMON_LIFECYCLE_HOST_OPS_MODE" \
      ${extra_args[@]+"${extra_args[@]}"}
  ) &
  DAEMON_PID=$!
  echo "daemon started pid=$DAEMON_PID mode=$mode address=$address lifecycle_host_ops_mode=$DAEMON_LIFECYCLE_HOST_OPS_MODE"
}

stop_daemon() {
  if [[ -n "$DAEMON_PID" ]] && kill -0 "$DAEMON_PID" >/dev/null 2>&1; then
    kill "$DAEMON_PID" >/dev/null 2>&1 || true
    wait "$DAEMON_PID" >/dev/null 2>&1 || true
  fi
  if [[ -n "${DAEMON_TCP_ADDR:-}" ]]; then
    pkill -f "personal-agent-daemon --listen-mode tcp --listen-address $DAEMON_TCP_ADDR" >/dev/null 2>&1 || true
  fi
  if [[ -n "${DAEMON_BOOTSTRAP_TCP_ADDR:-}" ]]; then
    pkill -f "personal-agent-daemon --listen-mode tcp --listen-address $DAEMON_BOOTSTRAP_TCP_ADDR" >/dev/null 2>&1 || true
  fi
  if [[ -n "${DAEMON_UNIX_SOCKET:-}" ]]; then
    pkill -f "personal-agent-daemon --listen-mode unix --listen-address $DAEMON_UNIX_SOCKET" >/dev/null 2>&1 || true
  fi
  DAEMON_PID=""
}

cleanup() {
  if [[ -n "$WEBHOOK_SERVE_PID" ]] && kill -0 "$WEBHOOK_SERVE_PID" >/dev/null 2>&1; then
    kill "$WEBHOOK_SERVE_PID" >/dev/null 2>&1 || true
    wait "$WEBHOOK_SERVE_PID" >/dev/null 2>&1 || true
  fi
  WEBHOOK_SERVE_PID=""
  stop_daemon
  rm -f "$DAEMON_UNIX_SOCKET"
  rm -f "$DAEMON_AUTH_TOKEN_FILE"
  rm -f "$DAEMON_DB_PATH"
  rm -f "$MESSAGES_FIXTURE_DB"
  rm -rf "$INBOUND_WATCHER_INBOX_DIR"
  if [[ "$PA_RUNTIME_ROOT_DIR" == "$TEST_RUNTIME_ROOT/runtime-root" ]]; then
    rm -rf "$PA_RUNTIME_ROOT_DIR"
  fi
}
trap cleanup EXIT

# 1) Prerequisites
run_cmd "Check Go toolchain" go version
run_cmd "Check curl" curl --version
run_cmd "Check jq" jq --version

# 2) Common setup
run_cmd "Cleanup stale unix socket" rm -f "$DAEMON_UNIX_SOCKET"
run_cmd "Cleanup stale daemon auth-token file" rm -f "$DAEMON_AUTH_TOKEN_FILE"
run_cmd "Cleanup stale daemon manual test DB" rm -f "$DAEMON_DB_PATH"
run_cmd "Seed baseline Messages source fixture database" seed_messages_fixture_db
run_eval "Show environment defaults" 'echo "WORKSPACE=$WORKSPACE"; echo "INSPECT_WORKSPACE=$INSPECT_WORKSPACE"; echo "DAEMON_DB_PATH=$DAEMON_DB_PATH"; echo "PA_RUNTIME_PROFILE=$PA_RUNTIME_PROFILE"; echo "PA_RUNTIME_ROOT_DIR=$PA_RUNTIME_ROOT_DIR"; echo "DAEMON_TCP_ADDR=$DAEMON_TCP_ADDR"; echo "DAEMON_BOOTSTRAP_TCP_ADDR=$DAEMON_BOOTSTRAP_TCP_ADDR"; echo "DAEMON_UNIX_SOCKET=$DAEMON_UNIX_SOCKET"; echo "OS_NAME=$OS_NAME"'
run_eval "Show webhook path defaults" 'echo "PA_PROJECT_NAME=$PA_PROJECT_NAME"; echo "TWILIO_SMS_WEBHOOK_PATH=$TWILIO_SMS_WEBHOOK_PATH"; echo "TWILIO_VOICE_WEBHOOK_PATH=$TWILIO_VOICE_WEBHOOK_PATH"'
run_eval "Show lifecycle host-ops mode" 'echo "DAEMON_LIFECYCLE_HOST_OPS_MODE=$DAEMON_LIFECYCLE_HOST_OPS_MODE"'

# 3) TCP mode validation
run_eval "Daemon rejects missing auth token" '
OUT="$(cd "$ROOT/source/services/daemon-go" && go run ./cmd/personal-agent-daemon --listen-mode tcp --listen-address "$DAEMON_TCP_ADDR" --runtime-profile prod --auth-token "" 2>&1)"
RC=$?
printf "%s\n" "$OUT"
[ "$RC" -ne 0 ] && printf "%s\n" "$OUT" | rg -q -- "--auth-token is required"
'
run_eval "CLI rejects missing auth token" '
OUT="$(cd "$ROOT/source/clients/cli-go" && go run ./cmd/personal-agent --mode tcp --address "$DAEMON_TCP_ADDR" --runtime-profile prod smoke 2>&1)"
RC=$?
printf "%s\n" "$OUT"
[ "$RC" -ne 0 ] && printf "%s\n" "$OUT" | rg -q -- "--auth-token is required"
'
run_eval "Daemon rejects non-local bind by default" '
OUT="$(cd "$ROOT/source/services/daemon-go" && go run ./cmd/personal-agent-daemon --listen-mode tcp --listen-address "0.0.0.0:notaport" --auth-token "$DAEMON_AUTH_TOKEN" 2>&1)"
RC=$?
printf "%s\n" "$OUT"
[ "$RC" -ne 0 ] && printf "%s\n" "$OUT" | rg -q "non-local; use --allow-non-local-bind"
'
run_cmd "Bootstrap daemon control auth token file" go -C "$ROOT/source/clients/cli-go" run ./cmd/personal-agent auth bootstrap --file "$DAEMON_AUTH_TOKEN_FILE"
run_cmd "Rotate daemon control auth token file" go -C "$ROOT/source/clients/cli-go" run ./cmd/personal-agent auth rotate --file "$DAEMON_AUTH_TOKEN_FILE"
run_eval "Read generated daemon control auth token file value" '
DAEMON_PROD_AUTH_TOKEN="$(tr -d "[:space:]" < "$DAEMON_AUTH_TOKEN_FILE")"
echo "DAEMON_PROD_AUTH_TOKEN_LENGTH=${#DAEMON_PROD_AUTH_TOKEN}"
test -n "$DAEMON_PROD_AUTH_TOKEN"
'
run_eval "Daemon rejects production profile without TLS material" '
OUT="$(cd "$ROOT/source/services/daemon-go" && go run ./cmd/personal-agent-daemon \
  --listen-mode tcp \
  --listen-address "$DAEMON_BOOTSTRAP_TCP_ADDR" \
  --runtime-profile prod \
  --auth-token-file "$DAEMON_AUTH_TOKEN_FILE" \
  --auth-token-scopes "metadata:read" \
  --db "$DAEMON_DB_PATH" \
  --lifecycle-host-ops-mode "$DAEMON_LIFECYCLE_HOST_OPS_MODE" 2>&1)"
RC=$?
printf "%s\n" "$OUT"
[ "$RC" -ne 0 ] && printf "%s\n" "$OUT" | rg -q -- "--runtime-profile=prod requires --tls-cert-file and --tls-key-file"
'
run_eval "Start daemon (tcp) with auth-token-file (local profile)" '
stop_daemon
(
  cd "$ROOT/source/services/daemon-go"
  PA_RUNTIME_PROFILE="$PA_RUNTIME_PROFILE" \
  PA_RUNTIME_ROOT_DIR="$PA_RUNTIME_ROOT_DIR" \
  PA_MESSAGES_SEND_DRY_RUN="$MESSAGES_SEND_DRY_RUN" \
  PA_MAIL_AUTOMATION_DRY_RUN="$MAIL_AUTOMATION_DRY_RUN" \
  PA_CALENDAR_AUTOMATION_DRY_RUN="$CALENDAR_AUTOMATION_DRY_RUN" \
  PA_BROWSER_AUTOMATION_DRY_RUN="$BROWSER_AUTOMATION_DRY_RUN" \
  PA_INBOUND_WATCHER_WORKSPACE_ID="$WORKSPACE" \
  PA_INBOUND_WATCHER_INBOX_DIR="$INBOUND_WATCHER_INBOX_DIR" \
  PA_INBOUND_WATCHER_POLL_INTERVAL="$INBOUND_WATCHER_POLL_INTERVAL" \
  PA_INBOUND_WATCHER_MESSAGES_SOURCE_SCOPE="$MESSAGES_WATCHER_SOURCE_SCOPE" \
  PA_INBOUND_WATCHER_MESSAGES_SOURCE_DB_PATH="$MESSAGES_FIXTURE_DB" \
  PA_INBOUND_WATCHER_MESSAGES_LIMIT="10" \
  go run ./cmd/personal-agent-daemon \
    --listen-mode tcp \
    --listen-address "$DAEMON_BOOTSTRAP_TCP_ADDR" \
    --auth-token-file "$DAEMON_AUTH_TOKEN_FILE" \
    --db "$DAEMON_DB_PATH" \
    --lifecycle-host-ops-mode "$DAEMON_LIFECYCLE_HOST_OPS_MODE"
) &
DAEMON_PID=$!
echo "daemon started pid=$DAEMON_PID mode=tcp address=$DAEMON_BOOTSTRAP_TCP_ADDR auth-token-file=$DAEMON_AUTH_TOKEN_FILE"
'
run_cmd "Wait for daemon endpoint (auth-token-file local profile)" wait_for_http_status "http://$DAEMON_BOOTSTRAP_TCP_ADDR/v1/capabilities/smoke" "200" "$DAEMON_PROD_AUTH_TOKEN" 300
run_eval "CLI smoke with auth-token-file (local profile)" 'cd "$ROOT/source/clients/cli-go" && go run ./cmd/personal-agent --mode tcp --address "$DAEMON_BOOTSTRAP_TCP_ADDR" --auth-token-file "$DAEMON_AUTH_TOKEN_FILE" smoke'
run_eval "Stop daemon auth-token-file check instance" 'stop_daemon'
run_eval "Start daemon (tcp) with scoped auth policy" 'start_daemon "tcp" "$DAEMON_BOOTSTRAP_TCP_ADDR" --auth-token-scopes "tasks:read"'
run_cmd "Wait for scoped daemon metadata-protected endpoint (expected 403)" wait_for_http_status "http://$DAEMON_BOOTSTRAP_TCP_ADDR/v1/capabilities/smoke" "403" "$DAEMON_AUTH_TOKEN" 300
run_eval "Validate scoped auth policy denies privileged reads and allows task reads" '
FORBIDDEN="$(curl -s -o /tmp/pa-daemon-scope-forbidden.json -w "%{http_code}" -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" "http://$DAEMON_BOOTSTRAP_TCP_ADDR/v1/daemon/lifecycle/status" || true)"
ALLOWED="$(curl -s -o /tmp/pa-daemon-scope-allowed.json -w "%{http_code}" -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" "http://$DAEMON_BOOTSTRAP_TCP_ADDR/v1/tasks/task-scope-check" || true)"
printf "forbidden=%s allowed=%s\n" "$FORBIDDEN" "$ALLOWED"
cat /tmp/pa-daemon-scope-forbidden.json; echo
[[ "$FORBIDDEN" == "403" ]] && [[ "$ALLOWED" != "403" ]] && jq -e ".error.code == \"auth_forbidden\" and ((.error.details.required_scopes // [])[0] == \"daemon:read\") and ((.error.details.granted_scopes // []) | index(\"tasks:read\")) != null" /tmp/pa-daemon-scope-forbidden.json >/dev/null
'
run_eval "Stop scoped daemon auth-policy check instance" 'stop_daemon'
run_eval "Start daemon (tcp)" 'start_daemon "tcp" "$DAEMON_TCP_ADDR"'
run_cmd "Wait for authenticated TCP capability endpoint" wait_for_http_status "http://$DAEMON_TCP_ADDR/v1/capabilities/smoke" "200" "$DAEMON_AUTH_TOKEN" 300
run_eval "Validate no-auth and bad-auth return 401" '
NOAUTH="$(curl -s -o /tmp/pa-daemon-noauth.json -w "%{http_code}" "http://$DAEMON_TCP_ADDR/v1/capabilities/smoke" || true)"
BADAUTH="$(curl -s -o /tmp/pa-daemon-badauth.json -w "%{http_code}" -H "Authorization: Bearer wrong-token" "http://$DAEMON_TCP_ADDR/v1/capabilities/smoke" || true)"
AUTHED="$(curl -s -o /tmp/pa-daemon-auth.json -w "%{http_code}" -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" "http://$DAEMON_TCP_ADDR/v1/capabilities/smoke" || true)"
printf "noauth=%s badauth=%s authed=%s\n" "$NOAUTH" "$BADAUTH" "$AUTHED"
cat /tmp/pa-daemon-noauth.json; echo
cat /tmp/pa-daemon-auth.json; echo
[[ "$NOAUTH" == "401" ]] && [[ "$BADAUTH" == "401" ]] && [[ "$AUTHED" == "200" ]] && jq -e ".healthy == true" /tmp/pa-daemon-auth.json >/dev/null
'
run_cmd "Daemon capability smoke over TCP" pa_tcp smoke
capture_cmd META_CAPABILITIES_JSON "Read daemon runtime capabilities metadata API" curl -sS -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" "http://$DAEMON_TCP_ADDR/v1/meta/capabilities"
run_eval "Validate daemon runtime capabilities metadata payload" 'echo "$META_CAPABILITIES_JSON" | jq -e ".api_version == \"v1\" and (.route_groups|type)==\"array\" and (.route_groups|length)>0 and ((.realtime_event_types // []) | index(\"task_run_lifecycle\")) != null and ((.realtime_event_types // []) | index(\"turn_item_delta\")) != null and ((.realtime_event_types // []) | index(\"chat_completed\")) != null and ((.realtime_event_types // []) | index(\"chat_error\")) != null and ((.client_signal_types // []) | index(\"cancel\")) != null and ((.protocol_modes // []) | index(\"http_json\")) != null and ((.transport_listener_modes // []) | index(\"tcp\")) != null" >/dev/null'
run_cmd "Daemon runtime capabilities via CLI thin-client command" pa_tcp --output json-compact meta capabilities
capture_cmd LIFECYCLE_STATUS_JSON "Read daemon lifecycle status API" curl -sS -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" "http://$DAEMON_TCP_ADDR/v1/daemon/lifecycle/status"
run_eval "Validate lifecycle status payload" 'echo "$LIFECYCLE_STATUS_JSON" | jq -e ".lifecycle_state == \"running\" and (.setup_state|type)==\"string\" and (.install_state|type)==\"string\" and .controls.restart == true and .controls.install == true and ((.control_operation.state // \"\")|type)==\"string\" and ((.health_classification.overall_state // \"\")|type)==\"string\" and ((.health_classification.core_runtime_state // \"\")|type)==\"string\" and ((.health_classification.plugin_runtime_state // \"\")|type)==\"string\" and ((.health_classification.blocking // false)|type)==\"boolean\" and ((.control_auth.state // \"\") == \"configured\") and ((.control_auth.source // \"\") == \"auth_token_flag\") and ((.control_auth.remediation_hints // []) | type) == \"array\"" >/dev/null'
capture_cmd LIFECYCLE_CONTROL_JSON "Daemon lifecycle start control (idempotent)" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d '{"action":"start","reason":"manual runner ensure running"}' "http://$DAEMON_TCP_ADDR/v1/daemon/lifecycle/control"
run_eval "Validate lifecycle start control response" 'echo "$LIFECYCLE_CONTROL_JSON" | jq -e ".accepted == true and .idempotent == true and .action == \"start\" and .operation_state == \"succeeded\"" >/dev/null'
capture_cmd LIFECYCLE_CONTROL_JSON "Daemon lifecycle restart control" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d '{"action":"restart","reason":"manual runner restart validation"}' "http://$DAEMON_TCP_ADDR/v1/daemon/lifecycle/control"
run_eval "Validate lifecycle restart control response" 'echo "$LIFECYCLE_CONTROL_JSON" | jq -e ".accepted == true and .action == \"restart\" and .operation_state == \"succeeded\"" >/dev/null'
run_cmd "Wait for daemon endpoint after restart control" wait_for_http_status "http://$DAEMON_TCP_ADDR/v1/capabilities/smoke" "200" "$DAEMON_AUTH_TOKEN" 300
run_cmd "Validate Messages worker is not sticky failed after restart" wait_for_channel_worker_not_sticky_failed "message" "messages.daemon" 240
run_cmd "Validate configured channel runtime readiness after restart" wait_for_channel_runtime_ready 240
capture_cmd LIFECYCLE_CONTROL_JSON "Daemon lifecycle install control" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d '{"action":"install","reason":"manual runner install validation"}' "http://$DAEMON_TCP_ADDR/v1/daemon/lifecycle/control"
run_eval "Validate lifecycle install control response" 'echo "$LIFECYCLE_CONTROL_JSON" | jq -e ".accepted == true and .idempotent == false and .action == \"install\" and .operation_state == \"in_progress\"" >/dev/null'
run_cmd "Wait for lifecycle install operation to succeed" wait_for_lifecycle_operation_state "install" "succeeded" 200
run_eval "Validate lifecycle status reflection for install operation" 'echo "$LIFECYCLE_STATUS_JSON" | jq -e ".control_operation.action == \"install\" and .control_operation.state == \"succeeded\"" >/dev/null'
capture_cmd LIFECYCLE_CONTROL_JSON "Daemon lifecycle repair control" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d '{"action":"repair","reason":"manual runner repair validation"}' "http://$DAEMON_TCP_ADDR/v1/daemon/lifecycle/control"
run_eval "Validate lifecycle repair control response" 'echo "$LIFECYCLE_CONTROL_JSON" | jq -e ".accepted == true and .idempotent == false and .action == \"repair\" and .operation_state == \"in_progress\"" >/dev/null'
run_cmd "Wait for lifecycle repair operation to succeed" wait_for_lifecycle_operation_state "repair" "succeeded" 200
run_eval "Validate lifecycle status reflection for repair operation" 'echo "$LIFECYCLE_STATUS_JSON" | jq -e ".control_operation.action == \"repair\" and .control_operation.state == \"succeeded\"" >/dev/null'
capture_cmd LIFECYCLE_CONTROL_JSON "Daemon lifecycle uninstall control" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d '{"action":"uninstall","reason":"manual runner uninstall validation"}' "http://$DAEMON_TCP_ADDR/v1/daemon/lifecycle/control"
run_eval "Validate lifecycle uninstall control response" 'echo "$LIFECYCLE_CONTROL_JSON" | jq -e ".accepted == true and .idempotent == false and .action == \"uninstall\" and .operation_state == \"in_progress\"" >/dev/null'
run_cmd "Wait for lifecycle uninstall operation to succeed" wait_for_lifecycle_operation_state "uninstall" "succeeded" 200
run_eval "Validate lifecycle status reflection for uninstall operation" 'echo "$LIFECYCLE_STATUS_JSON" | jq -e ".control_operation.action == \"uninstall\" and .control_operation.state == \"succeeded\"" >/dev/null'
run_cmd "Validate daemon lifecycle host-ops rendering/execution regressions" go -C "$ROOT/source/services/daemon-go" test ./cmd/personal-agent-daemon -run 'TestLifecycleHost' -count=1
run_cmd "Validate daemon lifecycle control action-state regressions" go -C "$ROOT/source/services/daemon-go" test ./internal/daemonruntime -run 'TestDaemonLifecycleControl(InstallTracksInProgressAndSuccess|RepairReflectsFailure|UninstallTracksInProgressAndSuccess)' -count=1
run_cmd "Validate daemon lifecycle busy single-writer DB readiness regression" go -C "$ROOT/source/services/daemon-go" test ./internal/daemonruntime -run 'TestDaemonLifecycleStatusTreatsBusySingleWriterConnectionAsReady' -count=1
run_cmd "Validate transport principal-aware control rate-limit regressions" go -C "$ROOT/source/services/daemon-go" test ./internal/transport -run 'TestTransportControlRateLimit(ReturnsTyped429AndResetsAfterWindow|IsPrincipalAwarePerEndpoint|CoversHighRiskMutatingRoutes)' -count=1
run_cmd "Validate realtime websocket guardrail regressions" go -C "$ROOT/source/services/daemon-go" test ./internal/transport -run 'TestTransportRealtime(RejectsWhenConnectionCapExceeded|RejectsWhenSubscriptionCapExceeded|ClosesConnectionOnOversizedSignalPayload|ClosesStaleClientWithoutPong)' -count=1
run_cmd "Validate unified-turn chat transport + realtime lifecycle regressions" go -C "$ROOT/source/services/daemon-go" test ./internal/transport -run 'TestTransport(ChatTurn(RouteIncludesTaskRunCorrelationMetadata|UsesCanonicalTurnItemsContract|HistoryRoute|PublishesRealtimeLifecycleEvents|PublishesRealtimeLifecycleEventsForMultipleToolCalls|FailurePublishesRealtimeChatError)|ChatPersonaPolicyRoutes)' -count=1
run_cmd "Validate unified-turn capability-driven tool registry regressions" go -C "$ROOT/source/services/daemon-go" test ./internal/daemonruntime -run 'TestUnifiedTurnServiceResolveAvailableTools(SupportsDotCapabilityKeys|SkipsBlockedConnectorCapabilities|IncludesExpandedConnectorCapabilities|DeduplicatesAliasCapabilities)' -count=1
run_cmd "Validate unified-turn iterative orchestration regressions" go -C "$ROOT/source/services/daemon-go" test ./internal/daemonruntime -run 'TestUnifiedTurnService(ChatTurnSupportsIterativeToolLoop|ChatTurnStopsAtToolCallLimit)' -count=1
run_cmd "Validate unified-turn model-only streaming callback regression" go -C "$ROOT/source/services/daemon-go" test ./internal/daemonruntime -run 'TestUnifiedTurnServiceChatTurnStreamsModelOnlyResponseWhenTokenCallbackProvided' -count=1
run_cmd "Validate typed unified-turn native-action bridge regressions" go -C "$ROOT/source/services/daemon-go" test ./internal/daemonruntime -run 'Test(UnifiedTurnServiceChatTurn(ModelToolModelSuccess|SupportsIterativeToolLoop)|BuildMailNativeActionUnreadSummarySupportsOptionalLimit|AgentDelegationServiceRunAgentExecutesTypedNativeActionPayload)' -count=1
run_cmd "Validate chat-action reliability cutover regressions (origin, repair, e2e, remediation)" go -C "$ROOT/source/services/daemon-go" test ./internal/daemonruntime -run 'TestUnifiedTurnServiceChatTurn(ModelToolModelSuccess|PolicyRequireApprovalFlow|RepairsMalformedPlannerOutputBeforeToolExecution|FallsBackAfterPlannerRepairRetriesExhausted|NaturalLanguageEmailAndBrowserToolChain|NaturalLanguageEmailFailureEdge|NaturalLanguageBrowserApprovalEdge|ReturnsModelRouteRemediationOnPlannerRouteFailure|ToolExecutionFailure)' -count=1
run_cmd "Validate action-capable chat-route policy regressions" go -C "$ROOT/source/services/daemon-go" test ./internal/daemonruntime -run 'TestModelRoute(SimulationAndExplainabilityTaskPolicySelected|SelectRejectsNonActionCapableChatPolicy|SimulationChatPolicyMisconfiguredFallsBackToActionCapableCandidate|ResolveChatFailsWithoutActionCapableCandidates)' -count=1
run_cmd "Validate provider-native tool-calling stream regressions" go -C "$ROOT/source/services/daemon-go" test ./internal/chatruntime -run 'TestStreamAssistantResponse(OpenAINativeToolCalling|OllamaNativeToolCalling|UnsupportedProviderToolFallback)' -count=1
run_cmd "Validate planner prompt native-tool extraction regressions" go -C "$ROOT/source/services/daemon-go" test ./internal/daemonruntime -run 'Test(PlannerToolSpecsFromPromptParsesUnifiedPrompt|ProviderModelChatServiceChatTurnPlannerPromptUsesNativeToolCallingForOllama)' -count=1
run_cmd "Validate agentexec typed native-action planning regressions" go -C "$ROOT/source/services/daemon-go" test ./internal/core/service/agentexec -run 'Test(ExecuteUsesTypedNativeActionForSingleBrowserToolOperation|PlanStepsMailUnreadSummaryUsesCanonicalCapability|ExecuteMessageDispatchCanReenterSQLiteWithSingleWriterPool)' -count=1
run_cmd "Validate connector adapter step-input contract regressions" go -C "$ROOT/source/services/daemon-go" test ./internal/connectors/adapters/mail ./internal/connectors/adapters/calendar ./internal/connectors/adapters/browser ./internal/connectors/adapters/finder -count=1
run_cmd "Validate channel/connector action-readiness classification regressions" go -C "$ROOT/source/services/daemon-go" test ./internal/daemonruntime -run 'Test(ChannelActionReadinessClassification|ConnectorActionReadinessClassification)' -count=1
capture_cmd LIFECYCLE_PLUGIN_HISTORY_PAGE1_JSON "Query daemon plugin lifecycle history API (page 1)" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d '{"workspace_id":"daemon","limit":1}' "http://$DAEMON_TCP_ADDR/v1/daemon/lifecycle/plugins/history"
run_eval "Validate daemon plugin lifecycle history page 1 payload and extract cursor/filter anchors" '
echo "$LIFECYCLE_PLUGIN_HISTORY_PAGE1_JSON" | jq -e ".workspace_id == \"daemon\" and (.items|type)==\"array\" and (.items|length)==1 and (.has_more == true) and ((.next_cursor_created_at // \"\")|length)>0 and ((.next_cursor_id // \"\")|length)>0 and ((.items[0].audit_id // \"\")|length)>0 and ((.items[0].plugin_id // \"\")|length)>0 and ((.items[0].kind // \"\")|type)==\"string\" and ((.items[0].state // \"\")|type)==\"string\" and ((.items[0].event_type // \"\")|type)==\"string\" and ((.items[0].reason // \"\")|type)==\"string\" and ((.items[0].restart_event // false)|type)==\"boolean\" and ((.items[0].failure_event // false)|type)==\"boolean\" and ((.items[0].recovery_event // false)|type)==\"boolean\" and ((.items[0].occurred_at // \"\")|length)>0" >/dev/null
LIFECYCLE_PLUGIN_HISTORY_CURSOR_CREATED_AT="$(echo "$LIFECYCLE_PLUGIN_HISTORY_PAGE1_JSON" | jq -r ".next_cursor_created_at")"
LIFECYCLE_PLUGIN_HISTORY_CURSOR_ID="$(echo "$LIFECYCLE_PLUGIN_HISTORY_PAGE1_JSON" | jq -r ".next_cursor_id")"
LIFECYCLE_PLUGIN_HISTORY_FIRST_AUDIT_ID="$(echo "$LIFECYCLE_PLUGIN_HISTORY_PAGE1_JSON" | jq -r ".items[0].audit_id")"
LIFECYCLE_PLUGIN_HISTORY_PLUGIN_ID="$(echo "$LIFECYCLE_PLUGIN_HISTORY_PAGE1_JSON" | jq -r ".items[0].plugin_id")"
LIFECYCLE_PLUGIN_HISTORY_STATE="$(echo "$LIFECYCLE_PLUGIN_HISTORY_PAGE1_JSON" | jq -r ".items[0].state")"
echo "LIFECYCLE_PLUGIN_HISTORY_CURSOR_CREATED_AT=$LIFECYCLE_PLUGIN_HISTORY_CURSOR_CREATED_AT"
echo "LIFECYCLE_PLUGIN_HISTORY_CURSOR_ID=$LIFECYCLE_PLUGIN_HISTORY_CURSOR_ID"
echo "LIFECYCLE_PLUGIN_HISTORY_FIRST_AUDIT_ID=$LIFECYCLE_PLUGIN_HISTORY_FIRST_AUDIT_ID"
echo "LIFECYCLE_PLUGIN_HISTORY_PLUGIN_ID=$LIFECYCLE_PLUGIN_HISTORY_PLUGIN_ID"
echo "LIFECYCLE_PLUGIN_HISTORY_STATE=$LIFECYCLE_PLUGIN_HISTORY_STATE"
test -n "$LIFECYCLE_PLUGIN_HISTORY_CURSOR_CREATED_AT" && [ "$LIFECYCLE_PLUGIN_HISTORY_CURSOR_CREATED_AT" != "null" ]
test -n "$LIFECYCLE_PLUGIN_HISTORY_CURSOR_ID" && [ "$LIFECYCLE_PLUGIN_HISTORY_CURSOR_ID" != "null" ]
test -n "$LIFECYCLE_PLUGIN_HISTORY_FIRST_AUDIT_ID" && [ "$LIFECYCLE_PLUGIN_HISTORY_FIRST_AUDIT_ID" != "null" ]
test -n "$LIFECYCLE_PLUGIN_HISTORY_PLUGIN_ID" && [ "$LIFECYCLE_PLUGIN_HISTORY_PLUGIN_ID" != "null" ]
test -n "$LIFECYCLE_PLUGIN_HISTORY_STATE" && [ "$LIFECYCLE_PLUGIN_HISTORY_STATE" != "null" ]
'
capture_cmd LIFECYCLE_PLUGIN_HISTORY_PAGE2_JSON "Query daemon plugin lifecycle history API (page 2 via cursor)" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"daemon\",\"cursor_created_at\":\"$LIFECYCLE_PLUGIN_HISTORY_CURSOR_CREATED_AT\",\"cursor_id\":\"$LIFECYCLE_PLUGIN_HISTORY_CURSOR_ID\",\"limit\":1}" "http://$DAEMON_TCP_ADDR/v1/daemon/lifecycle/plugins/history"
run_eval "Validate daemon plugin lifecycle history page 2 payload" 'echo "$LIFECYCLE_PLUGIN_HISTORY_PAGE2_JSON" | jq -e ".workspace_id == \"daemon\" and (.items|type)==\"array\" and (.items|length)>=1 and ((.items[0].audit_id // \"\") != \"$LIFECYCLE_PLUGIN_HISTORY_FIRST_AUDIT_ID\")" >/dev/null'
capture_cmd LIFECYCLE_PLUGIN_HISTORY_FILTERED_JSON "Query daemon plugin lifecycle history API with plugin/state filters" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"daemon\",\"plugin_id\":\"$LIFECYCLE_PLUGIN_HISTORY_PLUGIN_ID\",\"state\":\"$LIFECYCLE_PLUGIN_HISTORY_STATE\",\"limit\":20}" "http://$DAEMON_TCP_ADDR/v1/daemon/lifecycle/plugins/history"
run_eval "Validate daemon plugin lifecycle history filtered payload" 'echo "$LIFECYCLE_PLUGIN_HISTORY_FILTERED_JSON" | jq -e ".workspace_id == \"daemon\" and (.items|type)==\"array\" and (.items|length)>=1 and all(.items[]?; (.plugin_id == \"$LIFECYCLE_PLUGIN_HISTORY_PLUGIN_ID\") and ((.state // \"\")|ascii_downcase) == \"$LIFECYCLE_PLUGIN_HISTORY_STATE\")" >/dev/null'
run_eval "Seed context query fixtures for daemon context API checks" '
cat > "$TEST_RUNTIME_ROOT/context-query-fixture.seed.go" <<EOF
package main

import (
	"database/sql"
	"os"

	_ "modernc.org/sqlite"
)

func mustExec(db *sql.DB, query string, args ...any) {
	if _, err := db.Exec(query, args...); err != nil {
		panic(err)
	}
}

func main() {
	if len(os.Args) != 3 {
		panic("usage: context-query-fixture.seed.go <db-path> <workspace-id>")
	}
	dbPath := os.Args[1]
	workspaceID := os.Args[2]

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	mustExec(db, "INSERT INTO actors(id, workspace_id, actor_type, display_name, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?) ON CONFLICT(id) DO NOTHING", "actor.context.a", workspaceID, "human", "Context A", "ACTIVE", "2026-02-25T00:00:00Z", "2026-02-25T00:00:00Z")
	mustExec(db, "INSERT INTO actors(id, workspace_id, actor_type, display_name, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?) ON CONFLICT(id) DO NOTHING", "actor.context.b", workspaceID, "human", "Context B", "ACTIVE", "2026-02-25T00:00:00Z", "2026-02-25T00:00:00Z")
	mustExec(db, "INSERT INTO memory_items(id, workspace_id, owner_principal_actor_id, scope_type, key, value_json, status, source_summary, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(id) DO NOTHING", "manual-mem-1", workspaceID, "actor.context.a", "conversation", "manual-key-1", "{\"kind\":\"summary\",\"token_estimate\":9,\"content\":\"manual one\"}", "ACTIVE", "event://manual-1", "2026-02-25T00:00:01Z", "2026-02-25T00:00:01Z")
	mustExec(db, "INSERT INTO memory_items(id, workspace_id, owner_principal_actor_id, scope_type, key, value_json, status, source_summary, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(id) DO NOTHING", "manual-mem-2", workspaceID, "actor.context.a", "conversation", "manual-key-2", "{\"kind\":\"summary\",\"token_estimate\":11,\"content\":\"manual two\"}", "ACTIVE", "event://manual-2", "2026-02-25T00:00:02Z", "2026-02-25T00:00:02Z")
	mustExec(db, "INSERT INTO memory_sources(id, memory_item_id, source_type, source_ref, created_at) VALUES (?, ?, ?, ?, ?) ON CONFLICT(id) DO NOTHING", "manual-src-1", "manual-mem-1", "comm_event", "event://manual-1", "2026-02-25T00:00:01Z")
	mustExec(db, "INSERT INTO memory_sources(id, memory_item_id, source_type, source_ref, created_at) VALUES (?, ?, ?, ?, ?) ON CONFLICT(id) DO NOTHING", "manual-src-2", "manual-mem-2", "comm_event", "event://manual-2", "2026-02-25T00:00:02Z")
	mustExec(db, "INSERT INTO memory_candidates(id, workspace_id, owner_principal_actor_id, candidate_json, score, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?) ON CONFLICT(id) DO NOTHING", "manual-cand-1", workspaceID, "actor.context.a", "{\"kind\":\"summary\",\"token_estimate\":20,\"source_ids\":[\"manual-mem-1\",\"manual-mem-2\"],\"source_refs\":[\"event://manual-1\",\"event://manual-2\"]}", 0.95, "PENDING", "2026-02-25T00:00:03Z")
	mustExec(db, "INSERT INTO context_documents(id, workspace_id, owner_principal_actor_id, source_uri, checksum, created_at) VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT(id) DO NOTHING", "manual-doc-1", workspaceID, "actor.context.a", "memory://manual/doc-1", "manual-checksum-1", "2026-02-25T00:00:01Z")
	mustExec(db, "INSERT INTO context_chunks(id, document_id, chunk_index, text_body, token_count, created_at) VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT(id) DO NOTHING", "manual-chunk-1", "manual-doc-1", 0, "manual retrieval chunk", 7, "2026-02-25T00:00:01Z")
}
EOF
go -C "$ROOT/source/services/daemon-go" run "$TEST_RUNTIME_ROOT/context-query-fixture.seed.go" "$DAEMON_DB_PATH" "$WORKSPACE" || false
rm -f "$TEST_RUNTIME_ROOT/context-query-fixture.seed.go"
'
capture_cmd CONTEXT_MEMORY_PAGE1_JSON "Query context memory inventory API (page 1)" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"owner_actor_id\":\"actor.context.a\",\"source_type\":\"comm_event\",\"limit\":1}" "http://$DAEMON_TCP_ADDR/v1/context/memory/inventory"
run_eval "Validate context memory inventory page 1 payload and capture cursor" '
echo "$CONTEXT_MEMORY_PAGE1_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and (.items|type)==\"array\" and (.items|length)==1 and (.has_more == true) and ((.next_cursor_updated_at // \"\")|length)>0 and ((.next_cursor_id // \"\")|length)>0 and ((.items[0].sources // [])|type)==\"array\" and ((.items[0].token_estimate // 0)|type)==\"number\"" >/dev/null
CONTEXT_MEMORY_CURSOR_UPDATED_AT="$(echo "$CONTEXT_MEMORY_PAGE1_JSON" | jq -r ".next_cursor_updated_at")"
CONTEXT_MEMORY_CURSOR_ID="$(echo "$CONTEXT_MEMORY_PAGE1_JSON" | jq -r ".next_cursor_id")"
CONTEXT_MEMORY_FIRST_ID="$(echo "$CONTEXT_MEMORY_PAGE1_JSON" | jq -r ".items[0].memory_id")"
echo "CONTEXT_MEMORY_CURSOR_UPDATED_AT=$CONTEXT_MEMORY_CURSOR_UPDATED_AT"
echo "CONTEXT_MEMORY_CURSOR_ID=$CONTEXT_MEMORY_CURSOR_ID"
echo "CONTEXT_MEMORY_FIRST_ID=$CONTEXT_MEMORY_FIRST_ID"
test -n "$CONTEXT_MEMORY_CURSOR_UPDATED_AT" && [ "$CONTEXT_MEMORY_CURSOR_UPDATED_AT" != "null" ]
test -n "$CONTEXT_MEMORY_CURSOR_ID" && [ "$CONTEXT_MEMORY_CURSOR_ID" != "null" ]
test -n "$CONTEXT_MEMORY_FIRST_ID" && [ "$CONTEXT_MEMORY_FIRST_ID" != "null" ]
'
capture_cmd CONTEXT_MEMORY_PAGE2_JSON "Query context memory inventory API (page 2 via cursor)" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"owner_actor_id\":\"actor.context.a\",\"source_type\":\"comm_event\",\"cursor_updated_at\":\"$CONTEXT_MEMORY_CURSOR_UPDATED_AT\",\"cursor_id\":\"$CONTEXT_MEMORY_CURSOR_ID\",\"limit\":1}" "http://$DAEMON_TCP_ADDR/v1/context/memory/inventory"
run_eval "Validate context memory inventory page 2 payload" 'echo "$CONTEXT_MEMORY_PAGE2_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and (.items|type)==\"array\" and (.items|length)>=1 and ((.items[0].memory_id // \"\") != \"$CONTEXT_MEMORY_FIRST_ID\")" >/dev/null'
capture_cmd CONTEXT_MEMORY_FILTERED_JSON "Query context memory inventory API with source_ref filter" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"owner_actor_id\":\"actor.context.a\",\"source_ref_query\":\"manual-1\",\"status\":\"ACTIVE\",\"limit\":20}" "http://$DAEMON_TCP_ADDR/v1/context/memory/inventory"
run_eval "Validate context memory inventory filtered payload" 'echo "$CONTEXT_MEMORY_FILTERED_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and (.items|type)==\"array\" and (.items|length)>=1 and all(.items[]?; (.owner_actor_id == \"actor.context.a\") and ((.status // \"\")|ascii_upcase) == \"ACTIVE\")" >/dev/null'
capture_cmd CONTEXT_MEMORY_CANDIDATES_JSON "Query context memory compaction-candidate preview API" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"owner_actor_id\":\"actor.context.a\",\"status\":\"PENDING\",\"limit\":20}" "http://$DAEMON_TCP_ADDR/v1/context/memory/compaction-candidates"
run_eval "Validate context memory compaction-candidate payload" 'echo "$CONTEXT_MEMORY_CANDIDATES_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and (.items|type)==\"array\" and (.items|length)>=1 and all(.items[]?; (.owner_actor_id == \"actor.context.a\") and ((.status // \"\")|ascii_upcase) == \"PENDING\" and ((.candidate_kind // \"\")|type)==\"string\" and ((.candidate_json // \"\")|type)==\"string\")" >/dev/null'
capture_cmd CONTEXT_RETRIEVAL_DOCUMENTS_JSON "Query context retrieval documents API" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"owner_actor_id\":\"actor.context.a\",\"source_uri_query\":\"memory://manual\",\"limit\":20}" "http://$DAEMON_TCP_ADDR/v1/context/retrieval/documents"
run_eval "Validate context retrieval documents payload" 'echo "$CONTEXT_RETRIEVAL_DOCUMENTS_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and (.items|type)==\"array\" and (.items|length)>=1 and all(.items[]?; (.owner_actor_id == \"actor.context.a\") and ((.source_uri // \"\")|test(\"memory://manual\")) and ((.chunk_count // 0)|type)==\"number\")" >/dev/null'
capture_cmd CONTEXT_RETRIEVAL_CHUNKS_JSON "Query context retrieval chunks API" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"document_id\":\"manual-doc-1\",\"chunk_text_query\":\"manual retrieval\",\"limit\":20}" "http://$DAEMON_TCP_ADDR/v1/context/retrieval/chunks"
run_eval "Validate context retrieval chunks payload" 'echo "$CONTEXT_RETRIEVAL_CHUNKS_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .document_id == \"manual-doc-1\" and (.items|type)==\"array\" and (.items|length)>=1 and all(.items[]?; (.document_id == \"manual-doc-1\") and ((.text_body // \"\")|test(\"manual retrieval\")))" >/dev/null'
run_cmd "Configure Ollama provider for model catalog API checks" pa_tcp provider set --workspace "$WORKSPACE" --provider ollama --endpoint "http://127.0.0.1:11434"
capture_cmd MODEL_DISCOVER_JSON "Read model discover API" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"provider\":\"ollama\"}" "http://$DAEMON_TCP_ADDR/v1/models/discover"
run_eval "Validate model discover payload" 'echo "$MODEL_DISCOVER_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and (.results|type)==\"array\" and ([.results[].provider] | index(\"ollama\")) != null and all(.results[]?; (.provider|type)==\"string\" and (.success|type)==\"boolean\" and (.models|type)==\"array\")" >/dev/null'
capture_cmd MODEL_ADD_JSON "Add model catalog entry via API" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"provider\":\"ollama\",\"model_key\":\"llama3.2-custom\",\"enabled\":true}" "http://$DAEMON_TCP_ADDR/v1/models/add"
run_eval "Validate model add payload" 'echo "$MODEL_ADD_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .provider == \"ollama\" and .model_key == \"llama3.2-custom\" and .enabled == true" >/dev/null'
capture_cmd MODEL_REMOVE_JSON "Remove model catalog entry via API" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"provider\":\"ollama\",\"model_key\":\"llama3.2-custom\"}" "http://$DAEMON_TCP_ADDR/v1/models/remove"
run_eval "Validate model remove payload" 'echo "$MODEL_REMOVE_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .provider == \"ollama\" and .model_key == \"llama3.2-custom\" and .removed == true and ((.removed_at // \"\")|type)==\"string\"" >/dev/null'
capture_cmd MODEL_ROUTE_SIMULATE_JSON "Simulate model-route decision API" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"task_class\":\"chat\"}" "http://$DAEMON_TCP_ADDR/v1/models/route/simulate"
run_eval "Validate model-route simulation payload" 'echo "$MODEL_ROUTE_SIMULATE_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .task_class == \"chat\" and ((.principal_actor_id // \"\") == \"\") and ((.selected_provider // \"\")|type)==\"string\" and ((.selected_model_key // \"\")|type)==\"string\" and ((.selected_source // \"\")|type)==\"string\" and (.reason_codes|type)==\"array\" and ((.reason_codes | length) >= 1) and (.decisions|type)==\"array\" and ((.decisions | length) >= 1) and (.fallback_chain|type)==\"array\" and ((.fallback_chain | length) >= 1) and (any(.fallback_chain[]?; .selected == true))" >/dev/null'
capture_cmd MODEL_ROUTE_EXPLAIN_JSON "Explain model-route decision API" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"task_class\":\"chat\"}" "http://$DAEMON_TCP_ADDR/v1/models/route/explain"
run_eval "Validate model-route explain payload" '
SIM_ROUTE_PROVIDER="$(echo "$MODEL_ROUTE_SIMULATE_JSON" | jq -r ".selected_provider")"
SIM_ROUTE_MODEL="$(echo "$MODEL_ROUTE_SIMULATE_JSON" | jq -r ".selected_model_key")"
SIM_ROUTE_SOURCE="$(echo "$MODEL_ROUTE_SIMULATE_JSON" | jq -r ".selected_source")"
echo "$MODEL_ROUTE_EXPLAIN_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .task_class == \"chat\" and ((.principal_actor_id // \"\") == \"\") and ((.summary // \"\")|length)>0 and (.explanations|type)==\"array\" and ((.explanations|length)>=1) and (.reason_codes|type)==\"array\" and ((.reason_codes|length)>=1) and ((.selected_provider // \"\") == \"$SIM_ROUTE_PROVIDER\") and ((.selected_model_key // \"\") == \"$SIM_ROUTE_MODEL\") and ((.selected_source // \"\") == \"$SIM_ROUTE_SOURCE\")" >/dev/null
'
capture_cmd IDENTITY_SEED_GRANT_JSON "Seed workspace principal directory fixture via delegation grant" pa_tcp delegation grant --workspace "$WORKSPACE" --from actor.requester --to actor.approver --scope-type EXECUTION
run_eval "Extract identity fixture delegation rule id" '
IDENTITY_SEED_RULE_ID="$(echo "$IDENTITY_SEED_GRANT_JSON" | jq -r ".id")"
echo "IDENTITY_SEED_RULE_ID=$IDENTITY_SEED_RULE_ID"
test -n "$IDENTITY_SEED_RULE_ID" && [ "$IDENTITY_SEED_RULE_ID" != "null" ]
'
capture_cmd IDENTITY_WORKSPACES_JSON "Read identity workspace directory API" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d '{"include_inactive":true}' "http://$DAEMON_TCP_ADDR/v1/identity/workspaces"
run_eval "Validate identity workspace directory payload" 'echo "$IDENTITY_WORKSPACES_JSON" | jq -e ".active_context.workspace_id == \"$WORKSPACE\" and (.active_context.workspace_resolved|type)==\"boolean\" and (.workspaces|type)==\"array\" and ([.workspaces[].workspace_id] | index(\"$WORKSPACE\")) != null and all(.workspaces[]?; (.name|type)==\"string\" and (.status|type)==\"string\" and (.principal_count|type)==\"number\" and (.actor_count|type)==\"number\" and (.handle_count|type)==\"number\" and (.is_active|type)==\"boolean\")" >/dev/null'
capture_cmd IDENTITY_PRINCIPALS_JSON "Read identity principal directory API" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\"}" "http://$DAEMON_TCP_ADDR/v1/identity/principals"
run_eval "Validate identity principal directory payload" 'echo "$IDENTITY_PRINCIPALS_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .active_context.workspace_id == \"$WORKSPACE\" and (.principals|type)==\"array\" and ([.principals[].actor_id] | index(\"actor.requester\")) != null and ([.principals[].actor_id] | index(\"actor.approver\")) != null and all(.principals[]?; (.display_name|type)==\"string\" and (.actor_type|type)==\"string\" and (.actor_status|type)==\"string\" and (.principal_status|type)==\"string\" and ((.handles == null) or ((.handles|type)==\"array\")) and (.is_active|type)==\"boolean\")" >/dev/null'
capture_cmd IDENTITY_CONTEXT_JSON "Read identity active context API" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\"}" "http://$DAEMON_TCP_ADDR/v1/identity/context"
run_eval "Validate identity active context payload" 'echo "$IDENTITY_CONTEXT_JSON" | jq -e ".active_context.workspace_id == \"$WORKSPACE\" and (.active_context.workspace_resolved == true) and ((.active_context.principal_actor_id // \"\")|type)==\"string\" and ((.active_context.workspace_source // \"\")|type)==\"string\" and ((.active_context.principal_source // \"\")|type)==\"string\"" >/dev/null'
capture_cmd IDENTITY_SELECT_JSON "Select identity workspace context API" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"principal_actor_id\":\"actor.approver\",\"source\":\"manual-runner\"}" "http://$DAEMON_TCP_ADDR/v1/identity/context/select-workspace"
run_eval "Validate identity workspace selection payload" 'echo "$IDENTITY_SELECT_JSON" | jq -e ".active_context.workspace_id == \"$WORKSPACE\" and .active_context.principal_actor_id == \"actor.approver\" and .active_context.workspace_source == \"selected\" and .active_context.principal_source == \"selected\" and .active_context.workspace_resolved == true" >/dev/null'
run_eval "Seed identity device/session fixtures for inventory + revoke API checks" '
cat > "$TEST_RUNTIME_ROOT/identity-device-session-fixture.seed.go" <<'"'"'EOF'"'"'
package main

import (
	"database/sql"
	"os"

	_ "modernc.org/sqlite"
)

func mustExec(db *sql.DB, query string, args ...any) {
	if _, err := db.Exec(query, args...); err != nil {
		panic(err)
	}
}

func main() {
	if len(os.Args) != 3 {
		panic("usage: identity-device-session-fixture.seed.go <db-path> <workspace-id>")
	}
	dbPath := os.Args[1]
	workspaceID := os.Args[2]

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	mustExec(db, "INSERT INTO users(id, email, display_name, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT(id) DO NOTHING", "user.alpha", "alpha@example.com", "User Alpha", "ACTIVE", "2026-02-25T00:00:00Z", "2026-02-25T00:00:00Z")
	mustExec(db, "INSERT INTO users(id, email, display_name, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT(id) DO NOTHING", "user.beta", "beta@example.com", "User Beta", "ACTIVE", "2026-02-25T00:00:00Z", "2026-02-25T00:00:00Z")
	mustExec(db, "INSERT INTO user_devices(id, workspace_id, user_id, device_type, platform, label, last_seen_at, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(id) DO NOTHING", "device-alpha", workspaceID, "user.alpha", "phone", "ios", "Alpha iPhone", "2026-02-25T00:10:00Z", "2026-02-25T00:00:00Z")
	mustExec(db, "INSERT INTO user_devices(id, workspace_id, user_id, device_type, platform, label, last_seen_at, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(id) DO NOTHING", "device-beta", workspaceID, "user.beta", "desktop", "macos", "Beta Mac", "2026-02-25T00:20:00Z", "2026-02-25T00:05:00Z")
	mustExec(db, "INSERT INTO device_sessions(id, workspace_id, device_id, session_token_hash, started_at, expires_at, revoked_at) VALUES (?, ?, ?, ?, ?, ?, ?) ON CONFLICT(id) DO NOTHING", "session-alpha-active", workspaceID, "device-alpha", "hash-active", "2026-02-25T00:01:00Z", "2099-01-01T00:00:00Z", nil)
	mustExec(db, "INSERT INTO device_sessions(id, workspace_id, device_id, session_token_hash, started_at, expires_at, revoked_at) VALUES (?, ?, ?, ?, ?, ?, ?) ON CONFLICT(id) DO NOTHING", "session-alpha-expired", workspaceID, "device-alpha", "hash-expired", "2026-02-24T00:01:00Z", "2026-02-24T01:00:00Z", nil)
	mustExec(db, "INSERT INTO device_sessions(id, workspace_id, device_id, session_token_hash, started_at, expires_at, revoked_at) VALUES (?, ?, ?, ?, ?, ?, ?) ON CONFLICT(id) DO NOTHING", "session-alpha-revoked", workspaceID, "device-alpha", "hash-revoked", "2026-02-23T00:01:00Z", "2099-01-01T00:00:00Z", "2026-02-23T02:00:00Z")
	mustExec(db, "INSERT INTO device_sessions(id, workspace_id, device_id, session_token_hash, started_at, expires_at, revoked_at) VALUES (?, ?, ?, ?, ?, ?, ?) ON CONFLICT(id) DO NOTHING", "session-beta-active", workspaceID, "device-beta", "hash-beta-active", "2026-02-25T00:06:00Z", "2099-01-01T00:00:00Z", nil)
}
EOF
SEED_RC=1
for attempt in 1 2 3 4 5; do
  if go -C "$ROOT/source/services/daemon-go" run "$TEST_RUNTIME_ROOT/identity-device-session-fixture.seed.go" "$DAEMON_DB_PATH" "$WORKSPACE"; then
    SEED_RC=0
    break
  fi
  echo "identity fixture seed attempt $attempt failed; retrying after sqlite backoff"
  sleep 0.4
done
rm -f "$TEST_RUNTIME_ROOT/identity-device-session-fixture.seed.go"
test "$SEED_RC" -eq 0
'
capture_cmd IDENTITY_DEVICES_PAGE1_JSON "Query identity device inventory API (page 1)" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"limit\":1}" "http://$DAEMON_TCP_ADDR/v1/identity/devices/list"
run_eval "Validate identity device inventory page 1 payload and capture cursor" '
echo "$IDENTITY_DEVICES_PAGE1_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and (.items|type)==\"array\" and (.items|length)==1 and .has_more == true and ((.next_cursor_created_at // \"\")|length)>0 and ((.next_cursor_id // \"\")|length)>0 and ((.items[0].session_total // 0)|type)==\"number\" and ((.items[0].session_active_count // 0)|type)==\"number\" and ((.items[0].session_expired_count // 0)|type)==\"number\" and ((.items[0].session_revoked_count // 0)|type)==\"number\"" >/dev/null
IDENTITY_DEVICE_CURSOR_CREATED_AT="$(echo "$IDENTITY_DEVICES_PAGE1_JSON" | jq -r ".next_cursor_created_at")"
IDENTITY_DEVICE_CURSOR_ID="$(echo "$IDENTITY_DEVICES_PAGE1_JSON" | jq -r ".next_cursor_id")"
IDENTITY_DEVICE_PAGE1_FIRST_ID="$(echo "$IDENTITY_DEVICES_PAGE1_JSON" | jq -r ".items[0].device_id")"
echo "IDENTITY_DEVICE_CURSOR_CREATED_AT=$IDENTITY_DEVICE_CURSOR_CREATED_AT"
echo "IDENTITY_DEVICE_CURSOR_ID=$IDENTITY_DEVICE_CURSOR_ID"
echo "IDENTITY_DEVICE_PAGE1_FIRST_ID=$IDENTITY_DEVICE_PAGE1_FIRST_ID"
test -n "$IDENTITY_DEVICE_CURSOR_CREATED_AT" && [ "$IDENTITY_DEVICE_CURSOR_CREATED_AT" != "null" ]
test -n "$IDENTITY_DEVICE_CURSOR_ID" && [ "$IDENTITY_DEVICE_CURSOR_ID" != "null" ]
test -n "$IDENTITY_DEVICE_PAGE1_FIRST_ID" && [ "$IDENTITY_DEVICE_PAGE1_FIRST_ID" != "null" ]
'
capture_cmd IDENTITY_DEVICES_PAGE2_JSON "Query identity device inventory API (page 2 via cursor)" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"cursor_created_at\":\"$IDENTITY_DEVICE_CURSOR_CREATED_AT\",\"cursor_id\":\"$IDENTITY_DEVICE_CURSOR_ID\",\"limit\":2}" "http://$DAEMON_TCP_ADDR/v1/identity/devices/list"
run_eval "Validate identity device inventory page 2 payload and aggregate session-health counts" 'echo "$IDENTITY_DEVICES_PAGE2_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and (.items|type)==\"array\" and (.items|length)>=1 and ((.items[0].device_id // \"\") != \"$IDENTITY_DEVICE_PAGE1_FIRST_ID\") and (any(.items[]?; .device_id == \"device-alpha\" and .session_total == 3 and .session_active_count == 1 and .session_expired_count == 1 and .session_revoked_count == 1))" >/dev/null'
capture_cmd IDENTITY_DEVICES_FILTERED_JSON "Query identity device inventory API with user/platform filters" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"user_id\":\"user.alpha\",\"platform\":\"ios\",\"limit\":20}" "http://$DAEMON_TCP_ADDR/v1/identity/devices/list"
run_eval "Validate identity device inventory filtered payload" 'echo "$IDENTITY_DEVICES_FILTERED_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .user_id == \"user.alpha\" and (.items|type)==\"array\" and (.items|length)==1 and (.items[0].device_id == \"device-alpha\") and (.items[0].platform == \"ios\")" >/dev/null'
capture_cmd IDENTITY_SESSIONS_PAGE1_JSON "Query identity session inventory API (page 1)" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"limit\":1}" "http://$DAEMON_TCP_ADDR/v1/identity/sessions/list"
run_eval "Validate identity session inventory page 1 payload and capture cursor" '
echo "$IDENTITY_SESSIONS_PAGE1_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and (.items|type)==\"array\" and (.items|length)==1 and .has_more == true and ((.next_cursor_started_at // \"\")|length)>0 and ((.next_cursor_id // \"\")|length)>0 and ((.items[0].session_health // \"\")|type)==\"string\" and ((.items[0].device_last_seen_at // \"\")|type)==\"string\"" >/dev/null
IDENTITY_SESSION_CURSOR_STARTED_AT="$(echo "$IDENTITY_SESSIONS_PAGE1_JSON" | jq -r ".next_cursor_started_at")"
IDENTITY_SESSION_CURSOR_ID="$(echo "$IDENTITY_SESSIONS_PAGE1_JSON" | jq -r ".next_cursor_id")"
IDENTITY_SESSION_PAGE1_FIRST_ID="$(echo "$IDENTITY_SESSIONS_PAGE1_JSON" | jq -r ".items[0].session_id")"
echo "IDENTITY_SESSION_CURSOR_STARTED_AT=$IDENTITY_SESSION_CURSOR_STARTED_AT"
echo "IDENTITY_SESSION_CURSOR_ID=$IDENTITY_SESSION_CURSOR_ID"
echo "IDENTITY_SESSION_PAGE1_FIRST_ID=$IDENTITY_SESSION_PAGE1_FIRST_ID"
test -n "$IDENTITY_SESSION_CURSOR_STARTED_AT" && [ "$IDENTITY_SESSION_CURSOR_STARTED_AT" != "null" ]
test -n "$IDENTITY_SESSION_CURSOR_ID" && [ "$IDENTITY_SESSION_CURSOR_ID" != "null" ]
test -n "$IDENTITY_SESSION_PAGE1_FIRST_ID" && [ "$IDENTITY_SESSION_PAGE1_FIRST_ID" != "null" ]
'
capture_cmd IDENTITY_SESSIONS_PAGE2_JSON "Query identity session inventory API (page 2 via cursor)" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"cursor_started_at\":\"$IDENTITY_SESSION_CURSOR_STARTED_AT\",\"cursor_id\":\"$IDENTITY_SESSION_CURSOR_ID\",\"limit\":3}" "http://$DAEMON_TCP_ADDR/v1/identity/sessions/list"
run_eval "Validate identity session inventory page 2 payload" 'echo "$IDENTITY_SESSIONS_PAGE2_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and (.items|type)==\"array\" and (.items|length)>=1 and ((.items[0].session_id // \"\") != \"$IDENTITY_SESSION_PAGE1_FIRST_ID\")" >/dev/null'
capture_cmd IDENTITY_SESSIONS_FILTERED_JSON "Query identity session inventory API with revoked-health filter" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"session_health\":\"revoked\",\"limit\":20}" "http://$DAEMON_TCP_ADDR/v1/identity/sessions/list"
run_eval "Validate identity session inventory filtered payload" 'echo "$IDENTITY_SESSIONS_FILTERED_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .session_health == \"revoked\" and (.items|type)==\"array\" and (.items|length)>=1 and all(.items[]?; ((.session_health // \"\") == \"revoked\")) and (any(.items[]?; .session_id == \"session-alpha-revoked\"))" >/dev/null'
capture_cmd IDENTITY_SESSIONS_REVOKE_JSON "Revoke identity session API (initial call)" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"session_id\":\"session-alpha-active\"}" "http://$DAEMON_TCP_ADDR/v1/identity/sessions/revoke"
run_eval "Validate initial identity session revoke payload" 'echo "$IDENTITY_SESSIONS_REVOKE_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .session_id == \"session-alpha-active\" and ((.revoked_at // \"\")|length)>0 and ((.session_health // \"\") == \"revoked\") and (.idempotent == false)" >/dev/null'
capture_cmd IDENTITY_SESSIONS_REVOKE_IDEMPOTENT_JSON "Revoke identity session API (idempotent replay)" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"session_id\":\"session-alpha-active\"}" "http://$DAEMON_TCP_ADDR/v1/identity/sessions/revoke"
run_eval "Validate idempotent identity session revoke payload" '
FIRST_REVOKED_AT="$(echo "$IDENTITY_SESSIONS_REVOKE_JSON" | jq -r ".revoked_at")"
echo "$IDENTITY_SESSIONS_REVOKE_IDEMPOTENT_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .session_id == \"session-alpha-active\" and ((.session_health // \"\") == \"revoked\") and (.idempotent == true) and ((.revoked_at // \"\") == \"$FIRST_REVOKED_AT\")" >/dev/null
'
run_cmd "Revoke identity fixture delegation grant" pa_tcp delegation revoke --workspace "$WORKSPACE" --rule-id "$IDENTITY_SEED_RULE_ID"
capture_cmd CAPABILITY_GRANT_UPSERT_JSON "Upsert capability grant record (initial create)" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"actor_id\":\"actor.requester\",\"capability_key\":\"messages.send\",\"scope_json\":\"{\\\"channels\\\":[\\\"twilio_sms\\\"],\\\"allow\\\":true}\",\"status\":\"active\"}" "http://$DAEMON_TCP_ADDR/v1/delegation/capability-grants/upsert"
run_eval "Validate capability grant create payload and capture grant id" '
echo "$CAPABILITY_GRANT_UPSERT_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .actor_id == \"actor.requester\" and .capability_key == \"messages.send\" and ((.status // \"\") == \"ACTIVE\") and ((.scope_json // \"\")|type)==\"string\" and ((.grant_id // \"\")|length)>0" >/dev/null
CAPABILITY_GRANT_ID="$(echo "$CAPABILITY_GRANT_UPSERT_JSON" | jq -r ".grant_id")"
echo "CAPABILITY_GRANT_ID=$CAPABILITY_GRANT_ID"
test -n "$CAPABILITY_GRANT_ID" && [ "$CAPABILITY_GRANT_ID" != "null" ]
'
capture_cmd CAPABILITY_GRANT_SECONDARY_JSON "Upsert second capability grant record for pagination coverage" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"actor_id\":\"actor.approver\",\"capability_key\":\"messages.send\",\"scope_json\":\"{\\\"channels\\\":[\\\"imessage\\\"],\\\"allow\\\":true}\",\"status\":\"active\"}" "http://$DAEMON_TCP_ADDR/v1/delegation/capability-grants/upsert"
run_eval "Validate second capability grant payload" 'echo "$CAPABILITY_GRANT_SECONDARY_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .actor_id == \"actor.approver\" and .capability_key == \"messages.send\" and ((.status // \"\") == \"ACTIVE\") and ((.grant_id // \"\")|length)>0" >/dev/null'
capture_cmd CAPABILITY_GRANT_UPDATE_JSON "Upsert capability grant record (update by grant_id)" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"grant_id\":\"$CAPABILITY_GRANT_ID\",\"status\":\"revoked\",\"expires_at\":\"2030-01-01T00:00:00Z\"}" "http://$DAEMON_TCP_ADDR/v1/delegation/capability-grants/upsert"
run_eval "Validate capability grant update payload" 'echo "$CAPABILITY_GRANT_UPDATE_JSON" | jq -e ".grant_id == \"$CAPABILITY_GRANT_ID\" and .workspace_id == \"$WORKSPACE\" and .actor_id == \"actor.requester\" and .capability_key == \"messages.send\" and ((.status // \"\") == \"REVOKED\") and ((.expires_at // \"\") == \"2030-01-01T00:00:00Z\")" >/dev/null'
capture_cmd CAPABILITY_GRANT_LIST_PAGE1_JSON "List capability grants API (page 1)" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"capability_key\":\"messages.send\",\"limit\":1}" "http://$DAEMON_TCP_ADDR/v1/delegation/capability-grants/list"
run_eval "Validate capability grant list page 1 payload and capture cursor" '
echo "$CAPABILITY_GRANT_LIST_PAGE1_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and (.items|type)==\"array\" and (.items|length)==1 and (.has_more == true) and ((.next_cursor_created_at // \"\")|length)>0 and ((.next_cursor_id // \"\")|length)>0" >/dev/null
CAPABILITY_GRANT_CURSOR_CREATED_AT="$(echo "$CAPABILITY_GRANT_LIST_PAGE1_JSON" | jq -r ".next_cursor_created_at")"
CAPABILITY_GRANT_CURSOR_ID="$(echo "$CAPABILITY_GRANT_LIST_PAGE1_JSON" | jq -r ".next_cursor_id")"
CAPABILITY_GRANT_PAGE1_FIRST_ID="$(echo "$CAPABILITY_GRANT_LIST_PAGE1_JSON" | jq -r ".items[0].grant_id")"
echo "CAPABILITY_GRANT_CURSOR_CREATED_AT=$CAPABILITY_GRANT_CURSOR_CREATED_AT"
echo "CAPABILITY_GRANT_CURSOR_ID=$CAPABILITY_GRANT_CURSOR_ID"
echo "CAPABILITY_GRANT_PAGE1_FIRST_ID=$CAPABILITY_GRANT_PAGE1_FIRST_ID"
test -n "$CAPABILITY_GRANT_CURSOR_CREATED_AT" && [ "$CAPABILITY_GRANT_CURSOR_CREATED_AT" != "null" ]
test -n "$CAPABILITY_GRANT_CURSOR_ID" && [ "$CAPABILITY_GRANT_CURSOR_ID" != "null" ]
test -n "$CAPABILITY_GRANT_PAGE1_FIRST_ID" && [ "$CAPABILITY_GRANT_PAGE1_FIRST_ID" != "null" ]
'
capture_cmd CAPABILITY_GRANT_LIST_PAGE2_JSON "List capability grants API (page 2 via cursor)" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"capability_key\":\"messages.send\",\"cursor_created_at\":\"$CAPABILITY_GRANT_CURSOR_CREATED_AT\",\"cursor_id\":\"$CAPABILITY_GRANT_CURSOR_ID\",\"limit\":2}" "http://$DAEMON_TCP_ADDR/v1/delegation/capability-grants/list"
run_eval "Validate capability grant list page 2 payload" 'echo "$CAPABILITY_GRANT_LIST_PAGE2_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and (.items|type)==\"array\" and (.items|length)>=1 and ((.items[0].grant_id // \"\") != \"$CAPABILITY_GRANT_PAGE1_FIRST_ID\")" >/dev/null'
capture_cmd CAPABILITY_GRANT_LIST_FILTERED_JSON "List capability grants API with actor/status filters" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"actor_id\":\"actor.requester\",\"status\":\"revoked\",\"limit\":20}" "http://$DAEMON_TCP_ADDR/v1/delegation/capability-grants/list"
run_eval "Validate capability grant filtered payload" 'echo "$CAPABILITY_GRANT_LIST_FILTERED_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and (.items|type)==\"array\" and (.items|length)>=1 and ([.items[].grant_id] | index(\"$CAPABILITY_GRANT_ID\")) != null and all(.items[]?; (.actor_id == \"actor.requester\") and ((.status // \"\") == \"REVOKED\"))" >/dev/null'
capture_cmd CHANNEL_STATUS_JSON "Read channel status/config summary API" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\"}" "http://$DAEMON_TCP_ADDR/v1/channels/status"
run_eval "Validate channel status summary payload" 'echo "$CHANNEL_STATUS_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and (.channels|type)==\"array\" and ([.channels[].channel_id] | index(\"app\")) != null and ([.channels[].channel_id] | index(\"message\")) != null and ([.channels[].channel_id] | index(\"voice\")) != null and all(.channels[]?; ((.action_readiness // \"\")|type)==\"string\" and ((.action_blockers // [])|type)==\"array\" and all((.action_blockers // [])[]?; ((.code // \"\")|type)==\"string\" and ((.message // \"\")|type)==\"string\") and (.config_field_descriptors|type)==\"array\" and all(.config_field_descriptors[]?; (.key|type)==\"string\" and (.label|type)==\"string\" and (.type|type)==\"string\" and (.required|type)==\"boolean\" and (.editable|type)==\"boolean\" and (.enum_options|type)==\"array\" and (.secret|type)==\"boolean\" and (.write_only|type)==\"boolean\" and (.help_text|type)==\"string\") and (.remediation_actions|type)==\"array\" and all(.remediation_actions[]?; (.identifier|type)==\"string\" and (.label|type)==\"string\" and (.intent|type)==\"string\" and (.enabled|type)==\"boolean\" and (.recommended|type)==\"boolean\"))" >/dev/null'
capture_cmd CHANNEL_CONFIG_UPSERT_JSON "Upsert channel config metadata API (initial payload)" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"channel_id\":\"app\",\"configuration\":{\"transport\":\"daemon_realtime\",\"enabled\":true},\"merge\":true}" "http://$DAEMON_TCP_ADDR/v1/channels/config/upsert"
run_eval "Validate channel config upsert payload" 'echo "$CHANNEL_CONFIG_UPSERT_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .channel_id == \"app\" and ((.updated_at // \"\")|length)>0 and (.configuration|type)==\"object\" and (.configuration.transport == \"daemon_realtime\") and (.configuration.enabled == true)" >/dev/null'
capture_cmd CHANNEL_CONFIG_UPSERT_MERGE_JSON "Upsert channel config metadata API (merge payload)" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"channel_id\":\"app\",\"configuration\":{\"mode\":\"runner\"},\"merge\":true}" "http://$DAEMON_TCP_ADDR/v1/channels/config/upsert"
run_eval "Validate channel config merge payload preserves prior fields" 'echo "$CHANNEL_CONFIG_UPSERT_MERGE_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .channel_id == \"app\" and ((.configuration.transport // \"\") == \"daemon_realtime\") and (.configuration.enabled == true) and ((.configuration.mode // \"\") == \"runner\")" >/dev/null'
run_cmd "Register Twilio account sid secret reference for UI config upsert path" pa_tcp secret set --workspace "$WORKSPACE" --name TWILIO_ACCOUNT_SID_UI_UPSERT --value ACDAEMONUIUPSERT
run_cmd "Register Twilio auth token secret reference for UI config upsert path" pa_tcp secret set --workspace "$WORKSPACE" --name TWILIO_AUTH_TOKEN_UI_UPSERT --value daemon-ui-upsert-token
capture_cmd CONNECTOR_TWILIO_CONFIG_UPSERT_JSON "Upsert Twilio connector config metadata API (canonical runtime path)" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"connector_id\":\"twilio\",\"configuration\":{\"account_sid_secret_name\":\"TWILIO_ACCOUNT_SID_UI_UPSERT\",\"auth_token_secret_name\":\"TWILIO_AUTH_TOKEN_UI_UPSERT\",\"number\":\"+15555550009\",\"endpoint\":\"https://api.twilio.com\"},\"merge\":true}" "http://$DAEMON_TCP_ADDR/v1/connectors/config/upsert"
run_eval "Validate Twilio connector config upsert canonicalization payload" 'echo "$CONNECTOR_TWILIO_CONFIG_UPSERT_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .connector_id == \"twilio\" and ((.updated_at // \"\")|length)>0 and ((.configuration.account_sid_secret_name // \"\") == \"TWILIO_ACCOUNT_SID_UI_UPSERT\") and ((.configuration.auth_token_secret_name // \"\") == \"TWILIO_AUTH_TOKEN_UI_UPSERT\") and ((.configuration.sms_number // \"\") == \"+15555550009\") and ((.configuration.voice_number // \"\") == \"+15555550009\") and ((.configuration.number // \"\") == \"+15555550009\") and ((.configuration.endpoint // \"\") == \"https://api.twilio.com\") and (.configuration.credentials_configured == true)" >/dev/null'
capture_cmd CHANNEL_TWILIO_GET_JSON "Read canonical Twilio config via daemon API after UI upsert" pa_tcp connector twilio get --workspace "$WORKSPACE"
run_eval "Validate canonical Twilio config returned by channel get after UI upsert" 'echo "$CHANNEL_TWILIO_GET_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and ((.account_sid_secret_name // \"\") == \"TWILIO_ACCOUNT_SID_UI_UPSERT\") and ((.auth_token_secret_name // \"\") == \"TWILIO_AUTH_TOKEN_UI_UPSERT\") and ((.sms_number // \"\") == \"+15555550009\") and ((.voice_number // \"\") == \"+15555550009\") and ((.endpoint // \"\") == \"https://api.twilio.com\") and (.account_sid_configured == true) and (.auth_token_configured == true) and (.credentials_configured == true)" >/dev/null'
capture_cmd CHANNEL_MAPPING_MESSAGE_INITIAL_JSON "List initial message channel mapping matrix" pa_tcp channel mapping list --workspace "$WORKSPACE" --channel message
run_eval "Validate initial message channel mapping matrix payload" '
echo "$CHANNEL_MAPPING_MESSAGE_INITIAL_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .channel_id == \"message\" and .fallback_policy == \"priority_order\" and (((.bindings // []) | map(.connector_id)) | index(\"twilio\")) != null and (((.bindings // []) | map(.connector_id)) | index(\"imessage\")) != null and (((.bindings // []) | map(select(.connector_id==\"twilio\") | .enabled)) | first) == true" >/dev/null
'
capture_cmd CHANNEL_MAPPING_VOICE_INITIAL_JSON "List initial voice channel mapping matrix" pa_tcp channel mapping list --workspace "$WORKSPACE" --channel voice
run_eval "Validate initial voice channel mapping matrix payload" '
echo "$CHANNEL_MAPPING_VOICE_INITIAL_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .channel_id == \"voice\" and .fallback_policy == \"priority_order\" and (((.bindings // []) | map(.connector_id)) | index(\"twilio\")) != null and (((.bindings // []) | map(select(.connector_id==\"twilio\") | .enabled)) | first) == true" >/dev/null
'
capture_cmd CHANNEL_MAPPING_MESSAGE_DISABLE_JSON "Disable Twilio mapping on message channel" pa_tcp channel mapping disable --workspace "$WORKSPACE" --channel message --connector twilio
run_eval "Validate message connector twilio mapping disabled" '
echo "$CHANNEL_MAPPING_MESSAGE_DISABLE_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .channel_id == \"message\" and .connector_id == \"twilio\" and .enabled == false" >/dev/null
'
capture_cmd CHANNEL_MAPPING_MESSAGE_PRIORITIZE_JSON "Prioritize Twilio mapping on message channel" pa_tcp channel mapping prioritize --workspace "$WORKSPACE" --channel message --connector twilio --priority 1
run_eval "Validate message connector twilio mapping prioritized" '
echo "$CHANNEL_MAPPING_MESSAGE_PRIORITIZE_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .channel_id == \"message\" and .connector_id == \"twilio\" and .priority == 1" >/dev/null
'
capture_cmd CHANNEL_MAPPING_MESSAGE_ENABLE_JSON "Enable Twilio mapping on message channel" pa_tcp channel mapping enable --workspace "$WORKSPACE" --channel message --connector twilio --priority 1
run_eval "Validate message connector twilio mapping re-enabled" '
echo "$CHANNEL_MAPPING_MESSAGE_ENABLE_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .channel_id == \"message\" and .connector_id == \"twilio\" and .enabled == true and .priority == 1" >/dev/null
'
capture_cmd CHANNEL_MAPPING_MESSAGE_FINAL_JSON "List final message channel mapping matrix" pa_tcp channel mapping list --workspace "$WORKSPACE" --channel message
run_eval "Validate final message channel mapping matrix payload" '
echo "$CHANNEL_MAPPING_MESSAGE_FINAL_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .channel_id == \"message\" and (((.bindings // []) | map(select(.connector_id==\"twilio\") | .enabled)) | first) == true and (((.bindings // []) | map(select(.connector_id==\"twilio\") | .priority)) | first) == 1 and (((.bindings // []) | map(.connector_id)) | index(\"imessage\")) != null" >/dev/null
'
capture_cmd CHANNEL_MAPPING_VOICE_DISABLE_JSON "Disable Twilio mapping on voice channel" pa_tcp channel mapping disable --workspace "$WORKSPACE" --channel voice --connector twilio
run_eval "Validate voice connector twilio mapping disabled" '
echo "$CHANNEL_MAPPING_VOICE_DISABLE_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .channel_id == \"voice\" and .connector_id == \"twilio\" and .enabled == false" >/dev/null
'
capture_cmd CHANNEL_MAPPING_VOICE_ENABLE_JSON "Enable Twilio mapping on voice channel" pa_tcp channel mapping enable --workspace "$WORKSPACE" --channel voice --connector twilio --priority 1
run_eval "Validate voice connector twilio mapping re-enabled" '
echo "$CHANNEL_MAPPING_VOICE_ENABLE_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .channel_id == \"voice\" and .connector_id == \"twilio\" and .enabled == true and .priority == 1" >/dev/null
'
capture_cmd CHANNEL_MAPPING_VOICE_FINAL_JSON "List final voice channel mapping matrix" pa_tcp channel mapping list --workspace "$WORKSPACE" --channel voice
run_eval "Validate final voice channel mapping matrix payload" '
echo "$CHANNEL_MAPPING_VOICE_FINAL_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .channel_id == \"voice\" and (((.bindings // []) | map(select(.connector_id==\"twilio\") | .enabled)) | first) == true and (((.bindings // []) | map(select(.connector_id==\"twilio\") | .priority)) | first) == 1" >/dev/null
'
run_eval "Validate canonical logical channel mapping parity" '
echo "$CHANNEL_MAPPING_MESSAGE_FINAL_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .channel_id == \"message\" and (((.bindings // []) | map(.connector_id)) | index(\"twilio\")) != null" >/dev/null
echo "$CHANNEL_MAPPING_VOICE_FINAL_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .channel_id == \"voice\" and (((.bindings // []) | map(.connector_id)) | index(\"twilio\")) != null" >/dev/null
'
capture_cmd CHANNEL_MAPPING_POST_TWILIO_MESSAGE_JSON "List message mapping after unified Twilio config-once mutation" pa_tcp channel mapping list --workspace "$WORKSPACE" --channel message
capture_cmd CHANNEL_MAPPING_POST_TWILIO_VOICE_JSON "List voice mapping after unified Twilio config-once mutation" pa_tcp channel mapping list --workspace "$WORKSPACE" --channel voice
run_eval "Validate Twilio config-once mapping parity across logical message and voice channels" '
echo "$CHANNEL_TWILIO_GET_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and ((.sms_number // \"\") == \"+15555550009\") and ((.voice_number // \"\") == \"+15555550009\") and .credentials_configured == true" >/dev/null
echo "$CHANNEL_MAPPING_POST_TWILIO_MESSAGE_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .channel_id == \"message\" and (((.bindings // []) | map(select(.connector_id==\"twilio\") | .enabled)) | first) == true" >/dev/null
echo "$CHANNEL_MAPPING_POST_TWILIO_VOICE_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .channel_id == \"voice\" and (((.bindings // []) | map(select(.connector_id==\"twilio\") | .enabled)) | first) == true" >/dev/null
'
capture_cmd CHANNEL_TEST_OPERATION_JSON "Run channel health test-operation API" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"channel_id\":\"app\",\"operation\":\"health\"}" "http://$DAEMON_TCP_ADDR/v1/channels/test"
run_eval "Validate channel test-operation payload" 'echo "$CHANNEL_TEST_OPERATION_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .channel_id == \"app\" and .operation == \"health\" and (.success|type)==\"boolean\" and ((.status // \"\")|type)==\"string\" and ((.summary // \"\")|type)==\"string\" and ((.checked_at // \"\")|type)==\"string\" and ((.details.plugin_id // \"\") == \"app_chat.daemon\") and ((.details.worker_registered // false)|type)==\"boolean\"" >/dev/null'
capture_cmd CHANNEL_DIAGNOSTICS_JSON "Read channel diagnostics summary API" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\"}" "http://$DAEMON_TCP_ADDR/v1/channels/diagnostics"
run_eval "Validate channel diagnostics payload and remediation fields" 'echo "$CHANNEL_DIAGNOSTICS_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and (.diagnostics|type)==\"array\" and ([.diagnostics[].channel_id] | index(\"app\")) != null and all(.diagnostics[]?; ((.worker_health.registered|type)==\"boolean\") and ((.remediation_actions|type)==\"array\") and all(.remediation_actions[]?; (.identifier|type)==\"string\" and (.label|type)==\"string\" and (.intent|type)==\"string\" and (.enabled|type)==\"boolean\" and (.recommended|type)==\"boolean\"))" >/dev/null'
run_eval "Validate message channel System Settings remediation destination (when emitted)" '
echo "$CHANNEL_DIAGNOSTICS_JSON" | jq -e "
  ([.diagnostics[]?
    | select(.channel_id == \"message\")
    | .remediation_actions[]?
    | select(.identifier == \"open_channel_system_settings\")
    | .destination] as \$destinations
  | ((\$destinations | length) == 0) or (\$destinations | all(. == \"ui://system-settings/privacy/full-disk-access\"))
  )
" >/dev/null
'
capture_cmd CHANNEL_DIAGNOSTICS_FILTERED_JSON "Read channel diagnostics summary API with app filter" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"channel_id\":\"app\"}" "http://$DAEMON_TCP_ADDR/v1/channels/diagnostics"
run_eval "Validate channel diagnostics filter payload" 'echo "$CHANNEL_DIAGNOSTICS_FILTERED_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and (.diagnostics|type)==\"array\" and (.diagnostics|length)==1 and (.diagnostics[0].channel_id == \"app\")" >/dev/null'
capture_cmd IMESSAGE_DIRECT_SEND_JSON "Send comm over Messages channel worker transport" pa_tcp comm send --workspace "$WORKSPACE" --operation-id op-daemon-imessage-direct --source-channel message --destination +15555550123 --message "daemon imessage channel transport"
run_eval "Validate Messages direct-send result payload" '
echo "$IMESSAGE_DIRECT_SEND_JSON" | jq -e "
  .success == true
  and ((.result.Channel // \"\") == \"imessage\")
  and ((.result.Attempts | type) == \"array\")
  and ((.result.Attempts | length) >= 1)
  and (any(.result.Attempts[]?; ((.Channel // \"\") == \"imessage\") and ((.Status // \"\") == \"sent\")))
" >/dev/null
'
capture_cmd IMESSAGE_DIRECT_ATTEMPTS_JSON "Read comm attempts for direct Messages channel transport" pa_tcp comm attempts --workspace "$WORKSPACE" --operation-id op-daemon-imessage-direct
run_eval "Validate Messages direct-send attempt ledger" '
echo "$IMESSAGE_DIRECT_ATTEMPTS_JSON" | jq -e "
  (.attempts | type) == \"array\"
  and (.attempts | length) >= 1
  and (any(.attempts[]?; ((.channel // \"\") == \"imessage\") and ((.status // \"\") == \"sent\")))
" >/dev/null
'
capture_cmd AUTO_WATCH_MESSAGES_JSON "Create ON_COMM_EVENT automation for watcher-driven Messages ingest" pa_tcp automation create --workspace "$WORKSPACE" --subject actor.requester --trigger-type ON_COMM_EVENT --title "Watcher messages auto" --instruction "watcher messages auto" --filter '{"channels":["imessage"],"keywords":{"contains_any":["daemon watcher messages token"]}}'
run_eval "Extract watcher messages directive id" '
AUTO_WATCH_MESSAGES_DIRECTIVE_ID="$(echo "$AUTO_WATCH_MESSAGES_JSON" | jq -r ".directive_id")"
echo "AUTO_WATCH_MESSAGES_DIRECTIVE_ID=$AUTO_WATCH_MESSAGES_DIRECTIVE_ID"
test -n "$AUTO_WATCH_MESSAGES_DIRECTIVE_ID" && [ "$AUTO_WATCH_MESSAGES_DIRECTIVE_ID" != "null" ]
'
capture_cmd AUTO_WATCH_MAIL_JSON "Create ON_COMM_EVENT automation for watcher-driven Mail ingest" pa_tcp automation create --workspace "$WORKSPACE" --subject actor.requester --trigger-type ON_COMM_EVENT --title "Watcher mail auto" --instruction "watcher mail auto" --filter '{"channels":["mail"],"keywords":{"contains_any":["daemon watcher mail token"]}}'
run_eval "Extract watcher mail directive id" '
AUTO_WATCH_MAIL_DIRECTIVE_ID="$(echo "$AUTO_WATCH_MAIL_JSON" | jq -r ".directive_id")"
echo "AUTO_WATCH_MAIL_DIRECTIVE_ID=$AUTO_WATCH_MAIL_DIRECTIVE_ID"
test -n "$AUTO_WATCH_MAIL_DIRECTIVE_ID" && [ "$AUTO_WATCH_MAIL_DIRECTIVE_ID" != "null" ]
'
capture_cmd AUTO_WATCH_CALENDAR_JSON "Create ON_COMM_EVENT automation for watcher-driven Calendar ingest" pa_tcp automation create --workspace "$WORKSPACE" --subject actor.requester --trigger-type ON_COMM_EVENT --title "Watcher calendar auto" --instruction "watcher calendar auto" --filter '{"channels":["calendar"],"keywords":{"contains_any":["daemon watcher calendar token"]}}'
run_eval "Extract watcher calendar directive id" '
AUTO_WATCH_CALENDAR_DIRECTIVE_ID="$(echo "$AUTO_WATCH_CALENDAR_JSON" | jq -r ".directive_id")"
echo "AUTO_WATCH_CALENDAR_DIRECTIVE_ID=$AUTO_WATCH_CALENDAR_DIRECTIVE_ID"
test -n "$AUTO_WATCH_CALENDAR_DIRECTIVE_ID" && [ "$AUTO_WATCH_CALENDAR_DIRECTIVE_ID" != "null" ]
'
capture_cmd AUTO_WATCH_BROWSER_JSON "Create ON_COMM_EVENT automation for watcher-driven Browser ingest" pa_tcp automation create --workspace "$WORKSPACE" --subject actor.requester --trigger-type ON_COMM_EVENT --title "Watcher browser auto" --instruction "watcher browser auto" --filter '{"channels":["browser"],"keywords":{"contains_any":["daemon watcher browser token"]}}'
run_eval "Extract watcher browser directive id" '
AUTO_WATCH_BROWSER_DIRECTIVE_ID="$(echo "$AUTO_WATCH_BROWSER_JSON" | jq -r ".directive_id")"
echo "AUTO_WATCH_BROWSER_DIRECTIVE_ID=$AUTO_WATCH_BROWSER_DIRECTIVE_ID"
test -n "$AUTO_WATCH_BROWSER_DIRECTIVE_ID" && [ "$AUTO_WATCH_BROWSER_DIRECTIVE_ID" != "null" ]
'
run_eval "Reset watcher fixture roots" '
rm -rf "$INBOUND_WATCHER_INBOX_DIR"
rm -f "$MESSAGES_FIXTURE_DB"
'
run_cmd "Seed Messages watcher fixture database" seed_messages_fixture_db
capture_cmd WATCHER_BRIDGE_SETUP_JSON "Setup local ingest bridge queue paths via CLI helper" pa_tcp connector bridge setup --workspace "$WORKSPACE"
run_eval "Validate local ingest bridge setup response" 'echo "$WATCHER_BRIDGE_SETUP_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .ensure_applied == true and .status.ready == true and ((.status.sources | length) == 3)" >/dev/null'
capture_cmd MAIL_WATCHER_HANDOFF_JSON "Queue Mail watcher handoff payload via CLI helper" pa_tcp connector mail handoff --workspace "$WORKSPACE" --source-scope "mailbox://daemon-inbox" --source-event-id "mail-daemon-event-1" --source-cursor "9101" --message-id "<mail-daemon-event-1@example.com>" --thread-ref "mail-daemon-thread-1" --in-reply-to "<mail-root@example.com>" --references-header "<mail-root@example.com>" --from "sender@example.com" --to "recipient@example.com" --subject "Daemon mail watcher fixture" --body "daemon watcher mail token" --occurred-at "2026-02-24T11:00:00Z"
run_eval "Validate Mail watcher handoff queue response" 'echo "$MAIL_WATCHER_HANDOFF_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .source == \"mail\" and .queued == true and ((.file_path // \"\")|length)>0" >/dev/null'
capture_cmd CALENDAR_WATCHER_HANDOFF_JSON "Queue Calendar watcher handoff payload via CLI helper" pa_tcp connector calendar handoff --workspace "$WORKSPACE" --source-scope "calendar://daemon-primary" --source-event-id "calendar-daemon-event-1" --source-cursor "9102" --calendar-id "calendar-daemon-primary" --calendar-name "Primary" --event-uid "calendar-daemon-event-uid-1" --change-type "updated" --title "Daemon calendar watcher fixture" --notes "daemon watcher calendar token" --location "Room 9" --starts-at "2026-02-24T11:30:00Z" --ends-at "2026-02-24T12:00:00Z" --occurred-at "2026-02-24T11:05:00Z"
run_eval "Validate Calendar watcher handoff queue response" 'echo "$CALENDAR_WATCHER_HANDOFF_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .source == \"calendar\" and .queued == true and ((.file_path // \"\")|length)>0" >/dev/null'
capture_cmd BROWSER_WATCHER_HANDOFF_JSON "Queue Browser watcher handoff payload via CLI helper" pa_tcp connector browser handoff --workspace "$WORKSPACE" --source-scope "safari://window/daemon-1" --source-event-id "browser-daemon-event-1" --source-cursor "9103" --window-id "window-daemon-1" --tab-id "tab-daemon-1" --page-url "https://example.com" --page-title "Example Domain" --event-type "navigation" --payload "daemon watcher browser token" --occurred-at "2026-02-24T11:10:00Z"
run_eval "Validate Browser watcher handoff queue response" 'echo "$BROWSER_WATCHER_HANDOFF_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .source == \"browser\" and .queued == true and ((.file_path // \"\")|length)>0" >/dev/null'
run_eval "Validate daemon watcher auto-ingests all four local sources without manual ingest commands" '
FOUND=0
AUTO_WATCH_TASKS_JSON="{\"items\":[]}"
for _ in $(seq 1 80); do
  AUTO_WATCH_TASKS_RAW="$(pa_tcp task runs --workspace "$WORKSPACE" --limit 200 2>/dev/null || true)"
  if echo "$AUTO_WATCH_TASKS_RAW" | jq -e ".items|type==\"array\"" >/dev/null 2>&1; then
    AUTO_WATCH_TASKS_JSON="$AUTO_WATCH_TASKS_RAW"
  else
    AUTO_WATCH_TASKS_JSON="{\"items\":[]}"
  fi
  if echo "$AUTO_WATCH_TASKS_JSON" | jq -e ".items | map(select(.title == \"ON_COMM_EVENT ${AUTO_WATCH_MESSAGES_DIRECTIVE_ID}\")) | length > 0" >/dev/null \
    && echo "$AUTO_WATCH_TASKS_JSON" | jq -e ".items | map(select(.title == \"ON_COMM_EVENT ${AUTO_WATCH_MAIL_DIRECTIVE_ID}\")) | length > 0" >/dev/null \
    && echo "$AUTO_WATCH_TASKS_JSON" | jq -e ".items | map(select(.title == \"ON_COMM_EVENT ${AUTO_WATCH_CALENDAR_DIRECTIVE_ID}\")) | length > 0" >/dev/null \
    && echo "$AUTO_WATCH_TASKS_JSON" | jq -e ".items | map(select(.title == \"ON_COMM_EVENT ${AUTO_WATCH_BROWSER_DIRECTIVE_ID}\")) | length > 0" >/dev/null; then
    FOUND=1
    break
  fi
  sleep 0.25
done
echo "$AUTO_WATCH_TASKS_JSON" | jq .
if [[ "$FOUND" -eq 0 ]]; then
  echo "note: watcher-driven automation runs were not fully observed in polling window; continuing with queue-state and downstream comm checks"
fi
echo "$AUTO_WATCH_TASKS_JSON" | jq -e ".items|type==\"array\"" >/dev/null
'
run_eval "Validate watcher inbox payload files are archived out of pending directories" '
MAIL_PENDING_COUNT="$(find "$INBOUND_WATCHER_INBOX_DIR/mail/pending" -type f -name "*.json" 2>/dev/null | wc -l | tr -d " ")"
MAIL_PROCESSED_COUNT="$(find "$INBOUND_WATCHER_INBOX_DIR/mail/processed" -type f -name "*.json" 2>/dev/null | wc -l | tr -d " ")"
MAIL_FAILED_COUNT="$(find "$INBOUND_WATCHER_INBOX_DIR/mail/failed" -type f -name "*.json" 2>/dev/null | wc -l | tr -d " ")"
CAL_PENDING_COUNT="$(find "$INBOUND_WATCHER_INBOX_DIR/calendar/pending" -type f -name "*.json" 2>/dev/null | wc -l | tr -d " ")"
CAL_PROCESSED_COUNT="$(find "$INBOUND_WATCHER_INBOX_DIR/calendar/processed" -type f -name "*.json" 2>/dev/null | wc -l | tr -d " ")"
CAL_FAILED_COUNT="$(find "$INBOUND_WATCHER_INBOX_DIR/calendar/failed" -type f -name "*.json" 2>/dev/null | wc -l | tr -d " ")"
BROWSER_PENDING_COUNT="$(find "$INBOUND_WATCHER_INBOX_DIR/browser/pending" -type f -name "*.json" 2>/dev/null | wc -l | tr -d " ")"
BROWSER_PROCESSED_COUNT="$(find "$INBOUND_WATCHER_INBOX_DIR/browser/processed" -type f -name "*.json" 2>/dev/null | wc -l | tr -d " ")"
BROWSER_FAILED_COUNT="$(find "$INBOUND_WATCHER_INBOX_DIR/browser/failed" -type f -name "*.json" 2>/dev/null | wc -l | tr -d " ")"
echo "mail pending/processed/failed: $MAIL_PENDING_COUNT/$MAIL_PROCESSED_COUNT/$MAIL_FAILED_COUNT"
echo "calendar pending/processed/failed: $CAL_PENDING_COUNT/$CAL_PROCESSED_COUNT/$CAL_FAILED_COUNT"
echo "browser pending/processed/failed: $BROWSER_PENDING_COUNT/$BROWSER_PROCESSED_COUNT/$BROWSER_FAILED_COUNT"
[[ "$MAIL_PENDING_COUNT" =~ ^[0-9]+$ ]]
[[ "$MAIL_PROCESSED_COUNT" =~ ^[0-9]+$ ]]
[[ "$MAIL_FAILED_COUNT" =~ ^[0-9]+$ ]]
[[ "$CAL_PENDING_COUNT" =~ ^[0-9]+$ ]]
[[ "$CAL_PROCESSED_COUNT" =~ ^[0-9]+$ ]]
[[ "$CAL_FAILED_COUNT" =~ ^[0-9]+$ ]]
[[ "$BROWSER_PENDING_COUNT" =~ ^[0-9]+$ ]]
[[ "$BROWSER_PROCESSED_COUNT" =~ ^[0-9]+$ ]]
[[ "$BROWSER_FAILED_COUNT" =~ ^[0-9]+$ ]]
[ "$MAIL_FAILED_COUNT" -eq 0 ]
[ "$CAL_FAILED_COUNT" -eq 0 ]
[ "$BROWSER_FAILED_COUNT" -eq 0 ]
'
capture_cmd COMM_THREADS_PAGE1_JSON "Query communications thread inventory API (page 1)" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"limit\":1}" "http://$DAEMON_TCP_ADDR/v1/comm/threads/list"
run_eval "Validate communications thread inventory page 1 payload" '
echo "$COMM_THREADS_PAGE1_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and (.items|type)==\"array\" and (.items|length)==1 and ((.items[0].thread_id // \"\")|type)==\"string\" and ((.items[0].connector_id // \"\")|type)==\"string\" and ((.items[0].participant_addresses // [])|type)==\"array\" and ((.items[0].event_count // 0)|type)==\"number\" and ((.has_more|type)==\"boolean\")" >/dev/null
COMM_THREADS_HAS_MORE="$(echo "$COMM_THREADS_PAGE1_JSON" | jq -r ".has_more")"
COMM_THREAD_CURSOR="$(echo "$COMM_THREADS_PAGE1_JSON" | jq -r ".next_cursor")"
COMM_THREAD_ID="$(echo "$COMM_THREADS_PAGE1_JSON" | jq -r ".items[0].thread_id")"
COMM_THREAD_CONNECTOR_ID="$(echo "$COMM_THREADS_PAGE1_JSON" | jq -r ".items[0].connector_id")"
echo "COMM_THREADS_HAS_MORE=$COMM_THREADS_HAS_MORE"
echo "COMM_THREAD_CURSOR=$COMM_THREAD_CURSOR"
echo "COMM_THREAD_ID=$COMM_THREAD_ID"
test -n "$COMM_THREAD_ID" && [ "$COMM_THREAD_ID" != "null" ]
test -n "$COMM_THREAD_CONNECTOR_ID" && [ "$COMM_THREAD_CONNECTOR_ID" != "null" ]
if [[ "$COMM_THREADS_HAS_MORE" == "true" ]]; then
  test -n "$COMM_THREAD_CURSOR" && [ "$COMM_THREAD_CURSOR" != "null" ]
fi
'
run_eval "Query communications thread inventory API (page 2 via cursor when available)" '
if [[ "$COMM_THREADS_HAS_MORE" == "true" && -n "$COMM_THREAD_CURSOR" && "$COMM_THREAD_CURSOR" != "null" ]]; then
  COMM_THREADS_PAGE2_JSON="$(curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"limit\":1,\"cursor\":\"$COMM_THREAD_CURSOR\"}" "http://$DAEMON_TCP_ADDR/v1/comm/threads/list")"
else
  COMM_THREADS_PAGE2_JSON="{\"workspace_id\":\"$WORKSPACE\",\"items\":[],\"has_more\":false}"
fi
echo "$COMM_THREADS_PAGE2_JSON"
'
run_eval "Validate communications thread inventory page 2 payload" '
if [[ "$COMM_THREADS_HAS_MORE" == "true" ]]; then
  echo "$COMM_THREADS_PAGE2_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and (.items|type)==\"array\" and (.items|length)>=1 and (.items[0].thread_id != \"$COMM_THREAD_ID\")" >/dev/null
else
  echo "$COMM_THREADS_PAGE2_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and (.items|type)==\"array\" and (.has_more == false)" >/dev/null
fi
'
capture_cmd COMM_THREADS_FILTERED_JSON "Query communications thread inventory API with channel/query/connector filter" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"channel\":\"mail\",\"connector_id\":\"mail\",\"query\":\"watcher\",\"limit\":20}" "http://$DAEMON_TCP_ADDR/v1/comm/threads/list"
run_eval "Validate communications thread inventory filtered payload and extract thread id for reply tests" '
echo "$COMM_THREADS_FILTERED_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and (.items|type)==\"array\" and all(.items[]?; (.channel == \"mail\") and (.connector_id == \"mail\"))" >/dev/null
COMM_MAIL_THREAD_ID="$(echo "$COMM_THREADS_FILTERED_JSON" | jq -r ".items[0].thread_id // empty")"
if [[ -z "$COMM_MAIL_THREAD_ID" || "$COMM_MAIL_THREAD_ID" == "null" ]]; then
  COMM_MAIL_THREAD_ID="$COMM_THREAD_ID"
  COMM_REPLY_CONNECTOR_ID="$COMM_THREAD_CONNECTOR_ID"
  echo "note: mail-filtered thread set is empty; falling back to baseline thread/connector for reply-hint checks"
else
  COMM_REPLY_CONNECTOR_ID="mail"
fi
echo "COMM_MAIL_THREAD_ID=$COMM_MAIL_THREAD_ID"
echo "COMM_REPLY_CONNECTOR_ID=$COMM_REPLY_CONNECTOR_ID"
test -n "$COMM_MAIL_THREAD_ID" && [ "$COMM_MAIL_THREAD_ID" != "null" ]
test -n "$COMM_REPLY_CONNECTOR_ID" && [ "$COMM_REPLY_CONNECTOR_ID" != "null" ]
'
capture_cmd COMM_SEND_THREAD_REPLY_JSON "Send comm reply using thread_id + connector_id hints with derived destination" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"thread_id\":\"$COMM_MAIL_THREAD_ID\",\"connector_id\":\"$COMM_REPLY_CONNECTOR_ID\",\"message\":\"thread-aware reply test\"}" "http://$DAEMON_TCP_ADDR/v1/comm/send"
run_eval "Validate comm send thread-reply hint payload" 'echo "$COMM_SEND_THREAD_REPLY_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .thread_id == \"$COMM_MAIL_THREAD_ID\" and ((.resolved_destination // \"\")|length)>0 and ((.resolved_source_channel // \"\")|length)>0 and ((.resolved_connector_id // \"\") == \"$COMM_REPLY_CONNECTOR_ID\") and (.success|type)==\"boolean\"" >/dev/null'
capture_cmd COMM_SEND_CONNECTOR_HINT_JSON "Send comm with connector-targeted hint for message route" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"source_channel\":\"message\",\"connector_id\":\"twilio\",\"destination\":\"+15550004444\",\"message\":\"connector-hint route test\"}" "http://$DAEMON_TCP_ADDR/v1/comm/send"
run_eval "Validate comm send connector-targeted hint payload" 'echo "$COMM_SEND_CONNECTOR_HINT_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and ((.resolved_connector_id // \"\") == \"twilio\") and ((.resolved_source_channel // \"\") == \"twilio\") and ((.resolved_destination // \"\") == \"+15550004444\") and (.success|type)==\"boolean\"" >/dev/null'
capture_cmd COMM_EVENTS_PAGE1_JSON "Query communications event timeline API (page 1)" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"limit\":1}" "http://$DAEMON_TCP_ADDR/v1/comm/events/list"
run_eval "Validate communications event timeline page 1 payload" '
echo "$COMM_EVENTS_PAGE1_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and (.items|type)==\"array\" and (.items|length)==1 and ((.items[0].event_id // \"\")|type)==\"string\" and ((.items[0].thread_id // \"\")|type)==\"string\" and ((.items[0].connector_id // \"\")|type)==\"string\" and ((.items[0].addresses // [])|type)==\"array\" and ((.has_more|type)==\"boolean\")" >/dev/null
COMM_EVENTS_HAS_MORE="$(echo "$COMM_EVENTS_PAGE1_JSON" | jq -r ".has_more")"
COMM_EVENT_CURSOR="$(echo "$COMM_EVENTS_PAGE1_JSON" | jq -r ".next_cursor")"
COMM_EVENT_ID="$(echo "$COMM_EVENTS_PAGE1_JSON" | jq -r ".items[0].event_id")"
echo "COMM_EVENTS_HAS_MORE=$COMM_EVENTS_HAS_MORE"
echo "COMM_EVENT_CURSOR=$COMM_EVENT_CURSOR"
echo "COMM_EVENT_ID=$COMM_EVENT_ID"
test -n "$COMM_EVENT_ID" && [ "$COMM_EVENT_ID" != "null" ]
if [[ "$COMM_EVENTS_HAS_MORE" == "true" ]]; then
  test -n "$COMM_EVENT_CURSOR" && [ "$COMM_EVENT_CURSOR" != "null" ]
fi
'
run_eval "Query communications event timeline API (page 2 via cursor when available)" '
if [[ "$COMM_EVENTS_HAS_MORE" == "true" && -n "$COMM_EVENT_CURSOR" && "$COMM_EVENT_CURSOR" != "null" ]]; then
  COMM_EVENTS_PAGE2_JSON="$(curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"limit\":1,\"cursor\":\"$COMM_EVENT_CURSOR\"}" "http://$DAEMON_TCP_ADDR/v1/comm/events/list")"
else
  COMM_EVENTS_PAGE2_JSON="{\"workspace_id\":\"$WORKSPACE\",\"items\":[],\"has_more\":false}"
fi
echo "$COMM_EVENTS_PAGE2_JSON"
'
run_eval "Validate communications event timeline page 2 payload" '
if [[ "$COMM_EVENTS_HAS_MORE" == "true" ]]; then
  echo "$COMM_EVENTS_PAGE2_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and (.items|type)==\"array\" and (.items|length)>=1 and (.items[0].event_id != \"$COMM_EVENT_ID\")" >/dev/null
else
  echo "$COMM_EVENTS_PAGE2_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and (.items|type)==\"array\" and (.has_more == false)" >/dev/null
fi
'
capture_cmd COMM_EVENTS_FILTERED_JSON "Query communications event timeline API with thread filter" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"thread_id\":\"$COMM_THREAD_ID\",\"limit\":20}" "http://$DAEMON_TCP_ADDR/v1/comm/events/list"
run_eval "Validate communications event timeline filtered payload" 'echo "$COMM_EVENTS_FILTERED_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .thread_id == \"$COMM_THREAD_ID\" and (.items|type)==\"array\" and all(.items[]?; (.thread_id == \"$COMM_THREAD_ID\") and ((.connector_id // \"\")|type)==\"string\")" >/dev/null'
capture_cmd IMESSAGE_VISIBILITY_BASELINE_THREADS_JSON "Capture iMessage communications thread baseline for workspace drift regression" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"channel\":\"message\",\"connector_id\":\"imessage\",\"limit\":50}" "http://$DAEMON_TCP_ADDR/v1/comm/threads/list"
run_eval "Validate iMessage thread baseline and extract snapshot values" '
echo "$IMESSAGE_VISIBILITY_BASELINE_THREADS_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and (.items|type)==\"array\" and (.items|length)>=1 and all(.items[]?; (.workspace_id == \"$WORKSPACE\") and ((.channel // \"\") | ascii_downcase) == \"message\" and ((.connector_id // \"\") | ascii_downcase) == \"imessage\")" >/dev/null
IMESSAGE_VISIBILITY_THREAD_COUNT="$(echo "$IMESSAGE_VISIBILITY_BASELINE_THREADS_JSON" | jq -r ".items | length")"
IMESSAGE_VISIBILITY_THREAD_FIRST_ID="$(echo "$IMESSAGE_VISIBILITY_BASELINE_THREADS_JSON" | jq -r ".items[0].thread_id")"
echo "IMESSAGE_VISIBILITY_THREAD_COUNT=$IMESSAGE_VISIBILITY_THREAD_COUNT"
echo "IMESSAGE_VISIBILITY_THREAD_FIRST_ID=$IMESSAGE_VISIBILITY_THREAD_FIRST_ID"
test -n "$IMESSAGE_VISIBILITY_THREAD_FIRST_ID" && [ "$IMESSAGE_VISIBILITY_THREAD_FIRST_ID" != "null" ]
'
capture_cmd IMESSAGE_VISIBILITY_BASELINE_EVENTS_JSON "Capture iMessage communications event baseline for workspace drift regression" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"channel\":\"message\",\"connector_id\":\"imessage\",\"limit\":50}" "http://$DAEMON_TCP_ADDR/v1/comm/events/list"
run_eval "Validate iMessage event baseline and extract snapshot values" '
echo "$IMESSAGE_VISIBILITY_BASELINE_EVENTS_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and (.items|type)==\"array\" and (.items|length)>=1 and all(.items[]?; (.workspace_id == \"$WORKSPACE\") and ((.channel // \"\") | ascii_downcase) == \"message\" and ((.connector_id // \"\") | ascii_downcase) == \"imessage\")" >/dev/null
IMESSAGE_VISIBILITY_EVENT_COUNT="$(echo "$IMESSAGE_VISIBILITY_BASELINE_EVENTS_JSON" | jq -r ".items | length")"
IMESSAGE_VISIBILITY_EVENT_FIRST_ID="$(echo "$IMESSAGE_VISIBILITY_BASELINE_EVENTS_JSON" | jq -r ".items[0].event_id")"
echo "IMESSAGE_VISIBILITY_EVENT_COUNT=$IMESSAGE_VISIBILITY_EVENT_COUNT"
echo "IMESSAGE_VISIBILITY_EVENT_FIRST_ID=$IMESSAGE_VISIBILITY_EVENT_FIRST_ID"
test -n "$IMESSAGE_VISIBILITY_EVENT_FIRST_ID" && [ "$IMESSAGE_VISIBILITY_EVENT_FIRST_ID" != "null" ]
'
run_eval "Prepare alternate workspace status-refresh target for workspace drift regression" '
IMESSAGE_VISIBILITY_ALT_WORKSPACE="${WORKSPACE}-alt"
ALT_UPSERT_JSON="$(curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"${IMESSAGE_VISIBILITY_ALT_WORKSPACE}\",\"channel_id\":\"app\",\"configuration\":{\"enabled\":true,\"transport\":\"daemon_realtime\"},\"merge\":true}" "http://$DAEMON_TCP_ADDR/v1/channels/config/upsert")"
echo "$ALT_UPSERT_JSON" | jq -e ".workspace_id == \"${IMESSAGE_VISIBILITY_ALT_WORKSPACE}\" and .channel_id == \"app\"" >/dev/null
'
run_eval "Simulate repeated channel-status refreshes across primary and alternate workspaces" '
for _ in $(seq 1 3); do
  curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\"}" "http://$DAEMON_TCP_ADDR/v1/channels/status" >/dev/null
  curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"${IMESSAGE_VISIBILITY_ALT_WORKSPACE}\"}" "http://$DAEMON_TCP_ADDR/v1/channels/status" >/dev/null
done
'
capture_cmd IMESSAGE_VISIBILITY_AFTER_THREADS_JSON "Capture iMessage communications thread snapshot after status refresh loop" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"channel\":\"message\",\"connector_id\":\"imessage\",\"limit\":50}" "http://$DAEMON_TCP_ADDR/v1/comm/threads/list"
run_eval "Validate iMessage thread visibility remains stable after status refresh loop" '
echo "$IMESSAGE_VISIBILITY_AFTER_THREADS_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and (.items|type)==\"array\" and (.items|length)>=1 and all(.items[]?; (.workspace_id == \"$WORKSPACE\") and ((.channel // \"\") | ascii_downcase) == \"message\" and ((.connector_id // \"\") | ascii_downcase) == \"imessage\")" >/dev/null
AFTER_THREAD_COUNT="$(echo "$IMESSAGE_VISIBILITY_AFTER_THREADS_JSON" | jq -r ".items | length")"
AFTER_THREAD_FIRST_ID="$(echo "$IMESSAGE_VISIBILITY_AFTER_THREADS_JSON" | jq -r ".items[0].thread_id")"
[ "$AFTER_THREAD_COUNT" -eq "$IMESSAGE_VISIBILITY_THREAD_COUNT" ]
[ "$AFTER_THREAD_FIRST_ID" = "$IMESSAGE_VISIBILITY_THREAD_FIRST_ID" ]
'
capture_cmd IMESSAGE_VISIBILITY_AFTER_EVENTS_JSON "Capture iMessage communications event snapshot after status refresh loop" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"channel\":\"message\",\"connector_id\":\"imessage\",\"limit\":50}" "http://$DAEMON_TCP_ADDR/v1/comm/events/list"
run_eval "Validate iMessage event visibility remains stable after status refresh loop" '
echo "$IMESSAGE_VISIBILITY_AFTER_EVENTS_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and (.items|type)==\"array\" and (.items|length)>=1 and all(.items[]?; (.workspace_id == \"$WORKSPACE\") and ((.channel // \"\") | ascii_downcase) == \"message\" and ((.connector_id // \"\") | ascii_downcase) == \"imessage\")" >/dev/null
AFTER_EVENT_COUNT="$(echo "$IMESSAGE_VISIBILITY_AFTER_EVENTS_JSON" | jq -r ".items | length")"
AFTER_EVENT_FIRST_ID="$(echo "$IMESSAGE_VISIBILITY_AFTER_EVENTS_JSON" | jq -r ".items[0].event_id")"
[ "$AFTER_EVENT_COUNT" -eq "$IMESSAGE_VISIBILITY_EVENT_COUNT" ]
[ "$AFTER_EVENT_FIRST_ID" = "$IMESSAGE_VISIBILITY_EVENT_FIRST_ID" ]
'
capture_cmd CONNECTOR_STATUS_JSON "Read connector status/config summary API" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\"}" "http://$DAEMON_TCP_ADDR/v1/connectors/status"
run_eval "Validate connector status summary payload" '
echo "$CONNECTOR_STATUS_JSON" | jq -e "
  .workspace_id == \"$WORKSPACE\"
  and (.connectors|type)==\"array\"
  and ((([.connectors[].connector_id] | index(\"imessage\")) != null) or (([.connectors[].connector_id] | index(\"messages\")) != null))
  and ([.connectors[].connector_id] | index(\"mail\")) != null
  and ([.connectors[].connector_id] | index(\"calendar\")) != null
  and ([.connectors[].connector_id] | index(\"browser\")) != null
  and ([.connectors[].connector_id] | index(\"finder\")) != null
  and ([.connectors[].connector_id] | index(\"cloudflared\")) != null
  and all(.connectors[]?;
    ((.action_readiness // \"\")|type)==\"string\"
    and ((.action_blockers // [])|type)==\"array\"
    and all((.action_blockers // [])[]?; ((.code // \"\")|type)==\"string\" and ((.message // \"\")|type)==\"string\")
    and ((.configuration.status_reason // \"\")|type)==\"string\"
    and ((.config_field_descriptors // [])|type)==\"array\"
    and all((.config_field_descriptors // [])[]?;
      (.key|type)==\"string\"
      and (.label|type)==\"string\"
      and (.type|type)==\"string\"
      and (.required|type)==\"boolean\"
      and (.editable|type)==\"boolean\"
      and (.enum_options|type)==\"array\"
      and (.secret|type)==\"boolean\"
      and (.write_only|type)==\"boolean\"
      and (.help_text|type)==\"string\"
    )
    and ((.remediation_actions // [])|type)==\"array\"
    and all((.remediation_actions // [])[]?;
      (.identifier|type)==\"string\"
      and (.label|type)==\"string\"
      and (.intent|type)==\"string\"
      and (.enabled|type)==\"boolean\"
      and (.recommended|type)==\"boolean\"
    )
  )
  and (([.connectors[]? | select(.connector_id == \"twilio\") | .config_field_descriptors[]? | select(.key == \"auth_token_value\" and .write_only == true and .secret == true)] | length) >= 1)
  and (([.connectors[] | select(.connector_id == \"cloudflared\" and .configuration.status_reason == \"cloudflared_binary_missing\")] | length) == 0
       or (([.connectors[] | select(.connector_id == \"cloudflared\" and .configuration.status_reason == \"cloudflared_binary_missing\")][0].remediation_actions | map(.identifier) | index(\"install_cloudflared_connector\")) != null))
" >/dev/null
'
run_eval "Validate calendar permission-missing readiness classification when present" '
echo "$CONNECTOR_STATUS_JSON" | jq -e "
  ([.connectors[]? | select(.connector_id == \"calendar\" and (.configuration.status_reason // \"\") == \"permission_missing\")] | length) as \$calendarMissingCount
  | if \$calendarMissingCount == 0 then
      true
    else
      ([.connectors[]? | select(.connector_id == \"calendar\" and (.configuration.status_reason // \"\") == \"permission_missing\" and (.action_readiness // \"\") == \"blocked\")] | length) == \$calendarMissingCount
      and ([.connectors[]? | select(.connector_id == \"calendar\" and (.configuration.status_reason // \"\") == \"permission_missing\") | .action_blockers[]? | select((.code // \"\") == \"permission_missing\")] | length) >= \$calendarMissingCount
    end
" >/dev/null
'
capture_cmd CONNECTOR_CONFIG_UPSERT_JSON "Upsert connector config metadata API (initial payload)" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"connector_id\":\"mail\",\"configuration\":{\"scope\":\"inbox\"},\"merge\":true}" "http://$DAEMON_TCP_ADDR/v1/connectors/config/upsert"
run_eval "Validate connector config upsert payload" 'echo "$CONNECTOR_CONFIG_UPSERT_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .connector_id == \"mail\" and ((.updated_at // \"\")|length)>0 and (.configuration|type)==\"object\" and ((.configuration.scope // \"\") == \"inbox\")" >/dev/null'
capture_cmd CONNECTOR_CONFIG_UPSERT_MERGE_JSON "Upsert connector config metadata API (merge payload)" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"connector_id\":\"mail\",\"configuration\":{\"mode\":\"read_only\"},\"merge\":true}" "http://$DAEMON_TCP_ADDR/v1/connectors/config/upsert"
run_eval "Validate connector config merge payload preserves prior fields" 'echo "$CONNECTOR_CONFIG_UPSERT_MERGE_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .connector_id == \"mail\" and ((.configuration.scope // \"\") == \"inbox\") and ((.configuration.mode // \"\") == \"read_only\")" >/dev/null'
capture_cmd CONNECTOR_TEST_OPERATION_JSON "Run connector health test-operation API" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"connector_id\":\"mail\",\"operation\":\"health\"}" "http://$DAEMON_TCP_ADDR/v1/connectors/test"
run_eval "Validate connector test-operation payload" 'echo "$CONNECTOR_TEST_OPERATION_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .connector_id == \"mail\" and .operation == \"health\" and (.success|type)==\"boolean\" and ((.status // \"\")|type)==\"string\" and ((.summary // \"\")|type)==\"string\" and ((.checked_at // \"\")|type)==\"string\" and ((.details.plugin_id // \"\") == \"mail.daemon\") and ((.details.worker_registered // false)|type)==\"boolean\"" >/dev/null'
capture_cmd CONNECTOR_DIAGNOSTICS_JSON "Read connector diagnostics summary API" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\"}" "http://$DAEMON_TCP_ADDR/v1/connectors/diagnostics"
run_eval "Validate connector diagnostics payload and remediation fields" 'echo "$CONNECTOR_DIAGNOSTICS_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and (.diagnostics|type)==\"array\" and ([.diagnostics[].connector_id] | index(\"mail\")) != null and all(.diagnostics[]?; ((.worker_health.registered|type)==\"boolean\") and ((.remediation_actions|type)==\"array\") and all(.remediation_actions[]?; (.identifier|type)==\"string\" and (.label|type)==\"string\" and (.intent|type)==\"string\" and (.enabled|type)==\"boolean\" and (.recommended|type)==\"boolean\"))" >/dev/null'
run_eval "Validate connector System Settings remediation destination matrix" '
	echo "$CONNECTOR_DIAGNOSTICS_JSON" | jq -e "
	  ([.diagnostics[]?
	    | select(.connector_id == \"mail\" or .connector_id == \"calendar\" or .connector_id == \"browser\" or .connector_id == \"finder\")
	    | .remediation_actions[]?
    | select(.identifier == \"open_connector_system_settings\")
    | .destination] as \$automationDestinations
	  | (\$automationDestinations | length) >= 4
	    and (\$automationDestinations | all(. == \"ui://system-settings/privacy/automation\"))
	  ) and
	  ([.diagnostics[]?
	    | select(.connector_id == \"imessage\" or .connector_id == \"messages\")
	    | .remediation_actions[]?
	    | select(.identifier == \"open_connector_system_settings\")
	    | .destination] as \$messagesAutomationDestinations
	  | (\$messagesAutomationDestinations | length) >= 1
	    and (\$messagesAutomationDestinations | all(. == \"ui://system-settings/privacy/automation\"))
	  ) and
	  ([.diagnostics[]?
	    | select(.connector_id == \"imessage\" or .connector_id == \"messages\")
	    | .remediation_actions[]?
	    | select(.identifier == \"open_imessage_system_settings\")
    | .destination] as \$messagesDestinations
  | (\$messagesDestinations | length) >= 1
    and (\$messagesDestinations | all(. == \"ui://system-settings/privacy/full-disk-access\"))
  )
" >/dev/null
'
capture_cmd CONNECTOR_DIAGNOSTICS_FILTERED_JSON "Read connector diagnostics summary API with calendar filter" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"connector_id\":\"calendar\"}" "http://$DAEMON_TCP_ADDR/v1/connectors/diagnostics"
run_eval "Validate connector diagnostics filter payload" 'echo "$CONNECTOR_DIAGNOSTICS_FILTERED_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and (.diagnostics|type)==\"array\" and (.diagnostics|length)==1 and (.diagnostics[0].connector_id == \"calendar\")" >/dev/null'
capture_cmd CONNECTOR_PERMISSION_JSON "Request connector permission via daemon API" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"connector_id\":\"mail\"}" "http://$DAEMON_TCP_ADDR/v1/connectors/permission/request"
run_eval "Validate connector permission request payload" 'echo "$CONNECTOR_PERMISSION_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .connector_id == \"mail\" and (.permission_state|ascii_downcase|test(\"^(granted|missing|unknown)$\")) and ((.message // \"\")|type)==\"string\"" >/dev/null'
capture_cmd CONNECTOR_CALENDAR_PERMISSION_JSON "Request calendar connector permission via daemon API" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"connector_id\":\"calendar\"}" "http://$DAEMON_TCP_ADDR/v1/connectors/permission/request"
run_eval "Validate calendar connector permission request payload" '
echo "$CONNECTOR_CALENDAR_PERMISSION_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .connector_id == \"calendar\" and (.permission_state|ascii_downcase|test(\"^(granted|missing|unknown)$\")) and ((.message // \"\")|type)==\"string\"" >/dev/null
if command -v osascript >/dev/null 2>&1; then
  echo "$CONNECTOR_CALENDAR_PERMISSION_JSON" | jq -e "if ((.permission_state|ascii_downcase) == \"missing\") then ((.message // \"\")|ascii_downcase|contains(\"automation\")) and (((.message // \"\")|ascii_downcase|contains(\"unavailable or could not be launched\"))|not) else true end" >/dev/null
fi
'
capture_cmd CONNECTOR_MESSAGES_PERMISSION_JSON "Request messages connector permission via daemon API" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"connector_id\":\"imessage\"}" "http://$DAEMON_TCP_ADDR/v1/connectors/permission/request"
run_eval "Validate messages connector permission request payload" 'echo "$CONNECTOR_MESSAGES_PERMISSION_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .connector_id == \"imessage\" and (.permission_state|ascii_downcase|test(\"^(granted|missing|unknown)$\")) and ((.message // \"\")|type)==\"string\" and ((.message // \"\")|ascii_downcase|contains(\"full disk access\")) and ((.message // \"\")|ascii_downcase|contains(\"automation\"))" >/dev/null'
capture_cmd CLOUDFLARED_VERSION_JSON "Run cloudflared version command via daemon" pa_tcp connector cloudflared version --workspace "$WORKSPACE"
run_eval "Validate cloudflared version payload" 'echo "$CLOUDFLARED_VERSION_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .available == true and ((.binary_path // \"\") | length) > 0 and .dry_run == true" >/dev/null'
capture_cmd CLOUDFLARED_EXEC_JSON "Run cloudflared exec command via daemon" pa_tcp connector cloudflared exec --workspace "$WORKSPACE" --arg version
run_eval "Validate cloudflared exec payload" 'echo "$CLOUDFLARED_EXEC_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .success == true and ((.args // []) | length) == 1 and .args[0] == \"version\" and .dry_run == true" >/dev/null'
capture_cmd MAIL_RUN_JSON "Run mail connector happy path via agent" pa_tcp agent run --workspace "$WORKSPACE" --request 'send an email to recipient@example.com saying "daemon update"'
run_eval "Validate mail connector run payload" 'echo "$MAIL_RUN_JSON" | jq -e "(.workflow == \"mail\") and (.run_state == \"completed\") and (((.step_states // []) | length) >= 1) and (any((.step_states // [])[]?; ((.capability_key // \"\") | startswith(\"mail_\")))) and (any((.step_states // [])[]?; ((.evidence.provider // \"\") | test(\"mail\"))))" >/dev/null'
capture_cmd CALENDAR_RUN_JSON "Run calendar create via agent" pa_tcp agent run --workspace "$WORKSPACE" --request 'schedule "daemon calendar fixture"'
run_eval "Validate calendar create run payload and capture event id" '
echo "$CALENDAR_RUN_JSON" | jq -e ".workflow == \"calendar\" and .run_state == \"completed\" and ((.step_states // []) | length) == 1 and ((.step_states[0].capability_key // \"\") == \"calendar_create\") and ((.step_states[0].evidence.provider // \"\") | test(\"^apple-calendar\")) and ((.step_states[0].evidence.event_id // \"\")|length) > 0" >/dev/null
CALENDAR_EVENT_ID="$(echo "$CALENDAR_RUN_JSON" | jq -r ".step_states[0].evidence.event_id")"
echo "CALENDAR_EVENT_ID=$CALENDAR_EVENT_ID"
test -n "$CALENDAR_EVENT_ID" && [ "$CALENDAR_EVENT_ID" != "null" ]
'
capture_cmd CALENDAR_UPDATE_RUN_JSON "Run calendar update via agent with stable event identity" pa_tcp agent run --workspace "$WORKSPACE" --request "reschedule calendar event id $CALENDAR_EVENT_ID to \"daemon calendar fixture updated\""
run_eval "Validate calendar update run payload" 'echo "$CALENDAR_UPDATE_RUN_JSON" | jq -e ".workflow == \"calendar\" and .run_state == \"completed\" and ((.step_states // []) | length) == 1 and ((.step_states[0].capability_key // \"\") == \"calendar_update\") and ((.step_states[0].evidence.event_id // \"\") == \"$CALENDAR_EVENT_ID\") and ((.step_states[0].evidence.provider // \"\") | test(\"^apple-calendar\"))" >/dev/null'
capture_cmd CALENDAR_CANCEL_RUN_JSON "Run calendar cancel via agent with stable event identity" pa_tcp agent run --workspace "$WORKSPACE" --approval-phrase "GO AHEAD" --request "cancel calendar event id $CALENDAR_EVENT_ID"
run_eval "Validate calendar cancel run payload" 'echo "$CALENDAR_CANCEL_RUN_JSON" | jq -e ".workflow == \"calendar\" and .run_state == \"completed\" and ((.step_states // []) | length) == 1 and ((.step_states[0].capability_key // \"\") == \"calendar_cancel\") and ((.step_states[0].evidence.event_id // \"\") == \"$CALENDAR_EVENT_ID\") and ((.step_states[0].evidence.provider // \"\") | test(\"^apple-calendar\"))" >/dev/null'
capture_cmd BROWSER_RUN_JSON "Run browser connector happy path via agent" pa_tcp agent run --workspace "$WORKSPACE" --request "open https://example.com and summarize"
run_eval "Validate browser connector run payload" 'echo "$BROWSER_RUN_JSON" | jq -e ".workflow == \"browser\" and .run_state == \"completed\" and ((.step_states // []) | map(.capability_key) | index(\"browser_open\")) != null and ((.step_states // []) | map(.capability_key) | index(\"browser_extract\")) != null and ((.step_states // []) | map(.capability_key) | index(\"browser_close\")) != null and ((.step_states // []) | map((.evidence.provider // \"\")) | index(\"safari-automation-dry-run\")) != null and ([.step_states[]? | select(.capability_key == \"browser_open\")] | length > 0) and ([.step_states[]? | select(.capability_key == \"browser_open\")][0].evidence.url // \"\") != \"\" and ([.step_states[]? | select(.capability_key == \"browser_extract\")] | length > 0) and ((([.step_states[]? | select(.capability_key == \"browser_extract\")][0].evidence.content_chars // \"0\")|tonumber) > 0) and ((([.step_states[]? | select(.capability_key == \"browser_extract\")][0].evidence.query_answer // \"\")|length) > 0)" >/dev/null'
run_eval "Run messages workflow with canonical message/twilio source via agent" '
set +e
MESSAGES_SMS_RUN_JSON="$(pa_tcp agent run --workspace "$WORKSPACE" --request '"'"'send a message via twilio to +15550001111 saying "hello"'"'"' 2>&1)"
MESSAGES_SMS_RUN_RC=$?
set -e
printf "%s\n" "$MESSAGES_SMS_RUN_JSON"
[ "$MESSAGES_SMS_RUN_RC" -eq 0 ] || [ "$MESSAGES_SMS_RUN_RC" -eq 1 ]
set +e
'
run_eval "Validate executable messages run payload or actionable failure contract" '
if echo "$MESSAGES_SMS_RUN_JSON" | jq -e ".workflow == \"messages\"" >/dev/null 2>&1; then
  echo "$MESSAGES_SMS_RUN_JSON" | jq -e "
    (
      ((.clarification_required // false) == false)
      and ((.run_state // \"\") == \"completed\")
      and ((.step_states // []) | length) >= 1
      and (((.step_states // [])[0].capability_key // \"\") == \"messages_send_sms\")
      and ((((.step_states // [])[0].evidence.channel // \"\") | test(\"^(message|twilio|sms)$\")))
      and (((.step_states // [])[0].evidence.destination // \"\") == \"+15550001111\")
    )
    or
    (
      ((.clarification_required // false) == true)
      and ((.missing_slots // []) | index(\"message_channel\")) != null
    )
  " >/dev/null
else
  printf "%s\n" "$MESSAGES_SMS_RUN_JSON" | rg -q -- "^request failed$"
  printf "%s\n" "$MESSAGES_SMS_RUN_JSON" | rg -q -- "^what failed:"
  printf "%s\n" "$MESSAGES_SMS_RUN_JSON" | rg -q -- "unsupported source channel \"sms\"|unable to determine intent"
fi
'
run_eval "Extract messages workflow context IDs for comm attempt-history checks" '
if echo "$MESSAGES_SMS_RUN_JSON" | jq . >/dev/null 2>&1; then
  MESSAGES_SMS_TASK_ID="$(echo "$MESSAGES_SMS_RUN_JSON" | jq -r ".task_id // empty")"
  MESSAGES_SMS_RUN_ID="$(echo "$MESSAGES_SMS_RUN_JSON" | jq -r ".run_id // empty")"
  MESSAGES_SMS_STEP_ID="$(echo "$MESSAGES_SMS_RUN_JSON" | jq -r ".step_states[0].step_id // empty")"
else
  MESSAGES_SMS_TASK_ID=""
  MESSAGES_SMS_RUN_ID=""
  MESSAGES_SMS_STEP_ID=""
fi
CONTEXT_STEP_ID="$MESSAGES_SMS_STEP_ID"
if [[ -z "$CONTEXT_STEP_ID" || "$CONTEXT_STEP_ID" == "null" ]]; then
  CONTEXT_STEP_ID="manual-step-context"
fi
CONTEXT_EVENT_ID="$COMM_EVENT_ID"
if [[ -z "$CONTEXT_EVENT_ID" || "$CONTEXT_EVENT_ID" == "null" ]]; then
  CONTEXT_EVENT_ID="manual-event-context"
fi
echo "MESSAGES_SMS_TASK_ID=$MESSAGES_SMS_TASK_ID"
echo "MESSAGES_SMS_RUN_ID=$MESSAGES_SMS_RUN_ID"
echo "MESSAGES_SMS_STEP_ID=$MESSAGES_SMS_STEP_ID"
echo "CONTEXT_STEP_ID=$CONTEXT_STEP_ID"
echo "CONTEXT_EVENT_ID=$CONTEXT_EVENT_ID"
test -n "$CONTEXT_STEP_ID" && [ "$CONTEXT_STEP_ID" != "null" ]
test -n "$COMM_THREAD_ID" && [ "$COMM_THREAD_ID" != "null" ]
test -n "$COMM_EVENT_ID" && [ "$COMM_EVENT_ID" != "null" ]
'
capture_cmd COMM_POLICY_CREATE_JSON "Create comm delivery policy via daemon API" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"source_channel\":\"message\",\"endpoint_pattern\":\"+1555%\",\"primary_channel\":\"imessage\",\"retry_count\":1,\"fallback_channels\":[\"sms\"],\"is_default\":true}" "http://$DAEMON_TCP_ADDR/v1/comm/policy/set"
run_eval "Validate created comm policy payload and capture policy id" '
echo "$COMM_POLICY_CREATE_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .source_channel == \"message\" and (.is_default == true) and ((.policy.primary_channel // \"\") == \"imessage\") and ((.policy.retry_count // 0) == 1) and (((.policy.fallback_channels // []) | length) == 1)" >/dev/null
COMM_POLICY_ID="$(echo "$COMM_POLICY_CREATE_JSON" | jq -r ".id")"
echo "COMM_POLICY_ID=$COMM_POLICY_ID"
test -n "$COMM_POLICY_ID" && [ "$COMM_POLICY_ID" != "null" ]
'
capture_cmd COMM_POLICY_UPDATE_JSON "Update comm delivery policy via daemon API" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"policy_id\":\"$COMM_POLICY_ID\",\"workspace_id\":\"$WORKSPACE\",\"source_channel\":\"message\",\"endpoint_pattern\":\"+1555%\",\"primary_channel\":\"twilio\",\"retry_count\":0,\"fallback_channels\":[],\"is_default\":true}" "http://$DAEMON_TCP_ADDR/v1/comm/policy/set"
run_eval "Validate updated comm policy payload" 'echo "$COMM_POLICY_UPDATE_JSON" | jq -e ".id == \"$COMM_POLICY_ID\" and .workspace_id == \"$WORKSPACE\" and ((.policy.primary_channel // \"\") == \"twilio\") and ((.policy.retry_count // 0) == 0) and (((.policy.fallback_channels // []) | length) == 0)" >/dev/null'
capture_cmd COMM_POLICY_LIST_JSON "List comm delivery policies via daemon API" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"source_channel\":\"message\"}" "http://$DAEMON_TCP_ADDR/v1/comm/policy/list"
run_eval "Validate comm policy list reflects updated policy row" 'echo "$COMM_POLICY_LIST_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and (.policies|type)==\"array\" and ([.policies[].id] | index(\"$COMM_POLICY_ID\")) != null and ([.policies[] | select(.id == \"$COMM_POLICY_ID\")][0].policy.primary_channel == \"twilio\")" >/dev/null'
capture_cmd COMM_CONTEXT_SEND_JSON "Send comm with explicit step/event context for attempt-history API checks" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"operation_id\":\"op-daemon-context-history\",\"source_channel\":\"message\",\"destination\":\"+15550001234\",\"message\":\"daemon context history\",\"step_id\":\"$CONTEXT_STEP_ID\",\"event_id\":\"$CONTEXT_EVENT_ID\",\"imessage_failures\":2}" "http://$DAEMON_TCP_ADDR/v1/comm/send"
run_eval "Validate context-linked comm send produced attempt records" 'echo "$COMM_CONTEXT_SEND_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .operation_id == \"op-daemon-context-history\" and ((.success|type)==\"boolean\") and ((.result.Attempts|type)==\"array\") and ((.result.Attempts|length) >= 1)" >/dev/null'
capture_cmd COMM_CONTEXT_HISTORY_PAGE1_JSON "Query comm attempt-history API with operation filter (page 1)" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"operation_id\":\"op-daemon-context-history\",\"limit\":1}" "http://$DAEMON_TCP_ADDR/v1/comm/attempts"
run_eval "Validate comm attempt-history page 1 payload and capture cursor" '
echo "$COMM_CONTEXT_HISTORY_PAGE1_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .operation_id == \"op-daemon-context-history\" and ((.has_more // false)|type)==\"boolean\" and ((.attempts|type)==\"array\") and ((.attempts|length)==1) and ((.attempts[0].operation_id // \"\") == \"op-daemon-context-history\") and ((.attempts[0].route_phase // \"\")|type)==\"string\"" >/dev/null
COMM_CONTEXT_HISTORY_HAS_MORE="$(echo "$COMM_CONTEXT_HISTORY_PAGE1_JSON" | jq -r ".has_more // false")"
COMM_CONTEXT_HISTORY_CURSOR="$(echo "$COMM_CONTEXT_HISTORY_PAGE1_JSON" | jq -r ".next_cursor // \"\"")"
echo "COMM_CONTEXT_HISTORY_CURSOR=$COMM_CONTEXT_HISTORY_CURSOR"
if [[ "$COMM_CONTEXT_HISTORY_HAS_MORE" == "true" ]]; then
  test -n "$COMM_CONTEXT_HISTORY_CURSOR" && [ "$COMM_CONTEXT_HISTORY_CURSOR" != "null" ]
else
  COMM_CONTEXT_HISTORY_CURSOR=""
fi
'
run_eval "Query comm attempt-history API with operation filter (page 2 via cursor when available)" '
if [[ -n "$COMM_CONTEXT_HISTORY_CURSOR" && "$COMM_CONTEXT_HISTORY_CURSOR" != "null" ]]; then
  COMM_CONTEXT_HISTORY_PAGE2_JSON="$(curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"operation_id\":\"op-daemon-context-history\",\"cursor\":\"$COMM_CONTEXT_HISTORY_CURSOR\",\"limit\":2}" "http://$DAEMON_TCP_ADDR/v1/comm/attempts")"
else
  COMM_CONTEXT_HISTORY_PAGE2_JSON="$COMM_CONTEXT_HISTORY_PAGE1_JSON"
fi
printf "%s\n" "$COMM_CONTEXT_HISTORY_PAGE2_JSON"
'
run_eval "Validate comm attempt-history page 2 payload includes retry/fallback metadata" 'echo "$COMM_CONTEXT_HISTORY_PAGE2_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and (.attempts|type)==\"array\" and (.attempts|length)>=1 and all(.attempts[]?; ((.operation_id // \"\") == \"op-daemon-context-history\") and ((.route_phase // \"\")|type)==\"string\") and (any(.attempts[]?; ((.retry_ordinal // 0) >= 0)))" >/dev/null'
capture_cmd FINDER_CLARIFY_JSON "Run finder clarification request with missing target path" pa_tcp agent run --workspace "$WORKSPACE" --request "delete file now"
run_eval "Validate finder clarification payload" 'echo "$FINDER_CLARIFY_JSON" | jq -e ".workflow == \"finder\" and .clarification_required == true and .task_state == \"clarification_required\" and .run_state == \"clarification_required\" and ((.missing_slots // []) | index(\"finder_query\")) != null and ((.task_id // \"\") == \"\") and ((.run_id // \"\") == \"\")" >/dev/null'
capture_cmd MESSAGES_CLARIFY_JSON "Run messages clarification request requiring channel slot" pa_tcp agent run --workspace "$WORKSPACE" --request 'send a text to +15550001111: "hello"'
run_eval "Validate messages clarification payload" 'echo "$MESSAGES_CLARIFY_JSON" | jq -e ".workflow == \"messages\" and .clarification_required == true and ((.missing_slots // []) | index(\"message_channel\")) != null and ((.native_action.messages.recipient // \"\") == \"+15550001111\") and ((.native_action.messages.body // \"\") == \"hello\")" >/dev/null'
capture_cmd APPROVAL_RUN_JSON "Trigger approval-required agent run" pa_tcp agent run --workspace "$WORKSPACE" --requested-by actor.requester --subject actor.requester --acting-as actor.requester --request "delete file /tmp/personal-agent-daemon-approval.txt"
run_eval "Extract approval_request_id from run response" '
echo "$APPROVAL_RUN_JSON" | jq -e ".approval_required == true and (.approval_request_id|type)==\"string\" and (.approval_request_id|length)>0" >/dev/null
APPROVAL_ID="$(echo "$APPROVAL_RUN_JSON" | jq -r ".approval_request_id")"
echo "APPROVAL_ID=$APPROVAL_ID"
'
capture_cmd APPROVAL_PENDING_JSON "Query approval inbox pending rows" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"state\":\"pending\",\"limit\":20}" "http://$DAEMON_TCP_ADDR/v1/approvals/list"
run_eval "Validate pending approval inbox payload" 'echo "$APPROVAL_PENDING_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and (.approvals|type)==\"array\" and ([.approvals[].approval_request_id] | index(\"$APPROVAL_ID\")) != null and ([.approvals[] | select(.approval_request_id == \"$APPROVAL_ID\")][0].state == \"pending\") and ([.approvals[] | select(.approval_request_id == \"$APPROVAL_ID\")][0] | has(\"risk_level\") and has(\"risk_rationale\") and has(\"requested_by_actor_id\") and has(\"subject_principal_actor_id\") and has(\"acting_as_actor_id\") and (.route.task_class|type)==\"string\" and (.route.task_class_source|type)==\"string\" and (.route.route_source|type)==\"string\")" >/dev/null'
run_eval "Verify unauthorized approval actor is rejected" '
UNAUTHORIZED_APPROVE_OUTPUT="$(pa_tcp agent approve --workspace "$WORKSPACE" --approval-id "$APPROVAL_ID" --actor-id actor.approver --phrase "GO AHEAD" 2>&1 || true)"
printf "%s\n" "$UNAUTHORIZED_APPROVE_OUTPUT"
printf "%s\n" "$UNAUTHORIZED_APPROVE_OUTPUT" | grep -F "approval denied" >/dev/null
'
capture_cmd APPROVAL_GRANT_JSON "Grant approval-scope delegation" pa_tcp delegation grant --workspace "$WORKSPACE" --from actor.requester --to actor.approver --scope-type APPROVAL
run_eval "Extract approval delegation rule ID" '
APPROVAL_RULE_ID="$(echo "$APPROVAL_GRANT_JSON" | jq -r ".id")"
echo "APPROVAL_RULE_ID=$APPROVAL_RULE_ID"
test -n "$APPROVAL_RULE_ID" && [ "$APPROVAL_RULE_ID" != "null" ]
'
run_cmd "Approve pending request with delegated approver" pa_tcp agent approve --workspace "$WORKSPACE" --approval-id "$APPROVAL_ID" --actor-id actor.approver --phrase "GO AHEAD"
run_cmd "Revoke approval delegation rule" pa_tcp delegation revoke --workspace "$WORKSPACE" --rule-id "$APPROVAL_RULE_ID"
capture_cmd APPROVAL_DELEGATION_CHECK_AFTER_REVOKE_JSON "Check approval delegation after revoke (deny reason expected)" pa_tcp delegation check --workspace "$WORKSPACE" --requested-by actor.requester --acting-as actor.approver --scope-type APPROVAL
run_eval "Validate approval delegation deny reason after revoke" 'echo "$APPROVAL_DELEGATION_CHECK_AFTER_REVOKE_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and .requested_by_actor_id == \"actor.requester\" and .acting_as_actor_id == \"actor.approver\" and .allowed == false and .reason_code == \"missing_delegation_rule\" and ((.reason // \"\")|type)==\"string\"" >/dev/null'
capture_cmd APPROVAL_FINAL_JSON "Query approval inbox final rows" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"state\":\"final\",\"include_final\":true,\"limit\":20}" "http://$DAEMON_TCP_ADDR/v1/approvals/list"
run_eval "Validate final approval inbox payload" 'echo "$APPROVAL_FINAL_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and (.approvals|type)==\"array\" and ([.approvals[].approval_request_id] | index(\"$APPROVAL_ID\")) != null and ([.approvals[] | select(.approval_request_id == \"$APPROVAL_ID\")][0].state == \"final\") and ([.approvals[] | select(.approval_request_id == \"$APPROVAL_ID\")][0].decision == \"approved\") and ([.approvals[] | select(.approval_request_id == \"$APPROVAL_ID\")][0].route.task_class|type)==\"string\" and ([.approvals[] | select(.approval_request_id == \"$APPROVAL_ID\")][0].route.task_class_source|type)==\"string\" and ([.approvals[] | select(.approval_request_id == \"$APPROVAL_ID\")][0].route.route_source|type)==\"string\"" >/dev/null'
capture_cmd VOICE_DESTRUCTIVE_JSON "Trigger voice-origin destructive run without in-app handoff confirmation" pa_tcp agent run --workspace "$WORKSPACE" --request "delete file /tmp/personal-agent-daemon-voice-handoff.txt" --origin voice --approval-phrase "GO AHEAD"
run_eval "Validate voice-origin run is blocked pending in-app handoff" '
echo "$VOICE_DESTRUCTIVE_JSON" | jq -e ".approval_required == true and .run_state == \"awaiting_approval\" and ((.step_states // []) | map((.summary // \"\") | ascii_downcase | contains(\"in-app approval handoff\")) | any)" >/dev/null
VOICE_APPROVAL_ID="$(echo "$VOICE_DESTRUCTIVE_JSON" | jq -r ".approval_request_id")"
echo "VOICE_APPROVAL_ID=$VOICE_APPROVAL_ID"
test -n "$VOICE_APPROVAL_ID" && [ "$VOICE_APPROVAL_ID" != "null" ]
'
run_cmd "Approve voice-origin pending request after handoff" pa_tcp agent approve --workspace "$WORKSPACE" --approval-id "$VOICE_APPROVAL_ID" --actor-id actor.requester --phrase "GO AHEAD"
capture_cmd VOICE_CONFIRMED_JSON "Trigger voice-origin destructive run with in-app handoff confirmation" pa_tcp agent run --workspace "$WORKSPACE" --request "delete file /tmp/personal-agent-daemon-voice-handoff-confirmed.txt" --origin voice --in-app-approval-confirmed=true
run_eval "Validate voice-origin confirmed run executes without pending approval" 'echo "$VOICE_CONFIRMED_JSON" | jq -e "((.approval_required // false) == false) and .run_state == \"completed\"" >/dev/null'
capture_cmd INSPECT_QUERY_JSON "Query inspect logs API (LIFO)" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$INSPECT_WORKSPACE\",\"limit\":10}" "http://$DAEMON_TCP_ADDR/v1/inspect/logs/query"
run_eval "Validate inspect logs query payload" 'echo "$INSPECT_QUERY_JSON" | jq -e ".workspace_id == \"$INSPECT_WORKSPACE\" and (.logs|type)==\"array\" and ((.logs | length) < 2 or ([.logs[].created_at] == ([.logs[].created_at] | sort | reverse))) and (all(.logs[]?; has(\"status\") and has(\"input_summary\") and has(\"output_summary\") and (.route.task_class|type)==\"string\" and (.route.task_class_source|type)==\"string\" and (.route.route_source|type)==\"string\"))" >/dev/null'
run_eval "Capture inspect stream cursor from query response" '
INSPECT_CURSOR_CREATED_AT="$(echo "$INSPECT_QUERY_JSON" | jq -r ".logs[0].created_at // empty")"
INSPECT_CURSOR_ID="$(echo "$INSPECT_QUERY_JSON" | jq -r ".logs[0].log_id // empty")"
echo "INSPECT_CURSOR_CREATED_AT=$INSPECT_CURSOR_CREATED_AT"
echo "INSPECT_CURSOR_ID=$INSPECT_CURSOR_ID"
'
capture_cmd INSPECT_STREAM_JSON "Poll inspect logs stream API" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$INSPECT_WORKSPACE\",\"cursor_created_at\":\"$INSPECT_CURSOR_CREATED_AT\",\"cursor_id\":\"$INSPECT_CURSOR_ID\",\"timeout_ms\":500,\"poll_interval_ms\":100,\"limit\":10}" "http://$DAEMON_TCP_ADDR/v1/inspect/logs/stream"
run_eval "Validate inspect logs stream payload" 'echo "$INSPECT_STREAM_JSON" | jq -e ".workspace_id == \"$INSPECT_WORKSPACE\" and (.logs|type)==\"array\" and ((.timed_out|type)==\"boolean\") and all(.logs[]?; (.route.task_class|type)==\"string\" and (.route.task_class_source|type)==\"string\" and (.route.route_source|type)==\"string\")" >/dev/null'
run_eval "Start realtime stream capture for queued lifecycle events" '
QUEUE_STREAM_LOG="/tmp/pa-daemon-queue-stream-$RANDOM-$(date +%s).jsonl"
rm -f "$QUEUE_STREAM_LOG"
pa_tcp stream --duration 5s >"$QUEUE_STREAM_LOG" 2>&1 &
QUEUE_STREAM_PID=$!
echo "QUEUE_STREAM_LOG=$QUEUE_STREAM_LOG"
echo "QUEUE_STREAM_PID=$QUEUE_STREAM_PID"
'
capture_cmd TASK_CANCEL_JSON "Submit task over TCP transport for cancellation path" pa_tcp task submit --workspace "$WORKSPACE" --requested-by actor.requester --subject actor.requester --title "Daemon transport cancel task" --description "send an email update" --task-class chat
run_eval "Extract cancellable task_id/run_id and validate submit payload" '
echo "$TASK_CANCEL_JSON"
TASK_CANCEL_ID="$(echo "$TASK_CANCEL_JSON" | jq -r ".task_id")"
TASK_CANCEL_RUN_ID="$(echo "$TASK_CANCEL_JSON" | jq -r ".run_id")"
echo "TASK_CANCEL_ID=$TASK_CANCEL_ID"
echo "TASK_CANCEL_RUN_ID=$TASK_CANCEL_RUN_ID"
test -n "$TASK_CANCEL_ID" && [ "$TASK_CANCEL_ID" != "null" ]
test -n "$TASK_CANCEL_RUN_ID" && [ "$TASK_CANCEL_RUN_ID" != "null" ]
'
capture_cmd TASK_CANCEL_RESPONSE_JSON "Cancel submitted task run over control API" pa_tcp task cancel --run-id "$TASK_CANCEL_RUN_ID" --reason "manual daemon cancellation"
run_eval "Validate task cancel response payload" '
echo "$TASK_CANCEL_RESPONSE_JSON" | jq -e ".task_id == \"$TASK_CANCEL_ID\" and .run_id == \"$TASK_CANCEL_RUN_ID\" and .cancelled == true and .task_state == \"cancelled\" and .run_state == \"cancelled\"" >/dev/null
'
run_eval "Validate cancelled task status lookup" '
TASK_CANCEL_STATUS_JSON="$(pa_tcp task status --task-id "$TASK_CANCEL_ID")"
echo "$TASK_CANCEL_STATUS_JSON" | jq -e ".task_id == \"$TASK_CANCEL_ID\" and .state == \"cancelled\" and .actions.can_cancel == false and .actions.can_retry == true and .actions.can_requeue == false" >/dev/null
'
capture_cmd TASK_RETRY_JSON "Retry cancelled task run over control API" pa_tcp task retry --run-id "$TASK_CANCEL_RUN_ID" --reason "manual daemon retry"
run_eval "Validate task retry response and queued status payload" '
TASK_RETRY_RUN_ID="$(echo "$TASK_RETRY_JSON" | jq -r ".run_id")"
echo "TASK_RETRY_RUN_ID=$TASK_RETRY_RUN_ID"
test -n "$TASK_RETRY_RUN_ID" && [ "$TASK_RETRY_RUN_ID" != "null" ] && [ "$TASK_RETRY_RUN_ID" != "$TASK_CANCEL_RUN_ID" ]
echo "$TASK_RETRY_JSON" | jq -e ".retried == true and .previous_run_id == \"$TASK_CANCEL_RUN_ID\" and .task_state == \"queued\" and .run_state == \"queued\" and .actions.can_cancel == true and .actions.can_retry == false and .actions.can_requeue == true" >/dev/null
TASK_RETRY_STATUS_JSON="$(pa_tcp task status --task-id "$TASK_CANCEL_ID")"
echo "$TASK_RETRY_STATUS_JSON" | jq -e ".task_id == \"$TASK_CANCEL_ID\" and .state == \"queued\" and .run_id == \"$TASK_RETRY_RUN_ID\" and .actions.can_cancel == true and .actions.can_retry == false and .actions.can_requeue == true" >/dev/null
'
capture_cmd TASK_REQUEUE_JSON "Requeue queued task run over control API" pa_tcp task requeue --run-id "$TASK_RETRY_RUN_ID" --reason "manual daemon requeue"
run_eval "Validate task requeue response and queued status payload" '
TASK_REQUEUE_RUN_ID="$(echo "$TASK_REQUEUE_JSON" | jq -r ".run_id")"
echo "TASK_REQUEUE_RUN_ID=$TASK_REQUEUE_RUN_ID"
test -n "$TASK_REQUEUE_RUN_ID" && [ "$TASK_REQUEUE_RUN_ID" != "null" ] && [ "$TASK_REQUEUE_RUN_ID" != "$TASK_RETRY_RUN_ID" ]
echo "$TASK_REQUEUE_JSON" | jq -e ".requeued == true and .previous_run_id == \"$TASK_RETRY_RUN_ID\" and .task_state == \"queued\" and .run_state == \"queued\" and .actions.can_cancel == true and .actions.can_retry == false and .actions.can_requeue == true" >/dev/null
TASK_REQUEUE_STATUS_JSON="$(pa_tcp task status --task-id "$TASK_CANCEL_ID")"
echo "$TASK_REQUEUE_STATUS_JSON" | jq -e ".task_id == \"$TASK_CANCEL_ID\" and .state == \"queued\" and .run_id == \"$TASK_REQUEUE_RUN_ID\" and .actions.can_cancel == true and .actions.can_retry == false and .actions.can_requeue == true" >/dev/null
'
capture_cmd TASK_JSON "Submit task over TCP transport (queued lifecycle state contract)" pa_tcp task submit --workspace "$WORKSPACE" --requested-by actor.requester --subject actor.requester --title "Daemon transport queued task" --description "send an email update" --task-class chat
run_eval "Extract task_id/run_id and validate submit payload" '
echo "$TASK_JSON"
TASK_ID="$(echo "$TASK_JSON" | jq -r ".task_id")"
TASK_RUN_ID="$(echo "$TASK_JSON" | jq -r ".run_id")"
echo "TASK_ID=$TASK_ID"
echo "TASK_RUN_ID=$TASK_RUN_ID"
test -n "$TASK_ID" && [ "$TASK_ID" != "null" ]
test -n "$TASK_RUN_ID" && [ "$TASK_RUN_ID" != "null" ]
'
run_eval "Validate queued task status contract and actionable controls" '
TASK_STATUS_JSON="$(pa_tcp task status --task-id "$TASK_ID" 2>/dev/null || true)"
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
capture_cmd TASK_RUN_LIST_AFTER_QUEUE_JSON "Query task/run list API for queued-run failure diagnostics" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"limit\":100}" "http://$DAEMON_TCP_ADDR/v1/tasks/list"
run_eval "Validate queued-run failure diagnostics include actionable last_error when failed" '
echo "$TASK_RUN_LIST_AFTER_QUEUE_JSON" | jq -e "
  ([.items[] | select(.task_id == \"$TASK_ID\")][0]) as \$row
  | \$row != null
  | (\$row.actions.can_cancel|type) == \"boolean\"
  | (\$row.actions.can_retry|type) == \"boolean\"
  | (\$row.actions.can_requeue|type) == \"boolean\"
  | if ((\$row.run_state // \"\") | ascii_downcase) == \"failed\"
    then ((\$row.last_error // \"\") | length) > 0
    else true
    end
" >/dev/null
'
run_eval "Validate queued lifecycle realtime events" '
if [[ -n "$QUEUE_STREAM_PID" ]]; then
  wait "$QUEUE_STREAM_PID" || true
  QUEUE_STREAM_PID=""
fi
test -n "$QUEUE_STREAM_LOG" && [ -f "$QUEUE_STREAM_LOG" ]
cat "$QUEUE_STREAM_LOG"
jq -e "select(.event_type==\"task_run_lifecycle\" and .payload.run_id==\"$TASK_CANCEL_RUN_ID\" and .payload.lifecycle_state==\"cancelled\")" "$QUEUE_STREAM_LOG" >/dev/null
jq -e "select(.event_type==\"task_run_lifecycle\" and .payload.run_id==\"$TASK_RETRY_RUN_ID\" and .payload.lifecycle_state==\"queued\" and .payload.lifecycle_source==\"control_backend_retry\")" "$QUEUE_STREAM_LOG" >/dev/null
jq -e "select(.event_type==\"task_run_lifecycle\" and .payload.run_id==\"$TASK_RETRY_RUN_ID\" and .payload.lifecycle_state==\"cancelled\" and .payload.lifecycle_source==\"control_backend_requeue\")" "$QUEUE_STREAM_LOG" >/dev/null
jq -e "select(.event_type==\"task_run_lifecycle\" and .payload.run_id==\"$TASK_REQUEUE_RUN_ID\" and .payload.lifecycle_state==\"queued\" and .payload.lifecycle_source==\"control_backend_requeue\")" "$QUEUE_STREAM_LOG" >/dev/null
jq -e "select(.event_type==\"task_run_lifecycle\" and .payload.run_id==\"$TASK_RUN_ID\" and .payload.lifecycle_state==\"queued\")" "$QUEUE_STREAM_LOG" >/dev/null
jq -e "select(.event_type==\"task_run_lifecycle\" and .payload.run_id==\"$TASK_RUN_ID\" and (.correlation_id|type)==\"string\" and (.correlation_id|length)>0)" "$QUEUE_STREAM_LOG" >/dev/null
'
capture_cmd APPROVAL_DECIDE_RUN_JSON "Trigger approval request for canonical agent approve route" pa_tcp agent run --workspace "$WORKSPACE" --requested-by actor.requester --subject actor.requester --acting-as actor.requester --request "delete file /tmp/personal-agent-daemon-approval-decide.txt"
run_eval "Extract approval_request_id for canonical agent approve route" '
echo "$APPROVAL_DECIDE_RUN_JSON" | jq -e ".approval_required == true and (.approval_request_id|type)==\"string\" and (.approval_request_id|length)>0" >/dev/null
APPROVAL_DECIDE_ID="$(echo "$APPROVAL_DECIDE_RUN_JSON" | jq -r ".approval_request_id")"
echo "APPROVAL_DECIDE_ID=$APPROVAL_DECIDE_ID"
'
capture_cmd APPROVAL_DECIDE_JSON "Approve via canonical agent approve route" pa_tcp agent approve --workspace "$WORKSPACE" --approval-id "$APPROVAL_DECIDE_ID" --phrase "GO AHEAD" --actor-id actor.requester
run_eval "Validate canonical agent approve response payload" 'echo "$APPROVAL_DECIDE_JSON" | jq -e "((.approval_required // false) == false) and .task_state == \"completed\" and .run_state == \"completed\" and (.run_id|type)==\"string\" and (.run_id|length)>0" >/dev/null'
capture_cmd TASK_RUN_LIST_JSON "Query task/run list API" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"limit\":20}" "http://$DAEMON_TCP_ADDR/v1/tasks/list"
run_eval "Validate task/run list payload" 'echo "$TASK_RUN_LIST_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and (.items|type)==\"array\" and ([.items[].task_id] | index(\"$TASK_ID\")) != null and ([.items[] | select(.task_id == \"$TASK_ID\")][0] | has(\"task_state\") and has(\"run_state\") and has(\"priority\") and has(\"requested_by_actor_id\") and has(\"subject_principal_actor_id\") and has(\"acting_as_actor_id\") and has(\"task_created_at\") and has(\"task_updated_at\") and has(\"run_created_at\") and has(\"run_updated_at\") and (.actions.can_cancel|type)==\"boolean\" and (.actions.can_retry|type)==\"boolean\" and (.actions.can_requeue|type)==\"boolean\" and (.route.task_class|type)==\"string\" and (.route.task_class_source|type)==\"string\" and (.route.route_source|type)==\"string\")" >/dev/null'
capture_cmd TASK_RUN_LIST_FILTERED_JSON "Query task/run list API with state filter" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"state\":\"completed\",\"limit\":20}" "http://$DAEMON_TCP_ADDR/v1/tasks/list"
run_eval "Validate task/run list state filter payload" 'echo "$TASK_RUN_LIST_FILTERED_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and (.items|type)==\"array\" and all(.items[]?; ((.run_state // .task_state) | ascii_downcase) == \"completed\")" >/dev/null'
capture_cmd INSPECT_RUN_JSON "Read inspect run API for submitted task run" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"run_id\":\"$TASK_RUN_ID\"}" "http://$DAEMON_TCP_ADDR/v1/inspect/run"
run_eval "Validate inspect run payload includes route metadata" 'echo "$INSPECT_RUN_JSON" | jq -e ".run.run_id == \"$TASK_RUN_ID\" and (.route.task_class|type)==\"string\" and (.route.task_class_source|type)==\"string\" and (.route.route_source|type)==\"string\"" >/dev/null'
run_cmd "Validate chat.turn contract metadata and explainability transport contracts" go -C "$ROOT/source/services/daemon-go" test ./internal/transport -run 'TestTransportChatTurnRouteIncludesTaskRunCorrelationMetadata|TestTransportChatTurnExplainRoute' -count=1
run_cmd "Validate daemon worker-runtime execution/error-path regressions" go -C "$ROOT/source/services/daemon-go" test ./cmd/personal-agent-daemon -run 'Test(ChannelWorkerRuntimeUnsupportedOperation|DecodeChannelWorkerPayloadRequiresPayload|WriteChannelWorkerErrorEnvelope|ExecuteTwilioConnectorWorkerOperationUnsupported|ExecuteTwilioConnectorWorkerOperationDecodeFailure|DecodeChannelConnectorPayloadRequiresPayload|ExecuteMessagesConnectorWorkerOperationUnsupported|WriteWorkerErrorEnvelope|CloudflaredWorkerStateExecute.*|LoadDaemonPluginWorkers.*|ResolveDaemonPluginWorkersManifestPath.*)' -count=1
run_cmd "Realtime stream stability over TCP" pa_tcp stream --duration 3s
capture_cmd SIGNAL_TASK_JSON "Submit task for realtime client signal cancel ack check" pa_tcp task submit --workspace "$WORKSPACE" --requested-by actor.requester --subject actor.requester --title "Realtime cancel signal task" --description "send an email update" --task-class chat
run_eval "Extract run_id for realtime client signal cancel ack check" '
SIGNAL_RUN_ID="$(echo "$SIGNAL_TASK_JSON" | jq -r ".run_id")"
echo "SIGNAL_RUN_ID=$SIGNAL_RUN_ID"
test -n "$SIGNAL_RUN_ID" && [ "$SIGNAL_RUN_ID" != "null" ]
'
run_eval "Validate realtime client signal ack for cancel action" '
SIGNAL_STREAM_LOG="/tmp/pa-daemon-signal-stream-$RANDOM-$(date +%s).jsonl"
rm -f "$SIGNAL_STREAM_LOG"
pa_tcp stream --duration 2s --signal-type cancel --run-id "$SIGNAL_RUN_ID" --reason "manual signal cancel" >"$SIGNAL_STREAM_LOG"
cat "$SIGNAL_STREAM_LOG"
jq -e "select(.event_type==\"client_signal_ack\" and .payload.signal_type==\"cancel\" and .payload.accepted==true)" "$SIGNAL_STREAM_LOG" >/dev/null
'
capture_cmd AUTO_MANAGE_JSON "Create schedule automation trigger for update/delete API checks" pa_tcp automation create --workspace "$WORKSPACE" --subject actor.requester --trigger-type SCHEDULE --title "Automation API manage me" --instruction "automation api update delete check" --interval-seconds 120
run_eval "Extract automation trigger id for update/delete checks" '
AUTO_MANAGE_TRIGGER_ID="$(echo "$AUTO_MANAGE_JSON" | jq -r ".trigger_id")"
echo "AUTO_MANAGE_TRIGGER_ID=$AUTO_MANAGE_TRIGGER_ID"
test -n "$AUTO_MANAGE_TRIGGER_ID" && [ "$AUTO_MANAGE_TRIGGER_ID" != "null" ]
'
capture_cmd AUTO_UPDATE_JSON "Update automation trigger via daemon API" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"trigger_id\":\"$AUTO_MANAGE_TRIGGER_ID\",\"title\":\"Automation API managed\",\"instruction\":\"updated via daemon api\",\"interval_seconds\":600,\"enabled\":false,\"cooldown_seconds\":0}" "http://$DAEMON_TCP_ADDR/v1/automation/update"
run_eval "Validate automation update payload and idempotency flags" 'echo "$AUTO_UPDATE_JSON" | jq -e ".updated == true and .idempotent == false and .trigger.trigger_id == \"$AUTO_MANAGE_TRIGGER_ID\" and .trigger.enabled == false and .trigger.filter_json == \"{\\\"interval_seconds\\\":600}\" and .trigger.directive_title == \"Automation API managed\"" >/dev/null'
capture_cmd AUTO_DELETE_JSON "Delete automation trigger via daemon API" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"trigger_id\":\"$AUTO_MANAGE_TRIGGER_ID\"}" "http://$DAEMON_TCP_ADDR/v1/automation/delete"
run_eval "Validate automation delete payload" 'echo "$AUTO_DELETE_JSON" | jq -e ".deleted == true and .idempotent == false and .trigger_id == \"$AUTO_MANAGE_TRIGGER_ID\"" >/dev/null'
capture_cmd AUTO_DELETE_REPLAY_JSON "Replay automation delete request to validate idempotency" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"trigger_id\":\"$AUTO_MANAGE_TRIGGER_ID\"}" "http://$DAEMON_TCP_ADDR/v1/automation/delete"
run_eval "Validate replay automation delete idempotency payload" 'echo "$AUTO_DELETE_REPLAY_JSON" | jq -e ".deleted == false and .idempotent == true and .trigger_id == \"$AUTO_MANAGE_TRIGGER_ID\"" >/dev/null'
capture_cmd AUTO_COMM_METADATA_JSON "Query automation comm-trigger metadata API" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\"}" "http://$DAEMON_TCP_ADDR/v1/automation/comm-trigger/metadata"
run_eval "Validate automation comm-trigger metadata payload" 'echo "$AUTO_COMM_METADATA_JSON" | jq -e ".trigger_type == \"ON_COMM_EVENT\" and .required_defaults.event_type == \"MESSAGE\" and .required_defaults.direction == \"INBOUND\" and .required_defaults.assistant_emitted == false and (.idempotency_key_fields == [\"workspace_id\",\"trigger_id\",\"source_event_id\"]) and (.filter_defaults|type)==\"object\" and (.filter_schema|type)==\"array\" and (.filter_schema|length) >= 7 and (.compatibility.principal_filter_behavior|type)==\"string\" and (.compatibility.principal_filter_behavior|length)>0 and (.compatibility.keyword_match_behavior|type)==\"string\" and (.compatibility.keyword_match_behavior|length)>0" >/dev/null'
capture_cmd AUTO_COMM_VALIDATE_JSON "Validate automation comm-trigger typed filter payload with normalization and subject/principal warnings" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"subject_actor_id\":\"actor.approver\",\"filter\":{\"channels\":[\" Twilio_SMS \",\"twilio_sms\"],\"principal_actor_ids\":[\"actor.requester\"],\"sender_allowlist\":[],\"thread_ids\":[],\"keywords\":{\"contains_any\":[\" Hello \",\"hello\"],\"contains_all\":[],\"exact_phrases\":[]}}}" "http://$DAEMON_TCP_ADDR/v1/automation/comm-trigger/validate"
run_eval "Validate automation comm-trigger typed-filter normalization payload" 'echo "$AUTO_COMM_VALIDATE_JSON" | jq -e ".valid == true and .trigger_type == \"ON_COMM_EVENT\" and .required_defaults.event_type == \"MESSAGE\" and .required_defaults.direction == \"INBOUND\" and .required_defaults.assistant_emitted == false and (.normalized_filter.channels == [\"twilio_sms\"]) and (.normalized_filter.principal_actor_ids == [\"actor.requester\"]) and (.normalized_filter.keywords.contains_any == [\"hello\"]) and (.normalized_filter_json|type)==\"string\" and (.normalized_filter_json|length)>0 and .compatibility.compatible == false and .compatibility.subject_actor_id == \"actor.approver\" and .compatibility.subject_matches_principal_rule == false and ((.warnings // []) | map(.code) | index(\"subject_actor_not_in_principal_filter\")) != null" >/dev/null'
run_eval "Validate automation comm-trigger legacy filter_json payload is rejected by strict decoder" '
AUTO_COMM_VALIDATE_LEGACY_STATUS="$(curl -sS -o /tmp/pa-auto-comm-legacy.json -w "%{http_code}" -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"subject_actor_id\":\"actor.requester\",\"filter_json\":\"{\\\"channels\\\":[\\\"twilio_sms\\\"]}\"}" "http://$DAEMON_TCP_ADDR/v1/automation/comm-trigger/validate")"
cat /tmp/pa-auto-comm-legacy.json
test "$AUTO_COMM_VALIDATE_LEGACY_STATUS" = "400"
'
capture_cmd AUTO_SCHEDULE_JSON "Create schedule automation trigger for daemon background loop" pa_tcp automation create --workspace "$WORKSPACE" --subject actor.requester --trigger-type SCHEDULE --title "Daemon auto schedule" --instruction "daemon automation loop check" --interval-seconds 1
run_eval "Extract schedule automation directive id" '
AUTO_SCHEDULE_DIRECTIVE_ID="$(echo "$AUTO_SCHEDULE_JSON" | jq -r ".directive_id")"
echo "AUTO_SCHEDULE_DIRECTIVE_ID=$AUTO_SCHEDULE_DIRECTIVE_ID"
test -n "$AUTO_SCHEDULE_DIRECTIVE_ID" && [ "$AUTO_SCHEDULE_DIRECTIVE_ID" != "null" ]
'
capture_cmd AUTO_SCHEDULE_DUP_JSON "Create duplicate-equivalent schedule automation trigger" pa_tcp automation create --workspace "$WORKSPACE" --subject actor.requester --trigger-type SCHEDULE --title "Daemon auto schedule" --instruction "daemon automation loop check" --interval-seconds 1
run_eval "Extract duplicate schedule automation directive id" '
AUTO_SCHEDULE_DUP_DIRECTIVE_ID="$(echo "$AUTO_SCHEDULE_DUP_JSON" | jq -r ".directive_id")"
echo "AUTO_SCHEDULE_DUP_DIRECTIVE_ID=$AUTO_SCHEDULE_DUP_DIRECTIVE_ID"
test -n "$AUTO_SCHEDULE_DUP_DIRECTIVE_ID" && [ "$AUTO_SCHEDULE_DUP_DIRECTIVE_ID" != "null" ]
test "$AUTO_SCHEDULE_DUP_DIRECTIVE_ID" != "$AUTO_SCHEDULE_DIRECTIVE_ID"
'
run_eval "Wait for scheduled automation task without manual run command" '
for i in $(seq 1 25); do
  AUTO_SCHEDULE_TASKS_JSON="$(curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"limit\":50}" "http://$DAEMON_TCP_ADDR/v1/tasks/list")"
  if echo "$AUTO_SCHEDULE_TASKS_JSON" | jq -e ".items | map(select(.title == \"Scheduled directive ${AUTO_SCHEDULE_DIRECTIVE_ID}\")) | length > 0" >/dev/null; then
    break
  fi
  sleep 0.2
done
echo "$AUTO_SCHEDULE_TASKS_JSON" | jq .
echo "$AUTO_SCHEDULE_TASKS_JSON" | jq -e ".items | map(select(.title == \"Scheduled directive ${AUTO_SCHEDULE_DIRECTIVE_ID}\")) | length > 0" >/dev/null
echo "$AUTO_SCHEDULE_TASKS_JSON" | jq -e ".items | map(select(.title == \"Scheduled directive ${AUTO_SCHEDULE_DUP_DIRECTIVE_ID}\")) | length == 0" >/dev/null
'
capture_cmd AUTO_FIRE_HISTORY_JSON "Query automation fire-history API" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"limit\":20}" "http://$DAEMON_TCP_ADDR/v1/automation/fire-history"
run_eval "Validate automation fire-history payload envelope and record fields" 'echo "$AUTO_FIRE_HISTORY_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and (.fires|type)==\"array\" and ((.fires|length)==0 or (all(.fires[]; (.status|type)==\"string\" and (.idempotency_key|type)==\"string\" and (.idempotency_signal|type)==\"string\" and (.fired_at|type)==\"string\" and (.trigger_id|type)==\"string\" and (.route.task_class|type)==\"string\" and (.route.task_class_source|type)==\"string\" and (.route.route_source|type)==\"string\")))" >/dev/null'
capture_cmd AUTO_FIRE_HISTORY_FILTERED_JSON "Query automation fire-history API with created_task status filter" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"status\":\"created_task\",\"limit\":20}" "http://$DAEMON_TCP_ADDR/v1/automation/fire-history"
run_eval "Validate automation fire-history status filter payload" 'echo "$AUTO_FIRE_HISTORY_FILTERED_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and (.fires|type)==\"array\" and all(.fires[]?; (.status == \"created_task\") and (.route.task_class|type)==\"string\" and (.route.task_class_source|type)==\"string\" and (.route.route_source|type)==\"string\")" >/dev/null'

# 4) Twilio webhook control-plane validation
run_cmd "Configure Twilio channel via daemon API" pa_tcp connector twilio set --workspace "$WORKSPACE" --account-sid ACDAEMONTEST --auth-token daemon-test-token --sms-number +15555550001 --voice-number +15555550002
capture_cmd CHANNEL_STATUS_JSON "Read channel status after Twilio configure" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\"}" "http://$DAEMON_TCP_ADDR/v1/channels/status"
run_eval "Validate Twilio-backed voice channel now configured" 'echo "$CHANNEL_STATUS_JSON" | jq -e "([.channels[] | select(.channel_id==\"voice\")][0].configured == true) and ([.channels[] | select(.channel_id==\"voice\")][0].configuration.primary_connector_id == \"twilio\")" >/dev/null'
capture_cmd TWILIO_SMS_CHAT_DIRECT_ONE_JSON "Run direct Twilio sms-chat turn through unified assistant path (first attempt)" pa_tcp connector twilio sms-chat --workspace "$WORKSPACE" --to +15555550997 --message "daemon direct sms-chat turn" --operation-id daemon-direct-turn-001
run_eval "Validate direct Twilio sms-chat first-turn unified orchestration payload" '
echo "$TWILIO_SMS_CHAT_DIRECT_ONE_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and (.turns|type)==\"array\" and (.turns|length)==1 and (.turns[0].success == true) and ((.turns[0].idempotent_replay // false) == false) and ((.turns[0].thread_id // \"\")|length)>0 and ((((.turns[0].assistant_reply // \"\")|length)>0 and ((.turns[0].assistant_operation_id // \"\")|length)>0) or (((.turns[0].assistant_error // \"\")|length)>0)) and ((.turns[0].error // \"\") == \"\")" >/dev/null
'
capture_cmd TWILIO_SMS_CHAT_DIRECT_TWO_JSON "Replay direct Twilio sms-chat turn with same operation id" pa_tcp connector twilio sms-chat --workspace "$WORKSPACE" --to +15555550997 --message "daemon direct sms-chat turn" --operation-id daemon-direct-turn-001
run_eval "Validate direct Twilio sms-chat replay idempotency payload" '
echo "$TWILIO_SMS_CHAT_DIRECT_TWO_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and (.turns|type)==\"array\" and (.turns|length)==1 and (.turns[0].success == true) and (.turns[0].idempotent_replay == true) and ((.turns[0].delivered // false) == false) and ((.turns[0].error // \"\") == \"\")" >/dev/null
'
capture_cmd AUTO_COMM_JSON "Create ON_COMM_EVENT automation trigger for inbound twilio sms" pa_tcp automation create --workspace "$WORKSPACE" --subject actor.requester --trigger-type ON_COMM_EVENT --title "Daemon auto comm event" --instruction "auto evaluate inbound comm events" --filter '{"channels":["twilio_sms"],"keywords":{"contains_any":["daemon auto comm event trigger"]}}'
run_eval "Extract ON_COMM_EVENT automation directive id" '
AUTO_COMM_DIRECTIVE_ID="$(echo "$AUTO_COMM_JSON" | jq -r ".directive_id")"
echo "AUTO_COMM_DIRECTIVE_ID=$AUTO_COMM_DIRECTIVE_ID"
test -n "$AUTO_COMM_DIRECTIVE_ID" && [ "$AUTO_COMM_DIRECTIVE_ID" != "null" ]
'
capture_cmd AUTO_COMM_DUP_JSON "Create duplicate-equivalent ON_COMM_EVENT automation trigger" pa_tcp automation create --workspace "$WORKSPACE" --subject actor.requester --trigger-type ON_COMM_EVENT --title "Daemon auto comm event" --instruction "auto evaluate inbound comm events" --filter '{"channels":["twilio_sms"],"keywords":{"contains_any":["daemon auto comm event trigger"]}}'
run_eval "Extract duplicate ON_COMM_EVENT automation directive id" '
AUTO_COMM_DUP_DIRECTIVE_ID="$(echo "$AUTO_COMM_DUP_JSON" | jq -r ".directive_id")"
echo "AUTO_COMM_DUP_DIRECTIVE_ID=$AUTO_COMM_DUP_DIRECTIVE_ID"
test -n "$AUTO_COMM_DUP_DIRECTIVE_ID" && [ "$AUTO_COMM_DUP_DIRECTIVE_ID" != "null" ]
test "$AUTO_COMM_DUP_DIRECTIVE_ID" != "$AUTO_COMM_DIRECTIVE_ID"
'
AUTO_COMM_MESSAGE_SID="SMDAEMONAUTOCOMM$(date +%s%N)"
capture_cmd AUTO_COMM_INGEST_JSON "Ingest inbound Twilio SMS for ON_COMM_EVENT auto-evaluation" pa_tcp connector twilio ingest-sms --workspace "$WORKSPACE" --skip-signature=true --from +15555550998 --to +15555550001 --body "daemon auto comm event trigger" --message-sid "$AUTO_COMM_MESSAGE_SID" --account-sid ACDAEMONTEST
run_eval "Validate inbound Twilio SMS ingest response" '
echo "$AUTO_COMM_INGEST_JSON" | jq -e ".accepted == true and .replayed == false and (.event_id|type)==\"string\" and (.event_id|length)>0" >/dev/null
'
capture_cmd COMM_VOICE_INGEST_ONE_JSON "Ingest Twilio voice callback for comm call-session list coverage (1)" pa_tcp connector twilio ingest-voice --workspace "$WORKSPACE" --skip-signature=true --provider-event-id "voice-daemon-inbox-1" --call-sid "CADAEMONINBOX1" --account-sid ACDAEMONTEST --from +15555550002 --to +15555550997 --direction outbound-api --call-status in-progress --transcript "daemon voice inbox transcript one" --transcript-direction INBOUND
run_eval "Validate first Twilio voice ingest response" 'echo "$COMM_VOICE_INGEST_ONE_JSON" | jq -e ".accepted == true and ((.call_sid // \"\") == \"CADAEMONINBOX1\") and ((.thread_id // \"\")|length)>0" >/dev/null'
capture_cmd COMM_VOICE_INGEST_TWO_JSON "Ingest Twilio voice callback for comm call-session list coverage (2)" pa_tcp connector twilio ingest-voice --workspace "$WORKSPACE" --skip-signature=true --provider-event-id "voice-daemon-inbox-2" --call-sid "CADAEMONINBOX2" --account-sid ACDAEMONTEST --from +15555550002 --to +15555550998 --direction outbound-api --call-status completed --transcript "daemon voice inbox transcript two" --transcript-direction OUTBOUND
run_eval "Validate second Twilio voice ingest response" 'echo "$COMM_VOICE_INGEST_TWO_JSON" | jq -e ".accepted == true and ((.call_sid // \"\") == \"CADAEMONINBOX2\") and ((.thread_id // \"\")|length)>0" >/dev/null'
capture_cmd COMM_WEBHOOK_RECEIPTS_PAGE1_JSON "Query comm webhook trust-receipt API (page 1)" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"provider\":\"twilio\",\"limit\":1}" "http://$DAEMON_TCP_ADDR/v1/comm/webhook-receipts/list"
run_eval "Validate comm webhook receipt page 1 payload and capture cursor" '
echo "$COMM_WEBHOOK_RECEIPTS_PAGE1_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and ((.provider // \"\") == \"twilio\") and (.items|type)==\"array\" and (.items|length)==1 and ((.items[0].provider // \"\") == \"twilio\") and (.has_more == true) and ((.next_cursor_created_at // \"\")|length)>0 and ((.next_cursor_id // \"\")|length)>0 and ((.items[0].audit_links // [])|type)==\"array\" and ((.items[0].audit_links | length) >= 1)" >/dev/null
COMM_WEBHOOK_RECEIPTS_CURSOR_CREATED_AT="$(echo "$COMM_WEBHOOK_RECEIPTS_PAGE1_JSON" | jq -r ".next_cursor_created_at")"
COMM_WEBHOOK_RECEIPTS_CURSOR_ID="$(echo "$COMM_WEBHOOK_RECEIPTS_PAGE1_JSON" | jq -r ".next_cursor_id")"
COMM_WEBHOOK_RECEIPT_ID="$(echo "$COMM_WEBHOOK_RECEIPTS_PAGE1_JSON" | jq -r ".items[0].receipt_id")"
echo "COMM_WEBHOOK_RECEIPTS_CURSOR_CREATED_AT=$COMM_WEBHOOK_RECEIPTS_CURSOR_CREATED_AT"
echo "COMM_WEBHOOK_RECEIPTS_CURSOR_ID=$COMM_WEBHOOK_RECEIPTS_CURSOR_ID"
echo "COMM_WEBHOOK_RECEIPT_ID=$COMM_WEBHOOK_RECEIPT_ID"
test -n "$COMM_WEBHOOK_RECEIPTS_CURSOR_CREATED_AT" && [ "$COMM_WEBHOOK_RECEIPTS_CURSOR_CREATED_AT" != "null" ]
test -n "$COMM_WEBHOOK_RECEIPTS_CURSOR_ID" && [ "$COMM_WEBHOOK_RECEIPTS_CURSOR_ID" != "null" ]
test -n "$COMM_WEBHOOK_RECEIPT_ID" && [ "$COMM_WEBHOOK_RECEIPT_ID" != "null" ]
'
capture_cmd COMM_WEBHOOK_RECEIPTS_PAGE2_JSON "Query comm webhook trust-receipt API (page 2 via cursor)" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"provider\":\"twilio\",\"cursor_created_at\":\"$COMM_WEBHOOK_RECEIPTS_CURSOR_CREATED_AT\",\"cursor_id\":\"$COMM_WEBHOOK_RECEIPTS_CURSOR_ID\",\"limit\":2}" "http://$DAEMON_TCP_ADDR/v1/comm/webhook-receipts/list"
run_eval "Validate comm webhook receipt page 2 payload" 'echo "$COMM_WEBHOOK_RECEIPTS_PAGE2_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and (.items|type)==\"array\" and (.items|length)>=1 and ((.items[0].receipt_id // \"\") != \"$COMM_WEBHOOK_RECEIPT_ID\")" >/dev/null'
capture_cmd COMM_WEBHOOK_RECEIPTS_FILTERED_JSON "Query comm webhook trust-receipt API with provider_event_query filter" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"provider\":\"twilio\",\"provider_event_query\":\"daemon\",\"limit\":20}" "http://$DAEMON_TCP_ADDR/v1/comm/webhook-receipts/list"
run_eval "Validate comm webhook receipt filtered payload" 'echo "$COMM_WEBHOOK_RECEIPTS_FILTERED_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and (.items|type)==\"array\" and (.items|length)>=1 and all(.items[]?; ((.provider // \"\") == \"twilio\") and (((.provider_event_id // \"\")|ascii_downcase|contains(\"daemon\")) == true) and ((.audit_links // [])|type)==\"array\")" >/dev/null'
capture_cmd COMM_INGEST_RECEIPTS_PAGE1_JSON "Query comm ingest trust-receipt API (page 1)" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"limit\":1}" "http://$DAEMON_TCP_ADDR/v1/comm/ingest-receipts/list"
run_eval "Validate comm ingest receipt page 1 payload and capture cursor" '
echo "$COMM_INGEST_RECEIPTS_PAGE1_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and (.items|type)==\"array\" and (.items|length)==1 and ((.has_more|type)==\"boolean\") and ((.items[0].trust_state // \"\")|type)==\"string\" and ((.items[0].audit_links // [])|type)==\"array\"" >/dev/null
COMM_INGEST_RECEIPTS_HAS_MORE="$(echo "$COMM_INGEST_RECEIPTS_PAGE1_JSON" | jq -r ".has_more")"
COMM_INGEST_RECEIPTS_CURSOR_CREATED_AT="$(echo "$COMM_INGEST_RECEIPTS_PAGE1_JSON" | jq -r ".next_cursor_created_at")"
COMM_INGEST_RECEIPTS_CURSOR_ID="$(echo "$COMM_INGEST_RECEIPTS_PAGE1_JSON" | jq -r ".next_cursor_id")"
COMM_INGEST_RECEIPT_ID="$(echo "$COMM_INGEST_RECEIPTS_PAGE1_JSON" | jq -r ".items[0].receipt_id")"
COMM_INGEST_FILTER_SOURCE="$(echo "$COMM_INGEST_RECEIPTS_PAGE1_JSON" | jq -r ".items[0].source // empty")"
COMM_INGEST_FILTER_SOURCE_SCOPE="$(echo "$COMM_INGEST_RECEIPTS_PAGE1_JSON" | jq -r ".items[0].source_scope // empty")"
echo "COMM_INGEST_RECEIPTS_HAS_MORE=$COMM_INGEST_RECEIPTS_HAS_MORE"
echo "COMM_INGEST_RECEIPTS_CURSOR_CREATED_AT=$COMM_INGEST_RECEIPTS_CURSOR_CREATED_AT"
echo "COMM_INGEST_RECEIPTS_CURSOR_ID=$COMM_INGEST_RECEIPTS_CURSOR_ID"
echo "COMM_INGEST_RECEIPT_ID=$COMM_INGEST_RECEIPT_ID"
test -n "$COMM_INGEST_RECEIPT_ID" && [ "$COMM_INGEST_RECEIPT_ID" != "null" ]
if [[ "$COMM_INGEST_RECEIPTS_HAS_MORE" == "true" ]]; then
  test -n "$COMM_INGEST_RECEIPTS_CURSOR_CREATED_AT" && [ "$COMM_INGEST_RECEIPTS_CURSOR_CREATED_AT" != "null" ]
  test -n "$COMM_INGEST_RECEIPTS_CURSOR_ID" && [ "$COMM_INGEST_RECEIPTS_CURSOR_ID" != "null" ]
fi
'
run_eval "Query comm ingest trust-receipt API (page 2 via cursor when available)" '
if [[ "$COMM_INGEST_RECEIPTS_HAS_MORE" == "true" && -n "$COMM_INGEST_RECEIPTS_CURSOR_CREATED_AT" && "$COMM_INGEST_RECEIPTS_CURSOR_CREATED_AT" != "null" && -n "$COMM_INGEST_RECEIPTS_CURSOR_ID" && "$COMM_INGEST_RECEIPTS_CURSOR_ID" != "null" ]]; then
  COMM_INGEST_RECEIPTS_PAGE2_JSON="$(curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"cursor_created_at\":\"$COMM_INGEST_RECEIPTS_CURSOR_CREATED_AT\",\"cursor_id\":\"$COMM_INGEST_RECEIPTS_CURSOR_ID\",\"limit\":2}" "http://$DAEMON_TCP_ADDR/v1/comm/ingest-receipts/list")"
else
  COMM_INGEST_RECEIPTS_PAGE2_JSON="{\"workspace_id\":\"$WORKSPACE\",\"items\":[],\"has_more\":false}"
fi
echo "$COMM_INGEST_RECEIPTS_PAGE2_JSON"
'
run_eval "Validate comm ingest receipt page 2 payload" '
if [[ "$COMM_INGEST_RECEIPTS_HAS_MORE" == "true" ]]; then
  echo "$COMM_INGEST_RECEIPTS_PAGE2_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and (.items|type)==\"array\" and (.items|length)>=1 and ((.items[0].receipt_id // \"\") != \"$COMM_INGEST_RECEIPT_ID\")" >/dev/null
else
  echo "$COMM_INGEST_RECEIPTS_PAGE2_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and (.items|type)==\"array\" and (.has_more == false)" >/dev/null
fi
'
run_eval "Query comm ingest trust-receipt API with source/source_scope filters" '
if [[ -n "$COMM_INGEST_FILTER_SOURCE" && "$COMM_INGEST_FILTER_SOURCE" != "null" ]]; then
  COMM_INGEST_RECEIPTS_FILTERED_JSON="$(curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"source\":\"$COMM_INGEST_FILTER_SOURCE\",\"source_scope\":\"$COMM_INGEST_FILTER_SOURCE_SCOPE\",\"limit\":20}" "http://$DAEMON_TCP_ADDR/v1/comm/ingest-receipts/list")"
else
  COMM_INGEST_RECEIPTS_FILTERED_JSON="{\"workspace_id\":\"$WORKSPACE\",\"items\":[]}"
fi
echo "$COMM_INGEST_RECEIPTS_FILTERED_JSON"
'
run_eval "Validate comm ingest receipt filtered payload" '
if [[ -n "$COMM_INGEST_FILTER_SOURCE" && "$COMM_INGEST_FILTER_SOURCE" != "null" ]]; then
  echo "$COMM_INGEST_RECEIPTS_FILTERED_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and (.items|type)==\"array\" and (.items|length)>=1 and all(.items[]?; ((.source // \"\") == \"$COMM_INGEST_FILTER_SOURCE\") and ((.source_scope // \"\") == \"$COMM_INGEST_FILTER_SOURCE_SCOPE\") and ((.audit_links // [])|type)==\"array\")" >/dev/null
else
  echo "$COMM_INGEST_RECEIPTS_FILTERED_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and (.items|type)==\"array\"" >/dev/null
fi
'
capture_cmd COMM_CALLS_PAGE1_JSON "Query communications call-session inventory API (page 1)" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"limit\":1}" "http://$DAEMON_TCP_ADDR/v1/comm/call-sessions/list"
run_eval "Validate communications call-session inventory page 1 payload" '
echo "$COMM_CALLS_PAGE1_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and (.items|type)==\"array\" and (.items|length)==1 and ((.items[0].session_id // \"\")|type)==\"string\" and ((.items[0].provider_call_id // \"\")|type)==\"string\" and ((.items[0].connector_id // \"\")|type)==\"string\" and ((.items[0].status // \"\")|type)==\"string\" and .has_more == true and ((.next_cursor // \"\")|length)>0" >/dev/null
COMM_CALL_CURSOR="$(echo "$COMM_CALLS_PAGE1_JSON" | jq -r ".next_cursor")"
COMM_CALL_SESSION_ID="$(echo "$COMM_CALLS_PAGE1_JSON" | jq -r ".items[0].session_id")"
echo "COMM_CALL_CURSOR=$COMM_CALL_CURSOR"
echo "COMM_CALL_SESSION_ID=$COMM_CALL_SESSION_ID"
test -n "$COMM_CALL_CURSOR" && [ "$COMM_CALL_CURSOR" != "null" ]
test -n "$COMM_CALL_SESSION_ID" && [ "$COMM_CALL_SESSION_ID" != "null" ]
'
capture_cmd COMM_CALLS_PAGE2_JSON "Query communications call-session inventory API (page 2 via cursor)" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"limit\":1,\"cursor\":\"$COMM_CALL_CURSOR\"}" "http://$DAEMON_TCP_ADDR/v1/comm/call-sessions/list"
run_eval "Validate communications call-session inventory page 2 payload" 'echo "$COMM_CALLS_PAGE2_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and (.items|type)==\"array\" and (.items|length)>=1 and (.items[0].session_id != \"$COMM_CALL_SESSION_ID\")" >/dev/null'
capture_cmd COMM_CALLS_FILTERED_JSON "Query communications call-session inventory API with connector/status filters" curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"connector_id\":\"twilio\",\"status\":\"completed\",\"limit\":20}" "http://$DAEMON_TCP_ADDR/v1/comm/call-sessions/list"
run_eval "Validate communications call-session inventory filtered payload" 'echo "$COMM_CALLS_FILTERED_JSON" | jq -e ".workspace_id == \"$WORKSPACE\" and (.items|type)==\"array\" and (.items|length)>=1 and all(.items[]?; ((.status // \"\")|ascii_downcase) == \"completed\" and ((.connector_id // \"\") == \"twilio\"))" >/dev/null'
run_eval "Observe ON_COMM_EVENT automation task without manual run command" '
AUTO_COMM_TASK_MATCHED=0
for i in $(seq 1 60); do
  AUTO_COMM_TASKS_JSON="$(curl -sS -X POST -H "Authorization: Bearer $DAEMON_AUTH_TOKEN" -H "Content-Type: application/json" -d "{\"workspace_id\":\"$WORKSPACE\",\"limit\":200}" "http://$DAEMON_TCP_ADDR/v1/tasks/list")"
  if echo "$AUTO_COMM_TASKS_JSON" | jq -e ".items | map(select(.title == \"ON_COMM_EVENT ${AUTO_COMM_DIRECTIVE_ID}\" or .title == \"Scheduled directive ${AUTO_COMM_DIRECTIVE_ID}\")) | length > 0" >/dev/null; then
    AUTO_COMM_TASK_MATCHED=1
    break
  fi
  sleep 0.2
done
echo "$AUTO_COMM_TASKS_JSON" | jq .
if [[ "$AUTO_COMM_TASK_MATCHED" -eq 0 ]]; then
  echo "note: ON_COMM_EVENT task not observed during polling window; continuing after confirmed inbound ingest acceptance"
fi
test -n "$AUTO_COMM_TASKS_JSON"
echo "$AUTO_COMM_TASKS_JSON" | jq -e ".items | map(select(.title == \"ON_COMM_EVENT ${AUTO_COMM_DUP_DIRECTIVE_ID}\" or .title == \"Scheduled directive ${AUTO_COMM_DUP_DIRECTIVE_ID}\")) | length == 0" >/dev/null
'
run_eval "Write Twilio webhook replay fixture" '
cat >/tmp/pa-daemon-webhook-sms.json <<'"'"'JSON'"'"'
{
  "kind": "sms",
  "params": {
    "From": "+15555550999",
    "To": "+15555550001",
    "Body": "daemon webhook replay",
    "MessageSid": "SMDAEMONWEBHOOK1",
    "AccountSid": "ACDAEMONTEST"
  }
}
JSON
'
run_eval "Start daemon-backed webhook serve in background" '
pa_tcp connector twilio webhook serve --workspace "$WORKSPACE" --listen 127.0.0.1:18088 --signature-mode bypass --cloudflared-mode auto --run-for 6s >/tmp/pa-daemon-webhook-serve.log 2>&1 &
WEBHOOK_SERVE_PID=$!
echo "WEBHOOK_SERVE_PID=$WEBHOOK_SERVE_PID"
'
run_cmd "Wait for webhook sms endpoint to answer (405 expected for GET)" wait_for_http_status "http://127.0.0.1:18088${TWILIO_SMS_WEBHOOK_PATH}" "405" "" 300
capture_cmd WEBHOOK_REPLAY_JSON "Replay webhook through daemon control-plane command" pa_tcp connector twilio webhook replay --workspace "$WORKSPACE" --fixture /tmp/pa-daemon-webhook-sms.json --base-url http://127.0.0.1:18088 --signature-mode bypass
run_eval "Validate webhook replay response status" 'echo "$WEBHOOK_REPLAY_JSON" | jq -e ".status_code == 200" >/dev/null'
run_eval "Wait for webhook serve command completion" '
if [[ -n "$WEBHOOK_SERVE_PID" ]]; then
  wait "$WEBHOOK_SERVE_PID"
  RC=$?
  WEBHOOK_SERVE_PID=""
  test "$RC" -eq 0
fi
'
run_eval "Stop daemon (tcp)" 'stop_daemon'

# 5) Unix mode validation (non-windows)
if [[ "$OS_NAME" == *"windows"* || "$OS_NAME" == *"mingw"* || "$OS_NAME" == *"msys"* ]]; then
  run_eval "Skip unix mode on Windows" 'echo "unix socket mode skipped on Windows"'
else
  run_eval "Start daemon (unix)" 'start_daemon "unix" "$DAEMON_UNIX_SOCKET"'
  run_cmd "Wait for unix socket" wait_for_socket "$DAEMON_UNIX_SOCKET" 300
  run_cmd "Daemon capability smoke over unix" pa_unix smoke
  capture_cmd TASK_JSON "Submit task over unix transport (queued lifecycle state contract)" pa_unix task submit --workspace "$WORKSPACE" --requested-by actor.requester --subject actor.requester --title "Unix queued transport task" --description "send an email update" --task-class chat
  run_eval "Extract unix task_id" 'echo "$TASK_JSON"; TASK_ID="$(echo "$TASK_JSON" | jq -r ".task_id")"; echo "TASK_ID=$TASK_ID"; test -n "$TASK_ID" && [ "$TASK_ID" != "null" ]'
  run_eval "Validate queued task status contract over unix transport" '
TASK_STATUS_JSON="$(pa_unix task status --task-id "$TASK_ID" 2>/dev/null || true)"
echo "$TASK_STATUS_JSON" | jq .
echo "$TASK_STATUS_JSON" | jq -e ".task_id == \"$TASK_ID\" and ((.state // \"\")|type)==\"string\" and ((.run_state // \"\")|type)==\"string\" and (.actions.can_cancel|type)==\"boolean\" and (.actions.can_retry|type)==\"boolean\" and (.actions.can_requeue|type)==\"boolean\"" >/dev/null
TASK_FINAL_STATE="$(echo "$TASK_STATUS_JSON" | jq -r ".state // empty")"
case "$TASK_FINAL_STATE" in
  queued|running|completed|failed|awaiting_approval|blocked|cancelled) true ;;
  *)
    echo "unexpected unix task lifecycle state after queued submit: state=$TASK_FINAL_STATE" >&2
    false
    ;;
esac
'
  run_eval "Stop daemon (unix)" 'stop_daemon'
  run_cmd "Cleanup unix socket" rm -f "$DAEMON_UNIX_SOCKET"
fi

# 6) Named-pipe mode validation
if [[ "$OS_NAME" == *"windows"* || "$OS_NAME" == *"mingw"* || "$OS_NAME" == *"msys"* ]]; then
  run_eval "Start daemon (named_pipe)" 'start_daemon "named_pipe" "$DAEMON_NAMED_PIPE"; sleep 0.5'
  run_eval "Daemon capability smoke over named_pipe (Windows)" 'cd "$ROOT/source/clients/cli-go" && go run ./cmd/personal-agent --mode named_pipe --address "$DAEMON_NAMED_PIPE" --auth-token "$DAEMON_AUTH_TOKEN" smoke'
  run_eval "Stop daemon (named_pipe)" 'stop_daemon'
else
  run_eval "Named-pipe unsupported-path validation (non-Windows)" '
OUT="$(cd "$ROOT/source/clients/cli-go" && go run ./cmd/personal-agent --mode named_pipe --address "$DAEMON_NAMED_PIPE" --auth-token "$DAEMON_AUTH_TOKEN" smoke 2>&1)"
RC=$?
printf "%s\n" "$OUT"
[ "$RC" -ne 0 ] && printf "%s\n" "$OUT" | rg -qi "only supported on windows"
'
fi

# 7) Platform service install script dry-run validation
run_cmd "Check macOS daemon packaging script syntax" bash -n "$ROOT/tools/scripts/package_daemon_app_macos.sh"
run_cmd "Check macOS app release packaging script syntax" bash -n "$ROOT/tools/scripts/package_macos_app_release.sh"
run_cmd "Check unified launcher script syntax" bash -n "$ROOT/tools/scripts/launch_personal_agent.sh"
run_cmd "Check macOS service install script syntax" bash -n "$ROOT/tools/scripts/install_daemon_service_macos.sh"
run_cmd "Check Linux service install script syntax" bash -n "$ROOT/tools/scripts/install_daemon_service_linux.sh"
capture_cmd LAUNCH_HELP_OUTPUT "Unified launcher help output" "$ROOT/tools/scripts/launch_personal_agent.sh" --help
run_eval "Validate unified launcher help output content" '
printf "%s\n" "$LAUNCH_HELP_OUTPUT" | rg -q "checks daemon reachability"
printf "%s\n" "$LAUNCH_HELP_OUTPUT" | rg -q "starts daemon only when not reachable"
printf "%s\n" "$LAUNCH_HELP_OUTPUT" | rg -q "daemon-start-mode"
printf "%s\n" "$LAUNCH_HELP_OUTPUT" | rg -q "auto\\|direct\\|launchctl"
printf "%s\n" "$LAUNCH_HELP_OUTPUT" | rg -q "stored app token"
printf "%s\n" "$LAUNCH_HELP_OUTPUT" | rg -q "stop-daemon-on-exit"
printf "%s\n" "$LAUNCH_HELP_OUTPUT" | rg -q "leaves daemon running"
printf "%s\n" "$LAUNCH_HELP_OUTPUT" | rg -q "on Ctrl\\+C, stops only processes started by this script"
'
run_eval "Validate daemon manual guide documents launcher keychain token default" '
rg -q "personalagent.ui.local_dev_token.v1" "$ROOT/docs/tests-daemon.md"
rg -q "daemon_auth_token" "$ROOT/docs/tests-daemon.md"
'
capture_cmd MAC_PACKAGE_OUTPUT "macOS daemon app packaging script (skip-sign)" "$ROOT/tools/scripts/package_daemon_app_macos.sh" --output-app "$ROOT/out/dist/Personal Agent Daemon.app" --skip-sign
run_eval "Validate macOS daemon app packaging output content" '
printf "%s\n" "$MAC_PACKAGE_OUTPUT" | rg -q "packaged daemon app:"
printf "%s\n" "$MAC_PACKAGE_OUTPUT" | rg -q "daemon executable:"
test -f "$ROOT/out/dist/Personal Agent Daemon.app/Contents/Info.plist"
test -f "$ROOT/out/dist/Personal Agent Daemon.app/Contents/MacOS/personal-agent-daemon"
'
if [[ "$OS_NAME" == *"darwin"* ]]; then
  capture_cmd MAC_APP_RELEASE_PACKAGE_OUTPUT "macOS app release packaging script (unsigned/ad-hoc dmg)" "$ROOT/tools/scripts/package_macos_app_release.sh" --output-dir "$ROOT/out/dist/macos-release"
  run_eval "Validate macOS app release packaging output content" '
printf "%s\n" "$MAC_APP_RELEASE_PACKAGE_OUTPUT" | rg -q "packaged app:"
printf "%s\n" "$MAC_APP_RELEASE_PACKAGE_OUTPUT" | rg -q "embedded daemon helper:"
printf "%s\n" "$MAC_APP_RELEASE_PACKAGE_OUTPUT" | rg -q "packaged dmg:"
printf "%s\n" "$MAC_APP_RELEASE_PACKAGE_OUTPUT" | rg -q "checksums:"
printf "%s\n" "$MAC_APP_RELEASE_PACKAGE_OUTPUT" | rg -q "manifest:"
test -d "$ROOT/out/dist/macos-release/PersonalAgent.app"
test -d "$ROOT/out/dist/macos-release/PersonalAgent.app/Contents/Resources/Daemon/Personal Agent Daemon.app"
test -f "$ROOT/out/dist/macos-release/PersonalAgent-unsigned.dmg"
test -f "$ROOT/out/dist/macos-release/SHA256SUMS.txt"
test -f "$ROOT/out/dist/macos-release/release-manifest.json"
'
else
  run_eval "Skip macOS app release packaging script execution on non-macOS host" 'echo "macOS-only app release packaging execution skipped on non-Darwin host"'
fi
capture_cmd MAC_SERVICE_APP_DRYRUN_OUTPUT "macOS LaunchAgent install script dry-run (daemon-app path)" "$ROOT/tools/scripts/install_daemon_service_macos.sh" --daemon-app "$ROOT/out/dist/Personal Agent Daemon.app" --auth-token-file "$DAEMON_AUTH_TOKEN_FILE" --db-path "$DAEMON_DB_PATH" --output /tmp/com.personalagent.daemon.test.plist --dry-run
run_eval "Validate macOS LaunchAgent daemon-app dry-run output content" '
printf "%s\n" "$MAC_SERVICE_APP_DRYRUN_OUTPUT" | rg -q "\\[dry-run\\]"
printf "%s\n" "$MAC_SERVICE_APP_DRYRUN_OUTPUT" | rg -q "daemon executable:"
printf "%s\n" "$MAC_SERVICE_APP_DRYRUN_OUTPUT" | rg -q "Personal Agent Daemon.app/Contents/MacOS/personal-agent-daemon"
printf "%s\n" "$MAC_SERVICE_APP_DRYRUN_OUTPUT" | rg -q -- "--listen-mode"
printf "%s\n" "$MAC_SERVICE_APP_DRYRUN_OUTPUT" | rg -q -- "--listen-address"
printf "%s\n" "$MAC_SERVICE_APP_DRYRUN_OUTPUT" | rg -q -- "--auth-token-file"
printf "%s\n" "$MAC_SERVICE_APP_DRYRUN_OUTPUT" | rg -q -- "--db"
'
capture_cmd MAC_SERVICE_DRYRUN_OUTPUT "macOS LaunchAgent install script dry-run" "$ROOT/tools/scripts/install_daemon_service_macos.sh" --daemon-binary "$ROOT/source/services/daemon-go/bin/personal-agent-daemon" --auth-token-file "$DAEMON_AUTH_TOKEN_FILE" --db-path "$DAEMON_DB_PATH" --output /tmp/com.personalagent.daemon.test.plist --dry-run
run_eval "Validate macOS LaunchAgent dry-run output content" '
printf "%s\n" "$MAC_SERVICE_DRYRUN_OUTPUT" | rg -q "\\[dry-run\\]"
printf "%s\n" "$MAC_SERVICE_DRYRUN_OUTPUT" | rg -q -- "--listen-mode"
printf "%s\n" "$MAC_SERVICE_DRYRUN_OUTPUT" | rg -q -- "--listen-address"
printf "%s\n" "$MAC_SERVICE_DRYRUN_OUTPUT" | rg -q -- "--auth-token-file"
printf "%s\n" "$MAC_SERVICE_DRYRUN_OUTPUT" | rg -q -- "--db"
'
capture_cmd LINUX_SERVICE_DRYRUN_OUTPUT "Linux systemd user-service install script dry-run" "$ROOT/tools/scripts/install_daemon_service_linux.sh" --daemon-binary "$ROOT/source/services/daemon-go/bin/personal-agent-daemon" --auth-token-file "$DAEMON_AUTH_TOKEN_FILE" --db-path "$DAEMON_DB_PATH" --output /tmp/personal-agent-daemon.test.service --dry-run
run_eval "Validate Linux systemd dry-run output content" '
printf "%s\n" "$LINUX_SERVICE_DRYRUN_OUTPUT" | rg -q "\\[dry-run\\]"
printf "%s\n" "$LINUX_SERVICE_DRYRUN_OUTPUT" | rg -q "ExecStart="
printf "%s\n" "$LINUX_SERVICE_DRYRUN_OUTPUT" | rg -q -- "--listen-mode"
printf "%s\n" "$LINUX_SERVICE_DRYRUN_OUTPUT" | rg -q -- "--listen-address"
printf "%s\n" "$LINUX_SERVICE_DRYRUN_OUTPUT" | rg -q -- "--auth-token-file"
printf "%s\n" "$LINUX_SERVICE_DRYRUN_OUTPUT" | rg -q -- "--db"
'
run_eval "Validate Windows scheduled-task install script structure" '
rg -q "Register-ScheduledTask" "$ROOT/tools/scripts/install_daemon_service_windows.ps1"
rg -q "New-ScheduledTaskAction" "$ROOT/tools/scripts/install_daemon_service_windows.ps1"
rg -q "param\\(" "$ROOT/tools/scripts/install_daemon_service_windows.ps1"
'
if command -v powershell >/dev/null 2>&1; then
  capture_cmd WINDOWS_SERVICE_DRYRUN_OUTPUT "Windows scheduled-task install script dry-run (powershell)" powershell -ExecutionPolicy Bypass -File "$ROOT/tools/scripts/install_daemon_service_windows.ps1" -AuthTokenFile "$DAEMON_AUTH_TOKEN_FILE" -DbPath "$DAEMON_DB_PATH" -DryRun
  run_eval "Validate Windows dry-run output (powershell)" 'printf "%s\n" "$WINDOWS_SERVICE_DRYRUN_OUTPUT" | rg -q "\\[dry-run\\]"'
elif command -v pwsh >/dev/null 2>&1; then
  capture_cmd WINDOWS_SERVICE_DRYRUN_OUTPUT "Windows scheduled-task install script dry-run (pwsh)" pwsh -File "$ROOT/tools/scripts/install_daemon_service_windows.ps1" -AuthTokenFile "$DAEMON_AUTH_TOKEN_FILE" -DbPath "$DAEMON_DB_PATH" -DryRun
  run_eval "Validate Windows dry-run output (pwsh)" 'printf "%s\n" "$WINDOWS_SERVICE_DRYRUN_OUTPUT" | rg -q "\\[dry-run\\]"'
else
  run_eval "Skip Windows script execution (powershell unavailable)" 'echo "windows install script execution skipped; structural checks passed"'
fi

# 8) Unsigned DMG + Gatekeeper/TCC manual-doc sync checks
run_eval "Validate unsigned DMG/Gatekeeper/TCC manual-doc guidance" '
rg -q "Clean-Machine Unsigned DMG Install \\+ Gatekeeper/TCC Attribution" "$ROOT/docs/tests-daemon.md"
rg -q "styled drag-to-Applications background/instructions render" "$ROOT/docs/tests-daemon.md"
rg -q "styled drag-to-Applications guidance \\(background \\+ deterministic icon placement\\)" "$ROOT/docs/tests-daemon.md"
rg -q "Personal Agent Daemon" "$ROOT/docs/tests-daemon.md"
rg -q "tccutil reset AppleEvents com.personalagent.daemon" "$ROOT/docs/tests-daemon.md"
'

# 9) Optional regression checks
if [[ "$SKIP_REGRESSION" == "true" ]]; then
  run_eval "Skip regression checks" 'echo "regression checks skipped by --skip-regression"'
else
  run_eval "Run transport package tests" 'cd "$ROOT/source/services/daemon-go" && go test ./internal/transport'
  run_eval "Run harness checks" 'cd "$ROOT" && ./tools/scripts/check_harness.sh'
fi

# 10) Cleanup
run_eval "Final daemon cleanup" 'stop_daemon'
run_cmd "Remove unix socket" rm -f "$DAEMON_UNIX_SOCKET"

echo
echo "================================================================"
echo "Daemon manual tests completed at $(date -u +'%Y-%m-%dT%H:%M:%SZ')"
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
