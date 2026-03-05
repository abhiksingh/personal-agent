#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "usage: $0 <log-file> [--max-lines N]" >&2
  exit 2
fi

LOG_FILE="$1"
shift

MAX_LINES=200
while [[ $# -gt 0 ]]; do
  case "$1" in
    --max-lines)
      MAX_LINES="${2:-}"
      shift 2
      ;;
    *)
      echo "unknown argument: $1" >&2
      echo "usage: $0 <log-file> [--max-lines N]" >&2
      exit 2
      ;;
  esac
done

if [[ ! -f "$LOG_FILE" ]]; then
  echo "log file not found: $LOG_FILE" >&2
  exit 1
fi

PATTERN='error:|\bfailed\b|\bFAIL\b|panic:|fatal:|Assertion|XCTAssert|uncaught exception|unhandled exception|fatal exception|terminating due to uncaught'

if rg -n -i "$PATTERN" "$LOG_FILE" >/tmp/parse_failure_lines.matches.$$ 2>/dev/null; then
  head -n "$MAX_LINES" /tmp/parse_failure_lines.matches.$$
  rm -f /tmp/parse_failure_lines.matches.$$
  exit 0
fi

rm -f /tmp/parse_failure_lines.matches.$$ || true

echo "No failure-like lines matched. Tail of log:" >&2
tail -n "$MAX_LINES" "$LOG_FILE"
