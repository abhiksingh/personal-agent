package transport

import coretypes "personalagent/runtime/internal/core/types"

type DelegationGrantRequest struct {
	WorkspaceID string `json:"workspace_id"`
	FromActorID string `json:"from_actor_id"`
	ToActorID   string `json:"to_actor_id"`
	ScopeType   string `json:"scope_type,omitempty"`
	ScopeKey    string `json:"scope_key,omitempty"`
	ExpiresAt   string `json:"expires_at,omitempty"`
}

type DelegationRuleRecord struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspace_id"`
	FromActorID string `json:"from_actor_id"`
	ToActorID   string `json:"to_actor_id"`
	ScopeType   string `json:"scope_type"`
	ScopeKey    string `json:"scope_key,omitempty"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	ExpiresAt   string `json:"expires_at,omitempty"`
}

type DelegationListRequest struct {
	WorkspaceID string `json:"workspace_id"`
	FromActorID string `json:"from_actor_id,omitempty"`
	ToActorID   string `json:"to_actor_id,omitempty"`
}

type DelegationListResponse struct {
	WorkspaceID string                 `json:"workspace_id"`
	Rules       []DelegationRuleRecord `json:"rules"`
}

type DelegationRevokeRequest struct {
	WorkspaceID string `json:"workspace_id"`
	RuleID      string `json:"rule_id"`
}

type DelegationRevokeResponse struct {
	WorkspaceID string `json:"workspace_id"`
	RuleID      string `json:"rule_id"`
	Status      string `json:"status"`
}

type DelegationCheckRequest struct {
	WorkspaceID        string `json:"workspace_id"`
	RequestedByActorID string `json:"requested_by_actor_id"`
	ActingAsActorID    string `json:"acting_as_actor_id"`
	ScopeType          string `json:"scope_type,omitempty"`
	ScopeKey           string `json:"scope_key,omitempty"`
}

type DelegationCheckResponse struct {
	WorkspaceID        string `json:"workspace_id"`
	RequestedByActorID string `json:"requested_by_actor_id"`
	ActingAsActorID    string `json:"acting_as_actor_id"`
	Allowed            bool   `json:"allowed"`
	Reason             string `json:"reason"`
	ReasonCode         string `json:"reason_code,omitempty"`
	DelegationRuleID   string `json:"delegation_rule_id,omitempty"`
}

type CapabilityGrantUpsertRequest struct {
	WorkspaceID   string `json:"workspace_id"`
	GrantID       string `json:"grant_id,omitempty"`
	ActorID       string `json:"actor_id,omitempty"`
	CapabilityKey string `json:"capability_key,omitempty"`
	ScopeJSON     string `json:"scope_json,omitempty"`
	Status        string `json:"status,omitempty"`
	ExpiresAt     string `json:"expires_at,omitempty"`
}

type CapabilityGrantRecord struct {
	GrantID       string `json:"grant_id"`
	WorkspaceID   string `json:"workspace_id"`
	ActorID       string `json:"actor_id"`
	CapabilityKey string `json:"capability_key"`
	ScopeJSON     string `json:"scope_json,omitempty"`
	Status        string `json:"status"`
	CreatedAt     string `json:"created_at"`
	ExpiresAt     string `json:"expires_at,omitempty"`
}

type CapabilityGrantListRequest struct {
	WorkspaceID     string `json:"workspace_id"`
	ActorID         string `json:"actor_id,omitempty"`
	CapabilityKey   string `json:"capability_key,omitempty"`
	Status          string `json:"status,omitempty"`
	CursorCreatedAt string `json:"cursor_created_at,omitempty"`
	CursorID        string `json:"cursor_id,omitempty"`
	Limit           int    `json:"limit,omitempty"`
}

type CapabilityGrantListResponse struct {
	WorkspaceID         string                  `json:"workspace_id"`
	Items               []CapabilityGrantRecord `json:"items"`
	HasMore             bool                    `json:"has_more"`
	NextCursorCreatedAt string                  `json:"next_cursor_created_at,omitempty"`
	NextCursorID        string                  `json:"next_cursor_id,omitempty"`
}

type CommSendRequest struct {
	WorkspaceID      string `json:"workspace_id"`
	OperationID      string `json:"operation_id"`
	SourceChannel    string `json:"source_channel"`
	ThreadID         string `json:"thread_id,omitempty"`
	ConnectorID      string `json:"connector_id,omitempty"`
	Destination      string `json:"destination"`
	Message          string `json:"message"`
	StepID           string `json:"step_id,omitempty"`
	EventID          string `json:"event_id,omitempty"`
	IMessagesFailure int    `json:"imessage_failures,omitempty"`
	SMSFailures      int    `json:"sms_failures,omitempty"`
}

type CommSendResponse struct {
	WorkspaceID           string                   `json:"workspace_id"`
	OperationID           string                   `json:"operation_id"`
	ThreadID              string                   `json:"thread_id,omitempty"`
	ResolvedSourceChannel string                   `json:"resolved_source_channel,omitempty"`
	ResolvedConnectorID   string                   `json:"resolved_connector_id,omitempty"`
	ResolvedDestination   string                   `json:"resolved_destination,omitempty"`
	Success               bool                     `json:"success"`
	Result                coretypes.DeliveryResult `json:"result"`
	Error                 string                   `json:"error,omitempty"`
}

type CommAttemptsRequest struct {
	WorkspaceID string `json:"workspace_id"`
	OperationID string `json:"operation_id,omitempty"`
	ThreadID    string `json:"thread_id,omitempty"`
	TaskID      string `json:"task_id,omitempty"`
	RunID       string `json:"run_id,omitempty"`
	StepID      string `json:"step_id,omitempty"`
	Channel     string `json:"channel,omitempty"`
	Status      string `json:"status,omitempty"`
	Cursor      string `json:"cursor,omitempty"`
	Limit       int    `json:"limit,omitempty"`
}

type CommAttemptRecord struct {
	AttemptID           string `json:"attempt_id"`
	WorkspaceID         string `json:"workspace_id"`
	OperationID         string `json:"operation_id,omitempty"`
	TaskID              string `json:"task_id,omitempty"`
	RunID               string `json:"run_id,omitempty"`
	StepID              string `json:"step_id,omitempty"`
	EventID             string `json:"event_id,omitempty"`
	ThreadID            string `json:"thread_id,omitempty"`
	DestinationEndpoint string `json:"destination_endpoint"`
	IdempotencyKey      string `json:"idempotency_key"`
	Channel             string `json:"channel"`
	RouteIndex          int    `json:"route_index"`
	RoutePhase          string `json:"route_phase,omitempty"`
	RetryOrdinal        int    `json:"retry_ordinal,omitempty"`
	FallbackFromChannel string `json:"fallback_from_channel,omitempty"`
	Status              string `json:"status"`
	ProviderReceipt     string `json:"provider_receipt,omitempty"`
	Error               string `json:"error,omitempty"`
	AttemptedAt         string `json:"attempted_at"`
}

type CommAttemptsResponse struct {
	WorkspaceID string              `json:"workspace_id"`
	OperationID string              `json:"operation_id,omitempty"`
	ThreadID    string              `json:"thread_id,omitempty"`
	TaskID      string              `json:"task_id,omitempty"`
	RunID       string              `json:"run_id,omitempty"`
	StepID      string              `json:"step_id,omitempty"`
	HasMore     bool                `json:"has_more"`
	NextCursor  string              `json:"next_cursor,omitempty"`
	Attempts    []CommAttemptRecord `json:"attempts"`
}

type ReceiptAuditLink struct {
	AuditID   string `json:"audit_id"`
	EventType string `json:"event_type"`
	CreatedAt string `json:"created_at"`
}

type CommWebhookReceiptListRequest struct {
	WorkspaceID        string `json:"workspace_id"`
	Provider           string `json:"provider,omitempty"`
	ProviderEventID    string `json:"provider_event_id,omitempty"`
	ProviderEventQuery string `json:"provider_event_query,omitempty"`
	EventID            string `json:"event_id,omitempty"`
	CursorCreatedAt    string `json:"cursor_created_at,omitempty"`
	CursorID           string `json:"cursor_id,omitempty"`
	Limit              int    `json:"limit,omitempty"`
}

type CommWebhookReceiptItem struct {
	ReceiptID             string             `json:"receipt_id"`
	WorkspaceID           string             `json:"workspace_id"`
	Provider              string             `json:"provider"`
	ProviderEventID       string             `json:"provider_event_id"`
	TrustState            string             `json:"trust_state"`
	SignatureValid        bool               `json:"signature_valid"`
	SignatureValuePresent bool               `json:"signature_value_present"`
	PayloadHash           string             `json:"payload_hash,omitempty"`
	EventID               string             `json:"event_id,omitempty"`
	ThreadID              string             `json:"thread_id,omitempty"`
	ReceivedAt            string             `json:"received_at"`
	CreatedAt             string             `json:"created_at"`
	AuditLinks            []ReceiptAuditLink `json:"audit_links"`
}

type CommWebhookReceiptListResponse struct {
	WorkspaceID         string                   `json:"workspace_id"`
	Provider            string                   `json:"provider,omitempty"`
	Items               []CommWebhookReceiptItem `json:"items"`
	HasMore             bool                     `json:"has_more"`
	NextCursorCreatedAt string                   `json:"next_cursor_created_at,omitempty"`
	NextCursorID        string                   `json:"next_cursor_id,omitempty"`
}

type CommIngestReceiptListRequest struct {
	WorkspaceID      string `json:"workspace_id"`
	Source           string `json:"source,omitempty"`
	SourceScope      string `json:"source_scope,omitempty"`
	SourceEventID    string `json:"source_event_id,omitempty"`
	SourceEventQuery string `json:"source_event_query,omitempty"`
	TrustState       string `json:"trust_state,omitempty"`
	EventID          string `json:"event_id,omitempty"`
	CursorCreatedAt  string `json:"cursor_created_at,omitempty"`
	CursorID         string `json:"cursor_id,omitempty"`
	Limit            int    `json:"limit,omitempty"`
}

type CommIngestReceiptItem struct {
	ReceiptID     string             `json:"receipt_id"`
	WorkspaceID   string             `json:"workspace_id"`
	Source        string             `json:"source"`
	SourceScope   string             `json:"source_scope"`
	SourceEventID string             `json:"source_event_id"`
	SourceCursor  string             `json:"source_cursor,omitempty"`
	TrustState    string             `json:"trust_state"`
	PayloadHash   string             `json:"payload_hash,omitempty"`
	EventID       string             `json:"event_id,omitempty"`
	ThreadID      string             `json:"thread_id,omitempty"`
	ReceivedAt    string             `json:"received_at"`
	CreatedAt     string             `json:"created_at"`
	AuditLinks    []ReceiptAuditLink `json:"audit_links"`
}

type CommIngestReceiptListResponse struct {
	WorkspaceID         string                  `json:"workspace_id"`
	Source              string                  `json:"source,omitempty"`
	SourceScope         string                  `json:"source_scope,omitempty"`
	Items               []CommIngestReceiptItem `json:"items"`
	HasMore             bool                    `json:"has_more"`
	NextCursorCreatedAt string                  `json:"next_cursor_created_at,omitempty"`
	NextCursorID        string                  `json:"next_cursor_id,omitempty"`
}

type CommPolicySetRequest struct {
	PolicyID         string   `json:"policy_id,omitempty"`
	WorkspaceID      string   `json:"workspace_id"`
	SourceChannel    string   `json:"source_channel"`
	EndpointPattern  string   `json:"endpoint_pattern,omitempty"`
	PrimaryChannel   string   `json:"primary_channel"`
	RetryCount       int      `json:"retry_count"`
	FallbackChannels []string `json:"fallback_channels"`
	IsDefault        bool     `json:"is_default"`
}

type CommPolicyRecord struct {
	ID              string                          `json:"id"`
	WorkspaceID     string                          `json:"workspace_id"`
	SourceChannel   string                          `json:"source_channel"`
	EndpointPattern string                          `json:"endpoint_pattern,omitempty"`
	IsDefault       bool                            `json:"is_default"`
	Policy          coretypes.ChannelDeliveryPolicy `json:"policy"`
	CreatedAt       string                          `json:"created_at"`
	UpdatedAt       string                          `json:"updated_at"`
}

type CommPolicyListRequest struct {
	WorkspaceID   string `json:"workspace_id"`
	SourceChannel string `json:"source_channel,omitempty"`
}

type CommPolicyListResponse struct {
	WorkspaceID string             `json:"workspace_id"`
	Policies    []CommPolicyRecord `json:"policies"`
}
