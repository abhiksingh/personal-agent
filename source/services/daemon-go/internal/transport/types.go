package transport

import (
	"crypto/tls"
	"time"
)

type ListenerMode string

const (
	ListenerModeTCP       ListenerMode = "tcp"
	ListenerModeUnix      ListenerMode = "unix"
	ListenerModeNamedPipe ListenerMode = "named_pipe"
)

const (
	DefaultTCPAddress       = "127.0.0.1:7071"
	DefaultNamedPipeAddress = `\\.\pipe\personal-agent`
)

type ServerConfig struct {
	ListenerMode                ListenerMode
	Address                     string
	RuntimeProfile              string
	AuthToken                   string
	AuthTokenScopes             []string
	WebSocketOriginAllowlist    []string
	TLSConfig                   *tls.Config
	ReadHeaderTimeout           time.Duration
	ReadTimeout                 time.Duration
	WriteTimeout                time.Duration
	IdleTimeout                 time.Duration
	MaxHeaderBytes              int
	RequestBodyBytesLimit       int64
	ControlRateLimitWindow      time.Duration
	ControlRateLimitMaxRequests int
	RealtimeReadLimitBytes      int64
	RealtimeWriteTimeout        time.Duration
	RealtimePongTimeout         time.Duration
	RealtimePingInterval        time.Duration
	RealtimeMaxConnections      int
	RealtimeMaxSubscriptions    int
	DaemonLifecycle             DaemonLifecycleService
	WorkflowQueries             WorkflowQueryService
	SecretReferences            SecretReferenceService
	Providers                   ProviderService
	Models                      ModelService
	Chat                        ChatService
	Agent                       AgentService
	Delegation                  DelegationService
	Comm                        CommService
	Twilio                      TwilioChannelService
	Cloudflared                 CloudflaredConnectorService
	Automation                  AutomationService
	Inspect                     InspectService
	Retention                   RetentionService
	ContextOps                  ContextOpsService
	UIStatus                    UIStatusService
	IdentityDirectory           IdentityDirectoryService
}

type SubmitTaskRequest struct {
	WorkspaceID             string `json:"workspace_id"`
	RequestedByActorID      string `json:"requested_by_actor_id"`
	SubjectPrincipalActorID string `json:"subject_principal_actor_id"`
	Title                   string `json:"title"`
	Description             string `json:"description,omitempty"`
	TaskClass               string `json:"task_class,omitempty"`
}

type SubmitTaskResponse struct {
	TaskID        string `json:"task_id"`
	RunID         string `json:"run_id"`
	State         string `json:"state"`
	CorrelationID string `json:"correlation_id"`
}

type TaskRunActionAvailability struct {
	CanCancel  bool `json:"can_cancel"`
	CanRetry   bool `json:"can_retry"`
	CanRequeue bool `json:"can_requeue"`
}

type TaskStatusResponse struct {
	TaskID        string                    `json:"task_id"`
	RunID         string                    `json:"run_id,omitempty"`
	State         string                    `json:"state"`
	RunState      string                    `json:"run_state,omitempty"`
	LastError     string                    `json:"last_error,omitempty"`
	Actions       TaskRunActionAvailability `json:"actions"`
	UpdatedAt     time.Time                 `json:"updated_at"`
	CorrelationID string                    `json:"correlation_id"`
}

type TaskCancelRequest struct {
	WorkspaceID string `json:"workspace_id,omitempty"`
	TaskID      string `json:"task_id,omitempty"`
	RunID       string `json:"run_id,omitempty"`
	Reason      string `json:"reason,omitempty"`
}

type TaskCancelResponse struct {
	WorkspaceID       string `json:"workspace_id"`
	TaskID            string `json:"task_id"`
	RunID             string `json:"run_id"`
	PreviousTaskState string `json:"previous_task_state"`
	PreviousRunState  string `json:"previous_run_state"`
	TaskState         string `json:"task_state"`
	RunState          string `json:"run_state"`
	Cancelled         bool   `json:"cancelled"`
	AlreadyTerminal   bool   `json:"already_terminal"`
	Reason            string `json:"reason,omitempty"`
	CorrelationID     string `json:"correlation_id"`
}

type TaskRetryRequest struct {
	WorkspaceID string `json:"workspace_id,omitempty"`
	TaskID      string `json:"task_id,omitempty"`
	RunID       string `json:"run_id,omitempty"`
	Reason      string `json:"reason,omitempty"`
}

type TaskRetryResponse struct {
	WorkspaceID       string                    `json:"workspace_id"`
	TaskID            string                    `json:"task_id"`
	PreviousRunID     string                    `json:"previous_run_id"`
	RunID             string                    `json:"run_id"`
	PreviousTaskState string                    `json:"previous_task_state"`
	PreviousRunState  string                    `json:"previous_run_state"`
	TaskState         string                    `json:"task_state"`
	RunState          string                    `json:"run_state"`
	Retried           bool                      `json:"retried"`
	Reason            string                    `json:"reason,omitempty"`
	Actions           TaskRunActionAvailability `json:"actions"`
	CorrelationID     string                    `json:"correlation_id"`
}

type TaskRequeueRequest struct {
	WorkspaceID string `json:"workspace_id,omitempty"`
	TaskID      string `json:"task_id,omitempty"`
	RunID       string `json:"run_id,omitempty"`
	Reason      string `json:"reason,omitempty"`
}

type TaskRequeueResponse struct {
	WorkspaceID       string                    `json:"workspace_id"`
	TaskID            string                    `json:"task_id"`
	PreviousRunID     string                    `json:"previous_run_id"`
	RunID             string                    `json:"run_id"`
	PreviousTaskState string                    `json:"previous_task_state"`
	PreviousRunState  string                    `json:"previous_run_state"`
	TaskState         string                    `json:"task_state"`
	RunState          string                    `json:"run_state"`
	Requeued          bool                      `json:"requeued"`
	Reason            string                    `json:"reason,omitempty"`
	Actions           TaskRunActionAvailability `json:"actions"`
	CorrelationID     string                    `json:"correlation_id"`
}

type CapabilitySmokeResponse struct {
	DaemonVersion string   `json:"daemon_version"`
	Channels      []string `json:"channels"`
	Connectors    []string `json:"connectors"`
	Healthy       bool     `json:"healthy"`
	CorrelationID string   `json:"correlation_id"`
}

type RealtimeEventEnvelope struct {
	EventID                string               `json:"event_id"`
	Sequence               int64                `json:"sequence"`
	EventType              string               `json:"event_type"`
	OccurredAt             time.Time            `json:"occurred_at"`
	CorrelationID          string               `json:"correlation_id,omitempty"`
	ContractVersion        string               `json:"contract_version,omitempty"`
	LifecycleSchemaVersion string               `json:"lifecycle_schema_version,omitempty"`
	Payload                RealtimeEventPayload `json:"payload"`
}

type ClientSignal struct {
	SignalType    string `json:"signal_type"`
	TaskID        string `json:"task_id,omitempty"`
	RunID         string `json:"run_id,omitempty"`
	Reason        string `json:"reason,omitempty"`
	CorrelationID string `json:"correlation_id,omitempty"`
}
