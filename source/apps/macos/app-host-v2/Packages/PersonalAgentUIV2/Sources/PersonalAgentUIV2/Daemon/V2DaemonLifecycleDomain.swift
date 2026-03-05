import Foundation

public struct V2DaemonLifecycleWorkerSummary: Decodable, Sendable, Equatable {
    public let total: Int
    public let running: Int
    public let failed: Int

    enum CodingKeys: String, CodingKey {
        case total
        case running
        case failed
    }

    public init(total: Int = 0, running: Int = 0, failed: Int = 0) {
        self.total = total
        self.running = running
        self.failed = failed
    }
}

public struct V2DaemonLifecycleControlAuthState: Decodable, Sendable, Equatable {
    public let state: String
    public let source: String
    public let remediationHints: [String]

    enum CodingKeys: String, CodingKey {
        case state
        case source
        case remediationHints = "remediation_hints"
    }

    public init(state: String = "unknown", source: String = "unknown", remediationHints: [String] = []) {
        self.state = state
        self.source = source
        self.remediationHints = remediationHints
    }
}

public struct V2DaemonLifecycleHealthClassification: Decodable, Sendable, Equatable {
    public let overallState: String
    public let coreRuntimeState: String
    public let pluginRuntimeState: String
    public let blocking: Bool

    enum CodingKeys: String, CodingKey {
        case overallState = "overall_state"
        case coreRuntimeState = "core_runtime_state"
        case pluginRuntimeState = "plugin_runtime_state"
        case blocking
    }

    public init(
        overallState: String = "unknown",
        coreRuntimeState: String = "unknown",
        pluginRuntimeState: String = "unknown",
        blocking: Bool = false
    ) {
        self.overallState = overallState
        self.coreRuntimeState = coreRuntimeState
        self.pluginRuntimeState = pluginRuntimeState
        self.blocking = blocking
    }
}

public struct V2DaemonLifecycleStatusResponse: Decodable, Sendable, Equatable {
    public let lifecycleState: String
    public let setupState: String
    public let installState: String
    public let needsInstall: Bool
    public let needsRepair: Bool
    public let repairHint: String?
    public let configuredAddress: String?
    public let boundAddress: String?
    public let controlAuth: V2DaemonLifecycleControlAuthState
    public let workerSummary: V2DaemonLifecycleWorkerSummary
    public let healthClassification: V2DaemonLifecycleHealthClassification

    enum CodingKeys: String, CodingKey {
        case lifecycleState = "lifecycle_state"
        case setupState = "setup_state"
        case installState = "install_state"
        case needsInstall = "needs_install"
        case needsRepair = "needs_repair"
        case repairHint = "repair_hint"
        case configuredAddress = "configured_address"
        case boundAddress = "bound_address"
        case controlAuth = "control_auth"
        case workerSummary = "worker_summary"
        case healthClassification = "health_classification"
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        lifecycleState = try container.decodeIfPresent(String.self, forKey: .lifecycleState) ?? "unknown"
        setupState = try container.decodeIfPresent(String.self, forKey: .setupState) ?? "unknown"
        installState = try container.decodeIfPresent(String.self, forKey: .installState) ?? "unknown"
        needsInstall = try container.decodeIfPresent(Bool.self, forKey: .needsInstall) ?? false
        needsRepair = try container.decodeIfPresent(Bool.self, forKey: .needsRepair) ?? false
        repairHint = try container.decodeIfPresent(String.self, forKey: .repairHint)
        configuredAddress = try container.decodeIfPresent(String.self, forKey: .configuredAddress)
        boundAddress = try container.decodeIfPresent(String.self, forKey: .boundAddress)
        controlAuth = try container.decodeIfPresent(V2DaemonLifecycleControlAuthState.self, forKey: .controlAuth)
            ?? V2DaemonLifecycleControlAuthState()
        workerSummary = try container.decodeIfPresent(V2DaemonLifecycleWorkerSummary.self, forKey: .workerSummary)
            ?? V2DaemonLifecycleWorkerSummary()
        healthClassification = try container.decodeIfPresent(V2DaemonLifecycleHealthClassification.self, forKey: .healthClassification)
            ?? V2DaemonLifecycleHealthClassification()
    }
}

public struct V2DaemonLifecycleAPI {
    private let client: V2DaemonAPIClient

    init(client: V2DaemonAPIClient) {
        self.client = client
    }

    public func status(baseURL: URL, authToken: String) async throws -> V2DaemonLifecycleStatusResponse {
        try await client.request(
            baseURL: baseURL,
            path: "/v1/daemon/lifecycle/status",
            method: "GET",
            authToken: authToken,
            body: Optional<V2DaemonWorkspaceRequest>.none
        )
    }
}
