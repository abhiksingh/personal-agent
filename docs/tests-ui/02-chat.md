# UI Tests: Chat Panel

Source: [full guide](../tests-ui/full.md)

Regression anchors:
- run [Autonomous Unified-Turn Workflows (Manual-Only)](./12-autonomous-unified-turn-workflows.md) for end-to-end lifecycle progression and recovery checks.
- run [Auth Scope + Rate Limit + Realtime Hardening (Manual-Only)](./11-auth-rate-limit-realtime-hardening.md) for typed-problem and realtime fallback class checks.

## 6) Chat Panel Checks

1. Select `Chat`.
2. Confirm transcript starts without seeded assistant placeholder messages and shows explicit first-turn guidance.
3. Confirm large transcript area + multiline composer are visible, and composer shows `Advanced Override` collapsed by default with `Auto` summary.
3a. Expand `Advanced Override`, verify `Acting As` picker is available, select a non-default actor, and confirm disclosure stays visible until returned to `default`.
4. Confirm transcript renders typed timeline rows (user/assistant rows plus tool/approval/system rows when present), not a message-only fallback transcript path.
5. Confirm the composer exposes one autonomous `Send` path and no `Ask`/`Act` mode override controls.
6. Type multiline content using `Shift+Enter` for a newline.
7. Press `Enter` to send.
8. Observe realtime streaming state copy in the header/composer and token deltas appearing in transcript when daemon realtime is reachable.
9. While a chat turn is in flight, click `Interrupt`.
10. Verify chat request is cancelled and status copy updates deterministically (`Interrupt requested…` then `Chat interrupted.`).
11. Submit representative action prompts and verify deterministic autonomous outcome rendering for:
   - approval-required path
   - successful action path
   - blocked-readiness path
11a. In ready smoke-fixture mode, validate canonical prompt outcomes:
   - `Send an email update to finance` -> assistant text includes `send_email is waiting for approval (request approval-fixture-email).` and timeline shows approval controls.
   - `Text the team that launch is complete` -> assistant text includes `send_message completed successfully.`
   - `Find files for Q1 report` -> assistant text includes `find_files is blocked: connector permission is missing.`
   - `Browse website for competitive updates` -> assistant text includes `browse_web completed successfully.`
11b. Before asserting assistant/tool outcomes, scroll the timeline to newest entries to avoid stale viewport checks against older rows.
11c. While new timeline items are being appended (streaming tokens or new turn rows), confirm the transcript auto-follows to the latest row; after the turn settles, manually scroll up/down and confirm the view no longer force-snaps to bottom until a new message is appended.
11d. Submit one prompt that returns markdown with fenced code (for example JSON/YAML) and verify chat renders markdown formatting plus code-block containers with monospaced content and language labels when provided.
11e. Submit one prompt that returns markdown image syntax (`![alt](https://...)`) and verify chat renders the image inline with deterministic loading/failure placeholder copy plus visible alt-text caption.
11f. Submit one prompt that returns a markdown table and verify assistant row layout expands to available transcript width for wide structured output before requiring horizontal scroll.
12. In multi-step tool workflows, verify tool/approval rows expose chain context labels (`Chain n`, `Step x of y`) and row status badges map to deterministic labels (`Pending`, `Running`, `Blocked`, `Complete`, `Failed`).
13. For tool and approval timeline rows, verify action buttons are deterministic (`Resume Turn`, `Retry Turn`, `Open Inspect`, `Open Approvals`, `Open Tasks`, `Cancel`) and failed/blocked tool rows include inline remediation actions (`Open Configuration`, `Open Connectors`, `Open Channels`) when applicable.
14. In one `approval_request` row:
   - verify chat keeps `Open Approvals` available for full evidence and canonical decision controls.
   - for `policy` risk approvals, verify `Low-Risk Inline Decision` disclosure is available and collapsed by default.
   - expand it and verify bounded inline controls (`Action`, `Decision By`, required phrase helper for approve, confirm dialog, `Undo Draft`) plus deterministic validation/in-flight/result copy.
   - for destructive/unknown-risk approvals, verify inline decision controls are not shown and copy routes user to `Approvals`.
15. Trigger one remediation/timeline action and verify row-level action status copy updates inline and action buttons disable while that action is in flight.
16. Expand one timeline row `Details` disclosure and verify technical metadata appears; confirm disclosures are collapsed by default on first render.
17. In a workspace with no enabled ready chat model route, click `Send` and verify preflight blocks daemon turn submission with remediation copy.
18. Verify chat shows one primary route-remediation action (`Fix and Continue`) plus explicit fallback controls (`Open Models`, `Check Again`), and does not append raw daemon 400 text as an assistant transcript message.
19. Click `Fix and Continue`; if setup is required, verify app opens `Models` with handoff copy. Configure/enable/select a valid chat route, return to `Chat`, and verify remediation re-check runs automatically and resumes the pending message when draft context exists.
19a. From `Models` quickstart `Open Chat Test`, confirm chat deep-link includes `Route: <Provider> • <Model>` drill-in chip and quickstart route-validation status copy.
19b. If chat composer is empty before quickstart deep-link, confirm a seeded quickstart validation draft is present; if composer already had user text, confirm draft is preserved.
20. Trigger one chat turn with daemon realtime intentionally unavailable (for example by blocking websocket endpoint while keeping `/v1/chat/turn` reachable) and verify fallback status copy is shown.
20a. Trigger realtime guardrail failures and verify deterministic fallback messaging classes:
   - websocket capacity rejection (`429 rate_limit_exceeded`) -> chat status mentions realtime capacity and fallback mode.
   - stale/heartbeat disconnect during stream -> chat status mentions realtime session expiration or disconnect with fallback mode.
20b. Open `Actions` in chat header, run `Retry Realtime Stream`, and verify deterministic reconnect result copy:
   - success -> `Realtime stream reachable. Next turn will stream live.`
   - failure -> class-specific fallback copy remains visible with guidance.
20c. Submit one intentionally long-form prompt on a slower route/model and verify the assistant response completes without abrupt timeout-driven truncation (no early cut-off with an unfinished sentence when transport remains healthy).
21. Trigger one non-route chat failure condition (for example daemon connectivity or auth failure) and verify chat shows one primary `Fix and Continue` action with explicit fallback controls (`Refresh Daemon`, `Restore Prompt`, `Open Configuration`).
21a. Trigger chat typed-problem failures (`auth_scope` or `rate_limit_exceeded`) and verify the shared panel remediation card appears with deterministic actions (`Open Configuration`, `Retry`, `Open Inspect`); when retry is already in progress, verify `Retry` is disabled with explicit reason copy.
21b. Trigger a transient transport interruption after realtime output has already appeared, then verify chat reconciles from history when available (no false hard-failure `Could not reach daemon while loading Chat` blip for the completed turn).
22. Verify partial streamed output remains visible when daemon receipt fails after realtime lifecycle completion.
23. Use `Restore Prompt` fallback and verify the last failed prompt is restored into the composer for user-controlled resend.
24. Verify chat header status + composer footnote do not repeat provider/model route metadata; route ownership remains in `Effective Workflow Context`.
25. During an in-flight turn and after one successful turn, verify a single `Effective Workflow Context` card is visible; expand it and confirm provider/model route context appears there.
25a. In `Effective Workflow Context`, verify the default body renders summary-first reliability copy in order: `What happened`, `What needs action`, and `What next`.
25b. When chat timeline metadata includes response shaping, verify `Effective Workflow Context` shows shaping badges for channel/profile/persona source.
26. Expand `Effective Workflow Context > Details` and verify technical metadata rows (route source, task/run state and IDs, correlation ID, approval/clarification hints when present) are available when daemon data exists; collapse by default on first render.
26b. In `Effective Workflow Context > Details`, verify shaping rows are present when available (`Shaping Channel`, `Shaping Profile`, `Persona Source`, `Shaping Guardrails`, `Shaping Instructions`).
26a. Expand `Route + Tool Explainability` and verify deterministic state handling:
   - loading copy appears while refresh is in flight,
   - ready state shows route reason codes/explanations and tool catalog/policy decision disclosures without duplicating selected provider/model/source rows,
   - empty/failure states show guided remediation actions (`Refresh`, `Open Models`, `Open Inspect`) with explicit disabled reason copy when `Open Inspect` lacks context.
27. In `Effective Workflow Context`, click `Open Related Tasks` and verify navigation switches to `Tasks` with identifier search prefilled from run/task/correlation or route fallback context.
28. Return to `Chat`, click `Open Related Inspect` from the same context card, and verify navigation switches to `Inspect` with run filter or metadata search prefilled from chat context.

Expected:

- Transcript starts clean (no synthetic assistant bootstrap turn) with explicit empty-state guidance.
- Chat empty state surfaces direct remediation CTAs with deterministic visibility (`Open Configuration` when token missing, `Fix and Continue` + `Open Models`/`Check Again` fallback when route remediation is active, runtime refresh when disconnected/degraded).
- `Shift+Enter` inserts newline.
- `Enter` sends.
- Chat uses a single autonomous `Send` path and does not expose ask/act mode overrides.
- Autonomous send supports tool/action progression via canonical typed timeline items without manual mode switching.
- Canonical smoke-fixture autonomous outcomes remain deterministic (`send_email` awaiting approval, `send_message` completed, `find_files` blocked by connector permission, `browse_web` completed).
- Outcome validation is performed against newest timeline entries (not stale viewport rows) before asserting status/action controls.
- Transcript auto-follows while new timeline items are appended and allows manual scrolling after activity settles; it only jumps back to newest when a new message/timeline item arrives.
- Message rows render markdown and fenced code snippets clearly (including json/yaml language fences) with readable monospaced code containers.
- Message rows render markdown image links inline with deterministic loading/failure states and visible alt text when provided.
- Assistant message rows keep readable width for prose but expand to available transcript width for wide markdown blocks (for example tables/code) so structured output is not constrained to the prose max width.
- Chat composer keeps `Acting As` under collapsed-by-default `Advanced Override`; selecting non-default actor keeps override visible until reset.
- When selected actor is outside active identity scope, chat submit is blocked with deterministic delegation-safe validation copy.
- User message appears in transcript.
- User-message bubble width tracks message content (short prompts should not expand to the assistant max-width layout).
- Transcript renders typed timeline items and does not rely on legacy message-only fallback branches.
- Assistant tokens stream via realtime when available and reconcile to final daemon `chat.turn` response text.
- `Interrupt` is visible only while streaming, sends best-effort cancel signal, and cancels local in-flight request deterministically.
- Iterative tool workflows show chain context labels (`Chain n`, `Step x of y`) and deterministic row-state badges (`Pending`, `Running`, `Blocked`, `Complete`, `Failed`).
- Tool and approval timeline rows provide deterministic action affordances (including `Resume Turn`) with explicit disabled reasons, inline row-level action status copy, and default-collapsed technical details.
- Approval-request timeline rows keep concise handoff/status copy and preserve `Open Approvals` canonical ownership; only low-risk (`policy`) approvals may use bounded inline fast-path controls with explicit confirmation and undo-draft affordance.
- In approval-required autonomous flows, the chat timeline approval row prioritizes `Open Approvals` as the primary decision action and keeps `Resume Turn` contextual for post-decision continuation.
- Chat send performs a route preflight check and surfaces direct Models remediation when no enabled ready chat route is available.
- Route remediation does not inject raw daemon 400 route errors into transcript assistant messages.
- Models quickstart deep-link preserves existing chat drafts and seeds validation text only when the composer is empty.
- Non-route chat failures surface deterministic guided retry/remediation with one primary `Fix and Continue` path and direct runtime/setup fallback navigation.
- Chat typed-problem failures (`auth_scope`, `rate_limit_exceeded`) use the shared panel remediation card/actions contract with deterministic disabled/in-flight reason copy.
- Realtime fallback status copy is class-specific for websocket capacity rejection, stale-session heartbeat expiry, and mid-turn disconnect paths (not a single generic fallback string).
- Chat `Actions` menu includes explicit `Retry Realtime Stream` affordance with deterministic in-flight disablement and success/failure copy.
- Last failed prompt can be restored into composer for retry without manual retyping.
- Long-running turns complete without early timeout-driven assistant truncation in healthy transport conditions.
- If chat HTTP receipt transport blips after daemon persisted the turn, chat recovers from `/v1/chat/history` by correlation and avoids false terminal failure copy for that completed turn.
- When realtime is unavailable/disconnected, chat still completes via one-shot `/v1/chat/turn` with explicit fallback messaging.
- Realtime lifecycle completion/error events finalize chat turns deterministically, preserving partial streamed output when final HTTP receipt fails.
- Chat header/composer avoid duplicating resolved provider/model route metadata; canonical route ownership remains in `Effective Workflow Context`.
- Effective workflow-context card default copy is summary-first (`What happened`, `What needs action`, `What next`) and surfaces deterministic next-step/recovery guidance for approval-required, clarification-required, failed-step, and active-in-flight states.
- Effective workflow-context card surfaces response-shaping context (channel/profile/persona source) as scan-first badges when metadata is present.
- Successful turns surface daemon-provided `task_run_correlation` metadata plus unified-turn approval/clarification hints in `Effective Workflow Context > Details`, while default card copy stays workflow-action focused.
- `Effective Workflow Context > Details` includes shaping metadata/count rows (`Shaping Channel`, `Shaping Profile`, `Persona Source`, `Shaping Guardrails`, `Shaping Instructions`) when provided.
- Chat workflow context renders chat explainability context (`tool_catalog`, `policy_decisions`, route reason/explanation signals) with deterministic loading/empty/error/remediation states and explicit action affordances.
- Chat workflow-context card provides direct `Open Related Tasks` and `Open Related Inspect` navigation affordances with context seeding.
