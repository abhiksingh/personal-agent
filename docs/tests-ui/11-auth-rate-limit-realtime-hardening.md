# UI Tests: Auth Scope + Rate Limit + Realtime Hardening (Manual-Only)

Source: [index](../tests-ui.md)

## Goal

Validate deterministic remediation UX for scoped-auth denial, throttling, and realtime fallback/recovery while automation-driven UI tests are paused.

## Preconditions

1. Launch app in deterministic fixture mode (or equivalent local state) that can produce:
   - `auth_scope` typed panel problem responses,
   - `rate_limit_exceeded` typed panel problem responses,
   - realtime websocket fallback classes (capacity rejection, stale/disconnect).
2. Ensure `Chat`, `Models`, `Channels`, `Connectors`, `Automation`, `Approvals`, and `Tasks` are reachable.

## Scenario A: Shared typed panel-problem remediation contract

1. Trigger `auth_scope` for one workflow panel (for example `Tasks`).
2. Verify shared remediation card appears with deterministic actions:
   - `Open Configuration`
   - `Retry`
   - `Open Inspect`
3. While refresh is in flight, verify `Retry` is disabled with explicit reason copy.
4. Repeat for `rate_limit_exceeded` on one different panel (for example `Models`) and verify the same action contract/copy semantics.
5. Spot-check remaining impacted panels (`Chat`, `Channels`, `Connectors`, `Automation`, `Approvals`) to confirm the same remediation card/action contract is reused.
6. In Chat route/failure remediation cards, verify `Fix and Continue` remains the primary recovery action while typed panel-problem cards continue using canonical shared actions (`Open Configuration`, `Retry`, `Open Inspect`).

## Scenario B: Chat realtime fallback classification and recovery

1. Open `Chat` and trigger websocket capacity rejection (`429 rate_limit_exceeded`) while `/v1/chat/turn` remains reachable.
2. Verify chat status copy explicitly indicates realtime capacity fallback (not generic disconnect text).
3. Trigger stale-session or mid-stream disconnect fallback.
4. Verify status copy explicitly indicates stale/disconnect fallback class.
5. Open `Chat > Actions` and run `Retry Realtime Stream`.
6. Verify deterministic reconnect outcome:
   - success path: app connection state returns from `Degraded` to `Connected`;
   - failure path: deterministic retry failure copy and reconnect action remains available.

## Scenario C: Cross-panel routing/remediation continuity

1. From each typed-problem card state, click `Open Configuration` and verify navigation opens `Configuration`.
2. Return and click `Open Inspect` when context is available; verify `Inspect` opens with seeded context.
3. Click `Retry` after remediation and verify panel transitions from problem state to deterministic loading/ready/failure state without raw decode-placeholder text.
4. Confirm action naming/order matches canonical pattern (`Open Configuration` primary remediation, retry/navigation secondary).

## Expected

- Auth/rate-limit failures use one shared remediation card/action contract across all impacted panels.
- Chat keeps one primary `Fix and Continue` recovery path for route/failure remediation without replacing typed panel-problem shared action contracts.
- Retry disablement reason copy is explicit and deterministic during in-flight refresh.
- Chat realtime fallback copy is class-specific (capacity vs stale/disconnect) and `Retry Realtime Stream` behaves deterministically.
- Navigation/remediation actions preserve context continuity and avoid raw decode-placeholder failure copy.

## Notes

- This checklist is the canonical manual regression replacement while automation-driven UI testing is paused.
- Keep app-host stabilization work tracked separately under `U-234`.
