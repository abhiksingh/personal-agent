import XCTest
@testable import PersonalAgentUI

@MainActor
final class AppShellStatePanelErrorMappingTests: XCTestCase {
    func testModelsPanelServiceNotConfiguredUsesRemediationCopy() {
        let state = AppShellState()
        let error = DaemonAPIError.serverProblem(
            statusCode: 501,
            message: "provider service is not configured",
            code: "service_not_configured",
            details: DaemonProblemDetails(
                category: "service_not_configured",
                domain: "providers",
                service: DaemonProblemService(
                    id: "provider",
                    label: "Provider service",
                    configField: "Providers"
                ),
                remediation: DaemonProblemRemediation(
                    action: "configure_server_service",
                    label: "Configure Service Dependency",
                    hint: "Set ServerConfig.Providers."
                )
            ),
            correlationID: "corr-models-problem"
        )

        let message = state.panelErrorMessageForTesting(error, panelContext: .models)

        XCTAssertEqual(
            message,
            "Provider service is not configured yet. Open Configuration, configure service dependency, then refresh Models."
        )
    }

    func testTasksPanelUnknownRouteShowsVersionMismatchGuidance() {
        let state = AppShellState()
        let error = DaemonAPIError.serverProblem(
            statusCode: 404,
            message: "unknown control route GET /v1/tasks/list",
            code: "resource_not_found",
            details: nil,
            correlationID: "corr-unknown-route"
        )

        let message = state.panelErrorMessageForTesting(error, panelContext: .tasks)

        XCTAssertEqual(
            message,
            "Tasks is unavailable because this app build is behind daemon API changes. Update app/daemon and refresh."
        )
    }

    func testChannelsPanelDecodingErrorSuppressesRawPayloadDetails() {
        let state = AppShellState()
        let error = DaemonAPIError.decoding("keyNotFound(CodingKeys(stringValue: \\\"channels\\\"))")

        let message = state.panelErrorMessageForTesting(error, panelContext: .channels)

        XCTAssertEqual(
            message,
            "Received an unexpected daemon response while loading Channels. Refresh and try again."
        )
        XCTAssertFalse(message.localizedCaseInsensitiveContains("keyNotFound"))
    }

    func testChatPanelMissingRouteUsesSharedGuidance() {
        let state = AppShellState()
        let error = DaemonAPIError.server(
            statusCode: 400,
            message: "no enabled models with ready provider configuration for workspace \"ws1\""
        )

        let message = state.panelErrorMessageForTesting(error, panelContext: .chat)

        XCTAssertEqual(
            message,
            "No enabled chat model is ready. Open Models, configure a provider, enable a model, and save a chat route policy."
        )
    }

    func testAuthScopeProblemBuildsSharedPanelRemediationContext() {
        let state = AppShellState()
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
                    label: "Update Assistant Access Token",
                    hint: "Token must include control.read and control.write scopes."
                )
            ),
            correlationID: "corr-auth-scope"
        )

        let message = state.panelErrorMessageForTesting(error, panelContext: .channels)
        XCTAssertEqual(
            message,
            "Additional token scope is required for Channels. Open Configuration, update Assistant Access Token permissions, then retry."
        )

        let remediation = state.panelProblemRemediation(for: .channels)
        XCTAssertNotNil(remediation)
        XCTAssertEqual(remediation?.kind, .authScope)
        XCTAssertEqual(remediation?.actions.map(\.actionID), [.openConfiguration, .retry, .openInspect])
        XCTAssertEqual(remediation?.actions.first?.role, .primary)
        XCTAssertEqual(remediation?.actions.first?.title, "Open Configuration")
        XCTAssertEqual(
            remediation?.detail,
            "Token must include control.read and control.write scopes."
        )
    }

    func testRateLimitProblemBuildsRetryDisabledReasonWhenPanelAlreadyLoading() {
        let state = AppShellState()
        state.isTasksLoading = true
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

        let message = state.panelErrorMessageForTesting(error, panelContext: .tasks)
        XCTAssertEqual(
            message,
            "Requests for Tasks are temporarily rate limited. Wait a moment, then retry or inspect diagnostics."
        )

        guard let remediation = state.panelProblemRemediation(for: .tasks) else {
            return XCTFail("Expected shared panel remediation context for tasks rate-limit error.")
        }
        XCTAssertEqual(remediation.kind, .rateLimitExceeded)
        XCTAssertEqual(remediation.actions.map(\.actionID), [.openConfiguration, .retry, .openInspect])

        let retry = remediation.actions.first { $0.actionID == .retry }
        XCTAssertEqual(retry?.role, .primary)
        XCTAssertEqual(retry?.isEnabled, false)
        XCTAssertEqual(retry?.disabledReason, "Retry is already in progress for Tasks.")
    }

    func testPanelProblemRemediationActionsNavigateToConfigurationAndInspect() {
        let state = AppShellState()
        let error = DaemonAPIError.serverProblem(
            statusCode: 403,
            message: "token scope is missing",
            code: "auth_scope",
            details: nil,
            correlationID: "corr-actions"
        )

        _ = state.panelErrorMessageForTesting(error, panelContext: .approvals)

        state.selectedSection = .approvals
        state.performPanelProblemRemediationAction(.openConfiguration, section: .approvals)
        XCTAssertEqual(state.selectedSection, .configuration)

        state.performPanelProblemRemediationAction(.openInspect, section: .approvals)
        XCTAssertEqual(state.selectedSection, .inspect)
        XCTAssertEqual(
            state.inspectStatusMessage,
            "Opened Inspect for Approvals remediation."
        )
    }
}
