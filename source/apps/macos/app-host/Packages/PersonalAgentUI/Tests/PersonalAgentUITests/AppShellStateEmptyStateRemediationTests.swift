import XCTest
@testable import PersonalAgentUI

@MainActor
final class AppShellStateEmptyStateRemediationTests: XCTestCase {
    func testChatEmptyStateActionsWithoutTokenShowsConfigurationCTAOnly() {
        let state = AppShellState()
        state.clearLocalDevToken()
        state.chatRouteRemediationMessage = nil

        XCTAssertEqual(
            state.chatEmptyStateRemediationActions.map(\.actionID),
            [.openConfiguration]
        )
    }

    func testChatEmptyStateActionsWithRouteRemediationShowsModelsAndRecheck() {
        let state = AppShellState()
        state.localDevTokenConfigured = true
        state.chatRouteRemediationMessage = "No enabled chat model is ready."

        XCTAssertEqual(
            state.chatEmptyStateRemediationActions.map(\.actionID),
            [.openModels, .recheckChatRoute]
        )
    }

    func testCommunicationsEmptyStateActionsWithoutTokenPrioritizeConfiguration() {
        let state = AppShellState()
        state.clearLocalDevToken()

        XCTAssertEqual(
            state.communicationsEmptyStateRemediationActions.map(\.actionID),
            [.openConfiguration, .openChannels]
        )
    }

    func testModelsEmptyStateActionsWithTokenExposeRefreshAndChecks() {
        let state = AppShellState()
        state.localDevTokenConfigured = true

        XCTAssertEqual(
            state.modelsEmptyStateRemediationActions.map(\.actionID),
            [.refreshModels, .runProviderChecks]
        )
    }

    func testChatEmptyStateActionsUseConfigurationWhenDaemonReportsMissingAuth() {
        let state = AppShellState()
        state.localDevTokenConfigured = true
        state.hasLoadedDaemonStatus = true
        state.connectionStatus = .connected
        state.daemonControlAuthState = .missing
        state.chatRouteRemediationMessage = nil

        XCTAssertEqual(
            state.chatEmptyStateRemediationActions.map(\.actionID),
            [.openConfiguration]
        )
    }

    func testPerformEmptyStateRemediationNavigationActionsSelectExpectedSections() {
        let state = AppShellState()
        state.selectedSection = .chat

        state.performEmptyStateRemediationAction(.openChannels)
        XCTAssertEqual(state.selectedSection, .channels)
        XCTAssertEqual(state.channelsStatusMessage, "Opened Channels for remediation.")

        state.performEmptyStateRemediationAction(.openConnectors)
        XCTAssertEqual(state.selectedSection, .connectors)
        XCTAssertEqual(state.connectorsStatusMessage, "Opened Connectors for remediation.")

        state.performEmptyStateRemediationAction(.openModels)
        XCTAssertEqual(state.selectedSection, .models)

        state.performEmptyStateRemediationAction(.openTasks)
        XCTAssertEqual(state.selectedSection, .tasks)
    }
}
