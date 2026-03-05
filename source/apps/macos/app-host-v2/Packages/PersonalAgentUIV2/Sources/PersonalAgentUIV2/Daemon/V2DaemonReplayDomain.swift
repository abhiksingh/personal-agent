import Foundation

// MARK: - Approvals

private struct V2DaemonApprovalDecisionRequest: Encodable {
    let workspaceID: String
    let approvalRequestID: String
    let decision: String
    let decisionByActorID: String?
    let rationale: String?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case approvalRequestID = "approval_request_id"
        case decision
        case decisionByActorID = "decision_by_actor_id"
        case rationale
    }
}

private struct V2DaemonApprovalInboxRequest: Encodable {
    let workspaceID: String
    let state: String?
    let limit: Int

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case state
        case limit
    }
}

public struct V2DaemonApprovalDecisionResponse: Decodable, Sendable, Equatable {
    public let approvalID: String
    public let decision: String
    public let accepted: Bool

    enum CodingKeys: String, CodingKey {
        case approvalID = "approval_id"
        case decision
        case accepted
    }
}

public struct V2DaemonApprovalInboxRecord: Decodable, Sendable, Equatable {
    public let approvalRequestID: String
    public let workspaceID: String
    public let state: String
    public let decision: String?
    public let requestedPhrase: String?
    public let riskLevel: String
    public let riskRationale: String
    public let requestedAt: String
    public let taskID: String?
    public let taskTitle: String?
    public let taskState: String?
    public let runID: String?
    public let runState: String?
    public let route: V2DaemonWorkflowRouteMetadata?

    enum CodingKeys: String, CodingKey {
        case approvalRequestID = "approval_request_id"
        case workspaceID = "workspace_id"
        case state
        case decision
        case requestedPhrase = "requested_phrase"
        case riskLevel = "risk_level"
        case riskRationale = "risk_rationale"
        case requestedAt = "requested_at"
        case taskID = "task_id"
        case taskTitle = "task_title"
        case taskState = "task_state"
        case runID = "run_id"
        case runState = "run_state"
        case route
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        approvalRequestID = try container.decodeIfPresent(String.self, forKey: .approvalRequestID) ?? ""
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        state = try container.decodeIfPresent(String.self, forKey: .state) ?? "pending"
        decision = try container.decodeIfPresent(String.self, forKey: .decision)
        requestedPhrase = try container.decodeIfPresent(String.self, forKey: .requestedPhrase)
        riskLevel = try container.decodeIfPresent(String.self, forKey: .riskLevel) ?? "unknown"
        riskRationale = try container.decodeIfPresent(String.self, forKey: .riskRationale) ?? ""
        requestedAt = try container.decodeIfPresent(String.self, forKey: .requestedAt) ?? ""
        taskID = try container.decodeIfPresent(String.self, forKey: .taskID)
        taskTitle = try container.decodeIfPresent(String.self, forKey: .taskTitle)
        taskState = try container.decodeIfPresent(String.self, forKey: .taskState)
        runID = try container.decodeIfPresent(String.self, forKey: .runID)
        runState = try container.decodeIfPresent(String.self, forKey: .runState)
        route = try container.decodeIfPresent(V2DaemonWorkflowRouteMetadata.self, forKey: .route)
    }
}

public struct V2DaemonApprovalInboxResponse: Decodable, Sendable, Equatable {
    public let workspaceID: String
    public let approvals: [V2DaemonApprovalInboxRecord]

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case approvals
    }
}

public struct V2DaemonApprovalsAPI {
    private let client: V2DaemonAPIClient

    init(client: V2DaemonAPIClient) {
        self.client = client
    }

    public func approvalInbox(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        state: String? = nil,
        limit: Int = 120
    ) async throws -> V2DaemonApprovalInboxResponse {
        try await client.request(
            baseURL: baseURL,
            path: "/v1/approvals/inbox",
            method: "POST",
            authToken: authToken,
            body: V2DaemonApprovalInboxRequest(workspaceID: workspaceID, state: state, limit: limit)
        )
    }

    public func approvalDecision(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        approvalRequestID: String,
        decision: String,
        decisionByActorID: String? = nil,
        rationale: String? = nil
    ) async throws -> V2DaemonApprovalDecisionResponse {
        try await client.request(
            baseURL: baseURL,
            path: "/v1/approvals/decision",
            method: "POST",
            authToken: authToken,
            body: V2DaemonApprovalDecisionRequest(
                workspaceID: workspaceID,
                approvalRequestID: approvalRequestID,
                decision: decision,
                decisionByActorID: decisionByActorID,
                rationale: rationale
            )
        )
    }
}

// MARK: - Tasks

private struct V2DaemonTaskRunListRequest: Encodable {
    let workspaceID: String
    let state: String?
    let limit: Int

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case state
        case limit
    }
}

private struct V2DaemonTaskRunControlRequest: Encodable {
    let workspaceID: String
    let taskID: String?
    let runID: String?
    let reason: String?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case taskID = "task_id"
        case runID = "run_id"
        case reason
    }
}

public struct V2DaemonTaskRunActionAvailability: Decodable, Sendable, Equatable {
    public let canCancel: Bool
    public let canRetry: Bool
    public let canRequeue: Bool

    enum CodingKeys: String, CodingKey {
        case canCancel = "can_cancel"
        case canRetry = "can_retry"
        case canRequeue = "can_requeue"
    }

    public init(canCancel: Bool = false, canRetry: Bool = false, canRequeue: Bool = false) {
        self.canCancel = canCancel
        self.canRetry = canRetry
        self.canRequeue = canRequeue
    }
}

public struct V2DaemonTaskRunListRecord: Decodable, Sendable, Equatable {
    public let taskID: String
    public let runID: String?
    public let workspaceID: String
    public let title: String
    public let taskState: String
    public let runState: String?
    public let lastError: String?
    public let taskCreatedAt: String
    public let taskUpdatedAt: String
    public let runCreatedAt: String?
    public let runUpdatedAt: String?
    public let actions: V2DaemonTaskRunActionAvailability?
    public let route: V2DaemonWorkflowRouteMetadata?

    enum CodingKeys: String, CodingKey {
        case taskID = "task_id"
        case runID = "run_id"
        case workspaceID = "workspace_id"
        case title
        case taskState = "task_state"
        case runState = "run_state"
        case lastError = "last_error"
        case taskCreatedAt = "task_created_at"
        case taskUpdatedAt = "task_updated_at"
        case runCreatedAt = "run_created_at"
        case runUpdatedAt = "run_updated_at"
        case actions
        case route
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        taskID = try container.decodeIfPresent(String.self, forKey: .taskID) ?? ""
        runID = try container.decodeIfPresent(String.self, forKey: .runID)
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        title = try container.decodeIfPresent(String.self, forKey: .title) ?? ""
        taskState = try container.decodeIfPresent(String.self, forKey: .taskState) ?? "unknown"
        runState = try container.decodeIfPresent(String.self, forKey: .runState)
        lastError = try container.decodeIfPresent(String.self, forKey: .lastError)
        taskCreatedAt = try container.decodeIfPresent(String.self, forKey: .taskCreatedAt) ?? ""
        taskUpdatedAt = try container.decodeIfPresent(String.self, forKey: .taskUpdatedAt) ?? ""
        runCreatedAt = try container.decodeIfPresent(String.self, forKey: .runCreatedAt)
        runUpdatedAt = try container.decodeIfPresent(String.self, forKey: .runUpdatedAt)
        actions = try container.decodeIfPresent(V2DaemonTaskRunActionAvailability.self, forKey: .actions)
        route = try container.decodeIfPresent(V2DaemonWorkflowRouteMetadata.self, forKey: .route)
    }
}

public struct V2DaemonTaskRunListResponse: Decodable, Sendable, Equatable {
    public let workspaceID: String
    public let items: [V2DaemonTaskRunListRecord]

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case items
    }
}

public struct V2DaemonTaskRetryResponse: Decodable, Sendable, Equatable {
    public let workspaceID: String
    public let taskID: String
    public let runID: String
    public let taskState: String
    public let runState: String

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case taskID = "task_id"
        case runID = "run_id"
        case taskState = "task_state"
        case runState = "run_state"
    }
}

public struct V2DaemonTaskCancelResponse: Decodable, Sendable, Equatable {
    public let workspaceID: String
    public let taskID: String
    public let runID: String
    public let taskState: String
    public let runState: String

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case taskID = "task_id"
        case runID = "run_id"
        case taskState = "task_state"
        case runState = "run_state"
    }
}

public struct V2DaemonTaskRequeueResponse: Decodable, Sendable, Equatable {
    public let workspaceID: String
    public let taskID: String
    public let runID: String
    public let taskState: String
    public let runState: String

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case taskID = "task_id"
        case runID = "run_id"
        case taskState = "task_state"
        case runState = "run_state"
    }
}

public struct V2DaemonTasksAPI {
    private let client: V2DaemonAPIClient

    init(client: V2DaemonAPIClient) {
        self.client = client
    }

    public func taskRunList(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        state: String? = nil,
        limit: Int = 120
    ) async throws -> V2DaemonTaskRunListResponse {
        try await client.request(
            baseURL: baseURL,
            path: "/v1/tasks/runs/list",
            method: "POST",
            authToken: authToken,
            body: V2DaemonTaskRunListRequest(workspaceID: workspaceID, state: state, limit: limit)
        )
    }

    public func taskRetry(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        taskID: String,
        runID: String?,
        reason: String? = nil
    ) async throws -> V2DaemonTaskRetryResponse {
        try await client.request(
            baseURL: baseURL,
            path: "/v1/tasks/retry",
            method: "POST",
            authToken: authToken,
            body: V2DaemonTaskRunControlRequest(workspaceID: workspaceID, taskID: taskID, runID: runID, reason: reason)
        )
    }

    public func taskCancel(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        taskID: String,
        runID: String?,
        reason: String? = nil
    ) async throws -> V2DaemonTaskCancelResponse {
        try await client.request(
            baseURL: baseURL,
            path: "/v1/tasks/cancel",
            method: "POST",
            authToken: authToken,
            body: V2DaemonTaskRunControlRequest(workspaceID: workspaceID, taskID: taskID, runID: runID, reason: reason)
        )
    }

    public func taskRequeue(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        taskID: String,
        runID: String?,
        reason: String? = nil
    ) async throws -> V2DaemonTaskRequeueResponse {
        try await client.request(
            baseURL: baseURL,
            path: "/v1/tasks/requeue",
            method: "POST",
            authToken: authToken,
            body: V2DaemonTaskRunControlRequest(workspaceID: workspaceID, taskID: taskID, runID: runID, reason: reason)
        )
    }
}

// MARK: - Inspect

private struct V2DaemonInspectLogQueryRequest: Encodable {
    let workspaceID: String
    let runID: String?
    let beforeCreatedAt: String?
    let beforeID: String?
    let limit: Int

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case runID = "run_id"
        case beforeCreatedAt = "before_created_at"
        case beforeID = "before_id"
        case limit
    }
}

private struct V2DaemonInspectRunRequest: Encodable {
    let runID: String

    enum CodingKeys: String, CodingKey {
        case runID = "run_id"
    }
}

public struct V2DaemonInspectLogRecord: Decodable, Sendable, Equatable {
    public let logID: String
    public let workspaceID: String
    public let runID: String?
    public let stepID: String?
    public let eventType: String
    public let status: String
    public let inputSummary: String
    public let outputSummary: String
    public let createdAt: String
    public let metadata: [String: V2DaemonJSONValue]?
    public let route: V2DaemonWorkflowRouteMetadata

    enum CodingKeys: String, CodingKey {
        case logID = "log_id"
        case workspaceID = "workspace_id"
        case runID = "run_id"
        case stepID = "step_id"
        case eventType = "event_type"
        case status
        case inputSummary = "input_summary"
        case outputSummary = "output_summary"
        case createdAt = "created_at"
        case metadata
        case route
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        logID = try container.decodeIfPresent(String.self, forKey: .logID) ?? UUID().uuidString.lowercased()
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        runID = try container.decodeIfPresent(String.self, forKey: .runID)
        stepID = try container.decodeIfPresent(String.self, forKey: .stepID)
        eventType = try container.decodeIfPresent(String.self, forKey: .eventType) ?? "unknown"
        status = try container.decodeIfPresent(String.self, forKey: .status) ?? "unknown"
        inputSummary = try container.decodeIfPresent(String.self, forKey: .inputSummary) ?? ""
        outputSummary = try container.decodeIfPresent(String.self, forKey: .outputSummary) ?? ""
        createdAt = try container.decodeIfPresent(String.self, forKey: .createdAt) ?? ""
        metadata = try container.decodeIfPresent([String: V2DaemonJSONValue].self, forKey: .metadata)
        route = try container.decodeIfPresent(V2DaemonWorkflowRouteMetadata.self, forKey: .route) ?? V2DaemonWorkflowRouteMetadata()
    }
}

public struct V2DaemonInspectLogQueryResponse: Decodable, Sendable, Equatable {
    public let workspaceID: String
    public let logs: [V2DaemonInspectLogRecord]
    public let nextCursorCreatedAt: String?
    public let nextCursorID: String?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case logs
        case nextCursorCreatedAt = "next_cursor_created_at"
        case nextCursorID = "next_cursor_id"
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        logs = try container.decodeIfPresent([V2DaemonInspectLogRecord].self, forKey: .logs) ?? []
        nextCursorCreatedAt = try container.decodeIfPresent(String.self, forKey: .nextCursorCreatedAt)
        nextCursorID = try container.decodeIfPresent(String.self, forKey: .nextCursorID)
    }
}

public struct V2DaemonInspectRunStep: Decodable, Sendable, Equatable {
    public let stepID: String
    public let name: String
    public let status: String
    public let capabilityKey: String?
    public let lastError: String?

    enum CodingKeys: String, CodingKey {
        case stepID = "step_id"
        case name
        case status
        case capabilityKey = "capability_key"
        case lastError = "last_error"
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        stepID = try container.decodeIfPresent(String.self, forKey: .stepID) ?? ""
        name = try container.decodeIfPresent(String.self, forKey: .name) ?? ""
        status = try container.decodeIfPresent(String.self, forKey: .status) ?? "unknown"
        capabilityKey = try container.decodeIfPresent(String.self, forKey: .capabilityKey)
        lastError = try container.decodeIfPresent(String.self, forKey: .lastError)
    }
}

public struct V2DaemonInspectRunAuditEntry: Decodable, Sendable, Equatable {
    public let eventType: String
    public let payloadJSON: String?
    public let createdAt: String

    enum CodingKeys: String, CodingKey {
        case eventType = "event_type"
        case payloadJSON = "payload_json"
        case createdAt = "created_at"
    }

    public var payloadValue: V2DaemonJSONValue? {
        guard let payloadJSON, let data = payloadJSON.data(using: .utf8) else {
            return nil
        }
        return try? JSONDecoder().decode(V2DaemonJSONValue.self, from: data)
    }
}

public struct V2DaemonInspectRunRecord: Decodable, Sendable, Equatable {
    public let runID: String
    public let workspaceID: String
    public let taskID: String
    public let state: String
    public let lastError: String?
    public let createdAt: String
    public let updatedAt: String

    enum CodingKeys: String, CodingKey {
        case runID = "run_id"
        case workspaceID = "workspace_id"
        case taskID = "task_id"
        case state
        case lastError = "last_error"
        case createdAt = "created_at"
        case updatedAt = "updated_at"
    }
}

public struct V2DaemonInspectRunTask: Decodable, Sendable, Equatable {
    public let taskID: String
    public let title: String
    public let state: String

    enum CodingKeys: String, CodingKey {
        case taskID = "task_id"
        case title
        case state
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        taskID = try container.decodeIfPresent(String.self, forKey: .taskID) ?? ""
        title = try container.decodeIfPresent(String.self, forKey: .title) ?? ""
        state = try container.decodeIfPresent(String.self, forKey: .state) ?? "unknown"
    }
}

public struct V2DaemonInspectRunResponse: Decodable, Sendable, Equatable {
    public let task: V2DaemonInspectRunTask
    public let run: V2DaemonInspectRunRecord
    public let steps: [V2DaemonInspectRunStep]
    public let auditEntries: [V2DaemonInspectRunAuditEntry]
    public let route: V2DaemonWorkflowRouteMetadata?

    enum CodingKeys: String, CodingKey {
        case task
        case run
        case steps
        case auditEntries = "audit_entries"
        case route
    }
}

public struct V2DaemonInspectAPI {
    private let client: V2DaemonAPIClient

    init(client: V2DaemonAPIClient) {
        self.client = client
    }

    public func inspectLogsQuery(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        runID: String? = nil,
        beforeCreatedAt: String? = nil,
        beforeID: String? = nil,
        limit: Int = 80
    ) async throws -> V2DaemonInspectLogQueryResponse {
        try await client.request(
            baseURL: baseURL,
            path: "/v1/inspect/logs/query",
            method: "POST",
            authToken: authToken,
            body: V2DaemonInspectLogQueryRequest(
                workspaceID: workspaceID,
                runID: runID,
                beforeCreatedAt: beforeCreatedAt,
                beforeID: beforeID,
                limit: limit
            )
        )
    }

    public func inspectRun(
        baseURL: URL,
        authToken: String,
        runID: String
    ) async throws -> V2DaemonInspectRunResponse {
        try await client.request(
            baseURL: baseURL,
            path: "/v1/inspect/run",
            method: "POST",
            authToken: authToken,
            body: V2DaemonInspectRunRequest(runID: runID)
        )
    }
}

// MARK: - Chat

private struct V2DaemonChatTurnRequestItem: Encodable {
    let type: String
    let role: String?
    let status: String?
    let content: String?
}

private struct V2DaemonChatTurnRequest: Encodable {
    let workspaceID: String
    let taskClass: String
    let requestedByActorID: String?
    let subjectActorID: String?
    let actingAsActorID: String?
    let channel: V2DaemonChatTurnChannelContext?
    let items: [V2DaemonChatTurnRequestItem]

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case taskClass = "task_class"
        case requestedByActorID = "requested_by_actor_id"
        case subjectActorID = "subject_actor_id"
        case actingAsActorID = "acting_as_actor_id"
        case channel
        case items
    }
}

private struct V2DaemonChatTurnHistoryRequest: Encodable {
    let workspaceID: String
    let channelID: String?
    let correlationID: String?
    let beforeCreatedAt: String?
    let beforeItemID: String?
    let limit: Int

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case channelID = "channel_id"
        case correlationID = "correlation_id"
        case beforeCreatedAt = "before_created_at"
        case beforeItemID = "before_item_id"
        case limit
    }
}

public struct V2DaemonChatTurnChannelContext: Codable, Sendable, Equatable {
    public let channelID: String?
    public let connectorID: String?
    public let threadID: String?

    enum CodingKeys: String, CodingKey {
        case channelID = "channel_id"
        case connectorID = "connector_id"
        case threadID = "thread_id"
    }

    public init(channelID: String? = nil, connectorID: String? = nil, threadID: String? = nil) {
        self.channelID = channelID
        self.connectorID = connectorID
        self.threadID = threadID
    }
}

public struct V2DaemonChatTurnItemMetadata: Decodable, Sendable, Equatable {
    public let taskClass: String?
    public let provider: String?
    public let modelKey: String?
    public let additional: [String: V2DaemonJSONValue]

    enum CodingKeys: String, CodingKey {
        case taskClass = "task_class"
        case provider
        case modelKey = "model_key"
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        taskClass = try container.decodeLossyString(forKey: .taskClass)
        provider = try container.decodeLossyString(forKey: .provider)
        modelKey = try container.decodeLossyString(forKey: .modelKey)

        var extras = v2DecodeDaemonJSONObject(from: decoder)
        for key in ["task_class", "provider", "model_key"] {
            extras.removeValue(forKey: key)
        }
        additional = extras
    }
}

public struct V2DaemonChatTurnItem: Decodable, Sendable, Equatable {
    public let itemID: String?
    public let type: String
    public let role: String?
    public let status: String?
    public let content: String?
    public let toolName: String?
    public let toolCallID: String?
    public let arguments: [String: V2DaemonJSONValue]?
    public let output: [String: V2DaemonJSONValue]?
    public let errorCode: String?
    public let errorMessage: String?
    public let approvalRequestID: String?
    public let metadata: V2DaemonChatTurnItemMetadata?

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

    public init(
        itemID: String? = nil,
        type: String,
        role: String? = nil,
        status: String? = nil,
        content: String? = nil,
        toolName: String? = nil,
        toolCallID: String? = nil,
        arguments: [String: V2DaemonJSONValue]? = nil,
        output: [String: V2DaemonJSONValue]? = nil,
        errorCode: String? = nil,
        errorMessage: String? = nil,
        approvalRequestID: String? = nil,
        metadata: V2DaemonChatTurnItemMetadata? = nil
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

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        itemID = try container.decodeIfPresent(String.self, forKey: .itemID)
        type = try container.decodeIfPresent(String.self, forKey: .type) ?? "assistant_message"
        role = try container.decodeIfPresent(String.self, forKey: .role)
        status = try container.decodeIfPresent(String.self, forKey: .status)
        content = try container.decodeIfPresent(String.self, forKey: .content)
        toolName = try container.decodeIfPresent(String.self, forKey: .toolName)
        toolCallID = try container.decodeIfPresent(String.self, forKey: .toolCallID)
        arguments = try container.decodeIfPresent([String: V2DaemonJSONValue].self, forKey: .arguments)
        output = try container.decodeIfPresent([String: V2DaemonJSONValue].self, forKey: .output)
        errorCode = try container.decodeIfPresent(String.self, forKey: .errorCode)
        errorMessage = try container.decodeIfPresent(String.self, forKey: .errorMessage)
        approvalRequestID = try container.decodeIfPresent(String.self, forKey: .approvalRequestID)
        metadata = try container.decodeIfPresent(V2DaemonChatTurnItemMetadata.self, forKey: .metadata)
    }
}

public struct V2DaemonChatTurnTaskRunCorrelation: Decodable, Sendable, Equatable {
    public let available: Bool
    public let source: String
    public let taskID: String?
    public let runID: String?
    public let taskState: String?
    public let runState: String?

    enum CodingKeys: String, CodingKey {
        case available
        case source
        case taskID = "task_id"
        case runID = "run_id"
        case taskState = "task_state"
        case runState = "run_state"
    }

    public init(
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
}

public struct V2DaemonChatTurnResponse: Decodable, Sendable, Equatable {
    public let workspaceID: String
    public let taskClass: String
    public let provider: String
    public let modelKey: String
    public let correlationID: String
    public let channel: V2DaemonChatTurnChannelContext?
    public let items: [V2DaemonChatTurnItem]
    public let taskRunCorrelation: V2DaemonChatTurnTaskRunCorrelation

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case taskClass = "task_class"
        case provider
        case modelKey = "model_key"
        case correlationID = "correlation_id"
        case channel
        case items
        case taskRunCorrelation = "task_run_correlation"
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        taskClass = try container.decodeIfPresent(String.self, forKey: .taskClass) ?? "chat"
        provider = try container.decodeIfPresent(String.self, forKey: .provider) ?? "unknown"
        modelKey = try container.decodeIfPresent(String.self, forKey: .modelKey) ?? "unknown"
        correlationID = try container.decodeIfPresent(String.self, forKey: .correlationID) ?? ""
        channel = try container.decodeIfPresent(V2DaemonChatTurnChannelContext.self, forKey: .channel)
        items = try container.decodeIfPresent([V2DaemonChatTurnItem].self, forKey: .items) ?? []
        taskRunCorrelation = try container.decodeIfPresent(V2DaemonChatTurnTaskRunCorrelation.self, forKey: .taskRunCorrelation)
            ?? V2DaemonChatTurnTaskRunCorrelation()
    }
}

public struct V2DaemonChatTurnHistoryRecord: Decodable, Sendable, Equatable {
    public let recordID: String
    public let turnID: String
    public let correlationID: String
    public let channelID: String
    public let itemIndex: Int
    public let item: V2DaemonChatTurnItem
    public let taskRunReference: V2DaemonChatTurnTaskRunCorrelation
    public let createdAt: String

    enum CodingKeys: String, CodingKey {
        case recordID = "record_id"
        case turnID = "turn_id"
        case correlationID = "correlation_id"
        case channelID = "channel_id"
        case itemIndex = "item_index"
        case item
        case taskRunReference = "task_run_reference"
        case createdAt = "created_at"
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        recordID = try container.decodeIfPresent(String.self, forKey: .recordID) ?? ""
        turnID = try container.decodeIfPresent(String.self, forKey: .turnID) ?? ""
        correlationID = try container.decodeIfPresent(String.self, forKey: .correlationID) ?? ""
        channelID = try container.decodeIfPresent(String.self, forKey: .channelID) ?? "app"
        itemIndex = try container.decodeIfPresent(Int.self, forKey: .itemIndex) ?? 0
        item = try container.decodeIfPresent(V2DaemonChatTurnItem.self, forKey: .item)
            ?? V2DaemonChatTurnItem(type: "assistant_message", content: "")
        taskRunReference = try container.decodeIfPresent(V2DaemonChatTurnTaskRunCorrelation.self, forKey: .taskRunReference)
            ?? V2DaemonChatTurnTaskRunCorrelation()
        createdAt = try container.decodeIfPresent(String.self, forKey: .createdAt) ?? ""
    }
}

public struct V2DaemonChatTurnHistoryResponse: Decodable, Sendable, Equatable {
    public let workspaceID: String
    public let items: [V2DaemonChatTurnHistoryRecord]
    public let hasMore: Bool
    public let nextCursorCreatedAt: String?
    public let nextCursorItemID: String?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case items
        case hasMore = "has_more"
        case nextCursorCreatedAt = "next_cursor_created_at"
        case nextCursorItemID = "next_cursor_item_id"
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        items = try container.decodeIfPresent([V2DaemonChatTurnHistoryRecord].self, forKey: .items) ?? []
        hasMore = try container.decodeIfPresent(Bool.self, forKey: .hasMore) ?? false
        nextCursorCreatedAt = try container.decodeIfPresent(String.self, forKey: .nextCursorCreatedAt)
        nextCursorItemID = try container.decodeIfPresent(String.self, forKey: .nextCursorItemID)
    }
}

public struct V2DaemonChatAPI {
    private let client: V2DaemonAPIClient

    init(client: V2DaemonAPIClient) {
        self.client = client
    }

    public func chatTurn(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        messages: [(role: String, content: String)],
        requestedByActorID: String? = nil,
        subjectActorID: String? = nil,
        actingAsActorID: String? = nil,
        correlationID: String? = nil
    ) async throws -> V2DaemonChatTurnResponse {
        let requestItems = messages.compactMap { message -> V2DaemonChatTurnRequestItem? in
            let role = message.role.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
            let content = message.content.trimmingCharacters(in: .newlines)
            guard !content.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else {
                return nil
            }
            if role == "assistant" {
                return V2DaemonChatTurnRequestItem(type: "assistant_message", role: "assistant", status: "completed", content: content)
            }
            return V2DaemonChatTurnRequestItem(type: "user_message", role: "user", status: "completed", content: content)
        }

        return try await client.request(
            baseURL: baseURL,
            path: "/v1/chat/turn",
            method: "POST",
            authToken: authToken,
            correlationID: correlationID,
            timeoutInterval: 300,
            body: V2DaemonChatTurnRequest(
                workspaceID: workspaceID,
                taskClass: "chat",
                requestedByActorID: requestedByActorID,
                subjectActorID: subjectActorID,
                actingAsActorID: actingAsActorID,
                channel: V2DaemonChatTurnChannelContext(channelID: "app"),
                items: requestItems
            )
        )
    }

    public func chatTurnHistory(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        channelID: String? = nil,
        correlationID: String? = nil,
        beforeCreatedAt: String? = nil,
        beforeItemID: String? = nil,
        limit: Int = 120
    ) async throws -> V2DaemonChatTurnHistoryResponse {
        try await client.request(
            baseURL: baseURL,
            path: "/v1/chat/history",
            method: "POST",
            authToken: authToken,
            body: V2DaemonChatTurnHistoryRequest(
                workspaceID: workspaceID,
                channelID: channelID,
                correlationID: correlationID,
                beforeCreatedAt: beforeCreatedAt,
                beforeItemID: beforeItemID,
                limit: limit
            )
        )
    }
}
