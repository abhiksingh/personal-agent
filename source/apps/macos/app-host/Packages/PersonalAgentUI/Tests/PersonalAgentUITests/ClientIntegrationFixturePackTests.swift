import XCTest
@testable import PersonalAgentUI

final class ClientIntegrationFixturePackTests: XCTestCase {
    private let decoder = JSONDecoder()

    func testLifecycleControlAuthConfiguredFixtureDecodes() throws {
        let data = try ClientIntegrationFixtureLoader.data(named: "daemon_lifecycle_control_auth_configured.json")
        let decoded = try decoder.decode(DaemonLifecycleStatusResponse.self, from: data)

        XCTAssertEqual(decoded.lifecycleState, "running")
        XCTAssertEqual(decoded.controlAuth.state, "configured")
        XCTAssertEqual(decoded.controlAuth.source, "auth_token_file")
        XCTAssertEqual(decoded.controlAuth.remediationHints, [])
    }

    func testLifecycleControlAuthMissingFixtureDecodes() throws {
        let data = try ClientIntegrationFixtureLoader.data(named: "daemon_lifecycle_control_auth_missing.json")
        let decoded = try decoder.decode(DaemonLifecycleStatusResponse.self, from: data)

        XCTAssertEqual(decoded.lifecycleState, "stopped")
        XCTAssertEqual(decoded.controlAuth.state, "missing")
        XCTAssertEqual(decoded.controlAuth.source, "auth_token_file")
        XCTAssertFalse(decoded.controlAuth.remediationHints.isEmpty)
    }

    func testTaskRunActionDefaultsFixtureDecodes() throws {
        let data = try ClientIntegrationFixtureLoader.data(named: "task_run_list_actions_defaulted.json")
        let decoded = try decoder.decode(DaemonTaskRunListResponse.self, from: data)

        XCTAssertEqual(decoded.workspaceID, "ws1")
        XCTAssertEqual(decoded.items.count, 1)
        XCTAssertEqual(decoded.items.first?.taskID, "task-contract-list")
        XCTAssertEqual(decoded.items.first?.actions?.canCancel, false)
        XCTAssertEqual(decoded.items.first?.actions?.canRetry, false)
        XCTAssertEqual(decoded.items.first?.actions?.canRequeue, false)
    }

    func testChannelDescriptorDefaultsFixtureDecodes() throws {
        let data = try ClientIntegrationFixtureLoader.data(named: "channel_status_descriptor_defaults.json")
        let decoded = try decoder.decode(DaemonChannelStatusResponse.self, from: data)

        XCTAssertEqual(decoded.workspaceID, "ws1")
        XCTAssertEqual(decoded.channels.count, 1)
        XCTAssertEqual(decoded.channels.first?.channelID, "message")
        XCTAssertEqual(decoded.channels.first?.configFieldDescriptors.count, 1)
        XCTAssertEqual(decoded.channels.first?.configFieldDescriptors.first?.key, "poll_interval_seconds")
        XCTAssertEqual(decoded.channels.first?.configFieldDescriptors.first?.type, "number")
        XCTAssertEqual(decoded.channels.first?.configuration?["status_reason"]?.stringValue, "healthy")
    }

    func testConnectorDescriptorDefaultsFixtureDecodes() throws {
        let data = try ClientIntegrationFixtureLoader.data(named: "connector_status_descriptor_defaults.json")
        let decoded = try decoder.decode(DaemonConnectorStatusResponse.self, from: data)

        XCTAssertEqual(decoded.workspaceID, "ws1")
        XCTAssertEqual(decoded.connectors.count, 1)
        XCTAssertEqual(decoded.connectors.first?.connectorID, "twilio")
        XCTAssertEqual(decoded.connectors.first?.configFieldDescriptors.count, 1)
        XCTAssertEqual(decoded.connectors.first?.configFieldDescriptors.first?.key, "auth_token_secret_ref")
        XCTAssertTrue(decoded.connectors.first?.configFieldDescriptors.first?.secret ?? false)
        XCTAssertTrue(decoded.connectors.first?.configFieldDescriptors.first?.writeOnly ?? false)
        XCTAssertEqual(decoded.connectors.first?.configuration?["status_reason"]?.stringValue, "healthy")
    }
}
