import XCTest
@testable import PersonalAgentUI

@MainActor
final class AppInspectStoreTests: XCTestCase {
    func testTransitionInspectContextResetsSnapshotWhenRunChanges() {
        let store = AppInspectStore()
        store.inspectLogs = [
            InspectLogItem(
                id: "log-1",
                timestamp: Date(timeIntervalSince1970: 100),
                createdAtRaw: "2026-03-05T10:00:00Z",
                event: "task.step",
                status: .running,
                inputSummary: "in",
                outputSummary: "out",
                metadataSummary: "meta"
            )
        ]
        store.inspectCursorCreatedAt = "2026-03-05T10:00:00Z"
        store.inspectCursorID = "log-1"

        store.transitionInspectContext(
            focusedRunID: "run-123",
            searchSeed: nil,
            statusMessage: "Loading inspect logs for run run-123…"
        )

        XCTAssertEqual(store.inspectFocusedRunID, "run-123")
        XCTAssertTrue(store.inspectLogs.isEmpty)
        XCTAssertNil(store.inspectCursorCreatedAt)
        XCTAssertNil(store.inspectCursorID)
        XCTAssertEqual(store.inspectStatusMessage, "Loading inspect logs for run run-123…")
    }

    func testMapInspectLogRecordIncludesMetadataAndRouteSummary() throws {
        let store = AppInspectStore()
        let record = try decodeInspectLogRecord(
            from: """
            {
              "log_id": "log-1",
              "workspace_id": "ws1",
              "run_id": "run-9",
              "step_id": "step-3",
              "event_type": "APPROVAL.REQUIRED",
              "status": "awaiting_approval",
              "input_summary": "",
              "output_summary": "",
              "correlation_id": "corr-77",
              "created_at": "2026-03-05T10:00:00Z",
              "metadata": {
                "task_id": "task-5",
                "plugin": "mail"
              },
              "route": {
                "available": true,
                "task_class": "chat",
                "provider": "openai",
                "model_key": "gpt-4.1",
                "task_class_source": "policy",
                "route_source": "workspace_policy"
              }
            }
            """
        )

        let item = store.mapInspectLogRecord(
            record,
            parseDaemonTimestamp: { _ in Date(timeIntervalSince1970: 42) },
            mapWorkflowRoute: { route in
                WorkflowRouteContext(
                    available: route?.available ?? false,
                    taskClass: route?.taskClass,
                    provider: route?.provider,
                    modelKey: route?.modelKey,
                    taskClassSource: route?.taskClassSource,
                    routeSource: route?.routeSource,
                    notes: route?.notes
                )
            }
        )

        XCTAssertEqual(item.event, "approval.required")
        XCTAssertEqual(item.status, .running)
        XCTAssertEqual(item.inputSummary, "No input summary.")
        XCTAssertEqual(item.outputSummary, "No output summary.")
        XCTAssertEqual(item.taskID, "task-5")
        XCTAssertEqual(item.runID, "run-9")
        XCTAssertEqual(item.stepID, "step-3")
        XCTAssertEqual(item.correlationID, "corr-77")
        XCTAssertTrue(item.metadataSummary.contains("task_id=task-5"))
        XCTAssertTrue(item.metadataSummary.contains("provider=openai"))
        XCTAssertTrue(item.metadataSummary.contains("model_key=gpt-4.1"))
    }

    func testMergeInspectLogsDeduplicatesSortsAndTrims() {
        let store = AppInspectStore(maxInspectLogCount: 2)
        store.inspectLogs = [
            InspectLogItem(
                id: "log-older",
                timestamp: Date(timeIntervalSince1970: 10),
                createdAtRaw: "2026-03-05T10:00:10Z",
                event: "older",
                status: .running,
                inputSummary: "in",
                outputSummary: "out",
                metadataSummary: "meta"
            )
        ]

        store.mergeInspectLogs([
            InspectLogItem(
                id: "log-new",
                timestamp: Date(timeIntervalSince1970: 30),
                createdAtRaw: "2026-03-05T10:00:30Z",
                event: "new",
                status: .success,
                inputSummary: "in",
                outputSummary: "out",
                metadataSummary: "meta"
            ),
            InspectLogItem(
                id: "log-older",
                timestamp: Date(timeIntervalSince1970: 10),
                createdAtRaw: "2026-03-05T10:00:10Z",
                event: "duplicate",
                status: .failure,
                inputSummary: "in",
                outputSummary: "out",
                metadataSummary: "meta"
            ),
            InspectLogItem(
                id: "log-mid",
                timestamp: Date(timeIntervalSince1970: 20),
                createdAtRaw: "2026-03-05T10:00:20Z",
                event: "mid",
                status: .running,
                inputSummary: "in",
                outputSummary: "out",
                metadataSummary: "meta"
            )
        ])

        XCTAssertEqual(store.inspectLogs.map(\.id), ["log-new", "log-mid"])
        XCTAssertEqual(store.inspectCursorID, "log-new")
        XCTAssertEqual(store.inspectCursorCreatedAt, "2026-03-05T10:00:30Z")
    }

    func testApplyInspectQuerySnapshotBuildsStatusForFocusedRun() {
        let store = AppInspectStore()
        store.applyInspectQuerySnapshot(
            logs: [
                InspectLogItem(
                    id: "log-1",
                    timestamp: Date(timeIntervalSince1970: 10),
                    createdAtRaw: "2026-03-05T10:00:10Z",
                    event: "task.step",
                    status: .running,
                    inputSummary: "in",
                    outputSummary: "out",
                    metadataSummary: "meta"
                )
            ],
            focusedRunID: "run-123",
            workspaceID: "ws1"
        )

        XCTAssertEqual(store.inspectStatusMessage, "Loaded 1 inspect log(s) for run run-123.")
        XCTAssertEqual(store.inspectCursorID, "log-1")
    }

    private func decodeInspectLogRecord(from json: String) throws -> DaemonInspectLogRecord {
        let data = Data(json.utf8)
        return try JSONDecoder().decode(DaemonInspectLogRecord.self, from: data)
    }
}
