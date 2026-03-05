# Connector File Touch Map

Canonical map of where connector implementation changes usually land.

## Required in Most Connector Changes

| Area | Path | Why |
|---|---|---|
| Adapter contract | `source/services/daemon-go/internal/connectors/adapters/<connector>/adapter.go` | Connector behavior implementation. |
| Worker runtime | `source/services/daemon-go/cmd/personal-agent-daemon/connector_worker_runtime.go` | Worker type routing + execute operation wiring. |
| Worker manifest | `source/services/daemon-go/cmd/personal-agent-daemon/plugin_workers_manifest.json` | Manifest-driven lifecycle bootstrap. |
| Registry wiring | `source/services/daemon-go/internal/daemonruntime/agent_delegation_service.go` | Capability-based agent execution selection. |
| UI status cards | `source/services/daemon-go/internal/daemonruntime/ui_status_status_runtime.go` | Connector cards/status/configuration output. |
| Config descriptors + plugin mapping | `source/services/daemon-go/internal/daemonruntime/ui_status_service_config_runtime.go` | Editable field schema + `connector_id -> plugin_id` mapping. |
| Connector tests | `source/services/daemon-go/internal/daemonruntime/*connector*_test.go` and adapter tests | Runtime/dispatch/status behavior coverage. |

## Conditional (Connector-Dependent)

| Area | Path | Apply When |
|---|---|---|
| Permission probing/remediation | `source/services/daemon-go/internal/daemonruntime/ui_status_service_permission_runtime.go` and `ui_status_service.go` | Connector needs permission state handling. |
| Logical channel mapping validation | `source/services/daemon-go/internal/daemonruntime/ui_status_service_mapping_helpers_runtime.go` | Connector can bind to `app|message|voice`. |
| Default channel bindings migration | `source/services/daemon-go/internal/persistence/migrations/sql/0010_channel_connector_bindings.sql` (or new migration) | Connector should be seeded or remapped by default. |
| Inbound watcher ingest adapter | `source/services/daemon-go/internal/daemonruntime/inbound_watcher_runtime.go` | Connector consumes local queued ingress events. |
| Ingest persistence service | `source/services/daemon-go/internal/daemonruntime/comm_<connector>_ingest_service.go` | Connector produces inbound comm events. |
| Transport routes and types | `source/services/daemon-go/internal/transport/server_routes_*.go`, `types_*.go`, `client_*.go` | New connector-specific control APIs are needed. |
| Auth scopes and limiter policy | `source/services/daemon-go/internal/transport/server_auth_scope.go`, `server_rate_limit_policy.go` | New control routes are added. |
| Unified tool exposure | `source/services/daemon-go/internal/daemonruntime/unified_turn_tools.go` | Connector capability should be model-callable tool. |
| Intent/planner workflow family | `source/services/daemon-go/internal/core/service/agentexec/intent.go`, `engine_planner.go` | New workflow family or typed action surface is introduced. |

## Docs and Tracking

| Area | Path | Apply When |
|---|---|---|
| Product/runtime contract docs | `docs/spec/spec.md`, `docs/spec/data-model.md` | Runtime contracts or persistence invariants changed. |
| Manual test docs | `docs/tests-cli.md`, `docs/tests-daemon.md`, `docs/tests-ui.md` | User-testable flow changed. |
| Runner scripts | `tools/scripts/run_tests_cli.sh`, `tools/scripts/run_tests_daemon.sh`, `tools/scripts/run_tests_ui.sh` | Manual steps were changed. |
