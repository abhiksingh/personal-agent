# PersonalAgent CLI Manual Test Guide

This guide walks through manual validation of the current PersonalAgent MVP (CLI-first), including:

1. Core CLI workflows (identity context, secrets, providers/models, chat, agent execution, approvals, delegation, comm, automation, inspect/retention/context)
2. Twilio SMS/voice workflows (ingest, outbound, status, transcript, webhook runtime)
3. Conversational Twilio webhook mode (SMS auto-reply + voice TwiML loop)
4. Cloudflared connector control-plane workflows (version, exec passthrough)

Daemon-focused transport/capability validation is maintained in:

- `docs/tests-daemon.md`

## Quick Run (Automated)

Run the saved CLI manual-test runner from repo root:

```bash
./tools/scripts/run_tests_cli.sh
```

To run CLI + daemon + UI runners together, use:

```bash
./tools/scripts/run_tests_all.sh
```

The runner exits with process code `0` only when all steps pass; any recorded step failure returns non-zero.

Useful options:

```bash
# Require live Twilio env and fail if missing
./tools/scripts/run_tests_cli.sh --live-mode strict

# Skip live Twilio step entirely
./tools/scripts/run_tests_cli.sh --live-mode off

# Write to a specific log path
./tools/scripts/run_tests_cli.sh --log-file out/logs/manual-tests/tests-cli-manual.log
```

Use this from repository root:

```bash
cd <repo-root>
```

## 1) Prerequisites

- Go toolchain installed.
- `curl` installed.
- `jq` installed (used by copy/paste commands in this guide for extracting IDs from JSON output).

## 2) Test Environment Setup

Set base variables and helper function:

```bash
export TEST_RUNTIME_ROOT="$PWD/out/test-state/cli"
export WORKSPACE="${WORKSPACE:-test-ws1}"
export DB_PATH="${DB_PATH:-$TEST_RUNTIME_ROOT/manual-test-cli.db}"
export DAEMON_ADDR="127.0.0.1:17071"
export DAEMON_AUTH_TOKEN="cli-test-token"
export CONTROL_AUTH_TOKEN_FILE="${CONTROL_AUTH_TOKEN_FILE:-$TEST_RUNTIME_ROOT/manual-test-cli.control.token}"
export LOCAL_DEV_BOOTSTRAP_TOKEN_FILE="${LOCAL_DEV_BOOTSTRAP_TOKEN_FILE:-$TEST_RUNTIME_ROOT/manual-test-cli.local-dev.token}"
export PA_RUNTIME_PROFILE="${PA_RUNTIME_PROFILE:-test}"
export PA_RUNTIME_ROOT_DIR="${PA_RUNTIME_ROOT_DIR:-$TEST_RUNTIME_ROOT/runtime-root}"
export PA_MESSAGES_SEND_DRY_RUN="${PA_MESSAGES_SEND_DRY_RUN:-1}" # deterministic iMessage send behavior on non-macOS/TCC-restricted hosts
export PA_MAIL_AUTOMATION_DRY_RUN="1" # forced for mail-test safety (prevents real sends)
export PA_CALENDAR_AUTOMATION_DRY_RUN="${PA_CALENDAR_AUTOMATION_DRY_RUN:-1}" # deterministic Calendar connector behavior on non-macOS/TCC-restricted hosts
export PA_BROWSER_AUTOMATION_DRY_RUN="${PA_BROWSER_AUTOMATION_DRY_RUN:-1}" # deterministic Safari connector behavior on non-macOS/TCC-restricted hosts
export PA_CLOUDFLARED_DRY_RUN="${PA_CLOUDFLARED_DRY_RUN:-1}" # deterministic cloudflared connector behavior for manual/CI runs
mkdir -p "$TEST_RUNTIME_ROOT" "$PA_RUNTIME_ROOT_DIR"
rm -f "$DB_PATH"
rm -f "$CONTROL_AUTH_TOKEN_FILE"
rm -f "$LOCAL_DEV_BOOTSTRAP_TOKEN_FILE"

pa() {
  (cd source/clients/cli-go && go run ./cmd/personal-agent --db "$DB_PATH" "$@")
}

pa_daemon() {
  pa --mode tcp --address "$DAEMON_ADDR" --auth-token "$DAEMON_AUTH_TOKEN" "$@"
}
```

Bootstrap and rotate a production-grade daemon control token file (metadata-only output, no raw token print):

```bash
pa auth bootstrap --file "$CONTROL_AUTH_TOKEN_FILE"
pa auth rotate --file "$CONTROL_AUTH_TOKEN_FILE"
```

Expected:

- both commands succeed and output includes `token_file` and `token_sha256`.
- output does not include the raw token value.

One-command local-dev bootstrap (token + CLI profile):

```bash
pa auth bootstrap-local-dev \
  --profile local-dev-bootstrap \
  --mode tcp \
  --address "$DAEMON_ADDR" \
  --workspace "$WORKSPACE" \
  --token-file "$LOCAL_DEV_BOOTSTRAP_TOKEN_FILE" \
  --rotate-token

pa profile get --name local-dev-bootstrap
```

Expected:

- `bootstrap-local-dev` succeeds and returns:
  - `operation=bootstrap_local_dev`
  - `token_file=$LOCAL_DEV_BOOTSTRAP_TOKEN_FILE`
  - `token_sha256` metadata without raw token value
  - `profile` metadata (`name=local-dev-bootstrap`, daemon endpoint/workspace/auth-token-file)
  - `active_profile=local-dev-bootstrap` (default activation behavior)
  - `defaults` metadata for `workspace`, `profile`, and `token_file` with:
    - selected `value`
    - `source` (`explicit|default`)
    - `override_flag` (`--workspace|--profile|--token-file`)
    - `override_hints[]` containing concrete override guidance.
- `pa profile get --name local-dev-bootstrap` returns the same metadata-only profile state.

Workspace resolution sanity check (env default + explicit override):

```bash
PERSONAL_AGENT_WORKSPACE_ID=ws-cli-context pa connector bridge status
PERSONAL_AGENT_WORKSPACE_ID= pa connector bridge status
PERSONAL_AGENT_WORKSPACE_ID=ws-cli-context pa connector bridge status --workspace default
```

Expected:

- when `PERSONAL_AGENT_WORKSPACE_ID=ws-cli-context`, response `workspace_id` is `ws-cli-context`.
- when env is empty, response `workspace_id` is canonical `ws1`.
- explicit `--workspace default` is treated as explicit workspace id `default`.

## 3) Start Local Mock Providers (Offline-Friendly)

This lets you test everything without real OpenAI/Twilio credentials.

### 3.1 Start Go-based OpenAI + Twilio mocks

```bash
go -C source/services/daemon-go run ./cmd/personal-agent-mock \
  --mode all \
  --openai-listen 127.0.0.1:18080 \
  --twilio-listen 127.0.0.1:19080 &
MOCKS_PID=$!

until curl -sSf http://127.0.0.1:18080/v1/models >/dev/null; do sleep 0.2; done
until curl -sSf http://127.0.0.1:19080/2010-04-01/Accounts/AC123.json >/dev/null; do sleep 0.2; done
```

Expected startup output includes:
- `openai mock listening on http://127.0.0.1:18080`
- `twilio mock listening on http://127.0.0.1:19080`

## 4) Baseline Health Checks

```bash
cd source/services/daemon-go && go test ./...
cd ../..
./tools/scripts/check_harness.sh
```

Expected: all pass.

## 5) Core CLI Workflow Manual Tests

Start daemon runtime used by daemon-backed CLI flows (`secret`, `provider`, `model`, `chat`, `agent`, `delegation`):

```bash
cd source/services/daemon-go && PA_RUNTIME_PROFILE="$PA_RUNTIME_PROFILE" PA_RUNTIME_ROOT_DIR="$PA_RUNTIME_ROOT_DIR" go run ./cmd/personal-agent-daemon \
  --listen-mode tcp \
  --listen-address "$DAEMON_ADDR" \
  --auth-token "$DAEMON_AUTH_TOKEN" \
  --db "$DB_PATH" &
DAEMON_PID=$!
cd ../..

until curl -sS "http://$DAEMON_ADDR/v1/capabilities/smoke" >/dev/null; do sleep 0.2; done
```

### 5.0.1 CLI profile defaults (endpoint/auth/workspace)

Create a daemon-auth token file for profile defaults:

```bash
PROFILE_AUTH_TOKEN_FILE="${PROFILE_AUTH_TOKEN_FILE:-$TEST_RUNTIME_ROOT/manual-test-cli.profile.token}"
printf "%s\n" "$DAEMON_AUTH_TOKEN" > "$PROFILE_AUTH_TOKEN_FILE"
chmod 600 "$PROFILE_AUTH_TOKEN_FILE"
```

Set, inspect, and use a named CLI profile:

```bash
pa profile set --name local-daemon --mode tcp --address "$DAEMON_ADDR" --workspace "$WORKSPACE" --auth-token-file "$PROFILE_AUTH_TOKEN_FILE"
pa profile list
pa profile get
pa profile use --name local-daemon
pa profile active
pa profile rename --name local-daemon --to local-daemon-renamed
pa profile get --name local-daemon-renamed
pa profile delete --name local-daemon-renamed
pa profile active
pa profile set --name local-dev-bootstrap --mode tcp --address "$DAEMON_ADDR" --workspace "$WORKSPACE" --auth-token-file "$PROFILE_AUTH_TOKEN_FILE"
pa profile active
pa smoke
```

Expected:

- `profile set` returns profile metadata with `active_profile` and `path`; when a bootstrap profile already exists, new profiles are added inactive until explicitly selected.
- `profile list` includes both `local-daemon` and `local-dev-bootstrap` with deterministic sorted ordering.
- `profile get` (without `--name`) returns the current active profile and only metadata (auth token file reference path, no token value).
- `profile use --name local-daemon` activates `local-daemon`.
- `profile active` returns explicit active-profile inspection state (`active_profile`, `profile_exists`, optional `profile` object).
- `profile rename --name local-daemon --to local-daemon-renamed` updates profile identity and preserves active selection when renaming the active profile.
- `profile delete --name local-daemon-renamed` deletes that profile and deterministically falls back active selection to the next available profile (`local-dev-bootstrap` in this flow).
- second `profile active` confirms post-delete active fallback metadata.
- updating `local-dev-bootstrap` with the daemon token file preserves fallback behavior while restoring auth parity for profile-default smoke checks.
- third `profile active` confirms fallback profile now references `PROFILE_AUTH_TOKEN_FILE`.
- `pa smoke` succeeds without passing `--mode/--address/--auth-token`.

### 5.0.1.1 CLI exit-code semantics (help/usage/runtime classes)

Validate representative usage-error paths:

```bash
set +e
PROFILE_SET_USAGE_TEXT="$(pa profile set 2>&1 >/dev/null)"
PROFILE_SET_USAGE_RC=$?
AUTH_ROTATE_USAGE_TEXT="$(pa auth rotate 2>&1 >/dev/null)"
AUTH_ROTATE_USAGE_RC=$?
set -e
echo "$PROFILE_SET_USAGE_TEXT"
echo "profile_set_exit=$PROFILE_SET_USAGE_RC"
echo "$AUTH_ROTATE_USAGE_TEXT"
echo "auth_rotate_exit=$AUTH_ROTATE_USAGE_RC"
```

Expected:

- `pa help` and other help/discovery flows exit `0`.
- usage/argument failures return non-zero and include `exit status 2` in stderr:
  - `pa profile set` without `--name`
  - `pa auth rotate` without `--file`
- runtime/request failures exit `1` (for example provider/task checks against an unconfigured or unreachable daemon).

### 5.0.2 AI-agent output mode checks (`--output` + `--error-output`)

Verify compact machine-readable success output:

```bash
pa --output json-compact smoke
```

Expected:

- command succeeds.
- output is valid JSON on a single compact line (no indented pretty-print blocks).
- payload contains `healthy=true`.

Verify structured JSON error output on a failing command:

```bash
pa --output json-compact --error-output json task status --task-id task-does-not-exist
```

Expected:

- command exits non-zero.
- stderr emits parseable JSON with top-level `error` object.
- `error` includes `code`, `message`, and `status_code` fields for daemon HTTP failures.
- when command errors are CLI-local (for example unknown command), `error.code` is `cli.command_failed` and message includes the original CLI error text.

Verify human-readable text output for high-frequency diagnostics:

```bash
DOCTOR_TEXT_OUTPUT="$(pa --output text doctor --workspace "$WORKSPACE" 2>/dev/null)"; DOCTOR_TEXT_RC=$?; echo "$DOCTOR_TEXT_OUTPUT"
[ "$DOCTOR_TEXT_RC" -eq 0 ] || [ "$DOCTOR_TEXT_RC" -eq 1 ]
```

Expected:

- output is plain text (not JSON) and includes `doctor report`, `overall_status:`, and `daemon.connectivity`.
- command exit code remains `0` on pass or `1` when readiness checks fail.

Verify build/version metadata output contract:

```bash
VERSION_JSON="$(pa --output json-compact version)"
VERSION_TEXT_OUTPUT="$(pa --output text version)"
echo "$VERSION_JSON"
echo "$VERSION_TEXT_OUTPUT"
```

Expected:

- JSON output includes `schema_version`, `program`, `version`, `go_version`, and `platform`.
- `program` is `personal-agent`.
- text output is human-readable (not JSON) and includes `personal-agent version`, `version:`, and `platform:`.

### 5.0.3 Goal-oriented help output

Print usage using the goal-oriented help IA:

```bash
pa help
pa help task
pa --auth-token "" task --help
pa --auth-token "" task submit --help
```

Expected:

- command succeeds and prints usage text.
- output includes skim-first help headers:
  - `Quickstart workflows (copy/paste):`
  - `Skim command groups:`
  - `Full command reference (generated from schema):`
- output includes concise quickstart examples (for example `personal-agent quickstart ...`) and generated command reference entries for core command families (`help`, `doctor`, `task`, `agent`, `comm`).
- output includes `version` in setup/local command listings.
- `pa help task` prints command-scoped usage (`Usage: personal-agent task <subcommand> [flags]`) with task subcommand listing.
- `pa --auth-token "" task --help` prints command-scoped usage and exits `0` without requiring daemon auth configuration.
- `pa --auth-token "" task submit --help` prints subcommand-scoped usage (`Usage: personal-agent task submit [flags]`) and required flags (`--workspace`, `--requested-by`, `--subject`, `--title`).

### 5.0.4 Interactive assistant mode (common workflows)

Task-submit flow with deterministic prompts and `back` support:

```bash
ASSISTANT_TASK_JSON="$(printf "actor.requester\nback\nactor.requester\nactor.requester\nAssistant task title\nAssistant description\nchat\n" | pa_daemon assistant --workspace "$WORKSPACE" --flow task_submit)"
echo "$ASSISTANT_TASK_JSON"
```

Comm-send flow cancellation:

```bash
ASSISTANT_CANCEL_JSON="$(printf "cancel\n" | pa_daemon assistant --workspace "$WORKSPACE" --flow comm_send)"
echo "$ASSISTANT_CANCEL_JSON"
```

Expected:

- task-submit flow returns `flow=task_submit`, `success=true`, `cancelled=false`, `backtracks>=1`, and `result.task_id`/`result.run_id`.
- cancel flow returns `flow=comm_send`, `cancelled=true`, `success=false`, and no daemon mutation result object.

### 5.0.5 CLI machine-readable command manifest (`meta schema`)

Emit CLI schema metadata for AI-agent clients:

```bash
pa --output json-compact meta schema
```

Expected:

- command succeeds and returns parseable JSON.
- payload includes:
  - `schema_version`
  - `program`
  - `output_modes`
  - `error_output_modes`
  - `global_flags[]`
  - `commands[]`
- `output_modes` includes `json`, `json-compact`, and `text`.
- `global_flags[]` contains `--output` and `--error-output`.
- `commands[]` includes `task -> submit` with required flags (`--workspace`, `--requested-by`, `--subject`, `--title`).
- `commands[]` includes `help` and `quickstart` with `requires_daemon=false`.
- `commands[]` includes `version` with `requires_daemon=false`.
- `commands[]` includes `assistant` with `requires_daemon=true`.
- `commands[]` marks `chat` as `supports_streaming=true` and `machine_output_safe=false` (interactive stream text contract).
- `commands[]` includes nested subcommand trees for key workflows (for example `connector -> twilio -> webhook -> serve`, `channel -> mapping -> enable`, `automation -> run -> comm-event`).
- `commands[]` marks `stream` with `supports_streaming=true`.

### 5.0.6 Daemon runtime discovery metadata (`meta capabilities`)

Query typed daemon discovery metadata for AI-agent clients:

```bash
pa --output json-compact meta capabilities
```

Expected:

- command succeeds and returns parseable JSON.
- payload includes:
  - `api_version`
  - `route_groups[]` (`id`, `prefix`, optional `description`)
  - `realtime_event_types[]`
  - `client_signal_types[]`
  - `protocol_modes[]`
  - `transport_listener_modes[]`
- `api_version` is `v1`.
- `client_signal_types[]` includes `cancel`.
- `protocol_modes[]` includes `http_json` and `websocket_json`.
- `transport_listener_modes[]` includes `tcp`.

### 5.0.7 Workspace readiness diagnostics (`doctor`)

Emit machine-readable workspace readiness diagnostics:

```bash
DOCTOR_JSON="$(pa --output json-compact doctor --workspace "$WORKSPACE")"; DOCTOR_RC=$?; echo "$DOCTOR_JSON"
```

Expected:

- command exits `0` when all checks pass, or `1` when one or more checks report `fail` (both are valid for this validation step).
- output is parseable JSON and includes:
  - `schema_version`
  - `generated_at`
  - `workspace_id`
  - `overall_status`
  - `summary` (`pass|warn|fail|skipped` counts)
  - `checks[]`
- `checks[]` includes these stable IDs:
  - `daemon.connectivity`
  - `daemon.lifecycle`
  - `workspace.context`
  - `providers.readiness`
  - `models.route_readiness`
  - `channels.mappings`
  - `secrets.references`
  - `plugins.health`
  - `tooling.optional`
- `daemon.lifecycle` check details include `control_auth.state` (`configured|missing`) and `control_auth.source` (`auth_token_flag|auth_token_file|unknown`) with remediation hints when auth is missing.

Run fast first-line diagnostics without deep provider/model/plugin probes:

```bash
DOCTOR_QUICK_JSON="$(pa --output json-compact doctor --workspace "$WORKSPACE" --quick)"; DOCTOR_QUICK_RC=$?; echo "$DOCTOR_QUICK_JSON"
[ "$DOCTOR_QUICK_RC" -eq 0 ] || [ "$DOCTOR_QUICK_RC" -eq 1 ]
```

Expected:

- output is parseable JSON with the same stable check IDs.
- first-line checks (`daemon.connectivity`, `daemon.lifecycle`, `workspace.context`) are evaluated.
- deep checks are `skipped` in quick mode:
  - `providers.readiness`
  - `models.route_readiness`
  - `channels.mappings`
  - `secrets.references`
  - `plugins.health`
  - `tooling.optional`

### 5.0.8 Actionable text-mode error remediation blocks

Trigger an expected daemon-side error in default text mode:

```bash
set +e
ACTIONABLE_ERROR_TEXT="$(pa task status --task-id task-does-not-exist 2>&1 >/dev/null)"
ACTIONABLE_ERROR_RC=$?
set -e
echo "$ACTIONABLE_ERROR_TEXT"
echo "exit=$ACTIONABLE_ERROR_RC"
```

Expected:

- command exits with `1`.
- stderr uses remediation block format:
  - `request failed`
  - `what failed: ...`
  - `why: ...`
  - `do next:`
- output includes guidance to re-run with `--error-output json`.
- output does not include raw transport debug strings like `status=<code>` or `code=<value>`.

### 5.0.9 Shell completion and typo-recovery suggestions

Generate deterministic shell completion scripts:

```bash
BASH_COMPLETION_SCRIPT="$(pa completion --shell bash)"
FISH_COMPLETION_SCRIPT="$(pa completion --shell fish)"
ZSH_COMPLETION_SCRIPT="$(pa completion zsh)"
printf "%s\n" "$BASH_COMPLETION_SCRIPT" | head -n 10
printf "%s\n" "$FISH_COMPLETION_SCRIPT" | head -n 10
printf "%s\n" "$ZSH_COMPLETION_SCRIPT" | head -n 10
```

Expected:

- bash output includes `_personal_agent_completion()` and `complete -F _personal_agent_completion personal-agent`.
- fish output includes `function __pa_dynamic_completions` and `complete -c personal-agent -f -a`.
- zsh output includes `#compdef personal-agent` and `compdef _personal_agent personal-agent`.
- completion scripts include deep command-path completion coverage (for example `connector twilio webhook` and `channel mapping`).
- completion scripts include contextual flag completion hints for high-use paths:
  - `task submit` (`--workspace`, `--requested-by`, `--subject`, `--title`)
  - `provider set` (`--provider`, `--endpoint`, `--api-key-secret`)
  - `model select` (`--task-class`, `--provider`, `--model`)
  - `comm policy set` (`--source-channel`, `--primary-channel`, `--fallback-channels`)
  - `connector twilio webhook serve` (`--signature-mode`, `--listen`)
  - `identity bootstrap` (`--workspace`, `--principal`, `--display-name`)
- all commands exit `0`.

Validate typo-aware unknown-command hint:

```bash
set +e
UNKNOWN_COMMAND_TEXT="$(pa provder 2>&1 >/dev/null)"
UNKNOWN_COMMAND_RC=$?
set -e
echo "$UNKNOWN_COMMAND_TEXT"
echo "exit=$UNKNOWN_COMMAND_RC"
```

Expected:

- command exits non-zero and stderr includes `exit status 2`.
- stderr includes `unknown command "provder"`.
- stderr includes a suggestion hint like `did you mean "provider"?`.
- stderr includes an actionable next step like `run personal-agent help` to discover valid commands.

Validate typo-aware unknown-subcommand hint:

```bash
set +e
UNKNOWN_SUBCOMMAND_TEXT="$(pa meta schem 2>&1 >/dev/null)"
UNKNOWN_SUBCOMMAND_RC=$?
set -e
echo "$UNKNOWN_SUBCOMMAND_TEXT"
echo "exit=$UNKNOWN_SUBCOMMAND_RC"
```

Expected:

- command exits non-zero and stderr includes `exit status 2`.
- stderr includes `unknown meta subcommand "schem"`.
- stderr includes a suggestion hint like `did you mean "schema"?`.
- stderr includes an actionable next step like `run personal-agent help meta` to discover valid subcommands.

### 5.0.10 Guided quickstart flow

Run guided quickstart against the local daemon/mock provider setup:

```bash
printf "%s\n" "$DAEMON_AUTH_TOKEN" > "$TEST_RUNTIME_ROOT/manual-test-cli.quickstart.token"
chmod 600 "$TEST_RUNTIME_ROOT/manual-test-cli.quickstart.token"
QUICKSTART_JSON="$(pa quickstart --workspace "$WORKSPACE" --profile quickstart-manual --mode tcp --address "$DAEMON_ADDR" --token-file "$TEST_RUNTIME_ROOT/manual-test-cli.quickstart.token" --provider openai --endpoint "http://127.0.0.1:18080/v1" --api-key "sk-local-quickstart-mock" --model gpt-4.1-mini --task-class chat --skip-doctor=true)"
echo "$QUICKSTART_JSON"
```

Expected:

- `schema_version` is `1.0.0`, `overall_status=pass`, and `success=true`.
- `defaults` metadata is present and reports explicit selections for `workspace`, `profile`, and `token_file` plus override hints for each flag.
- step `auth.bootstrap` reports `status=pass`.
- step `daemon.connectivity` reports `status=pass`.
- step `provider.configure` reports `status=pass`.
- step `model.route` reports `status=pass`.
- step `readiness.doctor` reports `status=skipped` when `--skip-doctor=true`.
- output includes structured remediation fields (`remediation.human_summary`, optional `remediation.next_steps`) and does not include plaintext API key material.

### 5.0.11 Guided quickstart failure remediation

Run quickstart against an intentionally unreachable daemon address:

```bash
set +e
QUICKSTART_FAILURE_JSON="$(pa quickstart --workspace "$WORKSPACE" --profile quickstart-remediation --activate=false --mode tcp --address 127.0.0.1:17999 --token-file "$TEST_RUNTIME_ROOT/manual-test-cli.quickstart.token" --provider openai --skip-provider-setup=true --skip-model-route=true --skip-doctor=true)"
QUICKSTART_FAILURE_RC=$?
set -e
echo "$QUICKSTART_FAILURE_JSON"
echo "exit=$QUICKSTART_FAILURE_RC"
```

Expected:

- command exits `1`.
- output remains parseable JSON.
- step `auth.bootstrap` reports `status=pass`.
- step `daemon.connectivity` reports `status=fail`.
- `overall_status=fail` and `success=false`.
- `defaults` metadata is present and reports explicit selections for `workspace`, `profile`, and `token_file` plus override hints.
- `remediation.human_summary` is populated.
- `remediation.next_steps[]` contains environment-aware commands that include the resolved mode/address/token/profile context, for example:
  - daemon restart command with resolved listener values (`personal-agent-daemon --listen-mode ... --listen-address ... --auth-token-file ...`)
  - profile activation command when `--activate=false` (`personal-agent profile use --name ...`)
  - smoke verification command bound to the resolved transport/auth context.

### 5.1 Identity context and workspace selection

```bash
pa_daemon identity bootstrap --workspace "$WORKSPACE" --workspace-name "Manual Test Workspace" --principal actor.bootstrap.cli --display-name "CLI Bootstrap User" --actor-type human --principal-status ACTIVE --handle-channel app --handle-value "bootstrap-${WORKSPACE}" --handle-primary=true
pa_daemon identity bootstrap --workspace "$WORKSPACE" --workspace-name "Manual Test Workspace" --principal actor.bootstrap.cli --display-name "CLI Bootstrap User" --actor-type human --principal-status ACTIVE --handle-channel app --handle-value "bootstrap-${WORKSPACE}" --handle-primary=true
pa_daemon identity workspaces --include-inactive=true
pa_daemon identity context
pa_daemon identity principals --workspace "$WORKSPACE"
pa_daemon identity select-workspace --workspace "$WORKSPACE"
pa_daemon identity devices --workspace "$WORKSPACE" --limit 5
IDENTITY_SESSIONS_JSON="$(pa_daemon identity sessions --workspace "$WORKSPACE" --limit 5)"
IDENTITY_ACTIVE_SESSION_ID="$(echo "$IDENTITY_SESSIONS_JSON" | jq -r '.items[0].session_id')"
pa_daemon identity revoke-session --workspace "$WORKSPACE" --session-id "$IDENTITY_ACTIVE_SESSION_ID"
pa_daemon identity revoke-session --workspace "$WORKSPACE" --session-id "$IDENTITY_ACTIVE_SESSION_ID"
```

Expected:

- first `identity bootstrap` call returns `workspace_created|principal_created|principal_linked=true`.
- second identical `identity bootstrap` call returns `idempotent=true` and no duplicate-create flags.
- `identity workspaces` returns JSON containing `active_context` and `workspaces[]`.
- `identity context` returns daemon active context payload including `mutation_source`, `mutation_reason`, and monotonic `selection_version`.
- `identity principals` returns workspace principal list for the requested workspace.
- `identity select-workspace` updates active context and returns `active_context.workspace_id` set to `$WORKSPACE`.
- `identity devices` returns device inventory rows and pagination cursors when `has_more=true`.
- `identity sessions` returns session inventory rows and supports session-health/device/user filters.
- `identity revoke-session` revokes the target session; repeating the same command is idempotent (`idempotent=true` on replay).

### 5.1.1 Channel-to-connector mapping controls

List current logical-channel connector mappings:

```bash
pa_daemon channel mapping list --workspace "$WORKSPACE" --channel message
pa_daemon channel mapping list --workspace "$WORKSPACE" --channel voice
```

Disable Twilio on message channel:

```bash
pa_daemon channel mapping disable --workspace "$WORKSPACE" --channel message --connector twilio
```

Reprioritize Twilio to primary slot and re-enable:

```bash
pa_daemon channel mapping prioritize --workspace "$WORKSPACE" --channel message --connector twilio --priority 1
pa_daemon channel mapping enable --workspace "$WORKSPACE" --channel message --connector twilio --priority 1
pa_daemon channel mapping list --workspace "$WORKSPACE" --channel message
```

Disable and re-enable Twilio on voice channel:

```bash
pa_daemon channel mapping disable --workspace "$WORKSPACE" --channel voice --connector twilio
pa_daemon channel mapping enable --workspace "$WORKSPACE" --channel voice --connector twilio --priority 1
pa_daemon channel mapping list --workspace "$WORKSPACE" --channel voice
```

Validate canonical logical channel mapping state:

```bash
pa_daemon channel mapping list --workspace "$WORKSPACE" --channel message
pa_daemon channel mapping list --workspace "$WORKSPACE" --channel voice
```

Expected:

- clean-install list responses include `fallback_policy="priority_order"` and seeded logical bindings:
  - `message` includes `imessage` + `twilio`
  - `voice` includes `twilio`
- disable response returns `connector_id="twilio"` and `enabled=false`.
- prioritize response returns `connector_id="twilio"` and `priority=1`.
- final enable/list confirms `twilio.enabled=true` with priority `1` and `imessage` still present.
- voice disable/enable responses return `channel_id="voice"` with `connector_id="twilio"` and `enabled` transitions `false -> true`.
- channel/connector parity holds after mutations: `twilio` is enabled on both `message` and `voice`.

### 5.2 Secret storage

```bash
pa_daemon secret set --workspace "$WORKSPACE" --name OPENAI_API_KEY --value "sk-local-mock"
pa_daemon secret get --workspace "$WORKSPACE" --name OPENAI_API_KEY
```

Expected:

- `set` succeeds and returns only secret metadata (`workspace_id`, `name`, secure-store reference fields, `registered=true`).
- `get` succeeds and returns secret metadata only (no `value` field and no plaintext secret material).

### 5.3 Provider and model setup

```bash
pa_daemon provider set --workspace "$WORKSPACE" --provider openai --endpoint "http://127.0.0.1:18080/v1" --api-key-secret OPENAI_API_KEY
pa_daemon provider set --workspace "$WORKSPACE" --provider ollama --endpoint "http://127.0.0.1:11434"
pa_daemon provider list --workspace "$WORKSPACE"
pa_daemon --output text provider list --workspace "$WORKSPACE"
pa_daemon provider check --workspace "$WORKSPACE" --provider openai
pa_daemon model list --workspace "$WORKSPACE"
pa_daemon --output text model list --workspace "$WORKSPACE"
pa_daemon model discover --workspace "$WORKSPACE" --provider openai
pa_daemon model add --workspace "$WORKSPACE" --provider openai --model gpt-5-codex --enabled=true
pa_daemon model remove --workspace "$WORKSPACE" --provider openai --model gpt-5-codex
pa_daemon model select --workspace "$WORKSPACE" --task-class chat --provider openai --model gpt-4.1-mini
pa_daemon model resolve --workspace "$WORKSPACE" --task-class chat
pa provider list
```

Expected:

- OpenAI provider check reports `success=true`.
- model discover returns provider-scoped `results[]` with discovery status and model entries.
- model add returns catalog record for `openai/gpt-5-codex` (enabled as requested).
- model remove returns `removed=true` and the removed provider/model identity.
- model resolve returns `provider=openai` and `model_key=gpt-4.1-mini`.
- `--output text` provider/model list outputs are human-readable (not JSON) and include `workspace: $WORKSPACE`.
- profile-driven `pa provider list` (without daemon flags and without `--workspace`) succeeds and returns workspace entries for `$WORKSPACE`.

### 5.3.1 Cloudflared connector control-plane checks

```bash
pa_daemon connector cloudflared version --workspace "$WORKSPACE"
pa_daemon connector cloudflared exec --workspace "$WORKSPACE" --arg version
```

Expected:

- `version` returns `available=true` and `binary_path` value.
- `exec` returns `success=true` and echoes requested `args`.
- when `PA_CLOUDFLARED_DRY_RUN=1`, responses include `dry_run=true` and no external cloudflared process/network dependency.

### 5.4 Chat streaming

```bash
pa_daemon chat --workspace "$WORKSPACE" --task-class chat --message "Say hello for manual test"
```

Expected: streamed assistant response contains `mock reply`.

### 5.5 Agent execution (non-destructive connector happy paths)

```bash
pa_daemon agent run --workspace "$WORKSPACE" --request "open https://example.com"
pa_daemon agent run --workspace "$WORKSPACE" --request 'send an email to recipient@example.com saying "cli update"'
pa_daemon agent run --workspace "$WORKSPACE" --request "schedule event with the team" --approval-phrase "GO AHEAD"
pa_daemon agent run --workspace "$WORKSPACE" --request 'send an sms to +15550001111: "hello"'
```

Expected:

- browser run includes `workflow=browser` and `run_state=completed`.
- mail run includes `workflow=mail`, `run_state=completed`, and mail step evidence provider `apple-mail-dry-run`.
- calendar run with explicit `--approval-phrase "GO AHEAD"` includes `workflow=calendar` and `run_state=completed`.
- messages run with explicit SMS channel either executes (`workflow=messages`, `clarification_required=false`, `run_state=completed`, `messages_send_sms`) or returns actionable text error output (`request failed`, `what failed`, and either `unsupported source channel "sms"` or `unable to determine intent`).
- each response includes `task_id` and `run_id`.
- Finder connector destructive flow is covered in section `5.5` (`delete file ...`).

### 5.5.1 Agent clarification loop (missing slots)

Run an under-specified destructive request:

```bash
FINDER_CLARIFY_JSON="$(pa_daemon agent run --workspace "$WORKSPACE" --request "delete file now")"
echo "$FINDER_CLARIFY_JSON"
```

Expected:

- `clarification_required=true`
- `task_state=clarification_required` and `run_state=clarification_required`
- `missing_slots` includes `finder_query`
- no `task_id`/`run_id` is created because execution did not start

Run an under-specified messages request:

```bash
MESSAGE_CLARIFY_JSON="$(pa_daemon agent run --workspace "$WORKSPACE" --request 'send a text to +15550001111: \"hello\"')"
echo "$MESSAGE_CLARIFY_JSON"
```

Expected:

- `workflow=messages`
- `clarification_required=true`
- `missing_slots` includes `message_channel`
- response includes a typed `native_action.messages` payload with parsed recipient/body fields

### 5.6 Agent destructive-action approval flow

Run destructive request:

```bash
DESTRUCTIVE_JSON="$(pa_daemon agent run --workspace "$WORKSPACE" --request "delete file /tmp/manual-test.txt")"
echo "$DESTRUCTIVE_JSON"
```

Expected:

- `approval_required=true`
- `approval_request_id` present
- `run_state=awaiting_approval`

Verify unauthorized approver is denied:

```bash
APPROVAL_ID="$(echo "$DESTRUCTIVE_JSON" | jq -r '.approval_request_id')"
test -n "$APPROVAL_ID" && [ "$APPROVAL_ID" != "null" ]
pa_daemon agent approve --workspace "$WORKSPACE" --approval-id "$APPROVAL_ID" --actor-id actor.approver --phrase "GO AHEAD"
```

Expected: command fails with `approval denied` because `actor.approver` is not `acting_as` and has no approval delegation yet.

Grant approval-scope delegation and approve:

```bash
APPROVAL_GRANT_JSON="$(pa_daemon delegation grant --workspace "$WORKSPACE" --from actor.requester --to actor.approver --scope-type APPROVAL)"
echo "$APPROVAL_GRANT_JSON"
APPROVAL_RULE_ID="$(echo "$APPROVAL_GRANT_JSON" | jq -r '.id')"
test -n "$APPROVAL_RULE_ID" && [ "$APPROVAL_RULE_ID" != "null" ]
pa_daemon agent approve --workspace "$WORKSPACE" --approval-id "$APPROVAL_ID" --actor-id actor.approver --phrase "GO AHEAD"
pa_daemon delegation revoke --workspace "$WORKSPACE" --rule-id "$APPROVAL_RULE_ID"
```

Expected: delegated approver command succeeds and resumed run returns `run_state=completed`.

Run explicit calendar-cancel request without upfront approval phrase to validate `calendar_cancel` capability gate:

```bash
CALENDAR_EVENT_ID="$(echo "$CALENDAR_RUN_JSON" | jq -r '.step_states[] | select(.capability_key=="calendar_create") | .evidence.event_id')"
test -n "$CALENDAR_EVENT_ID" && [ "$CALENDAR_EVENT_ID" != "null" ]
CALENDAR_DESTRUCTIVE_JSON="$(pa_daemon agent run --workspace "$WORKSPACE" --request "cancel calendar event id $CALENDAR_EVENT_ID")"
echo "$CALENDAR_DESTRUCTIVE_JSON"
CALENDAR_APPROVAL_ID="$(echo "$CALENDAR_DESTRUCTIVE_JSON" | jq -r '.approval_request_id')"
test -n "$CALENDAR_APPROVAL_ID" && [ "$CALENDAR_APPROVAL_ID" != "null" ]
pa_daemon agent approve --workspace "$WORKSPACE" --approval-id "$CALENDAR_APPROVAL_ID" --actor-id actor.requester --phrase "GO AHEAD"
```

Expected:

- calendar run returns `approval_required=true` and `run_state=awaiting_approval`.
- pending step set includes capability `calendar_cancel` for the same `event_id`.
- approval with `actor.requester` succeeds and resumed run completes.

### 5.6.1 Voice destructive-action handoff gate

Run a voice-origin destructive request without in-app confirmation:

```bash
VOICE_DESTRUCTIVE_JSON="$(pa_daemon agent run --workspace "$WORKSPACE" --request "delete file /tmp/manual-voice-handoff.txt" --origin voice --approval-phrase "GO AHEAD")"
echo "$VOICE_DESTRUCTIVE_JSON"
```

Expected:

- `approval_required=true`
- `run_state=awaiting_approval`
- at least one step summary contains `in-app approval handoff`

Approve the pending voice request and confirm resume:

```bash
VOICE_APPROVAL_ID="$(echo "$VOICE_DESTRUCTIVE_JSON" | jq -r '.approval_request_id')"
test -n "$VOICE_APPROVAL_ID" && [ "$VOICE_APPROVAL_ID" != "null" ]
pa_daemon agent approve --workspace "$WORKSPACE" --approval-id "$VOICE_APPROVAL_ID" --actor-id actor.requester --phrase "GO AHEAD"
```

Expected: approval succeeds and the resumed run completes.

Run voice-origin destructive request with in-app confirmation:

```bash
VOICE_CONFIRMED_JSON="$(pa_daemon agent run --workspace "$WORKSPACE" --request "delete file /tmp/manual-voice-handoff-confirmed.txt" --origin voice --in-app-approval-confirmed=true)"
echo "$VOICE_CONFIRMED_JSON"
```

Expected:

- `approval_required=false`
- `run_state=completed`

### 5.7 Delegation allow/deny

Check deny before grant:

```bash
pa_daemon delegation check --workspace "$WORKSPACE" --requested-by actor.alice --acting-as actor.bob --scope-type EXECUTION
```

Expected: `allowed=false`.

Grant, re-check, revoke:

```bash
GRANT_JSON="$(pa_daemon delegation grant --workspace "$WORKSPACE" --from actor.alice --to actor.bob --scope-type EXECUTION)"
echo "$GRANT_JSON"
RULE_ID="$(echo "$GRANT_JSON" | jq -r '.id')"
test -n "$RULE_ID" && [ "$RULE_ID" != "null" ]
pa_daemon delegation check --workspace "$WORKSPACE" --requested-by actor.alice --acting-as actor.bob --scope-type EXECUTION
pa_daemon delegation revoke --workspace "$WORKSPACE" --rule-id "$RULE_ID"
pa_daemon delegation check --workspace "$WORKSPACE" --requested-by actor.alice --acting-as actor.bob --scope-type EXECUTION
```

Expected:

- check after grant: `allowed=true`
- check after revoke: `allowed=false`

### 5.8 Task submit/status/cancel/retry/requeue + queued lifecycle contract + canonical agent approve (daemon persisted backend)

Submit and cancel a queued task run:

```bash
TASK_CANCEL_JSON="$(pa_daemon task submit --workspace "$WORKSPACE" --requested-by actor.requester --subject actor.requester --title "manual cancel task" --description "send an email update")"
echo "$TASK_CANCEL_JSON"
TASK_CANCEL_ID="$(echo "$TASK_CANCEL_JSON" | jq -r '.task_id')"
TASK_CANCEL_RUN_ID="$(echo "$TASK_CANCEL_JSON" | jq -r '.run_id')"
test -n "$TASK_CANCEL_ID" && [ "$TASK_CANCEL_ID" != "null" ]
test -n "$TASK_CANCEL_RUN_ID" && [ "$TASK_CANCEL_RUN_ID" != "null" ]
pa_daemon task cancel --run-id "$TASK_CANCEL_RUN_ID" --reason "manual cli cancellation"
pa_daemon task status --task-id "$TASK_CANCEL_ID"
pa_daemon --output text task status --task-id "$TASK_CANCEL_ID"

TASK_RETRY_JSON="$(pa_daemon task retry --run-id "$TASK_CANCEL_RUN_ID" --reason "manual cli retry")"
echo "$TASK_RETRY_JSON"
TASK_RETRY_RUN_ID="$(echo "$TASK_RETRY_JSON" | jq -r '.run_id')"
test -n "$TASK_RETRY_RUN_ID" && [ "$TASK_RETRY_RUN_ID" != "null" ] && [ "$TASK_RETRY_RUN_ID" != "$TASK_CANCEL_RUN_ID" ]
pa_daemon task status --task-id "$TASK_CANCEL_ID" | jq -e ".state == \"queued\" and .run_id == \"$TASK_RETRY_RUN_ID\" and .actions.can_cancel == true and .actions.can_retry == false and .actions.can_requeue == true"

TASK_REQUEUE_JSON="$(pa_daemon task requeue --run-id "$TASK_RETRY_RUN_ID" --reason "manual cli requeue")"
echo "$TASK_REQUEUE_JSON"
TASK_REQUEUE_RUN_ID="$(echo "$TASK_REQUEUE_JSON" | jq -r '.run_id')"
test -n "$TASK_REQUEUE_RUN_ID" && [ "$TASK_REQUEUE_RUN_ID" != "null" ] && [ "$TASK_REQUEUE_RUN_ID" != "$TASK_RETRY_RUN_ID" ]
pa_daemon task status --task-id "$TASK_CANCEL_ID" | jq -e ".state == \"queued\" and .run_id == \"$TASK_REQUEUE_RUN_ID\" and .actions.can_cancel == true and .actions.can_retry == false and .actions.can_requeue == true"
```

Expected:

- `task cancel` returns `cancelled=true`.
- cancelled run persists `task_state=cancelled` and `run_state=cancelled`.
- `task retry` creates a fresh queued run for the same task (`run_id` changes).
- `task requeue` creates a fresh queued run for the same task (`run_id` changes).
- `task status` includes action availability metadata (`actions.can_cancel`, `actions.can_retry`, `actions.can_requeue`).
- `--output text task status` emits a readable status summary (`task status`, `task_id:`, `actions:`) without JSON formatting.

Submit and validate queued task status contract:

```bash
TASK_JSON="$(pa_daemon task submit --workspace "$WORKSPACE" --requested-by actor.requester --subject actor.requester --title "manual persisted task" --description "send an email update")"
echo "$TASK_JSON"
TASK_ID="$(echo "$TASK_JSON" | jq -r '.task_id')"
test -n "$TASK_ID" && [ "$TASK_ID" != "null" ]

TASK_STATUS_JSON="$(pa_daemon task status --task-id "$TASK_ID" 2>/dev/null || true)"
echo "$TASK_STATUS_JSON" | jq .
echo "$TASK_STATUS_JSON" | jq -e ".task_id == \"$TASK_ID\" and ((.state // \"\")|type)==\"string\" and ((.run_state // \"\")|type)==\"string\" and (.actions.can_cancel|type)==\"boolean\" and (.actions.can_retry|type)==\"boolean\" and (.actions.can_requeue|type)==\"boolean\"" >/dev/null
TASK_FINAL_STATE="$(echo "$TASK_STATUS_JSON" | jq -r '.state // empty')"
case "$TASK_FINAL_STATE" in
  queued|running|completed|failed|awaiting_approval|blocked|cancelled) ;;
  *) echo "unexpected task lifecycle state after queued submit: state=$TASK_FINAL_STATE" >&2; exit 1 ;;
esac
```

Expected:

- `task submit` returns `task_id` + `run_id`.
- `task status` always returns typed lifecycle fields plus action availability metadata (`actions.can_cancel`, `actions.can_retry`, `actions.can_requeue`).
- lifecycle state is one of `queued|running|completed|failed|awaiting_approval|blocked|cancelled`.

Exercise canonical approval decision route (`/v1/agent/approve`):

```bash
APPROVAL_DECIDE_RUN_JSON="$(pa_daemon agent run --workspace "$WORKSPACE" --request "delete file /tmp/manual-approval-decide.txt")"
echo "$APPROVAL_DECIDE_RUN_JSON"
APPROVAL_DECIDE_ID="$(echo "$APPROVAL_DECIDE_RUN_JSON" | jq -r '.approval_request_id')"
test -n "$APPROVAL_DECIDE_ID" && [ "$APPROVAL_DECIDE_ID" != "null" ]
pa_daemon agent approve --workspace "$WORKSPACE" --approval-id "$APPROVAL_DECIDE_ID" --phrase "GO AHEAD" --actor-id actor.requester
```

Expected:

- `agent approve` returns `task_state=completed` and `run_state=completed`.
- approval/run state transitions are persisted by daemon runtime (not in-memory control state).

### 5.9 Communication fallback + idempotency

```bash
pa_daemon comm send --workspace "$WORKSPACE" --operation-id op-manual-imessage-direct --source-channel message --destination +15555550123 --message "manual imessage direct test"
pa_daemon comm attempts --workspace "$WORKSPACE" --operation-id op-manual-imessage-direct

pa_daemon comm send --workspace "$WORKSPACE" --operation-id op-manual-001 --source-channel message --destination +15555550123 --message "manual comm test" --imessage-failures 2
pa_daemon comm attempts --workspace "$WORKSPACE" --operation-id op-manual-001
pa_daemon comm send --workspace "$WORKSPACE" --operation-id op-manual-001 --source-channel message --destination +15555550123 --message "manual comm test" --imessage-failures 2

MESSAGES_FIXTURE_DB="$PWD/runtime/messages-cli-fixture.db"
cat > "$MESSAGES_FIXTURE_DB.seed.go" <<EOF
package main

import (
  "database/sql"
  "os"
  _ "modernc.org/sqlite"
)

func main() {
  target := os.Args[1]
  _ = os.Remove(target)
  db, err := sql.Open("sqlite", target)
  if err != nil { panic(err) }
  defer db.Close()
  stmts := []string{
    "CREATE TABLE message (ROWID INTEGER PRIMARY KEY, guid TEXT, text TEXT, date INTEGER, is_from_me INTEGER, handle_id INTEGER, service TEXT);",
    "CREATE TABLE chat (ROWID INTEGER PRIMARY KEY, guid TEXT);",
    "CREATE TABLE chat_message_join (chat_id INTEGER, message_id INTEGER);",
    "CREATE TABLE handle (ROWID INTEGER PRIMARY KEY, id TEXT);",
    "INSERT INTO handle(ROWID, id) VALUES (1, '+15555550100');",
    "INSERT INTO chat(ROWID, guid) VALUES (1, 'chat-guid-cli-1');",
    "INSERT INTO message(ROWID, guid, text, date, is_from_me, handle_id, service) VALUES (2001, 'imessage-guid-cli-1', 'cli inbound fixture', 1000000000, 0, 1, 'iMessage');",
    "INSERT INTO chat_message_join(chat_id, message_id) VALUES (1, 2001);",
  }
  for _, stmt := range stmts {
    if _, err := db.Exec(stmt); err != nil { panic(err) }
  }
}
EOF
go -C source/services/daemon-go run "$MESSAGES_FIXTURE_DB.seed.go" "$MESSAGES_FIXTURE_DB"
rm -f "$MESSAGES_FIXTURE_DB.seed.go"

pa_daemon channel messages ingest --workspace "$WORKSPACE" --source-scope cli-fixture-scope --source-db-path "$MESSAGES_FIXTURE_DB" --limit 10
pa_daemon connector mail ingest --workspace "$WORKSPACE" --source-scope mailbox://inbox --source-event-id mail-cli-event-1 --source-cursor 7001 --message-id "<mail-cli-event-1@example.com>" --thread-ref mail-cli-thread-1 --in-reply-to "<mail-root@example.com>" --references-header "<mail-root@example.com>" --from sender@example.com --to recipient@example.com --subject "CLI mail ingest fixture" --body "CLI mail ingest body" --occurred-at "2026-02-24T10:00:00Z"
pa_daemon connector calendar ingest --workspace "$WORKSPACE" --source-scope calendar://primary --source-event-id calendar-cli-event-1 --source-cursor 7002 --calendar-id calendar-primary --calendar-name Primary --event-uid calendar-event-uid-1 --change-type updated --title "CLI calendar ingest fixture" --notes "CLI calendar ingest body" --location "Room 42" --starts-at "2026-02-24T10:30:00Z" --ends-at "2026-02-24T11:00:00Z" --occurred-at "2026-02-24T10:10:00Z"
pa_daemon connector browser ingest --workspace "$WORKSPACE" --source-scope safari://window/cli-1 --source-event-id browser-cli-event-1 --source-cursor 7003 --window-id window-cli-1 --tab-id tab-cli-1 --page-url https://example.com --page-title "Example Domain" --event-type navigation --payload "CLI browser ingest payload" --occurred-at "2026-02-24T10:20:00Z"

pa_daemon connector bridge status --workspace "$WORKSPACE"
pa_daemon connector bridge setup --workspace "$WORKSPACE"
pa_daemon connector mail handoff --workspace "$WORKSPACE" --source-scope mailbox://inbox --source-event-id mail-cli-handoff-1 --source-cursor 7101 --message-id "<mail-cli-handoff-1@example.com>" --thread-ref mail-cli-handoff-thread-1 --from sender@example.com --to recipient@example.com --subject "CLI mail handoff fixture" --body "CLI mail handoff body" --occurred-at "2026-02-24T10:40:00Z"
pa_daemon connector calendar handoff --workspace "$WORKSPACE" --source-scope calendar://primary --source-event-id calendar-cli-handoff-1 --source-cursor 7102 --calendar-id calendar-primary --calendar-name Primary --event-uid calendar-handoff-uid-1 --change-type updated --title "CLI calendar handoff fixture" --notes "CLI calendar handoff body" --location "Room 43" --starts-at "2026-02-24T10:45:00Z" --ends-at "2026-02-24T11:15:00Z" --occurred-at "2026-02-24T10:41:00Z"
pa_daemon connector browser handoff --workspace "$WORKSPACE" --source-scope safari://window/cli-2 --source-event-id browser-cli-handoff-1 --source-cursor 7103 --window-id window-cli-2 --tab-id tab-cli-2 --page-url https://example.com --page-title "Example Domain" --event-type navigation --payload "CLI browser handoff payload" --occurred-at "2026-02-24T10:42:00Z"
```

Expected:

- direct message send returns `success=true`, exactly one logged attempt, and resolved channel is either `imessage` or `twilio` depending on channel mapping state.
- first send falls back to Twilio (`result.Channel` should be `twilio`).
- attempts shows 3 attempts (2 iMessage failures + 1 Twilio success).
- second send with same `operation-id` should be replay-safe (`result.IdempotentReplay=true`).
- messages ingest returns `source=apple_messages_chatdb` with non-negative counters (`polled`, `accepted`, `replayed`); accepted count may be `0` when no new fixtures qualify.
- mail ingest returns `source=apple_mail_rule`, `accepted=true`, and persisted `event_id`/`thread_id` values.
- calendar ingest returns `source=apple_calendar_eventkit`, `accepted=true`, and persisted `event_id`/`thread_id` values.
- browser ingest returns `source=apple_safari_extension`, `accepted=true`, and persisted `event_id`/`thread_id` values.
- `connector bridge status` reports watcher queue readiness for `mail`, `calendar`, and `browser` sources.
- `connector bridge setup` returns `ensure_applied=true` and `status.ready=true`.
- `connector <mail|calendar|browser> handoff` returns `queued=true` with a non-empty `file_path`.

### 5.10 Automation create/list/run

Create ON_COMM_EVENT trigger:

```bash
pa_daemon automation create --workspace "$WORKSPACE" --subject actor.requester --trigger-type ON_COMM_EVENT --filter '{"channels":["message"]}'
pa_daemon automation list --workspace "$WORKSPACE"
```

Run same event twice:

```bash
pa_daemon automation run comm-event --workspace "$WORKSPACE" --event-id manual-evt-1 --channel message --body "hello" --sender sender@example.com
pa_daemon automation run comm-event --workspace "$WORKSPACE" --event-id manual-evt-1 --channel message --body "hello" --sender sender@example.com
```

Expected:

- first run result has `Created=1`
- second run result has `Created=0` (idempotent replay)

### 5.11 Inspect + retention + context

Inspect run details:

```bash
RUN_JSON="$(pa_daemon agent run --workspace "$WORKSPACE" --request 'send an email to recipient@example.com saying "inspect flow update"')"
echo "$RUN_JSON"
RUN_ID="$(echo "$RUN_JSON" | jq -r '.run_id')"
test -n "$RUN_ID" && [ "$RUN_ID" != "null" ]
pa_daemon inspect run --run-id "$RUN_ID"
```

Inspect transcript/memory:

```bash
pa_daemon inspect transcript --workspace "$WORKSPACE" --limit 20
pa_daemon inspect memory --workspace "$WORKSPACE" --limit 20
```

Retention and context:

```bash
pa_daemon retention purge --trace-days 7 --transcript-days 7 --memory-days 7
pa_daemon retention compact-memory --workspace "$WORKSPACE" --owner actor.requester --apply
pa_daemon context samples --workspace "$WORKSPACE" --task-class chat --limit 20
pa_daemon context tune --workspace "$WORKSPACE" --task-class chat
```

Expected: commands succeed and return JSON payloads.

## 6) Twilio Channel Manual Tests (Local Mock Endpoint)

### 6.1 Twilio setup/check

```bash
pa_daemon connector twilio set --workspace "$WORKSPACE" --account-sid AC123 --auth-token twilio-local-token --sms-number +15555550001 --voice-number +15555550002 --endpoint http://127.0.0.1:19080
pa_daemon connector twilio get --workspace "$WORKSPACE"
pa_daemon connector twilio check --workspace "$WORKSPACE"
pa_daemon channel mapping list --workspace "$WORKSPACE" --channel message
pa_daemon channel mapping list --workspace "$WORKSPACE" --channel voice
```

Expected:

- `connector twilio set` configures one shared Twilio connector profile (config-once) for both SMS and voice.
- `connector twilio get` returns both `sms_number` and `voice_number` from the same connector record.
- `connector twilio check` returns `success=true`.
- mapping parity remains intact after config-once setup: `twilio` is enabled on both `message` and `voice` channels.

### 6.2 SMS workflow + replay-safe ingest

```bash
pa_daemon connector twilio sms-chat --workspace "$WORKSPACE" --to +15555550999 --message "twilio sms chat test"

pa_daemon connector twilio ingest-sms --workspace "$WORKSPACE" --skip-signature=true --from +15555550999 --to +15555550001 --body "inbound sms one" --message-sid SMINMANUAL1 --account-sid AC123
pa_daemon connector twilio ingest-sms --workspace "$WORKSPACE" --skip-signature=true --from +15555550999 --to +15555550001 --body "inbound sms one" --message-sid SMINMANUAL1 --account-sid AC123
```

Expected:

- `sms-chat` returns success turn.
- first ingest: `accepted=true`, `replayed=false`
- second ingest same `message-sid`: `accepted=true`, `replayed=true`

### 6.3 Voice workflow + replay-safe ingest

```bash
START_CALL_JSON="$(pa_daemon connector twilio start-call --workspace "$WORKSPACE" --to +15555550999 --twiml-url https://agent.local/twiml/voice)"
echo "$START_CALL_JSON"
CALL_SID="$(echo "$START_CALL_JSON" | jq -r '.call_sid')"
test -n "$CALL_SID" && [ "$CALL_SID" != "null" ]

pa_daemon connector twilio ingest-voice --workspace "$WORKSPACE" --skip-signature=true --provider-event-id voice-manual-1 --call-sid "$CALL_SID" --account-sid AC123 --from +15555550002 --to +15555550999 --direction outbound-api --call-status in-progress --transcript "voice transcript one" --transcript-direction INBOUND
pa_daemon connector twilio ingest-voice --workspace "$WORKSPACE" --skip-signature=true --provider-event-id voice-manual-1 --call-sid "$CALL_SID" --account-sid AC123 --from +15555550002 --to +15555550999 --direction outbound-api --call-status in-progress --transcript "voice transcript one" --transcript-direction INBOUND

pa_daemon connector twilio call-status --workspace "$WORKSPACE" --call-sid "$CALL_SID"
pa_daemon connector twilio transcript --workspace "$WORKSPACE" --call-sid "$CALL_SID" --limit 20
```

Expected:

- first ingest voice: `accepted=true`, `replayed=false`
- second ingest same provider-event-id: `replayed=true`
- call-status shows session status `in_progress`
- transcript contains voice events, including transcript event.

## 7) Conversational Webhook Runtime Manual Tests

Run webhook server with conversational mode enabled:

```bash
pa_daemon --timeout 70s connector twilio webhook serve \
  --workspace "$WORKSPACE" \
  --listen 127.0.0.1:8088 \
  --signature-mode bypass \
  --cloudflared-mode auto \
  --assistant-replies=true \
  --assistant-task-class chat \
  --voice-response-mode twiml \
  --run-for 60s
```

Expected serve output includes versioned webhook paths (`/<project-name>/v1/connector/twilio/{sms|voice}`), where `<project-name>` defaults from daemon process name (or `PA_PROJECT_NAME` when set).
When cloudflared is installed and reachable, `sms_webhook_url` / `voice_webhook_url` resolve to public URLs; otherwise command continues with local URLs and a warning.

While it is running, send callbacks from another terminal.

```bash
PROJECT_NAME="${PA_PROJECT_NAME:-personalagent}"
SMS_WEBHOOK_PATH="/${PROJECT_NAME}/v1/connector/twilio/sms"
VOICE_WEBHOOK_PATH="/${PROJECT_NAME}/v1/connector/twilio/voice"
```

### 7.1 Inbound SMS webhook turn

```bash
curl -s -X POST "http://127.0.0.1:8088${SMS_WEBHOOK_PATH}" \
  --data-urlencode "From=+15555550999" \
  --data-urlencode "To=+15555550001" \
  --data-urlencode "Body=Please reply from webhook mode" \
  --data-urlencode "MessageSid=SMWEBHOOKMANUAL1" \
  --data-urlencode "AccountSid=AC123"
```

Expected JSON includes:

- `accepted=true`
- `assistant_reply` (from model)
- `assistant_delivered=true` (using mock Twilio endpoint)

### 7.2 Inbound voice webhook turn

```bash
curl -s -X POST "http://127.0.0.1:8088${VOICE_WEBHOOK_PATH}" \
  --data-urlencode "CallSid=CAWEBHOOKMANUAL1" \
  --data-urlencode "AccountSid=AC123" \
  --data-urlencode "From=+15555550999" \
  --data-urlencode "To=+15555550002" \
  --data-urlencode "Direction=inbound" \
  --data-urlencode "CallStatus=in-progress" \
  --data-urlencode "SpeechResult=Hello from voice webhook"
```

Expected XML response contains:

- `<Response>`
- `<Gather ...>`
- generated assistant text (for mock path this is `mock reply`)

Post-check evidence:

```bash
pa_daemon connector twilio transcript --workspace "$WORKSPACE" --limit 30
pa_daemon connector twilio call-status --workspace "$WORKSPACE" --limit 20
```

## 8) Daemon Validation Guide

For daemon transport/control/realtime validation steps, use:

- `docs/tests-daemon.md`

## 9) Live Twilio/Carrier-Network Validation

For real public tunnel + carrier path validation, run:

```bash
./tools/scripts/twilio_live_cli_smoke.sh all
```

Default behavior is daemon-first: if OpenAI/Twilio are already configured in the daemon workspace, the script reuses that config and does not require re-exporting those secret env vars.
Strict mode still fails with clear missing-env guidance for anything not already configured (for `all`, this still includes `PUBLIC_BASE_URL`).
For offline/manual suites where live Twilio execution should be skipped without failing:

```bash
./tools/scripts/twilio_live_cli_smoke.sh all-skip-missing-env
# or:
./tools/scripts/twilio_live_cli_smoke.sh all --skip-missing-env
```

Detailed runbook:

- `docs/ops/twilio-live-cli-smoke.md`

## 10) Cleanup

Stop mock servers and remove local DB:

```bash
kill "$DAEMON_PID" 2>/dev/null || true
kill "$MOCKS_PID" 2>/dev/null || true
rm -f "$DB_PATH"
```
