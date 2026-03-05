import Foundation
import SwiftUI

public enum DaemonStatus: Equatable {
    case unknown
    case running
    case stopped
    case missing
    case broken

    public var label: String {
        switch self {
        case .unknown:
            return "Daemon: Unknown"
        case .running:
            return "Daemon: Running"
        case .stopped:
            return "Daemon: Stopped"
        case .missing:
            return "Daemon: Not Installed"
        case .broken:
            return "Daemon: Needs Repair"
        }
    }

    public var symbolName: String {
        switch self {
        case .unknown:
            return "questionmark.circle"
        case .running:
            return "checkmark.circle.fill"
        case .stopped:
            return "pause.circle"
        case .missing:
            return "exclamationmark.triangle"
        case .broken:
            return "wrench.and.screwdriver"
        }
    }

    public var tint: Color {
        switch self {
        case .unknown:
            return .secondary
        case .running:
            return .green
        case .stopped:
            return .orange
        case .missing, .broken:
            return .red
        }
    }
}

public enum ConnectionStatus: Equatable {
    case connected
    case disconnected
    case degraded

    public var label: String {
        switch self {
        case .connected:
            return "App Connection: Connected"
        case .disconnected:
            return "App Connection: Disconnected"
        case .degraded:
            return "App Connection: Degraded"
        }
    }

    public var symbolName: String {
        switch self {
        case .connected:
            return "bolt.horizontal.circle.fill"
        case .disconnected:
            return "bolt.slash.circle"
        case .degraded:
            return "exclamationmark.circle"
        }
    }

    public var tint: Color {
        switch self {
        case .connected:
            return .green
        case .disconnected:
            return .secondary
        case .degraded:
            return .orange
        }
    }
}

public enum AppInformationDensityMode: String, CaseIterable, Sendable, Equatable {
    case simple
    case advanced

    public var title: String {
        switch self {
        case .simple:
            return "Simple"
        case .advanced:
            return "Advanced"
        }
    }

    public var subtitle: String {
        switch self {
        case .simple:
            return "Focus on primary outcomes and actions."
        case .advanced:
            return "Show full operator metadata and internals."
        }
    }

    public var symbolName: String {
        switch self {
        case .simple:
            return "text.justify.left"
        case .advanced:
            return "text.justify"
        }
    }
}

public struct DrillInNavigationContext: Sendable, Equatable {
    public let sourceSection: AppSection
    public let destinationSection: AppSection
    public let chips: [String]

    public init(
        sourceSection: AppSection,
        destinationSection: AppSection,
        chips: [String] = []
    ) {
        self.sourceSection = sourceSection
        self.destinationSection = destinationSection
        self.chips = chips
    }
}

public enum DaemonControlAuthState: String, Equatable {
    case unknown
    case configured
    case missing
}

public enum ChatMessageRole: Sendable {
    case user
    case assistant

    var label: String {
        switch self {
        case .user:
            return "You"
        case .assistant:
            return "Assistant"
        }
    }
}

public struct ChatMessageItem: Identifiable, Sendable {
    public let id: UUID
    public let role: ChatMessageRole
    public let content: String
    public let includeInDaemonContext: Bool

    public init(
        id: UUID = UUID(),
        role: ChatMessageRole,
        content: String,
        includeInDaemonContext: Bool = true
    ) {
        self.id = id
        self.role = role
        self.content = content
        self.includeInDaemonContext = includeInDaemonContext
    }
}

public enum InspectLogStatus: Equatable, Sendable {
    case success
    case failure
    case running

    var label: String {
        switch self {
        case .success:
            return "Success"
        case .failure:
            return "Failure"
        case .running:
            return "Running"
        }
    }

    var symbolName: String {
        switch self {
        case .success:
            return "checkmark.circle.fill"
        case .failure:
            return "xmark.octagon.fill"
        case .running:
            return "clock.arrow.circlepath"
        }
    }

    var tint: Color {
        switch self {
        case .success:
            return .green
        case .failure:
            return .red
        case .running:
            return .orange
        }
    }
}

public struct InspectLogItem: Identifiable, Sendable {
    public let id: String
    public let timestamp: Date
    public let createdAtRaw: String
    public let event: String
    public let status: InspectLogStatus
    public let inputSummary: String
    public let outputSummary: String
    public let metadataSummary: String
    public let taskID: String?
    public let runID: String?
    public let stepID: String?
    public let correlationID: String?
    public let route: WorkflowRouteContext

    public init(
        id: String = UUID().uuidString,
        timestamp: Date,
        createdAtRaw: String,
        event: String,
        status: InspectLogStatus,
        inputSummary: String,
        outputSummary: String,
        metadataSummary: String,
        taskID: String? = nil,
        runID: String? = nil,
        stepID: String? = nil,
        correlationID: String? = nil,
        route: WorkflowRouteContext = WorkflowRouteContext()
    ) {
        self.id = id
        self.timestamp = timestamp
        self.createdAtRaw = createdAtRaw
        self.event = event
        self.status = status
        self.inputSummary = inputSummary
        self.outputSummary = outputSummary
        self.metadataSummary = metadataSummary
        self.taskID = taskID
        self.runID = runID
        self.stepID = stepID
        self.correlationID = correlationID
        self.route = route
    }

    public var hasCrossViewContext: Bool {
        taskID != nil || runID != nil || stepID != nil || correlationID != nil
    }
}

public enum ChannelCardStatus: Sendable, Equatable {
    case active
    case degraded
    case setupRequired

    var label: String {
        switch self {
        case .active:
            return "Ready"
        case .degraded:
            return "Degraded"
        case .setupRequired:
            return "Blocked"
        }
    }

    var symbolName: String {
        switch self {
        case .active:
            return "checkmark.circle.fill"
        case .degraded:
            return "exclamationmark.triangle.fill"
        case .setupRequired:
            return "wrench.and.screwdriver.fill"
        }
    }

    var tint: Color {
        switch self {
        case .active:
            return .green
        case .degraded:
            return .orange
        case .setupRequired:
            return .secondary
        }
    }
}

public struct DiagnosticsActionItem: Identifiable, Sendable {
    public let id: String
    public let title: String
    public let intent: String
    public let destination: String?
    public let parameters: [String: String]
    public let enabled: Bool
    public let recommended: Bool
    public let reason: String?

    public init(
        id: String,
        title: String,
        intent: String,
        destination: String?,
        parameters: [String: String] = [:],
        enabled: Bool,
        recommended: Bool,
        reason: String?
    ) {
        self.id = id
        self.title = title
        self.intent = intent
        self.destination = destination
        self.parameters = parameters
        self.enabled = enabled
        self.recommended = recommended
        self.reason = reason
    }
}

public enum ConfigurationDraftValueKind: Sendable, Equatable {
    case string
    case number
    case bool
    case null
    case object
    case array

    var supportsInlineEditing: Bool {
        switch self {
        case .string, .number, .bool, .null:
            return true
        case .object, .array:
            return false
        }
    }

    var label: String {
        switch self {
        case .string:
            return "string"
        case .number:
            return "number"
        case .bool:
            return "bool"
        case .null:
            return "null"
        case .object:
            return "object"
        case .array:
            return "array"
        }
    }
}

public struct ConfigurationFieldDescriptorItem: Identifiable, Sendable, Equatable {
    public let key: String
    public let label: String
    public let required: Bool
    public let enumOptions: [String]
    public let editable: Bool
    public let secret: Bool
    public let writeOnly: Bool
    public let helpText: String?
    public let draftKind: ConfigurationDraftValueKind

    public var id: String { key }
}

public struct ChannelCardItem: Identifiable, Sendable {
    public let id: String
    public let name: String
    public let logicalChannelID: String
    public let mappedConnectorIDs: [String]
    public let enabledConnectorIDs: [String]
    public let primaryConnectorID: String?
    public let declaredCapabilities: [String]
    public let status: ChannelCardStatus
    public let summary: String
    public let details: [String: String]
    public let editableConfiguration: [String: String]
    public let editableConfigurationKinds: [String: ConfigurationDraftValueKind]
    public let configurationFieldDescriptors: [ConfigurationFieldDescriptorItem]
    public let readOnlyConfiguration: [String: String]
    public let actions: [DiagnosticsActionItem]
    public let unavailableActionReason: String
    public var isExpanded: Bool

    public init(
        id: String,
        name: String,
        logicalChannelID: String? = nil,
        mappedConnectorIDs: [String] = [],
        enabledConnectorIDs: [String] = [],
        primaryConnectorID: String? = nil,
        declaredCapabilities: [String] = [],
        status: ChannelCardStatus,
        summary: String,
        details: [String: String],
        editableConfiguration: [String: String],
        editableConfigurationKinds: [String: ConfigurationDraftValueKind],
        configurationFieldDescriptors: [ConfigurationFieldDescriptorItem] = [],
        readOnlyConfiguration: [String: String],
        actions: [DiagnosticsActionItem],
        unavailableActionReason: String,
        isExpanded: Bool = false
    ) {
        self.id = id
        self.name = name
        self.logicalChannelID = logicalChannelID ?? id
        self.mappedConnectorIDs = mappedConnectorIDs
        self.enabledConnectorIDs = enabledConnectorIDs
        self.primaryConnectorID = primaryConnectorID
        self.declaredCapabilities = declaredCapabilities
        self.status = status
        self.summary = summary
        self.details = details
        self.editableConfiguration = editableConfiguration
        self.editableConfigurationKinds = editableConfigurationKinds
        self.configurationFieldDescriptors = configurationFieldDescriptors
        self.readOnlyConfiguration = readOnlyConfiguration
        self.actions = actions
        self.unavailableActionReason = unavailableActionReason
        self.isExpanded = isExpanded
    }
}

public enum ConnectorHealthStatus: Sendable, Equatable {
    case ready
    case needsPermission
    case unavailable

    var label: String {
        switch self {
        case .ready:
            return "Ready"
        case .needsPermission:
            return "Blocked"
        case .unavailable:
            return "Degraded"
        }
    }

    var symbolName: String {
        switch self {
        case .ready:
            return "checkmark.seal.fill"
        case .needsPermission:
            return "exclamationmark.triangle.fill"
        case .unavailable:
            return "exclamationmark.octagon.fill"
        }
    }

    var tint: Color {
        switch self {
        case .ready:
            return .green
        case .needsPermission:
            return .orange
        case .unavailable:
            return .secondary
        }
    }
}

public enum ConnectorPermissionState: Sendable, Equatable {
    case granted
    case missing
    case unknown

    var label: String {
        switch self {
        case .granted:
            return "Permission Granted"
        case .missing:
            return "Permission Required"
        case .unknown:
            return "Permission Unknown"
        }
    }

    var symbolName: String {
        switch self {
        case .granted:
            return "checkmark.circle.fill"
        case .missing:
            return "exclamationmark.triangle.fill"
        case .unknown:
            return "questionmark.circle.fill"
        }
    }

    var tint: Color {
        switch self {
        case .granted:
            return .green
        case .missing:
            return .orange
        case .unknown:
            return .secondary
        }
    }
}

public struct ConnectorCardItem: Identifiable, Sendable {
    public let id: String
    public let name: String
    public let logicalConnectorID: String
    public let declaredCapabilities: [String]
    public let requiresPermission: Bool
    public var health: ConnectorHealthStatus
    public var permissionState: ConnectorPermissionState
    public let permissionScope: String
    public let statusReason: String?
    public let summary: String
    public let details: [String: String]
    public let editableConfiguration: [String: String]
    public let editableConfigurationKinds: [String: ConfigurationDraftValueKind]
    public let configurationFieldDescriptors: [ConfigurationFieldDescriptorItem]
    public let readOnlyConfiguration: [String: String]
    public let actions: [DiagnosticsActionItem]
    public let unavailableActionReason: String
    public var isExpanded: Bool

    public init(
        id: String,
        name: String,
        logicalConnectorID: String? = nil,
        declaredCapabilities: [String] = [],
        requiresPermission: Bool = false,
        health: ConnectorHealthStatus,
        permissionState: ConnectorPermissionState,
        permissionScope: String,
        statusReason: String? = nil,
        summary: String,
        details: [String: String],
        editableConfiguration: [String: String],
        editableConfigurationKinds: [String: ConfigurationDraftValueKind],
        configurationFieldDescriptors: [ConfigurationFieldDescriptorItem] = [],
        readOnlyConfiguration: [String: String],
        actions: [DiagnosticsActionItem],
        unavailableActionReason: String,
        isExpanded: Bool = false
    ) {
        self.id = id
        self.name = name
        self.logicalConnectorID = logicalConnectorID ?? id
        self.declaredCapabilities = declaredCapabilities
        self.requiresPermission = requiresPermission
        self.health = health
        self.permissionState = permissionState
        self.permissionScope = permissionScope
        self.statusReason = statusReason
        self.summary = summary
        self.details = details
        self.editableConfiguration = editableConfiguration
        self.editableConfigurationKinds = editableConfigurationKinds
        self.configurationFieldDescriptors = configurationFieldDescriptors
        self.readOnlyConfiguration = readOnlyConfiguration
        self.actions = actions
        self.unavailableActionReason = unavailableActionReason
        self.isExpanded = isExpanded
    }
}

public struct LogicalChannelConnectorRollupItem: Identifiable, Sendable {
    public let connectorID: String
    public let connectorName: String
    public let health: ConnectorHealthStatus
    public let reason: String?

    public var id: String { connectorID }
}

public struct ChannelConnectorMappingItem: Identifiable, Sendable, Equatable {
    public let channelID: String
    public let connectorID: String
    public var enabled: Bool
    public var priority: Int
    public let capabilities: [String]
    public let createdAtLabel: String?
    public let updatedAtLabel: String?

    public var id: String { "\(channelID)::\(connectorID)" }
}

public struct LogicalChannelCardItem: Identifiable, Sendable {
    public let channelID: String
    public let displayName: String
    public let status: ChannelCardStatus
    public let summary: String
    public let details: [String: String]
    public let primaryChannelCardID: String
    public let channelCardIDs: [String]
    public let actions: [DiagnosticsActionItem]
    public let unavailableActionReason: String
    public let mappedConnectorRollups: [LogicalChannelConnectorRollupItem]
    public let connectorActionTitles: [String]
    public let connectorReasonSummary: String?

    public var id: String { channelID }
    public var title: String { displayName }
}

public struct LogicalConnectorCardItem: Identifiable, Sendable {
    public let id: String
    public let title: String
    public let health: ConnectorHealthStatus
    public let permissionState: ConnectorPermissionState
    public let permissionScope: String
    public let statusReason: String?
    public let summary: String
    public let details: [String: String]
    public let primaryConnectorCardID: String
    public let connectorCardIDs: [String]
    public let actions: [DiagnosticsActionItem]
    public let unavailableActionReason: String
}

public struct ConfigurationTestResultItem: Sendable {
    public let operation: String
    public let success: Bool
    public let status: String
    public let summary: String
    public let checkedAtLabel: String
    public let details: [String: String]
}

public enum ProviderReadinessStatus: Sendable {
    case healthy
    case configured
    case missingSetup
    case checkFailed

    var label: String {
        switch self {
        case .healthy:
            return "Healthy"
        case .configured:
            return "Configured"
        case .missingSetup:
            return "Setup Required"
        case .checkFailed:
            return "Check Failed"
        }
    }

    var symbolName: String {
        switch self {
        case .healthy:
            return "checkmark.circle.fill"
        case .configured:
            return "checkmark.seal"
        case .missingSetup:
            return "exclamationmark.triangle.fill"
        case .checkFailed:
            return "xmark.octagon.fill"
        }
    }

    var tint: Color {
        switch self {
        case .healthy:
            return .green
        case .configured:
            return .secondary
        case .missingSetup:
            return .orange
        case .checkFailed:
            return .red
        }
    }
}

public struct ProviderReadinessItem: Identifiable, Sendable {
    public let id: String
    public let provider: String
    public let endpoint: String
    public let status: ProviderReadinessStatus
    public let detail: String
    public let updatedAtLabel: String
}

public struct ModelRouteSummary: Sendable {
    public let provider: String
    public let modelKey: String
    public let source: String
    public let notes: String?
}

public struct WorkflowRouteContext: Sendable {
    public let available: Bool
    public let taskClass: String?
    public let provider: String?
    public let modelKey: String?
    public let taskClassSource: String?
    public let routeSource: String?
    public let notes: String?

    public init(
        available: Bool = false,
        taskClass: String? = nil,
        provider: String? = nil,
        modelKey: String? = nil,
        taskClassSource: String? = nil,
        routeSource: String? = nil,
        notes: String? = nil
    ) {
        self.available = available
        self.taskClass = taskClass
        self.provider = provider
        self.modelKey = modelKey
        self.taskClassSource = taskClassSource
        self.routeSource = routeSource
        self.notes = notes
    }

    public var routeLabel: String? {
        guard let provider, let modelKey else {
            return nil
        }
        return "\(provider) • \(modelKey)"
    }
}

public struct ChatTaskRunTraceabilityItem: Sendable {
    public let available: Bool
    public let source: String
    public let taskID: String?
    public let runID: String?
    public let taskState: String?
    public let runState: String?
    public let correlationID: String?
    public let taskClass: String?
    public let provider: String?
    public let modelKey: String?
    public let routeSource: String?
    public let turnContractVersion: String?
    public let turnItemSchemaVersion: String?
    public let realtimeEventContractVersion: String?
    public let realtimeLifecycleContractVersion: String?
    public let realtimeLifecycleSchemaVersion: String?
    public let approvalRequired: Bool
    public let approvalRequestID: String?
    public let clarificationRequired: Bool
    public let clarificationPrompt: String?
    public let responseShapingChannel: String?
    public let responseShapingProfile: String?
    public let personaPolicySource: String?
    public let responseShapingGuardrailCount: Int?
    public let responseShapingInstructionCount: Int?

    public init(
        available: Bool,
        source: String,
        taskID: String?,
        runID: String?,
        taskState: String?,
        runState: String?,
        correlationID: String?,
        taskClass: String? = nil,
        provider: String? = nil,
        modelKey: String? = nil,
        routeSource: String? = nil,
        turnContractVersion: String? = nil,
        turnItemSchemaVersion: String? = nil,
        realtimeEventContractVersion: String? = nil,
        realtimeLifecycleContractVersion: String? = nil,
        realtimeLifecycleSchemaVersion: String? = nil,
        approvalRequired: Bool = false,
        approvalRequestID: String? = nil,
        clarificationRequired: Bool = false,
        clarificationPrompt: String? = nil,
        responseShapingChannel: String? = nil,
        responseShapingProfile: String? = nil,
        personaPolicySource: String? = nil,
        responseShapingGuardrailCount: Int? = nil,
        responseShapingInstructionCount: Int? = nil
    ) {
        self.available = available
        self.source = source
        self.taskID = taskID
        self.runID = runID
        self.taskState = taskState
        self.runState = runState
        self.correlationID = correlationID
        self.taskClass = taskClass
        self.provider = provider
        self.modelKey = modelKey
        self.routeSource = routeSource
        self.turnContractVersion = turnContractVersion
        self.turnItemSchemaVersion = turnItemSchemaVersion
        self.realtimeEventContractVersion = realtimeEventContractVersion
        self.realtimeLifecycleContractVersion = realtimeLifecycleContractVersion
        self.realtimeLifecycleSchemaVersion = realtimeLifecycleSchemaVersion
        self.approvalRequired = approvalRequired
        self.approvalRequestID = approvalRequestID
        self.clarificationRequired = clarificationRequired
        self.clarificationPrompt = clarificationPrompt
        self.responseShapingChannel = responseShapingChannel
        self.responseShapingProfile = responseShapingProfile
        self.personaPolicySource = personaPolicySource
        self.responseShapingGuardrailCount = responseShapingGuardrailCount
        self.responseShapingInstructionCount = responseShapingInstructionCount
    }

    public var hasTaskOrRunIdentity: Bool {
        taskID != nil || runID != nil
    }

    public var hasRouteContext: Bool {
        taskClass != nil || provider != nil || modelKey != nil || routeSource != nil
    }

    public var hasLifecycleContractContext: Bool {
        turnContractVersion != nil
            || turnItemSchemaVersion != nil
            || realtimeEventContractVersion != nil
            || realtimeLifecycleContractVersion != nil
            || realtimeLifecycleSchemaVersion != nil
    }

    public var hasResponseShapingContext: Bool {
        responseShapingChannel != nil
            || responseShapingProfile != nil
            || personaPolicySource != nil
            || responseShapingGuardrailCount != nil
            || responseShapingInstructionCount != nil
    }
}

public struct ChatTurnExplainabilityToolCatalogItem: Identifiable, Sendable {
    public let name: String
    public let description: String?
    public let capabilityKeys: [String]
    public let inputSchemaSummary: String?

    public var id: String { name }
}

public struct ChatTurnExplainabilityPolicyDecisionItem: Identifiable, Sendable {
    public let id: String
    public let toolName: String
    public let capabilityKey: String?
    public let decision: String
    public let reason: String?
}

public struct ChatTurnExplainabilityItem: Sendable {
    public let workspaceID: String
    public let taskClass: String
    public let requestedByActorID: String?
    public let subjectActorID: String?
    public let actingAsActorID: String?
    public let contractVersion: String
    public let selectedProvider: String?
    public let selectedModelKey: String?
    public let selectedSource: String?
    public let routeSummary: String?
    public let routeReasonCodes: [String]
    public let routeExplanations: [String]
    public let toolCatalog: [ChatTurnExplainabilityToolCatalogItem]
    public let policyDecisions: [ChatTurnExplainabilityPolicyDecisionItem]

    public var routeLabel: String? {
        guard let selectedProvider, let selectedModelKey else {
            return nil
        }
        return "\(selectedProvider) • \(selectedModelKey)"
    }

    public var hasToolCatalog: Bool {
        !toolCatalog.isEmpty
    }

    public var hasPolicyDecisions: Bool {
        !policyDecisions.isEmpty
    }
}

public enum ChatPersonaScopeType: String, CaseIterable, Sendable {
    case workspace
    case principal
    case channel
    case principalChannel

    public var title: String {
        switch self {
        case .workspace:
            return "Workspace Default"
        case .principal:
            return "Principal"
        case .channel:
            return "Channel"
        case .principalChannel:
            return "Principal + Channel"
        }
    }

    public var subtitle: String {
        switch self {
        case .workspace:
            return "Applies when no principal/channel-specific override exists."
        case .principal:
            return "Applies to one principal across channels."
        case .channel:
            return "Applies to one channel across principals."
        case .principalChannel:
            return "Most specific scope for one principal in one channel."
        }
    }
}

public struct ChatPersonaPolicyItem: Sendable, Equatable {
    public let workspaceID: String
    public let principalActorID: String?
    public let channelID: String?
    public let stylePrompt: String
    public let guardrails: [String]
    public let source: String
    public let updatedAtRaw: String?
    public let updatedAtLabel: String?

    public init(
        workspaceID: String,
        principalActorID: String?,
        channelID: String?,
        stylePrompt: String,
        guardrails: [String],
        source: String,
        updatedAtRaw: String?,
        updatedAtLabel: String?
    ) {
        self.workspaceID = workspaceID
        self.principalActorID = principalActorID
        self.channelID = channelID
        self.stylePrompt = stylePrompt
        self.guardrails = guardrails
        self.source = source
        self.updatedAtRaw = updatedAtRaw
        self.updatedAtLabel = updatedAtLabel
    }
}

public struct ChatPersonaPolicyMutationInput: Sendable {
    public let principalActorID: String?
    public let channelID: String?
    public let stylePrompt: String
    public let guardrailsText: String

    public init(
        principalActorID: String?,
        channelID: String?,
        stylePrompt: String,
        guardrailsText: String
    ) {
        self.principalActorID = principalActorID
        self.channelID = channelID
        self.stylePrompt = stylePrompt
        self.guardrailsText = guardrailsText
    }
}

public struct AutomationTriggerItem: Identifiable, Sendable {
    public let id: String
    public let triggerType: String
    public let enabled: Bool
    public let directiveTitle: String
    public let directiveInstruction: String
    public let directiveStatus: String
    public let subjectPrincipalActor: String
    public let cooldownSeconds: Int
    public let filterSummary: String
    public let updatedAtLabel: String
}

public enum AutomationFireHistoryStatus: Sendable {
    case createdTask
    case pending
    case failed
    case other(String)

    var label: String {
        switch self {
        case .createdTask:
            return "Created Task"
        case .pending:
            return "Pending"
        case .failed:
            return "Failed"
        case .other(let raw):
            return raw.replacingOccurrences(of: "_", with: " ").capitalized
        }
    }

    var symbolName: String {
        switch self {
        case .createdTask:
            return "checkmark.circle.fill"
        case .pending:
            return "clock.badge.questionmark.fill"
        case .failed:
            return "xmark.octagon.fill"
        case .other:
            return "info.circle.fill"
        }
    }

    var tint: Color {
        switch self {
        case .createdTask:
            return .green
        case .pending:
            return .orange
        case .failed:
            return .red
        case .other:
            return .secondary
        }
    }
}

public struct AutomationFireHistoryItem: Identifiable, Sendable {
    public let id: String
    public let triggerID: String
    public let triggerType: String
    public let status: AutomationFireHistoryStatus
    public let outcome: String
    public let idempotencySignal: String
    public let idempotencyKey: String
    public let firedAtLabel: String
    public let taskID: String?
    public let runID: String?
    public let sortTimestamp: Date
    public let route: WorkflowRouteContext

    public var hasWorkflowContext: Bool {
        taskID != nil
            || runID != nil
            || route.available
            || route.taskClass != nil
            || route.provider != nil
            || route.modelKey != nil
    }
}

public struct AutomationTriggerMutationInput: Sendable {
    public let triggerType: String
    public let subjectActorID: String
    public let title: String
    public let instruction: String
    public let enabled: Bool
    public let cooldownSeconds: Int
    public let scheduleIntervalSeconds: Int?
    public let commEventFilterJSON: String?

    public init(
        triggerType: String,
        subjectActorID: String,
        title: String,
        instruction: String,
        enabled: Bool,
        cooldownSeconds: Int,
        scheduleIntervalSeconds: Int?,
        commEventFilterJSON: String?
    ) {
        self.triggerType = triggerType
        self.subjectActorID = subjectActorID
        self.title = title
        self.instruction = instruction
        self.enabled = enabled
        self.cooldownSeconds = cooldownSeconds
        self.scheduleIntervalSeconds = scheduleIntervalSeconds
        self.commEventFilterJSON = commEventFilterJSON
    }
}

public struct RuntimePluginLifecycleEventItem: Identifiable, Sendable {
    public let id: String
    public let workspaceID: String
    public let pluginID: String
    public let kind: String
    public let state: String
    public let eventType: String
    public let processID: Int
    public let restartCount: Int
    public let reason: String
    public let error: String?
    public let restartEvent: Bool
    public let failureEvent: Bool
    public let recoveryEvent: Bool
    public let lastHeartbeatAtLabel: String?
    public let lastTransitionAtLabel: String?
    public let occurredAtLabel: String
    public let sortTimestamp: Date

    public var hasError: Bool {
        guard let error else {
            return false
        }
        return !error.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
    }

    public var kindLabel: String {
        kind.replacingOccurrences(of: "_", with: " ").capitalized
    }
}

public struct RuntimePluginLifecycleTrendItem: Identifiable, Sendable {
    public let id: String
    public let pluginID: String
    public let kind: String
    public let latestState: String
    public let latestEventType: String
    public let latestOccurredAtLabel: String
    public let restartEvents: Int
    public let failureEvents: Int
    public let recoveryEvents: Int
    public let totalEvents: Int

    public var kindLabel: String {
        kind.replacingOccurrences(of: "_", with: " ").capitalized
    }
}

public struct MemorySourceItem: Identifiable, Sendable {
    public let id: String
    public let sourceType: String
    public let sourceRef: String
    public let createdAtLabel: String
}

public struct MemoryInventoryItem: Identifiable, Sendable {
    public let id: String
    public let workspaceID: String
    public let ownerActorID: String
    public let scopeType: String
    public let key: String
    public let status: String
    public let kind: String
    public let isCanonical: Bool
    public let tokenEstimate: Int
    public let sourceSummary: String
    public let sourceCount: Int
    public let createdAtLabel: String
    public let updatedAtLabel: String
    public let valueSummary: String
    public let sources: [MemorySourceItem]
    public let sortTimestamp: Date
}

public struct MemoryCompactionCandidateItem: Identifiable, Sendable {
    public let id: String
    public let workspaceID: String
    public let ownerActorID: String
    public let status: String
    public let score: Double
    public let candidateKind: String
    public let tokenEstimate: Int
    public let sourceIDs: [String]
    public let sourceRefs: [String]
    public let candidateSummary: String
    public let createdAtLabel: String
    public let sortTimestamp: Date
}

public struct RetrievalDocumentItem: Identifiable, Sendable {
    public let id: String
    public let workspaceID: String
    public let ownerActorID: String
    public let sourceURI: String
    public let checksum: String
    public let chunkCount: Int
    public let createdAtLabel: String
    public let sortTimestamp: Date
}

public struct RetrievalChunkItem: Identifiable, Sendable {
    public let id: String
    public let workspaceID: String
    public let documentID: String
    public let ownerActorID: String
    public let sourceURI: String
    public let chunkIndex: Int
    public let tokenCount: Int
    public let textPreview: String
    public let createdAtLabel: String
    public let sortTimestamp: Date
}

public struct CapabilityGrantItem: Identifiable, Sendable {
    public let id: String
    public let workspaceID: String
    public let actorID: String
    public let capabilityKey: String
    public let scopeJSON: String
    public let scopeSummary: String
    public let status: String
    public let createdAtLabel: String
    public let createdAtRaw: String
    public let expiresAtRaw: String?
    public let expiresAtLabel: String?
    public let sortTimestamp: Date

    public var isRevoked: Bool {
        status.uppercased() == "REVOKED"
    }
}

public struct TrustReceiptAuditLinkItem: Identifiable, Sendable {
    public let id: String
    public let eventType: String
    public let createdAtLabel: String
}

public struct WebhookTrustReceiptItem: Identifiable, Sendable {
    public let id: String
    public let workspaceID: String
    public let provider: String
    public let providerEventID: String
    public let trustState: String
    public let signatureValid: Bool
    public let signatureValuePresent: Bool
    public let payloadHash: String
    public let eventID: String?
    public let threadID: String?
    public let receivedAtLabel: String?
    public let createdAtLabel: String
    public let auditLinks: [TrustReceiptAuditLinkItem]
    public let sortTimestamp: Date
}

public struct IngestTrustReceiptItem: Identifiable, Sendable {
    public let id: String
    public let workspaceID: String
    public let source: String
    public let sourceScope: String
    public let sourceEventID: String
    public let sourceCursor: String?
    public let trustState: String
    public let payloadHash: String
    public let eventID: String?
    public let threadID: String?
    public let receivedAtLabel: String?
    public let createdAtLabel: String
    public let auditLinks: [TrustReceiptAuditLinkItem]
    public let sortTimestamp: Date
}

public enum ApprovalInboxDecisionState: Sendable {
    case pending
    case final

    var label: String {
        switch self {
        case .pending:
            return "Pending"
        case .final:
            return "Finalized"
        }
    }

    var symbolName: String {
        switch self {
        case .pending:
            return "hourglass.circle.fill"
        case .final:
            return "checkmark.circle.fill"
        }
    }

    var tint: Color {
        switch self {
        case .pending:
            return .orange
        case .final:
            return .secondary
        }
    }
}

public enum ApprovalInboxDecisionOutcome: Sendable {
    case approved
    case rejected
    case other(String)

    var label: String {
        switch self {
        case .approved:
            return "Approved"
        case .rejected:
            return "Rejected"
        case .other(let raw):
            return raw.replacingOccurrences(of: "_", with: " ").capitalized
        }
    }

    var symbolName: String {
        switch self {
        case .approved:
            return "checkmark.circle.fill"
        case .rejected:
            return "xmark.circle.fill"
        case .other:
            return "questionmark.circle.fill"
        }
    }

    var tint: Color {
        switch self {
        case .approved:
            return .green
        case .rejected:
            return .red
        case .other:
            return .secondary
        }
    }
}

public enum ApprovalInboxRiskLevel: Sendable {
    case destructive
    case policy
    case other(String)

    var label: String {
        switch self {
        case .destructive:
            return "Destructive"
        case .policy:
            return "Policy"
        case .other(let raw):
            return raw.replacingOccurrences(of: "_", with: " ").capitalized
        }
    }

    var symbolName: String {
        switch self {
        case .destructive:
            return "exclamationmark.triangle.fill"
        case .policy:
            return "shield.checkered"
        case .other:
            return "questionmark.circle.fill"
        }
    }

    var tint: Color {
        switch self {
        case .destructive:
            return .orange
        case .policy:
            return .blue
        case .other:
            return .secondary
        }
    }
}

public struct ApprovalInboxItem: Identifiable, Sendable {
    public let id: String
    public let taskTitle: String
    public let decisionState: ApprovalInboxDecisionState
    public let decisionOutcome: ApprovalInboxDecisionOutcome?
    public let riskLevel: ApprovalInboxRiskLevel
    public let riskRationale: String
    public let requestedAtLabel: String
    public let decidedAtLabel: String?
    public let decisionByActorID: String?
    public let decisionRationale: String?
    public let requestedPhrase: String?
    public let taskState: String
    public let runState: String
    public let stepName: String
    public let stepCapabilityKey: String?
    public let requestedByActorID: String
    public let subjectPrincipalActorID: String
    public let actingAsActorID: String
    public let taskID: String?
    public let runID: String?
    public let stepID: String?
    public let route: WorkflowRouteContext
}

public struct ApprovalEvidenceStepItem: Sendable {
    public let stepID: String
    public let name: String
    public let statusLabel: String
    public let capability: String?
    public let interactionLevel: String?
    public let updatedAtLabel: String
    public let inputSummary: String
    public let outputSummary: String
    public let lastError: String?
}

public struct ApprovalEvidenceArtifactItem: Identifiable, Sendable {
    public let id: String
    public let type: String
    public let stepID: String?
    public let uri: String?
    public let contentHash: String?
    public let createdAtLabel: String
}

public struct ApprovalEvidenceAuditItem: Identifiable, Sendable {
    public let id: String
    public let eventType: String
    public let createdAtLabel: String
    public let payloadSummary: String?
}

public struct ApprovalEvidenceItem: Identifiable, Sendable {
    public let id: String
    public let runID: String
    public let taskID: String
    public let title: String
    public let updatedAtLabel: String
    public let step: ApprovalEvidenceStepItem?
    public let artifacts: [ApprovalEvidenceArtifactItem]
    public let auditEntries: [ApprovalEvidenceAuditItem]
}

public struct IdentityWorkspaceItem: Identifiable, Sendable {
    public let id: String
    public let name: String
    public let status: String
    public let principalCount: Int
    public let actorCount: Int
    public let handleCount: Int
    public let updatedAtLabel: String
    public let isActive: Bool
}

public struct IdentityPrincipalHandleItem: Identifiable, Sendable {
    public let id: String
    public let channel: String
    public let handleValue: String
    public let isPrimary: Bool
    public let updatedAtLabel: String
}

public struct IdentityPrincipalItem: Identifiable, Sendable {
    public let id: String
    public let displayName: String
    public let actorType: String
    public let actorStatus: String
    public let principalStatus: String
    public let isActive: Bool
    public let handles: [IdentityPrincipalHandleItem]
}

public struct IdentityDisplayValue: Sendable, Equatable {
    public let displayText: String
    public let rawID: String?
}

public struct IdentityActiveContextItem: Sendable {
    public let workspaceID: String
    public let principalActorID: String
    public let workspaceSource: String
    public let principalSource: String
    public let lastUpdatedLabel: String?
    public let workspaceResolved: Bool
}

public struct IdentityDeviceItem: Identifiable, Sendable {
    public let id: String
    public let workspaceID: String
    public let userID: String
    public let deviceType: String
    public let platform: String
    public let label: String?
    public let lastSeenAtLabel: String?
    public let createdAtLabel: String
    public let sessionTotal: Int
    public let sessionActiveCount: Int
    public let sessionExpiredCount: Int
    public let sessionRevokedCount: Int
    public let sessionLatestStartedAtLabel: String?
    public let sortTimestamp: Date
}

public struct IdentitySessionItem: Identifiable, Sendable {
    public let id: String
    public let workspaceID: String
    public let deviceID: String
    public let userID: String
    public let deviceType: String
    public let platform: String
    public let deviceLabel: String?
    public let deviceLastSeenAtLabel: String?
    public let startedAtLabel: String
    public let expiresAtLabel: String
    public let revokedAtLabel: String?
    public let sessionHealth: String
    public let sortTimestamp: Date

    public var canRevoke: Bool {
        sessionHealth.lowercased() == "active"
    }
}

public struct DelegationRuleItem: Identifiable, Sendable {
    public let id: String
    public let workspaceID: String
    public let fromActorID: String
    public let toActorID: String
    public let scopeType: String
    public let scopeKey: String?
    public let status: String
    public let createdAtLabel: String
    public let expiresAtLabel: String?
}

public struct DelegationGrantInput: Sendable {
    public let fromActorID: String
    public let toActorID: String
    public let scopeType: String
    public let scopeKey: String?
    public let expiresAt: String?

    public init(
        fromActorID: String,
        toActorID: String,
        scopeType: String,
        scopeKey: String?,
        expiresAt: String?
    ) {
        self.fromActorID = fromActorID
        self.toActorID = toActorID
        self.scopeType = scopeType
        self.scopeKey = scopeKey
        self.expiresAt = expiresAt
    }
}

public struct CapabilityGrantMutationInput: Sendable {
    public let grantID: String?
    public let actorID: String
    public let capabilityKey: String
    public let scopeJSON: String?
    public let status: String
    public let expiresAt: String?

    public init(
        grantID: String?,
        actorID: String,
        capabilityKey: String,
        scopeJSON: String?,
        status: String,
        expiresAt: String?
    ) {
        self.grantID = grantID
        self.actorID = actorID
        self.capabilityKey = capabilityKey
        self.scopeJSON = scopeJSON
        self.status = status
        self.expiresAt = expiresAt
    }
}

public enum TaskRunWorkflowState: Sendable {
    case queued
    case planning
    case awaitingApproval
    case running
    case blocked
    case completed
    case failed
    case cancelled
    case unknown(String)

    var label: String {
        switch self {
        case .queued:
            return "Queued"
        case .planning:
            return "Planning"
        case .awaitingApproval:
            return "Awaiting Approval"
        case .running:
            return "Running"
        case .blocked:
            return "Blocked"
        case .completed:
            return "Completed"
        case .failed:
            return "Failed"
        case .cancelled:
            return "Cancelled"
        case .unknown(let raw):
            return raw.replacingOccurrences(of: "_", with: " ").capitalized
        }
    }

    var symbolName: String {
        switch self {
        case .queued:
            return "clock.fill"
        case .planning:
            return "list.bullet.clipboard"
        case .awaitingApproval:
            return "hand.raised.fill"
        case .running:
            return "arrow.triangle.2.circlepath"
        case .blocked:
            return "pause.circle.fill"
        case .completed:
            return "checkmark.circle.fill"
        case .failed:
            return "xmark.octagon.fill"
        case .cancelled:
            return "slash.circle.fill"
        case .unknown:
            return "questionmark.circle.fill"
        }
    }

    var tint: Color {
        switch self {
        case .completed:
            return .green
        case .failed:
            return .red
        case .running, .planning:
            return .blue
        case .awaitingApproval, .blocked, .queued:
            return .orange
        case .cancelled, .unknown:
            return .secondary
        }
    }
}

public enum TaskRunControlAction: String, CaseIterable, Sendable {
    case cancel
    case retry
    case requeue

    public var title: String {
        switch self {
        case .cancel:
            return "Cancel Run"
        case .retry:
            return "Retry Run"
        case .requeue:
            return "Requeue Run"
        }
    }

    public var symbolName: String {
        switch self {
        case .cancel:
            return "xmark.circle"
        case .retry:
            return "arrow.clockwise.circle"
        case .requeue:
            return "arrow.triangle.2.circlepath.circle"
        }
    }

    var confirmationTitle: String {
        switch self {
        case .cancel:
            return "Cancel Task Run?"
        case .retry:
            return "Retry Task Run?"
        case .requeue:
            return "Requeue Task Run?"
        }
    }

    var confirmationButtonTitle: String {
        switch self {
        case .cancel:
            return "Cancel Run"
        case .retry:
            return "Retry Run"
        case .requeue:
            return "Requeue Run"
        }
    }

    var isDestructive: Bool {
        self == .cancel
    }

    var statusActionID: String {
        switch self {
        case .cancel:
            return "task_cancel"
        case .retry:
            return "task_retry"
        case .requeue:
            return "task_requeue"
        }
    }
}

public struct TaskRunActionAvailabilityItem: Sendable, Equatable {
    public let canCancel: Bool
    public let canRetry: Bool
    public let canRequeue: Bool

    public static let unavailable = TaskRunActionAvailabilityItem(
        canCancel: false,
        canRetry: false,
        canRequeue: false
    )
}

public struct TaskRunListRowItem: Identifiable, Sendable {
    public let id: String
    public let title: String
    public let taskID: String
    public let runID: String?
    public let taskState: String
    public let runState: String
    public let effectiveState: TaskRunWorkflowState
    public let priority: Int
    public let priorityLabel: String
    public let requestedByActorID: String
    public let subjectPrincipalActorID: String
    public let actingAsActorID: String
    public let taskCreatedAtLabel: String
    public let taskUpdatedAtLabel: String
    public let runCreatedAtLabel: String?
    public let runUpdatedAtLabel: String?
    public let startedAtLabel: String?
    public let finishedAtLabel: String?
    public let lastError: String?
    public let actions: TaskRunActionAvailabilityItem
    public let sortTimestamp: Date
    public let route: WorkflowRouteContext
}

public struct WorkflowCardSummary: Sendable, Equatable {
    public let whatHappened: String
    public let whatNeedsAction: String
    public let whatNext: String

    public init(
        whatHappened: String,
        whatNeedsAction: String,
        whatNext: String
    ) {
        self.whatHappened = whatHappened
        self.whatNeedsAction = whatNeedsAction
        self.whatNext = whatNext
    }
}

public struct TaskRunDetailStepItem: Identifiable, Sendable {
    public let id: String
    public let index: Int
    public let name: String
    public let statusLabel: String
    public let capability: String?
    public let interactionLevel: String?
    public let retryLabel: String
    public let timeoutLabel: String
    public let updatedAtLabel: String
    public let lastError: String?
}

public struct TaskRunDetailArtifactItem: Identifiable, Sendable {
    public let id: String
    public let type: String
    public let stepID: String?
    public let uri: String?
    public let contentHash: String?
    public let createdAtLabel: String
}

public struct TaskRunDetailAuditItem: Identifiable, Sendable {
    public let id: String
    public let eventType: String
    public let actorID: String?
    public let actingAsActorID: String?
    public let correlationID: String?
    public let payloadSummary: String?
    public let createdAtLabel: String
}

public struct TaskRunDetailItem: Identifiable, Sendable {
    public let id: String
    public let taskID: String
    public let runID: String
    public let title: String
    public let taskState: String
    public let runState: String
    public let priorityLabel: String
    public let requestedByActorID: String
    public let subjectPrincipalActorID: String
    public let actingAsActorID: String
    public let startedAtLabel: String?
    public let finishedAtLabel: String?
    public let updatedAtLabel: String
    public let lastError: String?
    public let actions: TaskRunActionAvailabilityItem
    public let route: WorkflowRouteContext
    public let steps: [TaskRunDetailStepItem]
    public let artifacts: [TaskRunDetailArtifactItem]
    public let auditEntries: [TaskRunDetailAuditItem]
}

public struct TaskSubmissionReceiptItem: Identifiable, Sendable {
    public let taskID: String
    public let runID: String
    public let state: String
    public let correlationID: String?
    public let submittedAt: Date

    public var id: String {
        if let correlationID {
            return correlationID
        }
        return "\(taskID)::\(runID)"
    }
}

public struct TaskSubmitDraftSeed: Identifiable, Sendable, Equatable {
    public let id: String
    public let title: String
    public let description: String?
    public let taskClass: String
    public let requestedByActorID: String?
    public let subjectPrincipalActorID: String?

    public init(
        id: String = UUID().uuidString.lowercased(),
        title: String,
        description: String?,
        taskClass: String,
        requestedByActorID: String?,
        subjectPrincipalActorID: String?
    ) {
        self.id = id
        self.title = title
        self.description = description
        self.taskClass = taskClass
        self.requestedByActorID = requestedByActorID
        self.subjectPrincipalActorID = subjectPrincipalActorID
    }
}

public struct CommunicationSendReceiptItem: Identifiable, Sendable {
    public let operationID: String
    public let threadID: String?
    public let sourceChannel: String
    public let connectorID: String?
    public let destination: String?
    public let success: Bool
    public let sentAt: Date

    public var id: String {
        operationID
    }
}

public struct CommunicationThreadItem: Identifiable, Sendable {
    public let id: String
    public let workspaceID: String
    public let channel: String
    public let connectorID: String?
    public let title: String
    public let externalRef: String?
    public let lastEventID: String?
    public let lastEventType: String?
    public let lastDirection: String?
    public let lastOccurredAtLabel: String?
    public let lastBodyPreview: String?
    public let participantAddresses: [String]
    public let eventCount: Int
    public let createdAtLabel: String
    public let updatedAtLabel: String
    public let sortTimestamp: Date
}

public struct CommunicationEventAddressItem: Identifiable, Sendable {
    public let id: String
    public let role: String
    public let value: String
    public let display: String?
    public let position: Int
}

public struct CommunicationEventItem: Identifiable, Sendable {
    public let id: String
    public let workspaceID: String
    public let threadID: String
    public let channel: String
    public let connectorID: String?
    public let eventType: String
    public let direction: String
    public let assistantEmitted: Bool
    public let bodyText: String?
    public let occurredAtLabel: String
    public let createdAtLabel: String
    public let addresses: [CommunicationEventAddressItem]
    public let sortTimestamp: Date
}

public struct CommunicationCallSessionItem: Identifiable, Sendable {
    public let id: String
    public let workspaceID: String
    public let provider: String
    public let connectorID: String?
    public let providerCallID: String?
    public let threadID: String?
    public let direction: String
    public let fromAddress: String?
    public let toAddress: String?
    public let status: String
    public let startedAtLabel: String?
    public let endedAtLabel: String?
    public let updatedAtLabel: String
    public let sortTimestamp: Date
}

public struct ChannelDeliveryPolicyItem: Identifiable, Sendable {
    public let id: String
    public let workspaceID: String
    public let sourceChannel: String
    public let endpointPattern: String?
    public let isDefault: Bool
    public let primaryChannel: String
    public let retryCount: Int
    public let fallbackChannels: [String]
    public let createdAtLabel: String
    public let updatedAtLabel: String
    public let sortTimestamp: Date
}

public struct ChannelDeliveryPolicyDraft: Sendable, Equatable {
    public var policyID: String?
    public var endpointPattern: String
    public var primaryChannel: String
    public var retryCount: Int
    public var fallbackChannelsText: String
    public var isDefault: Bool

    public init(
        policyID: String? = nil,
        endpointPattern: String = "",
        primaryChannel: String,
        retryCount: Int = 0,
        fallbackChannelsText: String = "",
        isDefault: Bool = true
    ) {
        self.policyID = policyID
        self.endpointPattern = endpointPattern
        self.primaryChannel = primaryChannel
        self.retryCount = retryCount
        self.fallbackChannelsText = fallbackChannelsText
        self.isDefault = isDefault
    }
}

public struct CommunicationDeliveryAttemptItem: Identifiable, Sendable {
    public let id: String
    public let workspaceID: String
    public let operationID: String?
    public let taskID: String?
    public let runID: String?
    public let stepID: String?
    public let eventID: String?
    public let threadID: String?
    public let destinationEndpoint: String
    public let idempotencyKey: String
    public let channel: String
    public let routeIndex: Int
    public let routePhase: String
    public let retryOrdinal: Int
    public let fallbackFromChannel: String?
    public let status: String
    public let providerReceipt: String?
    public let error: String?
    public let attemptedAtLabel: String
    public let sortTimestamp: Date
}

public struct CommunicationContinuityItem: Identifiable, Sendable {
    public let id: String
    public let turnID: String
    public let workspaceID: String
    public let channel: String
    public let connectorID: String?
    public let threadID: String?
    public let correlationID: String?
    public let taskClass: String
    public let itemType: String
    public let itemStatus: String
    public let summary: String
    public let taskID: String?
    public let runID: String?
    public let taskState: String?
    public let runState: String?
    public let responseShapingChannel: String?
    public let responseShapingProfile: String?
    public let personaPolicySource: String?
    public let responseShapingGuardrailCount: Int?
    public let responseShapingInstructionCount: Int?
    public let createdAtLabel: String
    public let sortTimestamp: Date

    public init(
        id: String,
        turnID: String,
        workspaceID: String,
        channel: String,
        connectorID: String?,
        threadID: String?,
        correlationID: String?,
        taskClass: String,
        itemType: String,
        itemStatus: String,
        summary: String,
        taskID: String?,
        runID: String?,
        taskState: String?,
        runState: String?,
        responseShapingChannel: String? = nil,
        responseShapingProfile: String? = nil,
        personaPolicySource: String? = nil,
        responseShapingGuardrailCount: Int? = nil,
        responseShapingInstructionCount: Int? = nil,
        createdAtLabel: String,
        sortTimestamp: Date
    ) {
        self.id = id
        self.turnID = turnID
        self.workspaceID = workspaceID
        self.channel = channel
        self.connectorID = connectorID
        self.threadID = threadID
        self.correlationID = correlationID
        self.taskClass = taskClass
        self.itemType = itemType
        self.itemStatus = itemStatus
        self.summary = summary
        self.taskID = taskID
        self.runID = runID
        self.taskState = taskState
        self.runState = runState
        self.responseShapingChannel = responseShapingChannel
        self.responseShapingProfile = responseShapingProfile
        self.personaPolicySource = personaPolicySource
        self.responseShapingGuardrailCount = responseShapingGuardrailCount
        self.responseShapingInstructionCount = responseShapingInstructionCount
        self.createdAtLabel = createdAtLabel
        self.sortTimestamp = sortTimestamp
    }
}

public struct ModelCatalogEntryItem: Identifiable, Sendable {
    public let id: String
    public let provider: String
    public let modelKey: String
    public let enabled: Bool
    public let providerReady: Bool
    public let providerEndpoint: String
}

public struct DiscoveredModelEntryItem: Identifiable, Sendable {
    public let id: String
    public let provider: String
    public let modelKey: String
    public let displayName: String
    public let source: String
    public let inCatalog: Bool
    public let enabled: Bool
}

public struct ModelPolicyItem: Identifiable, Sendable {
    public let id: String
    public let taskClass: String
    public let provider: String
    public let modelKey: String
    public let updatedAtLabel: String
}

public struct ModelRouteDecisionTraceItem: Identifiable, Sendable {
    public let id: String
    public let step: String
    public let decision: String
    public let reasonCode: String
    public let provider: String?
    public let modelKey: String?
    public let note: String?
}

public struct ModelRouteFallbackTraceItem: Identifiable, Sendable {
    public let id: String
    public let rank: Int
    public let provider: String
    public let modelKey: String
    public let selected: Bool
    public let reasonCode: String
}

public struct ModelRouteSimulationResultItem: Sendable {
    public let workspaceID: String
    public let taskClass: String
    public let principalActorID: String?
    public let selectedProvider: String
    public let selectedModelKey: String
    public let selectedSource: String
    public let notes: String?
    public let reasonCodes: [String]
    public let decisions: [ModelRouteDecisionTraceItem]
    public let fallbackChain: [ModelRouteFallbackTraceItem]
}

public struct ModelRouteExplainResultItem: Sendable {
    public let workspaceID: String
    public let taskClass: String
    public let principalActorID: String?
    public let selectedProvider: String
    public let selectedModelKey: String
    public let selectedSource: String
    public let summary: String
    public let explanations: [String]
    public let reasonCodes: [String]
    public let decisions: [ModelRouteDecisionTraceItem]
    public let fallbackChain: [ModelRouteFallbackTraceItem]
}

public struct OnboardingCheckItem: Identifiable, Sendable {
    public let id: String
    public let title: String
    public let detail: String
    public let isComplete: Bool
}

public enum OnboardingSetupStepStatus: Sendable, Equatable {
    case loading
    case blocked
    case complete

    public var label: String {
        switch self {
        case .loading:
            return "Checking"
        case .blocked:
            return "Needs Attention"
        case .complete:
            return "Ready"
        }
    }

    public var symbolName: String {
        switch self {
        case .loading:
            return "clock.arrow.circlepath"
        case .blocked:
            return "exclamationmark.triangle.fill"
        case .complete:
            return "checkmark.circle.fill"
        }
    }

    public var tint: Color {
        switch self {
        case .loading:
            return .secondary
        case .blocked:
            return .orange
        case .complete:
            return .green
        }
    }

    public var isComplete: Bool {
        self == .complete
    }

    public var isBlocked: Bool {
        self == .blocked
    }
}

public enum OnboardingSetupActionKind: String, Sendable {
    case openConfiguration
    case openModels
    case openChannels
    case openConnectors
    case refreshChecks
    case startDaemon
    case installDaemon
    case repairDaemon
}

public struct OnboardingSetupAction: Identifiable, Sendable {
    public let kind: OnboardingSetupActionKind
    public let title: String
    public let detail: String?
    public let isEnabled: Bool

    public var id: String { kind.rawValue }
}

public struct OnboardingSetupStep: Identifiable, Sendable {
    public let id: String
    public let title: String
    public let priority: Int
    public let status: OnboardingSetupStepStatus
    public let detail: String
    public let remediationAction: OnboardingSetupAction?
}

public enum HomeFirstSessionStepID: String, CaseIterable, Codable, Sendable {
    case sendMessage = "send_message"
    case sendCommunication = "send_communication"
    case createTask = "create_task"
    case reviewApprovals = "review_approvals"
}

public struct HomeFirstSessionStep: Identifiable, Sendable, Equatable {
    public let id: HomeFirstSessionStepID
    public let title: String
    public let detail: String
    public let actionTitle: String
    public let destinationSection: AppSection
    public let isComplete: Bool
}

public struct HomeFirstSessionGuidanceContext: Sendable, Equatable {
    public let step: HomeFirstSessionStep
    public let stepNumber: Int
    public let totalSteps: Int
    public let isCurrentSectionDestination: Bool

    public var progressLabel: String {
        "Step \(stepNumber) of \(totalSteps)"
    }
}

public struct HomeFirstSessionFunnelMilestoneItem: Identifiable, Sendable, Equatable {
    public let id: HomeFirstSessionStepID
    public let title: String
    public let isComplete: Bool
    public let completedAtRaw: String?
    public let completedAtLabel: String?
    public let completionSource: String?
    public let completionSourceLabel: String?
}

public struct HomeFirstSessionFunnelDiagnostics: Sendable, Equatable {
    public let workspaceID: String
    public let completedCount: Int
    public let totalCount: Int
    public let firstCompletedAtRaw: String?
    public let firstCompletedAtLabel: String?
    public let latestCompletedAtRaw: String?
    public let latestCompletedAtLabel: String?
    public let milestones: [HomeFirstSessionFunnelMilestoneItem]

    public var completionRateLabel: String {
        guard totalCount > 0 else {
            return "0%"
        }
        let percentage = Int((Double(completedCount) / Double(totalCount)) * 100.0)
        return "\(percentage)%"
    }
}

public enum AppNotificationLevel: String, Codable, Sendable, Equatable {
    case success
    case error
    case info
    case progress

    public var symbolName: String {
        switch self {
        case .success:
            return "checkmark.circle.fill"
        case .error:
            return "exclamationmark.octagon.fill"
        case .info:
            return "info.circle.fill"
        case .progress:
            return "clock.arrow.circlepath"
        }
    }

    public var tint: Color {
        switch self {
        case .success:
            return .green
        case .error:
            return .red
        case .info:
            return .secondary
        case .progress:
            return .orange
        }
    }
}

public struct AppNotificationItem: Identifiable, Codable, Sendable, Equatable {
    public let id: String
    public let workspaceID: String
    public let source: String
    public let action: String
    public let level: AppNotificationLevel
    public let message: String
    public let createdAt: Date
    public var readAt: Date?

    public init(
        id: String = UUID().uuidString,
        workspaceID: String,
        source: String,
        action: String,
        level: AppNotificationLevel,
        message: String,
        createdAt: Date = Date(),
        readAt: Date? = nil
    ) {
        self.id = id
        self.workspaceID = workspaceID
        self.source = source
        self.action = action
        self.level = level
        self.message = message
        self.createdAt = createdAt
        self.readAt = readAt
    }

    public var isRead: Bool {
        readAt != nil
    }
}

public enum NotificationInboxIntent: String, CaseIterable, Sendable, Equatable, Hashable {
    case needsAttention
    case workflow
    case runtime
    case diagnostics
    case general

    public var title: String {
        switch self {
        case .needsAttention:
            return "Needs Attention"
        case .workflow:
            return "Workflow Updates"
        case .runtime:
            return "Runtime and Setup"
        case .diagnostics:
            return "Diagnostics"
        case .general:
            return "General"
        }
    }

    public var symbolName: String {
        switch self {
        case .needsAttention:
            return "exclamationmark.triangle.fill"
        case .workflow:
            return "checklist"
        case .runtime:
            return "gearshape.2"
        case .diagnostics:
            return "waveform.path.ecg"
        case .general:
            return "bell"
        }
    }

    public var tint: Color {
        switch self {
        case .needsAttention:
            return .orange
        case .workflow:
            return .green
        case .runtime:
            return .secondary
        case .diagnostics:
            return .blue
        case .general:
            return .secondary
        }
    }

    public var sortPriority: Int {
        switch self {
        case .needsAttention:
            return 0
        case .workflow:
            return 1
        case .runtime:
            return 2
        case .diagnostics:
            return 3
        case .general:
            return 4
        }
    }
}

public struct NotificationInboxSection: Identifiable, Sendable, Equatable {
    public let intent: NotificationInboxIntent
    public let items: [AppNotificationItem]

    public var id: String {
        intent.rawValue
    }

    public var unreadCount: Int {
        items.filter { !$0.isRead }.count
    }
}

public struct NotificationInboxAction: Identifiable, Sendable, Equatable {
    public enum Kind: Sendable, Equatable {
        case openSection(AppSection)
    }

    public let kind: Kind
    public let title: String
    public let symbolName: String

    public var id: String {
        switch kind {
        case .openSection(let section):
            return "open_section_\(section.rawValue)"
        }
    }
}

public enum EmptyStateRemediationActionID: String, Sendable, Equatable {
    case openConfiguration
    case openModels
    case openChannels
    case openConnectors
    case openTasks
    case openChat
    case openAutomation
    case refreshDaemonStatus
    case recheckChatRoute
    case refreshCommunications
    case refreshAutomation
    case refreshApprovals
    case refreshTasks
    case refreshInspect
    case refreshChannels
    case refreshConnectors
    case refreshModels
    case runProviderChecks
}

public struct EmptyStateRemediationAction: Identifiable, Sendable, Equatable {
    public let actionID: EmptyStateRemediationActionID
    public let title: String
    public let symbolName: String
    public let isProminent: Bool
    public let isDisabled: Bool

    public init(
        actionID: EmptyStateRemediationActionID,
        title: String,
        symbolName: String,
        isProminent: Bool = false,
        isDisabled: Bool = false
    ) {
        self.actionID = actionID
        self.title = title
        self.symbolName = symbolName
        self.isProminent = isProminent
        self.isDisabled = isDisabled
    }

    public var id: String { actionID.rawValue }
}

enum PanelProblemRemediationActionID: String, Sendable, Equatable {
    case openConfiguration
    case retry
    case openInspect
}

struct PanelProblemRemediationAction: Identifiable, Sendable, Equatable {
    let actionID: PanelProblemRemediationActionID
    let title: String
    let symbolName: String
    let role: PanelActionRole
    let isEnabled: Bool
    let disabledReason: String?

    init(
        actionID: PanelProblemRemediationActionID,
        title: String,
        symbolName: String,
        role: PanelActionRole,
        isEnabled: Bool = true,
        disabledReason: String? = nil
    ) {
        self.actionID = actionID
        self.title = title
        self.symbolName = symbolName
        self.role = role
        self.isEnabled = isEnabled
        self.disabledReason = disabledReason
    }

    var id: String {
        actionID.rawValue
    }
}

struct PanelProblemRemediationContext: Sendable, Equatable {
    enum Kind: String, Sendable, Equatable {
        case authScope = "auth_scope"
        case rateLimitExceeded = "rate_limit_exceeded"

        var title: String {
            switch self {
            case .authScope:
                return "Additional Access Is Required"
            case .rateLimitExceeded:
                return "Requests Are Being Throttled"
            }
        }

        var symbolName: String {
            switch self {
            case .authScope:
                return "lock.trianglebadge.exclamationmark"
            case .rateLimitExceeded:
                return "gauge.with.dots.needle.50percent"
            }
        }

        var tint: Color {
            switch self {
            case .authScope:
                return .orange
            case .rateLimitExceeded:
                return .yellow
            }
        }
    }

    let section: AppSection
    let kind: Kind
    let detail: String
    let actions: [PanelProblemRemediationAction]

    init(
        section: AppSection,
        kind: Kind,
        detail: String,
        actions: [PanelProblemRemediationAction]
    ) {
        self.section = section
        self.kind = kind
        self.detail = detail
        self.actions = actions
    }
}

public struct HighImpactActionConfirmation: Identifiable, Sendable, Equatable {
    public let id: String
    public let title: String
    public let message: String
    public let confirmButtonTitle: String
    public let isDestructive: Bool
    public let irreversibleNote: String?

    public init(
        id: String = UUID().uuidString,
        title: String,
        message: String,
        confirmButtonTitle: String,
        isDestructive: Bool,
        irreversibleNote: String? = nil
    ) {
        self.id = id
        self.title = title
        self.message = message
        self.confirmButtonTitle = confirmButtonTitle
        self.isDestructive = isDestructive
        self.irreversibleNote = irreversibleNote
    }

    public var fullMessage: String {
        if let irreversibleNote, !irreversibleNote.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            return "\(message)\n\n\(irreversibleNote)"
        }
        return message
    }
}

public struct UndoActionPrompt: Identifiable, Sendable, Equatable {
    public let id: String
    public let title: String
    public let message: String
    public let actionTitle: String

    public init(
        id: String = UUID().uuidString,
        title: String,
        message: String,
        actionTitle: String = "Undo"
    ) {
        self.id = id
        self.title = title
        self.message = message
        self.actionTitle = actionTitle
    }
}
