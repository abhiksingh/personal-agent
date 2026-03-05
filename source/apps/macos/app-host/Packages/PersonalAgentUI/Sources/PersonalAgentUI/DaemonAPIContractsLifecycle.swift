import Foundation

public struct DaemonLifecycleWorkerSummary: Decodable, Sendable, Equatable {
    let total: Int
    let registered: Int
    let starting: Int
    let running: Int
    let restarting: Int
    let stopped: Int
    let failed: Int

    enum CodingKeys: String, CodingKey {
        case total
        case registered
        case starting
        case running
        case restarting
        case stopped
        case failed
    }

    init(
        total: Int = 0,
        registered: Int = 0,
        starting: Int = 0,
        running: Int = 0,
        restarting: Int = 0,
        stopped: Int = 0,
        failed: Int = 0
    ) {
        self.total = total
        self.registered = registered
        self.starting = starting
        self.running = running
        self.restarting = restarting
        self.stopped = stopped
        self.failed = failed
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        total = try container.decodeIfPresent(Int.self, forKey: .total) ?? 0
        registered = try container.decodeIfPresent(Int.self, forKey: .registered) ?? 0
        starting = try container.decodeIfPresent(Int.self, forKey: .starting) ?? 0
        running = try container.decodeIfPresent(Int.self, forKey: .running) ?? 0
        restarting = try container.decodeIfPresent(Int.self, forKey: .restarting) ?? 0
        stopped = try container.decodeIfPresent(Int.self, forKey: .stopped) ?? 0
        failed = try container.decodeIfPresent(Int.self, forKey: .failed) ?? 0
    }
}

struct DaemonLifecycleControls: Decodable, Sendable {
    let start: Bool
    let stop: Bool
    let restart: Bool
    let install: Bool
    let uninstall: Bool
    let repair: Bool

    enum CodingKeys: String, CodingKey {
        case start
        case stop
        case restart
        case install
        case uninstall
        case repair
    }

    init(
        start: Bool = false,
        stop: Bool = false,
        restart: Bool = false,
        install: Bool = false,
        uninstall: Bool = false,
        repair: Bool = false
    ) {
        self.start = start
        self.stop = stop
        self.restart = restart
        self.install = install
        self.uninstall = uninstall
        self.repair = repair
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        start = try container.decodeIfPresent(Bool.self, forKey: .start) ?? false
        stop = try container.decodeIfPresent(Bool.self, forKey: .stop) ?? false
        restart = try container.decodeIfPresent(Bool.self, forKey: .restart) ?? false
        install = try container.decodeIfPresent(Bool.self, forKey: .install) ?? false
        uninstall = try container.decodeIfPresent(Bool.self, forKey: .uninstall) ?? false
        repair = try container.decodeIfPresent(Bool.self, forKey: .repair) ?? false
    }
}

struct DaemonLifecycleControlOperation: Decodable, Sendable {
    let action: String
    let state: String
    let message: String?
    let error: String?
    let requestedAt: String?
    let completedAt: String?

    enum CodingKeys: String, CodingKey {
        case action
        case state
        case message
        case error
        case requestedAt = "requested_at"
        case completedAt = "completed_at"
    }

    init(
        action: String = "",
        state: String = "idle",
        message: String? = nil,
        error: String? = nil,
        requestedAt: String? = nil,
        completedAt: String? = nil
    ) {
        self.action = action
        self.state = state
        self.message = message
        self.error = error
        self.requestedAt = requestedAt
        self.completedAt = completedAt
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        action = try container.decodeIfPresent(String.self, forKey: .action) ?? ""
        state = try container.decodeIfPresent(String.self, forKey: .state) ?? "idle"
        message = try container.decodeIfPresent(String.self, forKey: .message)
        error = try container.decodeIfPresent(String.self, forKey: .error)
        requestedAt = try container.decodeIfPresent(String.self, forKey: .requestedAt)
        completedAt = try container.decodeIfPresent(String.self, forKey: .completedAt)
    }
}

struct DaemonLifecycleHealthClassification: Decodable, Sendable {
    let overallState: String
    let coreRuntimeState: String
    let pluginRuntimeState: String
    let blocking: Bool
    let coreReason: String?
    let pluginReason: String?

    enum CodingKeys: String, CodingKey {
        case overallState = "overall_state"
        case coreRuntimeState = "core_runtime_state"
        case pluginRuntimeState = "plugin_runtime_state"
        case blocking
        case coreReason = "core_reason"
        case pluginReason = "plugin_reason"
    }

    init(
        overallState: String = "unknown",
        coreRuntimeState: String = "unknown",
        pluginRuntimeState: String = "unknown",
        blocking: Bool = false,
        coreReason: String? = nil,
        pluginReason: String? = nil
    ) {
        self.overallState = overallState
        self.coreRuntimeState = coreRuntimeState
        self.pluginRuntimeState = pluginRuntimeState
        self.blocking = blocking
        self.coreReason = coreReason
        self.pluginReason = pluginReason
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        overallState = try container.decodeIfPresent(String.self, forKey: .overallState) ?? "unknown"
        coreRuntimeState = try container.decodeIfPresent(String.self, forKey: .coreRuntimeState) ?? "unknown"
        pluginRuntimeState = try container.decodeIfPresent(String.self, forKey: .pluginRuntimeState) ?? "unknown"
        blocking = try container.decodeIfPresent(Bool.self, forKey: .blocking) ?? false
        coreReason = try container.decodeIfPresent(String.self, forKey: .coreReason)
        pluginReason = try container.decodeIfPresent(String.self, forKey: .pluginReason)
    }
}

struct DaemonLifecycleControlAuthState: Decodable, Sendable {
    let state: String
    let source: String
    let remediationHints: [String]

    enum CodingKeys: String, CodingKey {
        case state
        case source
        case remediationHints = "remediation_hints"
    }

    init(
        state: String = "unknown",
        source: String = "unknown",
        remediationHints: [String] = []
    ) {
        self.state = state
        self.source = source
        self.remediationHints = remediationHints
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        state = try container.decodeIfPresent(String.self, forKey: .state) ?? "unknown"
        source = try container.decodeIfPresent(String.self, forKey: .source) ?? "unknown"
        remediationHints = try container.decodeIfPresent([String].self, forKey: .remediationHints) ?? []
    }
}

struct DaemonLifecycleStatusResponse: Decodable, Sendable {
    let lifecycleState: String
    let processID: Int
    let startedAt: String
    let lastTransitionAt: String
    let runtimeMode: String?
    let configuredAddress: String?
    let boundAddress: String?
    let setupState: String
    let installState: String
    let needsInstall: Bool
    let needsRepair: Bool
    let repairHint: String?
    let healthClassification: DaemonLifecycleHealthClassification
    let executablePath: String?
    let databasePath: String?
    let databaseReady: Bool
    let databaseError: String?
    let controlAuth: DaemonLifecycleControlAuthState
    let workerSummary: DaemonLifecycleWorkerSummary
    let controls: DaemonLifecycleControls
    let controlOperation: DaemonLifecycleControlOperation

    enum CodingKeys: String, CodingKey {
        case lifecycleState = "lifecycle_state"
        case processID = "process_id"
        case startedAt = "started_at"
        case lastTransitionAt = "last_transition_at"
        case runtimeMode = "runtime_mode"
        case configuredAddress = "configured_address"
        case boundAddress = "bound_address"
        case setupState = "setup_state"
        case installState = "install_state"
        case needsInstall = "needs_install"
        case needsRepair = "needs_repair"
        case repairHint = "repair_hint"
        case healthClassification = "health_classification"
        case executablePath = "executable_path"
        case databasePath = "database_path"
        case databaseReady = "database_ready"
        case databaseError = "database_error"
        case controlAuth = "control_auth"
        case workerSummary = "worker_summary"
        case controls
        case controlOperation = "control_operation"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        lifecycleState = try container.decodeIfPresent(String.self, forKey: .lifecycleState) ?? "unknown"
        processID = try container.decodeIfPresent(Int.self, forKey: .processID) ?? 0
        startedAt = try container.decodeIfPresent(String.self, forKey: .startedAt) ?? ""
        lastTransitionAt = try container.decodeIfPresent(String.self, forKey: .lastTransitionAt) ?? ""
        runtimeMode = try container.decodeIfPresent(String.self, forKey: .runtimeMode)
        configuredAddress = try container.decodeIfPresent(String.self, forKey: .configuredAddress)
        boundAddress = try container.decodeIfPresent(String.self, forKey: .boundAddress)
        setupState = try container.decodeIfPresent(String.self, forKey: .setupState) ?? ""
        installState = try container.decodeIfPresent(String.self, forKey: .installState) ?? ""
        needsInstall = try container.decodeIfPresent(Bool.self, forKey: .needsInstall) ?? false
        needsRepair = try container.decodeIfPresent(Bool.self, forKey: .needsRepair) ?? false
        repairHint = try container.decodeIfPresent(String.self, forKey: .repairHint)
        healthClassification = try container.decodeIfPresent(DaemonLifecycleHealthClassification.self, forKey: .healthClassification)
            ?? DaemonLifecycleHealthClassification()
        executablePath = try container.decodeIfPresent(String.self, forKey: .executablePath)
        databasePath = try container.decodeIfPresent(String.self, forKey: .databasePath)
        databaseReady = try container.decodeIfPresent(Bool.self, forKey: .databaseReady) ?? false
        databaseError = try container.decodeIfPresent(String.self, forKey: .databaseError)
        controlAuth = try container.decodeIfPresent(DaemonLifecycleControlAuthState.self, forKey: .controlAuth)
            ?? DaemonLifecycleControlAuthState()
        workerSummary = try container.decodeIfPresent(DaemonLifecycleWorkerSummary.self, forKey: .workerSummary)
            ?? DaemonLifecycleWorkerSummary()
        controls = try container.decodeIfPresent(DaemonLifecycleControls.self, forKey: .controls)
            ?? DaemonLifecycleControls()
        controlOperation = try container.decodeIfPresent(DaemonLifecycleControlOperation.self, forKey: .controlOperation)
            ?? DaemonLifecycleControlOperation()
    }
}

struct DaemonLifecycleControlRequest: Encodable {
    let action: String
    let reason: String?
}

struct DaemonLifecycleControlResponse: Decodable, Sendable {
    let action: String
    let accepted: Bool
    let idempotent: Bool
    let lifecycleState: String
    let message: String?
    let operationState: String
    let requestedAt: String?
    let completedAt: String?
    let error: String?

    enum CodingKeys: String, CodingKey {
        case action
        case accepted
        case idempotent
        case lifecycleState = "lifecycle_state"
        case message
        case operationState = "operation_state"
        case requestedAt = "requested_at"
        case completedAt = "completed_at"
        case error
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        action = try container.decodeIfPresent(String.self, forKey: .action) ?? ""
        accepted = try container.decodeIfPresent(Bool.self, forKey: .accepted) ?? false
        idempotent = try container.decodeIfPresent(Bool.self, forKey: .idempotent) ?? false
        lifecycleState = try container.decodeIfPresent(String.self, forKey: .lifecycleState) ?? ""
        message = try container.decodeIfPresent(String.self, forKey: .message)
        operationState = try container.decodeIfPresent(String.self, forKey: .operationState)
            ?? (accepted ? "succeeded" : "failed")
        requestedAt = try container.decodeIfPresent(String.self, forKey: .requestedAt)
        completedAt = try container.decodeIfPresent(String.self, forKey: .completedAt)
        error = try container.decodeIfPresent(String.self, forKey: .error)
    }
}

struct DaemonPluginLifecycleHistoryRequest: Encodable {
    let workspaceID: String?
    let pluginID: String?
    let kind: String?
    let state: String?
    let eventType: String?
    let cursorCreatedAt: String?
    let cursorID: String?
    let limit: Int?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case pluginID = "plugin_id"
        case kind
        case state
        case eventType = "event_type"
        case cursorCreatedAt = "cursor_created_at"
        case cursorID = "cursor_id"
        case limit
    }
}

struct DaemonPluginLifecycleHistoryRecord: Decodable, Sendable {
    let auditID: String
    let workspaceID: String
    let pluginID: String
    let kind: String
    let state: String
    let eventType: String
    let processID: Int
    let restartCount: Int
    let reason: String
    let error: String?
    let restartEvent: Bool
    let failureEvent: Bool
    let recoveryEvent: Bool
    let lastHeartbeatAt: String?
    let lastTransitionAt: String?
    let occurredAt: String

    enum CodingKeys: String, CodingKey {
        case auditID = "audit_id"
        case workspaceID = "workspace_id"
        case pluginID = "plugin_id"
        case kind
        case state
        case eventType = "event_type"
        case processID = "process_id"
        case restartCount = "restart_count"
        case reason
        case error
        case restartEvent = "restart_event"
        case failureEvent = "failure_event"
        case recoveryEvent = "recovery_event"
        case lastHeartbeatAt = "last_heartbeat_at"
        case lastTransitionAt = "last_transition_at"
        case occurredAt = "occurred_at"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        auditID = try container.decodeIfPresent(String.self, forKey: .auditID) ?? ""
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        pluginID = try container.decodeIfPresent(String.self, forKey: .pluginID) ?? ""
        kind = try container.decodeIfPresent(String.self, forKey: .kind) ?? "unknown"
        state = try container.decodeIfPresent(String.self, forKey: .state) ?? "unknown"
        eventType = try container.decodeIfPresent(String.self, forKey: .eventType) ?? "unknown"
        processID = try container.decodeIfPresent(Int.self, forKey: .processID) ?? 0
        restartCount = try container.decodeIfPresent(Int.self, forKey: .restartCount) ?? 0
        reason = try container.decodeIfPresent(String.self, forKey: .reason) ?? "unknown"
        error = try container.decodeIfPresent(String.self, forKey: .error)
        restartEvent = try container.decodeIfPresent(Bool.self, forKey: .restartEvent) ?? false
        failureEvent = try container.decodeIfPresent(Bool.self, forKey: .failureEvent) ?? false
        recoveryEvent = try container.decodeIfPresent(Bool.self, forKey: .recoveryEvent) ?? false
        lastHeartbeatAt = try container.decodeIfPresent(String.self, forKey: .lastHeartbeatAt)
        lastTransitionAt = try container.decodeIfPresent(String.self, forKey: .lastTransitionAt)
        occurredAt = try container.decodeIfPresent(String.self, forKey: .occurredAt) ?? ""
    }
}

struct DaemonPluginLifecycleHistoryResponse: Decodable, Sendable {
    let workspaceID: String
    let items: [DaemonPluginLifecycleHistoryRecord]
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
        items = try container.decodeIfPresent([DaemonPluginLifecycleHistoryRecord].self, forKey: .items) ?? []
        hasMore = try container.decodeIfPresent(Bool.self, forKey: .hasMore) ?? false
        nextCursorCreatedAt = try container.decodeIfPresent(String.self, forKey: .nextCursorCreatedAt)
        nextCursorID = try container.decodeIfPresent(String.self, forKey: .nextCursorID)
    }
}
