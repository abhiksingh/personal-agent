import Foundation

private struct V2DaemonConnectorDiagnosticsRequest: Encodable {
    let workspaceID: String
    let connectorID: String?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case connectorID = "connector_id"
    }
}

private struct V2DaemonConnectorConfigUpsertRequest: Encodable {
    let workspaceID: String
    let connectorID: String
    let configuration: [String: V2DaemonJSONValue]
    let merge: Bool

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case connectorID = "connector_id"
        case configuration
        case merge
    }
}

private struct V2DaemonConnectorTestOperationRequest: Encodable {
    let workspaceID: String
    let connectorID: String
    let operation: String

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case connectorID = "connector_id"
        case operation
    }
}

private struct V2DaemonConnectorPermissionRequest: Encodable {
    let workspaceID: String
    let connectorID: String

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case connectorID = "connector_id"
    }
}

public struct V2DaemonActionReadinessBlocker: Decodable, Sendable, Equatable {
    public let code: String
    public let message: String
    public let remediationAction: String?

    enum CodingKeys: String, CodingKey {
        case code
        case message
        case remediationAction = "remediation_action"
    }

    public init(code: String, message: String, remediationAction: String? = nil) {
        self.code = code
        self.message = message
        self.remediationAction = remediationAction
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        code = try container.decodeIfPresent(String.self, forKey: .code) ?? ""
        message = try container.decodeIfPresent(String.self, forKey: .message) ?? ""
        remediationAction = try container.decodeIfPresent(String.self, forKey: .remediationAction)
    }
}

public struct V2DaemonDiagnosticsRemediationAction: Decodable, Sendable, Equatable {
    public let identifier: String
    public let label: String
    public let intent: String
    public let destination: String?
    public let enabled: Bool
    public let reason: String?

    enum CodingKeys: String, CodingKey {
        case identifier
        case actionID = "action_id"
        case label
        case title
        case intent
        case destination
        case target
        case enabled
        case reason
    }

    public init(
        identifier: String,
        label: String,
        intent: String,
        destination: String? = nil,
        enabled: Bool = true,
        reason: String? = nil
    ) {
        self.identifier = identifier
        self.label = label
        self.intent = intent
        self.destination = destination
        self.enabled = enabled
        self.reason = reason
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        identifier = try container.decodeIfPresent(String.self, forKey: .identifier)
            ?? container.decodeIfPresent(String.self, forKey: .actionID)
            ?? "unknown_action"
        label = try container.decodeIfPresent(String.self, forKey: .label)
            ?? container.decodeIfPresent(String.self, forKey: .title)
            ?? identifier
        destination = try container.decodeIfPresent(String.self, forKey: .destination)
            ?? container.decodeIfPresent(String.self, forKey: .target)
        intent = try container.decodeIfPresent(String.self, forKey: .intent) ?? "unknown"
        enabled = try container.decodeIfPresent(Bool.self, forKey: .enabled) ?? true
        reason = try container.decodeIfPresent(String.self, forKey: .reason)
    }
}

public struct V2DaemonUIStatusMappedConnector: Decodable, Sendable, Equatable {
    public let connectorID: String?
    public let enabled: Bool?
    public let priority: Int?
    public let configured: Bool?
    public let status: String?
    public let summary: String?
    public let additional: [String: V2DaemonJSONValue]

    enum CodingKeys: String, CodingKey {
        case connectorID = "connector_id"
        case enabled
        case priority
        case configured
        case status
        case summary
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        connectorID = try container.decodeLossyString(forKey: .connectorID)
        enabled = try container.decodeLossyBool(forKey: .enabled)
        priority = try container.decodeLossyInt(forKey: .priority)
        configured = try container.decodeLossyBool(forKey: .configured)
        status = try container.decodeLossyString(forKey: .status)
        summary = try container.decodeLossyString(forKey: .summary)

        var extras = v2DecodeDaemonJSONObject(from: decoder)
        for key in ["connector_id", "enabled", "priority", "configured", "status", "summary"] {
            extras.removeValue(forKey: key)
        }
        additional = extras
    }
}

public struct V2DaemonUIStatusConfiguration: Decodable, Sendable, Equatable {
    public let enabled: Bool?
    public let transport: String?
    public let mode: String?
    public let statusReason: String?
    public let primaryConnectorID: String?
    public let mappedConnectorIDs: [String]
    public let enabledConnectorIDs: [String]
    public let mappedConnectors: [V2DaemonUIStatusMappedConnector]
    public let permissionState: String?
    public let additional: [String: V2DaemonJSONValue]

    enum CodingKeys: String, CodingKey {
        case enabled
        case transport
        case mode
        case statusReason = "status_reason"
        case primaryConnectorID = "primary_connector_id"
        case mappedConnectorIDs = "mapped_connector_ids"
        case enabledConnectorIDs = "enabled_connector_ids"
        case mappedConnectors = "mapped_connectors"
        case permissionState = "permission_state"
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        enabled = try container.decodeLossyBool(forKey: .enabled)
        transport = try container.decodeLossyString(forKey: .transport)
        mode = try container.decodeLossyString(forKey: .mode)
        statusReason = try container.decodeLossyString(forKey: .statusReason)
        primaryConnectorID = try container.decodeLossyString(forKey: .primaryConnectorID)
        mappedConnectorIDs = try container.decodeLossyStringArray(forKey: .mappedConnectorIDs) ?? []
        enabledConnectorIDs = try container.decodeLossyStringArray(forKey: .enabledConnectorIDs) ?? []
        mappedConnectors = try container.decodeIfPresent([V2DaemonUIStatusMappedConnector].self, forKey: .mappedConnectors) ?? []
        permissionState = try container.decodeLossyString(forKey: .permissionState)

        var extras = v2DecodeDaemonJSONObject(from: decoder)
        for key in [
            "enabled", "transport", "mode", "status_reason", "primary_connector_id",
            "mapped_connector_ids", "enabled_connector_ids", "mapped_connectors", "permission_state"
        ] {
            extras.removeValue(forKey: key)
        }
        additional = extras
    }
}

public struct V2DaemonConnectorStatusCard: Decodable, Sendable, Equatable {
    public let connectorID: String
    public let pluginID: String
    public let displayName: String
    public let enabled: Bool
    public let configured: Bool
    public let status: String
    public let summary: String?
    public let configuration: V2DaemonUIStatusConfiguration?
    public let actionReadiness: String
    public let actionBlockers: [V2DaemonActionReadinessBlocker]
    public let remediationActions: [V2DaemonDiagnosticsRemediationAction]

    enum CodingKeys: String, CodingKey {
        case connectorID = "connector_id"
        case pluginID = "plugin_id"
        case displayName = "display_name"
        case enabled
        case configured
        case status
        case summary
        case configuration
        case actionReadiness = "action_readiness"
        case actionBlockers = "action_blockers"
        case remediationActions = "remediation_actions"
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        connectorID = try container.decodeIfPresent(String.self, forKey: .connectorID) ?? ""
        pluginID = try container.decodeIfPresent(String.self, forKey: .pluginID) ?? connectorID
        displayName = try container.decodeIfPresent(String.self, forKey: .displayName) ?? connectorID
        enabled = try container.decodeIfPresent(Bool.self, forKey: .enabled) ?? true
        configured = try container.decodeIfPresent(Bool.self, forKey: .configured) ?? false
        status = try container.decodeIfPresent(String.self, forKey: .status) ?? "unknown"
        summary = try container.decodeIfPresent(String.self, forKey: .summary)
        configuration = try container.decodeIfPresent(V2DaemonUIStatusConfiguration.self, forKey: .configuration)
        actionReadiness = try container.decodeIfPresent(String.self, forKey: .actionReadiness) ?? "unknown"
        actionBlockers = try container.decodeIfPresent([V2DaemonActionReadinessBlocker].self, forKey: .actionBlockers) ?? []
        remediationActions = try container.decodeIfPresent([V2DaemonDiagnosticsRemediationAction].self, forKey: .remediationActions) ?? []
    }
}

public struct V2DaemonConnectorStatusResponse: Decodable, Sendable, Equatable {
    public let workspaceID: String
    public let connectors: [V2DaemonConnectorStatusCard]

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case connectors
    }
}

public struct V2DaemonConnectorConfigUpsertResponse: Decodable, Sendable, Equatable {
    public let workspaceID: String
    public let connectorID: String
    public let configuration: [String: V2DaemonJSONValue]
    public let updatedAt: String

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case connectorID = "connector_id"
        case configuration
        case updatedAt = "updated_at"
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        connectorID = try container.decodeIfPresent(String.self, forKey: .connectorID) ?? ""
        configuration = try container.decodeIfPresent([String: V2DaemonJSONValue].self, forKey: .configuration) ?? [:]
        updatedAt = try container.decodeIfPresent(String.self, forKey: .updatedAt) ?? ""
    }
}

public struct V2DaemonUIStatusTestOperationDetails: Decodable, Sendable, Equatable {
    public let pluginID: String?
    public let workerState: String?
    public let configured: Bool?
    public let permissionState: String?
    public let endpoint: String?
    public let additional: [String: V2DaemonJSONValue]

    enum CodingKeys: String, CodingKey {
        case pluginID = "plugin_id"
        case workerState = "worker_state"
        case configured
        case permissionState = "permission_state"
        case endpoint
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        pluginID = try container.decodeLossyString(forKey: .pluginID)
        workerState = try container.decodeLossyString(forKey: .workerState)
        configured = try container.decodeLossyBool(forKey: .configured)
        permissionState = try container.decodeLossyString(forKey: .permissionState)
        endpoint = try container.decodeLossyString(forKey: .endpoint)

        var extras = v2DecodeDaemonJSONObject(from: decoder)
        for key in ["plugin_id", "worker_state", "configured", "permission_state", "endpoint"] {
            extras.removeValue(forKey: key)
        }
        additional = extras
    }
}

public struct V2DaemonConnectorTestOperationResponse: Decodable, Sendable, Equatable {
    public let workspaceID: String
    public let connectorID: String
    public let operation: String
    public let success: Bool
    public let status: String
    public let summary: String
    public let checkedAt: String
    public let details: V2DaemonUIStatusTestOperationDetails?

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

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        connectorID = try container.decodeIfPresent(String.self, forKey: .connectorID) ?? ""
        operation = try container.decodeIfPresent(String.self, forKey: .operation) ?? ""
        success = try container.decodeIfPresent(Bool.self, forKey: .success) ?? false
        status = try container.decodeIfPresent(String.self, forKey: .status) ?? "unknown"
        summary = try container.decodeIfPresent(String.self, forKey: .summary) ?? ""
        checkedAt = try container.decodeIfPresent(String.self, forKey: .checkedAt) ?? ""
        details = try container.decodeIfPresent(V2DaemonUIStatusTestOperationDetails.self, forKey: .details)
    }
}

public struct V2DaemonConnectorPermissionResponse: Decodable, Sendable, Equatable {
    public let workspaceID: String
    public let connectorID: String
    public let permissionState: String
    public let message: String?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case connectorID = "connector_id"
        case permissionState = "permission_state"
        case message
    }
}

public struct V2DaemonConnectorDiagnosticsSummary: Decodable, Sendable, Equatable {
    public let connectorID: String
    public let displayName: String
    public let status: String
    public let summary: String?
    public let remediationActions: [V2DaemonDiagnosticsRemediationAction]

    enum CodingKeys: String, CodingKey {
        case connectorID = "connector_id"
        case displayName = "display_name"
        case status
        case summary
        case remediationActions = "remediation_actions"
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        connectorID = try container.decodeIfPresent(String.self, forKey: .connectorID) ?? ""
        displayName = try container.decodeIfPresent(String.self, forKey: .displayName) ?? connectorID
        status = try container.decodeIfPresent(String.self, forKey: .status) ?? "unknown"
        summary = try container.decodeIfPresent(String.self, forKey: .summary)
        remediationActions = try container.decodeIfPresent([V2DaemonDiagnosticsRemediationAction].self, forKey: .remediationActions) ?? []
    }
}

public struct V2DaemonConnectorDiagnosticsResponse: Decodable, Sendable, Equatable {
    public let workspaceID: String
    public let diagnostics: [V2DaemonConnectorDiagnosticsSummary]

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case diagnostics
    }
}

public struct V2DaemonConnectorsAPI {
    private let client: V2DaemonAPIClient

    init(client: V2DaemonAPIClient) {
        self.client = client
    }

    public func connectorStatus(
        baseURL: URL,
        authToken: String,
        workspaceID: String
    ) async throws -> V2DaemonConnectorStatusResponse {
        try await client.request(
            baseURL: baseURL,
            path: "/v1/connectors/status",
            method: "POST",
            authToken: authToken,
            body: V2DaemonWorkspaceRequest(workspaceID: workspaceID)
        )
    }

    public func connectorConfigUpsert(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        connectorID: String,
        configuration: [String: V2DaemonJSONValue],
        merge: Bool = true
    ) async throws -> V2DaemonConnectorConfigUpsertResponse {
        try await client.request(
            baseURL: baseURL,
            path: "/v1/connectors/config/upsert",
            method: "POST",
            authToken: authToken,
            body: V2DaemonConnectorConfigUpsertRequest(
                workspaceID: workspaceID,
                connectorID: connectorID,
                configuration: configuration,
                merge: merge
            )
        )
    }

    public func connectorTestOperation(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        connectorID: String,
        operation: String
    ) async throws -> V2DaemonConnectorTestOperationResponse {
        try await client.request(
            baseURL: baseURL,
            path: "/v1/connectors/test-operation",
            method: "POST",
            authToken: authToken,
            body: V2DaemonConnectorTestOperationRequest(
                workspaceID: workspaceID,
                connectorID: connectorID,
                operation: operation
            )
        )
    }

    public func connectorPermissionRequest(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        connectorID: String
    ) async throws -> V2DaemonConnectorPermissionResponse {
        try await client.request(
            baseURL: baseURL,
            path: "/v1/connectors/permission/request",
            method: "POST",
            authToken: authToken,
            body: V2DaemonConnectorPermissionRequest(workspaceID: workspaceID, connectorID: connectorID)
        )
    }

    public func connectorDiagnostics(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        connectorID: String? = nil
    ) async throws -> V2DaemonConnectorDiagnosticsResponse {
        try await client.request(
            baseURL: baseURL,
            path: "/v1/connectors/diagnostics",
            method: "POST",
            authToken: authToken,
            body: V2DaemonConnectorDiagnosticsRequest(workspaceID: workspaceID, connectorID: connectorID)
        )
    }
}
