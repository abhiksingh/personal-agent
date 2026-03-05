import Foundation
import Testing
@testable import PersonalAgentUIV2

@Test("assistant workspace sections are stable")
func sectionOrderIsStable() {
    #expect(AssistantWorkspaceSection.allCases == [.replayAndAsk, .getStarted, .connectorsAndModels])
}

@MainActor
@Test("store defaults to Replay & Ask")
func defaultSectionIsReplayAndAsk() {
    let store = makeStore(withToken: false)
    #expect(store.selectedSection == .replayAndAsk)
    #expect(store.statusFilter == .needsApproval)
}

@MainActor
@Test("inline approval submits daemon decision and reconciles replay event")
func inlineApprovalSubmitsDaemonDecisionAndReconcilesReplayEvent() async {
    let event = makeReplayApprovalEvent(approvalRequestID: "apr-approve", risk: .low)
    let session = makeMockSession { request in
        switch request.url?.path ?? "" {
        case "/v1/approvals/decision":
            return (200, jsonData(["approval_id": "apr-approve", "decision": "approve", "accepted": true]))
        case "/v1/approvals/inbox":
            return (
                200,
                jsonData([
                    "workspace_id": "ws1",
                    "approvals": [
                        [
                            "approval_request_id": "apr-approve",
                            "workspace_id": "ws1",
                            "state": "approved",
                            "decision": "approve",
                            "risk_level": "low",
                            "risk_rationale": "Approved by user.",
                            "requested_phrase": "Send status update to team",
                            "requested_at": "2026-03-05T03:00:00Z"
                        ]
                    ]
                ])
            )
        case "/v1/tasks/runs/list":
            return (200, jsonData(["workspace_id": "ws1", "items": []]))
        case "/v1/chat/history":
            return (200, jsonData(["workspace_id": "ws1", "items": [], "has_more": false]))
        default:
            return (404, jsonData(["error": ["code": "not_found", "message": "no fixture"]]))
        }
    }

    let store = makeStore(
        replayEvents: [event],
        daemonClient: V2DaemonAPIClient(session: session),
        withToken: true
    )
    store.statusFilter = .all
    store.selectEvent(event.id)
    store.approveSelectedEvent()

    #expect(store.mutationLifecycle(for: .replayApprove).phase == .inFlight)

    await waitForMutationCompletion(store, actionID: .replayApprove)

    #expect(store.mutationLifecycle(for: .replayApprove).phase == .succeeded)
    let updated = store.replayEvents.first(where: { $0.replayKey == "approval:apr-approve" })
    #expect(updated?.status == .completed)
    #expect(updated?.actionSummary.localizedCaseInsensitiveContains("approval") == true)
    #expect(store.lastFeedback?.contains("Replay state reconciled.") == true)
}

@MainActor
@Test("sending ask uses daemon chat turn and reconciles replay item")
func sendAskUsesDaemonChatTurnAndReconcilesReplayItem() async {
    let session = makeMockSession { request in
        switch request.url?.path ?? "" {
        case "/v1/chat/turn":
            return (
                200,
                jsonData([
                    "workspace_id": "ws1",
                    "task_class": "chat",
                    "provider": "built_in",
                    "model_key": "personalagent_default",
                    "correlation_id": "corr-ask-1",
                    "channel": [
                        "channel_id": "app"
                    ],
                    "items": [
                        [
                            "item_id": "item-user-1",
                            "type": "user_message",
                            "role": "user",
                            "status": "completed",
                            "content": "Why was approval required on the last run?"
                        ],
                        [
                            "item_id": "item-assistant-1",
                            "type": "assistant_message",
                            "role": "assistant",
                            "status": "completed",
                            "content": "Approval was required because the action impacted a shared resource."
                        ]
                    ],
                    "task_run_correlation": [
                        "available": false,
                        "source": "none"
                    ]
                ])
            )
        case "/v1/approvals/inbox":
            return (200, jsonData(["workspace_id": "ws1", "approvals": []]))
        case "/v1/tasks/runs/list":
            return (200, jsonData(["workspace_id": "ws1", "items": []]))
        case "/v1/chat/history":
            return (
                200,
                jsonData([
                    "workspace_id": "ws1",
                    "items": [
                        [
                            "record_id": "hist-ask-1",
                            "turn_id": "turn-ask-1",
                            "correlation_id": "corr-ask-1",
                            "channel_id": "app",
                            "item_index": 0,
                            "item": [
                                "item_id": "item-user-1",
                                "type": "user_message",
                                "role": "user",
                                "status": "completed",
                                "content": "Why was approval required on the last run?"
                            ],
                            "task_run_reference": [
                                "available": false,
                                "source": "none"
                            ],
                            "created_at": "2026-03-05T03:00:00Z"
                        ],
                        [
                            "record_id": "hist-ask-2",
                            "turn_id": "turn-ask-1",
                            "correlation_id": "corr-ask-1",
                            "channel_id": "app",
                            "item_index": 1,
                            "item": [
                                "item_id": "item-assistant-1",
                                "type": "assistant_message",
                                "role": "assistant",
                                "status": "completed",
                                "content": "Approval was required because the action impacted a shared resource."
                            ],
                            "task_run_reference": [
                                "available": false,
                                "source": "none"
                            ],
                            "created_at": "2026-03-05T03:00:01Z"
                        ]
                    ],
                    "has_more": false
                ])
            )
        default:
            return (404, jsonData(["error": ["code": "not_found", "message": "no fixture"]]))
        }
    }

    let store = makeStore(
        replayEvents: [],
        daemonClient: V2DaemonAPIClient(session: session),
        withToken: true
    )
    store.statusFilter = .all
    store.askDraft = "Why was approval required on the last run?"
    store.sendAsk()

    #expect(store.askDraft.isEmpty)
    #expect(store.mutationLifecycle(for: .askSend).phase == .inFlight)

    await waitForMutationCompletion(store, actionID: .askSend)

    #expect(store.mutationLifecycle(for: .askSend).phase == .succeeded)
    #expect(store.replayEvents.first?.source == .app)
    #expect(store.replayEvents.first?.status == .completed)
    #expect(store.replayEvents.first?.instruction.localizedCaseInsensitiveContains("approval required") == true)
    #expect(store.replayEvents.first?.daemonLocator?.correlationID == "corr-ask-1")
    #expect(store.statusFilter == .running)
}

@MainActor
@Test("ask send failure restores draft and marks optimistic replay row failed")
func askSendFailureRestoresDraftAndMarksOptimisticReplayRowFailed() async {
    let session = makeMockSession { request in
        switch request.url?.path ?? "" {
        case "/v1/chat/turn":
            return (
                500,
                jsonData([
                    "error": [
                        "code": "internal_error",
                        "message": "chat turn failed"
                    ]
                ])
            )
        default:
            return (404, jsonData(["error": ["code": "not_found", "message": "no fixture"]]))
        }
    }

    let store = makeStore(
        replayEvents: [],
        daemonClient: V2DaemonAPIClient(session: session),
        withToken: true
    )
    store.askDraft = "What guardrails did you use?"
    store.sendAsk()

    #expect(store.replayEvents.first?.status == .running)
    await waitForMutationCompletion(store, actionID: .askSend)

    #expect(store.mutationLifecycle(for: .askSend).phase == .failed)
    #expect(store.askDraft == "What guardrails did you use?")
    #expect(store.replayEvents.first?.status == .failed)
    #expect(store.replayEvents.first?.failureRecoveryHint?.localizedCaseInsensitiveContains("Retry") == true)
}

@MainActor
@Test("status and search filters narrow replay list")
func filtersNarrowReplayList() {
    let store = makeStore()

    store.statusFilter = .failed
    #expect(!store.filteredEvents.isEmpty)
    #expect(store.filteredEvents.allSatisfy({ $0.status == .failed }))

    store.statusFilter = .all
    store.searchQuery = "daily plan"
    #expect(store.filteredEvents.count == 1)
    #expect(store.filteredEvents.first?.instruction.localizedCaseInsensitiveContains("daily plan") == true)
}

@MainActor
@Test("clear filters returns replay to needs-approval baseline")
func clearFiltersReturnsToNeedsApprovalBaseline() {
    let store = makeStore()

    store.statusFilter = .failed
    store.searchQuery = "travel"
    store.toggleSource(.email)
    store.clearFilters()

    #expect(store.statusFilter == .needsApproval)
    #expect(store.searchQuery.isEmpty)
    #expect(store.selectedSources.isEmpty)
}

@MainActor
@Test("needs approval filter excludes failed events")
func needsApprovalFilterExcludesFailedEvents() {
    let store = makeStore()

    store.statusFilter = .needsApproval

    #expect(!store.filteredEvents.isEmpty)
    #expect(store.filteredEvents.allSatisfy({ $0.status == .awaitingApproval }))
    #expect(store.filteredEvents.contains(where: { $0.status == .failed }) == false)
}

@MainActor
@Test("cannot disable the last enabled model")
func cannotDisableLastEnabledModel() {
    let onlyModel = ModelOption(providerName: "Built-In", modelName: "Default", enabled: true)
    let store = makeStore(
        connectors: AppShellV2Store.defaultConnectors,
        models: [onlyModel],
        replayEvents: AppShellV2Store.defaultReplayEvents,
        withToken: true
    )

    #expect(store.disableModelReason(for: onlyModel.id) == "At least one model must stay enabled.")

    store.toggleModelEnabled(onlyModel.id)

    #expect(store.models.first?.enabled == true)
    #expect(store.lastFeedback == "At least one model must stay enabled.")
}

@MainActor
@Test("model inventory refresh projects live catalog and route resolution")
func modelInventoryRefreshProjectsLiveCatalogAndRoute() async {
    let session = makeMockSession { request in
        switch request.url?.path ?? "" {
        case "/v1/models/list":
            return (
                200,
                jsonData([
                    "workspace_id": "ws1",
                    "models": [
                        [
                            "workspace_id": "ws1",
                            "provider": "built_in",
                            "model_key": "personalagent_default",
                            "enabled": true,
                            "provider_ready": true,
                            "provider_endpoint": "local"
                        ],
                        [
                            "workspace_id": "ws1",
                            "provider": "openai",
                            "model_key": "gpt-4.1",
                            "enabled": true,
                            "provider_ready": true,
                            "provider_endpoint": "https://api.openai.com/v1"
                        ]
                    ]
                ])
            )
        case "/v1/models/resolve":
            return (
                200,
                jsonData([
                    "workspace_id": "ws1",
                    "task_class": "chat",
                    "provider": "openai",
                    "model_key": "gpt-4.1",
                    "source": "policy",
                    "notes": "Policy-selected route."
                ])
            )
        default:
            return (404, jsonData(["error": ["code": "not_found", "message": "no fixture"]]))
        }
    }

    let store = makeStore(
        models: [],
        daemonClient: V2DaemonAPIClient(session: session),
        withToken: true
    )

    await store.refreshModelsInventory(force: true)

    #expect(store.hasLoadedModelInventory == true)
    #expect(store.models.count == 2)
    #expect(store.modelRouteResolution?.provider == "openai")
    #expect(store.modelRouteResolution?.modelKey == "gpt-4.1")
    #expect(store.activeModelID == "openai::gpt-4.1")
    #expect(store.modelRouteStatusMessage?.contains("OpenAI / gpt-4.1") == true)
}

@MainActor
@Test("model toggle uses daemon enable/disable and refreshes model inventory")
func modelToggleUsesDaemonEnableDisableAndRefreshesInventory() async {
    struct ModelToggleBody: Decodable {
        let provider: String
        let modelKey: String

        enum CodingKeys: String, CodingKey {
            case provider
            case modelKey = "model_key"
        }
    }

    let capturedProvider = LockedBox<String?>(nil)
    let capturedModelKey = LockedBox<String?>(nil)

    let seedModel = ModelOption(
        providerID: "anthropic",
        providerName: "Anthropic",
        modelKey: "claude-sonnet",
        enabled: false,
        providerReady: true,
        providerEndpoint: "https://api.anthropic.com/v1"
    )

    let session = makeMockSession { request in
        switch request.url?.path ?? "" {
        case "/v1/models/enable":
            if let body = requestBodyData(request),
               let payload = try? JSONDecoder().decode(ModelToggleBody.self, from: body) {
                capturedProvider.withValue { $0 = payload.provider }
                capturedModelKey.withValue { $0 = payload.modelKey }
            }
            return (
                200,
                jsonData([
                    "workspace_id": "ws1",
                    "provider": "anthropic",
                    "model_key": "claude-sonnet",
                    "enabled": true,
                    "updated_at": "2026-03-05T06:00:00Z"
                ])
            )
        case "/v1/models/list":
            return (
                200,
                jsonData([
                    "workspace_id": "ws1",
                    "models": [
                        [
                            "workspace_id": "ws1",
                            "provider": "anthropic",
                            "model_key": "claude-sonnet",
                            "enabled": true,
                            "provider_ready": true,
                            "provider_endpoint": "https://api.anthropic.com/v1"
                        ]
                    ]
                ])
            )
        case "/v1/models/resolve":
            return (
                200,
                jsonData([
                    "workspace_id": "ws1",
                    "task_class": "chat",
                    "provider": "anthropic",
                    "model_key": "claude-sonnet",
                    "source": "policy"
                ])
            )
        default:
            return (404, jsonData(["error": ["code": "not_found", "message": "no fixture"]]))
        }
    }

    let store = makeStore(
        models: [seedModel],
        daemonClient: V2DaemonAPIClient(session: session),
        withToken: true
    )

    store.toggleModelEnabled(seedModel.id)
    await waitForMutationCompletion(store, actionID: .modelToggle)
    await store.refreshModelsInventory(force: true)

    #expect(store.mutationLifecycle(for: .modelToggle).phase == .succeeded)
    #expect(store.models.first?.enabled == true)
    #expect(capturedProvider.snapshot() == "anthropic")
    #expect(capturedModelKey.snapshot() == "claude-sonnet")
}

@MainActor
@Test("setting primary model uses daemon select and reconciles route state")
func setPrimaryModelUsesDaemonSelectAndReconcilesRouteState() async {
    struct ModelSelectBody: Decodable {
        let taskClass: String
        let provider: String
        let modelKey: String

        enum CodingKeys: String, CodingKey {
            case taskClass = "task_class"
            case provider
            case modelKey = "model_key"
        }
    }

    let capturedTaskClass = LockedBox<String?>(nil)
    let seedModels: [ModelOption] = [
        ModelOption(providerID: "openai", providerName: "OpenAI", modelKey: "gpt-4.1", enabled: true, providerReady: true),
        ModelOption(providerID: "anthropic", providerName: "Anthropic", modelKey: "claude-sonnet", enabled: true, providerReady: true)
    ]
    let targetModelID = "anthropic::claude-sonnet"

    let session = makeMockSession { request in
        switch request.url?.path ?? "" {
        case "/v1/models/select":
            if let body = requestBodyData(request),
               let payload = try? JSONDecoder().decode(ModelSelectBody.self, from: body) {
                capturedTaskClass.withValue { $0 = payload.taskClass }
            }
            return (
                200,
                jsonData([
                    "workspace_id": "ws1",
                    "task_class": "chat",
                    "provider": "anthropic",
                    "model_key": "claude-sonnet"
                ])
            )
        case "/v1/models/list":
            return (
                200,
                jsonData([
                    "workspace_id": "ws1",
                    "models": [
                        [
                            "workspace_id": "ws1",
                            "provider": "openai",
                            "model_key": "gpt-4.1",
                            "enabled": true,
                            "provider_ready": true,
                            "provider_endpoint": "https://api.openai.com/v1"
                        ],
                        [
                            "workspace_id": "ws1",
                            "provider": "anthropic",
                            "model_key": "claude-sonnet",
                            "enabled": true,
                            "provider_ready": true,
                            "provider_endpoint": "https://api.anthropic.com/v1"
                        ]
                    ]
                ])
            )
        case "/v1/models/resolve":
            return (
                200,
                jsonData([
                    "workspace_id": "ws1",
                    "task_class": "chat",
                    "provider": "anthropic",
                    "model_key": "claude-sonnet",
                    "source": "policy"
                ])
            )
        default:
            return (404, jsonData(["error": ["code": "not_found", "message": "no fixture"]]))
        }
    }

    let store = makeStore(
        models: seedModels,
        daemonClient: V2DaemonAPIClient(session: session),
        withToken: true
    )
    store.modelRouteSimulation = decodeJSON([
        "workspace_id": "ws1",
        "task_class": "chat",
        "selected_provider": "openai",
        "selected_model_key": "gpt-4.1",
        "selected_source": "policy",
        "reason_codes": [],
        "decisions": [],
        "fallback_chain": []
    ])
    store.modelRouteExplainability = decodeJSON([
        "workspace_id": "ws1",
        "task_class": "chat",
        "selected_provider": "openai",
        "selected_model_key": "gpt-4.1",
        "selected_source": "policy",
        "summary": "Prior route summary.",
        "explanations": [],
        "reason_codes": [],
        "decisions": [],
        "fallback_chain": []
    ])

    store.setActiveModel(targetModelID)
    await waitForMutationCompletion(store, actionID: .modelSetPrimary)
    await store.refreshModelsInventory(force: true)

    #expect(store.mutationLifecycle(for: .modelSetPrimary).phase == .succeeded)
    #expect(capturedTaskClass.snapshot() == "chat")
    #expect(store.activeModelID == targetModelID)
    #expect(store.modelRouteResolution?.provider == "anthropic")
    #expect(store.modelRouteResolution?.modelKey == "claude-sonnet")
    #expect(store.modelRouteSimulation == nil)
    #expect(store.modelRouteExplainability == nil)
}

@MainActor
@Test("route simulation and explainability load daemon-backed model routing evidence")
func routeSimulationAndExplainabilityLoadDaemonEvidence() async {
    let session = makeMockSession { request in
        switch request.url?.path ?? "" {
        case "/v1/models/route/simulate":
            return (
                200,
                jsonData([
                    "workspace_id": "ws1",
                    "task_class": "chat",
                    "selected_provider": "openai",
                    "selected_model_key": "gpt-4.1",
                    "selected_source": "policy",
                    "notes": "Routed by policy.",
                    "reason_codes": ["policy_match"],
                    "decisions": [
                        [
                            "step": "policy",
                            "decision": "selected",
                            "reason_code": "policy_match",
                            "provider": "openai",
                            "model_key": "gpt-4.1"
                        ]
                    ],
                    "fallback_chain": []
                ])
            )
        case "/v1/models/route/explain":
            return (
                200,
                jsonData([
                    "workspace_id": "ws1",
                    "task_class": "chat",
                    "selected_provider": "openai",
                    "selected_model_key": "gpt-4.1",
                    "selected_source": "policy",
                    "summary": "Route selected by explicit workspace policy.",
                    "explanations": ["Workspace policy pinned OpenAI for chat."],
                    "reason_codes": ["policy_match"],
                    "decisions": [
                        [
                            "step": "policy",
                            "decision": "selected",
                            "reason_code": "policy_match",
                            "provider": "openai",
                            "model_key": "gpt-4.1"
                        ]
                    ],
                    "fallback_chain": [
                        [
                            "rank": 1,
                            "provider": "openai",
                            "model_key": "gpt-4.1",
                            "selected": true,
                            "reason_code": "policy_match"
                        ]
                    ]
                ])
            )
        default:
            return (404, jsonData(["error": ["code": "not_found", "message": "no fixture"]]))
        }
    }

    let store = makeStore(
        daemonClient: V2DaemonAPIClient(session: session),
        withToken: true
    )

    store.simulateModelRoute()
    await waitForMutationCompletion(store, actionID: .modelRouteSimulate)
    #expect(store.mutationLifecycle(for: .modelRouteSimulate).phase == .succeeded)
    #expect(store.modelRouteSimulation?.selectedProvider == "openai")
    #expect(store.modelRouteSimulationStatusMessage?.contains("Simulated chat route") == true)

    store.explainModelRoute()
    await waitForMutationCompletion(store, actionID: .modelRouteExplain)
    #expect(store.mutationLifecycle(for: .modelRouteExplain).phase == .succeeded)
    #expect(store.modelRouteExplainability?.summary == "Route selected by explicit workspace policy.")
    #expect(store.modelRouteExplainStatusMessage == "Route selected by explicit workspace policy.")
}

@MainActor
@Test("high-risk replay event requires manual approval flow")
func highRiskReplayEventRequiresManualApprovalFlow() {
    let store = makeStore()
    guard let highRiskEvent = store.replayEvents.first(where: { $0.status == .awaitingApproval && $0.risk == .high }) else {
        Issue.record("Expected a high-risk approval event in seed data")
        return
    }

    #expect(highRiskEvent.canInlineApprove == false)
    #expect(highRiskEvent.inlineApprovalDisabledReason?.contains("High-impact request") == true)
}

@MainActor
@Test("rejecting approval submits daemon decision and reconciles replay event")
func rejectingApprovalSubmitsDaemonDecisionAndReconcilesReplayEvent() async {
    let event = makeReplayApprovalEvent(approvalRequestID: "apr-reject", risk: .low)
    let session = makeMockSession { request in
        switch request.url?.path ?? "" {
        case "/v1/approvals/decision":
            return (200, jsonData(["approval_id": "apr-reject", "decision": "reject", "accepted": true]))
        case "/v1/approvals/inbox":
            return (
                200,
                jsonData([
                    "workspace_id": "ws1",
                    "approvals": [
                        [
                            "approval_request_id": "apr-reject",
                            "workspace_id": "ws1",
                            "state": "rejected",
                            "decision": "reject",
                            "risk_level": "low",
                            "risk_rationale": "Rejected by user.",
                            "requested_phrase": "Send status update to team",
                            "requested_at": "2026-03-05T03:00:00Z"
                        ]
                    ]
                ])
            )
        case "/v1/tasks/runs/list":
            return (200, jsonData(["workspace_id": "ws1", "items": []]))
        case "/v1/chat/history":
            return (200, jsonData(["workspace_id": "ws1", "items": [], "has_more": false]))
        default:
            return (404, jsonData(["error": ["code": "not_found", "message": "no fixture"]]))
        }
    }

    let store = makeStore(
        replayEvents: [event],
        daemonClient: V2DaemonAPIClient(session: session),
        withToken: true
    )
    store.statusFilter = .all
    store.selectEvent(event.id)
    store.rejectSelectedEvent()

    #expect(store.mutationLifecycle(for: .replayReject).phase == .inFlight)
    await waitForMutationCompletion(store, actionID: .replayReject)
    #expect(store.mutationLifecycle(for: .replayReject).phase == .succeeded)

    let updated = store.replayEvents.first(where: { $0.replayKey == "approval:apr-reject" })
    #expect(updated?.status == .failed)
    #expect(store.lastFeedback?.contains("Replay state reconciled.") == true)
}

@MainActor
@Test("email source drill-in metadata is present")
func sourceDrillInMetadataExists() {
    let store = makeStore()
    guard let emailEvent = store.replayEvents.first(where: { $0.source == .email }) else {
        Issue.record("Expected an email replay event in seed data")
        return
    }

    let fields = emailEvent.sourceContext.fields
    #expect(fields.contains(where: { $0.label == "Sender" }))
    #expect(fields.contains(where: { $0.label == "Mailbox" }))
}

@Test("typed daemon problem mapping for missing auth is actionable")
func daemonProblemMappingMissingAuthIsActionable() {
    let mapped = V2DaemonProblemMapper.map(error: V2DaemonAPIError.missingAuthToken, context: .setup)
    #expect(mapped.kind == .missingAuth)
    #expect(mapped.actions.first?.actionID == .openGetStarted)
}

@MainActor
@Test("probe daemon invalid URL maps validation problem state")
func probeDaemonInvalidURLMapsValidationState() async {
    let store = makeStore()
    await store.probeDaemon(baseURLString: "://bad-url", authToken: "token")

    let problem = store.panelProblem(for: .setup)
    #expect(problem?.kind == .validation)
    #expect(problem?.actions.contains(where: { $0.actionID == .openGetStarted }) == true)
}

@MainActor
@Test("session config store restores startup configuration and token state")
func sessionConfigStoreRestoresStartupConfiguration() {
    let suite = "personalagent.ui.v2.tests.\(UUID().uuidString)"
    guard let defaults = UserDefaults(suiteName: suite) else {
        Issue.record("Expected suite-scoped defaults")
        return
    }
    defaults.removePersistentDomain(forName: suite)
    let secrets = V2InMemorySecretStore()

    let initial = V2SessionConfigStore(
        userDefaults: defaults,
        secretStore: secrets
    )
    initial.daemonBaseURL = "https://example.test:7443"
    initial.workspaceID = "ws-trust"
    initial.principalActorID = "principal.alex"
    initial.informationDensityMode = .advanced
    initial.persistSelectedSection(.connectorsAndModels)
    try? initial.saveAccessToken("test-token")

    let restored = V2SessionConfigStore(
        userDefaults: defaults,
        secretStore: secrets
    )

    #expect(restored.daemonBaseURL == "https://example.test:7443")
    #expect(restored.workspaceID == "ws-trust")
    #expect(restored.principalActorID == "principal.alex")
    #expect(restored.informationDensityMode == .advanced)
    #expect(restored.selectedSection == .connectorsAndModels)
    #expect(restored.hasStoredAccessToken == true)
    #expect(restored.readiness.isReadyForDaemonMutations == true)
}

@MainActor
@Test("high-impact actions are blocked when token is missing")
func highImpactActionsBlockedWithoutToken() {
    let store = makeStore(withToken: false)
    guard let pending = store.replayEvents.first(where: { $0.status == .awaitingApproval }) else {
        Issue.record("Expected pending approval event")
        return
    }

    store.selectEvent(pending.id)
    store.approveSelectedEvent()

    let unchanged = store.replayEvents.first(where: { $0.id == pending.id })
    #expect(unchanged?.status == .awaitingApproval)
    #expect(store.selectedSection == .getStarted)
    #expect(store.lastFeedback?.contains("Save an Assistant Access Token") == true)
}

@MainActor
@Test("get started panel state is degraded when token is missing")
func getStartedPanelStateDegradedWithoutToken() {
    let store = makeStore(withToken: false)

    let panelState = store.panelLifecycleState(for: .getStarted)
    #expect(panelState.kind == .degraded)
    #expect(panelState.summary.contains("Save an Assistant Access Token"))
}

@MainActor
@Test("replay panel state becomes empty when filters remove all rows")
func replayPanelStateEmptyWhenFilteredOut() {
    let store = makeStore()

    store.searchQuery = "no-match-string"

    let panelState = store.panelLifecycleState(for: .replayAndAsk)
    #expect(panelState.kind == .empty)
    #expect(panelState.actions.contains(where: { $0.actionID == .clearReplayFilters }))
}

@MainActor
@Test("replay realtime event classifier flags workflow lifecycle updates for refresh")
func replayRealtimeEventClassifierFlagsWorkflowLifecycleUpdatesForRefresh() {
    let store = makeStore()

    let relevantEvent = V2DaemonRealtimeEventEnvelope(
        eventID: "evt-replay-1",
        sequence: 1,
        eventType: "approval_recorded",
        occurredAt: "2026-03-05T03:00:00Z",
        payload: V2DaemonRealtimeEventPayload(
            approvalRequestID: "apr-1",
            status: "completed"
        )
    )
    let heartbeatEvent = V2DaemonRealtimeEventEnvelope(
        eventID: "evt-replay-2",
        sequence: 2,
        eventType: "heartbeat",
        occurredAt: "2026-03-05T03:00:01Z",
        payload: V2DaemonRealtimeEventPayload()
    )

    #expect(store.replayRealtimeEventRequiresRefresh(relevantEvent))
    #expect(store.replayRealtimeEventRequiresRefresh(heartbeatEvent) == false)
}

@MainActor
@Test("retry realtime panel action routes to setup when session readiness is incomplete")
func retryRealtimePanelActionRoutesToSetupWhenSessionReadinessIsIncomplete() async {
    let store = makeStore(withToken: false)

    store.performPanelStateAction(.retryRealtime, workflow: .replayAndAsk)
    try? await Task.sleep(nanoseconds: 75_000_000)

    #expect(store.selectedSection == .getStarted)
    #expect(store.lastFeedback?.contains("Assistant Access Token") == true)
}

@MainActor
@Test("replay approve mutation lifecycle reports success")
func replayApproveMutationLifecycleSuccess() async {
    let event = makeReplayApprovalEvent(approvalRequestID: "apr-success", risk: .low)
    let session = makeMockSession { request in
        switch request.url?.path ?? "" {
        case "/v1/approvals/decision":
            return (200, jsonData(["approval_id": "apr-success", "decision": "approve", "accepted": true]))
        case "/v1/approvals/inbox":
            return (200, jsonData(["workspace_id": "ws1", "approvals": []]))
        case "/v1/tasks/runs/list":
            return (200, jsonData(["workspace_id": "ws1", "items": []]))
        case "/v1/chat/history":
            return (200, jsonData(["workspace_id": "ws1", "items": [], "has_more": false]))
        default:
            return (404, jsonData(["error": ["code": "not_found", "message": "no fixture"]]))
        }
    }

    let store = makeStore(
        replayEvents: [event],
        daemonClient: V2DaemonAPIClient(session: session),
        withToken: true
    )
    store.statusFilter = .all
    store.selectEvent(event.id)
    store.approveSelectedEvent()

    await waitForMutationCompletion(store, actionID: .replayApprove)
    #expect(store.mutationLifecycle(for: .replayApprove).phase == .succeeded)
}

@MainActor
@Test("connector mutation lifecycle exposes disabled reason when setup is incomplete")
func connectorMutationLifecycleDisabledWhenSetupIncomplete() {
    let store = makeStore(withToken: false)

    let lifecycle = store.mutationLifecycle(for: .connectorToggle)
    #expect(lifecycle.phase == .disabled)
    #expect(lifecycle.message?.contains("Save an Assistant Access Token") == true)
}

@MainActor
@Test("connector inventory refresh maps daemon status cards into live connector state")
func connectorInventoryRefreshMapsDaemonStatusCards() async {
    let session = makeMockSession { request in
        switch request.url?.path ?? "" {
        case "/v1/connectors/status":
            return (
                200,
                jsonData([
                    "workspace_id": "ws1",
                    "connectors": [
                        [
                            "connector_id": "telegram",
                            "plugin_id": "telegram",
                            "display_name": "Telegram",
                            "enabled": false,
                            "configured": false,
                            "status": "permission_missing",
                            "summary": "Permission is required.",
                            "action_readiness": "blocked",
                            "action_blockers": [
                                [
                                    "code": "permission_missing",
                                    "message": "System permission is required before this connector can run."
                                ]
                            ],
                            "remediation_actions": [
                                [
                                    "identifier": "request_permission",
                                    "label": "Request Permission",
                                    "intent": "request_permission",
                                    "enabled": true
                                ]
                            ],
                            "configuration": [
                                "enabled": false,
                                "permission_state": "missing",
                                "mapped_connector_ids": ["telegram"],
                                "poll_interval_seconds": 30
                            ]
                        ]
                    ]
                ])
            )
        default:
            return (404, jsonData(["error": ["code": "not_found", "message": "no fixture"]]))
        }
    }

    let store = makeStore(
        connectors: [],
        daemonClient: V2DaemonAPIClient(session: session),
        withToken: true
    )

    await store.refreshConnectorsInventory(force: true)

    #expect(store.hasLoadedConnectorInventory)
    guard let telegram = store.connectors.first(where: { $0.id == "telegram" }) else {
        Issue.record("Expected telegram connector to be projected from daemon status.")
        return
    }

    #expect(telegram.status == .notConnected)
    #expect(telegram.summary == "Permission is required.")
    #expect(telegram.configurationDraft["enabled"] == "false")
    #expect(telegram.configurationDraft["poll_interval_seconds"] == "30")
}

@MainActor
@Test("connector toggle submits daemon config upsert and reconciles inventory")
func connectorToggleSubmitsConfigUpsertAndReconcilesInventory() async {
    struct ConnectorUpsertBody: Decodable {
        let configuration: [String: V2DaemonJSONValue]
    }

    let seedConnector = ConnectorState(
        id: "telegram",
        name: "Telegram",
        status: .notConnected,
        summary: "Disconnected.",
        enabled: false,
        configured: true,
        actionReadiness: "ready",
        configurationBaseline: ["enabled": "false"]
    )

    let capturedEnabledValue = LockedBox<Bool?>(nil)
    let session = makeMockSession { request in
        switch request.url?.path ?? "" {
        case "/v1/connectors/config/upsert":
            if let body = requestBodyData(request),
               let payload = try? JSONDecoder().decode(ConnectorUpsertBody.self, from: body),
               let enabled = payload.configuration["enabled"]?.boolValue {
                capturedEnabledValue.withValue { stored in
                    stored = enabled
                }
            }
            return (
                200,
                jsonData([
                    "workspace_id": "ws1",
                    "connector_id": "telegram",
                    "configuration": ["enabled": true],
                    "updated_at": "2026-03-05T04:00:00Z"
                ])
            )
        case "/v1/connectors/status":
            return (
                200,
                jsonData([
                    "workspace_id": "ws1",
                    "connectors": [
                        [
                            "connector_id": "telegram",
                            "plugin_id": "telegram",
                            "display_name": "Telegram",
                            "enabled": true,
                            "configured": true,
                            "status": "connected",
                            "summary": "Connected.",
                            "action_readiness": "ready",
                            "action_blockers": [],
                            "remediation_actions": [],
                            "configuration": ["enabled": true]
                        ]
                    ]
                ])
            )
        default:
            return (404, jsonData(["error": ["code": "not_found", "message": "no fixture"]]))
        }
    }

    let store = makeStore(
        connectors: [seedConnector],
        daemonClient: V2DaemonAPIClient(session: session),
        withToken: true
    )

    store.toggleConnector("telegram")
    await waitForMutationCompletion(store, actionID: .connectorToggle)
    await store.refreshConnectorsInventory(force: true)

    #expect(store.mutationLifecycle(for: .connectorToggle).phase == .succeeded)
    #expect(store.connectors.first(where: { $0.id == "telegram" })?.enabled == true)
    #expect(capturedEnabledValue.snapshot() == true)
}

@MainActor
@Test("connector config save sends typed values and clears draft changes")
func connectorConfigSaveSendsTypedValuesAndClearsDraftChanges() async {
    struct ConnectorUpsertBody: Decodable {
        let configuration: [String: V2DaemonJSONValue]
    }

    let seedConnector = ConnectorState(
        id: "email",
        name: "Email",
        status: .connected,
        summary: "Connected.",
        enabled: true,
        configured: true,
        actionReadiness: "ready",
        configurationBaseline: [
            "enabled": "true",
            "timeout_seconds": "10",
            "endpoint": "https://old.example"
        ]
    )

    let capturedTimeoutValue = LockedBox<Double?>(nil)
    let capturedEnabledValue = LockedBox<Bool?>(nil)
    let session = makeMockSession { request in
        switch request.url?.path ?? "" {
        case "/v1/connectors/config/upsert":
            if let body = requestBodyData(request),
               let payload = try? JSONDecoder().decode(ConnectorUpsertBody.self, from: body) {
                capturedTimeoutValue.withValue { value in
                    if case .number(let timeout)? = payload.configuration["timeout_seconds"] {
                        value = timeout
                    } else {
                        value = nil
                    }
                }
                capturedEnabledValue.withValue { value in
                    value = payload.configuration["enabled"]?.boolValue
                }
            }
            return (
                200,
                jsonData([
                    "workspace_id": "ws1",
                    "connector_id": "email",
                    "configuration": [
                        "enabled": true,
                        "timeout_seconds": 12.5,
                        "endpoint": "https://new.example"
                    ],
                    "updated_at": "2026-03-05T04:10:00Z"
                ])
            )
        case "/v1/connectors/status":
            return (
                200,
                jsonData([
                    "workspace_id": "ws1",
                    "connectors": [
                        [
                            "connector_id": "email",
                            "plugin_id": "smtp",
                            "display_name": "Email",
                            "enabled": true,
                            "configured": true,
                            "status": "connected",
                            "summary": "Connected.",
                            "action_readiness": "ready",
                            "action_blockers": [],
                            "remediation_actions": [],
                            "configuration": [
                                "enabled": true,
                                "timeout_seconds": 12.5,
                                "endpoint": "https://new.example"
                            ]
                        ]
                    ]
                ])
            )
        default:
            return (404, jsonData(["error": ["code": "not_found", "message": "no fixture"]]))
        }
    }

    let store = makeStore(
        connectors: [seedConnector],
        daemonClient: V2DaemonAPIClient(session: session),
        withToken: true
    )
    store.setConnectorConfigurationDraftValue(connectorID: "email", key: "timeout_seconds", value: "12.5")
    store.setConnectorConfigurationDraftValue(connectorID: "email", key: "endpoint", value: "https://new.example")
    store.saveConnectorConfiguration("email")

    await waitForMutationCompletion(store, actionID: .connectorSaveConfig)
    await store.refreshConnectorsInventory(force: true)

    #expect(store.mutationLifecycle(for: .connectorSaveConfig).phase == .succeeded)
    #expect(store.connectors.first(where: { $0.id == "email" })?.hasConfigDraftChanges == false)
    #expect(capturedEnabledValue.snapshot() == true)
    #expect(capturedTimeoutValue.snapshot() == 12.5)
}

@MainActor
@Test("connector remediation request-permission action uses daemon permission endpoint")
func connectorRemediationRequestPermissionUsesDaemonEndpoint() async {
    let seedConnector = ConnectorState(
        id: "telegram",
        name: "Telegram",
        status: .needsAttention,
        summary: "Permission missing.",
        enabled: false,
        configured: false,
        actionReadiness: "blocked",
        remediationActions: [
            V2DaemonDiagnosticsRemediationAction(
                identifier: "request_permission",
                label: "Request Permission",
                intent: "request_permission",
                destination: nil,
                enabled: true
            )
        ],
        permissionState: "missing",
        configurationBaseline: ["enabled": "false"]
    )

    let permissionRequestCount = LockedBox<Int>(0)
    let session = makeMockSession { request in
        switch request.url?.path ?? "" {
        case "/v1/connectors/permission/request":
            permissionRequestCount.withValue { count in
                count += 1
            }
            return (
                200,
                jsonData([
                    "workspace_id": "ws1",
                    "connector_id": "telegram",
                    "permission_state": "prompted",
                    "message": "System permission prompt opened."
                ])
            )
        case "/v1/connectors/status":
            return (
                200,
                jsonData([
                    "workspace_id": "ws1",
                    "connectors": [
                        [
                            "connector_id": "telegram",
                            "plugin_id": "telegram",
                            "display_name": "Telegram",
                            "enabled": false,
                            "configured": false,
                            "status": "permission_missing",
                            "summary": "Permission prompt requested.",
                            "action_readiness": "blocked",
                            "action_blockers": [],
                            "remediation_actions": [],
                            "configuration": [
                                "enabled": false,
                                "permission_state": "missing"
                            ]
                        ]
                    ]
                ])
            )
        default:
            return (404, jsonData(["error": ["code": "not_found", "message": "no fixture"]]))
        }
    }

    let store = makeStore(
        connectors: [seedConnector],
        daemonClient: V2DaemonAPIClient(session: session),
        withToken: true
    )

    store.performConnectorRemediation(connectorID: "telegram", actionID: "request_permission")
    await waitForMutationCompletion(store, actionID: .connectorPermission)

    #expect(store.mutationLifecycle(for: .connectorPermission).phase == .succeeded)
    #expect(store.connectorActionStatus(for: "telegram")?.contains("permission") == true)
    #expect(permissionRequestCount.snapshot() == 1)
}

@MainActor
@Test("connector action disabled reason surfaces daemon blocker message")
func connectorActionDisabledReasonSurfacesDaemonBlockerMessage() {
    let blockedConnector = ConnectorState(
        id: "voice",
        name: "Voice",
        status: .needsAttention,
        summary: "Permission missing.",
        enabled: false,
        configured: false,
        actionReadiness: "blocked",
        actionBlockers: [
            V2DaemonActionReadinessBlocker(
                code: "permission_missing",
                message: "Grant microphone access before connecting this channel."
            )
        ],
        configurationBaseline: ["enabled": "false"]
    )
    let store = makeStore(connectors: [blockedConnector], withToken: true)

    let reason = store.connectorActionDisabledReason(for: "voice", action: .toggle)
    #expect(reason == "Grant microphone access before connecting this channel.")
}

@MainActor
@Test("replay retry mutation submits daemon task retry and reconciles feed")
func replayRetryMutationSubmitsDaemonTaskRetryAndReconcilesFeed() async {
    let event = makeReplayTaskEvent(
        replayKey: "task:task-retry",
        status: .failed,
        taskID: "task-retry"
    )
    let session = makeMockSession { request in
        switch request.url?.path ?? "" {
        case "/v1/tasks/retry":
            return (
                200,
                jsonData([
                    "workspace_id": "ws1",
                    "task_id": "task-retry",
                    "run_id": "run-retry-2",
                    "task_state": "running",
                    "run_state": "running"
                ])
            )
        case "/v1/approvals/inbox":
            return (200, jsonData(["workspace_id": "ws1", "approvals": []]))
        case "/v1/tasks/runs/list":
            return (
                200,
                jsonData([
                    "workspace_id": "ws1",
                    "items": [
                        [
                            "task_id": "task-retry",
                            "run_id": "run-retry-2",
                            "workspace_id": "ws1",
                            "title": "Retry failed operation",
                            "task_state": "running",
                            "run_state": "running",
                            "task_created_at": "2026-03-05T03:00:00Z",
                            "task_updated_at": "2026-03-05T03:01:00Z"
                        ]
                    ]
                ])
            )
        case "/v1/chat/history":
            return (200, jsonData(["workspace_id": "ws1", "items": [], "has_more": false]))
        default:
            return (404, jsonData(["error": ["code": "not_found", "message": "no fixture"]]))
        }
    }

    let store = makeStore(
        replayEvents: [event],
        daemonClient: V2DaemonAPIClient(session: session),
        withToken: true
    )
    store.statusFilter = .all
    store.selectEvent(event.id)
    store.retrySelectedEvent()

    await waitForMutationCompletion(store, actionID: .replayRetry)
    #expect(store.mutationLifecycle(for: .replayRetry).phase == .succeeded)
    #expect(store.replayEvents.contains(where: { $0.status == .running && $0.instruction.contains("Retry failed operation") }))
}

@MainActor
@Test("replay running action submits daemon task cancel and reconciles feed")
func replayRunningActionSubmitsDaemonTaskCancelAndReconcilesFeed() async {
    let event = makeReplayTaskEvent(
        replayKey: "task:task-stop",
        status: .running,
        taskID: "task-stop"
    )
    let session = makeMockSession { request in
        switch request.url?.path ?? "" {
        case "/v1/tasks/cancel":
            return (
                200,
                jsonData([
                    "workspace_id": "ws1",
                    "task_id": "task-stop",
                    "run_id": "run-stop-1",
                    "task_state": "failed",
                    "run_state": "cancelled"
                ])
            )
        case "/v1/approvals/inbox":
            return (200, jsonData(["workspace_id": "ws1", "approvals": []]))
        case "/v1/tasks/runs/list":
            return (
                200,
                jsonData([
                    "workspace_id": "ws1",
                    "items": [
                        [
                            "task_id": "task-stop",
                            "run_id": "run-stop-1",
                            "workspace_id": "ws1",
                            "title": "Cancelled operation",
                            "task_state": "failed",
                            "run_state": "cancelled",
                            "task_created_at": "2026-03-05T03:00:00Z",
                            "task_updated_at": "2026-03-05T03:01:00Z"
                        ]
                    ]
                ])
            )
        case "/v1/chat/history":
            return (200, jsonData(["workspace_id": "ws1", "items": [], "has_more": false]))
        default:
            return (404, jsonData(["error": ["code": "not_found", "message": "no fixture"]]))
        }
    }

    let store = makeStore(
        replayEvents: [event],
        daemonClient: V2DaemonAPIClient(session: session),
        withToken: true
    )
    store.statusFilter = .all
    store.selectEvent(event.id)
    store.completeSelectedRunningEvent()

    await waitForMutationCompletion(store, actionID: .replayComplete)
    #expect(store.mutationLifecycle(for: .replayComplete).phase == .succeeded)
    #expect(store.replayEvents.contains(where: { $0.status == .failed && $0.instruction.contains("Cancelled operation") }))
}

@MainActor
@Test("replay approve mutation failure rolls back optimistic state")
func replayApproveMutationFailureRollsBackOptimisticState() async {
    let event = makeReplayApprovalEvent(approvalRequestID: "apr-fail", risk: .low)
    let session = makeMockSession { request in
        switch request.url?.path ?? "" {
        case "/v1/approvals/decision":
            return (
                500,
                jsonData([
                    "error": [
                        "code": "internal_error",
                        "message": "decision write failed"
                    ]
                ])
            )
        default:
            return (404, jsonData(["error": ["code": "not_found", "message": "no fixture"]]))
        }
    }

    let store = makeStore(
        replayEvents: [event],
        daemonClient: V2DaemonAPIClient(session: session),
        withToken: true
    )
    store.statusFilter = .all
    store.selectEvent(event.id)
    store.approveSelectedEvent()

    #expect(store.replayEvents.first?.status == .running)

    await waitForMutationCompletion(store, actionID: .replayApprove)

    #expect(store.mutationLifecycle(for: .replayApprove).phase == .failed)
    let rolledBack = store.replayEvents.first(where: { $0.replayKey == "approval:apr-fail" })
    #expect(rolledBack?.status == .awaitingApproval)
    #expect(rolledBack?.actionSummary == "Waiting for user approval")
    #expect(store.lastFeedback?.localizedCaseInsensitiveContains("failed") == true)
}

@MainActor
@Test("get started readiness refresh projects live daemon state into checklist milestones")
func getStartedReadinessRefreshProjectsLiveDaemonState() async {
    let session = makeMockSession { request in
        let path = request.url?.path ?? ""
        switch path {
        case "/v1/daemon/lifecycle/status":
            return (
                200,
                jsonData([
                    "lifecycle_state": "running",
                    "setup_state": "ready",
                    "install_state": "installed",
                    "needs_install": false,
                    "needs_repair": false,
                    "control_auth": [
                        "state": "ready",
                        "source": "token",
                        "remediation_hints": []
                    ],
                    "worker_summary": [
                        "total": 4,
                        "running": 4,
                        "failed": 0
                    ],
                    "health_classification": [
                        "overall_state": "healthy",
                        "core_runtime_state": "healthy",
                        "plugin_runtime_state": "healthy",
                        "blocking": false
                    ]
                ])
            )
        case "/v1/models/resolve":
            return (
                200,
                jsonData([
                    "workspace_id": "ws1",
                    "task_class": "chat",
                    "provider": "built_in",
                    "model_key": "personalagent_default",
                    "source": "workspace_policy",
                    "notes": "ready"
                ])
            )
        case "/v1/connectors/status":
            return (
                200,
                jsonData([
                    "workspace_id": "ws1",
                    "connectors": [
                        [
                            "connector_id": "imessage",
                            "plugin_id": "imessage",
                            "display_name": "iMessage",
                            "enabled": true,
                            "configured": true,
                            "status": "connected",
                            "summary": "healthy",
                            "action_readiness": "ready",
                            "action_blockers": [],
                            "remediation_actions": []
                        ]
                    ]
                ])
            )
        case "/v1/approvals/inbox":
            return (
                200,
                jsonData([
                    "workspace_id": "ws1",
                    "approvals": [
                        [
                            "approval_request_id": "apr-1",
                            "workspace_id": "ws1",
                            "state": "pending",
                            "risk_level": "low",
                            "risk_rationale": "summary",
                            "requested_at": "2026-03-05T01:00:00Z"
                        ]
                    ]
                ])
            )
        case "/v1/tasks/runs/list":
            return (
                200,
                jsonData([
                    "workspace_id": "ws1",
                    "items": [
                        [
                            "task_id": "task-1",
                            "run_id": "run-1",
                            "workspace_id": "ws1",
                            "title": "Follow up",
                            "task_state": "running",
                            "run_state": "running",
                            "task_created_at": "2026-03-05T01:00:00Z",
                            "task_updated_at": "2026-03-05T01:05:00Z"
                        ]
                    ]
                ])
            )
        case "/v1/chat/history":
            return (
                200,
                jsonData([
                    "workspace_id": "ws1",
                    "items": [],
                    "has_more": false
                ])
            )
        default:
            return (
                404,
                jsonData([
                    "error": [
                        "code": "not_found",
                        "message": "no fixture"
                    ]
                ])
            )
        }
    }
    defer { V2MockURLProtocol.requestHandler = nil }
    let store = makeStore(daemonClient: V2DaemonAPIClient(session: session), withToken: true)

    let success = await store.refreshGetStartedReadiness()

    #expect(success)
    #expect(store.getStartedReadinessSnapshot.lifecycleIsOperational)
    #expect(store.getStartedReadinessSnapshot.hasRouteResolution)
    #expect(store.connectedLiveConnectorCount == 1)
    #expect(store.liveReplayInstructionCount == 2)
    #expect(store.setupChecklist.filter { !$0.isDone }.isEmpty)
}

@MainActor
@Test("fix next routes to canonical owner for the active setup blocker")
func fixNextRoutesToCanonicalOwner() {
    let store = makeStore(withToken: true)
    let lifecycleStatus: V2DaemonLifecycleStatusResponse = decodeJSON([
        "lifecycle_state": "running",
        "setup_state": "ready",
        "install_state": "installed",
        "needs_install": false,
        "needs_repair": false,
        "control_auth": [
            "state": "ready",
            "source": "token",
            "remediation_hints": []
        ],
        "worker_summary": [
            "total": 3,
            "running": 3,
            "failed": 0
        ],
        "health_classification": [
            "overall_state": "healthy",
            "core_runtime_state": "healthy",
            "plugin_runtime_state": "healthy",
            "blocking": false
        ]
    ])
    let connectorCard: V2DaemonConnectorStatusCard = decodeJSON([
        "connector_id": "imessage",
        "plugin_id": "imessage",
        "display_name": "iMessage",
        "enabled": true,
        "configured": true,
        "status": "connected",
        "action_readiness": "ready",
        "action_blockers": [],
        "remediation_actions": []
    ])
    store.getStartedReadinessSnapshot = V2GetStartedReadinessSnapshot(
        lifecycleStatus: lifecycleStatus,
        modelRoute: nil,
        connectorCards: [connectorCard],
        replayAvailability: V2ReplayAvailabilitySummary(approvalCount: 1, taskCount: 0, historyCount: 0)
    )
    store.selectedSection = .replayAndAsk

    guard let blocker = store.currentSetupBlocker else {
        Issue.record("Expected unresolved blocker")
        return
    }
    #expect(blocker.id == .defaultRouteResolved)
    #expect(store.shouldShowSetupBlockerRibbon)

    store.fixNextSetupBlocker()

    #expect(store.selectedSection == .connectorsAndModels)
    #expect(store.setupActionStatus(for: .defaultRouteResolved)?.contains("default chat route") == true)
}

@MainActor
@Test("replay refresh loads daemon-backed merged feed with deterministic filtering")
func replayRefreshLoadsDaemonBackedMergedFeed() async {
    let session = makeMockSession { request in
        let path = request.url?.path ?? ""
        switch path {
        case "/v1/approvals/inbox":
            return (
                200,
                jsonData([
                    "workspace_id": "ws1",
                    "approvals": [
                        [
                            "approval_request_id": "apr-1",
                            "workspace_id": "ws1",
                            "state": "pending",
                            "risk_level": "low",
                            "risk_rationale": "Requires explicit confirmation.",
                            "requested_phrase": "Send update to leadership",
                            "requested_at": "2026-03-05T01:00:00Z",
                            "task_id": "task-1",
                            "run_id": "run-1"
                        ]
                    ]
                ])
            )
        case "/v1/tasks/runs/list":
            return (
                200,
                jsonData([
                    "workspace_id": "ws1",
                    "items": [
                        [
                            "task_id": "task-1",
                            "run_id": "run-1",
                            "workspace_id": "ws1",
                            "title": "Send leadership update",
                            "task_state": "running",
                            "run_state": "running",
                            "task_created_at": "2026-03-05T01:00:00Z",
                            "task_updated_at": "2026-03-05T01:03:00Z"
                        ]
                    ]
                ])
            )
        case "/v1/chat/history":
            return (
                200,
                jsonData([
                    "workspace_id": "ws1",
                    "items": [
                        [
                            "record_id": "hist-1",
                            "turn_id": "turn-1",
                            "correlation_id": "corr-1",
                            "channel_id": "app",
                            "item_index": 0,
                            "item": [
                                "item_id": "item-1",
                                "type": "user_message",
                                "role": "user",
                                "status": "completed",
                                "content": "Send update to leadership"
                            ],
                            "task_run_reference": [
                                "available": true,
                                "source": "task",
                                "task_id": "task-1",
                                "run_id": "run-1",
                                "task_state": "running",
                                "run_state": "running"
                            ],
                            "created_at": "2026-03-05T01:00:00Z"
                        ],
                        [
                            "record_id": "hist-2",
                            "turn_id": "turn-1",
                            "correlation_id": "corr-1",
                            "channel_id": "app",
                            "item_index": 1,
                            "item": [
                                "item_id": "item-2",
                                "type": "assistant_message",
                                "role": "assistant",
                                "status": "pending",
                                "content": "Waiting for your approval."
                            ],
                            "task_run_reference": [
                                "available": true,
                                "source": "task",
                                "task_id": "task-1",
                                "run_id": "run-1",
                                "task_state": "running",
                                "run_state": "running"
                            ],
                            "created_at": "2026-03-05T01:01:00Z"
                        ],
                        [
                            "record_id": "hist-3",
                            "turn_id": "turn-2",
                            "correlation_id": "corr-2",
                            "channel_id": "voice",
                            "item_index": 0,
                            "item": [
                                "item_id": "item-3",
                                "type": "user_message",
                                "role": "user",
                                "status": "completed",
                                "content": "Book travel to NYC."
                            ],
                            "task_run_reference": [
                                "available": true,
                                "source": "task",
                                "task_id": "task-2",
                                "run_id": "run-2",
                                "task_state": "completed",
                                "run_state": "completed"
                            ],
                            "created_at": "2026-03-05T00:30:00Z"
                        ]
                    ],
                    "has_more": false
                ])
            )
        default:
            return (404, jsonData(["error": ["code": "not_found", "message": "no fixture"]]))
        }
    }
    defer { V2MockURLProtocol.requestHandler = nil }

    let store = makeStore(
        daemonClient: V2DaemonAPIClient(session: session),
        withToken: true
    )

    await store.refreshReplayFeed(resetPagination: true)

    #expect(store.replayFeedQueryState.hasLoadedOnce)
    #expect(store.replayFeedQueryState.lastLoadedAt != nil)
    #expect(!store.replayEvents.isEmpty)
    #expect(store.replayEvents.contains(where: { $0.instruction.contains("Send update to leadership") }))
    #expect(store.replayEvents.contains(where: { $0.source == .voice }))

    store.statusFilter = .all
    store.selectedSources = [.voice]
    #expect(!store.filteredEvents.isEmpty)
    #expect(store.filteredEvents.allSatisfy({ $0.source == .voice }))
}

@MainActor
@Test("replay load more advances pagination state when daemon indicates more history")
func replayLoadMoreAdvancesPaginationState() async {
    let session = makeMockSession { request in
        let path = request.url?.path ?? ""
        switch path {
        case "/v1/approvals/inbox":
            return (200, jsonData(["workspace_id": "ws1", "approvals": []]))
        case "/v1/tasks/runs/list":
            return (200, jsonData(["workspace_id": "ws1", "items": []]))
        case "/v1/chat/history":
            return (
                200,
                jsonData([
                    "workspace_id": "ws1",
                    "items": [
                        [
                            "record_id": "hist-page",
                            "turn_id": "turn-page",
                            "correlation_id": "corr-page",
                            "channel_id": "app",
                            "item_index": 0,
                            "item": [
                                "item_id": "item-page",
                                "type": "user_message",
                                "role": "user",
                                "status": "completed",
                                "content": "Summarize account status."
                            ],
                            "task_run_reference": [
                                "available": false,
                                "source": "none"
                            ],
                            "created_at": "2026-03-05T00:10:00Z"
                        ]
                    ],
                    "has_more": true
                ])
            )
        default:
            return (404, jsonData(["error": ["code": "not_found", "message": "no fixture"]]))
        }
    }
    defer { V2MockURLProtocol.requestHandler = nil }

    let store = makeStore(
        daemonClient: V2DaemonAPIClient(session: session),
        withToken: true
    )

    await store.refreshReplayFeed(resetPagination: true)
    #expect(store.replayFeedQueryState.requestedPage == 1)
    #expect(store.replayFeedQueryState.canLoadMore)

    await store.loadMoreReplayFeed()
    #expect(store.replayFeedQueryState.requestedPage == 2)
}

@MainActor
@Test("replay detail evidence refresh projects inspect and history signals")
func replayDetailEvidenceRefreshProjectsInspectAndHistorySignals() async {
    let session = makeMockSession { request in
        let path = request.url?.path ?? ""
        switch path {
        case "/v1/inspect/run":
            return (
                200,
                jsonData([
                    "task": [
                        "task_id": "task-9",
                        "title": "Send customer follow-up",
                        "state": "running"
                    ],
                    "run": [
                        "run_id": "run-9",
                        "workspace_id": "ws1",
                        "task_id": "task-9",
                        "state": "running",
                        "created_at": "2026-03-05T02:00:00Z",
                        "updated_at": "2026-03-05T02:05:00Z"
                    ],
                    "steps": [
                        [
                            "step_id": "step-1",
                            "name": "Compose Draft",
                            "status": "completed"
                        ],
                        [
                            "step_id": "step-2",
                            "name": "Send Message",
                            "status": "pending"
                        ]
                    ],
                    "audit_entries": [],
                    "route": [
                        "available": true,
                        "task_class": "chat",
                        "provider": "built_in",
                        "model_key": "personalagent_default",
                        "task_class_source": "workspace_policy",
                        "route_source": "workspace_policy",
                        "notes": "Route selected from workspace defaults."
                    ]
                ])
            )
        case "/v1/inspect/logs/query":
            return (
                200,
                jsonData([
                    "workspace_id": "ws1",
                    "logs": [
                        [
                            "log_id": "log-1",
                            "workspace_id": "ws1",
                            "run_id": "run-9",
                            "event_type": "tool_call_started",
                            "status": "completed",
                            "input_summary": "mail_send",
                            "output_summary": "draft prepared",
                            "created_at": "2026-03-05T02:03:00Z",
                            "route": ["available": true, "task_class": "chat", "task_class_source": "policy", "route_source": "policy"]
                        ]
                    ],
                    "next_cursor_created_at": nil,
                    "next_cursor_id": nil
                ])
            )
        case "/v1/chat/history":
            return (
                200,
                jsonData([
                    "workspace_id": "ws1",
                    "items": [
                        [
                            "record_id": "hist-9-1",
                            "turn_id": "turn-9",
                            "correlation_id": "corr-9",
                            "channel_id": "app",
                            "item_index": 0,
                            "item": [
                                "item_id": "item-9-1",
                                "type": "user_message",
                                "role": "user",
                                "status": "completed",
                                "content": "Send customer follow-up email."
                            ],
                            "task_run_reference": [
                                "available": true,
                                "source": "task",
                                "task_id": "task-9",
                                "run_id": "run-9",
                                "task_state": "running",
                                "run_state": "running"
                            ],
                            "created_at": "2026-03-05T02:00:00Z"
                        ],
                        [
                            "record_id": "hist-9-2",
                            "turn_id": "turn-9",
                            "correlation_id": "corr-9",
                            "channel_id": "app",
                            "item_index": 1,
                            "item": [
                                "item_id": "item-9-2",
                                "type": "assistant_message",
                                "role": "assistant",
                                "status": "completed",
                                "content": "Prepared a concise customer follow-up draft."
                            ],
                            "task_run_reference": [
                                "available": true,
                                "source": "task",
                                "task_id": "task-9",
                                "run_id": "run-9",
                                "task_state": "running",
                                "run_state": "running"
                            ],
                            "created_at": "2026-03-05T02:01:00Z"
                        ]
                    ],
                    "has_more": false
                ])
            )
        case "/v1/approvals/inbox":
            return (
                200,
                jsonData([
                    "workspace_id": "ws1",
                    "approvals": [
                        [
                            "approval_request_id": "apr-9",
                            "workspace_id": "ws1",
                            "state": "pending",
                            "risk_level": "medium",
                            "risk_rationale": "Customer comms require explicit approval.",
                            "requested_at": "2026-03-05T02:00:00Z"
                        ]
                    ]
                ])
            )
        default:
            return (404, jsonData(["error": ["code": "not_found", "message": "no fixture"]]))
        }
    }

    let replayEvent = ReplayEvent(
        source: .app,
        sourceContext: .app(AppReplaySourceContext(workspace: "ws1", sessionID: "session-9", messageID: "msg-9")),
        receivedAt: Date(),
        instruction: "Send customer follow-up email.",
        interpretedIntent: "Draft and send follow-up.",
        actionSummary: "Pending evidence refresh.",
        status: .running,
        risk: .low,
        channelsTouched: ["app"],
        decisionTrace: AppShellV2Store.trace(
            received: "Captured",
            intent: "Parsed",
            planning: "Planned",
            execution: "Running",
            executionStatus: .pending
        ),
        daemonLocator: ReplayEventDaemonLocator(
            correlationID: "corr-9",
            turnID: "turn-9",
            historyRecordIDs: ["hist-9-1", "hist-9-2"],
            approvalRequestID: "apr-9",
            taskID: "task-9",
            runID: "run-9",
            channelID: "app"
        )
    )

    let store = makeStore(
        replayEvents: [replayEvent],
        daemonClient: V2DaemonAPIClient(session: session),
        withToken: true
    )
    store.statusFilter = .all
    store.selectEvent(replayEvent.id)

    await store.refreshSelectedReplayDetailEvidence(force: true)

    guard let evidence = store.replayDetailEvidence(for: store.selectedEvent) else {
        Issue.record("Expected replay detail evidence state")
        return
    }

    #expect(evidence.phase == .ready)
    #expect(evidence.whatCameIn?.contains("customer follow-up email") == true)
    #expect(
        (evidence.whatHappened?.contains("follow-up draft") == true)
            || (evidence.whatHappened?.contains("Run state: running.") == true)
    )
    #expect(evidence.approvalContext?.contains("explicit approval") == true)
    #expect(evidence.decisionTrace?.contains(where: { $0.title == "Compose Draft" }) == true)
    #expect(evidence.sourceContextFields?.contains(where: { $0.label == "Run ID" && $0.value == "run-9" }) == true)
}

@MainActor
@Test("replay detail evidence returns empty phase when no daemon locator exists")
func replayDetailEvidenceEmptyWhenNoLocatorExists() async {
    let replayEvent = ReplayEvent(
        source: .app,
        receivedAt: Date(),
        instruction: "Summarize my inbox",
        interpretedIntent: "Generate summary",
        actionSummary: "Summary in progress.",
        status: .running,
        risk: .low,
        channelsTouched: ["app"],
        decisionTrace: AppShellV2Store.trace(
            received: "Captured",
            intent: "Parsed",
            planning: "Planned",
            execution: "Running",
            executionStatus: .pending
        )
    )

    let store = makeStore(replayEvents: [replayEvent], withToken: true)
    store.statusFilter = .all
    store.selectEvent(replayEvent.id)

    await store.refreshSelectedReplayDetailEvidence(force: true)

    guard let evidence = store.replayDetailEvidence(for: store.selectedEvent) else {
        Issue.record("Expected replay detail evidence state")
        return
    }

    #expect(evidence.phase == .empty)
    #expect(evidence.summary?.contains("No daemon evidence locator") == true)
}

@MainActor
private func waitForMutationCompletion(
    _ store: AppShellV2Store,
    actionID: V2MutationActionID,
    timeoutNanoseconds: UInt64 = 2_000_000_000
) async {
    let start = DispatchTime.now().uptimeNanoseconds
    while store.mutationLifecycle(for: actionID).phase == .inFlight {
        if DispatchTime.now().uptimeNanoseconds - start > timeoutNanoseconds {
            break
        }
        try? await Task.sleep(nanoseconds: 10_000_000)
    }
}

@MainActor
private func makeReplayApprovalEvent(
    approvalRequestID: String,
    risk: ReplayRiskLevel
) -> ReplayEvent {
    ReplayEvent(
        replayKey: "approval:\(approvalRequestID.lowercased())",
        source: .iMessage,
        sourceContext: .iMessage(
            IMessageReplaySourceContext(
                contactName: "Alex",
                contactPhoneSuffix: "+1 •••• 0000",
                threadID: "thread-\(approvalRequestID)"
            )
        ),
        receivedAt: Date(),
        instruction: "Send status update to team",
        interpretedIntent: "Deliver a quick project update",
        actionSummary: "Waiting for user approval",
        status: .awaitingApproval,
        risk: risk,
        approvalReason: "Team-wide outbound message",
        channelsTouched: ["slack"],
        decisionTrace: [
            ReplayDecisionStage(title: "Instruction", detail: "Captured", status: .completed),
            ReplayDecisionStage(title: "Execution", detail: "Waiting", status: .pending)
        ],
        daemonLocator: ReplayEventDaemonLocator(approvalRequestID: approvalRequestID)
    )
}

@MainActor
private func makeReplayTaskEvent(
    replayKey: String,
    status: ReplayEventStatus,
    taskID: String
) -> ReplayEvent {
    ReplayEvent(
        replayKey: replayKey,
        source: .app,
        sourceContext: .app(
            AppReplaySourceContext(
                workspace: "ws1",
                sessionID: "session-\(taskID)",
                messageID: "msg-\(taskID)"
            )
        ),
        receivedAt: Date(),
        instruction: "Run task \(taskID)",
        interpretedIntent: "Recover workflow execution",
        actionSummary: status == .failed ? "Execution failed." : "Execution running.",
        status: status,
        risk: .low,
        channelsTouched: ["tasks"],
        decisionTrace: AppShellV2Store.trace(
            received: "Captured",
            intent: "Mapped to task",
            planning: "Queued",
            execution: status == .failed ? "Failed" : "Running",
            executionStatus: status == .failed ? .blocked : .pending
        ),
        daemonLocator: ReplayEventDaemonLocator(taskID: taskID)
    )
}

@MainActor
private func makeStore(
    connectors: [ConnectorState] = AppShellV2Store.defaultConnectors,
    models: [ModelOption] = AppShellV2Store.defaultModels,
    replayEvents: [ReplayEvent] = AppShellV2Store.defaultReplayEvents,
    daemonClient: V2DaemonAPIClient = V2DaemonAPIClient(),
    withToken: Bool = true
) -> AppShellV2Store {
    let sessionStore = makeSessionStore(withToken: withToken)
    return AppShellV2Store(
        connectors: connectors,
        models: models,
        replayEvents: replayEvents,
        daemonClient: daemonClient,
        sessionConfigStore: sessionStore
    )
}

@MainActor
private func makeSessionStore(withToken: Bool) -> V2SessionConfigStore {
    let suite = "personalagent.ui.v2.tests.\(UUID().uuidString)"
    let defaults = UserDefaults(suiteName: suite) ?? .standard
    defaults.removePersistentDomain(forName: suite)
    let sessionStore = V2SessionConfigStore(
        userDefaults: defaults,
        secretStore: V2InMemorySecretStore()
    )
    sessionStore.daemonBaseURL = "http://127.0.0.1:7071"
    sessionStore.workspaceID = "ws1"
    sessionStore.principalActorID = "default"
    if withToken {
        try? sessionStore.saveAccessToken("token-test")
    }
    return sessionStore
}

private func makeMockSession(
    responder: @escaping @Sendable (URLRequest) throws -> (statusCode: Int, data: Data)
) -> URLSession {
    let mockID = UUID().uuidString.lowercased()
    V2MockURLProtocol.registerHandler(responder, id: mockID)
    let config = URLSessionConfiguration.ephemeral
    config.httpAdditionalHeaders = [V2MockURLProtocol.mockHeaderKey: mockID]
    config.protocolClasses = [V2MockURLProtocol.self]
    return URLSession(configuration: config)
}

private func jsonData(_ object: Any) -> Data {
    (try? JSONSerialization.data(withJSONObject: object, options: [])) ?? Data("{}".utf8)
}

private func requestBodyData(_ request: URLRequest) -> Data? {
    if let body = request.httpBody {
        return body
    }

    guard let stream = request.httpBodyStream else {
        return nil
    }

    stream.open()
    defer { stream.close() }

    var data = Data()
    let chunkSize = 1024
    let buffer = UnsafeMutablePointer<UInt8>.allocate(capacity: chunkSize)
    defer { buffer.deallocate() }

    while stream.hasBytesAvailable {
        let count = stream.read(buffer, maxLength: chunkSize)
        if count < 0 {
            return nil
        }
        if count == 0 {
            break
        }
        data.append(buffer, count: count)
    }

    return data.isEmpty ? nil : data
}

private func decodeJSON<T: Decodable>(_ object: Any) -> T {
    let data = jsonData(object)
    return (try? JSONDecoder().decode(T.self, from: data))
        ?? {
            fatalError("Failed to decode \(T.self) from JSON fixture.")
        }()
}

private final class LockedBox<Value>: @unchecked Sendable {
    private let lock = NSLock()
    private var value: Value

    init(_ value: Value) {
        self.value = value
    }

    func withValue(_ mutate: (inout Value) -> Void) {
        lock.lock()
        defer { lock.unlock() }
        mutate(&value)
    }

    func snapshot() -> Value {
        lock.lock()
        defer { lock.unlock() }
        return value
    }
}

private final class V2MockURLProtocol: URLProtocol, @unchecked Sendable {
    static let mockHeaderKey = "X-PA-V2-Mock-ID"
    nonisolated(unsafe) static var requestHandler: (@Sendable (URLRequest) throws -> (statusCode: Int, data: Data))?
    nonisolated(unsafe) private static var handlersByID: [String: (@Sendable (URLRequest) throws -> (statusCode: Int, data: Data))] = [:]
    nonisolated(unsafe) private static let handlersLock = NSLock()

    static func registerHandler(
        _ handler: @escaping @Sendable (URLRequest) throws -> (statusCode: Int, data: Data),
        id: String
    ) {
        handlersLock.lock()
        defer { handlersLock.unlock() }
        handlersByID[id] = handler
    }

    override class func canInit(with request: URLRequest) -> Bool {
        true
    }

    override class func canonicalRequest(for request: URLRequest) -> URLRequest {
        request
    }

    override func startLoading() {
        let handler: (@Sendable (URLRequest) throws -> (statusCode: Int, data: Data))?
        Self.handlersLock.lock()
        if let mockID = request.value(forHTTPHeaderField: Self.mockHeaderKey) {
            handler = Self.handlersByID[mockID]
        } else {
            handler = Self.requestHandler
        }
        Self.handlersLock.unlock()

        guard let handler else {
            client?.urlProtocol(self, didFailWithError: URLError(.badServerResponse))
            return
        }

        do {
            let (statusCode, data) = try handler(request)
            let response = HTTPURLResponse(
                url: request.url ?? URL(string: "http://localhost")!,
                statusCode: statusCode,
                httpVersion: "HTTP/1.1",
                headerFields: ["Content-Type": "application/json"]
            )!

            client?.urlProtocol(self, didReceive: response, cacheStoragePolicy: .notAllowed)
            client?.urlProtocol(self, didLoad: data)
            client?.urlProtocolDidFinishLoading(self)
        } catch {
            client?.urlProtocol(self, didFailWithError: error)
        }
    }

    override func stopLoading() {}
}
