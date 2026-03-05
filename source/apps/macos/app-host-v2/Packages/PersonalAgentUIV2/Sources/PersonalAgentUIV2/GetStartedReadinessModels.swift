import Foundation

public struct V2ReplayAvailabilitySummary: Sendable, Equatable {
    public var approvalCount: Int
    public var taskCount: Int
    public var historyCount: Int

    public init(approvalCount: Int = 0, taskCount: Int = 0, historyCount: Int = 0) {
        self.approvalCount = max(approvalCount, 0)
        self.taskCount = max(taskCount, 0)
        self.historyCount = max(historyCount, 0)
    }

    public var total: Int {
        approvalCount + taskCount + historyCount
    }
}

public struct V2GetStartedReadinessSnapshot: Sendable, Equatable {
    public var lifecycleStatus: V2DaemonLifecycleStatusResponse?
    public var lifecycleError: String?
    public var modelRoute: V2DaemonModelResolveResponse?
    public var modelRouteError: String?
    public var connectorCards: [V2DaemonConnectorStatusCard]
    public var connectorError: String?
    public var replayAvailability: V2ReplayAvailabilitySummary?
    public var replayError: String?
    public var lastUpdatedAt: Date?

    public init(
        lifecycleStatus: V2DaemonLifecycleStatusResponse? = nil,
        lifecycleError: String? = nil,
        modelRoute: V2DaemonModelResolveResponse? = nil,
        modelRouteError: String? = nil,
        connectorCards: [V2DaemonConnectorStatusCard] = [],
        connectorError: String? = nil,
        replayAvailability: V2ReplayAvailabilitySummary? = nil,
        replayError: String? = nil,
        lastUpdatedAt: Date? = nil
    ) {
        self.lifecycleStatus = lifecycleStatus
        self.lifecycleError = lifecycleError
        self.modelRoute = modelRoute
        self.modelRouteError = modelRouteError
        self.connectorCards = connectorCards
        self.connectorError = connectorError
        self.replayAvailability = replayAvailability
        self.replayError = replayError
        self.lastUpdatedAt = lastUpdatedAt
    }

    public var connectedConnectorCount: Int {
        connectorCards.filter(\.isReadyForTrustWorkflow).count
    }

    public var connectorsNeedingAttentionCount: Int {
        connectorCards.filter { !$0.isReadyForTrustWorkflow }.count
    }

    public var hasReplayActivity: Bool {
        (replayAvailability?.total ?? 0) > 0
    }

    public var hasRouteResolution: Bool {
        guard let route = modelRoute else {
            return false
        }
        let provider = route.provider.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        let modelKey = route.modelKey.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        guard !provider.isEmpty, !modelKey.isEmpty else {
            return false
        }
        return provider != "unknown" && modelKey != "unknown"
    }

    public var lifecycleIsOperational: Bool {
        guard let status = lifecycleStatus else {
            return false
        }
        if status.needsInstall || status.needsRepair {
            return false
        }
        let lifecycleState = status.lifecycleState.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        let authState = status.controlAuth.state.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        if !["running", "ready"].contains(lifecycleState) {
            return false
        }
        if ["missing", "invalid", "expired", "error", "denied"].contains(authState) {
            return false
        }
        return status.workerSummary.failed == 0
    }
}

private extension V2DaemonConnectorStatusCard {
    var isReadyForTrustWorkflow: Bool {
        let normalizedStatus = status.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        let healthyStatus = ["healthy", "ready", "connected", "ok", "running", "active"]
        return enabled && configured && healthyStatus.contains(normalizedStatus)
    }
}
