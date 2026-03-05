import XCTest
@testable import PersonalAgentUI

final class ProgressiveDisclosureDetailsTests: XCTestCase {
    func testChatWorkflowDetailsIncludeDeterministicTechnicalRows() {
        let traceability = ChatTaskRunTraceabilityItem(
            available: true,
            source: "execution",
            taskID: "task-123",
            runID: "run-123",
            taskState: "running",
            runState: "running",
            correlationID: "corr-123",
            taskClass: "chat",
            provider: "openai",
            modelKey: "gpt-4.1",
            routeSource: "policy"
        )

        let rows = ProgressiveDisclosureDetails.chatWorkflowDetails(
            traceability: traceability,
            routeSource: "policy",
            correlationID: "corr-123"
        )

        XCTAssertEqual(
            rows,
            [
                ProgressiveDisclosureDetailRow(label: "Route Source", value: "policy"),
                ProgressiveDisclosureDetailRow(label: "Task ID", value: "task-123"),
                ProgressiveDisclosureDetailRow(label: "Run ID", value: "run-123"),
                ProgressiveDisclosureDetailRow(label: "Task State", value: "running"),
                ProgressiveDisclosureDetailRow(label: "Run State", value: "running"),
                ProgressiveDisclosureDetailRow(label: "Trace Source", value: "Execution"),
                ProgressiveDisclosureDetailRow(label: "Correlation ID", value: "corr-123")
            ]
        )
    }

    func testChatWorkflowDetailsOmitEmptyValues() {
        let traceability = ChatTaskRunTraceabilityItem(
            available: false,
            source: "audit",
            taskID: "  ",
            runID: nil,
            taskState: "",
            runState: nil,
            correlationID: nil
        )

        let rows = ProgressiveDisclosureDetails.chatWorkflowDetails(
            traceability: traceability,
            routeSource: nil,
            correlationID: nil
        )

        XCTAssertEqual(rows, [ProgressiveDisclosureDetailRow(label: "Trace Source", value: "Audit")])
    }

    func testChatWorkflowDetailsIncludeWorkflowSignalRows() {
        let traceability = ChatTaskRunTraceabilityItem(
            available: true,
            source: "agent_run",
            taskID: "task-action-1",
            runID: "run-action-1",
            taskState: "awaiting_approval",
            runState: "awaiting_approval",
            correlationID: "corr-action-1",
            approvalRequired: true,
            approvalRequestID: "approval-123",
            clarificationRequired: true,
            clarificationPrompt: "Need recipient email address."
        )

        let rows = ProgressiveDisclosureDetails.chatWorkflowDetails(
            traceability: traceability,
            routeSource: "policy",
            correlationID: "corr-action-1"
        )

        XCTAssertTrue(rows.contains(ProgressiveDisclosureDetailRow(label: "Approval Required", value: "true")))
        XCTAssertTrue(rows.contains(ProgressiveDisclosureDetailRow(label: "Approval Request ID", value: "approval-123")))
        XCTAssertTrue(rows.contains(ProgressiveDisclosureDetailRow(label: "Clarification Required", value: "true")))
        XCTAssertTrue(rows.contains(ProgressiveDisclosureDetailRow(label: "Clarification Prompt", value: "Need recipient email address.")))
        XCTAssertTrue(rows.contains(ProgressiveDisclosureDetailRow(label: "Trace Source", value: "Agent Run")))
    }

    func testChatWorkflowDetailsIncludeContractV2RowsWhenAvailable() {
        let traceability = ChatTaskRunTraceabilityItem(
            available: true,
            source: "chat.turn",
            taskID: "task-contract-1",
            runID: "run-contract-1",
            taskState: "running",
            runState: "running",
            correlationID: "corr-contract-1",
            turnContractVersion: "chat_turn.v2",
            turnItemSchemaVersion: "chat_turn_item.v2",
            realtimeEventContractVersion: "chat_realtime_lifecycle.v2",
            realtimeLifecycleContractVersion: "chat_realtime_lifecycle.v2",
            realtimeLifecycleSchemaVersion: "chat_lifecycle_item.v2"
        )

        let rows = ProgressiveDisclosureDetails.chatWorkflowDetails(
            traceability: traceability,
            routeSource: "policy",
            correlationID: "corr-contract-1"
        )

        XCTAssertTrue(rows.contains(ProgressiveDisclosureDetailRow(label: "Turn Contract", value: "chat_turn.v2")))
        XCTAssertTrue(rows.contains(ProgressiveDisclosureDetailRow(label: "Turn Item Schema", value: "chat_turn_item.v2")))
        XCTAssertTrue(rows.contains(ProgressiveDisclosureDetailRow(label: "Realtime Event Contract", value: "chat_realtime_lifecycle.v2")))
        XCTAssertTrue(rows.contains(ProgressiveDisclosureDetailRow(label: "Realtime Lifecycle Contract", value: "chat_realtime_lifecycle.v2")))
        XCTAssertTrue(rows.contains(ProgressiveDisclosureDetailRow(label: "Realtime Lifecycle Schema", value: "chat_lifecycle_item.v2")))
    }

    func testCommunicationThreadDetailsIncludeExpectedRows() {
        let thread = CommunicationThreadItem(
            id: "thread-42",
            workspaceID: "ws1",
            channel: "message",
            connectorID: "imessage",
            title: "Family",
            externalRef: "ext-42",
            lastEventID: "event-42",
            lastEventType: "MESSAGE",
            lastDirection: "inbound",
            lastOccurredAtLabel: "now",
            lastBodyPreview: "hello",
            participantAddresses: ["mom@example.com"],
            eventCount: 3,
            createdAtLabel: "today",
            updatedAtLabel: "today",
            sortTimestamp: .now
        )

        XCTAssertEqual(
            ProgressiveDisclosureDetails.communicationThreadDetails(thread),
            [
                ProgressiveDisclosureDetailRow(label: "Thread ID", value: "thread-42"),
                ProgressiveDisclosureDetailRow(label: "External Reference", value: "ext-42"),
                ProgressiveDisclosureDetailRow(label: "Last Event ID", value: "event-42"),
                ProgressiveDisclosureDetailRow(label: "Last Event Type", value: "MESSAGE")
            ]
        )
    }

    func testCommunicationDeliveryAttemptDetailsIncludeWorkflowTraceRows() {
        let attempt = CommunicationDeliveryAttemptItem(
            id: "attempt-1",
            workspaceID: "ws1",
            operationID: "op-1",
            taskID: "task-1",
            runID: "run-1",
            stepID: "step-1",
            eventID: "event-1",
            threadID: "thread-1",
            destinationEndpoint: "+15550000001",
            idempotencyKey: "idem-1",
            channel: "message",
            routeIndex: 2,
            routePhase: "retry",
            retryOrdinal: 1,
            fallbackFromChannel: "imessage",
            status: "delivered",
            providerReceipt: "receipt-1",
            error: nil,
            attemptedAtLabel: "now",
            sortTimestamp: .now
        )

        XCTAssertEqual(
            ProgressiveDisclosureDetails.communicationDeliveryAttemptDetails(attempt),
            [
                ProgressiveDisclosureDetailRow(label: "Attempt ID", value: "attempt-1"),
                ProgressiveDisclosureDetailRow(label: "Operation ID", value: "op-1"),
                ProgressiveDisclosureDetailRow(label: "Idempotency Key", value: "idem-1"),
                ProgressiveDisclosureDetailRow(label: "Route Index", value: "2"),
                ProgressiveDisclosureDetailRow(label: "Retry Ordinal", value: "1"),
                ProgressiveDisclosureDetailRow(label: "Provider Receipt", value: "receipt-1"),
                ProgressiveDisclosureDetailRow(label: "Task ID", value: "task-1"),
                ProgressiveDisclosureDetailRow(label: "Run ID", value: "run-1"),
                ProgressiveDisclosureDetailRow(label: "Step ID", value: "step-1"),
                ProgressiveDisclosureDetailRow(label: "Event ID", value: "event-1")
            ]
        )
    }

    func testCommunicationSendReceiptDetailsOmitMissingThreadID() {
        let receipt = CommunicationSendReceiptItem(
            operationID: "op-send",
            threadID: nil,
            sourceChannel: "message",
            connectorID: "imessage",
            destination: "user@example.com",
            success: true,
            sentAt: .now
        )

        XCTAssertEqual(
            ProgressiveDisclosureDetails.communicationSendReceiptDetails(receipt),
            [ProgressiveDisclosureDetailRow(label: "Operation ID", value: "op-send")]
        )
    }
}
