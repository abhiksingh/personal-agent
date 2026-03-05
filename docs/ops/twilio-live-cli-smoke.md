# Twilio Live CLI Smoke Runbook

Use this runbook to validate real-network Twilio SMS and voice conversational flows with the Go CLI/runtime.

This complements deterministic local tests by covering:

- public webhook tunnel path
- Twilio signature validation against public URL
- carrier/network delivery and call behavior

## Prerequisites

1. Twilio account with:
   - Account SID and Auth Token
   - SMS-capable phone number (`TWILIO_SMS_NUMBER`)
   - Voice-capable phone number (`TWILIO_VOICE_NUMBER`)
2. OpenAI API key for conversational reply generation.
3. Public HTTPS tunnel routing to local webhook server (for example ngrok/cloudflared) mapped to `http://127.0.0.1:8088`.
4. Optional test handset number (`TEST_PHONE`) for outbound smoke checks.
5. Personal Agent daemon is running and reachable by CLI (`go -C source/clients/cli-go run ./cmd/personal-agent smoke` succeeds).

## Default Behavior

By default (`TWILIO_SMOKE_RUNTIME_MODE=auto`), the helper script probes daemon workspace config first:

- If OpenAI provider API key is already configured in daemon workspace, `OPENAI_API_KEY` is not required.
- If Twilio channel credentials are already configured in daemon workspace, `TWILIO_ACCOUNT_SID`, `TWILIO_AUTH_TOKEN`, `TWILIO_SMS_NUMBER`, and `TWILIO_VOICE_NUMBER` are not required.
- `PUBLIC_BASE_URL` is still required for `serve`/`all` because webhook URLs are printed and used for voice call TwiML callbacks.

## Environment Variables

Bootstrap variables (required only when daemon workspace is not already configured):

```bash
export OPENAI_API_KEY="..."
export TWILIO_ACCOUNT_SID="AC..."
export TWILIO_AUTH_TOKEN="..."
export TWILIO_SMS_NUMBER="+1..."
export TWILIO_VOICE_NUMBER="+1..."
```

Always-required for live webhook/tunnel use (`serve` and `all`):

```bash
export PUBLIC_BASE_URL="https://<your-public-tunnel-domain>"
```

Optional:

```bash
export TEST_PHONE="+1..."                        # enables outbound sms/call checks
export TEST_RUNTIME_ROOT="${PWD}/out/test-state/twilio-live"
export WORKSPACE="${WORKSPACE:-test-ws1}"        # isolated test workspace default
export DB_PATH="${DB_PATH:-$TEST_RUNTIME_ROOT/twilio-live.db}"
export PA_RUNTIME_PROFILE="${PA_RUNTIME_PROFILE:-test}"
export PA_RUNTIME_ROOT_DIR="${PA_RUNTIME_ROOT_DIR:-$TEST_RUNTIME_ROOT/runtime-root}"
export LISTEN_ADDR="127.0.0.1:8088"
export RUN_FOR="0"                               # 0 = run until interrupted
export OPENAI_ENDPOINT="https://api.openai.com/v1"
export TWILIO_ENDPOINT="https://api.twilio.com"
export PA_PROJECT_NAME="personalagent"           # optional override for default /<project-name>/v1 webhook paths
export PA_DAEMON_MODE="tcp"                      # optional CLI override
export PA_DAEMON_ADDRESS="127.0.0.1:7071"        # optional CLI override
export PA_DAEMON_AUTH_TOKEN="daemon-test-token"   # optional CLI override
export PA_DAEMON_AUTH_TOKEN_FILE=""              # optional CLI override
mkdir -p "$TEST_RUNTIME_ROOT" "$PA_RUNTIME_ROOT_DIR"
```

Force legacy env-only preflight (ignore daemon config probe):

```bash
export TWILIO_SMOKE_RUNTIME_MODE="env" # values: auto|daemon|env
```

## Helper Script Usage

Strict live mode (default):

```bash
./tools/scripts/twilio_live_cli_smoke.sh all
```

If required env vars are missing for the selected mode, strict mode exits with code `1` and prints:

- all missing variable names
- a setup hint referencing this runbook
- an explicit offline skip command hint

Offline/manual-suite skip mode (non-failing, explicit only):

```bash
./tools/scripts/twilio_live_cli_smoke.sh all-skip-missing-env
# equivalent:
./tools/scripts/twilio_live_cli_smoke.sh all --skip-missing-env
```

Skip mode exits `0` when live env vars are missing and intentionally skips Twilio live setup/run steps.

Modes:

- `setup`: secrets/provider/model/twilio channel configuration only (reuses existing daemon config by default).
- `serve`: run conversational webhook server only.
- `all`: setup + optional outbound checks (`TEST_PHONE`) + webhook serve.
- `all-skip-missing-env`: same as `all`, but returns success when live env is missing and skips execution.

## Twilio Console Webhook Configuration

Set Twilio callbacks to:

- SMS webhook URL: `${PUBLIC_BASE_URL}/${PA_PROJECT_NAME:-personalagent}/v1/connector/twilio/sms`
- Voice webhook URL: `${PUBLIC_BASE_URL}/${PA_PROJECT_NAME:-personalagent}/v1/connector/twilio/voice`

Use `HTTP POST`.

The helper script runs webhook serve with `--cloudflared-mode off` because `PUBLIC_BASE_URL` is expected to come from your existing public tunnel setup.

## Validation Workflow

1. Run helper script in `all` mode.
2. Outbound checks (if `TEST_PHONE` is set):
   - script sends one SMS using `channel twilio sms-chat`
   - script initiates one call using `channel twilio start-call`
3. Inbound SMS check:
   - text your Twilio SMS number
   - expect assistant auto-reply via conversational webhook mode
4. Inbound voice check:
   - call your Twilio voice number
   - speak a prompt
   - expect TwiML conversational turn response (voice `Gather` + assistant spoken reply)
5. Evidence checks:
   - `go -C source/clients/cli-go run ./cmd/personal-agent --db "$DB_PATH" channel twilio transcript --workspace "$WORKSPACE" --limit 20`
   - `go -C source/clients/cli-go run ./cmd/personal-agent --db "$DB_PATH" channel twilio call-status --workspace "$WORKSPACE" --limit 20`

## Teardown

- Stop the webhook process (`Ctrl+C`).
- Remove local DB if needed:
  - `rm -f "$DB_PATH"`
