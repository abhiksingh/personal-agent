import Foundation

struct ChatTurnSignals {
    let approvalRequired: Bool
    let approvalRequestID: String?
    let clarificationRequired: Bool
    let clarificationPrompt: String?
}

private struct ChatResponseShapingSignals {
    let channel: String?
    let profile: String?
    let personaPolicySource: String?
    let guardrailCount: Int?
    let instructionCount: Int?
}

@MainActor
final class ChatTurnContextStore {
    private(set) var latestTurnTraceability: ChatTaskRunTraceabilityItem?
    private(set) var latestTurnExplainability: ChatTurnExplainabilityItem?
    private(set) var explainabilityStatusMessage: String? = "No chat explainability loaded yet."
    private(set) var explainabilityErrorMessage: String?

    func clearExplainabilityForMissingToken() {
        latestTurnExplainability = nil
        explainabilityErrorMessage = nil
        explainabilityStatusMessage = "Set Assistant Access Token before loading chat explainability."
    }

    func resetAllForMissingToken() {
        latestTurnTraceability = nil
        clearExplainabilityForMissingToken()
    }

    func resetForNewTurn() {
        latestTurnTraceability = nil
        latestTurnExplainability = nil
        explainabilityErrorMessage = nil
        explainabilityStatusMessage = "Waiting for chat explainability."
    }

    func markExplainabilityLoading(userInitiated: Bool, isAlreadyInFlight: Bool) -> Bool {
        guard !isAlreadyInFlight else {
            if userInitiated {
                explainabilityStatusMessage = "Chat explainability is already loading."
            }
            return false
        }
        explainabilityErrorMessage = nil
        explainabilityStatusMessage = "Loading chat explainability trace…"
        return true
    }

    func applyExplainabilitySuccess(
        _ response: DaemonChatTurnExplainResponse,
        defaultWorkspaceID: String
    ) {
        let mapped = mapChatTurnExplainabilityResponse(
            response,
            defaultWorkspaceID: defaultWorkspaceID
        )
        latestTurnExplainability = mapped
        explainabilityErrorMessage = nil
        explainabilityStatusMessage = chatTurnExplainabilitySummaryMessage(mapped)
    }

    func applyExplainabilityFailure(message: String, retainPreviousOnFailure: Bool) {
        if !retainPreviousOnFailure {
            latestTurnExplainability = nil
        }
        explainabilityErrorMessage = message
        explainabilityStatusMessage = message
    }

    func markExplainabilityInterrupted() {
        latestTurnExplainability = nil
        explainabilityErrorMessage = nil
        explainabilityStatusMessage = "Chat was interrupted before explainability could load."
    }

    func markExplainabilityUnavailableUntilTurnCompletes() {
        latestTurnExplainability = nil
        explainabilityErrorMessage = nil
        explainabilityStatusMessage = "Chat explainability is unavailable until a turn completes."
    }

    func markExplainabilityUnavailableForRouteSetup() {
        latestTurnExplainability = nil
        explainabilityErrorMessage = nil
        explainabilityStatusMessage = "No chat explainability available while route setup is incomplete."
    }

    func updateTraceabilityFromTurnResponse(
        _ correlation: DaemonChatTurnTaskRunCorrelation,
        fallbackCorrelationID: String?,
        taskClass: String?,
        provider: String?,
        modelKey: String?,
        routeSource: String?,
        channelID: String?,
        turnContractVersion: String?,
        turnItemSchemaVersion: String?,
        realtimeEventContractVersion: String?,
        items: [DaemonChatTurnItem]
    ) {
        let signals = chatTurnSignals(from: items)
        let responseShaping = responseShapingSignals(
            from: items,
            fallbackChannelID: nonEmpty(channelID)
        )
        latestTurnTraceability = makeChatTurnTraceability(
            available: correlation.available,
            source: nonEmpty(correlation.source) ?? "none",
            taskID: nonEmpty(correlation.taskID),
            runID: nonEmpty(correlation.runID),
            taskState: nonEmpty(correlation.taskState),
            runState: nonEmpty(correlation.runState),
            correlationID: nonEmpty(fallbackCorrelationID),
            taskClass: nonEmpty(taskClass),
            provider: nonEmpty(provider),
            modelKey: nonEmpty(modelKey),
            routeSource: nonEmpty(routeSource),
            turnContractVersion: nonEmpty(turnContractVersion),
            turnItemSchemaVersion: nonEmpty(turnItemSchemaVersion),
            realtimeEventContractVersion: nonEmpty(realtimeEventContractVersion),
            realtimeLifecycleContractVersion: nonEmpty(latestTurnTraceability?.realtimeLifecycleContractVersion),
            realtimeLifecycleSchemaVersion: nonEmpty(latestTurnTraceability?.realtimeLifecycleSchemaVersion),
            approvalRequired: signals.approvalRequired,
            approvalRequestID: signals.approvalRequestID,
            clarificationRequired: signals.clarificationRequired,
            clarificationPrompt: signals.clarificationPrompt,
            responseShapingChannel: responseShaping.channel,
            responseShapingProfile: responseShaping.profile,
            personaPolicySource: responseShaping.personaPolicySource,
            responseShapingGuardrailCount: responseShaping.guardrailCount,
            responseShapingInstructionCount: responseShaping.instructionCount
        )
    }

    @discardableResult
    func mergeRealtimeLifecycleContractMetadata(
        from event: DaemonRealtimeEventEnvelope,
        fallbackCorrelationID: String,
        routeProvider: String?,
        routeModelKey: String?,
        routeSource: String?
    ) -> Bool {
        let lifecycleContractVersion = nonEmpty(event.contractVersion)
        let lifecycleSchemaVersion = nonEmpty(event.lifecycleSchemaVersion)
        guard lifecycleContractVersion != nil || lifecycleSchemaVersion != nil else {
            return false
        }

        let existing = latestTurnTraceability
        let mergedSource = nonEmpty(existing?.source) ?? "realtime"
        let mergedCorrelationID = nonEmpty(existing?.correlationID)
            ?? nonEmpty(event.correlationID)
            ?? nonEmpty(fallbackCorrelationID)
        let mergedTaskClass = nonEmpty(existing?.taskClass) ?? "chat"
        let mergedProvider = nonEmpty(existing?.provider) ?? nonEmpty(routeProvider)
        let mergedModelKey = nonEmpty(existing?.modelKey) ?? nonEmpty(routeModelKey)
        let mergedRouteSource = nonEmpty(existing?.routeSource) ?? nonEmpty(routeSource)

        latestTurnTraceability = makeChatTurnTraceability(
            available: existing?.available ?? false,
            source: mergedSource,
            taskID: nonEmpty(existing?.taskID),
            runID: nonEmpty(existing?.runID),
            taskState: nonEmpty(existing?.taskState),
            runState: nonEmpty(existing?.runState),
            correlationID: mergedCorrelationID,
            taskClass: mergedTaskClass,
            provider: mergedProvider,
            modelKey: mergedModelKey,
            routeSource: mergedRouteSource,
            turnContractVersion: nonEmpty(existing?.turnContractVersion),
            turnItemSchemaVersion: nonEmpty(existing?.turnItemSchemaVersion),
            realtimeEventContractVersion: nonEmpty(existing?.realtimeEventContractVersion),
            realtimeLifecycleContractVersion: lifecycleContractVersion
                ?? nonEmpty(existing?.realtimeLifecycleContractVersion),
            realtimeLifecycleSchemaVersion: lifecycleSchemaVersion
                ?? nonEmpty(existing?.realtimeLifecycleSchemaVersion),
            approvalRequired: existing?.approvalRequired ?? false,
            approvalRequestID: nonEmpty(existing?.approvalRequestID),
            clarificationRequired: existing?.clarificationRequired ?? false,
            clarificationPrompt: nonEmpty(existing?.clarificationPrompt),
            responseShapingChannel: nonEmpty(existing?.responseShapingChannel),
            responseShapingProfile: nonEmpty(existing?.responseShapingProfile),
            personaPolicySource: nonEmpty(existing?.personaPolicySource),
            responseShapingGuardrailCount: existing?.responseShapingGuardrailCount,
            responseShapingInstructionCount: existing?.responseShapingInstructionCount
        )

        return true
    }

    func chatTurnSignals(from items: [DaemonChatTurnItem]) -> ChatTurnSignals {
        var approvalRequestID: String?
        var clarificationPrompt: String?
        var clarificationRequired = false
        var approvalRequired = false

        for item in items {
            let itemType = item.type.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
            if itemType == "approval_request" {
                approvalRequired = true
            }
            if approvalRequestID == nil {
                approvalRequestID = nonEmpty(item.approvalRequestID)
                    ?? nonEmpty(item.output?["approval_request_id"]?.stringValue)
            }
            if normalizedBoolean(item.output?["approval_required"]) == true {
                approvalRequired = true
            }
            if normalizedBoolean(item.output?["clarification_required"]) == true {
                clarificationRequired = true
            }
            if clarificationPrompt == nil {
                clarificationPrompt = nonEmpty(item.output?["clarification_prompt"]?.stringValue)
                    ?? ((itemType == "approval_request" || itemType == "tool_result") ? nonEmpty(item.content) : nil)
            }
        }

        if approvalRequestID != nil {
            approvalRequired = true
        }
        return ChatTurnSignals(
            approvalRequired: approvalRequired,
            approvalRequestID: approvalRequestID,
            clarificationRequired: clarificationRequired,
            clarificationPrompt: clarificationPrompt
        )
    }

    private func makeChatTurnTraceability(
        available: Bool,
        source: String,
        taskID: String?,
        runID: String?,
        taskState: String?,
        runState: String?,
        correlationID: String?,
        taskClass: String?,
        provider: String?,
        modelKey: String?,
        routeSource: String?,
        turnContractVersion: String?,
        turnItemSchemaVersion: String?,
        realtimeEventContractVersion: String?,
        realtimeLifecycleContractVersion: String?,
        realtimeLifecycleSchemaVersion: String?,
        approvalRequired: Bool,
        approvalRequestID: String?,
        clarificationRequired: Bool,
        clarificationPrompt: String?,
        responseShapingChannel: String?,
        responseShapingProfile: String?,
        personaPolicySource: String?,
        responseShapingGuardrailCount: Int?,
        responseShapingInstructionCount: Int?
    ) -> ChatTaskRunTraceabilityItem {
        ChatTaskRunTraceabilityItem(
            available: available,
            source: source,
            taskID: taskID,
            runID: runID,
            taskState: taskState,
            runState: runState,
            correlationID: correlationID,
            taskClass: taskClass,
            provider: provider,
            modelKey: modelKey,
            routeSource: routeSource,
            turnContractVersion: turnContractVersion,
            turnItemSchemaVersion: turnItemSchemaVersion,
            realtimeEventContractVersion: realtimeEventContractVersion,
            realtimeLifecycleContractVersion: realtimeLifecycleContractVersion,
            realtimeLifecycleSchemaVersion: realtimeLifecycleSchemaVersion,
            approvalRequired: approvalRequired,
            approvalRequestID: approvalRequestID,
            clarificationRequired: clarificationRequired,
            clarificationPrompt: clarificationPrompt,
            responseShapingChannel: responseShapingChannel,
            responseShapingProfile: responseShapingProfile,
            personaPolicySource: personaPolicySource,
            responseShapingGuardrailCount: responseShapingGuardrailCount,
            responseShapingInstructionCount: responseShapingInstructionCount
        )
    }

    private func mapChatTurnExplainabilityResponse(
        _ response: DaemonChatTurnExplainResponse,
        defaultWorkspaceID: String
    ) -> ChatTurnExplainabilityItem {
        let selectedRoute = response.selectedRoute
        let toolCatalog = response.toolCatalog.map { entry in
            ChatTurnExplainabilityToolCatalogItem(
                name: nonEmpty(entry.name) ?? "unknown_tool",
                description: nonEmpty(entry.description),
                capabilityKeys: entry.capabilityKeys.compactMap { nonEmpty($0) },
                inputSchemaSummary: chatToolInputSchemaSummary(entry.inputSchema)
            )
        }
        let policyDecisions = response.policyDecisions.enumerated().map { index, decision in
            let idSeed = [
                nonEmpty(decision.toolName) ?? "tool",
                nonEmpty(decision.capabilityKey) ?? "capability",
                nonEmpty(decision.decision) ?? "decision",
                "\(index)"
            ].joined(separator: "::")
            return ChatTurnExplainabilityPolicyDecisionItem(
                id: idSeed,
                toolName: nonEmpty(decision.toolName) ?? "unknown_tool",
                capabilityKey: nonEmpty(decision.capabilityKey),
                decision: nonEmpty(decision.decision) ?? "unknown",
                reason: nonEmpty(decision.reason)
            )
        }

        return ChatTurnExplainabilityItem(
            workspaceID: nonEmpty(response.workspaceID) ?? defaultWorkspaceID,
            taskClass: nonEmpty(response.taskClass) ?? "chat",
            requestedByActorID: nonEmpty(response.requestedByActorID),
            subjectActorID: nonEmpty(response.subjectActorID),
            actingAsActorID: nonEmpty(response.actingAsActorID),
            contractVersion: nonEmpty(response.contractVersion) ?? "chat_turn_explain.v1",
            selectedProvider: nonEmpty(selectedRoute?.selectedProvider),
            selectedModelKey: nonEmpty(selectedRoute?.selectedModelKey),
            selectedSource: nonEmpty(selectedRoute?.selectedSource),
            routeSummary: nonEmpty(selectedRoute?.summary),
            routeReasonCodes: selectedRoute?.reasonCodes.compactMap { nonEmpty($0) } ?? [],
            routeExplanations: selectedRoute?.explanations.compactMap { nonEmpty($0) } ?? [],
            toolCatalog: toolCatalog,
            policyDecisions: policyDecisions
        )
    }

    private func chatTurnExplainabilitySummaryMessage(_ item: ChatTurnExplainabilityItem) -> String {
        if let routeLabel = item.routeLabel {
            return "Loaded chat explainability for \(routeLabel)."
        }
        return "Loaded chat explainability details."
    }

    private func chatToolInputSchemaSummary(_ schema: [String: DaemonJSONValue]?) -> String? {
        guard let schema, !schema.isEmpty else {
            return nil
        }
        let keys = schema.keys.sorted {
            $0.localizedCaseInsensitiveCompare($1) == .orderedAscending
        }
        if keys.isEmpty {
            return nil
        }
        if keys.count <= 4 {
            return keys.joined(separator: ", ")
        }
        let prefix = keys.prefix(4).joined(separator: ", ")
        return "\(prefix), +\(keys.count - 4) more"
    }

    private func normalizedBoolean(_ value: DaemonJSONValue?) -> Bool? {
        switch value {
        case .bool(let flag):
            return flag
        case .string(let raw):
            switch raw.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() {
            case "true", "1", "yes", "ok":
                return true
            case "false", "0", "no":
                return false
            default:
                return nil
            }
        default:
            return nil
        }
    }

    private func responseShapingSignals(
        from items: [DaemonChatTurnItem],
        fallbackChannelID: String?
    ) -> ChatResponseShapingSignals {
        var channelID: String?
        var profileID: String?
        var personaSource: String?
        var guardrailCount: Int?
        var instructionCount: Int?

        for item in items.reversed() {
            guard let metadata = item.metadata, !metadata.isEmpty else {
                continue
            }
            if channelID == nil {
                channelID = normalizeResponseShapingChannel(
                    nonEmpty(metadata.responseShapingChannel)
                )
            }
            if profileID == nil {
                profileID = nonEmpty(metadata.responseShapingProfile)
            }
            if personaSource == nil {
                personaSource = nonEmpty(metadata.personaPolicySource)
            }
            if guardrailCount == nil {
                guardrailCount = metadata.responseShapingGuardrailCount
            }
            if instructionCount == nil {
                instructionCount = metadata.responseShapingInstructionCount
            }
        }

        let resolvedChannelID = channelID
            ?? normalizeResponseShapingChannel(fallbackChannelID)
        let resolvedProfileID = profileID
            ?? responseShapingProfile(for: resolvedChannelID)
        return ChatResponseShapingSignals(
            channel: resolvedChannelID,
            profile: resolvedProfileID,
            personaPolicySource: personaSource,
            guardrailCount: guardrailCount,
            instructionCount: instructionCount
        )
    }

    private func normalizeResponseShapingChannel(_ raw: String?) -> String? {
        guard let raw = nonEmpty(raw)?.lowercased() else {
            return nil
        }
        switch raw {
        case "app", "app_chat":
            return "app"
        case "message", "sms", "imessage", "twilio_sms", "twilio":
            return "message"
        case "voice", "twilio_voice":
            return "voice"
        default:
            return raw
        }
    }

    private func responseShapingProfile(for channelID: String?) -> String? {
        switch normalizeResponseShapingChannel(channelID) {
        case "message":
            return "message.compact"
        case "voice":
            return "voice.spoken"
        case "app":
            return "app.default"
        default:
            return nil
        }
    }

    private func nonEmpty(_ value: String?) -> String? {
        guard let trimmed = value?.trimmingCharacters(in: .whitespacesAndNewlines), !trimmed.isEmpty else {
            return nil
        }
        return trimmed
    }
}
