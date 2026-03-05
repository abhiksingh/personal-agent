# macOS Daemon Packaging and TCC Attribution

Use this runbook when you want Automation/TCC prompts to be attributed to `Personal Agent Daemon` (packaged daemon host) instead of a transient terminal/Codex-hosted process.

## Distribution trust model (current)

- This runbook targets local/internal developer-style packaging and install workflows.
- It does **not** provide Developer ID signing or notarization guarantees.
- On hosts enforcing Gatekeeper trust checks for transferred artifacts, first launch may require explicit override (`right-click Open` or `System Settings -> Privacy & Security -> Open Anyway`).
- In this mode, trust-block failures should be remediated first; daemon lifecycle troubleshooting starts only after the host allows the app/binary launch.

## Why this matters

- Launching daemon from `go run` under Codex/terminal can cause macOS Automation prompts to be associated with that host process chain.
- Running daemon from a stable `.app` bundle via LaunchAgent provides a consistent executable identity for TCC tracking.

## 0) Optional: Build unsigned drag-install app-host DMG (local/internal)

From repo root:

```bash
./tools/scripts/package_macos_app_release.sh \
  --output-dir "$PWD/out/dist/macos-release"
```

Expected output artifacts:

- `out/dist/macos-release/PersonalAgent.app`
- `out/dist/macos-release/PersonalAgent-unsigned.dmg`
- `out/dist/macos-release/SHA256SUMS.txt`
- `out/dist/macos-release/release-manifest.json`

DMG presentation defaults to a styled Finder install canvas (background art + deterministic icon layout) that makes the drag target explicit. If you need a plain DMG (for example in constrained/headless sessions), use:

```bash
./tools/scripts/package_macos_app_release.sh \
  --output-dir "$PWD/out/dist/macos-release" \
  --no-dmg-style
```

## 1) Package daemon as a macOS app bundle

From repo root:

```bash
./tools/scripts/package_daemon_app_macos.sh \
  --output-app "$HOME/Applications/Personal Agent Daemon.app"
```

Optional (skip signing while iterating locally):

```bash
./tools/scripts/package_daemon_app_macos.sh \
  --output-app "$HOME/Applications/Personal Agent Daemon.app" \
  --skip-sign
```

Expected output includes:

- `packaged daemon app: ...`
- `daemon executable: .../Contents/MacOS/personal-agent-daemon`

## 2) Install LaunchAgent using packaged daemon app

```bash
./tools/scripts/install_daemon_service_macos.sh \
  --daemon-app "$HOME/Applications/Personal Agent Daemon.app" \
  --auth-token-file "$HOME/.config/personal-agent/control.token" \
  --db-path "$HOME/Library/Application Support/personal-agent/runtime.db"
```

The script writes and loads `~/Library/LaunchAgents/com.personalagent.daemon.plist`.

## 3) Verify service process path

```bash
launchctl print "gui/$(id -u)/com.personalagent.daemon" | rg "program|path|state"
```

Expected daemon executable path:

- `.../Personal Agent Daemon.app/Contents/MacOS/personal-agent-daemon`

## 4) Verify daemon API availability

```bash
go -C source/clients/cli-go run ./cmd/personal-agent \
  --mode tcp \
  --address 127.0.0.1:7071 \
  --auth-token-file "$HOME/.config/personal-agent/control.token" \
  smoke
```

Expected: JSON response with `healthy=true`.

## 5) TCC reset (optional when switching identities)

If prior prompts were granted/denied under terminal/Codex identity and you want a clean prompt flow:

```bash
tccutil reset AppleEvents
```

Then trigger a Mail/Calendar/Messages/Safari automation action through the daemon and grant permission to the packaged daemon identity.

## 6) Unified local launcher (daemon + app)

For day-to-day local use, run:

```bash
./tools/scripts/launch_personal_agent.sh \
  --daemon-auth-token-file "$HOME/.config/personal-agent/control.token"
```

Behavior:

- Rebuilds daemon binary on each launcher invocation.
- Without explicit `--daemon-auth-token` / `--daemon-auth-token-file`, launcher resolves auth from app Keychain token (`service=personalagent.ui.local_dev_token.v1`, `account=daemon_auth_token`) and persists resolved token there for app/daemon parity.
- Reuses daemon if already reachable.
- Starts daemon only if needed.
- On macOS, default `--daemon-start-mode auto` prefers launchctl startup (daemon attribution path decoupled from Terminal parent process) and falls back to direct spawn if launchctl startup cannot make daemon reachable.
- Launches `PersonalAgent.app`.
- On `Ctrl+C`, stops only processes started by the script.

Useful overrides:

```bash
# Explicit app bundle (default deterministic build output)
./tools/scripts/launch_personal_agent.sh --app-bundle "$PWD/out/build/xcode-derived-data/Build/Products/Debug/PersonalAgent.app"

# Rebuild daemon + app, then launch
./tools/scripts/launch_personal_agent.sh

# Force direct spawn (bypass launchctl attribution path)
./tools/scripts/launch_personal_agent.sh --daemon-start-mode direct
```
