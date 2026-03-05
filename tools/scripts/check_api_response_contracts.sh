#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
if [[ ! -f "${ROOT}/AGENTS.md" && -f "${SCRIPT_DIR}/../../AGENTS.md" ]]; then
  ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
fi
OPENAPI_FILE="$ROOT/source/packages/contracts/control/openapi.yaml"
SERVER_FILE="$ROOT/source/services/daemon-go/internal/transport/server.go"

if [[ ! -f "$OPENAPI_FILE" ]]; then
  echo "missing OpenAPI contract: $OPENAPI_FILE"
  exit 1
fi
if [[ ! -f "$SERVER_FILE" ]]; then
  echo "missing transport server file: $SERVER_FILE"
  exit 1
fi

check_component_header() {
  local header_name="$1"
  if ! awk -v key="$header_name" '
    /^  headers:/ {in_headers=1; next}
    in_headers && /^  [a-zA-Z]/ {in_headers=0}
    in_headers && $0 ~ "^    " key ":" {found=1}
    END {exit(found ? 0 : 1)}
  ' "$OPENAPI_FILE"; then
    echo "OpenAPI headers component missing: $header_name"
    exit 1
  fi
}

check_response_block_contains() {
  local response_name="$1"
  local required_text="$2"
  if ! awk -v key="$response_name" -v required="$required_text" '
    /^  schemas:/ {if (in_block) exit(found ? 0 : 1)}
    $0 ~ "^    " key ":" {in_block=1; next}
    in_block && $0 ~ "^    [A-Za-z0-9_]+:" {exit(found ? 0 : 1)}
    in_block && index($0, required) > 0 {found=1}
    END {if (in_block) exit(found ? 0 : 1); exit(1)}
  ' "$OPENAPI_FILE"; then
    echo "OpenAPI response $response_name missing required field: $required_text"
    exit 1
  fi
}

check_component_header "CorrelationID"
check_component_header "APIVersion"

for response in BadRequest Unauthorized NotFound NotImplemented; do
  check_response_block_contains "$response" "application/problem+json"
  check_response_block_contains "$response" "X-Correlation-ID"
  check_response_block_contains "$response" "X-PersonalAgent-API-Version"
done

if ! rg -n 'responseHeaderCorrelationID|responseHeaderAPIVersion|responseContentTypeProblem' "$SERVER_FILE" >/dev/null; then
  echo "transport server is missing expected response-header/content-type constants"
  exit 1
fi
if ! rg -n 'writeJSONWithContentType\(writer, statusCode, buildTransportErrorEnvelope\(' "$SERVER_FILE" >/dev/null; then
  echo "transport server writeJSONError is not using the standardized envelope writer"
  exit 1
fi
if ! rg -n 'responseContentTypeProblem' "$SERVER_FILE" >/dev/null; then
  echo "transport server is missing problem+json content-type enforcement"
  exit 1
fi

go -C "$ROOT/source/services/daemon-go" test ./internal/transport -run 'TestTransportSuccessResponsesIncludeCorrelationAndAPIVersionHeaders|TestTransportRegistersDaemonDomainEndpointGroups' -count=1

echo "API response contract drift check passed."
