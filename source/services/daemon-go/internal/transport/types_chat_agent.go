package transport

type ChatTurnChannelContext struct {
	ChannelID   string `json:"channel_id,omitempty"`
	ConnectorID string `json:"connector_id,omitempty"`
	ThreadID    string `json:"thread_id,omitempty"`
}

const (
	ChatTurnContractVersionV2              = "chat_turn.v2"
	ChatTurnItemSchemaVersionV1            = "chat_turn_item.v1"
	ChatRealtimeLifecycleContractVersionV2 = "chat_realtime_lifecycle.v2"
	ChatTurnExplainContractVersionV1       = "chat_turn_explain.v1"
)

type ChatTurnItem struct {
	ItemID            string               `json:"item_id,omitempty"`
	Type              string               `json:"type"`
	Role              string               `json:"role,omitempty"`
	Status            string               `json:"status,omitempty"`
	Content           string               `json:"content,omitempty"`
	ToolName          string               `json:"tool_name,omitempty"`
	ToolCallID        string               `json:"tool_call_id,omitempty"`
	Arguments         map[string]any       `json:"arguments,omitempty"`
	Output            map[string]any       `json:"output,omitempty"`
	ErrorCode         string               `json:"error_code,omitempty"`
	ErrorMessage      string               `json:"error_message,omitempty"`
	ApprovalRequestID string               `json:"approval_request_id,omitempty"`
	Metadata          ChatTurnItemMetadata `json:"metadata,omitempty"`
}

type ChatTurnRequest struct {
	WorkspaceID        string                     `json:"workspace_id"`
	TaskClass          string                     `json:"task_class,omitempty"`
	RequestedByActorID string                     `json:"requested_by_actor_id,omitempty"`
	SubjectActorID     string                     `json:"subject_actor_id,omitempty"`
	ActingAsActorID    string                     `json:"acting_as_actor_id,omitempty"`
	ProviderOverride   string                     `json:"provider,omitempty"`
	ModelOverride      string                     `json:"model,omitempty"`
	SystemPrompt       string                     `json:"system_prompt,omitempty"`
	Channel            ChatTurnChannelContext     `json:"channel,omitempty"`
	ToolCatalog        []ChatTurnToolCatalogEntry `json:"tool_catalog,omitempty"`
	Items              []ChatTurnItem             `json:"items"`
}

type ChatTurnTaskRunCorrelation struct {
	Available bool   `json:"available"`
	Source    string `json:"source"`
	TaskID    string `json:"task_id,omitempty"`
	RunID     string `json:"run_id,omitempty"`
	TaskState string `json:"task_state,omitempty"`
	RunState  string `json:"run_state,omitempty"`
}

type ChatTurnResponse struct {
	WorkspaceID                  string                     `json:"workspace_id"`
	TaskClass                    string                     `json:"task_class"`
	Provider                     string                     `json:"provider"`
	ModelKey                     string                     `json:"model_key"`
	CorrelationID                string                     `json:"correlation_id"`
	ContractVersion              string                     `json:"contract_version,omitempty"`
	TurnItemSchemaVersion        string                     `json:"turn_item_schema_version,omitempty"`
	RealtimeEventContractVersion string                     `json:"realtime_event_contract_version,omitempty"`
	Channel                      ChatTurnChannelContext     `json:"channel,omitempty"`
	Items                        []ChatTurnItem             `json:"items"`
	TaskRunCorrelation           ChatTurnTaskRunCorrelation `json:"task_run_correlation"`
}

type ChatTurnExplainRequest struct {
	WorkspaceID        string                 `json:"workspace_id"`
	TaskClass          string                 `json:"task_class,omitempty"`
	RequestedByActorID string                 `json:"requested_by_actor_id,omitempty"`
	SubjectActorID     string                 `json:"subject_actor_id,omitempty"`
	ActingAsActorID    string                 `json:"acting_as_actor_id,omitempty"`
	Channel            ChatTurnChannelContext `json:"channel,omitempty"`
}

type ChatTurnToolCatalogEntry struct {
	Name           string         `json:"name"`
	Description    string         `json:"description,omitempty"`
	CapabilityKeys []string       `json:"capability_keys,omitempty"`
	InputSchema    map[string]any `json:"input_schema,omitempty"`
}

type ChatTurnToolPolicyDecision struct {
	ToolName      string `json:"tool_name"`
	CapabilityKey string `json:"capability_key,omitempty"`
	Decision      string `json:"decision"`
	Reason        string `json:"reason"`
}

type ChatTurnExplainResponse struct {
	WorkspaceID        string                       `json:"workspace_id"`
	TaskClass          string                       `json:"task_class"`
	RequestedByActorID string                       `json:"requested_by_actor_id,omitempty"`
	SubjectActorID     string                       `json:"subject_actor_id,omitempty"`
	ActingAsActorID    string                       `json:"acting_as_actor_id,omitempty"`
	Channel            ChatTurnChannelContext       `json:"channel,omitempty"`
	ContractVersion    string                       `json:"contract_version"`
	SelectedRoute      ModelRouteExplainResponse    `json:"selected_route"`
	ToolCatalog        []ChatTurnToolCatalogEntry   `json:"tool_catalog"`
	PolicyDecisions    []ChatTurnToolPolicyDecision `json:"policy_decisions"`
}

type ChatTurnHistoryRequest struct {
	WorkspaceID     string `json:"workspace_id"`
	ChannelID       string `json:"channel_id,omitempty"`
	ConnectorID     string `json:"connector_id,omitempty"`
	ThreadID        string `json:"thread_id,omitempty"`
	CorrelationID   string `json:"correlation_id,omitempty"`
	BeforeCreatedAt string `json:"before_created_at,omitempty"`
	BeforeItemID    string `json:"before_item_id,omitempty"`
	Limit           int    `json:"limit,omitempty"`
}

type ChatTurnHistoryRecord struct {
	RecordID         string                     `json:"record_id"`
	TurnID           string                     `json:"turn_id"`
	WorkspaceID      string                     `json:"workspace_id"`
	TaskClass        string                     `json:"task_class"`
	CorrelationID    string                     `json:"correlation_id"`
	ChannelID        string                     `json:"channel_id"`
	ConnectorID      string                     `json:"connector_id,omitempty"`
	ThreadID         string                     `json:"thread_id,omitempty"`
	ItemIndex        int                        `json:"item_index"`
	Item             ChatTurnItem               `json:"item"`
	TaskRunReference ChatTurnTaskRunCorrelation `json:"task_run_reference"`
	CreatedAt        string                     `json:"created_at"`
}

type ChatTurnHistoryResponse struct {
	WorkspaceID         string                  `json:"workspace_id"`
	Items               []ChatTurnHistoryRecord `json:"items"`
	HasMore             bool                    `json:"has_more"`
	NextCursorCreatedAt string                  `json:"next_cursor_created_at,omitempty"`
	NextCursorItemID    string                  `json:"next_cursor_item_id,omitempty"`
}

type ChatPersonaPolicyRequest struct {
	WorkspaceID      string `json:"workspace_id"`
	PrincipalActorID string `json:"principal_actor_id,omitempty"`
	ChannelID        string `json:"channel_id,omitempty"`
}

type ChatPersonaPolicyUpsertRequest struct {
	WorkspaceID      string   `json:"workspace_id"`
	PrincipalActorID string   `json:"principal_actor_id,omitempty"`
	ChannelID        string   `json:"channel_id,omitempty"`
	StylePrompt      string   `json:"style_prompt"`
	Guardrails       []string `json:"guardrails,omitempty"`
}

type ChatPersonaPolicyResponse struct {
	WorkspaceID      string   `json:"workspace_id"`
	PrincipalActorID string   `json:"principal_actor_id,omitempty"`
	ChannelID        string   `json:"channel_id,omitempty"`
	StylePrompt      string   `json:"style_prompt"`
	Guardrails       []string `json:"guardrails,omitempty"`
	Source           string   `json:"source"`
	UpdatedAt        string   `json:"updated_at,omitempty"`
}

type AgentRunRequest struct {
	WorkspaceID            string             `json:"workspace_id"`
	RequestText            string             `json:"request_text"`
	NativeAction           *AgentNativeAction `json:"native_action,omitempty"`
	RequestedByActorID     string             `json:"requested_by_actor_id,omitempty"`
	SubjectActorID         string             `json:"subject_actor_id,omitempty"`
	ActingAsActorID        string             `json:"acting_as_actor_id,omitempty"`
	Origin                 string             `json:"origin,omitempty"`
	InAppApprovalConfirmed bool               `json:"in_app_approval_confirmed,omitempty"`
	CorrelationID          string             `json:"correlation_id,omitempty"`
	ApprovalPhrase         string             `json:"approval_phrase,omitempty"`
	PreferredAdapterID     string             `json:"preferred_adapter_id,omitempty"`
}

type AgentApproveRequest struct {
	WorkspaceID       string `json:"workspace_id"`
	ApprovalRequestID string `json:"approval_request_id"`
	Phrase            string `json:"phrase,omitempty"`
	DecisionByActorID string `json:"decision_by_actor_id"`
	CorrelationID     string `json:"correlation_id,omitempty"`
}

type AgentStepState struct {
	StepID        string            `json:"step_id"`
	StepIndex     int               `json:"step_index"`
	Name          string            `json:"name"`
	CapabilityKey string            `json:"capability_key"`
	AdapterID     string            `json:"adapter_id,omitempty"`
	Status        string            `json:"status"`
	Summary       string            `json:"summary"`
	Evidence      map[string]string `json:"evidence,omitempty"`
}

type AgentNativeAction struct {
	Connector string               `json:"connector"`
	Operation string               `json:"operation"`
	Mail      *AgentMailAction     `json:"mail,omitempty"`
	Calendar  *AgentCalendarAction `json:"calendar,omitempty"`
	Messages  *AgentMessagesAction `json:"messages,omitempty"`
	Browser   *AgentBrowserAction  `json:"browser,omitempty"`
	Finder    *AgentFinderAction   `json:"finder,omitempty"`
}

type AgentMailAction struct {
	Operation string `json:"operation"`
	Recipient string `json:"recipient,omitempty"`
	Subject   string `json:"subject,omitempty"`
	Body      string `json:"body,omitempty"`
	Limit     int    `json:"limit,omitempty"`
}

type AgentCalendarAction struct {
	Operation string `json:"operation"`
	EventID   string `json:"event_id,omitempty"`
	Title     string `json:"title,omitempty"`
	Notes     string `json:"notes,omitempty"`
}

type AgentMessagesAction struct {
	Operation string `json:"operation"`
	Channel   string `json:"channel,omitempty"`
	Recipient string `json:"recipient,omitempty"`
	Body      string `json:"body,omitempty"`
}

type AgentBrowserAction struct {
	Operation string `json:"operation"`
	TargetURL string `json:"target_url,omitempty"`
	Query     string `json:"query,omitempty"`
}

type AgentFinderAction struct {
	Operation  string `json:"operation"`
	TargetPath string `json:"target_path,omitempty"`
	Query      string `json:"query,omitempty"`
	RootPath   string `json:"root_path,omitempty"`
}

type AgentRunResponse struct {
	Workflow              string             `json:"workflow"`
	NativeAction          *AgentNativeAction `json:"native_action,omitempty"`
	TaskID                string             `json:"task_id"`
	RunID                 string             `json:"run_id"`
	TaskState             string             `json:"task_state"`
	RunState              string             `json:"run_state"`
	ClarificationRequired bool               `json:"clarification_required,omitempty"`
	ClarificationPrompt   string             `json:"clarification_prompt,omitempty"`
	MissingSlots          []string           `json:"missing_slots,omitempty"`
	ApprovalRequired      bool               `json:"approval_required,omitempty"`
	ApprovalRequestID     string             `json:"approval_request_id,omitempty"`
	StepStates            []AgentStepState   `json:"step_states"`
}
