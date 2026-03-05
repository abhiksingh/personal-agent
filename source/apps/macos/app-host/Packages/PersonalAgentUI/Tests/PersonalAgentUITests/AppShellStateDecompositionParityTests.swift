import XCTest
@testable import PersonalAgentUI

@MainActor
final class AppShellStateDecompositionParityTests: XCTestCase {
    private let channelsUnsavedDraftSummary =
        "Channels has 1 unsaved draft change(s) across configuration, delivery policy, or connector mappings."

    func testSelectionParityRequestSectionSelectionMatchesNavigationStoreContract() {
        let expected = AppShellNavigationStore()
        expected.selectedSection = .channels
        _ = expected.requestSectionSelection(
            .connectors,
            preservingDrillInContext: false,
            hasUnsavedDraftChanges: true,
            unsavedDraftSummary: channelsUnsavedDraftSummary
        )

        let state = AppShellState()
        state.selectedSection = .channels
        state.channelCards = [editableChannelCard()]
        state.channelConfigDraftByID["app_chat"] = ["mode": "manual"]

        state.requestSectionSelection(.connectors)

        assertNavigationParity(state: state, expected: expected)
    }

    func testSelectionParityDiscardPendingNavigationMatchesNavigationStoreOutcome() {
        let expected = AppShellNavigationStore()
        expected.selectedSection = .channels
        _ = expected.requestSectionSelection(
            .chat,
            preservingDrillInContext: false,
            hasUnsavedDraftChanges: true,
            unsavedDraftSummary: channelsUnsavedDraftSummary
        )
        _ = expected.discardPendingSectionNavigationAndSelectTarget()

        let state = AppShellState()
        state.selectedSection = .channels
        state.channelCards = [editableChannelCard()]
        state.channelConfigDraftByID["app_chat"] = ["mode": "manual"]
        state.requestSectionSelection(.chat)

        state.discardPendingSectionNavigationChanges()

        assertNavigationParity(state: state, expected: expected)
        XCTAssertFalse(state.channelConfigHasDraftChanges(channelID: "app_chat"))
    }

    func testReducerParityMarkDaemonMissingMatchesRuntimeLifecycleStoreProjection() {
        let expected = AppRuntimeLifecycleStore()
        expected.markDaemonMissing()

        let state = AppShellState()
        state.markDaemonMissing()

        assertRuntimeLifecycleParity(state: state, expected: expected)
    }

    func testReducerParityMarkDaemonBrokenMatchesRuntimeLifecycleStoreProjection() {
        let expected = AppRuntimeLifecycleStore()
        expected.markDaemonBroken()

        let state = AppShellState()
        state.markDaemonBroken()

        assertRuntimeLifecycleParity(state: state, expected: expected)
    }

    func testReducerParityPanelProblemMappingMatchesStoreContract() {
        let error = DaemonAPIError.serverProblem(
            statusCode: 429,
            message: "too many requests",
            code: "rate_limit_exceeded",
            details: DaemonProblemDetails(
                category: "rate_limit_exceeded",
                domain: "limits",
                service: nil,
                remediation: DaemonProblemRemediation(
                    action: "retry_later",
                    label: "Retry later",
                    hint: "Wait 30 seconds before retry."
                )
            ),
            correlationID: "corr-rate-limit-parity"
        )

        let expectedStore = AppPanelProblemStore()
        let expectedMessage = expectedStore.typedRemediationMessage(
            daemonError: error,
            section: .tasks,
            sectionTitle: "Tasks"
        )
        let expectedRemediation = expectedStore.remediationContext(
            for: .tasks,
            retryInFlight: true
        )

        let state = AppShellState()
        state.isTasksLoading = true
        let actualMessage = state.panelErrorMessageForTesting(error, panelContext: .tasks)
        let actualRemediation = state.panelProblemRemediation(for: .tasks)

        XCTAssertEqual(actualMessage, expectedMessage)
        assertPanelRemediationParity(actualRemediation, expectedRemediation)
    }

    func testReducerParityPanelLatencyProjectionMatchesStoreContract() {
        let expected = AppPanelLatencyStore()
        let capturedAt = Date(timeIntervalSince1970: 120)

        expected.recordPanelLatencySample(
            section: .tasks,
            category: .refresh,
            durationMS: 925,
            capturedAt: capturedAt
        )

        let state = AppShellState()
        state._test_recordPanelLatencySample(
            section: .tasks,
            category: .refresh,
            durationMS: 925,
            capturedAt: capturedAt
        )

        XCTAssertEqual(state.panelLatencySampleCount, expected.panelLatencySampleCount)
        XCTAssertEqual(state.panelLatencyStatusMessage, expected.panelLatencyStatusMessage)
        XCTAssertEqual(
            normalizedPanelLatencySamples(state.panelLatencyRegressionSamples),
            normalizedPanelLatencySamples(expected.panelLatencyRegressionSamples)
        )
        XCTAssertEqual(
            normalizedPanelLatencySamples(state.panelLatencyLatestSamplesSorted),
            normalizedPanelLatencySamples(expected.panelLatencyLatestSamplesSorted)
        )
        XCTAssertEqual(
            normalizedPanelLatencySamples(state.panelLatencySamples),
            normalizedPanelLatencySamples(expected.panelLatencySamples)
        )
        XCTAssertEqual(
            normalizedPanelLatencyBySection(state.panelLatencyLatestBySectionID),
            normalizedPanelLatencyBySection(expected.panelLatencyLatestBySectionID)
        )

        state.clearPanelLatencySamples()
        expected.clearPanelLatencySamples()

        XCTAssertEqual(state.panelLatencySampleCount, expected.panelLatencySampleCount)
        XCTAssertEqual(state.panelLatencyStatusMessage, expected.panelLatencyStatusMessage)
        XCTAssertEqual(
            normalizedPanelLatencySamples(state.panelLatencySamples),
            normalizedPanelLatencySamples(expected.panelLatencySamples)
        )
        XCTAssertEqual(
            normalizedPanelLatencyBySection(state.panelLatencyLatestBySectionID),
            normalizedPanelLatencyBySection(expected.panelLatencyLatestBySectionID)
        )
    }

    func testReducerParityNotificationProjectionMatchesStoreContract() {
        let defaults = appShellStateTestUserDefaults()
        let defaultsKey = "personalagent.ui.notifications.u270.parity"
        defaults.removeObject(forKey: defaultsKey)
        let expected = AppNotificationCenterStore(
            userDefaults: defaults,
            defaultsKey: defaultsKey,
            defaultWorkspaceID: "ws1",
            notificationHistoryLimit: 250
        )
        expected.clearAllNotifications()
        expected.postNotification(
            workspaceID: "ws1",
            source: "tasks",
            action: "status_update",
            message: "Loaded 2 task run rows.",
            level: .success
        )

        let state = AppShellState()
        state.clearAllNotifications()
        state.postNotification(
            source: "tasks",
            action: "status_update",
            message: "Loaded 2 task run rows.",
            level: .success
        )

        XCTAssertEqual(state.notificationItems.count, expected.notificationItems.count)
        XCTAssertEqual(state.notificationToastItems.count, expected.notificationToastItems.count)
        XCTAssertEqual(state.unreadNotificationCount, expected.unreadNotificationCount())
        XCTAssertEqual(state.successNotificationPulse(for: "tasks"), expected.successNotificationPulse(for: "tasks"))

        if let notificationID = state.notificationItems.first?.id {
            state.markNotificationRead(notificationID: notificationID)
        }
        if let expectedID = expected.notificationItems.first?.id {
            expected.markNotificationRead(notificationID: expectedID)
        }

        XCTAssertEqual(state.unreadNotificationCount, expected.unreadNotificationCount())
    }

    func testReducerParityCommunicationsThreadSelectionMatchesStoreContract() {
        let expectedStore = AppCommunicationsStore()
        let currentContext = CommunicationsFilterContext(
            searchText: "previous",
            channelFilterID: "voice",
            directionFilterRawValue: "outbound",
            threadFilterID: "thread-old",
            compactScanModeEnabled: true
        )
        let expectedContext = expectedStore.threadSelectionFilterContext(
            threadID: "thread-123",
            currentContext: currentContext
        )

        let state = AppShellState()
        state.updateCommunicationsFilterContext(currentContext)

        state.performCommandPaletteObjectAction(.thread(threadID: "thread-123"))

        XCTAssertEqual(state.communicationsFilterContext(), expectedContext)
        XCTAssertEqual(state.communicationsStatusMessage, "Opened thread thread-123.")
        XCTAssertEqual(state.selectedSection, .communications)
    }

    func testReducerParityConnectionConfigReorderMatchesStoreContract() {
        let initialMappings: [ChannelConnectorMappingItem] = [
            makeMapping(channelID: "app", connectorID: "mail", priority: 1),
            makeMapping(channelID: "app", connectorID: "finder", priority: 2),
        ]

        let expected = AppConnectionConfigStore()
        expected.channelConnectorMappingDraftByChannelID["app"] = initialMappings
        expected.reorderChannelConnectorMapping(
            channelID: "app",
            connectorID: "mail",
            direction: 1,
            normalizeChannelID: { $0 },
            normalizeConnectorID: { $0 },
            sortedMappings: { $0.sorted { $0.priority < $1.priority } },
            rebalanceMappings: { mappings in
                mappings.enumerated().map { index, item in
                    var mutable = item
                    mutable.priority = index + 1
                    return mutable
                }
            },
            connectorDisplayName: { $0.capitalized }
        )

        let state = AppShellState()
        state.channelConnectorMappingDraftByChannelID["app"] = initialMappings
        state.moveChannelConnectorMappingDown(channelID: "app", connectorID: "mail")

        XCTAssertEqual(
            state.channelConnectorMappingDraftByChannelID["app"]?.map(\.connectorID),
            expected.channelConnectorMappingDraftByChannelID["app"]?.map(\.connectorID)
        )
        XCTAssertEqual(
            state.channelConnectorMappingDraftByChannelID["app"]?.map(\.priority),
            expected.channelConnectorMappingDraftByChannelID["app"]?.map(\.priority)
        )
        XCTAssertEqual(
            state.channelConnectorMappingActionStatusByChannelID["app"],
            expected.channelConnectorMappingActionStatusByChannelID["app"]
        )
    }

    func testReducerParityModelRouteResetOutputsMatchesStoreContract() {
        let expected = AppModelsRouteStore()
        expected.modelRouteSimulationResult = ModelRouteSimulationResultItem(
            workspaceID: "ws1",
            taskClass: "chat",
            principalActorID: "default",
            selectedProvider: "openai",
            selectedModelKey: "gpt-4.1",
            selectedSource: "workspace_policy",
            notes: nil,
            reasonCodes: [],
            decisions: [],
            fallbackChain: []
        )
        expected.modelRouteExplainResult = ModelRouteExplainResultItem(
            workspaceID: "ws1",
            taskClass: "chat",
            principalActorID: "default",
            selectedProvider: "openai",
            selectedModelKey: "gpt-4.1",
            selectedSource: "workspace_policy",
            summary: "summary",
            explanations: [],
            reasonCodes: [],
            decisions: [],
            fallbackChain: []
        )
        expected.modelRouteSimulationStatusMessage = "stale"
        expected.modelRouteExplainStatusMessage = "stale"
        expected.resetModelRouteSimulationOutputs()

        let state = AppShellState()
        state.modelRouteSimulationResult = ModelRouteSimulationResultItem(
            workspaceID: "ws1",
            taskClass: "chat",
            principalActorID: "default",
            selectedProvider: "openai",
            selectedModelKey: "gpt-4.1",
            selectedSource: "workspace_policy",
            notes: nil,
            reasonCodes: [],
            decisions: [],
            fallbackChain: []
        )
        state.modelRouteExplainResult = ModelRouteExplainResultItem(
            workspaceID: "ws1",
            taskClass: "chat",
            principalActorID: "default",
            selectedProvider: "openai",
            selectedModelKey: "gpt-4.1",
            selectedSource: "workspace_policy",
            summary: "summary",
            explanations: [],
            reasonCodes: [],
            decisions: [],
            fallbackChain: []
        )
        state.modelRouteSimulationStatusMessage = "stale"
        state.modelRouteExplainStatusMessage = "stale"
        state.resetModelRouteSimulationOutputs()

        XCTAssertEqual(state.modelRouteSimulationResult?.selectedModelKey, expected.modelRouteSimulationResult?.selectedModelKey)
        XCTAssertEqual(state.modelRouteExplainResult?.summary, expected.modelRouteExplainResult?.summary)
        XCTAssertEqual(state.modelRouteSimulationStatusMessage, expected.modelRouteSimulationStatusMessage)
        XCTAssertEqual(state.modelRouteExplainStatusMessage, expected.modelRouteExplainStatusMessage)
    }

    func testReducerParityTaskRunControlInFlightProjectionMatchesWorkflowQueueStoreContract() {
        let expected = AppWorkflowQueueStore()
        expected.beginTaskRunControl(runID: "run-123", inFlightMessage: "Retrying run run-123…")

        let state = AppShellState()
        state.taskRunControlInFlightRunIDs = expected.taskRunControlInFlightRunIDs
        state.taskRunControlStatusByRunID = expected.taskRunControlStatusByRunID

        XCTAssertTrue(state.isTaskRunControlInFlight(runID: "run-123"))
        XCTAssertEqual(state.taskRunControlStatus(runID: "run-123"), expected.taskRunControlStatusByRunID["run-123"])
        XCTAssertEqual(
            state.taskRunControlDisabledReason(
                .retry,
                runID: "run-123",
                actions: TaskRunActionAvailabilityItem(canCancel: true, canRetry: true, canRequeue: true)
            ),
            "Another run control action is already in progress for this run."
        )
    }

    func testReducerParityClearInspectRunFocusMatchesInspectStoreContract() {
        let expected = AppInspectStore()
        expected.inspectFocusedRunID = "run-123"
        expected.inspectLogs = [
            InspectLogItem(
                id: "log-1",
                timestamp: Date(timeIntervalSince1970: 100),
                createdAtRaw: "2026-03-05T10:00:00Z",
                event: "task.step",
                status: .running,
                inputSummary: "in",
                outputSummary: "out",
                metadataSummary: "meta"
            )
        ]
        _ = expected.clearInspectRunFocus()

        let state = AppShellState()
        state.inspectFocusedRunID = "run-123"
        state.inspectLogs = [
            InspectLogItem(
                id: "log-1",
                timestamp: Date(timeIntervalSince1970: 100),
                createdAtRaw: "2026-03-05T10:00:00Z",
                event: "task.step",
                status: .running,
                inputSummary: "in",
                outputSummary: "out",
                metadataSummary: "meta"
            )
        ]
        state.clearInspectRunFocus()

        XCTAssertEqual(state.inspectFocusedRunID, expected.inspectFocusedRunID)
        XCTAssertEqual(state.inspectLogs.map(\.id), expected.inspectLogs.map(\.id))
        XCTAssertEqual(state.inspectStatusMessage, expected.inspectStatusMessage)
    }

    func testEventParityRecoveredChatTurnSnapshotMatchesExecutionStoreProjection() throws {
        let payload = """
        {
          "workspace_id": "ws1",
          "items": [
            {
              "record_id": "record-old-1",
              "turn_id": "turn-old",
              "workspace_id": "ws1",
              "task_class": "chat",
              "correlation_id": "corr-123",
              "channel_id": "app",
              "item_index": 0,
              "item": { "type": "assistant_message", "content": "old" },
              "task_run_reference": { "task_id": "task-old", "run_id": "run-old", "run_state": "completed" },
              "created_at": "2026-03-04T10:00:00Z"
            },
            {
              "record_id": "record-new-2",
              "turn_id": "turn-new",
              "workspace_id": "ws1",
              "task_class": "chat",
              "correlation_id": "corr-123",
              "channel_id": "app",
              "item_index": 1,
              "item": { "type": "assistant_message", "content": "done" },
              "task_run_reference": { "task_id": "task-new", "run_id": "run-new", "run_state": "completed" },
              "created_at": "2026-03-04T11:00:02Z"
            },
            {
              "record_id": "record-new-1",
              "turn_id": "turn-new",
              "workspace_id": "ws1",
              "task_class": "chat",
              "correlation_id": "corr-123",
              "channel_id": "app",
              "item_index": 0,
              "item": { "type": "user_message", "content": "hello" },
              "task_run_reference": { "task_id": "task-new", "run_id": "run-new", "run_state": "completed" },
              "created_at": "2026-03-04T11:00:01Z"
            }
          ],
          "has_more": false
        }
        """
        let response = try JSONDecoder().decode(
            DaemonChatTurnHistoryResponse.self,
            from: Data(payload.utf8)
        )

        let formatter = ISO8601DateFormatter()
        let expected = ChatTurnExecutionStore.recoveredChatTurnSnapshot(
            from: response,
            correlationID: "corr-123",
            parseDaemonTimestamp: { formatter.date(from: $0) }
        )
        let state = AppShellState()
        let actual = state._test_recoveredChatTurnSnapshot(
            history: response,
            correlationID: "corr-123"
        )

        XCTAssertEqual(actual?.correlationID, expected?.correlationID)
        XCTAssertEqual(actual?.taskClass, expected?.taskClass)
        XCTAssertEqual(actual?.channelID, expected?.channelID)
        XCTAssertEqual(actual?.itemTypes, expected?.items.map(\.type))
    }

    private func assertNavigationParity(state: AppShellState, expected: AppShellNavigationStore) {
        XCTAssertEqual(state.selectedSection, expected.selectedSection)
        XCTAssertEqual(state.pendingSectionNavigationSource, expected.pendingSectionNavigationSource)
        XCTAssertEqual(state.pendingSectionNavigationTarget, expected.pendingSectionNavigationTarget)
        XCTAssertEqual(state.pendingSectionNavigationSummary, expected.pendingSectionNavigationSummary)
        XCTAssertEqual(state.showsUnsavedChangesNavigationAlert, expected.showsUnsavedChangesNavigationAlert)
    }

    private func assertRuntimeLifecycleParity(
        state: AppShellState,
        expected: AppRuntimeLifecycleStore
    ) {
        XCTAssertEqual(state.daemonStatus, expected.daemonStatus)
        XCTAssertEqual(state.connectionStatus, expected.connectionStatus)
        XCTAssertEqual(state.daemonCanStart, expected.daemonCanStart)
        XCTAssertEqual(state.daemonCanStop, expected.daemonCanStop)
        XCTAssertEqual(state.daemonCanRestart, expected.daemonCanRestart)
        XCTAssertEqual(state.daemonCanInstall, expected.daemonCanInstall)
        XCTAssertEqual(state.daemonCanUninstall, expected.daemonCanUninstall)
        XCTAssertEqual(state.daemonCanRepair, expected.daemonCanRepair)
        XCTAssertEqual(state.daemonNeedsInstall, expected.daemonNeedsInstall)
        XCTAssertEqual(state.daemonNeedsRepair, expected.daemonNeedsRepair)
        XCTAssertEqual(state.daemonSetupState, expected.daemonSetupState)
        XCTAssertEqual(state.daemonStatusDetail, expected.daemonStatusDetail)
        XCTAssertEqual(state.daemonLifecycleOverallState, expected.daemonLifecycleOverallState)
        XCTAssertEqual(state.daemonCoreRuntimeState, expected.daemonCoreRuntimeState)
        XCTAssertEqual(state.daemonPluginRuntimeState, expected.daemonPluginRuntimeState)
        XCTAssertEqual(state.daemonLifecycleBlocking, expected.daemonLifecycleBlocking)
        XCTAssertEqual(state.daemonControlAuthState, expected.daemonControlAuthState)
        XCTAssertEqual(state.daemonControlAuthSource, expected.daemonControlAuthSource)
        XCTAssertEqual(state.daemonControlAuthRemediationHints, expected.daemonControlAuthRemediationHints)
        XCTAssertEqual(state.hasLoadedDaemonStatus, expected.hasLoadedDaemonStatus)
    }

    private func assertPanelRemediationParity(
        _ actual: PanelProblemRemediationContext?,
        _ expected: PanelProblemRemediationContext?
    ) {
        XCTAssertEqual(actual, expected)
    }

    private func normalizedPanelLatencySamples(_ samples: [UIPanelLatencySample]) -> [UIPanelLatencySample] {
        samples.map(normalizedPanelLatencySample)
    }

    private func normalizedPanelLatencyBySection(
        _ samples: [String: UIPanelLatencySample]
    ) -> [String: UIPanelLatencySample] {
        samples.mapValues(normalizedPanelLatencySample)
    }

    private func normalizedPanelLatencySample(_ sample: UIPanelLatencySample) -> UIPanelLatencySample {
        UIPanelLatencySample(
            id: "parity",
            sectionID: sample.sectionID,
            category: sample.category,
            durationMS: sample.durationMS,
            budgetMS: sample.budgetMS,
            isOverBudget: sample.isOverBudget,
            capturedAt: sample.capturedAt
        )
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

    private func makeMapping(channelID: String, connectorID: String, priority: Int) -> ChannelConnectorMappingItem {
        ChannelConnectorMappingItem(
            channelID: channelID,
            connectorID: connectorID,
            enabled: true,
            priority: priority,
            capabilities: [],
            createdAtLabel: nil,
            updatedAtLabel: nil
        )
    }
}
