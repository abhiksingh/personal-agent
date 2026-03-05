# AGENTS

This file defines public-safe guidance for coding agents working in this repository.

## Start Here

1. Read `README.md` and `CONTRIBUTING.md`.
2. Read `docs/spec/bootstrap.md`.
3. Load `docs/spec/spec.md` and `docs/spec/data-model.md` only when runtime or schema contracts are changing.
4. For UI or app behavior changes, read `docs/spec/spec-ui.md`, `docs/tests-ui.md`, and only the panel guide(s) touched under `docs/tests-ui/`.
5. For packaging or install-flow work, read `docs/ops/macos-daemon-packaging.md`.

## Core Principles

- Simplicity first: prefer the smallest coherent change.
- Keep source, docs, and validation in sync.
- Do not assume private workstation paths, private task ledgers, or unpublished internal artifacts exist.
- Route generated output to `out/` and keep it out of Git.

## Working Rules

- Prefer diff-first context loading:
  - `git diff --name-only`
  - `git diff --cached --name-only`
  - `git ls-files --others --exclude-standard`
- Load touched files plus direct canonical contracts before opening broader docs.
- Keep `docs/spec/spec.md` product-first and `docs/spec/data-model.md` schema-first.
- Use repo-relative paths in docs and comments.
- When behavior changes, update the matching manual guides in `docs/tests-cli.md`, `docs/tests-daemon.md`, and/or `docs/tests-ui.md`.
- If manual test steps change, update the matching runner script in the same change.
- Save long command output to logs and inspect failure-only lines first with `tools/scripts/parse_failure_lines.sh <log-file>`.
- Run focused checks first, then `tools/scripts/check_harness.sh` before finalizing.
- Do not add internal planning, backlog, or archive artifacts to the public branch.

## Completion Checklist

1. Acceptance criteria are satisfied.
2. Relevant source, spec, and manual-test docs are aligned.
3. `tools/scripts/check_harness.sh` passes.
4. Relevant module tests/builds pass.
5. No generated `out/`, editor, or workstation-specific artifacts are tracked.
