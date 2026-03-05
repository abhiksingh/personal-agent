import Foundation

struct DaemonChatTurnChannelContext: Codable, Sendable {
    let channelID: String?
    let connectorID: String?
    let threadID: String?

    enum CodingKeys: String, CodingKey {
        case channelID = "channel_id"
        case connectorID = "connector_id"
        case threadID = "thread_id"
    }

    init(
        channelID: String? = nil,
        connectorID: String? = nil,
        threadID: String? = nil
    ) {
        self.channelID = channelID
        self.connectorID = connectorID
        self.threadID = threadID
    }
}

struct DaemonChatTurnItem: Decodable, Sendable {
    let itemID: String?
    let type: String
    let role: String?
    let status: String?
    let content: String?
    let toolName: String?
    let toolCallID: String?
    let arguments: [String: DaemonJSONValue]?
    let output: [String: DaemonJSONValue]?
    let errorCode: String?
    let errorMessage: String?
    let approvalRequestID: String?
    let metadata: DaemonChatTurnItemMetadata?

    enum CodingKeys: String, CodingKey {
        case itemID = "item_id"
        case type
        case role
        case status
        case content
        case toolName = "tool_name"
        case toolCallID = "tool_call_id"
        case arguments
        case output
        case errorCode = "error_code"
        case errorMessage = "error_message"
        case approvalRequestID = "approval_request_id"
        case metadata
    }

    init(
        itemID: String? = nil,
        type: String,
        role: String? = nil,
        status: String? = nil,
        content: String? = nil,
        toolName: String? = nil,
        toolCallID: String? = nil,
        arguments: [String: DaemonJSONValue]? = nil,
        output: [String: DaemonJSONValue]? = nil,
        errorCode: String? = nil,
        errorMessage: String? = nil,
        approvalRequestID: String? = nil,
        metadata: DaemonChatTurnItemMetadata? = nil
    ) {
        self.itemID = itemID
        self.type = type
        self.role = role
        self.status = status
        self.content = content
        self.toolName = toolName
        self.toolCallID = toolCallID
        self.arguments = arguments
        self.output = output
        self.errorCode = errorCode
        self.errorMessage = errorMessage
        self.approvalRequestID = approvalRequestID
        self.metadata = metadata
    }
}

struct DaemonChatTurnRequestItem: Encodable {
    let type: String
    let role: String?
    let status: String?
    let content: String?

    enum CodingKeys: String, CodingKey {
        case type
        case role
        case status
        case content
    }
}

struct DaemonChatTurnRequest: Encodable {
    let workspaceID: String
    let taskClass: String
    let requestedByActorID: String?
    let subjectActorID: String?
    let actingAsActorID: String?
    let provider: String?
    let model: String?
    let systemPrompt: String?
    let channel: DaemonChatTurnChannelContext?
    let items: [DaemonChatTurnRequestItem]

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case taskClass = "task_class"
        case requestedByActorID = "requested_by_actor_id"
        case subjectActorID = "subject_actor_id"
        case actingAsActorID = "acting_as_actor_id"
        case provider
        case model
        case systemPrompt = "system_prompt"
        case channel
        case items
    }
}

struct DaemonChatTurnHistoryRequest: Encodable {
    let workspaceID: String
    let channelID: String?
    let connectorID: String?
    let threadID: String?
    let correlationID: String?
    let beforeCreatedAt: String?
    let beforeItemID: String?
    let limit: Int

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case channelID = "channel_id"
        case connectorID = "connector_id"
        case threadID = "thread_id"
        case correlationID = "correlation_id"
        case beforeCreatedAt = "before_created_at"
        case beforeItemID = "before_item_id"
        case limit
    }
}

struct DaemonChatTurnExplainRequest: Encodable {
    let workspaceID: String
    let taskClass: String
    let requestedByActorID: String?
    let subjectActorID: String?
    let actingAsActorID: String?
    let channel: DaemonChatTurnChannelContext?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case taskClass = "task_class"
        case requestedByActorID = "requested_by_actor_id"
        case subjectActorID = "subject_actor_id"
        case actingAsActorID = "acting_as_actor_id"
        case channel
    }
}

struct DaemonChatPersonaPolicyRequest: Encodable {
    let workspaceID: String
    let principalActorID: String?
    let channelID: String?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case principalActorID = "principal_actor_id"
        case channelID = "channel_id"
    }
}

struct DaemonChatPersonaPolicyUpsertRequest: Encodable {
    let workspaceID: String
    let principalActorID: String?
    let channelID: String?
    let stylePrompt: String
    let guardrails: [String]

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case principalActorID = "principal_actor_id"
        case channelID = "channel_id"
        case stylePrompt = "style_prompt"
        case guardrails
    }
}

struct DaemonChatTurnResponse: Decodable, Sendable {
    let workspaceID: String
    let taskClass: String
    let provider: String
    let modelKey: String
    let correlationID: String
    let contractVersion: String?
    let turnItemSchemaVersion: String?
    let realtimeEventContractVersion: String?
    let channel: DaemonChatTurnChannelContext?
    let items: [DaemonChatTurnItem]
    let taskRunCorrelation: DaemonChatTurnTaskRunCorrelation

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case taskClass = "task_class"
        case provider
        case modelKey = "model_key"
        case correlationID = "correlation_id"
        case contractVersion = "contract_version"
        case turnItemSchemaVersion = "turn_item_schema_version"
        case realtimeEventContractVersion = "realtime_event_contract_version"
        case channel
        case items
        case taskRunCorrelation = "task_run_correlation"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        taskClass = try container.decodeIfPresent(String.self, forKey: .taskClass) ?? "chat"
        provider = try container.decodeIfPresent(String.self, forKey: .provider) ?? "unknown"
        modelKey = try container.decodeIfPresent(String.self, forKey: .modelKey) ?? "unknown"
        correlationID = try container.decodeIfPresent(String.self, forKey: .correlationID) ?? ""
        contractVersion = try container.decodeIfPresent(String.self, forKey: .contractVersion)
        turnItemSchemaVersion = try container.decodeIfPresent(String.self, forKey: .turnItemSchemaVersion)
        realtimeEventContractVersion = try container.decodeIfPresent(String.self, forKey: .realtimeEventContractVersion)
        channel = try container.decodeIfPresent(DaemonChatTurnChannelContext.self, forKey: .channel)
        items = try container.decodeIfPresent([DaemonChatTurnItem].self, forKey: .items) ?? []
        taskRunCorrelation = try container.decodeIfPresent(
            DaemonChatTurnTaskRunCorrelation.self,
            forKey: .taskRunCorrelation
        ) ?? DaemonChatTurnTaskRunCorrelation()
    }
}

struct DaemonChatTurnToolCatalogEntry: Decodable, Sendable {
    let name: String
    let description: String?
    let capabilityKeys: [String]
    let inputSchema: [String: DaemonJSONValue]?

    enum CodingKeys: String, CodingKey {
        case name
        case description
        case capabilityKeys = "capability_keys"
        case inputSchema = "input_schema"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        name = try container.decodeIfPresent(String.self, forKey: .name) ?? ""
        description = try container.decodeIfPresent(String.self, forKey: .description)
        capabilityKeys = try container.decodeIfPresent([String].self, forKey: .capabilityKeys) ?? []
        inputSchema = try container.decodeIfPresent([String: DaemonJSONValue].self, forKey: .inputSchema)
    }
}

struct DaemonChatTurnToolPolicyDecision: Decodable, Sendable {
    let toolName: String
    let capabilityKey: String?
    let decision: String
    let reason: String?

    enum CodingKeys: String, CodingKey {
        case toolName = "tool_name"
        case capabilityKey = "capability_key"
        case decision
        case reason
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        toolName = try container.decodeIfPresent(String.self, forKey: .toolName) ?? ""
        capabilityKey = try container.decodeIfPresent(String.self, forKey: .capabilityKey)
        decision = try container.decodeIfPresent(String.self, forKey: .decision) ?? "unknown"
        reason = try container.decodeIfPresent(String.self, forKey: .reason)
    }
}

struct DaemonChatTurnExplainResponse: Decodable, Sendable {
    let workspaceID: String
    let taskClass: String
    let requestedByActorID: String?
    let subjectActorID: String?
    let actingAsActorID: String?
    let channel: DaemonChatTurnChannelContext?
    let contractVersion: String
    let selectedRoute: DaemonModelRouteExplainResponse?
    let toolCatalog: [DaemonChatTurnToolCatalogEntry]
    let policyDecisions: [DaemonChatTurnToolPolicyDecision]

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case taskClass = "task_class"
        case requestedByActorID = "requested_by_actor_id"
        case subjectActorID = "subject_actor_id"
        case actingAsActorID = "acting_as_actor_id"
        case channel
        case contractVersion = "contract_version"
        case selectedRoute = "selected_route"
        case toolCatalog = "tool_catalog"
        case policyDecisions = "policy_decisions"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        taskClass = try container.decodeIfPresent(String.self, forKey: .taskClass) ?? "chat"
        requestedByActorID = try container.decodeIfPresent(String.self, forKey: .requestedByActorID)
        subjectActorID = try container.decodeIfPresent(String.self, forKey: .subjectActorID)
        actingAsActorID = try container.decodeIfPresent(String.self, forKey: .actingAsActorID)
        channel = try container.decodeIfPresent(DaemonChatTurnChannelContext.self, forKey: .channel)
        contractVersion = try container.decodeIfPresent(String.self, forKey: .contractVersion) ?? "chat_turn_explain.v1"
        selectedRoute = try container.decodeIfPresent(DaemonModelRouteExplainResponse.self, forKey: .selectedRoute)
        toolCatalog = try container.decodeIfPresent([DaemonChatTurnToolCatalogEntry].self, forKey: .toolCatalog) ?? []
        policyDecisions = try container.decodeIfPresent([DaemonChatTurnToolPolicyDecision].self, forKey: .policyDecisions) ?? []
    }
}

struct DaemonChatTurnHistoryRecord: Decodable, Sendable {
    let recordID: String
    let turnID: String
    let workspaceID: String
    let taskClass: String
    let correlationID: String
    let channelID: String
    let connectorID: String?
    let threadID: String?
    let itemIndex: Int
    let item: DaemonChatTurnItem
    let taskRunReference: DaemonChatTurnTaskRunCorrelation
    let createdAt: String

    enum CodingKeys: String, CodingKey {
        case recordID = "record_id"
        case turnID = "turn_id"
        case workspaceID = "workspace_id"
        case taskClass = "task_class"
        case correlationID = "correlation_id"
        case channelID = "channel_id"
        case connectorID = "connector_id"
        case threadID = "thread_id"
        case itemIndex = "item_index"
        case item
        case taskRunReference = "task_run_reference"
        case createdAt = "created_at"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        recordID = try container.decodeIfPresent(String.self, forKey: .recordID) ?? ""
        turnID = try container.decodeIfPresent(String.self, forKey: .turnID) ?? ""
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        taskClass = try container.decodeIfPresent(String.self, forKey: .taskClass) ?? "chat"
        correlationID = try container.decodeIfPresent(String.self, forKey: .correlationID) ?? ""
        channelID = try container.decodeIfPresent(String.self, forKey: .channelID) ?? "unknown"
        connectorID = try container.decodeIfPresent(String.self, forKey: .connectorID)
        threadID = try container.decodeIfPresent(String.self, forKey: .threadID)
        itemIndex = try container.decodeIfPresent(Int.self, forKey: .itemIndex) ?? 0
        item = try container.decodeIfPresent(DaemonChatTurnItem.self, forKey: .item)
            ?? DaemonChatTurnItem(type: "assistant_message", content: "")
        taskRunReference = try container.decodeIfPresent(
            DaemonChatTurnTaskRunCorrelation.self,
            forKey: .taskRunReference
        ) ?? DaemonChatTurnTaskRunCorrelation()
        createdAt = try container.decodeIfPresent(String.self, forKey: .createdAt) ?? ""
    }
}

struct DaemonChatTurnHistoryResponse: Decodable, Sendable {
    let workspaceID: String
    let items: [DaemonChatTurnHistoryRecord]
    let hasMore: Bool
    let nextCursorCreatedAt: String?
    let nextCursorItemID: String?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case items
        case hasMore = "has_more"
        case nextCursorCreatedAt = "next_cursor_created_at"
        case nextCursorItemID = "next_cursor_item_id"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        items = try container.decodeIfPresent([DaemonChatTurnHistoryRecord].self, forKey: .items) ?? []
        hasMore = try container.decodeIfPresent(Bool.self, forKey: .hasMore) ?? false
        nextCursorCreatedAt = try container.decodeIfPresent(String.self, forKey: .nextCursorCreatedAt)
        nextCursorItemID = try container.decodeIfPresent(String.self, forKey: .nextCursorItemID)
    }
}

struct DaemonChatPersonaPolicyResponse: Decodable, Sendable {
    let workspaceID: String
    let principalActorID: String?
    let channelID: String?
    let stylePrompt: String
    let guardrails: [String]
    let source: String
    let updatedAt: String?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case principalActorID = "principal_actor_id"
        case channelID = "channel_id"
        case stylePrompt = "style_prompt"
        case guardrails
        case source
        case updatedAt = "updated_at"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        principalActorID = try container.decodeIfPresent(String.self, forKey: .principalActorID)
        channelID = try container.decodeIfPresent(String.self, forKey: .channelID)
        stylePrompt = try container.decodeIfPresent(String.self, forKey: .stylePrompt) ?? ""
        guardrails = try container.decodeIfPresent([String].self, forKey: .guardrails) ?? []
        source = try container.decodeIfPresent(String.self, forKey: .source) ?? "default"
        updatedAt = try container.decodeIfPresent(String.self, forKey: .updatedAt)
    }
}

struct DaemonChatTurnTaskRunCorrelation: Decodable, Sendable {
    let available: Bool
    let source: String
    let taskID: String?
    let runID: String?
    let taskState: String?
    let runState: String?

    enum CodingKeys: String, CodingKey {
        case available
        case source
        case taskID = "task_id"
        case runID = "run_id"
        case taskState = "task_state"
        case runState = "run_state"
    }

    init(
        available: Bool = false,
        source: String = "none",
        taskID: String? = nil,
        runID: String? = nil,
        taskState: String? = nil,
        runState: String? = nil
    ) {
        self.available = available
        self.source = source
        self.taskID = taskID
        self.runID = runID
        self.taskState = taskState
        self.runState = runState
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        available = try container.decodeIfPresent(Bool.self, forKey: .available) ?? false
        source = try container.decodeIfPresent(String.self, forKey: .source) ?? "none"
        taskID = try container.decodeIfPresent(String.self, forKey: .taskID)
        runID = try container.decodeIfPresent(String.self, forKey: .runID)
        taskState = try container.decodeIfPresent(String.self, forKey: .taskState)
        runState = try container.decodeIfPresent(String.self, forKey: .runState)
    }
}

struct DaemonRealtimeEventEnvelope: Decodable, Sendable {
    let eventID: String
    let sequence: Int64
    let eventType: String
    let occurredAt: String
    let correlationID: String?
    let contractVersion: String?
    let lifecycleSchemaVersion: String?
    let payload: DaemonRealtimeEventPayload

    enum CodingKeys: String, CodingKey {
        case eventID = "event_id"
        case sequence
        case eventType = "event_type"
        case occurredAt = "occurred_at"
        case correlationID = "correlation_id"
        case contractVersion = "contract_version"
        case lifecycleSchemaVersion = "lifecycle_schema_version"
        case payload
    }

    init(
        eventID: String,
        sequence: Int64,
        eventType: String,
        occurredAt: String,
        correlationID: String? = nil,
        contractVersion: String? = nil,
        lifecycleSchemaVersion: String? = nil,
        payload: DaemonRealtimeEventPayload = DaemonRealtimeEventPayload()
    ) {
        self.eventID = eventID
        self.sequence = sequence
        self.eventType = eventType
        self.occurredAt = occurredAt
        self.correlationID = correlationID
        self.contractVersion = contractVersion
        self.lifecycleSchemaVersion = lifecycleSchemaVersion
        self.payload = payload
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        eventID = try container.decodeIfPresent(String.self, forKey: .eventID) ?? UUID().uuidString.lowercased()
        sequence = try container.decodeIfPresent(Int64.self, forKey: .sequence) ?? 0
        eventType = try container.decodeIfPresent(String.self, forKey: .eventType) ?? "unknown"
        occurredAt = try container.decodeIfPresent(String.self, forKey: .occurredAt) ?? ""
        correlationID = try container.decodeIfPresent(String.self, forKey: .correlationID)
        contractVersion = try container.decodeIfPresent(String.self, forKey: .contractVersion)
        lifecycleSchemaVersion = try container.decodeIfPresent(String.self, forKey: .lifecycleSchemaVersion)
        payload = try container.decodeIfPresent(DaemonRealtimeEventPayload.self, forKey: .payload)
            ?? DaemonRealtimeEventPayload()
    }
}

struct DaemonRealtimeClientSignal: Encodable, Sendable {
    let signalType: String
    let taskID: String?
    let runID: String?
    let reason: String?
    let correlationID: String?

    enum CodingKeys: String, CodingKey {
        case signalType = "signal_type"
        case taskID = "task_id"
        case runID = "run_id"
        case reason
        case correlationID = "correlation_id"
    }
}
