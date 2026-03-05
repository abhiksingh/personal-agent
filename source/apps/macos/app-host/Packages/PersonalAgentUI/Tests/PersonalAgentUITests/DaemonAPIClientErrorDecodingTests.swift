import XCTest
@testable import PersonalAgentUI

private final class DaemonAPIClientMockURLProtocol: URLProtocol {
    nonisolated(unsafe) static var handler: ((URLRequest) throws -> (HTTPURLResponse, Data))?

    override class func canInit(with request: URLRequest) -> Bool {
        true
    }

    override class func canonicalRequest(for request: URLRequest) -> URLRequest {
        request
    }

    override func startLoading() {
        guard let handler = Self.handler else {
            client?.urlProtocol(
                self,
                didFailWithError: URLError(.badServerResponse)
            )
            return
        }

        do {
            let (response, data) = try handler(request)
            client?.urlProtocol(self, didReceive: response, cacheStoragePolicy: .notAllowed)
            client?.urlProtocol(self, didLoad: data)
            client?.urlProtocolDidFinishLoading(self)
        } catch {
            client?.urlProtocol(self, didFailWithError: error)
        }
    }

    override func stopLoading() {}
}

final class DaemonAPIClientErrorDecodingTests: XCTestCase {
    override func tearDown() {
        super.tearDown()
        DaemonAPIClientMockURLProtocol.handler = nil
    }

    func testRequestDecodesTypedProblemDetailsAsServerProblem() async {
        DaemonAPIClientMockURLProtocol.handler = { request in
            XCTAssertEqual(request.httpMethod, "GET")
            XCTAssertEqual(request.url?.path, "/v1/capabilities/smoke")
            let body = """
            {
              "error": {
                "code": "service_not_configured",
                "message": "provider service is not configured",
                "details": {
                  "category": "service_not_configured",
                  "domain": "providers",
                  "service": {
                    "id": "provider",
                    "label": "provider service",
                    "config_field": "Providers"
                  },
                  "remediation": {
                    "action": "configure_server_service",
                    "label": "Configure Service Dependency",
                    "hint": "Set ServerConfig.Providers with a non-nil implementation before calling this endpoint."
                  }
                }
              },
              "correlation_id": "corr-problem-typed",
              "type": "https://personalagent.dev/problems/service_not_configured",
              "title": "Not Implemented",
              "status": 501,
              "detail": "provider service is not configured",
              "instance": "/v1/errors/corr-problem-typed"
            }
            """
            let response = HTTPURLResponse(
                url: request.url!,
                statusCode: 501,
                httpVersion: nil,
                headerFields: ["Content-Type": "application/problem+json"]
            )!
            return (response, Data(body.utf8))
        }

        let client = makeClient()
        do {
            _ = try await client.capabilitySmoke(
                baseURL: URL(string: "http://unit.test")!,
                authToken: "test-token"
            )
            XCTFail("expected typed problem error")
        } catch let error as DaemonAPIError {
            switch error {
            case .serverProblem(let statusCode, let message, let code, let details, let correlationID):
                XCTAssertEqual(statusCode, 501)
                XCTAssertEqual(message, "provider service is not configured")
                XCTAssertEqual(code, "service_not_configured")
                XCTAssertEqual(details?.domain, "providers")
                XCTAssertEqual(details?.service?.id, "provider")
                XCTAssertEqual(details?.service?.configField, "Providers")
                XCTAssertEqual(details?.remediation?.action, "configure_server_service")
                XCTAssertEqual(correlationID, "corr-problem-typed")
            default:
                XCTFail("expected .serverProblem, got \(error)")
            }
        } catch {
            XCTFail("expected DaemonAPIError, got \(error)")
        }
    }

    func testRequestFallsBackToProblemDetailWhenErrorObjectMissing() async {
        DaemonAPIClientMockURLProtocol.handler = { request in
            let body = #"{"detail":"legacy unauthorized"}"#
            let response = HTTPURLResponse(
                url: request.url!,
                statusCode: 401,
                httpVersion: nil,
                headerFields: ["Content-Type": "application/json"]
            )!
            return (response, Data(body.utf8))
        }

        let client = makeClient()
        do {
            _ = try await client.capabilitySmoke(
                baseURL: URL(string: "http://unit.test")!,
                authToken: "test-token"
            )
            XCTFail("expected server error")
        } catch let error as DaemonAPIError {
            switch error {
            case .server(let statusCode, let message):
                XCTAssertEqual(statusCode, 401)
                XCTAssertEqual(message, "legacy unauthorized")
            default:
                XCTFail("expected .server for detail fallback payload, got \(error)")
            }
        } catch {
            XCTFail("expected DaemonAPIError, got \(error)")
        }
    }

    func testRequestDerivesProblemCodeFromTypeWhenCodeIsMissing() async {
        DaemonAPIClientMockURLProtocol.handler = { request in
            let body = """
            {
              "error": {
                "message": "unknown control route GET /v1/unknown"
              },
              "type": "https://personalagent.dev/problems/resource_not_found",
              "status": 404,
              "detail": "unknown control route GET /v1/unknown",
              "correlation_id": "corr-unknown-route"
            }
            """
            let response = HTTPURLResponse(
                url: request.url!,
                statusCode: 404,
                httpVersion: nil,
                headerFields: ["Content-Type": "application/problem+json"]
            )!
            return (response, Data(body.utf8))
        }

        let client = makeClient()
        do {
            _ = try await client.capabilitySmoke(
                baseURL: URL(string: "http://unit.test")!,
                authToken: "test-token"
            )
            XCTFail("expected unknown-route error")
        } catch let error as DaemonAPIError {
            switch error {
            case .serverProblem(let statusCode, let message, let code, _, let correlationID):
                XCTAssertEqual(statusCode, 404)
                XCTAssertEqual(message, "unknown control route GET /v1/unknown")
                XCTAssertEqual(code, "resource_not_found")
                XCTAssertEqual(correlationID, "corr-unknown-route")
            default:
                XCTFail("expected .serverProblem from problem type, got \(error)")
            }
        } catch {
            XCTFail("expected DaemonAPIError, got \(error)")
        }
    }

    private func makeClient() -> DaemonAPIClient {
        let configuration = URLSessionConfiguration.ephemeral
        configuration.protocolClasses = [DaemonAPIClientMockURLProtocol.self]
        let session = URLSession(configuration: configuration)
        return DaemonAPIClient(session: session, fixtureScenario: nil)
    }
}
