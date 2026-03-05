import Foundation

@MainActor
extension AppShellV2Store {
    public func refreshSelectedReplayDetailEvidenceIfNeeded(force: Bool = false) async {
        guard let event = selectedEvent else {
            return
        }
        if !force,
           let existing = replayDetailEvidenceByReplayKey[event.replayKey],
           (existing.phase == .ready || existing.phase == .empty),
           existing.lastUpdatedAt != nil {
            return
        }
        await refreshReplayDetailEvidence(for: event, force: force)
    }

    public func refreshSelectedReplayDetailEvidence(force: Bool = false) async {
        guard let event = selectedEvent else {
            return
        }
        await refreshReplayDetailEvidence(for: event, force: force)
    }

    public func openReplayMaintenanceFromDetail() {
        selectedSection = .connectorsAndModels
    }

    private func refreshReplayDetailEvidence(for event: ReplayEvent, force: Bool) async {
        let key = event.replayKey
        if replayDetailFetchInFlightKeys.contains(key) {
            return
        }
        if !force,
           let existing = replayDetailEvidenceByReplayKey[key],
           existing.phase == .loading {
            return
        }

        replayDetailFetchInFlightKeys.insert(key)
        replayDetailEvidenceByReplayKey[key] = V2ReplayDetailEvidenceState(
            replayKey: key,
            phase: .loading,
            summary: "Loading replay evidence…"
        )
        defer {
            replayDetailFetchInFlightKeys.remove(key)
        }

        guard sessionReadiness.isReadyForDaemonMutations else {
            replayDetailEvidenceByReplayKey[key] = V2ReplayDetailEvidenceState(
                replayKey: key,
                phase: .failed,
                summary: sessionReadiness.setupSummary,
                whatCameIn: event.instruction,
                whatAssistantUnderstood: event.interpretedIntent,
                whatHappened: event.actionSummary,
                approvalContext: event.approvalReason,
                sourceContextFields: event.sourceContext.fields,
                decisionTrace: event.decisionTrace,
                channelsTouched: event.channelsTouched,
                confidenceScore: event.confidenceScore,
                failureHint: event.failureRecoveryHint,
                lastUpdatedAt: Date()
            )
            return
        }

        guard let baseURL = validatedReplayDetailBaseURL() else {
            replayDetailEvidenceByReplayKey[key] = V2ReplayDetailEvidenceState(
                replayKey: key,
                phase: .failed,
                summary: mappedReplayDetailProblemSummary(V2DaemonAPIError.invalidBaseURL),
                whatCameIn: event.instruction,
                whatAssistantUnderstood: event.interpretedIntent,
                whatHappened: event.actionSummary,
                approvalContext: event.approvalReason,
                sourceContextFields: event.sourceContext.fields,
                decisionTrace: event.decisionTrace,
                channelsTouched: event.channelsTouched,
                confidenceScore: event.confidenceScore,
                failureHint: event.failureRecoveryHint,
                lastUpdatedAt: Date()
            )
            return
        }

        guard let authToken = resolvedReplayDetailToken() else {
            replayDetailEvidenceByReplayKey[key] = V2ReplayDetailEvidenceState(
                replayKey: key,
                phase: .failed,
                summary: mappedReplayDetailProblemSummary(V2DaemonAPIError.missingAuthToken),
                whatCameIn: event.instruction,
                whatAssistantUnderstood: event.interpretedIntent,
                whatHappened: event.actionSummary,
                approvalContext: event.approvalReason,
                sourceContextFields: event.sourceContext.fields,
                decisionTrace: event.decisionTrace,
                channelsTouched: event.channelsTouched,
                confidenceScore: event.confidenceScore,
                failureHint: event.failureRecoveryHint,
                lastUpdatedAt: Date()
            )
            return
        }

        guard let locator = event.daemonLocator else {
            replayDetailEvidenceByReplayKey[key] = V2ReplayDetailEvidenceState(
                replayKey: key,
                phase: .empty,
                summary: "No daemon evidence locator is available for this replay item.",
                whatCameIn: event.instruction,
                whatAssistantUnderstood: event.interpretedIntent,
                whatHappened: event.actionSummary,
                approvalContext: event.approvalReason,
                sourceContextFields: event.sourceContext.fields,
                decisionTrace: event.decisionTrace,
                channelsTouched: event.channelsTouched,
                confidenceScore: event.confidenceScore,
                failureHint: event.failureRecoveryHint,
                lastUpdatedAt: Date()
            )
            return
        }

        async let inspectRunResult: Result<V2DaemonInspectRunResponse?, Error> = captureReplayDetailRequest { [self] in
            guard let runID = locator.runID?.trimmingCharacters(in: .whitespacesAndNewlines), !runID.isEmpty else {
                return nil
            }
            return try await self.daemonClient.inspect.inspectRun(
                baseURL: baseURL,
                authToken: authToken,
                runID: runID
            )
        }

        async let inspectLogsResult: Result<V2DaemonInspectLogQueryResponse?, Error> = captureReplayDetailRequest { [self] in
            guard let runID = locator.runID?.trimmingCharacters(in: .whitespacesAndNewlines), !runID.isEmpty else {
                return nil
            }
            return try await self.daemonClient.inspect.inspectLogsQuery(
                baseURL: baseURL,
                authToken: authToken,
                workspaceID: self.workspaceID,
                runID: runID,
                limit: 40
            )
        }

        async let historyResult: Result<V2DaemonChatTurnHistoryResponse?, Error> = captureReplayDetailRequest { [self] in
            guard let correlationID = locator.correlationID?.trimmingCharacters(in: .whitespacesAndNewlines), !correlationID.isEmpty else {
                return nil
            }
            return try await self.daemonClient.chat.chatTurnHistory(
                baseURL: baseURL,
                authToken: authToken,
                workspaceID: self.workspaceID,
                correlationID: correlationID,
                limit: 80
            )
        }

        async let approvalResult: Result<V2DaemonApprovalInboxResponse?, Error> = captureReplayDetailRequest { [self] in
            guard locator.approvalRequestID != nil else {
                return nil
            }
            return try await self.daemonClient.approvals.approvalInbox(
                baseURL: baseURL,
                authToken: authToken,
                workspaceID: self.workspaceID,
                limit: 80
            )
        }

        let inspectRunResolved = await inspectRunResult
        let inspectLogsResolved = await inspectLogsResult
        let historyResolved = await historyResult
        let approvalResolved = await approvalResult

        var firstError: Error?
        var inspectRun: V2DaemonInspectRunResponse?
        var inspectLogs: [V2DaemonInspectLogRecord] = []
        var historyRecords: [V2DaemonChatTurnHistoryRecord] = []
        var approvalRecord: V2DaemonApprovalInboxRecord?

        switch inspectRunResolved {
        case .success(let response):
            inspectRun = response
        case .failure(let error):
            firstError = firstError ?? error
        }

        switch inspectLogsResolved {
        case .success(let response):
            inspectLogs = response?.logs ?? []
        case .failure(let error):
            firstError = firstError ?? error
        }

        switch historyResolved {
        case .success(let response):
            let correlationID = nonEmpty(locator.correlationID)
            if let correlationID {
                historyRecords = (response?.items ?? []).filter {
                    $0.correlationID.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() == correlationID.lowercased()
                }
            } else {
                historyRecords = response?.items ?? []
            }
        case .failure(let error):
            firstError = firstError ?? error
        }

        switch approvalResolved {
        case .success(let response):
            if let approvalID = nonEmpty(locator.approvalRequestID) {
                approvalRecord = response?.approvals.first(where: {
                    $0.approvalRequestID.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() == approvalID.lowercased()
                })
            }
        case .failure(let error):
            firstError = firstError ?? error
        }

        var sourceContextFields = event.sourceContext.fields
        var decisionTrace = event.decisionTrace
        var channelsTouched = event.channelsTouched
        var whatCameIn = event.instruction
        var whatUnderstood = event.interpretedIntent
        var whatHappened = event.actionSummary
        var approvalContext = event.approvalReason
        var confidenceScore = event.confidenceScore
        var failureHint = event.failureRecoveryHint
        var didEnhance = false

        if !historyRecords.isEmpty {
            let sorted = historyRecords.sorted { lhs, rhs in
                if lhs.itemIndex != rhs.itemIndex {
                    return lhs.itemIndex < rhs.itemIndex
                }
                return parseReplayDetailDate(lhs.createdAt) < parseReplayDetailDate(rhs.createdAt)
            }

            if let userMessage = sorted.first(where: { record in
                let itemType = record.item.type.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
                return itemType == "user_message" || record.item.role?.lowercased() == "user"
            })?.item.content?.trimmingCharacters(in: .whitespacesAndNewlines), !userMessage.isEmpty {
                whatCameIn = userMessage
                didEnhance = true
            }

            if let assistantMessage = sorted.last(where: { record in
                let itemType = record.item.type.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
                return itemType == "assistant_message"
            })?.item.content?.trimmingCharacters(in: .whitespacesAndNewlines), !assistantMessage.isEmpty {
                whatHappened = assistantMessage
                didEnhance = true
            }

            if let taskClass = sorted.compactMap({ $0.item.metadata?.taskClass }).first,
               !taskClass.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                whatUnderstood = "Interpreted as \(taskClass) workflow."
                didEnhance = true
            }

            for record in sorted {
                if let channelID = nonEmpty(record.channelID), !channelsTouched.contains(channelID) {
                    channelsTouched.append(channelID)
                    didEnhance = true
                }
                if let confidenceValue = replayDetailConfidence(from: record.item.metadata?.additional["confidence"]) {
                    confidenceScore = confidenceValue
                    didEnhance = true
                }
                if let connectorID = record.item.metadata?.additional["connector_id"]?.stringValue,
                   let normalized = nonEmpty(connectorID),
                   !channelsTouched.contains(normalized) {
                    channelsTouched.append(normalized)
                    didEnhance = true
                }
            }
        }

        if let inspectRun {
            if let routeNotes = nonEmpty(inspectRun.route?.notes) {
                whatUnderstood = routeNotes
                didEnhance = true
            }
            if let runError = nonEmpty(inspectRun.run.lastError) {
                whatHappened = runError
                failureHint = "Resolve this run error, then retry from Replay."
                didEnhance = true
            } else {
                whatHappened = "Run state: \(inspectRun.run.state)."
                didEnhance = true
            }

            if !inspectRun.steps.isEmpty {
                decisionTrace = inspectRun.steps.map { step in
                    ReplayDecisionStage(
                        title: step.name,
                        detail: step.lastError ?? "State: \(step.status)",
                        status: replayDetailTraceStatus(step.status)
                    )
                }
                didEnhance = true
            }

            appendReplayDetailSourceField(
                label: "Run ID",
                value: inspectRun.run.runID,
                into: &sourceContextFields,
                didEnhance: &didEnhance
            )
            appendReplayDetailSourceField(
                label: "Task ID",
                value: inspectRun.run.taskID,
                into: &sourceContextFields,
                didEnhance: &didEnhance
            )
            appendReplayDetailSourceField(
                label: "Run State",
                value: inspectRun.run.state,
                into: &sourceContextFields,
                didEnhance: &didEnhance
            )
        }

        if !inspectLogs.isEmpty && (inspectRun == nil || decisionTrace == event.decisionTrace) {
            decisionTrace = inspectLogs.prefix(6).map { log in
                ReplayDecisionStage(
                    title: replayDetailStageTitle(log.eventType),
                    detail: nonEmpty(log.outputSummary) ?? nonEmpty(log.inputSummary) ?? "Status: \(log.status)",
                    status: replayDetailTraceStatus(log.status)
                )
            }
            didEnhance = true
        }

        if let approvalRecord {
            if let rationale = nonEmpty(approvalRecord.riskRationale) {
                approvalContext = rationale
                didEnhance = true
            }
            appendReplayDetailSourceField(
                label: "Approval ID",
                value: approvalRecord.approvalRequestID,
                into: &sourceContextFields,
                didEnhance: &didEnhance
            )
        }

        channelsTouched = Array(Set(channelsTouched)).sorted()

        if !didEnhance {
            if let firstError {
                replayDetailEvidenceByReplayKey[key] = V2ReplayDetailEvidenceState(
                    replayKey: key,
                    phase: .failed,
                    summary: mappedReplayDetailProblemSummary(firstError),
                    whatCameIn: whatCameIn,
                    whatAssistantUnderstood: whatUnderstood,
                    whatHappened: whatHappened,
                    approvalContext: approvalContext,
                    sourceContextFields: sourceContextFields,
                    decisionTrace: decisionTrace,
                    channelsTouched: channelsTouched,
                    confidenceScore: confidenceScore,
                    failureHint: failureHint,
                    lastUpdatedAt: Date()
                )
            } else {
                replayDetailEvidenceByReplayKey[key] = V2ReplayDetailEvidenceState(
                    replayKey: key,
                    phase: .empty,
                    summary: "No additional daemon evidence found for this replay item.",
                    whatCameIn: whatCameIn,
                    whatAssistantUnderstood: whatUnderstood,
                    whatHappened: whatHappened,
                    approvalContext: approvalContext,
                    sourceContextFields: sourceContextFields,
                    decisionTrace: decisionTrace,
                    channelsTouched: channelsTouched,
                    confidenceScore: confidenceScore,
                    failureHint: failureHint,
                    lastUpdatedAt: Date()
                )
            }
            return
        }

        replayDetailEvidenceByReplayKey[key] = V2ReplayDetailEvidenceState(
            replayKey: key,
            phase: .ready,
            summary: "Replay evidence refreshed.",
            whatCameIn: whatCameIn,
            whatAssistantUnderstood: whatUnderstood,
            whatHappened: whatHappened,
            approvalContext: approvalContext,
            sourceContextFields: sourceContextFields,
            decisionTrace: decisionTrace,
            channelsTouched: channelsTouched,
            confidenceScore: confidenceScore,
            failureHint: failureHint,
            lastUpdatedAt: Date()
        )
    }

    private func replayDetailTraceStatus(_ rawStatus: String) -> ReplayDecisionStageStatus {
        let normalized = rawStatus.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        if normalized.contains("complete") || normalized.contains("success") {
            return .completed
        }
        if normalized.contains("fail") || normalized.contains("error") || normalized.contains("blocked") || normalized.contains("cancel") {
            return .blocked
        }
        return .pending
    }

    private func replayDetailStageTitle(_ rawEventType: String) -> String {
        let cleaned = rawEventType
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .replacingOccurrences(of: "_", with: " ")
            .replacingOccurrences(of: ".", with: " ")
        return cleaned
            .split(separator: " ")
            .map { token in token.prefix(1).uppercased() + token.dropFirst().lowercased() }
            .joined(separator: " ")
    }

    private func appendReplayDetailSourceField(
        label: String,
        value: String?,
        into fields: inout [ReplaySourceContextField],
        didEnhance: inout Bool
    ) {
        guard let normalized = nonEmpty(value) else {
            return
        }
        if fields.contains(where: { $0.label == label && $0.value == normalized }) {
            return
        }
        fields.append(ReplaySourceContextField(label: label, value: normalized))
        didEnhance = true
    }

    private func replayDetailConfidence(from value: V2DaemonJSONValue?) -> Int? {
        guard let value else {
            return nil
        }
        switch value {
        case .number(let number):
            return Int(max(0, min(100, number.rounded())))
        case .string(let raw):
            let trimmed = raw.trimmingCharacters(in: .whitespacesAndNewlines)
            guard let parsed = Int(trimmed) else {
                return nil
            }
            return max(0, min(100, parsed))
        default:
            return nil
        }
    }

    private func parseReplayDetailDate(_ rawValue: String) -> Date {
        let trimmed = rawValue.trimmingCharacters(in: .whitespacesAndNewlines)
        if trimmed.isEmpty {
            return Date.distantPast
        }

        let withFractional = ISO8601DateFormatter()
        withFractional.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        if let parsed = withFractional.date(from: trimmed) {
            return parsed
        }

        let withoutFractional = ISO8601DateFormatter()
        withoutFractional.formatOptions = [.withInternetDateTime]
        return withoutFractional.date(from: trimmed) ?? Date.distantPast
    }

    private func validatedReplayDetailBaseURL() -> URL? {
        let trimmedBaseURL = daemonBaseURL.trimmingCharacters(in: .whitespacesAndNewlines)
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

    private func resolvedReplayDetailToken() -> String? {
        guard let token = sessionConfigStore.resolvedAccessToken() else {
            return nil
        }
        let trimmed = token.trimmingCharacters(in: .whitespacesAndNewlines)
        return trimmed.isEmpty ? nil : trimmed
    }

    private func mappedReplayDetailProblemSummary(_ error: Error) -> String {
        V2DaemonProblemMapper.map(error: error, context: .replay).summary
    }

    private func captureReplayDetailRequest<T>(_ operation: @escaping () async throws -> T) async -> Result<T, Error> {
        do {
            return .success(try await operation())
        } catch {
            return .failure(error)
        }
    }

    private func nonEmpty(_ rawValue: String?) -> String? {
        guard let trimmed = rawValue?.trimmingCharacters(in: .whitespacesAndNewlines), !trimmed.isEmpty else {
            return nil
        }
        return trimmed
    }
}
