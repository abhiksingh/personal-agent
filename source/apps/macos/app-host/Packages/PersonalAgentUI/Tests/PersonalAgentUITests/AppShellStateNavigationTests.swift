import XCTest
@testable import PersonalAgentUI

@MainActor
final class AppShellStateNavigationTests: XCTestCase {
    private let homeFirstSessionProgressDefaultsKey = "personalagent.ui.home_first_session_progress.v1"

    private func isoTimestamp(_ date: Date) -> String {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        return formatter.string(from: date)
    }

    private func withIsolatedHomeFirstSessionProgressDefaults(_ body: () -> Void) {
        let defaults = appShellStateTestUserDefaults()
        let priorValue = defaults.object(forKey: homeFirstSessionProgressDefaultsKey)
        defer {
            if let priorValue {
                defaults.set(priorValue, forKey: homeFirstSessionProgressDefaultsKey)
            } else {
                defaults.removeObject(forKey: homeFirstSessionProgressDefaultsKey)
            }
        }
        defaults.removeObject(forKey: homeFirstSessionProgressDefaultsKey)
        body()
    }

    func testDefaultSelectedSectionIsHome() {
        let state = AppShellState()

        XCTAssertEqual(state.selectedSection, .home)
    }

    func testNavigateToSectionUpdatesSelection() {
        let state = AppShellState()

        state.navigateToSection(.models)

        XCTAssertEqual(state.selectedSection, .models)
    }

    func testNavigateToSectionRefreshesCurrentSectionWhenAlreadySelected() async {
        let state = AppShellState()
        state.clearLocalDevToken()

        state.navigateToSection(.tasks)
        try? await Task.sleep(for: .milliseconds(80))

        state.tasksStatusMessage = "stale"
        state.navigateToSection(.tasks)

        let expectedMessage = "Set Assistant Access Token to query task/runs."
        for _ in 0..<20 {
            if state.tasksStatusMessage == expectedMessage {
                break
            }
            try? await Task.sleep(for: .milliseconds(20))
        }

        XCTAssertEqual(state.tasksStatusMessage, expectedMessage)
    }

    func testRequestSectionSelectionNavigatesImmediatelyWhenNoUnsavedDrafts() {
        let state = AppShellState()
        state.selectedSection = .channels
        state.channelCards = [editableChannelCard()]
        state.channelConfigDraftByID["app_chat"] = ["mode": "daemon"]

        state.requestSectionSelection(.chat)

        XCTAssertEqual(state.selectedSection, .chat)
        XCTAssertFalse(state.showsUnsavedChangesNavigationAlert)
        XCTAssertNil(state.pendingSectionNavigationSource)
        XCTAssertNil(state.pendingSectionNavigationTarget)
        XCTAssertNil(state.pendingSectionNavigationSummary)
    }

    func testNavigateToSectionWithDrillInContextPreservesContextForDestination() {
        let state = AppShellState()
        state.selectedSection = .inspect

        state.navigateToSection(
            .tasks,
            drillInContext: DrillInNavigationContext(
                sourceSection: .inspect,
                destinationSection: .tasks,
                chips: ["Run: run-123", "Task: task-123"]
            )
        )

        XCTAssertEqual(state.selectedSection, .tasks)
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.sourceSection, .inspect)
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.destinationSection, .tasks)
        XCTAssertEqual(
            state.activeDrillInContextForSelectedSection?.chips,
            ["Run: run-123", "Task: task-123"]
        )
    }

    func testReturnToDrillInSourceSectionNavigatesBackAndClearsContext() {
        let state = AppShellState()
        state.selectedSection = .inspect
        state.navigateToSection(
            .tasks,
            drillInContext: DrillInNavigationContext(
                sourceSection: .inspect,
                destinationSection: .tasks,
                chips: ["Run: run-123"]
            )
        )

        state.returnToDrillInSourceSection()

        XCTAssertEqual(state.selectedSection, .inspect)
        XCTAssertNil(state.activeDrillInNavigationContext)
        XCTAssertNil(state.activeDrillInContextForSelectedSection)
    }

    func testManualSectionSelectionClearsActiveDrillInContext() {
        let state = AppShellState()
        state.selectedSection = .inspect
        state.navigateToSection(
            .tasks,
            drillInContext: DrillInNavigationContext(
                sourceSection: .inspect,
                destinationSection: .tasks,
                chips: ["Run: run-123"]
            )
        )

        state.requestSectionSelection(.chat)

        XCTAssertEqual(state.selectedSection, .chat)
        XCTAssertNil(state.activeDrillInNavigationContext)
    }

    func testRequestSectionSelectionPromptsWhenSourceSectionHasUnsavedDrafts() {
        let state = AppShellState()
        state.selectedSection = .channels
        state.channelCards = [editableChannelCard()]
        state.channelConfigDraftByID["app_chat"] = ["mode": "manual"]

        state.requestSectionSelection(.chat)

        XCTAssertEqual(state.selectedSection, .channels)
        XCTAssertTrue(state.showsUnsavedChangesNavigationAlert)
        XCTAssertEqual(state.pendingSectionNavigationSource, .channels)
        XCTAssertEqual(state.pendingSectionNavigationTarget, .chat)
        XCTAssertEqual(
            state.pendingSectionNavigationSummary,
            "Channels has 1 unsaved draft change(s) across configuration, delivery policy, or connector mappings."
        )
    }

    func testCancelPendingSectionNavigationClearsPromptAndKeepsCurrentSection() {
        let state = AppShellState()
        state.selectedSection = .channels
        state.channelCards = [editableChannelCard()]
        state.channelConfigDraftByID["app_chat"] = ["mode": "manual"]
        state.requestSectionSelection(.chat)

        state.cancelPendingSectionNavigation()

        XCTAssertEqual(state.selectedSection, .channels)
        XCTAssertFalse(state.showsUnsavedChangesNavigationAlert)
        XCTAssertNil(state.pendingSectionNavigationSource)
        XCTAssertNil(state.pendingSectionNavigationTarget)
        XCTAssertNil(state.pendingSectionNavigationSummary)
        XCTAssertTrue(state.channelConfigHasDraftChanges(channelID: "app_chat"))
    }

    func testDiscardPendingSectionNavigationDropsDraftAndNavigatesToTarget() {
        let state = AppShellState()
        state.selectedSection = .channels
        state.channelCards = [editableChannelCard()]
        state.channelConfigDraftByID["app_chat"] = ["mode": "manual"]
        state.requestSectionSelection(.chat)

        state.discardPendingSectionNavigationChanges()

        XCTAssertEqual(state.selectedSection, .chat)
        XCTAssertFalse(state.showsUnsavedChangesNavigationAlert)
        XCTAssertNil(state.pendingSectionNavigationSource)
        XCTAssertNil(state.pendingSectionNavigationTarget)
        XCTAssertNil(state.pendingSectionNavigationSummary)
        XCTAssertFalse(state.channelConfigHasDraftChanges(channelID: "app_chat"))
        XCTAssertEqual(state.channelConfigDraftValue(channelID: "app_chat", key: "mode"), "daemon")
        XCTAssertEqual(state.channelsStatusMessage, "Discarded unsaved channel drafts.")
    }

    func testHomeFirstSessionReviewApprovalsStepMarksCompleteAndPersists() {
        withIsolatedHomeFirstSessionProgressDefaults {
            let first = AppShellState()
            first._test_resetHomeFirstSessionProgress()
            first.selectedSection = .home

            let pendingFirst = first.homeFirstSessionSteps.first { $0.id == .reviewApprovals }
            XCTAssertEqual(pendingFirst?.isComplete, false)

            first.performHomeFirstSessionStep(.reviewApprovals)

            XCTAssertEqual(first.selectedSection, .approvals)
            XCTAssertEqual(
                first.homeFirstSessionSteps.first { $0.id == .reviewApprovals }?.isComplete,
                true
            )

            let second = AppShellState()
            XCTAssertEqual(
                second.homeFirstSessionSteps.first { $0.id == .reviewApprovals }?.isComplete,
                true
            )
        }
    }

    func testHomeFirstSessionStepsExposeExpectedPrimaryWorkflowActions() {
        let state = AppShellState()

        let stepsByID = Dictionary(uniqueKeysWithValues: state.homeFirstSessionSteps.map { ($0.id, $0) })

        XCTAssertEqual(stepsByID[.sendMessage]?.actionTitle, "Open Chat")
        XCTAssertEqual(stepsByID[.createTask]?.actionTitle, "Open Tasks")
        XCTAssertEqual(stepsByID[.reviewApprovals]?.actionTitle, "Open Approvals")
    }

    func testHomeFirstSessionNavigationForChatAndTaskActions() {
        let state = AppShellState()
        state.selectedSection = .home

        state.performHomeFirstSessionStep(.sendMessage)
        XCTAssertEqual(state.selectedSection, .chat)

        state.performHomeFirstSessionStep(.createTask)
        XCTAssertEqual(state.selectedSection, .tasks)
    }

    func testHomeFirstSessionGuidanceContextOnlyAppearsInWorkflowSections() {
        let state = AppShellState()
        state._test_resetHomeFirstSessionProgress()

        state.selectedSection = .home
        XCTAssertNil(state.homeFirstSessionGuidanceContext)

        state.selectedSection = .configuration
        XCTAssertNil(state.homeFirstSessionGuidanceContext)

        markSetupReadinessComplete(state)
        state.selectedSection = .chat
        let context = state.homeFirstSessionGuidanceContext
        XCTAssertEqual(context?.step.id, .sendMessage)
        XCTAssertEqual(context?.progressLabel, "Step 1 of 4")
        XCTAssertEqual(context?.isCurrentSectionDestination, true)
    }

    func testHomeFirstSessionGuidancePrimaryActionNavigatesToNextStepDestination() {
        let state = AppShellState()
        state._test_resetHomeFirstSessionProgress()
        markSetupReadinessComplete(state)
        state.selectedSection = .tasks

        XCTAssertEqual(state.homeFirstSessionGuidanceContext?.step.id, .sendMessage)
        XCTAssertEqual(state.homeFirstSessionGuidanceContext?.isCurrentSectionDestination, false)

        state.performHomeFirstSessionGuidancePrimaryAction()

        XCTAssertEqual(state.selectedSection, .chat)
    }

    func testHomeFirstSessionGuidanceAdvancesAfterCompletionAndRoutesToNextMilestone() {
        let state = AppShellState()
        state._test_resetHomeFirstSessionProgress()
        markSetupReadinessComplete(state)
        state._test_markHomeFirstSessionStepComplete(.sendMessage)
        state.selectedSection = .tasks

        let context = state.homeFirstSessionGuidanceContext
        XCTAssertEqual(context?.step.id, .sendCommunication)
        XCTAssertEqual(context?.progressLabel, "Step 2 of 4")

        state.performHomeFirstSessionGuidancePrimaryAction()

        XCTAssertEqual(state.selectedSection, .communications)
    }

    func testHomeFirstSessionFunnelDiagnosticsCapturesFirstCompletionEvidence() {
        withIsolatedHomeFirstSessionProgressDefaults {
            let state = AppShellState()
            state._test_resetHomeFirstSessionProgress()
            let firstCompletion = Date(timeIntervalSince1970: 1_700_000_000)
            let laterCompletion = Date(timeIntervalSince1970: 1_700_000_100)

            state._test_markHomeFirstSessionStepComplete(
                .sendMessage,
                source: "chat_turn",
                completedAt: firstCompletion
            )
            state._test_markHomeFirstSessionStepComplete(
                .sendMessage,
                source: "task_submit",
                completedAt: laterCompletion
            )

            let diagnostics = state.homeFirstSessionFunnelDiagnostics
            let sendMessageMilestone = diagnostics.milestones.first { $0.id == .sendMessage }

            XCTAssertEqual(sendMessageMilestone?.isComplete, true)
            XCTAssertEqual(sendMessageMilestone?.completionSource, "chat_turn")
            XCTAssertEqual(sendMessageMilestone?.completionSourceLabel, "Chat Turn")
            XCTAssertEqual(sendMessageMilestone?.completedAtRaw, isoTimestamp(firstCompletion))
            XCTAssertEqual(diagnostics.firstCompletedAtRaw, isoTimestamp(firstCompletion))
            XCTAssertEqual(diagnostics.latestCompletedAtRaw, isoTimestamp(firstCompletion))
        }
    }

    func testHomeFirstSessionFunnelDiagnosticsPersistsCompletionEvidence() {
        withIsolatedHomeFirstSessionProgressDefaults {
            let first = AppShellState()
            first._test_resetHomeFirstSessionProgress()
            let completionDate = Date(timeIntervalSince1970: 1_700_000_200)
            first._test_markHomeFirstSessionStepComplete(
                .createTask,
                source: "task_submit",
                completedAt: completionDate
            )

            let second = AppShellState()
            let diagnostics = second.homeFirstSessionFunnelDiagnostics
            let createTaskMilestone = diagnostics.milestones.first { $0.id == .createTask }

            XCTAssertEqual(createTaskMilestone?.completionSource, "task_submit")
            XCTAssertEqual(createTaskMilestone?.completionSourceLabel, "Task Submit")
            XCTAssertEqual(createTaskMilestone?.completedAtRaw, isoTimestamp(completionDate))
        }
    }

    func testHomeFirstSessionFunnelDiagnosticsMapsGuidedChecklistSourceLabel() {
        withIsolatedHomeFirstSessionProgressDefaults {
            let state = AppShellState()
            state._test_resetHomeFirstSessionProgress()
            state._test_markHomeFirstSessionStepComplete(.reviewApprovals, source: "home_checklist")

            let diagnostics = state.homeFirstSessionFunnelDiagnostics
            let approvalsMilestone = diagnostics.milestones.first { $0.id == .reviewApprovals }

            XCTAssertEqual(approvalsMilestone?.completionSource, "home_checklist")
            XCTAssertEqual(approvalsMilestone?.completionSourceLabel, "Guided Checklist")
        }
    }

    private func editableChannelCard() -> ChannelCardItem {
        ChannelCardItem(
            id: "app_chat",
            name: "App Chat",
            status: .active,
            summary: "Ready",
            details: [:],
            editableConfiguration: ["mode": "daemon"],
            editableConfigurationKinds: ["mode": .string],
            readOnlyConfiguration: [:],
            actions: [],
            unavailableActionReason: "n/a"
        )
    }

    private func markSetupReadinessComplete(_ state: AppShellState) {
        state.localDevTokenConfigured = true
        state.daemonStatus = .running
        state.connectionStatus = .connected
        state.providerReadinessItems = [
            ProviderReadinessItem(
                id: "openai",
                provider: "openai",
                endpoint: "https://api.openai.com/v1",
                status: .healthy,
                detail: "Provider is ready.",
                updatedAtLabel: "now"
            )
        ]
        state.modelCatalogItems = [
            ModelCatalogEntryItem(
                id: "openai:gpt-4.1",
                provider: "openai",
                modelKey: "gpt-4.1",
                enabled: true,
                providerReady: true,
                providerEndpoint: "https://api.openai.com/v1"
            )
        ]
        state.modelRouteSummary = ModelRouteSummary(
            provider: "openai",
            modelKey: "gpt-4.1",
            source: "policy",
            notes: nil
        )
        state.channelConnectorMappingsByChannelID = [
            "app": [
                ChannelConnectorMappingItem(
                    channelID: "app",
                    connectorID: "builtin.app",
                    enabled: true,
                    priority: 1,
                    capabilities: ["chat"],
                    createdAtLabel: nil,
                    updatedAtLabel: nil
                )
            ],
            "message": [
                ChannelConnectorMappingItem(
                    channelID: "message",
                    connectorID: "twilio",
                    enabled: true,
                    priority: 1,
                    capabilities: ["send_message"],
                    createdAtLabel: nil,
                    updatedAtLabel: nil
                )
            ],
            "voice": [
                ChannelConnectorMappingItem(
                    channelID: "voice",
                    connectorID: "twilio",
                    enabled: true,
                    priority: 1,
                    capabilities: ["voice_call"],
                    createdAtLabel: nil,
                    updatedAtLabel: nil
                )
            ]
        ]
        state.connectorCards = [
            ConnectorCardItem(
                id: "builtin.app",
                name: "App Connector",
                logicalConnectorID: "builtin.app",
                declaredCapabilities: ["chat"],
                requiresPermission: false,
                health: .ready,
                permissionState: .granted,
                permissionScope: "app",
                statusReason: nil,
                summary: "Ready",
                details: [:],
                editableConfiguration: [:],
                editableConfigurationKinds: [:],
                readOnlyConfiguration: [:],
                actions: [],
                unavailableActionReason: "n/a"
            ),
            ConnectorCardItem(
                id: "twilio",
                name: "Twilio",
                logicalConnectorID: "twilio",
                declaredCapabilities: ["send_message", "voice_call"],
                requiresPermission: false,
                health: .ready,
                permissionState: .granted,
                permissionScope: "message,voice",
                statusReason: nil,
                summary: "Ready",
                details: [:],
                editableConfiguration: [:],
                editableConfigurationKinds: [:],
                readOnlyConfiguration: [:],
                actions: [],
                unavailableActionReason: "n/a"
            )
        ]
    }
}
