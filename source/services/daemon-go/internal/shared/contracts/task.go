package contracts

import "fmt"

type TaskState string

const (
	TaskStateQueued           TaskState = "queued"
	TaskStatePlanning         TaskState = "planning"
	TaskStateAwaitingApproval TaskState = "awaiting_approval"
	TaskStateRunning          TaskState = "running"
	TaskStateBlocked          TaskState = "blocked"
	TaskStateCompleted        TaskState = "completed"
	TaskStateFailed           TaskState = "failed"
	TaskStateCancelled        TaskState = "cancelled"
)

func (s TaskState) IsTerminal() bool {
	return s == TaskStateCompleted || s == TaskStateFailed || s == TaskStateCancelled
}

type Task struct {
	ID                 string    `json:"id"`
	WorkspaceID        string    `json:"workspace_id"`
	RequestedByActorID string    `json:"requested_by_actor_id"`
	SubjectPrincipalID string    `json:"subject_principal_actor_id"`
	Title              string    `json:"title"`
	Description        string    `json:"description,omitempty"`
	Priority           int       `json:"priority"`
	DeadlineAt         string    `json:"deadline_at,omitempty"`
	Channel            string    `json:"channel,omitempty"`
	State              TaskState `json:"state"`
	CreatedAt          string    `json:"created_at"`
	UpdatedAt          string    `json:"updated_at"`
}

func (t Task) Validate() error {
	if t.ID == "" {
		return fmt.Errorf("task.id is required")
	}
	if t.WorkspaceID == "" {
		return fmt.Errorf("task.workspace_id is required")
	}
	if t.RequestedByActorID == "" {
		return fmt.Errorf("task.requested_by_actor_id is required")
	}
	if t.SubjectPrincipalID == "" {
		return fmt.Errorf("task.subject_principal_actor_id is required")
	}
	if t.State == "" {
		return fmt.Errorf("task.state is required")
	}
	return nil
}

type TaskRun struct {
	ID             string    `json:"id"`
	WorkspaceID    string    `json:"workspace_id"`
	TaskID         string    `json:"task_id"`
	ActingAsActor  string    `json:"acting_as_actor_id"`
	State          TaskState `json:"state"`
	StartedAt      string    `json:"started_at,omitempty"`
	FinishedAt     string    `json:"finished_at,omitempty"`
	LastError      string    `json:"last_error,omitempty"`
	CorrelationID  string    `json:"correlation_id,omitempty"`
	RequestedBy    string    `json:"requested_by_actor_id,omitempty"`
	SubjectContext string    `json:"subject_principal_actor_id,omitempty"`
}

func (r TaskRun) Validate() error {
	if r.ID == "" {
		return fmt.Errorf("task_run.id is required")
	}
	if r.WorkspaceID == "" {
		return fmt.Errorf("task_run.workspace_id is required")
	}
	if r.TaskID == "" {
		return fmt.Errorf("task_run.task_id is required")
	}
	if r.ActingAsActor == "" {
		return fmt.Errorf("task_run.acting_as_actor_id is required")
	}
	if r.State == "" {
		return fmt.Errorf("task_run.state is required")
	}
	return nil
}

type TaskStepStatus string

const (
	TaskStepStatusPending   TaskStepStatus = "pending"
	TaskStepStatusRunning   TaskStepStatus = "running"
	TaskStepStatusCompleted TaskStepStatus = "completed"
	TaskStepStatusFailed    TaskStepStatus = "failed"
	TaskStepStatusSkipped   TaskStepStatus = "skipped"
)

type TaskStep struct {
	ID               string         `json:"id"`
	RunID            string         `json:"run_id"`
	StepIndex        int            `json:"step_index"`
	Name             string         `json:"name"`
	Status           TaskStepStatus `json:"status"`
	CapabilityKey    string         `json:"capability_key,omitempty"`
	Input            map[string]any `json:"input,omitempty"`
	InteractionLevel string         `json:"interaction_level,omitempty"`
	TimeoutSeconds   int            `json:"timeout_seconds,omitempty"`
	RetryMax         int            `json:"retry_max,omitempty"`
	RetryCount       int            `json:"retry_count,omitempty"`
	LastError        string         `json:"last_error,omitempty"`
	CreatedAt        string         `json:"created_at,omitempty"`
	UpdatedAt        string         `json:"updated_at,omitempty"`
}

func (s TaskStep) Validate() error {
	if s.ID == "" {
		return fmt.Errorf("task_step.id is required")
	}
	if s.RunID == "" {
		return fmt.Errorf("task_step.run_id is required")
	}
	if s.StepIndex < 0 {
		return fmt.Errorf("task_step.step_index must be >= 0")
	}
	if s.Name == "" {
		return fmt.Errorf("task_step.name is required")
	}
	if s.Status == "" {
		return fmt.Errorf("task_step.status is required")
	}
	return nil
}
