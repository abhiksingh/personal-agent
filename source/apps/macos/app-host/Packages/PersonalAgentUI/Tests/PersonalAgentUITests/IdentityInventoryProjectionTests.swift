import Foundation
import XCTest
@testable import PersonalAgentUI

final class IdentityInventoryProjectionTests: XCTestCase {
    func testHealthFilterNormalizationAndLimitClamping() {
        XCTAssertNil(IdentityInventoryProjection.normalizedSessionHealthFilter(""))
        XCTAssertNil(IdentityInventoryProjection.normalizedSessionHealthFilter(" all "))
        XCTAssertEqual(
            IdentityInventoryProjection.normalizedSessionHealthFilter("HEALTHY"),
            "healthy"
        )

        XCTAssertEqual(IdentityInventoryProjection.clampedQueryLimit(-1), 25)
        XCTAssertEqual(IdentityInventoryProjection.clampedQueryLimit(25), 25)
        XCTAssertEqual(IdentityInventoryProjection.clampedQueryLimit(300), 200)
    }

    func testDeviceMappingSortingAndSummaryMessage() throws {
        let older = try decodeDeviceRecord(
            deviceID: "device-1",
            workspaceID: "ws-one",
            createdAt: "2026-03-05T09:00:00Z"
        )
        let newer = try decodeDeviceRecord(
            deviceID: "device-2",
            workspaceID: "ws-two",
            createdAt: "2026-03-05T10:00:00Z"
        )
        let mapped = IdentityInventoryProjection.mapAndSortDeviceItems(
            [older, newer],
            fallbackWorkspaceID: "fallback",
            canonicalWorkspaceID: { raw in
                raw?.uppercased()
            },
            parseTimestamp: parseISO8601Timestamp,
            formatTimestamp: { value in
                "formatted(\(value))"
            }
        )

        XCTAssertEqual(mapped.map(\.id), ["device-2", "device-1"])
        XCTAssertEqual(mapped.first?.workspaceID, "WS-TWO")
        XCTAssertEqual(mapped.first?.createdAtLabel, "formatted(2026-03-05T10:00:00Z)")

        let summary = IdentityInventoryProjection.deviceSummaryMessage(
            itemCount: 2,
            hasMore: true,
            userID: "actor.requester.ws1",
            deviceType: "desktop",
            platform: "macos"
        )
        XCTAssertEqual(
            summary,
            "Identity devices • 2 item(s) • user=actor.requester.ws1 • device_type=desktop • platform=macos • more available"
        )
    }

    func testSessionMappingPruningAndSummaryMessage() throws {
        let first = try decodeSessionRecord(
            sessionID: "session-1",
            workspaceID: "ws-one",
            startedAt: "2026-03-05T09:00:00Z",
            expiresAt: "2026-03-06T09:00:00Z",
            sessionHealth: "HEALTHY"
        )
        let second = try decodeSessionRecord(
            sessionID: "session-2",
            workspaceID: "ws-two",
            startedAt: "2026-03-05T10:00:00Z",
            expiresAt: "2026-03-06T10:00:00Z",
            sessionHealth: "DEGRADED"
        )
        let mapped = IdentityInventoryProjection.mapAndSortSessionItems(
            [first, second],
            fallbackWorkspaceID: "fallback",
            canonicalWorkspaceID: { raw in
                raw?.uppercased()
            },
            parseTimestamp: parseISO8601Timestamp,
            formatTimestamp: { value in
                "formatted(\(value))"
            }
        )
        XCTAssertEqual(mapped.map(\.id), ["session-2", "session-1"])
        XCTAssertEqual(mapped.first?.sessionHealth, "degraded")

        let pruned = IdentityInventoryProjection.pruneSessionActionState(
            activeSessionIDs: Set(["session-2"]),
            actionStatusByID: ["session-1": "stale", "session-2": "active"],
            revokeInFlightIDs: Set(["session-1", "session-2"])
        )
        XCTAssertEqual(pruned.actionStatusByID, ["session-2": "active"])
        XCTAssertEqual(pruned.revokeInFlightIDs, Set(["session-2"]))

        let summary = IdentityInventoryProjection.sessionSummaryMessage(
            itemCount: 1,
            hasMore: false,
            deviceID: "device-2",
            userID: "actor.requester.ws1",
            sessionHealth: "degraded"
        )
        XCTAssertEqual(
            summary,
            "Identity sessions • 1 item(s) • device=device-2 • user=actor.requester.ws1 • health=degraded"
        )
    }

    private func parseISO8601Timestamp(_ raw: String) -> Date? {
        ISO8601DateFormatter().date(from: raw)
    }

    private func decodeDeviceRecord(
        deviceID: String,
        workspaceID: String,
        createdAt: String
    ) throws -> DaemonIdentityDeviceRecord {
        let json = """
        {
          "device_id": "\(deviceID)",
          "workspace_id": "\(workspaceID)",
          "user_id": "actor.requester.ws1",
          "device_type": "desktop",
          "platform": "macos",
          "label": "My Mac",
          "last_seen_at": "2026-03-05T10:00:00Z",
          "created_at": "\(createdAt)",
          "session_total": 4,
          "session_active_count": 2,
          "session_expired_count": 1,
          "session_revoked_count": 1,
          "session_latest_started_at": "2026-03-05T10:00:00Z"
        }
        """
        return try JSONDecoder().decode(
            DaemonIdentityDeviceRecord.self,
            from: Data(json.utf8)
        )
    }

    private func decodeSessionRecord(
        sessionID: String,
        workspaceID: String,
        startedAt: String,
        expiresAt: String,
        sessionHealth: String
    ) throws -> DaemonIdentitySessionRecord {
        let json = """
        {
          "session_id": "\(sessionID)",
          "workspace_id": "\(workspaceID)",
          "device_id": "device-2",
          "user_id": "actor.requester.ws1",
          "device_type": "desktop",
          "platform": "macos",
          "device_label": "My Mac",
          "device_last_seen_at": "2026-03-05T10:00:00Z",
          "started_at": "\(startedAt)",
          "expires_at": "\(expiresAt)",
          "revoked_at": null,
          "session_health": "\(sessionHealth)"
        }
        """
        return try JSONDecoder().decode(
            DaemonIdentitySessionRecord.self,
            from: Data(json.utf8)
        )
    }
}
