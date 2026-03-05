import Foundation

enum IdentityInventoryProjection {
    static let defaultQueryLimit = 25
    static let maxQueryLimit = 200
    static let defaultHealthFilterSelection = "all"
    static let missingTokenDeviceMessage = "Set Assistant Access Token to query identity devices."
    static let missingTokenSessionMessage = "Set Assistant Access Token to query identity sessions."

    static func normalizedSessionHealthFilter(_ raw: String) -> String? {
        let trimmed = raw.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        guard !trimmed.isEmpty, trimmed != defaultHealthFilterSelection else {
            return nil
        }
        return trimmed
    }

    static func clampedQueryLimit(_ raw: Int) -> Int {
        switch raw {
        case ..<1:
            return defaultQueryLimit
        case 1...maxQueryLimit:
            return raw
        default:
            return maxQueryLimit
        }
    }

    static func mapAndSortDeviceItems(
        _ records: [DaemonIdentityDeviceRecord],
        fallbackWorkspaceID: String,
        canonicalWorkspaceID: (String?) -> String?,
        parseTimestamp: (String) -> Date?,
        formatTimestamp: (String) -> String
    ) -> [IdentityDeviceItem] {
        records.map {
            mapDeviceItem(
                $0,
                fallbackWorkspaceID: fallbackWorkspaceID,
                canonicalWorkspaceID: canonicalWorkspaceID,
                parseTimestamp: parseTimestamp,
                formatTimestamp: formatTimestamp
            )
        }
        .sorted { lhs, rhs in
            if lhs.sortTimestamp == rhs.sortTimestamp {
                return lhs.id > rhs.id
            }
            return lhs.sortTimestamp > rhs.sortTimestamp
        }
    }

    static func mapAndSortSessionItems(
        _ records: [DaemonIdentitySessionRecord],
        fallbackWorkspaceID: String,
        canonicalWorkspaceID: (String?) -> String?,
        parseTimestamp: (String) -> Date?,
        formatTimestamp: (String) -> String
    ) -> [IdentitySessionItem] {
        records.map {
            mapSessionItem(
                $0,
                fallbackWorkspaceID: fallbackWorkspaceID,
                canonicalWorkspaceID: canonicalWorkspaceID,
                parseTimestamp: parseTimestamp,
                formatTimestamp: formatTimestamp
            )
        }
        .sorted { lhs, rhs in
            if lhs.sortTimestamp == rhs.sortTimestamp {
                return lhs.id > rhs.id
            }
            return lhs.sortTimestamp > rhs.sortTimestamp
        }
    }

    static func deviceSummaryMessage(
        itemCount: Int,
        hasMore: Bool,
        userID: String?,
        deviceType: String?,
        platform: String?
    ) -> String {
        var parts: [String] = ["Identity devices • \(itemCount) item(s)"]
        if let userID {
            parts.append("user=\(userID)")
        }
        if let deviceType {
            parts.append("device_type=\(deviceType)")
        }
        if let platform {
            parts.append("platform=\(platform)")
        }
        if hasMore {
            parts.append("more available")
        }
        return parts.joined(separator: " • ")
    }

    static func sessionSummaryMessage(
        itemCount: Int,
        hasMore: Bool,
        deviceID: String?,
        userID: String?,
        sessionHealth: String?
    ) -> String {
        var parts: [String] = ["Identity sessions • \(itemCount) item(s)"]
        if let deviceID {
            parts.append("device=\(deviceID)")
        }
        if let userID {
            parts.append("user=\(userID)")
        }
        if let sessionHealth {
            parts.append("health=\(sessionHealth)")
        }
        if hasMore {
            parts.append("more available")
        }
        return parts.joined(separator: " • ")
    }

    static func pruneSessionActionState(
        activeSessionIDs: Set<String>,
        actionStatusByID: [String: String],
        revokeInFlightIDs: Set<String>
    ) -> (actionStatusByID: [String: String], revokeInFlightIDs: Set<String>) {
        (
            actionStatusByID.filter { activeSessionIDs.contains($0.key) },
            Set(revokeInFlightIDs.filter { activeSessionIDs.contains($0) })
        )
    }

    private static func mapDeviceItem(
        _ record: DaemonIdentityDeviceRecord,
        fallbackWorkspaceID: String,
        canonicalWorkspaceID: (String?) -> String?,
        parseTimestamp: (String) -> Date?,
        formatTimestamp: (String) -> String
    ) -> IdentityDeviceItem {
        let deviceID = nonEmpty(record.deviceID) ?? UUID().uuidString.lowercased()
        let createdAt = parseTimestamp(record.createdAt) ?? .distantPast
        let normalizedWorkspaceID = canonicalWorkspaceID(nonEmpty(record.workspaceID)) ?? fallbackWorkspaceID
        return IdentityDeviceItem(
            id: deviceID,
            workspaceID: normalizedWorkspaceID,
            userID: nonEmpty(record.userID) ?? "unknown",
            deviceType: nonEmpty(record.deviceType) ?? "unknown",
            platform: nonEmpty(record.platform) ?? "unknown",
            label: nonEmpty(record.label),
            lastSeenAtLabel: nonEmpty(record.lastSeenAt).map(formatTimestamp),
            createdAtLabel: nonEmpty(record.createdAt).map(formatTimestamp) ?? "n/a",
            sessionTotal: max(0, record.sessionTotal),
            sessionActiveCount: max(0, record.sessionActiveCount),
            sessionExpiredCount: max(0, record.sessionExpiredCount),
            sessionRevokedCount: max(0, record.sessionRevokedCount),
            sessionLatestStartedAtLabel: nonEmpty(record.sessionLatestStartedAt).map(formatTimestamp),
            sortTimestamp: createdAt
        )
    }

    private static func mapSessionItem(
        _ record: DaemonIdentitySessionRecord,
        fallbackWorkspaceID: String,
        canonicalWorkspaceID: (String?) -> String?,
        parseTimestamp: (String) -> Date?,
        formatTimestamp: (String) -> String
    ) -> IdentitySessionItem {
        let sessionID = nonEmpty(record.sessionID) ?? UUID().uuidString.lowercased()
        let startedAt = parseTimestamp(record.startedAt)
            ?? parseTimestamp(record.expiresAt)
            ?? .distantPast
        let normalizedWorkspaceID = canonicalWorkspaceID(nonEmpty(record.workspaceID)) ?? fallbackWorkspaceID
        return IdentitySessionItem(
            id: sessionID,
            workspaceID: normalizedWorkspaceID,
            deviceID: nonEmpty(record.deviceID) ?? "unknown",
            userID: nonEmpty(record.userID) ?? "unknown",
            deviceType: nonEmpty(record.deviceType) ?? "unknown",
            platform: nonEmpty(record.platform) ?? "unknown",
            deviceLabel: nonEmpty(record.deviceLabel),
            deviceLastSeenAtLabel: nonEmpty(record.deviceLastSeenAt).map(formatTimestamp),
            startedAtLabel: nonEmpty(record.startedAt).map(formatTimestamp) ?? "n/a",
            expiresAtLabel: nonEmpty(record.expiresAt).map(formatTimestamp) ?? "n/a",
            revokedAtLabel: nonEmpty(record.revokedAt).map(formatTimestamp),
            sessionHealth: nonEmpty(record.sessionHealth)?.lowercased() ?? "unknown",
            sortTimestamp: startedAt
        )
    }

    private static func nonEmpty(_ raw: String?) -> String? {
        guard let value = raw?.trimmingCharacters(in: .whitespacesAndNewlines), !value.isEmpty else {
            return nil
        }
        return value
    }
}
