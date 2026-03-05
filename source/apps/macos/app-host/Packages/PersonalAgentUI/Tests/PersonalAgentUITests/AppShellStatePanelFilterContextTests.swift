import XCTest
@testable import PersonalAgentUI

@MainActor
final class AppShellStatePanelFilterContextTests: XCTestCase {
    private let panelFilterDefaultsKey = "personalagent.ui.panel_filter_context.v1"
    private let communicationsTriageDefaultsKey = "personalagent.ui.communications_triage_context.v1"
    private let workspaceContinuityDefaultsKey = "personalagent.ui.workspace_continuity_context.v1"

    private func withIsolatedPanelFilterDefaults(_ body: () -> Void) {
        let defaults = appShellStateTestUserDefaults()
        let priorValue = defaults.object(forKey: panelFilterDefaultsKey)
        let priorTriageValue = defaults.object(forKey: communicationsTriageDefaultsKey)
        let priorContinuityValue = defaults.object(forKey: workspaceContinuityDefaultsKey)
        defer {
            if let priorValue {
                defaults.set(priorValue, forKey: panelFilterDefaultsKey)
            } else {
                defaults.removeObject(forKey: panelFilterDefaultsKey)
            }
            if let priorTriageValue {
                defaults.set(priorTriageValue, forKey: communicationsTriageDefaultsKey)
            } else {
                defaults.removeObject(forKey: communicationsTriageDefaultsKey)
            }
            if let priorContinuityValue {
                defaults.set(priorContinuityValue, forKey: workspaceContinuityDefaultsKey)
            } else {
                defaults.removeObject(forKey: workspaceContinuityDefaultsKey)
            }
        }
        defaults.removeObject(forKey: panelFilterDefaultsKey)
        defaults.removeObject(forKey: communicationsTriageDefaultsKey)
        defaults.removeObject(forKey: workspaceContinuityDefaultsKey)
        body()
    }

    func testPanelFilterContextsDefaultToDeterministicValues() {
        withIsolatedPanelFilterDefaults {
            let state = AppShellState()

            XCTAssertEqual(state.communicationsFilterContext(), CommunicationsFilterContext())
            XCTAssertEqual(state.tasksFilterContext(), TasksFilterContext())
            XCTAssertEqual(state.approvalsFilterContext(), ApprovalsFilterContext())
            XCTAssertEqual(state.inspectFilterContext(), InspectFilterContext())
        }
    }

    func testPanelFilterContextsPersistSeparatelyPerWorkspace() {
        withIsolatedPanelFilterDefaults {
            let state = AppShellState()

            state._test_applyWorkspaceContextExplicitSelection("ws-alpha")
            state.updateCommunicationsFilterContext(
                CommunicationsFilterContext(
                    searchText: "urgent",
                    channelFilterID: "message",
                    directionFilterRawValue: "inbound",
                    threadFilterID: "thread-a"
                )
            )

            state._test_applyWorkspaceContextExplicitSelection("ws-beta")
            XCTAssertEqual(state.communicationsFilterContext(), CommunicationsFilterContext())

            state.updateCommunicationsFilterContext(
                CommunicationsFilterContext(
                    searchText: "voice",
                    channelFilterID: "voice",
                    directionFilterRawValue: "outbound",
                    threadFilterID: "thread-b"
                )
            )

            state._test_applyWorkspaceContextExplicitSelection("ws-alpha")
            XCTAssertEqual(state.communicationsFilterContext().searchText, "urgent")
            XCTAssertEqual(state.communicationsFilterContext().threadFilterID, "thread-a")

            state._test_applyWorkspaceContextExplicitSelection("ws-beta")
            XCTAssertEqual(state.communicationsFilterContext().searchText, "voice")
            XCTAssertEqual(state.communicationsFilterContext().threadFilterID, "thread-b")
        }
    }

    func testResetFilterContextsReturnDeterministicDefaults() {
        withIsolatedPanelFilterDefaults {
            let state = AppShellState()

            state.updateTasksFilterContext(
                TasksFilterContext(
                    searchText: "run-123",
                    stateFilter: "Running",
                    priorityFilterRawValue: "high",
                    principalFilter: "agent"
                )
            )
            state.updateApprovalsFilterContext(
                ApprovalsFilterContext(searchText: "approval-123")
            )
            state.updateInspectFilterContext(
                InspectFilterContext(
                    metadataFilterText: "task-abc",
                    statusFilterRawValue: "failure",
                    metadataScopeRawValue: "task",
                    groupingRawValue: "provider",
                    inspectModeRawValue: InspectPresentationMode.trace.rawValue
                )
            )

            XCTAssertEqual(state.resetTasksFilterContext(), TasksFilterContext())
            XCTAssertEqual(state.resetApprovalsFilterContext(), ApprovalsFilterContext())
            XCTAssertEqual(state.resetInspectFilterContext(), InspectFilterContext())
        }
    }

    func testPanelFilterContextsPersistAcrossStateReinitialization() {
        withIsolatedPanelFilterDefaults {
            let workspaceID = "ws-persist-\(UUID().uuidString.lowercased())"
            let searchText = "run-\(UUID().uuidString.lowercased())"

            let first = AppShellState()
            first._test_applyWorkspaceContextExplicitSelection(workspaceID)
            first.updateTasksFilterContext(
                TasksFilterContext(
                    searchText: searchText,
                    stateFilter: "Running",
                    priorityFilterRawValue: "medium",
                    principalFilter: "default"
                )
            )
            let second = AppShellState()
            second._test_applyWorkspaceContextExplicitSelection(workspaceID)

            XCTAssertEqual(second.tasksFilterContext().searchText, searchText)
            XCTAssertEqual(second.tasksFilterContext().stateFilter, "Running")
            XCTAssertEqual(second.tasksFilterContext().priorityFilterRawValue, "medium")
        }
    }

    func testFilterContextActiveSummaryPartsExposeOnlyNonDefaultFilters() {
        let communications = CommunicationsFilterContext(
            searchText: "urgent",
            channelFilterID: "message",
            directionFilterRawValue: "inbound",
            threadFilterID: "thread-42"
        )
        XCTAssertEqual(
            communications.activeFilterSummaryParts,
            [
                "Search: urgent",
                "Channel: message",
                "Direction: Inbound",
                "Thread: thread-42"
            ]
        )

        let tasks = TasksFilterContext(
            searchText: "run-123",
            stateFilter: "Running",
            priorityFilterRawValue: "high",
            principalFilter: "agent-alpha"
        )
        XCTAssertEqual(
            tasks.activeFilterSummaryParts,
            [
                "Search: run-123",
                "State: Running",
                "Priority: High",
                "Principal: agent-alpha"
            ]
        )

        let approvals = ApprovalsFilterContext(searchText: "approval-999")
        XCTAssertEqual(approvals.activeFilterSummaryParts, ["Search: approval-999"])

        let inspect = InspectFilterContext(
            metadataFilterText: "task_1",
            statusFilterRawValue: "failure",
            metadataScopeRawValue: "run",
            groupingRawValue: "provider",
            inspectModeRawValue: InspectPresentationMode.trace.rawValue
        )
        XCTAssertEqual(
            inspect.activeFilterSummaryParts,
            [
                "Match: task_1",
                "Status: Failure",
                "Field: Run",
                "Group: Provider"
            ]
        )

        let activityInspect = InspectFilterContext(
            metadataFilterText: "task_1",
            statusFilterRawValue: "failure",
            metadataScopeRawValue: "run",
            groupingRawValue: "provider",
            inspectModeRawValue: InspectPresentationMode.activity.rawValue
        )
        XCTAssertEqual(
            activityInspect.activeFilterSummaryParts,
            [
                "Match: task_1",
                "Status: Failure"
            ]
        )

        let galleryInspect = InspectFilterContext(
            metadataFilterText: "task_1",
            statusFilterRawValue: "failure",
            metadataScopeRawValue: "run",
            groupingRawValue: "provider",
            inspectModeRawValue: InspectPresentationMode.gallery.rawValue
        )
        XCTAssertEqual(galleryInspect.activeFilterSummaryParts, [])
    }

    func testInspectFilterContextLegacyDecodeDefaultsToActivityMode() throws {
        let payload = """
        {
          "metadataFilterText": "task_legacy",
          "statusFilterRawValue": "failure",
          "metadataScopeRawValue": "run",
          "groupingRawValue": "provider"
        }
        """.data(using: .utf8)!

        let decoded = try JSONDecoder().decode(InspectFilterContext.self, from: payload)
        XCTAssertEqual(decoded.inspectMode, .activity)
        XCTAssertEqual(decoded.inspectModeRawValue, InspectPresentationMode.activity.rawValue)
    }

    func testInspectFilterContextPersistsTraceModeAcrossStateReinitialization() {
        withIsolatedPanelFilterDefaults {
            let workspaceID = "ws-inspect-mode-\(UUID().uuidString.lowercased())"

            let first = AppShellState()
            first._test_applyWorkspaceContextExplicitSelection(workspaceID)
            first.updateInspectFilterContext(
                InspectFilterContext(
                    metadataFilterText: "task-123",
                    statusFilterRawValue: "failure",
                    metadataScopeRawValue: "provider",
                    groupingRawValue: "task",
                    inspectModeRawValue: InspectPresentationMode.trace.rawValue
                )
            )

            let second = AppShellState()
            second._test_applyWorkspaceContextExplicitSelection(workspaceID)
            XCTAssertEqual(second.inspectFilterContext().inspectMode, .trace)
            XCTAssertEqual(second.inspectFilterContext().metadataScopeRawValue, "provider")
            XCTAssertEqual(second.inspectFilterContext().groupingRawValue, "task")
        }
    }

    func testFilterContextActiveSummaryPartsAreEmptyForDefaultContexts() {
        XCTAssertTrue(CommunicationsFilterContext().activeFilterSummaryParts.isEmpty)
        XCTAssertTrue(TasksFilterContext().activeFilterSummaryParts.isEmpty)
        XCTAssertTrue(ApprovalsFilterContext().activeFilterSummaryParts.isEmpty)
        XCTAssertTrue(InspectFilterContext().activeFilterSummaryParts.isEmpty)
    }

    func testCommunicationsCompactScanModeAppearsInActiveSummary() {
        let communications = CommunicationsFilterContext(
            searchText: "",
            channelFilterID: CommunicationsFilterContext.allChannelsID,
            directionFilterRawValue: "all",
            threadFilterID: CommunicationsFilterContext.allThreadsID,
            compactScanModeEnabled: true
        )

        XCTAssertEqual(communications.activeFilterSummaryParts, ["Density: Compact"])
    }

    func testCommunicationsTriageContextPersistsSeparatelyPerWorkspace() {
        withIsolatedPanelFilterDefaults {
            let state = AppShellState()
            state._test_applyWorkspaceContextExplicitSelection("ws-triage-a")
            state.updateCommunicationsTriageContext(
                CommunicationsTriageContext(
                    handledThreadIDs: ["thread-a"],
                    followUpThreadIDs: ["thread-b"],
                    seenThreadIDs: ["thread-a", "thread-b"]
                )
            )

            state._test_applyWorkspaceContextExplicitSelection("ws-triage-b")
            XCTAssertEqual(state.communicationsTriageContext(), CommunicationsTriageContext())

            state.updateCommunicationsTriageContext(
                CommunicationsTriageContext(
                    handledThreadIDs: ["thread-c"],
                    followUpThreadIDs: [],
                    seenThreadIDs: ["thread-c"]
                )
            )

            state._test_applyWorkspaceContextExplicitSelection("ws-triage-a")
            XCTAssertEqual(state.communicationsTriageContext().handledThreadIDs, ["thread-a"])
            XCTAssertEqual(state.communicationsTriageContext().followUpThreadIDs, ["thread-b"])

            state._test_applyWorkspaceContextExplicitSelection("ws-triage-b")
            XCTAssertEqual(state.communicationsTriageContext().handledThreadIDs, ["thread-c"])
        }
    }

    func testWorkspaceContinuityContextPersistsComposeTaskDraftAndExpandedCardsPerWorkspace() {
        withIsolatedPanelFilterDefaults {
            let state = AppShellState()
            state._test_applyWorkspaceContextExplicitSelection("ws-continuity-a")
            state.updateCommunicationsComposeDraftContext(
                CommunicationsComposeDraftContext(
                    isPresented: true,
                    flowID: "reply",
                    sourceChannel: "message",
                    threadID: "thread-a",
                    connectorID: "imessage",
                    destination: "mom@example.com",
                    message: "Draft body"
                )
            )
            state.updateTasksSubmitDraftContext(
                TasksSubmitDraftContext(
                    isPresented: true,
                    title: "Follow up",
                    description: "Draft task",
                    taskClass: "chat",
                    requestedByActorID: "default",
                    subjectPrincipalActorID: "default"
                )
            )
            state.channelCards = [
                ChannelCardItem(
                    id: "message",
                    name: "Message",
                    status: .active,
                    summary: "ok",
                    details: [:],
                    editableConfiguration: [:],
                    editableConfigurationKinds: [:],
                    readOnlyConfiguration: [:],
                    actions: [],
                    unavailableActionReason: ""
                )
            ]
            state.connectorCards = [
                ConnectorCardItem(
                    id: "twilio",
                    name: "Twilio",
                    health: .ready,
                    permissionState: .granted,
                    permissionScope: "messages",
                    summary: "ok",
                    details: [:],
                    editableConfiguration: [:],
                    editableConfigurationKinds: [:],
                    readOnlyConfiguration: [:],
                    actions: [],
                    unavailableActionReason: ""
                )
            ]
            state.toggleChannelCard("message")
            state.toggleConnectorCard("twilio")

            XCTAssertEqual(state.expandedChannelCardIDsContinuity(), Set(["message"]))
            XCTAssertEqual(state.expandedConnectorCardIDsContinuity(), Set(["twilio"]))

            state._test_applyWorkspaceContextExplicitSelection("ws-continuity-b")
            XCTAssertNil(state.communicationsComposeDraftContext())
            XCTAssertNil(state.tasksSubmitDraftContext())
            XCTAssertTrue(state.expandedChannelCardIDsContinuity().isEmpty)
            XCTAssertTrue(state.expandedConnectorCardIDsContinuity().isEmpty)

            state.updateCommunicationsComposeDraftContext(
                CommunicationsComposeDraftContext(
                    isPresented: false,
                    flowID: "new_message",
                    sourceChannel: "app",
                    threadID: "",
                    connectorID: "",
                    destination: "user-2",
                    message: "Ping"
                )
            )

            state._test_applyWorkspaceContextExplicitSelection("ws-continuity-a")
            XCTAssertEqual(state.communicationsComposeDraftContext()?.flowID, "reply")
            XCTAssertEqual(state.communicationsComposeDraftContext()?.threadID, "thread-a")
            XCTAssertEqual(state.tasksSubmitDraftContext()?.title, "Follow up")
            XCTAssertEqual(state.expandedChannelCardIDsContinuity(), Set(["message"]))
            XCTAssertEqual(state.expandedConnectorCardIDsContinuity(), Set(["twilio"]))
        }
    }

    func testTasksSubmitDraftPriorityPersistsSeparatelyPerWorkspace() {
        withIsolatedPanelFilterDefaults {
            let state = AppShellState()
            state._test_applyWorkspaceContextExplicitSelection("ws-priority-a")
            state.updateTasksSubmitDraftContext(
                TasksSubmitDraftContext(
                    isPresented: true,
                    title: "Goal A",
                    description: "Details A",
                    taskClass: "chat",
                    priorityRawValue: "high",
                    requestedByActorID: "default",
                    subjectPrincipalActorID: "default"
                )
            )

            state._test_applyWorkspaceContextExplicitSelection("ws-priority-b")
            state.updateTasksSubmitDraftContext(
                TasksSubmitDraftContext(
                    isPresented: true,
                    title: "Goal B",
                    description: "Details B",
                    taskClass: "chat",
                    priorityRawValue: "low",
                    requestedByActorID: "default",
                    subjectPrincipalActorID: "default"
                )
            )

            state._test_applyWorkspaceContextExplicitSelection("ws-priority-a")
            XCTAssertEqual(state.tasksSubmitDraftContext()?.priorityRawValue, "high")

            state._test_applyWorkspaceContextExplicitSelection("ws-priority-b")
            XCTAssertEqual(state.tasksSubmitDraftContext()?.priorityRawValue, "low")
        }
    }

    func testTasksSubmitDraftContextLegacyDecodeDefaultsPriorityToMedium() throws {
        let legacyJSON = """
        {
          "isPresented": true,
          "title": "Legacy Goal",
          "description": "Legacy Details",
          "taskClass": "chat",
          "requestedByActorID": "default",
          "subjectPrincipalActorID": "default"
        }
        """
        let decoded = try JSONDecoder().decode(
            TasksSubmitDraftContext.self,
            from: Data(legacyJSON.utf8)
        )
        XCTAssertEqual(decoded.priorityRawValue, "medium")
    }

    func testResetWorkspaceContinuityContextClearsPersistedDraftsForCurrentWorkspace() {
        withIsolatedPanelFilterDefaults {
            let state = AppShellState()
            state._test_applyWorkspaceContextExplicitSelection("ws-continuity-reset")
            state.updateCommunicationsComposeDraftContext(
                CommunicationsComposeDraftContext(
                    isPresented: true,
                    flowID: "reply",
                    sourceChannel: "message",
                    threadID: "thread-reset",
                    connectorID: "twilio",
                    destination: "+15550001111",
                    message: "Draft"
                )
            )
            state.updateTasksSubmitDraftContext(
                TasksSubmitDraftContext(
                    isPresented: true,
                    title: "Reset me",
                    description: "draft",
                    taskClass: "chat",
                    requestedByActorID: "default",
                    subjectPrincipalActorID: "default"
                )
            )
            state.channelCards = [
                ChannelCardItem(
                    id: "voice",
                    name: "Voice",
                    status: .active,
                    summary: "ok",
                    details: [:],
                    editableConfiguration: [:],
                    editableConfigurationKinds: [:],
                    readOnlyConfiguration: [:],
                    actions: [],
                    unavailableActionReason: ""
                )
            ]
            state.connectorCards = [
                ConnectorCardItem(
                    id: "imessage",
                    name: "Messages",
                    health: .ready,
                    permissionState: .granted,
                    permissionScope: "messages",
                    summary: "ok",
                    details: [:],
                    editableConfiguration: [:],
                    editableConfigurationKinds: [:],
                    readOnlyConfiguration: [:],
                    actions: [],
                    unavailableActionReason: ""
                )
            ]
            state.toggleChannelCard("voice")
            state.toggleConnectorCard("imessage")

            state.resetWorkspaceContinuityContext()

            XCTAssertNil(state.communicationsComposeDraftContext())
            XCTAssertNil(state.tasksSubmitDraftContext())
            XCTAssertTrue(state.expandedChannelCardIDsContinuity().isEmpty)
            XCTAssertTrue(state.expandedConnectorCardIDsContinuity().isEmpty)
        }
    }

    func testActiveFilterCountBySectionReflectsPersistedWorkspaceContext() {
        withIsolatedPanelFilterDefaults {
            let state = AppShellState()
            state._test_applyWorkspaceContextExplicitSelection("ws-active-filter-counts")

            XCTAssertEqual(state.activeFilterCount(for: .communications), 0)
            XCTAssertEqual(state.activeFilterCount(for: .tasks), 0)
            XCTAssertEqual(state.activeFilterCount(for: .approvals), 0)
            XCTAssertEqual(state.activeFilterCount(for: .inspect), 0)
            XCTAssertEqual(state.activeFilterCount(for: .chat), 0)
            XCTAssertNil(state.activeFilterSummary(for: .communications))
            XCTAssertNil(state.activeFilterSummary(for: .chat))

            state.updateCommunicationsFilterContext(
                CommunicationsFilterContext(
                    searchText: "urgent",
                    channelFilterID: "message",
                    directionFilterRawValue: "inbound",
                    threadFilterID: CommunicationsFilterContext.allThreadsID
                )
            )
            state.updateTasksFilterContext(
                TasksFilterContext(
                    searchText: "run-1",
                    stateFilter: "Running",
                    priorityFilterRawValue: "all",
                    principalFilter: "All Principals"
                )
            )
            state.updateApprovalsFilterContext(
                ApprovalsFilterContext(searchText: "approval-1")
            )
            state.updateInspectFilterContext(
                InspectFilterContext(
                    metadataFilterText: "",
                    statusFilterRawValue: "failure",
                    metadataScopeRawValue: "all",
                    groupingRawValue: "none"
                )
            )

            XCTAssertEqual(state.activeFilterCount(for: .communications), 3)
            XCTAssertEqual(state.activeFilterCount(for: .tasks), 2)
            XCTAssertEqual(state.activeFilterCount(for: .approvals), 1)
            XCTAssertEqual(state.activeFilterCount(for: .inspect), 1)
            XCTAssertEqual(
                state.activeFilterSummary(for: .communications),
                "Search: urgent • Channel: message • Direction: Inbound"
            )
            XCTAssertEqual(
                state.activeFilterSummary(for: .tasks),
                "Search: run-1 • State: Running"
            )
            XCTAssertEqual(
                state.activeFilterSummary(for: .approvals),
                "Search: approval-1"
            )
            XCTAssertEqual(
                state.activeFilterSummary(for: .inspect),
                "Status: Failure"
            )
        }
    }

    func testLegacyDefaultWorkspaceScopedContextsAreIgnoredWithoutMigration() {
        withIsolatedPanelFilterDefaults {
            let defaults = appShellStateTestUserDefaults()
            defaults.set(
                try? JSONEncoder().encode([
                    "default": WorkspacePanelFilterContext(
                        communications: CommunicationsFilterContext(
                            searchText: "legacy-urgent",
                            channelFilterID: "message",
                            directionFilterRawValue: "inbound",
                            threadFilterID: "thread-legacy"
                        ),
                        tasks: TasksFilterContext(
                            searchText: "legacy-run",
                            stateFilter: "Running",
                            priorityFilterRawValue: "high",
                            principalFilter: "default"
                        ),
                        approvals: ApprovalsFilterContext(searchText: "approval-legacy"),
                        inspect: InspectFilterContext(
                            metadataFilterText: "legacy",
                            statusFilterRawValue: "failure",
                            metadataScopeRawValue: "task",
                            groupingRawValue: "provider"
                        )
                    )
                ]),
                forKey: panelFilterDefaultsKey
            )
            defaults.set(
                try? JSONEncoder().encode([
                    "default": CommunicationsTriageContext(
                        handledThreadIDs: ["thread-legacy"],
                        followUpThreadIDs: ["thread-follow-up"],
                        seenThreadIDs: ["thread-legacy", "thread-follow-up"]
                    )
                ]),
                forKey: communicationsTriageDefaultsKey
            )
            defaults.set(
                try? JSONEncoder().encode([
                    "default": WorkspaceContinuityContext(
                        expandedChannelCardIDs: ["message"],
                        expandedConnectorCardIDs: ["twilio"],
                        communicationsComposeDraft: CommunicationsComposeDraftContext(
                            isPresented: true,
                            flowID: "reply",
                            sourceChannel: "message",
                            threadID: "thread-legacy",
                            connectorID: "apple.messages",
                            destination: "legacy-user",
                            message: "legacy body"
                        ),
                        tasksSubmitDraft: TasksSubmitDraftContext(
                            isPresented: true,
                            title: "Legacy Task",
                            description: "legacy draft",
                            taskClass: "chat",
                            requestedByActorID: "default",
                            subjectPrincipalActorID: "default"
                        )
                    )
                ]),
                forKey: workspaceContinuityDefaultsKey
            )

            let state = AppShellState()

            XCTAssertEqual(state.workspaceID, "ws1")
            XCTAssertEqual(state.communicationsFilterContext().searchText, "")
            XCTAssertEqual(state.tasksFilterContext().searchText, "")
            XCTAssertEqual(state.approvalsFilterContext().searchText, "")
            XCTAssertEqual(state.inspectFilterContext().metadataFilterText, "")
            XCTAssertEqual(state.inspectFilterContext().inspectMode, .activity)
            XCTAssertEqual(state.communicationsTriageContext().handledThreadIDs, [])
            XCTAssertNil(state.communicationsComposeDraftContext())
            XCTAssertNil(state.tasksSubmitDraftContext())

            let panelData = defaults.data(forKey: panelFilterDefaultsKey)
            let triageData = defaults.data(forKey: communicationsTriageDefaultsKey)
            let continuityData = defaults.data(forKey: workspaceContinuityDefaultsKey)

            guard
                let panelData,
                let triageData,
                let continuityData,
                let panelMap = try? JSONDecoder().decode([String: WorkspacePanelFilterContext].self, from: panelData),
                let triageMap = try? JSONDecoder().decode([String: CommunicationsTriageContext].self, from: triageData),
                let continuityMap = try? JSONDecoder().decode([String: WorkspaceContinuityContext].self, from: continuityData)
            else {
                XCTFail("Expected persisted workspace-scoped context payloads to be present.")
                return
            }

            XCTAssertTrue(panelMap.keys.contains("default"))
            XCTAssertFalse(panelMap.keys.contains("ws1"))
            XCTAssertTrue(triageMap.keys.contains("default"))
            XCTAssertFalse(triageMap.keys.contains("ws1"))
            XCTAssertTrue(continuityMap.keys.contains("default"))
            XCTAssertFalse(continuityMap.keys.contains("ws1"))

            let restored = AppShellState()
            XCTAssertEqual(restored.communicationsFilterContext().searchText, "")
            XCTAssertNil(restored.tasksSubmitDraftContext())
        }
    }
}
