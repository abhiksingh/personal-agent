import Foundation

@MainActor
extension AppShellV2Store {
    public func refreshReplayFeedIfNeeded(force: Bool = false) async {
        if replayFeedQueryState.isRefreshing || replayFeedQueryState.isLoadingMore {
            return
        }
        if !force, replayFeedQueryState.lastLoadedAt != nil {
            return
        }
        await refreshReplayFeed(resetPagination: true)
    }

    public func refreshReplayFeed(resetPagination: Bool = true) async {
        if replayFeedQueryState.isRefreshing || replayFeedQueryState.isLoadingMore {
            return
        }
        let page = resetPagination ? 1 : replayFeedQueryState.requestedPage
        await fetchReplayFeed(requestedPage: page, loadingMore: false)
    }

    public func loadMoreReplayFeed() async {
        if replayFeedQueryState.isRefreshing || replayFeedQueryState.isLoadingMore {
            return
        }
        guard replayFeedQueryState.canLoadMore else {
            return
        }
        await fetchReplayFeed(requestedPage: replayFeedQueryState.requestedPage + 1, loadingMore: true)
    }

    private func fetchReplayFeed(requestedPage: Int, loadingMore: Bool) async {
        if !sessionReadiness.isReadyForDaemonMutations {
            replayFeedQueryState = V2ReplayFeedQueryState(
                pageSize: replayFeedQueryState.pageSize,
                requestedPage: 1,
                hasLoadedOnce: false,
                isRefreshing: false,
                isLoadingMore: false,
                canLoadMore: false,
                lastLoadedAt: replayFeedQueryState.lastLoadedAt,
                lastErrorSummary: nil
            )
            return
        }

        guard let baseURL = validatedReplayDaemonBaseURL() else {
            let error = V2DaemonAPIError.invalidBaseURL
            setPanelProblem(error, context: .replay)
            replayFeedQueryState = V2ReplayFeedQueryState(
                pageSize: replayFeedQueryState.pageSize,
                requestedPage: 1,
                hasLoadedOnce: replayFeedQueryState.hasLoadedOnce,
                isRefreshing: false,
                isLoadingMore: false,
                canLoadMore: false,
                lastLoadedAt: replayFeedQueryState.lastLoadedAt,
                lastErrorSummary: mappedReplayProblemSummary(error)
            )
            return
        }

        guard let authToken = resolvedReplayAuthToken() else {
            let error = V2DaemonAPIError.missingAuthToken
            setPanelProblem(error, context: .replay)
            replayFeedQueryState = V2ReplayFeedQueryState(
                pageSize: replayFeedQueryState.pageSize,
                requestedPage: 1,
                hasLoadedOnce: replayFeedQueryState.hasLoadedOnce,
                isRefreshing: false,
                isLoadingMore: false,
                canLoadMore: false,
                lastLoadedAt: replayFeedQueryState.lastLoadedAt,
                lastErrorSummary: mappedReplayProblemSummary(error)
            )
            return
        }

        var pendingState = replayFeedQueryState
        pendingState.requestedPage = max(1, requestedPage)
        pendingState.isRefreshing = !loadingMore
        pendingState.isLoadingMore = loadingMore
        pendingState.lastErrorSummary = nil
        replayFeedQueryState = pendingState
        let approvalLimit = pendingState.approvalLimit
        let taskLimit = pendingState.taskLimit
        let historyLimit = pendingState.historyLimit

        async let approvalsResult: Result<V2DaemonApprovalInboxResponse, Error> = captureReplayRequest { [self] in
            try await self.daemonClient.approvals.approvalInbox(
                baseURL: baseURL,
                authToken: authToken,
                workspaceID: self.workspaceID,
                limit: approvalLimit
            )
        }
        async let tasksResult: Result<V2DaemonTaskRunListResponse, Error> = captureReplayRequest { [self] in
            try await self.daemonClient.tasks.taskRunList(
                baseURL: baseURL,
                authToken: authToken,
                workspaceID: self.workspaceID,
                limit: taskLimit
            )
        }
        async let historyResult: Result<V2DaemonChatTurnHistoryResponse, Error> = captureReplayRequest { [self] in
            try await self.daemonClient.chat.chatTurnHistory(
                baseURL: baseURL,
                authToken: authToken,
                workspaceID: self.workspaceID,
                limit: historyLimit
            )
        }

        let approvalsResolved = await approvalsResult
        let tasksResolved = await tasksResult
        let historyResolved = await historyResult

        var firstError: Error?
        let approvals: [V2DaemonApprovalInboxRecord]
        switch approvalsResolved {
        case .success(let response):
            approvals = response.approvals
        case .failure(let error):
            approvals = []
            firstError = firstError ?? error
        }

        let tasks: [V2DaemonTaskRunListRecord]
        switch tasksResolved {
        case .success(let response):
            tasks = response.items
        case .failure(let error):
            tasks = []
            firstError = firstError ?? error
        }

        let historyItems: [V2DaemonChatTurnHistoryRecord]
        var historyHasMore = false
        switch historyResolved {
        case .success(let response):
            historyItems = response.items
            historyHasMore = response.hasMore
        case .failure(let error):
            historyItems = []
            firstError = firstError ?? error
        }

        if approvals.isEmpty && tasks.isEmpty && historyItems.isEmpty, let firstError {
            setPanelProblem(firstError, context: .replay)
            var failedState = pendingState
            failedState.isRefreshing = false
            failedState.isLoadingMore = false
            failedState.lastErrorSummary = mappedReplayProblemSummary(firstError)
            replayFeedQueryState = failedState
            return
        }

        clearPanelProblem(for: .replay)

        let projected = projectReplayEvents(
            approvals: approvals,
            tasks: tasks,
            historyItems: historyItems,
            workspaceID: workspaceID
        )
        replaceReplayEvents(projected.events)

        let approvalsHasMoreHint = approvals.count >= approvalLimit
        let tasksHasMoreHint = tasks.count >= taskLimit
        let historyHasMoreHint = historyHasMore || historyItems.count >= historyLimit

        replayFeedQueryState = V2ReplayFeedQueryState(
            pageSize: pendingState.pageSize,
            requestedPage: pendingState.requestedPage,
            hasLoadedOnce: true,
            isRefreshing: false,
            isLoadingMore: false,
            canLoadMore: approvalsHasMoreHint || tasksHasMoreHint || historyHasMoreHint,
            lastLoadedAt: Date(),
            lastErrorSummary: nil
        )
        startReplayRealtimeIfNeeded()
    }

    private func projectReplayEvents(
        approvals: [V2DaemonApprovalInboxRecord],
        tasks: [V2DaemonTaskRunListRecord],
        historyItems: [V2DaemonChatTurnHistoryRecord],
        workspaceID: String
    ) -> V2ReplayProjectionResult {
        var events: [ReplayEvent] = []
        var usedApprovalRequestIDs: Set<String> = []
        var usedTaskRunKeys: Set<String> = []

        let approvalsByID = Dictionary(uniqueKeysWithValues: approvals.map { ($0.approvalRequestID, $0) })
        let tasksByRunKey = Dictionary(uniqueKeysWithValues: tasks.map { record in
            let runComponent = record.runID?.trimmingCharacters(in: .whitespacesAndNewlines)
            let runKey = runComponent.flatMap { $0.isEmpty ? nil : $0 } ?? "task:\(record.taskID)"
            return (runKey.lowercased(), record)
        })

        let groupedHistory = Dictionary(grouping: historyItems) { record in
            if let correlation = nonEmpty(record.correlationID) {
                return "corr:\(correlation.lowercased())"
            }
            if let turn = nonEmpty(record.turnID) {
                return "turn:\(turn.lowercased())"
            }
            return "history:\(record.recordID.lowercased())"
        }

        for (groupKey, groupRecords) in groupedHistory {
            let sortedRecords = groupRecords.sorted { lhs, rhs in
                if lhs.itemIndex != rhs.itemIndex {
                    return lhs.itemIndex < rhs.itemIndex
                }
                return parseReplayDate(lhs.createdAt) < parseReplayDate(rhs.createdAt)
            }

            let firstRecord = sortedRecords.first!
            let lastRecord = sortedRecords.last!
            let userItem = sortedRecords.first(where: { record in
                let itemType = record.item.type.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
                let role = record.item.role?.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
                return itemType == "user_message" || role == "user"
            })
            let pendingApprovalItem = sortedRecords.first(where: { record in
                record.item.type.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() == "approval_request"
            })

            let correlationID = nonEmpty(firstRecord.correlationID)
            let turnID = nonEmpty(firstRecord.turnID)
            let runID = sortedRecords.compactMap { nonEmpty($0.taskRunReference.runID) }.first
            let taskID = sortedRecords.compactMap { nonEmpty($0.taskRunReference.taskID) }.first
            let approvalRequestID = pendingApprovalItem?.item.approvalRequestID.flatMap(nonEmpty)

            if let approvalRequestID {
                usedApprovalRequestIDs.insert(approvalRequestID)
            }
            if let runID {
                usedTaskRunKeys.insert(runID.lowercased())
            } else if let taskID {
                usedTaskRunKeys.insert("task:\(taskID.lowercased())")
            }

            let taskRecord = runID.flatMap { tasksByRunKey[$0.lowercased()] }
                ?? taskID.flatMap { tasksByRunKey["task:\($0.lowercased())"] }
            let approvalRecord = approvalRequestID.flatMap { approvalsByID[$0] }

            let instruction = nonEmpty(userItem?.item.content)
                ?? nonEmpty(lastRecord.item.content)
                ?? nonEmpty(taskRecord?.title)
                ?? nonEmpty(approvalRecord?.requestedPhrase)
                ?? "Instruction received"

            let source = replaySource(
                channelID: nonEmpty(firstRecord.channelID),
                connectorHint: connectorHint(from: sortedRecords)
            )

            let replayStatus = replayStatusForHistoryGroup(
                records: sortedRecords,
                taskRecord: taskRecord,
                approvalRecord: approvalRecord
            )

            let risk = replayRisk(for: approvalRecord?.riskLevel, fallbackStatus: replayStatus)
            let approvalReason = nonEmpty(approvalRecord?.riskRationale)
                ?? nonEmpty(pendingApprovalItem?.item.content)
            let actionSummary = replayActionSummary(
                latestRecord: lastRecord,
                status: replayStatus,
                taskRecord: taskRecord,
                approvalRecord: approvalRecord
            )

            let channelsTouched = replayChannelsTouched(
                channelID: nonEmpty(firstRecord.channelID),
                connectorHint: connectorHint(from: sortedRecords),
                taskRecord: taskRecord,
                approvalRecord: approvalRecord
            )

            let interpretedIntent = replayIntentSummary(taskRecord: taskRecord, approvalRecord: approvalRecord, records: sortedRecords)
            let receivedAt = parseReplayDate(lastRecord.createdAt)
            let replayKey = stableReplayKey(
                preferred: correlationID,
                fallbackTurnID: turnID,
                fallbackApprovalID: approvalRequestID,
                fallbackRunID: runID,
                fallbackTaskID: taskID,
                fallbackGroupKey: groupKey
            )
            let locator = ReplayEventDaemonLocator(
                correlationID: correlationID,
                turnID: turnID,
                historyRecordIDs: sortedRecords.map(\.recordID),
                approvalRequestID: approvalRequestID,
                taskID: taskID,
                runID: runID,
                channelID: nonEmpty(firstRecord.channelID)
            )

            events.append(
                ReplayEvent(
                    id: stableReplayEventID(for: replayKey),
                    replayKey: replayKey,
                    source: source,
                    sourceContext: sourceContext(
                        source: source,
                        workspaceID: workspaceID,
                        channelID: nonEmpty(firstRecord.channelID),
                        correlationID: correlationID,
                        turnID: turnID,
                        connectorHint: connectorHint(from: sortedRecords)
                    ),
                    receivedAt: receivedAt,
                    instruction: instruction,
                    interpretedIntent: interpretedIntent,
                    actionSummary: actionSummary,
                    status: replayStatus,
                    risk: risk,
                    approvalReason: approvalReason,
                    channelsTouched: channelsTouched,
                    decisionTrace: replayDecisionTrace(
                        instruction: instruction,
                        interpretedIntent: interpretedIntent,
                        records: sortedRecords,
                        status: replayStatus
                    ),
                    confidenceScore: replayConfidence(records: sortedRecords),
                    failureRecoveryHint: replayFailureHint(status: replayStatus, taskRecord: taskRecord),
                    daemonLocator: locator
                )
            )
        }

        for approval in approvals where !usedApprovalRequestIDs.contains(approval.approvalRequestID) {
            let source: ReplaySource = .app
            let status = replayStatusForApproval(approval)
            let replayKey = "approval:\(approval.approvalRequestID.lowercased())"
            let instruction = nonEmpty(approval.requestedPhrase)
                ?? nonEmpty(approval.taskTitle)
                ?? "Approval request"
            events.append(
                ReplayEvent(
                    id: stableReplayEventID(for: replayKey),
                    replayKey: replayKey,
                    source: source,
                    sourceContext: sourceContext(
                        source: source,
                        workspaceID: workspaceID,
                        channelID: nil,
                        correlationID: nil,
                        turnID: nil,
                        connectorHint: nil
                    ),
                    receivedAt: parseReplayDate(approval.requestedAt),
                    instruction: instruction,
                    interpretedIntent: approval.route?.notes ?? "Decision required before execution.",
                    actionSummary: status == .awaitingApproval
                        ? "Waiting for approval decision."
                        : (status == .failed ? "Request was rejected." : "Approval completed."),
                    status: status,
                    risk: replayRisk(for: approval.riskLevel, fallbackStatus: status),
                    approvalReason: nonEmpty(approval.riskRationale),
                    channelsTouched: ["approvals"],
                    decisionTrace: Self.trace(
                        received: "Approval request captured.",
                        intent: "Classified as approval-gated operation.",
                        planning: "Prepared request for decision and audit trail.",
                        execution: status == .awaitingApproval ? "Waiting for decision." : "Decision recorded.",
                        executionStatus: traceStatus(for: status)
                    ),
                    confidenceScore: 82,
                    failureRecoveryHint: status == .failed ? "Update instruction context and retry." : nil,
                    daemonLocator: ReplayEventDaemonLocator(
                        approvalRequestID: approval.approvalRequestID,
                        taskID: nonEmpty(approval.taskID),
                        runID: nonEmpty(approval.runID)
                    )
                )
            )
        }

        for task in tasks {
            let key = nonEmpty(task.runID)?.lowercased() ?? "task:\(task.taskID.lowercased())"
            if usedTaskRunKeys.contains(key) {
                continue
            }

            let status = replayStatusForTask(task)
            let replayKey = "task:\(key)"
            events.append(
                ReplayEvent(
                    id: stableReplayEventID(for: replayKey),
                    replayKey: replayKey,
                    source: .app,
                    sourceContext: sourceContext(
                        source: .app,
                        workspaceID: workspaceID,
                        channelID: nil,
                        correlationID: nil,
                        turnID: nil,
                        connectorHint: nil
                    ),
                    receivedAt: parseReplayDate(task.taskUpdatedAt),
                    instruction: nonEmpty(task.title) ?? "Task execution",
                    interpretedIntent: task.route?.notes ?? "Execute queued assistant task.",
                    actionSummary: replayTaskSummary(task),
                    status: status,
                    risk: status == .awaitingApproval ? .medium : .low,
                    channelsTouched: ["tasks"],
                    decisionTrace: Self.trace(
                        received: "Task request queued for execution.",
                        intent: "Mapped into persisted run pipeline.",
                        planning: "Task run prepared with route policy checks.",
                        execution: replayTaskSummary(task),
                        executionStatus: traceStatus(for: status)
                    ),
                    confidenceScore: 80,
                    failureRecoveryHint: status == .failed ? "Use Retry after validating route and connector health." : nil,
                    daemonLocator: ReplayEventDaemonLocator(
                        taskID: task.taskID,
                        runID: nonEmpty(task.runID)
                    )
                )
            )
        }

        events.sort(by: { $0.receivedAt > $1.receivedAt })
        return V2ReplayProjectionResult(events: events)
    }

    private func replaySource(channelID: String?, connectorHint: String?) -> ReplaySource {
        let normalizedChannel = channelID?.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() ?? ""
        let normalizedConnector = connectorHint?.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() ?? ""

        if normalizedChannel == "voice" {
            return .voice
        }
        if normalizedConnector.contains("whatsapp") {
            return .whatsapp
        }
        if normalizedConnector.contains("telegram") {
            return .telegram
        }
        if normalizedConnector.contains("mail") || normalizedConnector.contains("email") {
            return .email
        }
        if normalizedConnector.contains("imessage") {
            return .iMessage
        }
        if normalizedChannel == "app" {
            return .app
        }
        if normalizedChannel == "message" {
            return .iMessage
        }
        return .app
    }

    private func replayStatusForHistoryGroup(
        records: [V2DaemonChatTurnHistoryRecord],
        taskRecord: V2DaemonTaskRunListRecord?,
        approvalRecord: V2DaemonApprovalInboxRecord?
    ) -> ReplayEventStatus {
        if let approvalRecord, replayStatusForApproval(approvalRecord) == .awaitingApproval {
            return .awaitingApproval
        }

        if records.contains(where: { record in
            let itemType = record.item.type.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
            return itemType == "approval_request"
                && record.item.status?.lowercased().contains("pending") != false
        }) {
            return .awaitingApproval
        }

        if let taskRecord {
            let status = replayStatusForTask(taskRecord)
            if status == .failed || status == .running || status == .awaitingApproval {
                return status
            }
        }

        if records.contains(where: { record in
            let status = record.item.status?.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() ?? ""
            if status.contains("fail") || status.contains("error") || status.contains("cancel") || status.contains("blocked") {
                return true
            }
            return record.item.errorCode != nil || record.item.errorMessage != nil
        }) {
            return .failed
        }

        if records.contains(where: { record in
            let status = record.item.status?.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() ?? ""
            return status.contains("pending") || status.contains("running") || status.contains("progress")
        }) {
            return .running
        }

        return .completed
    }

    private func replayStatusForApproval(_ approval: V2DaemonApprovalInboxRecord) -> ReplayEventStatus {
        let state = approval.state.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        let decision = approval.decision?.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        if state == "pending" || state.contains("await") {
            return .awaitingApproval
        }
        if decision == "reject" || state.contains("reject") || state.contains("denied") {
            return .failed
        }
        if state.contains("fail") || state.contains("error") {
            return .failed
        }
        return .completed
    }

    private func replayStatusForTask(_ task: V2DaemonTaskRunListRecord) -> ReplayEventStatus {
        let taskState = task.taskState.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        let runState = task.runState?.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() ?? ""
        let state = [runState, taskState].joined(separator: " ")

        if state.contains("approval") || state.contains("await") {
            return .awaitingApproval
        }
        if state.contains("fail") || state.contains("error") || state.contains("cancel") || state.contains("blocked") {
            return .failed
        }
        if state.contains("running") || state.contains("queue") || state.contains("pending") || state.contains("progress") {
            return .running
        }
        return .completed
    }

    private func replayRisk(for rawRisk: String?, fallbackStatus: ReplayEventStatus) -> ReplayRiskLevel {
        let risk = rawRisk?.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() ?? ""
        if risk == "high" {
            return .high
        }
        if risk == "medium" {
            return .medium
        }
        if risk == "low" {
            return .low
        }
        return fallbackStatus == .awaitingApproval ? .medium : .low
    }

    private func replayActionSummary(
        latestRecord: V2DaemonChatTurnHistoryRecord,
        status: ReplayEventStatus,
        taskRecord: V2DaemonTaskRunListRecord?,
        approvalRecord: V2DaemonApprovalInboxRecord?
    ) -> String {
        if let approvalRecord, status == .awaitingApproval {
            return nonEmpty(approvalRecord.riskRationale) ?? "Awaiting approval decision."
        }

        if let output = nonEmpty(latestRecord.item.content) {
            return output
        }

        if let taskRecord {
            return replayTaskSummary(taskRecord)
        }

        switch status {
        case .awaitingApproval:
            return "Awaiting approval decision."
        case .running:
            return "Execution in progress."
        case .failed:
            return "Execution failed."
        case .completed:
            return "Execution completed."
        }
    }

    private func replayTaskSummary(_ task: V2DaemonTaskRunListRecord) -> String {
        if let error = nonEmpty(task.lastError) {
            return error
        }
        let runState = task.runState?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        if !runState.isEmpty {
            return "Run status: \(runState)."
        }
        return "Task status: \(task.taskState)."
    }

    private func replayIntentSummary(
        taskRecord: V2DaemonTaskRunListRecord?,
        approvalRecord: V2DaemonApprovalInboxRecord?,
        records: [V2DaemonChatTurnHistoryRecord]
    ) -> String {
        if let notes = taskRecord?.route?.notes.flatMap(nonEmpty) {
            return notes
        }
        if let notes = approvalRecord?.route?.notes.flatMap(nonEmpty) {
            return notes
        }

        let taskClass = records
            .compactMap { $0.item.metadata?.taskClass }
            .first?
            .trimmingCharacters(in: .whitespacesAndNewlines)
        if let taskClass, !taskClass.isEmpty {
            return "Interpreted as \(taskClass) workflow."
        }
        return "Assistant interpreted this instruction and selected an execution path."
    }

    private func replayDecisionTrace(
        instruction: String,
        interpretedIntent: String,
        records: [V2DaemonChatTurnHistoryRecord],
        status: ReplayEventStatus
    ) -> [ReplayDecisionStage] {
        let toolCallCount = records.filter { record in
            record.item.type.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() == "tool_call"
        }.count
        let planningDetail = toolCallCount > 0
            ? "Planned \(toolCallCount) tool step(s) for this instruction."
            : "No tool step required; assistant generated a direct response."

        let executionDetail: String
        switch status {
        case .awaitingApproval:
            executionDetail = "Execution paused for approval."
        case .running:
            executionDetail = "Execution is currently in progress."
        case .failed:
            executionDetail = "Execution failed; review failure hints before retrying."
        case .completed:
            executionDetail = "Execution completed and result was recorded."
        }

        return Self.trace(
            received: "Instruction captured: \(instruction)",
            intent: interpretedIntent,
            planning: planningDetail,
            execution: executionDetail,
            executionStatus: traceStatus(for: status)
        )
    }

    private func replayChannelsTouched(
        channelID: String?,
        connectorHint: String?,
        taskRecord: V2DaemonTaskRunListRecord?,
        approvalRecord: V2DaemonApprovalInboxRecord?
    ) -> [String] {
        var channels: [String] = []
        if let channelID = nonEmpty(channelID) {
            channels.append(channelID)
        }
        if let connectorHint = nonEmpty(connectorHint) {
            channels.append(connectorHint)
        }
        if let provider = nonEmpty(taskRecord?.route?.provider) {
            channels.append(provider)
        }
        if let provider = nonEmpty(approvalRecord?.route?.provider) {
            channels.append(provider)
        }
        if channels.isEmpty {
            channels = ["app"]
        }
        return Array(Set(channels)).sorted()
    }

    private func replayConfidence(records: [V2DaemonChatTurnHistoryRecord]) -> Int {
        let confidenceSamples: [Int] = records.compactMap { record in
            guard let rawValue = record.item.metadata?.additional["confidence"] else {
                return nil
            }
            switch rawValue {
            case .number(let value):
                return Int(max(0, min(100, value.rounded())))
            case .string(let value):
                return Int(value.trimmingCharacters(in: .whitespacesAndNewlines))
            default:
                return nil
            }
        }
        if let sample = confidenceSamples.first {
            return sample
        }
        return records.contains(where: { $0.item.type.lowercased() == "tool_result" }) ? 90 : 82
    }

    private func replayFailureHint(status: ReplayEventStatus, taskRecord: V2DaemonTaskRunListRecord?) -> String? {
        guard status == .failed else {
            return nil
        }
        if let error = nonEmpty(taskRecord?.lastError) {
            return "\(error) Retry after resolving the blocker."
        }
        return "Retry after checking connector health and route readiness."
    }

    private func sourceContext(
        source: ReplaySource,
        workspaceID: String,
        channelID: String?,
        correlationID: String?,
        turnID: String?,
        connectorHint: String?
    ) -> ReplaySourceContext {
        switch source {
        case .app:
            return .app(
                AppReplaySourceContext(
                    workspace: workspaceID,
                    sessionID: shortID(from: correlationID, fallbackPrefix: "session"),
                    messageID: shortID(from: turnID, fallbackPrefix: "turn")
                )
            )
        case .iMessage:
            return .iMessage(
                IMessageReplaySourceContext(
                    contactName: "Message Contact",
                    contactPhoneSuffix: "••••",
                    threadID: shortID(from: turnID, fallbackPrefix: "message-thread")
                )
            )
        case .whatsapp:
            return .whatsapp(
                WhatsAppReplaySourceContext(
                    contactName: "WhatsApp Contact",
                    chatID: shortID(from: channelID, fallbackPrefix: "wa-chat"),
                    phoneSuffix: "••••"
                )
            )
        case .telegram:
            return .telegram(
                TelegramReplaySourceContext(
                    handle: "@contact",
                    chatID: shortID(from: channelID, fallbackPrefix: "tg-chat"),
                    botID: shortID(from: connectorHint, fallbackPrefix: "tg-bot")
                )
            )
        case .email:
            return .email(
                EmailReplaySourceContext(
                    sender: "inbox@assistant",
                    subject: "Email instruction",
                    mailbox: workspaceID
                )
            )
        case .voice:
            return .voice(
                VoiceReplaySourceContext(
                    deviceName: "Voice Channel",
                    transcriptConfidence: 85,
                    utteranceDurationSeconds: 6
                )
            )
        }
    }

    private func connectorHint(from records: [V2DaemonChatTurnHistoryRecord]) -> String? {
        for record in records {
            if let connectorID = record.item.metadata?.additional["connector_id"]?.stringValue?.trimmingCharacters(in: .whitespacesAndNewlines), !connectorID.isEmpty {
                return connectorID
            }
            if let connectorID = record.item.metadata?.additional["connector"]?.stringValue?.trimmingCharacters(in: .whitespacesAndNewlines), !connectorID.isEmpty {
                return connectorID
            }
        }
        return nil
    }

    private func stableReplayKey(
        preferred correlationID: String?,
        fallbackTurnID: String?,
        fallbackApprovalID: String?,
        fallbackRunID: String?,
        fallbackTaskID: String?,
        fallbackGroupKey: String
    ) -> String {
        if let correlationID = nonEmpty(correlationID) {
            return "corr:\(correlationID.lowercased())"
        }
        if let turnID = nonEmpty(fallbackTurnID) {
            return "turn:\(turnID.lowercased())"
        }
        if let approvalID = nonEmpty(fallbackApprovalID) {
            return "approval:\(approvalID.lowercased())"
        }
        if let runID = nonEmpty(fallbackRunID) {
            return "run:\(runID.lowercased())"
        }
        if let taskID = nonEmpty(fallbackTaskID) {
            return "task:\(taskID.lowercased())"
        }
        return fallbackGroupKey
    }

    private func shortID(from value: String?, fallbackPrefix: String) -> String {
        if let value = nonEmpty(value) {
            let trimmed = value.trimmingCharacters(in: .whitespacesAndNewlines)
            if trimmed.count <= 12 {
                return trimmed
            }
            return String(trimmed.prefix(12))
        }
        return "\(fallbackPrefix)-local"
    }

    private func parseReplayDate(_ rawValue: String) -> Date {
        if let parsed = V2ReplayProjectionDateParser.parse(rawValue) {
            return parsed
        }
        return Date()
    }

    private func nonEmpty(_ rawValue: String?) -> String? {
        guard let value = rawValue?.trimmingCharacters(in: .whitespacesAndNewlines), !value.isEmpty else {
            return nil
        }
        return value
    }

    private func traceStatus(for status: ReplayEventStatus) -> ReplayDecisionStageStatus {
        switch status {
        case .completed:
            return .completed
        case .awaitingApproval, .running:
            return .pending
        case .failed:
            return .blocked
        }
    }

    private func validatedReplayDaemonBaseURL() -> URL? {
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

    private func resolvedReplayAuthToken() -> String? {
        guard let token = sessionConfigStore.resolvedAccessToken() else {
            return nil
        }
        let trimmed = token.trimmingCharacters(in: .whitespacesAndNewlines)
        return trimmed.isEmpty ? nil : trimmed
    }

    private func mappedReplayProblemSummary(_ error: Error) -> String {
        V2DaemonProblemMapper.map(error: error, context: .replay).summary
    }

    private func captureReplayRequest<T>(_ operation: @escaping () async throws -> T) async -> Result<T, Error> {
        do {
            return .success(try await operation())
        } catch {
            return .failure(error)
        }
    }
}

private struct V2ReplayProjectionResult {
    let events: [ReplayEvent]
}

private enum V2ReplayProjectionDateParser {
    static func parse(_ rawValue: String) -> Date? {
        let trimmed = rawValue.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else {
            return nil
        }

        let withFractional = ISO8601DateFormatter()
        withFractional.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        if let parsed = withFractional.date(from: trimmed) {
            return parsed
        }

        let withoutFractional = ISO8601DateFormatter()
        withoutFractional.formatOptions = [.withInternetDateTime]
        return withoutFractional.date(from: trimmed)
    }
}
