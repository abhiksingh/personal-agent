# PersonalAgent CLI

This module hosts the canonical Go CLI for PersonalAgent (`personalagent/runtime/cli`).

## Layout

- `cmd/personal-agent`: CLI entrypoint
- `internal/cliapp`: command/runtime orchestration
- `internal/commands`: command composition helpers
- `internal/client`: transport client helpers
- `internal/auth`: auth and runtime-profile helpers
- `internal/output`: human and machine output formatting

## Related Modules

- Daemon runtime: [`../../services/daemon-go`](../../services/daemon-go)
- Shared contracts: [`../../packages/contracts`](../../packages/contracts)
