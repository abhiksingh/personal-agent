import Foundation
import SwiftUI

@MainActor
final class AppPanelProblemStore: ObservableObject {
    struct Signal: Equatable {
        let kind: PanelProblemRemediationContext.Kind
        let detail: String
        let correlationID: String?
    }

    @Published private var signalsBySection: [AppSection: Signal] = [:]

    func remediationContext(for section: AppSection, retryInFlight: Bool) -> PanelProblemRemediationContext? {
        guard let signal = signalsBySection[section] else {
            return nil
        }
        let retryAction = PanelProblemRemediationAction(
            actionID: .retry,
            title: "Retry",
            symbolName: "arrow.clockwise",
            role: signal.kind == .rateLimitExceeded ? .primary : .secondary,
            isEnabled: !retryInFlight,
            disabledReason: retryInFlight ? "Retry is already in progress for \(section.title)." : nil
        )
        let openConfigurationAction = PanelProblemRemediationAction(
            actionID: .openConfiguration,
            title: "Open Configuration",
            symbolName: "gearshape",
            role: signal.kind == .authScope ? .primary : .secondary
        )
        let actions = [
            openConfigurationAction,
            retryAction,
            PanelProblemRemediationAction(
                actionID: .openInspect,
                title: "Open Inspect",
                symbolName: "doc.text.magnifyingglass",
                role: .secondary
            )
        ]
        return PanelProblemRemediationContext(
            section: section,
            kind: signal.kind,
            detail: signal.detail,
            actions: actions
        )
    }

    func clearSignal(for section: AppSection) {
        signalsBySection.removeValue(forKey: section)
    }

    @discardableResult
    func typedRemediationMessage(
        daemonError: DaemonAPIError,
        section: AppSection,
        sectionTitle: String
    ) -> String? {
        guard let kind = problemKind(for: daemonError) else {
            return nil
        }
        let detail = nonEmpty(daemonError.serverDetails?.remediation?.hint)
            ?? nonEmpty(daemonError.serverDetails?.remediation?.label)
            ?? nonEmpty(daemonError.serverMessage)
            ?? "\(sectionTitle) request failed."
        signalsBySection[section] = Signal(
            kind: kind,
            detail: detail,
            correlationID: daemonError.serverCorrelationID
        )

        switch kind {
        case .authScope:
            return "Additional token scope is required for \(sectionTitle). Open Configuration, update Assistant Access Token permissions, then retry."
        case .rateLimitExceeded:
            return "Requests for \(sectionTitle) are temporarily rate limited. Wait a moment, then retry or inspect diagnostics."
        }
    }

    private func problemKind(for daemonError: DaemonAPIError) -> PanelProblemRemediationContext.Kind? {
        let code = daemonError.serverCode?.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        switch code {
        case "auth_scope":
            return .authScope
        case "rate_limit_exceeded":
            return .rateLimitExceeded
        default:
            if daemonError.serverStatusCode == 429 {
                return .rateLimitExceeded
            }
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
