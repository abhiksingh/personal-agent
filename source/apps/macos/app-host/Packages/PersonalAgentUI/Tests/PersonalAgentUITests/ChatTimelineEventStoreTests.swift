import XCTest
@testable import PersonalAgentUI

@MainActor
final class ChatTimelineEventStoreTests: XCTestCase {
    func testAssistantStreamingDeltaReconcilesToSingleCompletedReceiptItem() {
        let store = ChatTimelineEventStore()
        store.resetForNewTurn(existingTimeline: [])

        store.appendStreamingDelta("Hello", activeCorrelationID: "corr-1", itemID: "assistant-1")
        store.appendStreamingDelta(", world", activeCorrelationID: "corr-1", itemID: "assistant-1")

        let receiptItems = [
            DaemonChatTurnItem(
                itemID: "assistant-1",
                type: "assistant_message",
                role: "assistant",
                status: "completed",
                content: "Hello, world!"
            )
        ]

        store.reconcileChatTurnTimelineItems(
            items: receiptItems,
            correlationID: "corr-1",
            taskCorrelation: DaemonChatTurnTaskRunCorrelation(),
            activeCorrelationID: "corr-1"
        )

        let assistantItems = store.timelineItems.filter { $0.kind == .assistantMessage }
        XCTAssertEqual(assistantItems.count, 1)
        XCTAssertEqual(assistantItems.first?.state, .completed)
        XCTAssertEqual(assistantItems.first?.content, "Hello, world!")
    }

    func testAssistantReceiptPreservesLeadingAndTrailingWhitespace() {
        let store = ChatTimelineEventStore()
        store.resetForNewTurn(existingTimeline: [])

        store.reconcileChatTurnTimelineItems(
            items: [
                DaemonChatTurnItem(
                    itemID: "assistant-whitespace",
                    type: "assistant_message",
                    role: "assistant",
                    status: "completed",
                    content: "  hello from assistant\n"
                )
            ],
            correlationID: "corr-whitespace",
            taskCorrelation: DaemonChatTurnTaskRunCorrelation(),
            activeCorrelationID: "corr-whitespace"
        )

        let assistantItems = store.timelineItems.filter { $0.kind == .assistantMessage }
        XCTAssertEqual(assistantItems.count, 1)
        XCTAssertEqual(assistantItems.first?.content, "  hello from assistant\n")
    }

    func testRealtimeToolEventsRemainPrimaryWhileReceiptReconcilesByToolCallID() {
        let store = ChatTimelineEventStore()
        store.resetForNewTurn(existingTimeline: [])

        let realtimeStart = realtimeEvent(
            type: "tool_call_started",
            correlationID: "corr-2",
            payload: [
                "tool_name": .string("mail.send"),
                "tool_call_id": .string("call-1"),
                "task_id": .string("task-1"),
                "run_id": .string("run-1")
            ]
        )
        store.appendTimelineFromRealtimeEvent(realtimeStart, correlationID: "corr-2", activeCorrelationID: "corr-2")

        let realtimeOutput = realtimeEvent(
            type: "tool_call_output",
            correlationID: "corr-2",
            payload: [
                "tool_name": .string("mail.send"),
                "tool_call_id": .string("call-1"),
                "status": .string("running"),
                "output": .string("Preparing message")
            ]
        )
        store.appendTimelineFromRealtimeEvent(realtimeOutput, correlationID: "corr-2", activeCorrelationID: "corr-2")

        XCTAssertEqual(store.timelineItems.filter { $0.kind == .toolCall }.count, 1)
        XCTAssertEqual(store.timelineItems.filter { $0.kind == .toolResult }.count, 1)

        let receiptItems = [
            DaemonChatTurnItem(
                itemID: "item-tool-call",
                type: "tool_call",
                status: "running",
                toolName: "mail.send",
                toolCallID: "call-1"
            ),
            DaemonChatTurnItem(
                itemID: "item-tool-result",
                type: "tool_result",
                status: "completed",
                toolName: "mail.send",
                toolCallID: "call-1",
                output: ["result": .string("sent")]
            )
        ]

        store.reconcileChatTurnTimelineItems(
            items: receiptItems,
            correlationID: "corr-2",
            taskCorrelation: DaemonChatTurnTaskRunCorrelation(
                available: true,
                source: "audit",
                taskID: "task-1",
                runID: "run-1"
            ),
            activeCorrelationID: "corr-2"
        )

        XCTAssertEqual(store.timelineItems.filter { $0.kind == .toolCall }.count, 1)
        XCTAssertEqual(store.timelineItems.filter { $0.kind == .toolResult }.count, 1)

        let toolCallItem = store.timelineItems.first { $0.kind == .toolCall }
        let toolResultItem = store.timelineItems.first { $0.kind == .toolResult }
        XCTAssertEqual(toolCallItem?.toolCallID, "call-1")
        XCTAssertEqual(toolResultItem?.toolCallID, "call-1")
        XCTAssertEqual(toolResultItem?.state, .completed)
    }

    func testTurnItemDeltaPreservesWhitespaceAndNewlines() {
        let store = ChatTimelineEventStore()
        store.resetForNewTurn(existingTimeline: [])

        let deltas = ["Hello", " ", "world", "\nline2"]
        for delta in deltas {
            let deltaEvent = realtimeEvent(
                type: "turn_item_delta",
                correlationID: "corr-stream",
                payload: [
                    "item_type": .string("assistant_message"),
                    "delta": .string(delta)
                ]
            )
            store.appendTimelineFromRealtimeEvent(
                deltaEvent,
                correlationID: "corr-stream",
                activeCorrelationID: "corr-stream"
            )
        }

        let assistantItems = store.timelineItems.filter { $0.kind == .assistantMessage }
        XCTAssertEqual(assistantItems.count, 1)
        XCTAssertEqual(assistantItems.first?.state, .inFlight)
        XCTAssertEqual(assistantItems.first?.content, "Hello world\nline2")
    }

    func testAssistantTurnLifecycleReusesSingleTimelineItemID() {
        let store = ChatTimelineEventStore()
        store.resetForNewTurn(existingTimeline: [])

        let assistantItemID = "assistant-stream-1"
        store.appendTimelineFromRealtimeEvent(
            realtimeEvent(
                type: "turn_item_started",
                correlationID: "corr-assistant",
                payload: [
                    "item_type": .string("assistant_message"),
                    "item_id": .string(assistantItemID),
                    "status": .string("started")
                ]
            ),
            correlationID: "corr-assistant",
            activeCorrelationID: "corr-assistant"
        )

        store.appendTimelineFromRealtimeEvent(
            realtimeEvent(
                type: "turn_item_delta",
                correlationID: "corr-assistant",
                payload: [
                    "item_type": .string("assistant_message"),
                    "item_id": .string(assistantItemID),
                    "delta": .string("Line 1\n- item")
                ]
            ),
            correlationID: "corr-assistant",
            activeCorrelationID: "corr-assistant"
        )

        store.appendTimelineFromRealtimeEvent(
            realtimeEvent(
                type: "turn_item_completed",
                correlationID: "corr-assistant",
                payload: [
                    "item_type": .string("assistant_message"),
                    "item_id": .string(assistantItemID),
                    "status": .string("completed")
                ]
            ),
            correlationID: "corr-assistant",
            activeCorrelationID: "corr-assistant"
        )

        let assistantItems = store.timelineItems.filter { $0.kind == .assistantMessage }
        XCTAssertEqual(assistantItems.count, 1)
        XCTAssertEqual(assistantItems.first?.id, assistantItemID)
        XCTAssertEqual(assistantItems.first?.content, "Line 1\n- item")
        XCTAssertEqual(assistantItems.first?.state, .completed)
    }

    func testAssistantDeltaBeforeLifecycleDoesNotDuplicateAssistantRows() {
        let store = ChatTimelineEventStore()
        store.resetForNewTurn(existingTimeline: [])

        store.appendTimelineFromRealtimeEvent(
            realtimeEvent(
                type: "turn_item_delta",
                correlationID: "corr-assistant-2",
                payload: [
                    "item_type": .string("assistant_message"),
                    "delta": .string("Hello")
                ]
            ),
            correlationID: "corr-assistant-2",
            activeCorrelationID: "corr-assistant-2"
        )

        store.appendTimelineFromRealtimeEvent(
            realtimeEvent(
                type: "turn_item_started",
                correlationID: "corr-assistant-2",
                payload: [
                    "item_type": .string("assistant_message"),
                    "item_id": .string("assistant-canonical"),
                    "status": .string("started")
                ]
            ),
            correlationID: "corr-assistant-2",
            activeCorrelationID: "corr-assistant-2"
        )

        store.appendTimelineFromRealtimeEvent(
            realtimeEvent(
                type: "turn_item_delta",
                correlationID: "corr-assistant-2",
                payload: [
                    "item_type": .string("assistant_message"),
                    "item_id": .string("assistant-canonical"),
                    "delta": .string(" world")
                ]
            ),
            correlationID: "corr-assistant-2",
            activeCorrelationID: "corr-assistant-2"
        )

        store.appendTimelineFromRealtimeEvent(
            realtimeEvent(
                type: "turn_item_completed",
                correlationID: "corr-assistant-2",
                payload: [
                    "item_type": .string("assistant_message"),
                    "item_id": .string("assistant-canonical"),
                    "status": .string("completed")
                ]
            ),
            correlationID: "corr-assistant-2",
            activeCorrelationID: "corr-assistant-2"
        )

        let assistantItems = store.timelineItems.filter { $0.kind == .assistantMessage }
        XCTAssertEqual(assistantItems.count, 1)
        XCTAssertEqual(assistantItems.first?.id, "assistant-canonical")
        XCTAssertEqual(assistantItems.first?.content, "Hello world")
        XCTAssertEqual(assistantItems.first?.state, .completed)
    }

    func testAssistantDeltaBeforeLifecycleThenReceiptKeepsSingleFormattedAssistantRow() {
        let store = ChatTimelineEventStore()
        store.resetForNewTurn(existingTimeline: [])

        store.appendTimelineFromRealtimeEvent(
            realtimeEvent(
                type: "turn_item_delta",
                correlationID: "corr-assistant-3",
                payload: [
                    "item_type": .string("assistant_message"),
                    "delta": .string("I can help with:\\n- email")
                ]
            ),
            correlationID: "corr-assistant-3",
            activeCorrelationID: "corr-assistant-3"
        )

        store.appendTimelineFromRealtimeEvent(
            realtimeEvent(
                type: "turn_item_started",
                correlationID: "corr-assistant-3",
                payload: [
                    "item_type": .string("assistant_message"),
                    "item_id": .string("assistant-canonical-3"),
                    "status": .string("started")
                ]
            ),
            correlationID: "corr-assistant-3",
            activeCorrelationID: "corr-assistant-3"
        )

        store.reconcileChatTurnTimelineItems(
            items: [
                DaemonChatTurnItem(
                    itemID: "assistant-canonical-3",
                    type: "assistant_message",
                    role: "assistant",
                    status: "completed",
                    content: "I can help with:\n- email\n- calendar"
                )
            ],
            correlationID: "corr-assistant-3",
            taskCorrelation: DaemonChatTurnTaskRunCorrelation(),
            activeCorrelationID: "corr-assistant-3"
        )

        let assistantItems = store.timelineItems.filter { $0.kind == .assistantMessage }
        XCTAssertEqual(assistantItems.count, 1)
        XCTAssertEqual(assistantItems.first?.id, "assistant-canonical-3")
        XCTAssertEqual(assistantItems.first?.state, .completed)
        XCTAssertEqual(assistantItems.first?.content, "I can help with:\n- email\n- calendar")
    }

    private func realtimeEvent(
        type: String,
        correlationID: String,
        payload: [String: DaemonJSONValue]
    ) -> DaemonRealtimeEventEnvelope {
        let outputPayload: [String: DaemonJSONValue]? = {
            if let object = payload["output"]?.objectValue {
                return object
            }
            if let rawString = payload["output"]?.stringValue {
                return ["message": .string(rawString)]
            }
            return nil
        }()

        let realtimePayload = DaemonRealtimeEventPayload(
            taskID: payload["task_id"]?.stringValue,
            runID: payload["run_id"]?.stringValue,
            itemID: payload["item_id"]?.stringValue,
            itemType: payload["item_type"]?.stringValue,
            status: payload["status"]?.stringValue,
            delta: payload["delta"]?.stringValue,
            toolName: payload["tool_name"]?.stringValue ?? payload["name"]?.stringValue,
            toolCallID: payload["tool_call_id"]?.stringValue ?? payload["call_id"]?.stringValue,
            output: outputPayload,
            approvalRequestID: payload["approval_request_id"]?.stringValue,
            message: payload["message"]?.stringValue,
            additional: payload
        )

        return DaemonRealtimeEventEnvelope(
            eventID: UUID().uuidString.lowercased(),
            sequence: 1,
            eventType: type,
            occurredAt: "2026-03-03T00:00:00Z",
            correlationID: correlationID,
            contractVersion: nil,
            lifecycleSchemaVersion: nil,
            payload: realtimePayload
        )
    }
}
