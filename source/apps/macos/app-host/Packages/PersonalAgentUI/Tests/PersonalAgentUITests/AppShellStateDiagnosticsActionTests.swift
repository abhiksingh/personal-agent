import XCTest
@testable import PersonalAgentUI

@MainActor
final class AppShellStateDiagnosticsActionTests: XCTestCase {
    func testChannelDiagnosticsOpenLogsNavigatesToInspect() {
        let state = AppShellState()
        let action = DiagnosticsActionItem(
            id: "open_channel_logs",
            title: "Open Inspect Logs",
            intent: "navigate",
            destination: "ui://inspect/logs?scope=channel:app_chat",
            parameters: ["channel_id": "app_chat"],
            enabled: true,
            recommended: true,
            reason: nil
        )

        state.performChannelDiagnosticsAction(channelID: "app_chat", action: action)

        XCTAssertEqual(state.selectedSection, .inspect)
        XCTAssertEqual(state.channelsStatusMessage, "Opened Inspect for channel diagnostics.")
    }

    func testConnectorDiagnosticsDisabledActionUsesReasonMessage() {
        let state = AppShellState()
        let action = DiagnosticsActionItem(
            id: "repair_daemon_runtime",
            title: "Run Daemon Repair",
            intent: "daemon_lifecycle_control",
            destination: "/v1/daemon/lifecycle/control",
            parameters: ["action": "repair"],
            enabled: false,
            recommended: true,
            reason: "Worker is already starting."
        )

        state.performConnectorDiagnosticsAction(connectorID: "mail", action: action)

        XCTAssertEqual(state.connectorsStatusMessage, "Worker is already starting.")
    }

    func testConnectorDiagnosticsOpenSystemSettingsSetsRefreshStatus() {
        let state = AppShellState()
        let action = DiagnosticsActionItem(
            id: "open_connector_system_settings",
            title: "Open System Settings",
            intent: "open_system_settings",
            destination: "ui://system-settings/privacy/automation",
            parameters: ["connector_id": "mail"],
            enabled: true,
            recommended: true,
            reason: nil
        )

        state.performConnectorDiagnosticsAction(connectorID: "mail", action: action)

        XCTAssertEqual(
            state.connectorPermissionActionStatusByID["mail"],
            "Opened System Settings. Return to the app to refresh permission status."
        )
        XCTAssertEqual(state.connectorsStatusMessage, "Opened System Settings for mail permission checks.")
    }

    func testSystemSettingsURLForConnectorActionPrefersDaemonDestination() {
        let state = AppShellState()
        let action = DiagnosticsActionItem(
            id: "open_connector_system_settings",
            title: "Open Full Disk Access",
            intent: "open_system_settings",
            destination: "ui://system-settings/privacy/full-disk-access",
            parameters: ["connector_id": "mail"],
            enabled: true,
            recommended: true,
            reason: nil
        )

        let resolved = state.systemSettingsURLForConnectorAction(connectorID: "mail", action: action)

        XCTAssertEqual(
            resolved.absoluteString,
            "x-apple.systempreferences:com.apple.preference.security?Privacy_AllFiles"
        )
    }

    func testSystemSettingsURLForConnectorActionFallsBackWhenDestinationAbsent() {
        let state = AppShellState()
        let action = DiagnosticsActionItem(
            id: "open_connector_system_settings",
            title: "Open System Settings",
            intent: "open_system_settings",
            destination: nil,
            parameters: ["connector_id": "calendar"],
            enabled: true,
            recommended: true,
            reason: nil
        )

        let resolved = state.systemSettingsURLForConnectorAction(connectorID: "calendar", action: action)

        XCTAssertEqual(
            resolved.absoluteString,
            "x-apple.systempreferences:com.apple.preference.security?Privacy_Automation"
        )
    }

    func testChannelDiagnosticsOpenSystemSettingsIsPerformableAndUpdatesStatus() {
        let state = AppShellState()
        let action = DiagnosticsActionItem(
            id: "open_channel_system_settings",
            title: "Open Full Disk Access",
            intent: "open_system_settings",
            destination: "ui://system-settings/privacy/full-disk-access",
            parameters: [
                "channel_id": "message",
                "connector_id": "imessage"
            ],
            enabled: true,
            recommended: true,
            reason: nil
        )

        XCTAssertTrue(state.canPerformChannelDiagnosticsAction(channelID: "message", action: action))

        state.performChannelDiagnosticsAction(channelID: "message", action: action)

        XCTAssertEqual(
            state.channelsStatusMessage,
            "Opened System Settings for message channel remediation."
        )
    }

    func testSystemSettingsURLForChannelActionPrefersDaemonDestination() {
        let state = AppShellState()
        let action = DiagnosticsActionItem(
            id: "open_channel_system_settings",
            title: "Open Full Disk Access",
            intent: "open_system_settings",
            destination: "ui://system-settings/privacy/full-disk-access",
            parameters: ["channel_id": "message"],
            enabled: true,
            recommended: true,
            reason: nil
        )

        let resolved = state.systemSettingsURLForChannelAction(channelID: "message", action: action)

        XCTAssertEqual(
            resolved.absoluteString,
            "x-apple.systempreferences:com.apple.preference.security?Privacy_AllFiles"
        )
    }

    func testSystemSettingsURLForChannelActionFallsBackToMessageConnectorMapping() {
        let state = AppShellState()
        let action = DiagnosticsActionItem(
            id: "open_channel_system_settings",
            title: "Open System Settings",
            intent: "open_system_settings",
            destination: nil,
            parameters: ["channel_id": "message"],
            enabled: true,
            recommended: true,
            reason: nil
        )

        let resolved = state.systemSettingsURLForChannelAction(channelID: "message", action: action)

        XCTAssertEqual(
            resolved.absoluteString,
            "x-apple.systempreferences:com.apple.preference.security?Privacy_AllFiles"
        )
    }

    func testChannelDiagnosticsOpenSystemSettingsCanonicalizesLegacyMessageAlias() {
        let state = AppShellState()
        let action = DiagnosticsActionItem(
            id: "open_channel_system_settings",
            title: "Open iMessage Access",
            intent: "open_system_settings",
            destination: nil,
            parameters: ["channel_id": "imessage_sms_bridge"],
            enabled: true,
            recommended: true,
            reason: nil
        )

        XCTAssertTrue(state.canPerformChannelDiagnosticsAction(channelID: "imessage_sms_bridge", action: action))

        state.performChannelDiagnosticsAction(channelID: "imessage_sms_bridge", action: action)

        XCTAssertEqual(
            state.channelsStatusMessage,
            "Opened System Settings for message channel remediation."
        )
        XCTAssertEqual(
            state.systemSettingsURLForChannelAction(channelID: "imessage_sms_bridge", action: action).absoluteString,
            "x-apple.systempreferences:com.apple.preference.security?Privacy_AllFiles"
        )
    }

    func testChannelSetupActionNavigatesToChannelSpecificDestinations() {
        let setupAction = DiagnosticsActionItem(
            id: "open_channel_setup",
            title: "Open Channel Setup",
            intent: "navigate",
            destination: nil,
            parameters: [:],
            enabled: true,
            recommended: true,
            reason: nil
        )

        let appChatState = AppShellState()
        appChatState.performChannelDiagnosticsAction(channelID: "app_chat", action: setupAction)
        XCTAssertEqual(appChatState.selectedSection, .chat)
        XCTAssertEqual(appChatState.channelsStatusMessage, "Opened Chat for app channel checks.")

        let bridgeState = AppShellState()
        bridgeState.performChannelDiagnosticsAction(channelID: "imessage_sms_bridge", action: setupAction)
        XCTAssertEqual(bridgeState.selectedSection, .connectors)
        XCTAssertEqual(bridgeState.channelsStatusMessage, "Opened Connectors for message channel setup and permissions.")

        let messageState = AppShellState()
        messageState.performChannelDiagnosticsAction(channelID: "twilio_sms", action: setupAction)
        XCTAssertEqual(messageState.selectedSection, .connectors)
        XCTAssertEqual(messageState.channelsStatusMessage, "Opened Connectors for message channel setup and permissions.")

        let voiceState = AppShellState()
        voiceState.performChannelDiagnosticsAction(channelID: "twilio_voice", action: setupAction)
        XCTAssertEqual(voiceState.selectedSection, .configuration)
        XCTAssertEqual(voiceState.channelsStatusMessage, "Opened Configuration for voice channel setup.")

        let defaultState = AppShellState()
        defaultState.performChannelDiagnosticsAction(channelID: "unknown_channel", action: setupAction)
        XCTAssertEqual(defaultState.selectedSection, .configuration)
        XCTAssertEqual(defaultState.channelsStatusMessage, "Opened Configuration for channel setup actions.")
    }

    func testChannelSetupActionUsesDaemonDestinationWhenPresent() {
        let state = AppShellState()
        let action = DiagnosticsActionItem(
            id: "configure_twilio_channel",
            title: "Configure Twilio Credentials",
            intent: "navigate",
            destination: "ui://configuration/channels/twilio",
            parameters: ["channel_family": "twilio"],
            enabled: true,
            recommended: true,
            reason: nil
        )

        state.performChannelDiagnosticsAction(channelID: "twilio_sms", action: action)

        XCTAssertEqual(state.selectedSection, .configuration)
        XCTAssertEqual(state.channelsStatusMessage, "Opened Configuration for Twilio channel setup.")
    }

    func testUnsupportedChannelActionIsNotPerformable() {
        let state = AppShellState()
        let action = DiagnosticsActionItem(
            id: "unknown_channel_action",
            title: "Unknown",
            intent: "unknown",
            destination: "ui://system-settings/privacy",
            parameters: [:],
            enabled: true,
            recommended: false,
            reason: nil
        )

        XCTAssertFalse(state.canPerformChannelDiagnosticsAction(channelID: "app_chat", action: action))
    }

    func testConnectorPermissionActionIsPerformable() {
        let state = AppShellState()
        let action = DiagnosticsActionItem(
            id: "request_connector_permission",
            title: "Request Permission",
            intent: "request_permission",
            destination: "ui://connectors/request-permission/mail",
            parameters: ["connector_id": "mail"],
            enabled: true,
            recommended: true,
            reason: nil
        )

        XCTAssertTrue(state.canPerformConnectorDiagnosticsAction(connectorID: "mail", action: action))
    }

    func testUnsupportedConnectorActionIsNotPerformable() {
        let state = AppShellState()
        let action = DiagnosticsActionItem(
            id: "unknown_connector_action",
            title: "Unknown",
            intent: "unknown",
            destination: "ui://unknown/path",
            parameters: [:],
            enabled: true,
            recommended: false,
            reason: nil
        )

        XCTAssertFalse(state.canPerformConnectorDiagnosticsAction(connectorID: "mail", action: action))
    }

    func testChannelSetupFallbackActionTitleMatchesChannelDestination() {
        let state = AppShellState()

        XCTAssertEqual(state.channelSetupFallbackActionTitle(channelID: "app_chat"), "Open App Channel")
        XCTAssertEqual(state.channelSetupFallbackActionTitle(channelID: "imessage_sms_bridge"), "Open Message Setup")
        XCTAssertEqual(state.channelSetupFallbackActionTitle(channelID: "twilio_sms"), "Open Message Setup")
        XCTAssertEqual(state.channelSetupFallbackActionTitle(channelID: "twilio_voice"), "Open Voice Setup")
        XCTAssertEqual(state.channelSetupFallbackActionTitle(channelID: "unknown_channel"), "Open Channel Setup")
    }

    func testChannelDiagnosticsNavigationStatusUsesCanonicalChannelID() {
        let state = AppShellState()
        let action = DiagnosticsActionItem(
            id: "open_channel_tasks",
            title: "Open Tasks",
            intent: "navigate",
            destination: "ui://tasks",
            parameters: ["channel_id": "app_chat"],
            enabled: true,
            recommended: false,
            reason: nil
        )

        state.performChannelDiagnosticsAction(channelID: "app_chat", action: action)

        XCTAssertEqual(state.selectedSection, .tasks)
        XCTAssertEqual(state.channelsStatusMessage, "Opened Tasks for channel app diagnostics.")
    }
}
