import XCTest
@testable import PersonalAgentUI

@MainActor
final class AppShellStateLoadingStateTests: XCTestCase {
    func testRuntimeStatusBootstrapLoadingTracksLifecycleLoadFlag() {
        let state = AppShellState()
        state.hasLoadedDaemonStatus = false

        XCTAssertTrue(state.isRuntimeStatusBootstrapLoading)

        state.hasLoadedDaemonStatus = true
        XCTAssertFalse(state.isRuntimeStatusBootstrapLoading)
    }

    func testPanelLoadFlagsFlipTrueAfterFirstTokenlessRefresh() async {
        let state = AppShellState()
        state.clearLocalDevToken()

        state.refreshInspectLogs()
        state.refreshCommunicationsInbox()
        state.refreshAutomationTriggers()
        state.refreshApprovalsInbox()
        state.refreshTaskRunList()

        try? await Task.sleep(for: .milliseconds(120))

        XCTAssertTrue(state.hasLoadedInspectLogs)
        XCTAssertTrue(state.hasLoadedCommunicationsInbox)
        XCTAssertTrue(state.hasLoadedAutomationPanelData)
        XCTAssertTrue(state.hasLoadedApprovalsInbox)
        XCTAssertTrue(state.hasLoadedTaskRunList)
    }

    func testClearTokenResetsPanelLoadFlagsForSkeletonBootstrap() {
        let state = AppShellState()
        state.hasLoadedInspectLogs = true
        state.hasLoadedCommunicationsInbox = true
        state.hasLoadedAutomationPanelData = true
        state.hasLoadedApprovalsInbox = true
        state.hasLoadedTaskRunList = true

        state.clearLocalDevToken()

        XCTAssertFalse(state.hasLoadedInspectLogs)
        XCTAssertFalse(state.hasLoadedCommunicationsInbox)
        XCTAssertFalse(state.hasLoadedAutomationPanelData)
        XCTAssertFalse(state.hasLoadedApprovalsInbox)
        XCTAssertFalse(state.hasLoadedTaskRunList)
    }
}
