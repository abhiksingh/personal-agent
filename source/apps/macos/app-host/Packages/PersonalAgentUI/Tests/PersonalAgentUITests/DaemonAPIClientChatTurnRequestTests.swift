import XCTest
@testable import PersonalAgentUI

private final class DaemonAPIClientChatTurnMockURLProtocol: URLProtocol {
    nonisolated(unsafe) static var handler: ((URLRequest) throws -> (HTTPURLResponse, Data))?

    override class func canInit(with request: URLRequest) -> Bool {
        true
    }

    override class func canonicalRequest(for request: URLRequest) -> URLRequest {
        request
    }

    override func startLoading() {
        guard let handler = Self.handler else {
            client?.urlProtocol(self, didFailWithError: URLError(.badServerResponse))
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

final class DaemonAPIClientChatTurnRequestTests: XCTestCase {
    override func tearDown() {
        super.tearDown()
        DaemonAPIClientChatTurnMockURLProtocol.handler = nil
    }

    func testChatTurnRequestOmitsModeOverrideSystemPrompt() async throws {
        var capturedBody: [String: Any] = [:]
        DaemonAPIClientChatTurnMockURLProtocol.handler = { request in
            XCTAssertEqual(request.httpMethod, "POST")
            XCTAssertEqual(request.url?.path, "/v1/chat/turn")
            let bodyData = try Self.requestBodyData(from: request)
            capturedBody = try XCTUnwrap(
                JSONSerialization.jsonObject(with: bodyData) as? [String: Any]
            )
            let response = HTTPURLResponse(
                url: try XCTUnwrap(request.url),
                statusCode: 200,
                httpVersion: nil,
                headerFields: ["Content-Type": "application/json"]
            )!
            return (response, Data("{}".utf8))
        }

        let client = makeClient()
        _ = try await client.chatTurn(
            baseURL: URL(string: "http://unit.test")!,
            authToken: "test-token",
            workspaceID: "ws1",
            messages: [(role: "user", content: "send status update")],
            correlationID: "corr-chat-request"
        )

        XCTAssertNil(capturedBody["submission_mode"])
        XCTAssertNil(capturedBody["system_prompt"])
    }

    func testChatTurnRequestUsesProvidedSystemPromptWithoutModePrefix() async throws {
        var capturedBody: [String: Any] = [:]
        DaemonAPIClientChatTurnMockURLProtocol.handler = { request in
            let bodyData = try Self.requestBodyData(from: request)
            capturedBody = try XCTUnwrap(
                JSONSerialization.jsonObject(with: bodyData) as? [String: Any]
            )
            let response = HTTPURLResponse(
                url: try XCTUnwrap(request.url),
                statusCode: 200,
                httpVersion: nil,
                headerFields: ["Content-Type": "application/json"]
            )!
            return (response, Data("{}".utf8))
        }

        let client = makeClient()
        _ = try await client.chatTurn(
            baseURL: URL(string: "http://unit.test")!,
            authToken: "test-token",
            workspaceID: "ws1",
            messages: [
                (role: "assistant", content: "Prior answer"),
                (role: "user", content: "Draft and send the reply")
            ],
            systemPrompt: "  Keep responses concise and outcome-first.  "
        )

        XCTAssertEqual(
            capturedBody["system_prompt"] as? String,
            "Keep responses concise and outcome-first."
        )
        let items = try XCTUnwrap(capturedBody["items"] as? [[String: Any]])
        XCTAssertEqual(items.count, 2)
        XCTAssertEqual(items[0]["type"] as? String, "assistant_message")
        XCTAssertEqual(items[1]["type"] as? String, "user_message")
    }

    func testChatTurnRequestPreservesMessageWhitespaceAndDropsWhitespaceOnlyItems() async throws {
        var capturedBody: [String: Any] = [:]
        DaemonAPIClientChatTurnMockURLProtocol.handler = { request in
            let bodyData = try Self.requestBodyData(from: request)
            capturedBody = try XCTUnwrap(
                JSONSerialization.jsonObject(with: bodyData) as? [String: Any]
            )
            let response = HTTPURLResponse(
                url: try XCTUnwrap(request.url),
                statusCode: 200,
                httpVersion: nil,
                headerFields: ["Content-Type": "application/json"]
            )!
            return (response, Data("{}".utf8))
        }

        let client = makeClient()
        _ = try await client.chatTurn(
            baseURL: URL(string: "http://unit.test")!,
            authToken: "test-token",
            workspaceID: "ws1",
            messages: [
                (role: "user", content: "   \n\t "),
                (role: "assistant", content: "  Prior response with leading space.\n"),
                (role: "user", content: "\nLine one\nLine two\n")
            ]
        )

        let items = try XCTUnwrap(capturedBody["items"] as? [[String: Any]])
        XCTAssertEqual(items.count, 2)
        XCTAssertEqual(items[0]["type"] as? String, "assistant_message")
        XCTAssertEqual(items[0]["content"] as? String, "  Prior response with leading space.\n")
        XCTAssertEqual(items[1]["type"] as? String, "user_message")
        XCTAssertEqual(items[1]["content"] as? String, "\nLine one\nLine two\n")
    }

    func testChatTurnRequestUsesExtendedTimeoutForLongRunningTurns() async throws {
        var capturedTimeout: TimeInterval?
        DaemonAPIClientChatTurnMockURLProtocol.handler = { request in
            capturedTimeout = request.timeoutInterval
            let response = HTTPURLResponse(
                url: try XCTUnwrap(request.url),
                statusCode: 200,
                httpVersion: nil,
                headerFields: ["Content-Type": "application/json"]
            )!
            return (response, Data("{}".utf8))
        }

        let client = makeClient()
        _ = try await client.chatTurn(
            baseURL: URL(string: "http://unit.test")!,
            authToken: "test-token",
            workspaceID: "ws1",
            messages: [(role: "user", content: "Long answer please.")]
        )

        let timeout = try XCTUnwrap(capturedTimeout)
        XCTAssertEqual(timeout, 300, accuracy: 0.01)
    }

    func testChatTurnExplainRequestUsesCanonicalChatExplainPayload() async throws {
        var capturedBody: [String: Any] = [:]
        DaemonAPIClientChatTurnMockURLProtocol.handler = { request in
            XCTAssertEqual(request.httpMethod, "POST")
            XCTAssertEqual(request.url?.path, "/v1/chat/turn/explain")
            let bodyData = try Self.requestBodyData(from: request)
            capturedBody = try XCTUnwrap(
                JSONSerialization.jsonObject(with: bodyData) as? [String: Any]
            )
            let response = HTTPURLResponse(
                url: try XCTUnwrap(request.url),
                statusCode: 200,
                httpVersion: nil,
                headerFields: ["Content-Type": "application/json"]
            )!
            return (response, Data("{}".utf8))
        }

        let client = makeClient()
        _ = try await client.chatTurnExplain(
            baseURL: URL(string: "http://unit.test")!,
            authToken: "test-token",
            workspaceID: "ws1",
            requestedByActorID: "actor.requester",
            subjectActorID: "actor.subject",
            actingAsActorID: "actor.delegate"
        )

        XCTAssertEqual(capturedBody["workspace_id"] as? String, "ws1")
        XCTAssertEqual(capturedBody["task_class"] as? String, "chat")
        XCTAssertEqual(capturedBody["requested_by_actor_id"] as? String, "actor.requester")
        XCTAssertEqual(capturedBody["subject_actor_id"] as? String, "actor.subject")
        XCTAssertEqual(capturedBody["acting_as_actor_id"] as? String, "actor.delegate")
        let channel = try XCTUnwrap(capturedBody["channel"] as? [String: Any])
        XCTAssertEqual(channel["channel_id"] as? String, "app")
    }

    private func makeClient() -> DaemonAPIClient {
        let configuration = URLSessionConfiguration.ephemeral
        configuration.protocolClasses = [DaemonAPIClientChatTurnMockURLProtocol.self]
        let session = URLSession(configuration: configuration)
        return DaemonAPIClient(session: session, fixtureScenario: nil)
    }

    private static func requestBodyData(from request: URLRequest) throws -> Data {
        if let body = request.httpBody {
            return body
        }
        guard let stream = request.httpBodyStream else {
            throw URLError(.badServerResponse)
        }
        stream.open()
        defer { stream.close() }
        var data = Data()
        var buffer = [UInt8](repeating: 0, count: 4096)
        while stream.hasBytesAvailable {
            let count = stream.read(&buffer, maxLength: buffer.count)
            if count < 0 {
                throw stream.streamError ?? URLError(.cannotDecodeRawData)
            }
            if count == 0 {
                break
            }
            data.append(buffer, count: count)
        }
        return data
    }
}
