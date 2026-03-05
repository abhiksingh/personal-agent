import Foundation

@MainActor
final class ChatTimelineEventStore {
    private(set) var timelineItems: [ChatTimelineItem] = []
    private(set) var streamedAssistantText = ""
    private(set) var streamingAssistantTimelineItemID: String?
    private(set) var realtimeCompletionReceived = false
    private(set) var realtimeErrorMessage: String?

    private var toolChainIndexByCallID: [String: Int] = [:]
    private var toolChainIndexByApprovalID: [String: Int] = [:]
    private var nextToolChainIndex = 1

    func synchronizeTimeline(_ items: [ChatTimelineItem]) {
        timelineItems = items
    }

    func resetForNewTurn(existingTimeline: [ChatTimelineItem]) {
        timelineItems = existingTimeline
        streamedAssistantText = ""
        streamingAssistantTimelineItemID = nil
        toolChainIndexByCallID = [:]
        toolChainIndexByApprovalID = [:]
        nextToolChainIndex = 1
        realtimeCompletionReceived = false
        realtimeErrorMessage = nil
    }

    func markRealtimeCompleted() {
        realtimeCompletionReceived = true
    }

    func markRealtimeError(_ message: String?) {
        realtimeCompletionReceived = true
        realtimeErrorMessage = ChatTextNormalization.normalizedNonEmpty(message) ?? "Realtime stream reported an error."
    }

    func appendStreamingDelta(_ delta: String, activeCorrelationID: String?, itemID: String?) {
        guard !delta.isEmpty else {
            return
        }
        if let explicitItemID = ChatTextNormalization.normalizedNonEmpty(itemID) {
            reconcileAssistantStreamingItemID(to: explicitItemID)
        }
        streamedAssistantText += delta
        let targetItemID = ChatTextNormalization.normalizedNonEmpty(streamingAssistantTimelineItemID)

        if let targetItemID,
           let index = timelineItems.firstIndex(where: { $0.id == targetItemID }) {
            let existing = timelineItems[index]
            timelineItems[index] = ChatTimelineItem(
                id: existing.id,
                kind: .assistantMessage,
                state: .inFlight,
                title: existing.title,
                summary: existing.summary,
                content: streamedAssistantText,
                timestamp: existing.timestamp,
                daemonRole: "assistant",
                includeInDaemonContext: true,
                correlationID: existing.correlationID,
                taskID: existing.taskID,
                runID: existing.runID,
                details: existing.details
            )
        } else {
            let resolvedItemID = targetItemID ?? UUID().uuidString.lowercased()
            let newItem = ChatTimelineItem(
                id: resolvedItemID,
                kind: .assistantMessage,
                state: .inFlight,
                title: "Assistant",
                summary: "Streaming response",
                content: streamedAssistantText,
                daemonRole: "assistant",
                includeInDaemonContext: true,
                correlationID: ChatTextNormalization.normalizedNonEmpty(activeCorrelationID)
            )
            streamingAssistantTimelineItemID = newItem.id
            timelineItems.append(newItem)
        }
    }

    func reconcileStreamedAssistantTimelineItem(finalAssistantMessage: String, activeCorrelationID: String?) {
        let hasFinalMessage = ChatTextNormalization.nonEmptyPreservingWhitespace(finalAssistantMessage) != nil
        if let itemID = streamingAssistantTimelineItemID,
           let index = timelineItems.firstIndex(where: { $0.id == itemID }) {
            let existing = timelineItems[index]
            if hasFinalMessage {
                timelineItems[index] = ChatTimelineItem(
                    id: existing.id,
                    kind: .assistantMessage,
                    state: .completed,
                    title: existing.title,
                    summary: "Response complete",
                    content: finalAssistantMessage,
                    timestamp: existing.timestamp,
                    daemonRole: "assistant",
                    includeInDaemonContext: true,
                    correlationID: existing.correlationID,
                    taskID: existing.taskID,
                    runID: existing.runID,
                    details: existing.details
                )
            } else if streamedAssistantText.isEmpty {
                timelineItems.remove(at: index)
            }
        } else if hasFinalMessage {
            timelineItems.append(
                ChatTimelineItem(
                    kind: .assistantMessage,
                    state: .completed,
                    title: "Assistant",
                    summary: "Response complete",
                    content: finalAssistantMessage,
                    daemonRole: "assistant",
                    includeInDaemonContext: true,
                    correlationID: ChatTextNormalization.normalizedNonEmpty(activeCorrelationID)
                )
            )
        }
    }

    func reconcileChatTurnTimelineItems(
        items: [DaemonChatTurnItem],
        correlationID: String,
        taskCorrelation: DaemonChatTurnTaskRunCorrelation,
        activeCorrelationID: String?
    ) {
        let hasStreamingAssistantContext =
            ChatTextNormalization.normalizedNonEmpty(streamingAssistantTimelineItemID) != nil || !streamedAssistantText.isEmpty

        guard !items.isEmpty else {
            if hasStreamingAssistantContext {
                reconcileStreamedAssistantTimelineItem(finalAssistantMessage: "", activeCorrelationID: activeCorrelationID)
            }
            return
        }

        var assistantMessageCandidate = ChatTextNormalization.nonEmptyPreservingWhitespace(items.last(where: {
            $0.type.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() == "assistant_message"
        })?.content)
        if assistantMessageCandidate == nil {
            assistantMessageCandidate = ChatTextNormalization.nonEmptyPreservingWhitespace(items.last(where: {
                $0.role?.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() == "assistant"
            })?.content)
        }
        if hasStreamingAssistantContext {
            if let assistantMessageCandidate {
                reconcileStreamedAssistantTimelineItem(
                    finalAssistantMessage: assistantMessageCandidate,
                    activeCorrelationID: activeCorrelationID
                )
            } else {
                reconcileStreamedAssistantTimelineItem(finalAssistantMessage: "", activeCorrelationID: activeCorrelationID)
            }
        }

        let toolChainContexts = chatToolChainContexts(for: items)
        for (index, item) in items.enumerated() {
            guard let mapped = mapChatTurnItemToTimelineItem(
                item,
                index: index,
                correlationID: correlationID,
                taskCorrelation: taskCorrelation,
                toolChainContext: toolChainContexts[index]
            ) else {
                continue
            }
            upsertTimelineItem(mapped)
        }
    }

    func appendTimelineUserMessage(content: String) {
        timelineItems.append(
            ChatTimelineItem(
                kind: .userMessage,
                state: .completed,
                title: "You",
                summary: "Message sent",
                content: content,
                daemonRole: "user",
                includeInDaemonContext: true
            )
        )
    }

    func appendTimelineSystemStatus(
        state: ChatTimelineItemState,
        title: String,
        summary: String,
        correlationID: String?
    ) {
        timelineItems.append(
            ChatTimelineItem(
                kind: .systemStatus,
                state: state,
                title: title,
                summary: summary,
                content: summary,
                includeInDaemonContext: false,
                correlationID: ChatTextNormalization.normalizedNonEmpty(correlationID)
            )
        )
    }

    func appendTimelineFromRealtimeEvent(
        _ event: DaemonRealtimeEventEnvelope,
        correlationID: String,
        activeCorrelationID: String?
    ) {
        let payload = event.payload
        let itemCorrelationID = ChatTextNormalization.normalizedNonEmpty(payload.additional["correlation_id"]?.stringValue)
            ?? ChatTextNormalization.normalizedNonEmpty(event.correlationID)
            ?? correlationID
        let taskID = ChatTextNormalization.normalizedNonEmpty(payload.taskID)
        let runID = ChatTextNormalization.normalizedNonEmpty(payload.runID)

        switch event.eventType {
        case "turn_item_delta":
            let itemType = ChatTextNormalization.normalizedNonEmpty(payload.itemType)?.lowercased() ?? ""
            if itemType == "assistant_message", let delta = payload.delta, !delta.isEmpty {
                appendStreamingDelta(
                    delta,
                    activeCorrelationID: activeCorrelationID,
                    itemID: payload.itemID
                )
            }
        case "turn_item_started", "turn_item_completed":
            let itemType = ChatTextNormalization.normalizedNonEmpty(payload.itemType)?.lowercased() ?? "system_status"
            let itemID = ChatTextNormalization.normalizedNonEmpty(payload.itemID) ?? UUID().uuidString.lowercased()
            let status = ChatTextNormalization.normalizedNonEmpty(payload.status)
            let toolCallID = ChatTextNormalization.normalizedNonEmpty(payload.toolCallID) ?? ChatTextNormalization.normalizedNonEmpty(payload.callID)
            let approvalRequestID = ChatTextNormalization.normalizedNonEmpty(payload.approvalRequestID)
            let kind = chatTimelineItemKind(fromTurnItemType: itemType)
            if kind == .assistantMessage {
                reconcileAssistantStreamingItemID(to: itemID)
            }
            let state: ChatTimelineItemState = event.eventType == "turn_item_started"
                ? .inFlight
                : Self.normalizedTimelineItemState(status)
            let chainIndex = realtimeToolChainIndex(
                toolCallID: toolCallID,
                approvalRequestID: approvalRequestID,
                createIfNeeded: itemType == "tool_call"
            )
            let title = chatTimelineTitle(for: kind)
            let summary = chatTimelineSummary(
                kind: kind,
                state: state,
                toolName: nil,
                errorMessage: nil,
                status: status
            )
            upsertTimelineItem(
                ChatTimelineItem(
                    id: itemID,
                    kind: kind,
                    state: state,
                    title: title,
                    summary: summary,
                    content: kind == .assistantMessage ? ChatTextNormalization.nonEmptyPreservingWhitespace(streamedAssistantText) : nil,
                    daemonRole: kind == .assistantMessage ? "assistant" : (kind == .userMessage ? "user" : nil),
                    includeInDaemonContext: kind == .assistantMessage || kind == .userMessage,
                    correlationID: itemCorrelationID,
                    taskID: taskID,
                    runID: runID,
                    approvalRequestID: approvalRequestID,
                    toolCallID: toolCallID,
                    toolChainIndex: chainIndex
                )
            )
        case "tool_call_started":
            let toolName = ChatTextNormalization.normalizedNonEmpty(payload.toolName)
                ?? ChatTextNormalization.normalizedNonEmpty(payload.name)
                ?? "tool"
            let toolCallID = ChatTextNormalization.normalizedNonEmpty(payload.toolCallID)
                ?? ChatTextNormalization.normalizedNonEmpty(payload.callID)
                ?? UUID().uuidString.lowercased()
            let chainIndex = realtimeToolChainIndex(
                toolCallID: toolCallID,
                approvalRequestID: nil,
                createIfNeeded: true
            )
            let item = ChatTimelineItem(
                kind: .toolCall,
                state: .inFlight,
                title: "Tool Call",
                summary: "Running \(toolName)…",
                content: toolName,
                correlationID: itemCorrelationID,
                taskID: taskID,
                runID: runID,
                toolCallID: toolCallID,
                toolName: toolName,
                toolChainIndex: chainIndex,
                details: [
                    ChatTimelineDetailItem(label: "Tool", value: toolName),
                    ChatTimelineDetailItem(label: "Tool Call", value: toolCallID)
                ]
            )
            upsertTimelineItem(item)
        case "tool_call_output":
            let toolCallID = ChatTextNormalization.normalizedNonEmpty(payload.toolCallID)
                ?? ChatTextNormalization.normalizedNonEmpty(payload.callID)
            let outputPayload = payload.output
            let output = Self.toolOutputSummary(outputPayload)
                ?? ChatTextNormalization.normalizedNonEmpty(payload.message)
                ?? "Tool emitted output."
            let toolName = ChatTextNormalization.normalizedNonEmpty(payload.toolName)
                ?? ChatTextNormalization.normalizedNonEmpty(payload.name)
                ?? "tool"
            let status = ChatTextNormalization.normalizedNonEmpty(payload.status)
            let errorCode = ChatTextNormalization.normalizedNonEmpty(payload.errorCode)
            let errorMessage = ChatTextNormalization.normalizedNonEmpty(payload.error)
            let outputObject = outputPayload
            let inferredTaskID = ChatTextNormalization.normalizedNonEmpty(outputObject?["task_id"]?.stringValue) ?? taskID
            let inferredRunID = ChatTextNormalization.normalizedNonEmpty(outputObject?["run_id"]?.stringValue) ?? runID
            let approvalRequestID = ChatTextNormalization.normalizedNonEmpty(payload.approvalRequestID)
                ?? ChatTextNormalization.normalizedNonEmpty(outputObject?["approval_request_id"]?.stringValue)
            let chainIndex = realtimeToolChainIndex(
                toolCallID: toolCallID,
                approvalRequestID: approvalRequestID,
                createIfNeeded: ChatTextNormalization.normalizedNonEmpty(toolCallID) != nil
            )
            let outputState = Self.normalizedTimelineItemState(status)
            let item = ChatTimelineItem(
                kind: .toolResult,
                state: outputState == .pending ? .inFlight : outputState,
                title: "Tool Output",
                summary: output,
                content: output,
                correlationID: itemCorrelationID,
                taskID: inferredTaskID,
                runID: inferredRunID,
                approvalRequestID: approvalRequestID,
                toolCallID: toolCallID,
                toolName: toolName,
                toolChainIndex: chainIndex,
                details: [
                    ChatTimelineDetailItem(label: "Tool", value: toolName),
                    ChatTimelineDetailItem(label: "Tool Call", value: ChatTextNormalization.normalizedNonEmpty(toolCallID) ?? "unknown"),
                    ChatTimelineDetailItem(label: "Status", value: ChatTextNormalization.normalizedNonEmpty(status) ?? "running"),
                    ChatTimelineDetailItem(label: "Error Code", value: ChatTextNormalization.normalizedNonEmpty(errorCode) ?? ""),
                    ChatTimelineDetailItem(label: "Error", value: ChatTextNormalization.normalizedNonEmpty(errorMessage) ?? "")
                ].filter { ChatTextNormalization.normalizedNonEmpty($0.value) != nil }
            )
            upsertTimelineItem(item)
        case "tool_call_completed":
            let toolName = ChatTextNormalization.normalizedNonEmpty(payload.toolName)
                ?? ChatTextNormalization.normalizedNonEmpty(payload.name)
                ?? "tool"
            let toolCallID = ChatTextNormalization.normalizedNonEmpty(payload.toolCallID)
                ?? ChatTextNormalization.normalizedNonEmpty(payload.callID)
            let completedStatus = ChatTextNormalization.normalizedNonEmpty(payload.status)
            let approvalRequestID = ChatTextNormalization.normalizedNonEmpty(payload.approvalRequestID)
            let chainIndex = realtimeToolChainIndex(
                toolCallID: toolCallID,
                approvalRequestID: approvalRequestID,
                createIfNeeded: ChatTextNormalization.normalizedNonEmpty(toolCallID) != nil
            )
            let state: ChatTimelineItemState = Self.normalizedTimelineItemState(completedStatus)
            let summary = ChatTextNormalization.normalizedNonEmpty(payload.message)
                ?? (state == .completed ? "\(toolName) completed." : "\(toolName) failed.")
            let item = ChatTimelineItem(
                kind: .toolResult,
                state: state == .pending ? .completed : state,
                title: "Tool Result",
                summary: summary,
                content: Self.toolOutputSummary(payload.output),
                correlationID: itemCorrelationID,
                taskID: taskID,
                runID: runID,
                approvalRequestID: approvalRequestID,
                toolCallID: toolCallID,
                toolName: toolName,
                toolChainIndex: chainIndex,
                details: [
                    ChatTimelineDetailItem(label: "Tool", value: toolName),
                    ChatTimelineDetailItem(label: "Tool Call", value: ChatTextNormalization.normalizedNonEmpty(toolCallID) ?? "unknown"),
                    ChatTimelineDetailItem(label: "Status", value: ChatTextNormalization.normalizedNonEmpty(completedStatus) ?? state.label.lowercased()),
                    ChatTimelineDetailItem(label: "State", value: state.label)
                ]
            )
            upsertTimelineItem(item)
        default:
            break
        }
    }

    private struct ChatToolChainContext {
        let chainIndex: Int
        let stepIndex: Int
        let stepCount: Int
    }

    private func chatToolChainContexts(for items: [DaemonChatTurnItem]) -> [Int: ChatToolChainContext] {
        var chainIndexByToolCallID: [String: Int] = [:]
        var chainIndexByApprovalID: [String: Int] = [:]
        var chainItemIndexes: [Int: [Int]] = [:]
        var lastChainIndex: Int? = nil
        var nextChainIndex = 1

        for (index, item) in items.enumerated() {
            let itemType = item.type.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
            let toolCallID = ChatTextNormalization.normalizedNonEmpty(item.toolCallID)
                ?? ChatTextNormalization.normalizedNonEmpty(item.output?["tool_call_id"]?.stringValue)
            let approvalRequestID = ChatTextNormalization.normalizedNonEmpty(item.approvalRequestID)
                ?? ChatTextNormalization.normalizedNonEmpty(item.output?["approval_request_id"]?.stringValue)

            var chainIndex: Int? = nil
            switch itemType {
            case "tool_call":
                if let toolCallID, let existing = chainIndexByToolCallID[toolCallID] {
                    chainIndex = existing
                } else {
                    chainIndex = nextChainIndex
                    nextChainIndex += 1
                    if let toolCallID {
                        chainIndexByToolCallID[toolCallID] = chainIndex
                    }
                }
            case "tool_result":
                if let toolCallID, let existing = chainIndexByToolCallID[toolCallID] {
                    chainIndex = existing
                } else if let approvalRequestID, let existing = chainIndexByApprovalID[approvalRequestID] {
                    chainIndex = existing
                } else {
                    chainIndex = lastChainIndex
                }
            case "approval_request", "approval_requested", "approval_decision", "approval_decided":
                if let approvalRequestID, let existing = chainIndexByApprovalID[approvalRequestID] {
                    chainIndex = existing
                } else if let toolCallID, let existing = chainIndexByToolCallID[toolCallID] {
                    chainIndex = existing
                } else {
                    chainIndex = lastChainIndex
                }
            default:
                break
            }

            guard let chainIndex else {
                continue
            }
            lastChainIndex = chainIndex
            if let toolCallID {
                chainIndexByToolCallID[toolCallID] = chainIndex
            }
            if let approvalRequestID {
                chainIndexByApprovalID[approvalRequestID] = chainIndex
            }
            chainItemIndexes[chainIndex, default: []].append(index)
        }

        var contexts: [Int: ChatToolChainContext] = [:]
        for (chainIndex, indexes) in chainItemIndexes {
            let stepCount = indexes.count
            for (stepOffset, itemIndex) in indexes.enumerated() {
                contexts[itemIndex] = ChatToolChainContext(
                    chainIndex: chainIndex,
                    stepIndex: stepOffset + 1,
                    stepCount: stepCount
                )
            }
        }
        return contexts
    }

    private func mapChatTurnItemToTimelineItem(
        _ item: DaemonChatTurnItem,
        index: Int,
        correlationID: String,
        taskCorrelation: DaemonChatTurnTaskRunCorrelation,
        toolChainContext: ChatToolChainContext?
    ) -> ChatTimelineItem? {
        let itemType = item.type.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        let kind = chatTimelineItemKind(fromTurnItemType: itemType)
        let itemID = (kind == .assistantMessage ? ChatTextNormalization.normalizedNonEmpty(streamingAssistantTimelineItemID) : nil)
            ?? ChatTextNormalization.normalizedNonEmpty(item.itemID)
            ?? "\(correlationID)-\(itemType)-\(index)"
        let toolName = ChatTextNormalization.normalizedNonEmpty(item.toolName)
        let toolCallID = ChatTextNormalization.normalizedNonEmpty(item.toolCallID)
            ?? ChatTextNormalization.normalizedNonEmpty(item.output?["tool_call_id"]?.stringValue)
        let approvalRequestID = ChatTextNormalization.normalizedNonEmpty(item.approvalRequestID)
            ?? ChatTextNormalization.normalizedNonEmpty(item.output?["approval_request_id"]?.stringValue)
        let status = ChatTextNormalization.normalizedNonEmpty(item.status)
        let outputTaskID = ChatTextNormalization.normalizedNonEmpty(item.output?["task_id"]?.stringValue)
        let outputRunID = ChatTextNormalization.normalizedNonEmpty(item.output?["run_id"]?.stringValue)
        let taskID = outputTaskID ?? taskCorrelation.taskID
        let runID = outputRunID ?? taskCorrelation.runID
        let state = Self.normalizedTimelineItemState(status)
        let errorMessage = ChatTextNormalization.normalizedNonEmpty(item.errorMessage)
        let outputSummary = Self.toolOutputSummary(item.output)
        let title = chatTimelineTitle(for: kind)
        let summary = chatTimelineSummary(
            kind: kind,
            state: state,
            toolName: toolName,
            errorMessage: errorMessage,
            status: status
        )
        let content: String? = {
            if let explicitContent = ChatTextNormalization.nonEmptyPreservingWhitespace(item.content) {
                return explicitContent
            }
            if let clarificationPrompt = ChatTextNormalization.nonEmptyPreservingWhitespace(item.output?["clarification_prompt"]?.stringValue) {
                return clarificationPrompt
            }
            if kind == .toolResult {
                return outputSummary
            }
            return nil
        }()

        var details: [ChatTimelineDetailItem] = []
        if let status {
            details.append(ChatTimelineDetailItem(label: "Status", value: status))
        }
        if let toolName {
            details.append(ChatTimelineDetailItem(label: "Tool", value: toolName))
        }
        if let toolChainContext {
            details.append(ChatTimelineDetailItem(label: "Chain", value: "Chain \(toolChainContext.chainIndex)"))
            details.append(
                ChatTimelineDetailItem(
                    label: "Chain Step",
                    value: "\(toolChainContext.stepIndex) of \(toolChainContext.stepCount)"
                )
            )
        }
        if let toolCallID {
            details.append(ChatTimelineDetailItem(label: "Tool Call", value: toolCallID))
        }
        if let approvalRequestID {
            details.append(ChatTimelineDetailItem(label: "Approval Request", value: approvalRequestID))
        }
        if let errorCode = ChatTextNormalization.normalizedNonEmpty(item.errorCode) {
            details.append(ChatTimelineDetailItem(label: "Error Code", value: errorCode))
        }
        if let errorMessage {
            details.append(ChatTimelineDetailItem(label: "Error", value: errorMessage))
        }
        details.append(contentsOf: chatTimelineDetailItems(prefix: "Arg", values: item.arguments, limit: 4))
        details.append(contentsOf: chatTimelineDetailItems(prefix: "Output", values: item.output, limit: 6))
        details.append(contentsOf: chatTimelineDetailItems(prefix: "Meta", values: item.metadata?.allValues, limit: 4))
        if !correlationID.isEmpty {
            details.append(ChatTimelineDetailItem(label: "Correlation", value: correlationID))
        }

        return ChatTimelineItem(
            id: itemID,
            kind: kind,
            state: state,
            title: title,
            summary: summary,
            content: content,
            daemonRole: daemonRole(for: kind, fallbackRole: item.role),
            includeInDaemonContext: kind == .assistantMessage || kind == .userMessage,
            correlationID: correlationID,
            taskID: taskID,
            runID: runID,
            approvalRequestID: approvalRequestID,
            toolCallID: toolCallID,
            toolName: toolName,
            toolChainIndex: toolChainContext?.chainIndex,
            toolChainStep: toolChainContext?.stepIndex,
            toolChainStepCount: toolChainContext?.stepCount,
            details: details
        )
    }

    private func chatTimelineItemKind(fromTurnItemType itemType: String) -> ChatTimelineItemKind {
        switch itemType {
        case "assistant_message":
            return .assistantMessage
        case "user_message":
            return .userMessage
        case "tool_call":
            return .toolCall
        case "tool_result":
            return .toolResult
        case "approval_request", "approval_requested":
            return .approvalRequest
        case "approval_decision", "approval_decided":
            return .approvalDecision
        default:
            return .systemStatus
        }
    }

    private func daemonRole(for kind: ChatTimelineItemKind, fallbackRole: String?) -> String? {
        switch kind {
        case .assistantMessage:
            return "assistant"
        case .userMessage:
            return "user"
        default:
            return ChatTextNormalization.normalizedNonEmpty(fallbackRole)
        }
    }

    private func chatTimelineTitle(for kind: ChatTimelineItemKind) -> String {
        switch kind {
        case .assistantMessage:
            return "Assistant"
        case .userMessage:
            return "You"
        case .toolCall:
            return "Tool Call"
        case .toolResult:
            return "Tool Result"
        case .approvalRequest:
            return "Approval Required"
        case .approvalDecision:
            return "Approval Decision"
        case .systemStatus:
            return "System"
        }
    }

    private func chatTimelineSummary(
        kind: ChatTimelineItemKind,
        state: ChatTimelineItemState,
        toolName: String?,
        errorMessage: String?,
        status: String?
    ) -> String {
        let normalizedToolName = ChatTextNormalization.normalizedNonEmpty(toolName) ?? "tool"
        switch kind {
        case .assistantMessage:
            return state == .inFlight ? "Streaming response" : "Response complete"
        case .userMessage:
            return "Message sent"
        case .toolCall:
            if state == .inFlight || state == .pending {
                return "Running \(normalizedToolName)…"
            }
            if state == .blocked {
                return "\(normalizedToolName) is waiting for approval."
            }
            if state == .failed {
                return "\(normalizedToolName) failed."
            }
            return "\(normalizedToolName) completed."
        case .toolResult:
            if let errorMessage {
                return errorMessage
            }
            if state == .blocked {
                return "\(normalizedToolName) produced output and is waiting for approval."
            }
            if state == .failed {
                return "\(normalizedToolName) failed."
            }
            if state == .inFlight || state == .pending {
                return "\(normalizedToolName) output is in progress."
            }
            return "\(normalizedToolName) completed."
        case .approvalRequest:
            return "Approval is required before execution can continue."
        case .approvalDecision:
            if state == .completed {
                return "Approval was granted."
            }
            if state == .failed {
                return "Approval was rejected."
            }
            return "Approval decision is in progress."
        case .systemStatus:
            if let status = ChatTextNormalization.normalizedNonEmpty(status) {
                return "Turn status: \(status)."
            }
            return state.label
        }
    }

    private func chatTimelineDetailItems(
        prefix: String,
        values: [String: DaemonJSONValue]?,
        limit: Int
    ) -> [ChatTimelineDetailItem] {
        guard let values, !values.isEmpty else {
            return []
        }
        return values.keys
            .sorted { lhs, rhs in
                lhs.localizedCaseInsensitiveCompare(rhs) == .orderedAscending
            }
            .prefix(limit)
            .compactMap { key in
                guard let value = ChatTextNormalization.normalizedNonEmpty(values[key]?.displayText) else {
                    return nil
                }
                return ChatTimelineDetailItem(label: "\(prefix) \(key)", value: value)
            }
    }

    private func realtimeToolChainIndex(
        toolCallID: String?,
        approvalRequestID: String?,
        createIfNeeded: Bool
    ) -> Int? {
        if let toolCallID, let existing = toolChainIndexByCallID[toolCallID] {
            if let approvalRequestID {
                toolChainIndexByApprovalID[approvalRequestID] = existing
            }
            return existing
        }
        if let approvalRequestID, let existing = toolChainIndexByApprovalID[approvalRequestID] {
            if let toolCallID {
                toolChainIndexByCallID[toolCallID] = existing
            }
            return existing
        }
        guard createIfNeeded else {
            return nil
        }
        let created = nextToolChainIndex
        nextToolChainIndex += 1
        if let toolCallID {
            toolChainIndexByCallID[toolCallID] = created
        }
        if let approvalRequestID {
            toolChainIndexByApprovalID[approvalRequestID] = created
        }
        return created
    }

    private func upsertTimelineItem(_ item: ChatTimelineItem) {
        if let index = timelineItems.firstIndex(where: { timelineItem in
            if timelineItem.id == item.id {
                return true
            }
            if let lhs = ChatTextNormalization.normalizedNonEmpty(timelineItem.toolCallID), let rhs = ChatTextNormalization.normalizedNonEmpty(item.toolCallID), lhs == rhs,
               timelineItem.kind == item.kind {
                return true
            }
            return false
        }) {
            let existing = timelineItems[index]
            timelineItems[index] = mergedTimelineItem(existing: existing, incoming: item)
        } else {
            timelineItems.append(item)
        }
    }

    private func reconcileAssistantStreamingItemID(to canonicalItemID: String) {
        guard let canonical = ChatTextNormalization.normalizedNonEmpty(canonicalItemID) else {
            return
        }

        defer {
            streamingAssistantTimelineItemID = canonical
        }

        guard let existingStreamingID = ChatTextNormalization.normalizedNonEmpty(streamingAssistantTimelineItemID),
              existingStreamingID != canonical else {
            return
        }
        guard let existingIndex = timelineItems.firstIndex(where: { $0.id == existingStreamingID }) else {
            return
        }
        guard timelineItems[existingIndex].kind == .assistantMessage else {
            return
        }

        if let canonicalIndex = timelineItems.firstIndex(where: { $0.id == canonical }) {
            let merged = mergedTimelineItem(
                existing: timelineItems[canonicalIndex],
                incoming: timelineItems[existingIndex]
            )
            timelineItems[canonicalIndex] = ChatTimelineItem(
                id: canonical,
                kind: merged.kind,
                state: merged.state,
                title: merged.title,
                summary: merged.summary,
                content: merged.content,
                timestamp: merged.timestamp,
                daemonRole: merged.daemonRole,
                includeInDaemonContext: merged.includeInDaemonContext,
                correlationID: merged.correlationID,
                taskID: merged.taskID,
                runID: merged.runID,
                approvalRequestID: merged.approvalRequestID,
                toolCallID: merged.toolCallID,
                toolName: merged.toolName,
                toolChainIndex: merged.toolChainIndex,
                toolChainStep: merged.toolChainStep,
                toolChainStepCount: merged.toolChainStepCount,
                details: merged.details
            )
            timelineItems.remove(at: existingIndex)
            return
        }

        let existing = timelineItems[existingIndex]
        timelineItems[existingIndex] = ChatTimelineItem(
            id: canonical,
            kind: existing.kind,
            state: existing.state,
            title: existing.title,
            summary: existing.summary,
            content: existing.content,
            timestamp: existing.timestamp,
            daemonRole: existing.daemonRole,
            includeInDaemonContext: existing.includeInDaemonContext,
            correlationID: existing.correlationID,
            taskID: existing.taskID,
            runID: existing.runID,
            approvalRequestID: existing.approvalRequestID,
            toolCallID: existing.toolCallID,
            toolName: existing.toolName,
            toolChainIndex: existing.toolChainIndex,
            toolChainStep: existing.toolChainStep,
            toolChainStepCount: existing.toolChainStepCount,
            details: existing.details
        )
    }

    private func mergedTimelineItem(existing: ChatTimelineItem, incoming: ChatTimelineItem) -> ChatTimelineItem {
        let contentToUse: String? = {
            if incoming.kind == .assistantMessage, ChatTextNormalization.nonEmptyPreservingWhitespace(incoming.content) == nil {
                return ChatTextNormalization.nonEmptyPreservingWhitespace(existing.content)
            }
            return incoming.content
        }()

        return ChatTimelineItem(
            id: incoming.id,
            kind: incoming.kind,
            state: incoming.state,
            title: incoming.title,
            summary: incoming.summary,
            content: contentToUse,
            timestamp: existing.timestamp,
            daemonRole: incoming.daemonRole,
            includeInDaemonContext: incoming.includeInDaemonContext,
            correlationID: incoming.correlationID,
            taskID: incoming.taskID,
            runID: incoming.runID,
            approvalRequestID: incoming.approvalRequestID,
            toolCallID: incoming.toolCallID,
            toolName: incoming.toolName,
            toolChainIndex: incoming.toolChainIndex,
            toolChainStep: incoming.toolChainStep,
            toolChainStepCount: incoming.toolChainStepCount,
            details: incoming.details
        )
    }

    static func normalizedTimelineItemState(_ raw: String?) -> ChatTimelineItemState {
        switch ChatTextNormalization.normalizedNonEmpty(raw)?.lowercased() {
        case "pending", "queued":
            return .pending
        case "started", "running", "in_progress", "in_flight":
            return .inFlight
        case "awaiting_approval", "blocked", "paused":
            return .blocked
        case "completed", "done", "success", "succeeded":
            return .completed
        case "failed", "error", "denied", "rejected", "cancelled", "canceled":
            return .failed
        default:
            return .pending
        }
    }

    static func toolOutputSummary(_ output: [String: DaemonJSONValue]?) -> String? {
        guard let output, !output.isEmpty else {
            return nil
        }
        if let prompt = ChatTextNormalization.nonEmptyPreservingWhitespace(output["clarification_prompt"]?.stringValue) {
            return prompt
        }
        if let runState = ChatTextNormalization.normalizedNonEmpty(output["run_state"]?.stringValue) {
            return "Run state: \(runState)"
        }
        let joined = output.keys.sorted().compactMap { key -> String? in
            guard let value = ChatTextNormalization.normalizedNonEmpty(output[key]?.displayText) else {
                return nil
            }
            return "\(key)=\(value)"
        }
        guard !joined.isEmpty else {
            return nil
        }
        return joined.joined(separator: " • ")
    }

}
