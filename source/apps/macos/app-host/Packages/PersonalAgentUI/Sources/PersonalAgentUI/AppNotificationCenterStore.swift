import Foundation
import SwiftUI

@MainActor
final class AppNotificationCenterStore: ObservableObject {
    @Published private(set) var notificationItems: [AppNotificationItem] = []
    @Published private(set) var notificationToastItems: [AppNotificationItem] = []
    @Published private(set) var successNotificationPulseBySource: [String: Int] = [:]

    private var notificationToastDismissTasksByID: [String: Task<Void, Never>] = [:]
    private var isRestoringNotifications = false

    private let userDefaults: UserDefaults
    private let defaultsKey: String
    private let defaultWorkspaceID: String
    private let notificationHistoryLimit: Int
    private let toastLimit: Int

    init(
        userDefaults: UserDefaults,
        defaultsKey: String,
        defaultWorkspaceID: String,
        notificationHistoryLimit: Int = 250,
        toastLimit: Int = 4
    ) {
        self.userDefaults = userDefaults
        self.defaultsKey = defaultsKey
        self.defaultWorkspaceID = defaultWorkspaceID
        self.notificationHistoryLimit = max(1, notificationHistoryLimit)
        self.toastLimit = max(1, toastLimit)
    }

    func notificationSourceOptions() -> [String] {
        var sources = Set(notificationItems.map(\.source))
        sources.insert("all")
        return sources.sorted()
    }

    func unreadNotificationCount() -> Int {
        notificationItems.filter { !$0.isRead }.count
    }

    func filteredNotificationItems(
        query: String,
        sourceFilter: String
    ) -> [AppNotificationItem] {
        let normalizedQuery = query.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        let normalizedSourceFilter = sourceFilter.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        return notificationItems.filter { item in
            if normalizedSourceFilter != "all",
               !normalizedSourceFilter.isEmpty,
               item.source.lowercased() != normalizedSourceFilter {
                return false
            }
            guard !normalizedQuery.isEmpty else {
                return true
            }
            let haystack = [
                item.source,
                item.action,
                item.workspaceID,
                item.message
            ]
            .joined(separator: " ")
            .lowercased()
            return haystack.contains(normalizedQuery)
        }
    }

    func successNotificationPulse(for source: String) -> Int {
        let normalizedSource = source.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        guard !normalizedSource.isEmpty else {
            return 0
        }
        return successNotificationPulseBySource[normalizedSource] ?? 0
    }

    func postNotification(
        workspaceID: String,
        source: String,
        action: String,
        message: String,
        level: AppNotificationLevel
    ) {
        appendNotification(
            workspaceID: workspaceID,
            source: source,
            action: action,
            message: message,
            level: level
        )
    }

    func recordStatusNotification(
        workspaceID: String,
        source: String,
        oldValue: String?,
        newValue: String?
    ) {
        guard !isRestoringNotifications else {
            return
        }
        guard let message = nonEmpty(newValue) else {
            return
        }
        if let oldValue = nonEmpty(oldValue),
           oldValue.caseInsensitiveCompare(message) == .orderedSame {
            return
        }
        guard shouldCaptureStatusNotification(message: message) else {
            return
        }
        appendNotification(
            workspaceID: workspaceID,
            source: source,
            action: "status_update",
            message: message,
            level: notificationLevel(for: message)
        )
    }

    func dismissNotificationToast(notificationID: String, markAsRead: Bool = true) {
        let normalizedID = notificationID.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !normalizedID.isEmpty else {
            return
        }
        notificationToastDismissTasksByID[normalizedID]?.cancel()
        notificationToastDismissTasksByID.removeValue(forKey: normalizedID)
        notificationToastItems.removeAll { $0.id == normalizedID }
        if markAsRead {
            markNotificationRead(notificationID: normalizedID)
        }
    }

    func markNotificationRead(notificationID: String) {
        let normalizedID = notificationID.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !normalizedID.isEmpty else {
            return
        }
        guard let index = notificationItems.firstIndex(where: { $0.id == normalizedID }) else {
            return
        }
        guard notificationItems[index].readAt == nil else {
            return
        }
        notificationItems[index].readAt = Date()
        persistNotifications()
    }

    func markAllNotificationsRead() {
        let now = Date()
        var didMutate = false
        for index in notificationItems.indices where notificationItems[index].readAt == nil {
            notificationItems[index].readAt = now
            didMutate = true
        }
        if didMutate {
            persistNotifications()
        }
    }

    func clearNotification(notificationID: String) {
        let normalizedID = notificationID.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !normalizedID.isEmpty else {
            return
        }
        dismissNotificationToast(notificationID: normalizedID, markAsRead: false)
        notificationItems.removeAll { $0.id == normalizedID }
        persistNotifications()
    }

    func clearReadNotifications() {
        let readIDs = Set(notificationItems.filter(\.isRead).map(\.id))
        for id in readIDs {
            dismissNotificationToast(notificationID: id, markAsRead: false)
        }
        notificationItems.removeAll(where: \.isRead)
        persistNotifications()
    }

    func clearAllNotifications() {
        for id in notificationItems.map(\.id) {
            dismissNotificationToast(notificationID: id, markAsRead: false)
        }
        notificationItems = []
        persistNotifications()
    }

    func loadPersistedNotifications() {
        guard let data = userDefaults.data(forKey: defaultsKey) else {
            return
        }
        do {
            let decoder = JSONDecoder()
            decoder.dateDecodingStrategy = .iso8601
            let decoded = try decoder.decode([AppNotificationItem].self, from: data)
            isRestoringNotifications = true
            notificationItems = Array(
                normalizedNotificationOrdering(decoded).prefix(notificationHistoryLimit)
            )
            notificationToastItems = []
            notificationToastDismissTasksByID = [:]
            isRestoringNotifications = false
        } catch {
            isRestoringNotifications = false
            notificationItems = []
            notificationToastItems = []
            notificationToastDismissTasksByID = [:]
        }
    }

    private func shouldCaptureStatusNotification(message: String) -> Bool {
        let normalized = message.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        guard !normalized.isEmpty else {
            return false
        }
        if normalized.hasPrefix("waiting for ") {
            return false
        }
        if normalized.hasPrefix("provider: ") {
            return false
        }
        if normalized.contains("has not been queried yet") {
            return false
        }
        if normalized == "no create/edit action run yet." ||
            normalized == "no route policy changes submitted." ||
            normalized == "no route simulation run yet." ||
            normalized == "no route explainability run yet." {
            return false
        }
        return true
    }

    private func notificationLevel(for message: String) -> AppNotificationLevel {
        let normalized = message.lowercased()
        let errorMarkers = [
            "failed",
            "error",
            "unreachable",
            "denied",
            "invalid",
            "unable",
            "needs repair",
            "missing"
        ]
        if errorMarkers.contains(where: { normalized.contains($0) }) {
            return .error
        }
        let progressMarkers = [
            "in progress",
            "refreshing",
            "loading",
            "checking",
            "requested",
            "running"
        ]
        if progressMarkers.contains(where: { normalized.contains($0) }) {
            return .progress
        }
        let successMarkers = [
            "saved",
            "updated",
            "completed",
            "loaded",
            "opened",
            "ready",
            "resolved",
            "connected",
            "granted",
            "configured",
            "revoked"
        ]
        if successMarkers.contains(where: { normalized.contains($0) }) {
            return .success
        }
        return .info
    }

    private func appendNotification(
        workspaceID: String,
        source: String,
        action: String,
        message: String,
        level: AppNotificationLevel
    ) {
        let normalizedSource = nonEmpty(source)?.lowercased() ?? "system"
        let normalizedAction = nonEmpty(action)?.lowercased() ?? "status_update"
        let normalizedMessage = message.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !normalizedMessage.isEmpty else {
            return
        }

        if let first = notificationItems.first,
           first.source == normalizedSource,
           first.action == normalizedAction,
           first.message.caseInsensitiveCompare(normalizedMessage) == .orderedSame,
           first.level == level,
           abs(first.createdAt.timeIntervalSinceNow) < 1 {
            return
        }

        let item = AppNotificationItem(
            workspaceID: nonEmpty(workspaceID) ?? defaultWorkspaceID,
            source: normalizedSource,
            action: normalizedAction,
            level: level,
            message: normalizedMessage
        )
        if level == .success {
            successNotificationPulseBySource[normalizedSource, default: 0] += 1
        }
        notificationItems.insert(item, at: 0)
        notificationItems = normalizedNotificationOrdering(notificationItems)
        if notificationItems.count > notificationHistoryLimit {
            let overflowIDs = Set(notificationItems.dropFirst(notificationHistoryLimit).map(\.id))
            for id in overflowIDs {
                notificationToastDismissTasksByID[id]?.cancel()
                notificationToastDismissTasksByID.removeValue(forKey: id)
            }
            notificationItems = Array(notificationItems.prefix(notificationHistoryLimit))
        }
        persistNotifications()
        presentToast(item)
    }

    private func presentToast(_ notification: AppNotificationItem) {
        notificationToastItems.removeAll { $0.id == notification.id }
        notificationToastItems.insert(notification, at: 0)
        notificationToastItems = normalizedNotificationOrdering(notificationToastItems)
        if notificationToastItems.count > toastLimit {
            let overflowIDs = Set(notificationToastItems.dropFirst(toastLimit).map(\.id))
            notificationToastItems = Array(notificationToastItems.prefix(toastLimit))
            for id in overflowIDs {
                notificationToastDismissTasksByID[id]?.cancel()
                notificationToastDismissTasksByID.removeValue(forKey: id)
            }
        }
        scheduleToastDismiss(notificationID: notification.id)
    }

    private func scheduleToastDismiss(notificationID: String) {
        notificationToastDismissTasksByID[notificationID]?.cancel()
        notificationToastDismissTasksByID[notificationID] = Task { [weak self] in
            try? await Task.sleep(for: .seconds(5))
            await MainActor.run {
                self?.dismissNotificationToast(notificationID: notificationID)
            }
        }
    }

    private func normalizedNotificationOrdering(
        _ notifications: [AppNotificationItem]
    ) -> [AppNotificationItem] {
        notifications.sorted { lhs, rhs in
            if lhs.createdAt == rhs.createdAt {
                return lhs.id > rhs.id
            }
            return lhs.createdAt > rhs.createdAt
        }
    }

    private func persistNotifications() {
        guard !isRestoringNotifications else {
            return
        }
        do {
            let encoder = JSONEncoder()
            encoder.dateEncodingStrategy = .iso8601
            let data = try encoder.encode(notificationItems)
            userDefaults.set(data, forKey: defaultsKey)
        } catch {
            // Keep UI responsive; persistence failures are non-fatal.
        }
    }

    private func nonEmpty(_ value: String?) -> String? {
        guard let trimmed = value?.trimmingCharacters(in: .whitespacesAndNewlines), !trimmed.isEmpty else {
            return nil
        }
        return trimmed
    }
}
