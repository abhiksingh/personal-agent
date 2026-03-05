package transport

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

type ControlBackend interface {
	SubmitTask(ctx context.Context, request SubmitTaskRequest, correlationID string) (SubmitTaskResponse, error)
	TaskStatus(ctx context.Context, taskID string, correlationID string) (TaskStatusResponse, error)
	CancelTask(ctx context.Context, request TaskCancelRequest, correlationID string) (TaskCancelResponse, error)
	RetryTask(ctx context.Context, request TaskRetryRequest, correlationID string) (TaskRetryResponse, error)
	RequeueTask(ctx context.Context, request TaskRequeueRequest, correlationID string) (TaskRequeueResponse, error)
	CapabilitySmoke(ctx context.Context, correlationID string) (CapabilitySmokeResponse, error)
}

type inMemoryTaskRecord struct {
	WorkspaceID string
	TaskID      string
	RunID       string
	State       string
	LastError   string
	UpdatedAt   time.Time
}

type InMemoryControlBackend struct {
	mu          sync.RWMutex
	tasks       map[string]inMemoryTaskRecord
	replayTasks map[string]SubmitTaskResponse
	version     string
	channels    []string
	connectors  []string
	eventBroker *EventBroker
}

func NewInMemoryControlBackend(eventBroker *EventBroker) *InMemoryControlBackend {
	return &InMemoryControlBackend{
		tasks:       map[string]inMemoryTaskRecord{},
		replayTasks: map[string]SubmitTaskResponse{},
		version:     "0.1.0",
		channels:    DefaultCapabilitySmokeChannels(),
		connectors:  DefaultCapabilitySmokeConnectors(),
		eventBroker: eventBroker,
	}
}

func (b *InMemoryControlBackend) SubmitTask(_ context.Context, request SubmitTaskRequest, correlationID string) (SubmitTaskResponse, error) {
	if strings.TrimSpace(request.WorkspaceID) == "" {
		return SubmitTaskResponse{}, errors.New("workspace_id is required")
	}
	if strings.TrimSpace(request.RequestedByActorID) == "" {
		return SubmitTaskResponse{}, errors.New("requested_by_actor_id is required")
	}
	if strings.TrimSpace(request.SubjectPrincipalActorID) == "" {
		return SubmitTaskResponse{}, errors.New("subject_principal_actor_id is required")
	}
	if strings.TrimSpace(request.Title) == "" {
		return SubmitTaskResponse{}, errors.New("title is required")
	}

	replayKey := strings.TrimSpace(correlationID)
	if replayKey != "" {
		b.mu.RLock()
		replayResponse, exists := b.replayTasks[replayKey]
		b.mu.RUnlock()
		if exists {
			return replayResponse, nil
		}
	}

	taskID, err := randomID()
	if err != nil {
		return SubmitTaskResponse{}, err
	}
	runID, err := randomID()
	if err != nil {
		return SubmitTaskResponse{}, err
	}
	now := time.Now().UTC()

	response := SubmitTaskResponse{
		TaskID:        taskID,
		RunID:         runID,
		State:         "queued",
		CorrelationID: correlationID,
	}

	b.mu.Lock()
	b.tasks[taskID] = inMemoryTaskRecord{
		WorkspaceID: strings.TrimSpace(request.WorkspaceID),
		TaskID:      taskID,
		RunID:       runID,
		State:       "queued",
		LastError:   "",
		UpdatedAt:   now,
	}
	if replayKey != "" {
		b.replayTasks[replayKey] = response
	}
	b.mu.Unlock()

	if b.eventBroker != nil {
		_ = b.eventBroker.Publish(RealtimeEventEnvelope{
			EventID:       mustRandomID(),
			EventType:     "task_submitted",
			OccurredAt:    now,
			CorrelationID: correlationID,
			Payload: RealtimeEventPayload{
				TaskID: taskID,
				RunID:  runID,
				State:  "queued",
			},
		})
		_ = b.eventBroker.Publish(RealtimeEventEnvelope{
			EventID:       mustRandomID(),
			EventType:     "task_run_lifecycle",
			OccurredAt:    now,
			CorrelationID: correlationID,
			Payload: RealtimeEventPayload{
				TaskID:          taskID,
				RunID:           runID,
				TaskState:       "queued",
				RunState:        "queued",
				LifecycleState:  "queued",
				LifecycleSource: "control_backend_submit",
			},
		})
	}

	return response, nil
}

func (b *InMemoryControlBackend) TaskStatus(_ context.Context, taskID string, correlationID string) (TaskStatusResponse, error) {
	b.mu.RLock()
	record, ok := b.tasks[taskID]
	b.mu.RUnlock()
	if !ok {
		return TaskStatusResponse{}, NewTaskControlNotFoundError(fmt.Sprintf("task not found: %s", taskID))
	}

	return TaskStatusResponse{
		TaskID:        record.TaskID,
		RunID:         record.RunID,
		State:         record.State,
		RunState:      record.State,
		LastError:     strings.TrimSpace(record.LastError),
		Actions:       ResolveTaskRunActionAvailability(record.State, record.State),
		UpdatedAt:     record.UpdatedAt,
		CorrelationID: correlationID,
	}, nil
}

func (b *InMemoryControlBackend) CancelTask(_ context.Context, request TaskCancelRequest, correlationID string) (TaskCancelResponse, error) {
	taskID := strings.TrimSpace(request.TaskID)
	runID := strings.TrimSpace(request.RunID)
	workspaceID := strings.TrimSpace(request.WorkspaceID)
	if taskID == "" && runID == "" {
		return TaskCancelResponse{}, NewTaskControlMissingReferenceError("task_id or run_id is required")
	}

	b.mu.Lock()
	response, eventToPublish, err := func() (TaskCancelResponse, *RealtimeEventEnvelope, error) {
		resolvedTaskID := taskID
		var record inMemoryTaskRecord
		var found bool
		switch {
		case resolvedTaskID != "":
			record, found = b.tasks[resolvedTaskID]
		case runID != "":
			for candidateTaskID, candidate := range b.tasks {
				if strings.TrimSpace(candidate.RunID) != runID {
					continue
				}
				resolvedTaskID = candidateTaskID
				record = candidate
				found = true
				break
			}
		}
		if !found {
			return TaskCancelResponse{}, nil, NewTaskControlNotFoundError("task run not found")
		}
		if runID != "" && strings.TrimSpace(record.RunID) != runID {
			return TaskCancelResponse{}, nil, NewTaskControlReferenceMismatchError("task and run id mismatch")
		}
		if workspaceID != "" && !strings.EqualFold(strings.TrimSpace(record.WorkspaceID), workspaceID) {
			return TaskCancelResponse{}, nil, NewTaskControlReferenceMismatchError("workspace mismatch for task/run")
		}

		previousTaskState := normalizeTaskLifecycleState(record.State)
		previousRunState := previousTaskState
		finalTaskState := previousTaskState
		finalRunState := previousRunState
		cancelled := false
		alreadyTerminal := isTerminalTaskLifecycleState(previousRunState)
		reason := strings.TrimSpace(request.Reason)
		if reason == "" {
			reason = "cancel requested by control api"
		}

		var event *RealtimeEventEnvelope
		if !alreadyTerminal {
			record.State = "cancelled"
			record.LastError = strings.TrimSpace(reason)
			record.UpdatedAt = time.Now().UTC()
			b.tasks[resolvedTaskID] = record
			finalTaskState = "cancelled"
			finalRunState = "cancelled"
			cancelled = true

			event = &RealtimeEventEnvelope{
				EventID:       mustRandomID(),
				EventType:     "task_run_lifecycle",
				OccurredAt:    record.UpdatedAt,
				CorrelationID: correlationID,
				Payload: RealtimeEventPayload{
					WorkspaceID:     record.WorkspaceID,
					TaskID:          resolvedTaskID,
					RunID:           record.RunID,
					TaskState:       finalTaskState,
					RunState:        finalRunState,
					LifecycleState:  finalRunState,
					LifecycleSource: "control_backend_cancel",
					LastError:       reason,
				},
			}
		}

		return TaskCancelResponse{
			WorkspaceID:       record.WorkspaceID,
			TaskID:            resolvedTaskID,
			RunID:             record.RunID,
			PreviousTaskState: previousTaskState,
			PreviousRunState:  previousRunState,
			TaskState:         finalTaskState,
			RunState:          finalRunState,
			Cancelled:         cancelled,
			AlreadyTerminal:   alreadyTerminal,
			Reason:            reason,
			CorrelationID:     correlationID,
		}, event, nil
	}()
	b.mu.Unlock()
	if err != nil {
		return TaskCancelResponse{}, err
	}
	if eventToPublish != nil && b.eventBroker != nil {
		if publishErr := b.eventBroker.Publish(*eventToPublish); publishErr != nil {
			return TaskCancelResponse{}, publishErr
		}
	}
	return response, nil
}

func (b *InMemoryControlBackend) RetryTask(_ context.Context, request TaskRetryRequest, correlationID string) (TaskRetryResponse, error) {
	taskID := strings.TrimSpace(request.TaskID)
	runID := strings.TrimSpace(request.RunID)
	workspaceID := strings.TrimSpace(request.WorkspaceID)
	if taskID == "" && runID == "" {
		return TaskRetryResponse{}, NewTaskControlMissingReferenceError("task_id or run_id is required")
	}

	b.mu.Lock()
	response, eventToPublish, err := func() (TaskRetryResponse, *RealtimeEventEnvelope, error) {
		resolvedTaskID := taskID
		var record inMemoryTaskRecord
		var found bool
		switch {
		case resolvedTaskID != "":
			record, found = b.tasks[resolvedTaskID]
		case runID != "":
			for candidateTaskID, candidate := range b.tasks {
				if strings.TrimSpace(candidate.RunID) != runID {
					continue
				}
				resolvedTaskID = candidateTaskID
				record = candidate
				found = true
				break
			}
		}
		if !found {
			return TaskRetryResponse{}, nil, NewTaskControlNotFoundError("task run not found")
		}
		if runID != "" && strings.TrimSpace(record.RunID) != runID {
			return TaskRetryResponse{}, nil, NewTaskControlReferenceMismatchError("task and run id mismatch")
		}
		if workspaceID != "" && !strings.EqualFold(strings.TrimSpace(record.WorkspaceID), workspaceID) {
			return TaskRetryResponse{}, nil, NewTaskControlReferenceMismatchError("workspace mismatch for task/run")
		}

		previousTaskState := normalizeTaskLifecycleState(record.State)
		previousRunState := previousTaskState
		if previousRunState != "failed" && previousRunState != "cancelled" {
			return TaskRetryResponse{}, nil, NewTaskControlStateConflictError(fmt.Sprintf("task run state %q is not retryable", previousRunState))
		}
		previousRunID := strings.TrimSpace(record.RunID)

		newRunID, err := randomID()
		if err != nil {
			return TaskRetryResponse{}, nil, err
		}
		now := time.Now().UTC()
		reason := strings.TrimSpace(request.Reason)
		if reason == "" {
			reason = "retry requested by control api"
		}

		record.RunID = newRunID
		record.State = "queued"
		record.LastError = ""
		record.UpdatedAt = now
		b.tasks[resolvedTaskID] = record

		return TaskRetryResponse{
				WorkspaceID:       record.WorkspaceID,
				TaskID:            resolvedTaskID,
				PreviousRunID:     previousRunID,
				RunID:             newRunID,
				PreviousTaskState: previousTaskState,
				PreviousRunState:  previousRunState,
				TaskState:         "queued",
				RunState:          "queued",
				Retried:           true,
				Reason:            reason,
				Actions:           ResolveTaskRunActionAvailability("queued", "queued"),
				CorrelationID:     correlationID,
			}, &RealtimeEventEnvelope{
				EventID:       mustRandomID(),
				EventType:     "task_run_lifecycle",
				OccurredAt:    now,
				CorrelationID: correlationID,
				Payload: RealtimeEventPayload{
					WorkspaceID:     record.WorkspaceID,
					TaskID:          resolvedTaskID,
					RunID:           newRunID,
					TaskState:       "queued",
					RunState:        "queued",
					LifecycleState:  "queued",
					LifecycleSource: "control_backend_retry",
					LastError:       reason,
				},
			}, nil
	}()
	b.mu.Unlock()
	if err != nil {
		return TaskRetryResponse{}, err
	}
	if eventToPublish != nil && b.eventBroker != nil {
		_ = b.eventBroker.Publish(*eventToPublish)
	}
	return response, nil
}

func (b *InMemoryControlBackend) RequeueTask(_ context.Context, request TaskRequeueRequest, correlationID string) (TaskRequeueResponse, error) {
	taskID := strings.TrimSpace(request.TaskID)
	runID := strings.TrimSpace(request.RunID)
	workspaceID := strings.TrimSpace(request.WorkspaceID)
	if taskID == "" && runID == "" {
		return TaskRequeueResponse{}, NewTaskControlMissingReferenceError("task_id or run_id is required")
	}

	b.mu.Lock()
	response, eventsToPublish, err := func() (TaskRequeueResponse, []RealtimeEventEnvelope, error) {
		resolvedTaskID := taskID
		var record inMemoryTaskRecord
		var found bool
		switch {
		case resolvedTaskID != "":
			record, found = b.tasks[resolvedTaskID]
		case runID != "":
			for candidateTaskID, candidate := range b.tasks {
				if strings.TrimSpace(candidate.RunID) != runID {
					continue
				}
				resolvedTaskID = candidateTaskID
				record = candidate
				found = true
				break
			}
		}
		if !found {
			return TaskRequeueResponse{}, nil, NewTaskControlNotFoundError("task run not found")
		}
		if runID != "" && strings.TrimSpace(record.RunID) != runID {
			return TaskRequeueResponse{}, nil, NewTaskControlReferenceMismatchError("task and run id mismatch")
		}
		if workspaceID != "" && !strings.EqualFold(strings.TrimSpace(record.WorkspaceID), workspaceID) {
			return TaskRequeueResponse{}, nil, NewTaskControlReferenceMismatchError("workspace mismatch for task/run")
		}

		previousTaskState := normalizeTaskLifecycleState(record.State)
		previousRunState := previousTaskState
		switch previousRunState {
		case "queued", "planning", "awaiting_approval", "blocked":
		default:
			return TaskRequeueResponse{}, nil, NewTaskControlStateConflictError(fmt.Sprintf("task run state %q is not requeueable", previousRunState))
		}

		previousRunID := strings.TrimSpace(record.RunID)
		newRunID, err := randomID()
		if err != nil {
			return TaskRequeueResponse{}, nil, err
		}
		now := time.Now().UTC()
		reason := strings.TrimSpace(request.Reason)
		if reason == "" {
			reason = "requeue requested by control api"
		}

		record.RunID = newRunID
		record.State = "queued"
		record.LastError = ""
		record.UpdatedAt = now
		b.tasks[resolvedTaskID] = record

		events := []RealtimeEventEnvelope{
			{
				EventID:       mustRandomID(),
				EventType:     "task_run_lifecycle",
				OccurredAt:    now,
				CorrelationID: correlationID,
				Payload: RealtimeEventPayload{
					WorkspaceID:     record.WorkspaceID,
					TaskID:          resolvedTaskID,
					RunID:           previousRunID,
					TaskState:       "cancelled",
					RunState:        "cancelled",
					LifecycleState:  "cancelled",
					LifecycleSource: "control_backend_requeue",
					LastError:       reason,
				},
			},
			{
				EventID:       mustRandomID(),
				EventType:     "task_run_lifecycle",
				OccurredAt:    now,
				CorrelationID: correlationID,
				Payload: RealtimeEventPayload{
					WorkspaceID:     record.WorkspaceID,
					TaskID:          resolvedTaskID,
					RunID:           newRunID,
					TaskState:       "queued",
					RunState:        "queued",
					LifecycleState:  "queued",
					LifecycleSource: "control_backend_requeue",
					LastError:       reason,
				},
			},
		}

		return TaskRequeueResponse{
			WorkspaceID:       record.WorkspaceID,
			TaskID:            resolvedTaskID,
			PreviousRunID:     previousRunID,
			RunID:             newRunID,
			PreviousTaskState: previousTaskState,
			PreviousRunState:  previousRunState,
			TaskState:         "queued",
			RunState:          "queued",
			Requeued:          true,
			Reason:            reason,
			Actions:           ResolveTaskRunActionAvailability("queued", "queued"),
			CorrelationID:     correlationID,
		}, events, nil
	}()
	b.mu.Unlock()
	if err != nil {
		return TaskRequeueResponse{}, err
	}
	if b.eventBroker != nil {
		for _, event := range eventsToPublish {
			_ = b.eventBroker.Publish(event)
		}
	}
	return response, nil
}

func (b *InMemoryControlBackend) CapabilitySmoke(_ context.Context, correlationID string) (CapabilitySmokeResponse, error) {
	return CapabilitySmokeResponse{
		DaemonVersion: b.version,
		Channels:      append([]string{}, b.channels...),
		Connectors:    append([]string{}, b.connectors...),
		Healthy:       true,
		CorrelationID: correlationID,
	}, nil
}

func randomID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

func mustRandomID() string {
	id, err := randomID()
	if err != nil {
		return fmt.Sprintf("fallback-%d", time.Now().UTC().UnixNano())
	}
	return id
}

func normalizeTaskLifecycleState(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

func isTerminalTaskLifecycleState(state string) bool {
	switch normalizeTaskLifecycleState(state) {
	case "completed", "failed", "cancelled":
		return true
	default:
		return false
	}
}
