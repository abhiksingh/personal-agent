# UI Tests: Channels Panel

Source: [full guide](../tests-ui/full.md)

Regression anchor: run [Auth Scope + Rate Limit + Realtime Hardening (Manual-Only)](./11-auth-rate-limit-realtime-hardening.md) for shared panel-problem remediation checks on channel queries.

## 9) Channels Panel Checks

1. Select `Channels`.
2. Confirm cards exist for:
   - `App`
   - `Message`
   - `Voice`
3. Confirm each channel card is collapsed by default on first render.
4. Click `Refresh`.
5. Expand one card and verify daemon-declared diagnostics/remediation actions are functional when present.
6. In the `Message` channel card, if `Open Full Disk Access` (or equivalent `open_system_settings` remediation) is shown, click it and verify System Settings opens to Full Disk Access.
7. Click `Open Inspect Logs` on one card and verify navigation switches to `Inspect`.
8. In the `Message` card, verify mapped implementation/context fields include canonical logical channel IDs and mapped connector rollup details (health/reason/action summary) sourced from current daemon channel-connector bindings.
9. In one channel card `Configuration` section:
   - verify guided fields render typed controls and metadata (`Required`, enum picker, bool toggle, secret/write-only/help text) from daemon descriptors or synthesized editable-config metadata.
   - update one guided field value and click `Save Config`.
   - verify deterministic save status text is shown and the card refresh preserves the updated value.
10. Expand `Advanced` (default collapsed), add one raw key/value field that is not represented by guided descriptors, and verify it appears only in the advanced fallback list.
11. Click `Reset Draft` and verify draft values reset to last daemon-backed card values.
12. Click `Run Health Check` and verify inline result rendering includes:
   - status badge (`Healthy` or `Needs Attention`)
   - summary text
   - checked-at timestamp
   - structured detail rows when daemon returns detail payload.
13. In one channel card `Delivery Policy` section:
   - verify existing daemon policies render with default/non-default visibility.
   - click `Edit` on one policy and verify editor fields load (`endpoint pattern`, `primary channel`, `retry count`, `fallback channels`, `default toggle`).
   - update at least one field and click `Save Policy`; verify confirmation appears, then confirm and verify deterministic save status text plus refreshed policy data.
14. Click `New Policy`, enter values, and click `Save Policy`; verify confirmation appears, then confirm and verify the new or updated policy appears in the list.
15. Click `Reset Policy Draft` and verify draft values return to daemon-backed defaults.
16. In one channel card `Connector Mapping` section, verify mapped connector rows are visible with:
   - connector label + canonical connector id
   - enabled toggle
   - priority reorder controls (up/down)
   - capability summary text (when daemon returns capability metadata)
   - fallback policy label.
17. Toggle one mapping enabled state and move one mapping priority; verify draft-change behavior updates deterministically (`Save Mapping` becomes enabled and status copy reflects draft updates).
18. Click `Save Mapping` and verify deterministic save status text, then click `Refresh` and confirm mapping enabled/priority state persists.
19. In any workspace where `Message` has both `imessage` and `twilio` mappings enabled, change connector priority ordering in `Connector Mapping`, save, and verify the logical `Message` card summary/details track the mapping-priority-selected implementation after refresh.
20. In `Message` and `Voice` cards, verify constraint copy is explicit (`message -> imessage/twilio`, `voice -> twilio`) and capability-validation failures (if triggered) surface daemon error text inline in the mapping status row.
21. Collapse and expand each card.
22. If daemon returns no channel remediation actions for a card, verify the card shows deterministic unavailability copy, surfaces `Open Connectors`, and does not render generic fallback setup buttons.
22a. In the same no-action scenario, verify connector reason/action diagnostics are not repeated as extra stacked helper lines under actions (they should remain owned by the card details surface in `Advanced`).
23. If daemon returns zero channels, verify a deterministic empty-state view is shown instead of a blank panel.
23a. In the zero-channel state, verify empty-state body does not repeat the same status sentence already shown in the Channels header subtitle.
24. Make at least one unsaved channel edit (config, delivery policy draft, or connector mapping) without clicking per-card save.
25. Verify Channels header shows `Unsaved changes` with enabled `Discard All` and `Save All` buttons, and edited card shows `Unsaved draft changes`.
26. Click `Discard All` and verify all channel drafts reset to daemon-backed values with deterministic section status copy.
27. Re-create at least one unsaved edit, click `Save All`, and verify all changed channel drafts persist with deterministic section status copy.
28. Expand at least one channel card, switch to another section, then return to `Channels` and verify expanded-card state restores for the active workspace.
29. Click `Refresh` and verify expanded/collapsed state remains stable after channel-status reload.
30. Trigger `Reset Context` from `Communications` or `Tasks`, return to `Channels`, and verify channel cards return to default-collapsed state for the active workspace.
31. Trigger typed panel-problem responses (`auth_scope` and `rate_limit_exceeded`) for channel queries and verify the shared remediation card appears with deterministic actions (`Open Configuration`, `Retry`, `Open Inspect`) and explicit retry disabled reason while refresh is already in flight.

Expected:

- Card status/detail data is populated from daemon `/v1/channels/status`.
- Channels render logical cards (`App`, `Message`, `Voice`) from canonical daemon channel IDs only.
- Logical channel mapping matrix remains stable across clean-install and upgrade-path payloads: `twilio` remains enabled under both `Message` and `Voice` when mapped.
- Guided channel configuration fields render by default from daemon descriptors (`config_field_descriptors`) and fall back to synthesized typed descriptors from editable config metadata when daemon descriptors are absent.
- Channel `Advanced` config disclosure is default-collapsed and retains raw key/value fallback editing only for non-descriptor keys.
- Configuration edits persist via daemon `/v1/channels/config/upsert` with deterministic save/reset status copy and no stale draft flicker after refresh.
- Channel health checks execute via daemon `/v1/channels/test` and render structured inline result summaries/details.
- Channel delivery-policy edits persist via daemon `/v1/comm/policy/set` and policy inventory refreshes via `/v1/comm/policy/list` with deterministic draft/save/reset behavior.
- Channel delivery-policy save actions require explicit confirmation before daemon mutation dispatch.
- Channel connector mapping inventory/actions are daemon-backed (`/v1/channels/mappings/list` and `/v1/channels/mappings/upsert`) with deterministic enable/disable and priority reorder behavior per logical channel.
- Mapping UX enforces explicit MVP constraints in copy (`app -> builtin.app`, `message -> imessage|twilio`, `voice -> twilio`) and surfaces daemon capability-validation failures without local override drift.
- Each card preserves status/details, shows mapped-connector rollups (health/reasons/actions) from daemon mapping bindings (not capability-name heuristics), and shows daemon-backed diagnostics actions when available.
- Dual-implementation `Message` primary-card selection follows mapping priority so connector-order changes produce deterministic logical-card summaries after refresh.
- No dead placeholder action buttons are shown in channel cards; unsupported actions are hidden and daemon-disabled actions remain visible with reason copy.
- Channel-card action helper copy avoids duplicate connector reason/action lines in the primary action area; connector diagnostics details remain in the advanced details surface.
- Message-channel `open_system_settings` remediation routes to Full Disk Access when daemon emits iMessage ingest-remediation action metadata.
- Collapse/expand uses native disclosure affordances, defaults to collapsed on first render, and remains smooth/deterministic.
- Channel-card expansion state persists workspace-scoped across section switches/refresh/relaunch and clears under explicit continuity reset.
- On first section load with no cached channel inventory, channels body renders deterministic skeleton placeholders before transitioning to populated or empty/error states.
- Channels section exposes deterministic dirty-state affordances (`Unsaved changes`, `Discard All`, `Save All`) and card-level unsaved labels when drafts diverge.
- Channels typed-problem failures (`auth_scope`, `rate_limit_exceeded`) use the shared panel remediation card/actions contract with deterministic retry disablement messaging during in-flight refresh.
- When channel inventory is empty, the panel shows explicit empty-state copy with status context.
- Channels empty-state status copy is suppressed when it matches the header subtitle, leaving one canonical status owner.
- Channel empty-state copy includes direct remediation CTAs (`Refresh Channels` plus `Open Connectors`/`Open Configuration` based on setup readiness).
