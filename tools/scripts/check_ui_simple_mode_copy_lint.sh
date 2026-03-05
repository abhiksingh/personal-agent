#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
if [[ ! -f "${ROOT}/AGENTS.md" && -f "${SCRIPT_DIR}/../../AGENTS.md" ]]; then
  ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
fi
TARGET="$ROOT/source/apps/macos/app-host/Packages/PersonalAgentUI/Sources/PersonalAgentUI/AppShellState.swift"

if [[ ! -f "$TARGET" ]]; then
  echo "Missing UI source for simple-mode copy lint: $TARGET" >&2
  exit 1
fi

python3 - "$TARGET" <<'PY'
import pathlib
import re
import sys

source = pathlib.Path(sys.argv[1])
lines = source.read_text(encoding='utf-8').splitlines()

# Tokens considered internal/operator jargon that should not appear in
# default Simple-mode status/user copy.
banned_terms = [
    ("payload", re.compile(r"\bpayload\b", re.IGNORECASE)),
    ("json", re.compile(r"\bjson\b", re.IGNORECASE)),
    ("idempotency", re.compile(r"\bidempotency\b", re.IGNORECASE)),
    ("task_run", re.compile(r"task_run", re.IGNORECASE)),
    ("route source", re.compile(r"route\s+source", re.IGNORECASE)),
    ("rpc", re.compile(r"\brpc\b", re.IGNORECASE)),
    ("grpc", re.compile(r"\bgrpc\b", re.IGNORECASE)),
    ("xpc", re.compile(r"\bxpc\b", re.IGNORECASE)),
    ("/v1/", re.compile(r"/v1/", re.IGNORECASE)),
    ("raw key/value", re.compile(r"raw\s+key\s*/\s*value", re.IGNORECASE)),
]

string_re = re.compile(r'"([^"\\]*(?:\\.[^"\\]*)*)"')

entries = []  # (line_number, context, text)
in_simple_case = False

for idx, raw_line in enumerate(lines, start=1):
    stripped = raw_line.strip()

    if stripped.startswith("case .simple"):
        in_simple_case = True
        continue
    if in_simple_case and stripped.startswith("case .advanced"):
        in_simple_case = False

    if in_simple_case:
        for match in string_re.finditer(raw_line):
            entries.append((idx, "simple_case", match.group(1)))

    # Status-message assignments are default user-facing copy in Simple mode.
    if "StatusMessage" in raw_line and "=" in raw_line:
        for match in string_re.finditer(raw_line):
            entries.append((idx, "status_message", match.group(1)))

violations = []
for line_number, context, text in entries:
    normalized = text.strip()
    if not normalized:
        continue
    # Ignore interpolation variable names and lint only visible user-facing words.
    visible_text = re.sub(r'\\\([^)]*\)', '', normalized).strip()
    if not visible_text:
        continue
    for label, pattern in banned_terms:
        if pattern.search(visible_text):
            violations.append((line_number, context, label, visible_text))

if violations:
    print("Simple-mode copy lint failed.")
    print("Banned internal jargon detected in user-facing Simple-mode copy:")
    for line_number, context, label, text in violations:
        print(f"  - {source}:{line_number} [{context}] contains '{label}': {text}")
    print("Use plain-language user wording or move technical wording to Advanced/details surfaces.")
    raise SystemExit(1)

print("Simple-mode copy lint passed.")
PY
