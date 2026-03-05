import Foundation

struct DaemonPluginWorkerStatusCard: Decodable, Sendable {
    let pluginID: String
    let kind: String
    let state: String
    let processID: Int
    let restartCount: Int
    let lastError: String?
    let lastHeartbeat: String?
    let lastTransition: String?

    enum CodingKeys: String, CodingKey {
        case pluginID = "plugin_id"
        case kind
        case state
        case processID = "process_id"
        case restartCount = "restart_count"
        case lastError = "last_error"
        case lastHeartbeat = "last_heartbeat"
        case lastTransition = "last_transition"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        pluginID = try container.decodeIfPresent(String.self, forKey: .pluginID) ?? ""
        kind = try container.decodeIfPresent(String.self, forKey: .kind) ?? "unknown"
        state = try container.decodeIfPresent(String.self, forKey: .state) ?? "unknown"
        processID = try container.decodeIfPresent(Int.self, forKey: .processID) ?? 0
        restartCount = try container.decodeIfPresent(Int.self, forKey: .restartCount) ?? 0
        lastError = try container.decodeIfPresent(String.self, forKey: .lastError)
        lastHeartbeat = try container.decodeIfPresent(String.self, forKey: .lastHeartbeat)
        lastTransition = try container.decodeIfPresent(String.self, forKey: .lastTransition)
    }
}

struct DaemonConfigFieldDescriptor: Decodable, Sendable {
    let key: String
    let label: String
    let type: String
    let required: Bool
    let enumOptions: [String]
    let editable: Bool
    let secret: Bool
    let writeOnly: Bool
    let helpText: String?

    enum CodingKeys: String, CodingKey {
        case key
        case label
        case type
        case required
        case enumOptions = "enum_options"
        case editable
        case secret
        case writeOnly = "write_only"
        case helpText = "help_text"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        key = try container.decodeIfPresent(String.self, forKey: .key) ?? ""
        label = try container.decodeIfPresent(String.self, forKey: .label) ?? key
        type = try container.decodeIfPresent(String.self, forKey: .type) ?? "string"
        required = try container.decodeIfPresent(Bool.self, forKey: .required) ?? false
        enumOptions = try container.decodeIfPresent([String].self, forKey: .enumOptions) ?? []
        editable = try container.decodeIfPresent(Bool.self, forKey: .editable) ?? true
        secret = try container.decodeIfPresent(Bool.self, forKey: .secret) ?? false
        writeOnly = try container.decodeIfPresent(Bool.self, forKey: .writeOnly) ?? false
        helpText = try container.decodeIfPresent(String.self, forKey: .helpText)
    }
}

struct DaemonActionReadinessBlocker: Decodable, Sendable {
    let code: String
    let message: String
    let remediationAction: String?

    enum CodingKeys: String, CodingKey {
        case code
        case message
        case remediationAction = "remediation_action"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        code = try container.decodeIfPresent(String.self, forKey: .code) ?? ""
        message = try container.decodeIfPresent(String.self, forKey: .message) ?? ""
        remediationAction = try container.decodeIfPresent(String.self, forKey: .remediationAction)
    }
}

struct DaemonChannelStatusCard: Decodable, Sendable {
    let channelID: String
    let displayName: String
    let category: String
    let enabled: Bool
    let configured: Bool
    let status: String
    let summary: String?
    let configuration: DaemonUIStatusConfiguration?
    let configFieldDescriptors: [DaemonConfigFieldDescriptor]
    let capabilities: [String]?
    let actionReadiness: String
    let actionBlockers: [DaemonActionReadinessBlocker]
    let remediationActions: [DaemonDiagnosticsRemediationAction]?
    let worker: DaemonPluginWorkerStatusCard?

    enum CodingKeys: String, CodingKey {
        case channelID = "channel_id"
        case displayName = "display_name"
        case category
        case enabled
        case configured
        case status
        case summary
        case configuration
        case configFieldDescriptors = "config_field_descriptors"
        case capabilities
        case actionReadiness = "action_readiness"
        case actionBlockers = "action_blockers"
        case remediationActions = "remediation_actions"
        case worker
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        channelID = try container.decodeIfPresent(String.self, forKey: .channelID) ?? ""
        displayName = try container.decodeIfPresent(String.self, forKey: .displayName) ?? channelID
        category = try container.decodeIfPresent(String.self, forKey: .category) ?? "unknown"
        enabled = try container.decodeIfPresent(Bool.self, forKey: .enabled) ?? true
        configured = try container.decodeIfPresent(Bool.self, forKey: .configured) ?? false
        status = try container.decodeIfPresent(String.self, forKey: .status) ?? "unknown"
        summary = try container.decodeIfPresent(String.self, forKey: .summary)
        configuration = try container.decodeIfPresent(DaemonUIStatusConfiguration.self, forKey: .configuration)
        configFieldDescriptors = try container.decodeIfPresent([DaemonConfigFieldDescriptor].self, forKey: .configFieldDescriptors) ?? []
        capabilities = try container.decodeIfPresent([String].self, forKey: .capabilities)
        actionReadiness = try container.decodeIfPresent(String.self, forKey: .actionReadiness) ?? ""
        actionBlockers = try container.decodeIfPresent([DaemonActionReadinessBlocker].self, forKey: .actionBlockers) ?? []
        remediationActions = try container.decodeIfPresent([DaemonDiagnosticsRemediationAction].self, forKey: .remediationActions)
        worker = try container.decodeIfPresent(DaemonPluginWorkerStatusCard.self, forKey: .worker)
    }
}

struct DaemonChannelStatusResponse: Decodable, Sendable {
    let workspaceID: String
    let channels: [DaemonChannelStatusCard]

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case channels
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        channels = try container.decodeIfPresent([DaemonChannelStatusCard].self, forKey: .channels) ?? []
    }
}

struct DaemonChannelConnectorMappingListRequest: Encodable {
    let workspaceID: String
    let channelID: String?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case channelID = "channel_id"
    }
}

struct DaemonChannelConnectorMappingRecord: Decodable, Sendable, Equatable {
    let channelID: String
    let connectorID: String
    let enabled: Bool
    let priority: Int
    let capabilities: [String]
    let createdAt: String?
    let updatedAt: String?

    enum CodingKeys: String, CodingKey {
        case channelID = "channel_id"
        case connectorID = "connector_id"
        case enabled
        case priority
        case capabilities
        case createdAt = "created_at"
        case updatedAt = "updated_at"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        channelID = try container.decodeIfPresent(String.self, forKey: .channelID) ?? ""
        connectorID = try container.decodeIfPresent(String.self, forKey: .connectorID) ?? ""
        enabled = try container.decodeIfPresent(Bool.self, forKey: .enabled) ?? false
        priority = try container.decodeIfPresent(Int.self, forKey: .priority) ?? 0
        capabilities = try container.decodeIfPresent([String].self, forKey: .capabilities) ?? []
        createdAt = try container.decodeIfPresent(String.self, forKey: .createdAt)
        updatedAt = try container.decodeIfPresent(String.self, forKey: .updatedAt)
    }
}

struct DaemonChannelConnectorMappingListResponse: Decodable, Sendable {
    let workspaceID: String
    let channelID: String?
    let fallbackPolicy: String
    let bindings: [DaemonChannelConnectorMappingRecord]

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case channelID = "channel_id"
        case fallbackPolicy = "fallback_policy"
        case bindings
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        channelID = try container.decodeIfPresent(String.self, forKey: .channelID)
        fallbackPolicy = try container.decodeIfPresent(String.self, forKey: .fallbackPolicy) ?? "priority_order"
        bindings = try container.decodeIfPresent([DaemonChannelConnectorMappingRecord].self, forKey: .bindings) ?? []
    }
}

struct DaemonChannelConnectorMappingUpsertRequest: Encodable {
    let workspaceID: String
    let channelID: String
    let connectorID: String
    let enabled: Bool
    let priority: Int?
    let fallbackPolicy: String?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case channelID = "channel_id"
        case connectorID = "connector_id"
        case enabled
        case priority
        case fallbackPolicy = "fallback_policy"
    }
}

struct DaemonChannelConnectorMappingUpsertResponse: Decodable, Sendable {
    let workspaceID: String
    let channelID: String
    let connectorID: String
    let enabled: Bool
    let priority: Int
    let fallbackPolicy: String
    let updatedAt: String
    let bindings: [DaemonChannelConnectorMappingRecord]

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case channelID = "channel_id"
        case connectorID = "connector_id"
        case enabled
        case priority
        case fallbackPolicy = "fallback_policy"
        case updatedAt = "updated_at"
        case bindings
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        channelID = try container.decodeIfPresent(String.self, forKey: .channelID) ?? ""
        connectorID = try container.decodeIfPresent(String.self, forKey: .connectorID) ?? ""
        enabled = try container.decodeIfPresent(Bool.self, forKey: .enabled) ?? false
        priority = try container.decodeIfPresent(Int.self, forKey: .priority) ?? 0
        fallbackPolicy = try container.decodeIfPresent(String.self, forKey: .fallbackPolicy) ?? "priority_order"
        updatedAt = try container.decodeIfPresent(String.self, forKey: .updatedAt) ?? ""
        bindings = try container.decodeIfPresent([DaemonChannelConnectorMappingRecord].self, forKey: .bindings) ?? []
    }
}

struct DaemonConnectorStatusCard: Decodable, Sendable {
    let connectorID: String
    let pluginID: String
    let displayName: String
    let enabled: Bool
    let configured: Bool
    let status: String
    let summary: String?
    let configuration: DaemonUIStatusConfiguration?
    let configFieldDescriptors: [DaemonConfigFieldDescriptor]
    let capabilities: [String]?
    let actionReadiness: String
    let actionBlockers: [DaemonActionReadinessBlocker]
    let remediationActions: [DaemonDiagnosticsRemediationAction]?
    let worker: DaemonPluginWorkerStatusCard?

    enum CodingKeys: String, CodingKey {
        case connectorID = "connector_id"
        case pluginID = "plugin_id"
        case displayName = "display_name"
        case enabled
        case configured
        case status
        case summary
        case configuration
        case configFieldDescriptors = "config_field_descriptors"
        case capabilities
        case actionReadiness = "action_readiness"
        case actionBlockers = "action_blockers"
        case remediationActions = "remediation_actions"
        case worker
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        connectorID = try container.decodeIfPresent(String.self, forKey: .connectorID) ?? ""
        pluginID = try container.decodeIfPresent(String.self, forKey: .pluginID) ?? connectorID
        displayName = try container.decodeIfPresent(String.self, forKey: .displayName) ?? connectorID
        enabled = try container.decodeIfPresent(Bool.self, forKey: .enabled) ?? true
        configured = try container.decodeIfPresent(Bool.self, forKey: .configured) ?? false
        status = try container.decodeIfPresent(String.self, forKey: .status) ?? "unknown"
        summary = try container.decodeIfPresent(String.self, forKey: .summary)
        configuration = try container.decodeIfPresent(DaemonUIStatusConfiguration.self, forKey: .configuration)
        configFieldDescriptors = try container.decodeIfPresent([DaemonConfigFieldDescriptor].self, forKey: .configFieldDescriptors) ?? []
        capabilities = try container.decodeIfPresent([String].self, forKey: .capabilities)
        actionReadiness = try container.decodeIfPresent(String.self, forKey: .actionReadiness) ?? ""
        actionBlockers = try container.decodeIfPresent([DaemonActionReadinessBlocker].self, forKey: .actionBlockers) ?? []
        remediationActions = try container.decodeIfPresent([DaemonDiagnosticsRemediationAction].self, forKey: .remediationActions)
        worker = try container.decodeIfPresent(DaemonPluginWorkerStatusCard.self, forKey: .worker)
    }
}

struct DaemonConnectorStatusResponse: Decodable, Sendable {
    let workspaceID: String
    let connectors: [DaemonConnectorStatusCard]

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case connectors
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        connectors = try container.decodeIfPresent([DaemonConnectorStatusCard].self, forKey: .connectors) ?? []
    }
}

struct DaemonChannelDiagnosticsRequest: Encodable {
    let workspaceID: String
    let channelID: String?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case channelID = "channel_id"
    }
}

struct DaemonConnectorDiagnosticsRequest: Encodable {
    let workspaceID: String
    let connectorID: String?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case connectorID = "connector_id"
    }
}

struct DaemonConnectorPermissionRequest: Encodable {
    let workspaceID: String
    let connectorID: String

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case connectorID = "connector_id"
    }
}

struct DaemonChannelConfigUpsertRequest: Encodable {
    let workspaceID: String
    let channelID: String
    let configuration: [String: DaemonConfigMutationValue]
    let merge: Bool

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case channelID = "channel_id"
        case configuration
        case merge
    }
}

struct DaemonChannelConfigUpsertResponse: Decodable, Sendable {
    let workspaceID: String
    let channelID: String
    let configuration: [String: DaemonJSONValue]
    let updatedAt: String

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case channelID = "channel_id"
        case configuration
        case updatedAt = "updated_at"
    }
}

struct DaemonConnectorConfigUpsertRequest: Encodable {
    let workspaceID: String
    let connectorID: String
    let configuration: [String: DaemonConfigMutationValue]
    let merge: Bool

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case connectorID = "connector_id"
        case configuration
        case merge
    }
}

struct DaemonConnectorConfigUpsertResponse: Decodable, Sendable {
    let workspaceID: String
    let connectorID: String
    let configuration: [String: DaemonJSONValue]
    let updatedAt: String

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case connectorID = "connector_id"
        case configuration
        case updatedAt = "updated_at"
    }
}

struct DaemonChannelTestOperationRequest: Encodable {
    let workspaceID: String
    let channelID: String
    let operation: String

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case channelID = "channel_id"
        case operation
    }
}

struct DaemonChannelTestOperationResponse: Decodable, Sendable {
    let workspaceID: String
    let channelID: String
    let operation: String
    let success: Bool
    let status: String
    let summary: String
    let checkedAt: String
    let details: DaemonUIStatusTestOperationDetails?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case channelID = "channel_id"
        case operation
        case success
        case status
        case summary
        case checkedAt = "checked_at"
        case details
    }
}

struct DaemonConnectorTestOperationRequest: Encodable {
    let workspaceID: String
    let connectorID: String
    let operation: String

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case connectorID = "connector_id"
        case operation
    }
}

struct DaemonConnectorTestOperationResponse: Decodable, Sendable {
    let workspaceID: String
    let connectorID: String
    let operation: String
    let success: Bool
    let status: String
    let summary: String
    let checkedAt: String
    let details: DaemonUIStatusTestOperationDetails?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case connectorID = "connector_id"
        case operation
        case success
        case status
        case summary
        case checkedAt = "checked_at"
        case details
    }
}

struct DaemonConnectorPermissionResponse: Decodable, Sendable {
    let workspaceID: String
    let connectorID: String
    let permissionState: String
    let message: String?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case connectorID = "connector_id"
        case permissionState = "permission_state"
        case message
    }
}

struct DaemonWorkerHealthSnapshot: Decodable, Sendable {
    let registered: Bool
    let worker: DaemonPluginWorkerStatusCard?

    enum CodingKeys: String, CodingKey {
        case registered
        case worker
    }

    init(registered: Bool, worker: DaemonPluginWorkerStatusCard?) {
        self.registered = registered
        self.worker = worker
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        registered = try container.decodeIfPresent(Bool.self, forKey: .registered) ?? false
        worker = try container.decodeIfPresent(DaemonPluginWorkerStatusCard.self, forKey: .worker)
    }
}

struct DaemonDiagnosticsRemediationAction: Decodable, Sendable {
    let identifier: String
    let label: String
    let intent: String
    let destination: String?
    let parameters: [String: String]
    let enabled: Bool
    let recommended: Bool
    let reason: String?

    enum CodingKeys: String, CodingKey {
        case identifier
        case actionID = "action_id"
        case label
        case title
        case intent
        case destination
        case target
        case parameters
        case enabled
        case recommended
        case reason
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        let resolvedIdentifier = try container.decodeIfPresent(String.self, forKey: .identifier)
            ?? container.decodeIfPresent(String.self, forKey: .actionID)
            ?? "unknown_action"
        let resolvedDestination = try container.decodeIfPresent(String.self, forKey: .destination)
            ?? container.decodeIfPresent(String.self, forKey: .target)
        let resolvedIntent = try container.decodeIfPresent(String.self, forKey: .intent)
            ?? Self.inferIntent(identifier: resolvedIdentifier, destination: resolvedDestination)

        identifier = resolvedIdentifier
        label = try container.decodeIfPresent(String.self, forKey: .label)
            ?? container.decodeIfPresent(String.self, forKey: .title)
            ?? resolvedIdentifier
        intent = resolvedIntent
        destination = resolvedDestination
        parameters = try container.decodeIfPresent([String: String].self, forKey: .parameters) ?? [:]
        enabled = try container.decodeIfPresent(Bool.self, forKey: .enabled) ?? true
        recommended = try container.decodeIfPresent(Bool.self, forKey: .recommended) ?? false
        reason = try container.decodeIfPresent(String.self, forKey: .reason)
    }

    private static func inferIntent(identifier: String, destination: String?) -> String {
        switch identifier {
        case "refresh_channel_status", "refresh_connector_status":
            return "refresh_status"
        case "open_channel_setup", "configure_twilio_channel", "open_channel_logs", "open_connector_logs":
            return "navigate"
        case "repair_daemon_runtime":
            return "daemon_lifecycle_control"
        case "request_connector_permission":
            return "request_permission"
        case "open_connector_system_settings":
            return "open_system_settings"
        default:
            if let destination, destination.hasPrefix("/v1/daemon/lifecycle/control") {
                return "daemon_lifecycle_control"
            }
            if let destination, destination.hasPrefix("ui://") {
                return "navigate"
            }
            if let destination, destination.hasPrefix("/v1/") {
                return "refresh_status"
            }
            return "unknown"
        }
    }
}

struct DaemonChannelDiagnosticsSummary: Decodable, Sendable {
    let channelID: String
    let displayName: String
    let category: String
    let configured: Bool
    let status: String
    let summary: String?
    let workerHealth: DaemonWorkerHealthSnapshot
    let remediationActions: [DaemonDiagnosticsRemediationAction]

    enum CodingKeys: String, CodingKey {
        case channelID = "channel_id"
        case displayName = "display_name"
        case category
        case configured
        case status
        case summary
        case workerHealth = "worker_health"
        case remediationActions = "remediation_actions"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        channelID = try container.decodeIfPresent(String.self, forKey: .channelID) ?? ""
        displayName = try container.decodeIfPresent(String.self, forKey: .displayName) ?? channelID
        category = try container.decodeIfPresent(String.self, forKey: .category) ?? "unknown"
        configured = try container.decodeIfPresent(Bool.self, forKey: .configured) ?? false
        status = try container.decodeIfPresent(String.self, forKey: .status) ?? "unknown"
        summary = try container.decodeIfPresent(String.self, forKey: .summary)
        workerHealth = try container.decodeIfPresent(DaemonWorkerHealthSnapshot.self, forKey: .workerHealth)
            ?? DaemonWorkerHealthSnapshot(registered: false, worker: nil)
        remediationActions = try container.decodeIfPresent([DaemonDiagnosticsRemediationAction].self, forKey: .remediationActions) ?? []
    }
}

struct DaemonChannelDiagnosticsResponse: Decodable, Sendable {
    let workspaceID: String
    let diagnostics: [DaemonChannelDiagnosticsSummary]

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case diagnostics
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        diagnostics = try container.decodeIfPresent([DaemonChannelDiagnosticsSummary].self, forKey: .diagnostics) ?? []
    }
}

struct DaemonConnectorDiagnosticsSummary: Decodable, Sendable {
    let connectorID: String
    let pluginID: String
    let displayName: String
    let configured: Bool
    let status: String
    let summary: String?
    let workerHealth: DaemonWorkerHealthSnapshot
    let remediationActions: [DaemonDiagnosticsRemediationAction]

    enum CodingKeys: String, CodingKey {
        case connectorID = "connector_id"
        case pluginID = "plugin_id"
        case displayName = "display_name"
        case configured
        case status
        case summary
        case workerHealth = "worker_health"
        case remediationActions = "remediation_actions"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        connectorID = try container.decodeIfPresent(String.self, forKey: .connectorID) ?? ""
        pluginID = try container.decodeIfPresent(String.self, forKey: .pluginID) ?? connectorID
        displayName = try container.decodeIfPresent(String.self, forKey: .displayName) ?? connectorID
        configured = try container.decodeIfPresent(Bool.self, forKey: .configured) ?? false
        status = try container.decodeIfPresent(String.self, forKey: .status) ?? "unknown"
        summary = try container.decodeIfPresent(String.self, forKey: .summary)
        workerHealth = try container.decodeIfPresent(DaemonWorkerHealthSnapshot.self, forKey: .workerHealth)
            ?? DaemonWorkerHealthSnapshot(registered: false, worker: nil)
        remediationActions = try container.decodeIfPresent([DaemonDiagnosticsRemediationAction].self, forKey: .remediationActions) ?? []
    }
}

struct DaemonConnectorDiagnosticsResponse: Decodable, Sendable {
    let workspaceID: String
    let diagnostics: [DaemonConnectorDiagnosticsSummary]

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case diagnostics
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        diagnostics = try container.decodeIfPresent([DaemonConnectorDiagnosticsSummary].self, forKey: .diagnostics) ?? []
    }
}

struct DaemonChannelDeliveryPolicy: Decodable, Sendable {
    let primaryChannel: String
    let retryCount: Int
    let fallbackChannels: [String]

    enum CodingKeys: String, CodingKey {
        case primaryChannel = "primary_channel"
        case retryCount = "retry_count"
        case fallbackChannels = "fallback_channels"
    }

    init(primaryChannel: String, retryCount: Int, fallbackChannels: [String]) {
        self.primaryChannel = primaryChannel
        self.retryCount = retryCount
        self.fallbackChannels = fallbackChannels
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        primaryChannel = try container.decodeIfPresent(String.self, forKey: .primaryChannel) ?? ""
        retryCount = try container.decodeIfPresent(Int.self, forKey: .retryCount) ?? 0
        fallbackChannels = try container.decodeIfPresent([String].self, forKey: .fallbackChannels) ?? []
    }
}
