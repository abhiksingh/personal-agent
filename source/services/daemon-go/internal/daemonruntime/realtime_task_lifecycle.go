package daemonruntime

import (
	"strings"
	"time"

	"personalagent/runtime/internal/transport"
)

const (
	realtimeEventTypeTaskRunLifecycle      = "task_run_lifecycle"
	taskRunLifecycleSourceControlSubmit    = "control_backend_submit"
	taskRunLifecycleSourceControlCancel    = "control_backend_cancel"
	taskRunLifecycleSourceControlRetry     = "control_backend_retry"
	taskRunLifecycleSourceControlRequeue   = "control_backend_requeue"
	taskRunLifecycleSourceQueuedTaskWorker = "queued_task_runtime"
)

func publishTaskRunLifecycleEvent(
	broker *transport.EventBroker,
	correlationID string,
	workspaceID string,
	taskID string,
	runID string,
	taskState string,
	runState string,
	source string,
	lastError string,
	occurredAt time.Time,
) {
	if broker == nil {
		return
	}

	normalizedTaskState := strings.TrimSpace(taskState)
	normalizedRunState := strings.TrimSpace(runState)
	if normalizedRunState == "" {
		normalizedRunState = normalizedTaskState
	}
	if normalizedTaskState == "" {
		normalizedTaskState = normalizedRunState
	}
	if normalizedRunState == "" {
		return
	}
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}

	payload := transport.RealtimeEventPayload{
		WorkspaceID:     strings.TrimSpace(workspaceID),
		TaskID:          strings.TrimSpace(taskID),
		RunID:           strings.TrimSpace(runID),
		TaskState:       normalizedTaskState,
		RunState:        normalizedRunState,
		LifecycleState:  normalizedRunState,
		LifecycleSource: strings.TrimSpace(source),
		LastError:       strings.TrimSpace(lastError),
	}

	_ = broker.Publish(transport.RealtimeEventEnvelope{
		EventID:       controlBackendMustRandomID(),
		EventType:     realtimeEventTypeTaskRunLifecycle,
		OccurredAt:    occurredAt.UTC(),
		CorrelationID: strings.TrimSpace(correlationID),
		Payload:       payload,
	})
}
