#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
if [[ ! -f "${ROOT}/AGENTS.md" && -f "${SCRIPT_DIR}/../../AGENTS.md" ]]; then
  ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
fi

required_paths=(
  "$ROOT/source/apps/macos/app-host/README.md"
  "$ROOT/source/apps/macos/app-host/Sources/PersonalAgentApp/main.swift"
  "$ROOT/source/services/daemon-go/go.mod"
  "$ROOT/source/services/daemon-go/cmd/personal-agent-daemon/main.go"
  "$ROOT/source/clients/cli-go/go.mod"
  "$ROOT/source/clients/cli-go/cmd/personal-agent/main.go"
  "$ROOT/source/services/daemon-go/internal/core/types/README.md"
  "$ROOT/source/services/daemon-go/internal/core/config/README.md"
  "$ROOT/source/services/daemon-go/internal/core/contract/README.md"
  "$ROOT/source/services/daemon-go/internal/core/repository/README.md"
  "$ROOT/source/services/daemon-go/internal/core/service/README.md"
  "$ROOT/source/services/daemon-go/internal/core/runtime/README.md"
  "$ROOT/source/services/daemon-go/internal/core/interface/README.md"
  "$ROOT/source/services/daemon-go/internal/channels/contract/README.md"
  "$ROOT/source/services/daemon-go/internal/connectors/contract/README.md"
  "$ROOT/source/services/daemon-go/internal/shared/contracts/README.md"
  "$ROOT/source/packages/contracts/control/openapi.yaml"
  "$ROOT/source/packages/contracts/realtime/event-envelope.schema.json"
)

for p in "${required_paths[@]}"; do
  if [[ ! -e "$p" ]]; then
    echo "Missing scaffold path: $p"
    exit 1
  fi
done

echo "Scaffold checks passed."
