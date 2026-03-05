import XCTest
@testable import PersonalAgentUI

@MainActor
final class AppContextRetentionStoreTests: XCTestCase {
    func testPanelFilterContextPersistsSeparatelyPerWorkspace() {
        withIsolatedStore { store, keys in
            store.updatePanelFilterContext(for: "ws-alpha") { context in
                context.communications = CommunicationsFilterContext(
                    searchText: "urgent",
                    channelFilterID: "message",
                    directionFilterRawValue: "inbound",
                    threadFilterID: "thread-a"
                )
            }
            store.updatePanelFilterContext(for: "ws-beta") { context in
                context.tasks = TasksFilterContext(
                    searchText: "run-42",
                    stateFilter: "Running",
                    priorityFilterRawValue: "high",
                    principalFilter: "default"
                )
            }

            XCTAssertEqual(store.panelFilterContext(for: "ws-alpha").communications.searchText, "urgent")
            XCTAssertEqual(store.panelFilterContext(for: "ws-beta").tasks.searchText, "run-42")

            let reloaded = makeStore(keys: keys)
            reloaded.loadPersistedPanelFilterContexts()
            XCTAssertEqual(reloaded.panelFilterContext(for: "ws-alpha").communications.searchText, "urgent")
            XCTAssertEqual(reloaded.panelFilterContext(for: "ws-beta").tasks.searchText, "run-42")
        }
    }

    func testWorkspaceContinuityResetAffectsOnlyCurrentWorkspace() {
        withIsolatedStore { store, _ in
            store.updateWorkspaceContinuityContext(for: "ws-a") { context in
                context.expandedChannelCardIDs = ["message"]
                context.expandedConnectorCardIDs = ["twilio"]
                context.communicationsComposeDraft = CommunicationsComposeDraftContext(
                    isPresented: true,
                    flowID: "reply",
                    sourceChannel: "message",
                    threadID: "thread-a",
                    connectorID: "twilio",
                    destination: "person@example.com",
                    message: "Draft"
                )
                context.tasksSubmitDraft = TasksSubmitDraftContext(
                    isPresented: true,
                    title: "Follow up",
                    description: "Task draft",
                    taskClass: "chat",
                    requestedByActorID: "default",
                    subjectPrincipalActorID: "default"
                )
            }
            store.updateWorkspaceContinuityContext(for: "ws-b") { context in
                context.expandedChannelCardIDs = ["voice"]
            }

            store.resetWorkspaceContinuityContext(for: "ws-a")

            XCTAssertEqual(store.workspaceContinuityContext(for: "ws-a"), WorkspaceContinuityContext())
            XCTAssertEqual(store.workspaceContinuityContext(for: "ws-b").expandedChannelCardIDs, ["voice"])
        }
    }

    func testInformationDensityModePersistsPerWorkspace() {
        withIsolatedStore { store, keys in
            store.setInformationDensityMode(.advanced, for: "ws-a")

            XCTAssertEqual(store.informationDensityMode(for: "ws-a"), .advanced)
            XCTAssertEqual(store.informationDensityMode(for: "ws-b"), .simple)

            let reloaded = makeStore(keys: keys)
            reloaded.loadPersistedInformationDensityModes()
            XCTAssertEqual(reloaded.informationDensityMode(for: "ws-a"), .advanced)
            XCTAssertEqual(reloaded.informationDensityMode(for: "ws-b"), .simple)
        }
    }

    func testHomeFirstSessionProgressPersistsAndResetClearsState() {
        withIsolatedStore { store, keys in
            store.updateHomeFirstSessionProgress(for: "ws-a") { progress in
                progress.sentMessage = true
                progress.milestoneEvidenceByStepID["send_message"] =
                    AppContextRetentionStore.HomeFirstSessionProgress.MilestoneEvidence(
                        completedAtRaw: "2026-03-04T21:00:00Z",
                        source: "chat_turn"
                    )
            }

            let reloaded = makeStore(keys: keys)
            reloaded.loadPersistedHomeFirstSessionProgress()
            XCTAssertTrue(reloaded.homeFirstSessionProgress(for: "ws-a").sentMessage)
            XCTAssertEqual(
                reloaded.homeFirstSessionProgress(for: "ws-a")
                    .milestoneEvidenceByStepID["send_message"]?.source,
                "chat_turn"
            )

            reloaded.resetHomeFirstSessionProgress()

            let afterReset = makeStore(keys: keys)
            afterReset.loadPersistedHomeFirstSessionProgress()
            XCTAssertEqual(afterReset.homeFirstSessionProgress(for: "ws-a"), .init())
        }
    }

    private struct StoreKeys {
        let panelFilter: String
        let triage: String
        let continuity: String
        let density: String
        let homeProgress: String
    }

    private func withIsolatedStore(
        _ body: (_ store: AppContextRetentionStore, _ keys: StoreKeys) -> Void
    ) {
        let suffix = UUID().uuidString.lowercased()
        let keys = StoreKeys(
            panelFilter: "test.panel_filter.\(suffix)",
            triage: "test.communications_triage.\(suffix)",
            continuity: "test.workspace_continuity.\(suffix)",
            density: "test.information_density.\(suffix)",
            homeProgress: "test.home_progress.\(suffix)"
        )
        let defaults = appShellStateTestUserDefaults()
        let allKeys = [keys.panelFilter, keys.triage, keys.continuity, keys.density, keys.homeProgress]
        for key in allKeys {
            defaults.removeObject(forKey: key)
        }
        defer {
            for key in allKeys {
                defaults.removeObject(forKey: key)
            }
        }

        body(makeStore(keys: keys), keys)
    }

    private func makeStore(keys: StoreKeys) -> AppContextRetentionStore {
        AppContextRetentionStore(
            userDefaults: appShellStateTestUserDefaults(),
            defaultWorkspaceID: "ws1",
            canonicalWorkspaceID: Self.canonicalWorkspaceID,
            panelFilterContextDefaultsKey: keys.panelFilter,
            communicationsTriageDefaultsKey: keys.triage,
            workspaceContinuityDefaultsKey: keys.continuity,
            informationDensityModeDefaultsKey: keys.density,
            homeFirstSessionProgressDefaultsKey: keys.homeProgress
        )
    }

    private static func canonicalWorkspaceID(_ raw: String?, _ fallbackToDefault: Bool) -> String? {
        let trimmed = raw?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        if trimmed.isEmpty {
            return fallbackToDefault ? "ws1" : nil
        }
        return trimmed
    }
}
