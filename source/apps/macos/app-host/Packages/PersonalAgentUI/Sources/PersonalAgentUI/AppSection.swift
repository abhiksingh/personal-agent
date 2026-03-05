import Foundation

public enum AppSection: String, CaseIterable, Identifiable, Sendable {
    case configuration
    case home
    case chat
    case communications
    case automation
    case approvals
    case tasks
    case inspect
    case channels
    case connectors
    case models

    public var id: String { rawValue }

    public var title: String {
        switch self {
        case .configuration:
            return "Configuration"
        case .home:
            return "Home"
        case .chat:
            return "Chat"
        case .communications:
            return "Communications"
        case .automation:
            return "Automation"
        case .approvals:
            return "Approvals"
        case .tasks:
            return "Tasks"
        case .inspect:
            return "Inspect"
        case .channels:
            return "Channels"
        case .connectors:
            return "Connectors"
        case .models:
            return "Models"
        }
    }

    public var symbolName: String {
        switch self {
        case .configuration:
            return "gearshape"
        case .home:
            return "house"
        case .chat:
            return "bubble.left.and.bubble.right"
        case .communications:
            return "tray.full"
        case .automation:
            return "clock.arrow.trianglehead.counterclockwise.rotate.90"
        case .approvals:
            return "checkmark.shield"
        case .tasks:
            return "checklist"
        case .inspect:
            return "doc.text.magnifyingglass"
        case .channels:
            return "point.3.connected.trianglepath.dotted"
        case .connectors:
            return "cable.connector"
        case .models:
            return "cube"
        }
    }

    public var isAdvancedSidebarDestination: Bool {
        Self.advancedSidebarSections.contains(self)
    }

    public static let primarySidebarSections: [AppSection] = [
        .home,
        .chat,
        .communications,
        .tasks,
        .approvals,
        .automation
    ]

    public static let advancedSidebarSections: [AppSection] = [
        .inspect,
        .channels,
        .connectors,
        .models
    ]

    public static let middleSidebarSections: [AppSection] =
        primarySidebarSections + advancedSidebarSections
}
