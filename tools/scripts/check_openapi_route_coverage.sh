#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
if [[ ! -f "${ROOT}/AGENTS.md" && -f "${SCRIPT_DIR}/../../AGENTS.md" ]]; then
  ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
fi
SERVER_DIR="$ROOT/source/services/daemon-go/internal/transport"
OPENAPI_FILE="$ROOT/source/packages/contracts/control/openapi.yaml"

if [[ ! -d "$SERVER_DIR" ]]; then
  echo "Missing daemon transport directory: $SERVER_DIR"
  exit 1
fi

if [[ ! -f "$OPENAPI_FILE" ]]; then
  echo "Missing OpenAPI contract file: $OPENAPI_FILE"
  exit 1
fi

normalize_server_path() {
  local path="$1"
  case "$path" in
    "/v1/tasks/")
      echo "/v1/tasks/{task_id}"
      ;;
    "/v1/secrets/refs/")
      echo "/v1/secrets/refs/{workspace_id}/{name}"
      ;;
    *)
      echo "$path"
      ;;
  esac
}

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

server_paths_file="$tmpdir/server-paths.txt"
openapi_paths_file="$tmpdir/openapi-paths.txt"
missing_file="$tmpdir/missing.txt"
extra_file="$tmpdir/extra.txt"

while IFS= read -r route; do
  normalize_server_path "$route"
done < <(
  rg -o 'mux\.HandleFunc\("/v1[^"]*"' "$SERVER_DIR" --glob '*.go' --glob '!**/*_test.go' \
    | sed -E 's/.*\("([^"]+)".*/\1/'
) \
  | LC_ALL=C sort -u > "$server_paths_file"

awk '
  /^paths:/ {in_paths=1; next}
  /^components:/ {in_paths=0}
  in_paths && $1 ~ /^\/v1/ {
    path = $1
    sub(/:$/, "", path)
    print path
  }
' "$OPENAPI_FILE" | LC_ALL=C sort -u > "$openapi_paths_file"

comm -23 "$server_paths_file" "$openapi_paths_file" > "$missing_file"
comm -13 "$server_paths_file" "$openapi_paths_file" > "$extra_file"

if [[ -s "$missing_file" || -s "$extra_file" ]]; then
  echo "OpenAPI route coverage drift detected."
  if [[ -s "$missing_file" ]]; then
    echo
    echo "Implemented daemon routes missing from OpenAPI:"
    cat "$missing_file"
  fi
  if [[ -s "$extra_file" ]]; then
    echo
    echo "OpenAPI routes not registered in daemon transport:"
    cat "$extra_file"
  fi
  exit 1
fi

echo "OpenAPI route coverage check passed."
