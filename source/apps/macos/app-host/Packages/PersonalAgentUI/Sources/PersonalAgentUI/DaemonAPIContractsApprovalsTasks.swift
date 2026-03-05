import Foundation

struct DaemonApprovalInboxRequest: Encodable {
    let workspaceID: String
    let includeFinal: Bool
    let limit: Int
    let state: String?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case includeFinal = "include_final"
        case limit
        case state
    }
}

struct DaemonApprovalDecisionRequest: Encodable {
    let decisionPhrase: String
    let decisionByActorID: String
    let rationale: String?

    enum CodingKeys: String, CodingKey {
        case decisionPhrase = "decision_phrase"
        case decisionByActorID = "decision_by_actor_id"
        case rationale
    }
}

struct DaemonApprovalDecisionResponse: Decodable, Sendable {
    let approvalID: String
    let decision: String
    let accepted: Bool
    let correlationID: String?

    enum CodingKeys: String, CodingKey {
        case approvalID = "approval_id"
        case decision
        case accepted
        case correlationID = "correlation_id"
    }
}

struct DaemonApprovalInboxRecord: Decodable, Sendable {
    let approvalRequestID: String
    let workspaceID: String
    let state: String
    let decision: String?
    let requestedPhrase: String?
    let riskLevel: String
    let riskRationale: String
    let requestedAt: String
    let decidedAt: String?
    let decisionByActorID: String?
    let decisionRationale: String?
    let taskID: String?
    let taskTitle: String?
    let taskState: String?
    let runID: String?
    let runState: String?
    let stepID: String?
    let stepName: String?
    let stepCapabilityKey: String?
    let requestedByActorID: String?
    let subjectPrincipalActorID: String?
    let actingAsActorID: String?
    let route: DaemonWorkflowRouteMetadata?

    enum CodingKeys: String, CodingKey {
        case approvalRequestID = "approval_request_id"
        case workspaceID = "workspace_id"
        case state
        case decision
        case requestedPhrase = "requested_phrase"
        case riskLevel = "risk_level"
        case riskRationale = "risk_rationale"
        case requestedAt = "requested_at"
        case decidedAt = "decided_at"
        case decisionByActorID = "decision_by_actor_id"
        case decisionRationale = "decision_rationale"
        case taskID = "task_id"
        case taskTitle = "task_title"
        case taskState = "task_state"
        case runID = "run_id"
        case runState = "run_state"
        case stepID = "step_id"
        case stepName = "step_name"
        case stepCapabilityKey = "step_capability_key"
        case requestedByActorID = "requested_by_actor_id"
        case subjectPrincipalActorID = "subject_principal_actor_id"
        case actingAsActorID = "acting_as_actor_id"
        case route
    }
}

struct DaemonApprovalInboxResponse: Decodable, Sendable {
    let workspaceID: String
    let approvals: [DaemonApprovalInboxRecord]

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case approvals
    }
}

struct DaemonTaskRunListRequest: Encodable {
    let workspaceID: String
    let state: String?
    let limit: Int

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case state
        case limit
    }
}

struct DaemonTaskSubmitRequest: Encodable {
    let workspaceID: String
    let requestedByActorID: String
    let subjectPrincipalActorID: String
    let title: String
    let description: String?
    let taskClass: String?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case requestedByActorID = "requested_by_actor_id"
        case subjectPrincipalActorID = "subject_principal_actor_id"
        case title
        case description
        case taskClass = "task_class"
    }
}

struct DaemonTaskSubmitResponse: Decodable, Sendable {
    let taskID: String
    let runID: String
    let state: String
    let correlationID: String?

    enum CodingKeys: String, CodingKey {
        case taskID = "task_id"
        case runID = "run_id"
        case state
        case correlationID = "correlation_id"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        taskID = try container.decodeIfPresent(String.self, forKey: .taskID) ?? ""
        runID = try container.decodeIfPresent(String.self, forKey: .runID) ?? ""
        state = try container.decodeIfPresent(String.self, forKey: .state) ?? "unknown"
        correlationID = try container.decodeIfPresent(String.self, forKey: .correlationID)
    }
}

struct DaemonTaskRunActionAvailability: Decodable, Sendable {
    let canCancel: Bool
    let canRetry: Bool
    let canRequeue: Bool

    enum CodingKeys: String, CodingKey {
        case canCancel = "can_cancel"
        case canRetry = "can_retry"
        case canRequeue = "can_requeue"
    }

    init(canCancel: Bool = false, canRetry: Bool = false, canRequeue: Bool = false) {
        self.canCancel = canCancel
        self.canRetry = canRetry
        self.canRequeue = canRequeue
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        canCancel = try container.decodeIfPresent(Bool.self, forKey: .canCancel) ?? false
        canRetry = try container.decodeIfPresent(Bool.self, forKey: .canRetry) ?? false
        canRequeue = try container.decodeIfPresent(Bool.self, forKey: .canRequeue) ?? false
    }
}

struct DaemonTaskRunControlRequest: Encodable {
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

struct DaemonTaskCancelResponse: Decodable, Sendable {
    let workspaceID: String
    let taskID: String
    let runID: String
    let previousTaskState: String
    let previousRunState: String
    let taskState: String
    let runState: String
    let cancelled: Bool
    let alreadyTerminal: Bool
    let reason: String?
    let correlationID: String?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case taskID = "task_id"
        case runID = "run_id"
        case previousTaskState = "previous_task_state"
        case previousRunState = "previous_run_state"
        case taskState = "task_state"
        case runState = "run_state"
        case cancelled
        case alreadyTerminal = "already_terminal"
        case reason
        case correlationID = "correlation_id"
    }
}

struct DaemonTaskRetryResponse: Decodable, Sendable {
    let workspaceID: String
    let taskID: String
    let previousRunID: String
    let runID: String
    let previousTaskState: String
    let previousRunState: String
    let taskState: String
    let runState: String
    let retried: Bool
    let reason: String?
    let actions: DaemonTaskRunActionAvailability
    let correlationID: String?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case taskID = "task_id"
        case previousRunID = "previous_run_id"
        case runID = "run_id"
        case previousTaskState = "previous_task_state"
        case previousRunState = "previous_run_state"
        case taskState = "task_state"
        case runState = "run_state"
        case retried
        case reason
        case actions
        case correlationID = "correlation_id"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        taskID = try container.decodeIfPresent(String.self, forKey: .taskID) ?? ""
        previousRunID = try container.decodeIfPresent(String.self, forKey: .previousRunID) ?? ""
        runID = try container.decodeIfPresent(String.self, forKey: .runID) ?? ""
        previousTaskState = try container.decodeIfPresent(String.self, forKey: .previousTaskState) ?? ""
        previousRunState = try container.decodeIfPresent(String.self, forKey: .previousRunState) ?? ""
        taskState = try container.decodeIfPresent(String.self, forKey: .taskState) ?? "unknown"
        runState = try container.decodeIfPresent(String.self, forKey: .runState) ?? "unknown"
        retried = try container.decodeIfPresent(Bool.self, forKey: .retried) ?? false
        reason = try container.decodeIfPresent(String.self, forKey: .reason)
        actions = try container.decodeIfPresent(DaemonTaskRunActionAvailability.self, forKey: .actions)
            ?? DaemonTaskRunActionAvailability()
        correlationID = try container.decodeIfPresent(String.self, forKey: .correlationID)
    }
}

struct DaemonTaskRequeueResponse: Decodable, Sendable {
    let workspaceID: String
    let taskID: String
    let previousRunID: String
    let runID: String
    let previousTaskState: String
    let previousRunState: String
    let taskState: String
    let runState: String
    let requeued: Bool
    let reason: String?
    let actions: DaemonTaskRunActionAvailability
    let correlationID: String?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case taskID = "task_id"
        case previousRunID = "previous_run_id"
        case runID = "run_id"
        case previousTaskState = "previous_task_state"
        case previousRunState = "previous_run_state"
        case taskState = "task_state"
        case runState = "run_state"
        case requeued
        case reason
        case actions
        case correlationID = "correlation_id"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        taskID = try container.decodeIfPresent(String.self, forKey: .taskID) ?? ""
        previousRunID = try container.decodeIfPresent(String.self, forKey: .previousRunID) ?? ""
        runID = try container.decodeIfPresent(String.self, forKey: .runID) ?? ""
        previousTaskState = try container.decodeIfPresent(String.self, forKey: .previousTaskState) ?? ""
        previousRunState = try container.decodeIfPresent(String.self, forKey: .previousRunState) ?? ""
        taskState = try container.decodeIfPresent(String.self, forKey: .taskState) ?? "unknown"
        runState = try container.decodeIfPresent(String.self, forKey: .runState) ?? "unknown"
        requeued = try container.decodeIfPresent(Bool.self, forKey: .requeued) ?? false
        reason = try container.decodeIfPresent(String.self, forKey: .reason)
        actions = try container.decodeIfPresent(DaemonTaskRunActionAvailability.self, forKey: .actions)
            ?? DaemonTaskRunActionAvailability()
        correlationID = try container.decodeIfPresent(String.self, forKey: .correlationID)
    }
}

struct DaemonTaskRunListRecord: Decodable, Sendable {
    let taskID: String
    let runID: String?
    let workspaceID: String
    let title: String
    let taskState: String
    let runState: String?
    let priority: Int
    let requestedByActorID: String
    let subjectPrincipalActorID: String
    let actingAsActorID: String?
    let lastError: String?
    let taskCreatedAt: String
    let taskUpdatedAt: String
    let runCreatedAt: String?
    let runUpdatedAt: String?
    let startedAt: String?
    let finishedAt: String?
    let actions: DaemonTaskRunActionAvailability?
    let route: DaemonWorkflowRouteMetadata?

    enum CodingKeys: String, CodingKey {
        case taskID = "task_id"
        case runID = "run_id"
        case workspaceID = "workspace_id"
        case title
        case taskState = "task_state"
        case runState = "run_state"
        case priority
        case requestedByActorID = "requested_by_actor_id"
        case subjectPrincipalActorID = "subject_principal_actor_id"
        case actingAsActorID = "acting_as_actor_id"
        case lastError = "last_error"
        case taskCreatedAt = "task_created_at"
        case taskUpdatedAt = "task_updated_at"
        case runCreatedAt = "run_created_at"
        case runUpdatedAt = "run_updated_at"
        case startedAt = "started_at"
        case finishedAt = "finished_at"
        case actions
        case route
    }
}

struct DaemonTaskRunListResponse: Decodable, Sendable {
    let workspaceID: String
    let items: [DaemonTaskRunListRecord]

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case items
    }
}
