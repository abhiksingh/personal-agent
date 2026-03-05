#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
if [[ ! -f "${ROOT}/AGENTS.md" && -f "${SCRIPT_DIR}/../../AGENTS.md" ]]; then
  ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
fi

LABEL="com.personalagent.daemon"
DEFAULT_DAEMON_APP="$HOME/Applications/Personal Agent Daemon.app"
DAEMON_EXECUTABLE_NAME="personal-agent-daemon"
DAEMON_BINARY=""
DAEMON_APP=""
LISTEN_MODE="tcp"
LISTEN_ADDRESS="127.0.0.1:7071"
AUTH_TOKEN_FILE=""
DB_PATH="$HOME/Library/Application Support/personal-agent/runtime.db"
OUTPUT_PLIST="$HOME/Library/LaunchAgents/${LABEL}.plist"
LOG_DIR="$HOME/Library/Logs/personal-agent"
STDOUT_LOG_PATH="$LOG_DIR/daemon-service-macos.out.log"
STDERR_LOG_PATH="$LOG_DIR/daemon-service-macos.err.log"
DRY_RUN=false

usage() {
  cat <<'EOF'
Usage: install_daemon_service_macos.sh [options]

Options:
  --label <value>              LaunchAgent label (default: com.personalagent.daemon)
  --daemon-binary <path>       Daemon executable path override
  --daemon-app <path>          Daemon .app bundle path (preferred over codex-hosted process chain)
  --listen-mode <mode>         Daemon listen mode (default: tcp)
  --listen-address <address>   Daemon listen address (default: 127.0.0.1:7071)
  --auth-token-file <path>     Required daemon auth-token file path
  --db-path <path>             Daemon sqlite path
  --output <path>              LaunchAgent plist output path
  --dry-run                    Generate plist content only, do not install/load
  --help                       Show this help
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --label)
      LABEL="${2:-}"
      shift 2
      ;;
    --daemon-binary)
      DAEMON_BINARY="${2:-}"
      shift 2
      ;;
    --daemon-app)
      DAEMON_APP="${2:-}"
      shift 2
      ;;
    --listen-mode)
      LISTEN_MODE="${2:-}"
      shift 2
      ;;
    --listen-address)
      LISTEN_ADDRESS="${2:-}"
      shift 2
      ;;
    --auth-token-file)
      AUTH_TOKEN_FILE="${2:-}"
      shift 2
      ;;
    --db-path)
      DB_PATH="${2:-}"
      shift 2
      ;;
    --output)
      OUTPUT_PLIST="${2:-}"
      shift 2
      ;;
    --dry-run)
      DRY_RUN=true
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

if [[ -z "${AUTH_TOKEN_FILE// }" ]]; then
  echo "--auth-token-file is required" >&2
  exit 2
fi

if [[ -n "${DAEMON_BINARY// }" && -n "${DAEMON_APP// }" ]]; then
  echo "provide only one of --daemon-binary or --daemon-app" >&2
  exit 2
fi

resolve_daemon_binary() {
  if [[ -n "${DAEMON_BINARY// }" ]]; then
    printf '%s' "${DAEMON_BINARY}"
    return 0
  fi

  if [[ -n "${DAEMON_APP// }" ]]; then
    local app_binary="${DAEMON_APP}/Contents/MacOS/${DAEMON_EXECUTABLE_NAME}"
    if [[ ! -f "${app_binary}" ]]; then
      echo "daemon app executable not found: ${app_binary}" >&2
      exit 1
    fi
    printf '%s' "${app_binary}"
    return 0
  fi

  local preferred_app_binary="${DEFAULT_DAEMON_APP}/Contents/MacOS/${DAEMON_EXECUTABLE_NAME}"
  if [[ -f "${preferred_app_binary}" ]]; then
    printf '%s' "${preferred_app_binary}"
    return 0
  fi

  printf '%s' "${DAEMON_EXECUTABLE_NAME}"
}

DAEMON_BINARY="$(resolve_daemon_binary)"

generate_plist() {
  cat <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>${LABEL}</string>
  <key>ProgramArguments</key>
  <array>
    <string>${DAEMON_BINARY}</string>
    <string>--listen-mode</string>
    <string>${LISTEN_MODE}</string>
    <string>--listen-address</string>
    <string>${LISTEN_ADDRESS}</string>
    <string>--auth-token-file</string>
    <string>${AUTH_TOKEN_FILE}</string>
    <string>--db</string>
    <string>${DB_PATH}</string>
  </array>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <true/>
  <key>ProcessType</key>
  <string>Background</string>
  <key>EnvironmentVariables</key>
  <dict>
    <key>HOME</key>
    <string>${HOME}</string>
    <key>PATH</key>
    <string>${PATH}</string>
  </dict>
  <key>StandardOutPath</key>
  <string>${STDOUT_LOG_PATH}</string>
  <key>StandardErrorPath</key>
  <string>${STDERR_LOG_PATH}</string>
</dict>
</plist>
EOF
}

if [[ "$DRY_RUN" == "true" ]]; then
  echo "[dry-run] would write LaunchAgent plist to: $OUTPUT_PLIST"
  echo "[dry-run] daemon executable: $DAEMON_BINARY"
  generate_plist
  exit 0
fi

mkdir -p "$(dirname "$OUTPUT_PLIST")"
mkdir -p "$LOG_DIR"
generate_plist >"$OUTPUT_PLIST"

LAUNCHD_DOMAIN="gui/$(id -u)"
launchctl bootout "$LAUNCHD_DOMAIN" "$OUTPUT_PLIST" >/dev/null 2>&1 || true
launchctl bootstrap "$LAUNCHD_DOMAIN" "$OUTPUT_PLIST"
launchctl enable "${LAUNCHD_DOMAIN}/${LABEL}"

echo "installed and loaded LaunchAgent: $OUTPUT_PLIST"
