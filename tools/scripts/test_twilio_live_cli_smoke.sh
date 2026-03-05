#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
if [[ ! -f "${ROOT}/AGENTS.md" && -f "${SCRIPT_DIR}/../../AGENTS.md" ]]; then
  ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
fi
TARGET="${ROOT}/tools/scripts/twilio_live_cli_smoke.sh"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

MOCK_GO="${TMP_DIR}/go-mock"
cat > "${MOCK_GO}" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

if [[ -n "${MOCK_PA_LOG:-}" ]]; then
  printf '%s\n' "$*" >> "${MOCK_PA_LOG}"
fi

mode="${MOCK_PA_MODE:-unreachable}"
cmd=" $* "

if [[ "${cmd}" == *" smoke "* ]]; then
  if [[ "${mode}" == "unreachable" ]]; then
    echo "request failed: dial tcp 127.0.0.1:7071: connect: connection refused" >&2
    exit 1
  fi
  cat <<'JSON'
{"healthy":true}
JSON
  exit 0
fi

if [[ "${cmd}" == *" provider list "* ]]; then
  if [[ "${mode}" == "configured" ]]; then
    cat <<'JSON'
{
  "workspace_id":"ws1",
  "providers":[
    {
      "workspace_id":"ws1",
      "provider":"openai",
      "endpoint":"https://api.openai.com/v1",
      "api_key_secret_name":"OPENAI_API_KEY",
      "api_key_configured":true,
      "updated_at":"2026-02-25T00:00:00Z"
    }
  ]
}
JSON
  else
    cat <<'JSON'
{"workspace_id":"ws1","providers":[]}
JSON
  fi
  exit 0
fi

if [[ "${cmd}" == *" connector twilio get "* ]]; then
  if [[ "${mode}" == "configured" ]]; then
    cat <<'JSON'
{
  "workspace_id":"ws1",
  "account_sid_secret_name":"TWILIO_ACCOUNT_SID",
  "auth_token_secret_name":"TWILIO_AUTH_TOKEN",
  "sms_number":"+15555550001",
  "voice_number":"+15555550002",
  "endpoint":"https://api.twilio.com",
  "account_sid_configured":true,
  "auth_token_configured":true,
  "credentials_configured":true,
  "updated_at":"2026-02-25T00:00:00Z"
}
JSON
  else
    echo "request failed: status=404 body={\"error\":\"twilio channel not configured\"}" >&2
    exit 1
  fi
  exit 0
fi

if [[ "${cmd}" == *" model select "* || "${cmd}" == *" secret set "* || "${cmd}" == *" provider set "* || "${cmd}" == *" connector twilio set "* || "${cmd}" == *" connector twilio webhook serve "* || "${cmd}" == *" connector twilio sms-chat "* || "${cmd}" == *" connector twilio start-call "* ]]; then
  cat <<'JSON'
{"ok":true}
JSON
  exit 0
fi

cat <<'JSON'
{}
JSON
EOF
chmod +x "${MOCK_GO}"

required_env=(
  OPENAI_API_KEY
  TWILIO_ACCOUNT_SID
  TWILIO_AUTH_TOKEN
  TWILIO_SMS_NUMBER
  TWILIO_VOICE_NUMBER
  PUBLIC_BASE_URL
)

assert_contains() {
  local output="$1"
  local needle="$2"
  if ! grep -Fq -- "$needle" <<<"${output}"; then
    echo "assertion failed: output did not contain '${needle}'" >&2
    echo "--- output begin ---" >&2
    printf '%s\n' "${output}" >&2
    echo "--- output end ---" >&2
    exit 1
  fi
}

run_clean() {
  local mode="$1"
  shift
  env -i \
    PATH="${PATH}" \
    HOME="${HOME:-/tmp}" \
    GO_BIN="${MOCK_GO}" \
    "$@" \
    bash "${TARGET}" "${mode}"
}

run_clean_with_extra() {
  local mode="$1"
  local extra_arg="$2"
  shift 2
  env -i \
    PATH="${PATH}" \
    HOME="${HOME:-/tmp}" \
    GO_BIN="${MOCK_GO}" \
    "$@" \
    bash "${TARGET}" "${mode}" "${extra_arg}"
}

set +e
strict_output="$(run_clean all MOCK_PA_MODE=unreachable 2>&1)"
strict_rc=$?
set -e
if [[ "${strict_rc}" -ne 1 ]]; then
  echo "expected strict mode missing-env failure exit code 1, got ${strict_rc}" >&2
  printf '%s\n' "${strict_output}" >&2
  exit 1
fi

assert_contains "${strict_output}" "[twilio-smoke] missing required live environment variables for mode=all:"
for name in "${required_env[@]}"; do
  assert_contains "${strict_output}" "  - ${name}"
done
assert_contains "${strict_output}" "docs/ops/twilio-live-cli-smoke.md"
assert_contains "${strict_output}" "all-skip-missing-env"

set +e
skip_output="$(run_clean all-skip-missing-env MOCK_PA_MODE=unreachable 2>&1)"
skip_rc=$?
set -e
if [[ "${skip_rc}" -ne 0 ]]; then
  echo "expected explicit skip mode to exit 0, got ${skip_rc}" >&2
  printf '%s\n' "${skip_output}" >&2
  exit 1
fi
assert_contains "${skip_output}" "explicit skip enabled; skipping live smoke setup/run."

set +e
skip_flag_output="$(run_clean_with_extra all --skip-missing-env MOCK_PA_MODE=unreachable 2>&1)"
skip_flag_rc=$?
set -e
if [[ "${skip_flag_rc}" -ne 0 ]]; then
  echo "expected --skip-missing-env mode to exit 0, got ${skip_flag_rc}" >&2
  printf '%s\n' "${skip_flag_output}" >&2
  exit 1
fi
assert_contains "${skip_flag_output}" "explicit skip enabled; skipping live smoke setup/run."

configured_log="${TMP_DIR}/configured-setup.log"
set +e
configured_setup_output="$(run_clean setup MOCK_PA_MODE=configured MOCK_PA_LOG="${configured_log}" 2>&1)"
configured_setup_rc=$?
set -e
if [[ "${configured_setup_rc}" -ne 0 ]]; then
  echo "expected configured daemon setup mode to succeed without env vars, got ${configured_setup_rc}" >&2
  printf '%s\n' "${configured_setup_output}" >&2
  exit 1
fi
assert_contains "${configured_setup_output}" "setup complete"
assert_contains "${configured_setup_output}" "reusing existing OpenAI provider configuration"
assert_contains "${configured_setup_output}" "reusing existing Twilio channel configuration"
if grep -Fq " secret set " "${configured_log}"; then
  echo "expected setup fallback to avoid secret set when daemon config is already present" >&2
  cat "${configured_log}" >&2
  exit 1
fi
if grep -Fq " provider set " "${configured_log}"; then
  echo "expected setup fallback to avoid provider set when daemon config is already present" >&2
  cat "${configured_log}" >&2
  exit 1
fi
if grep -Fq " connector twilio set " "${configured_log}"; then
  echo "expected setup fallback to avoid twilio set when daemon config is already present" >&2
  cat "${configured_log}" >&2
  exit 1
fi
assert_contains "$(cat "${configured_log}")" " model select "

set +e
configured_all_missing_public_output="$(run_clean all MOCK_PA_MODE=configured 2>&1)"
configured_all_missing_public_rc=$?
set -e
if [[ "${configured_all_missing_public_rc}" -ne 1 ]]; then
  echo "expected configured all mode without PUBLIC_BASE_URL to fail with exit code 1, got ${configured_all_missing_public_rc}" >&2
  printf '%s\n' "${configured_all_missing_public_output}" >&2
  exit 1
fi
assert_contains "${configured_all_missing_public_output}" "  - PUBLIC_BASE_URL"
if grep -Fq "  - OPENAI_API_KEY" <<<"${configured_all_missing_public_output}"; then
  echo "did not expect OPENAI_API_KEY to be required when daemon already has OpenAI configuration" >&2
  printf '%s\n' "${configured_all_missing_public_output}" >&2
  exit 1
fi
if grep -Fq "  - TWILIO_ACCOUNT_SID" <<<"${configured_all_missing_public_output}"; then
  echo "did not expect Twilio credential env vars to be required when daemon already has Twilio configuration" >&2
  printf '%s\n' "${configured_all_missing_public_output}" >&2
  exit 1
fi

echo "Twilio live smoke script checks passed."
