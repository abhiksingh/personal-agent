import Foundation

struct DaemonAPITransportCore {
    private let session: URLSession
    private let encoder: JSONEncoder
    private let decoder: JSONDecoder
    private let fixtureScenario: DaemonAPISmokeFixtureScenario?

    init(
        session: URLSession = .shared,
        fixtureScenario: DaemonAPISmokeFixtureScenario? = DaemonAPISmokeFixtureScenario.fromEnvironment()
    ) {
        self.session = session
        self.encoder = JSONEncoder()
        self.decoder = JSONDecoder()
        self.fixtureScenario = fixtureScenario
    }

    func request<Response: Decodable, RequestBody: Encodable>(
        baseURL: URL,
        path: String,
        method: String,
        authToken: String,
        correlationID: String? = nil,
        timeoutInterval: TimeInterval = 8,
        body: RequestBody?
    ) async throws -> Response {
        let trimmedToken = authToken.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmedToken.isEmpty else {
            throw DaemonAPIError.missingAuthToken
        }

        guard let url = URL(string: path, relativeTo: baseURL)?.absoluteURL else {
            throw DaemonAPIError.transport("Invalid daemon endpoint URL.")
        }
        var request = URLRequest(url: url)
        request.httpMethod = method
        request.timeoutInterval = max(1, timeoutInterval)
        request.setValue("Bearer \(trimmedToken)", forHTTPHeaderField: "Authorization")
        request.setValue("application/json", forHTTPHeaderField: "Accept")
        if let correlationID, !correlationID.isEmpty {
            request.setValue(correlationID, forHTTPHeaderField: "X-Correlation-ID")
        }

        var encodedBodyData: Data?
        if let body {
            request.setValue("application/json", forHTTPHeaderField: "Content-Type")
            let encodedBody = try encoder.encode(body)
            request.httpBody = encodedBody
            encodedBodyData = encodedBody
        }

        if let fixtureScenario {
            let fixtureData = try DaemonAPISmokeFixture.responseData(
                method: method,
                path: path,
                body: encodedBodyData,
                scenario: fixtureScenario
            )
            do {
                return try decoder.decode(Response.self, from: fixtureData)
            } catch {
                throw DaemonAPIError.decoding(error.localizedDescription)
            }
        }

        let data: Data
        let response: URLResponse
        do {
            (data, response) = try await session.data(for: request)
        } catch is CancellationError {
            throw CancellationError()
        } catch {
            if let urlError = error as? URLError {
                if urlError.code == .cancelled {
                    throw CancellationError()
                }
                throw DaemonAPIError.transport(urlError.localizedDescription)
            }
            throw DaemonAPIError.transport(error.localizedDescription)
        }

        guard let httpResponse = response as? HTTPURLResponse else {
            throw DaemonAPIError.invalidResponse
        }

        if !(200...299).contains(httpResponse.statusCode) {
            if let serverError = try? decoder.decode(DaemonErrorResponse.self, from: data) {
                if let code = serverError.code {
                    throw DaemonAPIError.serverProblem(
                        statusCode: httpResponse.statusCode,
                        message: serverError.message,
                        code: code,
                        details: serverError.details,
                        correlationID: serverError.correlationID
                    )
                }
                throw DaemonAPIError.server(
                    statusCode: httpResponse.statusCode,
                    message: serverError.message
                )
            }
            let serverMessage = HTTPURLResponse.localizedString(forStatusCode: httpResponse.statusCode)
            throw DaemonAPIError.server(statusCode: httpResponse.statusCode, message: serverMessage)
        }

        do {
            return try decoder.decode(Response.self, from: data)
        } catch {
            throw DaemonAPIError.decoding(error.localizedDescription)
        }
    }

    func connectRealtime(
        baseURL: URL,
        authToken: String,
        correlationID: String? = nil
    ) throws -> DaemonRealtimeSession {
        let trimmedToken = authToken.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmedToken.isEmpty else {
            throw DaemonAPIError.missingAuthToken
        }

        let url = try realtimeURL(from: baseURL)
        var request = URLRequest(url: url)
        request.timeoutInterval = 8
        request.setValue("Bearer \(trimmedToken)", forHTTPHeaderField: "Authorization")
        if let correlationID, !correlationID.isEmpty {
            request.setValue(correlationID, forHTTPHeaderField: "X-Correlation-ID")
        }
        return DaemonRealtimeSession(task: session.webSocketTask(with: request))
    }

    func realtimeURL(from baseURL: URL) throws -> URL {
        guard var components = URLComponents(url: baseURL, resolvingAgainstBaseURL: false) else {
            throw DaemonAPIError.transport("Invalid daemon base URL.")
        }
        switch components.scheme?.lowercased() {
        case "http":
            components.scheme = "ws"
        case "https":
            components.scheme = "wss"
        case "ws", "wss":
            break
        default:
            throw DaemonAPIError.transport("Unsupported daemon URL scheme for realtime stream.")
        }

        let trimmedPath = components.path.trimmingCharacters(in: CharacterSet(charactersIn: "/"))
        if trimmedPath.isEmpty {
            components.path = "/v1/realtime/ws"
        } else {
            components.path = "/\(trimmedPath)/v1/realtime/ws"
        }
        guard let url = components.url else {
            throw DaemonAPIError.transport("Invalid daemon realtime URL.")
        }
        return url
    }
}
