import Foundation

struct ProgressiveDisclosureDetailRow: Sendable, Equatable, Identifiable {
    let label: String
    let value: String

    var id: String { "\(label)::\(value)" }
}

enum ProgressiveDisclosureDetails {
    static func chatWorkflowDetails(
        traceability: ChatTaskRunTraceabilityItem?,
        routeSource: String?,
        correlationID: String?
    ) -> [ProgressiveDisclosureDetailRow] {
        detailRows(
            [
                ("Route Source", routeSource),
                ("Turn Contract", traceability?.turnContractVersion),
                ("Turn Item Schema", traceability?.turnItemSchemaVersion),
                ("Realtime Event Contract", traceability?.realtimeEventContractVersion),
                ("Realtime Lifecycle Contract", traceability?.realtimeLifecycleContractVersion),
                ("Realtime Lifecycle Schema", traceability?.realtimeLifecycleSchemaVersion),
                ("Task ID", traceability?.taskID),
                ("Run ID", traceability?.runID),
                ("Task State", traceability?.taskState),
                ("Run State", traceability?.runState),
                ("Shaping Channel", traceability?.responseShapingChannel),
                ("Shaping Profile", traceability?.responseShapingProfile),
                ("Persona Source", traceability?.personaPolicySource),
                ("Shaping Guardrails", traceability?.responseShapingGuardrailCount.map(String.init)),
                ("Shaping Instructions", traceability?.responseShapingInstructionCount.map(String.init)),
                ("Approval Required", traceability?.approvalRequired == true ? "true" : nil),
                ("Approval Request ID", traceability?.approvalRequestID),
                ("Clarification Required", traceability?.clarificationRequired == true ? "true" : nil),
                ("Clarification Prompt", traceability?.clarificationPrompt),
                ("Trace Source", traceSourceLabel(traceability?.source)),
                ("Correlation ID", correlationID)
            ]
        )
    }

    static func communicationThreadDetails(_ item: CommunicationThreadItem) -> [ProgressiveDisclosureDetailRow] {
        detailRows(
            [
                ("Thread ID", item.id),
                ("External Reference", item.externalRef),
                ("Last Event ID", item.lastEventID),
                ("Last Event Type", item.lastEventType)
            ]
        )
    }

    static func communicationEventDetails(_ item: CommunicationEventItem) -> [ProgressiveDisclosureDetailRow] {
        detailRows(
            [
                ("Event ID", item.id),
                ("Thread ID", item.threadID),
                ("Connector ID", item.connectorID)
            ]
        )
    }

    static func communicationCallSessionDetails(_ item: CommunicationCallSessionItem) -> [ProgressiveDisclosureDetailRow] {
        detailRows(
            [
                ("Session ID", item.id),
                ("Provider Call ID", item.providerCallID),
                ("Thread ID", item.threadID)
            ]
        )
    }

    static func communicationDeliveryAttemptDetails(
        _ item: CommunicationDeliveryAttemptItem
    ) -> [ProgressiveDisclosureDetailRow] {
        detailRows(
            [
                ("Attempt ID", item.id),
                ("Operation ID", item.operationID),
                ("Idempotency Key", item.idempotencyKey),
                ("Route Index", "\(item.routeIndex)"),
                ("Retry Ordinal", "\(item.retryOrdinal)"),
                ("Provider Receipt", item.providerReceipt),
                ("Task ID", item.taskID),
                ("Run ID", item.runID),
                ("Step ID", item.stepID),
                ("Event ID", item.eventID)
            ]
        )
    }

    static func communicationSendReceiptDetails(_ item: CommunicationSendReceiptItem) -> [ProgressiveDisclosureDetailRow] {
        detailRows(
            [
                ("Operation ID", item.operationID),
                ("Thread ID", item.threadID)
            ]
        )
    }

    private static func detailRows(
        _ fields: [(String, String?)]
    ) -> [ProgressiveDisclosureDetailRow] {
        fields.compactMap { field in
            guard let value = nonEmpty(field.1) else {
                return nil
            }
            return ProgressiveDisclosureDetailRow(label: field.0, value: value)
        }
    }

    private static func traceSourceLabel(_ source: String?) -> String? {
        guard let source = nonEmpty(source) else {
            return nil
        }
        switch source.lowercased() {
        case "audit":
            return "Audit"
        case "execution":
            return "Execution"
        case "agent_run":
            return "Agent Run"
        case "none":
            return "None"
        default:
            return source.capitalized
        }
    }

    private static func nonEmpty(_ value: String?) -> String? {
        guard let value else {
            return nil
        }
        let trimmed = value.trimmingCharacters(in: .whitespacesAndNewlines)
        return trimmed.isEmpty ? nil : trimmed
    }
}
