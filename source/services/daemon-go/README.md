# PersonalAgent Daemon Runtime

This module contains the Go daemon runtime for PersonalAgent.

## Main Components

- `cmd/personal-agent-daemon`: daemon entrypoint
- `cmd/personal-agent-migrate`: SQLite migration runner
- `internal/core`: runtime orchestration, policy, persistence, and shared contracts
- `internal/channels`: channel adapters and registries
- `internal/connectors`: connector adapters and registries
- `internal/api`: HTTP, realtime, and OpenAPI surfaces

## Related Modules

- CLI entrypoint: [`../../clients/cli-go/cmd/personal-agent`](../../clients/cli-go/cmd/personal-agent)
- Shared contracts: [`../../packages/contracts`](../../packages/contracts)

## Local Migration Example

```bash
cd source/services/daemon-go
go run ./cmd/personal-agent-migrate -db /tmp/personalagent.db
```
