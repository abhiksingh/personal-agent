#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
if [[ ! -f "${REPO_ROOT}/AGENTS.md" && -f "${SCRIPT_DIR}/../../AGENTS.md" ]]; then
  REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
fi
cd "${REPO_ROOT}"

MODE="${1:-all}"
EXTRA_ARG="${2:-}"

usage() {
  echo "usage: $0 [setup|serve|all|all-skip-missing-env] [--skip-missing-env]" >&2
}

SKIP_MISSING_ENV=0
if [[ "${EXTRA_ARG}" == "--skip-missing-env" ]]; then
  SKIP_MISSING_ENV=1
elif [[ -n "${EXTRA_ARG}" ]]; then
  usage
  exit 2
fi

if [[ "${MODE}" == "all-skip-missing-env" ]]; then
  MODE="all"
  SKIP_MISSING_ENV=1
fi

TEST_RUNTIME_ROOT="${PA_TEST_RUNTIME_ROOT:-${REPO_ROOT}/out/test-state/twilio-live}"
WORKSPACE="${WORKSPACE:-test-ws1}"
DB_PATH="${DB_PATH:-${TEST_RUNTIME_ROOT}/twilio-live.db}"
LISTEN_ADDR="${LISTEN_ADDR:-127.0.0.1:8088}"
RUN_FOR="${RUN_FOR:-0}"
TWILIO_ENDPOINT="${TWILIO_ENDPOINT:-https://api.twilio.com}"
OPENAI_ENDPOINT="${OPENAI_ENDPOINT:-https://api.openai.com/v1}"
GO_BIN="${GO_BIN:-go}"
TWILIO_SMOKE_RUNTIME_MODE="${TWILIO_SMOKE_RUNTIME_MODE:-auto}" # auto|daemon|env
PA_DAEMON_MODE="${PA_DAEMON_MODE:-}"
PA_DAEMON_ADDRESS="${PA_DAEMON_ADDRESS:-}"
PA_DAEMON_AUTH_TOKEN="${PA_DAEMON_AUTH_TOKEN:-}"
PA_DAEMON_AUTH_TOKEN_FILE="${PA_DAEMON_AUTH_TOKEN_FILE:-}"
PA_RUNTIME_PROFILE="${PA_RUNTIME_PROFILE:-test}"
PA_RUNTIME_ROOT_DIR="${PA_RUNTIME_ROOT_DIR:-${TEST_RUNTIME_ROOT}/runtime-root}"
export PA_RUNTIME_PROFILE
export PA_RUNTIME_ROOT_DIR
mkdir -p "${TEST_RUNTIME_ROOT}" "${PA_RUNTIME_ROOT_DIR}"

OPENAI_CONFIGURED=0
TWILIO_CONFIGURED=0
DAEMON_AVAILABLE=0
DAEMON_PROBE_ERROR=""

twilio_credential_env=(
  TWILIO_ACCOUNT_SID
  TWILIO_AUTH_TOKEN
  TWILIO_SMS_NUMBER
  TWILIO_VOICE_NUMBER
)
required_env=()
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

PROJECT_NAME="$(normalize_project_name "${PA_PROJECT_NAME:-personalagent}")"
WEBHOOK_API_VERSION="v1"
TWILIO_SMS_WEBHOOK_PATH="/${PROJECT_NAME}/${WEBHOOK_API_VERSION}/connector/twilio/sms"
TWILIO_VOICE_WEBHOOK_PATH="/${PROJECT_NAME}/${WEBHOOK_API_VERSION}/connector/twilio/voice"
export PA_PROJECT_NAME="${PROJECT_NAME}"

pa() {
  (
    cd "${REPO_ROOT}/source/clients/cli-go"
    local args=()
    args+=(run ./cmd/personal-agent)
    if [[ -n "${PA_DAEMON_MODE}" ]]; then
      args+=(--mode "${PA_DAEMON_MODE}")
    fi
    if [[ -n "${PA_DAEMON_ADDRESS}" ]]; then
      args+=(--address "${PA_DAEMON_ADDRESS}")
    fi
    if [[ -n "${PA_DAEMON_AUTH_TOKEN}" ]]; then
      args+=(--auth-token "${PA_DAEMON_AUTH_TOKEN}")
    fi
    if [[ -n "${PA_DAEMON_AUTH_TOKEN_FILE}" ]]; then
      args+=(--auth-token-file "${PA_DAEMON_AUTH_TOKEN_FILE}")
    fi
    args+=(--db "${DB_PATH}")
    args+=("$@")
    "${GO_BIN}" "${args[@]}"
  )
}

daemon_is_reachable() {
  local output
  if output="$(pa smoke 2>&1)"; then
    DAEMON_PROBE_ERROR=""
    return 0
  fi
  DAEMON_PROBE_ERROR="${output}"
  return 1
}

probe_openai_configured() {
  local output compact
  if ! output="$(pa provider list --workspace "${WORKSPACE}" 2>/dev/null)"; then
    return 1
  fi
  compact="$(printf '%s' "${output}" | tr -d '\n')"
  if printf '%s' "${compact}" | grep -Eq '\{[^{}]*"provider"[[:space:]]*:[[:space:]]*"openai"[^{}]*"api_key_configured"[[:space:]]*:[[:space:]]*true[^{}]*\}'; then
    return 0
  fi
  return 1
}

probe_twilio_configured() {
  local output
  if ! output="$(pa connector twilio get --workspace "${WORKSPACE}" 2>/dev/null)"; then
    return 1
  fi
  if printf '%s' "${output}" | grep -Eq '"credentials_configured"[[:space:]]*:[[:space:]]*true'; then
    return 0
  fi
  return 1
}

any_twilio_env_set() {
  local name
  for name in "${twilio_credential_env[@]}"; do
    if [[ -n "${!name:-}" ]]; then
      return 0
    fi
  done
  return 1
}

resolve_runtime_profile() {
  case "${TWILIO_SMOKE_RUNTIME_MODE}" in
    auto|daemon)
      if daemon_is_reachable; then
        DAEMON_AVAILABLE=1
        if probe_openai_configured; then
          OPENAI_CONFIGURED=1
        fi
        if probe_twilio_configured; then
          TWILIO_CONFIGURED=1
        fi
      fi
      if [[ "${TWILIO_SMOKE_RUNTIME_MODE}" == "daemon" && "${DAEMON_AVAILABLE}" -eq 0 ]]; then
        echo "[twilio-smoke] runtime mode is daemon-only, but daemon is not reachable." >&2
      fi
      ;;
    env)
      echo "[twilio-smoke] runtime mode=env; daemon configuration probes disabled." >&2
      ;;
    *)
      echo "[twilio-smoke] unsupported TWILIO_SMOKE_RUNTIME_MODE=${TWILIO_SMOKE_RUNTIME_MODE}; use auto|daemon|env." >&2
      exit 2
      ;;
  esac

  if [[ "${DAEMON_AVAILABLE}" -eq 1 ]]; then
    echo "[twilio-smoke] daemon config probe: openai_configured=${OPENAI_CONFIGURED} twilio_configured=${TWILIO_CONFIGURED}"
  else
    echo "[twilio-smoke] daemon config probe unavailable; using environment bootstrap requirements."
  fi
}

build_required_env() {
  required_env=()

  case "${MODE}" in
    all|serve)
      required_env+=(PUBLIC_BASE_URL)
      ;;
  esac

  case "${MODE}" in
    all|setup)
      if [[ "${OPENAI_CONFIGURED}" -eq 0 ]]; then
        required_env+=(OPENAI_API_KEY)
      fi

      if any_twilio_env_set || [[ "${TWILIO_CONFIGURED}" -eq 0 ]]; then
        required_env+=("${twilio_credential_env[@]}")
      fi
      ;;
  esac
}

validate_required_env() {
  local missing=()
  local name
  if [[ -n "${required_env[*]-}" ]]; then
    for name in "${required_env[@]}"; do
      if [[ -z "${!name:-}" ]]; then
        missing+=("${name}")
      fi
    done
  fi

  if [[ "${#missing[@]}" -eq 0 ]]; then
    return 0
  fi

  echo "[twilio-smoke] missing required live environment variables for mode=${MODE}:" >&2
  for name in "${missing[@]}"; do
    echo "  - ${name}" >&2
  done
  echo "[twilio-smoke] set the variables above (see docs/ops/twilio-live-cli-smoke.md)." >&2
  if [[ "${DAEMON_AVAILABLE}" -eq 1 ]]; then
    echo "[twilio-smoke] tip: configure OpenAI/Twilio once via CLI, then rerun without exporting those secrets." >&2
  fi

  if [[ "${SKIP_MISSING_ENV}" -eq 1 ]]; then
    echo "[twilio-smoke] explicit skip enabled; skipping live smoke setup/run." >&2
    return 10
  fi

  echo "[twilio-smoke] strict mode failed; rerun with '${0##*/} all-skip-missing-env' or '--skip-missing-env' for offline/manual suites." >&2
  return 1
}

assert_daemon_available() {
  if [[ "${DAEMON_AVAILABLE}" -eq 1 ]]; then
    return 0
  fi
  if daemon_is_reachable; then
    DAEMON_AVAILABLE=1
    return 0
  fi

  echo "[twilio-smoke] daemon is not reachable with current CLI transport settings." >&2
  if [[ -n "${DAEMON_PROBE_ERROR}" ]]; then
    printf '%s\n' "${DAEMON_PROBE_ERROR}" >&2
  fi
  echo "[twilio-smoke] start personal-agent-daemon and verify auth/address settings." >&2
  echo "[twilio-smoke] optional overrides: PA_DAEMON_MODE, PA_DAEMON_ADDRESS, PA_DAEMON_AUTH_TOKEN, PA_DAEMON_AUTH_TOKEN_FILE." >&2
  return 1
}

resolve_runtime_profile
build_required_env
if validate_required_env; then
  :
else
  rc=$?
  if [[ "${rc}" -eq 10 ]]; then
    exit 0
  fi
  exit "${rc}"
fi

assert_daemon_available

if [[ -n "${PUBLIC_BASE_URL:-}" ]]; then
  PUBLIC_BASE_URL="${PUBLIC_BASE_URL%/}"
fi

setup_runtime() {
  echo "[twilio-smoke] configuring secrets/providers/channels for workspace=${WORKSPACE}"

  if [[ -n "${OPENAI_API_KEY:-}" || "${OPENAI_CONFIGURED}" -eq 0 ]]; then
    echo "[twilio-smoke] applying OpenAI secret/provider configuration"
    pa secret set \
      --workspace "${WORKSPACE}" \
      --name "OPENAI_API_KEY" \
      --value "${OPENAI_API_KEY}" >/dev/null

    pa provider set \
      --workspace "${WORKSPACE}" \
      --provider "openai" \
      --endpoint "${OPENAI_ENDPOINT}" \
      --api-key-secret "OPENAI_API_KEY" >/dev/null
  else
    echo "[twilio-smoke] reusing existing OpenAI provider configuration"
  fi

  pa model select \
    --workspace "${WORKSPACE}" \
    --task-class "chat" \
    --provider "openai" \
    --model "gpt-4.1-mini" >/dev/null

  if any_twilio_env_set || [[ "${TWILIO_CONFIGURED}" -eq 0 ]]; then
    echo "[twilio-smoke] applying Twilio channel configuration"
    pa connector twilio set \
      --workspace "${WORKSPACE}" \
      --account-sid "${TWILIO_ACCOUNT_SID}" \
      --auth-token "${TWILIO_AUTH_TOKEN}" \
      --sms-number "${TWILIO_SMS_NUMBER}" \
      --voice-number "${TWILIO_VOICE_NUMBER}" \
      --endpoint "${TWILIO_ENDPOINT}" >/dev/null
  else
    echo "[twilio-smoke] reusing existing Twilio channel configuration"
  fi
}

run_outbound_checks() {
  if [[ -z "${TEST_PHONE:-}" ]]; then
    echo "[twilio-smoke] TEST_PHONE not set, skipping outbound sms/call checks"
    return 0
  fi

  echo "[twilio-smoke] running outbound SMS conversational check -> ${TEST_PHONE}"
  pa connector twilio sms-chat \
    --workspace "${WORKSPACE}" \
    --to "${TEST_PHONE}" \
    --message "PersonalAgent live smoke outbound sms check"

  echo "[twilio-smoke] running outbound call check -> ${TEST_PHONE}"
  pa connector twilio start-call \
    --workspace "${WORKSPACE}" \
    --to "${TEST_PHONE}" \
    --twiml-url "${PUBLIC_BASE_URL}${TWILIO_VOICE_WEBHOOK_PATH}"
}

start_webhook_serve() {
  echo "[twilio-smoke] start webhook server on ${LISTEN_ADDR}"
  echo "[twilio-smoke] configure Twilio Console webhooks:"
  echo "  SMS webhook URL:   ${PUBLIC_BASE_URL}${TWILIO_SMS_WEBHOOK_PATH}"
  echo "  Voice webhook URL: ${PUBLIC_BASE_URL}${TWILIO_VOICE_WEBHOOK_PATH}"
  echo "[twilio-smoke] local listen address: ${LISTEN_ADDR}"

  pa connector twilio webhook serve \
    --workspace "${WORKSPACE}" \
    --listen "${LISTEN_ADDR}" \
    --signature-mode "strict" \
    --cloudflared-mode "off" \
    --assistant-replies=true \
    --assistant-task-class "chat" \
    --voice-response-mode "twiml" \
    --run-for "${RUN_FOR}"
}

case "${MODE}" in
  setup)
    setup_runtime
    echo "[twilio-smoke] setup complete"
    ;;
  serve)
    start_webhook_serve
    ;;
  all)
    setup_runtime
    run_outbound_checks
    start_webhook_serve
    ;;
  *)
    usage
    exit 2
    ;;
esac
