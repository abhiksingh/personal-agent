# UI Tests: Models and Models-to-Chat E2E

Source: [full guide](../tests-ui/full.md)

Regression anchor: run [Auth Scope + Rate Limit + Realtime Hardening (Manual-Only)](./11-auth-rate-limit-realtime-hardening.md) for shared panel-problem remediation checks on model/provider queries.

## 12) Models Panel Checks

1. Select `Models`.
2. Confirm provider/model readiness section exists with:
   - quickstart card with progress bar and deterministic `Current Step` action
   - panel order keeps setup first: quickstart + provider readiness controls + provider cards appear before route-policy tuning cards
   - quickstart step list covering `Connect Provider`, `Choose and Enable Model`, `Set Chat Route`, and `Test in Chat`
   - provider cards for `OpenAI`, `Anthropic`, `Google`, and `Ollama`
   - each provider card supports expand/collapse and starts collapsed by default on first render
   - `Refresh Inventory` and `Run Checks` actions
   - route policy editor card with `Task Class`, `Provider`, and `Model` selectors plus `Save Route Policy`
   - per-provider setup controls (`Endpoint`, `Secret Name`, `API Key`, `Save Provider`, `Check`, `Reset Endpoint`)
   - chat model-route summary (or clear fallback status text)
   - provider-local model catalog entries showing model enablement + provider-ready state
   - provider-local routing-policy rows showing task-class -> provider/model mappings (or deterministic empty copy)
2a. In quickstart, verify `Current Step` shows one primary action button (no secondary competing primary buttons in the same card).
2b. Trigger at least one quickstart action (for example `Save + Check <Provider>` or `Enable <Model>`) and verify the related provider/model status updates in-place without leaving `Models`.
2c. After route is ready, trigger quickstart `Open Chat Test` and verify app navigates to `Chat` with a route chip sourced from `Models`.
2d. Verify quickstart seeded draft appears only when composer was empty before navigation.
3. Expand/collapse at least two provider cards.
4. For one API-key provider (for example `OpenAI`):
   - set `Endpoint` to target endpoint.
   - set/update `Secret Name`.
   - enter `API Key` value.
   - click `Save Provider`.
5. Verify provider status copy transitions through deterministic save stages (keychain save, secret-ref registration, provider save) and then refreshes inventory.
6. Click per-card `Check` and verify readiness badge + status copy transitions to either `Healthy` or `Check Failed`.
7. Click `Reset Endpoint` and verify endpoint input resets to provider default, then click `Save Provider` to persist reset endpoint.
8. Verify API key field is write-only (value is not echoed back after save).
9. In one provider card, click `Discover` and verify discover status copy updates plus a `Discovered` subsection appears when models are returned.
10. In discovered rows:
   - click `Add to Catalog` for one model not already in catalog.
   - verify status copy confirms add/upsert and model appears in catalog list.
11. Use manual add:
   - enter a model key in `Add model key`.
   - click `Add Model`.
   - verify status copy confirms add and input clears.
12. In provider model rows, toggle at least one model from `Enabled` to `Disabled` (or vice versa) using per-row action button.
13. Verify per-row in-flight indicator and deterministic status copy (`enabled`/`disabled`) appears.
14. Click `Remove` on one catalog row and verify status copy confirms removal plus row disappears after refresh.
15. Verify chat route summary card refreshes after successful catalog/mutation actions without leaving the Models panel.
16. If chat route is unresolved, verify a `Route Readiness Checklist` card appears with rows for `Assistant Access Token`, `Daemon Reachability`, `Provider Setup`, `Model Catalog`, and `Chat Route`, each showing deterministic `Ready`/`Needs Attention`/`Checking` status badges.
17. In one provider model row, click `Set as Chat Route`; if the model is disabled, verify it is enabled first and then route policy is saved.
18. Verify `Set as Chat Route` updates deterministic status copy and the selected row shows a `Chat Route` badge.
19. Verify chat route summary card reflects the selected provider/model and updated source context without leaving Models.
20. In route policy editor, select `chat`, select a provider/model pair from available catalog options, then click `Save Route Policy`.
21. Verify route-policy confirmation appears with selected task-class/provider/model context; confirm save.
22. Verify route-policy save status copy reports success and the provider-local routing-policy row for `chat` updates immediately.
23. Click `Refresh Inventory` and then `Run Checks`.
24. Confirm a `Route Simulation + Explainability` card is visible with:
   - `Task Class` picker
   - optional `Principal Actor ID` input
   - `Use Active Principal`, `Pick Principal`, and `Clear Principal` helper actions
   - `Simulate Route`, `Explain Route`, and `Reset Output` actions
   - shared `Route Selection` summary (provider/model/source + task class/principal) rendered once when outputs are available
   - `Simulation Result` and `Explainability` result groups.
25. In the route-analysis card, set `Task Class` to one non-default class (for example `automation`) and optionally set a principal actor id, then click `Simulate Route`.
26. Verify simulation status text updates deterministically and `Simulation Result` renders:
   - reason-code list
   - decision trace rows (`step`, `decision`, `reason_code`, optional provider/model/note)
   - fallback-chain rows with rank ordering and selected-candidate marker.
27. Click `Explain Route` for the same input and verify `Explainability` renders:
   - summary text
   - explanation bullet list
   - reason-code list
   - decision trace and fallback-chain context aligned to the explainability response.
27a. Verify route selection context (`provider/model/source`, `task class`, principal) remains in the shared summary and is not duplicated inside both `Simulation Result` and `Explainability` groups.
28. Click `Reset Output` and verify both `Simulation Result` and `Explainability` return to deterministic empty guidance copy.
29. Make at least one unsaved provider setup edit (`Endpoint`, `Secret Name`, and/or write-only `API Key`) without clicking `Save Provider`.
30. Verify Models header shows `Unsaved changes` with enabled `Discard All` and `Save All` buttons, and edited provider card shows `Unsaved setup draft`.
31. Click `Discard All` and verify provider setup drafts reset to source values with deterministic section status copy.
32. Re-create at least one unsaved provider setup edit, click `Save All`, and verify changed provider drafts persist with deterministic section status copy.
33. Trigger typed panel-problem responses (`auth_scope` and `rate_limit_exceeded`) for models/provider queries and verify the shared remediation card appears with deterministic actions (`Open Configuration`, `Retry`, `Open Inspect`) and explicit retry disabled reason while refresh is already in flight.
33a. In a workspace with zero providers, verify provider empty-state body does not repeat the same provider-status sentence already shown in the provider readiness section header.

Expected:

- Provider/model readiness reflects daemon inventory/check/resolve responses and remains readable when auth/config is missing.
- Models quickstart exposes exactly one deterministic current-step action and advances through provider -> model -> route -> chat validation without creating a second setup owner.
- Models keeps setup-before-policy ordering: provider setup/cards are above route-policy tuning so users can finish provider readiness before fine-tuning route controls.
- When provider inventory is empty, Models shows direct remediation CTAs (`Refresh Inventory` and setup-first actions such as `Open Configuration`/`Run Checks`) instead of blank space.
- Provider setup controls persist endpoint + secret reference using daemon mutation APIs.
- Provider-card actions (`Save`, `Check`, `Reset Endpoint`) are all functional and deterministic.
- Provider cards support daemon-backed model discovery plus explicit manual add/remove catalog management controls.
- When chat route is unresolved, Models surfaces a deterministic route-readiness checklist with actionable remediation buttons for unresolved setup blockers.
- Provider model rows expose one-click `Set as Chat Route` actions that can auto-enable disabled models before saving chat route policy.
- Route-policy editor only allows provider/model values that exist in current catalog entries and persists route updates through daemon model-select mutation.
- Route-policy saves require explicit confirmation before daemon mutation dispatch.
- Model rows expose daemon-backed `Enable`/`Disable` actions with deterministic in-flight/success/error status handling.
- Discover/add/remove actions produce deterministic provider-scoped status copy and keep provider-local catalog/discovered state coherent.
- Route-summary state updates after successful model toggle without requiring section re-entry.
- Provider readiness badges clearly differentiate `Setup Required`, `Configured`, `Healthy`, and `Check Failed`.
- Route simulation executes through daemon `/v1/models/route/simulate` with deterministic task-class/principal handling and renders reason-code decision/fallback context in-app.
- Route explainability executes through daemon `/v1/models/route/explain` and renders summary/explanations plus aligned reason-code decision/fallback context.
- Route simulation/explainability section keeps one shared route-selection summary owner instead of repeating provider/model/source blocks in both result groups.
- Quickstart `Open Chat Test` deep-link passes route context into `Chat` and seeds a validation draft only when the composer is empty.
- API key values remain write-only in UI and are never rendered in panel copy.
- Model catalog/policy visibility reflects daemon `models.list` and `models.policy` responses with deterministic empty/error states and provider-local grouping.
- On first section load with no cached provider inventory, models provider area renders deterministic skeleton placeholders before transitioning to populated or empty/error states.
- Models section exposes deterministic dirty-state affordances (`Unsaved changes`, `Discard All`, `Save All`) and provider-level unsaved labels when setup drafts diverge.
- Models typed-problem failures (`auth_scope`, `rate_limit_exceeded`) use the shared panel remediation card/actions contract with deterministic retry disablement messaging during in-flight refresh.
- Runtime banner language remains consistent with other daemon-backed sections.
- Models provider empty-state status copy is suppressed when it matches the provider section status owner.

## 15) Models-to-Chat End-to-End Flow

1. Select `Models`.
2. In one API-key provider card (for example `OpenAI`), enter/update:
   - `Endpoint`
   - `Secret Name`
   - `API Key` (write-only)
3. Click `Save Provider` and verify deterministic success status copy.
4. In the same provider card, ensure at least one target chat model is `Enabled` (toggle if needed).
5. In `Route Policy Editor`, set:
   - `Task Class` = `chat`
   - `Provider` = chosen provider
   - `Model` = enabled target model
6. Click `Save Route Policy` and verify success status copy and updated route summary.
7. Navigate to `Chat`, enter a prompt, and click `Send`.
8. Verify chat turn completes successfully without route-remediation warning state.
9. Verify chat status copy reflects provider/model metadata from successful response.

Expected:

- User can complete provider setup, model enablement, and route selection entirely from `Models`.
- After setup, chat send succeeds without unresolved-route remediation errors.
- `Chat` and `Models` route context remain consistent for the selected provider/model pair.
