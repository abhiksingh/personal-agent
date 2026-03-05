import Foundation

enum NotificationInboxRouting {
    static func sourceSection(for source: String) -> AppSection? {
        let normalized = source.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        switch normalized {
        case "home":
            return .home
        case "chat":
            return .chat
        case "communications":
            return .communications
        case "automation":
            return .automation
        case "approvals":
            return .approvals
        case "tasks":
            return .tasks
        case "inspect":
            return .inspect
        case "channels":
            return .channels
        case "connectors":
            return .connectors
        case "models":
            return .models
        case "configuration":
            return .configuration
        default:
            if normalized.hasPrefix("ui.configuration") {
                return .configuration
            }
            return nil
        }
    }

    static func inboxIntent(for item: AppNotificationItem) -> NotificationInboxIntent {
        if item.level == .error {
            return .needsAttention
        }
        if let section = sourceSection(for: item.source) {
            switch section {
            case .home, .chat, .communications, .automation, .approvals, .tasks:
                return .workflow
            case .inspect:
                return .diagnostics
            case .configuration, .channels, .connectors, .models:
                return .runtime
            }
        }
        if item.level == .progress {
            return .runtime
        }
        return .general
    }

    static func inboxActions(for item: AppNotificationItem) -> [NotificationInboxAction] {
        guard let section = sourceSection(for: item.source) else {
            if item.level == .error {
                return [
                    NotificationInboxAction(
                        kind: .openSection(.configuration),
                        title: "Open Configuration",
                        symbolName: AppSection.configuration.symbolName
                    )
                ]
            }
            return []
        }
        return [
            NotificationInboxAction(
                kind: .openSection(section),
                title: "Open \(section.title)",
                symbolName: section.symbolName
            )
        ]
    }
}
