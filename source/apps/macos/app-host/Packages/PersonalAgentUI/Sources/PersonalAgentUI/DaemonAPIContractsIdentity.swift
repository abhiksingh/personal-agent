import Foundation

struct DaemonDelegationListRequest: Encodable {
    let workspaceID: String
    let fromActorID: String?
    let toActorID: String?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case fromActorID = "from_actor_id"
        case toActorID = "to_actor_id"
    }
}

struct DaemonDelegationGrantRequest: Encodable {
    let workspaceID: String
    let fromActorID: String
    let toActorID: String
    let scopeType: String
    let scopeKey: String?
    let expiresAt: String?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case fromActorID = "from_actor_id"
        case toActorID = "to_actor_id"
        case scopeType = "scope_type"
        case scopeKey = "scope_key"
        case expiresAt = "expires_at"
    }
}

struct DaemonDelegationRevokeRequest: Encodable {
    let workspaceID: String
    let ruleID: String

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case ruleID = "rule_id"
    }
}

struct DaemonDelegationRuleRecord: Decodable, Sendable {
    let id: String
    let workspaceID: String
    let fromActorID: String
    let toActorID: String
    let scopeType: String
    let scopeKey: String?
    let status: String
    let createdAt: String
    let expiresAt: String?

    enum CodingKeys: String, CodingKey {
        case id
        case workspaceID = "workspace_id"
        case fromActorID = "from_actor_id"
        case toActorID = "to_actor_id"
        case scopeType = "scope_type"
        case scopeKey = "scope_key"
        case status
        case createdAt = "created_at"
        case expiresAt = "expires_at"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        id = try container.decodeIfPresent(String.self, forKey: .id) ?? ""
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        fromActorID = try container.decodeIfPresent(String.self, forKey: .fromActorID) ?? ""
        toActorID = try container.decodeIfPresent(String.self, forKey: .toActorID) ?? ""
        scopeType = try container.decodeIfPresent(String.self, forKey: .scopeType) ?? "EXECUTION"
        scopeKey = try container.decodeIfPresent(String.self, forKey: .scopeKey)
        status = try container.decodeIfPresent(String.self, forKey: .status) ?? "ACTIVE"
        createdAt = try container.decodeIfPresent(String.self, forKey: .createdAt) ?? ""
        expiresAt = try container.decodeIfPresent(String.self, forKey: .expiresAt)
    }
}

struct DaemonDelegationListResponse: Decodable, Sendable {
    let workspaceID: String
    let rules: [DaemonDelegationRuleRecord]

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case rules
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        rules = try container.decodeIfPresent([DaemonDelegationRuleRecord].self, forKey: .rules) ?? []
    }
}

struct DaemonDelegationRevokeResponse: Decodable, Sendable {
    let workspaceID: String
    let ruleID: String
    let status: String

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case ruleID = "rule_id"
        case status
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        ruleID = try container.decodeIfPresent(String.self, forKey: .ruleID) ?? ""
        status = try container.decodeIfPresent(String.self, forKey: .status) ?? "revoked"
    }
}

struct DaemonCapabilityGrantUpsertRequest: Encodable {
    let workspaceID: String
    let grantID: String?
    let actorID: String?
    let capabilityKey: String?
    let scopeJSON: String?
    let status: String?
    let expiresAt: String?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case grantID = "grant_id"
        case actorID = "actor_id"
        case capabilityKey = "capability_key"
        case scopeJSON = "scope_json"
        case status
        case expiresAt = "expires_at"
    }
}

struct DaemonCapabilityGrantListRequest: Encodable {
    let workspaceID: String
    let actorID: String?
    let capabilityKey: String?
    let status: String?
    let cursorCreatedAt: String?
    let cursorID: String?
    let limit: Int

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case actorID = "actor_id"
        case capabilityKey = "capability_key"
        case status
        case cursorCreatedAt = "cursor_created_at"
        case cursorID = "cursor_id"
        case limit
    }
}

struct DaemonCapabilityGrantRecord: Decodable, Sendable {
    let grantID: String
    let workspaceID: String
    let actorID: String
    let capabilityKey: String
    let scopeJSON: String
    let status: String
    let createdAt: String
    let expiresAt: String?

    enum CodingKeys: String, CodingKey {
        case grantID = "grant_id"
        case workspaceID = "workspace_id"
        case actorID = "actor_id"
        case capabilityKey = "capability_key"
        case scopeJSON = "scope_json"
        case status
        case createdAt = "created_at"
        case expiresAt = "expires_at"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        grantID = try container.decodeIfPresent(String.self, forKey: .grantID) ?? ""
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        actorID = try container.decodeIfPresent(String.self, forKey: .actorID) ?? ""
        capabilityKey = try container.decodeIfPresent(String.self, forKey: .capabilityKey) ?? ""
        scopeJSON = try container.decodeIfPresent(String.self, forKey: .scopeJSON) ?? "{}"
        status = try container.decodeIfPresent(String.self, forKey: .status) ?? "active"
        createdAt = try container.decodeIfPresent(String.self, forKey: .createdAt) ?? ""
        expiresAt = try container.decodeIfPresent(String.self, forKey: .expiresAt)
    }
}

struct DaemonCapabilityGrantListResponse: Decodable, Sendable {
    let workspaceID: String
    let items: [DaemonCapabilityGrantRecord]
    let hasMore: Bool
    let nextCursorCreatedAt: String?
    let nextCursorID: String?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case items
        case hasMore = "has_more"
        case nextCursorCreatedAt = "next_cursor_created_at"
        case nextCursorID = "next_cursor_id"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        items = try container.decodeIfPresent([DaemonCapabilityGrantRecord].self, forKey: .items) ?? []
        hasMore = try container.decodeIfPresent(Bool.self, forKey: .hasMore) ?? false
        nextCursorCreatedAt = try container.decodeIfPresent(String.self, forKey: .nextCursorCreatedAt)
        nextCursorID = try container.decodeIfPresent(String.self, forKey: .nextCursorID)
    }
}

struct DaemonIdentityWorkspacesRequest: Encodable {
    let includeInactive: Bool

    enum CodingKeys: String, CodingKey {
        case includeInactive = "include_inactive"
    }
}

struct DaemonIdentityPrincipalsRequest: Encodable {
    let workspaceID: String
    let includeInactive: Bool

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case includeInactive = "include_inactive"
    }
}

struct DaemonIdentityActiveContextRequest: Encodable {
    let workspaceID: String?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
    }
}

struct DaemonIdentitySelectWorkspaceRequest: Encodable {
    let workspaceID: String
    let principalActorID: String?
    let source: String?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case principalActorID = "principal_actor_id"
        case source
    }
}

struct DaemonIdentityActiveContext: Decodable, Sendable {
    let workspaceID: String
    let principalActorID: String?
    let workspaceSource: String?
    let principalSource: String?
    let lastUpdatedAt: String?
    let workspaceResolved: Bool

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case principalActorID = "principal_actor_id"
        case workspaceSource = "workspace_source"
        case principalSource = "principal_source"
        case lastUpdatedAt = "last_updated_at"
        case workspaceResolved = "workspace_resolved"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        principalActorID = try container.decodeIfPresent(String.self, forKey: .principalActorID)
        workspaceSource = try container.decodeIfPresent(String.self, forKey: .workspaceSource)
        principalSource = try container.decodeIfPresent(String.self, forKey: .principalSource)
        lastUpdatedAt = try container.decodeIfPresent(String.self, forKey: .lastUpdatedAt)
        workspaceResolved = try container.decodeIfPresent(Bool.self, forKey: .workspaceResolved) ?? false
    }
}

struct DaemonIdentityWorkspaceRecord: Decodable, Sendable {
    let workspaceID: String
    let name: String
    let status: String
    let principalCount: Int
    let actorCount: Int
    let handleCount: Int
    let updatedAt: String
    let isActive: Bool

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case name
        case status
        case principalCount = "principal_count"
        case actorCount = "actor_count"
        case handleCount = "handle_count"
        case updatedAt = "updated_at"
        case isActive = "is_active"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        name = try container.decodeIfPresent(String.self, forKey: .name) ?? workspaceID
        status = try container.decodeIfPresent(String.self, forKey: .status) ?? "UNKNOWN"
        principalCount = try container.decodeIfPresent(Int.self, forKey: .principalCount) ?? 0
        actorCount = try container.decodeIfPresent(Int.self, forKey: .actorCount) ?? 0
        handleCount = try container.decodeIfPresent(Int.self, forKey: .handleCount) ?? 0
        updatedAt = try container.decodeIfPresent(String.self, forKey: .updatedAt) ?? ""
        isActive = try container.decodeIfPresent(Bool.self, forKey: .isActive) ?? false
    }
}

struct DaemonIdentityWorkspacesResponse: Decodable, Sendable {
    let activeContext: DaemonIdentityActiveContext?
    let workspaces: [DaemonIdentityWorkspaceRecord]

    enum CodingKeys: String, CodingKey {
        case activeContext = "active_context"
        case workspaces
    }
}

struct DaemonIdentityActorHandleRecord: Decodable, Sendable {
    let channel: String
    let handleValue: String
    let isPrimary: Bool
    let updatedAt: String

    enum CodingKeys: String, CodingKey {
        case channel
        case handleValue = "handle_value"
        case isPrimary = "is_primary"
        case updatedAt = "updated_at"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        channel = try container.decodeIfPresent(String.self, forKey: .channel) ?? ""
        handleValue = try container.decodeIfPresent(String.self, forKey: .handleValue) ?? ""
        isPrimary = try container.decodeIfPresent(Bool.self, forKey: .isPrimary) ?? false
        updatedAt = try container.decodeIfPresent(String.self, forKey: .updatedAt) ?? ""
    }
}

struct DaemonIdentityPrincipalRecord: Decodable, Sendable {
    let actorID: String
    let displayName: String
    let actorType: String
    let actorStatus: String
    let principalStatus: String
    let handles: [DaemonIdentityActorHandleRecord]
    let isActive: Bool

    enum CodingKeys: String, CodingKey {
        case actorID = "actor_id"
        case displayName = "display_name"
        case actorType = "actor_type"
        case actorStatus = "actor_status"
        case principalStatus = "principal_status"
        case handles
        case isActive = "is_active"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        actorID = try container.decodeIfPresent(String.self, forKey: .actorID) ?? ""
        displayName = try container.decodeIfPresent(String.self, forKey: .displayName) ?? actorID
        actorType = try container.decodeIfPresent(String.self, forKey: .actorType) ?? "unknown"
        actorStatus = try container.decodeIfPresent(String.self, forKey: .actorStatus) ?? "UNKNOWN"
        principalStatus = try container.decodeIfPresent(String.self, forKey: .principalStatus) ?? "UNKNOWN"
        handles = try container.decodeIfPresent([DaemonIdentityActorHandleRecord].self, forKey: .handles) ?? []
        isActive = try container.decodeIfPresent(Bool.self, forKey: .isActive) ?? false
    }
}

struct DaemonIdentityPrincipalsResponse: Decodable, Sendable {
    let workspaceID: String
    let activeContext: DaemonIdentityActiveContext?
    let principals: [DaemonIdentityPrincipalRecord]

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case activeContext = "active_context"
        case principals
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        activeContext = try container.decodeIfPresent(DaemonIdentityActiveContext.self, forKey: .activeContext)
        principals = try container.decodeIfPresent([DaemonIdentityPrincipalRecord].self, forKey: .principals) ?? []
    }
}

struct DaemonIdentityActiveContextResponse: Decodable, Sendable {
    let activeContext: DaemonIdentityActiveContext?

    enum CodingKeys: String, CodingKey {
        case activeContext = "active_context"
    }
}

struct DaemonIdentityDeviceListRequest: Encodable {
    let workspaceID: String
    let userID: String?
    let deviceType: String?
    let platform: String?
    let cursorCreatedAt: String?
    let cursorID: String?
    let limit: Int?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case userID = "user_id"
        case deviceType = "device_type"
        case platform
        case cursorCreatedAt = "cursor_created_at"
        case cursorID = "cursor_id"
        case limit
    }
}

struct DaemonIdentityDeviceRecord: Decodable, Sendable {
    let deviceID: String
    let workspaceID: String
    let userID: String
    let deviceType: String
    let platform: String
    let label: String?
    let lastSeenAt: String?
    let createdAt: String
    let sessionTotal: Int
    let sessionActiveCount: Int
    let sessionExpiredCount: Int
    let sessionRevokedCount: Int
    let sessionLatestStartedAt: String?

    enum CodingKeys: String, CodingKey {
        case deviceID = "device_id"
        case workspaceID = "workspace_id"
        case userID = "user_id"
        case deviceType = "device_type"
        case platform
        case label
        case lastSeenAt = "last_seen_at"
        case createdAt = "created_at"
        case sessionTotal = "session_total"
        case sessionActiveCount = "session_active_count"
        case sessionExpiredCount = "session_expired_count"
        case sessionRevokedCount = "session_revoked_count"
        case sessionLatestStartedAt = "session_latest_started_at"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        deviceID = try container.decodeIfPresent(String.self, forKey: .deviceID) ?? ""
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        userID = try container.decodeIfPresent(String.self, forKey: .userID) ?? ""
        deviceType = try container.decodeIfPresent(String.self, forKey: .deviceType) ?? ""
        platform = try container.decodeIfPresent(String.self, forKey: .platform) ?? ""
        label = try container.decodeIfPresent(String.self, forKey: .label)
        lastSeenAt = try container.decodeIfPresent(String.self, forKey: .lastSeenAt)
        createdAt = try container.decodeIfPresent(String.self, forKey: .createdAt) ?? ""
        sessionTotal = try container.decodeIfPresent(Int.self, forKey: .sessionTotal) ?? 0
        sessionActiveCount = try container.decodeIfPresent(Int.self, forKey: .sessionActiveCount) ?? 0
        sessionExpiredCount = try container.decodeIfPresent(Int.self, forKey: .sessionExpiredCount) ?? 0
        sessionRevokedCount = try container.decodeIfPresent(Int.self, forKey: .sessionRevokedCount) ?? 0
        sessionLatestStartedAt = try container.decodeIfPresent(String.self, forKey: .sessionLatestStartedAt)
    }
}

struct DaemonIdentityDeviceListResponse: Decodable, Sendable {
    let workspaceID: String
    let userID: String?
    let deviceType: String?
    let platform: String?
    let items: [DaemonIdentityDeviceRecord]
    let hasMore: Bool
    let nextCursorCreatedAt: String?
    let nextCursorID: String?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case userID = "user_id"
        case deviceType = "device_type"
        case platform
        case items
        case hasMore = "has_more"
        case nextCursorCreatedAt = "next_cursor_created_at"
        case nextCursorID = "next_cursor_id"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        userID = try container.decodeIfPresent(String.self, forKey: .userID)
        deviceType = try container.decodeIfPresent(String.self, forKey: .deviceType)
        platform = try container.decodeIfPresent(String.self, forKey: .platform)
        items = try container.decodeIfPresent([DaemonIdentityDeviceRecord].self, forKey: .items) ?? []
        hasMore = try container.decodeIfPresent(Bool.self, forKey: .hasMore) ?? false
        nextCursorCreatedAt = try container.decodeIfPresent(String.self, forKey: .nextCursorCreatedAt)
        nextCursorID = try container.decodeIfPresent(String.self, forKey: .nextCursorID)
    }
}

struct DaemonIdentitySessionListRequest: Encodable {
    let workspaceID: String
    let deviceID: String?
    let userID: String?
    let sessionHealth: String?
    let cursorStartedAt: String?
    let cursorID: String?
    let limit: Int?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case deviceID = "device_id"
        case userID = "user_id"
        case sessionHealth = "session_health"
        case cursorStartedAt = "cursor_started_at"
        case cursorID = "cursor_id"
        case limit
    }
}

struct DaemonIdentitySessionRecord: Decodable, Sendable {
    let sessionID: String
    let workspaceID: String
    let deviceID: String
    let userID: String
    let deviceType: String
    let platform: String
    let deviceLabel: String?
    let deviceLastSeenAt: String?
    let startedAt: String
    let expiresAt: String
    let revokedAt: String?
    let sessionHealth: String

    enum CodingKeys: String, CodingKey {
        case sessionID = "session_id"
        case workspaceID = "workspace_id"
        case deviceID = "device_id"
        case userID = "user_id"
        case deviceType = "device_type"
        case platform
        case deviceLabel = "device_label"
        case deviceLastSeenAt = "device_last_seen_at"
        case startedAt = "started_at"
        case expiresAt = "expires_at"
        case revokedAt = "revoked_at"
        case sessionHealth = "session_health"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        sessionID = try container.decodeIfPresent(String.self, forKey: .sessionID) ?? ""
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        deviceID = try container.decodeIfPresent(String.self, forKey: .deviceID) ?? ""
        userID = try container.decodeIfPresent(String.self, forKey: .userID) ?? ""
        deviceType = try container.decodeIfPresent(String.self, forKey: .deviceType) ?? ""
        platform = try container.decodeIfPresent(String.self, forKey: .platform) ?? ""
        deviceLabel = try container.decodeIfPresent(String.self, forKey: .deviceLabel)
        deviceLastSeenAt = try container.decodeIfPresent(String.self, forKey: .deviceLastSeenAt)
        startedAt = try container.decodeIfPresent(String.self, forKey: .startedAt) ?? ""
        expiresAt = try container.decodeIfPresent(String.self, forKey: .expiresAt) ?? ""
        revokedAt = try container.decodeIfPresent(String.self, forKey: .revokedAt)
        sessionHealth = try container.decodeIfPresent(String.self, forKey: .sessionHealth) ?? "unknown"
    }
}

struct DaemonIdentitySessionListResponse: Decodable, Sendable {
    let workspaceID: String
    let deviceID: String?
    let userID: String?
    let sessionHealth: String?
    let items: [DaemonIdentitySessionRecord]
    let hasMore: Bool
    let nextCursorStartedAt: String?
    let nextCursorID: String?

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case deviceID = "device_id"
        case userID = "user_id"
        case sessionHealth = "session_health"
        case items
        case hasMore = "has_more"
        case nextCursorStartedAt = "next_cursor_started_at"
        case nextCursorID = "next_cursor_id"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        deviceID = try container.decodeIfPresent(String.self, forKey: .deviceID)
        userID = try container.decodeIfPresent(String.self, forKey: .userID)
        sessionHealth = try container.decodeIfPresent(String.self, forKey: .sessionHealth)
        items = try container.decodeIfPresent([DaemonIdentitySessionRecord].self, forKey: .items) ?? []
        hasMore = try container.decodeIfPresent(Bool.self, forKey: .hasMore) ?? false
        nextCursorStartedAt = try container.decodeIfPresent(String.self, forKey: .nextCursorStartedAt)
        nextCursorID = try container.decodeIfPresent(String.self, forKey: .nextCursorID)
    }
}

struct DaemonIdentitySessionRevokeRequest: Encodable {
    let workspaceID: String
    let sessionID: String

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case sessionID = "session_id"
    }
}

struct DaemonIdentitySessionRevokeResponse: Decodable, Sendable {
    let workspaceID: String
    let sessionID: String
    let deviceID: String
    let startedAt: String
    let expiresAt: String
    let revokedAt: String
    let deviceLastSeenAt: String?
    let sessionHealth: String
    let idempotent: Bool

    enum CodingKeys: String, CodingKey {
        case workspaceID = "workspace_id"
        case sessionID = "session_id"
        case deviceID = "device_id"
        case startedAt = "started_at"
        case expiresAt = "expires_at"
        case revokedAt = "revoked_at"
        case deviceLastSeenAt = "device_last_seen_at"
        case sessionHealth = "session_health"
        case idempotent
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        workspaceID = try container.decodeIfPresent(String.self, forKey: .workspaceID) ?? ""
        sessionID = try container.decodeIfPresent(String.self, forKey: .sessionID) ?? ""
        deviceID = try container.decodeIfPresent(String.self, forKey: .deviceID) ?? ""
        startedAt = try container.decodeIfPresent(String.self, forKey: .startedAt) ?? ""
        expiresAt = try container.decodeIfPresent(String.self, forKey: .expiresAt) ?? ""
        revokedAt = try container.decodeIfPresent(String.self, forKey: .revokedAt) ?? ""
        deviceLastSeenAt = try container.decodeIfPresent(String.self, forKey: .deviceLastSeenAt)
        sessionHealth = try container.decodeIfPresent(String.self, forKey: .sessionHealth) ?? "unknown"
        idempotent = try container.decodeIfPresent(Bool.self, forKey: .idempotent) ?? false
    }
}
