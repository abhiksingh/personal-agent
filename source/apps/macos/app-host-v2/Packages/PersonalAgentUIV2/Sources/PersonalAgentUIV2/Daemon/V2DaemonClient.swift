import Foundation

public struct V2DaemonAPIClient:  Sendable {
    private let core: V2DaemonTransportCore

    public init(session: URLSession = .shared) {
        self.core = V2DaemonTransportCore(session: session)
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

    public func connectRealtime(
        baseURL: URL,
        authToken: String,
        correlationID: String? = nil
    ) throws -> V2DaemonRealtimeSession {
        try core.connectRealtime(baseURL: baseURL, authToken: authToken, correlationID: correlationID)
    }
}

public extension V2DaemonAPIClient {
    var lifecycle: V2DaemonLifecycleAPI { V2DaemonLifecycleAPI(client: self) }
    var models: V2DaemonModelsAPI { V2DaemonModelsAPI(client: self) }
    var connectors: V2DaemonConnectorsAPI { V2DaemonConnectorsAPI(client: self) }
    var approvals: V2DaemonApprovalsAPI { V2DaemonApprovalsAPI(client: self) }
    var tasks: V2DaemonTasksAPI { V2DaemonTasksAPI(client: self) }
    var inspect: V2DaemonInspectAPI { V2DaemonInspectAPI(client: self) }
    var chat: V2DaemonChatAPI { V2DaemonChatAPI(client: self) }
}

struct V2DaemonWorkspaceRequest: Encodable {
    let workspaceID: String

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
    }
}

public struct V2DaemonWorkflowRouteMetadata: Decodable, Sendable, Equatable {
    public let available: Bool
    public let taskClass: String
    public let provider: String?
    public let modelKey: String?
    public let taskClassSource: String
    public let routeSource: String
    public let notes: String?

    enum CodingKeys: String, CodingKey {
        case available
        case taskClass = "task_class"
        case provider
        case modelKey = "model_key"
        case taskClassSource = "task_class_source"
        case routeSource = "route_source"
        case notes
    }

    init(
        available: Bool = false,
        taskClass: String = "",
        provider: String? = nil,
        modelKey: String? = nil,
        taskClassSource: String = "unknown",
        routeSource: String = "unknown",
        notes: String? = nil
    ) {
        self.available = available
        self.taskClass = taskClass
        self.provider = provider
        self.modelKey = modelKey
        self.taskClassSource = taskClassSource
        self.routeSource = routeSource
        self.notes = notes
    }
}
