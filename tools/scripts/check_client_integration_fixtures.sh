#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
if [[ ! -f "${ROOT}/AGENTS.md" && -f "${SCRIPT_DIR}/../../AGENTS.md" ]]; then
  ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
fi
PACK_DIR="$ROOT/source/packages/contracts/control/fixtures/client-integration"
MANIFEST="$PACK_DIR/manifest.json"

if [[ ! -f "$MANIFEST" ]]; then
  echo "missing fixture-pack manifest: $MANIFEST"
  exit 1
fi

if ! rg -n '"fixture_pack_id"\s*:\s*"personal-agent-control-client-integration-v1"' "$MANIFEST" >/dev/null; then
  echo "fixture pack id missing/invalid in manifest"
  exit 1
fi

if ! rg -n '"api_version"\s*:\s*"v1"' "$MANIFEST" >/dev/null; then
  echo "fixture pack api_version missing/invalid in manifest"
  exit 1
fi

fixture_ids=()
while IFS= read -r line; do
  fixture_ids+=("$line")
done < <(rg -o '"id"\s*:\s*"[^"]+"' "$MANIFEST" | sed -E 's/"id"[[:space:]]*:[[:space:]]*"([^"]+)"/\1/')

if [[ "${#fixture_ids[@]}" -eq 0 ]]; then
  echo "manifest has no fixture ids"
  exit 1
fi

duplicate_ids="$(printf '%s\n' "${fixture_ids[@]}" | sort | uniq -d || true)"
if [[ -n "$duplicate_ids" ]]; then
  echo "manifest has duplicate fixture ids:"
  echo "$duplicate_ids"
  exit 1
fi

fixture_paths=()
while IFS= read -r line; do
  fixture_paths+=("$line")
done < <(rg -o '"path"\s*:\s*"[^"]+"' "$MANIFEST" | sed -E 's/"path"[[:space:]]*:[[:space:]]*"([^"]+)"/\1/')

if [[ "${#fixture_paths[@]}" -eq 0 ]]; then
  echo "manifest has no fixture paths"
  exit 1
fi

for rel_path in "${fixture_paths[@]}"; do
  if [[ ! -f "$PACK_DIR/$rel_path" ]]; then
    echo "manifest fixture path missing file: $PACK_DIR/$rel_path"
    exit 1
  fi
done

fixture_runners=()
while IFS= read -r line; do
  fixture_runners+=("$line")
done < <(rg -o '"runner"\s*:\s*"[^"]+"' "$MANIFEST" | sed -E 's/"runner"[[:space:]]*:[[:space:]]*"([^"]+)"/\1/')

if [[ "${#fixture_runners[@]}" -eq 0 ]]; then
  echo "manifest has no runner references"
  exit 1
fi

for runner in "${fixture_runners[@]}"; do
  if [[ ! -f "$ROOT/$runner" ]]; then
    echo "manifest runner does not exist: $ROOT/$runner"
    exit 1
  fi
done

"$ROOT/tools/scripts/check_daemon_auth_state_contracts.sh"
"$ROOT/tools/scripts/check_task_actions_contracts.sh"
"$ROOT/tools/scripts/check_config_field_descriptor_contracts.sh"

echo "Client integration fixture pack checks passed."
