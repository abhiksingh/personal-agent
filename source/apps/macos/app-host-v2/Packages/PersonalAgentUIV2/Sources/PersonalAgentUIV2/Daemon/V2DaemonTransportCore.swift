import Foundation

struct V2DaemonTransportCore {
    private let session: URLSession
    private let encoder: JSONEncoder
    private let decoder: JSONDecoder

    init(session: URLSession = .shared) {
        self.session = session
        self.encoder = JSONEncoder()
        self.decoder = JSONDecoder()
    }

    func request<Response: Decodable, RequestBody: Encodable>(
        baseURL: URL,
        path: String,
        method: String,
        authToken: String,
        correlationID: String? = nil,
        timeoutInterval: TimeInterval = 10,
        body: RequestBody?
    ) async throws -> Response {
        let trimmedToken = authToken.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmedToken.isEmpty else {
            throw V2DaemonAPIError.missingAuthToken
        }

        guard let url = URL(string: path, relativeTo: baseURL)?.absoluteURL else {
            throw V2DaemonAPIError.invalidBaseURL
        }

        var request = URLRequest(url: url)
        request.httpMethod = method
        request.timeoutInterval = max(1, timeoutInterval)
        request.setValue("Bearer \(trimmedToken)", forHTTPHeaderField: "Authorization")
        request.setValue("application/json", forHTTPHeaderField: "Accept")

        if let correlationID, !correlationID.isEmpty {
            request.setValue(correlationID, forHTTPHeaderField: "X-Correlation-ID")
        }

        if let body {
            request.setValue("application/json", forHTTPHeaderField: "Content-Type")
            request.httpBody = try encoder.encode(body)
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
                throw V2DaemonAPIError.transport(urlError.localizedDescription)
            }
            throw V2DaemonAPIError.transport(error.localizedDescription)
        }

        guard let httpResponse = response as? HTTPURLResponse else {
            throw V2DaemonAPIError.invalidResponse
        }

        if !(200...299).contains(httpResponse.statusCode) {
            if let serverError = try? decoder.decode(V2DaemonErrorResponse.self, from: data) {
                if let code = serverError.code {
                    throw V2DaemonAPIError.serverProblem(
                        statusCode: httpResponse.statusCode,
                        message: serverError.message,
                        code: code,
                        details: serverError.details,
                        correlationID: serverError.correlationID
                    )
                }
                throw V2DaemonAPIError.server(statusCode: httpResponse.statusCode, message: serverError.message)
            }
            throw V2DaemonAPIError.server(
                statusCode: httpResponse.statusCode,
                message: HTTPURLResponse.localizedString(forStatusCode: httpResponse.statusCode)
            )
        }

        do {
            return try decoder.decode(Response.self, from: data)
        } catch {
            throw V2DaemonAPIError.decoding(error.localizedDescription)
        }
    }

    func connectRealtime(
        baseURL: URL,
        authToken: String,
        correlationID: String? = nil
    ) throws -> V2DaemonRealtimeSession {
        let trimmedToken = authToken.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmedToken.isEmpty else {
            throw V2DaemonAPIError.missingAuthToken
        }

        let url = try realtimeURL(from: baseURL)
        var request = URLRequest(url: url)
        request.timeoutInterval = 10
        request.setValue("Bearer \(trimmedToken)", forHTTPHeaderField: "Authorization")
        if let correlationID, !correlationID.isEmpty {
            request.setValue(correlationID, forHTTPHeaderField: "X-Correlation-ID")
        }
        return V2DaemonRealtimeSession(task: session.webSocketTask(with: request))
    }

    private func realtimeURL(from baseURL: URL) throws -> URL {
        guard var components = URLComponents(url: baseURL, resolvingAgainstBaseURL: false) else {
            throw V2DaemonAPIError.invalidBaseURL
        }

        switch components.scheme?.lowercased() {
        case "http":
            components.scheme = "ws"
        case "https":
            components.scheme = "wss"
        case "ws", "wss":
            break
        default:
            throw V2DaemonAPIError.transport("Unsupported daemon URL scheme for realtime stream.")
        }

        let trimmedPath = components.path.trimmingCharacters(in: CharacterSet(charactersIn: "/"))
        components.path = trimmedPath.isEmpty ? "/v1/realtime/ws" : "/\(trimmedPath)/v1/realtime/ws"
        guard let url = components.url else {
            throw V2DaemonAPIError.invalidBaseURL
        }
        return url
    }
}
