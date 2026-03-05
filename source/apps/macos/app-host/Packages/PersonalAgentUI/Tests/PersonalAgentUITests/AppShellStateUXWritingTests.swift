import XCTest
@testable import PersonalAgentUI

@MainActor
final class AppShellStateUXWritingTests: XCTestCase {
    func testWorkflowDefaultStatusMessagesUseCheckingLanguage() {
        let state = AppShellState()

        XCTAssertEqual(state.chatStatusMessage, "Checking assistant connection.")
        XCTAssertEqual(state.inspectStatusMessage, "Checking activity feed.")
        XCTAssertEqual(state.communicationsStatusMessage, "Checking communications inbox.")
        XCTAssertEqual(state.approvalsStatusMessage, "Checking approvals inbox.")
        XCTAssertEqual(state.tasksStatusMessage, "Checking tasks and runs.")
    }

    func testUnknownDaemonAuthStateUsesCheckingCopy() {
        let state = AppShellState()
        state.localDevTokenConfigured = false
        state.hasLoadedDaemonStatus = true
        state.connectionStatus = .connected
        state.daemonControlAuthState = .unknown

        XCTAssertEqual(state.daemonControlAuthSetupDetail, "Checking daemon auth state.")
    }

    func testDistributionTrustGuidanceIncludesGatekeeperOverrideSteps() {
        let state = AppShellState()

        XCTAssertTrue(state.distributionTrustGuidanceSummary.contains("unsigned"))
        XCTAssertTrue(
            state.distributionTrustGuidanceChecklist.contains { step in
                step.contains("Control-click") || step.contains("Right-click")
            }
        )
        XCTAssertTrue(
            state.distributionTrustGuidanceChecklist.contains { step in
                step.contains("Open Anyway")
            }
        )
        XCTAssertTrue(
            state.distributionTrustGuidanceChecklist.contains { step in
                step.contains("Personal Agent Daemon")
            }
        )
    }

    func testDistributionTrustRetryGuidanceMentionsRetryFlow() {
        let state = AppShellState()
        XCTAssertTrue(state.distributionTrustRetryGuidance.contains("retry"))
        XCTAssertTrue(state.distributionTrustRetryGuidance.contains("System Settings"))
    }
}
