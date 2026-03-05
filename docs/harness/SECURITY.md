# Security Baseline (MVP)

Security decisions and constraints for implementation.

## Secrets and Credentials

- Store secrets in Keychain only.
- Never persist raw credentials in SQLite or logs.
- Use scoped credentials for each connector/channel integration.

## Authorization

- `acting_as` must be explicit for each `TaskRun`.
- Cross-principal execution requires valid `DelegationRule`.
- Destructive approvals are limited to acting-as principal or delegated approvers.
- CLI-originated requests must use the same authorization and policy paths as app-originated requests.

## Data Protection

- Retention defaults to 7 days for traces, transcripts, and memory unless configured otherwise.
- PII in logs should be minimized and redacted where possible.
- Deleted/disabled memory items must be excluded from retrieval.

## Runtime Safeguards

- Destructive/non-reversible actions require approval.
- Low-confidence risk classification defaults to approval required.
- Voice destructive operations require in-app handoff and confirmation.
- Adapter execution is capability-scoped; adapters must not exceed granted connector/channel permissions.
- HTTP and WebSocket endpoints must bind to localhost by default and enforce authenticated caller identity/authorization checks before request handling.
- WebSocket upgrade/auth handshake must validate caller identity before opening any realtime stream.
- Any non-local transport exposure requires explicit configuration plus channel protection (TLS/mTLS) and equivalent policy enforcement.
- Runtime profile gate:
  - `local`: developer-friendly defaults for localhost workflows.
  - `prod`: rejects insecure defaults; daemon requires TCP + TLS + mTLS and file-backed auth token, while CLI requires file-backed auth token and disallows TLS verification bypass.

## Required Security Checks

1. Secret scan in CI before merge.
2. Policy enforcement tests for authorization and approvals.
3. Audit log integrity checks for critical action paths.
4. Local transport access control tests for daemon IPC endpoints.
5. Adapter registration allowlist/signature policy tests (or equivalent trusted-source validation in MVP).

## Static Scanner Gate and Waivers

- `tools/scripts/check_security_regressions.sh` is the canonical security gate entrypoint and must run:
  - `go mod verify`
  - `govulncheck ./...`
  - `gosec` with deterministic settings (`high` severity + `high` confidence) and machine-readable output parsing.
- Scanner findings must be resolved in code or explicitly waived in `docs/harness/security-scanner-waivers.json`.
- Waiver entries must include:
  - `rule_id`, `file`, `line`
  - `reason` (explicit rationale, not generic noise dismissal)
  - `tracked_by` (task ID that owns remediation or accepted-risk review)
  - `added_on` (date)
- Waivers are temporary risk records. Remove stale entries when findings are fixed or no longer reported.
