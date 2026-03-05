import Foundation

private func trimmedDaemonString(_ value: String?) -> String? {
    guard let value else {
        return nil
    }
    let trimmed = value.trimmingCharacters(in: .whitespacesAndNewlines)
    return trimmed.isEmpty ? nil : trimmed
}

private func parseDaemonStringList(_ raw: String) -> [String] {
    raw.split(whereSeparator: { character in
        character == "," || character == ";" || character.isNewline
    })
    .map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }
    .filter { !$0.isEmpty }
}

extension DaemonJSONValue {
    var scalarStringValue: String? {
        switch self {
        case .string(let value):
            return value
        case .number(let value):
            if value.rounded() == value {
                return String(Int(value))
            }
            return String(value)
        case .bool(let value):
            return value ? "true" : "false"
        case .array, .object, .null:
            return nil
        }
    }

    var boolValue: Bool? {
        switch self {
        case .bool(let value):
            return value
        case .number(let value):
            if value == 1 {
                return true
            }
            if value == 0 {
                return false
            }
            return nil
        case .string(let value):
            switch value.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() {
            case "true", "1", "yes", "y", "on":
                return true
            case "false", "0", "no", "n", "off":
                return false
            default:
                return nil
            }
        case .array, .object, .null:
            return nil
        }
    }

    var intValue: Int? {
        switch self {
        case .number(let value):
            let rounded = Int(value.rounded())
            return value == value.rounded() ? rounded : nil
        case .string(let value):
            return Int(value.trimmingCharacters(in: .whitespacesAndNewlines))
        case .bool(let value):
            return value ? 1 : 0
        case .array, .object, .null:
            return nil
        }
    }

    var arrayValue: [DaemonJSONValue]? {
        if case .array(let values) = self {
            return values
        }
        return nil
    }
}

private extension KeyedDecodingContainer {
    func decodeLossyString(forKey key: Key) throws -> String? {
        if let value = try decodeIfPresent(String.self, forKey: key) {
            return value
        }
        guard let value = try decodeIfPresent(DaemonJSONValue.self, forKey: key) else {
            return nil
        }
        return value.scalarStringValue
    }

    func decodeLossyBool(forKey key: Key) throws -> Bool? {
        if let value = try decodeIfPresent(Bool.self, forKey: key) {
            return value
        }
        guard let value = try decodeIfPresent(DaemonJSONValue.self, forKey: key) else {
            return nil
        }
        return value.boolValue
    }

    func decodeLossyInt(forKey key: Key) throws -> Int? {
        if let value = try decodeIfPresent(Int.self, forKey: key) {
            return value
        }
        guard let value = try decodeIfPresent(DaemonJSONValue.self, forKey: key) else {
            return nil
        }
        return value.intValue
    }

    func decodeLossyStringArray(forKey key: Key) throws -> [String]? {
        if let values = try decodeIfPresent([String].self, forKey: key) {
            return values
        }
        guard let value = try decodeIfPresent(DaemonJSONValue.self, forKey: key) else {
            return nil
        }
        switch value {
        case .array(let nestedValues):
            return nestedValues.compactMap { nested in
                trimmedDaemonString(nested.scalarStringValue)
            }
        case .string(let raw):
            return parseDaemonStringList(raw)
        default:
            return nil
        }
    }
}

private func decodeDaemonJSONObject(from decoder: Decoder) -> [String: DaemonJSONValue] {
    (try? decoder.singleValueContainer().decode([String: DaemonJSONValue].self)) ?? [:]
}

struct DaemonChatToolPolicyRationale: Decodable, Sendable {
    let policyVersion: String?
    let decision: String?
    let reasonCode: String?
    let reason: String?
    let capabilityKey: String?
    let capabilityName: String?
    let riskClass: String?
    let idempotency: String?
    let approvalMode: String?
    let channelConstraint: String?
    let needsApproval: Bool?
    let additional: [String: DaemonJSONValue]

    enum CodingKeys: String, CodingKey {
        case policyVersion = "policy_version"
        case decision
        case reasonCode = "reason_code"
        case reason
        case capabilityKey = "capability_key"
        case capabilityName = "capability_name"
        case riskClass = "risk_class"
        case idempotency
        case approvalMode = "approval_mode"
        case channelConstraint = "channel_constraint"
        case needsApproval = "needs_approval"
    }

    init(
        policyVersion: String? = nil,
        decision: String? = nil,
        reasonCode: String? = nil,
        reason: String? = nil,
        capabilityKey: String? = nil,
        capabilityName: String? = nil,
        riskClass: String? = nil,
        idempotency: String? = nil,
        approvalMode: String? = nil,
        channelConstraint: String? = nil,
        needsApproval: Bool? = nil,
        additional: [String: DaemonJSONValue] = [:]
    ) {
        self.policyVersion = policyVersion
        self.decision = decision
        self.reasonCode = reasonCode
        self.reason = reason
        self.capabilityKey = capabilityKey
        self.capabilityName = capabilityName
        self.riskClass = riskClass
        self.idempotency = idempotency
        self.approvalMode = approvalMode
        self.channelConstraint = channelConstraint
        self.needsApproval = needsApproval
        self.additional = additional
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        policyVersion = try container.decodeLossyString(forKey: .policyVersion)
        decision = try container.decodeLossyString(forKey: .decision)
        reasonCode = try container.decodeLossyString(forKey: .reasonCode)
        reason = try container.decodeLossyString(forKey: .reason)
        capabilityKey = try container.decodeLossyString(forKey: .capabilityKey)
        capabilityName = try container.decodeLossyString(forKey: .capabilityName)
        riskClass = try container.decodeLossyString(forKey: .riskClass)
        idempotency = try container.decodeLossyString(forKey: .idempotency)
        approvalMode = try container.decodeLossyString(forKey: .approvalMode)
        channelConstraint = try container.decodeLossyString(forKey: .channelConstraint)
        needsApproval = try container.decodeLossyBool(forKey: .needsApproval)

        var extras = decodeDaemonJSONObject(from: decoder)
        for key in [
            "policy_version",
            "decision",
            "reason_code",
            "reason",
            "capability_key",
            "capability_name",
            "risk_class",
            "idempotency",
            "approval_mode",
            "channel_constraint",
            "needs_approval"
        ] {
            extras.removeValue(forKey: key)
        }
        additional = extras
    }

    var allValues: [String: DaemonJSONValue] {
        var values = additional
        if let policyVersion = trimmedDaemonString(policyVersion) { values["policy_version"] = .string(policyVersion) }
        if let decision = trimmedDaemonString(decision) { values["decision"] = .string(decision) }
        if let reasonCode = trimmedDaemonString(reasonCode) { values["reason_code"] = .string(reasonCode) }
        if let reason = trimmedDaemonString(reason) { values["reason"] = .string(reason) }
        if let capabilityKey = trimmedDaemonString(capabilityKey) { values["capability_key"] = .string(capabilityKey) }
        if let capabilityName = trimmedDaemonString(capabilityName) { values["capability_name"] = .string(capabilityName) }
        if let riskClass = trimmedDaemonString(riskClass) { values["risk_class"] = .string(riskClass) }
        if let idempotency = trimmedDaemonString(idempotency) { values["idempotency"] = .string(idempotency) }
        if let approvalMode = trimmedDaemonString(approvalMode) { values["approval_mode"] = .string(approvalMode) }
        if let channelConstraint = trimmedDaemonString(channelConstraint) { values["channel_constraint"] = .string(channelConstraint) }
        if let needsApproval { values["needs_approval"] = .bool(needsApproval) }
        return values
    }
}

struct DaemonChatMetadataRemediation: Decodable, Sendable {
    let code: String?
    let domain: String?
    let summary: String?
    let primaryAction: String?
    let secondaryAction: String?
    let additional: [String: DaemonJSONValue]

    enum CodingKeys: String, CodingKey {
        case code
        case domain
        case summary
        case primaryAction = "primary_action"
        case secondaryAction = "secondary_action"
    }

    init(
        code: String? = nil,
        domain: String? = nil,
        summary: String? = nil,
        primaryAction: String? = nil,
        secondaryAction: String? = nil,
        additional: [String: DaemonJSONValue] = [:]
    ) {
        self.code = code
        self.domain = domain
        self.summary = summary
        self.primaryAction = primaryAction
        self.secondaryAction = secondaryAction
        self.additional = additional
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        code = try container.decodeLossyString(forKey: .code)
        domain = try container.decodeLossyString(forKey: .domain)
        summary = try container.decodeLossyString(forKey: .summary)
        primaryAction = try container.decodeLossyString(forKey: .primaryAction)
        secondaryAction = try container.decodeLossyString(forKey: .secondaryAction)

        var extras = decodeDaemonJSONObject(from: decoder)
        for key in ["code", "domain", "summary", "primary_action", "secondary_action"] {
            extras.removeValue(forKey: key)
        }
        additional = extras
    }

    var allValues: [String: DaemonJSONValue] {
        var values = additional
        if let code = trimmedDaemonString(code) { values["code"] = .string(code) }
        if let domain = trimmedDaemonString(domain) { values["domain"] = .string(domain) }
        if let summary = trimmedDaemonString(summary) { values["summary"] = .string(summary) }
        if let primaryAction = trimmedDaemonString(primaryAction) { values["primary_action"] = .string(primaryAction) }
        if let secondaryAction = trimmedDaemonString(secondaryAction) { values["secondary_action"] = .string(secondaryAction) }
        return values
    }
}

struct DaemonChatTurnItemMetadata: Decodable, Sendable {
    let policyDecision: String?
    let policyReasonCode: String?
    let policyRationale: DaemonChatToolPolicyRationale?
    let validationErrorCode: String?
    let validationArgument: String?
    let validationExpected: String?
    let schemaRegistryVersion: String?
    let responseShapingChannel: String?
    let responseShapingProfile: String?
    let responseShapingGuardrailCount: Int?
    let responseShapingInstructionCount: Int?
    let stopReason: String?
    let plannerRepairAttempts: Int?
    let code: String?
    let domain: String?
    let summary: String?
    let primaryAction: String?
    let secondaryAction: String?
    let remediation: DaemonChatMetadataRemediation?
    let personaPolicySource: String?
    let additional: [String: DaemonJSONValue]

    enum CodingKeys: String, CodingKey {
        case policyDecision = "policy_decision"
        case policyReasonCode = "policy_reason_code"
        case policyRationale = "policy_rationale"
        case validationErrorCode = "validation_error_code"
        case validationArgument = "validation_argument"
        case validationExpected = "validation_expected"
        case schemaRegistryVersion = "schema_registry_version"
        case responseShapingChannel = "response_shaping_channel"
        case responseShapingProfile = "response_shaping_profile"
        case responseShapingGuardrailCount = "response_shaping_guardrail_count"
        case responseShapingInstructionCount = "response_shaping_instruction_count"
        case stopReason = "stop_reason"
        case plannerRepairAttempts = "planner_repair_attempts"
        case code
        case domain
        case summary
        case primaryAction = "primary_action"
        case secondaryAction = "secondary_action"
        case remediation
        case personaPolicySource = "persona_policy_source"
    }

    init(
        policyDecision: String? = nil,
        policyReasonCode: String? = nil,
        policyRationale: DaemonChatToolPolicyRationale? = nil,
        validationErrorCode: String? = nil,
        validationArgument: String? = nil,
        validationExpected: String? = nil,
        schemaRegistryVersion: String? = nil,
        responseShapingChannel: String? = nil,
        responseShapingProfile: String? = nil,
        responseShapingGuardrailCount: Int? = nil,
        responseShapingInstructionCount: Int? = nil,
        stopReason: String? = nil,
        plannerRepairAttempts: Int? = nil,
        code: String? = nil,
        domain: String? = nil,
        summary: String? = nil,
        primaryAction: String? = nil,
        secondaryAction: String? = nil,
        remediation: DaemonChatMetadataRemediation? = nil,
        personaPolicySource: String? = nil,
        additional: [String: DaemonJSONValue] = [:]
    ) {
        self.policyDecision = policyDecision
        self.policyReasonCode = policyReasonCode
        self.policyRationale = policyRationale
        self.validationErrorCode = validationErrorCode
        self.validationArgument = validationArgument
        self.validationExpected = validationExpected
        self.schemaRegistryVersion = schemaRegistryVersion
        self.responseShapingChannel = responseShapingChannel
        self.responseShapingProfile = responseShapingProfile
        self.responseShapingGuardrailCount = responseShapingGuardrailCount
        self.responseShapingInstructionCount = responseShapingInstructionCount
        self.stopReason = stopReason
        self.plannerRepairAttempts = plannerRepairAttempts
        self.code = code
        self.domain = domain
        self.summary = summary
        self.primaryAction = primaryAction
        self.secondaryAction = secondaryAction
        self.remediation = remediation
        self.personaPolicySource = personaPolicySource
        self.additional = additional
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        policyDecision = try container.decodeLossyString(forKey: .policyDecision)
        policyReasonCode = try container.decodeLossyString(forKey: .policyReasonCode)
        policyRationale = try container.decodeIfPresent(DaemonChatToolPolicyRationale.self, forKey: .policyRationale)
        validationErrorCode = try container.decodeLossyString(forKey: .validationErrorCode)
        validationArgument = try container.decodeLossyString(forKey: .validationArgument)
        validationExpected = try container.decodeLossyString(forKey: .validationExpected)
        schemaRegistryVersion = try container.decodeLossyString(forKey: .schemaRegistryVersion)
        responseShapingChannel = try container.decodeLossyString(forKey: .responseShapingChannel)
        responseShapingProfile = try container.decodeLossyString(forKey: .responseShapingProfile)
        responseShapingGuardrailCount = try container.decodeLossyInt(forKey: .responseShapingGuardrailCount)
        responseShapingInstructionCount = try container.decodeLossyInt(forKey: .responseShapingInstructionCount)
        stopReason = try container.decodeLossyString(forKey: .stopReason)
        plannerRepairAttempts = try container.decodeLossyInt(forKey: .plannerRepairAttempts)
        code = try container.decodeLossyString(forKey: .code)
        domain = try container.decodeLossyString(forKey: .domain)
        summary = try container.decodeLossyString(forKey: .summary)
        primaryAction = try container.decodeLossyString(forKey: .primaryAction)
        secondaryAction = try container.decodeLossyString(forKey: .secondaryAction)
        remediation = try container.decodeIfPresent(DaemonChatMetadataRemediation.self, forKey: .remediation)
        personaPolicySource = try container.decodeLossyString(forKey: .personaPolicySource)

        var extras = decodeDaemonJSONObject(from: decoder)
        for key in [
            "policy_decision",
            "policy_reason_code",
            "policy_rationale",
            "validation_error_code",
            "validation_argument",
            "validation_expected",
            "schema_registry_version",
            "response_shaping_channel",
            "response_shaping_profile",
            "response_shaping_guardrail_count",
            "response_shaping_instruction_count",
            "stop_reason",
            "planner_repair_attempts",
            "code",
            "domain",
            "summary",
            "primary_action",
            "secondary_action",
            "remediation",
            "persona_policy_source"
        ] {
            extras.removeValue(forKey: key)
        }
        additional = extras
    }

    var allValues: [String: DaemonJSONValue] {
        var values = additional
        if let policyDecision = trimmedDaemonString(policyDecision) { values["policy_decision"] = .string(policyDecision) }
        if let policyReasonCode = trimmedDaemonString(policyReasonCode) { values["policy_reason_code"] = .string(policyReasonCode) }
        if let policyRationale { values["policy_rationale"] = .object(policyRationale.allValues) }
        if let validationErrorCode = trimmedDaemonString(validationErrorCode) { values["validation_error_code"] = .string(validationErrorCode) }
        if let validationArgument = trimmedDaemonString(validationArgument) { values["validation_argument"] = .string(validationArgument) }
        if let validationExpected = trimmedDaemonString(validationExpected) { values["validation_expected"] = .string(validationExpected) }
        if let schemaRegistryVersion = trimmedDaemonString(schemaRegistryVersion) { values["schema_registry_version"] = .string(schemaRegistryVersion) }
        if let responseShapingChannel = trimmedDaemonString(responseShapingChannel) { values["response_shaping_channel"] = .string(responseShapingChannel) }
        if let responseShapingProfile = trimmedDaemonString(responseShapingProfile) { values["response_shaping_profile"] = .string(responseShapingProfile) }
        if let responseShapingGuardrailCount { values["response_shaping_guardrail_count"] = .number(Double(responseShapingGuardrailCount)) }
        if let responseShapingInstructionCount { values["response_shaping_instruction_count"] = .number(Double(responseShapingInstructionCount)) }
        if let stopReason = trimmedDaemonString(stopReason) { values["stop_reason"] = .string(stopReason) }
        if let plannerRepairAttempts { values["planner_repair_attempts"] = .number(Double(plannerRepairAttempts)) }
        if let code = trimmedDaemonString(code) { values["code"] = .string(code) }
        if let domain = trimmedDaemonString(domain) { values["domain"] = .string(domain) }
        if let summary = trimmedDaemonString(summary) { values["summary"] = .string(summary) }
        if let primaryAction = trimmedDaemonString(primaryAction) { values["primary_action"] = .string(primaryAction) }
        if let secondaryAction = trimmedDaemonString(secondaryAction) { values["secondary_action"] = .string(secondaryAction) }
        if let remediation { values["remediation"] = .object(remediation.allValues) }
        if let personaPolicySource = trimmedDaemonString(personaPolicySource) { values["persona_policy_source"] = .string(personaPolicySource) }
        return values
    }

    subscript(key: String) -> DaemonJSONValue? {
        allValues[key]
    }

    var isEmpty: Bool {
        allValues.isEmpty
    }
}

struct DaemonRealtimeEventPayload: Decodable, Sendable {
    let workspaceID: String?
    let taskID: String?
    let runID: String?
    let state: String?
    let taskState: String?
    let runState: String?
    let lifecycleState: String?
    let lifecycleSource: String?
    let lastError: String?
    let signalType: String?
    let accepted: Bool?
    let reason: String?
    let cancelled: Bool?
    let alreadyTerminal: Bool?
    let itemID: String?
    let itemIndex: Int?
    let itemType: String?
    let status: String?
    let delta: String?
    let toolName: String?
    let toolCallID: String?
    let callID: String?
    let name: String?
    let arguments: [String: DaemonJSONValue]?
    let output: [String: DaemonJSONValue]?
    let errorCode: String?
    let error: String?
    let metadata: DaemonChatTurnItemMetadata?
    let taskClass: String?
    let provider: String?
    let modelKey: String?
    let assistantEmpty: Bool?
    let itemCount: Int?
    let toolCallCount: Int?
    let approvalCount: Int?
    let approvalRequestID: String?
    let message: String?
    let additional: [String: DaemonJSONValue]

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case taskID = "task_id"
        case runID = "run_id"
        case state
        case taskState = "task_state"
        case runState = "run_state"
        case lifecycleState = "lifecycle_state"
        case lifecycleSource = "lifecycle_source"
        case lastError = "last_error"
        case signalType = "signal_type"
        case accepted
        case reason
        case cancelled
        case alreadyTerminal = "already_terminal"
        case itemID = "item_id"
        case itemIndex = "item_index"
        case itemType = "item_type"
        case status
        case delta
        case toolName = "tool_name"
        case toolCallID = "tool_call_id"
        case callID = "call_id"
        case name
        case arguments
        case output
        case errorCode = "error_code"
        case error
        case metadata
        case taskClass = "task_class"
        case provider
        case modelKey = "model_key"
        case assistantEmpty = "assistant_empty"
        case itemCount = "item_count"
        case toolCallCount = "tool_call_count"
        case approvalCount = "approval_count"
        case approvalRequestID = "approval_request_id"
        case message
    }

    init(
        workspaceID: String? = nil,
        taskID: String? = nil,
        runID: String? = nil,
        state: String? = nil,
        taskState: String? = nil,
        runState: String? = nil,
        lifecycleState: String? = nil,
        lifecycleSource: String? = nil,
        lastError: String? = nil,
        signalType: String? = nil,
        accepted: Bool? = nil,
        reason: String? = nil,
        cancelled: Bool? = nil,
        alreadyTerminal: Bool? = nil,
        itemID: String? = nil,
        itemIndex: Int? = nil,
        itemType: String? = nil,
        status: String? = nil,
        delta: String? = nil,
        toolName: String? = nil,
        toolCallID: String? = nil,
        callID: String? = nil,
        name: String? = nil,
        arguments: [String: DaemonJSONValue]? = nil,
        output: [String: DaemonJSONValue]? = nil,
        errorCode: String? = nil,
        error: String? = nil,
        metadata: DaemonChatTurnItemMetadata? = nil,
        taskClass: String? = nil,
        provider: String? = nil,
        modelKey: String? = nil,
        assistantEmpty: Bool? = nil,
        itemCount: Int? = nil,
        toolCallCount: Int? = nil,
        approvalCount: Int? = nil,
        approvalRequestID: String? = nil,
        message: String? = nil,
        additional: [String: DaemonJSONValue] = [:]
    ) {
        self.workspaceID = workspaceID
        self.taskID = taskID
        self.runID = runID
        self.state = state
        self.taskState = taskState
        self.runState = runState
        self.lifecycleState = lifecycleState
        self.lifecycleSource = lifecycleSource
        self.lastError = lastError
        self.signalType = signalType
        self.accepted = accepted
        self.reason = reason
        self.cancelled = cancelled
        self.alreadyTerminal = alreadyTerminal
        self.itemID = itemID
        self.itemIndex = itemIndex
        self.itemType = itemType
        self.status = status
        self.delta = delta
        self.toolName = toolName
        self.toolCallID = toolCallID
        self.callID = callID
        self.name = name
        self.arguments = arguments
        self.output = output
        self.errorCode = errorCode
        self.error = error
        self.metadata = metadata
        self.taskClass = taskClass
        self.provider = provider
        self.modelKey = modelKey
        self.assistantEmpty = assistantEmpty
        self.itemCount = itemCount
        self.toolCallCount = toolCallCount
        self.approvalCount = approvalCount
        self.approvalRequestID = approvalRequestID
        self.message = message
        self.additional = additional
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        workspaceID = try container.decodeLossyString(forKey: .workspaceID)
        taskID = try container.decodeLossyString(forKey: .taskID)
        runID = try container.decodeLossyString(forKey: .runID)
        state = try container.decodeLossyString(forKey: .state)
        taskState = try container.decodeLossyString(forKey: .taskState)
        runState = try container.decodeLossyString(forKey: .runState)
        lifecycleState = try container.decodeLossyString(forKey: .lifecycleState)
        lifecycleSource = try container.decodeLossyString(forKey: .lifecycleSource)
        lastError = try container.decodeLossyString(forKey: .lastError)
        signalType = try container.decodeLossyString(forKey: .signalType)
        accepted = try container.decodeLossyBool(forKey: .accepted)
        reason = try container.decodeLossyString(forKey: .reason)
        cancelled = try container.decodeLossyBool(forKey: .cancelled)
        alreadyTerminal = try container.decodeLossyBool(forKey: .alreadyTerminal)
        itemID = try container.decodeLossyString(forKey: .itemID)
        itemIndex = try container.decodeLossyInt(forKey: .itemIndex)
        itemType = try container.decodeLossyString(forKey: .itemType)
        status = try container.decodeLossyString(forKey: .status)
        delta = try container.decodeLossyString(forKey: .delta)
        toolName = try container.decodeLossyString(forKey: .toolName)
        toolCallID = try container.decodeLossyString(forKey: .toolCallID)
        callID = try container.decodeLossyString(forKey: .callID)
        name = try container.decodeLossyString(forKey: .name)
        arguments = try container.decodeIfPresent([String: DaemonJSONValue].self, forKey: .arguments)
        output = try container.decodeIfPresent([String: DaemonJSONValue].self, forKey: .output)
        errorCode = try container.decodeLossyString(forKey: .errorCode)
        error = try container.decodeLossyString(forKey: .error)
        metadata = try container.decodeIfPresent(DaemonChatTurnItemMetadata.self, forKey: .metadata)
        taskClass = try container.decodeLossyString(forKey: .taskClass)
        provider = try container.decodeLossyString(forKey: .provider)
        modelKey = try container.decodeLossyString(forKey: .modelKey)
        assistantEmpty = try container.decodeLossyBool(forKey: .assistantEmpty)
        itemCount = try container.decodeLossyInt(forKey: .itemCount)
        toolCallCount = try container.decodeLossyInt(forKey: .toolCallCount)
        approvalCount = try container.decodeLossyInt(forKey: .approvalCount)
        approvalRequestID = try container.decodeLossyString(forKey: .approvalRequestID)
        message = try container.decodeLossyString(forKey: .message)

        var extras = decodeDaemonJSONObject(from: decoder)
        for key in [
            "workspace_id",
            "task_id",
            "run_id",
            "state",
            "task_state",
            "run_state",
            "lifecycle_state",
            "lifecycle_source",
            "last_error",
            "signal_type",
            "accepted",
            "reason",
            "cancelled",
            "already_terminal",
            "item_id",
            "item_index",
            "item_type",
            "status",
            "delta",
            "tool_name",
            "tool_call_id",
            "call_id",
            "name",
            "arguments",
            "output",
            "error_code",
            "error",
            "metadata",
            "task_class",
            "provider",
            "model_key",
            "assistant_empty",
            "item_count",
            "tool_call_count",
            "approval_count",
            "approval_request_id",
            "message"
        ] {
            extras.removeValue(forKey: key)
        }
        additional = extras
    }
}

struct DaemonUIStatusMappedConnector: Decodable, Sendable {
    let connectorID: String?
    let enabled: Bool?
    let priority: Int?
    let configured: Bool?
    let status: String?
    let summary: String?
    let additional: [String: DaemonJSONValue]

    enum CodingKeys: String, CodingKey {
        case connectorID = "connector_id"
        case enabled
        case priority
        case configured
        case status
        case summary
    }

    init(
        connectorID: String? = nil,
        enabled: Bool? = nil,
        priority: Int? = nil,
        configured: Bool? = nil,
        status: String? = nil,
        summary: String? = nil,
        additional: [String: DaemonJSONValue] = [:]
    ) {
        self.connectorID = connectorID
        self.enabled = enabled
        self.priority = priority
        self.configured = configured
        self.status = status
        self.summary = summary
        self.additional = additional
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        connectorID = try container.decodeLossyString(forKey: .connectorID)
        enabled = try container.decodeLossyBool(forKey: .enabled)
        priority = try container.decodeLossyInt(forKey: .priority)
        configured = try container.decodeLossyBool(forKey: .configured)
        status = try container.decodeLossyString(forKey: .status)
        summary = try container.decodeLossyString(forKey: .summary)

        var extras = decodeDaemonJSONObject(from: decoder)
        for key in ["connector_id", "enabled", "priority", "configured", "status", "summary"] {
            extras.removeValue(forKey: key)
        }
        additional = extras
    }

    var allValues: [String: DaemonJSONValue] {
        var values = additional
        if let connectorID = trimmedDaemonString(connectorID) { values["connector_id"] = .string(connectorID) }
        if let enabled { values["enabled"] = .bool(enabled) }
        if let priority { values["priority"] = .number(Double(priority)) }
        if let configured { values["configured"] = .bool(configured) }
        if let status = trimmedDaemonString(status) { values["status"] = .string(status) }
        if let summary = trimmedDaemonString(summary) { values["summary"] = .string(summary) }
        return values
    }
}

struct DaemonUIStatusConfiguration: Decodable, Sendable {
    let enabled: Bool?
    let transport: String?
    let mode: String?
    let number: String?
    let scope: String?
    let statusReason: String?
    let fallbackPolicy: String?
    let primaryConnectorID: String?
    let mappedConnectorIDs: [String]
    let enabledConnectorIDs: [String]
    let mappedConnectors: [DaemonUIStatusMappedConnector]
    let boundConnector: String?
    let boundToChannel: Bool?
    let ingestSourceScope: String?
    let ingestUpdatedAt: String?
    let ingestLastError: String?
    let credentialsConfigured: Bool?
    let permissionState: String?
    let executePathProbeReady: Bool?
    let executePathProbeStatusCode: Int?
    let executePathProbeError: String?
    let cloudflaredAvailable: Bool?
    let cloudflaredBinaryPath: String?
    let cloudflaredDryRun: Bool?
    let cloudflaredExitCode: Int?
    let cloudflaredError: String?
    let localIngestBridgeReady: Bool?
    let additional: [String: DaemonJSONValue]

    enum CodingKeys: String, CodingKey {
        case enabled
        case transport
        case mode
        case number
        case scope
        case statusReason = "status_reason"
        case fallbackPolicy = "fallback_policy"
        case primaryConnectorID = "primary_connector_id"
        case mappedConnectorIDs = "mapped_connector_ids"
        case enabledConnectorIDs = "enabled_connector_ids"
        case mappedConnectors = "mapped_connectors"
        case boundConnector = "bound_connector"
        case boundToChannel = "bound_to_channel"
        case ingestSourceScope = "ingest_source_scope"
        case ingestUpdatedAt = "ingest_updated_at"
        case ingestLastError = "ingest_last_error"
        case credentialsConfigured = "credentials_configured"
        case permissionState = "permission_state"
        case executePathProbeReady = "execute_path_probe_ready"
        case executePathProbeStatusCode = "execute_path_probe_status_code"
        case executePathProbeError = "execute_path_probe_error"
        case cloudflaredAvailable = "cloudflared_available"
        case cloudflaredBinaryPath = "cloudflared_binary_path"
        case cloudflaredDryRun = "cloudflared_dry_run"
        case cloudflaredExitCode = "cloudflared_exit_code"
        case cloudflaredError = "cloudflared_error"
        case localIngestBridgeReady = "local_ingest_bridge_ready"
    }

    init(
        enabled: Bool? = nil,
        transport: String? = nil,
        mode: String? = nil,
        number: String? = nil,
        scope: String? = nil,
        statusReason: String? = nil,
        fallbackPolicy: String? = nil,
        primaryConnectorID: String? = nil,
        mappedConnectorIDs: [String] = [],
        enabledConnectorIDs: [String] = [],
        mappedConnectors: [DaemonUIStatusMappedConnector] = [],
        boundConnector: String? = nil,
        boundToChannel: Bool? = nil,
        ingestSourceScope: String? = nil,
        ingestUpdatedAt: String? = nil,
        ingestLastError: String? = nil,
        credentialsConfigured: Bool? = nil,
        permissionState: String? = nil,
        executePathProbeReady: Bool? = nil,
        executePathProbeStatusCode: Int? = nil,
        executePathProbeError: String? = nil,
        cloudflaredAvailable: Bool? = nil,
        cloudflaredBinaryPath: String? = nil,
        cloudflaredDryRun: Bool? = nil,
        cloudflaredExitCode: Int? = nil,
        cloudflaredError: String? = nil,
        localIngestBridgeReady: Bool? = nil,
        additional: [String: DaemonJSONValue] = [:]
    ) {
        self.enabled = enabled
        self.transport = transport
        self.mode = mode
        self.number = number
        self.scope = scope
        self.statusReason = statusReason
        self.fallbackPolicy = fallbackPolicy
        self.primaryConnectorID = primaryConnectorID
        self.mappedConnectorIDs = mappedConnectorIDs
        self.enabledConnectorIDs = enabledConnectorIDs
        self.mappedConnectors = mappedConnectors
        self.boundConnector = boundConnector
        self.boundToChannel = boundToChannel
        self.ingestSourceScope = ingestSourceScope
        self.ingestUpdatedAt = ingestUpdatedAt
        self.ingestLastError = ingestLastError
        self.credentialsConfigured = credentialsConfigured
        self.permissionState = permissionState
        self.executePathProbeReady = executePathProbeReady
        self.executePathProbeStatusCode = executePathProbeStatusCode
        self.executePathProbeError = executePathProbeError
        self.cloudflaredAvailable = cloudflaredAvailable
        self.cloudflaredBinaryPath = cloudflaredBinaryPath
        self.cloudflaredDryRun = cloudflaredDryRun
        self.cloudflaredExitCode = cloudflaredExitCode
        self.cloudflaredError = cloudflaredError
        self.localIngestBridgeReady = localIngestBridgeReady
        self.additional = additional
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        enabled = try container.decodeLossyBool(forKey: .enabled)
        transport = try container.decodeLossyString(forKey: .transport)
        mode = try container.decodeLossyString(forKey: .mode)
        number = try container.decodeLossyString(forKey: .number)
        scope = try container.decodeLossyString(forKey: .scope)
        statusReason = try container.decodeLossyString(forKey: .statusReason)
        fallbackPolicy = try container.decodeLossyString(forKey: .fallbackPolicy)
        primaryConnectorID = try container.decodeLossyString(forKey: .primaryConnectorID)
        mappedConnectorIDs = try container.decodeLossyStringArray(forKey: .mappedConnectorIDs) ?? []
        enabledConnectorIDs = try container.decodeLossyStringArray(forKey: .enabledConnectorIDs) ?? []
        mappedConnectors = try container.decodeIfPresent([DaemonUIStatusMappedConnector].self, forKey: .mappedConnectors) ?? []
        boundConnector = try container.decodeLossyString(forKey: .boundConnector)
        boundToChannel = try container.decodeLossyBool(forKey: .boundToChannel)
        ingestSourceScope = try container.decodeLossyString(forKey: .ingestSourceScope)
        ingestUpdatedAt = try container.decodeLossyString(forKey: .ingestUpdatedAt)
        ingestLastError = try container.decodeLossyString(forKey: .ingestLastError)
        credentialsConfigured = try container.decodeLossyBool(forKey: .credentialsConfigured)
        permissionState = try container.decodeLossyString(forKey: .permissionState)
        executePathProbeReady = try container.decodeLossyBool(forKey: .executePathProbeReady)
        executePathProbeStatusCode = try container.decodeLossyInt(forKey: .executePathProbeStatusCode)
        executePathProbeError = try container.decodeLossyString(forKey: .executePathProbeError)
        cloudflaredAvailable = try container.decodeLossyBool(forKey: .cloudflaredAvailable)
        cloudflaredBinaryPath = try container.decodeLossyString(forKey: .cloudflaredBinaryPath)
        cloudflaredDryRun = try container.decodeLossyBool(forKey: .cloudflaredDryRun)
        cloudflaredExitCode = try container.decodeLossyInt(forKey: .cloudflaredExitCode)
        cloudflaredError = try container.decodeLossyString(forKey: .cloudflaredError)
        localIngestBridgeReady = try container.decodeLossyBool(forKey: .localIngestBridgeReady)

        var extras = decodeDaemonJSONObject(from: decoder)
        for key in [
            "enabled",
            "transport",
            "mode",
            "number",
            "scope",
            "status_reason",
            "fallback_policy",
            "primary_connector_id",
            "mapped_connector_ids",
            "enabled_connector_ids",
            "mapped_connectors",
            "bound_connector",
            "bound_to_channel",
            "ingest_source_scope",
            "ingest_updated_at",
            "ingest_last_error",
            "credentials_configured",
            "permission_state",
            "execute_path_probe_ready",
            "execute_path_probe_status_code",
            "execute_path_probe_error",
            "cloudflared_available",
            "cloudflared_binary_path",
            "cloudflared_dry_run",
            "cloudflared_exit_code",
            "cloudflared_error",
            "local_ingest_bridge_ready"
        ] {
            extras.removeValue(forKey: key)
        }
        additional = extras
    }

    var allValues: [String: DaemonJSONValue] {
        var values = additional
        if let enabled { values["enabled"] = .bool(enabled) }
        if let transport = trimmedDaemonString(transport) { values["transport"] = .string(transport) }
        if let mode = trimmedDaemonString(mode) { values["mode"] = .string(mode) }
        if let number = trimmedDaemonString(number) { values["number"] = .string(number) }
        if let scope = trimmedDaemonString(scope) { values["scope"] = .string(scope) }
        if let statusReason = trimmedDaemonString(statusReason) { values["status_reason"] = .string(statusReason) }
        if let fallbackPolicy = trimmedDaemonString(fallbackPolicy) { values["fallback_policy"] = .string(fallbackPolicy) }
        if let primaryConnectorID = trimmedDaemonString(primaryConnectorID) { values["primary_connector_id"] = .string(primaryConnectorID) }
        if !mappedConnectorIDs.isEmpty {
            values["mapped_connector_ids"] = .array(mappedConnectorIDs.map(DaemonJSONValue.string))
        }
        if !enabledConnectorIDs.isEmpty {
            values["enabled_connector_ids"] = .array(enabledConnectorIDs.map(DaemonJSONValue.string))
        }
        if !mappedConnectors.isEmpty {
            values["mapped_connectors"] = .array(mappedConnectors.map { .object($0.allValues) })
        }
        if let boundConnector = trimmedDaemonString(boundConnector) { values["bound_connector"] = .string(boundConnector) }
        if let boundToChannel { values["bound_to_channel"] = .bool(boundToChannel) }
        if let ingestSourceScope = trimmedDaemonString(ingestSourceScope) { values["ingest_source_scope"] = .string(ingestSourceScope) }
        if let ingestUpdatedAt = trimmedDaemonString(ingestUpdatedAt) { values["ingest_updated_at"] = .string(ingestUpdatedAt) }
        if let ingestLastError = trimmedDaemonString(ingestLastError) { values["ingest_last_error"] = .string(ingestLastError) }
        if let credentialsConfigured { values["credentials_configured"] = .bool(credentialsConfigured) }
        if let permissionState = trimmedDaemonString(permissionState) { values["permission_state"] = .string(permissionState) }
        if let executePathProbeReady { values["execute_path_probe_ready"] = .bool(executePathProbeReady) }
        if let executePathProbeStatusCode { values["execute_path_probe_status_code"] = .number(Double(executePathProbeStatusCode)) }
        if let executePathProbeError = trimmedDaemonString(executePathProbeError) { values["execute_path_probe_error"] = .string(executePathProbeError) }
        if let cloudflaredAvailable { values["cloudflared_available"] = .bool(cloudflaredAvailable) }
        if let cloudflaredBinaryPath = trimmedDaemonString(cloudflaredBinaryPath) { values["cloudflared_binary_path"] = .string(cloudflaredBinaryPath) }
        if let cloudflaredDryRun { values["cloudflared_dry_run"] = .bool(cloudflaredDryRun) }
        if let cloudflaredExitCode { values["cloudflared_exit_code"] = .number(Double(cloudflaredExitCode)) }
        if let cloudflaredError = trimmedDaemonString(cloudflaredError) { values["cloudflared_error"] = .string(cloudflaredError) }
        if let localIngestBridgeReady { values["local_ingest_bridge_ready"] = .bool(localIngestBridgeReady) }
        return values
    }

    subscript(key: String) -> DaemonJSONValue? {
        allValues[key]
    }
}

struct DaemonUIStatusTestOperationDetails: Decodable, Sendable {
    let pluginID: String?
    let workerRegistered: Bool?
    let workerState: String?
    let configured: Bool?
    let credentialsConfigured: Bool?
    let endpoint: String?
    let smsNumber: String?
    let voiceNumber: String?
    let executePathReady: Bool?
    let executePathProbeStatusCode: Int?
    let executePathProbeError: String?
    let available: Bool?
    let binaryPath: String?
    let dryRun: Bool?
    let stdout: String?
    let stderr: String?
    let probeError: String?
    let additional: [String: DaemonJSONValue]

    enum CodingKeys: String, CodingKey {
        case pluginID = "plugin_id"
        case workerRegistered = "worker_registered"
        case workerState = "worker_state"
        case configured
        case credentialsConfigured = "credentials_configured"
        case endpoint
        case smsNumber = "sms_number"
        case voiceNumber = "voice_number"
        case executePathReady = "execute_path_ready"
        case executePathProbeStatusCode = "execute_path_probe_status_code"
        case executePathProbeError = "execute_path_probe_error"
        case available
        case binaryPath = "binary_path"
        case dryRun = "dry_run"
        case stdout
        case stderr
        case probeError = "probe_error"
    }

    init(
        pluginID: String? = nil,
        workerRegistered: Bool? = nil,
        workerState: String? = nil,
        configured: Bool? = nil,
        credentialsConfigured: Bool? = nil,
        endpoint: String? = nil,
        smsNumber: String? = nil,
        voiceNumber: String? = nil,
        executePathReady: Bool? = nil,
        executePathProbeStatusCode: Int? = nil,
        executePathProbeError: String? = nil,
        available: Bool? = nil,
        binaryPath: String? = nil,
        dryRun: Bool? = nil,
        stdout: String? = nil,
        stderr: String? = nil,
        probeError: String? = nil,
        additional: [String: DaemonJSONValue] = [:]
    ) {
        self.pluginID = pluginID
        self.workerRegistered = workerRegistered
        self.workerState = workerState
        self.configured = configured
        self.credentialsConfigured = credentialsConfigured
        self.endpoint = endpoint
        self.smsNumber = smsNumber
        self.voiceNumber = voiceNumber
        self.executePathReady = executePathReady
        self.executePathProbeStatusCode = executePathProbeStatusCode
        self.executePathProbeError = executePathProbeError
        self.available = available
        self.binaryPath = binaryPath
        self.dryRun = dryRun
        self.stdout = stdout
        self.stderr = stderr
        self.probeError = probeError
        self.additional = additional
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        pluginID = try container.decodeLossyString(forKey: .pluginID)
        workerRegistered = try container.decodeLossyBool(forKey: .workerRegistered)
        workerState = try container.decodeLossyString(forKey: .workerState)
        configured = try container.decodeLossyBool(forKey: .configured)
        credentialsConfigured = try container.decodeLossyBool(forKey: .credentialsConfigured)
        endpoint = try container.decodeLossyString(forKey: .endpoint)
        smsNumber = try container.decodeLossyString(forKey: .smsNumber)
        voiceNumber = try container.decodeLossyString(forKey: .voiceNumber)
        executePathReady = try container.decodeLossyBool(forKey: .executePathReady)
        executePathProbeStatusCode = try container.decodeLossyInt(forKey: .executePathProbeStatusCode)
        executePathProbeError = try container.decodeLossyString(forKey: .executePathProbeError)
        available = try container.decodeLossyBool(forKey: .available)
        binaryPath = try container.decodeLossyString(forKey: .binaryPath)
        dryRun = try container.decodeLossyBool(forKey: .dryRun)
        stdout = try container.decodeLossyString(forKey: .stdout)
        stderr = try container.decodeLossyString(forKey: .stderr)
        probeError = try container.decodeLossyString(forKey: .probeError)

        var extras = decodeDaemonJSONObject(from: decoder)
        for key in [
            "plugin_id",
            "worker_registered",
            "worker_state",
            "configured",
            "credentials_configured",
            "endpoint",
            "sms_number",
            "voice_number",
            "execute_path_ready",
            "execute_path_probe_status_code",
            "execute_path_probe_error",
            "available",
            "binary_path",
            "dry_run",
            "stdout",
            "stderr",
            "probe_error"
        ] {
            extras.removeValue(forKey: key)
        }
        additional = extras
    }

    var allValues: [String: DaemonJSONValue] {
        var values = additional
        if let pluginID = trimmedDaemonString(pluginID) { values["plugin_id"] = .string(pluginID) }
        if let workerRegistered { values["worker_registered"] = .bool(workerRegistered) }
        if let workerState = trimmedDaemonString(workerState) { values["worker_state"] = .string(workerState) }
        if let configured { values["configured"] = .bool(configured) }
        if let credentialsConfigured { values["credentials_configured"] = .bool(credentialsConfigured) }
        if let endpoint = trimmedDaemonString(endpoint) { values["endpoint"] = .string(endpoint) }
        if let smsNumber = trimmedDaemonString(smsNumber) { values["sms_number"] = .string(smsNumber) }
        if let voiceNumber = trimmedDaemonString(voiceNumber) { values["voice_number"] = .string(voiceNumber) }
        if let executePathReady { values["execute_path_ready"] = .bool(executePathReady) }
        if let executePathProbeStatusCode { values["execute_path_probe_status_code"] = .number(Double(executePathProbeStatusCode)) }
        if let executePathProbeError = trimmedDaemonString(executePathProbeError) { values["execute_path_probe_error"] = .string(executePathProbeError) }
        if let available { values["available"] = .bool(available) }
        if let binaryPath = trimmedDaemonString(binaryPath) { values["binary_path"] = .string(binaryPath) }
        if let dryRun { values["dry_run"] = .bool(dryRun) }
        if let stdout = trimmedDaemonString(stdout) { values["stdout"] = .string(stdout) }
        if let stderr = trimmedDaemonString(stderr) { values["stderr"] = .string(stderr) }
        if let probeError = trimmedDaemonString(probeError) { values["probe_error"] = .string(probeError) }
        return values
    }
}
