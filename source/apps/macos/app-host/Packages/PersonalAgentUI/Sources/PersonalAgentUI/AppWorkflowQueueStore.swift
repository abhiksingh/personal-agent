import Foundation
import SwiftUI

@MainActor
final class AppWorkflowQueueStore: ObservableObject {
    @Published var isAutomationCreateInFlight = false
    @Published var automationUpdateInFlightIDs: Set<String> = []
    @Published var automationDeleteInFlightIDs: Set<String> = []
    @Published var automationActionStatusByID: [String: String] = [:]

    @Published var approvalsActionStatusByID: [String: String] = [:]
    @Published var approvalDecisionInFlightIDs: Set<String> = []

    @Published var taskRunControlStatusByRunID: [String: String] = [:]
    @Published var taskRunControlInFlightRunIDs: Set<String> = []
    @Published var isTaskSubmitInFlight = false
    @Published var taskSubmitStatusMessage: String? = "No task submit action run yet."
    @Published var latestTaskSubmissionReceipt: TaskSubmissionReceiptItem? = nil

    func beginAutomationCreate() -> Bool {
        guard !isAutomationCreateInFlight else {
            return false
        }
        isAutomationCreateInFlight = true
        return true
    }

    func endAutomationCreate() {
        isAutomationCreateInFlight = false
    }

    func beginAutomationUpdate(triggerID: String) -> String? {
        let normalizedID = trimmedIdentifier(triggerID)
        guard !normalizedID.isEmpty else {
            return nil
        }
        guard !automationUpdateInFlightIDs.contains(normalizedID) else {
            return nil
        }
        automationUpdateInFlightIDs.insert(normalizedID)
        return normalizedID
    }

    func endAutomationUpdate(triggerID: String) {
        let normalizedID = trimmedIdentifier(triggerID)
        guard !normalizedID.isEmpty else {
            return
        }
        automationUpdateInFlightIDs.remove(normalizedID)
    }

    func beginAutomationDelete(triggerID: String) -> String? {
        let normalizedID = trimmedIdentifier(triggerID)
        guard !normalizedID.isEmpty else {
            return nil
        }
        guard !automationDeleteInFlightIDs.contains(normalizedID) else {
            return nil
        }
        automationDeleteInFlightIDs.insert(normalizedID)
        return normalizedID
    }

    func endAutomationDelete(triggerID: String) {
        let normalizedID = trimmedIdentifier(triggerID)
        guard !normalizedID.isEmpty else {
            return
        }
        automationDeleteInFlightIDs.remove(normalizedID)
    }

    func setAutomationActionStatus(triggerID: String, message: String) {
        let normalizedID = trimmedIdentifier(triggerID)
        guard !normalizedID.isEmpty else {
            return
        }
        automationActionStatusByID[normalizedID] = message
    }

    func pruneAutomationActionStatus(validTriggerIDs: Set<String>) {
        automationActionStatusByID = automationActionStatusByID.filter { validTriggerIDs.contains($0.key) }
    }

    func beginTaskSubmit() -> Bool {
        guard !isTaskSubmitInFlight else {
            return false
        }
        isTaskSubmitInFlight = true
        return true
    }

    func endTaskSubmit() {
        isTaskSubmitInFlight = false
    }

    func canStartTaskRunControl(runID: String) -> Bool {
        let normalizedRunID = trimmedIdentifier(runID)
        guard !normalizedRunID.isEmpty else {
            return false
        }
        return !taskRunControlInFlightRunIDs.contains(normalizedRunID)
    }

    func beginTaskRunControl(runID: String, inFlightMessage: String) {
        let normalizedRunID = trimmedIdentifier(runID)
        guard !normalizedRunID.isEmpty else {
            return
        }
        taskRunControlInFlightRunIDs.insert(normalizedRunID)
        taskRunControlStatusByRunID[normalizedRunID] = inFlightMessage
    }

    func finishTaskRunControl(runID: String) {
        let normalizedRunID = trimmedIdentifier(runID)
        guard !normalizedRunID.isEmpty else {
            return
        }
        taskRunControlInFlightRunIDs.remove(normalizedRunID)
    }

    func setTaskRunControlStatus(runID: String, updatedRunID: String? = nil, message: String) {
        let normalizedRunID = trimmedIdentifier(runID)
        guard !normalizedRunID.isEmpty else {
            return
        }
        taskRunControlStatusByRunID[normalizedRunID] = message
        if let updatedRunID {
            let normalizedUpdatedID = trimmedIdentifier(updatedRunID)
            if !normalizedUpdatedID.isEmpty {
                taskRunControlStatusByRunID[normalizedUpdatedID] = message
            }
        }
    }

    func pruneTaskRunControlState(validRunIDs: Set<String>) {
        taskRunControlStatusByRunID = taskRunControlStatusByRunID.filter { validRunIDs.contains($0.key) }
        taskRunControlInFlightRunIDs.formIntersection(validRunIDs)
    }

    func beginApprovalDecision(approvalID: String) -> String? {
        let normalizedID = trimmedIdentifier(approvalID)
        guard !normalizedID.isEmpty else {
            return nil
        }
        guard !approvalDecisionInFlightIDs.contains(normalizedID) else {
            return nil
        }
        approvalDecisionInFlightIDs.insert(normalizedID)
        return normalizedID
    }

    func finishApprovalDecision(approvalID: String) {
        let normalizedID = trimmedIdentifier(approvalID)
        guard !normalizedID.isEmpty else {
            return
        }
        approvalDecisionInFlightIDs.remove(normalizedID)
    }

    func setApprovalActionStatus(approvalID: String, message: String) {
        let normalizedID = trimmedIdentifier(approvalID)
        guard !normalizedID.isEmpty else {
            return
        }
        approvalsActionStatusByID[normalizedID] = message
    }

    private func trimmedIdentifier(_ value: String) -> String {
        value.trimmingCharacters(in: .whitespacesAndNewlines)
    }
}
