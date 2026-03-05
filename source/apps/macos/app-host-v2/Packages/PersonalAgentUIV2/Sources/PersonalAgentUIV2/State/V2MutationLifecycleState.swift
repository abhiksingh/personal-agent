import Foundation

public enum V2MutationActionID: String, CaseIterable, Sendable, Hashable {
    case askSend
    case replayApprove
    case replayReject
    case replayRetry
    case replayComplete
    case connectorToggle
    case connectorCheck
    case connectorSaveConfig
    case connectorPermission
    case modelToggle
    case modelSetPrimary
    case modelRouteSimulate
    case modelRouteExplain
    case tokenSave
    case tokenClear
    case daemonProbe
}

public enum V2MutationLifecyclePhase: String, Sendable, Equatable {
    case idle
    case inFlight
    case succeeded
    case failed
    case disabled
}

public struct V2MutationLifecycleState: Sendable, Equatable {
    public var phase: V2MutationLifecyclePhase
    public var message: String?

    public init(phase: V2MutationLifecyclePhase, message: String? = nil) {
        self.phase = phase
        self.message = message
    }

    public static let idle = V2MutationLifecycleState(phase: .idle)

    public static func inFlight(_ message: String? = nil) -> V2MutationLifecycleState {
        V2MutationLifecycleState(phase: .inFlight, message: message)
    }

    public static func succeeded(_ message: String? = nil) -> V2MutationLifecycleState {
        V2MutationLifecycleState(phase: .succeeded, message: message)
    }

    public static func failed(_ message: String? = nil) -> V2MutationLifecycleState {
        V2MutationLifecycleState(phase: .failed, message: message)
    }

    public static func disabled(_ reason: String) -> V2MutationLifecycleState {
        V2MutationLifecycleState(phase: .disabled, message: reason)
    }

    public var isInFlight: Bool { phase == .inFlight }
    public var isDisabled: Bool { phase == .disabled }
}
