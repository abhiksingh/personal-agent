#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
if [[ ! -f "${ROOT}/AGENTS.md" && -f "${SCRIPT_DIR}/../../AGENTS.md" ]]; then
  ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
fi

required_files=(
  "$ROOT/README.md"
  "$ROOT/CONTRIBUTING.md"
  "$ROOT/CODE_OF_CONDUCT.md"
  "$ROOT/SECURITY.md"
  "$ROOT/SUPPORT.md"
  "$ROOT/LICENSE"
  "$ROOT/docs/context/index.yaml"
  "$ROOT/docs/spec/bootstrap.md"
  "$ROOT/docs/spec/spec.md"
  "$ROOT/docs/spec/data-model.md"
  "$ROOT/docs/spec/spec-ui.md"
  "$ROOT/docs/tests-cli.md"
  "$ROOT/docs/tests-daemon.md"
  "$ROOT/docs/ops/ui-doc-test-waivers.md"
  "$ROOT/docs/ops/macos-daemon-packaging.md"
  "$ROOT/docs/ops/twilio-live-cli-smoke.md"
  "$ROOT/docs/tests-ui.md"
  "$ROOT/AGENTS.md"
  "$ROOT/docs/harness/KNOWLEDGE_MAP.md"
  "$ROOT/docs/harness/ARCHITECTURE.md"
  "$ROOT/docs/harness/SECURITY.md"
  "$ROOT/docs/harness/security-scanner-waivers.json"
  "$ROOT/docs/harness/QUALITY_SCORE.md"
  "$ROOT/docs/harness/RELIABILITY.md"
  "$ROOT/tools/scripts/check_scaffold.sh"
  "$ROOT/tools/scripts/check_architecture_security_lint.sh"
  "$ROOT/tools/scripts/check_ui_doc_test_consistency.sh"
  "$ROOT/tools/scripts/check_ui_simple_mode_copy_lint.sh"
  "$ROOT/tools/scripts/check_openapi_route_coverage.sh"
  "$ROOT/tools/scripts/check_openapi_contract_lint.sh"
  "$ROOT/tools/scripts/check_openapi_task_approval_contracts.sh"
  "$ROOT/tools/scripts/check_legacy_normalization_guardrails.sh"
  "$ROOT/tools/scripts/check_task_actions_contracts.sh"
  "$ROOT/tools/scripts/check_config_field_descriptor_contracts.sh"
  "$ROOT/tools/scripts/check_daemon_auth_state_contracts.sh"
  "$ROOT/tools/scripts/check_security_regressions.sh"
  "$ROOT/tools/scripts/check_client_integration_fixtures.sh"
  "$ROOT/tools/scripts/check_daemon_single_writer_sqlite.sh"
  "$ROOT/tools/scripts/check_go_vet.sh"
  "$ROOT/tools/scripts/check_api_response_contracts.sh"
  "$ROOT/tools/scripts/check_api_cli_machine_contracts.sh"
  "$ROOT/tools/scripts/parse_failure_lines.sh"
  "$ROOT/tools/scripts/package_daemon_app_macos.sh"
  "$ROOT/tools/scripts/launch_personal_agent.sh"
  "$ROOT/tools/scripts/run_tests_all.sh"
  "$ROOT/tools/scripts/install_daemon_service_macos.sh"
  "$ROOT/tools/scripts/install_daemon_service_linux.sh"
  "$ROOT/tools/scripts/install_daemon_service_windows.ps1"
  "$ROOT/tools/scripts/test_twilio_live_cli_smoke.sh"
)

for f in "${required_files[@]}"; do
  if [[ ! -f "$f" ]]; then
    echo "Missing required file: $f"
    exit 1
  fi
done

for internal_path in \
  "$ROOT/docs/plans" \
  "$ROOT/docs/ops/backlogs" \
  "$ROOT/docs/ops/archive"; do
  if [[ -e "$internal_path" ]]; then
    echo "Internal planning/archive path should not exist in public export: $internal_path"
    exit 1
  fi
done

if rg -n --glob '!**/tools/scripts/check_harness.sh' "old-spec|old spec|Deprecated Spec" "$ROOT" >/dev/null 2>&1; then
  echo "Found deprecated old-spec references."
  exit 1
fi

if rg -n \
  --glob '*.md' \
  --glob '*.yaml' \
  --glob '*.yml' \
  --glob '*.txt' \
  --glob '!out/**' \
  "/Users/|Documents/Projects/PersonalAgent|\\.codex/worktrees" \
  "$ROOT/README.md" \
  "$ROOT/AGENTS.md" \
  "$ROOT/CONTRIBUTING.md" \
  "$ROOT/CODE_OF_CONDUCT.md" \
  "$ROOT/SECURITY.md" \
  "$ROOT/SUPPORT.md" \
  "$ROOT/.github" \
  "$ROOT/docs" \
  "$ROOT/source" >/dev/null 2>&1; then
  echo "Found workstation-specific absolute paths in public repo files."
  exit 1
fi

if ! rg -n "data-model.md" "$ROOT/docs/spec/spec.md" >/dev/null 2>&1; then
  echo "spec.md must reference data-model.md"
  exit 1
fi

if ! rg -n "docs/tests-ui.md" "$ROOT/docs/spec/spec-ui.md" >/dev/null 2>&1; then
  echo "spec-ui.md must reference docs/tests-ui.md"
  exit 1
fi

for marker in "README.md" "docs/spec/spec-ui.md" "docs/tests-ui.md"; do
  if ! rg -n "$marker" "$ROOT/docs/harness/KNOWLEDGE_MAP.md" >/dev/null 2>&1; then
    echo "KNOWLEDGE_MAP.md must reference $marker"
    exit 1
  fi
done

"$ROOT/tools/scripts/check_scaffold.sh"
"$ROOT/tools/scripts/check_architecture_security_lint.sh"
"$ROOT/tools/scripts/check_ui_doc_test_consistency.sh"
"$ROOT/tools/scripts/check_ui_simple_mode_copy_lint.sh"
"$ROOT/tools/scripts/check_openapi_route_coverage.sh"
"$ROOT/tools/scripts/check_openapi_contract_lint.sh"
"$ROOT/tools/scripts/check_openapi_task_approval_contracts.sh"
"$ROOT/tools/scripts/check_legacy_normalization_guardrails.sh"
"$ROOT/tools/scripts/check_task_actions_contracts.sh"
"$ROOT/tools/scripts/check_config_field_descriptor_contracts.sh"
"$ROOT/tools/scripts/check_daemon_auth_state_contracts.sh"
"$ROOT/tools/scripts/check_security_regressions.sh"
"$ROOT/tools/scripts/check_client_integration_fixtures.sh"
"$ROOT/tools/scripts/check_daemon_single_writer_sqlite.sh"
"$ROOT/tools/scripts/check_go_vet.sh"
"$ROOT/tools/scripts/check_api_response_contracts.sh"
"$ROOT/tools/scripts/check_api_cli_machine_contracts.sh"
"$ROOT/tools/scripts/test_twilio_live_cli_smoke.sh"

echo "Harness checks passed."
