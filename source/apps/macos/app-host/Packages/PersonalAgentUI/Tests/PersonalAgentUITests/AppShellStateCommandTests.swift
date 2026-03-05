import XCTest
@testable import PersonalAgentUI

@MainActor
final class AppShellStateCommandTests: XCTestCase {
    private let recentCommandsDefaultsKey = "personalagent.ui.recent_app_commands.v1"
    private let informationDensityDefaultsKey = "personalagent.ui.information_density_mode.v1"

    private func withIsolatedRecentCommandsDefaults(_ body: () -> Void) {
        let defaults = appShellStateTestUserDefaults()
        let priorValue = defaults.object(forKey: recentCommandsDefaultsKey)
        defer {
            if let priorValue {
                defaults.set(priorValue, forKey: recentCommandsDefaultsKey)
            } else {
                defaults.removeObject(forKey: recentCommandsDefaultsKey)
            }
        }
        defaults.removeObject(forKey: recentCommandsDefaultsKey)
        body()
    }

    private func withIsolatedRecentCommandsDefaults(_ body: () async -> Void) async {
        let defaults = appShellStateTestUserDefaults()
        let priorValue = defaults.object(forKey: recentCommandsDefaultsKey)
        defer {
            if let priorValue {
                defaults.set(priorValue, forKey: recentCommandsDefaultsKey)
            } else {
                defaults.removeObject(forKey: recentCommandsDefaultsKey)
            }
        }
        defaults.removeObject(forKey: recentCommandsDefaultsKey)
        await body()
    }

    private func withIsolatedInformationDensityDefaults(_ body: () -> Void) {
        let defaults = appShellStateTestUserDefaults()
        let priorValue = defaults.object(forKey: informationDensityDefaultsKey)
        defer {
            if let priorValue {
                defaults.set(priorValue, forKey: informationDensityDefaultsKey)
            } else {
                defaults.removeObject(forKey: informationDensityDefaultsKey)
            }
        }
        defaults.removeObject(forKey: informationDensityDefaultsKey)
        body()
    }

    func testPresentAndDismissCommandPaletteResetSearchState() {
        let state = AppShellState()
        state.commandPaletteSearchQuery = "inspect"

        state.presentCommandPalette()

        XCTAssertTrue(state.isCommandPalettePresented)
        XCTAssertEqual(state.commandPaletteSearchQuery, "")

        state.commandPaletteSearchQuery = "channels"
        state.dismissCommandPalette()

        XCTAssertFalse(state.isCommandPalettePresented)
        XCTAssertEqual(state.commandPaletteSearchQuery, "")
    }

    func testPresentDoEntryPointPrefillsIntentQuery() {
        let state = AppShellState()
        state.commandPaletteSearchQuery = "chat"

        state.presentDoEntryPoint()

        XCTAssertTrue(state.isCommandPalettePresented)
        XCTAssertEqual(state.commandPaletteSearchQuery, "do ")
    }

    func testCommandActionItemsIncludeCoreWorkflowsAndDiagnostics() {
        let state = AppShellState()
        let actionIDs = Set(state.appCommandActionItems.map(\.actionID))

        XCTAssertTrue(actionIDs.contains(.doSendMessage))
        XCTAssertTrue(actionIDs.contains(.doSendEmail))
        XCTAssertTrue(actionIDs.contains(.doCreateTask))
        XCTAssertTrue(actionIDs.contains(.doReviewApprovals))
        XCTAssertTrue(actionIDs.contains(.doInspectIssue))
        XCTAssertTrue(actionIDs.contains(.openChat))
        XCTAssertTrue(actionIDs.contains(.openTasks))
        XCTAssertTrue(actionIDs.contains(.openInspect))
        XCTAssertTrue(actionIDs.contains(.openChannels))
        XCTAssertTrue(actionIDs.contains(.openConnectors))
        XCTAssertTrue(actionIDs.contains(.refreshCurrentSection))
        XCTAssertTrue(actionIDs.contains(.openNotificationCenter))
        XCTAssertTrue(actionIDs.contains(.setSimpleDensityMode))
        XCTAssertTrue(actionIDs.contains(.setAdvancedDensityMode))
        XCTAssertTrue(actionIDs.contains(.startDaemon))
    }

    func testInformationDensityCommandsToggleModesAndEnablement() {
        withIsolatedRecentCommandsDefaults {
            withIsolatedInformationDensityDefaults {
                let state = AppShellState()

                XCTAssertEqual(state.informationDensityMode, .simple)
                XCTAssertFalse(state.isAppCommandEnabled(.setSimpleDensityMode))
                XCTAssertTrue(state.isAppCommandEnabled(.setAdvancedDensityMode))

                state.performAppCommand(.setAdvancedDensityMode)

                XCTAssertEqual(state.informationDensityMode, .advanced)
                XCTAssertTrue(state.isAppCommandEnabled(.setSimpleDensityMode))
                XCTAssertFalse(state.isAppCommandEnabled(.setAdvancedDensityMode))
                XCTAssertEqual(
                    state.commandDisabledReason(for: .setAdvancedDensityMode),
                    "Information density is already set to Advanced."
                )
            }
        }
    }

    func testPerformNavigationCommandSwitchesSection() {
        withIsolatedRecentCommandsDefaults {
            let state = AppShellState()

            state.performAppCommand(.openConnectors)

            XCTAssertEqual(state.selectedSection, .connectors)
        }
    }

    func testPerformRefreshCurrentSectionCommandRefreshesSelectedPanel() async {
        await withIsolatedRecentCommandsDefaults {
            let state = AppShellState()
            state.clearLocalDevToken()
            state.selectedSection = .tasks
            state.tasksStatusMessage = "stale"

            state.performAppCommand(.refreshCurrentSection)

            let expectedMessage = "Set Assistant Access Token to query task/runs."
            for _ in 0..<20 {
                if state.tasksStatusMessage == expectedMessage {
                    break
                }
                try? await Task.sleep(for: .milliseconds(20))
            }

            XCTAssertEqual(state.tasksStatusMessage, expectedMessage)
        }
    }

    func testRuntimeCommandEnablementFollowsDaemonControls() {
        let state = AppShellState()
        state.daemonCanStart = false
        state.daemonCanStop = true
        state.daemonCanRestart = false
        state.isDaemonControlInFlight = false

        XCTAssertFalse(state.isAppCommandEnabled(.startDaemon))
        XCTAssertTrue(state.isAppCommandEnabled(.stopDaemon))
        XCTAssertFalse(state.isAppCommandEnabled(.restartDaemon))

        state.isDaemonControlInFlight = true

        XCTAssertFalse(state.isAppCommandEnabled(.stopDaemon))
    }

    func testDisabledRuntimeCommandDoesNotQueueConfirmation() {
        withIsolatedRecentCommandsDefaults {
            let state = AppShellState()
            state.daemonCanStart = false

            state.performAppCommand(.startDaemon)

            XCTAssertNil(state.pendingHighImpactActionConfirmation)
        }
    }

    func testEnabledRuntimeCommandQueuesConfirmation() {
        withIsolatedRecentCommandsDefaults {
            let state = AppShellState()
            state.daemonCanStart = true

            state.performAppCommand(.startDaemon)

            XCTAssertNotNil(state.pendingHighImpactActionConfirmation)
        }
    }

    func testFixNextCommandNavigatesToConfigurationWhenTokenMissing() {
        withIsolatedRecentCommandsDefaults {
            let state = AppShellState()
            state.clearLocalDevToken()
            state.selectedSection = .chat

            XCTAssertTrue(state.isAppCommandEnabled(.performOnboardingFixNextStep))

            state.performAppCommand(.performOnboardingFixNextStep)

            XCTAssertEqual(state.selectedSection, .configuration)
        }
    }

    func testDisabledReasonSurfacesDeterministicRuntimeCopy() {
        let state = AppShellState()
        state.daemonCanStart = false
        state.isDaemonControlInFlight = false
        XCTAssertEqual(
            state.commandDisabledReason(for: .startDaemon),
            "Daemon start control is unavailable in the current lifecycle state."
        )

        state.daemonCanStart = true
        state.isDaemonControlInFlight = true
        XCTAssertEqual(
            state.commandDisabledReason(for: .startDaemon),
            "Daemon lifecycle action is already in progress."
        )
    }

    func testCommandActionItemsIncludeDisabledReasonWhenUnavailable() {
        let state = AppShellState()
        state.daemonCanStart = false
        state.isDaemonControlInFlight = false

        guard let startItem = state.appCommandActionItems.first(where: { $0.actionID == .startDaemon }) else {
            XCTFail("Expected Start Daemon command item.")
            return
        }

        XCTAssertFalse(startItem.isEnabled)
        XCTAssertEqual(
            startItem.disabledReason,
            "Daemon start control is unavailable in the current lifecycle state."
        )
    }

    func testRecentCommandUsageMaintainsMostRecentFirstUniquenessAndLimit() {
        withIsolatedRecentCommandsDefaults {
            let state = AppShellState()
            let commandSequence: [AppCommandActionID] = [
                .openConfiguration,
                .openChat,
                .openCommunications,
                .openAutomation,
                .openApprovals,
                .openTasks,
                .openModels,
                .openInspect,
                .openChannels,
                .openConnectors,
                .refreshCurrentSection,
                .openNotificationCenter,
                .openChat
            ]

            for command in commandSequence {
                state.performAppCommand(command)
            }

            XCTAssertEqual(state.recentAppCommandActionIDs.first, .openChat)
            XCTAssertEqual(state.recentAppCommandActionIDs.count, 8)
            XCTAssertEqual(state.recentAppCommandActionIDs.filter { $0 == .openChat }.count, 1)
        }
    }

    func testRecentCommandOrderingPrioritizesRecentlyUsedItemsInCatalog() {
        withIsolatedRecentCommandsDefaults {
            let state = AppShellState()
            state.performAppCommand(.openInspect)
            state.performAppCommand(.openChat)

            let items = state.appCommandActionItems
            XCTAssertGreaterThanOrEqual(items.count, 2)
            XCTAssertEqual(items[0].actionID, .openChat)
            XCTAssertEqual(items[1].actionID, .openInspect)
        }
    }

    func testIntentRankingPrioritizesNaturalLanguageQueryMatches() {
        withIsolatedRecentCommandsDefaults {
            let state = AppShellState()
            state.daemonCanStart = true
            state.isDaemonControlInFlight = false

            let ranked = state.rankedAppCommandActionItems(for: "start service")
            XCTAssertEqual(ranked.first?.actionID, .startDaemon)
            XCTAssertTrue(ranked.first?.isEnabled ?? false)
        }
    }

    func testIntentRankingPrioritizesDoOutcomeForSendEmailQuery() {
        withIsolatedRecentCommandsDefaults {
            let state = AppShellState()
            let ranked = state.rankedAppCommandActionItems(for: "send email to finance")

            XCTAssertEqual(ranked.first?.actionID, .doSendEmail)
            XCTAssertTrue(ranked.first?.isEnabled ?? false)
        }
    }

    func testIntentRankingTieBreaksDeterministicallyUsingCatalogOrder() {
        withIsolatedRecentCommandsDefaults {
            let state = AppShellState()

            let ranked = state.rankedAppCommandActionItems(for: "open")
            XCTAssertGreaterThanOrEqual(ranked.count, 4)
            XCTAssertEqual(ranked[0].actionID, .openApprovals)
            XCTAssertEqual(ranked[1].actionID, .openAutomation)
            XCTAssertEqual(ranked[2].actionID, .openChat)
        }
    }

    func testFirstEnabledMatchSkipsDisabledTopRuntimeResult() {
        withIsolatedRecentCommandsDefaults {
            let state = AppShellState()
            state.daemonCanStart = true
            state.daemonCanStop = false
            state.daemonCanRestart = true
            state.isDaemonControlInFlight = false

            let ranked = state.rankedAppCommandActionItems(for: "stop daemon service")
            XCTAssertEqual(ranked.first?.actionID, .stopDaemon)
            XCTAssertFalse(ranked.first?.isEnabled ?? true)

            let firstEnabled = state.firstEnabledAppCommandAction(for: "stop daemon service")
            let expectedFirstEnabled = ranked.first(where: \.isEnabled)
            XCTAssertEqual(firstEnabled?.actionID, expectedFirstEnabled?.actionID)
            XCTAssertNotEqual(firstEnabled?.actionID, ranked.first?.actionID)
            XCTAssertTrue(firstEnabled?.isEnabled ?? false)
        }
    }

    func testPerformDoSendEmailSeedsChatDraftAndNavigates() {
        withIsolatedRecentCommandsDefaults {
            let state = AppShellState()
            state.selectedSection = .home
            state.chatDraft = ""

            state.performAppCommand(.doSendEmail)

            XCTAssertEqual(state.selectedSection, .chat)
            XCTAssertEqual(state.chatDraft, "Draft and send an email to ")
            XCTAssertEqual(
                state.chatStatusMessage,
                "Do: Send an Email ready. Enter recipient and intent, then send."
            )
        }
    }

    func testPerformDoSendEmailPreservesExistingDraft() {
        withIsolatedRecentCommandsDefaults {
            let state = AppShellState()
            state.chatDraft = "Existing draft"

            state.performAppCommand(.doSendEmail)

            XCTAssertEqual(state.chatDraft, "Existing draft")
        }
    }

    func testPerformDoCreateTaskNavigatesAndSetsTaskStatus() {
        withIsolatedRecentCommandsDefaults {
            let state = AppShellState()
            state.selectedSection = .chat

            state.performAppCommand(.doCreateTask)

            XCTAssertEqual(state.selectedSection, .tasks)
            XCTAssertEqual(
                state.tasksStatusMessage,
                "Do: Create a Task ready. Use New Task to draft and submit."
            )
        }
    }

    func testRecentCommandsPersistAcrossStateReinitialization() {
        withIsolatedRecentCommandsDefaults {
            let first = AppShellState()
            first.performAppCommand(.openInspect)
            first.performAppCommand(.openChannels)

            let second = AppShellState()

            XCTAssertEqual(
                Array(second.recentAppCommandActionIDs.prefix(2)),
                [.openChannels, .openInspect]
            )
        }
    }

    func testObjectSearchRanksTaskRunByRunIDDeterministically() {
        let state = AppShellState()
        state.taskRunItems = [
            makeTaskRow(
                id: "task-1::run-object-1",
                taskID: "task-1",
                runID: "run-object-1",
                title: "Send update email"
            )
        ]
        state.approvalInboxItems = [
            makeApprovalItem(id: "approval-1", taskTitle: "Send update email")
        ]

        let ranked = state.rankedCommandPaletteObjectItems(for: "run-object-1")

        XCTAssertFalse(ranked.isEmpty)
        XCTAssertEqual(ranked.first?.kind, .taskRun)
        XCTAssertEqual(ranked.first?.target, .taskRun(rowID: "task-1::run-object-1"))
    }

    func testPerformCommandPaletteObjectActionTaskRunSeedsTaskSearchAndNavigates() {
        let state = AppShellState()
        state.selectedSection = .chat
        state.taskRunItems = [
            makeTaskRow(
                id: "task-1::run-open-1",
                taskID: "task-1",
                runID: "run-open-1",
                title: "Open payroll report"
            )
        ]

        state.performCommandPaletteObjectAction(.taskRun(rowID: "task-1::run-open-1"))

        XCTAssertEqual(state.selectedSection, .tasks)
        XCTAssertEqual(state.tasksSearchSeed, "run-open-1")
        XCTAssertEqual(state.tasksStatusMessage, "Opened task result for run-open-1.")
    }

    func testPerformCommandPaletteObjectActionThreadSeedsThreadFilterAndNavigates() {
        let state = AppShellState()
        state.selectedSection = .chat
        state.communicationThreads = [
            makeThread(id: "thread-object-1", title: "Family Group")
        ]

        state.performCommandPaletteObjectAction(.thread(threadID: "thread-object-1"))

        XCTAssertEqual(state.selectedSection, .communications)
        XCTAssertEqual(state.communicationsFilterContext().threadFilterID, "thread-object-1")
        XCTAssertEqual(state.communicationsFilterContext().searchText, "thread-object-1")
        XCTAssertEqual(state.communicationsStatusMessage, "Opened thread thread-object-1.")
    }

    func testPerformCommandPaletteObjectActionModelNavigatesAndSetsStatus() {
        let state = AppShellState()
        state.selectedSection = .chat

        state.performCommandPaletteObjectAction(.model(providerID: "openai", modelKey: "gpt-5"))

        XCTAssertEqual(state.selectedSection, .models)
        XCTAssertEqual(state.modelCatalogStatusMessage, "Opened model openai/gpt-5.")
    }

    private func makeTaskRow(
        id: String,
        taskID: String,
        runID: String?,
        title: String
    ) -> TaskRunListRowItem {
        TaskRunListRowItem(
            id: id,
            title: title,
            taskID: taskID,
            runID: runID,
            taskState: "running",
            runState: "running",
            effectiveState: .running,
            priority: 2,
            priorityLabel: "Priority Medium",
            requestedByActorID: "default",
            subjectPrincipalActorID: "default",
            actingAsActorID: "default",
            taskCreatedAtLabel: "now",
            taskUpdatedAtLabel: "now",
            runCreatedAtLabel: "now",
            runUpdatedAtLabel: "now",
            startedAtLabel: "now",
            finishedAtLabel: nil,
            lastError: nil,
            actions: .unavailable,
            sortTimestamp: .now,
            route: WorkflowRouteContext(
                available: true,
                taskClass: "chat",
                provider: "openai",
                modelKey: "gpt-5",
                taskClassSource: "policy",
                routeSource: "policy"
            )
        )
    }

    private func makeApprovalItem(id: String, taskTitle: String) -> ApprovalInboxItem {
        ApprovalInboxItem(
            id: id,
            taskTitle: taskTitle,
            decisionState: .pending,
            decisionOutcome: nil,
            riskLevel: .policy,
            riskRationale: "Policy review required.",
            requestedAtLabel: "now",
            decidedAtLabel: nil,
            decisionByActorID: nil,
            decisionRationale: nil,
            requestedPhrase: nil,
            taskState: "awaiting_approval",
            runState: "awaiting_approval",
            stepName: "Review",
            stepCapabilityKey: nil,
            requestedByActorID: "default",
            subjectPrincipalActorID: "default",
            actingAsActorID: "default",
            taskID: "task-approval-1",
            runID: "run-object-1",
            stepID: "step-1",
            route: WorkflowRouteContext(
                available: true,
                taskClass: "approval",
                provider: "openai",
                modelKey: "gpt-5",
                taskClassSource: "policy",
                routeSource: "policy"
            )
        )
    }

    private func makeThread(id: String, title: String) -> CommunicationThreadItem {
        CommunicationThreadItem(
            id: id,
            workspaceID: "default",
            channel: "message",
            connectorID: "imessage",
            title: title,
            externalRef: nil,
            lastEventID: "event-1",
            lastEventType: "message",
            lastDirection: "inbound",
            lastOccurredAtLabel: "now",
            lastBodyPreview: "pizza tonight?",
            participantAddresses: ["+15551234567"],
            eventCount: 3,
            createdAtLabel: "now",
            updatedAtLabel: "now",
            sortTimestamp: .now
        )
    }
}
