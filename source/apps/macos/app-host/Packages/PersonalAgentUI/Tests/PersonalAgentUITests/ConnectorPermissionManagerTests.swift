import XCTest
@testable import PersonalAgentUI

@MainActor
final class ConnectorPermissionManagerTests: XCTestCase {
    func testSystemSettingsURLMapsAutomationConnectorsToAutomationPane() {
        XCTAssertEqual(
            ConnectorPermissionManager.systemSettingsURL(for: "mail").absoluteString,
            "x-apple.systempreferences:com.apple.preference.security?Privacy_Automation"
        )
        XCTAssertEqual(
            ConnectorPermissionManager.systemSettingsURL(for: "calendar").absoluteString,
            "x-apple.systempreferences:com.apple.preference.security?Privacy_Automation"
        )
        XCTAssertEqual(
            ConnectorPermissionManager.systemSettingsURL(for: "browser").absoluteString,
            "x-apple.systempreferences:com.apple.preference.security?Privacy_Automation"
        )
        XCTAssertEqual(
            ConnectorPermissionManager.systemSettingsURL(for: "finder").absoluteString,
            "x-apple.systempreferences:com.apple.preference.security?Privacy_Automation"
        )
    }

    func testSystemSettingsURLUsesCanonicalIMessagesConnectorForFullDiskAccessPane() {
        XCTAssertEqual(
            ConnectorPermissionManager.systemSettingsURL(for: "messages").absoluteString,
            "x-apple.systempreferences:com.apple.preference.security?Privacy"
        )
        XCTAssertEqual(
            ConnectorPermissionManager.systemSettingsURL(for: "imessage").absoluteString,
            "x-apple.systempreferences:com.apple.preference.security?Privacy_AllFiles"
        )
    }

    func testSystemSettingsURLFallsBackToPrivacyPaneForUnknownConnector() {
        XCTAssertEqual(
            ConnectorPermissionManager.systemSettingsURL(for: "unknown-connector").absoluteString,
            "x-apple.systempreferences:com.apple.preference.security?Privacy"
        )
    }

    func testSystemSettingsURLFromDaemonDestinationMapsPrivacyAutomation() {
        XCTAssertEqual(
            ConnectorPermissionManager.systemSettingsURL(
                fromDaemonDestination: "ui://system-settings/privacy/automation"
            )?.absoluteString,
            "x-apple.systempreferences:com.apple.preference.security?Privacy_Automation"
        )
    }

    func testSystemSettingsURLFromDaemonDestinationMapsPrivacyFullDiskAccess() {
        XCTAssertEqual(
            ConnectorPermissionManager.systemSettingsURL(
                fromDaemonDestination: "ui://system-settings/privacy/full-disk-access"
            )?.absoluteString,
            "x-apple.systempreferences:com.apple.preference.security?Privacy_AllFiles"
        )
    }
}
