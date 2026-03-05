#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
if [[ ! -f "${ROOT}/AGENTS.md" && -f "${SCRIPT_DIR}/../../AGENTS.md" ]]; then
  ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
fi
TEST_DIR="$ROOT/source/services/daemon-go/internal/transport"

if [[ ! -d "$TEST_DIR" ]]; then
  echo "missing transport test directory: $TEST_DIR"
  exit 1
fi

if ! rg -n "TestTransportChannelAndConnectorStatusDescriptorMetadataDefaults" "$TEST_DIR" --glob '*test.go' >/dev/null; then
  echo "missing channel/connector descriptor metadata default contract test"
  exit 1
fi

if ! rg -n "assertConfigFieldDescriptorPayload" "$TEST_DIR" --glob '*test.go' >/dev/null; then
  echo "missing descriptor payload assertion helper"
  exit 1
fi

go -C "$ROOT/source/services/daemon-go" test ./internal/transport -run 'TestTransportChannelAndConnectorStatusRoutes|TestTransportChannelAndConnectorStatusDescriptorMetadataDefaults' -count=1

echo "Config-field descriptor contract checks passed."
