#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
if [[ ! -f "${ROOT}/AGENTS.md" && -f "${SCRIPT_DIR}/../../AGENTS.md" ]]; then
  ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
fi
WAIVER_DOC="$ROOT/docs/ops/ui-doc-test-waivers.md"

if [[ ! -f "$WAIVER_DOC" ]]; then
  echo "Missing waiver registry: $WAIVER_DOC" >&2
  exit 1
fi

collect_changed_files() {
  {
    git -C "$ROOT" diff --name-only
    git -C "$ROOT" diff --cached --name-only
    git -C "$ROOT" ls-files --others --exclude-standard
  } | sed '/^$/d' | sort -u
}

contains_line() {
  local needle="$1"
  local haystack="$2"
  if [[ -z "$haystack" ]]; then
    return 1
  fi
  printf '%s\n' "$haystack" | rg -Fx -- "$needle" >/dev/null 2>&1
}

changed_files="$(collect_changed_files)"

if [[ -z "$changed_files" ]]; then
  if git -C "$ROOT" rev-parse --verify HEAD >/dev/null 2>&1; then
    changed_files="$(git -C "$ROOT" diff-tree --no-commit-id --name-only -r HEAD | sed '/^$/d' | sort -u)"
  fi
fi

if [[ -z "$changed_files" ]]; then
  echo "UI doc/test consistency check: no changed files detected; skipping."
  exit 0
fi

ui_impacting_files="$(printf '%s\n' "$changed_files" | rg '^source/apps/macos/app-host/' || true)"

if [[ -z "$ui_impacting_files" ]]; then
  echo "UI doc/test consistency check passed: no UI-impacting file changes detected."
  exit 0
fi

required_files=$'docs/spec/spec-ui.md\ndocs/tests-ui.md\ntools/scripts/run_tests_ui.sh'
missing_files=""

while IFS= read -r path; do
  [[ -z "$path" ]] && continue
  if ! contains_line "$path" "$changed_files"; then
    missing_files+="$path"$'\n'
  fi
done <<< "$required_files"

if [[ -z "$missing_files" ]]; then
  echo "UI doc/test consistency check passed: UI-impacting changes include spec/tests/runner updates."
  exit 0
fi

waiver_tokens="$(python3 - "$WAIVER_DOC" <<'PY'
import pathlib
import re
import sys

waiver_doc = pathlib.Path(sys.argv[1])
lines = waiver_doc.read_text(encoding='utf-8').splitlines()

indexes = {}

def parse_row(raw: str):
    stripped = raw.strip()
    if not stripped.startswith('|'):
        return None
    cells = [cell.strip() for cell in stripped.split('|')[1:-1]]
    return cells if cells else None

def is_separator(cells):
    if not cells:
        return False
    for cell in cells:
        compact = cell.replace(' ', '')
        if not re.fullmatch(r':?-+:?', compact):
            return False
    return True

for raw in lines:
    cells = parse_row(raw)
    if cells is None or is_separator(cells):
        continue

    lowered = [c.lower() for c in cells]
    if 'id' in lowered and 'status' in lowered and 'waives' in lowered:
        indexes = {name.lower(): i for i, name in enumerate(cells)}
        continue

    if not indexes or len(cells) <= max(indexes.values()):
        continue

    status = cells[indexes['status']].strip().lower()
    if status != 'active':
        continue

    waives = cells[indexes['waives']].strip().lower()
    for token in re.split(r'[\s,]+', waives):
        if token:
            print(token)
PY
)"

token_for_path() {
  local path="$1"
  case "$path" in
    "docs/spec/spec-ui.md")
      printf 'spec-ui\n'
      ;;
    "docs/tests-ui.md")
      printf 'tests-ui\n'
      ;;
    "tools/scripts/run_tests_ui.sh")
      printf 'run-tests-ui\n'
      ;;
    *)
      printf '\n'
      ;;
  esac
}

uncovered_files=""
while IFS= read -r path; do
  [[ -z "$path" ]] && continue
  token="$(token_for_path "$path")"
  if contains_line "all" "$waiver_tokens" || contains_line "$token" "$waiver_tokens"; then
    continue
  fi
  uncovered_files+="$path"$'\n'
done <<< "$missing_files"

if [[ -z "$uncovered_files" ]]; then
  echo "UI doc/test consistency check passed via explicit active waiver(s) in docs/ops/ui-doc-test-waivers.md."
  exit 0
fi

echo "UI doc/test consistency check failed." >&2
echo "UI-impacting files changed:" >&2
while IFS= read -r line; do
  [[ -z "$line" ]] && continue
  echo "  - $line" >&2
done <<< "$ui_impacting_files"

echo "Required files missing updates:" >&2
while IFS= read -r line; do
  [[ -z "$line" ]] && continue
  echo "  - $line" >&2
done <<< "$uncovered_files"

echo "Update the required files or add an active scoped waiver in docs/ops/ui-doc-test-waivers.md." >&2
exit 1
