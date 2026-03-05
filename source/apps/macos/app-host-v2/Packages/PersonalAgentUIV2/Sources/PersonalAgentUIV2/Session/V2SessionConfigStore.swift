import Combine
import Foundation

public enum V2InformationDensityMode: String, CaseIterable, Codable, Sendable, Equatable {
    case simple
    case advanced

    public var title: String {
        switch self {
        case .simple:
            return "Simple"
        case .advanced:
            return "Advanced"
        }
    }
}

public enum V2SessionConfigError: LocalizedError {
    case emptyToken

    public var errorDescription: String? {
        switch self {
        case .emptyToken:
            return "Assistant access token cannot be empty."
        }
    }
}

public struct V2SessionReadiness: Sendable, Equatable {
    public let hasValidDaemonBaseURL: Bool
    public let hasStoredAccessToken: Bool
    public let hasWorkspaceID: Bool
    public let hasPrincipalActorID: Bool

    public var isReadyForDaemonMutations: Bool {
        hasValidDaemonBaseURL && hasStoredAccessToken && hasWorkspaceID && hasPrincipalActorID
    }

    public var setupSummary: String {
        if isReadyForDaemonMutations {
            return "Session is configured and ready for assistant actions."
        }
        if !hasValidDaemonBaseURL {
            return "Set a valid daemon URL."
        }
        if !hasStoredAccessToken {
            return "Save an Assistant Access Token."
        }
        if !hasWorkspaceID {
            return "Set a workspace ID."
        }
        if !hasPrincipalActorID {
            return "Set a principal."
        }
        return "Session requires setup."
    }

    public var highImpactActionDisabledReason: String? {
        if isReadyForDaemonMutations {
            return nil
        }
        if !hasValidDaemonBaseURL {
            return "Set a valid daemon URL in Get Started before running assistant actions."
        }
        if !hasStoredAccessToken {
            return "Save an Assistant Access Token in Get Started before running assistant actions."
        }
        if !hasWorkspaceID {
            return "Set a workspace ID in Get Started before running assistant actions."
        }
        if !hasPrincipalActorID {
            return "Set a principal in Get Started before running assistant actions."
        }
        return "Complete setup in Get Started before running assistant actions."
    }
}

@MainActor
public final class V2SessionConfigStore: ObservableObject {
    private static let uiDefaultsSuiteEnvKey = "PA_UI_DEFAULTS_SUITE"
    private static let daemonBaseURLDefaultsKey = "personalagent.ui.v2.daemon_base_url"
    private static let workspaceDefaultsKey = "personalagent.ui.v2.workspace_id"
    private static let principalDefaultsKey = "personalagent.ui.v2.principal_actor_id"
    private static let selectedSectionDefaultsKey = "personalagent.ui.v2.selected_section"
    private static let densityModeDefaultsKey = "personalagent.ui.v2.information_density_mode.v1"
    private static let defaultDaemonBaseURL = "http://127.0.0.1:7071"
    private static let defaultWorkspaceID = "ws1"
    private static let defaultPrincipalActorID = "default"

    @Published public var daemonBaseURL: String {
        didSet {
            guard !isHydrating else { return }
            let normalized = normalize(daemonBaseURL, fallback: Self.defaultDaemonBaseURL)
            if daemonBaseURL != normalized {
                daemonBaseURL = normalized
                return
            }
            persistDaemonBaseURL()
            refreshReadiness()
        }
    }
    @Published public var workspaceID: String {
        didSet {
            guard !isHydrating else { return }
            let normalized = normalize(workspaceID, fallback: Self.defaultWorkspaceID)
            if workspaceID != normalized {
                workspaceID = normalized
                return
            }
            persistWorkspaceID()
            applyWorkspaceScopedDensityMode()
            refreshReadiness()
        }
    }
    @Published public var principalActorID: String {
        didSet {
            guard !isHydrating else { return }
            let normalized = normalize(principalActorID, fallback: Self.defaultPrincipalActorID)
            if principalActorID != normalized {
                principalActorID = normalized
                return
            }
            persistPrincipalActorID()
            refreshReadiness()
        }
    }
    @Published public var informationDensityMode: V2InformationDensityMode {
        didSet {
            guard !isHydrating else { return }
            persistDensityModes()
        }
    }
    @Published public private(set) var selectedSection: AssistantWorkspaceSection
    @Published public private(set) var hasStoredAccessToken: Bool
    @Published public private(set) var readiness: V2SessionReadiness

    public let tokenReference: V2SessionTokenReference

    private let userDefaults: UserDefaults
    private let secretStore: any V2SecretStoring
    private var densityModeByWorkspaceID: [String: V2InformationDensityMode]
    private var isHydrating = false

    public init(
        userDefaults: UserDefaults? = nil,
        secretStore: any V2SecretStoring = V2KeychainSecretStore(),
        tokenReference: V2SessionTokenReference = .defaultReference
    ) {
        self.userDefaults = userDefaults ?? Self.defaultUserDefaultsStore()
        self.secretStore = secretStore
        self.tokenReference = tokenReference
        self.daemonBaseURL = Self.defaultDaemonBaseURL
        self.workspaceID = Self.defaultWorkspaceID
        self.principalActorID = Self.defaultPrincipalActorID
        self.informationDensityMode = .simple
        self.selectedSection = .replayAndAsk
        self.densityModeByWorkspaceID = [:]
        self.hasStoredAccessToken = false
        self.readiness = V2SessionReadiness(
            hasValidDaemonBaseURL: false,
            hasStoredAccessToken: false,
            hasWorkspaceID: false,
            hasPrincipalActorID: false
        )
        hydrateFromPersistence()
    }

    public var tokenReferenceLabel: String {
        "\(tokenReference.service) • \(tokenReference.account)"
    }

    public func resolvedAccessToken() -> String? {
        let loadedToken: String?
        do {
            loadedToken = try secretStore.readSecret(reference: tokenReference)
        } catch {
            return nil
        }

        guard let trimmedToken = loadedToken?.trimmingCharacters(in: .whitespacesAndNewlines),
              !trimmedToken.isEmpty else {
            return nil
        }
        return trimmedToken
    }

    public func saveAccessToken(_ rawToken: String) throws {
        let trimmedToken = rawToken.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmedToken.isEmpty else {
            throw V2SessionConfigError.emptyToken
        }
        try secretStore.upsertSecret(value: trimmedToken, reference: tokenReference)
        hasStoredAccessToken = true
        refreshReadiness()
    }

    public func clearAccessToken() throws {
        try secretStore.deleteSecret(reference: tokenReference)
        hasStoredAccessToken = false
        refreshReadiness()
    }

    public func refreshTokenState() {
        hasStoredAccessToken = resolvedAccessToken() != nil
        refreshReadiness()
    }

    public func persistSelectedSection(_ section: AssistantWorkspaceSection) {
        selectedSection = section
        userDefaults.set(section.rawValue, forKey: Self.selectedSectionDefaultsKey)
    }

    private func hydrateFromPersistence() {
        isHydrating = true
        defer {
            isHydrating = false
            refreshTokenState()
            applyWorkspaceScopedDensityMode()
            refreshReadiness()
        }

        daemonBaseURL = normalize(
            userDefaults.string(forKey: Self.daemonBaseURLDefaultsKey),
            fallback: Self.defaultDaemonBaseURL
        )
        workspaceID = normalize(
            userDefaults.string(forKey: Self.workspaceDefaultsKey),
            fallback: Self.defaultWorkspaceID
        )
        principalActorID = normalize(
            userDefaults.string(forKey: Self.principalDefaultsKey),
            fallback: Self.defaultPrincipalActorID
        )
        densityModeByWorkspaceID = loadDensityModes()

        if let rawSection = userDefaults.string(forKey: Self.selectedSectionDefaultsKey),
           let persistedSection = AssistantWorkspaceSection(rawValue: rawSection) {
            selectedSection = persistedSection
        } else {
            selectedSection = .replayAndAsk
        }
    }

    private func refreshReadiness() {
        readiness = V2SessionReadiness(
            hasValidDaemonBaseURL: Self.isValidDaemonBaseURL(daemonBaseURL),
            hasStoredAccessToken: hasStoredAccessToken,
            hasWorkspaceID: !workspaceID.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty,
            hasPrincipalActorID: !principalActorID.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
        )
    }

    private func persistDaemonBaseURL() {
        userDefaults.set(daemonBaseURL, forKey: Self.daemonBaseURLDefaultsKey)
    }

    private func persistWorkspaceID() {
        userDefaults.set(workspaceID, forKey: Self.workspaceDefaultsKey)
    }

    private func persistPrincipalActorID() {
        userDefaults.set(principalActorID, forKey: Self.principalDefaultsKey)
    }

    private func applyWorkspaceScopedDensityMode() {
        informationDensityMode = densityModeByWorkspaceID[workspaceID] ?? .simple
    }

    private func loadDensityModes() -> [String: V2InformationDensityMode] {
        guard let data = userDefaults.data(forKey: Self.densityModeDefaultsKey),
              let raw = try? JSONDecoder().decode([String: String].self, from: data) else {
            return [:]
        }

        var mapped: [String: V2InformationDensityMode] = [:]
        for (workspace, modeRaw) in raw {
            let normalizedWorkspace = normalize(workspace, fallback: "")
            guard !normalizedWorkspace.isEmpty,
                  let mode = V2InformationDensityMode(rawValue: modeRaw) else {
                continue
            }
            mapped[normalizedWorkspace] = mode
        }
        return mapped
    }

    private func persistDensityModes() {
        densityModeByWorkspaceID[workspaceID] = informationDensityMode
        let raw = Dictionary(
            uniqueKeysWithValues: densityModeByWorkspaceID.map { key, mode in
                (key, mode.rawValue)
            }
        )
        guard let data = try? JSONEncoder().encode(raw) else {
            return
        }
        userDefaults.set(data, forKey: Self.densityModeDefaultsKey)
    }

    private func normalize(_ value: String?, fallback: String) -> String {
        guard let trimmed = value?.trimmingCharacters(in: .whitespacesAndNewlines), !trimmed.isEmpty else {
            return fallback
        }
        return trimmed
    }

    private static func defaultUserDefaultsStore() -> UserDefaults {
        let suiteName = ProcessInfo.processInfo.environment[uiDefaultsSuiteEnvKey]?
            .trimmingCharacters(in: .whitespacesAndNewlines)
        if let suiteName, !suiteName.isEmpty, let suiteDefaults = UserDefaults(suiteName: suiteName) {
            return suiteDefaults
        }
        return .standard
    }

    private static func isValidDaemonBaseURL(_ rawValue: String) -> Bool {
        let trimmed = rawValue.trimmingCharacters(in: .whitespacesAndNewlines)
        guard let url = URL(string: trimmed),
              let components = URLComponents(url: url, resolvingAgainstBaseURL: false),
              let scheme = components.scheme?.lowercased(),
              ["http", "https", "ws", "wss"].contains(scheme),
              components.host != nil else {
            return false
        }
        return true
    }
}
