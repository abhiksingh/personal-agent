import Foundation

public struct V2DaemonRealtimeEventPayload: Decodable, Sendable, Equatable {
    public let taskID: String?
    public let runID: String?
    public let approvalRequestID: String?
    public let status: String?
    public let state: String?
    public let channelID: String?
    public let connectorID: String?
    public let message: String?
    public let additional: [String: V2DaemonJSONValue]

    enum CodingKeys: String, CodingKey {
        case taskID = "task_id"
        case runID = "run_id"
        case approvalRequestID = "approval_request_id"
        case status
        case state
        case channelID = "channel_id"
        case connectorID = "connector_id"
        case message
    }

    public init(
        taskID: String? = nil,
        runID: String? = nil,
        approvalRequestID: String? = nil,
        status: String? = nil,
        state: String? = nil,
        channelID: String? = nil,
        connectorID: String? = nil,
        message: String? = nil,
        additional: [String: V2DaemonJSONValue] = [:]
    ) {
        self.taskID = taskID
        self.runID = runID
        self.approvalRequestID = approvalRequestID
        self.status = status
        self.state = state
        self.channelID = channelID
        self.connectorID = connectorID
        self.message = message
        self.additional = additional
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        taskID = try container.decodeLossyString(forKey: .taskID)
        runID = try container.decodeLossyString(forKey: .runID)
        approvalRequestID = try container.decodeLossyString(forKey: .approvalRequestID)
        status = try container.decodeLossyString(forKey: .status)
        state = try container.decodeLossyString(forKey: .state)
        channelID = try container.decodeLossyString(forKey: .channelID)
        connectorID = try container.decodeLossyString(forKey: .connectorID)
        message = try container.decodeLossyString(forKey: .message)

        var extras = v2DecodeDaemonJSONObject(from: decoder)
        for key in ["task_id", "run_id", "approval_request_id", "status", "state", "channel_id", "connector_id", "message"] {
            extras.removeValue(forKey: key)
        }
        additional = extras
    }
}

public struct V2DaemonRealtimeEventEnvelope: Decodable, Sendable, Equatable {
    public let eventID: String
    public let sequence: Int64
    public let eventType: String
    public let occurredAt: String
    public let correlationID: String?
    public let contractVersion: String?
    public let payload: V2DaemonRealtimeEventPayload

    enum CodingKeys: String, CodingKey {
        case eventID = "event_id"
        case sequence
        case eventType = "event_type"
        case occurredAt = "occurred_at"
        case correlationID = "correlation_id"
        case contractVersion = "contract_version"
        case payload
    }

    public init(
        eventID: String,
        sequence: Int64,
        eventType: String,
        occurredAt: String,
        correlationID: String? = nil,
        contractVersion: String? = nil,
        payload: V2DaemonRealtimeEventPayload = V2DaemonRealtimeEventPayload()
    ) {
        self.eventID = eventID
        self.sequence = sequence
        self.eventType = eventType
        self.occurredAt = occurredAt
        self.correlationID = correlationID
        self.contractVersion = contractVersion
        self.payload = payload
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        eventID = try container.decodeIfPresent(String.self, forKey: .eventID) ?? UUID().uuidString.lowercased()
        sequence = try container.decodeIfPresent(Int64.self, forKey: .sequence) ?? 0
        eventType = try container.decodeIfPresent(String.self, forKey: .eventType) ?? "unknown"
        occurredAt = try container.decodeIfPresent(String.self, forKey: .occurredAt) ?? ""
        correlationID = try container.decodeIfPresent(String.self, forKey: .correlationID)
        contractVersion = try container.decodeIfPresent(String.self, forKey: .contractVersion)
        payload = try container.decodeIfPresent(V2DaemonRealtimeEventPayload.self, forKey: .payload)
            ?? V2DaemonRealtimeEventPayload()
    }
}

public struct V2DaemonRealtimeClientSignal: Encodable, Sendable {
    public let signalType: String
    public let taskID: String?
    public let runID: String?
    public let reason: String?
    public let correlationID: String?

    public init(signalType: String, taskID: String? = nil, runID: String? = nil, reason: String? = nil, correlationID: String? = nil) {
        self.signalType = signalType
        self.taskID = taskID
        self.runID = runID
        self.reason = reason
        self.correlationID = correlationID
    }

    enum CodingKeys: String, CodingKey {
        case signalType = "signal_type"
        case taskID = "task_id"
        case runID = "run_id"
        case reason
        case correlationID = "correlation_id"
    }
}

public actor V2DaemonRealtimeSession {
    private let task: URLSessionWebSocketTask
    private let decoder = JSONDecoder()
    private let encoder = JSONEncoder()

    init(task: URLSessionWebSocketTask) {
        self.task = task
        self.task.resume()
    }

    public func receive() async throws -> V2DaemonRealtimeEventEnvelope {
        let message: URLSessionWebSocketTask.Message
        do {
            message = try await task.receive()
        } catch {
            throw classifyRealtimeTransportError(error, operation: "receive")
        }

        let data: Data
        switch message {
        case .string(let text):
            data = Data(text.utf8)
        case .data(let value):
            data = value
        @unknown default:
            throw V2DaemonAPIError.transport("Received unsupported realtime payload format.")
        }

        do {
            return try decoder.decode(V2DaemonRealtimeEventEnvelope.self, from: data)
        } catch {
            throw V2DaemonAPIError.decoding(error.localizedDescription)
        }
    }

    public func sendSignal(_ signal: V2DaemonRealtimeClientSignal) async throws {
        do {
            let data = try encoder.encode(signal)
            guard let text = String(data: data, encoding: .utf8) else {
                throw V2DaemonAPIError.transport("Failed to encode realtime signal payload.")
            }
            try await task.send(.string(text))
        } catch let daemonError as V2DaemonAPIError {
            throw daemonError
        } catch {
            throw classifyRealtimeTransportError(error, operation: "send")
        }
    }

    public func ping() async throws {
        do {
            try await withCheckedThrowingContinuation { (continuation: CheckedContinuation<Void, Error>) in
                task.sendPing { error in
                    if let error {
                        continuation.resume(throwing: error)
                    } else {
                        continuation.resume(returning: ())
                    }
                }
            }
        } catch let daemonError as V2DaemonAPIError {
            throw daemonError
        } catch {
            throw classifyRealtimeTransportError(error, operation: "ping")
        }
    }

    public func close() {
        task.cancel(with: .normalClosure, reason: nil)
    }

    private func classifyRealtimeTransportError(_ error: Error, operation: String) -> V2DaemonAPIError {
        if let daemonError = error as? V2DaemonAPIError {
            return daemonError
        }

        let nsError = error as NSError
        if let response = nsError.userInfo["NSErrorFailingURLResponseKey"] as? HTTPURLResponse {
            switch response.statusCode {
            case 401:
                return .server(statusCode: 401, message: "Realtime websocket authorization failed.")
            case 429:
                return .serverProblem(
                    statusCode: 429,
                    message: "Realtime websocket capacity exceeded.",
                    code: "rate_limit_exceeded",
                    details: V2DaemonProblemDetails(
                        category: "realtime_capacity",
                        domain: "realtime",
                        service: V2DaemonProblemService(id: "realtime.ws", label: "Realtime Stream", configField: nil),
                        remediation: V2DaemonProblemRemediation(action: "retry_later", label: "Retry Realtime Stream", hint: "Wait for active streams to close, then reconnect.")
                    ),
                    correlationID: nil
                )
            default:
                return .server(statusCode: response.statusCode, message: "Realtime websocket request failed.")
            }
        }

        let closeCode = task.closeCode
        let closeReason = task.closeReason.flatMap { String(data: $0, encoding: .utf8) }?
            .trimmingCharacters(in: .whitespacesAndNewlines)
        let normalizedCloseReason = closeReason?.lowercased() ?? ""

        if normalizedCloseReason.contains("stale") || normalizedCloseReason.contains("heartbeat") {
            return .transport("realtime_stale_session: \(closeReason ?? "heartbeat timeout")")
        }

        if closeCode != .invalid {
            var parts = ["Realtime websocket disconnected", "operation=\(operation)", "close_code=\(closeCode.rawValue)"]
            if let closeReason, !closeReason.isEmpty {
                parts.append("reason=\(closeReason)")
            }
            return .transport(parts.joined(separator: "; "))
        }

        return .transport(error.localizedDescription)
    }
}
