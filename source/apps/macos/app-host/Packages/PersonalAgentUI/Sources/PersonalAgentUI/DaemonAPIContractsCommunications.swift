import Foundation

struct DaemonCommSendRequest: Encodable {
    let workspaceID: String
    let operationID: String
    let sourceChannel: String
    let threadID: String?
    let connectorID: String?
    let destination: String?
    let message: String
    let stepID: String?
    let eventID: String?
    let iMessageFailures: Int?
    let smsFailures: Int?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case operationID = "operation_id"
        case sourceChannel = "source_channel"
        case threadID = "thread_id"
        case connectorID = "connector_id"
        case destination
        case message
        case stepID = "step_id"
        case eventID = "event_id"
        case iMessageFailures = "imessage_failures"
        case smsFailures = "sms_failures"
    }
}

struct DaemonCommSendResponse: Decodable, Sendable {
    let workspaceID: String
    let operationID: String
    let threadID: String?
    let resolvedSourceChannel: String?
    let resolvedConnectorID: String?
    let resolvedDestination: String?
    let success: Bool
    let result: DaemonJSONValue?
    let error: String?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case operationID = "operation_id"
        case threadID = "thread_id"
        case resolvedSourceChannel = "resolved_source_channel"
        case resolvedConnectorID = "resolved_connector_id"
        case resolvedDestination = "resolved_destination"
        case success
        case result
        case error
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        operationID = try container.decodeIfPresent(String.self, forKey: .operationID) ?? ""
        threadID = try container.decodeIfPresent(String.self, forKey: .threadID)
        resolvedSourceChannel = try container.decodeIfPresent(String.self, forKey: .resolvedSourceChannel)
        resolvedConnectorID = try container.decodeIfPresent(String.self, forKey: .resolvedConnectorID)
        resolvedDestination = try container.decodeIfPresent(String.self, forKey: .resolvedDestination)
        success = try container.decodeIfPresent(Bool.self, forKey: .success) ?? false
        result = try container.decodeIfPresent(DaemonJSONValue.self, forKey: .result)
        error = try container.decodeIfPresent(String.self, forKey: .error)
    }
}

struct DaemonCommThreadListRequest: Encodable {
    let workspaceID: String
    let channel: String?
    let connectorID: String?
    let query: String?
    let cursor: String?
    let limit: Int

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case channel
        case connectorID = "connector_id"
        case query
        case cursor
        case limit
    }
}

struct DaemonCommThreadListRecord: Decodable, Sendable {
    let threadID: String
    let workspaceID: String
    let channel: String
    let connectorID: String?
    let externalRef: String?
    let title: String?
    let lastEventID: String?
    let lastEventType: String?
    let lastDirection: String?
    let lastOccurredAt: String?
    let lastBodyPreview: String?
    let participantAddresses: [String]
    let eventCount: Int
    let createdAt: String
    let updatedAt: String

    enum CodingKeys: String, CodingKey {
        case threadID = "thread_id"
        case workspaceID = "workspace_id"
        case channel
        case connectorID = "connector_id"
        case externalRef = "external_ref"
        case title
        case lastEventID = "last_event_id"
        case lastEventType = "last_event_type"
        case lastDirection = "last_direction"
        case lastOccurredAt = "last_occurred_at"
        case lastBodyPreview = "last_body_preview"
        case participantAddresses = "participant_addresses"
        case eventCount = "event_count"
        case createdAt = "created_at"
        case updatedAt = "updated_at"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        threadID = try container.decodeIfPresent(String.self, forKey: .threadID) ?? ""
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        channel = try container.decodeIfPresent(String.self, forKey: .channel) ?? "unknown"
        connectorID = try container.decodeIfPresent(String.self, forKey: .connectorID)
        externalRef = try container.decodeIfPresent(String.self, forKey: .externalRef)
        title = try container.decodeIfPresent(String.self, forKey: .title)
        lastEventID = try container.decodeIfPresent(String.self, forKey: .lastEventID)
        lastEventType = try container.decodeIfPresent(String.self, forKey: .lastEventType)
        lastDirection = try container.decodeIfPresent(String.self, forKey: .lastDirection)
        lastOccurredAt = try container.decodeIfPresent(String.self, forKey: .lastOccurredAt)
        lastBodyPreview = try container.decodeIfPresent(String.self, forKey: .lastBodyPreview)
        participantAddresses = try container.decodeIfPresent([String].self, forKey: .participantAddresses) ?? []
        eventCount = try container.decodeIfPresent(Int.self, forKey: .eventCount) ?? 0
        createdAt = try container.decodeIfPresent(String.self, forKey: .createdAt) ?? ""
        updatedAt = try container.decodeIfPresent(String.self, forKey: .updatedAt) ?? ""
    }
}

struct DaemonCommThreadListResponse: Decodable, Sendable {
    let workspaceID: String
    let items: [DaemonCommThreadListRecord]
    let hasMore: Bool
    let nextCursor: String?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case items
        case hasMore = "has_more"
        case nextCursor = "next_cursor"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        items = try container.decodeIfPresent([DaemonCommThreadListRecord].self, forKey: .items) ?? []
        hasMore = try container.decodeIfPresent(Bool.self, forKey: .hasMore) ?? false
        nextCursor = try container.decodeIfPresent(String.self, forKey: .nextCursor)
    }
}

struct DaemonCommEventTimelineRequest: Encodable {
    let workspaceID: String
    let threadID: String?
    let channel: String?
    let connectorID: String?
    let eventType: String?
    let direction: String?
    let query: String?
    let cursor: String?
    let limit: Int

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case threadID = "thread_id"
        case channel
        case connectorID = "connector_id"
        case eventType = "event_type"
        case direction
        case query
        case cursor
        case limit
    }
}

struct DaemonCommEventAddressRecord: Decodable, Sendable {
    let role: String
    let value: String
    let display: String?
    let position: Int

    enum CodingKeys: String, CodingKey {
        case role
        case value
        case display
        case position
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        role = try container.decodeIfPresent(String.self, forKey: .role) ?? "unknown"
        value = try container.decodeIfPresent(String.self, forKey: .value) ?? ""
        display = try container.decodeIfPresent(String.self, forKey: .display)
        position = try container.decodeIfPresent(Int.self, forKey: .position) ?? 0
    }
}

struct DaemonCommEventTimelineRecord: Decodable, Sendable {
    let eventID: String
    let workspaceID: String
    let threadID: String
    let channel: String
    let connectorID: String?
    let eventType: String
    let direction: String
    let assistantEmitted: Bool
    let bodyText: String?
    let occurredAt: String
    let createdAt: String
    let addresses: [DaemonCommEventAddressRecord]

    enum CodingKeys: String, CodingKey {
        case eventID = "event_id"
        case workspaceID = "workspace_id"
        case threadID = "thread_id"
        case channel
        case connectorID = "connector_id"
        case eventType = "event_type"
        case direction
        case assistantEmitted = "assistant_emitted"
        case bodyText = "body_text"
        case occurredAt = "occurred_at"
        case createdAt = "created_at"
        case addresses
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        eventID = try container.decodeIfPresent(String.self, forKey: .eventID) ?? ""
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        threadID = try container.decodeIfPresent(String.self, forKey: .threadID) ?? ""
        channel = try container.decodeIfPresent(String.self, forKey: .channel) ?? "unknown"
        connectorID = try container.decodeIfPresent(String.self, forKey: .connectorID)
        eventType = try container.decodeIfPresent(String.self, forKey: .eventType) ?? "unknown"
        direction = try container.decodeIfPresent(String.self, forKey: .direction) ?? "unknown"
        assistantEmitted = try container.decodeIfPresent(Bool.self, forKey: .assistantEmitted) ?? false
        bodyText = try container.decodeIfPresent(String.self, forKey: .bodyText)
        occurredAt = try container.decodeIfPresent(String.self, forKey: .occurredAt) ?? ""
        createdAt = try container.decodeIfPresent(String.self, forKey: .createdAt) ?? ""
        addresses = try container.decodeIfPresent([DaemonCommEventAddressRecord].self, forKey: .addresses) ?? []
    }
}

struct DaemonCommEventTimelineResponse: Decodable, Sendable {
    let workspaceID: String
    let threadID: String?
    let items: [DaemonCommEventTimelineRecord]
    let hasMore: Bool
    let nextCursor: String?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case threadID = "thread_id"
        case items
        case hasMore = "has_more"
        case nextCursor = "next_cursor"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        threadID = try container.decodeIfPresent(String.self, forKey: .threadID)
        items = try container.decodeIfPresent([DaemonCommEventTimelineRecord].self, forKey: .items) ?? []
        hasMore = try container.decodeIfPresent(Bool.self, forKey: .hasMore) ?? false
        nextCursor = try container.decodeIfPresent(String.self, forKey: .nextCursor)
    }
}

struct DaemonCommCallSessionListRequest: Encodable {
    let workspaceID: String
    let threadID: String?
    let provider: String?
    let connectorID: String?
    let direction: String?
    let status: String?
    let providerCallID: String?
    let query: String?
    let cursor: String?
    let limit: Int

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case threadID = "thread_id"
        case provider
        case connectorID = "connector_id"
        case direction
        case status
        case providerCallID = "provider_call_id"
        case query
        case cursor
        case limit
    }
}

struct DaemonCommCallSessionListRecord: Decodable, Sendable {
    let sessionID: String
    let workspaceID: String
    let provider: String
    let connectorID: String?
    let providerCallID: String?
    let threadID: String?
    let direction: String
    let fromAddress: String?
    let toAddress: String?
    let status: String
    let startedAt: String?
    let endedAt: String?
    let updatedAt: String

    enum CodingKeys: String, CodingKey {
        case sessionID = "session_id"
        case workspaceID = "workspace_id"
        case provider
        case connectorID = "connector_id"
        case providerCallID = "provider_call_id"
        case threadID = "thread_id"
        case direction
        case fromAddress = "from_address"
        case toAddress = "to_address"
        case status
        case startedAt = "started_at"
        case endedAt = "ended_at"
        case updatedAt = "updated_at"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        sessionID = try container.decodeIfPresent(String.self, forKey: .sessionID) ?? ""
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        provider = try container.decodeIfPresent(String.self, forKey: .provider) ?? "unknown"
        connectorID = try container.decodeIfPresent(String.self, forKey: .connectorID)
        providerCallID = try container.decodeIfPresent(String.self, forKey: .providerCallID)
        threadID = try container.decodeIfPresent(String.self, forKey: .threadID)
        direction = try container.decodeIfPresent(String.self, forKey: .direction) ?? "unknown"
        fromAddress = try container.decodeIfPresent(String.self, forKey: .fromAddress)
        toAddress = try container.decodeIfPresent(String.self, forKey: .toAddress)
        status = try container.decodeIfPresent(String.self, forKey: .status) ?? "unknown"
        startedAt = try container.decodeIfPresent(String.self, forKey: .startedAt)
        endedAt = try container.decodeIfPresent(String.self, forKey: .endedAt)
        updatedAt = try container.decodeIfPresent(String.self, forKey: .updatedAt) ?? ""
    }
}

struct DaemonCommCallSessionListResponse: Decodable, Sendable {
    let workspaceID: String
    let items: [DaemonCommCallSessionListRecord]
    let hasMore: Bool
    let nextCursor: String?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case items
        case hasMore = "has_more"
        case nextCursor = "next_cursor"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        items = try container.decodeIfPresent([DaemonCommCallSessionListRecord].self, forKey: .items) ?? []
        hasMore = try container.decodeIfPresent(Bool.self, forKey: .hasMore) ?? false
        nextCursor = try container.decodeIfPresent(String.self, forKey: .nextCursor)
    }
}

struct DaemonCommAttemptsRequest: Encodable {
    let workspaceID: String
    let operationID: String?
    let threadID: String?
    let taskID: String?
    let runID: String?
    let stepID: String?
    let channel: String?
    let status: String?
    let cursor: String?
    let limit: Int

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case operationID = "operation_id"
        case threadID = "thread_id"
        case taskID = "task_id"
        case runID = "run_id"
        case stepID = "step_id"
        case channel
        case status
        case cursor
        case limit
    }
}

struct DaemonCommAttemptRecord: Decodable, Sendable {
    let attemptID: String
    let workspaceID: String
    let operationID: String?
    let taskID: String?
    let runID: String?
    let stepID: String?
    let eventID: String?
    let threadID: String?
    let destinationEndpoint: String
    let idempotencyKey: String
    let channel: String
    let routeIndex: Int
    let routePhase: String?
    let retryOrdinal: Int
    let fallbackFromChannel: String?
    let status: String
    let providerReceipt: String?
    let error: String?
    let attemptedAt: String

    enum CodingKeys: String, CodingKey {
        case attemptID = "attempt_id"
        case workspaceID = "workspace_id"
        case operationID = "operation_id"
        case taskID = "task_id"
        case runID = "run_id"
        case stepID = "step_id"
        case eventID = "event_id"
        case threadID = "thread_id"
        case destinationEndpoint = "destination_endpoint"
        case idempotencyKey = "idempotency_key"
        case channel
        case routeIndex = "route_index"
        case routePhase = "route_phase"
        case retryOrdinal = "retry_ordinal"
        case fallbackFromChannel = "fallback_from_channel"
        case status
        case providerReceipt = "provider_receipt"
        case error
        case attemptedAt = "attempted_at"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        attemptID = try container.decodeIfPresent(String.self, forKey: .attemptID) ?? ""
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        operationID = try container.decodeIfPresent(String.self, forKey: .operationID)
        taskID = try container.decodeIfPresent(String.self, forKey: .taskID)
        runID = try container.decodeIfPresent(String.self, forKey: .runID)
        stepID = try container.decodeIfPresent(String.self, forKey: .stepID)
        eventID = try container.decodeIfPresent(String.self, forKey: .eventID)
        threadID = try container.decodeIfPresent(String.self, forKey: .threadID)
        destinationEndpoint = try container.decodeIfPresent(String.self, forKey: .destinationEndpoint) ?? ""
        idempotencyKey = try container.decodeIfPresent(String.self, forKey: .idempotencyKey) ?? ""
        channel = try container.decodeIfPresent(String.self, forKey: .channel) ?? "unknown"
        routeIndex = try container.decodeIfPresent(Int.self, forKey: .routeIndex) ?? 0
        routePhase = try container.decodeIfPresent(String.self, forKey: .routePhase)
        retryOrdinal = try container.decodeIfPresent(Int.self, forKey: .retryOrdinal) ?? 0
        fallbackFromChannel = try container.decodeIfPresent(String.self, forKey: .fallbackFromChannel)
        status = try container.decodeIfPresent(String.self, forKey: .status) ?? "unknown"
        providerReceipt = try container.decodeIfPresent(String.self, forKey: .providerReceipt)
        error = try container.decodeIfPresent(String.self, forKey: .error)
        attemptedAt = try container.decodeIfPresent(String.self, forKey: .attemptedAt) ?? ""
    }
}

struct DaemonCommAttemptsResponse: Decodable, Sendable {
    let workspaceID: String
    let operationID: String?
    let threadID: String?
    let taskID: String?
    let runID: String?
    let stepID: String?
    let hasMore: Bool
    let nextCursor: String?
    let attempts: [DaemonCommAttemptRecord]

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case operationID = "operation_id"
        case threadID = "thread_id"
        case taskID = "task_id"
        case runID = "run_id"
        case stepID = "step_id"
        case hasMore = "has_more"
        case nextCursor = "next_cursor"
        case attempts
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        operationID = try container.decodeIfPresent(String.self, forKey: .operationID)
        threadID = try container.decodeIfPresent(String.self, forKey: .threadID)
        taskID = try container.decodeIfPresent(String.self, forKey: .taskID)
        runID = try container.decodeIfPresent(String.self, forKey: .runID)
        stepID = try container.decodeIfPresent(String.self, forKey: .stepID)
        hasMore = try container.decodeIfPresent(Bool.self, forKey: .hasMore) ?? false
        nextCursor = try container.decodeIfPresent(String.self, forKey: .nextCursor)
        attempts = try container.decodeIfPresent([DaemonCommAttemptRecord].self, forKey: .attempts) ?? []
    }
}

struct DaemonReceiptAuditLinkRecord: Decodable, Sendable {
    let auditID: String
    let eventType: String
    let createdAt: String

    enum CodingKeys: String, CodingKey {
        case auditID = "audit_id"
        case eventType = "event_type"
        case createdAt = "created_at"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        auditID = try container.decodeIfPresent(String.self, forKey: .auditID) ?? ""
        eventType = try container.decodeIfPresent(String.self, forKey: .eventType) ?? "unknown"
        createdAt = try container.decodeIfPresent(String.self, forKey: .createdAt) ?? ""
    }
}

struct DaemonCommWebhookReceiptListRequest: Encodable {
    let workspaceID: String
    let provider: String?
    let providerEventID: String?
    let providerEventQuery: String?
    let eventID: String?
    let cursorCreatedAt: String?
    let cursorID: String?
    let limit: Int

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case provider
        case providerEventID = "provider_event_id"
        case providerEventQuery = "provider_event_query"
        case eventID = "event_id"
        case cursorCreatedAt = "cursor_created_at"
        case cursorID = "cursor_id"
        case limit
    }
}

struct DaemonCommWebhookReceiptItem: Decodable, Sendable {
    let receiptID: String
    let workspaceID: String
    let provider: String
    let providerEventID: String
    let trustState: String
    let signatureValid: Bool
    let signatureValuePresent: Bool
    let payloadHash: String
    let eventID: String?
    let threadID: String?
    let receivedAt: String?
    let createdAt: String
    let auditLinks: [DaemonReceiptAuditLinkRecord]

    enum CodingKeys: String, CodingKey {
        case receiptID = "receipt_id"
        case workspaceID = "workspace_id"
        case provider
        case providerEventID = "provider_event_id"
        case trustState = "trust_state"
        case signatureValid = "signature_valid"
        case signatureValuePresent = "signature_value_present"
        case payloadHash = "payload_hash"
        case eventID = "event_id"
        case threadID = "thread_id"
        case receivedAt = "received_at"
        case createdAt = "created_at"
        case auditLinks = "audit_links"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        receiptID = try container.decodeIfPresent(String.self, forKey: .receiptID) ?? ""
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        provider = try container.decodeIfPresent(String.self, forKey: .provider) ?? "unknown"
        providerEventID = try container.decodeIfPresent(String.self, forKey: .providerEventID) ?? ""
        trustState = try container.decodeIfPresent(String.self, forKey: .trustState) ?? "unknown"
        signatureValid = try container.decodeIfPresent(Bool.self, forKey: .signatureValid) ?? false
        signatureValuePresent = try container.decodeIfPresent(Bool.self, forKey: .signatureValuePresent) ?? false
        payloadHash = try container.decodeIfPresent(String.self, forKey: .payloadHash) ?? ""
        eventID = try container.decodeIfPresent(String.self, forKey: .eventID)
        threadID = try container.decodeIfPresent(String.self, forKey: .threadID)
        receivedAt = try container.decodeIfPresent(String.self, forKey: .receivedAt)
        createdAt = try container.decodeIfPresent(String.self, forKey: .createdAt) ?? ""
        auditLinks = try container.decodeIfPresent([DaemonReceiptAuditLinkRecord].self, forKey: .auditLinks) ?? []
    }
}

struct DaemonCommWebhookReceiptListResponse: Decodable, Sendable {
    let workspaceID: String
    let provider: String?
    let items: [DaemonCommWebhookReceiptItem]
    let hasMore: Bool
    let nextCursorCreatedAt: String?
    let nextCursorID: String?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case provider
        case items
        case hasMore = "has_more"
        case nextCursorCreatedAt = "next_cursor_created_at"
        case nextCursorID = "next_cursor_id"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        provider = try container.decodeIfPresent(String.self, forKey: .provider)
        items = try container.decodeIfPresent([DaemonCommWebhookReceiptItem].self, forKey: .items) ?? []
        hasMore = try container.decodeIfPresent(Bool.self, forKey: .hasMore) ?? false
        nextCursorCreatedAt = try container.decodeIfPresent(String.self, forKey: .nextCursorCreatedAt)
        nextCursorID = try container.decodeIfPresent(String.self, forKey: .nextCursorID)
    }
}

struct DaemonCommIngestReceiptListRequest: Encodable {
    let workspaceID: String
    let source: String?
    let sourceScope: String?
    let sourceEventID: String?
    let sourceEventQuery: String?
    let trustState: String?
    let eventID: String?
    let cursorCreatedAt: String?
    let cursorID: String?
    let limit: Int

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case source
        case sourceScope = "source_scope"
        case sourceEventID = "source_event_id"
        case sourceEventQuery = "source_event_query"
        case trustState = "trust_state"
        case eventID = "event_id"
        case cursorCreatedAt = "cursor_created_at"
        case cursorID = "cursor_id"
        case limit
    }
}

struct DaemonCommIngestReceiptItem: Decodable, Sendable {
    let receiptID: String
    let workspaceID: String
    let source: String
    let sourceScope: String
    let sourceEventID: String
    let sourceCursor: String?
    let trustState: String
    let payloadHash: String
    let eventID: String?
    let threadID: String?
    let receivedAt: String?
    let createdAt: String
    let auditLinks: [DaemonReceiptAuditLinkRecord]

    enum CodingKeys: String, CodingKey {
        case receiptID = "receipt_id"
        case workspaceID = "workspace_id"
        case source
        case sourceScope = "source_scope"
        case sourceEventID = "source_event_id"
        case sourceCursor = "source_cursor"
        case trustState = "trust_state"
        case payloadHash = "payload_hash"
        case eventID = "event_id"
        case threadID = "thread_id"
        case receivedAt = "received_at"
        case createdAt = "created_at"
        case auditLinks = "audit_links"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        receiptID = try container.decodeIfPresent(String.self, forKey: .receiptID) ?? ""
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        source = try container.decodeIfPresent(String.self, forKey: .source) ?? "unknown"
        sourceScope = try container.decodeIfPresent(String.self, forKey: .sourceScope) ?? "unknown"
        sourceEventID = try container.decodeIfPresent(String.self, forKey: .sourceEventID) ?? ""
        sourceCursor = try container.decodeIfPresent(String.self, forKey: .sourceCursor)
        trustState = try container.decodeIfPresent(String.self, forKey: .trustState) ?? "unknown"
        payloadHash = try container.decodeIfPresent(String.self, forKey: .payloadHash) ?? ""
        eventID = try container.decodeIfPresent(String.self, forKey: .eventID)
        threadID = try container.decodeIfPresent(String.self, forKey: .threadID)
        receivedAt = try container.decodeIfPresent(String.self, forKey: .receivedAt)
        createdAt = try container.decodeIfPresent(String.self, forKey: .createdAt) ?? ""
        auditLinks = try container.decodeIfPresent([DaemonReceiptAuditLinkRecord].self, forKey: .auditLinks) ?? []
    }
}

struct DaemonCommIngestReceiptListResponse: Decodable, Sendable {
    let workspaceID: String
    let source: String?
    let sourceScope: String?
    let items: [DaemonCommIngestReceiptItem]
    let hasMore: Bool
    let nextCursorCreatedAt: String?
    let nextCursorID: String?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case source
        case sourceScope = "source_scope"
        case items
        case hasMore = "has_more"
        case nextCursorCreatedAt = "next_cursor_created_at"
        case nextCursorID = "next_cursor_id"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        source = try container.decodeIfPresent(String.self, forKey: .source)
        sourceScope = try container.decodeIfPresent(String.self, forKey: .sourceScope)
        items = try container.decodeIfPresent([DaemonCommIngestReceiptItem].self, forKey: .items) ?? []
        hasMore = try container.decodeIfPresent(Bool.self, forKey: .hasMore) ?? false
        nextCursorCreatedAt = try container.decodeIfPresent(String.self, forKey: .nextCursorCreatedAt)
        nextCursorID = try container.decodeIfPresent(String.self, forKey: .nextCursorID)
    }
}

struct DaemonCommPolicySetRequest: Encodable {
    let policyID: String?
    let workspaceID: String
    let sourceChannel: String
    let endpointPattern: String?
    let primaryChannel: String
    let retryCount: Int
    let fallbackChannels: [String]
    let isDefault: Bool

    enum CodingKeys: String, CodingKey {
        case policyID = "policy_id"
        case workspaceID = "workspace_id"
        case sourceChannel = "source_channel"
        case endpointPattern = "endpoint_pattern"
        case primaryChannel = "primary_channel"
        case retryCount = "retry_count"
        case fallbackChannels = "fallback_channels"
        case isDefault = "is_default"
    }
}

struct DaemonCommPolicyRecord: Decodable, Sendable {
    let id: String
    let workspaceID: String
    let sourceChannel: String
    let endpointPattern: String?
    let isDefault: Bool
    let policy: DaemonChannelDeliveryPolicy
    let createdAt: String
    let updatedAt: String

    enum CodingKeys: String, CodingKey {
        case id
        case workspaceID = "workspace_id"
        case sourceChannel = "source_channel"
        case endpointPattern = "endpoint_pattern"
        case isDefault = "is_default"
        case policy
        case createdAt = "created_at"
        case updatedAt = "updated_at"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        id = try container.decodeIfPresent(String.self, forKey: .id) ?? ""
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        sourceChannel = try container.decodeIfPresent(String.self, forKey: .sourceChannel) ?? ""
        endpointPattern = try container.decodeIfPresent(String.self, forKey: .endpointPattern)
        isDefault = try container.decodeIfPresent(Bool.self, forKey: .isDefault) ?? false
        policy = try container.decodeIfPresent(DaemonChannelDeliveryPolicy.self, forKey: .policy)
            ?? DaemonChannelDeliveryPolicy(primaryChannel: "", retryCount: 0, fallbackChannels: [])
        createdAt = try container.decodeIfPresent(String.self, forKey: .createdAt) ?? ""
        updatedAt = try container.decodeIfPresent(String.self, forKey: .updatedAt) ?? ""
    }
}

struct DaemonCommPolicyListRequest: Encodable {
    let workspaceID: String
    let sourceChannel: String?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case sourceChannel = "source_channel"
    }
}

struct DaemonCommPolicyListResponse: Decodable, Sendable {
    let workspaceID: String
    let policies: [DaemonCommPolicyRecord]

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case policies
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        policies = try container.decodeIfPresent([DaemonCommPolicyRecord].self, forKey: .policies) ?? []
    }
}
