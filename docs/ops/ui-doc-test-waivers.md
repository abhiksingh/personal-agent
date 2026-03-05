# UI Doc/Test Consistency Waivers

Use this file only when a UI-impacting change cannot update one or more required artifacts in the same task.

Required artifacts for UI-impacting changes:

- `docs/spec/spec-ui.md`
- `docs/tests-ui.md`
- `tools/scripts/run_tests_ui.sh`

## Rules

1. Prefer updating required artifacts over adding waivers.
2. Keep waivers short-lived and scoped.
3. Set `Status` to `active` only while the exception is needed.
4. Move finished waivers to `closed`.

## Waiver Registry

| ID | Status | Scope | Waives | Reason | Owner | Expires On |
| --- | --- | --- | --- | --- | --- | --- |
| CTX-008 | active | `AppShellState` type-contract extraction to `AppShellStateModels.swift` (behavior-preserving code organization change) | spec-ui,tests-ui,run-tests-ui | Internal decomposition to reduce source/context footprint with no user-visible contract change; full UI package tests pass. | codex | 2026-03-07 |
| U-182 | closed | Chat orchestration store extraction in `AppShellState` + `ChatOrchestrationStore` | spec-ui,tests-ui,run-tests-ui | Internal refactor completed with behavior-preserving validation; waiver closed after follow-up UI doc/test sync. | codex | 2026-03-07 |

`Waives` tokens:

- `spec-ui` (waives `docs/spec/spec-ui.md`)
- `tests-ui` (waives `docs/tests-ui.md`)
- `run-tests-ui` (waives `tools/scripts/run_tests_ui.sh`)
- `all` (waives all three)
