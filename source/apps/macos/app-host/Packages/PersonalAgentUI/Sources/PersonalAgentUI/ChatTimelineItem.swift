import Foundation
import SwiftUI

public enum ChatTimelineItemKind: String, Sendable, Equatable {
    case userMessage = "user_message"
    case assistantMessage = "assistant_message"
    case toolCall = "tool_call"
    case toolResult = "tool_result"
    case approvalRequest = "approval_request"
    case approvalDecision = "approval_decision"
    case systemStatus = "system_status"
}

public enum ChatTimelineItemState: String, Sendable, Equatable {
    case pending
    case inFlight = "in_flight"
    case blocked
    case completed
    case failed

    public var label: String {
        switch self {
        case .pending:
            return "Pending"
        case .inFlight:
            return "Running"
        case .blocked:
            return "Blocked"
        case .completed:
            return "Complete"
        case .failed:
            return "Failed"
        }
    }

    public var symbolName: String {
        switch self {
        case .pending:
            return "clock"
        case .inFlight:
            return "hourglass"
        case .blocked:
            return "exclamationmark.triangle.fill"
        case .completed:
            return "checkmark.circle.fill"
        case .failed:
            return "xmark.octagon.fill"
        }
    }

    public var tint: Color {
        switch self {
        case .pending:
            return .secondary
        case .inFlight:
            return .blue
        case .blocked:
            return .orange
        case .completed:
            return .green
        case .failed:
            return .red
        }
    }
}

public enum ChatTimelineActionStyle: Sendable, Equatable {
    case primary
    case secondary
    case destructive
}

public enum ChatTimelineActionIntent: String, Sendable, Equatable {
    case openApprovals = "open_approvals"
    case openTasks = "open_tasks"
    case openInspect = "open_inspect"
    case openConfiguration = "open_configuration"
    case openConnectors = "open_connectors"
    case openChannels = "open_channels"
    case resumeTurn = "resume_turn"
    case retryTurn = "retry_turn"
    case cancelTurn = "cancel_turn"
}

public struct ChatTimelineActionItem: Identifiable, Sendable, Equatable {
    public let id: String
    public let title: String
    public let intent: ChatTimelineActionIntent
    public let style: ChatTimelineActionStyle
    public let enabled: Bool
    public let disabledReason: String?

    public init(
        id: String,
        title: String,
        intent: ChatTimelineActionIntent,
        style: ChatTimelineActionStyle = .secondary,
        enabled: Bool = true,
        disabledReason: String? = nil
    ) {
        self.id = id
        self.title = title
        self.intent = intent
        self.style = style
        self.enabled = enabled
        self.disabledReason = disabledReason
    }
}

public struct ChatTimelineDetailItem: Identifiable, Sendable, Equatable {
    public let id: String
    public let label: String
    public let value: String

    public init(label: String, value: String) {
        self.id = "\(label)::\(value)"
        self.label = label
        self.value = value
    }
}

public struct ChatTimelineItem: Identifiable, Sendable, Equatable {
    public let id: String
    public let kind: ChatTimelineItemKind
    public let state: ChatTimelineItemState
    public let title: String
    public let summary: String
    public let content: String?
    public let timestamp: Date
    public let daemonRole: String?
    public let includeInDaemonContext: Bool
    public let correlationID: String?
    public let taskID: String?
    public let runID: String?
    public let approvalRequestID: String?
    public let toolCallID: String?
    public let toolName: String?
    public let toolChainIndex: Int?
    public let toolChainStep: Int?
    public let toolChainStepCount: Int?
    public let details: [ChatTimelineDetailItem]

    public init(
        id: String = UUID().uuidString.lowercased(),
        kind: ChatTimelineItemKind,
        state: ChatTimelineItemState,
        title: String,
        summary: String,
        content: String? = nil,
        timestamp: Date = .now,
        daemonRole: String? = nil,
        includeInDaemonContext: Bool = false,
        correlationID: String? = nil,
        taskID: String? = nil,
        runID: String? = nil,
        approvalRequestID: String? = nil,
        toolCallID: String? = nil,
        toolName: String? = nil,
        toolChainIndex: Int? = nil,
        toolChainStep: Int? = nil,
        toolChainStepCount: Int? = nil,
        details: [ChatTimelineDetailItem] = []
    ) {
        self.id = id
        self.kind = kind
        self.state = state
        self.title = title
        self.summary = summary
        self.content = content
        self.timestamp = timestamp
        self.daemonRole = daemonRole
        self.includeInDaemonContext = includeInDaemonContext
        self.correlationID = correlationID
        self.taskID = taskID
        self.runID = runID
        self.approvalRequestID = approvalRequestID
        self.toolCallID = toolCallID
        self.toolName = toolName
        self.toolChainIndex = toolChainIndex
        self.toolChainStep = toolChainStep
        self.toolChainStepCount = toolChainStepCount
        self.details = details
    }

    public var daemonContextMessage: (role: String, content: String)? {
        guard includeInDaemonContext,
              let daemonRole,
              let content,
              !content.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else {
            return nil
        }
        return (role: daemonRole, content: content)
    }

    public var isUserMessage: Bool {
        kind == .userMessage
    }

    public var hasDetails: Bool {
        !details.isEmpty
    }

    public var toolChainLabel: String? {
        guard let toolChainIndex else {
            return nil
        }
        return "Chain \(toolChainIndex)"
    }

    public var toolChainStepLabel: String? {
        guard let toolChainStep, let toolChainStepCount, toolChainStepCount > 0 else {
            return nil
        }
        return "Step \(toolChainStep) of \(toolChainStepCount)"
    }
}
