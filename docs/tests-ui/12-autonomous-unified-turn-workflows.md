# UI Tests: Autonomous Unified-Turn Workflows (Manual-Only)

Source: [index](../tests-ui.md)

## Goal

Validate end-to-end autonomous chat-to-action lifecycle behavior across canonical unified-turn state classes while automation-driven UI tests are paused.

## Preconditions

1. Launch app in deterministic fixture mode (or equivalent local setup) with reachable `Chat`, `Approvals`, `Tasks`, `Inspect`, and `Communications`.
2. Ensure an actor/principal is selected and can submit chat turns.
3. Ensure fixture prompts can exercise approval-required, blocked, completed, and realtime fallback paths.

## Scenario A: Core lifecycle state progression in Chat

1. Open `Chat`.
2. Submit prompt: `Send an email update to finance`.
3. Verify lifecycle transitions render in-order:
   - in-flight state (`Interrupt` available while sending),
   - approval-required state (assistant copy includes `send_email is waiting for approval (request approval-fixture-email).`),
   - approval controls are visible (`Approve and Continue`).
4. Submit prompt: `Text the team that launch is complete`.
5. Verify success state renders (`send_message completed successfully.`) with deterministic row status badge and action affordances (`Retry Turn`, `Open Inspect`, optional `Cancel` disablement semantics).
6. Submit prompt: `Find files for Q1 report`.
7. Verify blocked state renders (`find_files is blocked: connector permission is missing.`) with remediation action affordances (`Open Configuration`/`Open Connectors`/`Open Channels` when available).
8. Submit prompt: `Browse website for competitive updates`.
9. Verify completed state renders (`browse_web completed successfully.`).

## Scenario B: Approval-to-completion continuity

1. From the pending email approval in Chat, use inline approval controls and submit decision.
2. Open `Approvals` and verify the same request reflects updated status with deterministic decision metadata.
3. Return to `Chat` and verify timeline reflects completion/continuation for the approved turn without losing context.

## Scenario C: Cross-panel drill-in continuity from autonomous turns

1. From Chat tool/result rows, run `Open Tasks` and verify `Tasks` opens with source ribbon/chips tied to originating turn context.
2. From related rows, run `Open Inspect` and verify `Inspect` opens with scoped run/task context.
3. Open `Communications` and verify continuity rows for `App`, `Message`, and `Voice` align with latest autonomous outcomes.
4. Use continuity drill-ins (`Open Chat`, `Open Related Tasks`, `Open Related Inspect`) and verify source context remains intact.

## Scenario D: Recovery and interruption controls

1. Start one turn and click `Interrupt` while in-flight.
2. Verify deterministic interrupt status copy (`Interrupt requested…` then `Chat interrupted.`) and composer recovers.
3. Trigger one route/failure remediation state and click `Fix and Continue`; verify this is the primary action and status copy reports in-flight/progress deterministically.
4. When remediation requires `Models` or `Configuration`, verify app navigates there with handoff copy, then returning to `Chat` automatically re-checks readiness and resumes pending draft context when available.
5. For a blocked/failed turn row, verify `Retry Turn` availability and deterministic retry-disabled reasons when retry is already in progress.
6. Confirm no raw decode-placeholder or fallback shim copy appears for lifecycle summaries.

## Expected

- Autonomous unified-turn state classes are all represented deterministically: in-flight, approval-required, blocked, completed, interrupted/retryable.
- Action affordances are deterministic and context-aware (`Approve and Continue`, `Fix and Continue`, `Retry Turn`, `Open Tasks`, `Open Inspect`, remediation navigation).
- Cross-panel drill-ins preserve source context and route to intended destination panels.
- Approval flow remains coherent between Chat inline controls and Approvals panel controls.
- Recovery/interruption paths are explicit and reversible without stale or raw decode error copy.

## Notes

- Use `10-cross-channel-lifecycle-parity.md` for dedicated channel-parity assertions and `11-auth-rate-limit-realtime-hardening.md` for typed-problem/realtime resilience assertions.
- Keep app-host automation stabilization work under `U-234` until automation is re-enabled.
