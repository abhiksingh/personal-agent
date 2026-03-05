import Foundation

public struct CommunicationsFilterContext: Codable, Sendable, Equatable {
    public static let allChannelsID = "__all_channels__"
    public static let allThreadsID = "__all_threads__"

    public var searchText: String
    public var channelFilterID: String
    public var directionFilterRawValue: String
    public var threadFilterID: String
    public var compactScanModeEnabled: Bool

    public init(
        searchText: String = "",
        channelFilterID: String = CommunicationsFilterContext.allChannelsID,
        directionFilterRawValue: String = "all",
        threadFilterID: String = CommunicationsFilterContext.allThreadsID,
        compactScanModeEnabled: Bool = false
    ) {
        self.searchText = searchText
        self.channelFilterID = channelFilterID
        self.directionFilterRawValue = directionFilterRawValue
        self.threadFilterID = threadFilterID
        self.compactScanModeEnabled = compactScanModeEnabled
    }

    public var activeFilterSummaryParts: [String] {
        var parts: [String] = []
        if let search = normalizedSearchText {
            parts.append("Search: \(search)")
        }
        if channelFilterID != Self.allChannelsID {
            parts.append("Channel: \(channelFilterID)")
        }
        if let directionLabel = Self.directionLabel(fromRawValue: directionFilterRawValue) {
            parts.append("Direction: \(directionLabel)")
        }
        if threadFilterID != Self.allThreadsID {
            parts.append("Thread: \(threadFilterID)")
        }
        if compactScanModeEnabled {
            parts.append("Density: Compact")
        }
        return parts
    }

    private var normalizedSearchText: String? {
        let trimmed = searchText.trimmingCharacters(in: .whitespacesAndNewlines)
        return trimmed.isEmpty ? nil : trimmed
    }

    private static func directionLabel(fromRawValue rawValue: String) -> String? {
        switch rawValue.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() {
        case "", "all":
            return nil
        case "inbound":
            return "Inbound"
        case "outbound":
            return "Outbound"
        case "other":
            return "Other"
        default:
            return rawValue
        }
    }
}

public struct CommunicationsTriageContext: Codable, Sendable, Equatable {
    public var handledThreadIDs: [String]
    public var followUpThreadIDs: [String]
    public var seenThreadIDs: [String]

    public init(
        handledThreadIDs: [String] = [],
        followUpThreadIDs: [String] = [],
        seenThreadIDs: [String] = []
    ) {
        self.handledThreadIDs = handledThreadIDs
        self.followUpThreadIDs = followUpThreadIDs
        self.seenThreadIDs = seenThreadIDs
    }
}

public struct TasksFilterContext: Codable, Sendable, Equatable {
    public var searchText: String
    public var stateFilter: String
    public var priorityFilterRawValue: String
    public var principalFilter: String

    public init(
        searchText: String = "",
        stateFilter: String = "All States",
        priorityFilterRawValue: String = "all",
        principalFilter: String = "All Principals"
    ) {
        self.searchText = searchText
        self.stateFilter = stateFilter
        self.priorityFilterRawValue = priorityFilterRawValue
        self.principalFilter = principalFilter
    }

    public var activeFilterSummaryParts: [String] {
        var parts: [String] = []
        if let search = normalizedSearchText {
            parts.append("Search: \(search)")
        }
        if stateFilter != "All States" {
            parts.append("State: \(stateFilter)")
        }
        if let priorityLabel = Self.priorityLabel(fromRawValue: priorityFilterRawValue) {
            parts.append("Priority: \(priorityLabel)")
        }
        if principalFilter != "All Principals" {
            parts.append("Principal: \(principalFilter)")
        }
        return parts
    }

    private var normalizedSearchText: String? {
        let trimmed = searchText.trimmingCharacters(in: .whitespacesAndNewlines)
        return trimmed.isEmpty ? nil : trimmed
    }

    private static func priorityLabel(fromRawValue rawValue: String) -> String? {
        switch rawValue.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() {
        case "", "all":
            return nil
        case "high":
            return "High"
        case "medium":
            return "Medium"
        case "low":
            return "Low"
        default:
            return rawValue
        }
    }
}

public struct ApprovalsFilterContext: Codable, Sendable, Equatable {
    public var searchText: String

    public init(searchText: String = "") {
        self.searchText = searchText
    }

    public var activeFilterSummaryParts: [String] {
        let trimmed = searchText.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else {
            return []
        }
        return ["Search: \(trimmed)"]
    }
}

public enum InspectPresentationMode: String, Codable, Sendable, Equatable {
    case activity
    case trace
    case gallery
}

public struct InspectFilterContext: Codable, Sendable, Equatable {
    public var metadataFilterText: String
    public var statusFilterRawValue: String
    public var metadataScopeRawValue: String
    public var groupingRawValue: String
    public var inspectModeRawValue: String

    public init(
        metadataFilterText: String = "",
        statusFilterRawValue: String = "all",
        metadataScopeRawValue: String = "all",
        groupingRawValue: String = "none",
        inspectModeRawValue: String = InspectPresentationMode.activity.rawValue
    ) {
        self.metadataFilterText = metadataFilterText
        self.statusFilterRawValue = statusFilterRawValue
        self.metadataScopeRawValue = metadataScopeRawValue
        self.groupingRawValue = groupingRawValue
        self.inspectModeRawValue = inspectModeRawValue
    }

    enum CodingKeys: String, CodingKey {
        case metadataFilterText
        case statusFilterRawValue
        case metadataScopeRawValue
        case groupingRawValue
        case inspectModeRawValue
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        metadataFilterText = try container.decodeIfPresent(String.self, forKey: .metadataFilterText) ?? ""
        statusFilterRawValue = try container.decodeIfPresent(String.self, forKey: .statusFilterRawValue) ?? "all"
        metadataScopeRawValue = try container.decodeIfPresent(String.self, forKey: .metadataScopeRawValue) ?? "all"
        groupingRawValue = try container.decodeIfPresent(String.self, forKey: .groupingRawValue) ?? "none"
        inspectModeRawValue =
            try container.decodeIfPresent(String.self, forKey: .inspectModeRawValue) ?? InspectPresentationMode.activity.rawValue
    }

    public var inspectMode: InspectPresentationMode {
        InspectPresentationMode(rawValue: inspectModeRawValue) ?? .activity
    }

    public var activeFilterSummaryParts: [String] {
        if inspectMode == .gallery {
            return []
        }
        var parts: [String] = []
        if let text = normalizedMetadataFilterText {
            parts.append("Match: \(text)")
        }
        if let statusLabel = Self.statusLabel(fromRawValue: statusFilterRawValue) {
            parts.append("Status: \(statusLabel)")
        }
        if inspectMode == .trace {
            if let scopeLabel = Self.scopeLabel(fromRawValue: metadataScopeRawValue) {
                parts.append("Field: \(scopeLabel)")
            }
            if let groupingLabel = Self.groupingLabel(fromRawValue: groupingRawValue) {
                parts.append("Group: \(groupingLabel)")
            }
        }
        return parts
    }

    private var normalizedMetadataFilterText: String? {
        let trimmed = metadataFilterText.trimmingCharacters(in: .whitespacesAndNewlines)
        return trimmed.isEmpty ? nil : trimmed
    }

    private static func statusLabel(fromRawValue rawValue: String) -> String? {
        switch rawValue.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() {
        case "", "all":
            return nil
        case "success":
            return "Success"
        case "running":
            return "Running"
        case "failure":
            return "Failure"
        default:
            return rawValue
        }
    }

    private static func scopeLabel(fromRawValue rawValue: String) -> String? {
        switch rawValue.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() {
        case "", "all":
            return nil
        case "task":
            return "Task"
        case "run":
            return "Run"
        case "correlation":
            return "Correlation"
        case "provider":
            return "Provider"
        case "model":
            return "Model"
        default:
            return rawValue
        }
    }

    private static func groupingLabel(fromRawValue rawValue: String) -> String? {
        switch rawValue.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() {
        case "", "none":
            return nil
        case "task":
            return "Task"
        case "run":
            return "Run"
        case "correlation":
            return "Correlation"
        case "provider":
            return "Provider"
        case "model":
            return "Model"
        default:
            return rawValue
        }
    }
}

struct WorkspacePanelFilterContext: Codable, Sendable, Equatable {
    var communications = CommunicationsFilterContext()
    var tasks = TasksFilterContext()
    var approvals = ApprovalsFilterContext()
    var inspect = InspectFilterContext()
}

public struct CommunicationsComposeDraftContext: Codable, Sendable, Equatable {
    public var isPresented: Bool
    public var flowID: String
    public var sourceChannel: String
    public var threadID: String
    public var connectorID: String
    public var destination: String
    public var message: String

    public init(
        isPresented: Bool = false,
        flowID: String = "new_message",
        sourceChannel: String = "message",
        threadID: String = "",
        connectorID: String = "",
        destination: String = "",
        message: String = ""
    ) {
        self.isPresented = isPresented
        self.flowID = flowID
        self.sourceChannel = sourceChannel
        self.threadID = threadID
        self.connectorID = connectorID
        self.destination = destination
        self.message = message
    }
}

public struct TasksSubmitDraftContext: Codable, Sendable, Equatable {
    public var isPresented: Bool
    public var title: String
    public var description: String
    public var taskClass: String
    public var priorityRawValue: String
    public var requestedByActorID: String
    public var subjectPrincipalActorID: String

    public init(
        isPresented: Bool = false,
        title: String = "",
        description: String = "",
        taskClass: String = "chat",
        priorityRawValue: String = "medium",
        requestedByActorID: String = "default",
        subjectPrincipalActorID: String = "default"
    ) {
        self.isPresented = isPresented
        self.title = title
        self.description = description
        self.taskClass = taskClass
        self.priorityRawValue = priorityRawValue
        self.requestedByActorID = requestedByActorID
        self.subjectPrincipalActorID = subjectPrincipalActorID
    }

    enum CodingKeys: String, CodingKey {
        case isPresented
        case title
        case description
        case taskClass
        case priorityRawValue
        case requestedByActorID
        case subjectPrincipalActorID
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        isPresented = try container.decodeIfPresent(Bool.self, forKey: .isPresented) ?? false
        title = try container.decodeIfPresent(String.self, forKey: .title) ?? ""
        description = try container.decodeIfPresent(String.self, forKey: .description) ?? ""
        taskClass = try container.decodeIfPresent(String.self, forKey: .taskClass) ?? "chat"
        priorityRawValue = try container.decodeIfPresent(String.self, forKey: .priorityRawValue) ?? "medium"
        requestedByActorID = try container.decodeIfPresent(String.self, forKey: .requestedByActorID) ?? "default"
        subjectPrincipalActorID =
            try container.decodeIfPresent(String.self, forKey: .subjectPrincipalActorID) ?? "default"
    }
}

public struct WorkspaceContinuityContext: Codable, Sendable, Equatable {
    public var expandedChannelCardIDs: [String]
    public var expandedConnectorCardIDs: [String]
    public var communicationsComposeDraft: CommunicationsComposeDraftContext?
    public var tasksSubmitDraft: TasksSubmitDraftContext?

    public init(
        expandedChannelCardIDs: [String] = [],
        expandedConnectorCardIDs: [String] = [],
        communicationsComposeDraft: CommunicationsComposeDraftContext? = nil,
        tasksSubmitDraft: TasksSubmitDraftContext? = nil
    ) {
        self.expandedChannelCardIDs = expandedChannelCardIDs
        self.expandedConnectorCardIDs = expandedConnectorCardIDs
        self.communicationsComposeDraft = communicationsComposeDraft
        self.tasksSubmitDraft = tasksSubmitDraft
    }
}
