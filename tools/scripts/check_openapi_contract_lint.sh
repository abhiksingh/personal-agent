#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
if [[ ! -f "${ROOT}/AGENTS.md" && -f "${SCRIPT_DIR}/../../AGENTS.md" ]]; then
  ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
fi
SERVER_DIR="$ROOT/source/services/daemon-go/internal/transport"
OPENAPI_FILE="$ROOT/source/packages/contracts/control/openapi.yaml"
GENERIC_ALLOWLIST_FILE="$ROOT/source/packages/contracts/control/openapi-generic-allowlist.txt"

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

registrations_file="$tmpdir/registrations.txt"
handler_methods_file="$tmpdir/handler-methods.txt"
handler_delegates_file="$tmpdir/handler-delegates.txt"
server_ops_raw_file="$tmpdir/server-ops-raw.txt"
server_ops_file="$tmpdir/server-ops.txt"
openapi_ops_file="$tmpdir/openapi-ops.txt"
missing_file="$tmpdir/missing.txt"
extra_file="$tmpdir/extra.txt"
generic_ops_file="$tmpdir/generic-ops.txt"
allowlist_ops_file="$tmpdir/allowlist-ops.txt"
disallowed_generic_file="$tmpdir/disallowed-generic.txt"
invalid_allowlist_file="$tmpdir/invalid-allowlist.txt"

rg -o 'mux\.HandleFunc\("/v1[^"]*", s\.handle[[:alnum:]_]+' \
  "$SERVER_DIR" \
  --glob '*.go' \
  --glob '!**/*_test.go' \
  | sed -E 's/.*HandleFunc\("([^"]+)", s\.(handle[[:alnum:]_]+).*/\1 \2/' \
  | LC_ALL=C sort -u > "$registrations_file"

if [[ ! -s "$registrations_file" ]]; then
  echo "failed to extract daemon transport route registrations"
  exit 1
fi

find "$SERVER_DIR" -type f -name '*.go' ! -name '*_test.go' -print0 \
  | xargs -0 awk '
      FNR == 1 {
        in_handler = 0
        handler = ""
      }
      $0 ~ /^func \(s \*Server\) handle[[:alnum:]_]+\(/ {
        handler = $0
        sub(/^func \(s \*Server\) /, "", handler)
        sub(/\(.*/, "", handler)
        in_handler = 1
        next
      }
      in_handler && /^func \(s \*Server\) / {
        in_handler = 0
        handler = ""
      }
      in_handler && handler != "" {
        line = $0
        while (match(line, /http\.Method[[:alpha:]]+/)) {
          token = substr(line, RSTART, RLENGTH)
          method = toupper(substr(token, 12))
          print handler, method
          line = substr(line, RSTART + RLENGTH)
        }
      }
    ' \
  | LC_ALL=C sort -u > "$handler_methods_file"

if [[ ! -s "$handler_methods_file" ]]; then
  echo "failed to infer daemon handler methods"
  exit 1
fi

find "$SERVER_DIR" -type f -name '*.go' ! -name '*_test.go' -print0 \
  | xargs -0 awk '
      FNR == 1 {
        in_handler = 0
        handler = ""
      }
      $0 ~ /^func \(s \*Server\) handle[[:alnum:]_]+\(/ {
        handler = $0
        sub(/^func \(s \*Server\) /, "", handler)
        sub(/\(.*/, "", handler)
        in_handler = 1
        next
      }
      in_handler && /^func \(s \*Server\) / {
        in_handler = 0
        handler = ""
      }
      in_handler && handler != "" {
        line = $0
        while (match(line, /s\.handle[[:alnum:]_]+\(/)) {
          token = substr(line, RSTART, RLENGTH)
          sub(/^s\./, "", token)
          sub(/\($/, "", token)
          if (token != handler) {
            print handler, token
          }
          line = substr(line, RSTART + RLENGTH)
        }
      }
    ' \
  | LC_ALL=C sort -u > "$handler_delegates_file"

resolve_handler_methods() {
  local handler="$1"
  local current="$handler"
  local depth=0
  local methods=""
  local visited=","

  while [[ $depth -lt 12 ]]; do
    methods="$({ awk -v h="$current" '$1 == h { print $2 }' "$handler_methods_file" || true; } | LC_ALL=C sort -u)"
    if [[ -n "$methods" ]]; then
      printf '%s\n' "$methods"
      return 0
    fi

    if [[ "$visited" == *",$current,"* ]]; then
      break
    fi
    visited="${visited}${current},"

    local delegate
    delegate="$({ awk -v h="$current" '$1 == h { print $2; exit }' "$handler_delegates_file" || true; })"
    if [[ -z "$delegate" ]]; then
      break
    fi

    current="$delegate"
    depth=$((depth + 1))
  done

  return 1
}

: > "$server_ops_raw_file"
while IFS=' ' read -r route handler; do
  normalized_route="$(normalize_server_path "$route")"
  methods="$(resolve_handler_methods "$handler" || true)"
  if [[ -z "$methods" ]]; then
    echo "failed to infer HTTP method(s) for handler $handler registered at $route"
    exit 1
  fi

  while IFS= read -r method; do
    [[ -z "$method" ]] && continue
    printf '%s %s\n' "$method" "$normalized_route" >> "$server_ops_raw_file"
  done <<< "$methods"
done < "$registrations_file"

LC_ALL=C sort -u "$server_ops_raw_file" > "$server_ops_file"

awk '
  /^paths:/ {in_paths=1; next}
  /^components:/ {in_paths=0}
  in_paths && $1 ~ /^\/v1/ {
    path = $1
    sub(/:$/, "", path)
    next
  }
  in_paths && $1 ~ /^(get|post|put|patch|delete|head|options):$/ {
    method = toupper(substr($1, 1, length($1)-1))
    if (path != "") {
      print method " " path
    }
  }
' "$OPENAPI_FILE" | LC_ALL=C sort -u > "$openapi_ops_file"

if [[ ! -s "$openapi_ops_file" ]]; then
  echo "failed to extract /v1 OpenAPI operations"
  exit 1
fi

comm -23 "$server_ops_file" "$openapi_ops_file" > "$missing_file"
comm -13 "$server_ops_file" "$openapi_ops_file" > "$extra_file"

awk '
  /^paths:/ {in_paths=1; next}
  /^components:/ {in_paths=0}
  in_paths && $1 ~ /^\/v1/ {
    path = $1
    sub(/:$/, "", path)
    method = ""
    op = ""
    next
  }
  in_paths && $1 ~ /^(get|post|put|patch|delete|head|options):$/ {
    method = toupper(substr($1, 1, length($1)-1))
    op = method " " path
    next
  }
  in_paths && op != "" {
    if ($0 ~ /#\/components\/requestBodies\/JsonObjectRequired/ ||
        $0 ~ /#\/components\/responses\/OKJSON/ ||
        $0 ~ /#\/components\/responses\/AcceptedJSON/ ||
        $0 ~ /#\/components\/schemas\/JsonObject/) {
      print op
    }
  }
' "$OPENAPI_FILE" | LC_ALL=C sort -u > "$generic_ops_file"

if [[ -f "$GENERIC_ALLOWLIST_FILE" ]]; then
  awk '
    {
      line = $0
      sub(/#.*/, "", line)
      gsub(/^[[:space:]]+|[[:space:]]+$/, "", line)
      if (line != "") {
        gsub(/[[:space:]]+/, " ", line)
        print line
      }
    }
  ' "$GENERIC_ALLOWLIST_FILE" | LC_ALL=C sort -u > "$allowlist_ops_file"
else
  : > "$allowlist_ops_file"
fi

if [[ -s "$allowlist_ops_file" ]]; then
  comm -23 "$allowlist_ops_file" "$openapi_ops_file" > "$invalid_allowlist_file"
else
  : > "$invalid_allowlist_file"
fi

if [[ -s "$generic_ops_file" ]]; then
  if [[ -s "$allowlist_ops_file" ]]; then
    comm -23 "$generic_ops_file" "$allowlist_ops_file" > "$disallowed_generic_file"
  else
    cp "$generic_ops_file" "$disallowed_generic_file"
  fi
else
  : > "$disallowed_generic_file"
fi

if [[ -s "$missing_file" || -s "$extra_file" || -s "$disallowed_generic_file" || -s "$invalid_allowlist_file" ]]; then
  echo "OpenAPI contract lint failed."

  if [[ -s "$missing_file" ]]; then
    echo
    echo "Implemented daemon operations missing from OpenAPI (method path):"
    cat "$missing_file"
  fi

  if [[ -s "$extra_file" ]]; then
    echo
    echo "OpenAPI operations not registered in daemon transport (method path):"
    cat "$extra_file"
  fi

  if [[ -s "$disallowed_generic_file" ]]; then
    echo
    echo "OpenAPI /v1 operations using disallowed generic envelopes (method path):"
    cat "$disallowed_generic_file"
    echo
    echo "Use typed request/response schemas or explicitly allowlist temporary exceptions in:"
    echo "$GENERIC_ALLOWLIST_FILE"
  fi

  if [[ -s "$invalid_allowlist_file" ]]; then
    echo
    echo "Generic-envelope allowlist entries that do not match current OpenAPI operations:"
    cat "$invalid_allowlist_file"
  fi

  exit 1
fi

echo "OpenAPI contract lint check passed."
