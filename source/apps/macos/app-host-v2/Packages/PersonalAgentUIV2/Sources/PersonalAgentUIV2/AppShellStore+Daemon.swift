import Foundation

@MainActor
extension AppShellV2Store {
    private struct V2ReplayAvailabilityProbeResult {
        var summary: V2ReplayAvailabilitySummary?
        var error: Error?
    }

    public func refreshGetStartedReadinessIfNeeded(force: Bool = false) async {
        if isReadinessRefreshInFlight {
            return
        }
        if !force,
           selectedSection != .getStarted,
           getStartedReadinessSnapshot.lastUpdatedAt != nil {
            return
        }
        _ = await refreshGetStartedReadiness()
    }

    @discardableResult
    public func refreshGetStartedReadiness() async -> Bool {
        guard !isReadinessRefreshInFlight else {
            return getStartedReadinessSnapshot.lifecycleIsOperational
        }

        isReadinessRefreshInFlight = true
        defer { isReadinessRefreshInFlight = false }

        let authToken = sessionConfigStore.resolvedAccessToken() ?? ""
        let normalizedToken = authToken.trimmingCharacters(in: .whitespacesAndNewlines)
        guard let baseURL = validatedDaemonBaseURL(from: daemonBaseURL) else {
            let error = V2DaemonAPIError.invalidBaseURL
            setPanelProblem(error, context: .setup)
            getStartedReadinessSnapshot = V2GetStartedReadinessSnapshot(
                lifecycleError: mappedProblemSummary(error, context: .setup),
                lastUpdatedAt: Date()
            )
            return false
        }

        guard !normalizedToken.isEmpty else {
            let error = V2DaemonAPIError.missingAuthToken
            setPanelProblem(error, context: .setup)
            getStartedReadinessSnapshot = V2GetStartedReadinessSnapshot(
                lifecycleError: mappedProblemSummary(error, context: .setup),
                lastUpdatedAt: Date()
            )
            return false
        }

        async let lifecycleResult: Result<V2DaemonLifecycleStatusResponse, Error> = captured { [self] in
            try await self.daemonClient.lifecycle.status(baseURL: baseURL, authToken: normalizedToken)
        }
        async let routeResult: Result<V2DaemonModelResolveResponse, Error> = captured { [self] in
            try await self.daemonClient.models.modelResolve(
                baseURL: baseURL,
                authToken: normalizedToken,
                workspaceID: self.workspaceID,
                taskClass: "chat"
            )
        }
        async let connectorsResult: Result<V2DaemonConnectorStatusResponse, Error> = captured { [self] in
            try await self.daemonClient.connectors.connectorStatus(
                baseURL: baseURL,
                authToken: normalizedToken,
                workspaceID: self.workspaceID
            )
        }
        async let replayProbeResult: V2ReplayAvailabilityProbeResult = probeReplayAvailability(
            baseURL: baseURL,
            authToken: normalizedToken
        )

        var snapshot = V2GetStartedReadinessSnapshot()

        switch await lifecycleResult {
        case .success(let lifecycleStatus):
            snapshot.lifecycleStatus = lifecycleStatus
            clearPanelProblem(for: .setup)
        case .failure(let error):
            snapshot.lifecycleError = mappedProblemSummary(error, context: .setup)
            setPanelProblem(error, context: .setup)
        }

        switch await routeResult {
        case .success(let route):
            snapshot.modelRoute = route
            snapshot.modelRouteError = nil
            clearPanelProblem(for: .models)
        case .failure(let error):
            snapshot.modelRouteError = mappedProblemSummary(error, context: .models)
            setPanelProblem(error, context: .models)
        }

        switch await connectorsResult {
        case .success(let connectorStatus):
            snapshot.connectorCards = connectorStatus.connectors
            snapshot.connectorError = nil
            clearPanelProblem(for: .connectors)
        case .failure(let error):
            snapshot.connectorError = mappedProblemSummary(error, context: .connectors)
            setPanelProblem(error, context: .connectors)
        }

        let replayResult = await replayProbeResult
        snapshot.replayAvailability = replayResult.summary
        if let replayError = replayResult.error {
            snapshot.replayError = mappedProblemSummary(replayError, context: .replay)
            setPanelProblem(replayError, context: .replay)
        } else {
            snapshot.replayError = nil
            clearPanelProblem(for: .replay)
        }

        snapshot.lastUpdatedAt = Date()
        getStartedReadinessSnapshot = snapshot
        if snapshot.lifecycleIsOperational {
            setFeedback("Daemon connection verified.")
        }
        return snapshot.lifecycleStatus != nil && snapshot.lifecycleError == nil
    }

    public func probeDaemonConnection() async {
        let lifecycle = mutationLifecycle(for: .daemonProbe)
        if lifecycle.isDisabled {
            setFeedback(lifecycle.message ?? "Verify daemon is unavailable.")
            selectedSection = .getStarted
            return
        }

        startMutation(.daemonProbe, message: "Verifying daemon connection…")
        isDaemonProbeInFlight = true
        defer { isDaemonProbeInFlight = false }
        let success = await refreshGetStartedReadiness()
        if success {
            setSetupActionStatus(.lifecycleOperational, message: "Daemon lifecycle verified.")
            completeMutation(.daemonProbe, message: "Daemon connection verified.")
        } else {
            setSetupActionStatus(
                .lifecycleOperational,
                message: panelProblem(for: .setup)?.summary ?? "Daemon verification failed."
            )
            failMutation(.daemonProbe, message: panelProblem(for: .setup)?.summary ?? "Could not verify daemon connection.")
        }
    }

    @discardableResult
    public func probeDaemon(baseURLString: String, authToken: String) async -> Bool {
        let trimmedBaseURL = baseURLString.trimmingCharacters(in: .whitespacesAndNewlines)
        guard let baseURL = URL(string: trimmedBaseURL),
              !trimmedBaseURL.isEmpty,
              let components = URLComponents(url: baseURL, resolvingAgainstBaseURL: false),
              let scheme = components.scheme?.lowercased(),
              ["http", "https", "ws", "wss"].contains(scheme),
              components.host != nil else {
            setPanelProblem(V2DaemonAPIError.invalidBaseURL, context: .setup)
            return false
        }

        do {
            _ = try await daemonClient.lifecycle.status(baseURL: baseURL, authToken: authToken)
            clearPanelProblem(for: .setup)
            setFeedback("Daemon connection verified.")
            return true
        } catch {
            setPanelProblem(error, context: .setup)
            return false
        }
    }

    private func validatedDaemonBaseURL(from rawValue: String) -> URL? {
        let trimmedBaseURL = rawValue.trimmingCharacters(in: .whitespacesAndNewlines)
        guard let baseURL = URL(string: trimmedBaseURL),
              !trimmedBaseURL.isEmpty,
              let components = URLComponents(url: baseURL, resolvingAgainstBaseURL: false),
              let scheme = components.scheme?.lowercased(),
              ["http", "https", "ws", "wss"].contains(scheme),
              components.host != nil else {
            return nil
        }
        return baseURL
    }

    private func captured<T>(_ operation: @escaping () async throws -> T) async -> Result<T, Error> {
        do {
            return .success(try await operation())
        } catch {
            return .failure(error)
        }
    }

    private func mappedProblemSummary(_ error: Error, context: V2ProblemContext) -> String {
        V2DaemonProblemMapper.map(error: error, context: context).summary
    }

    private func probeReplayAvailability(
        baseURL: URL,
        authToken: String
    ) async -> V2ReplayAvailabilityProbeResult {
        async let approvalsResult: Result<V2DaemonApprovalInboxResponse, Error> = captured { [self] in
            try await self.daemonClient.approvals.approvalInbox(
                baseURL: baseURL,
                authToken: authToken,
                workspaceID: self.workspaceID,
                limit: 24
            )
        }
        async let tasksResult: Result<V2DaemonTaskRunListResponse, Error> = captured { [self] in
            try await self.daemonClient.tasks.taskRunList(
                baseURL: baseURL,
                authToken: authToken,
                workspaceID: self.workspaceID,
                limit: 24
            )
        }
        async let historyResult: Result<V2DaemonChatTurnHistoryResponse, Error> = captured { [self] in
            try await self.daemonClient.chat.chatTurnHistory(
                baseURL: baseURL,
                authToken: authToken,
                workspaceID: self.workspaceID,
                limit: 24
            )
        }

        var summary = V2ReplayAvailabilitySummary()
        var successfulQueries = 0
        var firstError: Error?

        switch await approvalsResult {
        case .success(let response):
            summary.approvalCount = response.approvals.count
            successfulQueries += 1
        case .failure(let error):
            firstError = firstError ?? error
        }

        switch await tasksResult {
        case .success(let response):
            summary.taskCount = response.items.count
            successfulQueries += 1
        case .failure(let error):
            firstError = firstError ?? error
        }

        switch await historyResult {
        case .success(let response):
            summary.historyCount = response.items.count
            successfulQueries += 1
        case .failure(let error):
            firstError = firstError ?? error
        }

        if successfulQueries == 0 {
            return V2ReplayAvailabilityProbeResult(summary: nil, error: firstError)
        }
        return V2ReplayAvailabilityProbeResult(summary: summary, error: nil)
    }
}
