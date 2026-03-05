#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
if [[ ! -f "${ROOT}/AGENTS.md" && -f "${SCRIPT_DIR}/../../AGENTS.md" ]]; then
  ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
fi
OPENAPI_FILE="$ROOT/source/packages/contracts/control/openapi.yaml"
RUNTIMEPATHS_FILE="$ROOT/source/services/daemon-go/internal/runtimepaths/runtimepaths.go"
CLI_IMPL_DIR="$ROOT/source/clients/cli-go/internal/cliapp"
TRANSPORT_BACKEND_FILE="$ROOT/source/services/daemon-go/internal/transport/control_backend.go"
PERSISTED_BACKEND_FILE="$ROOT/source/services/daemon-go/internal/daemonruntime/control_backend_service.go"
TRANSPORT_TEST_DIR="$ROOT/source/services/daemon-go/internal/transport"

if [[ ! -f "$OPENAPI_FILE" ]]; then
  echo "missing openapi file: $OPENAPI_FILE"
  exit 1
fi
if [[ ! -f "$RUNTIMEPATHS_FILE" ]]; then
  echo "missing runtimepaths file: $RUNTIMEPATHS_FILE"
  exit 1
fi
if [[ ! -d "$CLI_IMPL_DIR" ]]; then
  echo "missing cli implementation directory: $CLI_IMPL_DIR"
  exit 1
fi
if [[ ! -f "$TRANSPORT_BACKEND_FILE" ]]; then
  echo "missing transport control backend file: $TRANSPORT_BACKEND_FILE"
  exit 1
fi
if [[ ! -f "$PERSISTED_BACKEND_FILE" ]]; then
  echo "missing persisted control backend file: $PERSISTED_BACKEND_FILE"
  exit 1
fi
if [[ ! -d "$TRANSPORT_TEST_DIR" ]]; then
  echo "missing transport test directory: $TRANSPORT_TEST_DIR"
  exit 1
fi

if rg -n '^  /v1/approvals/\{approval_id\}:' "$OPENAPI_FILE" >/dev/null; then
  echo "legacy approval route /v1/approvals/{approval_id} must not exist in OpenAPI"
  exit 1
fi

if rg -n 'approval decide' "$CLI_IMPL_DIR" --glob '!**/*_test.go' >/dev/null; then
  echo "legacy CLI command alias \"approval decide\" must not be reintroduced"
  exit 1
fi

if rg -n 'trimmed == "default"' "$RUNTIMEPATHS_FILE" >/dev/null; then
  echo "runtime profile helper must not alias explicit default profile to user"
  exit 1
fi

if ! rg -n 'channels:\s*DefaultCapabilitySmokeChannels\(\)' "$TRANSPORT_BACKEND_FILE" >/dev/null; then
  echo "in-memory control backend must use canonical capability smoke channel defaults"
  exit 1
fi
if ! rg -n 'connectors:\s*DefaultCapabilitySmokeConnectors\(\)' "$TRANSPORT_BACKEND_FILE" >/dev/null; then
  echo "in-memory control backend must use canonical capability smoke connector defaults"
  exit 1
fi
if ! rg -n 'channels:\s*transport\.DefaultCapabilitySmokeChannels\(\)' "$PERSISTED_BACKEND_FILE" >/dev/null; then
  echo "persisted control backend must use canonical capability smoke channel defaults"
  exit 1
fi
if ! rg -n 'connectors:\s*transport\.DefaultCapabilitySmokeConnectors\(\)' "$PERSISTED_BACKEND_FILE" >/dev/null; then
  echo "persisted control backend must use canonical capability smoke connector defaults"
  exit 1
fi

if ! rg -n "TestTransportAutomationCommTriggerValidateRejectsLegacyFilterJSONInput" "$TRANSPORT_TEST_DIR" --glob '*test.go' >/dev/null; then
  echo "missing transport regression for rejecting legacy filter_json payloads"
  exit 1
fi

go -C "$ROOT/source/services/daemon-go" test ./internal/runtimepaths -run 'TestNormalizeProfile|TestResolveRootDirWithProfile' -count=1
go -C "$ROOT/source/services/daemon-go" test ./internal/transport -run 'TestTransportAutomationCommTriggerValidateRejectsLegacyFilterJSONInput|TestInMemoryControlBackendCapabilitySmokeUsesCanonicalDefaults' -count=1
go -C "$ROOT/source/services/daemon-go" test ./internal/daemonruntime -run 'TestPersistedControlBackendCapabilitySmokeUsesCanonicalDefaults' -count=1

echo "Legacy-normalization guardrail checks passed."
