import Foundation

private struct V2DaemonProviderCheckRequest: Encodable {
    let workspaceID: String
    let provider: String?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case provider
    }
}

private struct V2DaemonModelListRequest: Encodable {
    let workspaceID: String
    let provider: String?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case provider
    }
}

private struct V2DaemonModelPolicyRequest: Encodable {
    let workspaceID: String
    let taskClass: String?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case taskClass = "task_class"
    }
}

private struct V2DaemonModelResolveRequest: Encodable {
    let workspaceID: String
    let taskClass: String

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case taskClass = "task_class"
    }
}

private struct V2DaemonModelToggleRequest: Encodable {
    let workspaceID: String
    let provider: String
    let modelKey: String

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case provider
        case modelKey = "model_key"
    }
}

private struct V2DaemonModelSelectRequest: Encodable {
    let workspaceID: String
    let taskClass: String
    let provider: String
    let modelKey: String

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case taskClass = "task_class"
        case provider
        case modelKey = "model_key"
    }
}

private struct V2DaemonModelRouteSimulationRequest: Encodable {
    let workspaceID: String
    let taskClass: String
    let principalActorID: String?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case taskClass = "task_class"
        case principalActorID = "principal_actor_id"
    }
}

public struct V2DaemonProviderConfigRecord: Decodable, Sendable, Equatable {
    public let workspaceID: String
    public let provider: String
    public let endpoint: String
    public let apiKeyConfigured: Bool

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case provider
        case endpoint
        case apiKeyConfigured = "api_key_configured"
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        provider = try container.decodeIfPresent(String.self, forKey: .provider) ?? ""
        endpoint = try container.decodeIfPresent(String.self, forKey: .endpoint) ?? ""
        apiKeyConfigured = try container.decodeIfPresent(Bool.self, forKey: .apiKeyConfigured) ?? false
    }
}

public struct V2DaemonProviderListResponse: Decodable, Sendable, Equatable {
    public let workspaceID: String
    public let providers: [V2DaemonProviderConfigRecord]

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case providers
    }
}

public struct V2DaemonProviderCheckItem: Decodable, Sendable, Equatable {
    public let provider: String
    public let endpoint: String
    public let success: Bool
    public let statusCode: Int
    public let latencyMS: Int64
    public let message: String

    enum CodingKeys: String, CodingKey {
        case provider
        case endpoint
        case success
        case statusCode = "status_code"
        case latencyMS = "latency_ms"
        case message
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        provider = try container.decodeIfPresent(String.self, forKey: .provider) ?? ""
        endpoint = try container.decodeIfPresent(String.self, forKey: .endpoint) ?? ""
        success = try container.decodeIfPresent(Bool.self, forKey: .success) ?? false
        statusCode = try container.decodeIfPresent(Int.self, forKey: .statusCode) ?? 0
        latencyMS = try container.decodeIfPresent(Int64.self, forKey: .latencyMS) ?? 0
        message = try container.decodeIfPresent(String.self, forKey: .message) ?? ""
    }
}

public struct V2DaemonProviderCheckResponse: Decodable, Sendable, Equatable {
    public let workspaceID: String
    public let success: Bool
    public let results: [V2DaemonProviderCheckItem]

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case success
        case results
    }
}

public struct V2DaemonModelCatalogRecord: Decodable, Sendable, Equatable {
    public let workspaceID: String
    public let provider: String
    public let modelKey: String
    public let enabled: Bool
    public let providerReady: Bool
    public let providerEndpoint: String?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case provider
        case modelKey = "model_key"
        case enabled
        case providerReady = "provider_ready"
        case providerEndpoint = "provider_endpoint"
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        provider = try container.decodeIfPresent(String.self, forKey: .provider) ?? ""
        modelKey = try container.decodeIfPresent(String.self, forKey: .modelKey) ?? ""
        enabled = try container.decodeIfPresent(Bool.self, forKey: .enabled) ?? false
        providerReady = try container.decodeIfPresent(Bool.self, forKey: .providerReady) ?? false
        providerEndpoint = try container.decodeIfPresent(String.self, forKey: .providerEndpoint)
    }
}

public struct V2DaemonModelListResponse: Decodable, Sendable, Equatable {
    public let workspaceID: String
    public let models: [V2DaemonModelCatalogRecord]

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case models
    }
}

public struct V2DaemonModelCatalogEntryRecord: Decodable, Sendable, Equatable {
    public let workspaceID: String
    public let provider: String
    public let modelKey: String
    public let enabled: Bool
    public let updatedAt: String

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case provider
        case modelKey = "model_key"
        case enabled
        case updatedAt = "updated_at"
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        provider = try container.decodeIfPresent(String.self, forKey: .provider) ?? ""
        modelKey = try container.decodeIfPresent(String.self, forKey: .modelKey) ?? ""
        enabled = try container.decodeIfPresent(Bool.self, forKey: .enabled) ?? false
        updatedAt = try container.decodeIfPresent(String.self, forKey: .updatedAt) ?? ""
    }
}

public struct V2DaemonModelRoutingPolicyRecord: Decodable, Sendable, Equatable {
    public let workspaceID: String
    public let taskClass: String
    public let provider: String
    public let modelKey: String

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case taskClass = "task_class"
        case provider
        case modelKey = "model_key"
    }
}

public struct V2DaemonModelPolicyResponse: Decodable, Sendable, Equatable {
    public let workspaceID: String
    public let policy: V2DaemonModelRoutingPolicyRecord?
    public let policies: [V2DaemonModelRoutingPolicyRecord]?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case policy
        case policies
    }
}

public struct V2DaemonModelResolveResponse: Decodable, Sendable, Equatable {
    public let workspaceID: String
    public let taskClass: String
    public let provider: String
    public let modelKey: String
    public let source: String
    public let notes: String?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case taskClass = "task_class"
        case provider
        case modelKey = "model_key"
        case source
        case notes
    }
}

public struct V2DaemonModelRouteDecision: Decodable, Sendable, Equatable {
    public let step: String
    public let decision: String
    public let reasonCode: String
    public let provider: String?
    public let modelKey: String?
    public let note: String?

    enum CodingKeys: String, CodingKey {
        case step
        case decision
        case reasonCode = "reason_code"
        case provider
        case modelKey = "model_key"
        case note
    }
}

public struct V2DaemonModelRouteFallbackDecision: Decodable, Sendable, Equatable {
    public let rank: Int
    public let provider: String
    public let modelKey: String
    public let selected: Bool
    public let reasonCode: String

    enum CodingKeys: String, CodingKey {
        case rank
        case provider
        case modelKey = "model_key"
        case selected
        case reasonCode = "reason_code"
    }
}

public struct V2DaemonModelRouteSimulationResponse: Decodable, Sendable, Equatable {
    public let workspaceID: String
    public let taskClass: String
    public let selectedProvider: String
    public let selectedModelKey: String
    public let selectedSource: String
    public let notes: String?
    public let reasonCodes: [String]
    public let decisions: [V2DaemonModelRouteDecision]
    public let fallbackChain: [V2DaemonModelRouteFallbackDecision]

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case taskClass = "task_class"
        case selectedProvider = "selected_provider"
        case selectedModelKey = "selected_model_key"
        case selectedSource = "selected_source"
        case notes
        case reasonCodes = "reason_codes"
        case decisions
        case fallbackChain = "fallback_chain"
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        taskClass = try container.decodeIfPresent(String.self, forKey: .taskClass) ?? "chat"
        selectedProvider = try container.decodeIfPresent(String.self, forKey: .selectedProvider) ?? ""
        selectedModelKey = try container.decodeIfPresent(String.self, forKey: .selectedModelKey) ?? ""
        selectedSource = try container.decodeIfPresent(String.self, forKey: .selectedSource) ?? "unknown"
        notes = try container.decodeIfPresent(String.self, forKey: .notes)
        reasonCodes = try container.decodeIfPresent([String].self, forKey: .reasonCodes) ?? []
        decisions = try container.decodeIfPresent([V2DaemonModelRouteDecision].self, forKey: .decisions) ?? []
        fallbackChain = try container.decodeIfPresent([V2DaemonModelRouteFallbackDecision].self, forKey: .fallbackChain) ?? []
    }
}

public struct V2DaemonModelRouteExplainResponse: Decodable, Sendable, Equatable {
    public let workspaceID: String
    public let taskClass: String
    public let selectedProvider: String
    public let selectedModelKey: String
    public let selectedSource: String
    public let summary: String
    public let explanations: [String]
    public let reasonCodes: [String]
    public let decisions: [V2DaemonModelRouteDecision]
    public let fallbackChain: [V2DaemonModelRouteFallbackDecision]

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case taskClass = "task_class"
        case selectedProvider = "selected_provider"
        case selectedModelKey = "selected_model_key"
        case selectedSource = "selected_source"
        case summary
        case explanations
        case reasonCodes = "reason_codes"
        case decisions
        case fallbackChain = "fallback_chain"
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        taskClass = try container.decodeIfPresent(String.self, forKey: .taskClass) ?? "chat"
        selectedProvider = try container.decodeIfPresent(String.self, forKey: .selectedProvider) ?? ""
        selectedModelKey = try container.decodeIfPresent(String.self, forKey: .selectedModelKey) ?? ""
        selectedSource = try container.decodeIfPresent(String.self, forKey: .selectedSource) ?? "unknown"
        summary = try container.decodeIfPresent(String.self, forKey: .summary) ?? ""
        explanations = try container.decodeIfPresent([String].self, forKey: .explanations) ?? []
        reasonCodes = try container.decodeIfPresent([String].self, forKey: .reasonCodes) ?? []
        decisions = try container.decodeIfPresent([V2DaemonModelRouteDecision].self, forKey: .decisions) ?? []
        fallbackChain = try container.decodeIfPresent([V2DaemonModelRouteFallbackDecision].self, forKey: .fallbackChain) ?? []
    }
}

public struct V2DaemonModelsAPI {
    private let client: V2DaemonAPIClient

    init(client: V2DaemonAPIClient) {
        self.client = client
    }

    public func providerList(baseURL: URL, authToken: String, workspaceID: String) async throws -> V2DaemonProviderListResponse {
        try await client.request(
            baseURL: baseURL,
            path: "/v1/providers/list",
            method: "POST",
            authToken: authToken,
            body: V2DaemonWorkspaceRequest(workspaceID: workspaceID)
        )
    }

    public func providerCheck(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        provider: String? = nil
    ) async throws -> V2DaemonProviderCheckResponse {
        try await client.request(
            baseURL: baseURL,
            path: "/v1/providers/check",
            method: "POST",
            authToken: authToken,
            body: V2DaemonProviderCheckRequest(workspaceID: workspaceID, provider: provider)
        )
    }

    public func modelList(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        provider: String? = nil
    ) async throws -> V2DaemonModelListResponse {
        try await client.request(
            baseURL: baseURL,
            path: "/v1/models/list",
            method: "POST",
            authToken: authToken,
            body: V2DaemonModelListRequest(workspaceID: workspaceID, provider: provider)
        )
    }

    public func modelPolicy(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        taskClass: String? = nil
    ) async throws -> V2DaemonModelPolicyResponse {
        try await client.request(
            baseURL: baseURL,
            path: "/v1/models/policy",
            method: "POST",
            authToken: authToken,
            body: V2DaemonModelPolicyRequest(workspaceID: workspaceID, taskClass: taskClass)
        )
    }

    public func modelResolve(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        taskClass: String = "chat"
    ) async throws -> V2DaemonModelResolveResponse {
        try await client.request(
            baseURL: baseURL,
            path: "/v1/models/resolve",
            method: "POST",
            authToken: authToken,
            body: V2DaemonModelResolveRequest(workspaceID: workspaceID, taskClass: taskClass)
        )
    }

    public func modelSelect(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        taskClass: String = "chat",
        provider: String,
        modelKey: String
    ) async throws -> V2DaemonModelRoutingPolicyRecord {
        try await client.request(
            baseURL: baseURL,
            path: "/v1/models/select",
            method: "POST",
            authToken: authToken,
            body: V2DaemonModelSelectRequest(
                workspaceID: workspaceID,
                taskClass: taskClass,
                provider: provider,
                modelKey: modelKey
            )
        )
    }

    public func modelEnable(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        provider: String,
        modelKey: String
    ) async throws -> V2DaemonModelCatalogEntryRecord {
        try await client.request(
            baseURL: baseURL,
            path: "/v1/models/enable",
            method: "POST",
            authToken: authToken,
            body: V2DaemonModelToggleRequest(workspaceID: workspaceID, provider: provider, modelKey: modelKey)
        )
    }

    public func modelDisable(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        provider: String,
        modelKey: String
    ) async throws -> V2DaemonModelCatalogEntryRecord {
        try await client.request(
            baseURL: baseURL,
            path: "/v1/models/disable",
            method: "POST",
            authToken: authToken,
            body: V2DaemonModelToggleRequest(workspaceID: workspaceID, provider: provider, modelKey: modelKey)
        )
    }

    public func modelRouteSimulate(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        taskClass: String = "chat",
        principalActorID: String? = nil
    ) async throws -> V2DaemonModelRouteSimulationResponse {
        try await client.request(
            baseURL: baseURL,
            path: "/v1/models/route/simulate",
            method: "POST",
            authToken: authToken,
            body: V2DaemonModelRouteSimulationRequest(
                workspaceID: workspaceID,
                taskClass: taskClass,
                principalActorID: principalActorID
            )
        )
    }

    public func modelRouteExplain(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        taskClass: String = "chat",
        principalActorID: String? = nil
    ) async throws -> V2DaemonModelRouteExplainResponse {
        try await client.request(
            baseURL: baseURL,
            path: "/v1/models/route/explain",
            method: "POST",
            authToken: authToken,
            body: V2DaemonModelRouteSimulationRequest(
                workspaceID: workspaceID,
                taskClass: taskClass,
                principalActorID: principalActorID
            )
        )
    }
}
