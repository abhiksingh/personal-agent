package transport

type IdentityActiveContext struct {
	WorkspaceID       string `json:"workspace_id"`
	PrincipalActorID  string `json:"principal_actor_id,omitempty"`
	WorkspaceSource   string `json:"workspace_source,omitempty"`
	PrincipalSource   string `json:"principal_source,omitempty"`
	LastUpdatedAt     string `json:"last_updated_at,omitempty"`
	WorkspaceResolved bool   `json:"workspace_resolved"`
	MutationSource    string `json:"mutation_source,omitempty"`
	MutationReason    string `json:"mutation_reason,omitempty"`
	SelectionVersion  int64  `json:"selection_version,omitempty"`
}

type IdentityWorkspacesRequest struct {
	IncludeInactive bool `json:"include_inactive"`
}

type IdentityWorkspaceRecord struct {
	WorkspaceID    string `json:"workspace_id"`
	Name           string `json:"name"`
	Status         string `json:"status"`
	PrincipalCount int    `json:"principal_count"`
	ActorCount     int    `json:"actor_count"`
	HandleCount    int    `json:"handle_count"`
	UpdatedAt      string `json:"updated_at"`
	IsActive       bool   `json:"is_active"`
}

type IdentityWorkspacesResponse struct {
	ActiveContext IdentityActiveContext     `json:"active_context"`
	Workspaces    []IdentityWorkspaceRecord `json:"workspaces"`
}

type IdentityPrincipalsRequest struct {
	WorkspaceID     string `json:"workspace_id"`
	IncludeInactive bool   `json:"include_inactive"`
}

type IdentityActorHandleRecord struct {
	Channel     string `json:"channel"`
	HandleValue string `json:"handle_value"`
	IsPrimary   bool   `json:"is_primary"`
	UpdatedAt   string `json:"updated_at"`
}

type IdentityPrincipalRecord struct {
	ActorID         string                      `json:"actor_id"`
	DisplayName     string                      `json:"display_name"`
	ActorType       string                      `json:"actor_type"`
	ActorStatus     string                      `json:"actor_status"`
	PrincipalStatus string                      `json:"principal_status"`
	Handles         []IdentityActorHandleRecord `json:"handles"`
	IsActive        bool                        `json:"is_active"`
}

type IdentityPrincipalsResponse struct {
	WorkspaceID   string                    `json:"workspace_id"`
	ActiveContext IdentityActiveContext     `json:"active_context"`
	Principals    []IdentityPrincipalRecord `json:"principals"`
}

type IdentityActiveContextRequest struct {
	WorkspaceID string `json:"workspace_id,omitempty"`
}

type IdentityActiveContextResponse struct {
	ActiveContext IdentityActiveContext `json:"active_context"`
}

type IdentityWorkspaceSelectRequest struct {
	WorkspaceID      string `json:"workspace_id"`
	PrincipalActorID string `json:"principal_actor_id,omitempty"`
	Source           string `json:"source,omitempty"`
}

type IdentityBootstrapHandle struct {
	Channel     string `json:"channel"`
	HandleValue string `json:"handle_value"`
	IsPrimary   bool   `json:"is_primary"`
}

type IdentityBootstrapRequest struct {
	WorkspaceID          string                   `json:"workspace_id"`
	WorkspaceName        string                   `json:"workspace_name,omitempty"`
	WorkspaceStatus      string                   `json:"workspace_status,omitempty"`
	PrincipalActorID     string                   `json:"principal_actor_id"`
	PrincipalDisplayName string                   `json:"principal_display_name,omitempty"`
	PrincipalActorType   string                   `json:"principal_actor_type,omitempty"`
	PrincipalStatus      string                   `json:"principal_status,omitempty"`
	Handle               *IdentityBootstrapHandle `json:"handle,omitempty"`
	Source               string                   `json:"source,omitempty"`
}

type IdentityBootstrapResponse struct {
	WorkspaceID      string                     `json:"workspace_id"`
	PrincipalActorID string                     `json:"principal_actor_id"`
	WorkspaceCreated bool                       `json:"workspace_created"`
	WorkspaceUpdated bool                       `json:"workspace_updated"`
	PrincipalCreated bool                       `json:"principal_created"`
	PrincipalUpdated bool                       `json:"principal_updated"`
	PrincipalLinked  bool                       `json:"principal_linked"`
	HandleCreated    bool                       `json:"handle_created"`
	HandleUpdated    bool                       `json:"handle_updated"`
	Idempotent       bool                       `json:"idempotent"`
	AuditLogID       string                     `json:"audit_log_id,omitempty"`
	Handle           *IdentityActorHandleRecord `json:"handle,omitempty"`
	ActiveContext    IdentityActiveContext      `json:"active_context"`
}

type IdentityDeviceListRequest struct {
	WorkspaceID     string `json:"workspace_id"`
	UserID          string `json:"user_id,omitempty"`
	DeviceType      string `json:"device_type,omitempty"`
	Platform        string `json:"platform,omitempty"`
	CursorCreatedAt string `json:"cursor_created_at,omitempty"`
	CursorID        string `json:"cursor_id,omitempty"`
	Limit           int    `json:"limit,omitempty"`
}

type IdentityDeviceRecord struct {
	DeviceID               string `json:"device_id"`
	WorkspaceID            string `json:"workspace_id"`
	UserID                 string `json:"user_id"`
	DeviceType             string `json:"device_type"`
	Platform               string `json:"platform"`
	Label                  string `json:"label,omitempty"`
	LastSeenAt             string `json:"last_seen_at,omitempty"`
	CreatedAt              string `json:"created_at"`
	SessionTotal           int    `json:"session_total"`
	SessionActiveCount     int    `json:"session_active_count"`
	SessionExpiredCount    int    `json:"session_expired_count"`
	SessionRevokedCount    int    `json:"session_revoked_count"`
	SessionLatestStartedAt string `json:"session_latest_started_at,omitempty"`
}

type IdentityDeviceListResponse struct {
	WorkspaceID         string                 `json:"workspace_id"`
	UserID              string                 `json:"user_id,omitempty"`
	DeviceType          string                 `json:"device_type,omitempty"`
	Platform            string                 `json:"platform,omitempty"`
	Items               []IdentityDeviceRecord `json:"items"`
	HasMore             bool                   `json:"has_more"`
	NextCursorCreatedAt string                 `json:"next_cursor_created_at,omitempty"`
	NextCursorID        string                 `json:"next_cursor_id,omitempty"`
}

type IdentitySessionListRequest struct {
	WorkspaceID     string `json:"workspace_id"`
	DeviceID        string `json:"device_id,omitempty"`
	UserID          string `json:"user_id,omitempty"`
	SessionHealth   string `json:"session_health,omitempty"`
	CursorStartedAt string `json:"cursor_started_at,omitempty"`
	CursorID        string `json:"cursor_id,omitempty"`
	Limit           int    `json:"limit,omitempty"`
}

type IdentitySessionRecord struct {
	SessionID        string `json:"session_id"`
	WorkspaceID      string `json:"workspace_id"`
	DeviceID         string `json:"device_id"`
	UserID           string `json:"user_id"`
	DeviceType       string `json:"device_type"`
	Platform         string `json:"platform"`
	DeviceLabel      string `json:"device_label,omitempty"`
	DeviceLastSeenAt string `json:"device_last_seen_at,omitempty"`
	StartedAt        string `json:"started_at"`
	ExpiresAt        string `json:"expires_at"`
	RevokedAt        string `json:"revoked_at,omitempty"`
	SessionHealth    string `json:"session_health"`
}

type IdentitySessionListResponse struct {
	WorkspaceID         string                  `json:"workspace_id"`
	DeviceID            string                  `json:"device_id,omitempty"`
	UserID              string                  `json:"user_id,omitempty"`
	SessionHealth       string                  `json:"session_health,omitempty"`
	Items               []IdentitySessionRecord `json:"items"`
	HasMore             bool                    `json:"has_more"`
	NextCursorStartedAt string                  `json:"next_cursor_started_at,omitempty"`
	NextCursorID        string                  `json:"next_cursor_id,omitempty"`
}

type IdentitySessionRevokeRequest struct {
	WorkspaceID string `json:"workspace_id"`
	SessionID   string `json:"session_id"`
}

type IdentitySessionRevokeResponse struct {
	WorkspaceID      string `json:"workspace_id"`
	SessionID        string `json:"session_id"`
	DeviceID         string `json:"device_id"`
	StartedAt        string `json:"started_at"`
	ExpiresAt        string `json:"expires_at"`
	RevokedAt        string `json:"revoked_at"`
	DeviceLastSeenAt string `json:"device_last_seen_at,omitempty"`
	SessionHealth    string `json:"session_health"`
	Idempotent       bool   `json:"idempotent"`
}
