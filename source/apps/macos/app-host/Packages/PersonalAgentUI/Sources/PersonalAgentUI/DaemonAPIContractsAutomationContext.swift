import Foundation

struct DaemonWorkspaceRequest: Encodable {
    let workspaceID: String

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
    }
}

struct DaemonAutomationListRequest: Encodable {
    let workspaceID: String
    let triggerType: String?
    let includeDisabled: Bool

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case triggerType = "trigger_type"
        case includeDisabled = "include_disabled"
    }
}

struct DaemonAutomationTriggerRecord: Decodable, Sendable {
    let triggerID: String
    let workspaceID: String
    let directiveID: String
    let triggerType: String
    let enabled: Bool
    let filterJSON: String
    let cooldownSeconds: Int
    let subjectPrincipalActor: String
    let directiveTitle: String
    let directiveInstruction: String
    let directiveStatus: String
    let createdAt: String
    let updatedAt: String

    enum CodingKeys: String, CodingKey {
        case triggerID = "trigger_id"
        case workspaceID = "workspace_id"
        case directiveID = "directive_id"
        case triggerType = "trigger_type"
        case enabled
        case filterJSON = "filter_json"
        case cooldownSeconds = "cooldown_seconds"
        case subjectPrincipalActor = "subject_principal_actor"
        case directiveTitle = "directive_title"
        case directiveInstruction = "directive_instruction"
        case directiveStatus = "directive_status"
        case createdAt = "created_at"
        case updatedAt = "updated_at"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        triggerID = try container.decode(String.self, forKey: .triggerID)
        workspaceID = try container.decode(String.self, forKey: .workspaceID)
        directiveID = try container.decode(String.self, forKey: .directiveID)
        triggerType = try container.decode(String.self, forKey: .triggerType)
        enabled = try container.decode(Bool.self, forKey: .enabled)
        filterJSON = try container.decode(String.self, forKey: .filterJSON)
        cooldownSeconds = try container.decodeIfPresent(Int.self, forKey: .cooldownSeconds) ?? 0
        subjectPrincipalActor = try container.decodeIfPresent(String.self, forKey: .subjectPrincipalActor) ?? ""
        directiveTitle = try container.decode(String.self, forKey: .directiveTitle)
        directiveInstruction = try container.decode(String.self, forKey: .directiveInstruction)
        directiveStatus = try container.decode(String.self, forKey: .directiveStatus)
        createdAt = try container.decode(String.self, forKey: .createdAt)
        updatedAt = try container.decode(String.self, forKey: .updatedAt)
    }
}

struct DaemonAutomationListResponse: Decodable, Sendable {
    let workspaceID: String
    let triggers: [DaemonAutomationTriggerRecord]

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case triggers
    }
}

struct DaemonAutomationFireHistoryRequest: Encodable {
    let workspaceID: String
    let triggerID: String?
    let status: String?
    let limit: Int

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case triggerID = "trigger_id"
        case status
        case limit
    }
}

struct DaemonAutomationFireHistoryRecord: Decodable, Sendable {
    let fireID: String
    let workspaceID: String
    let triggerID: String
    let triggerType: String
    let directiveID: String?
    let status: String
    let outcome: String?
    let idempotencyKey: String
    let idempotencySignal: String
    let firedAt: String
    let taskID: String?
    let runID: String?
    let route: DaemonWorkflowRouteMetadata?

    enum CodingKeys: String, CodingKey {
        case fireID = "fire_id"
        case workspaceID = "workspace_id"
        case triggerID = "trigger_id"
        case triggerType = "trigger_type"
        case directiveID = "directive_id"
        case status
        case outcome
        case idempotencyKey = "idempotency_key"
        case idempotencySignal = "idempotency_signal"
        case firedAt = "fired_at"
        case taskID = "task_id"
        case runID = "run_id"
        case route
    }
}

struct DaemonAutomationFireHistoryResponse: Decodable, Sendable {
    let workspaceID: String
    let fires: [DaemonAutomationFireHistoryRecord]

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case fires
    }
}

struct DaemonAutomationCreateRequest: Encodable {
    let workspaceID: String
    let subjectActorID: String?
    let triggerType: String
    let title: String?
    let instruction: String?
    let intervalSeconds: Int?
    let filterJSON: String?
    let cooldownSeconds: Int?
    let enabled: Bool

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case subjectActorID = "subject_actor_id"
        case triggerType = "trigger_type"
        case title
        case instruction
        case intervalSeconds = "interval_seconds"
        case filterJSON = "filter_json"
        case cooldownSeconds = "cooldown_seconds"
        case enabled
    }
}

struct DaemonAutomationUpdateRequest: Encodable {
    let workspaceID: String
    let triggerID: String
    let subjectActorID: String?
    let title: String?
    let instruction: String?
    let intervalSeconds: Int?
    let filterJSON: String?
    let cooldownSeconds: Int?
    let enabled: Bool?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case triggerID = "trigger_id"
        case subjectActorID = "subject_actor_id"
        case title
        case instruction
        case intervalSeconds = "interval_seconds"
        case filterJSON = "filter_json"
        case cooldownSeconds = "cooldown_seconds"
        case enabled
    }
}

struct DaemonAutomationUpdateResponse: Decodable, Sendable {
    let trigger: DaemonAutomationTriggerRecord
    let updated: Bool
    let idempotent: Bool
}

struct DaemonAutomationDeleteRequest: Encodable {
    let workspaceID: String
    let triggerID: String

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case triggerID = "trigger_id"
    }
}

struct DaemonAutomationDeleteResponse: Decodable, Sendable {
    let workspaceID: String
    let triggerID: String
    let directiveID: String?
    let deleted: Bool
    let idempotent: Bool

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case triggerID = "trigger_id"
        case directiveID = "directive_id"
        case deleted
        case idempotent
    }
}

struct DaemonAutomationRunScheduleRequest: Encodable {
    let at: String?
}

struct DaemonAutomationRunScheduleResponse: Decodable, Sendable {
    let at: String
    let result: DaemonJSONValue
}

struct DaemonAutomationRunCommEventRequest: Encodable {
    let workspaceID: String
    let eventID: String
    let seedEvent: Bool
    let threadID: String?
    let channel: String?
    let body: String?
    let sender: String?
    let eventType: String?
    let direction: String?
    let assistantEmitted: Bool
    let occurredAt: String?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case eventID = "event_id"
        case seedEvent = "seed_event"
        case threadID = "thread_id"
        case channel
        case body
        case sender
        case eventType = "event_type"
        case direction
        case assistantEmitted = "assistant_emitted"
        case occurredAt = "occurred_at"
    }
}

struct DaemonAutomationRunCommEventResponse: Decodable, Sendable {
    let eventID: String
    let seededEvent: Bool
    let result: DaemonJSONValue

    enum CodingKeys: String, CodingKey {
        case eventID = "event_id"
        case seededEvent = "seeded_event"
        case result
    }
}

struct DaemonRetentionPurgeRequest: Encodable {
    let traceDays: Int
    let transcriptDays: Int
    let memoryDays: Int

    enum CodingKeys: String, CodingKey {
        case traceDays = "trace_days"
        case transcriptDays = "transcript_days"
        case memoryDays = "memory_days"
    }
}

struct DaemonRetentionCompactMemoryRequest: Encodable {
    let workspaceID: String
    let ownerActor: String
    let tokenThreshold: Int
    let staleAfterHours: Int
    let limit: Int
    let apply: Bool

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case ownerActor = "owner_actor"
        case tokenThreshold = "token_threshold"
        case staleAfterHours = "stale_after_hours"
        case limit
        case apply
    }
}

struct DaemonContextSamplesRequest: Encodable {
    let workspaceID: String
    let taskClass: String
    let limit: Int

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case taskClass = "task_class"
        case limit
    }
}

struct DaemonContextTuneRequest: Encodable {
    let workspaceID: String
    let taskClass: String

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case taskClass = "task_class"
    }
}

struct DaemonContextMemoryInventoryRequest: Encodable {
    let workspaceID: String
    let ownerActorID: String?
    let scopeType: String?
    let status: String?
    let sourceType: String?
    let sourceRefQuery: String?
    let cursorUpdatedAt: String?
    let cursorID: String?
    let limit: Int

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case ownerActorID = "owner_actor_id"
        case scopeType = "scope_type"
        case status
        case sourceType = "source_type"
        case sourceRefQuery = "source_ref_query"
        case cursorUpdatedAt = "cursor_updated_at"
        case cursorID = "cursor_id"
        case limit
    }
}

struct DaemonContextMemorySourceRecord: Decodable, Sendable {
    let sourceID: String
    let sourceType: String
    let sourceRef: String
    let createdAt: String

    enum CodingKeys: String, CodingKey {
        case sourceID = "source_id"
        case sourceType = "source_type"
        case sourceRef = "source_ref"
        case createdAt = "created_at"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        sourceID = try container.decodeIfPresent(String.self, forKey: .sourceID) ?? ""
        sourceType = try container.decodeIfPresent(String.self, forKey: .sourceType) ?? "unknown"
        sourceRef = try container.decodeIfPresent(String.self, forKey: .sourceRef) ?? ""
        createdAt = try container.decodeIfPresent(String.self, forKey: .createdAt) ?? ""
    }
}

struct DaemonContextMemoryInventoryRecord: Decodable, Sendable {
    let memoryID: String
    let workspaceID: String
    let ownerActorID: String
    let scopeType: String
    let key: String
    let status: String
    let kind: String
    let isCanonical: Bool
    let tokenEstimate: Int
    let sourceSummary: String
    let sourceCount: Int
    let createdAt: String
    let updatedAt: String
    let valueJSON: String
    let sources: [DaemonContextMemorySourceRecord]

    enum CodingKeys: String, CodingKey {
        case memoryID = "memory_id"
        case workspaceID = "workspace_id"
        case ownerActorID = "owner_actor_id"
        case scopeType = "scope_type"
        case key
        case status
        case kind
        case isCanonical = "is_canonical"
        case tokenEstimate = "token_estimate"
        case sourceSummary = "source_summary"
        case sourceCount = "source_count"
        case createdAt = "created_at"
        case updatedAt = "updated_at"
        case valueJSON = "value_json"
        case sources
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        memoryID = try container.decodeIfPresent(String.self, forKey: .memoryID) ?? ""
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        ownerActorID = try container.decodeIfPresent(String.self, forKey: .ownerActorID) ?? ""
        scopeType = try container.decodeIfPresent(String.self, forKey: .scopeType) ?? "unknown"
        key = try container.decodeIfPresent(String.self, forKey: .key) ?? ""
        status = try container.decodeIfPresent(String.self, forKey: .status) ?? "unknown"
        kind = try container.decodeIfPresent(String.self, forKey: .kind) ?? "unknown"
        isCanonical = try container.decodeIfPresent(Bool.self, forKey: .isCanonical) ?? false
        tokenEstimate = try container.decodeIfPresent(Int.self, forKey: .tokenEstimate) ?? 0
        sourceSummary = try container.decodeIfPresent(String.self, forKey: .sourceSummary) ?? ""
        sourceCount = try container.decodeIfPresent(Int.self, forKey: .sourceCount) ?? 0
        createdAt = try container.decodeIfPresent(String.self, forKey: .createdAt) ?? ""
        updatedAt = try container.decodeIfPresent(String.self, forKey: .updatedAt) ?? ""
        valueJSON = try container.decodeIfPresent(String.self, forKey: .valueJSON) ?? ""
        sources = try container.decodeIfPresent([DaemonContextMemorySourceRecord].self, forKey: .sources) ?? []
    }
}

struct DaemonContextMemoryInventoryResponse: Decodable, Sendable {
    let workspaceID: String
    let items: [DaemonContextMemoryInventoryRecord]
    let hasMore: Bool
    let nextCursorUpdatedAt: String?
    let nextCursorID: String?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case items
        case hasMore = "has_more"
        case nextCursorUpdatedAt = "next_cursor_updated_at"
        case nextCursorID = "next_cursor_id"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        items = try container.decodeIfPresent([DaemonContextMemoryInventoryRecord].self, forKey: .items) ?? []
        hasMore = try container.decodeIfPresent(Bool.self, forKey: .hasMore) ?? false
        nextCursorUpdatedAt = try container.decodeIfPresent(String.self, forKey: .nextCursorUpdatedAt)
        nextCursorID = try container.decodeIfPresent(String.self, forKey: .nextCursorID)
    }
}

struct DaemonContextMemoryCandidatesRequest: Encodable {
    let workspaceID: String
    let ownerActorID: String?
    let status: String?
    let cursorCreatedAt: String?
    let cursorID: String?
    let limit: Int

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case ownerActorID = "owner_actor_id"
        case status
        case cursorCreatedAt = "cursor_created_at"
        case cursorID = "cursor_id"
        case limit
    }
}

struct DaemonContextMemoryCandidateRecord: Decodable, Sendable {
    let candidateID: String
    let workspaceID: String
    let ownerActorID: String
    let status: String
    let score: Double
    let candidateJSON: String
    let candidateKind: String
    let tokenEstimate: Int
    let sourceIDs: [String]
    let sourceRefs: [String]
    let createdAt: String

    enum CodingKeys: String, CodingKey {
        case candidateID = "candidate_id"
        case workspaceID = "workspace_id"
        case ownerActorID = "owner_actor_id"
        case status
        case score
        case candidateJSON = "candidate_json"
        case candidateKind = "candidate_kind"
        case tokenEstimate = "token_estimate"
        case sourceIDs = "source_ids"
        case sourceRefs = "source_refs"
        case createdAt = "created_at"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        candidateID = try container.decodeIfPresent(String.self, forKey: .candidateID) ?? ""
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        ownerActorID = try container.decodeIfPresent(String.self, forKey: .ownerActorID) ?? ""
        status = try container.decodeIfPresent(String.self, forKey: .status) ?? "unknown"
        score = try container.decodeIfPresent(Double.self, forKey: .score) ?? 0
        candidateJSON = try container.decodeIfPresent(String.self, forKey: .candidateJSON) ?? ""
        candidateKind = try container.decodeIfPresent(String.self, forKey: .candidateKind) ?? "unknown"
        tokenEstimate = try container.decodeIfPresent(Int.self, forKey: .tokenEstimate) ?? 0
        sourceIDs = try container.decodeIfPresent([String].self, forKey: .sourceIDs) ?? []
        sourceRefs = try container.decodeIfPresent([String].self, forKey: .sourceRefs) ?? []
        createdAt = try container.decodeIfPresent(String.self, forKey: .createdAt) ?? ""
    }
}

struct DaemonContextMemoryCandidatesResponse: Decodable, Sendable {
    let workspaceID: String
    let items: [DaemonContextMemoryCandidateRecord]
    let hasMore: Bool
    let nextCursorCreatedAt: String?
    let nextCursorID: String?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case items
        case hasMore = "has_more"
        case nextCursorCreatedAt = "next_cursor_created_at"
        case nextCursorID = "next_cursor_id"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        items = try container.decodeIfPresent([DaemonContextMemoryCandidateRecord].self, forKey: .items) ?? []
        hasMore = try container.decodeIfPresent(Bool.self, forKey: .hasMore) ?? false
        nextCursorCreatedAt = try container.decodeIfPresent(String.self, forKey: .nextCursorCreatedAt)
        nextCursorID = try container.decodeIfPresent(String.self, forKey: .nextCursorID)
    }
}

struct DaemonContextRetrievalDocumentsRequest: Encodable {
    let workspaceID: String
    let ownerActorID: String?
    let sourceURIQuery: String?
    let cursorCreatedAt: String?
    let cursorID: String?
    let limit: Int

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case ownerActorID = "owner_actor_id"
        case sourceURIQuery = "source_uri_query"
        case cursorCreatedAt = "cursor_created_at"
        case cursorID = "cursor_id"
        case limit
    }
}

struct DaemonContextRetrievalDocumentRecord: Decodable, Sendable {
    let documentID: String
    let workspaceID: String
    let ownerActorID: String
    let sourceURI: String
    let checksum: String
    let chunkCount: Int
    let createdAt: String

    enum CodingKeys: String, CodingKey {
        case documentID = "document_id"
        case workspaceID = "workspace_id"
        case ownerActorID = "owner_actor_id"
        case sourceURI = "source_uri"
        case checksum
        case chunkCount = "chunk_count"
        case createdAt = "created_at"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        documentID = try container.decodeIfPresent(String.self, forKey: .documentID) ?? ""
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        ownerActorID = try container.decodeIfPresent(String.self, forKey: .ownerActorID) ?? ""
        sourceURI = try container.decodeIfPresent(String.self, forKey: .sourceURI) ?? ""
        checksum = try container.decodeIfPresent(String.self, forKey: .checksum) ?? ""
        chunkCount = try container.decodeIfPresent(Int.self, forKey: .chunkCount) ?? 0
        createdAt = try container.decodeIfPresent(String.self, forKey: .createdAt) ?? ""
    }
}

struct DaemonContextRetrievalDocumentsResponse: Decodable, Sendable {
    let workspaceID: String
    let items: [DaemonContextRetrievalDocumentRecord]
    let hasMore: Bool
    let nextCursorCreatedAt: String?
    let nextCursorID: String?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case items
        case hasMore = "has_more"
        case nextCursorCreatedAt = "next_cursor_created_at"
        case nextCursorID = "next_cursor_id"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        items = try container.decodeIfPresent([DaemonContextRetrievalDocumentRecord].self, forKey: .items) ?? []
        hasMore = try container.decodeIfPresent(Bool.self, forKey: .hasMore) ?? false
        nextCursorCreatedAt = try container.decodeIfPresent(String.self, forKey: .nextCursorCreatedAt)
        nextCursorID = try container.decodeIfPresent(String.self, forKey: .nextCursorID)
    }
}

struct DaemonContextRetrievalChunksRequest: Encodable {
    let workspaceID: String
    let documentID: String
    let ownerActorID: String?
    let sourceURIQuery: String?
    let chunkTextQuery: String?
    let cursorCreatedAt: String?
    let cursorID: String?
    let limit: Int

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case documentID = "document_id"
        case ownerActorID = "owner_actor_id"
        case sourceURIQuery = "source_uri_query"
        case chunkTextQuery = "chunk_text_query"
        case cursorCreatedAt = "cursor_created_at"
        case cursorID = "cursor_id"
        case limit
    }
}

struct DaemonContextRetrievalChunkRecord: Decodable, Sendable {
    let chunkID: String
    let workspaceID: String
    let documentID: String
    let ownerActorID: String
    let sourceURI: String
    let chunkIndex: Int
    let textBody: String
    let tokenCount: Int
    let createdAt: String

    enum CodingKeys: String, CodingKey {
        case chunkID = "chunk_id"
        case workspaceID = "workspace_id"
        case documentID = "document_id"
        case ownerActorID = "owner_actor_id"
        case sourceURI = "source_uri"
        case chunkIndex = "chunk_index"
        case textBody = "text_body"
        case tokenCount = "token_count"
        case createdAt = "created_at"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        chunkID = try container.decodeIfPresent(String.self, forKey: .chunkID) ?? ""
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        documentID = try container.decodeIfPresent(String.self, forKey: .documentID) ?? ""
        ownerActorID = try container.decodeIfPresent(String.self, forKey: .ownerActorID) ?? ""
        sourceURI = try container.decodeIfPresent(String.self, forKey: .sourceURI) ?? ""
        chunkIndex = try container.decodeIfPresent(Int.self, forKey: .chunkIndex) ?? 0
        textBody = try container.decodeIfPresent(String.self, forKey: .textBody) ?? ""
        tokenCount = try container.decodeIfPresent(Int.self, forKey: .tokenCount) ?? 0
        createdAt = try container.decodeIfPresent(String.self, forKey: .createdAt) ?? ""
    }
}

struct DaemonContextRetrievalChunksResponse: Decodable, Sendable {
    let workspaceID: String
    let documentID: String
    let items: [DaemonContextRetrievalChunkRecord]
    let hasMore: Bool
    let nextCursorCreatedAt: String?
    let nextCursorID: String?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case documentID = "document_id"
        case items
        case hasMore = "has_more"
        case nextCursorCreatedAt = "next_cursor_created_at"
        case nextCursorID = "next_cursor_id"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        documentID = try container.decodeIfPresent(String.self, forKey: .documentID) ?? ""
        items = try container.decodeIfPresent([DaemonContextRetrievalChunkRecord].self, forKey: .items) ?? []
        hasMore = try container.decodeIfPresent(Bool.self, forKey: .hasMore) ?? false
        nextCursorCreatedAt = try container.decodeIfPresent(String.self, forKey: .nextCursorCreatedAt)
        nextCursorID = try container.decodeIfPresent(String.self, forKey: .nextCursorID)
    }
}
