import XCTest
@testable import PersonalAgentUI

final class NotificationInboxRoutingTests: XCTestCase {
    func testSourceSectionMappingSupportsCanonicalAndUiConfigurationPrefix() {
        XCTAssertEqual(NotificationInboxRouting.sourceSection(for: "chat"), .chat)
        XCTAssertEqual(NotificationInboxRouting.sourceSection(for: "models"), .models)
        XCTAssertEqual(NotificationInboxRouting.sourceSection(for: "ui.configuration.setup"), .configuration)
        XCTAssertNil(NotificationInboxRouting.sourceSection(for: "unknown"))
    }

    func testInboxIntentMappingMatchesExistingSemantics() {
        let workflowItem = AppNotificationItem(
            workspaceID: "ws1",
            source: "tasks",
            action: "refresh",
            level: .success,
            message: "Loaded task rows."
        )
        XCTAssertEqual(NotificationInboxRouting.inboxIntent(for: workflowItem), .workflow)

        let runtimeItem = AppNotificationItem(
            workspaceID: "ws1",
            source: "connectors",
            action: "status_update",
            level: .info,
            message: "Connector check complete."
        )
        XCTAssertEqual(NotificationInboxRouting.inboxIntent(for: runtimeItem), .runtime)

        let diagnosticsItem = AppNotificationItem(
            workspaceID: "ws1",
            source: "inspect",
            action: "refresh",
            level: .info,
            message: "Inspect loaded."
        )
        XCTAssertEqual(NotificationInboxRouting.inboxIntent(for: diagnosticsItem), .diagnostics)

        let needsAttentionItem = AppNotificationItem(
            workspaceID: "ws1",
            source: "unknown",
            action: "status_update",
            level: .error,
            message: "Operation failed."
        )
        XCTAssertEqual(NotificationInboxRouting.inboxIntent(for: needsAttentionItem), .needsAttention)

        let progressFallback = AppNotificationItem(
            workspaceID: "ws1",
            source: "unknown",
            action: "status_update",
            level: .progress,
            message: "Checking…"
        )
        XCTAssertEqual(NotificationInboxRouting.inboxIntent(for: progressFallback), .runtime)

        let generalFallback = AppNotificationItem(
            workspaceID: "ws1",
            source: "unknown",
            action: "status_update",
            level: .info,
            message: "FYI."
        )
        XCTAssertEqual(NotificationInboxRouting.inboxIntent(for: generalFallback), .general)
    }

    func testInboxActionsMapToSectionOrConfigurationFallbackForErrors() {
        let channelsItem = AppNotificationItem(
            workspaceID: "ws1",
            source: "channels",
            action: "status_update",
            level: .info,
            message: "Opened Channels for setup remediation."
        )
        let channelActions = NotificationInboxRouting.inboxActions(for: channelsItem)
        XCTAssertEqual(channelActions.count, 1)
        XCTAssertEqual(channelActions.first?.title, "Open Channels")

        let unknownErrorItem = AppNotificationItem(
            workspaceID: "ws1",
            source: "unknown",
            action: "status_update",
            level: .error,
            message: "Unknown error."
        )
        let fallbackActions = NotificationInboxRouting.inboxActions(for: unknownErrorItem)
        XCTAssertEqual(fallbackActions.count, 1)
        XCTAssertEqual(fallbackActions.first?.title, "Open Configuration")

        let unknownInfoItem = AppNotificationItem(
            workspaceID: "ws1",
            source: "unknown",
            action: "status_update",
            level: .info,
            message: "Unknown info."
        )
        XCTAssertTrue(NotificationInboxRouting.inboxActions(for: unknownInfoItem).isEmpty)
    }
}
