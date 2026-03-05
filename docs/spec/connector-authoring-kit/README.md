# Connector Authoring Kit

Reusable package for bootstrapping new connector implementation work with coding agents.

This kit is a living contract companion to:
- `docs/spec/spec.md`
- `docs/spec/data-model.md`
- `source/services/daemon-go/internal/shared/contracts/adapter.go`

## Goals

- Standardize how coding agents scope, implement, and validate new connectors.
- Keep connector implementation quality consistent (security, idempotency, operability).
- Centralize connector interface evolution in one tracked package.

## Package Contents

| File | Purpose |
|---|---|
| `agent-bootstrap-prompt-template.md` | Copy/paste prompt template to start a connector build with a coding agent. |
| `connector-input-template.yaml` | Structured connector brief used as source-of-truth input for agent runs. |
| `implementation-checklist.md` | End-to-end execution checklist from design to validation and docs sync. |
| `file-touch-map.md` | Canonical file map of where connector changes normally go. |
| `work-item-template.md` | Task/work-item template for connector-specific implementation tracking. |
| `CHANGELOG.md` | Change history for this kit and connector interface assumptions. |

## Recommended Usage

1. Copy `connector-input-template.yaml` into a task-specific file and fill all required fields.
2. Copy `agent-bootstrap-prompt-template.md` and replace placeholders with your connector details.
3. Run the coding agent with those two artifacts as inputs.
4. Use `implementation-checklist.md` as the done gate before review.
5. If connector contracts changed, update this kit and append an entry in `CHANGELOG.md`.

## Scope Notes

- This kit covers connector authoring. It does not replace canonical runtime/UI product specs.
- When canonical contracts conflict with this kit, canonical specs win. Update this kit in the same change.
