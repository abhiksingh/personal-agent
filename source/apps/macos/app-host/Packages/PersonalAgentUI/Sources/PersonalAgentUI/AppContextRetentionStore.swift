import Foundation
import SwiftUI

@MainActor
final class AppContextRetentionStore: ObservableObject {
    struct HomeFirstSessionProgress: Codable, Sendable, Equatable {
        struct MilestoneEvidence: Codable, Sendable, Equatable {
            var completedAtRaw = ""
            var source = ""
        }

        var sentMessage = false
        var sentCommunication = false
        var createdTask = false
        var reviewedApprovals = false
        var milestoneEvidenceByStepID: [String: MilestoneEvidence] = [:]
    }

    @Published private(set) var panelFilterContextByWorkspaceID: [String: WorkspacePanelFilterContext] = [:]
    @Published private(set) var communicationsTriageContextByWorkspaceID: [String: CommunicationsTriageContext] = [:]
    @Published private(set) var workspaceContinuityContextByWorkspaceID: [String: WorkspaceContinuityContext] = [:]
    @Published private(set) var informationDensityModeByWorkspaceID: [String: AppInformationDensityMode] = [:]
    @Published private(set) var homeFirstSessionProgressByWorkspaceID: [String: HomeFirstSessionProgress] = [:]

    private let userDefaults: UserDefaults
    private let defaultWorkspaceID: String
    private let canonicalWorkspaceID: @MainActor (String?, Bool) -> String?
    private let panelFilterContextDefaultsKey: String
    private let communicationsTriageDefaultsKey: String
    private let workspaceContinuityDefaultsKey: String
    private let informationDensityModeDefaultsKey: String
    private let homeFirstSessionProgressDefaultsKey: String

    init(
        userDefaults: UserDefaults,
        defaultWorkspaceID: String,
        canonicalWorkspaceID: @escaping @MainActor (String?, Bool) -> String?,
        panelFilterContextDefaultsKey: String,
        communicationsTriageDefaultsKey: String,
        workspaceContinuityDefaultsKey: String,
        informationDensityModeDefaultsKey: String,
        homeFirstSessionProgressDefaultsKey: String
    ) {
        self.userDefaults = userDefaults
        self.defaultWorkspaceID = defaultWorkspaceID
        self.canonicalWorkspaceID = canonicalWorkspaceID
        self.panelFilterContextDefaultsKey = panelFilterContextDefaultsKey
        self.communicationsTriageDefaultsKey = communicationsTriageDefaultsKey
        self.workspaceContinuityDefaultsKey = workspaceContinuityDefaultsKey
        self.informationDensityModeDefaultsKey = informationDensityModeDefaultsKey
        self.homeFirstSessionProgressDefaultsKey = homeFirstSessionProgressDefaultsKey
    }

    func currentWorkspaceID(for rawWorkspaceID: String?) -> String {
        canonicalWorkspaceID(rawWorkspaceID, true) ?? defaultWorkspaceID
    }

    func informationDensityMode(for rawWorkspaceID: String?) -> AppInformationDensityMode {
        let workspaceID = currentWorkspaceID(for: rawWorkspaceID)
        return informationDensityModeByWorkspaceID[workspaceID] ?? .simple
    }

    func setInformationDensityMode(_ mode: AppInformationDensityMode, for rawWorkspaceID: String?) {
        let workspaceID = currentWorkspaceID(for: rawWorkspaceID)
        informationDensityModeByWorkspaceID[workspaceID] = mode
        persistInformationDensityModes()
    }

    func panelFilterContext(for rawWorkspaceID: String?) -> WorkspacePanelFilterContext {
        panelFilterContextByWorkspaceID[currentWorkspaceID(for: rawWorkspaceID)] ?? WorkspacePanelFilterContext()
    }

    func updatePanelFilterContext(
        for rawWorkspaceID: String?,
        _ mutate: (inout WorkspacePanelFilterContext) -> Void
    ) {
        let workspaceID = currentWorkspaceID(for: rawWorkspaceID)
        var context = panelFilterContextByWorkspaceID[workspaceID] ?? WorkspacePanelFilterContext()
        mutate(&context)
        panelFilterContextByWorkspaceID[workspaceID] = context
        persistPanelFilterContexts()
    }

    func communicationsTriageContext(for rawWorkspaceID: String?) -> CommunicationsTriageContext {
        communicationsTriageContextByWorkspaceID[currentWorkspaceID(for: rawWorkspaceID)]
            ?? CommunicationsTriageContext()
    }

    func setCommunicationsTriageContext(
        _ context: CommunicationsTriageContext,
        for rawWorkspaceID: String?
    ) {
        communicationsTriageContextByWorkspaceID[currentWorkspaceID(for: rawWorkspaceID)] = context
        persistCommunicationsTriageContexts()
    }

    func workspaceContinuityContext(for rawWorkspaceID: String?) -> WorkspaceContinuityContext {
        workspaceContinuityContextByWorkspaceID[currentWorkspaceID(for: rawWorkspaceID)]
            ?? WorkspaceContinuityContext()
    }

    func updateWorkspaceContinuityContext(
        for rawWorkspaceID: String?,
        _ mutate: (inout WorkspaceContinuityContext) -> Void
    ) {
        let workspaceID = currentWorkspaceID(for: rawWorkspaceID)
        var context = workspaceContinuityContextByWorkspaceID[workspaceID] ?? WorkspaceContinuityContext()
        mutate(&context)
        workspaceContinuityContextByWorkspaceID[workspaceID] = context
        persistWorkspaceContinuityContexts()
    }

    func resetWorkspaceContinuityContext(for rawWorkspaceID: String?) {
        workspaceContinuityContextByWorkspaceID[currentWorkspaceID(for: rawWorkspaceID)] =
            WorkspaceContinuityContext()
        persistWorkspaceContinuityContexts()
    }

    func homeFirstSessionProgress(for rawWorkspaceID: String?) -> HomeFirstSessionProgress {
        homeFirstSessionProgressByWorkspaceID[currentWorkspaceID(for: rawWorkspaceID)] ?? HomeFirstSessionProgress()
    }

    func updateHomeFirstSessionProgress(
        for rawWorkspaceID: String?,
        _ mutate: (inout HomeFirstSessionProgress) -> Void
    ) {
        let workspaceID = currentWorkspaceID(for: rawWorkspaceID)
        var progress = homeFirstSessionProgressByWorkspaceID[workspaceID] ?? HomeFirstSessionProgress()
        mutate(&progress)
        homeFirstSessionProgressByWorkspaceID[workspaceID] = progress
        persistHomeFirstSessionProgress()
    }

    func resetHomeFirstSessionProgress() {
        homeFirstSessionProgressByWorkspaceID = [:]
        userDefaults.removeObject(forKey: homeFirstSessionProgressDefaultsKey)
    }

    func persistPanelFilterContexts() {
        do {
            let data = try JSONEncoder().encode(panelFilterContextByWorkspaceID)
            userDefaults.set(data, forKey: panelFilterContextDefaultsKey)
        } catch {
            // Keep UI responsive; persistence failures are non-fatal.
        }
    }

    func loadPersistedPanelFilterContexts() {
        guard let data = userDefaults.data(forKey: panelFilterContextDefaultsKey) else {
            return
        }
        do {
            let decoded = try JSONDecoder().decode([String: WorkspacePanelFilterContext].self, from: data)
            panelFilterContextByWorkspaceID = normalizeWorkspaceScopedDictionary(decoded)
        } catch {
            panelFilterContextByWorkspaceID = [:]
        }
    }

    func persistCommunicationsTriageContexts() {
        do {
            let data = try JSONEncoder().encode(communicationsTriageContextByWorkspaceID)
            userDefaults.set(data, forKey: communicationsTriageDefaultsKey)
        } catch {
            // Keep UI responsive; persistence failures are non-fatal.
        }
    }

    func loadPersistedCommunicationsTriageContexts() {
        guard let data = userDefaults.data(forKey: communicationsTriageDefaultsKey) else {
            communicationsTriageContextByWorkspaceID = [:]
            return
        }
        do {
            let decoded = try JSONDecoder().decode([String: CommunicationsTriageContext].self, from: data)
            communicationsTriageContextByWorkspaceID = normalizeWorkspaceScopedDictionary(decoded)
        } catch {
            communicationsTriageContextByWorkspaceID = [:]
        }
    }

    func persistWorkspaceContinuityContexts() {
        do {
            let data = try JSONEncoder().encode(workspaceContinuityContextByWorkspaceID)
            userDefaults.set(data, forKey: workspaceContinuityDefaultsKey)
        } catch {
            // Keep UI responsive; persistence failures are non-fatal.
        }
    }

    func loadPersistedWorkspaceContinuityContexts() {
        guard let data = userDefaults.data(forKey: workspaceContinuityDefaultsKey) else {
            workspaceContinuityContextByWorkspaceID = [:]
            return
        }
        do {
            let decoded = try JSONDecoder().decode([String: WorkspaceContinuityContext].self, from: data)
            workspaceContinuityContextByWorkspaceID = normalizeWorkspaceScopedDictionary(decoded)
        } catch {
            workspaceContinuityContextByWorkspaceID = [:]
        }
    }

    func persistHomeFirstSessionProgress() {
        do {
            let data = try JSONEncoder().encode(homeFirstSessionProgressByWorkspaceID)
            userDefaults.set(data, forKey: homeFirstSessionProgressDefaultsKey)
        } catch {
            // Keep UI responsive; persistence failures are non-fatal.
        }
    }

    func loadPersistedHomeFirstSessionProgress() {
        guard let data = userDefaults.data(forKey: homeFirstSessionProgressDefaultsKey) else {
            homeFirstSessionProgressByWorkspaceID = [:]
            return
        }
        do {
            let decoded = try JSONDecoder().decode([String: HomeFirstSessionProgress].self, from: data)
            homeFirstSessionProgressByWorkspaceID = normalizeWorkspaceScopedDictionary(decoded)
        } catch {
            homeFirstSessionProgressByWorkspaceID = [:]
        }
    }

    func persistInformationDensityModes() {
        let rawValues = Dictionary(
            uniqueKeysWithValues: informationDensityModeByWorkspaceID.map { key, mode in
                (key, mode.rawValue)
            }
        )
        do {
            let data = try JSONEncoder().encode(rawValues)
            userDefaults.set(data, forKey: informationDensityModeDefaultsKey)
        } catch {
            // Keep UI responsive; persistence failures are non-fatal.
        }
    }

    func loadPersistedInformationDensityModes() {
        guard let data = userDefaults.data(forKey: informationDensityModeDefaultsKey) else {
            informationDensityModeByWorkspaceID = [:]
            return
        }
        do {
            let decoded = try JSONDecoder().decode([String: String].self, from: data)
            var decodedModes: [String: AppInformationDensityMode] = [:]
            for (key, rawValue) in decoded {
                guard let mode = AppInformationDensityMode(rawValue: rawValue) else {
                    continue
                }
                decodedModes[key] = mode
            }
            informationDensityModeByWorkspaceID = normalizeWorkspaceScopedDictionary(decodedModes)
        } catch {
            informationDensityModeByWorkspaceID = [:]
        }
    }

    private func normalizeWorkspaceScopedDictionary<Value>(
        _ valuesByWorkspaceID: [String: Value]
    ) -> [String: Value] {
        guard !valuesByWorkspaceID.isEmpty else {
            return [:]
        }
        var normalized: [String: Value] = [:]
        for (rawKey, value) in valuesByWorkspaceID {
            guard let normalizedKey = canonicalWorkspaceID(rawKey, false) else {
                continue
            }
            normalized[normalizedKey] = value
        }
        return normalized
    }
}
