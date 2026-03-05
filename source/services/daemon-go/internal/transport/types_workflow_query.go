package transport

type ApprovalInboxRequest struct {
	WorkspaceID  string `json:"workspace_id"`
	IncludeFinal bool   `json:"include_final"`
	Limit        int    `json:"limit"`
	State        string `json:"state,omitempty"`
}

type ApprovalInboxItem struct {
	ApprovalRequestID       string                `json:"approval_request_id"`
	WorkspaceID             string                `json:"workspace_id"`
	State                   string                `json:"state"`
	Decision                string                `json:"decision,omitempty"`
	RequestedPhrase         string                `json:"requested_phrase,omitempty"`
	RiskLevel               string                `json:"risk_level"`
	RiskRationale           string                `json:"risk_rationale"`
	RequestedAt             string                `json:"requested_at"`
	DecidedAt               string                `json:"decided_at,omitempty"`
	DecisionByActorID       string                `json:"decision_by_actor_id,omitempty"`
	DecisionRationale       string                `json:"decision_rationale,omitempty"`
	TaskID                  string                `json:"task_id,omitempty"`
	TaskTitle               string                `json:"task_title,omitempty"`
	TaskState               string                `json:"task_state,omitempty"`
	RunID                   string                `json:"run_id,omitempty"`
	RunState                string                `json:"run_state,omitempty"`
	StepID                  string                `json:"step_id,omitempty"`
	StepName                string                `json:"step_name,omitempty"`
	StepCapabilityKey       string                `json:"step_capability_key,omitempty"`
	RequestedByActorID      string                `json:"requested_by_actor_id,omitempty"`
	SubjectPrincipalActorID string                `json:"subject_principal_actor_id,omitempty"`
	ActingAsActorID         string                `json:"acting_as_actor_id,omitempty"`
	Route                   WorkflowRouteMetadata `json:"route"`
}

type ApprovalInboxResponse struct {
	WorkspaceID string              `json:"workspace_id"`
	Approvals   []ApprovalInboxItem `json:"approvals"`
}

type TaskRunListRequest struct {
	WorkspaceID string `json:"workspace_id"`
	State       string `json:"state,omitempty"`
	Limit       int    `json:"limit"`
}

type TaskRunListItem struct {
	TaskID                  string                    `json:"task_id"`
	RunID                   string                    `json:"run_id,omitempty"`
	WorkspaceID             string                    `json:"workspace_id"`
	Title                   string                    `json:"title"`
	TaskState               string                    `json:"task_state"`
	RunState                string                    `json:"run_state,omitempty"`
	Priority                int                       `json:"priority"`
	RequestedByActorID      string                    `json:"requested_by_actor_id"`
	SubjectPrincipalActorID string                    `json:"subject_principal_actor_id"`
	ActingAsActorID         string                    `json:"acting_as_actor_id,omitempty"`
	LastError               string                    `json:"last_error,omitempty"`
	TaskCreatedAt           string                    `json:"task_created_at"`
	TaskUpdatedAt           string                    `json:"task_updated_at"`
	RunCreatedAt            string                    `json:"run_created_at,omitempty"`
	RunUpdatedAt            string                    `json:"run_updated_at,omitempty"`
	StartedAt               string                    `json:"started_at,omitempty"`
	FinishedAt              string                    `json:"finished_at,omitempty"`
	Actions                 TaskRunActionAvailability `json:"actions"`
	Route                   WorkflowRouteMetadata     `json:"route"`
}

type TaskRunListResponse struct {
	WorkspaceID string            `json:"workspace_id"`
	Items       []TaskRunListItem `json:"items"`
}

type CommThreadListRequest struct {
	WorkspaceID string `json:"workspace_id"`
	Channel     string `json:"channel,omitempty"`
	ConnectorID string `json:"connector_id,omitempty"`
	Query       string `json:"query,omitempty"`
	Cursor      string `json:"cursor,omitempty"`
	Limit       int    `json:"limit"`
}

type CommThreadListItem struct {
	ThreadID             string   `json:"thread_id"`
	WorkspaceID          string   `json:"workspace_id"`
	Channel              string   `json:"channel"`
	ConnectorID          string   `json:"connector_id"`
	ExternalRef          string   `json:"external_ref,omitempty"`
	Title                string   `json:"title,omitempty"`
	LastEventID          string   `json:"last_event_id,omitempty"`
	LastEventType        string   `json:"last_event_type,omitempty"`
	LastDirection        string   `json:"last_direction,omitempty"`
	LastOccurredAt       string   `json:"last_occurred_at,omitempty"`
	LastBodyPreview      string   `json:"last_body_preview,omitempty"`
	ParticipantAddresses []string `json:"participant_addresses,omitempty"`
	EventCount           int      `json:"event_count"`
	CreatedAt            string   `json:"created_at"`
	UpdatedAt            string   `json:"updated_at"`
}

type CommThreadListResponse struct {
	WorkspaceID string               `json:"workspace_id"`
	Items       []CommThreadListItem `json:"items"`
	HasMore     bool                 `json:"has_more"`
	NextCursor  string               `json:"next_cursor,omitempty"`
}

type CommEventTimelineRequest struct {
	WorkspaceID string `json:"workspace_id"`
	ThreadID    string `json:"thread_id,omitempty"`
	Channel     string `json:"channel,omitempty"`
	ConnectorID string `json:"connector_id,omitempty"`
	EventType   string `json:"event_type,omitempty"`
	Direction   string `json:"direction,omitempty"`
	Query       string `json:"query,omitempty"`
	Cursor      string `json:"cursor,omitempty"`
	Limit       int    `json:"limit"`
}

type CommEventAddressItem struct {
	Role     string `json:"role"`
	Value    string `json:"value"`
	Display  string `json:"display,omitempty"`
	Position int    `json:"position"`
}

type CommEventTimelineItem struct {
	EventID          string                 `json:"event_id"`
	WorkspaceID      string                 `json:"workspace_id"`
	ThreadID         string                 `json:"thread_id"`
	Channel          string                 `json:"channel"`
	ConnectorID      string                 `json:"connector_id"`
	EventType        string                 `json:"event_type"`
	Direction        string                 `json:"direction"`
	AssistantEmitted bool                   `json:"assistant_emitted"`
	BodyText         string                 `json:"body_text,omitempty"`
	OccurredAt       string                 `json:"occurred_at"`
	CreatedAt        string                 `json:"created_at"`
	Addresses        []CommEventAddressItem `json:"addresses,omitempty"`
}

type CommEventTimelineResponse struct {
	WorkspaceID string                  `json:"workspace_id"`
	ThreadID    string                  `json:"thread_id,omitempty"`
	Items       []CommEventTimelineItem `json:"items"`
	HasMore     bool                    `json:"has_more"`
	NextCursor  string                  `json:"next_cursor,omitempty"`
}

type CommCallSessionListRequest struct {
	WorkspaceID    string `json:"workspace_id"`
	ThreadID       string `json:"thread_id,omitempty"`
	Provider       string `json:"provider,omitempty"`
	ConnectorID    string `json:"connector_id,omitempty"`
	Direction      string `json:"direction,omitempty"`
	Status         string `json:"status,omitempty"`
	ProviderCallID string `json:"provider_call_id,omitempty"`
	Query          string `json:"query,omitempty"`
	Cursor         string `json:"cursor,omitempty"`
	Limit          int    `json:"limit"`
}

type CommCallSessionListItem struct {
	SessionID      string `json:"session_id"`
	WorkspaceID    string `json:"workspace_id"`
	Provider       string `json:"provider"`
	ConnectorID    string `json:"connector_id"`
	ProviderCallID string `json:"provider_call_id"`
	ThreadID       string `json:"thread_id"`
	Direction      string `json:"direction"`
	FromAddress    string `json:"from_address,omitempty"`
	ToAddress      string `json:"to_address,omitempty"`
	Status         string `json:"status"`
	StartedAt      string `json:"started_at,omitempty"`
	EndedAt        string `json:"ended_at,omitempty"`
	UpdatedAt      string `json:"updated_at"`
}

type CommCallSessionListResponse struct {
	WorkspaceID string                    `json:"workspace_id"`
	Items       []CommCallSessionListItem `json:"items"`
	HasMore     bool                      `json:"has_more"`
	NextCursor  string                    `json:"next_cursor,omitempty"`
}
