# Agent Bootstrap Prompt Template

Use this prompt as the starting point for any new connector task.

```text
You are implementing connector `<connector_id>` for PersonalAgent.

Read and follow:
- docs/spec/spec.md
- docs/spec/data-model.md
- docs/spec/connector-authoring-kit/README.md
- docs/spec/connector-authoring-kit/file-touch-map.md
- docs/spec/connector-authoring-kit/implementation-checklist.md

Connector brief (authoritative):
- connector_id: <connector_id>
- plugin_id: <plugin_id>
- worker_type: <worker_type>
- display_name: <display_name>
- capabilities: <capability_list>
- logical_channels (optional): <app|message|voice entries>
- ingest_required: <true|false>
- permissions_required: <none|describe>

Hard constraints:
1) Keep connector IDs and plugin IDs stable and canonical.
2) Worker lifecycle must stay manifest-driven and daemon-supervised.
3) Worker handshake metadata and capability registration must be valid.
4) Worker execute endpoint must enforce daemon-issued bearer auth.
5) Side-effecting behavior must be idempotent and auditable.
6) Do not expose secrets in config/status/evidence/log payloads.
7) Add/adjust tests for runtime, dispatch, status surfaces, and ingest paths (if any).
8) Sync docs/manual test runners when user-testable behavior changes.

Execution instructions:
1) Propose a short implementation plan and then execute.
2) Implement all required files from the touch map that apply to this connector.
3) Use deterministic error/status reasons in connector status/test responses.
4) Validate with targeted tests and summarize any unrun checks.
5) Return:
   - changed files
   - behavior summary
   - risk notes
   - follow-up items
```
