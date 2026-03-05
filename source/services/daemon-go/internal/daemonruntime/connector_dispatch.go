package daemonruntime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	connectorcontract "personalagent/runtime/internal/connectors/contract"
	connectorregistry "personalagent/runtime/internal/connectors/registry"
	shared "personalagent/runtime/internal/shared/contracts"
)

const connectorRuntimeExecAddressKey = "exec_address"

type ConnectorStepDispatcher interface {
	ResolveAdapter(capabilityKey string, preferredAdapterID string) (connectorcontract.Metadata, error)
	ExecuteStep(ctx context.Context, adapterID string, execCtx connectorcontract.ExecutionContext, step connectorcontract.TaskStep) (connectorcontract.StepExecutionResult, error)
}

type RegistryConnectorStepDispatcher struct {
	registry *connectorregistry.Registry
}

func NewRegistryConnectorStepDispatcher(registry *connectorregistry.Registry) *RegistryConnectorStepDispatcher {
	return &RegistryConnectorStepDispatcher{registry: registry}
}

func (d *RegistryConnectorStepDispatcher) ResolveAdapter(capabilityKey string, preferredAdapterID string) (connectorcontract.Metadata, error) {
	if d.registry == nil {
		return connectorcontract.Metadata{}, fmt.Errorf("connector registry is required")
	}
	adapter, err := d.registry.SelectByCapability(capabilityKey, preferredAdapterID)
	if err != nil {
		return connectorcontract.Metadata{}, err
	}
	return adapter.Metadata(), nil
}

func (d *RegistryConnectorStepDispatcher) ExecuteStep(ctx context.Context, adapterID string, execCtx connectorcontract.ExecutionContext, step connectorcontract.TaskStep) (connectorcontract.StepExecutionResult, error) {
	if d.registry == nil {
		return connectorcontract.StepExecutionResult{}, fmt.Errorf("connector registry is required")
	}
	trimmedID := strings.TrimSpace(adapterID)
	if trimmedID == "" {
		return connectorcontract.StepExecutionResult{}, fmt.Errorf("adapter id is required")
	}
	adapter, ok := d.registry.Get(trimmedID)
	if !ok {
		return connectorcontract.StepExecutionResult{}, fmt.Errorf("connector adapter not registered: %s", trimmedID)
	}
	return adapter.ExecuteStep(ctx, execCtx, step)
}

type SupervisorConnectorStepDispatcher struct {
	supervisor      PluginSupervisor
	httpClient      *http.Client
	restartBackoff  time.Duration
	restartDeadline time.Duration
	resilience      *workerDispatchResilience
}

func NewSupervisorConnectorStepDispatcher(supervisor PluginSupervisor, registry *connectorregistry.Registry) *SupervisorConnectorStepDispatcher {
	_ = registry
	return &SupervisorConnectorStepDispatcher{
		supervisor:      supervisor,
		httpClient:      &http.Client{Timeout: 4 * time.Second},
		restartBackoff:  50 * time.Millisecond,
		restartDeadline: 4 * time.Second,
		resilience:      defaultWorkerDispatchResilience(),
	}
}

func (d *SupervisorConnectorStepDispatcher) ResolveAdapter(capabilityKey string, preferredAdapterID string) (connectorcontract.Metadata, error) {
	metadata, found, err := d.resolveWorkerAdapter(capabilityKey, preferredAdapterID)
	if err != nil {
		return connectorcontract.Metadata{}, err
	}
	if found {
		return metadata, nil
	}
	return connectorcontract.Metadata{}, fmt.Errorf("no connector worker supports capability: %s", strings.TrimSpace(capabilityKey))
}

func (d *SupervisorConnectorStepDispatcher) ExecuteStep(ctx context.Context, adapterID string, execCtx connectorcontract.ExecutionContext, step connectorcontract.TaskStep) (connectorcontract.StepExecutionResult, error) {
	trimmedID := strings.TrimSpace(adapterID)
	if trimmedID == "" {
		return connectorcontract.StepExecutionResult{}, fmt.Errorf("adapter id is required")
	}
	if d.supervisor == nil {
		return connectorcontract.StepExecutionResult{}, fmt.Errorf("plugin supervisor is required")
	}

	status, ok := d.supervisor.WorkerStatus(trimmedID)
	if !ok || status.Kind != shared.AdapterKindConnector {
		return connectorcontract.StepExecutionResult{}, fmt.Errorf("connector worker not registered: %s", trimmedID)
	}

	if status.State != PluginWorkerStateRunning || workerExecAddress(status.Metadata) == "" {
		if err := d.supervisor.RestartWorker(ctx, trimmedID); err != nil {
			return connectorcontract.StepExecutionResult{}, fmt.Errorf("connector worker %s is unavailable: %w", trimmedID, err)
		}
		restartedStatus, err := d.waitForRunningWorker(ctx, trimmedID)
		if err != nil {
			return connectorcontract.StepExecutionResult{}, err
		}
		return d.executeWorkerStep(ctx, restartedStatus, execCtx, step)
	}

	if d.resilience == nil {
		d.resilience = defaultWorkerDispatchResilience()
	}

	currentStatus := status
	result, err := executeWorkerDispatchWithResilience(ctx, workerDispatchAttemptSpec[connectorcontract.StepExecutionResult]{
		resilience: d.resilience,
		workerID:   trimmedID,
		execute: func(attemptCtx context.Context) (connectorcontract.StepExecutionResult, error) {
			return d.executeWorkerStep(attemptCtx, currentStatus, execCtx, step)
		},
		isRetryable: connectorWorkerErrorRetryable,
		recoverWorker: func(retryCtx context.Context) {
			if recoveredStatus, recovered := d.tryRecoverConnectorWorker(retryCtx, trimmedID, currentStatus); recovered {
				currentStatus = recoveredStatus
			}
		},
		exhaustedErrMsg: "connector worker dispatch failed without explicit error",
	})
	if err != nil {
		return connectorcontract.StepExecutionResult{}, err
	}
	return result, nil
}

func (d *SupervisorConnectorStepDispatcher) resolveWorkerAdapter(capabilityKey string, preferredAdapterID string) (connectorcontract.Metadata, bool, error) {
	if d.supervisor == nil {
		return connectorcontract.Metadata{}, false, fmt.Errorf("plugin supervisor is required")
	}

	trimmedCapability := strings.TrimSpace(capabilityKey)
	if trimmedCapability == "" {
		return connectorcontract.Metadata{}, false, fmt.Errorf("capability key is required")
	}
	preferred := strings.TrimSpace(preferredAdapterID)

	if preferred != "" {
		status, ok := d.supervisor.WorkerStatus(preferred)
		if !ok || status.Kind != shared.AdapterKindConnector {
			return connectorcontract.Metadata{}, false, nil
		}
		if !supportsCapability(status.Metadata, trimmedCapability) {
			return connectorcontract.Metadata{}, false, fmt.Errorf("preferred connector adapter %s does not support capability %s", preferred, trimmedCapability)
		}
		if status.State != PluginWorkerStateRunning {
			return connectorcontract.Metadata{}, false, fmt.Errorf("preferred connector adapter %s is not running", preferred)
		}
		if workerExecAddress(status.Metadata) == "" {
			return connectorcontract.Metadata{}, false, fmt.Errorf("preferred connector adapter %s has no execution endpoint", preferred)
		}
		return status.Metadata, true, nil
	}

	statuses := d.supervisor.ListWorkers()
	sort.Slice(statuses, func(i, j int) bool {
		return statuses[i].PluginID < statuses[j].PluginID
	})
	for _, status := range statuses {
		if status.Kind != shared.AdapterKindConnector {
			continue
		}
		if status.State != PluginWorkerStateRunning {
			continue
		}
		if workerExecAddress(status.Metadata) == "" {
			continue
		}
		if !supportsCapability(status.Metadata, trimmedCapability) {
			continue
		}
		return status.Metadata, true, nil
	}
	return connectorcontract.Metadata{}, false, nil
}

func (d *SupervisorConnectorStepDispatcher) executeWorkerStep(ctx context.Context, status PluginWorkerStatus, execCtx connectorcontract.ExecutionContext, step connectorcontract.TaskStep) (connectorcontract.StepExecutionResult, error) {
	address := workerExecAddress(status.Metadata)
	if address == "" {
		return connectorcontract.StepExecutionResult{}, &connectorWorkerExecuteError{
			Retryable: true,
			Message:   fmt.Sprintf("connector worker %s has no execution endpoint", status.PluginID),
		}
	}
	authToken := strings.TrimSpace(status.execAuthToken)
	if authToken == "" {
		return connectorcontract.StepExecutionResult{}, &connectorWorkerExecuteError{
			Retryable: true,
			Message:   fmt.Sprintf("connector worker %s has no daemon-issued auth token", status.PluginID),
		}
	}

	payload, err := json.Marshal(workerExecuteRequest{
		ExecutionContext: execCtx,
		Step:             step,
	})
	if err != nil {
		return connectorcontract.StepExecutionResult{}, &connectorWorkerExecuteError{
			Retryable: false,
			Message:   fmt.Sprintf("marshal worker execute request: %v", err),
			Cause:     err,
		}
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://"+address+"/execute", bytes.NewReader(payload))
	if err != nil {
		return connectorcontract.StepExecutionResult{}, &connectorWorkerExecuteError{
			Retryable: false,
			Message:   fmt.Sprintf("build worker execute request: %v", err),
			Cause:     err,
		}
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer "+authToken)

	response, err := d.httpClient.Do(request)
	if err != nil {
		return connectorcontract.StepExecutionResult{}, &connectorWorkerExecuteError{
			Retryable: true,
			Message:   strings.TrimSpace(err.Error()),
			Cause:     err,
		}
	}
	defer response.Body.Close()

	body, truncated, readErr := readBoundedHTTPResponseBody(response.Body, daemonWorkerRPCResponseBodyLimitBytes)
	if readErr != nil {
		return connectorcontract.StepExecutionResult{}, &connectorWorkerExecuteError{
			Retryable: true,
			Message:   fmt.Sprintf("read worker execute response: %v", readErr),
			Cause:     readErr,
		}
	}
	if truncated {
		return connectorcontract.StepExecutionResult{}, &connectorWorkerExecuteError{
			Retryable: true,
			Message: fmt.Sprintf(
				"worker execute response exceeded max size of %d bytes",
				daemonWorkerRPCResponseBodyLimitBytes,
			),
		}
	}

	if response.StatusCode >= 400 {
		message := strings.TrimSpace(extractWorkerErrorMessage(body))
		if message == "" {
			message = strings.TrimSpace(string(body))
		}
		if message == "" {
			message = response.Status
		}
		return connectorcontract.StepExecutionResult{}, &connectorWorkerExecuteError{
			Retryable: response.StatusCode >= 500,
			Message:   fmt.Sprintf("worker execute failed: %s", message),
		}
	}

	var result connectorcontract.StepExecutionResult
	if err := json.Unmarshal(body, &result); err != nil {
		return connectorcontract.StepExecutionResult{}, &connectorWorkerExecuteError{
			Retryable: true,
			Message:   fmt.Sprintf("decode worker execute response: %v", err),
			Cause:     err,
		}
	}
	return result, nil
}

func (d *SupervisorConnectorStepDispatcher) tryRecoverConnectorWorker(
	ctx context.Context,
	adapterID string,
	previousStatus PluginWorkerStatus,
) (PluginWorkerStatus, bool) {
	latestStatus, waitErr := d.waitForRunningWorker(ctx, adapterID)
	if waitErr == nil {
		addressChanged := workerExecAddress(latestStatus.Metadata) != workerExecAddress(previousStatus.Metadata)
		processChanged := latestStatus.ProcessID != previousStatus.ProcessID || latestStatus.RestartCount != previousStatus.RestartCount
		if addressChanged || processChanged {
			return latestStatus, true
		}
	}
	if d.supervisor != nil {
		if restartErr := d.supervisor.RestartWorker(ctx, adapterID); restartErr == nil {
			restartedStatus, restartWaitErr := d.waitForRunningWorkerAfter(ctx, adapterID, previousStatus)
			if restartWaitErr == nil {
				return restartedStatus, true
			}
		}
	}
	return previousStatus, false
}

func (d *SupervisorConnectorStepDispatcher) waitForRunningWorker(ctx context.Context, adapterID string) (PluginWorkerStatus, error) {
	if d.supervisor == nil {
		return PluginWorkerStatus{}, fmt.Errorf("plugin supervisor is required")
	}
	return waitForPluginWorkerStatus(
		ctx,
		func() (PluginWorkerStatus, bool) {
			return d.supervisor.WorkerStatus(adapterID)
		},
		func(status PluginWorkerStatus) bool {
			return status.Kind == shared.AdapterKindConnector &&
				status.State == PluginWorkerStateRunning &&
				workerExecAddress(status.Metadata) != ""
		},
		d.restartBackoff,
		d.restartDeadline,
		fmt.Errorf("timed out waiting for connector worker %s restart", adapterID),
	)
}

func (d *SupervisorConnectorStepDispatcher) waitForRunningWorkerAfter(
	ctx context.Context,
	adapterID string,
	previous PluginWorkerStatus,
) (PluginWorkerStatus, error) {
	if d.supervisor == nil {
		return PluginWorkerStatus{}, fmt.Errorf("plugin supervisor is required")
	}
	return waitForPluginWorkerStatus(
		ctx,
		func() (PluginWorkerStatus, bool) {
			return d.supervisor.WorkerStatus(adapterID)
		},
		func(status PluginWorkerStatus) bool {
			if status.Kind != shared.AdapterKindConnector ||
				status.State != PluginWorkerStateRunning ||
				workerExecAddress(status.Metadata) == "" {
				return false
			}
			addressChanged := workerExecAddress(status.Metadata) != workerExecAddress(previous.Metadata)
			processChanged := status.ProcessID != previous.ProcessID || status.RestartCount != previous.RestartCount
			return addressChanged || processChanged
		},
		d.restartBackoff,
		d.restartDeadline,
		fmt.Errorf("timed out waiting for connector worker %s restart", adapterID),
	)
}

func workerExecAddress(metadata connectorcontract.Metadata) string {
	if len(metadata.Runtime) == 0 {
		return ""
	}
	return strings.TrimSpace(metadata.Runtime[connectorRuntimeExecAddressKey])
}

func supportsCapability(metadata connectorcontract.Metadata, capabilityKey string) bool {
	trimmed := strings.TrimSpace(capabilityKey)
	if trimmed == "" {
		return false
	}
	for _, capability := range metadata.Capabilities {
		if strings.TrimSpace(capability.Key) == trimmed {
			return true
		}
	}
	return false
}

func extractWorkerErrorMessage(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	payload := map[string]any{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	message, _ := payload["error"].(string)
	if strings.TrimSpace(message) != "" {
		return message
	}
	message, _ = payload["message"].(string)
	return message
}

type workerExecuteRequest struct {
	ExecutionContext connectorcontract.ExecutionContext `json:"execution_context"`
	Step             connectorcontract.TaskStep         `json:"step"`
}

type connectorWorkerExecuteError struct {
	Retryable bool
	Message   string
	Cause     error
}

func (e *connectorWorkerExecuteError) Error() string {
	if e == nil {
		return ""
	}
	if strings.TrimSpace(e.Message) != "" {
		return strings.TrimSpace(e.Message)
	}
	if e.Cause != nil {
		return strings.TrimSpace(e.Cause.Error())
	}
	return "connector worker execute failed"
}

func (e *connectorWorkerExecuteError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func connectorWorkerErrorRetryable(err error) bool {
	if err == nil {
		return false
	}
	var executeErr *connectorWorkerExecuteError
	if errors.As(err, &executeErr) {
		return executeErr.Retryable
	}
	return false
}

type dispatchConnectorSelector struct {
	dispatcher ConnectorStepDispatcher
}

func newDispatchConnectorSelector(dispatcher ConnectorStepDispatcher) *dispatchConnectorSelector {
	return &dispatchConnectorSelector{dispatcher: dispatcher}
}

func (s *dispatchConnectorSelector) SelectByCapability(capabilityKey string, preferredAdapterID string) (connectorcontract.Adapter, error) {
	if s.dispatcher == nil {
		return nil, fmt.Errorf("connector dispatcher is required")
	}
	metadata, err := s.dispatcher.ResolveAdapter(capabilityKey, preferredAdapterID)
	if err != nil {
		return nil, err
	}
	return &dispatchConnectorAdapter{
		metadata:   metadata,
		dispatcher: s.dispatcher,
	}, nil
}

type dispatchConnectorAdapter struct {
	metadata   connectorcontract.Metadata
	dispatcher ConnectorStepDispatcher
}

func (a *dispatchConnectorAdapter) Metadata() connectorcontract.Metadata {
	return a.metadata
}

func (a *dispatchConnectorAdapter) HealthCheck(_ context.Context) error {
	if a.dispatcher == nil {
		return fmt.Errorf("connector dispatcher is required")
	}
	return nil
}

func (a *dispatchConnectorAdapter) ExecuteStep(ctx context.Context, execCtx connectorcontract.ExecutionContext, step connectorcontract.TaskStep) (connectorcontract.StepExecutionResult, error) {
	if a.dispatcher == nil {
		return connectorcontract.StepExecutionResult{}, fmt.Errorf("connector dispatcher is required")
	}
	return a.dispatcher.ExecuteStep(ctx, a.metadata.ID, execCtx, step)
}
