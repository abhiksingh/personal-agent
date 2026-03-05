# UI Tests: Unsigned DMG Install, Gatekeeper Override, and TCC Attribution

Source: [full guide](../tests-ui/full.md)

Use this checklist on a clean macOS user profile (or clean machine) for unsigned/ad-hoc local/internal release validation.

## A) Install Path (Unsigned DMG)

1. Build release artifacts:

```bash
./tools/scripts/package_macos_app_release.sh --output-dir out/dist/macos-release
```

2. Copy `out/dist/macos-release/PersonalAgent-unsigned.dmg` to `~/Downloads`.
3. Mount the DMG in Finder and drag `PersonalAgent.app` into `/Applications`.
4. Eject the mounted DMG.

Expected:

- `PersonalAgent.app` exists in `/Applications`.
- `PersonalAgent-unsigned.dmg`, `SHA256SUMS.txt`, and `release-manifest.json` exist under `out/dist/macos-release`.

## B) Gatekeeper Override

1. From `/Applications`, open `PersonalAgent.app`.
2. If launch is blocked, perform one of these override paths:
   - Control-click (or right-click) `PersonalAgent.app` -> `Open` -> confirm `Open`.
   - Or open `System Settings > Privacy & Security` and click `Open Anyway` for `PersonalAgent`.
3. Re-open `PersonalAgent.app` after override.

Expected:

- App launches after one explicit trust override flow.
- `Home` `Finish Setup` card and `Configuration > Setup` surface first-run trust guidance with explicit `Open`/`Open Anyway` steps.
- `Open Security Settings` and `Retry Setup Checks` actions are visible and actionable.

## C) Daemon Setup From App Bundle

1. In app, go to `Configuration > Setup` and save an `Assistant Access Token`.
2. Go to `Configuration > Advanced`.
3. Run `Install` (or `Repair` if setup is stale).
4. Verify setup status transitions to completion and runtime checks recover.

Expected:

- Install/repair succeeds from bundled helper assets.
- If app is not in `/Applications`, status shows deterministic remediation: `Move PersonalAgent.app to /Applications before running daemon install or repair.`

## D) Daemon Launch Identity

1. In Terminal, verify the launch agent and resolved program path:

```bash
launchctl print "gui/$(id -u)/com.personalagent.daemon" | rg -n "program|Program|personal-agent-daemon|Personal Agent Daemon.app"
```

2. Verify helper install path exists:

```bash
test -d "$HOME/Library/Application Support/personal-agent/daemon/Personal Agent Daemon.app"
```

Expected:

- LaunchAgent references `Personal Agent Daemon.app/Contents/MacOS/personal-agent-daemon`.
- Installed helper path is under `~/Library/Application Support/personal-agent/daemon/`.

## E) Connector Permission Prompt Attribution (TCC)

1. Open `Connectors` and trigger a permission request for a connector that prompts macOS (for example Calendar, Messages/Mail automation, or Finder automation).
2. When the macOS permission prompt appears, verify the requesting app is `Personal Agent Daemon`.
3. After granting/denying, return to app and run the connector status/permission refresh action.
4. Confirm connector card status reflects updated permission state.

Expected:

- Permission prompts are attributed to `Personal Agent Daemon` (not Terminal or `PersonalAgent.app`).
- Post-prompt refresh updates connector remediation/status deterministically.

## F) Reset + Retry (If Prompt State Is Stale)

If prompts were dismissed/denied and do not reappear, reset and retry:

```bash
tccutil reset AppleEvents com.personalagent.daemon || true
tccutil reset Calendar com.personalagent.daemon || true
```

Then rerun connector permission request from app and revalidate attribution + status refresh.
