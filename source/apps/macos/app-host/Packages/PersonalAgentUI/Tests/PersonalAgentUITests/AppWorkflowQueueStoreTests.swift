import XCTest
@testable import PersonalAgentUI

@MainActor
final class AppWorkflowQueueStoreTests: XCTestCase {
    func testAutomationQueueReducersTrackInFlightAndPruneStatuses() {
        let store = AppWorkflowQueueStore()

        XCTAssertTrue(store.beginAutomationCreate())
        XCTAssertFalse(store.beginAutomationCreate())
        store.endAutomationCreate()
        XCTAssertFalse(store.isAutomationCreateInFlight)

        XCTAssertEqual(store.beginAutomationUpdate(triggerID: "  trig-1  "), "trig-1")
        XCTAssertNil(store.beginAutomationUpdate(triggerID: "trig-1"))
        store.setAutomationActionStatus(triggerID: "trig-1", message: "Updated")
        store.setAutomationActionStatus(triggerID: "stale", message: "Stale")
        store.pruneAutomationActionStatus(validTriggerIDs: ["trig-1"])

        XCTAssertEqual(store.automationActionStatusByID["trig-1"], "Updated")
        XCTAssertNil(store.automationActionStatusByID["stale"])
        store.endAutomationUpdate(triggerID: "trig-1")
        XCTAssertFalse(store.automationUpdateInFlightIDs.contains("trig-1"))

        XCTAssertEqual(store.beginAutomationDelete(triggerID: "trig-1"), "trig-1")
        store.endAutomationDelete(triggerID: "trig-1")
        XCTAssertFalse(store.automationDeleteInFlightIDs.contains("trig-1"))
    }

    func testTaskQueueReducersTrackSubmitAndRunControlLifecycle() {
        let store = AppWorkflowQueueStore()

        XCTAssertTrue(store.beginTaskSubmit())
        XCTAssertFalse(store.beginTaskSubmit())
        store.taskSubmitStatusMessage = "Submitted."
        store.endTaskSubmit()
        XCTAssertFalse(store.isTaskSubmitInFlight)
        XCTAssertEqual(store.taskSubmitStatusMessage, "Submitted.")

        XCTAssertTrue(store.canStartTaskRunControl(runID: " run-1 "))
        store.beginTaskRunControl(runID: " run-1 ", inFlightMessage: "Retrying run run-1…")
        XCTAssertFalse(store.canStartTaskRunControl(runID: "run-1"))
        XCTAssertEqual(store.taskRunControlStatusByRunID["run-1"], "Retrying run run-1…")

        store.setTaskRunControlStatus(
            runID: "run-1",
            updatedRunID: "run-2",
            message: "Retried run run-1 -> queued run run-2."
        )
        XCTAssertEqual(store.taskRunControlStatusByRunID["run-1"], "Retried run run-1 -> queued run run-2.")
        XCTAssertEqual(store.taskRunControlStatusByRunID["run-2"], "Retried run run-1 -> queued run run-2.")

        store.pruneTaskRunControlState(validRunIDs: ["run-2"])
        XCTAssertNil(store.taskRunControlStatusByRunID["run-1"])
        XCTAssertEqual(store.taskRunControlStatusByRunID["run-2"], "Retried run run-1 -> queued run run-2.")
        XCTAssertFalse(store.taskRunControlInFlightRunIDs.contains("run-1"))
    }

    func testApprovalDecisionReducersTrackInFlightAndStatus() {
        let store = AppWorkflowQueueStore()

        XCTAssertEqual(store.beginApprovalDecision(approvalID: " approval-1 "), "approval-1")
        XCTAssertNil(store.beginApprovalDecision(approvalID: "approval-1"))
        store.setApprovalActionStatus(approvalID: "approval-1", message: "Decision submitted: Approved. Run resumed.")
        XCTAssertEqual(store.approvalsActionStatusByID["approval-1"], "Decision submitted: Approved. Run resumed.")

        store.finishApprovalDecision(approvalID: "approval-1")
        XCTAssertFalse(store.approvalDecisionInFlightIDs.contains("approval-1"))
    }
}
