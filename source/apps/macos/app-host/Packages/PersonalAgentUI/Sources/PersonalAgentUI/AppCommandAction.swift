import Foundation

public enum AppCommandActionCategory: String, CaseIterable, Sendable {
    case navigation
    case diagnostics
    case workflow
    case runtime

    public var title: String {
        switch self {
        case .navigation:
            return "Navigation"
        case .diagnostics:
            return "Diagnostics"
        case .workflow:
            return "Workflow"
        case .runtime:
            return "Runtime"
        }
    }
}

public enum AppCommandActionID: String, CaseIterable, Identifiable, Sendable {
    case doSendMessage
    case doSendEmail
    case doCreateTask
    case doReviewApprovals
    case doInspectIssue
    case openConfiguration
    case openChat
    case openCommunications
    case openAutomation
    case openApprovals
    case openTasks
    case openModels
    case openInspect
    case openChannels
    case openConnectors
    case refreshCurrentSection
    case openNotificationCenter
    case setSimpleDensityMode
    case setAdvancedDensityMode
    case performOnboardingFixNextStep
    case startDaemon
    case stopDaemon
    case restartDaemon

    public var id: String { rawValue }
}

public struct AppCommandActionItem: Identifiable, Sendable, Equatable {
    public let actionID: AppCommandActionID
    public let category: AppCommandActionCategory
    public let title: String
    public let subtitle: String?
    public let symbolName: String
    public let shortcutHint: String?
    public let isEnabled: Bool
    public let disabledReason: String?

    public var id: String { actionID.rawValue }
}

public enum CommandPaletteObjectKind: String, CaseIterable, Sendable {
    case taskRun
    case approval
    case thread
    case connector
    case model

    public var title: String {
        switch self {
        case .taskRun:
            return "Task"
        case .approval:
            return "Approval"
        case .thread:
            return "Thread"
        case .connector:
            return "Connector"
        case .model:
            return "Model"
        }
    }

    public var symbolName: String {
        switch self {
        case .taskRun:
            return "checklist"
        case .approval:
            return "checkmark.shield"
        case .thread:
            return "bubble.left.and.bubble.right"
        case .connector:
            return "cable.connector"
        case .model:
            return "cpu"
        }
    }

    public var rankingOrder: Int {
        switch self {
        case .taskRun:
            return 0
        case .approval:
            return 1
        case .thread:
            return 2
        case .connector:
            return 3
        case .model:
            return 4
        }
    }
}

public enum CommandPaletteObjectTarget: Sendable, Equatable {
    case taskRun(rowID: String)
    case approval(approvalID: String)
    case thread(threadID: String)
    case connector(connectorID: String)
    case model(providerID: String, modelKey: String)
}

public struct CommandPaletteObjectSearchItem: Identifiable, Sendable, Equatable {
    public let id: String
    public let kind: CommandPaletteObjectKind
    public let title: String
    public let subtitle: String?
    public let target: CommandPaletteObjectTarget

    public init(
        id: String,
        kind: CommandPaletteObjectKind,
        title: String,
        subtitle: String?,
        target: CommandPaletteObjectTarget
    ) {
        self.id = id
        self.kind = kind
        self.title = title
        self.subtitle = subtitle
        self.target = target
    }

    public var symbolName: String { kind.symbolName }
}
