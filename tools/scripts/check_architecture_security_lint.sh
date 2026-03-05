#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
if [[ ! -f "${ROOT}/AGENTS.md" && -f "${SCRIPT_DIR}/../../AGENTS.md" ]]; then
  ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
fi

fail() {
  echo "Architecture/Security lint failed: $1"
  exit 1
}

check_forbidden_imports() {
  local layer_dir="$1"
  local pattern="$2"
  local description="$3"

  if [[ ! -d "$layer_dir" ]]; then
    return 0
  fi

  if rg -n --glob '*.go' "$pattern" "$layer_dir" >/tmp/codex_arch_lint_matches.txt 2>/dev/null; then
    echo "Forbidden imports detected for $description:"
    cat /tmp/codex_arch_lint_matches.txt
    fail "$description"
  fi
}

check_file_line_limit() {
  local file_path="$1"
  local max_lines="$2"
  local label="$3"

  if [[ ! -f "$file_path" ]]; then
    fail "missing hotspot file for line-limit check: $file_path"
  fi

  local line_count
  line_count="$(wc -l < "$file_path" | tr -d ' ')"
  if [[ "${line_count:-0}" -gt "$max_lines" ]]; then
    fail "$label exceeds line limit ${max_lines} (found ${line_count}): $file_path"
  fi
}

check_payload_decode_int_cast_guardrails() {
  local payload_helper_file="$ROOT/source/services/daemon-go/internal/transport/types_payload_helpers.go"
  if [[ ! -f "$payload_helper_file" ]]; then
    fail "missing payload helper file for integer-cast guardrails: $payload_helper_file"
  fi

  if awk '
    /func readAnyIntPointer\(value any\) \*int {/ { in_func = 1; next }
    in_func && /^[[:space:]]*}/ { in_func = 0; case_type = ""; next }
    !in_func { next }

    /^[[:space:]]*case[[:space:]]+uint64:/ { case_type = "uint64"; next }
    /^[[:space:]]*case[[:space:]]+uint32:/ { case_type = "uint32"; next }
    /^[[:space:]]*case[[:space:]]+float64:/ { case_type = "float64"; next }
    /^[[:space:]]*case[[:space:]]+float32:/ { case_type = "float32"; next }
    /^[[:space:]]*case[[:space:]]+json.Number:/ { case_type = "json.Number"; next }
    /^[[:space:]]*case[[:space:]]+/ { case_type = ""; next }
    /^[[:space:]]*default:/ { case_type = ""; next }

    case_type != "" && /int[[:space:]]*\(/ {
      printf("%d:%s [%s]\n", NR, $0, case_type)
      found = 1
    }

    END { exit found ? 0 : 1 }
  ' "$payload_helper_file" >/tmp/codex_payload_decode_int_cast_matches.txt; then
    echo "Unsafe payload decode helper int-cast patterns detected:"
    cat /tmp/codex_payload_decode_int_cast_matches.txt
    fail "payload decode helpers must not use direct int(...) coercion in uint32/uint64/float/json.Number branches"
  fi
}

# Core layering checks: types -> config -> contract -> repository -> service -> runtime -> interface
check_forbidden_imports "$ROOT/source/services/daemon-go/internal/core/types" '"personalagent/runtime/internal/core/(config|contract|repository|service|runtime|interface)' "types layer importing higher core layers"
check_forbidden_imports "$ROOT/source/services/daemon-go/internal/core/config" '"personalagent/runtime/internal/core/(contract|repository|service|runtime|interface)' "config layer importing higher core layers"
check_forbidden_imports "$ROOT/source/services/daemon-go/internal/core/contract" '"personalagent/runtime/internal/core/(repository|service|runtime|interface)' "contract layer importing higher core layers"
check_forbidden_imports "$ROOT/source/services/daemon-go/internal/core/repository" '"personalagent/runtime/internal/core/(service|runtime|interface)' "repository layer importing higher core layers"
check_forbidden_imports "$ROOT/source/services/daemon-go/internal/core/service" '"personalagent/runtime/internal/core/(runtime|interface)' "service layer importing higher core layers"
check_forbidden_imports "$ROOT/source/services/daemon-go/internal/core/runtime" '"personalagent/runtime/internal/core/interface' "runtime layer importing interface layer"
check_forbidden_imports "$ROOT/source/services/daemon-go/internal/transport" '"personalagent/runtime/cmd/' "transport importing CLI/daemon entrypoint packages"
check_forbidden_imports "$ROOT/source/services/daemon-go/internal/daemonruntime" '"personalagent/runtime/cmd/' "daemonruntime importing CLI/daemon entrypoint packages"

# Contract boundary checks for adapter contracts.
check_forbidden_imports "$ROOT/source/services/daemon-go/internal/channels/contract" '"personalagent/runtime/internal/core/' "channel contract importing core internals"
check_forbidden_imports "$ROOT/source/services/daemon-go/internal/connectors/contract" '"personalagent/runtime/internal/core/' "connector contract importing core internals"

# Security baseline checks for transport defaults and auth gates.
DAEMON_MAIN_GLOB="$ROOT/source/services/daemon-go/cmd/personal-agent-daemon/main*.go"
CLI_ROOT_REGISTRY_GLOB="$ROOT/source/clients/cli-go/internal/cliapp/main*.go"
CLI_CHANNEL="$ROOT/source/clients/cli-go/internal/cliapp/channel.go"
TRANSPORT_SERVER="$ROOT/source/services/daemon-go/internal/transport/server.go"
TRANSPORT_SERVER_FILES_GLOB="$ROOT/source/services/daemon-go/internal/transport/server*.go"
TRANSPORT_MIDDLEWARE="$ROOT/source/services/daemon-go/internal/transport/server_middleware.go"
CLI_TEST_RUNNER="$ROOT/tools/scripts/run_tests_cli.sh"

if ! rg -n 'listen-address", ("127\.0\.0\.1:7071"|transport\.DefaultTCPAddress)' $DAEMON_MAIN_GLOB >/dev/null; then
  fail "daemon default listen address must remain localhost"
fi

if ! rg -n 'runProviderDaemonCommand|runModelDaemonCommand|runChatDaemonCommand|runAgentCommand|runDelegationCommand|runCommDaemonCommand|runAutomationDaemonCommand|runInspectDaemonCommand|runRetentionDaemonCommand|runContextDaemonCommand' $CLI_ROOT_REGISTRY_GLOB >/dev/null; then
  fail "cli main must route command families through daemon command handlers"
fi

if rg -n 'runProviderCommand\(|runModelCommand\(|runChatCommand\(|runCommCommand\(|runAutomationCommand\(|runInspectCommand\(|runRetentionCommand\(|runContextCommand\(' $CLI_ROOT_REGISTRY_GLOB >/dev/null; then
  fail "cli main must not route user command families through local sqlite orchestration"
fi

if ! rg -n 'runConnectorTwilioIngestSMSDaemonCommand|runConnectorTwilioIngestVoiceDaemonCommand' "$CLI_CHANNEL" >/dev/null; then
  fail "twilio ingest command routes must use daemon handlers"
fi

if rg -n 'openTwilioLocalStore\(' "$CLI_CHANNEL" >/dev/null; then
  fail "cli channel command surface must not include local twilio sqlite store helpers"
fi

if rg -n 'runChannelTwilioIngestSMSCommand\(|runChannelTwilioIngestVoiceCommand\(' "$CLI_CHANNEL" >/dev/null; then
  fail "cli channel command surface must not include local twilio ingest implementations"
fi

if ! rg -n 'pa_daemon (channel|connector) twilio ingest-sms|pa_daemon (channel|connector) twilio ingest-voice' "$CLI_TEST_RUNNER" >/dev/null; then
  fail "cli manual test runner must use daemon-backed twilio ingest commands"
fi

if rg -n 'pa (channel|connector) twilio ingest-sms|pa (channel|connector) twilio ingest-voice' "$CLI_TEST_RUNNER" >/dev/null; then
  fail "cli manual test runner contains local twilio ingest command usage"
fi

if ! rg -n 'auth token is required' "$TRANSPORT_SERVER" >/dev/null; then
  fail "transport server must require auth token"
fi

if ! rg -n 'func authorizeBearerToken' "$TRANSPORT_SERVER" >/dev/null; then
  fail "transport constant-time bearer authorization helper missing"
fi

if ! rg -n 'ConstantTimeCompare\(' "$TRANSPORT_SERVER" >/dev/null; then
  fail "transport bearer authorization must use constant-time comparison"
fi

if rg -n 'authValue != expected|authValue == expected' "$TRANSPORT_SERVER" >/dev/null; then
  fail "transport bearer authorization must not use direct string equality"
fi

auth_call_count="$(rg -n 'authorize\(' $TRANSPORT_SERVER_FILES_GLOB | wc -l | tr -d ' ')"
if [[ "${auth_call_count:-0}" -lt 40 ]]; then
  fail "expected authorize() checks across split transport handlers"
fi

if ! rg -n 'func \(s \*Server\) requireAuthorizedMethod' "$TRANSPORT_MIDDLEWARE" >/dev/null; then
  fail "transport middleware must define requireAuthorizedMethod auth helper"
fi

auth_helper_usage_count="$(rg -n 'requireAuthorizedMethod\(' $TRANSPORT_SERVER_FILES_GLOB | wc -l | tr -d ' ')"
if [[ "${auth_helper_usage_count:-0}" -lt 20 ]]; then
  fail "expected requireAuthorizedMethod() usage across transport handler modules"
fi

if rg -n --glob 'server*.go' --glob '!server_middleware.go' 'json.NewDecoder\(request.Body\)\.Decode' "$ROOT/source/services/daemon-go/internal/transport" >/dev/null; then
  fail "transport handlers must decode request bodies via decodeRequestBody strict helper"
fi

if rg -n --glob 'server*.go' 'decodeStrictJSONPayload\(' "$ROOT/source/services/daemon-go/internal/transport" >/dev/null; then
  fail "legacy transport decodeStrictJSONPayload helper must be removed"
fi

check_payload_decode_int_cast_guardrails

HTTP_DEFAULT_CLIENT_FORBIDDEN=(
  "$ROOT/source/services/daemon-go/internal/chatruntime/stream.go"
  "$ROOT/source/services/daemon-go/internal/providercheck/checker.go"
  "$ROOT/source/services/daemon-go/internal/channelcheck/twilio_checker.go"
  "$ROOT/source/services/daemon-go/internal/channels/adapters/twilio/sms_client.go"
  "$ROOT/source/services/daemon-go/internal/channels/adapters/twilio/voice_client.go"
  "$ROOT/source/clients/cli-go/internal/cliapp/chat.go"
  "$ROOT/source/clients/cli-go/internal/cliapp/provider.go"
  "$ROOT/source/clients/cli-go/internal/cliapp/comm.go"
  "$ROOT/source/clients/cli-go/internal/cliapp/channel_workflows.go"
)

if rg -n 'http\.DefaultClient' "${HTTP_DEFAULT_CLIENT_FORBIDDEN[@]}" >/tmp/codex_http_default_client_matches.txt 2>/dev/null; then
  echo "Forbidden http.DefaultClient usage detected in scoped runtime/client files:"
  cat /tmp/codex_http_default_client_matches.txt
  fail "scoped runtime/client paths must use explicit timeout-configured http clients"
fi

# Hotspot file-size guardrails to keep decomposed modules from regressing into monoliths.
check_file_line_limit "$ROOT/source/services/daemon-go/internal/transport/server.go" 400 "transport server core"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/transport/server_daemon_ops_automation.go" 320 "transport daemon-ops automation routes module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/transport/server_daemon_ops_inspect.go" 240 "transport daemon-ops inspect routes module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/transport/server_daemon_ops_retention.go" 120 "transport daemon-ops retention routes module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/transport/server_daemon_ops_context.go" 220 "transport daemon-ops context routes module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/daemonruntime/channel_dispatch.go" 520 "daemonruntime channel dispatch module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/daemonruntime/connector_dispatch.go" 520 "daemonruntime connector dispatch module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/daemonruntime/worker_dispatch_resilience.go" 320 "daemonruntime worker-dispatch resilience module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/daemonruntime/worker_dispatch_execution_core.go" 180 "daemonruntime worker-dispatch execution core module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/daemonruntime/ui_status_service.go" 300 "ui status base module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/daemonruntime/ui_status_service_config_runtime.go" 800 "ui status config helper module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/daemonruntime/ui_status_service_cards_runtime.go" 450 "ui status cards helper module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/daemonruntime/ui_status_service_health_runtime.go" 450 "ui status ingest health helper module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/daemonruntime/ui_status_service_permission_runtime.go" 450 "ui status permission helper module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/daemonruntime/ui_status_service_diagnostics_helpers_runtime.go" 450 "ui status diagnostics helper module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/daemonruntime/ui_status_service_mapping_helpers_runtime.go" 700 "ui status mapping helper module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/daemonruntime/automation_inspect_retention_context_service.go" 120 "automation/inspect base module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/daemonruntime/automation_inspect_retention_context_automation.go" 1250 "automation trigger helper module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/daemonruntime/automation_inspect_retention_context_inspect.go" 850 "inspect/log helper module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/daemonruntime/automation_inspect_retention_context_retention.go" 350 "retention/context helper module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/daemonruntime/comm_twilio_service.go" 180 "comm twilio base module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/daemonruntime/comm_twilio_service_send.go" 450 "comm twilio send module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/daemonruntime/comm_twilio_service_policy.go" 400 "comm twilio policy module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/daemonruntime/comm_twilio_service_channel.go" 650 "comm twilio channel module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/daemonruntime/comm_twilio_service_delivery_sender.go" 260 "comm twilio delivery sender module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/daemonruntime/comm_twilio_service_utils.go" 160 "comm twilio utility module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/daemonruntime/comm_inbox_service.go" 120 "comm inbox base module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/daemonruntime/comm_inbox_service_threads.go" 260 "comm inbox thread query module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/daemonruntime/comm_inbox_service_events.go" 260 "comm inbox event query module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/daemonruntime/comm_inbox_service_calls.go" 220 "comm inbox call-session query module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/daemonruntime/comm_inbox_service_helpers.go" 220 "comm inbox helper module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/channels/adapters/twilio/voice_persistence.go" 120 "twilio voice persistence base module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/channels/adapters/twilio/voice_persistence_inbound.go" 320 "twilio voice persistence inbound module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/channels/adapters/twilio/voice_persistence_outbound.go" 220 "twilio voice persistence outbound module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/channels/adapters/twilio/voice_persistence_session_helpers.go" 320 "twilio voice persistence session helper module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/daemonruntime/comm_twilio_webhook_service.go" 180 "comm twilio webhook base module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/daemonruntime/comm_twilio_webhook_service_serve_replay.go" 420 "comm twilio webhook serve/replay module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/daemonruntime/comm_twilio_webhook_service_replay_policy.go" 220 "comm twilio webhook replay policy module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/daemonruntime/comm_twilio_webhook_service_handlers.go" 420 "comm twilio webhook handler module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/daemonruntime/comm_twilio_webhook_service_cloudflared.go" 280 "comm twilio webhook cloudflared module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/daemonruntime/comm_twilio_webhook_service_helpers.go" 260 "comm twilio webhook helper module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/chatruntime/stream.go" 160 "chatruntime stream dispatcher module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/chatruntime/stream_openai.go" 420 "chatruntime openai stream module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/chatruntime/stream_ollama.go" 360 "chatruntime ollama stream module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/chatruntime/stream_anthropic.go" 420 "chatruntime anthropic stream module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/chatruntime/stream_google.go" 420 "chatruntime google stream module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/transport/types.go" 260 "transport base types module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/transport/types_secret_provider_model.go" 320 "transport secret/provider/model types module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/transport/types_chat_agent.go" 320 "transport chat/agent types module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/transport/types_delegation_comm.go" 360 "transport delegation/comm types module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/transport/types_ingest_twilio_comm.go" 380 "transport ingest/twilio types module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/daemonruntime/identity_directory_service.go" 250 "identity directory base module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/daemonruntime/identity_directory_service_context.go" 500 "identity directory context module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/daemonruntime/identity_directory_service_bootstrap.go" 500 "identity directory bootstrap module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/daemonruntime/identity_directory_service_devices_sessions.go" 500 "identity directory device/session module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/daemonruntime/identity_directory_service_queries.go" 350 "identity directory query module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/daemonruntime/unified_turn_service.go" 320 "unified turn base module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/daemonruntime/unified_turn_chat_turn.go" 620 "unified turn chat-turn module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/daemonruntime/unified_turn_tools.go" 580 "unified turn tools module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/daemonruntime/unified_turn_planner.go" 520 "unified turn planner module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/daemonruntime/unified_turn_execution.go" 380 "unified turn execution module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/daemonruntime/unified_turn_context_policy.go" 560 "unified turn context-policy module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/daemonruntime/unified_turn_persistence.go" 280 "unified turn persistence module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/daemonruntime/unified_turn_tool_registry.go" 320 "unified turn tool-registry module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/daemonruntime/unified_turn_explainability.go" 220 "unified turn explainability module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/daemonruntime/unified_turn_response_shaping.go" 180 "unified turn response-shaping module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/daemonruntime/provider_model_chat_service.go" 140 "provider/model/chat base module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/daemonruntime/provider_model_chat_service_provider.go" 260 "provider/model/chat provider module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/daemonruntime/provider_model_chat_service_models.go" 340 "provider/model/chat model module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/daemonruntime/provider_model_chat_service_route.go" 460 "provider/model/chat route module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/daemonruntime/provider_model_chat_service_chat_turn.go" 320 "provider/model/chat turn module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/daemonruntime/provider_model_chat_service_common.go" 180 "provider/model/chat common module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/core/service/agentexec/intent.go" 140 "agent intent interpreter module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/core/service/agentexec/intent_types.go" 220 "agent intent contracts module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/core/service/agentexec/intent_workflow_parsing.go" 360 "agent intent workflow parsing module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/core/service/agentexec/intent_extraction_helpers.go" 320 "agent intent extraction helper module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/core/service/agentexec/intent_native_action.go" 340 "agent intent native-action module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/core/service/agentexec/intent_model_normalization.go" 140 "agent intent model-normalization module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/transport/high_traffic_openapi_adapter_helpers.go" 180 "high-traffic openapi helpers module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/transport/high_traffic_openapi_adapter_provider_model.go" 380 "high-traffic openapi provider/model module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/transport/high_traffic_openapi_adapter_workflow.go" 440 "high-traffic openapi workflow module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/transport/high_traffic_openapi_adapter_chat.go" 300 "high-traffic openapi chat module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/daemonruntime/queued_task_runtime.go" 780 "queued-task runtime module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/daemonruntime/control_backend_service.go" 1000 "daemon control-backend service module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/transport/control_backend.go" 620 "transport control-backend module"
check_file_line_limit "$ROOT/source/services/daemon-go/internal/daemonruntime/plugin_supervisor_process.go" 920 "plugin supervisor process module"
check_file_line_limit "$ROOT/source/clients/cli-go/internal/cliapp/main.go" 1200 "cli main entrypoint"
check_file_line_limit "$ROOT/source/clients/cli-go/internal/cliapp/main_root_registry.go" 180 "cli root registry core"
check_file_line_limit "$ROOT/source/clients/cli-go/internal/cliapp/main_root_registry_configuration.go" 220 "cli root registry configuration domain"
check_file_line_limit "$ROOT/source/clients/cli-go/internal/cliapp/main_root_registry_agent.go" 220 "cli root registry agent domain"
check_file_line_limit "$ROOT/source/clients/cli-go/internal/cliapp/main_root_registry_ops.go" 260 "cli root registry operations domain"
check_file_line_limit "$ROOT/source/clients/cli-go/internal/cliapp/command_discovery_types.go" 80 "cli command-discovery contracts module"
check_file_line_limit "$ROOT/source/clients/cli-go/internal/cliapp/command_discovery_completion_renderers.go" 420 "cli command-discovery shell-renderer module"
check_file_line_limit "$ROOT/source/clients/cli-go/internal/cliapp/command_discovery_completion_index.go" 360 "cli command-discovery completion-index module"
check_file_line_limit "$ROOT/source/clients/cli-go/internal/cliapp/command_discovery_unknown_suggestions.go" 260 "cli command-discovery unknown-suggestion module"
check_file_line_limit "$ROOT/source/clients/cli-go/internal/cliapp/quickstart.go" 920 "cli quickstart workflow module"
check_file_line_limit "$ROOT/source/clients/cli-go/internal/cliapp/doctor.go" 900 "cli doctor workflow module"

echo "Architecture/Security lint checks passed."
