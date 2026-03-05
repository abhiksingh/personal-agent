import Foundation

struct DaemonInspectLogQueryRequest: Encodable {
    let workspaceID: String
    let runID: String?
    let eventType: String?
    let beforeCreatedAt: String?
    let beforeID: String?
    let limit: Int

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case runID = "run_id"
        case eventType = "event_type"
        case beforeCreatedAt = "before_created_at"
        case beforeID = "before_id"
        case limit
    }
}

struct DaemonInspectRunRequest: Encodable {
    let runID: String

    enum CodingKeys: String, CodingKey {
        case runID = "run_id"
    }
}

struct DaemonInspectRunTask: Decodable, Sendable {
    let taskID: String
    let workspaceID: String
    let requestedByActorID: String
    let subjectPrincipalActorID: String
    let title: String
    let description: String?
    let state: String
    let priority: Int
    let deadlineAt: String?
    let channel: String?
    let createdAt: String
    let updatedAt: String

    enum CodingKeys: String, CodingKey {
        case taskID = "task_id"
        case workspaceID = "workspace_id"
        case requestedByActorID = "requested_by_actor_id"
        case subjectPrincipalActorID = "subject_principal_actor_id"
        case title
        case description
        case state
        case priority
        case deadlineAt = "deadline_at"
        case channel
        case createdAt = "created_at"
        case updatedAt = "updated_at"
    }
}

struct DaemonInspectRunRecord: Decodable, Sendable {
    let runID: String
    let workspaceID: String
    let taskID: String
    let actingAsActorID: String
    let state: String
    let startedAt: String?
    let finishedAt: String?
    let lastError: String?
    let createdAt: String
    let updatedAt: String

    enum CodingKeys: String, CodingKey {
        case runID = "run_id"
        case workspaceID = "workspace_id"
        case taskID = "task_id"
        case actingAsActorID = "acting_as_actor_id"
        case state
        case startedAt = "started_at"
        case finishedAt = "finished_at"
        case lastError = "last_error"
        case createdAt = "created_at"
        case updatedAt = "updated_at"
    }
}

struct DaemonInspectRunStep: Decodable, Sendable {
    let stepID: String
    let runID: String
    let stepIndex: Int
    let name: String
    let status: String
    let interactionLevel: String?
    let capabilityKey: String?
    let timeoutSeconds: Int?
    let retryMax: Int
    let retryCount: Int
    let lastError: String?
    let createdAt: String
    let updatedAt: String

    enum CodingKeys: String, CodingKey {
        case stepID = "step_id"
        case runID = "run_id"
        case stepIndex = "step_index"
        case name
        case status
        case interactionLevel = "interaction_level"
        case capabilityKey = "capability_key"
        case timeoutSeconds = "timeout_seconds"
        case retryMax = "retry_max"
        case retryCount = "retry_count"
        case lastError = "last_error"
        case createdAt = "created_at"
        case updatedAt = "updated_at"
    }
}

struct DaemonInspectRunArtifact: Decodable, Sendable {
    let artifactID: String
    let runID: String
    let stepID: String?
    let artifactType: String
    let uri: String?
    let contentHash: String?
    let createdAt: String

    enum CodingKeys: String, CodingKey {
        case artifactID = "artifact_id"
        case runID = "run_id"
        case stepID = "step_id"
        case artifactType = "artifact_type"
        case uri
        case contentHash = "content_hash"
        case createdAt = "created_at"
    }
}

struct DaemonInspectRunAuditEntry: Decodable, Sendable {
    let auditID: String
    let workspaceID: String
    let runID: String?
    let stepID: String?
    let eventType: String
    let actorID: String?
    let actingAsActorID: String?
    let correlationID: String?
    let payloadJSON: String?
    let createdAt: String

    var payloadValue: DaemonJSONValue? {
        guard let payloadJSON,
              let data = payloadJSON.data(using: .utf8) else {
            return nil
        }
        return try? JSONDecoder().decode(DaemonJSONValue.self, from: data)
    }

    var payloadObject: [String: DaemonJSONValue]? {
        payloadValue?.objectValue
    }

    enum CodingKeys: String, CodingKey {
        case auditID = "audit_id"
        case workspaceID = "workspace_id"
        case runID = "run_id"
        case stepID = "step_id"
        case eventType = "event_type"
        case actorID = "actor_id"
        case actingAsActorID = "acting_as_actor_id"
        case correlationID = "correlation_id"
        case payloadJSON = "payload_json"
        case createdAt = "created_at"
    }
}

struct DaemonInspectRunResponse: Decodable, Sendable {
    let task: DaemonInspectRunTask
    let run: DaemonInspectRunRecord
    let steps: [DaemonInspectRunStep]
    let artifacts: [DaemonInspectRunArtifact]
    let auditEntries: [DaemonInspectRunAuditEntry]
    let route: DaemonWorkflowRouteMetadata?

    enum CodingKeys: String, CodingKey {
        case task
        case run
        case steps
        case artifacts
        case auditEntries = "audit_entries"
        case route
    }
}

struct DaemonInspectLogStreamRequest: Encodable {
    let workspaceID: String
    let runID: String?
    let eventType: String?
    let cursorCreatedAt: String?
    let cursorID: String?
    let limit: Int
    let timeoutMS: Int64
    let pollIntervalMS: Int64

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case runID = "run_id"
        case eventType = "event_type"
        case cursorCreatedAt = "cursor_created_at"
        case cursorID = "cursor_id"
        case limit
        case timeoutMS = "timeout_ms"
        case pollIntervalMS = "poll_interval_ms"
    }
}

struct DaemonWorkflowRouteMetadata: Decodable, Sendable {
    let available: Bool
    let taskClass: String
    let provider: String?
    let modelKey: String?
    let taskClassSource: String
    let routeSource: String
    let notes: String?

    enum CodingKeys: String, CodingKey {
        case available
        case taskClass = "task_class"
        case provider
        case modelKey = "model_key"
        case taskClassSource = "task_class_source"
        case routeSource = "route_source"
        case notes
    }

    init(
        available: Bool = false,
        taskClass: String = "",
        provider: String? = nil,
        modelKey: String? = nil,
        taskClassSource: String = "unknown",
        routeSource: String = "unknown",
        notes: String? = nil
    ) {
        self.available = available
        self.taskClass = taskClass
        self.provider = provider
        self.modelKey = modelKey
        self.taskClassSource = taskClassSource
        self.routeSource = routeSource
        self.notes = notes
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        available = try container.decodeIfPresent(Bool.self, forKey: .available) ?? false
        taskClass = try container.decodeIfPresent(String.self, forKey: .taskClass) ?? ""
        provider = try container.decodeIfPresent(String.self, forKey: .provider)
        modelKey = try container.decodeIfPresent(String.self, forKey: .modelKey)
        taskClassSource = try container.decodeIfPresent(String.self, forKey: .taskClassSource) ?? "unknown"
        routeSource = try container.decodeIfPresent(String.self, forKey: .routeSource) ?? "unknown"
        notes = try container.decodeIfPresent(String.self, forKey: .notes)
    }
}

struct DaemonInspectLogRecord: Decodable, Sendable {
    let logID: String
    let workspaceID: String
    let runID: String?
    let stepID: String?
    let eventType: String
    let status: String
    let inputSummary: String
    let outputSummary: String
    let correlationID: String?
    let actorID: String?
    let actingAsActorID: String?
    let createdAt: String
    let metadata: [String: DaemonJSONValue]?
    let route: DaemonWorkflowRouteMetadata

    enum CodingKeys: String, CodingKey {
        case logID = "log_id"
        case workspaceID = "workspace_id"
        case runID = "run_id"
        case stepID = "step_id"
        case eventType = "event_type"
        case status
        case inputSummary = "input_summary"
        case outputSummary = "output_summary"
        case correlationID = "correlation_id"
        case actorID = "actor_id"
        case actingAsActorID = "acting_as_actor_id"
        case createdAt = "created_at"
        case metadata
        case route
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        logID = try container.decodeIfPresent(String.self, forKey: .logID) ?? UUID().uuidString.lowercased()
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        runID = try container.decodeIfPresent(String.self, forKey: .runID)
        stepID = try container.decodeIfPresent(String.self, forKey: .stepID)
        eventType = try container.decodeIfPresent(String.self, forKey: .eventType) ?? "unknown"
        status = try container.decodeIfPresent(String.self, forKey: .status) ?? "running"
        inputSummary = try container.decodeIfPresent(String.self, forKey: .inputSummary) ?? ""
        outputSummary = try container.decodeIfPresent(String.self, forKey: .outputSummary) ?? ""
        correlationID = try container.decodeIfPresent(String.self, forKey: .correlationID)
        actorID = try container.decodeIfPresent(String.self, forKey: .actorID)
        actingAsActorID = try container.decodeIfPresent(String.self, forKey: .actingAsActorID)
        createdAt = try container.decodeIfPresent(String.self, forKey: .createdAt) ?? ""
        metadata = try container.decodeIfPresent([String: DaemonJSONValue].self, forKey: .metadata)
        route = try container.decodeIfPresent(DaemonWorkflowRouteMetadata.self, forKey: .route) ?? DaemonWorkflowRouteMetadata()
    }
}

struct DaemonInspectLogQueryResponse: Decodable, Sendable {
    let workspaceID: String
    let logs: [DaemonInspectLogRecord]
    let nextCursorCreatedAt: String?
    let nextCursorID: String?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case logs
        case nextCursorCreatedAt = "next_cursor_created_at"
        case nextCursorID = "next_cursor_id"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        logs = try container.decodeIfPresent([DaemonInspectLogRecord].self, forKey: .logs) ?? []
        nextCursorCreatedAt = try container.decodeIfPresent(String.self, forKey: .nextCursorCreatedAt)
        nextCursorID = try container.decodeIfPresent(String.self, forKey: .nextCursorID)
    }
}

struct DaemonInspectLogStreamResponse: Decodable, Sendable {
    let workspaceID: String
    let logs: [DaemonInspectLogRecord]
    let cursorCreatedAt: String?
    let cursorID: String?
    let timedOut: Bool

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case logs
        case cursorCreatedAt = "cursor_created_at"
        case cursorID = "cursor_id"
        case timedOut = "timed_out"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        logs = try container.decodeIfPresent([DaemonInspectLogRecord].self, forKey: .logs) ?? []
        cursorCreatedAt = try container.decodeIfPresent(String.self, forKey: .cursorCreatedAt)
        cursorID = try container.decodeIfPresent(String.self, forKey: .cursorID)
        timedOut = try container.decodeIfPresent(Bool.self, forKey: .timedOut) ?? false
    }
}
