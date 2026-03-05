#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
if [[ ! -f "${ROOT}/AGENTS.md" && -f "${SCRIPT_DIR}/../../AGENTS.md" ]]; then
  ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
fi
SERVICE_CONTAINER_FILE="$ROOT/source/services/daemon-go/internal/daemonruntime/service_container.go"
SPEC_FILE="$ROOT/docs/spec/spec.md"
ARCH_FILE="$ROOT/docs/harness/ARCHITECTURE.md"

if ! rg -q "daemonSQLiteMaxOpenConns\\s*=\\s*1" "$SERVICE_CONTAINER_FILE"; then
  echo "daemon sqlite single-writer invariant missing: daemonSQLiteMaxOpenConns must be 1"
  exit 1
fi

if ! rg -q "daemonSQLiteMaxIdleConns\\s*=\\s*1" "$SERVICE_CONTAINER_FILE"; then
  echo "daemon sqlite single-writer invariant missing: daemonSQLiteMaxIdleConns must be 1"
  exit 1
fi

if ! rg -q "SetMaxOpenConns\\(daemonSQLiteMaxOpenConns\\)" "$SERVICE_CONTAINER_FILE"; then
  echo "daemon sqlite single-writer invariant missing: SetMaxOpenConns configuration"
  exit 1
fi

if ! rg -q "SetMaxIdleConns\\(daemonSQLiteMaxIdleConns\\)" "$SERVICE_CONTAINER_FILE"; then
  echo "daemon sqlite single-writer invariant missing: SetMaxIdleConns configuration"
  exit 1
fi

if ! rg -q "single-writer queue boundary" "$SPEC_FILE"; then
  echo "spec must document single-writer queue boundary policy"
  exit 1
fi

if ! rg -q "single-writer gate" "$ARCH_FILE"; then
  echo "architecture doc must document single-writer gate persistence boundary"
  exit 1
fi

echo "Daemon SQLite single-writer invariant check passed."
