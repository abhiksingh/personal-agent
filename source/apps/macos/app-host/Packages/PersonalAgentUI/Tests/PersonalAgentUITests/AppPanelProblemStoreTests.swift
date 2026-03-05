import XCTest
@testable import PersonalAgentUI

@MainActor
final class AppPanelProblemStoreTests: XCTestCase {
    func testTypedAuthScopeMessageStoresRemediationContext() {
        let store = AppPanelProblemStore()
        let error = DaemonAPIError.serverProblem(
            statusCode: 403,
            message: "token scope is missing",
            code: "auth_scope",
            details: DaemonProblemDetails(
                category: "auth_scope",
                domain: "auth",
                service: nil,
                remediation: DaemonProblemRemediation(
                    action: "update_token",
                    label: "Update token",
                    hint: "Token must include control.read and control.write scopes."
                )
            ),
            correlationID: "corr-auth-scope"
        )

        let message = store.typedRemediationMessage(
            daemonError: error,
            section: .channels,
            sectionTitle: "Channels"
        )

        XCTAssertEqual(
            message,
            "Additional token scope is required for Channels. Open Configuration, update Assistant Access Token permissions, then retry."
        )

        guard let remediation = store.remediationContext(for: .channels, retryInFlight: false) else {
            return XCTFail("Expected remediation context for auth-scope problem.")
        }
        XCTAssertEqual(remediation.kind, .authScope)
        XCTAssertEqual(remediation.actions.map(\.actionID), [.openConfiguration, .retry, .openInspect])
        XCTAssertEqual(remediation.actions.first?.role, .primary)
        XCTAssertEqual(remediation.actions.first?.title, "Open Configuration")
        XCTAssertEqual(
            remediation.detail,
            "Token must include control.read and control.write scopes."
        )
    }

    func testTypedRateLimitMessageMakesRetryPrimaryWithDisabledReasonWhenInFlight() {
        let store = AppPanelProblemStore()
        let error = DaemonAPIError.serverProblem(
            statusCode: 429,
            message: "too many requests",
            code: "rate_limit_exceeded",
            details: DaemonProblemDetails(
                category: "rate_limit_exceeded",
                domain: "limits",
                service: nil,
                remediation: DaemonProblemRemediation(
                    action: "retry_later",
                    label: "Retry later",
                    hint: "Wait 30 seconds before retry."
                )
            ),
            correlationID: "corr-rate-limit"
        )

        let message = store.typedRemediationMessage(
            daemonError: error,
            section: .tasks,
            sectionTitle: "Tasks"
        )
        XCTAssertEqual(
            message,
            "Requests for Tasks are temporarily rate limited. Wait a moment, then retry or inspect diagnostics."
        )

        guard let remediation = store.remediationContext(for: .tasks, retryInFlight: true) else {
            return XCTFail("Expected remediation context for rate-limit problem.")
        }
        XCTAssertEqual(remediation.kind, .rateLimitExceeded)
        XCTAssertEqual(remediation.actions.map(\.actionID), [.openConfiguration, .retry, .openInspect])

        let retry = remediation.actions.first { $0.actionID == .retry }
        XCTAssertEqual(retry?.role, .primary)
        XCTAssertEqual(retry?.isEnabled, false)
        XCTAssertEqual(retry?.disabledReason, "Retry is already in progress for Tasks.")
    }

    func testTypedRateLimitMessageFallsBackToStatusCodeWithoutServerCode() {
        let store = AppPanelProblemStore()
        let error = DaemonAPIError.serverProblem(
            statusCode: 429,
            message: "too many requests",
            code: "",
            details: nil,
            correlationID: "corr-rate-limit-status"
        )

        let message = store.typedRemediationMessage(
            daemonError: error,
            section: .automation,
            sectionTitle: "Automation"
        )

        XCTAssertEqual(
            message,
            "Requests for Automation are temporarily rate limited. Wait a moment, then retry or inspect diagnostics."
        )
        XCTAssertNotNil(store.remediationContext(for: .automation, retryInFlight: false))
    }

    func testUnsupportedProblemCodeDoesNotStoreSignal() {
        let store = AppPanelProblemStore()
        let error = DaemonAPIError.serverProblem(
            statusCode: 503,
            message: "service not configured",
            code: "service_not_configured",
            details: nil,
            correlationID: "corr-service"
        )

        let message = store.typedRemediationMessage(
            daemonError: error,
            section: .models,
            sectionTitle: "Models"
        )

        XCTAssertNil(message)
        XCTAssertNil(store.remediationContext(for: .models, retryInFlight: false))
    }

    func testClearSignalRemovesRemediationContext() {
        let store = AppPanelProblemStore()
        let error = DaemonAPIError.serverProblem(
            statusCode: 403,
            message: "token scope is missing",
            code: "auth_scope",
            details: nil,
            correlationID: "corr-clear"
        )

        _ = store.typedRemediationMessage(
            daemonError: error,
            section: .approvals,
            sectionTitle: "Approvals"
        )
        XCTAssertNotNil(store.remediationContext(for: .approvals, retryInFlight: false))

        store.clearSignal(for: .approvals)

        XCTAssertNil(store.remediationContext(for: .approvals, retryInFlight: false))
    }
}
