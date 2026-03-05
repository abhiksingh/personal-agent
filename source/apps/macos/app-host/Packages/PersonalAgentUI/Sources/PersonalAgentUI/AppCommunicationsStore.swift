import Foundation
import SwiftUI

@MainActor
final class AppCommunicationsStore: ObservableObject {
    @Published private(set) var communicationAttemptContextThreadID: String?

    func communicationsFilterContext(
        workspaceID: String?,
        contextRetentionStore: AppContextRetentionStore
    ) -> CommunicationsFilterContext {
        contextRetentionStore.panelFilterContext(for: workspaceID).communications
    }

    func setCommunicationsFilterContext(
        _ context: CommunicationsFilterContext,
        workspaceID: String?,
        contextRetentionStore: AppContextRetentionStore
    ) {
        contextRetentionStore.updatePanelFilterContext(for: workspaceID) { value in
            value.communications = context
        }
    }

    @discardableResult
    func resetCommunicationsFilterContext(
        workspaceID: String?,
        contextRetentionStore: AppContextRetentionStore
    ) -> CommunicationsFilterContext {
        let reset = CommunicationsFilterContext()
        setCommunicationsFilterContext(
            reset,
            workspaceID: workspaceID,
            contextRetentionStore: contextRetentionStore
        )
        return reset
    }

    func threadSelectionFilterContext(
        threadID: String,
        currentContext: CommunicationsFilterContext
    ) -> CommunicationsFilterContext {
        CommunicationsFilterContext(
            searchText: threadID,
            channelFilterID: CommunicationsFilterContext.allChannelsID,
            directionFilterRawValue: "all",
            threadFilterID: threadID,
            compactScanModeEnabled: currentContext.compactScanModeEnabled
        )
    }

    func communicationsTriageContext(
        workspaceID: String?,
        contextRetentionStore: AppContextRetentionStore
    ) -> CommunicationsTriageContext {
        contextRetentionStore.communicationsTriageContext(for: workspaceID)
    }

    func setCommunicationsTriageContext(
        _ context: CommunicationsTriageContext,
        workspaceID: String?,
        contextRetentionStore: AppContextRetentionStore
    ) {
        contextRetentionStore.setCommunicationsTriageContext(context, for: workspaceID)
    }

    func communicationsComposeDraftContext(
        workspaceID: String?,
        contextRetentionStore: AppContextRetentionStore
    ) -> CommunicationsComposeDraftContext? {
        contextRetentionStore.workspaceContinuityContext(for: workspaceID).communicationsComposeDraft
    }

    func setCommunicationsComposeDraftContext(
        _ context: CommunicationsComposeDraftContext?,
        workspaceID: String?,
        contextRetentionStore: AppContextRetentionStore
    ) {
        contextRetentionStore.updateWorkspaceContinuityContext(for: workspaceID) { value in
            value.communicationsComposeDraft = context
        }
    }

    @discardableResult
    func setCommunicationAttemptContextThreadID(_ threadID: String?) -> String? {
        let normalized = nonEmpty(threadID)
        communicationAttemptContextThreadID = normalized
        return normalized
    }

    func resetCommunicationAttemptContextThreadID() {
        communicationAttemptContextThreadID = nil
    }

    func mapCommunicationContinuityRecords(
        _ records: [DaemonChatTurnHistoryRecord],
        workspaceID: String,
        logicalCommunicationChannelID: (String) -> String,
        parseDaemonTimestamp: (String) -> Date?,
        formattedWorkflowTimestamp: (String) -> String,
        truncateText: (String, Int) -> String
    ) -> [CommunicationContinuityItem] {
        guard !records.isEmpty else {
            return []
        }

        var groupedByTurnKey: [String: [DaemonChatTurnHistoryRecord]] = [:]
        for record in records {
            let turnID = nonEmpty(record.turnID) ?? nonEmpty(record.recordID) ?? UUID().uuidString.lowercased()
            let channelID = logicalCommunicationChannelID(record.channelID)
            let connectorID = nonEmpty(record.connectorID) ?? "connector:none"
            let threadID = nonEmpty(record.threadID) ?? "thread:none"
            let correlationID = nonEmpty(record.correlationID) ?? "correlation:none"
            let key = [turnID, channelID, connectorID, threadID, correlationID].joined(separator: "::")
            groupedByTurnKey[key, default: []].append(record)
        }

        let mappedItems = groupedByTurnKey.values.compactMap { groupedRecords -> CommunicationContinuityItem? in
            let orderedRecords = groupedRecords.sorted { lhs, rhs in
                if lhs.itemIndex == rhs.itemIndex {
                    return lhs.createdAt > rhs.createdAt
                }
                return lhs.itemIndex < rhs.itemIndex
            }
            guard let latestRecord = orderedRecords.last else {
                return nil
            }

            let normalizedChannel = logicalCommunicationChannelID(latestRecord.channelID)
            let normalizedItemType = nonEmpty(latestRecord.item.type)?.lowercased() ?? "unknown"
            let normalizedStatus = nonEmpty(latestRecord.item.status)
                ?? nonEmpty(latestRecord.taskRunReference.runState)
                ?? "unknown"
            let summary = communicationContinuitySummary(
                item: latestRecord.item,
                itemType: normalizedItemType,
                fallbackStatus: normalizedStatus,
                truncateText: truncateText
            )
            let createdAtValue = nonEmpty(latestRecord.createdAt) ?? ""
            let sortTimestamp = parseDaemonTimestamp(createdAtValue) ?? .distantPast
            let turnID = nonEmpty(latestRecord.turnID) ?? nonEmpty(latestRecord.recordID) ?? UUID().uuidString.lowercased()
            let continuityID = "\(normalizedChannel)::\(turnID)::\(nonEmpty(latestRecord.correlationID) ?? "none")"
            let correlationID = nonEmpty(latestRecord.correlationID)
            let taskID = nonEmpty(latestRecord.taskRunReference.taskID)
            let runID = nonEmpty(latestRecord.taskRunReference.runID)
            let taskState = nonEmpty(latestRecord.taskRunReference.taskState)
            let runState = nonEmpty(latestRecord.taskRunReference.runState)
            let responseShaping = continuityResponseShaping(
                metadata: latestRecord.item.metadata,
                fallbackChannelID: normalizedChannel,
                logicalCommunicationChannelID: logicalCommunicationChannelID
            )

            return CommunicationContinuityItem(
                id: continuityID,
                turnID: turnID,
                workspaceID: nonEmpty(latestRecord.workspaceID) ?? workspaceID,
                channel: normalizedChannel,
                connectorID: nonEmpty(latestRecord.connectorID),
                threadID: nonEmpty(latestRecord.threadID),
                correlationID: correlationID,
                taskClass: nonEmpty(latestRecord.taskClass) ?? "chat",
                itemType: normalizedItemType,
                itemStatus: normalizedStatus,
                summary: summary,
                taskID: taskID,
                runID: runID,
                taskState: taskState,
                runState: runState,
                responseShapingChannel: responseShaping.channel,
                responseShapingProfile: responseShaping.profile,
                personaPolicySource: responseShaping.personaPolicySource,
                responseShapingGuardrailCount: responseShaping.guardrailCount,
                responseShapingInstructionCount: responseShaping.instructionCount,
                createdAtLabel: formattedWorkflowTimestamp(createdAtValue),
                sortTimestamp: sortTimestamp
            )
        }

        return mappedItems.sorted { lhs, rhs in
            if lhs.sortTimestamp == rhs.sortTimestamp {
                return lhs.id > rhs.id
            }
            return lhs.sortTimestamp > rhs.sortTimestamp
        }
    }

    private func communicationContinuitySummary(
        item: DaemonChatTurnItem,
        itemType: String,
        fallbackStatus: String,
        truncateText: (String, Int) -> String
    ) -> String {
        if let content = nonEmpty(item.content) {
            return truncateText(content, 180)
        }

        switch itemType {
        case "tool_call":
            let toolName = nonEmpty(item.toolName) ?? "Tool"
            let statusLabel = communicationContinuityStatusLabel(nonEmpty(item.status) ?? fallbackStatus)
            return "\(toolName) \(statusLabel)."
        case "tool_result":
            if let errorMessage = nonEmpty(item.errorMessage) {
                return truncateText(errorMessage, 180)
            }
            if let outputSummary = ChatTimelineEventStore.toolOutputSummary(item.output) {
                return truncateText(outputSummary, 180)
            }
            let toolName = nonEmpty(item.toolName) ?? "Tool"
            let statusLabel = communicationContinuityStatusLabel(nonEmpty(item.status) ?? fallbackStatus)
            return "\(toolName) result: \(statusLabel)."
        case "approval_request":
            if let approvalID = nonEmpty(item.approvalRequestID) {
                return "Approval requested (\(approvalID))."
            }
            return "Approval requested."
        case "approval_decision":
            let statusLabel = communicationContinuityStatusLabel(nonEmpty(item.status) ?? fallbackStatus)
            return "Approval decision: \(statusLabel)."
        case "user_message":
            return "User message sent."
        case "assistant_message":
            return "Assistant response recorded."
        default:
            let statusLabel = communicationContinuityStatusLabel(nonEmpty(item.status) ?? fallbackStatus)
            return "Turn item update: \(statusLabel)."
        }
    }

    private func communicationContinuityStatusLabel(_ rawStatus: String) -> String {
        switch rawStatus.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() {
        case "completed", "success", "done":
            return "completed"
        case "failed", "error":
            return "failed"
        case "awaiting_approval":
            return "awaiting approval"
        case "running", "in_progress":
            return "in progress"
        case "queued", "pending":
            return "queued"
        default:
            let trimmed = rawStatus.trimmingCharacters(in: .whitespacesAndNewlines)
            return trimmed.isEmpty ? "updated" : trimmed
        }
    }

    private func continuityResponseShaping(
        metadata: DaemonChatTurnItemMetadata?,
        fallbackChannelID: String,
        logicalCommunicationChannelID: (String) -> String
    ) -> (
        channel: String?,
        profile: String?,
        personaPolicySource: String?,
        guardrailCount: Int?,
        instructionCount: Int?
    ) {
        let resolvedChannel = continuityResponseShapingChannel(
            metadata?.responseShapingChannel,
            fallback: fallbackChannelID,
            logicalCommunicationChannelID: logicalCommunicationChannelID
        )
        return (
            channel: resolvedChannel,
            profile: nonEmpty(metadata?.responseShapingProfile)
                ?? continuityResponseShapingProfile(for: resolvedChannel),
            personaPolicySource: nonEmpty(metadata?.personaPolicySource),
            guardrailCount: metadata?.responseShapingGuardrailCount,
            instructionCount: metadata?.responseShapingInstructionCount
        )
    }

    private func continuityResponseShapingChannel(
        _ raw: String?,
        fallback: String,
        logicalCommunicationChannelID: (String) -> String
    ) -> String {
        let fallbackNormalized = logicalCommunicationChannelID(fallback)
        guard let raw = nonEmpty(raw) else {
            return fallbackNormalized
        }
        return logicalCommunicationChannelID(raw)
    }

    private func continuityResponseShapingProfile(for channelID: String?) -> String? {
        switch nonEmpty(channelID)?.lowercased() {
        case "message":
            return "message.compact"
        case "voice":
            return "voice.spoken"
        case "app":
            return "app.default"
        default:
            return nil
        }
    }

    private func nonEmpty(_ value: String?) -> String? {
        let trimmed = value?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        guard !trimmed.isEmpty else {
            return nil
        }
        return trimmed
    }
}
