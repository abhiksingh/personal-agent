#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
if [[ ! -f "${ROOT}/AGENTS.md" && -f "${SCRIPT_DIR}/../../AGENTS.md" ]]; then
  ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
fi
RUNTIME_GO="$ROOT/source/services/daemon-go"
SCANNER_WAIVER_FILE="$ROOT/docs/harness/security-scanner-waivers.json"

ENDPOINT_POLICY_TEST="$ROOT/source/services/daemon-go/internal/endpointpolicy/endpoint_policy_test.go"
WEBHOOK_HARDENING_TEST="$ROOT/source/services/daemon-go/internal/daemonruntime/comm_twilio_webhook_service_hardening_test.go"
WEBHOOK_REPLAY_POLICY_TEST="$ROOT/source/services/daemon-go/internal/daemonruntime/comm_twilio_webhook_replay_target_policy_test.go"
RATE_LIMIT_TEST="$ROOT/source/services/daemon-go/internal/transport/server_rate_limit_test.go"
RATE_LIMIT_BOUNDARY_TEST="$ROOT/source/services/daemon-go/internal/transport/server_client_workflow_domains_test.go"
DAEMON_AUTH_SCOPE_TEST="$ROOT/source/services/daemon-go/cmd/personal-agent-daemon/main_auth_test.go"

required_files=(
  "$ENDPOINT_POLICY_TEST"
  "$WEBHOOK_HARDENING_TEST"
  "$WEBHOOK_REPLAY_POLICY_TEST"
  "$RATE_LIMIT_TEST"
  "$RATE_LIMIT_BOUNDARY_TEST"
  "$DAEMON_AUTH_SCOPE_TEST"
  "$SCANNER_WAIVER_FILE"
)

for f in "${required_files[@]}"; do
  if [[ ! -f "$f" ]]; then
    echo "missing required security regression file: $f"
    exit 1
  fi
done

resolve_tool_bin() {
  local tool_name="$1"
  local go_path_bin=""
  go_path_bin="$(go env GOPATH 2>/dev/null || true)"
  if command -v "$tool_name" >/dev/null 2>&1; then
    command -v "$tool_name"
    return 0
  fi
  if [[ -n "$go_path_bin" && -x "$go_path_bin/bin/$tool_name" ]]; then
    echo "$go_path_bin/bin/$tool_name"
    return 0
  fi
  return 1
}

if ! command -v jq >/dev/null 2>&1; then
  echo "jq is required for security scanner result parsing"
  exit 1
fi

GOVULNCHECK_BIN=""
if ! GOVULNCHECK_BIN="$(resolve_tool_bin govulncheck)"; then
  echo "govulncheck is required; install with: go install golang.org/x/vuln/cmd/govulncheck@latest"
  exit 1
fi

GOSEC_BIN=""
if ! GOSEC_BIN="$(resolve_tool_bin gosec)"; then
  echo "gosec is required; install with: go install github.com/securego/gosec/v2/cmd/gosec@latest"
  exit 1
fi

if ! rg -n "TestParseAndValidateRejectsHTTPNonLoopbackByDefault" "$ENDPOINT_POLICY_TEST" >/dev/null; then
  echo "missing endpoint exfiltration regression test: TestParseAndValidateRejectsHTTPNonLoopbackByDefault"
  exit 1
fi
if ! rg -n "TestParseAndValidateRejectsPrivateHostByDefault" "$ENDPOINT_POLICY_TEST" >/dev/null; then
  echo "missing endpoint exfiltration regression test: TestParseAndValidateRejectsPrivateHostByDefault"
  exit 1
fi

if ! rg -n "TestParseTwilioWebhookFormRejectsOversizedBody" "$WEBHOOK_HARDENING_TEST" >/dev/null; then
  echo "missing webhook DoS regression test: TestParseTwilioWebhookFormRejectsOversizedBody"
  exit 1
fi
if ! rg -n "TestParseTwilioWebhookFormRejectsExcessiveFormFields" "$WEBHOOK_HARDENING_TEST" >/dev/null; then
  echo "missing webhook DoS regression test: TestParseTwilioWebhookFormRejectsExcessiveFormFields"
  exit 1
fi

if ! rg -n "TestUnsupportedTrustedRateLimitHeaders" "$RATE_LIMIT_TEST" >/dev/null; then
  echo "missing rate-limit spoofing regression test: TestUnsupportedTrustedRateLimitHeaders"
  exit 1
fi
if ! rg -n "TestTransportControlRateLimitRejectsTrustedScopeHeaders" "$RATE_LIMIT_BOUNDARY_TEST" >/dev/null; then
  echo "missing API boundary spoofing regression test: TestTransportControlRateLimitRejectsTrustedScopeHeaders"
  exit 1
fi

if ! rg -n "TestValidateTwilioWebhookReplayTargetRejectsPrivateIPByDefault" "$WEBHOOK_REPLAY_POLICY_TEST" >/dev/null; then
  echo "missing replay SSRF regression test: TestValidateTwilioWebhookReplayTargetRejectsPrivateIPByDefault"
  exit 1
fi
if ! rg -n "TestValidateTwilioWebhookReplayTargetRejectsMetadataByDefault" "$WEBHOOK_REPLAY_POLICY_TEST" >/dev/null; then
  echo "missing replay SSRF regression test: TestValidateTwilioWebhookReplayTargetRejectsMetadataByDefault"
  exit 1
fi

if ! rg -n "TestValidateDaemonRunConfig" "$DAEMON_AUTH_SCOPE_TEST" >/dev/null; then
  echo "missing auth-scope secure-default regression test: TestValidateDaemonRunConfig"
  exit 1
fi
if ! rg -n "TestDaemonAuthScopeWarnings" "$DAEMON_AUTH_SCOPE_TEST" >/dev/null; then
  echo "missing auth-scope warning regression test: TestDaemonAuthScopeWarnings"
  exit 1
fi

validate_gosec_with_waivers() {
  local gosec_json_file
  gosec_json_file="$(mktemp)"
  local scan_exit=0

  # Restrict gosec gate to high-severity/high-confidence findings and apply explicit waivers.
  set +e
  (
    cd "$RUNTIME_GO"
    "$GOSEC_BIN" -quiet -fmt=json -severity high -confidence high -out "$gosec_json_file" ./... >/dev/null 2>&1
  )
  scan_exit=$?
  set -e

  if [[ ! -s "$gosec_json_file" ]]; then
    echo "gosec did not produce a JSON report"
    rm -f "$gosec_json_file"
    return 1
  fi

  local findings_count=0
  findings_count="$(jq '.Issues | length' "$gosec_json_file")"
  if [[ "$findings_count" -eq 0 ]]; then
    rm -f "$gosec_json_file"
    return 0
  fi

  local unwaived_count=0
  local waived_count=0
  local stale_waiver_count=0
  local unwaived_rows=""
  local waived_rows=""
  local stale_rows=""

  unwaived_rows="$(jq -r --arg root "$ROOT/" --slurpfile waivers "$SCANNER_WAIVER_FILE" '
    [
      .Issues[]
      | {
          rule_id: .rule_id,
          file: (.file | sub("^" + $root; "")),
          line: (.line | tonumber),
          severity: .severity,
          confidence: .confidence,
          details: (.details | gsub("[\\r\\n\\t]+"; " "))
        } as $issue
      | select(
          (($waivers[0].gosec // [])
            | any(.rule_id == $issue.rule_id and .file == $issue.file and (.line | tonumber) == $issue.line)
          ) | not
        )
    ]
    | .[]
    | "\(.rule_id)|\(.file)|\(.line)\t\(.severity)\t\(.confidence)\t\(.details)"
  ' "$gosec_json_file")"
  unwaived_count="$(printf '%s\n' "$unwaived_rows" | sed '/^[[:space:]]*$/d' | wc -l | tr -d ' ')"

  waived_rows="$(jq -r --arg root "$ROOT/" --slurpfile waivers "$SCANNER_WAIVER_FILE" '
    [
      .Issues[]
      | {
          rule_id: .rule_id,
          file: (.file | sub("^" + $root; "")),
          line: (.line | tonumber)
        } as $issue
      | ($waivers[0].gosec // [])
      | .[]
      | select(.rule_id == $issue.rule_id and .file == $issue.file and (.line | tonumber) == $issue.line)
      | "\($issue.rule_id)|\($issue.file)|\($issue.line)\treason: \(.reason); tracked_by: \(.tracked_by)"
    ]
    | unique[]
  ' "$gosec_json_file")"
  waived_count="$(printf '%s\n' "$waived_rows" | sed '/^[[:space:]]*$/d' | wc -l | tr -d ' ')"

  stale_rows="$(jq -r --arg root "$ROOT/" --slurpfile waivers "$SCANNER_WAIVER_FILE" '
    [
      .Issues[]
      | {
          rule_id: .rule_id,
          file: (.file | sub("^" + $root; "")),
          line: (.line | tonumber)
        }
    ] as $issues
    | [
        ($waivers[0].gosec // [])[]
        | . as $waiver
        | select(
            ($issues
              | any(.rule_id == $waiver.rule_id and .file == $waiver.file and .line == ($waiver.line | tonumber))
            ) | not
          )
        | "\(.rule_id)|\(.file)|\(.line)"
      ]
    | unique[]
  ' "$gosec_json_file")"
  stale_waiver_count="$(printf '%s\n' "$stale_rows" | sed '/^[[:space:]]*$/d' | wc -l | tr -d ' ')"

  if [[ "$unwaived_count" -gt 0 ]]; then
    while IFS=$'\t' read -r key severity confidence details; do
      [[ -z "$key" ]] && continue
      echo "unwaived gosec finding: $key (severity=$severity confidence=$confidence)"
      echo "  details: $details"
    done <<< "$unwaived_rows"
  fi

  if [[ "$waived_count" -gt 0 ]]; then
    echo "waived gosec findings: $waived_count"
    while IFS=$'\t' read -r key reason; do
      [[ -z "$key" ]] && continue
      echo "  - $key ($reason)"
    done <<< "$waived_rows"
  fi

  if [[ "$unwaived_count" -gt 0 ]]; then
    rm -f "$gosec_json_file"
    return 1
  fi

  if [[ "$stale_waiver_count" -gt 0 ]]; then
    while IFS= read -r stale_key; do
      [[ -z "$stale_key" ]] && continue
      echo "stale gosec waiver (remove if resolved): $stale_key"
    done <<< "$stale_rows"
    echo "warning: $stale_waiver_count stale gosec waiver entries detected"
  fi

  if [[ "$scan_exit" -ne 0 && "$scan_exit" -ne 1 ]]; then
    echo "gosec exited with unexpected status code: $scan_exit"
    rm -f "$gosec_json_file"
    return 1
  fi

  rm -f "$gosec_json_file"
  return 0
}

go -C "$RUNTIME_GO" test ./internal/endpointpolicy -run 'TestParseAndValidateRejectsHTTPNonLoopbackByDefault|TestParseAndValidateRejectsPrivateHostByDefault' -count=1
go -C "$RUNTIME_GO" test ./internal/daemonruntime -run 'TestParseTwilioWebhookFormRejectsOversizedBody|TestParseTwilioWebhookFormRejectsExcessiveFormFields|TestValidateTwilioWebhookReplayTargetRejectsPrivateIPByDefault|TestValidateTwilioWebhookReplayTargetRejectsMetadataByDefault' -count=1
go -C "$RUNTIME_GO" test ./internal/transport -run 'TestUnsupportedTrustedRateLimitHeaders|TestTransportControlRateLimitRejectsTrustedScopeHeaders' -count=1
go -C "$RUNTIME_GO" test ./cmd/personal-agent-daemon -run 'TestValidateDaemonRunConfig|TestDaemonAuthScopeWarnings|TestParseDaemonAuthTokenScopes' -count=1

go -C "$RUNTIME_GO" mod verify
(
  cd "$RUNTIME_GO"
  "$GOVULNCHECK_BIN" ./...
)
validate_gosec_with_waivers

echo "Security abuse-case regression checks passed."
