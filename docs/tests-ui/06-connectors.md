# UI Tests: Connectors Panel

Source: [full guide](../tests-ui/full.md)

Regression anchor: run [Auth Scope + Rate Limit + Realtime Hardening (Manual-Only)](./11-auth-rate-limit-realtime-hardening.md) for shared panel-problem remediation checks on connector queries.

## 10) Connectors Panel Checks

1. Select `Connectors`.
2. Confirm connector cards exist and show permission status by default.
3. Verify UI shows one unified `Twilio` card (not separate SMS/Voice cards).
4. Verify `Twilio` card details include canonical connector identity (`twilio`), mapped connector IDs/capabilities, and summary reflects connector state.
5. Verify `iMessage` connector appears as its own connector-scoped card with canonical connector identity (`imessage`) in card details.
6. Confirm each connector card is collapsed by default on first render.
7. For a connector card where daemon diagnostics include a `request_permission` action and permission is missing:
   - Verify `Request Permission` is visible and enabled.
8. For a connector card where daemon diagnostics include a `request_permission` action and permission is granted:
   - Verify `Request Permission` stays visible but disabled.
9. For a connector card where daemon diagnostics do not include a `request_permission` action (for example cloudflared):
   - Verify no fallback `Request Permission` button is rendered and helper copy directs the user to `Open System Settings`/connector remediation actions.
10. Verify `Open System Settings` is visible and enabled on every card.
11. Validate connector remediation target matrix for `Open System Settings`:
   - `Mail` -> `Privacy & Security > Automation`
   - `Calendar` -> `Privacy & Security > Automation`
   - `Browser` -> `Privacy & Security > Automation`
   - `Finder` -> `Privacy & Security > Automation`
   - `Messages` (`Open System Settings`) -> `Privacy & Security > Automation`
   - `Messages` (`Open Full Disk Access`) -> `Privacy & Security > Full Disk Access`
12. Click `Open System Settings` on one connector card, then return to the app and verify the connector permission status re-check runs on app re-activation.
13. On at least one connector with non-granted permission, click `Request Permission`.
14. Verify the action is one-click (no second confirmation modal) and in-card request status messaging updates after completion.
15. For `Messages`, verify request-permission status copy references both Automation and Full Disk Access checks; for all permission-gated connectors, verify macOS permission attribution/prompt references `Personal Agent Daemon` (not the host process used to launch the app).
16. Expand one connector card and verify diagnostics/remediation action buttons are functional when shown (for example `Refresh Connector Status`, `Open Inspect Logs`, `Open System Settings`, and/or `Run Daemon Repair`).
17. In one connector card `Configuration` section:
   - verify guided fields render typed controls and metadata (`Required`, enum picker, bool toggle, secret/write-only/help text) from daemon descriptors or synthesized editable-config metadata.
   - update one guided field value and click `Save Config`.
   - verify deterministic save status text is shown and the card refresh preserves the updated value.
18. Expand `Advanced` (default collapsed), add one raw key/value field that is not represented by guided descriptors, and verify it appears only in the advanced fallback list.
19. Click `Reset Draft` and verify draft values reset to last daemon-backed card values.
20. Click `Run Health Check` and verify inline result rendering includes:
   - status badge (`Healthy` or `Needs Attention`)
   - summary text
   - checked-at timestamp
   - structured detail rows when daemon returns detail payload.
21. Click `Refresh` and verify connector status summaries update.
22. Collapse and expand cards.
22a. In at least one connector card with permission + remediation state metadata, verify only one helper-status line is rendered below action rows (no stacked duplicate status-reason + disabled/unavailable copy blocks).
23. If daemon returns zero connectors, verify a deterministic empty-state view is shown instead of a blank panel.
23a. In the zero-connector state, verify empty-state body does not repeat the same status sentence already shown in the Connectors header subtitle.
24. Make at least one unsaved connector configuration edit without clicking per-card save.
25. Verify Connectors header shows `Unsaved changes` with enabled `Discard All` and `Save All` buttons, and edited card shows `Unsaved draft changes`.
26. Click `Discard All` and verify connector drafts reset to daemon-backed values with deterministic section status copy.
27. Re-create at least one unsaved connector edit, click `Save All`, and verify changed connector drafts persist with deterministic section status copy.
28. Expand at least one connector card, switch to another section, then return to `Connectors` and verify expanded-card state restores for the active workspace.
29. Click `Refresh` and verify expanded/collapsed state remains stable after connector-status reload.
30. Trigger `Reset Context` from `Communications` or `Tasks`, return to `Connectors`, and verify connector cards return to default-collapsed state for the active workspace.
31. Trigger typed panel-problem responses (`auth_scope` and `rate_limit_exceeded`) for connector queries and verify the shared remediation card appears with deterministic actions (`Open Configuration`, `Retry`, `Open Inspect`) and explicit retry disabled reason while refresh is already in flight.

Expected:

- Connector status/detail data is populated from daemon `/v1/connectors/status`.
- Connector cards use canonical IDs (for example `twilio`, `imessage`) with deterministic primary-connector routing for actions/config.
- Guided connector configuration fields render by default from daemon descriptors (`config_field_descriptors`) and fall back to synthesized typed descriptors from editable config metadata when daemon descriptors are absent.
- Connector `Advanced` config disclosure is default-collapsed and retains raw key/value fallback editing only for non-descriptor keys.
- Configuration edits persist via daemon `/v1/connectors/config/upsert` with deterministic save/reset status copy and no stale draft flicker after refresh.
- Connector health checks execute via daemon `/v1/connectors/test` and render structured inline result summaries/details.
- Connector permission/health state reflects daemon metadata (`configuration.permission_state`, `configuration.status_reason`) with deterministic behavior.
- `Request Permission` is shown only when daemon declares a supported request action for that connector and is disabled when permission is already granted or daemon marks action unavailable.
- `Request Permission` routes through daemon API `/v1/connectors/permission/request` and performs daemon-managed native probe flows for supported connectors.
- `Request Permission` executes immediately (no extra confirmation modal before dispatch).
- `Messages` request-permission flow returns dual-permission guidance (Automation + Full Disk Access) with deterministic status copy.
- macOS permission attribution and/or privacy-pane entries for requested connector permissions reference `Personal Agent Daemon` instead of the parent host process.
- `Open System Settings` is always available and routes to connector-relevant privacy panes per matrix (`Mail/Calendar/Browser/Finder -> Automation`, `Messages -> Automation`) and `Messages` also exposes explicit `Open Full Disk Access` remediation for chat database access.
- Returning from System Settings triggers connector permission-state re-check on app activation, with deterministic status messaging.
- Diagnostics/remediation actions are daemon-backed when shown; dead placeholder actions are removed.
- Connector action ordering is deterministic with daemon-recommended actions prioritized and status-reason copy shown when available.
- Unsupported connector remediation actions are hidden; daemon-disabled actions stay visible with deterministic reason copy.
- Connector cards show one canonical helper-status line for permission/remediation state instead of repeating the same readiness reason across multiple stacked text rows.
- Collapse/expand uses native disclosure affordances and defaults to collapsed on first render.
- Connector-card expansion state persists workspace-scoped across section switches/refresh/relaunch and clears under explicit continuity reset.
- On first section load with no cached connector inventory, connectors body renders deterministic skeleton placeholders before transitioning to populated or empty/error states.
- Connectors section exposes deterministic dirty-state affordances (`Unsaved changes`, `Discard All`, `Save All`) and card-level unsaved labels when drafts diverge.
- Connectors typed-problem failures (`auth_scope`, `rate_limit_exceeded`) use the shared panel remediation card/actions contract with deterministic retry disablement messaging during in-flight refresh.
- When connector inventory is empty, the panel shows explicit empty-state copy with status context.
- Connectors empty-state status copy is suppressed when it matches the header subtitle, leaving one canonical status owner.
- Connector empty-state copy includes direct remediation CTAs (`Refresh Connectors` plus `Open Channels`/`Open Configuration` based on setup readiness).
