import Foundation

struct DaemonSecretReferenceUpsertRequest: Encodable {
    let workspaceID: String
    let name: String
    let backend: String
    let service: String
    let account: String

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case name
        case backend
        case service
        case account
    }
}

struct DaemonSecretReferenceRecord: Decodable, Sendable {
    let workspaceID: String
    let name: String
    let backend: String?
    let service: String
    let account: String

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case name
        case backend
        case service
        case account
    }
}

struct DaemonSecretReferenceResponse: Decodable, Sendable {
    let reference: DaemonSecretReferenceRecord
    let correlationID: String?

    enum CodingKeys: String, CodingKey {
        case reference
        case correlationID = "correlation_id"
    }
}

struct DaemonProviderCheckRequest: Encodable {
    let workspaceID: String
    let provider: String?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case provider
    }
}

struct DaemonProviderSetRequest: Encodable {
    let workspaceID: String
    let provider: String
    let endpoint: String?
    let apiKeySecretName: String?
    let clearAPIKey: Bool

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case provider
        case endpoint
        case apiKeySecretName = "api_key_secret_name"
        case clearAPIKey = "clear_api_key"
    }
}

struct DaemonProviderConfigRecord: Decodable, Sendable {
    let workspaceID: String
    let provider: String
    let endpoint: String
    let apiKeySecretName: String?
    let apiKeyConfigured: Bool
    let updatedAt: String

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case provider
        case endpoint
        case apiKeySecretName = "api_key_secret_name"
        case apiKeyConfigured = "api_key_configured"
        case updatedAt = "updated_at"
    }
}

struct DaemonProviderListResponse: Decodable, Sendable {
    let workspaceID: String
    let providers: [DaemonProviderConfigRecord]

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case providers
    }
}

struct DaemonProviderCheckItem: Decodable, Sendable {
    let provider: String
    let endpoint: String
    let success: Bool
    let statusCode: Int
    let latencyMS: Int64
    let message: String

    enum CodingKeys: String, CodingKey {
        case provider
        case endpoint
        case success
        case statusCode = "status_code"
        case latencyMS = "latency_ms"
        case message
    }
}

struct DaemonProviderCheckResponse: Decodable, Sendable {
    let workspaceID: String
    let success: Bool
    let results: [DaemonProviderCheckItem]

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case success
        case results
    }
}

struct DaemonModelResolveRequest: Encodable {
    let workspaceID: String
    let taskClass: String

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case taskClass = "task_class"
    }
}

struct DaemonModelListRequest: Encodable {
    let workspaceID: String
    let provider: String?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case provider
    }
}

struct DaemonModelDiscoverRequest: Encodable {
    let workspaceID: String
    let provider: String?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case provider
    }
}

struct DaemonModelDiscoverItem: Decodable, Sendable {
    let provider: String
    let modelKey: String
    let displayName: String
    let source: String
    let inCatalog: Bool
    let enabled: Bool

    enum CodingKeys: String, CodingKey {
        case provider
        case modelKey = "model_key"
        case displayName = "display_name"
        case source
        case inCatalog = "in_catalog"
        case enabled
    }
}

struct DaemonModelDiscoverProviderResult: Decodable, Sendable {
    let provider: String
    let providerReady: Bool
    let providerEndpoint: String?
    let success: Bool
    let message: String?
    let models: [DaemonModelDiscoverItem]

    enum CodingKeys: String, CodingKey {
        case provider
        case providerReady = "provider_ready"
        case providerEndpoint = "provider_endpoint"
        case success
        case message
        case models
    }
}

struct DaemonModelDiscoverResponse: Decodable, Sendable {
    let workspaceID: String
    let results: [DaemonModelDiscoverProviderResult]

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case results
    }
}

struct DaemonModelCatalogAddRequest: Encodable {
    let workspaceID: String
    let provider: String
    let modelKey: String
    let enabled: Bool

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case provider
        case modelKey = "model_key"
        case enabled
    }
}

struct DaemonModelCatalogRemoveRequest: Encodable {
    let workspaceID: String
    let provider: String
    let modelKey: String

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case provider
        case modelKey = "model_key"
    }
}

struct DaemonModelCatalogRemoveResponse: Decodable, Sendable {
    let workspaceID: String
    let provider: String
    let modelKey: String
    let removed: Bool
    let removedAt: String?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case provider
        case modelKey = "model_key"
        case removed
        case removedAt = "removed_at"
    }
}

struct DaemonModelToggleRequest: Encodable {
    let workspaceID: String
    let provider: String
    let modelKey: String

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case provider
        case modelKey = "model_key"
    }
}

struct DaemonModelCatalogRecord: Decodable, Sendable {
    let workspaceID: String
    let provider: String
    let modelKey: String
    let enabled: Bool
    let providerReady: Bool
    let providerEndpoint: String?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case provider
        case modelKey = "model_key"
        case enabled
        case providerReady = "provider_ready"
        case providerEndpoint = "provider_endpoint"
    }
}

struct DaemonModelListResponse: Decodable, Sendable {
    let workspaceID: String
    let models: [DaemonModelCatalogRecord]

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case models
    }
}

struct DaemonModelCatalogEntryRecord: Decodable, Sendable {
    let workspaceID: String
    let provider: String
    let modelKey: String
    let enabled: Bool
    let updatedAt: String

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case provider
        case modelKey = "model_key"
        case enabled
        case updatedAt = "updated_at"
    }
}

struct DaemonModelSelectRequest: Encodable {
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

struct DaemonModelPolicyRequest: Encodable {
    let workspaceID: String
    let taskClass: String?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case taskClass = "task_class"
    }
}

struct DaemonModelRoutingPolicyRecord: Decodable, Sendable {
    let workspaceID: String
    let taskClass: String
    let provider: String
    let modelKey: String
    let updatedAt: String

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case taskClass = "task_class"
        case provider
        case modelKey = "model_key"
        case updatedAt = "updated_at"
    }
}

struct DaemonModelPolicyResponse: Decodable, Sendable {
    let workspaceID: String
    let policy: DaemonModelRoutingPolicyRecord?
    let policies: [DaemonModelRoutingPolicyRecord]?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case policy
        case policies
    }
}

struct DaemonModelResolveResponse: Decodable, Sendable {
    let workspaceID: String
    let taskClass: String
    let provider: String
    let modelKey: String
    let source: String
    let notes: String?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case taskClass = "task_class"
        case provider
        case modelKey = "model_key"
        case source
        case notes
    }
}

struct DaemonModelRouteSimulationRequest: Encodable {
    let workspaceID: String
    let taskClass: String
    let principalActorID: String?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case taskClass = "task_class"
        case principalActorID = "principal_actor_id"
    }
}

struct DaemonModelRouteExplainRequest: Encodable {
    let workspaceID: String
    let taskClass: String
    let principalActorID: String?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case taskClass = "task_class"
        case principalActorID = "principal_actor_id"
    }
}

struct DaemonModelRouteDecision: Decodable, Sendable {
    let step: String
    let decision: String
    let reasonCode: String
    let provider: String?
    let modelKey: String?
    let note: String?

    enum CodingKeys: String, CodingKey {
        case step
        case decision
        case reasonCode = "reason_code"
        case provider
        case modelKey = "model_key"
        case note
    }
}

struct DaemonModelRouteFallbackDecision: Decodable, Sendable {
    let rank: Int
    let provider: String
    let modelKey: String
    let selected: Bool
    let reasonCode: String

    enum CodingKeys: String, CodingKey {
        case rank
        case provider
        case modelKey = "model_key"
        case selected
        case reasonCode = "reason_code"
    }
}

struct DaemonModelRouteSimulationResponse: Decodable, Sendable {
    let workspaceID: String
    let taskClass: String
    let principalActorID: String?
    let selectedProvider: String
    let selectedModelKey: String
    let selectedSource: String
    let notes: String?
    let reasonCodes: [String]
    let decisions: [DaemonModelRouteDecision]
    let fallbackChain: [DaemonModelRouteFallbackDecision]

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case taskClass = "task_class"
        case principalActorID = "principal_actor_id"
        case selectedProvider = "selected_provider"
        case selectedModelKey = "selected_model_key"
        case selectedSource = "selected_source"
        case notes
        case reasonCodes = "reason_codes"
        case decisions
        case fallbackChain = "fallback_chain"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        taskClass = try container.decodeIfPresent(String.self, forKey: .taskClass) ?? "chat"
        principalActorID = try container.decodeIfPresent(String.self, forKey: .principalActorID)
        selectedProvider = try container.decodeIfPresent(String.self, forKey: .selectedProvider) ?? ""
        selectedModelKey = try container.decodeIfPresent(String.self, forKey: .selectedModelKey) ?? ""
        selectedSource = try container.decodeIfPresent(String.self, forKey: .selectedSource) ?? "unknown"
        notes = try container.decodeIfPresent(String.self, forKey: .notes)
        reasonCodes = try container.decodeIfPresent([String].self, forKey: .reasonCodes) ?? []
        decisions = try container.decodeIfPresent([DaemonModelRouteDecision].self, forKey: .decisions) ?? []
        fallbackChain = try container.decodeIfPresent([DaemonModelRouteFallbackDecision].self, forKey: .fallbackChain) ?? []
    }
}

struct DaemonModelRouteExplainResponse: Decodable, Sendable {
    let workspaceID: String
    let taskClass: String
    let principalActorID: String?
    let selectedProvider: String
    let selectedModelKey: String
    let selectedSource: String
    let summary: String
    let explanations: [String]
    let reasonCodes: [String]
    let decisions: [DaemonModelRouteDecision]
    let fallbackChain: [DaemonModelRouteFallbackDecision]

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case taskClass = "task_class"
        case principalActorID = "principal_actor_id"
        case selectedProvider = "selected_provider"
        case selectedModelKey = "selected_model_key"
        case selectedSource = "selected_source"
        case summary
        case explanations
        case reasonCodes = "reason_codes"
        case decisions
        case fallbackChain = "fallback_chain"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        taskClass = try container.decodeIfPresent(String.self, forKey: .taskClass) ?? "chat"
        principalActorID = try container.decodeIfPresent(String.self, forKey: .principalActorID)
        selectedProvider = try container.decodeIfPresent(String.self, forKey: .selectedProvider) ?? ""
        selectedModelKey = try container.decodeIfPresent(String.self, forKey: .selectedModelKey) ?? ""
        selectedSource = try container.decodeIfPresent(String.self, forKey: .selectedSource) ?? "unknown"
        summary = try container.decodeIfPresent(String.self, forKey: .summary) ?? ""
        explanations = try container.decodeIfPresent([String].self, forKey: .explanations) ?? []
        reasonCodes = try container.decodeIfPresent([String].self, forKey: .reasonCodes) ?? []
        decisions = try container.decodeIfPresent([DaemonModelRouteDecision].self, forKey: .decisions) ?? []
        fallbackChain = try container.decodeIfPresent([DaemonModelRouteFallbackDecision].self, forKey: .fallbackChain) ?? []
    }
}
