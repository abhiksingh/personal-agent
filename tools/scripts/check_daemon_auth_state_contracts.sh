#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
if [[ ! -f "${ROOT}/AGENTS.md" && -f "${SCRIPT_DIR}/../../AGENTS.md" ]]; then
  ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
fi
TRANSPORT_TEST_DIR="$ROOT/source/services/daemon-go/internal/transport"
LIFECYCLE_TEST_FILE="$ROOT/source/services/daemon-go/internal/daemonruntime/daemon_lifecycle_service_test.go"

if [[ ! -d "$TRANSPORT_TEST_DIR" ]]; then
  echo "missing transport test directory: $TRANSPORT_TEST_DIR"
  exit 1
fi

if [[ ! -f "$LIFECYCLE_TEST_FILE" ]]; then
  echo "missing daemon lifecycle test file: $LIFECYCLE_TEST_FILE"
  exit 1
fi

if ! rg -n "TestTransportDaemonLifecycleStatusAndControl" "$TRANSPORT_TEST_DIR" --glob '*test.go' >/dev/null; then
  echo "missing transport daemon lifecycle control_auth contract test"
  exit 1
fi

if ! rg -n "TestDaemonControlAuthStateClassification" "$LIFECYCLE_TEST_FILE" >/dev/null; then
  echo "missing daemon lifecycle control_auth classification test"
  exit 1
fi

go -C "$ROOT/source/services/daemon-go" test ./internal/transport -run 'TestTransportDaemonLifecycleStatusAndControl' -count=1
go -C "$ROOT/source/services/daemon-go" test ./internal/daemonruntime -run 'TestDaemonControlAuthStateClassification' -count=1

echo "Daemon auth-state contract checks passed."
