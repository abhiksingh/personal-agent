package contracts

import "context"

type AdapterKind string

const (
	AdapterKindChannel   AdapterKind = "channel"
	AdapterKindConnector AdapterKind = "connector"
)

type CapabilityDescriptor struct {
	Key         string `json:"key"`
	Description string `json:"description,omitempty"`
}

type AdapterMetadata struct {
	ID           string                 `json:"id"`
	Kind         AdapterKind            `json:"kind"`
	DisplayName  string                 `json:"display_name"`
	Version      string                 `json:"version"`
	Capabilities []CapabilityDescriptor `json:"capabilities"`
	Runtime      map[string]string      `json:"runtime,omitempty"`
}

type ExecutionContext struct {
	WorkspaceID       string `json:"workspace_id"`
	TaskID            string `json:"task_id"`
	RunID             string `json:"run_id"`
	StepID            string `json:"step_id"`
	CorrelationID     string `json:"correlation_id"`
	RequestedByActor  string `json:"requested_by_actor_id"`
	SubjectPrincipal  string `json:"subject_principal_actor_id"`
	ActingAsActor     string `json:"acting_as_actor_id"`
	SourceChannel     string `json:"source_channel,omitempty"`
	ApprovalReference string `json:"approval_reference,omitempty"`
}

type StepExecutionResult struct {
	Status      TaskStepStatus    `json:"status"`
	Summary     string            `json:"summary,omitempty"`
	Evidence    map[string]string `json:"evidence,omitempty"`
	Output      map[string]any    `json:"output,omitempty"`
	Retryable   bool              `json:"retryable"`
	DurationMs  int64             `json:"duration_ms,omitempty"`
	ErrorReason string            `json:"error_reason,omitempty"`
}

type Adapter interface {
	Metadata() AdapterMetadata
	HealthCheck(ctx context.Context) error
}

type ChannelAdapter interface {
	Adapter
	ExecuteStep(ctx context.Context, execCtx ExecutionContext, step TaskStep) (StepExecutionResult, error)
}

type ConnectorAdapter interface {
	Adapter
	ExecuteStep(ctx context.Context, execCtx ExecutionContext, step TaskStep) (StepExecutionResult, error)
}
