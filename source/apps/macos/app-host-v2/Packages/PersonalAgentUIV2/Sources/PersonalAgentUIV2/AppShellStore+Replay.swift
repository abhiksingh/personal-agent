import Foundation

@MainActor
extension AppShellV2Store {
    public func toggleSource(_ source: ReplaySource) {
        if selectedSources.contains(source) {
            selectedSources.remove(source)
        } else {
            selectedSources.insert(source)
        }
    }

    public func clearFilters() {
        selectedSources.removeAll()
        statusFilter = .needsApproval
        searchQuery = ""
    }

    public func selectEvent(_ eventID: ReplayEvent.ID?) {
        selectedEventID = eventID
    }

    public func sendAsk() {
        let lifecycle = mutationLifecycle(for: .askSend)
        if lifecycle.isDisabled {
            setFeedback(lifecycle.message ?? "Ask is unavailable.")
            selectedSection = .getStarted
            return
        }
        guard !lifecycle.isInFlight else {
            return
        }

        let trimmed = askDraft.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else {
            return
        }

        guard let context = replayMutationContext() else {
            failMutation(.askSend, message: "Ask is unavailable. Complete setup first.")
            setFeedback(sessionReadiness.setupSummary)
            return
        }

        let correlationID = "ask-\(UUID().uuidString.lowercased())"
        let newEvent = ReplayEvent(
            replayKey: "corr:\(correlationID)",
            source: .app,
            sourceContext: .app(
                AppReplaySourceContext(
                    workspace: workspaceID,
                    sessionID: "session-\(UUID().uuidString.prefix(6))",
                    messageID: "msg-\(UUID().uuidString.prefix(8))"
                )
            ),
            receivedAt: Date(),
            instruction: trimmed,
            interpretedIntent: "Question submitted for assistant analysis.",
            actionSummary: "Waiting for assistant response.",
            status: .running,
            risk: .low,
            channelsTouched: ["app"],
            decisionTrace: Self.trace(
                received: "Message received from App Chat.",
                intent: "Classified as a contextual replay follow-up.",
                planning: "Prepared daemon chat-turn request with replay context.",
                execution: "Submitting question to daemon.",
                executionStatus: .pending
            ),
            confidenceScore: 78,
            daemonLocator: ReplayEventDaemonLocator(
                correlationID: correlationID,
                channelID: "app"
            )
        )

        var updated = replayEvents
        updated.insert(newEvent, at: 0)
        replaceReplayEvents(updated)

        selectedEventID = newEvent.id
        selectedSources.removeAll()
        statusFilter = .running
        searchQuery = ""
        askDraft = ""
        startMutation(.askSend, message: "Sending question…")

        Task { [weak self] in
            await self?.submitAskTurn(
                prompt: trimmed,
                optimisticEventID: newEvent.id,
                fallbackDraft: trimmed,
                correlationID: correlationID,
                baseURL: context.baseURL,
                authToken: context.authToken
            )
        }
    }

    public func seedAskFromSelectedEvent() {
        guard let selectedEvent else {
            askDraft = "What did you optimize for in the last task?"
            return
        }
        askDraft = "Why did you choose this path for: \"\(selectedEvent.instruction)\"?"
    }

    public func seedAskFromSelectedEventIfEmpty() {
        guard askDraft.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else {
            return
        }
        seedAskFromSelectedEvent()
    }

    public func seedFollowUpQuestion() {
        if askDraft.isEmpty {
            askDraft = "What guardrails did you apply in your last action?"
        }
        setFeedback("Seeded a follow-up in Ask.")
    }

    private func submitAskTurn(
        prompt: String,
        optimisticEventID: ReplayEvent.ID,
        fallbackDraft: String,
        correlationID: String,
        baseURL: URL,
        authToken: String
    ) async {
        do {
            let response = try await daemonClient.chat.chatTurn(
                baseURL: baseURL,
                authToken: authToken,
                workspaceID: workspaceID,
                messages: [("user", prompt)],
                requestedByActorID: principalActorID,
                subjectActorID: principalActorID,
                actingAsActorID: principalActorID,
                correlationID: correlationID
            )

            projectAskTurnResponse(response, optimisticEventID: optimisticEventID, fallbackPrompt: prompt, fallbackCorrelationID: correlationID)
            completeMutation(.askSend, message: "Question sent. Replay state reconciled.")
            setFeedback("Question sent. Replay state reconciled.")
            await refreshReplayAfterMutation()
        } catch {
            updateEvent(optimisticEventID) { draft in
                draft.status = .failed
                draft.actionSummary = "Question failed to send."
                draft.failureRecoveryHint = "Retry after resolving daemon connectivity or scope issues."
                draft.decisionTrace = Self.markTraceExecutionAsBlocked(draft.decisionTrace)
            }

            askDraft = fallbackDraft
            let summary = replayMutationErrorSummary(error)
            failMutation(.askSend, message: "Question failed. \(summary)")
            setFeedback("Question failed. \(summary)")
        }
    }

    private func projectAskTurnResponse(
        _ response: V2DaemonChatTurnResponse,
        optimisticEventID: ReplayEvent.ID,
        fallbackPrompt: String,
        fallbackCorrelationID: String
    ) {
        let correlationID = nonEmptyReplayMutationValue(response.correlationID) ?? fallbackCorrelationID
        let approvalItem = response.items.first(where: { item in
            let normalizedType = item.type.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
            return normalizedType == "approval_request" || nonEmptyReplayMutationValue(item.approvalRequestID) != nil
        })
        let assistantMessage = response.items.reversed().compactMap { item -> String? in
            let normalizedType = item.type.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
            guard normalizedType == "assistant_message" else {
                return nil
            }
            return nonEmptyReplayMutationValue(item.content)
        }.first
        let firstErrorSummary = response.items.compactMap { item -> String? in
            if let message = nonEmptyReplayMutationValue(item.errorMessage) {
                return message
            }
            return nonEmptyReplayMutationValue(item.errorCode)
        }.first

        let status = askReplayStatus(from: response, hasApprovalRequest: approvalItem != nil, hasError: firstErrorSummary != nil)
        let interpretedIntent = askIntentSummary(from: response)
        let actionSummary = askActionSummary(
            status: status,
            assistantMessage: assistantMessage,
            errorSummary: firstErrorSummary
        )
        let approvalReason = approvalItem.flatMap { nonEmptyReplayMutationValue($0.content) }
            ?? (status == .awaitingApproval ? "Approval is required before this question can execute." : nil)
        let channelsTouched = askChannelsTouched(from: response, includesApproval: status == .awaitingApproval)
        let confidenceScore = askConfidenceScore(from: response.items) ?? 80
        let decisionTrace = askDecisionTrace(from: response.items, status: status)
        let taskID = nonEmptyReplayMutationValue(response.taskRunCorrelation.taskID)
        let runID = nonEmptyReplayMutationValue(response.taskRunCorrelation.runID)

        updateEvent(optimisticEventID) { draft in
            draft.replayKey = "corr:\(correlationID.lowercased())"
            draft.receivedAt = Date()
            draft.instruction = fallbackPrompt
            draft.interpretedIntent = interpretedIntent
            draft.actionSummary = actionSummary
            draft.status = status
            draft.risk = status == .awaitingApproval ? .medium : .low
            draft.approvalReason = approvalReason
            draft.channelsTouched = channelsTouched
            draft.decisionTrace = decisionTrace
            draft.confidenceScore = confidenceScore
            draft.failureRecoveryHint = status == .failed ? "Retry this question or inspect replay evidence for failure details." : nil
            draft.daemonLocator = ReplayEventDaemonLocator(
                correlationID: correlationID,
                approvalRequestID: nonEmptyReplayMutationValue(approvalItem?.approvalRequestID),
                taskID: taskID,
                runID: runID,
                channelID: nonEmptyReplayMutationValue(response.channel?.channelID) ?? "app"
            )
        }
    }

    private func askReplayStatus(from response: V2DaemonChatTurnResponse, hasApprovalRequest: Bool, hasError: Bool) -> ReplayEventStatus {
        if hasApprovalRequest {
            return .awaitingApproval
        }
        if hasError {
            return .failed
        }

        let runState = nonEmptyReplayMutationValue(response.taskRunCorrelation.runState)?.lowercased() ?? ""
        let taskState = nonEmptyReplayMutationValue(response.taskRunCorrelation.taskState)?.lowercased() ?? ""
        let combinedState = "\(runState) \(taskState)"
        if combinedState.contains("fail") || combinedState.contains("error") || combinedState.contains("cancel") || combinedState.contains("blocked") {
            return .failed
        }
        if combinedState.contains("running") || combinedState.contains("queue") || combinedState.contains("pending") || combinedState.contains("progress") {
            return .running
        }
        if response.items.contains(where: { item in
            let itemStatus = item.status?.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() ?? ""
            return itemStatus.contains("pending") || itemStatus.contains("running") || itemStatus.contains("progress")
        }) {
            return .running
        }
        return .completed
    }

    private func askIntentSummary(from response: V2DaemonChatTurnResponse) -> String {
        askIntentSummary(from: response.items)
    }

    private func askIntentSummary(from items: [V2DaemonChatTurnItem]) -> String {
        if let taskClass = items.compactMap({ nonEmptyReplayMutationValue($0.metadata?.taskClass) }).first {
            return "Interpreted as \(taskClass) workflow."
        }
        return "Question submitted for assistant analysis."
    }

    private func askActionSummary(
        status: ReplayEventStatus,
        assistantMessage: String?,
        errorSummary: String?
    ) -> String {
        switch status {
        case .awaitingApproval:
            return "Awaiting approval before execution can continue."
        case .failed:
            return errorSummary ?? "Assistant reported a failure while processing this question."
        case .running:
            return "Assistant response is still in progress."
        case .completed:
            return assistantMessage ?? "Assistant response received."
        }
    }

    private func askChannelsTouched(from response: V2DaemonChatTurnResponse, includesApproval: Bool) -> [String] {
        var channels = Set<String>(["app"])

        if let channelID = nonEmptyReplayMutationValue(response.channel?.channelID) {
            channels.insert(channelID)
        }
        if let connectorID = nonEmptyReplayMutationValue(response.channel?.connectorID) {
            channels.insert(connectorID)
        }
        if nonEmptyReplayMutationValue(response.taskRunCorrelation.taskID) != nil {
            channels.insert("tasks")
        }
        if includesApproval {
            channels.insert("approvals")
        }

        return channels.sorted()
    }

    private func askConfidenceScore(from items: [V2DaemonChatTurnItem]) -> Int? {
        for item in items {
            guard let rawValue = item.metadata?.additional["confidence"] else {
                continue
            }
            switch rawValue {
            case .number(let number):
                return max(0, min(100, Int(number.rounded())))
            case .string(let text):
                if let parsed = Int(text.trimmingCharacters(in: .whitespacesAndNewlines)) {
                    return max(0, min(100, parsed))
                }
            default:
                continue
            }
        }
        return nil
    }

    private func askDecisionTrace(from items: [V2DaemonChatTurnItem], status: ReplayEventStatus) -> [ReplayDecisionStage] {
        let toolNames = items.compactMap { nonEmptyReplayMutationValue($0.toolName) }
        let planningDetail: String
        if toolNames.isEmpty {
            planningDetail = "Assistant prepared a direct response."
        } else {
            planningDetail = "Planned tool path: \(toolNames.joined(separator: ", "))."
        }

        let executionDetail: String
        if let errorMessage = items.compactMap({ nonEmptyReplayMutationValue($0.errorMessage) ?? nonEmptyReplayMutationValue($0.errorCode) }).first,
           status == .failed {
            executionDetail = errorMessage
        } else {
            executionDetail = askActionSummary(status: status, assistantMessage: nil, errorSummary: nil)
        }

        return Self.trace(
            received: "Question captured from Ask composer.",
            intent: askIntentSummary(from: items),
            planning: planningDetail,
            execution: executionDetail,
            executionStatus: askTraceStatus(for: status)
        )
    }

    private func askTraceStatus(for status: ReplayEventStatus) -> ReplayDecisionStageStatus {
        switch status {
        case .completed:
            return .completed
        case .failed:
            return .blocked
        case .awaitingApproval, .running:
            return .pending
        }
    }

    public func approveSelectedEvent() {
        let lifecycle = mutationLifecycle(for: .replayApprove)
        if lifecycle.isDisabled {
            setFeedback(lifecycle.message ?? "Approve is unavailable.")
            selectedSection = .getStarted
            return
        }
        guard !lifecycle.isInFlight else { return }
        guard let event = selectedEvent, event.status == .awaitingApproval else {
            return
        }
        guard let approvalRequestID = nonEmptyReplayMutationValue(event.daemonLocator?.approvalRequestID) else {
            setFeedback("Approval request ID is missing. Refresh replay evidence and try again.")
            return
        }

        let snapshot = event
        startMutation(.replayApprove, message: "Approving replay action…")

        updateEvent(event.id) { draft in
            draft.status = .running
            draft.actionSummary = "Approval submitted. Waiting for daemon confirmation."
            draft.approvalAudit.insert(
                ApprovalAuditEntry(
                    decidedAt: Date(),
                    action: .approve,
                    actor: "You",
                    note: "Approved from Replay detail (pending daemon confirmation)"
                ),
                at: 0
            )
        }

        Task { [weak self] in
            await self?.submitReplayApprovalDecision(
                actionID: .replayApprove,
                eventID: event.id,
                snapshot: snapshot,
                approvalRequestID: approvalRequestID,
                decision: "approve",
                successMessage: "Approval submitted. Replay state reconciled.",
                rollbackMessage: "Approval failed."
            )
        }
    }

    public func rejectSelectedEvent() {
        let lifecycle = mutationLifecycle(for: .replayReject)
        if lifecycle.isDisabled {
            setFeedback(lifecycle.message ?? "Reject is unavailable.")
            selectedSection = .getStarted
            return
        }
        guard !lifecycle.isInFlight else { return }
        guard let event = selectedEvent, event.status == .awaitingApproval else {
            return
        }
        guard let approvalRequestID = nonEmptyReplayMutationValue(event.daemonLocator?.approvalRequestID) else {
            setFeedback("Approval request ID is missing. Refresh replay evidence and try again.")
            return
        }

        let snapshot = event
        startMutation(.replayReject, message: "Rejecting replay action…")

        updateEvent(event.id) { draft in
            draft.approvalAudit.insert(
                ApprovalAuditEntry(
                    decidedAt: Date(),
                    action: .reject,
                    actor: "You",
                    note: "Rejected from Replay detail (pending daemon confirmation)"
                ),
                at: 0
            )
            draft.status = .failed
            draft.actionSummary = "Rejection submitted. Waiting for daemon confirmation."
            draft.failureRecoveryHint = "Update instruction details and resubmit for approval."
            draft.decisionTrace = Self.markTraceExecutionAsBlocked(draft.decisionTrace)
        }

        Task { [weak self] in
            await self?.submitReplayApprovalDecision(
                actionID: .replayReject,
                eventID: event.id,
                snapshot: snapshot,
                approvalRequestID: approvalRequestID,
                decision: "reject",
                successMessage: "Rejection submitted. Replay state reconciled.",
                rollbackMessage: "Rejection failed."
            )
        }
    }

    public func retrySelectedEvent() {
        let lifecycle = mutationLifecycle(for: .replayRetry)
        if lifecycle.isDisabled {
            setFeedback(lifecycle.message ?? "Retry is unavailable.")
            selectedSection = .getStarted
            return
        }
        guard !lifecycle.isInFlight else { return }
        guard let event = selectedEvent, event.status == .failed else {
            return
        }
        guard let taskID = nonEmptyReplayMutationValue(event.daemonLocator?.taskID) else {
            setFeedback("Task reference is missing for retry. Refresh replay evidence and try again.")
            return
        }
        let runID = nonEmptyReplayMutationValue(event.daemonLocator?.runID)
        let snapshot = event

        startMutation(.replayRetry, message: "Retrying replay action…")

        updateEvent(event.id) { draft in
            draft.status = .running
            draft.actionSummary = "Retry submitted. Waiting for daemon confirmation."
            draft.failureRecoveryHint = nil
            draft.decisionTrace = Self.resetBlockedTraceToPending(draft.decisionTrace)
        }

        Task { [weak self] in
            await self?.submitReplayTaskRetry(
                eventID: event.id,
                snapshot: snapshot,
                taskID: taskID,
                runID: runID
            )
        }
    }

    public func completeSelectedRunningEvent() {
        let lifecycle = mutationLifecycle(for: .replayComplete)
        if lifecycle.isDisabled {
            setFeedback(lifecycle.message ?? "Complete action is unavailable.")
            selectedSection = .getStarted
            return
        }
        guard !lifecycle.isInFlight else { return }
        guard let event = selectedEvent, event.status == .running else {
            return
        }
        guard let taskID = nonEmptyReplayMutationValue(event.daemonLocator?.taskID) else {
            setFeedback("Task reference is missing for run control. Refresh replay evidence and try again.")
            return
        }
        let runID = nonEmptyReplayMutationValue(event.daemonLocator?.runID)
        let snapshot = event

        startMutation(.replayComplete, message: "Stopping replay run…")

        updateEvent(event.id) { draft in
            draft.actionSummary = "Stop request submitted. Waiting for daemon confirmation."
        }

        Task { [weak self] in
            await self?.submitReplayTaskStop(
                eventID: event.id,
                snapshot: snapshot,
                taskID: taskID,
                runID: runID
            )
        }
    }

    private func submitReplayApprovalDecision(
        actionID: V2MutationActionID,
        eventID: ReplayEvent.ID,
        snapshot: ReplayEvent,
        approvalRequestID: String,
        decision: String,
        successMessage: String,
        rollbackMessage: String
    ) async {
        guard let context = replayMutationContext() else {
            rollbackReplayEvent(eventID, snapshot: snapshot)
            failMutation(actionID, message: "Replay action is unavailable. Complete setup first.")
            return
        }

        do {
            _ = try await daemonClient.approvals.approvalDecision(
                baseURL: context.baseURL,
                authToken: context.authToken,
                workspaceID: workspaceID,
                approvalRequestID: approvalRequestID,
                decision: decision,
                decisionByActorID: principalActorID
            )
            await refreshReplayAfterMutation()
            completeMutation(actionID, message: successMessage)
            setFeedback(successMessage)
        } catch {
            rollbackReplayEvent(eventID, snapshot: snapshot)
            let summary = replayMutationErrorSummary(error)
            failMutation(actionID, message: "\(rollbackMessage) \(summary)")
            setFeedback("\(rollbackMessage) \(summary)")
        }
    }

    private func submitReplayTaskRetry(
        eventID: ReplayEvent.ID,
        snapshot: ReplayEvent,
        taskID: String,
        runID: String?
    ) async {
        guard let context = replayMutationContext() else {
            rollbackReplayEvent(eventID, snapshot: snapshot)
            failMutation(.replayRetry, message: "Replay retry is unavailable. Complete setup first.")
            return
        }

        do {
            _ = try await daemonClient.tasks.taskRetry(
                baseURL: context.baseURL,
                authToken: context.authToken,
                workspaceID: workspaceID,
                taskID: taskID,
                runID: runID,
                reason: "replay_inline_retry"
            )
            await refreshReplayAfterMutation()
            completeMutation(.replayRetry, message: "Retry submitted. Replay state reconciled.")
            setFeedback("Retry submitted. Replay state reconciled.")
        } catch {
            rollbackReplayEvent(eventID, snapshot: snapshot)
            let summary = replayMutationErrorSummary(error)
            failMutation(.replayRetry, message: "Retry failed. \(summary)")
            setFeedback("Retry failed. \(summary)")
        }
    }

    private func submitReplayTaskStop(
        eventID: ReplayEvent.ID,
        snapshot: ReplayEvent,
        taskID: String,
        runID: String?
    ) async {
        guard let context = replayMutationContext() else {
            rollbackReplayEvent(eventID, snapshot: snapshot)
            failMutation(.replayComplete, message: "Run control is unavailable. Complete setup first.")
            return
        }

        do {
            _ = try await daemonClient.tasks.taskCancel(
                baseURL: context.baseURL,
                authToken: context.authToken,
                workspaceID: workspaceID,
                taskID: taskID,
                runID: runID,
                reason: "replay_inline_stop"
            )
            await refreshReplayAfterMutation()
            completeMutation(.replayComplete, message: "Stop request submitted. Replay state reconciled.")
            setFeedback("Stop request submitted. Replay state reconciled.")
        } catch {
            rollbackReplayEvent(eventID, snapshot: snapshot)
            let summary = replayMutationErrorSummary(error)
            failMutation(.replayComplete, message: "Run stop failed. \(summary)")
            setFeedback("Run stop failed. \(summary)")
        }
    }

    private func refreshReplayAfterMutation() async {
        await refreshReplayFeed(resetPagination: true)
        await refreshSelectedReplayDetailEvidence(force: true)
    }

    private func rollbackReplayEvent(_ eventID: ReplayEvent.ID, snapshot: ReplayEvent) {
        updateEvent(eventID) { draft in
            draft = snapshot
        }
    }

    private func replayMutationContext() -> (baseURL: URL, authToken: String)? {
        guard sessionReadiness.isReadyForDaemonMutations else {
            selectedSection = .getStarted
            return nil
        }

        let trimmedBaseURL = daemonBaseURL.trimmingCharacters(in: .whitespacesAndNewlines)
        guard let baseURL = URL(string: trimmedBaseURL),
              !trimmedBaseURL.isEmpty,
              let components = URLComponents(url: baseURL, resolvingAgainstBaseURL: false),
              let scheme = components.scheme?.lowercased(),
              ["http", "https", "ws", "wss"].contains(scheme),
              components.host != nil else {
            selectedSection = .getStarted
            return nil
        }

        guard let authToken = sessionConfigStore.resolvedAccessToken()?.trimmingCharacters(in: .whitespacesAndNewlines),
              !authToken.isEmpty else {
            selectedSection = .getStarted
            return nil
        }

        return (baseURL, authToken)
    }

    private func replayMutationErrorSummary(_ error: Error) -> String {
        V2DaemonProblemMapper.map(error: error, context: .replay).summary
    }

    private func nonEmptyReplayMutationValue(_ rawValue: String?) -> String? {
        guard let trimmed = rawValue?.trimmingCharacters(in: .whitespacesAndNewlines), !trimmed.isEmpty else {
            return nil
        }
        return trimmed
    }

    public func dismissFeedback() {
        clearFeedback()
    }
}
