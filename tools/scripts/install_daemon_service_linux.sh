#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
if [[ ! -f "${ROOT}/AGENTS.md" && -f "${SCRIPT_DIR}/../../AGENTS.md" ]]; then
  ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
fi

SERVICE_NAME="personal-agent-daemon"
DAEMON_BINARY="personal-agent-daemon"
LISTEN_MODE="tcp"
LISTEN_ADDRESS="127.0.0.1:7071"
AUTH_TOKEN_FILE=""
DB_PATH="$HOME/.config/personal-agent/runtime.db"
OUTPUT_UNIT="$HOME/.config/systemd/user/${SERVICE_NAME}.service"
DRY_RUN=false

usage() {
  cat <<'EOF'
Usage: install_daemon_service_linux.sh [options]

Options:
  --service-name <value>       systemd user-service name (default: personal-agent-daemon)
  --daemon-binary <path>       Daemon executable path (default: personal-agent-daemon)
  --listen-mode <mode>         Daemon listen mode (default: tcp)
  --listen-address <address>   Daemon listen address (default: 127.0.0.1:7071)
  --auth-token-file <path>     Required daemon auth-token file path
  --db-path <path>             Daemon sqlite path
  --output <path>              systemd unit output path
  --dry-run                    Generate unit content only, do not install/enable
  --help                       Show this help
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --service-name)
      SERVICE_NAME="${2:-}"
      shift 2
      ;;
    --daemon-binary)
      DAEMON_BINARY="${2:-}"
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
      OUTPUT_UNIT="${2:-}"
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

generate_unit() {
  cat <<EOF
[Unit]
Description=Personal Agent Daemon
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=${DAEMON_BINARY} --listen-mode ${LISTEN_MODE} --listen-address ${LISTEN_ADDRESS} --auth-token-file ${AUTH_TOKEN_FILE} --db ${DB_PATH}
Restart=always
RestartSec=2
Environment=HOME=${HOME}
WorkingDirectory=${ROOT}

[Install]
WantedBy=default.target
EOF
}

if [[ "$DRY_RUN" == "true" ]]; then
  echo "[dry-run] would write systemd user unit to: $OUTPUT_UNIT"
  generate_unit
  exit 0
fi

if ! command -v systemctl >/dev/null 2>&1; then
  echo "systemctl not found; cannot install systemd user service" >&2
  exit 1
fi

mkdir -p "$(dirname "$OUTPUT_UNIT")"
generate_unit >"$OUTPUT_UNIT"

systemctl --user daemon-reload
systemctl --user enable --now "${SERVICE_NAME}.service"

echo "installed and enabled systemd user service: ${SERVICE_NAME}.service"
