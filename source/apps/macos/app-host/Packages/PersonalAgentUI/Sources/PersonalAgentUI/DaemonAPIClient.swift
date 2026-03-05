import Foundation

enum DaemonAPIError: LocalizedError, Sendable {
    case missingAuthToken
    case invalidResponse
    case transport(String)
    case server(statusCode: Int, message: String)
    case serverProblem(
        statusCode: Int,
        message: String,
        code: String,
        details: DaemonProblemDetails?,
        correlationID: String?
    )
    case decoding(String)

    private var serverTuple: (
        statusCode: Int,
        message: String,
        code: String?,
        details: DaemonProblemDetails?,
        correlationID: String?
    )? {
        switch self {
        case .server(let statusCode, let message):
            return (statusCode, message, nil, nil, nil)
        case .serverProblem(
            let statusCode,
            let message,
            let code,
            let details,
            let correlationID
        ):
            return (statusCode, message, code, details, correlationID)
        default:
            return nil
        }
    }

    var errorDescription: String? {
        switch self {
        case .missingAuthToken:
            return "Assistant access token is not configured."
        case .invalidResponse:
            return "Daemon returned an invalid response."
        case .transport(let message):
            return message
        case .server(let statusCode, let message):
            return "Daemon request failed (\(statusCode)): \(message)"
        case .serverProblem(let statusCode, let message, _, _, _):
            return "Daemon request failed (\(statusCode)): \(message)"
        case .decoding(let message):
            return "Failed to decode daemon payload: \(message)"
        }
    }

    var isUnauthorized: Bool {
        serverTuple?.statusCode == 401
    }

    var isConnectivityIssue: Bool {
        if case .transport = self {
            return true
        }
        return false
    }

    var isMissingReadyChatModelRoute: Bool {
        guard let serverTuple, serverTuple.statusCode == 400 else {
            return false
        }
        return serverTuple.message.localizedCaseInsensitiveContains(
            "no enabled models with ready provider configuration"
        )
    }

    var serverStatusCode: Int? {
        serverTuple?.statusCode
    }

    var serverMessage: String? {
        serverTuple?.message
    }

    var serverCode: String? {
        serverTuple?.code
    }

    var serverDetails: DaemonProblemDetails? {
        serverTuple?.details
    }

    var serverCorrelationID: String? {
        serverTuple?.correlationID
    }
}

struct DaemonProblemService: Decodable, Sendable, Equatable {
    let id: String?
    let label: String?
    let configField: String?

    enum CodingKeys: String, CodingKey {
        case id
        case label
        case configField = "config_field"
    }
}

struct DaemonProblemRemediation: Decodable, Sendable, Equatable {
    let action: String?
    let label: String?
    let hint: String?
}

struct DaemonProblemDetails: Decodable, Sendable, Equatable {
    let category: String?
    let domain: String?
    let service: DaemonProblemService?
    let remediation: DaemonProblemRemediation?
}

enum DaemonJSONValue: Codable, Sendable {
    case string(String)
    case number(Double)
    case bool(Bool)
    case object([String: DaemonJSONValue])
    case array([DaemonJSONValue])
    case null

    public init(from decoder: Decoder) throws {
        let container = try decoder.singleValueContainer()
        if container.decodeNil() {
            self = .null
            return
        }
        if let boolValue = try? container.decode(Bool.self) {
            self = .bool(boolValue)
            return
        }
        if let intValue = try? container.decode(Int.self) {
            self = .number(Double(intValue))
            return
        }
        if let doubleValue = try? container.decode(Double.self) {
            self = .number(doubleValue)
            return
        }
        if let stringValue = try? container.decode(String.self) {
            self = .string(stringValue)
            return
        }
        if let objectValue = try? container.decode([String: DaemonJSONValue].self) {
            self = .object(objectValue)
            return
        }
        if let arrayValue = try? container.decode([DaemonJSONValue].self) {
            self = .array(arrayValue)
            return
        }
        throw DecodingError.typeMismatch(
            DaemonJSONValue.self,
            DecodingError.Context(codingPath: decoder.codingPath, debugDescription: "Unsupported daemon JSON value.")
        )
    }

    func encode(to encoder: Encoder) throws {
        var container = encoder.singleValueContainer()
        switch self {
        case .string(let value):
            try container.encode(value)
        case .number(let value):
            try container.encode(value)
        case .bool(let value):
            try container.encode(value)
        case .object(let value):
            try container.encode(value)
        case .array(let value):
            try container.encode(value)
        case .null:
            try container.encodeNil()
        }
    }

    var displayText: String {
        switch self {
        case .string(let value):
            return value
        case .number(let value):
            if value.rounded() == value {
                return String(Int(value))
            }
            return String(value)
        case .bool(let value):
            return value ? "true" : "false"
        case .null:
            return "null"
        case .array(let values):
            if values.isEmpty {
                return "[]"
            }
            return values.map(\.displayText).joined(separator: ", ")
        case .object(let values):
            if values.isEmpty {
                return "{}"
            }
            return values.keys.sorted().map { key in
                "\(key)=\(values[key]?.displayText ?? "")"
            }.joined(separator: ", ")
        }
    }

    var stringValue: String? {
        if case .string(let value) = self {
            return value
        }
        return nil
    }

    var objectValue: [String: DaemonJSONValue]? {
        if case .object(let value) = self {
            return value
        }
        return nil
    }
}

enum DaemonConfigMutationValue: Encodable, Sendable {
    case string(String)
    case number(Double)
    case bool(Bool)
    case null

    func encode(to encoder: Encoder) throws {
        var container = encoder.singleValueContainer()
        switch self {
        case .string(let value):
            try container.encode(value)
        case .number(let value):
            try container.encode(value)
        case .bool(let value):
            try container.encode(value)
        case .null:
            try container.encodeNil()
        }
    }
}

struct DaemonCapabilitySmokeResponse: Decodable, Sendable {
    let daemonVersion: String
    let channels: [String]
    let connectors: [String]
    let healthy: Bool
    let correlationID: String?

    enum CodingKeys: String, CodingKey {
        case daemonVersion = "daemon_version"
        case channels
        case connectors
        case healthy
        case correlationID = "correlation_id"
    }
}


















































































































































































































actor DaemonRealtimeSession {
    private let task: URLSessionWebSocketTask
    private let decoder = JSONDecoder()
    private let encoder = JSONEncoder()

    init(task: URLSessionWebSocketTask) {
        self.task = task
        self.task.resume()
    }

    func receive() async throws -> DaemonRealtimeEventEnvelope {
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
        case .data(let binary):
            data = binary
        @unknown default:
            throw DaemonAPIError.transport("Received unsupported realtime payload format.")
        }
        do {
            return try decoder.decode(DaemonRealtimeEventEnvelope.self, from: data)
        } catch {
            throw DaemonAPIError.decoding(error.localizedDescription)
        }
    }

    func sendSignal(_ signal: DaemonRealtimeClientSignal) async throws {
        do {
            let encoded = try encoder.encode(signal)
            guard let text = String(data: encoded, encoding: .utf8) else {
                throw DaemonAPIError.transport("Failed to encode realtime signal payload.")
            }
            try await task.send(.string(text))
        } catch let daemonError as DaemonAPIError {
            throw daemonError
        } catch {
            throw classifyRealtimeTransportError(error, operation: "send")
        }
    }

    func ping() async throws {
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
        } catch let daemonError as DaemonAPIError {
            throw daemonError
        } catch {
            throw classifyRealtimeTransportError(error, operation: "ping")
        }
    }

    func close() {
        task.cancel(with: .normalClosure, reason: nil)
    }

    private func classifyRealtimeTransportError(
        _ error: Error,
        operation: String
    ) -> DaemonAPIError {
        if let daemonError = error as? DaemonAPIError {
            return daemonError
        }

        let nsError = error as NSError
        if let response = failingHTTPResponse(from: nsError) {
            switch response.statusCode {
            case 401:
                return .server(statusCode: 401, message: "Realtime websocket authorization failed.")
            case 429:
                return .serverProblem(
                    statusCode: 429,
                    message: "Realtime websocket capacity exceeded.",
                    code: "rate_limit_exceeded",
                    details: DaemonProblemDetails(
                        category: "realtime_capacity",
                        domain: "realtime",
                        service: DaemonProblemService(id: "realtime.ws", label: "Realtime Stream", configField: nil),
                        remediation: DaemonProblemRemediation(
                            action: "retry_later",
                            label: "Retry Realtime Stream",
                            hint: "Wait for active realtime sessions to finish, then retry."
                        )
                    ),
                    correlationID: nil
                )
            default:
                return .server(statusCode: response.statusCode, message: "Realtime websocket request failed.")
            }
        }

        let closeCode = task.closeCode
        let closeReason = task.closeReason
            .flatMap { String(data: $0, encoding: .utf8) }?
            .trimmingCharacters(in: .whitespacesAndNewlines)
        let normalizedCloseReason = closeReason?.lowercased() ?? ""

        if normalizedCloseReason.contains("stale")
            || normalizedCloseReason.contains("heartbeat")
            || normalizedCloseReason.contains("pong")
        {
            return .transport("realtime_stale_session: \(closeReason ?? "heartbeat timeout")")
        }
        if normalizedCloseReason.contains("capacity")
            || normalizedCloseReason.contains("rate_limit")
            || normalizedCloseReason.contains("limit")
        {
            return .server(statusCode: 429, message: closeReason ?? "Realtime websocket capacity exceeded.")
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

    private func failingHTTPResponse(from error: NSError) -> HTTPURLResponse? {
        if let response = error.userInfo["NSErrorFailingURLResponseKey"] as? HTTPURLResponse {
            return response
        }
        return nil
    }
}

struct DaemonErrorResponse: Decodable {
    private struct ErrorObject: Decodable {
        let code: String?
        let message: String?
        let details: DaemonProblemDetails?
    }

    let message: String
    let code: String?
    let details: DaemonProblemDetails?
    let correlationID: String?
    let title: String?
    let type: String?

    enum CodingKeys: String, CodingKey {
        case error
        case correlationID = "correlation_id"
        case title
        case detail
        case type
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)

        let errorObject = try? container.decode(ErrorObject.self, forKey: .error)
        let messageFromObject = Self.trimmed(errorObject?.message)
        let messageFromDetail = Self.trimmed(try container.decodeIfPresent(String.self, forKey: .detail))
        let messageFromTitle = Self.trimmed(try container.decodeIfPresent(String.self, forKey: .title))

        message =
            messageFromObject
            ?? messageFromDetail
            ?? messageFromTitle
            ?? "Request failed"

        let codeFromObject = Self.trimmed(errorObject?.code)
        let problemType = Self.trimmed(try container.decodeIfPresent(String.self, forKey: .type))
        let codeFromProblemType = Self.problemCode(fromTypeURI: problemType)
        code = codeFromObject ?? codeFromProblemType

        details = errorObject?.details

        correlationID = Self.trimmed(try container.decodeIfPresent(String.self, forKey: .correlationID))
        title = messageFromTitle
        type = problemType
    }

    private static func trimmed(_ value: String?) -> String? {
        guard let value else {
            return nil
        }
        let trimmed = value.trimmingCharacters(in: .whitespacesAndNewlines)
        return trimmed.isEmpty ? nil : trimmed
    }

    private static func problemCode(fromTypeURI typeURI: String?) -> String? {
        guard let trimmedType = trimmed(typeURI) else {
            return nil
        }
        guard let slashIndex = trimmedType.lastIndex(of: "/") else {
            return nil
        }
        let candidate = String(trimmedType[trimmedType.index(after: slashIndex)...])
        return trimmed(candidate)
    }
}

struct DaemonAPIClient {
    private let core: DaemonAPITransportCore

    init(
        session: URLSession = .shared,
        fixtureScenario: DaemonAPISmokeFixtureScenario? = DaemonAPISmokeFixtureScenario.fromEnvironment()
    ) {
        self.core = DaemonAPITransportCore(
            session: session,
            fixtureScenario: fixtureScenario
        )
    }

    func capabilitySmoke(baseURL: URL, authToken: String) async throws -> DaemonCapabilitySmokeResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/capabilities/smoke",
            method: "GET",
            authToken: authToken,
            body: Optional<DaemonWorkspaceRequest>.none
        )
    }

    func daemonLifecycleStatus(baseURL: URL, authToken: String) async throws -> DaemonLifecycleStatusResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/daemon/lifecycle/status",
            method: "GET",
            authToken: authToken,
            body: Optional<DaemonWorkspaceRequest>.none
        )
    }

    func daemonLifecycleControl(
        baseURL: URL,
        authToken: String,
        action: String,
        reason: String
    ) async throws -> DaemonLifecycleControlResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/daemon/lifecycle/control",
            method: "POST",
            authToken: authToken,
            body: DaemonLifecycleControlRequest(action: action, reason: reason)
        )
    }

    func daemonPluginLifecycleHistory(
        baseURL: URL,
        authToken: String,
        workspaceID: String? = nil,
        pluginID: String? = nil,
        kind: String? = nil,
        state: String? = nil,
        eventType: String? = nil,
        cursorCreatedAt: String? = nil,
        cursorID: String? = nil,
        limit: Int? = nil
    ) async throws -> DaemonPluginLifecycleHistoryResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/daemon/lifecycle/plugins/history",
            method: "POST",
            authToken: authToken,
            body: DaemonPluginLifecycleHistoryRequest(
                workspaceID: workspaceID,
                pluginID: pluginID,
                kind: kind,
                state: state,
                eventType: eventType,
                cursorCreatedAt: cursorCreatedAt,
                cursorID: cursorID,
                limit: limit
            )
        )
    }

    func providerList(
        baseURL: URL,
        authToken: String,
        workspaceID: String
    ) async throws -> DaemonProviderListResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/providers/list",
            method: "POST",
            authToken: authToken,
            body: DaemonWorkspaceRequest(workspaceID: workspaceID)
        )
    }

    func secretReferenceUpsert(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        name: String,
        backend: String,
        service: String,
        account: String
    ) async throws -> DaemonSecretReferenceResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/secrets/refs",
            method: "POST",
            authToken: authToken,
            body: DaemonSecretReferenceUpsertRequest(
                workspaceID: workspaceID,
                name: name,
                backend: backend,
                service: service,
                account: account
            )
        )
    }

    func providerSet(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        provider: String,
        endpoint: String? = nil,
        apiKeySecretName: String? = nil,
        clearAPIKey: Bool = false
    ) async throws -> DaemonProviderConfigRecord {
        try await request(
            baseURL: baseURL,
            path: "/v1/providers/set",
            method: "POST",
            authToken: authToken,
            body: DaemonProviderSetRequest(
                workspaceID: workspaceID,
                provider: provider,
                endpoint: endpoint,
                apiKeySecretName: apiKeySecretName,
                clearAPIKey: clearAPIKey
            )
        )
    }

    func delegationList(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        fromActorID: String? = nil,
        toActorID: String? = nil
    ) async throws -> DaemonDelegationListResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/delegation/list",
            method: "POST",
            authToken: authToken,
            body: DaemonDelegationListRequest(
                workspaceID: workspaceID,
                fromActorID: fromActorID,
                toActorID: toActorID
            )
        )
    }

    func delegationGrant(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        fromActorID: String,
        toActorID: String,
        scopeType: String,
        scopeKey: String? = nil,
        expiresAt: String? = nil
    ) async throws -> DaemonDelegationRuleRecord {
        try await request(
            baseURL: baseURL,
            path: "/v1/delegation/grant",
            method: "POST",
            authToken: authToken,
            body: DaemonDelegationGrantRequest(
                workspaceID: workspaceID,
                fromActorID: fromActorID,
                toActorID: toActorID,
                scopeType: scopeType,
                scopeKey: scopeKey,
                expiresAt: expiresAt
            )
        )
    }

    func delegationRevoke(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        ruleID: String
    ) async throws -> DaemonDelegationRevokeResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/delegation/revoke",
            method: "POST",
            authToken: authToken,
            body: DaemonDelegationRevokeRequest(
                workspaceID: workspaceID,
                ruleID: ruleID
            )
        )
    }

    func capabilityGrantUpsert(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        grantID: String? = nil,
        actorID: String? = nil,
        capabilityKey: String? = nil,
        scopeJSON: String? = nil,
        status: String? = nil,
        expiresAt: String? = nil
    ) async throws -> DaemonCapabilityGrantRecord {
        try await request(
            baseURL: baseURL,
            path: "/v1/delegation/capability-grants/upsert",
            method: "POST",
            authToken: authToken,
            body: DaemonCapabilityGrantUpsertRequest(
                workspaceID: workspaceID,
                grantID: grantID,
                actorID: actorID,
                capabilityKey: capabilityKey,
                scopeJSON: scopeJSON,
                status: status,
                expiresAt: expiresAt
            )
        )
    }

    func capabilityGrantList(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        actorID: String? = nil,
        capabilityKey: String? = nil,
        status: String? = nil,
        cursorCreatedAt: String? = nil,
        cursorID: String? = nil,
        limit: Int = 50
    ) async throws -> DaemonCapabilityGrantListResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/delegation/capability-grants/list",
            method: "POST",
            authToken: authToken,
            body: DaemonCapabilityGrantListRequest(
                workspaceID: workspaceID,
                actorID: actorID,
                capabilityKey: capabilityKey,
                status: status,
                cursorCreatedAt: cursorCreatedAt,
                cursorID: cursorID,
                limit: limit
            )
        )
    }

    func identityWorkspaces(
        baseURL: URL,
        authToken: String,
        includeInactive: Bool = true
    ) async throws -> DaemonIdentityWorkspacesResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/identity/workspaces",
            method: "POST",
            authToken: authToken,
            body: DaemonIdentityWorkspacesRequest(includeInactive: includeInactive)
        )
    }

    func identityPrincipals(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        includeInactive: Bool = true
    ) async throws -> DaemonIdentityPrincipalsResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/identity/principals",
            method: "POST",
            authToken: authToken,
            body: DaemonIdentityPrincipalsRequest(
                workspaceID: workspaceID,
                includeInactive: includeInactive
            )
        )
    }

    func identityActiveContext(
        baseURL: URL,
        authToken: String,
        workspaceID: String? = nil
    ) async throws -> DaemonIdentityActiveContextResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/identity/context",
            method: "POST",
            authToken: authToken,
            body: DaemonIdentityActiveContextRequest(workspaceID: workspaceID)
        )
    }

    func identitySelectWorkspace(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        principalActorID: String? = nil,
        source: String? = "ui.configuration.identity_hub"
    ) async throws -> DaemonIdentityActiveContextResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/identity/context/select-workspace",
            method: "POST",
            authToken: authToken,
            body: DaemonIdentitySelectWorkspaceRequest(
                workspaceID: workspaceID,
                principalActorID: principalActorID,
                source: source
            )
        )
    }

    func identityDevicesList(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        userID: String? = nil,
        deviceType: String? = nil,
        platform: String? = nil,
        cursorCreatedAt: String? = nil,
        cursorID: String? = nil,
        limit: Int = 25
    ) async throws -> DaemonIdentityDeviceListResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/identity/devices/list",
            method: "POST",
            authToken: authToken,
            body: DaemonIdentityDeviceListRequest(
                workspaceID: workspaceID,
                userID: userID,
                deviceType: deviceType,
                platform: platform,
                cursorCreatedAt: cursorCreatedAt,
                cursorID: cursorID,
                limit: limit
            )
        )
    }

    func identitySessionsList(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        deviceID: String? = nil,
        userID: String? = nil,
        sessionHealth: String? = nil,
        cursorStartedAt: String? = nil,
        cursorID: String? = nil,
        limit: Int = 25
    ) async throws -> DaemonIdentitySessionListResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/identity/sessions/list",
            method: "POST",
            authToken: authToken,
            body: DaemonIdentitySessionListRequest(
                workspaceID: workspaceID,
                deviceID: deviceID,
                userID: userID,
                sessionHealth: sessionHealth,
                cursorStartedAt: cursorStartedAt,
                cursorID: cursorID,
                limit: limit
            )
        )
    }

    func identitySessionRevoke(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        sessionID: String
    ) async throws -> DaemonIdentitySessionRevokeResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/identity/sessions/revoke",
            method: "POST",
            authToken: authToken,
            body: DaemonIdentitySessionRevokeRequest(
                workspaceID: workspaceID,
                sessionID: sessionID
            )
        )
    }

    func connectRealtime(
        baseURL: URL,
        authToken: String,
        correlationID: String? = nil
    ) throws -> DaemonRealtimeSession {
        try core.connectRealtime(
            baseURL: baseURL,
            authToken: authToken,
            correlationID: correlationID
        )
    }

    func providerCheck(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        provider: String? = nil
    ) async throws -> DaemonProviderCheckResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/providers/check",
            method: "POST",
            authToken: authToken,
            body: DaemonProviderCheckRequest(workspaceID: workspaceID, provider: provider)
        )
    }

    func modelResolve(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        taskClass: String = "chat"
    ) async throws -> DaemonModelResolveResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/models/resolve",
            method: "POST",
            authToken: authToken,
            body: DaemonModelResolveRequest(workspaceID: workspaceID, taskClass: taskClass)
        )
    }

    func modelRouteSimulate(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        taskClass: String = "chat",
        principalActorID: String? = nil
    ) async throws -> DaemonModelRouteSimulationResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/models/route/simulate",
            method: "POST",
            authToken: authToken,
            body: DaemonModelRouteSimulationRequest(
                workspaceID: workspaceID,
                taskClass: taskClass,
                principalActorID: principalActorID
            )
        )
    }

    func modelRouteExplain(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        taskClass: String = "chat",
        principalActorID: String? = nil
    ) async throws -> DaemonModelRouteExplainResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/models/route/explain",
            method: "POST",
            authToken: authToken,
            body: DaemonModelRouteExplainRequest(
                workspaceID: workspaceID,
                taskClass: taskClass,
                principalActorID: principalActorID
            )
        )
    }

    func modelList(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        provider: String? = nil
    ) async throws -> DaemonModelListResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/models/list",
            method: "POST",
            authToken: authToken,
            body: DaemonModelListRequest(workspaceID: workspaceID, provider: provider)
        )
    }

    func modelDiscover(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        provider: String? = nil
    ) async throws -> DaemonModelDiscoverResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/models/discover",
            method: "POST",
            authToken: authToken,
            body: DaemonModelDiscoverRequest(workspaceID: workspaceID, provider: provider)
        )
    }

    func modelAdd(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        provider: String,
        modelKey: String,
        enabled: Bool = false
    ) async throws -> DaemonModelCatalogEntryRecord {
        try await request(
            baseURL: baseURL,
            path: "/v1/models/add",
            method: "POST",
            authToken: authToken,
            body: DaemonModelCatalogAddRequest(
                workspaceID: workspaceID,
                provider: provider,
                modelKey: modelKey,
                enabled: enabled
            )
        )
    }

    func modelRemove(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        provider: String,
        modelKey: String
    ) async throws -> DaemonModelCatalogRemoveResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/models/remove",
            method: "POST",
            authToken: authToken,
            body: DaemonModelCatalogRemoveRequest(
                workspaceID: workspaceID,
                provider: provider,
                modelKey: modelKey
            )
        )
    }

    func modelEnable(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        provider: String,
        modelKey: String
    ) async throws -> DaemonModelCatalogEntryRecord {
        try await request(
            baseURL: baseURL,
            path: "/v1/models/enable",
            method: "POST",
            authToken: authToken,
            body: DaemonModelToggleRequest(
                workspaceID: workspaceID,
                provider: provider,
                modelKey: modelKey
            )
        )
    }

    func modelDisable(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        provider: String,
        modelKey: String
    ) async throws -> DaemonModelCatalogEntryRecord {
        try await request(
            baseURL: baseURL,
            path: "/v1/models/disable",
            method: "POST",
            authToken: authToken,
            body: DaemonModelToggleRequest(
                workspaceID: workspaceID,
                provider: provider,
                modelKey: modelKey
            )
        )
    }

    func modelSelect(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        taskClass: String = "chat",
        provider: String,
        modelKey: String
    ) async throws -> DaemonModelRoutingPolicyRecord {
        try await request(
            baseURL: baseURL,
            path: "/v1/models/select",
            method: "POST",
            authToken: authToken,
            body: DaemonModelSelectRequest(
                workspaceID: workspaceID,
                taskClass: taskClass,
                provider: provider,
                modelKey: modelKey
            )
        )
    }

    func modelPolicy(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        taskClass: String? = nil
    ) async throws -> DaemonModelPolicyResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/models/policy",
            method: "POST",
            authToken: authToken,
            body: DaemonModelPolicyRequest(workspaceID: workspaceID, taskClass: taskClass)
        )
    }

    func channelStatus(
        baseURL: URL,
        authToken: String,
        workspaceID: String
    ) async throws -> DaemonChannelStatusResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/channels/status",
            method: "POST",
            authToken: authToken,
            body: DaemonWorkspaceRequest(workspaceID: workspaceID)
        )
    }

    func channelConnectorMappingsList(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        channelID: String? = nil
    ) async throws -> DaemonChannelConnectorMappingListResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/channels/mappings/list",
            method: "POST",
            authToken: authToken,
            body: DaemonChannelConnectorMappingListRequest(
                workspaceID: workspaceID,
                channelID: channelID
            )
        )
    }

    func channelConnectorMappingUpsert(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        channelID: String,
        connectorID: String,
        enabled: Bool,
        priority: Int? = nil,
        fallbackPolicy: String? = nil
    ) async throws -> DaemonChannelConnectorMappingUpsertResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/channels/mappings/upsert",
            method: "POST",
            authToken: authToken,
            body: DaemonChannelConnectorMappingUpsertRequest(
                workspaceID: workspaceID,
                channelID: channelID,
                connectorID: connectorID,
                enabled: enabled,
                priority: priority,
                fallbackPolicy: fallbackPolicy
            )
        )
    }

    func channelDiagnostics(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        channelID: String? = nil
    ) async throws -> DaemonChannelDiagnosticsResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/channels/diagnostics",
            method: "POST",
            authToken: authToken,
            body: DaemonChannelDiagnosticsRequest(workspaceID: workspaceID, channelID: channelID)
        )
    }

    func connectorStatus(
        baseURL: URL,
        authToken: String,
        workspaceID: String
    ) async throws -> DaemonConnectorStatusResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/connectors/status",
            method: "POST",
            authToken: authToken,
            body: DaemonWorkspaceRequest(workspaceID: workspaceID)
        )
    }

    func connectorDiagnostics(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        connectorID: String? = nil
    ) async throws -> DaemonConnectorDiagnosticsResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/connectors/diagnostics",
            method: "POST",
            authToken: authToken,
            body: DaemonConnectorDiagnosticsRequest(
                workspaceID: workspaceID,
                connectorID: connectorID
            )
        )
    }

    func channelConfigUpsert(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        channelID: String,
        configuration: [String: DaemonConfigMutationValue],
        merge: Bool = true
    ) async throws -> DaemonChannelConfigUpsertResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/channels/config/upsert",
            method: "POST",
            authToken: authToken,
            body: DaemonChannelConfigUpsertRequest(
                workspaceID: workspaceID,
                channelID: channelID,
                configuration: configuration,
                merge: merge
            )
        )
    }

    func connectorConfigUpsert(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        connectorID: String,
        configuration: [String: DaemonConfigMutationValue],
        merge: Bool = true
    ) async throws -> DaemonConnectorConfigUpsertResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/connectors/config/upsert",
            method: "POST",
            authToken: authToken,
            body: DaemonConnectorConfigUpsertRequest(
                workspaceID: workspaceID,
                connectorID: connectorID,
                configuration: configuration,
                merge: merge
            )
        )
    }

    func channelTestOperation(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        channelID: String,
        operation: String = "health"
    ) async throws -> DaemonChannelTestOperationResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/channels/test",
            method: "POST",
            authToken: authToken,
            body: DaemonChannelTestOperationRequest(
                workspaceID: workspaceID,
                channelID: channelID,
                operation: operation
            )
        )
    }

    func connectorTestOperation(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        connectorID: String,
        operation: String = "health"
    ) async throws -> DaemonConnectorTestOperationResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/connectors/test",
            method: "POST",
            authToken: authToken,
            body: DaemonConnectorTestOperationRequest(
                workspaceID: workspaceID,
                connectorID: connectorID,
                operation: operation
            )
        )
    }

    func connectorPermissionRequest(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        connectorID: String
    ) async throws -> DaemonConnectorPermissionResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/connectors/permission/request",
            method: "POST",
            authToken: authToken,
            body: DaemonConnectorPermissionRequest(
                workspaceID: workspaceID,
                connectorID: connectorID
            )
        )
    }

    func approvalInbox(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        includeFinal: Bool = true,
        limit: Int = 80,
        state: String? = nil
    ) async throws -> DaemonApprovalInboxResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/approvals/list",
            method: "POST",
            authToken: authToken,
            body: DaemonApprovalInboxRequest(
                workspaceID: workspaceID,
                includeFinal: includeFinal,
                limit: limit,
                state: state
            )
        )
    }

    func approvalDecision(
        baseURL: URL,
        authToken: String,
        approvalID: String,
        decisionPhrase: String,
        decisionByActorID: String,
        rationale: String? = nil
    ) async throws -> DaemonApprovalDecisionResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/approvals/\(approvalID)",
            method: "POST",
            authToken: authToken,
            body: DaemonApprovalDecisionRequest(
                decisionPhrase: decisionPhrase,
                decisionByActorID: decisionByActorID,
                rationale: rationale
            )
        )
    }

    func taskRunList(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        state: String? = nil,
        limit: Int = 80
    ) async throws -> DaemonTaskRunListResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/tasks/list",
            method: "POST",
            authToken: authToken,
            body: DaemonTaskRunListRequest(
                workspaceID: workspaceID,
                state: state,
                limit: limit
            )
        )
    }

    func taskCancel(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        taskID: String? = nil,
        runID: String? = nil,
        reason: String? = nil
    ) async throws -> DaemonTaskCancelResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/tasks/cancel",
            method: "POST",
            authToken: authToken,
            body: DaemonTaskRunControlRequest(
                workspaceID: workspaceID,
                taskID: taskID,
                runID: runID,
                reason: reason
            )
        )
    }

    func taskRetry(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        taskID: String? = nil,
        runID: String? = nil,
        reason: String? = nil
    ) async throws -> DaemonTaskRetryResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/tasks/retry",
            method: "POST",
            authToken: authToken,
            body: DaemonTaskRunControlRequest(
                workspaceID: workspaceID,
                taskID: taskID,
                runID: runID,
                reason: reason
            )
        )
    }

    func taskRequeue(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        taskID: String? = nil,
        runID: String? = nil,
        reason: String? = nil
    ) async throws -> DaemonTaskRequeueResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/tasks/requeue",
            method: "POST",
            authToken: authToken,
            body: DaemonTaskRunControlRequest(
                workspaceID: workspaceID,
                taskID: taskID,
                runID: runID,
                reason: reason
            )
        )
    }

    func taskSubmit(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        requestedByActorID: String,
        subjectPrincipalActorID: String,
        title: String,
        description: String? = nil,
        taskClass: String? = nil
    ) async throws -> DaemonTaskSubmitResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/tasks",
            method: "POST",
            authToken: authToken,
            body: DaemonTaskSubmitRequest(
                workspaceID: workspaceID,
                requestedByActorID: requestedByActorID,
                subjectPrincipalActorID: subjectPrincipalActorID,
                title: title,
                description: description,
                taskClass: taskClass
            )
        )
    }

    func commSend(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        operationID: String,
        sourceChannel: String,
        threadID: String? = nil,
        connectorID: String? = nil,
        destination: String? = nil,
        message: String,
        stepID: String? = nil,
        eventID: String? = nil,
        iMessageFailures: Int? = nil,
        smsFailures: Int? = nil
    ) async throws -> DaemonCommSendResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/comm/send",
            method: "POST",
            authToken: authToken,
            body: DaemonCommSendRequest(
                workspaceID: workspaceID,
                operationID: operationID,
                sourceChannel: sourceChannel,
                threadID: threadID,
                connectorID: connectorID,
                destination: destination,
                message: message,
                stepID: stepID,
                eventID: eventID,
                iMessageFailures: iMessageFailures,
                smsFailures: smsFailures
            )
        )
    }

    func commThreadList(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        channel: String? = nil,
        connectorID: String? = nil,
        query: String? = nil,
        cursor: String? = nil,
        limit: Int = 80
    ) async throws -> DaemonCommThreadListResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/comm/threads/list",
            method: "POST",
            authToken: authToken,
            body: DaemonCommThreadListRequest(
                workspaceID: workspaceID,
                channel: channel,
                connectorID: connectorID,
                query: query,
                cursor: cursor,
                limit: limit
            )
        )
    }

    func commEventTimeline(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        threadID: String? = nil,
        channel: String? = nil,
        connectorID: String? = nil,
        eventType: String? = nil,
        direction: String? = nil,
        query: String? = nil,
        cursor: String? = nil,
        limit: Int = 120
    ) async throws -> DaemonCommEventTimelineResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/comm/events/list",
            method: "POST",
            authToken: authToken,
            body: DaemonCommEventTimelineRequest(
                workspaceID: workspaceID,
                threadID: threadID,
                channel: channel,
                connectorID: connectorID,
                eventType: eventType,
                direction: direction,
                query: query,
                cursor: cursor,
                limit: limit
            )
        )
    }

    func commCallSessionList(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        threadID: String? = nil,
        provider: String? = nil,
        connectorID: String? = nil,
        direction: String? = nil,
        status: String? = nil,
        providerCallID: String? = nil,
        query: String? = nil,
        cursor: String? = nil,
        limit: Int = 80
    ) async throws -> DaemonCommCallSessionListResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/comm/call-sessions/list",
            method: "POST",
            authToken: authToken,
            body: DaemonCommCallSessionListRequest(
                workspaceID: workspaceID,
                threadID: threadID,
                provider: provider,
                connectorID: connectorID,
                direction: direction,
                status: status,
                providerCallID: providerCallID,
                query: query,
                cursor: cursor,
                limit: limit
            )
        )
    }

    func commAttempts(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        operationID: String? = nil,
        threadID: String? = nil,
        taskID: String? = nil,
        runID: String? = nil,
        stepID: String? = nil,
        channel: String? = nil,
        status: String? = nil,
        cursor: String? = nil,
        limit: Int = 120
    ) async throws -> DaemonCommAttemptsResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/comm/attempts",
            method: "POST",
            authToken: authToken,
            body: DaemonCommAttemptsRequest(
                workspaceID: workspaceID,
                operationID: operationID,
                threadID: threadID,
                taskID: taskID,
                runID: runID,
                stepID: stepID,
                channel: channel,
                status: status,
                cursor: cursor,
                limit: limit
            )
        )
    }

    func commWebhookReceiptList(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        provider: String? = nil,
        providerEventID: String? = nil,
        providerEventQuery: String? = nil,
        eventID: String? = nil,
        cursorCreatedAt: String? = nil,
        cursorID: String? = nil,
        limit: Int = 80
    ) async throws -> DaemonCommWebhookReceiptListResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/comm/webhook-receipts/list",
            method: "POST",
            authToken: authToken,
            body: DaemonCommWebhookReceiptListRequest(
                workspaceID: workspaceID,
                provider: provider,
                providerEventID: providerEventID,
                providerEventQuery: providerEventQuery,
                eventID: eventID,
                cursorCreatedAt: cursorCreatedAt,
                cursorID: cursorID,
                limit: limit
            )
        )
    }

    func commIngestReceiptList(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        source: String? = nil,
        sourceScope: String? = nil,
        sourceEventID: String? = nil,
        sourceEventQuery: String? = nil,
        trustState: String? = nil,
        eventID: String? = nil,
        cursorCreatedAt: String? = nil,
        cursorID: String? = nil,
        limit: Int = 80
    ) async throws -> DaemonCommIngestReceiptListResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/comm/ingest-receipts/list",
            method: "POST",
            authToken: authToken,
            body: DaemonCommIngestReceiptListRequest(
                workspaceID: workspaceID,
                source: source,
                sourceScope: sourceScope,
                sourceEventID: sourceEventID,
                sourceEventQuery: sourceEventQuery,
                trustState: trustState,
                eventID: eventID,
                cursorCreatedAt: cursorCreatedAt,
                cursorID: cursorID,
                limit: limit
            )
        )
    }

    func commPolicySet(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        policyID: String? = nil,
        sourceChannel: String,
        endpointPattern: String? = nil,
        primaryChannel: String,
        retryCount: Int,
        fallbackChannels: [String],
        isDefault: Bool
    ) async throws -> DaemonCommPolicyRecord {
        try await request(
            baseURL: baseURL,
            path: "/v1/comm/policy/set",
            method: "POST",
            authToken: authToken,
            body: DaemonCommPolicySetRequest(
                policyID: policyID,
                workspaceID: workspaceID,
                sourceChannel: sourceChannel,
                endpointPattern: endpointPattern,
                primaryChannel: primaryChannel,
                retryCount: retryCount,
                fallbackChannels: fallbackChannels,
                isDefault: isDefault
            )
        )
    }

    func commPolicyList(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        sourceChannel: String? = nil
    ) async throws -> DaemonCommPolicyListResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/comm/policy/list",
            method: "POST",
            authToken: authToken,
            body: DaemonCommPolicyListRequest(
                workspaceID: workspaceID,
                sourceChannel: sourceChannel
            )
        )
    }

    func automationList(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        triggerType: String? = nil,
        includeDisabled: Bool = true
    ) async throws -> DaemonAutomationListResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/automation/list",
            method: "POST",
            authToken: authToken,
            body: DaemonAutomationListRequest(
                workspaceID: workspaceID,
                triggerType: triggerType,
                includeDisabled: includeDisabled
            )
        )
    }

    func automationCreate(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        subjectActorID: String?,
        triggerType: String,
        title: String,
        instruction: String,
        intervalSeconds: Int? = nil,
        filterJSON: String? = nil,
        cooldownSeconds: Int? = nil,
        enabled: Bool
    ) async throws -> DaemonAutomationTriggerRecord {
        try await request(
            baseURL: baseURL,
            path: "/v1/automation/create",
            method: "POST",
            authToken: authToken,
            body: DaemonAutomationCreateRequest(
                workspaceID: workspaceID,
                subjectActorID: subjectActorID,
                triggerType: triggerType,
                title: title,
                instruction: instruction,
                intervalSeconds: intervalSeconds,
                filterJSON: filterJSON,
                cooldownSeconds: cooldownSeconds,
                enabled: enabled
            )
        )
    }

    func automationFireHistory(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        triggerID: String? = nil,
        status: String? = nil,
        limit: Int = 50
    ) async throws -> DaemonAutomationFireHistoryResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/automation/fire-history",
            method: "POST",
            authToken: authToken,
            body: DaemonAutomationFireHistoryRequest(
                workspaceID: workspaceID,
                triggerID: triggerID,
                status: status,
                limit: limit
            )
        )
    }

    func automationUpdate(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        triggerID: String,
        subjectActorID: String?,
        title: String?,
        instruction: String?,
        intervalSeconds: Int? = nil,
        filterJSON: String? = nil,
        cooldownSeconds: Int? = nil,
        enabled: Bool? = nil
    ) async throws -> DaemonAutomationUpdateResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/automation/update",
            method: "POST",
            authToken: authToken,
            body: DaemonAutomationUpdateRequest(
                workspaceID: workspaceID,
                triggerID: triggerID,
                subjectActorID: subjectActorID,
                title: title,
                instruction: instruction,
                intervalSeconds: intervalSeconds,
                filterJSON: filterJSON,
                cooldownSeconds: cooldownSeconds,
                enabled: enabled
            )
        )
    }

    func automationDelete(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        triggerID: String
    ) async throws -> DaemonAutomationDeleteResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/automation/delete",
            method: "POST",
            authToken: authToken,
            body: DaemonAutomationDeleteRequest(workspaceID: workspaceID, triggerID: triggerID)
        )
    }

    func automationRunSchedule(
        baseURL: URL,
        authToken: String,
        at: String? = nil
    ) async throws -> DaemonAutomationRunScheduleResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/automation/run/schedule",
            method: "POST",
            authToken: authToken,
            body: DaemonAutomationRunScheduleRequest(at: at)
        )
    }

    func automationRunCommEvent(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        eventID: String,
        channel: String,
        body: String,
        sender: String
    ) async throws -> DaemonAutomationRunCommEventResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/automation/run/comm-event",
            method: "POST",
            authToken: authToken,
            body: DaemonAutomationRunCommEventRequest(
                workspaceID: workspaceID,
                eventID: eventID,
                seedEvent: true,
                threadID: nil,
                channel: channel,
                body: body,
                sender: sender,
                eventType: "MESSAGE_RECEIVED",
                direction: "inbound",
                assistantEmitted: false,
                occurredAt: nil
            )
        )
    }

    func retentionPurge(
        baseURL: URL,
        authToken: String,
        traceDays: Int,
        transcriptDays: Int,
        memoryDays: Int
    ) async throws -> DaemonJSONValue {
        try await request(
            baseURL: baseURL,
            path: "/v1/retention/purge",
            method: "POST",
            authToken: authToken,
            body: DaemonRetentionPurgeRequest(
                traceDays: traceDays,
                transcriptDays: transcriptDays,
                memoryDays: memoryDays
            )
        )
    }

    func retentionCompactMemory(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        ownerActor: String,
        tokenThreshold: Int,
        staleAfterHours: Int,
        limit: Int,
        apply: Bool
    ) async throws -> DaemonJSONValue {
        try await request(
            baseURL: baseURL,
            path: "/v1/retention/compact-memory",
            method: "POST",
            authToken: authToken,
            body: DaemonRetentionCompactMemoryRequest(
                workspaceID: workspaceID,
                ownerActor: ownerActor,
                tokenThreshold: tokenThreshold,
                staleAfterHours: staleAfterHours,
                limit: limit,
                apply: apply
            )
        )
    }

    func contextSamples(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        taskClass: String,
        limit: Int
    ) async throws -> DaemonJSONValue {
        try await request(
            baseURL: baseURL,
            path: "/v1/context/samples",
            method: "POST",
            authToken: authToken,
            body: DaemonContextSamplesRequest(
                workspaceID: workspaceID,
                taskClass: taskClass,
                limit: limit
            )
        )
    }

    func contextTune(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        taskClass: String
    ) async throws -> DaemonJSONValue {
        try await request(
            baseURL: baseURL,
            path: "/v1/context/tune",
            method: "POST",
            authToken: authToken,
            body: DaemonContextTuneRequest(
                workspaceID: workspaceID,
                taskClass: taskClass
            )
        )
    }

    func contextMemoryInventory(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        ownerActorID: String? = nil,
        scopeType: String? = nil,
        status: String? = nil,
        sourceType: String? = nil,
        sourceRefQuery: String? = nil,
        cursorUpdatedAt: String? = nil,
        cursorID: String? = nil,
        limit: Int = 25
    ) async throws -> DaemonContextMemoryInventoryResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/context/memory/inventory",
            method: "POST",
            authToken: authToken,
            body: DaemonContextMemoryInventoryRequest(
                workspaceID: workspaceID,
                ownerActorID: ownerActorID,
                scopeType: scopeType,
                status: status,
                sourceType: sourceType,
                sourceRefQuery: sourceRefQuery,
                cursorUpdatedAt: cursorUpdatedAt,
                cursorID: cursorID,
                limit: limit
            )
        )
    }

    func contextMemoryCompactionCandidates(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        ownerActorID: String? = nil,
        status: String? = nil,
        cursorCreatedAt: String? = nil,
        cursorID: String? = nil,
        limit: Int = 25
    ) async throws -> DaemonContextMemoryCandidatesResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/context/memory/compaction-candidates",
            method: "POST",
            authToken: authToken,
            body: DaemonContextMemoryCandidatesRequest(
                workspaceID: workspaceID,
                ownerActorID: ownerActorID,
                status: status,
                cursorCreatedAt: cursorCreatedAt,
                cursorID: cursorID,
                limit: limit
            )
        )
    }

    func contextRetrievalDocuments(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        ownerActorID: String? = nil,
        sourceURIQuery: String? = nil,
        cursorCreatedAt: String? = nil,
        cursorID: String? = nil,
        limit: Int = 25
    ) async throws -> DaemonContextRetrievalDocumentsResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/context/retrieval/documents",
            method: "POST",
            authToken: authToken,
            body: DaemonContextRetrievalDocumentsRequest(
                workspaceID: workspaceID,
                ownerActorID: ownerActorID,
                sourceURIQuery: sourceURIQuery,
                cursorCreatedAt: cursorCreatedAt,
                cursorID: cursorID,
                limit: limit
            )
        )
    }

    func contextRetrievalChunks(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        documentID: String,
        ownerActorID: String? = nil,
        sourceURIQuery: String? = nil,
        chunkTextQuery: String? = nil,
        cursorCreatedAt: String? = nil,
        cursorID: String? = nil,
        limit: Int = 25
    ) async throws -> DaemonContextRetrievalChunksResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/context/retrieval/chunks",
            method: "POST",
            authToken: authToken,
            body: DaemonContextRetrievalChunksRequest(
                workspaceID: workspaceID,
                documentID: documentID,
                ownerActorID: ownerActorID,
                sourceURIQuery: sourceURIQuery,
                chunkTextQuery: chunkTextQuery,
                cursorCreatedAt: cursorCreatedAt,
                cursorID: cursorID,
                limit: limit
            )
        )
    }

    func inspectLogsQuery(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        runID: String? = nil,
        eventType: String? = nil,
        beforeCreatedAt: String? = nil,
        beforeID: String? = nil,
        limit: Int = 80
    ) async throws -> DaemonInspectLogQueryResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/inspect/logs/query",
            method: "POST",
            authToken: authToken,
            body: DaemonInspectLogQueryRequest(
                workspaceID: workspaceID,
                runID: runID,
                eventType: eventType,
                beforeCreatedAt: beforeCreatedAt,
                beforeID: beforeID,
                limit: limit
            )
        )
    }

    func inspectRun(
        baseURL: URL,
        authToken: String,
        runID: String
    ) async throws -> DaemonInspectRunResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/inspect/run",
            method: "POST",
            authToken: authToken,
            body: DaemonInspectRunRequest(runID: runID)
        )
    }

    func inspectLogsStream(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        runID: String? = nil,
        cursorCreatedAt: String?,
        cursorID: String?,
        limit: Int = 80,
        timeoutMS: Int64 = 1500,
        pollIntervalMS: Int64 = 200
    ) async throws -> DaemonInspectLogStreamResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/inspect/logs/stream",
            method: "POST",
            authToken: authToken,
            body: DaemonInspectLogStreamRequest(
                workspaceID: workspaceID,
                runID: runID,
                eventType: nil,
                cursorCreatedAt: cursorCreatedAt,
                cursorID: cursorID,
                limit: limit,
                timeoutMS: timeoutMS,
                pollIntervalMS: pollIntervalMS
            )
        )
    }

    func chatTurn(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        messages: [(role: String, content: String)],
        systemPrompt: String? = nil,
        requestedByActorID: String? = nil,
        subjectActorID: String? = nil,
        actingAsActorID: String? = nil,
        correlationID: String? = nil
    ) async throws -> DaemonChatTurnResponse {
        let requestItems = messages.compactMap { message -> DaemonChatTurnRequestItem? in
            let normalizedRole = message.role
                .trimmingCharacters(in: .whitespacesAndNewlines)
                .lowercased()
            guard let rawContent = ChatTextNormalization.nonEmptyPreservingWhitespace(message.content) else {
                return nil
            }
            switch normalizedRole {
            case "assistant":
                return DaemonChatTurnRequestItem(
                    type: "assistant_message",
                    role: "assistant",
                    status: "completed",
                    content: rawContent
                )
            default:
                return DaemonChatTurnRequestItem(
                    type: "user_message",
                    role: "user",
                    status: "completed",
                    content: rawContent
                )
            }
        }

        let trimmedSystemPrompt = ChatTextNormalization.normalizedNonEmpty(systemPrompt)

        return try await request(
            baseURL: baseURL,
            path: "/v1/chat/turn",
            method: "POST",
            authToken: authToken,
            correlationID: correlationID,
            timeoutInterval: 300,
            body: DaemonChatTurnRequest(
                workspaceID: workspaceID,
                taskClass: "chat",
                requestedByActorID: requestedByActorID,
                subjectActorID: subjectActorID,
                actingAsActorID: actingAsActorID,
                provider: nil,
                model: nil,
                systemPrompt: trimmedSystemPrompt,
                channel: DaemonChatTurnChannelContext(channelID: "app"),
                items: requestItems
            )
        )
    }

    func chatTurnExplain(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        taskClass: String = "chat",
        requestedByActorID: String? = nil,
        subjectActorID: String? = nil,
        actingAsActorID: String? = nil
    ) async throws -> DaemonChatTurnExplainResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/chat/turn/explain",
            method: "POST",
            authToken: authToken,
            body: DaemonChatTurnExplainRequest(
                workspaceID: workspaceID,
                taskClass: taskClass,
                requestedByActorID: requestedByActorID,
                subjectActorID: subjectActorID,
                actingAsActorID: actingAsActorID,
                channel: DaemonChatTurnChannelContext(channelID: "app")
            )
        )
    }

    func chatTurnHistory(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        channelID: String? = nil,
        connectorID: String? = nil,
        threadID: String? = nil,
        correlationID: String? = nil,
        beforeCreatedAt: String? = nil,
        beforeItemID: String? = nil,
        limit: Int = 120
    ) async throws -> DaemonChatTurnHistoryResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/chat/history",
            method: "POST",
            authToken: authToken,
            body: DaemonChatTurnHistoryRequest(
                workspaceID: workspaceID,
                channelID: channelID,
                connectorID: connectorID,
                threadID: threadID,
                correlationID: correlationID,
                beforeCreatedAt: beforeCreatedAt,
                beforeItemID: beforeItemID,
                limit: limit
            )
        )
    }

    func chatPersonaPolicyGet(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        principalActorID: String? = nil,
        channelID: String? = nil
    ) async throws -> DaemonChatPersonaPolicyResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/chat/persona/get",
            method: "POST",
            authToken: authToken,
            body: DaemonChatPersonaPolicyRequest(
                workspaceID: workspaceID,
                principalActorID: principalActorID,
                channelID: channelID
            )
        )
    }

    func chatPersonaPolicySet(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        principalActorID: String? = nil,
        channelID: String? = nil,
        stylePrompt: String,
        guardrails: [String]
    ) async throws -> DaemonChatPersonaPolicyResponse {
        try await request(
            baseURL: baseURL,
            path: "/v1/chat/persona/set",
            method: "POST",
            authToken: authToken,
            body: DaemonChatPersonaPolicyUpsertRequest(
                workspaceID: workspaceID,
                principalActorID: principalActorID,
                channelID: channelID,
                stylePrompt: stylePrompt,
                guardrails: guardrails
            )
        )
    }

    private func request<Response: Decodable, RequestBody: Encodable>(
        baseURL: URL,
        path: String,
        method: String,
        authToken: String,
        correlationID: String? = nil,
        timeoutInterval: TimeInterval = 8,
        body: RequestBody?
    ) async throws -> Response {
        try await core.request(
            baseURL: baseURL,
            path: path,
            method: method,
            authToken: authToken,
            correlationID: correlationID,
            timeoutInterval: timeoutInterval,
            body: body
        )
    }
}
