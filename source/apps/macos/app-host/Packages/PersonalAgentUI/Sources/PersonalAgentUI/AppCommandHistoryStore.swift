import Foundation

@MainActor
final class AppCommandHistoryStore: ObservableObject {
    @Published private(set) var recentUsage: [AppCommandActionID] = []

    private let userDefaults: UserDefaults
    private let defaultsKey: String
    private let historyLimit: Int

    init(
        userDefaults: UserDefaults,
        defaultsKey: String,
        historyLimit: Int
    ) {
        self.userDefaults = userDefaults
        self.defaultsKey = defaultsKey
        self.historyLimit = historyLimit
    }

    func contains(_ actionID: AppCommandActionID) -> Bool {
        recentUsage.contains(actionID)
    }

    func recordUsage(_ actionID: AppCommandActionID) {
        recentUsage.removeAll { $0 == actionID }
        recentUsage.insert(actionID, at: 0)
        if recentUsage.count > historyLimit {
            recentUsage = Array(recentUsage.prefix(historyLimit))
        }
        persist()
    }

    func loadPersistedUsage() {
        guard let rawValues = userDefaults.array(forKey: defaultsKey) as? [String] else {
            return
        }
        recentUsage = rawValues.compactMap(AppCommandActionID.init(rawValue:))
    }

    private func persist() {
        userDefaults.set(recentUsage.map(\.rawValue), forKey: defaultsKey)
    }
}
