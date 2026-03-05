import Foundation

public enum V2ProblemContext: String, Sendable, Equatable {
    case setup
    case replay
    case ask
    case connectors
    case models
    case realtime
}

public enum V2ProblemActionID: String, Sendable, Equatable {
    case retry
    case clearReplayFilters
    case openGetStarted
    case openConnectorsAndModels
    case openReplayAndAsk
    case reauthenticate
    case retryRealtime
}

public struct V2ProblemAction: Identifiable, Sendable, Equatable {
    public var id: String { actionID.rawValue }
    public let actionID: V2ProblemActionID
    public let label: String
    public let isPrimary: Bool
    public let isEnabled: Bool
    public let disabledReason: String?

    public init(
        actionID: V2ProblemActionID,
        label: String,
        isPrimary: Bool,
        isEnabled: Bool = true,
        disabledReason: String? = nil
    ) {
        self.actionID = actionID
        self.label = label
        self.isPrimary = isPrimary
        self.isEnabled = isEnabled
        self.disabledReason = disabledReason
    }
}

public struct V2PanelProblemState: Sendable, Equatable {
    public enum Kind: String, Sendable, Equatable {
        case missingAuth
        case authScope
        case rateLimited
        case daemonUnavailable
        case connectivity
        case decoding
        case validation
        case unknown
    }

    public let kind: Kind
    public let title: String
    public let summary: String
    public let correlationID: String?
    public let actions: [V2ProblemAction]

    public init(
        kind: Kind,
        title: String,
        summary: String,
        correlationID: String? = nil,
        actions: [V2ProblemAction]
    ) {
        self.kind = kind
        self.title = title
        self.summary = summary
        self.correlationID = correlationID
        self.actions = actions
    }
}

enum V2DaemonProblemMapper {
    static func map(error: Error, context: V2ProblemContext) -> V2PanelProblemState {
        guard let daemonError = error as? V2DaemonAPIError else {
            return V2PanelProblemState(
                kind: .unknown,
                title: "Unexpected failure",
                summary: error.localizedDescription,
                actions: defaultActions(for: context)
            )
        }

        switch daemonError {
        case .missingAuthToken:
            return V2PanelProblemState(
                kind: .missingAuth,
                title: "Assistant access token is required",
                summary: "Set an access token in Get Started before loading live assistant state.",
                actions: [
                    V2ProblemAction(actionID: .openGetStarted, label: "Open Get Started", isPrimary: true),
                    V2ProblemAction(actionID: .retry, label: "Retry", isPrimary: false)
                ]
            )
        case .transport(let message):
            return V2PanelProblemState(
                kind: .connectivity,
                title: context == .realtime ? "Realtime stream disconnected" : "Can’t reach daemon",
                summary: message,
                actions: [
                    V2ProblemAction(actionID: context == .realtime ? .retryRealtime : .retry, label: context == .realtime ? "Retry Realtime Stream" : "Retry", isPrimary: true),
                    remediationOpenAction(for: context)
                ]
            )
        case .decoding:
            return V2PanelProblemState(
                kind: .decoding,
                title: "Daemon response could not be parsed",
                summary: "Response schema mismatch detected. Refresh and retry. If this persists, inspect daemon contract version.",
                actions: defaultActions(for: context)
            )
        case .invalidBaseURL:
            return V2PanelProblemState(
                kind: .validation,
                title: "Daemon URL is invalid",
                summary: "Update daemon address in Get Started, then retry.",
                actions: [
                    V2ProblemAction(actionID: .openGetStarted, label: "Open Get Started", isPrimary: true),
                    V2ProblemAction(actionID: .retry, label: "Retry", isPrimary: false)
                ]
            )
        case .invalidResponse:
            return V2PanelProblemState(
                kind: .daemonUnavailable,
                title: "Daemon returned an invalid response",
                summary: "Check daemon health and retry.",
                actions: defaultActions(for: context)
            )
        case .server(let statusCode, let message):
            if statusCode == 401 {
                return V2PanelProblemState(
                    kind: .missingAuth,
                    title: "Assistant access token was rejected",
                    summary: "Token may be missing, expired, or scoped incorrectly. Update token and retry.",
                    actions: [
                        V2ProblemAction(actionID: .reauthenticate, label: "Update Token", isPrimary: true),
                        V2ProblemAction(actionID: .retry, label: "Retry", isPrimary: false)
                    ]
                )
            }
            if statusCode == 429 {
                return V2PanelProblemState(
                    kind: .rateLimited,
                    title: "Daemon is rate limited",
                    summary: message,
                    actions: [
                        V2ProblemAction(actionID: .retry, label: "Retry", isPrimary: true),
                        remediationOpenAction(for: context)
                    ]
                )
            }
            return V2PanelProblemState(
                kind: .daemonUnavailable,
                title: "Daemon request failed",
                summary: message,
                actions: defaultActions(for: context)
            )
        case .serverProblem(_, let message, let code, let details, let correlationID):
            if code == "auth_scope_denied" || code == "insufficient_scope" {
                return V2PanelProblemState(
                    kind: .authScope,
                    title: "Permission scope is missing",
                    summary: details?.remediation?.hint
                        ?? "The assistant token does not include the required scope for this action.",
                    correlationID: correlationID,
                    actions: [
                        V2ProblemAction(actionID: .reauthenticate, label: "Update Token Scope", isPrimary: true),
                        V2ProblemAction(actionID: .retry, label: "Retry", isPrimary: false)
                    ]
                )
            }

            if code == "rate_limit_exceeded" {
                return V2PanelProblemState(
                    kind: .rateLimited,
                    title: "Daemon is rate limited",
                    summary: details?.remediation?.hint ?? message,
                    correlationID: correlationID,
                    actions: [
                        V2ProblemAction(actionID: .retry, label: "Retry", isPrimary: true),
                        remediationOpenAction(for: context)
                    ]
                )
            }

            return V2PanelProblemState(
                kind: .daemonUnavailable,
                title: "Daemon rejected request",
                summary: message,
                correlationID: correlationID,
                actions: defaultActions(for: context)
            )
        }
    }

    private static func defaultActions(for context: V2ProblemContext) -> [V2ProblemAction] {
        [
            V2ProblemAction(actionID: .retry, label: "Retry", isPrimary: true),
            remediationOpenAction(for: context)
        ]
    }

    private static func remediationOpenAction(for context: V2ProblemContext) -> V2ProblemAction {
        switch context {
        case .setup:
            return V2ProblemAction(actionID: .openGetStarted, label: "Open Get Started", isPrimary: false)
        case .replay, .ask, .realtime:
            return V2ProblemAction(actionID: .openReplayAndAsk, label: "Open Replay & Ask", isPrimary: false)
        case .connectors, .models:
            return V2ProblemAction(actionID: .openConnectorsAndModels, label: "Open Connectors & Models", isPrimary: false)
        }
    }
}
