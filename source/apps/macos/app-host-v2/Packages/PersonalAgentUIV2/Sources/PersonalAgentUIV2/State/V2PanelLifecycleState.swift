import Foundation

public enum V2WorkflowPanel: String, CaseIterable, Sendable, Hashable {
    case getStarted
    case replayAndAsk
    case connectorsAndModels
}

public enum V2PanelLifecycleKind: String, Sendable, Equatable {
    case loading
    case ready
    case empty
    case degraded
    case error
}

public struct V2PanelLifecycleState: Sendable, Equatable {
    public var kind: V2PanelLifecycleKind
    public var summary: String
    public var actions: [V2ProblemAction]

    public init(kind: V2PanelLifecycleKind, summary: String, actions: [V2ProblemAction] = []) {
        self.kind = kind
        self.summary = summary
        self.actions = actions
    }

    public static func loading(_ summary: String, actions: [V2ProblemAction] = []) -> V2PanelLifecycleState {
        V2PanelLifecycleState(kind: .loading, summary: summary, actions: actions)
    }

    public static func ready(_ summary: String = "") -> V2PanelLifecycleState {
        V2PanelLifecycleState(kind: .ready, summary: summary, actions: [])
    }

    public static func empty(_ summary: String, actions: [V2ProblemAction] = []) -> V2PanelLifecycleState {
        V2PanelLifecycleState(kind: .empty, summary: summary, actions: actions)
    }

    public static func degraded(_ summary: String, actions: [V2ProblemAction] = []) -> V2PanelLifecycleState {
        V2PanelLifecycleState(kind: .degraded, summary: summary, actions: actions)
    }

    public static func error(_ summary: String, actions: [V2ProblemAction] = []) -> V2PanelLifecycleState {
        V2PanelLifecycleState(kind: .error, summary: summary, actions: actions)
    }
}
