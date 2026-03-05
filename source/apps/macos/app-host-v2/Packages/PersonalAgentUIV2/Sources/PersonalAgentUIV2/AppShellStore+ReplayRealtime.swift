import Foundation

@MainActor
extension AppShellV2Store {
    public func startReplayRealtimeIfNeeded() {
        guard replayRealtimeTask == nil, replayRealtimeReconnectTask == nil else {
            return
        }
        Task { [weak self] in
            await self?.connectReplayRealtimeStream(isRetry: false)
        }
    }

    public func retryReplayRealtimeStream() {
        Task { [weak self] in
            await self?.restartReplayRealtimeStream()
        }
    }

    public func stopReplayRealtimeStream() async {
        replayRealtimeReconnectTask?.cancel()
        replayRealtimeReconnectTask = nil
        replayRealtimeRefreshTask?.cancel()
        replayRealtimeRefreshTask = nil
        replayRealtimeTask?.cancel()
        replayRealtimeTask = nil
        replayRealtimeConnectionID = nil

        if let session = replayRealtimeSession {
            await session.close()
        }
        replayRealtimeSession = nil

        replayRealtimeState = V2ReplayRealtimeState(phase: .idle, lastEventAt: replayRealtimeState.lastEventAt)
    }

    func replayRealtimeEventRequiresRefresh(_ event: V2DaemonRealtimeEventEnvelope) -> Bool {
        let type = event.eventType.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        if type.contains("approval") ||
            type.contains("task") ||
            type.contains("turn_item") ||
            type.contains("tool_call") ||
            type.contains("tool_result") ||
            type.contains("assistant_message") ||
            type.contains("run_") ||
            type.contains("workflow") ||
            type.contains("agent") ||
            type.contains("chat") {
            return true
        }

        if replayRealtimeNonEmpty(event.payload.approvalRequestID) != nil ||
            replayRealtimeNonEmpty(event.payload.taskID) != nil ||
            replayRealtimeNonEmpty(event.payload.runID) != nil {
            return true
        }

        let statusText = "\(event.payload.status ?? "") \(event.payload.state ?? "")"
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .lowercased()
        if statusText.contains("pending") ||
            statusText.contains("running") ||
            statusText.contains("completed") ||
            statusText.contains("failed") ||
            statusText.contains("approval") {
            return true
        }

        return false
    }

    private func restartReplayRealtimeStream() async {
        let readiness = sessionReadiness
        guard readiness.isReadyForDaemonMutations else {
            setFeedback(readiness.setupSummary)
            selectedSection = .getStarted
            return
        }

        setFeedback("Retrying realtime stream…")
        await stopReplayRealtimeStream()
        await connectReplayRealtimeStream(isRetry: true)
    }

    private func connectReplayRealtimeStream(isRetry: Bool) async {
        guard replayRealtimeTask == nil else {
            return
        }
        guard let context = replayRealtimeContext() else {
            replayRealtimeState = V2ReplayRealtimeState(phase: .idle, lastEventAt: replayRealtimeState.lastEventAt)
            clearPanelProblem(for: .realtime)
            return
        }

        replayRealtimeState.phase = isRetry ? .reconnecting : .connecting
        replayRealtimeState.lastErrorSummary = nil

        do {
            let streamCorrelation = "replay-realtime-\(UUID().uuidString.lowercased())"
            let session = try daemonClient.connectRealtime(
                baseURL: context.baseURL,
                authToken: context.authToken,
                correlationID: streamCorrelation
            )

            let connectionID = UUID()
            replayRealtimeConnectionID = connectionID
            replayRealtimeSession = session
            replayRealtimeState.phase = .connected
            replayRealtimeState.reconnectAttempt = 0
            replayRealtimeState.lastErrorSummary = nil
            clearPanelProblem(for: .realtime)

            replayRealtimeTask = Task { [weak self] in
                await self?.consumeReplayRealtimeEvents(session: session, connectionID: connectionID)
            }
        } catch {
            handleReplayRealtimeFailure(error)
        }
    }

    private func consumeReplayRealtimeEvents(
        session: V2DaemonRealtimeSession,
        connectionID: UUID
    ) async {
        defer {
            if replayRealtimeConnectionID == connectionID {
                replayRealtimeTask = nil
                replayRealtimeSession = nil
            }
        }

        while !Task.isCancelled {
            do {
                let event = try await session.receive()
                handleReplayRealtimeEvent(event)
            } catch {
                if Task.isCancelled {
                    break
                }
                handleReplayRealtimeFailure(error)
                break
            }
        }
    }

    private func handleReplayRealtimeEvent(_ event: V2DaemonRealtimeEventEnvelope) {
        replayRealtimeState.phase = .connected
        replayRealtimeState.lastEventAt = Date()
        replayRealtimeState.lastErrorSummary = nil
        clearPanelProblem(for: .realtime)

        guard replayRealtimeEventRequiresRefresh(event) else {
            return
        }
        scheduleReplayRealtimeRefresh()
    }

    private func handleReplayRealtimeFailure(_ error: Error) {
        setPanelProblem(error, context: .realtime)
        let mapped = panelProblem(for: .realtime) ?? V2DaemonProblemMapper.map(error: error, context: .realtime)
        replayRealtimeState.phase = .disconnected
        replayRealtimeState.lastErrorSummary = mapped.summary
        replayRealtimeTask = nil
        replayRealtimeSession = nil

        if mapped.kind == .missingAuth || mapped.kind == .authScope || mapped.kind == .validation {
            return
        }
        scheduleReplayRealtimeReconnect()
    }

    private func scheduleReplayRealtimeReconnect() {
        guard sessionReadiness.isReadyForDaemonMutations else {
            replayRealtimeState.phase = .idle
            replayRealtimeState.reconnectAttempt = 0
            return
        }

        replayRealtimeReconnectTask?.cancel()
        let attempt = replayRealtimeState.reconnectAttempt + 1
        replayRealtimeState.reconnectAttempt = attempt
        replayRealtimeState.phase = .reconnecting

        let delaySeconds = min(max(1, attempt) * 2, 12)
        replayRealtimeReconnectTask = Task { [weak self] in
            try? await Task.sleep(nanoseconds: UInt64(delaySeconds) * 1_000_000_000)
            guard let self else {
                return
            }
            await self.connectReplayRealtimeStream(isRetry: true)
            self.replayRealtimeReconnectTask = nil
        }
    }

    private func scheduleReplayRealtimeRefresh() {
        guard replayRealtimeRefreshTask == nil else {
            return
        }

        replayRealtimeRefreshTask = Task { [weak self] in
            try? await Task.sleep(nanoseconds: 300_000_000)
            guard let self else {
                return
            }
            await self.refreshReplayFeed(resetPagination: true)
            await self.refreshSelectedReplayDetailEvidence(force: true)
            self.replayRealtimeRefreshTask = nil
        }
    }

    private func replayRealtimeContext() -> (baseURL: URL, authToken: String)? {
        guard sessionReadiness.isReadyForDaemonMutations else {
            return nil
        }

        let trimmedBaseURL = daemonBaseURL.trimmingCharacters(in: .whitespacesAndNewlines)
        guard let baseURL = URL(string: trimmedBaseURL),
              !trimmedBaseURL.isEmpty,
              let components = URLComponents(url: baseURL, resolvingAgainstBaseURL: false),
              let scheme = components.scheme?.lowercased(),
              ["http", "https", "ws", "wss"].contains(scheme),
              components.host != nil else {
            return nil
        }

        guard let authToken = sessionConfigStore.resolvedAccessToken()?.trimmingCharacters(in: .whitespacesAndNewlines),
              !authToken.isEmpty else {
            return nil
        }

        return (baseURL, authToken)
    }

    private func replayRealtimeNonEmpty(_ value: String?) -> String? {
        guard let trimmed = value?.trimmingCharacters(in: .whitespacesAndNewlines), !trimmed.isEmpty else {
            return nil
        }
        return trimmed
    }
}
