import Foundation

@MainActor
enum ConnectorPermissionManager {
    static func systemSettingsURL(for connectorID: String) -> URL {
        let urlString: String
        switch connectorID.lowercased() {
        case "imessage":
            urlString = "x-apple.systempreferences:com.apple.preference.security?Privacy_AllFiles"
        case "mail", "calendar", "browser", "finder":
            urlString = "x-apple.systempreferences:com.apple.preference.security?Privacy_Automation"
        default:
            urlString = "x-apple.systempreferences:com.apple.preference.security?Privacy"
        }
        return URL(string: urlString)
            ?? URL(string: "x-apple.systempreferences:com.apple.preference.security?Privacy")!
    }

    static func systemSettingsURL(fromDaemonDestination destination: String?) -> URL? {
        guard let destination = destination?.trimmingCharacters(in: .whitespacesAndNewlines),
              !destination.isEmpty,
              let components = URLComponents(string: destination),
              components.scheme?.lowercased() == "ui",
              components.host?.lowercased() == "system-settings" else {
            return nil
        }

        let pathComponents = components.path
            .split(separator: "/")
            .map { $0.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() }
        guard !pathComponents.isEmpty else {
            return URL(string: "x-apple.systempreferences:com.apple.preference.security?Privacy")
        }

        guard pathComponents[0] == "privacy" else {
            return URL(string: "x-apple.systempreferences:com.apple.preference.security?Privacy")
        }

        let pane: String
        if pathComponents.count < 2 {
            pane = "Privacy"
        } else {
            switch pathComponents[1] {
            case "automation":
                pane = "Privacy_Automation"
            case "full-disk-access", "all-files":
                pane = "Privacy_AllFiles"
            case "accessibility":
                pane = "Privacy_Accessibility"
            case "calendars":
                pane = "Privacy_Calendars"
            default:
                pane = "Privacy"
            }
        }

        return URL(string: "x-apple.systempreferences:com.apple.preference.security?\(pane)")
    }
}
