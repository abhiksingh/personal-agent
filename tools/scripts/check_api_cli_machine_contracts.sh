#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
if [[ ! -f "${ROOT}/AGENTS.md" && -f "${SCRIPT_DIR}/../../AGENTS.md" ]]; then
  ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
fi

go -C "$ROOT/source/services/daemon-go" test ./internal/transport -run 'TestBuildTransportErrorEnvelopeIncludesLegacyAndTypedFields|TestParseTransportHTTPErrorTypedEnvelope|TestTransportRegistersDaemonDomainEndpointGroups' -count=1
go -C "$ROOT/source/clients/cli-go" test ./internal/cliapp -run 'TestRunSmokeCommandJSONCompactOutput|TestRunSmokeCommandMachineErrorJSONOutput|TestRunUnknownCommandMachineErrorJSONOutput' -count=1
go -C "$ROOT/source/clients/cli-go" test ./cmd/personal-agent -count=1

echo "API/CLI machine-output contract checks passed."
