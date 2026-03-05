#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
if [[ ! -f "${ROOT}/AGENTS.md" && -f "${SCRIPT_DIR}/../../AGENTS.md" ]]; then
  ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
fi
OPENAPI_FILE="$ROOT/source/packages/contracts/control/openapi.yaml"

tasks_block="$(
  awk '
    /^  \/v1\/tasks:$/ {in_path=1; next}
    /^  \/v1\/tasks\/list:$/ {in_path=0}
    in_path {print}
  ' "$OPENAPI_FILE"
)"

task_status_block="$(
  awk '
    /^  \/v1\/tasks\/\{task_id\}:$/ {in_path=1; next}
    /^  \/v1\/approvals\/list:$/ {in_path=0}
    in_path {print}
  ' "$OPENAPI_FILE"
)"

agent_approve_block="$(
  awk '
    /^  \/v1\/agent\/approve:$/ {in_path=1; next}
    /^  \/v1\/delegation\/grant:$/ {in_path=0}
    in_path {print}
  ' "$OPENAPI_FILE"
)"

if [[ -z "$tasks_block" || -z "$task_status_block" || -z "$agent_approve_block" ]]; then
  echo "failed to extract one or more target OpenAPI path blocks"
  exit 1
fi

if ! printf '%s\n' "$tasks_block" | rg -q "TaskSubmitRequest"; then
  echo "/v1/tasks must use TaskSubmitRequest schema"
  exit 1
fi
if ! printf '%s\n' "$tasks_block" | rg -q "TaskSubmitResponse"; then
  echo "/v1/tasks must use TaskSubmitResponse schema"
  exit 1
fi
if printf '%s\n' "$tasks_block" | rg -q "JsonObjectRequired|AcceptedJSON"; then
  echo "/v1/tasks must not use generic JsonObjectRequired/AcceptedJSON contracts"
  exit 1
fi

if ! printf '%s\n' "$task_status_block" | rg -q "TaskStatusResponse"; then
  echo "/v1/tasks/{task_id} must use TaskStatusResponse schema"
  exit 1
fi
if printf '%s\n' "$task_status_block" | rg -q "OKJSON"; then
  echo "/v1/tasks/{task_id} must not use generic OKJSON contract"
  exit 1
fi

if ! printf '%s\n' "$agent_approve_block" | rg -q "AgentApproveRequest"; then
  echo "/v1/agent/approve must use AgentApproveRequest schema"
  exit 1
fi
if ! printf '%s\n' "$agent_approve_block" | rg -q "AgentRunResponse"; then
  echo "/v1/agent/approve must use AgentRunResponse schema"
  exit 1
fi
if printf '%s\n' "$agent_approve_block" | rg -q "JsonObjectRequired|OKJSON"; then
  echo "/v1/agent/approve must not use generic JsonObjectRequired/OKJSON contracts"
  exit 1
fi

echo "OpenAPI task/approval contract typing check passed."
