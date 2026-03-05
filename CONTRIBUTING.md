# Contributing to PersonalAgent

PersonalAgent is a macOS-first autonomous assistant project under active development. Contributions are welcome, but the bar is deliberate: changes should preserve determinism, auditability, and operational clarity.

## Before You Start

- Check existing GitHub Issues before opening a new one.
- For substantial changes, open or comment on an issue first so scope and direction stay aligned.
- Read [README.md](./README.md) for the current project shape and [SECURITY.md](./SECURITY.md) before reporting anything security-sensitive.

## Development Expectations

- Keep changes narrow and coherent. One pull request should tell one story.
- Match existing code and documentation standards. This repo prefers explicit contracts over implicit behavior.
- Update relevant docs when behavior, commands, or user-visible flows change.
- Do not commit generated build output, logs, local databases, secrets, or machine-specific workspace files.

## Local Validation

Run the harness before opening a pull request:

```bash
tools/scripts/check_harness.sh
```

Use targeted runners when your change is scoped:

```bash
tools/scripts/run_tests_cli.sh
tools/scripts/run_tests_daemon.sh
tools/scripts/run_tests_ui.sh
```

## Pull Request Guidelines

- Explain the user or operator problem the change addresses.
- Describe the behavioral change, not just the file delta.
- Note risks, tradeoffs, and any follow-up work you are intentionally leaving out.
- Include validation evidence for the paths you changed.

## Documentation Changes

If you change:

- CLI behavior: update `docs/tests-cli.md` when the user-testable flow changes.
- Daemon behavior: update `docs/tests-daemon.md` when the user-testable flow changes.
- App/UI behavior: update `docs/tests-ui.md` and the relevant panel doc when the user-testable flow changes.

## Security

Do not open public issues for vulnerabilities or suspected secret exposure. Report them privately using [SECURITY.md](./SECURITY.md).
