package transport

import coretypes "personalagent/runtime/internal/core/types"

type AutomationCreateRequest struct {
	WorkspaceID     string                       `json:"workspace_id"`
	SubjectActorID  string                       `json:"subject_actor_id,omitempty"`
	TriggerType     string                       `json:"trigger_type"`
	Title           string                       `json:"title,omitempty"`
	Instruction     string                       `json:"instruction,omitempty"`
	DirectiveID     string                       `json:"directive_id,omitempty"`
	TriggerID       string                       `json:"trigger_id,omitempty"`
	IntervalSeconds int                          `json:"interval_seconds,omitempty"`
	Filter          *AutomationCommTriggerFilter `json:"filter,omitempty"`
	CooldownSeconds int                          `json:"cooldown_seconds,omitempty"`
	Enabled         bool                         `json:"enabled"`
}

type AutomationListRequest struct {
	WorkspaceID     string `json:"workspace_id"`
	TriggerType     string `json:"trigger_type,omitempty"`
	IncludeDisabled bool   `json:"include_disabled"`
}

type AutomationTriggerRecord struct {
	TriggerID             string `json:"trigger_id"`
	WorkspaceID           string `json:"workspace_id"`
	DirectiveID           string `json:"directive_id"`
	TriggerType           string `json:"trigger_type"`
	Enabled               bool   `json:"enabled"`
	FilterJSON            string `json:"filter_json"`
	CooldownSeconds       int    `json:"cooldown_seconds"`
	SubjectPrincipalActor string `json:"subject_principal_actor"`
	DirectiveTitle        string `json:"directive_title"`
	DirectiveInstruction  string `json:"directive_instruction"`
	DirectiveStatus       string `json:"directive_status"`
	CreatedAt             string `json:"created_at"`
	UpdatedAt             string `json:"updated_at"`
}

type AutomationListResponse struct {
	WorkspaceID string                    `json:"workspace_id"`
	Triggers    []AutomationTriggerRecord `json:"triggers"`
}

type AutomationFireHistoryRequest struct {
	WorkspaceID string `json:"workspace_id"`
	TriggerID   string `json:"trigger_id,omitempty"`
	Status      string `json:"status,omitempty"`
	Limit       int    `json:"limit"`
}

type AutomationFireHistoryRecord struct {
	FireID            string                `json:"fire_id"`
	WorkspaceID       string                `json:"workspace_id"`
	TriggerID         string                `json:"trigger_id"`
	TriggerType       string                `json:"trigger_type"`
	DirectiveID       string                `json:"directive_id,omitempty"`
	Status            string                `json:"status"`
	Outcome           string                `json:"outcome,omitempty"`
	IdempotencyKey    string                `json:"idempotency_key"`
	IdempotencySignal string                `json:"idempotency_signal"`
	FiredAt           string                `json:"fired_at"`
	TaskID            string                `json:"task_id,omitempty"`
	RunID             string                `json:"run_id,omitempty"`
	Route             WorkflowRouteMetadata `json:"route"`
}

type AutomationFireHistoryResponse struct {
	WorkspaceID string                        `json:"workspace_id"`
	Fires       []AutomationFireHistoryRecord `json:"fires"`
}

type AutomationUpdateRequest struct {
	WorkspaceID     string                       `json:"workspace_id"`
	TriggerID       string                       `json:"trigger_id"`
	SubjectActorID  string                       `json:"subject_actor_id,omitempty"`
	Title           string                       `json:"title,omitempty"`
	Instruction     string                       `json:"instruction,omitempty"`
	IntervalSeconds *int                         `json:"interval_seconds,omitempty"`
	Filter          *AutomationCommTriggerFilter `json:"filter,omitempty"`
	CooldownSeconds *int                         `json:"cooldown_seconds,omitempty"`
	Enabled         *bool                        `json:"enabled,omitempty"`
}

type AutomationUpdateResponse struct {
	Trigger    AutomationTriggerRecord `json:"trigger"`
	Updated    bool                    `json:"updated"`
	Idempotent bool                    `json:"idempotent"`
}

type AutomationDeleteRequest struct {
	WorkspaceID string `json:"workspace_id"`
	TriggerID   string `json:"trigger_id"`
}

type AutomationDeleteResponse struct {
	WorkspaceID string `json:"workspace_id"`
	TriggerID   string `json:"trigger_id"`
	DirectiveID string `json:"directive_id,omitempty"`
	Deleted     bool   `json:"deleted"`
	Idempotent  bool   `json:"idempotent"`
}

type AutomationRunScheduleRequest struct {
	At string `json:"at,omitempty"`
}

type AutomationRunScheduleResponse struct {
	At     string                             `json:"at"`
	Result coretypes.ScheduleEvaluationResult `json:"result"`
}

type AutomationRunCommEventRequest struct {
	WorkspaceID      string `json:"workspace_id"`
	EventID          string `json:"event_id"`
	SeedEvent        bool   `json:"seed_event"`
	ThreadID         string `json:"thread_id,omitempty"`
	Channel          string `json:"channel,omitempty"`
	Body             string `json:"body,omitempty"`
	Sender           string `json:"sender,omitempty"`
	EventType        string `json:"event_type,omitempty"`
	Direction        string `json:"direction,omitempty"`
	AssistantEmitted bool   `json:"assistant_emitted"`
	OccurredAt       string `json:"occurred_at,omitempty"`
}

type AutomationRunCommEventResponse struct {
	EventID     string                                `json:"event_id"`
	SeededEvent bool                                  `json:"seeded_event"`
	Result      coretypes.CommTriggerEvaluationResult `json:"result"`
}

type AutomationCommTriggerMetadataRequest struct {
	WorkspaceID string `json:"workspace_id,omitempty"`
}

type AutomationCommTriggerRequiredDefaults struct {
	EventType        string `json:"event_type"`
	Direction        string `json:"direction"`
	AssistantEmitted bool   `json:"assistant_emitted"`
}

type AutomationCommTriggerKeywordFilter struct {
	ContainsAny  []string `json:"contains_any"`
	ContainsAll  []string `json:"contains_all"`
	ExactPhrases []string `json:"exact_phrases"`
}

type AutomationCommTriggerFilter struct {
	Channels          []string                           `json:"channels"`
	PrincipalActorIDs []string                           `json:"principal_actor_ids"`
	SenderAllowlist   []string                           `json:"sender_allowlist"`
	ThreadIDs         []string                           `json:"thread_ids"`
	Keywords          AutomationCommTriggerKeywordFilter `json:"keywords"`
}

type AutomationCommTriggerFilterFieldSchema struct {
	Field          string `json:"field"`
	ValueType      string `json:"value_type"`
	MatchSemantics string `json:"match_semantics"`
	Description    string `json:"description"`
}

type AutomationCommTriggerMetadataCompatibility struct {
	PrincipalFilterBehavior string `json:"principal_filter_behavior"`
	KeywordMatchBehavior    string `json:"keyword_match_behavior"`
}

type AutomationCommTriggerMetadataResponse struct {
	TriggerType          string                                     `json:"trigger_type"`
	RequiredDefaults     AutomationCommTriggerRequiredDefaults      `json:"required_defaults"`
	IdempotencyKeyFields []string                                   `json:"idempotency_key_fields"`
	FilterDefaults       AutomationCommTriggerFilter                `json:"filter_defaults"`
	FilterSchema         []AutomationCommTriggerFilterFieldSchema   `json:"filter_schema"`
	Compatibility        AutomationCommTriggerMetadataCompatibility `json:"compatibility"`
}

type AutomationCommTriggerValidateRequest struct {
	WorkspaceID    string                       `json:"workspace_id,omitempty"`
	SubjectActorID string                       `json:"subject_actor_id,omitempty"`
	Filter         *AutomationCommTriggerFilter `json:"filter,omitempty"`
}

type AutomationCommTriggerValidationIssue struct {
	Code    string `json:"code"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

type AutomationCommTriggerValidationCompatibility struct {
	Compatible                  bool   `json:"compatible"`
	SubjectActorID              string `json:"subject_actor_id,omitempty"`
	SubjectMatchesPrincipalRule bool   `json:"subject_matches_principal_rule"`
}

type AutomationCommTriggerValidateResponse struct {
	Valid                bool                                         `json:"valid"`
	TriggerType          string                                       `json:"trigger_type"`
	RequiredDefaults     AutomationCommTriggerRequiredDefaults        `json:"required_defaults"`
	NormalizedFilter     AutomationCommTriggerFilter                  `json:"normalized_filter"`
	NormalizedFilterJSON string                                       `json:"normalized_filter_json"`
	Errors               []AutomationCommTriggerValidationIssue       `json:"errors"`
	Warnings             []AutomationCommTriggerValidationIssue       `json:"warnings"`
	Compatibility        AutomationCommTriggerValidationCompatibility `json:"compatibility"`
}

type InspectRunRequest struct {
	RunID string `json:"run_id"`
}

type InspectTaskRun struct {
	RunID           string `json:"run_id"`
	WorkspaceID     string `json:"workspace_id"`
	TaskID          string `json:"task_id"`
	ActingAsActorID string `json:"acting_as_actor_id"`
	State           string `json:"state"`
	StartedAt       string `json:"started_at,omitempty"`
	FinishedAt      string `json:"finished_at,omitempty"`
	LastError       string `json:"last_error,omitempty"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

type InspectTask struct {
	TaskID                string `json:"task_id"`
	WorkspaceID           string `json:"workspace_id"`
	RequestedByActorID    string `json:"requested_by_actor_id"`
	SubjectPrincipalActor string `json:"subject_principal_actor_id"`
	Title                 string `json:"title"`
	Description           string `json:"description,omitempty"`
	State                 string `json:"state"`
	Priority              int    `json:"priority"`
	DeadlineAt            string `json:"deadline_at,omitempty"`
	Channel               string `json:"channel,omitempty"`
	CreatedAt             string `json:"created_at"`
	UpdatedAt             string `json:"updated_at"`
}

type InspectStep struct {
	StepID           string `json:"step_id"`
	RunID            string `json:"run_id"`
	StepIndex        int    `json:"step_index"`
	Name             string `json:"name"`
	Status           string `json:"status"`
	InteractionLevel string `json:"interaction_level,omitempty"`
	CapabilityKey    string `json:"capability_key,omitempty"`
	TimeoutSeconds   int    `json:"timeout_seconds,omitempty"`
	RetryMax         int    `json:"retry_max"`
	RetryCount       int    `json:"retry_count"`
	LastError        string `json:"last_error,omitempty"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

type InspectRunArtifact struct {
	ArtifactID   string `json:"artifact_id"`
	RunID        string `json:"run_id"`
	StepID       string `json:"step_id,omitempty"`
	ArtifactType string `json:"artifact_type"`
	URI          string `json:"uri,omitempty"`
	ContentHash  string `json:"content_hash,omitempty"`
	CreatedAt    string `json:"created_at"`
}

type InspectAuditEntry struct {
	AuditID         string `json:"audit_id"`
	WorkspaceID     string `json:"workspace_id"`
	RunID           string `json:"run_id,omitempty"`
	StepID          string `json:"step_id,omitempty"`
	EventType       string `json:"event_type"`
	ActorID         string `json:"actor_id,omitempty"`
	ActingAsActorID string `json:"acting_as_actor_id,omitempty"`
	CorrelationID   string `json:"correlation_id,omitempty"`
	PayloadJSON     string `json:"payload_json,omitempty"`
	CreatedAt       string `json:"created_at"`
}

type InspectRunResponse struct {
	Task         InspectTask           `json:"task"`
	Run          InspectTaskRun        `json:"run"`
	Steps        []InspectStep         `json:"steps"`
	Artifacts    []InspectRunArtifact  `json:"artifacts"`
	AuditEntries []InspectAuditEntry   `json:"audit_entries"`
	Route        WorkflowRouteMetadata `json:"route"`
}

type InspectTranscriptRequest struct {
	WorkspaceID string `json:"workspace_id"`
	ThreadID    string `json:"thread_id,omitempty"`
	Limit       int    `json:"limit"`
}

type InspectTranscriptEvent struct {
	EventID          string `json:"event_id"`
	WorkspaceID      string `json:"workspace_id"`
	ThreadID         string `json:"thread_id"`
	Channel          string `json:"channel"`
	EventType        string `json:"event_type"`
	Direction        string `json:"direction"`
	AssistantEmitted bool   `json:"assistant_emitted"`
	BodyText         string `json:"body_text,omitempty"`
	SenderAddress    string `json:"sender_address,omitempty"`
	OccurredAt       string `json:"occurred_at"`
	CreatedAt        string `json:"created_at"`
}

type InspectTranscriptResponse struct {
	WorkspaceID string                   `json:"workspace_id"`
	Events      []InspectTranscriptEvent `json:"events"`
}

type InspectMemoryRequest struct {
	WorkspaceID string `json:"workspace_id"`
	OwnerActor  string `json:"owner_actor,omitempty"`
	Status      string `json:"status,omitempty"`
	Limit       int    `json:"limit"`
}

type InspectMemoryItem struct {
	MemoryID      string `json:"memory_id"`
	WorkspaceID   string `json:"workspace_id"`
	OwnerActorID  string `json:"owner_actor_id"`
	ScopeType     string `json:"scope_type"`
	Key           string `json:"key"`
	Status        string `json:"status"`
	Kind          string `json:"kind"`
	IsCanonical   bool   `json:"is_canonical"`
	TokenEstimate int    `json:"token_estimate"`
	SourceSummary string `json:"source_summary,omitempty"`
	SourceCount   int    `json:"source_count"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
	ValueJSON     string `json:"value_json"`
}

type InspectMemoryResponse struct {
	WorkspaceID string              `json:"workspace_id"`
	Items       []InspectMemoryItem `json:"items"`
}

type InspectLogQueryRequest struct {
	WorkspaceID     string `json:"workspace_id"`
	RunID           string `json:"run_id,omitempty"`
	EventType       string `json:"event_type,omitempty"`
	BeforeCreatedAt string `json:"before_created_at,omitempty"`
	BeforeID        string `json:"before_id,omitempty"`
	Limit           int    `json:"limit"`
}

type InspectLogStreamRequest struct {
	WorkspaceID     string `json:"workspace_id"`
	RunID           string `json:"run_id,omitempty"`
	EventType       string `json:"event_type,omitempty"`
	CursorCreatedAt string `json:"cursor_created_at,omitempty"`
	CursorID        string `json:"cursor_id,omitempty"`
	Limit           int    `json:"limit"`
	TimeoutMS       int64  `json:"timeout_ms"`
	PollIntervalMS  int64  `json:"poll_interval_ms"`
}

type InspectLogRecord struct {
	LogID           string                `json:"log_id"`
	WorkspaceID     string                `json:"workspace_id"`
	RunID           string                `json:"run_id,omitempty"`
	StepID          string                `json:"step_id,omitempty"`
	EventType       string                `json:"event_type"`
	Status          string                `json:"status"`
	InputSummary    string                `json:"input_summary"`
	OutputSummary   string                `json:"output_summary"`
	CorrelationID   string                `json:"correlation_id,omitempty"`
	ActorID         string                `json:"actor_id,omitempty"`
	ActingAsActorID string                `json:"acting_as_actor_id,omitempty"`
	CreatedAt       string                `json:"created_at"`
	Metadata        map[string]any        `json:"metadata,omitempty"`
	Route           WorkflowRouteMetadata `json:"route"`
}

type InspectLogQueryResponse struct {
	WorkspaceID         string             `json:"workspace_id"`
	Logs                []InspectLogRecord `json:"logs"`
	NextCursorCreatedAt string             `json:"next_cursor_created_at,omitempty"`
	NextCursorID        string             `json:"next_cursor_id,omitempty"`
}

type InspectLogStreamResponse struct {
	WorkspaceID     string             `json:"workspace_id"`
	Logs            []InspectLogRecord `json:"logs"`
	CursorCreatedAt string             `json:"cursor_created_at,omitempty"`
	CursorID        string             `json:"cursor_id,omitempty"`
	TimedOut        bool               `json:"timed_out"`
}

type RetentionPurgeRequest struct {
	TraceDays      int `json:"trace_days,omitempty"`
	TranscriptDays int `json:"transcript_days,omitempty"`
	MemoryDays     int `json:"memory_days,omitempty"`
}

type RetentionPurgeResponse struct {
	EffectivePolicy coretypes.RetentionPolicy      `json:"effective_policy"`
	Result          coretypes.RetentionPurgeResult `json:"result"`
}

type RetentionCompactMemoryRequest struct {
	WorkspaceID     string `json:"workspace_id"`
	OwnerActor      string `json:"owner_actor"`
	TokenThreshold  int    `json:"token_threshold"`
	StaleAfterHours int    `json:"stale_after_hours"`
	Limit           int    `json:"limit"`
	Apply           bool   `json:"apply"`
}

type RetentionCompactMemoryResponse struct {
	WorkspaceID       string                           `json:"workspace_id"`
	OwnerActorID      string                           `json:"owner_actor_id"`
	Applied           bool                             `json:"applied"`
	CreatedSummaryIDs []string                         `json:"created_summary_ids"`
	Result            coretypes.MemoryCompactionResult `json:"result"`
}

type ContextSamplesRequest struct {
	WorkspaceID string `json:"workspace_id"`
	TaskClass   string `json:"task_class"`
	Limit       int    `json:"limit"`
}

type ContextSamplesResponse struct {
	WorkspaceID  string                               `json:"workspace_id"`
	TaskClass    string                               `json:"task_class"`
	Samples      []coretypes.ContextBudgetSample      `json:"samples"`
	Profile      coretypes.ContextBudgetTuningProfile `json:"profile"`
	ProfileFound bool                                 `json:"profile_found"`
}

type ContextTuneRequest struct {
	WorkspaceID string `json:"workspace_id"`
	TaskClass   string `json:"task_class"`
}

type ContextTuneResponse = coretypes.ContextBudgetTuningDecision

type ContextMemoryInventoryRequest struct {
	WorkspaceID     string `json:"workspace_id"`
	OwnerActorID    string `json:"owner_actor_id,omitempty"`
	ScopeType       string `json:"scope_type,omitempty"`
	Status          string `json:"status,omitempty"`
	SourceType      string `json:"source_type,omitempty"`
	SourceRefQuery  string `json:"source_ref_query,omitempty"`
	CursorUpdatedAt string `json:"cursor_updated_at,omitempty"`
	CursorID        string `json:"cursor_id,omitempty"`
	Limit           int    `json:"limit"`
}

type ContextMemorySourceRecord struct {
	SourceID   string `json:"source_id"`
	SourceType string `json:"source_type"`
	SourceRef  string `json:"source_ref"`
	CreatedAt  string `json:"created_at"`
}

type ContextMemoryInventoryItem struct {
	MemoryID      string                      `json:"memory_id"`
	WorkspaceID   string                      `json:"workspace_id"`
	OwnerActorID  string                      `json:"owner_actor_id"`
	ScopeType     string                      `json:"scope_type"`
	Key           string                      `json:"key"`
	Status        string                      `json:"status"`
	Kind          string                      `json:"kind"`
	IsCanonical   bool                        `json:"is_canonical"`
	TokenEstimate int                         `json:"token_estimate"`
	SourceSummary string                      `json:"source_summary,omitempty"`
	SourceCount   int                         `json:"source_count"`
	CreatedAt     string                      `json:"created_at"`
	UpdatedAt     string                      `json:"updated_at"`
	ValueJSON     string                      `json:"value_json"`
	Sources       []ContextMemorySourceRecord `json:"sources"`
}

type ContextMemoryInventoryResponse struct {
	WorkspaceID         string                       `json:"workspace_id"`
	Items               []ContextMemoryInventoryItem `json:"items"`
	HasMore             bool                         `json:"has_more"`
	NextCursorUpdatedAt string                       `json:"next_cursor_updated_at,omitempty"`
	NextCursorID        string                       `json:"next_cursor_id,omitempty"`
}

type ContextMemoryCandidatesRequest struct {
	WorkspaceID     string `json:"workspace_id"`
	OwnerActorID    string `json:"owner_actor_id,omitempty"`
	Status          string `json:"status,omitempty"`
	CursorCreatedAt string `json:"cursor_created_at,omitempty"`
	CursorID        string `json:"cursor_id,omitempty"`
	Limit           int    `json:"limit"`
}

type ContextMemoryCandidateItem struct {
	CandidateID   string   `json:"candidate_id"`
	WorkspaceID   string   `json:"workspace_id"`
	OwnerActorID  string   `json:"owner_actor_id"`
	Status        string   `json:"status"`
	Score         *float64 `json:"score,omitempty"`
	CandidateJSON string   `json:"candidate_json"`
	CandidateKind string   `json:"candidate_kind,omitempty"`
	TokenEstimate int      `json:"token_estimate,omitempty"`
	SourceIDs     []string `json:"source_ids,omitempty"`
	SourceRefs    []string `json:"source_refs,omitempty"`
	CreatedAt     string   `json:"created_at"`
}

type ContextMemoryCandidatesResponse struct {
	WorkspaceID         string                       `json:"workspace_id"`
	Items               []ContextMemoryCandidateItem `json:"items"`
	HasMore             bool                         `json:"has_more"`
	NextCursorCreatedAt string                       `json:"next_cursor_created_at,omitempty"`
	NextCursorID        string                       `json:"next_cursor_id,omitempty"`
}

type ContextRetrievalDocumentsRequest struct {
	WorkspaceID     string `json:"workspace_id"`
	OwnerActorID    string `json:"owner_actor_id,omitempty"`
	SourceURIQuery  string `json:"source_uri_query,omitempty"`
	CursorCreatedAt string `json:"cursor_created_at,omitempty"`
	CursorID        string `json:"cursor_id,omitempty"`
	Limit           int    `json:"limit"`
}

type ContextRetrievalDocumentItem struct {
	DocumentID   string `json:"document_id"`
	WorkspaceID  string `json:"workspace_id"`
	OwnerActorID string `json:"owner_actor_id,omitempty"`
	SourceURI    string `json:"source_uri,omitempty"`
	Checksum     string `json:"checksum,omitempty"`
	ChunkCount   int    `json:"chunk_count"`
	CreatedAt    string `json:"created_at"`
}

type ContextRetrievalDocumentsResponse struct {
	WorkspaceID         string                         `json:"workspace_id"`
	Items               []ContextRetrievalDocumentItem `json:"items"`
	HasMore             bool                           `json:"has_more"`
	NextCursorCreatedAt string                         `json:"next_cursor_created_at,omitempty"`
	NextCursorID        string                         `json:"next_cursor_id,omitempty"`
}

type ContextRetrievalChunksRequest struct {
	WorkspaceID     string `json:"workspace_id"`
	DocumentID      string `json:"document_id,omitempty"`
	OwnerActorID    string `json:"owner_actor_id,omitempty"`
	SourceURIQuery  string `json:"source_uri_query,omitempty"`
	ChunkTextQuery  string `json:"chunk_text_query,omitempty"`
	CursorCreatedAt string `json:"cursor_created_at,omitempty"`
	CursorID        string `json:"cursor_id,omitempty"`
	Limit           int    `json:"limit"`
}

type ContextRetrievalChunkItem struct {
	ChunkID      string `json:"chunk_id"`
	WorkspaceID  string `json:"workspace_id"`
	DocumentID   string `json:"document_id"`
	OwnerActorID string `json:"owner_actor_id,omitempty"`
	SourceURI    string `json:"source_uri,omitempty"`
	ChunkIndex   int    `json:"chunk_index"`
	TextBody     string `json:"text_body"`
	TokenCount   int    `json:"token_count,omitempty"`
	CreatedAt    string `json:"created_at"`
}

type ContextRetrievalChunksResponse struct {
	WorkspaceID         string                      `json:"workspace_id"`
	DocumentID          string                      `json:"document_id,omitempty"`
	Items               []ContextRetrievalChunkItem `json:"items"`
	HasMore             bool                        `json:"has_more"`
	NextCursorCreatedAt string                      `json:"next_cursor_created_at,omitempty"`
	NextCursorID        string                      `json:"next_cursor_id,omitempty"`
}
