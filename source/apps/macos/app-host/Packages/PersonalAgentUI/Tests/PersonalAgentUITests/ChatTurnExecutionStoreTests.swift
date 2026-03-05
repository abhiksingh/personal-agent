import XCTest
@testable import PersonalAgentUI

final class ChatTurnExecutionStoreTests: XCTestCase {
    func testRecoveredChatTurnSnapshotSelectsLatestTurnAndOrdersByItemIndex() throws {
        let payload = """
        {
          "workspace_id": "ws1",
          "items": [
            {
              "record_id": "record-old-1",
              "turn_id": "turn-old",
              "workspace_id": "ws1",
              "task_class": "chat",
              "correlation_id": "corr-123",
              "channel_id": "app",
              "item_index": 0,
              "item": { "type": "assistant_message", "content": "old" },
              "task_run_reference": { "task_id": "task-old", "run_id": "run-old", "run_state": "completed" },
              "created_at": "2026-03-04T10:00:00Z"
            },
            {
              "record_id": "record-new-2",
              "turn_id": "turn-new",
              "workspace_id": "ws1",
              "task_class": "chat",
              "correlation_id": "corr-123",
              "channel_id": "app",
              "item_index": 1,
              "item": { "type": "assistant_message", "content": "done" },
              "task_run_reference": { "task_id": "task-new", "run_id": "run-new", "run_state": "completed" },
              "created_at": "2026-03-04T11:00:02Z"
            },
            {
              "record_id": "record-new-1",
              "turn_id": "turn-new",
              "workspace_id": "ws1",
              "task_class": "chat",
              "correlation_id": "corr-123",
              "channel_id": "app",
              "item_index": 0,
              "item": { "type": "user_message", "content": "hello" },
              "task_run_reference": { "task_id": "task-new", "run_id": "run-new", "run_state": "completed" },
              "created_at": "2026-03-04T11:00:01Z"
            }
          ],
          "has_more": false
        }
        """

        let response = try JSONDecoder().decode(
            DaemonChatTurnHistoryResponse.self,
            from: Data(payload.utf8)
        )
        let formatter = ISO8601DateFormatter()
        let snapshot = ChatTurnExecutionStore.recoveredChatTurnSnapshot(
            from: response,
            correlationID: "corr-123",
            parseDaemonTimestamp: { formatter.date(from: $0) }
        )

        XCTAssertNotNil(snapshot)
        XCTAssertEqual(snapshot?.workspaceID, "ws1")
        XCTAssertEqual(snapshot?.correlationID, "corr-123")
        XCTAssertEqual(snapshot?.taskClass, "chat")
        XCTAssertEqual(snapshot?.channelID, "app")
        XCTAssertEqual(snapshot?.items.map(\.type), ["user_message", "assistant_message"])
        XCTAssertEqual(snapshot?.taskRunCorrelation.taskID, "task-new")
        XCTAssertEqual(snapshot?.taskRunCorrelation.runID, "run-new")
    }

    func testRecoveredChatTurnSnapshotReturnsNilWhenCorrelationMissing() throws {
        let payload = """
        {
          "workspace_id": "ws1",
          "items": [
            {
              "record_id": "record-1",
              "turn_id": "turn-1",
              "workspace_id": "ws1",
              "task_class": "chat",
              "correlation_id": "corr-present",
              "channel_id": "app",
              "item_index": 0,
              "item": { "type": "user_message", "content": "hello" },
              "task_run_reference": { "task_id": "task-1", "run_id": "run-1", "run_state": "completed" },
              "created_at": "2026-03-04T10:00:01Z"
            }
          ],
          "has_more": false
        }
        """

        let response = try JSONDecoder().decode(
            DaemonChatTurnHistoryResponse.self,
            from: Data(payload.utf8)
        )
        let formatter = ISO8601DateFormatter()
        let snapshot = ChatTurnExecutionStore.recoveredChatTurnSnapshot(
            from: response,
            correlationID: "corr-missing",
            parseDaemonTimestamp: { formatter.date(from: $0) }
        )

        XCTAssertNil(snapshot)
    }
}
