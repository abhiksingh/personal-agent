import Foundation

enum RuntimePluginLifecycleProjection {
    static let defaultFilterSelection = "all"
    static let defaultHistoryLimit = 40
    static let maxHistoryLimit = 200
    static let missingTokenStatusMessage = "Set Assistant Access Token to query runtime plugin lifecycle history."

    static func normalizedFilter(_ raw: String) -> String? {
        let trimmed = raw.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else {
            return nil
        }
        if trimmed.lowercased() == defaultFilterSelection {
            return nil
        }
        return trimmed
    }

    static func clampedLimit(_ raw: Int) -> Int {
        switch raw {
        case ..<1:
            return defaultHistoryLimit
        case 1...maxHistoryLimit:
            return raw
        default:
            return maxHistoryLimit
        }
    }

    static func summaryMessage(
        itemCount: Int,
        hasMore: Bool,
        pluginID: String?,
        kind: String?,
        state: String?,
        eventType: String?
    ) -> String {
        var parts: [String] = ["Runtime lifecycle history • \(itemCount) event(s)"]
        if let pluginID {
            parts.append("plugin=\(pluginID)")
        }
        if let kind {
            parts.append("kind=\(kind)")
        }
        if let state {
            parts.append("state=\(state)")
        }
        if let eventType {
            parts.append("event=\(eventType)")
        }
        if hasMore {
            parts.append("more available")
        }
        return parts.joined(separator: " • ")
    }

    static func mapAndSortRecords(
        _ records: [DaemonPluginLifecycleHistoryRecord],
        parseTimestamp: (String) -> Date?,
        formatTimestamp: (String) -> String
    ) -> [RuntimePluginLifecycleEventItem] {
        records.map { mapRecord($0, parseTimestamp: parseTimestamp, formatTimestamp: formatTimestamp) }
            .sorted { lhs, rhs in
                if lhs.sortTimestamp == rhs.sortTimestamp {
                    return lhs.id > rhs.id
                }
                return lhs.sortTimestamp > rhs.sortTimestamp
            }
    }

    static func buildTrendItems(
        from items: [RuntimePluginLifecycleEventItem]
    ) -> [RuntimePluginLifecycleTrendItem] {
        guard !items.isEmpty else {
            return []
        }

        let grouped = Dictionary(grouping: items) { nonEmpty($0.pluginID) ?? "unknown" }
        return grouped.values.compactMap { pluginItems in
            let sorted = pluginItems.sorted { lhs, rhs in
                if lhs.sortTimestamp == rhs.sortTimestamp {
                    return lhs.id > rhs.id
                }
                return lhs.sortTimestamp > rhs.sortTimestamp
            }
            guard let latest = sorted.first else {
                return nil
            }
            return RuntimePluginLifecycleTrendItem(
                id: latest.pluginID,
                pluginID: latest.pluginID,
                kind: latest.kind,
                latestState: latest.state,
                latestEventType: latest.eventType,
                latestOccurredAtLabel: latest.occurredAtLabel,
                restartEvents: sorted.filter { $0.restartEvent }.count,
                failureEvents: sorted.filter { $0.failureEvent }.count,
                recoveryEvents: sorted.filter { $0.recoveryEvent }.count,
                totalEvents: sorted.count
            )
        }
        .sorted { lhs, rhs in
            if lhs.totalEvents == rhs.totalEvents {
                return lhs.pluginID < rhs.pluginID
            }
            return lhs.totalEvents > rhs.totalEvents
        }
    }

    private static func mapRecord(
        _ record: DaemonPluginLifecycleHistoryRecord,
        parseTimestamp: (String) -> Date?,
        formatTimestamp: (String) -> String
    ) -> RuntimePluginLifecycleEventItem {
        let occurredAtDate = parseTimestamp(record.occurredAt) ?? .distantPast
        return RuntimePluginLifecycleEventItem(
            id: nonEmpty(record.auditID) ?? UUID().uuidString,
            workspaceID: nonEmpty(record.workspaceID) ?? "daemon",
            pluginID: nonEmpty(record.pluginID) ?? "unknown",
            kind: nonEmpty(record.kind)?.lowercased() ?? "unknown",
            state: nonEmpty(record.state)?.lowercased() ?? "unknown",
            eventType: nonEmpty(record.eventType) ?? "unknown",
            processID: record.processID,
            restartCount: max(0, record.restartCount),
            reason: nonEmpty(record.reason) ?? "unknown",
            error: nonEmpty(record.error),
            restartEvent: record.restartEvent,
            failureEvent: record.failureEvent,
            recoveryEvent: record.recoveryEvent,
            lastHeartbeatAtLabel: nonEmpty(record.lastHeartbeatAt).map(formatTimestamp),
            lastTransitionAtLabel: nonEmpty(record.lastTransitionAt).map(formatTimestamp),
            occurredAtLabel: formatTimestamp(record.occurredAt),
            sortTimestamp: occurredAtDate
        )
    }

    private static func nonEmpty(_ raw: String?) -> String? {
        guard let value = raw?.trimmingCharacters(in: .whitespacesAndNewlines), !value.isEmpty else {
            return nil
        }
        return value
    }
}
