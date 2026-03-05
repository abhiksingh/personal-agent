import Foundation
import XCTest
@testable import PersonalAgentUI

final class RuntimePluginLifecycleProjectionTests: XCTestCase {
    func testNormalizedFilterAndLimitClampingPreserveCurrentDefaults() {
        XCTAssertNil(RuntimePluginLifecycleProjection.normalizedFilter(""))
        XCTAssertNil(RuntimePluginLifecycleProjection.normalizedFilter(" all "))
        XCTAssertEqual(RuntimePluginLifecycleProjection.normalizedFilter("worker"), "worker")

        XCTAssertEqual(RuntimePluginLifecycleProjection.clampedLimit(-1), 40)
        XCTAssertEqual(RuntimePluginLifecycleProjection.clampedLimit(40), 40)
        XCTAssertEqual(RuntimePluginLifecycleProjection.clampedLimit(201), 200)
    }

    func testMapAndSortRecordsAndSummaryMessage() throws {
        let newer = try decodeRecord(
            auditID: "audit-2",
            pluginID: "messages.daemon",
            kind: "CHANNEL",
            state: "RUNNING",
            eventType: "PLUGIN_HEALTHY",
            occurredAt: "2026-03-05T10:00:00Z"
        )
        let older = try decodeRecord(
            auditID: "audit-1",
            pluginID: "messages.daemon",
            kind: "channel",
            state: "degraded",
            eventType: "PLUGIN_FAILURE",
            occurredAt: "2026-03-05T09:00:00Z"
        )
        let sorted = RuntimePluginLifecycleProjection.mapAndSortRecords(
            [older, newer],
            parseTimestamp: parseISO8601Timestamp,
            formatTimestamp: { value in
                "formatted(\(value))"
            }
        )

        XCTAssertEqual(sorted.map(\.id), ["audit-2", "audit-1"])
        XCTAssertEqual(sorted.first?.kind, "channel")
        XCTAssertEqual(sorted.first?.state, "running")
        XCTAssertEqual(sorted.first?.occurredAtLabel, "formatted(2026-03-05T10:00:00Z)")

        let summary = RuntimePluginLifecycleProjection.summaryMessage(
            itemCount: 2,
            hasMore: true,
            pluginID: "messages.daemon",
            kind: "channel",
            state: "running",
            eventType: "PLUGIN_HEALTHY"
        )
        XCTAssertEqual(
            summary,
            "Runtime lifecycle history • 2 event(s) • plugin=messages.daemon • kind=channel • state=running • event=PLUGIN_HEALTHY • more available"
        )
    }

    func testBuildTrendItemsAggregatesPerPluginAndSortsByEventVolume() {
        let now = Date()
        let items: [RuntimePluginLifecycleEventItem] = [
            RuntimePluginLifecycleEventItem(
                id: "a-1",
                workspaceID: "ws1",
                pluginID: "messages.daemon",
                kind: "channel",
                state: "running",
                eventType: "PLUGIN_HEALTHY",
                processID: 10,
                restartCount: 1,
                reason: "ok",
                error: nil,
                restartEvent: true,
                failureEvent: false,
                recoveryEvent: true,
                lastHeartbeatAtLabel: nil,
                lastTransitionAtLabel: nil,
                occurredAtLabel: "now",
                sortTimestamp: now
            ),
            RuntimePluginLifecycleEventItem(
                id: "a-0",
                workspaceID: "ws1",
                pluginID: "messages.daemon",
                kind: "channel",
                state: "degraded",
                eventType: "PLUGIN_FAILURE",
                processID: 9,
                restartCount: 1,
                reason: "fail",
                error: "error",
                restartEvent: false,
                failureEvent: true,
                recoveryEvent: false,
                lastHeartbeatAtLabel: nil,
                lastTransitionAtLabel: nil,
                occurredAtLabel: "earlier",
                sortTimestamp: now.addingTimeInterval(-10)
            ),
            RuntimePluginLifecycleEventItem(
                id: "b-1",
                workspaceID: "ws1",
                pluginID: "calendar.daemon",
                kind: "connector",
                state: "running",
                eventType: "PLUGIN_HEALTHY",
                processID: 11,
                restartCount: 0,
                reason: "ok",
                error: nil,
                restartEvent: false,
                failureEvent: false,
                recoveryEvent: true,
                lastHeartbeatAtLabel: nil,
                lastTransitionAtLabel: nil,
                occurredAtLabel: "now",
                sortTimestamp: now
            ),
        ]

        let trends = RuntimePluginLifecycleProjection.buildTrendItems(from: items)
        XCTAssertEqual(trends.map(\.pluginID), ["messages.daemon", "calendar.daemon"])
        XCTAssertEqual(trends.first?.totalEvents, 2)
        XCTAssertEqual(trends.first?.restartEvents, 1)
        XCTAssertEqual(trends.first?.failureEvents, 1)
        XCTAssertEqual(trends.first?.recoveryEvents, 1)
    }

    private func parseISO8601Timestamp(_ raw: String) -> Date? {
        ISO8601DateFormatter().date(from: raw)
    }

    private func decodeRecord(
        auditID: String,
        pluginID: String,
        kind: String,
        state: String,
        eventType: String,
        occurredAt: String
    ) throws -> DaemonPluginLifecycleHistoryRecord {
        let json = """
        {
          "audit_id": "\(auditID)",
          "workspace_id": "ws1",
          "plugin_id": "\(pluginID)",
          "kind": "\(kind)",
          "state": "\(state)",
          "event_type": "\(eventType)",
          "process_id": 12,
          "restart_count": 1,
          "reason": "ok",
          "error": null,
          "restart_event": false,
          "failure_event": false,
          "recovery_event": true,
          "last_heartbeat_at": null,
          "last_transition_at": null,
          "occurred_at": "\(occurredAt)"
        }
        """
        return try JSONDecoder().decode(
            DaemonPluginLifecycleHistoryRecord.self,
            from: Data(json.utf8)
        )
    }
}
