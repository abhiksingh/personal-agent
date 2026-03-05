import XCTest
@testable import PersonalAgentUI

@MainActor
final class AppNotificationCenterStoreTests: XCTestCase {
    private let defaultsKey = "personalagent.ui.notifications.u270.tests"

    func testPostNotificationCreatesItemToastAndSuccessPulse() {
        let defaults = appShellStateTestUserDefaults()
        defaults.removeObject(forKey: defaultsKey)
        let store = makeStore(defaults: defaults)

        store.postNotification(
            workspaceID: "ws1",
            source: "tasks",
            action: "task_submit",
            message: "Submitted task task-1.",
            level: .success
        )

        XCTAssertEqual(store.notificationItems.count, 1)
        XCTAssertEqual(store.notificationToastItems.first?.id, store.notificationItems.first?.id)
        XCTAssertEqual(store.successNotificationPulse(for: "tasks"), 1)
        XCTAssertEqual(store.unreadNotificationCount(), 1)
    }

    func testFilterSupportsSourceAndSearchQuery() {
        let defaults = appShellStateTestUserDefaults()
        defaults.removeObject(forKey: defaultsKey)
        let store = makeStore(defaults: defaults)

        store.postNotification(
            workspaceID: "ws1",
            source: "channels",
            action: "open_channels",
            message: "Opened Channels for setup remediation.",
            level: .info
        )
        store.postNotification(
            workspaceID: "ws1",
            source: "connectors",
            action: "status_update",
            message: "Connector diagnostics query failed.",
            level: .error
        )

        let sourceFiltered = store.filteredNotificationItems(
            query: "",
            sourceFilter: "connectors"
        )
        XCTAssertEqual(sourceFiltered.count, 1)
        XCTAssertEqual(sourceFiltered.first?.source, "connectors")

        let queryFiltered = store.filteredNotificationItems(
            query: "setup remediation",
            sourceFilter: "all"
        )
        XCTAssertEqual(queryFiltered.count, 1)
        XCTAssertEqual(queryFiltered.first?.source, "channels")
    }

    func testReadAndClearBehaviorsRemainDeterministic() {
        let defaults = appShellStateTestUserDefaults()
        defaults.removeObject(forKey: defaultsKey)
        let store = makeStore(defaults: defaults)

        store.postNotification(
            workspaceID: "ws1",
            source: "tasks",
            action: "refresh",
            message: "Loaded 3 task run rows.",
            level: .success
        )
        store.postNotification(
            workspaceID: "ws1",
            source: "approvals",
            action: "decision",
            message: "Approval decision submitted.",
            level: .success
        )

        guard let firstID = store.notificationItems.first?.id else {
            XCTFail("Expected notification item.")
            return
        }

        store.markNotificationRead(notificationID: firstID)
        XCTAssertLessThan(store.unreadNotificationCount(), store.notificationItems.count)

        store.clearReadNotifications()
        XCTAssertTrue(store.notificationItems.allSatisfy { !$0.isRead })

        store.clearAllNotifications()
        XCTAssertTrue(store.notificationItems.isEmpty)
        XCTAssertTrue(store.notificationToastItems.isEmpty)
    }

    func testStatusNotificationCaptureRulesAndLevelMapping() {
        let defaults = appShellStateTestUserDefaults()
        defaults.removeObject(forKey: defaultsKey)
        let store = makeStore(defaults: defaults)

        store.recordStatusNotification(
            workspaceID: "ws1",
            source: "chat",
            oldValue: nil,
            newValue: "Waiting for daemon status..."
        )
        XCTAssertTrue(store.notificationItems.isEmpty)

        store.recordStatusNotification(
            workspaceID: "ws1",
            source: "chat",
            oldValue: nil,
            newValue: "Chat turn failed due to timeout."
        )
        XCTAssertEqual(store.notificationItems.count, 1)
        XCTAssertEqual(store.notificationItems.first?.level, .error)

        store.recordStatusNotification(
            workspaceID: "ws1",
            source: "chat",
            oldValue: "Chat turn failed due to timeout.",
            newValue: "Chat turn failed due to timeout."
        )
        XCTAssertEqual(store.notificationItems.count, 1)
    }

    func testPersistenceRestoreLoadsSavedEntries() {
        let defaults = appShellStateTestUserDefaults()
        defaults.removeObject(forKey: defaultsKey)
        let first = makeStore(defaults: defaults)
        first.clearAllNotifications()
        first.postNotification(
            workspaceID: "ws1",
            source: "chat",
            action: "interrupt",
            message: "Chat interrupted.",
            level: .info
        )

        let second = makeStore(defaults: defaults)
        second.loadPersistedNotifications()

        XCTAssertTrue(second.notificationItems.contains(where: {
            $0.source == "chat" && $0.message == "Chat interrupted."
        }))
        XCTAssertTrue(second.notificationToastItems.isEmpty)
    }

    private func makeStore(defaults: UserDefaults) -> AppNotificationCenterStore {
        AppNotificationCenterStore(
            userDefaults: defaults,
            defaultsKey: defaultsKey,
            defaultWorkspaceID: "ws1",
            notificationHistoryLimit: 250
        )
    }
}
