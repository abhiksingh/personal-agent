import XCTest
@testable import PersonalAgentUI

@MainActor
final class AppShellStateInspectTests: XCTestCase {
    func testSetInspectLiveTailEnabledUpdatesToggleState() {
        let state = AppShellState()

        state.setInspectLiveTailEnabled(false)
        XCTAssertFalse(state.isInspectLiveTailEnabled)

        state.setInspectLiveTailEnabled(true)
        XCTAssertTrue(state.isInspectLiveTailEnabled)
    }

    func testOpenTasksForInspectLogSeedsRunIdentifierAndNavigates() {
        let state = AppShellState()
        let log = InspectLogItem(
            timestamp: .now,
            createdAtRaw: "2026-02-25T12:00:00Z",
            event: "task.step",
            status: .running,
            inputSummary: "input",
            outputSummary: "output",
            metadataSummary: "metadata",
            runID: "run-123"
        )

        state.openTasksForInspectLog(log)

        XCTAssertEqual(state.selectedSection, .tasks)
        XCTAssertEqual(state.tasksSearchSeed, "run-123")
        XCTAssertEqual(state.tasksStatusMessage, "Opened Tasks for inspect run run-123.")
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.sourceSection, .inspect)
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.destinationSection, .tasks)
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.chips.first, "Run: run-123")
    }

    func testOpenApprovalsForInspectLogSeedsTaskIdentifierAndNavigates() {
        let state = AppShellState()
        let log = InspectLogItem(
            timestamp: .now,
            createdAtRaw: "2026-02-25T12:00:00Z",
            event: "approval.required",
            status: .running,
            inputSummary: "input",
            outputSummary: "output",
            metadataSummary: "metadata",
            taskID: "task-789"
        )

        state.openApprovalsForInspectLog(log)

        XCTAssertEqual(state.selectedSection, .approvals)
        XCTAssertEqual(state.approvalsSearchSeed, "task-789")
        XCTAssertEqual(state.approvalsStatusMessage, "Opened Approvals for inspect task task-789.")
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.sourceSection, .inspect)
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.destinationSection, .approvals)
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.chips.first, "Task: task-789")
    }
}
