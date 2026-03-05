# Knowledge Map

Canonical map of where public repository knowledge lives.

| File | Purpose | Update Trigger |
|---|---|---|
| `README.md` | Public overview, prerequisites, and quickstart commands | Entry-point setup or contributor workflow changes |
| `CONTRIBUTING.md` | Public contribution workflow | Review/validation expectations change |
| `AGENTS.md` | Repo-local guidance for coding agents | Agent workflow or repo-check expectations change |
| `docs/context/index.yaml` | Machine-readable context-loading index | Docs topology or selective-load guidance changes |
| `docs/spec/bootstrap.md` | Compact bootstrap of canonical invariants | Foundational invariant or read-order change |
| `docs/spec/spec.md` | Product behavior, runtime policy, acceptance criteria | Product/runtime behavior changes |
| `docs/spec/data-model.md` | Persistence entities, invariants, index priorities | Schema or data invariant changes |
| `docs/spec/spec-ui.md` | Canonical UI behavior and acceptance criteria | User-visible app flow or interaction changes |
| `docs/spec/connector-authoring-kit/README.md` | Connector authoring workflow and templates | Connector implementation workflow changes |
| `docs/tests-cli.md` | CLI manual validation guide | User-testable CLI behavior changes |
| `docs/tests-daemon.md` | Daemon/control-plane manual validation guide | User-testable daemon behavior changes |
| `docs/tests-ui.md` | UI manual test index | UI validation topology changes |
| `docs/tests-ui/*.md` | Panel-specific UI validation flows | Shipped UI flow changes |
| `docs/harness/ARCHITECTURE.md` | Domain boundaries and dependency direction | Architecture boundary or layering changes |
| `docs/harness/SECURITY.md` | Security constraints and checks | Auth, secret, approval, or trust-boundary policy changes |
| `docs/harness/RELIABILITY.md` | Delivery semantics and failure handling | Reliability guarantees change |
| `docs/harness/QUALITY_SCORE.md` | Public quality baseline | Test/release posture changes |
| `docs/ops/macos-daemon-packaging.md` | macOS packaging and install guidance | Packaging or local install flow changes |
| `docs/ops/twilio-live-cli-smoke.md` | Live Twilio smoke validation workflow | Live smoke steps or prerequisites change |

## Drift Rules

- If behavior changes, update `docs/spec/spec.md` in the same change.
- If schema or invariants change, update `docs/spec/data-model.md` in the same change.
- If UI behavior changes, update `docs/spec/spec-ui.md`, `docs/tests-ui.md`, and the touched `docs/tests-ui/*.md` panel guide(s).
- If CLI or daemon manual steps change, update `docs/tests-cli.md` and/or `docs/tests-daemon.md` plus the matching runner script.
- Keep public docs repo-relative and free of workstation-specific absolute paths.
