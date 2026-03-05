import Foundation
import XCTest
@testable import PersonalAgentUI

@MainActor
final class AppShellStateChatTests: XCTestCase {
    private let tokenDefaultsKey = "personalagent.ui.local_dev_token"
    private let onboardingDefaultsKey = "personalagent.ui.onboarding_complete"

    override func setUp() {
        super.setUp()
        AppShellState._test_setLocalDevTokenSecretReference(
            service: "personalagent.ui.tests.chat.\(UUID().uuidString)",
            account: "daemon_auth_token"
        )
        AppShellState._test_clearPersistedLocalDevToken()
    }

    override func tearDown() {
        AppShellState._test_clearPersistedLocalDevToken()
        AppShellState._test_resetLocalDevTokenPersistenceHooks()
        super.tearDown()
    }

    func testSendChatDraftWithoutTokenShowsStatusAndDoesNotAppendMessage() async {
        let defaults = appShellStateTestUserDefaults()
        let priorToken = defaults.object(forKey: tokenDefaultsKey)
        let priorOnboarding = defaults.object(forKey: onboardingDefaultsKey)
        defer {
            if let priorToken {
                defaults.set(priorToken, forKey: tokenDefaultsKey)
            } else {
                defaults.removeObject(forKey: tokenDefaultsKey)
            }
            if let priorOnboarding {
                defaults.set(priorOnboarding, forKey: onboardingDefaultsKey)
            } else {
                defaults.removeObject(forKey: onboardingDefaultsKey)
            }
        }

        defaults.removeObject(forKey: tokenDefaultsKey)
        defaults.removeObject(forKey: onboardingDefaultsKey)

        let state = AppShellState()
        state.clearLocalDevToken()
        state.chatDraft = "hello from test"

        state.sendChatDraft()

        XCTAssertEqual(state.chatTimelineItems.count, 0)
        XCTAssertEqual(state.chatStatusMessage, "Set Assistant Access Token before sending chat turns.")
        XCTAssertFalse(state.isChatStreaming)
    }

    func testRefreshChatRoutePreflightWithoutTokenShowsStatus() async {
        let defaults = appShellStateTestUserDefaults()
        let priorToken = defaults.object(forKey: tokenDefaultsKey)
        let priorOnboarding = defaults.object(forKey: onboardingDefaultsKey)
        defer {
            if let priorToken {
                defaults.set(priorToken, forKey: tokenDefaultsKey)
            } else {
                defaults.removeObject(forKey: tokenDefaultsKey)
            }
            if let priorOnboarding {
                defaults.set(priorOnboarding, forKey: onboardingDefaultsKey)
            } else {
                defaults.removeObject(forKey: onboardingDefaultsKey)
            }
        }

        defaults.removeObject(forKey: tokenDefaultsKey)
        defaults.removeObject(forKey: onboardingDefaultsKey)

        let state = AppShellState()
        state.clearLocalDevToken()

        state.refreshChatRoutePreflight()
        try? await Task.sleep(for: .milliseconds(50))

        XCTAssertEqual(state.chatStatusMessage, "Set Assistant Access Token before checking chat route.")
        XCTAssertNil(state.chatRouteRemediationMessage)
    }

    func testRefreshChatTurnExplainabilityWithoutTokenSetsDeterministicStatus() async {
        let defaults = appShellStateTestUserDefaults()
        let priorToken = defaults.object(forKey: tokenDefaultsKey)
        let priorOnboarding = defaults.object(forKey: onboardingDefaultsKey)
        defer {
            if let priorToken {
                defaults.set(priorToken, forKey: tokenDefaultsKey)
            } else {
                defaults.removeObject(forKey: tokenDefaultsKey)
            }
            if let priorOnboarding {
                defaults.set(priorOnboarding, forKey: onboardingDefaultsKey)
            } else {
                defaults.removeObject(forKey: onboardingDefaultsKey)
            }
        }

        defaults.removeObject(forKey: tokenDefaultsKey)
        defaults.removeObject(forKey: onboardingDefaultsKey)

        let state = AppShellState()
        state.clearLocalDevToken()

        state.refreshChatTurnExplainability()
        try? await Task.sleep(for: .milliseconds(50))

        XCTAssertEqual(
            state.chatExplainabilityStatusMessage,
            "Set Assistant Access Token before loading chat explainability."
        )
        XCTAssertNil(state.chatLatestTurnExplainability)
        XCTAssertFalse(state.isChatExplainabilityInFlight)
    }

    func testSendChatDraftWithOutOfScopeActingAsSelectionShowsValidationAndDoesNotStream() {
        let state = AppShellState()
        state.localDevTokenInput = "test-token"
        state.saveLocalDevToken()
        state.chatDraft = "hello from test"
        state.identityPrincipalItems = [
            IdentityPrincipalItem(
                id: "actor.requester",
                displayName: "Requester",
                actorType: "human",
                actorStatus: "ACTIVE",
                principalStatus: "ACTIVE",
                isActive: true,
                handles: []
            )
        ]
        state.selectedPrincipal = "actor.unknown"

        state.sendChatDraft()

        XCTAssertEqual(
            state.chatStatusMessage,
            "Selected acting-as principal `actor.unknown` is not in the active workspace directory. Refresh Identity Directory in Configuration."
        )
        XCTAssertFalse(state.isChatStreaming)
    }

    func testOpenModelsForChatRemediationSelectsModelsSection() {
        let state = AppShellState()
        state.selectedSection = .chat

        state.openModelsForChatRemediation()

        XCTAssertEqual(state.selectedSection, .models)
    }

    func testRestoreLastFailedChatDraftForRetryRestoresDraftAndClearsGuidance() {
        let state = AppShellState()
        state.chatLastFailedDraft = "retry this prompt"
        state.chatFailureRemediationMessage = "Chat request failed. Review runtime status and retry."

        state.restoreLastFailedChatDraftForRetry()

        XCTAssertEqual(state.chatDraft, "retry this prompt")
        XCTAssertNil(state.chatFailureRemediationMessage)
        XCTAssertEqual(state.chatStatusMessage, "Restored your last message. Press Send to retry.")
    }

    func testFixAndContinueRouteRemediationWithoutTokenNavigatesToConfiguration() async {
        let state = AppShellState()
        state.clearLocalDevToken()
        state.selectedSection = .chat
        state.chatRouteRemediationMessage = "No enabled chat model is ready."

        state.runChatFixAndContinueFromRouteRemediation()
        try? await Task.sleep(for: .milliseconds(120))

        XCTAssertEqual(state.selectedSection, .configuration)
        XCTAssertFalse(state.isChatFixAndContinueInFlight)
        XCTAssertTrue(
            state.chatFixAndContinueStatusMessage?.contains("Assistant Access Token in Configuration") == true
        )
    }

    func testFixAndContinueFailureRemediationRestoresDraftBeforeNavigation() async {
        let state = AppShellState()
        state.clearLocalDevToken()
        state.selectedSection = .chat
        state.chatLastFailedDraft = "retry me"
        state.chatFailureRemediationMessage = "Chat request could not reach daemon. Start or refresh daemon status, then retry."

        state.runChatFixAndContinueFromFailureRemediation()
        try? await Task.sleep(for: .milliseconds(120))

        XCTAssertEqual(state.chatDraft, "retry me")
        XCTAssertEqual(state.selectedSection, .configuration)
        XCTAssertFalse(state.isChatFixAndContinueInFlight)
        XCTAssertTrue(
            state.chatFixAndContinueStatusMessage?.contains("return to Chat to continue automatically") == true
        )
    }

    func testFixAndContinuePendingStateAutoChecksWhenReturningToChat() async {
        let state = AppShellState()
        state.clearLocalDevToken()
        state.selectedSection = .chat
        state.chatRouteRemediationMessage = "No enabled chat model is ready."

        state.runChatFixAndContinueFromRouteRemediation()
        try? await Task.sleep(for: .milliseconds(120))
        XCTAssertEqual(state.selectedSection, .configuration)

        state.selectedSection = .chat
        try? await Task.sleep(for: .milliseconds(120))

        XCTAssertEqual(state.selectedSection, .chat)
        XCTAssertTrue(
            state.chatFixAndContinueStatusMessage?.contains("Assistant Access Token is still missing") == true
        )
    }

    func testOpenTasksForChatTraceabilitySeedsRunIDAndNavigates() {
        let state = AppShellState()
        state.selectedSection = .chat
        state.chatLatestTurnTraceability = ChatTaskRunTraceabilityItem(
            available: true,
            source: "audit",
            taskID: "task-chat-1",
            runID: "run-chat-1",
            taskState: "running",
            runState: "running",
            correlationID: "corr-chat-1",
            taskClass: "chat",
            provider: "openai",
            modelKey: "gpt-5",
            routeSource: "policy"
        )

        state.openTasksForChatTraceability()

        XCTAssertEqual(state.selectedSection, .tasks)
        XCTAssertEqual(state.tasksSearchSeed, "run-chat-1")
        XCTAssertEqual(state.tasksStatusMessage, "Opened Tasks for chat run run-chat-1.")
    }

    func testOpenTasksForChatTraceabilityFallsBackToModelRouteWhenIdentityMissing() {
        let state = AppShellState()
        state.selectedSection = .chat
        state.modelRouteSummary = ModelRouteSummary(
            provider: "openai",
            modelKey: "gpt-4o-mini",
            source: "policy",
            notes: nil
        )

        state.openTasksForChatTraceability()

        XCTAssertEqual(state.selectedSection, .tasks)
        XCTAssertEqual(state.tasksSearchSeed, "gpt-4o-mini")
        XCTAssertEqual(state.tasksStatusMessage, "Opened Tasks for chat route openai/gpt-4o-mini.")
    }

    func testOpenInspectForChatTraceabilitySeedsRunFilterAndNavigates() {
        let state = AppShellState()
        state.selectedSection = .chat
        state.chatLatestTurnTraceability = ChatTaskRunTraceabilityItem(
            available: true,
            source: "audit",
            taskID: "task-chat-2",
            runID: "run-chat-2",
            taskState: nil,
            runState: nil,
            correlationID: "corr-chat-2",
            taskClass: "chat",
            provider: "openai",
            modelKey: "gpt-5",
            routeSource: "policy"
        )

        state.openInspectForChatTraceability()

        XCTAssertEqual(state.selectedSection, .inspect)
        XCTAssertEqual(state.inspectFocusedRunID, "run-chat-2")
        XCTAssertNil(state.inspectSearchSeed)
        XCTAssertEqual(state.inspectStatusMessage, "Loading inspect logs for chat run run-chat-2…")
    }

    func testOpenInspectForChatTraceabilityFallsBackToCorrelationSeed() {
        let state = AppShellState()
        state.selectedSection = .chat
        state.chatLatestTurnTraceability = ChatTaskRunTraceabilityItem(
            available: false,
            source: "none",
            taskID: nil,
            runID: nil,
            taskState: nil,
            runState: nil,
            correlationID: "corr-chat-fallback",
            taskClass: "chat",
            provider: "openai",
            modelKey: "gpt-4o-mini",
            routeSource: "policy"
        )

        state.openInspectForChatTraceability()

        XCTAssertEqual(state.selectedSection, .inspect)
        XCTAssertNil(state.inspectFocusedRunID)
        XCTAssertEqual(state.inspectSearchSeed, "corr-chat-fallback")
        XCTAssertEqual(state.inspectStatusMessage, "Opened Inspect for chat correlation corr-chat-fallback.")
    }

    func testChatTimelineApprovalActionsExposeDeterministicDisabledReasons() {
        let state = AppShellState()
        let item = ChatTimelineItem(
            id: "approval-row-1",
            kind: .approvalRequest,
            state: .blocked,
            title: "Approval Required",
            summary: "Approval pending.",
            approvalRequestID: nil
        )

        let actions = state.chatTimelineActions(for: item)

        XCTAssertEqual(actions.first?.intent, .openApprovals)
        XCTAssertFalse(actions.first?.enabled ?? true)
        XCTAssertEqual(actions.first?.disabledReason, "Approval request ID is missing.")
        XCTAssertEqual(actions.dropFirst().first?.intent, .resumeTurn)
        XCTAssertFalse(actions.dropFirst().first?.enabled ?? true)
        XCTAssertEqual(actions.dropFirst().first?.disabledReason, "No interrupted turn draft is available to resume.")
    }

    func testPerformChatTimelineActionOpenApprovalsNavigatesAndSeedsFilter() {
        let state = AppShellState()
        state.selectedSection = .chat
        state.chatTimelineItems = [
            ChatTimelineItem(
                id: "approval-row-2",
                kind: .approvalRequest,
                state: .blocked,
                title: "Approval Required",
                summary: "Waiting for approval.",
                approvalRequestID: "approval-123"
            )
        ]

        state.performChatTimelineAction(itemID: "approval-row-2", intent: .openApprovals)

        XCTAssertEqual(state.selectedSection, .approvals)
        XCTAssertEqual(state.approvalsSearchSeed, "approval-123")
        XCTAssertEqual(state.approvalsStatusMessage, "Opened Approvals for request approval-123.")
    }

    func testChatTimelineApprovalResumeDisabledWhileDecisionPending() {
        let state = AppShellState()
        state.chatLastFailedDraft = "resume this turn"
        state.approvalInboxItems = [
            ApprovalInboxItem(
                id: "approval-123",
                taskTitle: "Send email",
                decisionState: .pending,
                decisionOutcome: nil,
                riskLevel: .destructive,
                riskRationale: "Destructive action",
                requestedAtLabel: "now",
                decidedAtLabel: nil,
                decisionByActorID: nil,
                decisionRationale: nil,
                requestedPhrase: "GO AHEAD",
                taskState: "awaiting_approval",
                runState: "awaiting_approval",
                stepName: "mail.send",
                stepCapabilityKey: "mail.send",
                requestedByActorID: "actor.requester",
                subjectPrincipalActorID: "actor.requester",
                actingAsActorID: "actor.requester",
                taskID: "task-1",
                runID: "run-1",
                stepID: "step-1",
                route: WorkflowRouteContext(taskClass: "chat", provider: "openai", modelKey: "gpt-5", routeSource: "policy")
            )
        ]
        let item = ChatTimelineItem(
            id: "approval-row-3",
            kind: .approvalRequest,
            state: .blocked,
            title: "Approval Required",
            summary: "Approval pending.",
            approvalRequestID: "approval-123"
        )

        let resumeAction = state.chatTimelineActions(for: item).first(where: { $0.intent == .resumeTurn })

        XCTAssertEqual(resumeAction?.enabled, false)
        XCTAssertEqual(resumeAction?.disabledReason, "Submit approve or reject before resuming the turn.")
    }

    func testChatInlineApprovalFastPathAllowsPolicyRiskApprovals() {
        let state = AppShellState()
        let approval = ApprovalInboxItem(
            id: "approval-policy-1",
            taskTitle: "Send status update",
            decisionState: .pending,
            decisionOutcome: nil,
            riskLevel: .policy,
            riskRationale: "Standard policy check",
            requestedAtLabel: "now",
            decidedAtLabel: nil,
            decisionByActorID: nil,
            decisionRationale: nil,
            requestedPhrase: "GO AHEAD",
            taskState: "awaiting_approval",
            runState: "awaiting_approval",
            stepName: "message.send",
            stepCapabilityKey: "message.send",
            requestedByActorID: "actor.requester",
            subjectPrincipalActorID: "actor.requester",
            actingAsActorID: "actor.requester",
            taskID: "task-1",
            runID: "run-1",
            stepID: "step-1",
            route: WorkflowRouteContext(taskClass: "chat", provider: "openai", modelKey: "gpt-4.1", routeSource: "policy")
        )

        XCTAssertNil(state.chatInlineApprovalFastPathBlockedReason(for: approval))
    }

    func testChatInlineApprovalFastPathBlocksDestructiveApprovals() {
        let state = AppShellState()
        let approval = ApprovalInboxItem(
            id: "approval-destructive-1",
            taskTitle: "Delete records",
            decisionState: .pending,
            decisionOutcome: nil,
            riskLevel: .destructive,
            riskRationale: "Destructive action",
            requestedAtLabel: "now",
            decidedAtLabel: nil,
            decisionByActorID: nil,
            decisionRationale: nil,
            requestedPhrase: "GO AHEAD",
            taskState: "awaiting_approval",
            runState: "awaiting_approval",
            stepName: "data.delete",
            stepCapabilityKey: "data.delete",
            requestedByActorID: "actor.requester",
            subjectPrincipalActorID: "actor.requester",
            actingAsActorID: "actor.requester",
            taskID: "task-1",
            runID: "run-1",
            stepID: "step-1",
            route: WorkflowRouteContext(taskClass: "chat", provider: "openai", modelKey: "gpt-4.1", routeSource: "policy")
        )

        XCTAssertEqual(
            state.chatInlineApprovalFastPathBlockedReason(for: approval),
            "High-risk approvals require full review in Approvals before submitting."
        )
    }

    func testPerformChatTimelineActionResumeTurnRestoresDraftAndAttemptsSend() {
        let state = AppShellState()
        state.clearLocalDevToken()
        state.selectedSection = .chat
        state.chatLastFailedDraft = "retry with approval"
        state.chatTimelineItems = [
            ChatTimelineItem(
                id: "approval-row-4",
                kind: .approvalDecision,
                state: .completed,
                title: "Approval Decision",
                summary: "Approval was granted.",
                approvalRequestID: "approval-123"
            )
        ]

        state.performChatTimelineAction(itemID: "approval-row-4", intent: .resumeTurn)

        XCTAssertEqual(state.chatDraft, "retry with approval")
        XCTAssertEqual(state.chatStatusMessage, "Set Assistant Access Token before sending chat turns.")
        XCTAssertEqual(
            state.chatTimelineActionStatus(for: "approval-row-4"),
            "Set Assistant Access Token before sending chat turns."
        )
    }

    func testChatTimelineToolFailureAddsConnectorRemediationAction() {
        let state = AppShellState()
        let item = ChatTimelineItem(
            id: "tool-result-1",
            kind: .toolResult,
            state: .failed,
            title: "Tool Result",
            summary: "mail connector permission is missing.",
            content: "Request permission before retrying."
        )

        let actions = state.chatTimelineActions(for: item)

        XCTAssertTrue(actions.contains(where: { $0.intent == .openConnectors }))
    }

    func testPerformChatTimelineActionOpenConnectorsUpdatesStatusAndNavigation() {
        let state = AppShellState()
        state.selectedSection = .chat
        state.chatTimelineItems = [
            ChatTimelineItem(
                id: "tool-result-2",
                kind: .toolResult,
                state: .failed,
                title: "Tool Result",
                summary: "permission denied",
                content: "mail connector requires permission"
            )
        ]

        state.performChatTimelineAction(itemID: "tool-result-2", intent: .openConnectors)

        XCTAssertEqual(state.selectedSection, .connectors)
        XCTAssertEqual(
            state.chatTimelineActionStatus(for: "tool-result-2"),
            "Opened Connectors for permission remediation."
        )
        XCTAssertFalse(state.isChatTimelineActionInFlight(itemID: "tool-result-2"))
    }

    func testChatWorkflowCardSummaryApprovalRequiredPrioritizesDecisionAction() {
        let state = AppShellState()
        state.setInformationDensityMode(.simple)
        defer { state.setInformationDensityMode(.simple) }
        state.chatLatestTurnTraceability = ChatTaskRunTraceabilityItem(
            available: true,
            source: "chat.turn",
            taskID: "task-approval-1",
            runID: "run-approval-1",
            taskState: "awaiting_approval",
            runState: "awaiting_approval",
            correlationID: "corr-approval-1",
            taskClass: "chat",
            provider: "openai",
            modelKey: "gpt-5",
            routeSource: "policy",
            approvalRequired: true,
            approvalRequestID: "approval-123"
        )

        let summary = state.chatWorkflowCardSummary()

        XCTAssertEqual(
            summary.whatHappened,
            "Assistant prepared an action and is waiting for approval."
        )
        XCTAssertTrue(summary.whatNeedsAction.contains("submit Approve or Reject"))
        XCTAssertTrue(summary.whatNext.contains("Open Approvals, submit a decision"))
        XCTAssertFalse(summary.whatNext.contains("approval-123"))

        state.setInformationDensityMode(.advanced)
        let advancedSummary = state.chatWorkflowCardSummary()
        XCTAssertTrue(advancedSummary.whatNext.contains("Open Approvals (approval-123)"))
    }

    func testChatWorkflowCardSummaryFailedToolStepSurfacesRetryRecovery() {
        let state = AppShellState()
        state.setInformationDensityMode(.simple)
        defer { state.setInformationDensityMode(.simple) }
        state.chatLastFailedDraft = "retry the failed run"
        state.chatTimelineItems = [
            ChatTimelineItem(
                id: "tool-result-failed",
                kind: .toolResult,
                state: .failed,
                title: "Tool Result",
                summary: "mail connector permission is missing.",
                content: "Request permission before retrying.",
                runID: "run-failed-1"
            )
        ]

        let summary = state.chatWorkflowCardSummary()

        XCTAssertTrue(summary.whatHappened.contains("Something went wrong"))
        XCTAssertTrue(summary.whatNeedsAction.contains("recovery action"))
        XCTAssertTrue(summary.whatNext.contains("`Retry Turn`"))

        state.setInformationDensityMode(.advanced)
        let advancedSummary = state.chatWorkflowCardSummary()
        XCTAssertTrue(advancedSummary.whatHappened.contains("Latest workflow step failed"))
    }

    func testChatWorkflowCardSummaryClarificationRequiredUsesPromptInNextStep() {
        let state = AppShellState()
        state.chatLatestTurnTraceability = ChatTaskRunTraceabilityItem(
            available: true,
            source: "chat.turn",
            taskID: "task-clarify-1",
            runID: nil,
            taskState: "blocked",
            runState: "blocked",
            correlationID: "corr-clarify-1",
            taskClass: "chat",
            provider: "openai",
            modelKey: "gpt-5",
            routeSource: "policy",
            approvalRequired: false,
            clarificationRequired: true,
            clarificationPrompt: "Which contact should receive this?"
        )

        let summary = state.chatWorkflowCardSummary()

        XCTAssertEqual(
            summary.whatHappened,
            "Assistant needs clarification before continuing the action."
        )
        XCTAssertTrue(summary.whatNeedsAction.contains("missing clarification"))
        XCTAssertTrue(summary.whatNext.contains("Which contact should receive this?"))
    }

    func testChatRealtimeFallbackContextMapsRateLimitToCapacityRejected() {
        let state = AppShellState()

        let context = state._test_chatRealtimeFallbackContext(
            error: DaemonAPIError.serverProblem(
                statusCode: 429,
                message: "realtime websocket capacity exceeded",
                code: "rate_limit_exceeded",
                details: nil,
                correlationID: "corr-cap"
            )
        )

        XCTAssertEqual(context.reason, .capacityRejected)
        XCTAssertTrue(context.statusMessage.contains("Realtime capacity reached"))
        XCTAssertTrue(context.progressDetail.contains("capacity"))
    }

    func testChatRealtimeFallbackContextMapsStaleSessionTransportMessage() {
        let state = AppShellState()

        let context = state._test_chatRealtimeFallbackContext(
            error: DaemonAPIError.transport("realtime_stale_session: heartbeat timeout")
        )

        XCTAssertEqual(context.reason, .staleSession)
        XCTAssertTrue(context.statusMessage.contains("session expired"))
    }

    func testChatRealtimeFallbackContextFromEventMapsUnauthorizedCode() {
        let state = AppShellState()

        let context = state._test_chatRealtimeFallbackContext(
            eventErrorCode: "auth_scope",
            message: "realtime token scope missing",
            defaultReason: .unavailable
        )

        XCTAssertEqual(context.reason, .unauthorized)
        XCTAssertTrue(context.statusMessage.contains("auth"))
    }

    func testChatRealtimeErrorSummaryForDisplayRedactsContextCanceledDetails() {
        let state = AppShellState()

        let summary = state._test_chatRealtimeErrorSummaryForDisplay(
            errorCode: nil,
            message: "begin tx: context canceled"
        )

        XCTAssertEqual(
            summary,
            "Realtime stream was interrupted before daemon receipt completed this turn."
        )
        XCTAssertFalse(summary.localizedCaseInsensitiveContains("begin tx"))
        XCTAssertFalse(summary.localizedCaseInsensitiveContains("context canceled"))
    }

    func testChatRealtimeErrorSummaryForDisplayMapsAuthAndCapacityCodes() {
        let state = AppShellState()

        let authSummary = state._test_chatRealtimeErrorSummaryForDisplay(
            errorCode: "auth_scope",
            message: "token scope missing"
        )
        XCTAssertEqual(
            authSummary,
            "Realtime auth failed before daemon receipt completed this turn."
        )

        let capacitySummary = state._test_chatRealtimeErrorSummaryForDisplay(
            errorCode: "rate_limit_exceeded",
            message: "capacity reached"
        )
        XCTAssertEqual(
            capacitySummary,
            "Realtime capacity was reached before daemon receipt completed this turn."
        )
    }

    func testChatRealtimeTransportConnectedForActiveTurnRequiresNoFallbackReason() {
        let state = AppShellState()

        state._test_setChatRealtimeTracking(connected: true, reason: nil)
        XCTAssertTrue(state._test_chatRealtimeTransportConnectedForActiveTurn())

        state._test_setChatRealtimeTracking(connected: true, reason: .disconnected)
        XCTAssertFalse(state._test_chatRealtimeTransportConnectedForActiveTurn())
    }

    func testRetryChatRealtimeStreamWithoutTokenSetsDeterministicStatus() {
        let state = AppShellState()
        state.clearLocalDevToken()

        state.retryChatRealtimeStream()

        XCTAssertEqual(state.chatStatusMessage, "Set Assistant Access Token before retrying realtime.")
        XCTAssertNil(state.chatProgressDetail)
        XCTAssertFalse(state.isChatRealtimeRetryInFlight)
    }

    func testRecoveredChatTurnSnapshotOrdersItemsByItemIndexForMatchingCorrelation() throws {
        let state = AppShellState()
        let payload = """
        {
          "workspace_id": "ws1",
          "items": [
            {
              "record_id": "record-2",
              "turn_id": "turn-new",
              "workspace_id": "ws1",
              "task_class": "chat",
              "correlation_id": "corr-123",
              "channel_id": "app",
              "item_index": 1,
              "item": { "type": "assistant_message", "content": "done" },
              "task_run_reference": { "task_id": "task-1", "run_id": "run-1", "run_state": "completed" },
              "created_at": "2026-03-03T12:00:02Z"
            },
            {
              "record_id": "record-1",
              "turn_id": "turn-new",
              "workspace_id": "ws1",
              "task_class": "chat",
              "correlation_id": "corr-123",
              "channel_id": "app",
              "item_index": 0,
              "item": { "type": "user_message", "content": "hello" },
              "task_run_reference": { "task_id": "task-1", "run_id": "run-1", "run_state": "completed" },
              "created_at": "2026-03-03T12:00:01Z"
            },
            {
              "record_id": "record-3",
              "turn_id": "turn-old",
              "workspace_id": "ws1",
              "task_class": "chat",
              "correlation_id": "corr-123",
              "channel_id": "app",
              "item_index": 0,
              "item": { "type": "system_status", "content": "old" },
              "task_run_reference": { "task_id": "task-0", "run_id": "run-0", "run_state": "completed" },
              "created_at": "2026-03-03T11:59:00Z"
            }
          ],
          "has_more": false
        }
        """
        let response = try JSONDecoder().decode(
            DaemonChatTurnHistoryResponse.self,
            from: Data(payload.utf8)
        )

        let snapshot = state._test_recoveredChatTurnSnapshot(
            history: response,
            correlationID: "corr-123"
        )

        XCTAssertNotNil(snapshot)
        XCTAssertEqual(snapshot?.correlationID, "corr-123")
        XCTAssertEqual(snapshot?.taskClass, "chat")
        XCTAssertEqual(snapshot?.channelID, "app")
        XCTAssertEqual(snapshot?.itemTypes, ["user_message", "assistant_message"])
    }

    func testRecoveredChatTurnSnapshotReturnsNilWhenCorrelationHasNoMatches() throws {
        let state = AppShellState()
        let payload = """
        {
          "workspace_id": "ws1",
          "items": [
            {
              "record_id": "record-1",
              "turn_id": "turn-1",
              "workspace_id": "ws1",
              "task_class": "chat",
              "correlation_id": "corr-present",
              "channel_id": "app",
              "item_index": 0,
              "item": { "type": "user_message", "content": "hello" },
              "task_run_reference": { "task_id": "task-1", "run_id": "run-1", "run_state": "completed" },
              "created_at": "2026-03-03T12:00:01Z"
            }
          ],
          "has_more": false
        }
        """
        let response = try JSONDecoder().decode(
            DaemonChatTurnHistoryResponse.self,
            from: Data(payload.utf8)
        )

        let snapshot = state._test_recoveredChatTurnSnapshot(
            history: response,
            correlationID: "corr-missing"
        )

        XCTAssertNil(snapshot)
    }
}
