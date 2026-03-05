import Foundation

enum UIAccessibilityContract {
    static let sidebarNavigationIdentifier = "sidebar-navigation-list"
    static let commandPaletteSearchIdentifier = "command-palette-search-field"
    static let approvalsSearchIdentifier = "approvals-search-field"
    static let tasksSearchIdentifier = "tasks-search-field"
    static let communicationsSearchIdentifier = "communications-search-field"

    static let sidebarNavigationLabel = "Primary navigation sidebar"
    static let sidebarNavigationHint = "Use arrow keys to choose a section and open its workflow panel."

    static let commandPaletteSearchLabel = "Search commands and objects"
    static let commandPaletteSearchHint = "Type a command or object name, then press Return to run the first match."

    static let approvalsSearchLabel = "Search approvals"
    static let approvalsSearchHint = "Filters approvals by task, run, actor, and route details."

    static let tasksSearchLabel = "Search tasks"
    static let tasksSearchHint = "Filters task runs by status, IDs, principal, and route metadata."

    static let communicationsSearchLabel = "Search communications"
    static let communicationsSearchHint = "Filters conversations, events, destinations, and call sessions."

    static let drillInDismissLabel = "Dismiss drill-in context"
    static let drillInDismissHint = "Removes source context chips and keeps the current panel open."

    static func panelLandmarkLabel(for sectionTitle: String) -> String {
        "\(sectionTitle) workflow panel"
    }
}
