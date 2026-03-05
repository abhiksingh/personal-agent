import Foundation

public enum AssistantWorkspaceSection: String, CaseIterable, Identifiable {
    case replayAndAsk
    case getStarted
    case connectorsAndModels

    public var id: String { rawValue }

    var title: String {
        switch self {
        case .replayAndAsk:
            return "Replay & Ask"
        case .getStarted:
            return "Get Started"
        case .connectorsAndModels:
            return "Connectors & Models"
        }
    }

    var subtitle: String {
        switch self {
        case .replayAndAsk:
            return "See exactly what came in, what the assistant did, and why"
        case .getStarted:
            return "Get to your first trusted outcome in minutes"
        case .connectorsAndModels:
            return "Control channels, connectors, and model routing"
        }
    }

    var systemImage: String {
        switch self {
        case .replayAndAsk:
            return "timeline.selection"
        case .getStarted:
            return "flag.checkered"
        case .connectorsAndModels:
            return "slider.horizontal.3"
        }
    }
}

public enum ConnectorStatus: String {
    case connected
    case notConnected
    case needsAttention

    var label: String {
        switch self {
        case .connected:
            return "Connected"
        case .notConnected:
            return "Not Connected"
        case .needsAttention:
            return "Needs Attention"
        }
    }
}

public enum V2ConnectorRowAction: String, Sendable, Equatable {
    case toggle
    case check
    case saveConfig
    case requestPermission
    case remediation
}

public enum V2ModelRowAction: String, Sendable, Equatable {
    case toggle
    case setPrimary
    case simulateRoute
    case explainRoute
}

public struct ConnectorState: Identifiable, Equatable {
    public typealias ID = String

    public let id: ID
    public var pluginID: String
    public var name: String
    public var status: ConnectorStatus
    public var summary: String
    public var enabled: Bool
    public var configured: Bool
    public var actionReadiness: String
    public var actionBlockers: [V2DaemonActionReadinessBlocker]
    public var remediationActions: [V2DaemonDiagnosticsRemediationAction]
    public var permissionState: String?
    public var configurationBaseline: [String: String]
    public var configurationDraft: [String: String]
    public var lastCheckAt: Date?
    public var lastCheckSummary: String?
    public var lastCheckSucceeded: Bool?

    public init(
        id: ID,
        pluginID: String? = nil,
        name: String,
        status: ConnectorStatus,
        summary: String = "",
        enabled: Bool = false,
        configured: Bool = false,
        actionReadiness: String = "unknown",
        actionBlockers: [V2DaemonActionReadinessBlocker] = [],
        remediationActions: [V2DaemonDiagnosticsRemediationAction] = [],
        permissionState: String? = nil,
        configurationBaseline: [String: String] = [:],
        configurationDraft: [String: String]? = nil,
        lastCheckAt: Date? = nil,
        lastCheckSummary: String? = nil,
        lastCheckSucceeded: Bool? = nil
    ) {
        self.id = id
        self.pluginID = pluginID ?? id
        self.name = name
        self.status = status
        self.summary = summary
        self.enabled = enabled
        self.configured = configured
        self.actionReadiness = actionReadiness
        self.actionBlockers = actionBlockers
        self.remediationActions = remediationActions
        self.permissionState = permissionState
        self.configurationBaseline = configurationBaseline
        self.configurationDraft = configurationDraft ?? configurationBaseline
        self.lastCheckAt = lastCheckAt
        self.lastCheckSummary = lastCheckSummary
        self.lastCheckSucceeded = lastCheckSucceeded
    }

    public var hasConfigDraftChanges: Bool {
        configurationDraft != configurationBaseline
    }

    public var connectActionLabel: String {
        enabled ? "Disconnect" : "Connect"
    }
}

public struct ModelOption: Identifiable, Hashable {
    public typealias ID = String

    public let id: ID
    public var providerID: String
    public var providerName: String
    public var modelKey: String
    public var enabled: Bool
    public var providerReady: Bool
    public var providerEndpoint: String?

    public init(
        id: ID? = nil,
        providerID: String,
        providerName: String,
        modelKey: String,
        enabled: Bool,
        providerReady: Bool = true,
        providerEndpoint: String? = nil
    ) {
        let normalizedProviderID = providerID.trimmingCharacters(in: .whitespacesAndNewlines)
        let normalizedModelKey = modelKey.trimmingCharacters(in: .whitespacesAndNewlines)
        self.id = id ?? "\(normalizedProviderID.lowercased())::\(normalizedModelKey.lowercased())"
        self.providerID = normalizedProviderID
        self.providerName = providerName
        self.modelKey = normalizedModelKey
        self.enabled = enabled
        self.providerReady = providerReady
        self.providerEndpoint = providerEndpoint
    }

    public init(id: UUID = UUID(), providerName: String, modelName: String, enabled: Bool) {
        let normalizedProviderID = providerName
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .lowercased()
            .replacingOccurrences(of: " ", with: "_")
        self.init(
            id: id.uuidString.lowercased(),
            providerID: normalizedProviderID,
            providerName: providerName,
            modelKey: modelName,
            enabled: enabled,
            providerReady: true,
            providerEndpoint: nil
        )
    }

    var routeLabel: String {
        "\(providerName) / \(modelKey)"
    }
}
