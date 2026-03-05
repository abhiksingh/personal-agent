import Foundation
import SwiftUI

@MainActor
final class AppInspectStore: ObservableObject {
    @Published var isInspectLoading = false
    @Published var hasLoadedInspectLogs = false
    @Published var inspectStatusMessage: String? = "Checking activity feed."
    @Published var isInspectLiveTailEnabled = true
    @Published var inspectFocusedRunID: String? = nil
    @Published var inspectSearchSeed: String? = nil
    @Published var inspectLogs: [InspectLogItem] = []
    @Published var inspectCursorCreatedAt: String? = nil
    @Published var inspectCursorID: String? = nil

    private let maxInspectLogCount: Int

    init(maxInspectLogCount: Int = 200) {
        self.maxInspectLogCount = maxInspectLogCount
    }

    @discardableResult
    func transitionInspectContext(
        focusedRunID: String?,
        searchSeed: String?,
        statusMessage: String? = nil,
        forceResetSnapshot: Bool = false
    ) -> Bool {
        let runChanged = inspectFocusedRunID != focusedRunID
        if runChanged {
            inspectFocusedRunID = focusedRunID
            resetInspectSnapshot()
        } else if forceResetSnapshot {
            resetInspectSnapshot()
        }
        inspectSearchSeed = searchSeed
        if let statusMessage {
            inspectStatusMessage = statusMessage
        }
        return runChanged
    }

    @discardableResult
    func clearInspectRunFocus() -> Bool {
        guard inspectFocusedRunID != nil else {
            return false
        }
        inspectFocusedRunID = nil
        resetInspectSnapshot()
        inspectStatusMessage = "Inspect run filter cleared."
        return true
    }

    func resetInspectSnapshot() {
        inspectCursorCreatedAt = nil
        inspectCursorID = nil
        inspectLogs = []
    }

    func applyInspectQuerySnapshot(
        logs: [InspectLogItem],
        focusedRunID: String?,
        workspaceID: String
    ) {
        inspectLogs = logs
        inspectCursorCreatedAt = logs.first?.createdAtRaw
        inspectCursorID = logs.first?.id

        if logs.isEmpty {
            if let focusedRunID {
                inspectStatusMessage = "No inspect logs returned yet for run \(focusedRunID)."
            } else {
                inspectStatusMessage = "No inspect logs returned for workspace \(workspaceID)."
            }
        } else if let focusedRunID {
            inspectStatusMessage = "Loaded \(logs.count) inspect log(s) for run \(focusedRunID)."
        } else {
            inspectStatusMessage = "Live inspect logs loaded (\(logs.count))."
        }
    }

    func mergeInspectLogs(_ incoming: [InspectLogItem]) {
        var seen = Set<String>()
        var merged: [InspectLogItem] = []

        for item in incoming + inspectLogs {
            if seen.insert(item.id).inserted {
                merged.append(item)
            }
        }

        merged.sort { lhs, rhs in
            if lhs.timestamp == rhs.timestamp {
                return lhs.id > rhs.id
            }
            return lhs.timestamp > rhs.timestamp
        }

        if merged.count > maxInspectLogCount {
            merged = Array(merged.prefix(maxInspectLogCount))
        }

        inspectLogs = merged
        inspectCursorCreatedAt = merged.first?.createdAtRaw
        inspectCursorID = merged.first?.id
    }

    func updateInspectCursor(createdAt: String?, cursorID: String?) {
        guard let cursorCreatedAt = nonEmpty(createdAt),
              let cursorID = nonEmpty(cursorID)
        else {
            return
        }
        inspectCursorCreatedAt = cursorCreatedAt
        inspectCursorID = cursorID
    }

    func mapInspectLogRecord(
        _ record: DaemonInspectLogRecord,
        parseDaemonTimestamp: (String) -> Date?,
        mapWorkflowRoute: (DaemonWorkflowRouteMetadata?) -> WorkflowRouteContext
    ) -> InspectLogItem {
        let taskID = extractMetadataIdentifier(
            record.metadata,
            candidates: ["task_id", "taskID", "taskId", "task"]
        )
        let runID = nonEmpty(record.runID)
        let stepID = nonEmpty(record.stepID)
        let correlationID = nonEmpty(record.correlationID)
        let route = mapWorkflowRoute(record.route)

        var metadataParts = (record.metadata ?? [:]).keys.sorted().compactMap { key -> String? in
            guard let value = record.metadata?[key] else {
                return nil
            }
            return "\(key)=\(value.displayText)"
        }
        if let taskID {
            metadataParts.append("task_id=\(taskID)")
        }
        if let runID {
            metadataParts.append("run_id=\(runID)")
        }
        if let stepID {
            metadataParts.append("step_id=\(stepID)")
        }
        if let correlationID {
            metadataParts.append("correlation_id=\(correlationID)")
        }
        if route.available {
            if let taskClass = route.taskClass {
                metadataParts.append("task_class=\(taskClass)")
            }
            if let provider = route.provider {
                metadataParts.append("provider=\(provider)")
            }
            if let modelKey = route.modelKey {
                metadataParts.append("model_key=\(modelKey)")
            }
            if let taskClassSource = route.taskClassSource {
                metadataParts.append("task_class_source=\(taskClassSource)")
            }
            if let routeSource = route.routeSource {
                metadataParts.append("route_source=\(routeSource)")
            }
        }

        let fixedInput = nonEmpty(record.inputSummary) ?? "No input summary."
        let fixedOutput = nonEmpty(record.outputSummary) ?? "No output summary."
        let metadataSummary = metadataParts.isEmpty
            ? "run=\(runID ?? "-"), step=\(stepID ?? "-")"
            : metadataParts.joined(separator: ", ")

        return InspectLogItem(
            id: record.logID,
            timestamp: parseDaemonTimestamp(record.createdAt) ?? .now,
            createdAtRaw: record.createdAt,
            event: record.eventType.lowercased(),
            status: inspectStatus(from: record.status),
            inputSummary: fixedInput,
            outputSummary: fixedOutput,
            metadataSummary: metadataSummary,
            taskID: taskID,
            runID: runID,
            stepID: stepID,
            correlationID: correlationID,
            route: route
        )
    }

    private func inspectStatus(from rawStatus: String) -> InspectLogStatus {
        switch rawStatus.lowercased() {
        case "success", "completed", "approved", "ready", "info":
            return .success
        case "running", "starting", "awaiting_approval":
            return .running
        case "failed", "failure", "error", "denied":
            return .failure
        default:
            return .running
        }
    }

    private func extractMetadataIdentifier(
        _ metadata: [String: DaemonJSONValue]?,
        candidates: [String]
    ) -> String? {
        guard let metadata else {
            return nil
        }
        for key in candidates {
            if let value = metadata[key] {
                if let string = nonEmpty(value.stringValue) {
                    return string
                }
                if let fallback = nonEmpty(value.displayText), fallback != "null" {
                    return fallback
                }
            }
        }
        return nil
    }

    private func nonEmpty(_ value: String?) -> String? {
        let trimmed = value?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        guard !trimmed.isEmpty else {
            return nil
        }
        return trimmed
    }
}
