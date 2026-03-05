#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
if [[ ! -f "${ROOT}/AGENTS.md" && -f "${SCRIPT_DIR}/../../AGENTS.md" ]]; then
  ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
fi

XCODE_PROJECT="${ROOT}/source/apps/macos/app-host/PersonalAgent.xcodeproj"
XCODE_SCHEME="PersonalAgent"
APP_NAME="PersonalAgent.app"
APP_EXECUTABLE_NAME="PersonalAgent"
APP_DERIVED_DATA_PATH="${APP_DERIVED_DATA_PATH:-${ROOT}/out/build/xcode-derived-data}"

DAEMON_LISTEN_MODE="${DAEMON_LISTEN_MODE:-tcp}"
DAEMON_ADDRESS="${DAEMON_ADDRESS:-127.0.0.1:7071}"
DAEMON_AUTH_TOKEN="${DAEMON_AUTH_TOKEN:-}"
DAEMON_AUTH_TOKEN_FILE="${DAEMON_AUTH_TOKEN_FILE:-}"
DAEMON_UI_KEYCHAIN_SERVICE="${DAEMON_UI_KEYCHAIN_SERVICE:-personalagent.ui.local_dev_token.v1}"
DAEMON_UI_KEYCHAIN_ACCOUNT="${DAEMON_UI_KEYCHAIN_ACCOUNT:-daemon_auth_token}"
DAEMON_DEFAULT_LOCAL_TOKEN_FILE="${DAEMON_DEFAULT_LOCAL_TOKEN_FILE:-${HOME}/Library/Application Support/personal-agent/control/local-dev.control.token}"
DAEMON_DB_PATH="${DAEMON_DB_PATH:-${HOME}/Library/Application Support/personal-agent/runtime.db}"
DAEMON_BINARY_OVERRIDE="${DAEMON_BINARY:-}"
DAEMON_START_MODE="${DAEMON_START_MODE:-auto}"
DAEMON_LAUNCHCTL_LOG_PATH="${DAEMON_LAUNCHCTL_LOG_PATH:-${HOME}/Library/Logs/personal-agent/launch-personal-agent-daemon.log}"
DAEMON_STOP_ON_EXIT="${DAEMON_STOP_ON_EXIT:-false}"

APP_BUNDLE_OVERRIDE="${APP_BUNDLE:-}"
NO_APP_LAUNCH=false

GO_BIN="${GO_BIN:-go}"
XCODEBUILD_BIN="${XCODEBUILD_BIN:-xcodebuild}"
XCODEGEN_BIN="${XCODEGEN_BIN:-xcodegen}"

APP_PID=""
DAEMON_PID=""
DAEMON_LAUNCH_LABEL=""
DAEMON_LAUNCH_PLIST=""
DAEMON_LAUNCH_DOMAIN=""
APP_STARTED_BY_SCRIPT=0
DAEMON_STARTED_BY_SCRIPT=0
DAEMON_STARTED_BY_LAUNCHCTL=0
CLEANUP_DONE=0
DAEMON_STOP_ON_EXIT_RESOLVED=0
FORCE_STOP_DAEMON_ON_SIGNAL=0
DAEMON_AUTH_TOKEN_RESOLVED=""
DAEMON_AUTH_SOURCE="unset"
declare -a DAEMON_AUTH_CANDIDATE_SOURCES=()
declare -a DAEMON_AUTH_CANDIDATE_TOKENS=()

usage() {
  cat <<'EOF'
Usage: launch_personal_agent.sh [options]

Options:
  --app-bundle <path>          Explicit app bundle path to launch.
  --no-app-launch              Ensure daemon is running but do not launch app.

  --daemon-binary <path>       Explicit daemon executable path.
  --daemon-listen-mode <mode>  Daemon listen mode (default: tcp).
  --daemon-address <address>   Daemon listen address (default: 127.0.0.1:7071).
  --daemon-auth-token <token>  Daemon auth token (defaults to stored app token in Keychain; launcher also probes the canonical local-dev token file for existing-daemon parity; fallback daemon-test-token).
  --daemon-auth-token-file <path>
                               Daemon auth token file (preferred over token literal).
  --daemon-db-path <path>      Daemon sqlite path.
  --daemon-start-mode <mode>   Daemon start mode: auto|direct|launchctl (default: auto).
  --stop-daemon-on-exit        Stop daemon on launcher exit (default leaves daemon running).

  --help                       Show this help.

Behavior:
  0) rebuilds daemon binary at out/bin/personal-agent-daemon on every launch
  1) regenerates the Xcode project and rebuilds PersonalAgent Debug app bundle into out/build/xcode-derived-data before launch (unless --no-app-launch)
  2) verifies app resources are packaged (Assets.car + AppIcon.icns)
  3) checks daemon reachability
  4) starts daemon only when not reachable (auto prefers launchctl on macOS)
  5) launches app bundle executable
  6) by default leaves daemon running when launcher exits; pass --stop-daemon-on-exit to tear it down
  7) on Ctrl+C, stops only processes started by this script
EOF
}

log() {
  printf '[launch] %s\n' "$*"
}

resolve_bool_flag() {
  local raw
  raw="$(printf '%s' "${1:-}" | tr '[:upper:]' '[:lower:]' | xargs)"
  case "${raw}" in
    1|true|yes|on)
      printf '1\n'
      ;;
    0|false|no|off|"")
      printf '0\n'
      ;;
    *)
      return 1
      ;;
  esac
}

trim_value() {
  printf '%s' "${1:-}" | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//'
}

read_token_from_file() {
  local token_file="$1"
  if [[ ! -f "${token_file}" ]]; then
    return 1
  fi
  trim_value "$(cat "${token_file}")"
}

read_stored_ui_token_from_keychain() {
  if [[ "$(uname -s)" != "Darwin" ]] || ! command -v security >/dev/null 2>&1; then
    return 1
  fi
  local stored
  stored="$(
    security find-generic-password \
      -w \
      -s "${DAEMON_UI_KEYCHAIN_SERVICE}" \
      -a "${DAEMON_UI_KEYCHAIN_ACCOUNT}" 2>/dev/null || true
  )"
  stored="$(trim_value "${stored}")"
  if [[ -z "${stored}" ]]; then
    return 1
  fi
  printf '%s\n' "${stored}"
}

read_default_local_token_file() {
  local token_file
  token_file="$(trim_value "${DAEMON_DEFAULT_LOCAL_TOKEN_FILE}")"
  if [[ -z "${token_file}" ]]; then
    return 1
  fi
  local stored
  stored="$(read_token_from_file "${token_file}")" || return 1
  stored="$(trim_value "${stored}")"
  if [[ -z "${stored}" ]]; then
    return 1
  fi
  printf '%s\n' "${stored}"
}

persist_ui_token_to_keychain() {
  local token="$1"
  if [[ -z "${token}" ]]; then
    return 0
  fi
  if [[ "$(uname -s)" != "Darwin" ]] || ! command -v security >/dev/null 2>&1; then
    return 0
  fi
  if security add-generic-password \
    -U \
    -s "${DAEMON_UI_KEYCHAIN_SERVICE}" \
    -a "${DAEMON_UI_KEYCHAIN_ACCOUNT}" \
    -w "${token}" >/dev/null 2>&1; then
    return 0
  fi
  log "warning: failed to persist daemon auth token to Keychain; app may require manual Assistant Access Token save."
  return 0
}

set_resolved_daemon_auth_material() {
  local source="$1"
  local token="$2"
  DAEMON_AUTH_TOKEN="${token}"
  DAEMON_AUTH_TOKEN_RESOLVED="${token}"
  DAEMON_AUTH_SOURCE="${source}"
}

register_daemon_auth_candidate() {
  local source="$1"
  local token
  token="$(trim_value "${2:-}")"
  if [[ -z "${token}" ]]; then
    return 0
  fi

  if [[ "${#DAEMON_AUTH_CANDIDATE_TOKENS[@]}" -gt 0 ]]; then
    local existing
    for existing in "${DAEMON_AUTH_CANDIDATE_TOKENS[@]}"; do
      if [[ "${existing}" == "${token}" ]]; then
        return 0
      fi
    done
  fi

  DAEMON_AUTH_CANDIDATE_SOURCES+=("${source}")
  DAEMON_AUTH_CANDIDATE_TOKENS+=("${token}")
}

collect_default_daemon_auth_candidates() {
  DAEMON_AUTH_CANDIDATE_SOURCES=()
  DAEMON_AUTH_CANDIDATE_TOKENS=()

  local stored_token=""
  if stored_token="$(read_stored_ui_token_from_keychain)"; then
    register_daemon_auth_candidate "stored_ui_token_keychain" "${stored_token}"
  fi

  local default_local_token=""
  if default_local_token="$(read_default_local_token_file)"; then
    register_daemon_auth_candidate "default_local_token_file" "${default_local_token}"
  fi

  register_daemon_auth_candidate "default_fallback" "daemon-test-token"
}

resolve_daemon_auth_material() {
  if [[ -n "${DAEMON_AUTH_TOKEN_FILE}" ]]; then
    local file_token
    file_token="$(read_token_from_file "${DAEMON_AUTH_TOKEN_FILE}")"
    if [[ -z "${file_token}" ]]; then
      echo "daemon auth token file is empty: ${DAEMON_AUTH_TOKEN_FILE}" >&2
      exit 1
    fi
    set_resolved_daemon_auth_material "token_file" "${file_token}"
    return 0
  fi

  local explicit_token
  explicit_token="$(trim_value "${DAEMON_AUTH_TOKEN}")"
  if [[ -n "${explicit_token}" ]]; then
    set_resolved_daemon_auth_material "token_flag" "${explicit_token}"
    return 0
  fi

  collect_default_daemon_auth_candidates
  if [[ "${#DAEMON_AUTH_CANDIDATE_TOKENS[@]}" -eq 0 ]]; then
    echo "failed to resolve daemon auth token candidates" >&2
    exit 1
  fi

  set_resolved_daemon_auth_material \
    "${DAEMON_AUTH_CANDIDATE_SOURCES[0]}" \
    "${DAEMON_AUTH_CANDIDATE_TOKENS[0]}"
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --app-bundle)
      APP_BUNDLE_OVERRIDE="${2:-}"
      shift 2
      ;;
    --no-app-launch)
      NO_APP_LAUNCH=true
      shift
      ;;
    --daemon-binary)
      DAEMON_BINARY_OVERRIDE="${2:-}"
      shift 2
      ;;
    --daemon-listen-mode)
      DAEMON_LISTEN_MODE="${2:-}"
      shift 2
      ;;
    --daemon-address)
      DAEMON_ADDRESS="${2:-}"
      shift 2
      ;;
    --daemon-auth-token)
      DAEMON_AUTH_TOKEN="${2:-}"
      shift 2
      ;;
    --daemon-auth-token-file)
      DAEMON_AUTH_TOKEN_FILE="${2:-}"
      shift 2
      ;;
    --daemon-db-path)
      DAEMON_DB_PATH="${2:-}"
      shift 2
      ;;
    --daemon-start-mode)
      DAEMON_START_MODE="${2:-}"
      shift 2
      ;;
    --stop-daemon-on-exit)
      DAEMON_STOP_ON_EXIT=true
      shift
      ;;
    --help|-h)
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

if [[ "${DAEMON_LISTEN_MODE}" != "tcp" ]]; then
  echo "this launcher currently supports tcp daemon mode only (received: ${DAEMON_LISTEN_MODE})" >&2
  exit 2
fi

if ! DAEMON_STOP_ON_EXIT_RESOLVED="$(resolve_bool_flag "${DAEMON_STOP_ON_EXIT}")"; then
  echo "invalid DAEMON_STOP_ON_EXIT value: ${DAEMON_STOP_ON_EXIT} (expected true|false)" >&2
  exit 2
fi

if [[ -n "${DAEMON_AUTH_TOKEN_FILE}" && ! -f "${DAEMON_AUTH_TOKEN_FILE}" ]]; then
  echo "daemon auth token file not found: ${DAEMON_AUTH_TOKEN_FILE}" >&2
  exit 1
fi

resolve_daemon_auth_material
persist_ui_token_to_keychain "${DAEMON_AUTH_TOKEN_RESOLVED}"

resolve_built_app_bundle() {
  local bundle="${APP_DERIVED_DATA_PATH}/Build/Products/Debug/${APP_NAME}"
  if [[ -d "${bundle}" ]]; then
    printf '%s\n' "${bundle}"
    return 0
  fi
  return 1
}

build_app_bundle() {
  if ! command -v "${XCODEGEN_BIN}" >/dev/null 2>&1; then
    echo "xcodegen is required to build the app bundle but was not found (command: ${XCODEGEN_BIN})" >&2
    exit 1
  fi
  mkdir -p "${APP_DERIVED_DATA_PATH}"
  log "generating Xcode project via ${XCODEGEN_BIN}"
  (
    cd "${ROOT}/source/apps/macos/app-host"
    "${XCODEGEN_BIN}" generate >/dev/null
  )
  log "building ${APP_NAME} via xcodebuild (derivedDataPath=${APP_DERIVED_DATA_PATH})"
  "${XCODEBUILD_BIN}" \
    -project "${XCODE_PROJECT}" \
    -scheme "${XCODE_SCHEME}" \
    -configuration Debug \
    -derivedDataPath "${APP_DERIVED_DATA_PATH}" \
    CODE_SIGNING_ALLOWED=NO \
    build >/dev/null
}

resolve_app_bundle() {
  if [[ -n "${APP_BUNDLE_OVERRIDE}" ]]; then
    if [[ ! -d "${APP_BUNDLE_OVERRIDE}" ]]; then
      echo "app bundle not found: ${APP_BUNDLE_OVERRIDE}" >&2
      exit 1
    fi
    printf '%s\n' "${APP_BUNDLE_OVERRIDE}"
    return 0
  fi

  local bundle=""
  if bundle="$(resolve_built_app_bundle)"; then
    printf '%s\n' "${bundle}"
    return 0
  fi

  if [[ -d "/Applications/${APP_NAME}" ]]; then
    printf '%s\n' "/Applications/${APP_NAME}"
    return 0
  fi

  echo "could not find ${APP_NAME}; pass --app-bundle or build the app via xcodebuild" >&2
  exit 1
}

verify_app_bundle_resources() {
  local bundle="$1"
  local assets_car="${bundle}/Contents/Resources/Assets.car"
  local app_icon="${bundle}/Contents/Resources/AppIcon.icns"

  if [[ ! -f "${assets_car}" ]]; then
    echo "app bundle is missing compiled asset catalog: ${assets_car}" >&2
    echo "run this launcher without --app-bundle to rebuild a fresh app bundle." >&2
    exit 1
  fi
  if [[ ! -f "${app_icon}" ]]; then
    echo "app bundle is missing app icon resource: ${app_icon}" >&2
    echo "run this launcher without --app-bundle to rebuild a fresh app bundle." >&2
    exit 1
  fi
}

resolve_app_executable() {
  local bundle="$1"
  local executable="${bundle}/Contents/MacOS/${APP_EXECUTABLE_NAME}"
  if [[ ! -x "${executable}" ]]; then
    echo "app executable not found or not executable: ${executable}" >&2
    exit 1
  fi
  printf '%s\n' "${executable}"
}

build_daemon_binary() {
  local output="${ROOT}/out/bin/personal-agent-daemon"
  mkdir -p "$(dirname "${output}")"
  log "building daemon binary at ${output}" >&2
  (
    cd "${ROOT}/source/services/daemon-go"
    "${GO_BIN}" build -o "${output}" ./cmd/personal-agent-daemon
  )
  printf '%s\n' "${output}"
}

resolve_daemon_binary() {
  if [[ -n "${DAEMON_BINARY_OVERRIDE}" ]]; then
    if [[ ! -x "${DAEMON_BINARY_OVERRIDE}" ]]; then
      echo "daemon binary not executable: ${DAEMON_BINARY_OVERRIDE}" >&2
      exit 1
    fi
    printf '%s\n' "${DAEMON_BINARY_OVERRIDE}"
    return 0
  fi

  # Always rebuild daemon for local runs so launcher uses latest code.
  build_daemon_binary
}

resolve_daemon_start_mode() {
  local requested
  requested="$(printf '%s' "${DAEMON_START_MODE}" | tr '[:upper:]' '[:lower:]')"
  case "${requested}" in
    auto)
      if [[ "$(uname -s)" == "Darwin" ]] && command -v launchctl >/dev/null 2>&1; then
        printf '%s\n' "launchctl"
      else
        printf '%s\n' "direct"
      fi
      ;;
    direct|launchctl)
      if [[ "${requested}" == "launchctl" ]] && ! command -v launchctl >/dev/null 2>&1; then
        echo "launchctl start mode requested but launchctl is unavailable on this host" >&2
        exit 2
      fi
      printf '%s\n' "${requested}"
      ;;
    *)
      echo "unsupported daemon start mode: ${DAEMON_START_MODE} (expected auto|direct|launchctl)" >&2
      exit 2
      ;;
  esac
}

xml_escape() {
  local value="$1"
  value="${value//&/&amp;}"
  value="${value//</&lt;}"
  value="${value//>/&gt;}"
  printf '%s' "${value}"
}

write_launchctl_plist() {
  local plist_path="$1"
  local label="$2"
  local log_path="$3"
  shift 3
  local daemon_args=("$@")

  mkdir -p "$(dirname "${plist_path}")"
  : >"${plist_path}"
  {
    echo '<?xml version="1.0" encoding="UTF-8"?>'
    echo '<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">'
    echo '<plist version="1.0">'
    echo '<dict>'
    echo "  <key>Label</key>"
    printf '  <string>%s</string>\n' "$(xml_escape "${label}")"
    echo "  <key>ProgramArguments</key>"
    echo "  <array>"
    local arg
    for arg in "${daemon_args[@]}"; do
      printf '    <string>%s</string>\n' "$(xml_escape "${arg}")"
    done
    echo "  </array>"
    echo "  <key>RunAtLoad</key>"
    echo "  <true/>"
    echo "  <key>KeepAlive</key>"
    echo "  <false/>"
    echo "  <key>ProcessType</key>"
    echo "  <string>Background</string>"
    echo "  <key>EnvironmentVariables</key>"
    echo "  <dict>"
    echo "    <key>HOME</key>"
    printf '    <string>%s</string>\n' "$(xml_escape "${HOME}")"
    echo "    <key>PATH</key>"
    printf '    <string>%s</string>\n' "$(xml_escape "${PATH}")"
    echo "  </dict>"
    echo "  <key>StandardOutPath</key>"
    printf '  <string>%s</string>\n' "$(xml_escape "${log_path}")"
    echo "  <key>StandardErrorPath</key>"
    printf '  <string>%s</string>\n' "$(xml_escape "${log_path}")"
    echo "</dict>"
    echo "</plist>"
  } >"${plist_path}"
}

start_daemon_via_launchctl() {
  local log_path="$1"
  shift
  local daemon_args=("$@")

  DAEMON_LAUNCH_DOMAIN="gui/$(id -u)"
  DAEMON_LAUNCH_LABEL="com.personalagent.dev.launcher.$(id -u).$(date +%s)"
  DAEMON_LAUNCH_PLIST="${ROOT}/out/logs/${DAEMON_LAUNCH_LABEL}.plist"

  write_launchctl_plist "${DAEMON_LAUNCH_PLIST}" "${DAEMON_LAUNCH_LABEL}" "${log_path}" "${daemon_args[@]}"
  launchctl bootout "${DAEMON_LAUNCH_DOMAIN}" "${DAEMON_LAUNCH_PLIST}" >/dev/null 2>&1 || true
  launchctl bootstrap "${DAEMON_LAUNCH_DOMAIN}" "${DAEMON_LAUNCH_PLIST}"
  launchctl enable "${DAEMON_LAUNCH_DOMAIN}/${DAEMON_LAUNCH_LABEL}" >/dev/null 2>&1 || true
  launchctl kickstart -k "${DAEMON_LAUNCH_DOMAIN}/${DAEMON_LAUNCH_LABEL}"
  DAEMON_STARTED_BY_SCRIPT=1
  DAEMON_STARTED_BY_LAUNCHCTL=1
  DAEMON_PID=""
}

stop_launchctl_job() {
  if [[ -n "${DAEMON_LAUNCH_DOMAIN}" && -n "${DAEMON_LAUNCH_PLIST}" ]]; then
    launchctl bootout "${DAEMON_LAUNCH_DOMAIN}" "${DAEMON_LAUNCH_PLIST}" >/dev/null 2>&1 || true
  elif [[ -n "${DAEMON_LAUNCH_LABEL}" ]]; then
    launchctl remove "${DAEMON_LAUNCH_LABEL}" >/dev/null 2>&1 || true
  fi
  if [[ -n "${DAEMON_LAUNCH_PLIST}" ]]; then
    rm -f "${DAEMON_LAUNCH_PLIST}" >/dev/null 2>&1 || true
  fi
  DAEMON_STARTED_BY_LAUNCHCTL=0
  DAEMON_LAUNCH_DOMAIN=""
  DAEMON_LAUNCH_LABEL=""
  DAEMON_LAUNCH_PLIST=""
}

daemon_http_status_for_token() {
  local token="$1"
  curl -sS -o /dev/null -w "%{http_code}" --max-time 1 \
    -H "Authorization: Bearer ${token}" \
    "http://${DAEMON_ADDRESS}/v1/capabilities/smoke" || true
}

daemon_http_status() {
  daemon_http_status_for_token "${DAEMON_AUTH_TOKEN_RESOLVED}"
}

daemon_http_status_no_auth() {
  curl -sS -o /dev/null -w "%{http_code}" --max-time 1 \
    "http://${DAEMON_ADDRESS}/v1/capabilities/smoke" || true
}

daemon_has_existing_listener() {
  local status
  status="$(daemon_http_status_no_auth)"
  case "${status}" in
    200|401|403)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

daemon_is_reachable() {
  daemon_is_reachable_with_token "${DAEMON_AUTH_TOKEN_RESOLVED}"
}

daemon_is_reachable_with_token() {
  local token="$1"
  local status
  status="$(daemon_http_status_for_token "${token}")"
  case "${status}" in
    200)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

reconcile_existing_listener_auth() {
  if [[ "${DAEMON_AUTH_SOURCE}" == "token_file" || "${DAEMON_AUTH_SOURCE}" == "token_flag" ]]; then
    return 1
  fi

  local index
  for index in "${!DAEMON_AUTH_CANDIDATE_TOKENS[@]}"; do
    local source="${DAEMON_AUTH_CANDIDATE_SOURCES[index]}"
    local token="${DAEMON_AUTH_CANDIDATE_TOKENS[index]}"
    if [[ "${token}" == "${DAEMON_AUTH_TOKEN_RESOLVED}" ]]; then
      continue
    fi
    if daemon_is_reachable_with_token "${token}"; then
      set_resolved_daemon_auth_material "${source}" "${token}"
      persist_ui_token_to_keychain "${DAEMON_AUTH_TOKEN_RESOLVED}"
      log "existing daemon authenticated via ${DAEMON_AUTH_SOURCE}; synced Keychain token for app parity"
      return 0
    fi
  done

  return 1
}

wait_for_daemon() {
  local attempts="${1:-60}"
  local sleep_seconds="${2:-0.25}"
  local i=0
  while (( i < attempts )); do
    if daemon_is_reachable; then
      return 0
    fi
    if [[ -n "${DAEMON_PID}" ]] && ! kill -0 "${DAEMON_PID}" 2>/dev/null; then
      return 1
    fi
    sleep "${sleep_seconds}"
    i=$((i + 1))
  done
  return 1
}

stop_pid_gracefully() {
  local pid="$1"
  local name="$2"
  if ! kill -0 "${pid}" 2>/dev/null; then
    return 0
  fi
  log "stopping ${name} (pid=${pid})"
  kill -TERM "${pid}" 2>/dev/null || true
  for _ in $(seq 1 25); do
    if ! kill -0 "${pid}" 2>/dev/null; then
      return 0
    fi
    sleep 0.2
  done
  if kill -0 "${pid}" 2>/dev/null; then
    log "${name} still running; forcing stop (pid=${pid})"
    kill -KILL "${pid}" 2>/dev/null || true
  fi
}

cleanup() {
  if [[ "${CLEANUP_DONE}" -eq 1 ]]; then
    return
  fi
  CLEANUP_DONE=1

  if [[ "${APP_STARTED_BY_SCRIPT}" -eq 1 && -n "${APP_PID}" ]]; then
    stop_pid_gracefully "${APP_PID}" "app"
  fi
  local stop_daemon=0
  if [[ "${DAEMON_STOP_ON_EXIT_RESOLVED}" -eq 1 || "${FORCE_STOP_DAEMON_ON_SIGNAL}" -eq 1 ]]; then
    stop_daemon=1
  fi
  if [[ "${DAEMON_STARTED_BY_SCRIPT}" -eq 1 && "${stop_daemon}" -eq 1 && "${DAEMON_STARTED_BY_LAUNCHCTL}" -eq 1 ]]; then
    log "stopping daemon launchctl job ${DAEMON_LAUNCH_LABEL}"
    stop_launchctl_job
  elif [[ "${DAEMON_STARTED_BY_SCRIPT}" -eq 1 && "${stop_daemon}" -eq 1 && -n "${DAEMON_PID}" ]]; then
    stop_pid_gracefully "${DAEMON_PID}" "daemon"
  elif [[ "${DAEMON_STARTED_BY_SCRIPT}" -eq 1 ]]; then
    log "leaving daemon running (stop-on-exit disabled)"
  fi
}

on_signal() {
  local signal_name="${1:-INT}"
  if [[ "${signal_name}" == "INT" ]]; then
    FORCE_STOP_DAEMON_ON_SIGNAL=1
  fi
  log "received ${signal_name}; cleaning up"
  cleanup
  exit 130
}

on_exit() {
  cleanup
}

trap 'on_signal INT' INT
trap 'on_signal TERM' TERM
trap on_exit EXIT

DAEMON_BINARY_RESOLVED="$(resolve_daemon_binary)"
DAEMON_START_MODE_RESOLVED="$(resolve_daemon_start_mode)"
APP_BUNDLE_RESOLVED=""
APP_EXECUTABLE=""

if [[ "${NO_APP_LAUNCH}" != "true" ]]; then
  build_app_bundle
  APP_BUNDLE_RESOLVED="$(resolve_app_bundle)"
  verify_app_bundle_resources "${APP_BUNDLE_RESOLVED}"
  APP_EXECUTABLE="$(resolve_app_executable "${APP_BUNDLE_RESOLVED}")"
fi

if daemon_is_reachable; then
  log "daemon already reachable at ${DAEMON_ADDRESS}; reusing existing process (latest binary built at ${DAEMON_BINARY_RESOLVED}, auth=${DAEMON_AUTH_SOURCE})"
else
  if daemon_has_existing_listener; then
    if reconcile_existing_listener_auth; then
      log "daemon already reachable at ${DAEMON_ADDRESS}; reusing existing process (latest binary built at ${DAEMON_BINARY_RESOLVED}, auth=${DAEMON_AUTH_SOURCE})"
    else
      echo "daemon is already listening at ${DAEMON_ADDRESS}, but auth token mismatch prevents access." >&2
      echo "launcher tried stored local auth sources. Pass --daemon-auth-token-file, or stop the existing daemon and relaunch." >&2
      exit 1
    fi
  fi

  if ! daemon_is_reachable; then
    ACTIVE_DAEMON_LOG_PATH=""
    mkdir -p "$(dirname "${DAEMON_DB_PATH}")"
    DAEMON_LOG_PATH="${ROOT}/out/logs/launch-personal-agent-daemon.log"
    ACTIVE_DAEMON_LOG_PATH="${DAEMON_LOG_PATH}"
    mkdir -p "$(dirname "${DAEMON_LOG_PATH}")"
    mkdir -p "$(dirname "${DAEMON_LAUNCHCTL_LOG_PATH}")"

    log "starting daemon: ${DAEMON_BINARY_RESOLVED} (mode=${DAEMON_START_MODE_RESOLVED}, auth=${DAEMON_AUTH_SOURCE})"
    daemon_cmd=(
      "${DAEMON_BINARY_RESOLVED}"
      --listen-mode "${DAEMON_LISTEN_MODE}"
      --listen-address "${DAEMON_ADDRESS}"
      --db "${DAEMON_DB_PATH}"
    )
    if [[ -n "${DAEMON_AUTH_TOKEN_FILE}" ]]; then
      daemon_cmd+=(--auth-token-file "${DAEMON_AUTH_TOKEN_FILE}")
    else
      daemon_cmd+=(--auth-token "${DAEMON_AUTH_TOKEN}")
    fi

    if [[ "${DAEMON_START_MODE_RESOLVED}" == "launchctl" ]]; then
      if start_daemon_via_launchctl "${DAEMON_LAUNCHCTL_LOG_PATH}" "${daemon_cmd[@]}"; then
        ACTIVE_DAEMON_LOG_PATH="${DAEMON_LAUNCHCTL_LOG_PATH}"
        log "daemon launchctl label=${DAEMON_LAUNCH_LABEL} (log=${DAEMON_LAUNCHCTL_LOG_PATH})"
      else
        log "launchctl daemon start failed; falling back to direct spawn"
        stop_launchctl_job
        DAEMON_START_MODE_RESOLVED="direct"
      fi
    fi

    if [[ "${DAEMON_START_MODE_RESOLVED}" == "direct" ]]; then
      ACTIVE_DAEMON_LOG_PATH="${DAEMON_LOG_PATH}"
      "${daemon_cmd[@]}" >"${DAEMON_LOG_PATH}" 2>&1 &
      DAEMON_PID="$!"
      DAEMON_STARTED_BY_SCRIPT=1
      DAEMON_STARTED_BY_LAUNCHCTL=0
      log "daemon pid=${DAEMON_PID} (log=${DAEMON_LOG_PATH})"
    fi

    if ! wait_for_daemon && [[ "${DAEMON_STARTED_BY_LAUNCHCTL}" -eq 1 ]]; then
      log "launchctl daemon did not become reachable; falling back to direct spawn"
      stop_launchctl_job
      ACTIVE_DAEMON_LOG_PATH="${DAEMON_LOG_PATH}"
      "${daemon_cmd[@]}" >"${DAEMON_LOG_PATH}" 2>&1 &
      DAEMON_PID="$!"
      DAEMON_STARTED_BY_SCRIPT=1
      DAEMON_STARTED_BY_LAUNCHCTL=0
      log "daemon pid=${DAEMON_PID} (log=${DAEMON_LOG_PATH})"
    fi

    if ! wait_for_daemon; then
      echo "daemon failed to become reachable at ${DAEMON_ADDRESS}" >&2
      if [[ -n "${ACTIVE_DAEMON_LOG_PATH}" && -f "${ACTIVE_DAEMON_LOG_PATH}" ]]; then
        echo "---- daemon log tail ----" >&2
        tail -n 80 "${ACTIVE_DAEMON_LOG_PATH}" >&2 || true
        echo "-------------------------" >&2
      fi
      if [[ "${DAEMON_STARTED_BY_LAUNCHCTL}" -eq 1 ]]; then
        stop_launchctl_job
      fi
      exit 1
    fi
    log "daemon is reachable"
  fi
fi

if [[ "${NO_APP_LAUNCH}" == "true" ]]; then
  log "daemon check/start complete; app launch skipped (--no-app-launch)"
  exit 0
fi

log "launching app: ${APP_BUNDLE_RESOLVED}"
"${APP_EXECUTABLE}" &
APP_PID="$!"
APP_STARTED_BY_SCRIPT=1
log "app pid=${APP_PID}"

wait "${APP_PID}" || true
log "app exited"
