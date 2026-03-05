import XCTest
@testable import PersonalAgentUI

@MainActor
final class AppShellStateNotificationCenterTests: XCTestCase {
    private let notificationsDefaultsKey = "personalagent.ui.notifications.v1"

    func testStatusUpdateCreatesNotificationAndToast() {
        let defaults = appShellStateTestUserDefaults()
        let priorNotifications = defaults.object(forKey: notificationsDefaultsKey)
        defer {
            if let priorNotifications {
                defaults.set(priorNotifications, forKey: notificationsDefaultsKey)
            } else {
                defaults.removeObject(forKey: notificationsDefaultsKey)
            }
        }

        defaults.removeObject(forKey: notificationsDefaultsKey)

        let state = AppShellState()
        state.clearAllNotifications()
        state.postNotification(
            source: "channels",
            action: "status_update",
            message: "Updated connector mappings for message.",
            level: .success
        )

        XCTAssertEqual(state.notificationItems.count, 1)
        XCTAssertEqual(state.notificationItems[0].source, "channels")
        XCTAssertEqual(state.notificationItems[0].action, "status_update")
        XCTAssertEqual(state.notificationItems[0].level, .success)
        XCTAssertEqual(state.notificationItems[0].message, "Updated connector mappings for message.")
        XCTAssertEqual(state.notificationToastItems.first?.id, state.notificationItems[0].id)
        XCTAssertEqual(state.unreadNotificationCount, 1)
    }

    func testNotificationFilteringSupportsSourceAndSearchQuery() {
        let defaults = appShellStateTestUserDefaults()
        let priorNotifications = defaults.object(forKey: notificationsDefaultsKey)
        defer {
            if let priorNotifications {
                defaults.set(priorNotifications, forKey: notificationsDefaultsKey)
            } else {
                defaults.removeObject(forKey: notificationsDefaultsKey)
            }
        }

        defaults.removeObject(forKey: notificationsDefaultsKey)

        let state = AppShellState()
        state.clearAllNotifications()
        state.postNotification(
            source: "channels",
            action: "open_channels",
            message: "Opened Channels for setup remediation.",
            level: .info
        )
        state.postNotification(
            source: "connectors",
            action: "status_update",
            message: "Connector diagnostics query failed.",
            level: .error
        )

        state.notificationCenterSourceFilter = "connectors"
        XCTAssertEqual(state.filteredNotificationItems.count, 1)
        XCTAssertEqual(state.filteredNotificationItems[0].source, "connectors")

        state.notificationCenterSourceFilter = "all"
        state.notificationCenterSearchQuery = "setup remediation"
        XCTAssertEqual(state.filteredNotificationItems.count, 1)
        XCTAssertEqual(state.filteredNotificationItems[0].source, "channels")
    }

    func testNotificationReadAndClearBehaviorsAreDeterministic() {
        let defaults = appShellStateTestUserDefaults()
        let priorNotifications = defaults.object(forKey: notificationsDefaultsKey)
        defer {
            if let priorNotifications {
                defaults.set(priorNotifications, forKey: notificationsDefaultsKey)
            } else {
                defaults.removeObject(forKey: notificationsDefaultsKey)
            }
        }

        defaults.removeObject(forKey: notificationsDefaultsKey)

        let state = AppShellState()
        state.clearAllNotifications()
        state.postNotification(
            source: "tasks",
            action: "refresh",
            message: "Loaded 3 task run rows.",
            level: .success
        )
        state.postNotification(
            source: "approvals",
            action: "decision",
            message: "Approval decision submitted.",
            level: .success
        )

        guard let firstID = state.notificationItems.first?.id else {
            XCTFail("Expected at least one notification item.")
            return
        }

        state.markNotificationRead(notificationID: firstID)
        XCTAssertLessThan(state.unreadNotificationCount, state.notificationItems.count)

        state.clearReadNotifications()
        XCTAssertTrue(state.notificationItems.allSatisfy { !$0.isRead })

        state.clearAllNotifications()
        XCTAssertTrue(state.notificationItems.isEmpty)
        XCTAssertTrue(state.notificationToastItems.isEmpty)
    }

    func testNotificationPersistenceRestoresSavedEntries() {
        let defaults = appShellStateTestUserDefaults()
        let priorNotifications = defaults.object(forKey: notificationsDefaultsKey)
        defer {
            if let priorNotifications {
                defaults.set(priorNotifications, forKey: notificationsDefaultsKey)
            } else {
                defaults.removeObject(forKey: notificationsDefaultsKey)
            }
        }

        defaults.removeObject(forKey: notificationsDefaultsKey)

        let firstState = AppShellState()
        firstState.clearAllNotifications()
        firstState.postNotification(
            source: "chat",
            action: "interrupt",
            message: "Chat interrupted.",
            level: .info
        )
        XCTAssertTrue(firstState.notificationItems.contains(where: {
            $0.source == "chat" && $0.message == "Chat interrupted."
        }))

        let secondState = AppShellState()
        XCTAssertTrue(secondState.notificationItems.contains(where: {
            $0.source == "chat" && $0.message == "Chat interrupted."
        }))
    }

    func testSuccessNotificationPulseIncrementsPerSource() {
        let defaults = appShellStateTestUserDefaults()
        let priorNotifications = defaults.object(forKey: notificationsDefaultsKey)
        defer {
            if let priorNotifications {
                defaults.set(priorNotifications, forKey: notificationsDefaultsKey)
            } else {
                defaults.removeObject(forKey: notificationsDefaultsKey)
            }
        }

        defaults.removeObject(forKey: notificationsDefaultsKey)

        let state = AppShellState()
        state.clearAllNotifications()

        state.postNotification(
            source: "tasks",
            action: "task_submit",
            message: "Submitted task task-1 • run run-1.",
            level: .success
        )
        state.postNotification(
            source: "tasks",
            action: "task_submit",
            message: "Task submit failed.",
            level: .error
        )
        state.postNotification(
            source: "chat",
            action: "chat_turn",
            message: "Sent chat turn via openai • gpt-4.1 (realtime).",
            level: .success
        )

        XCTAssertEqual(state.successNotificationPulse(for: "tasks"), 1)
        XCTAssertEqual(state.successNotificationPulse(for: "chat"), 1)
        XCTAssertEqual(state.successNotificationPulse(for: "connectors"), 0)
    }

    func testNotificationInboxGroupsByIntentPriority() {
        let defaults = appShellStateTestUserDefaults()
        let priorNotifications = defaults.object(forKey: notificationsDefaultsKey)
        defer {
            if let priorNotifications {
                defaults.set(priorNotifications, forKey: notificationsDefaultsKey)
            } else {
                defaults.removeObject(forKey: notificationsDefaultsKey)
            }
        }

        defaults.removeObject(forKey: notificationsDefaultsKey)

        let state = AppShellState()
        state.clearAllNotifications()
        state.postNotification(
            source: "connectors",
            action: "status_update",
            message: "Connector diagnostics query failed.",
            level: .error
        )
        state.postNotification(
            source: "tasks",
            action: "refresh",
            message: "Loaded 3 task run rows.",
            level: .success
        )
        state.postNotification(
            source: "models",
            action: "refresh",
            message: "Loaded model inventory.",
            level: .info
        )
        state.postNotification(
            source: "inspect",
            action: "refresh",
            message: "Loaded inspect logs.",
            level: .info
        )

        let intents = state.groupedFilteredNotificationSections.map(\.intent)
        XCTAssertEqual(intents, [.needsAttention, .workflow, .runtime, .diagnostics])
    }

    func testNotificationInboxActionMappingAndDispatchNavigatesAndMarksRead() {
        let defaults = appShellStateTestUserDefaults()
        let priorNotifications = defaults.object(forKey: notificationsDefaultsKey)
        defer {
            if let priorNotifications {
                defaults.set(priorNotifications, forKey: notificationsDefaultsKey)
            } else {
                defaults.removeObject(forKey: notificationsDefaultsKey)
            }
        }

        defaults.removeObject(forKey: notificationsDefaultsKey)

        let state = AppShellState()
        state.clearAllNotifications()
        state.postNotification(
            source: "channels",
            action: "status_update",
            message: "Opened Channels for setup remediation.",
            level: .info
        )

        guard let notification = state.notificationItems.first else {
            XCTFail("Expected notification item for inbox action test.")
            return
        }

        let actions = state.notificationInboxActions(for: notification)
        XCTAssertEqual(actions.count, 1)
        XCTAssertEqual(actions.first?.title, "Open Channels")

        if let firstAction = actions.first {
            state.performNotificationInboxAction(firstAction, notificationID: notification.id)
        }

        XCTAssertEqual(state.selectedSection, .channels)
        XCTAssertEqual(state.unreadNotificationCount, 0)
    }

    func testUnknownErrorNotificationFallsBackToOpenConfigurationAction() {
        let defaults = appShellStateTestUserDefaults()
        let priorNotifications = defaults.object(forKey: notificationsDefaultsKey)
        defer {
            if let priorNotifications {
                defaults.set(priorNotifications, forKey: notificationsDefaultsKey)
            } else {
                defaults.removeObject(forKey: notificationsDefaultsKey)
            }
        }

        defaults.removeObject(forKey: notificationsDefaultsKey)

        let state = AppShellState()
        state.clearAllNotifications()
        state.postNotification(
            source: "system",
            action: "status_update",
            message: "Runtime check failed.",
            level: .error
        )

        guard let notification = state.notificationItems.first else {
            XCTFail("Expected fallback notification item.")
            return
        }

        let actions = state.notificationInboxActions(for: notification)
        XCTAssertEqual(actions.count, 1)
        XCTAssertEqual(actions.first?.title, "Open Configuration")
    }

    func testRoutineChatStatusMessageDoesNotCreateStatusNotification() {
        let defaults = appShellStateTestUserDefaults()
        let priorNotifications = defaults.object(forKey: notificationsDefaultsKey)
        defer {
            if let priorNotifications {
                defaults.set(priorNotifications, forKey: notificationsDefaultsKey)
            } else {
                defaults.removeObject(forKey: notificationsDefaultsKey)
            }
        }

        defaults.removeObject(forKey: notificationsDefaultsKey)

        let state = AppShellState()
        state.clearAllNotifications()
        state.chatStatusMessage = "Checking assistant connection."

        XCTAssertEqual(state.notificationItems.count, 0)
        XCTAssertEqual(state.successNotificationPulse(for: "chat"), 0)
    }
}
