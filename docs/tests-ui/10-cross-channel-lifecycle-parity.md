# UI Tests: Cross-Channel Lifecycle Parity (Manual-Only)

Source: [index](../tests-ui.md)

## Goal

Validate canonical lifecycle parity for `app`, `message`, and `voice` orchestration paths without automation-driven UI tests.

## Preconditions

1. Launch app in ready fixture or equivalent deterministic local state.
2. Ensure `Chat`, `Communications`, `Tasks`, `Approvals`, and `Inspect` are reachable.
3. Use an actor/principal allowed to submit chat turns.

## Steps

1. Open `Chat`.
2. Submit prompt: `Send an email update to finance`.
3. Scroll timeline to latest entries and verify:
   - assistant text includes `send_email is waiting for approval (request approval-fixture-email).`
   - approval controls are visible in-chat (`Approve and Continue`).
   - tool-result/approval rows show deterministic lifecycle state (pending/blocked as rendered by current contract).
4. Submit prompt: `Text the team that launch is complete`.
5. Scroll timeline to latest entries and verify assistant text includes `send_message completed successfully.`
6. Submit prompt: `Find files for Q1 report`.
7. Scroll timeline to latest entries and verify assistant text includes `find_files is blocked: connector permission is missing.`
8. Submit prompt: `Browse website for competitive updates`.
9. Scroll timeline to latest entries and verify assistant text includes `browse_web completed successfully.`
10. Open `Communications` and verify continuity rows under logical channels (`App`, `Message`, `Voice`) show deterministic lifecycle differences.
11. From continuity rows, verify drill-ins:
    - `App` row -> `Open Related Tasks` opens `Tasks` with source ribbon/chips.
    - `Message` row -> `Open Related Inspect` opens `Inspect` with source ribbon/chips.
    - `Voice` row -> `Open Chat` opens `Chat` with source ribbon/chips.
12. Open `Approvals` and verify pending approval row can be actioned inline with deterministic decision controls (`Action`, `Decision By`, required phrase helper, submit).
13. Return to `Chat` and verify workflow context/summary remains coherent after approvals/tasks/inspect drill-ins.

## Expected

- Cross-channel outcomes remain deterministic across the four canonical prompts.
- Latest timeline assertions are validated on newest rows (not stale viewport content).
- Drill-ins preserve context continuity and route users to the intended panel without losing source context.
- In-chat approval flow and approvals panel flow stay consistent for the same pending request.
- No panel shows raw decode-placeholder errors for these flows.

## Notes

- Automation-driven parity tests are currently paused by directive; this checklist is the canonical replacement until that directive is lifted.
- Keep app-host stabilization work tracked under `U-234`.
