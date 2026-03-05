import XCTest
@testable import PersonalAgentUI

final class DaemonAPIErrorTests: XCTestCase {
    func testServerErrorDescriptionIncludesStatusCodeAndMessage() {
        let error = DaemonAPIError.server(statusCode: 400, message: "invalid provider payload")
        XCTAssertEqual(error.errorDescription, "Daemon request failed (400): invalid provider payload")
    }

    func testMissingAuthTokenDescriptionIsStable() {
        let error = DaemonAPIError.missingAuthToken
        XCTAssertEqual(error.errorDescription, "Assistant access token is not configured.")
    }

    func testErrorClassificationFlagsAreDeterministic() {
        XCTAssertTrue(DaemonAPIError.server(statusCode: 401, message: "unauthorized").isUnauthorized)
        XCTAssertFalse(DaemonAPIError.server(statusCode: 400, message: "bad request").isUnauthorized)
        XCTAssertTrue(DaemonAPIError.transport("offline").isConnectivityIssue)
        XCTAssertFalse(DaemonAPIError.decoding("bad payload").isConnectivityIssue)
        XCTAssertTrue(
            DaemonAPIError.server(
                statusCode: 400,
                message: "no enabled models with ready provider configuration for workspace \"ws1\""
            ).isMissingReadyChatModelRoute
        )
        XCTAssertFalse(DaemonAPIError.server(statusCode: 400, message: "invalid payload").isMissingReadyChatModelRoute)
    }

    func testTypedServerProblemExposesCodeDetailsAndClassification() {
        let error = DaemonAPIError.serverProblem(
            statusCode: 501,
            message: "provider service is not configured",
            code: "service_not_configured",
            details: DaemonProblemDetails(
                category: "service_not_configured",
                domain: "providers",
                service: DaemonProblemService(
                    id: "provider",
                    label: "provider service",
                    configField: "Providers"
                ),
                remediation: DaemonProblemRemediation(
                    action: "configure_server_service",
                    label: "Configure Service Dependency",
                    hint: "Set ServerConfig.Providers before calling this endpoint."
                )
            ),
            correlationID: "corr-problem-1"
        )

        XCTAssertEqual(
            error.errorDescription,
            "Daemon request failed (501): provider service is not configured"
        )
        XCTAssertEqual(error.serverStatusCode, 501)
        XCTAssertEqual(error.serverCode, "service_not_configured")
        XCTAssertEqual(error.serverDetails?.domain, "providers")
        XCTAssertEqual(error.serverDetails?.service?.configField, "Providers")
        XCTAssertEqual(error.serverCorrelationID, "corr-problem-1")
        XCTAssertFalse(error.isUnauthorized)
    }
}
