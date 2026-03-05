package daemonruntime

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	connectorcontract "personalagent/runtime/internal/connectors/contract"
	shared "personalagent/runtime/internal/shared/contracts"
)

func TestSupervisorConnectorStepDispatcherExecutesViaWorkerProcess(t *testing.T) {
	supervisor := NewProcessPluginSupervisor()
	spec := connectorDispatchWorkerSpec(t, "stable", "mail.daemon", []string{"mail_draft"}, 0, nil)
	if err := supervisor.RegisterWorker(spec); err != nil {
		t.Fatalf("register worker: %v", err)
	}
	if err := supervisor.Start(context.Background()); err != nil {
		t.Fatalf("start supervisor: %v", err)
	}
	t.Cleanup(func() {
		_ = supervisor.Stop(context.Background())
	})

	_ = waitForWorkerState(t, supervisor, spec.PluginID, 3*time.Second, func(status PluginWorkerStatus) bool {
		return status.State == PluginWorkerStateRunning && workerExecAddress(status.Metadata) != ""
	})

	dispatcher := NewSupervisorConnectorStepDispatcher(supervisor, nil)
	metadata, err := dispatcher.ResolveAdapter("mail_draft", "")
	if err != nil {
		t.Fatalf("resolve adapter: %v", err)
	}
	if metadata.ID != spec.PluginID {
		t.Fatalf("expected adapter id %s, got %s", spec.PluginID, metadata.ID)
	}

	result, err := dispatcher.ExecuteStep(context.Background(), metadata.ID, connectorcontract.ExecutionContext{
		WorkspaceID: "ws1",
		TaskID:      "task1",
		RunID:       "run1",
		StepID:      "step1",
	}, connectorcontract.TaskStep{
		ID:            "step1",
		RunID:         "run1",
		StepIndex:     0,
		Name:          "Draft",
		Status:        shared.TaskStepStatusRunning,
		CapabilityKey: "mail_draft",
	})
	if err != nil {
		t.Fatalf("execute step: %v", err)
	}
	if result.Status != shared.TaskStepStatusCompleted {
		t.Fatalf("expected completed status, got %s", result.Status)
	}
	if !strings.Contains(result.Summary, "worker executed mail_draft") {
		t.Fatalf("unexpected summary: %q", result.Summary)
	}
}

func TestSupervisorConnectorStepDispatcherFailsWithoutWorker(t *testing.T) {
	dispatcher := NewSupervisorConnectorStepDispatcher(nil, nil)
	_, err := dispatcher.ResolveAdapter("mail_draft", "")
	if err == nil {
		t.Fatalf("expected resolve error when supervisor is missing")
	}
	if !strings.Contains(err.Error(), "plugin supervisor is required") {
		t.Fatalf("unexpected resolve error: %v", err)
	}
}

func TestSupervisorConnectorStepDispatcherExecuteWorkerStepRequiresAuthToken(t *testing.T) {
	dispatcher := NewSupervisorConnectorStepDispatcher(nil, nil)
	status := PluginWorkerStatus{
		PluginID: "mail.daemon",
		Metadata: shared.AdapterMetadata{
			Runtime: map[string]string{
				connectorRuntimeExecAddressKey: "127.0.0.1:65535",
			},
		},
	}

	_, err := dispatcher.executeWorkerStep(context.Background(), status, connectorcontract.ExecutionContext{}, connectorcontract.TaskStep{})
	if err == nil {
		t.Fatalf("expected missing worker auth token error")
	}
	if !strings.Contains(err.Error(), "daemon-issued auth token") {
		t.Fatalf("expected auth-token error, got %v", err)
	}
}

func TestSupervisorConnectorStepDispatcherExecuteWorkerStepPropagatesUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if strings.TrimSpace(request.Header.Get("Authorization")) != "Bearer expected-token" {
			writer.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(writer).Encode(map[string]any{"error": "unauthorized"})
			return
		}
		_ = json.NewEncoder(writer).Encode(connectorcontract.StepExecutionResult{
			Status: shared.TaskStepStatusCompleted,
		})
	}))
	defer server.Close()

	dispatcher := &SupervisorConnectorStepDispatcher{
		httpClient: server.Client(),
	}
	status := PluginWorkerStatus{
		PluginID: "mail.daemon",
		Metadata: shared.AdapterMetadata{
			Runtime: map[string]string{
				connectorRuntimeExecAddressKey: strings.TrimPrefix(server.URL, "http://"),
			},
		},
		execAuthToken: "wrong-token",
	}

	_, err := dispatcher.executeWorkerStep(context.Background(), status, connectorcontract.ExecutionContext{}, connectorcontract.TaskStep{})
	if err == nil {
		t.Fatalf("expected unauthorized worker execute error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "unauthorized") {
		t.Fatalf("expected unauthorized error, got %v", err)
	}
}

func TestSupervisorConnectorStepDispatcherExecuteWorkerStepRejectsOversizedResponse(t *testing.T) {
	oversizedBody := strings.Repeat("a", int(daemonWorkerRPCResponseBodyLimitBytes+256))
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if strings.TrimSpace(request.Header.Get("Authorization")) != "Bearer expected-token" {
			writer.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(writer).Encode(map[string]any{"error": "unauthorized"})
			return
		}
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(oversizedBody))
	}))
	defer server.Close()

	dispatcher := &SupervisorConnectorStepDispatcher{
		httpClient: server.Client(),
	}
	status := PluginWorkerStatus{
		PluginID: "mail.daemon",
		Metadata: shared.AdapterMetadata{
			Runtime: map[string]string{
				connectorRuntimeExecAddressKey: strings.TrimPrefix(server.URL, "http://"),
			},
		},
		execAuthToken: "expected-token",
	}

	_, err := dispatcher.executeWorkerStep(context.Background(), status, connectorcontract.ExecutionContext{}, connectorcontract.TaskStep{})
	if err == nil {
		t.Fatalf("expected oversized worker response to be rejected")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "exceeded max size") {
		t.Fatalf("expected oversized response error, got %v", err)
	}
}

func TestSupervisorConnectorStepDispatcherRecoversAfterWorkerRegistration(t *testing.T) {
	supervisor := NewProcessPluginSupervisor()
	if err := supervisor.Start(context.Background()); err != nil {
		t.Fatalf("start supervisor: %v", err)
	}
	t.Cleanup(func() {
		_ = supervisor.Stop(context.Background())
	})

	dispatcher := NewSupervisorConnectorStepDispatcher(supervisor, nil)
	_, preErr := dispatcher.ResolveAdapter("mail_draft", "")
	if preErr == nil {
		t.Fatalf("expected resolve error before worker registration")
	}

	spec := connectorDispatchWorkerSpec(t, "stable", "mail.recovery", []string{"mail_draft"}, 0, nil)
	if err := supervisor.RegisterWorker(spec); err != nil {
		t.Fatalf("register worker: %v", err)
	}
	_ = waitForWorkerState(t, supervisor, spec.PluginID, 3*time.Second, func(status PluginWorkerStatus) bool {
		return status.State == PluginWorkerStateRunning && workerExecAddress(status.Metadata) != ""
	})

	metadata, err := dispatcher.ResolveAdapter("mail_draft", "")
	if err != nil {
		t.Fatalf("resolve adapter after registration: %v", err)
	}
	if metadata.ID != spec.PluginID {
		t.Fatalf("expected recovered worker id %s, got %s", spec.PluginID, metadata.ID)
	}
}

func TestSupervisorConnectorStepDispatcherRestartsStoppedWorkerAndRetries(t *testing.T) {
	supervisor := NewProcessPluginSupervisor()
	spec := connectorDispatchWorkerSpec(t, "stable", "finder.daemon", []string{"finder_list"}, 1, nil)
	if err := supervisor.RegisterWorker(spec); err != nil {
		t.Fatalf("register worker: %v", err)
	}
	if err := supervisor.Start(context.Background()); err != nil {
		t.Fatalf("start supervisor: %v", err)
	}
	t.Cleanup(func() {
		_ = supervisor.Stop(context.Background())
	})

	_ = waitForWorkerState(t, supervisor, spec.PluginID, 3*time.Second, func(status PluginWorkerStatus) bool {
		return status.State == PluginWorkerStateRunning && workerExecAddress(status.Metadata) != ""
	})
	if err := supervisor.StopWorker(context.Background(), spec.PluginID); err != nil {
		t.Fatalf("stop worker: %v", err)
	}
	_ = waitForWorkerState(t, supervisor, spec.PluginID, 3*time.Second, func(status PluginWorkerStatus) bool {
		return status.State == PluginWorkerStateStopped
	})

	dispatcher := NewSupervisorConnectorStepDispatcher(supervisor, nil)
	result, err := dispatcher.ExecuteStep(context.Background(), spec.PluginID, connectorcontract.ExecutionContext{
		WorkspaceID: "ws1",
		TaskID:      "task1",
		RunID:       "run1",
		StepID:      "step1",
	}, connectorcontract.TaskStep{
		ID:            "step1",
		RunID:         "run1",
		StepIndex:     0,
		Name:          "List",
		Status:        shared.TaskStepStatusRunning,
		CapabilityKey: "finder_list",
	})
	if err != nil {
		t.Fatalf("execute with restart retry: %v", err)
	}
	if result.Status != shared.TaskStepStatusCompleted {
		t.Fatalf("expected completed status after restart retry, got %s", result.Status)
	}

	status := waitForWorkerState(t, supervisor, spec.PluginID, 5*time.Second, func(status PluginWorkerStatus) bool {
		return status.State == PluginWorkerStateRunning
	})
	if workerExecAddress(status.Metadata) == "" {
		t.Fatalf("expected restarted worker execution endpoint")
	}
}

func TestSupervisorConnectorStepDispatcherBrowserWorkerRecoversUnderRestartChurn(t *testing.T) {
	supervisor := NewProcessPluginSupervisor()
	crashMarkerPath := filepathForWorkerMarker(t, "browser-dispatch-crash-once.marker")
	spec := connectorDispatchWorkerSpec(
		t,
		"crash_once",
		"browser.daemon",
		[]string{"browser_open"},
		2,
		[]string{"PA_CONNECTOR_DISPATCH_HELPER_CRASH_MARKER=" + crashMarkerPath},
	)
	if err := supervisor.RegisterWorker(spec); err != nil {
		t.Fatalf("register worker: %v", err)
	}
	if err := supervisor.Start(context.Background()); err != nil {
		t.Fatalf("start supervisor: %v", err)
	}
	t.Cleanup(func() {
		_ = supervisor.Stop(context.Background())
	})

	_ = waitForWorkerState(t, supervisor, spec.PluginID, 3*time.Second, func(status PluginWorkerStatus) bool {
		return status.State == PluginWorkerStateRunning && workerExecAddress(status.Metadata) != ""
	})

	dispatcher := NewSupervisorConnectorStepDispatcher(supervisor, nil)
	dispatcher.resilience = newWorkerDispatchResilience(workerDispatchResilienceOptions{
		OperationTimeout:     750 * time.Millisecond,
		MaxRetries:           2,
		RetryBackoffBase:     10 * time.Millisecond,
		RetryBackoffMax:      25 * time.Millisecond,
		RetryJitterFraction:  0,
		CircuitOpenThreshold: 8,
		CircuitOpenCooldown:  2 * time.Second,
	})

	execCtx := connectorcontract.ExecutionContext{
		WorkspaceID: "ws1",
		TaskID:      "task-browser",
		RunID:       "run-browser",
		StepID:      "step-browser",
	}
	step := connectorcontract.TaskStep{
		ID:            "step-browser",
		RunID:         "run-browser",
		StepIndex:     0,
		Name:          "Open Browser",
		Status:        shared.TaskStepStatusRunning,
		CapabilityKey: "browser_open",
	}

	for attempt := 1; attempt <= 4; attempt++ {
		result, err := dispatcher.ExecuteStep(context.Background(), spec.PluginID, execCtx, step)
		if err != nil {
			t.Fatalf("execute browser worker attempt %d: %v", attempt, err)
		}
		if result.Status != shared.TaskStepStatusCompleted {
			t.Fatalf("expected completed status on browser attempt %d, got %s", attempt, result.Status)
		}
	}
}

func TestSupervisorConnectorStepDispatcherTryRecoverRequiresFreshWorkerAfterRestart(t *testing.T) {
	previous := PluginWorkerStatus{
		PluginID:  "browser.daemon",
		Kind:      shared.AdapterKindConnector,
		State:     PluginWorkerStateRunning,
		ProcessID: 41,
		Metadata: shared.AdapterMetadata{
			ID:   "browser.daemon",
			Kind: shared.AdapterKindConnector,
			Runtime: map[string]string{
				connectorRuntimeExecAddressKey: "127.0.0.1:43999",
			},
		},
	}
	supervisor := &sequencedConnectorStatusSupervisorStub{
		statusBeforeRestart: previous,
		statusAfterRestart:  previous,
	}
	dispatcher := NewSupervisorConnectorStepDispatcher(supervisor, nil)
	dispatcher.restartBackoff = time.Millisecond
	dispatcher.restartDeadline = 12 * time.Millisecond

	recoveredStatus, recovered := dispatcher.tryRecoverConnectorWorker(context.Background(), previous.PluginID, previous)
	if recovered {
		t.Fatalf("expected stale worker status to be rejected as recovery, got %+v", recoveredStatus)
	}
	if strings.TrimSpace(workerExecAddress(recoveredStatus.Metadata)) != strings.TrimSpace(workerExecAddress(previous.Metadata)) {
		t.Fatalf("expected stale worker address to remain unchanged, got %+v", recoveredStatus.Metadata.Runtime)
	}
	if supervisor.restartCalls() == 0 {
		t.Fatalf("expected manual restart to be attempted for stale worker status")
	}
}

func TestSupervisorConnectorStepDispatcherHonorsRetryBudget(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestCount++
		writer.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(writer).Encode(map[string]any{"error": "temporary connector failure"})
	}))
	defer server.Close()

	supervisor := &channelDispatchSupervisorStub{
		status: PluginWorkerStatus{
			PluginID: "mail.daemon",
			Kind:     shared.AdapterKindConnector,
			State:    PluginWorkerStateRunning,
			Metadata: shared.AdapterMetadata{
				ID:   "mail.daemon",
				Kind: shared.AdapterKindConnector,
				Capabilities: []shared.CapabilityDescriptor{
					{Key: "mail_draft"},
				},
				Runtime: map[string]string{
					connectorRuntimeExecAddressKey: strings.TrimPrefix(server.URL, "http://"),
				},
			},
			execAuthToken: "worker-token",
		},
	}

	dispatcher := NewSupervisorConnectorStepDispatcher(supervisor, nil)
	dispatcher.httpClient = server.Client()
	dispatcher.resilience = newWorkerDispatchResilience(workerDispatchResilienceOptions{
		OperationTimeout:     200 * time.Millisecond,
		MaxRetries:           1,
		RetryBackoffBase:     time.Millisecond,
		RetryBackoffMax:      time.Millisecond,
		RetryJitterFraction:  0,
		CircuitOpenThreshold: 10,
		CircuitOpenCooldown:  time.Second,
	})

	_, err := dispatcher.ExecuteStep(context.Background(), "mail.daemon", connectorcontract.ExecutionContext{
		WorkspaceID: "ws1",
		TaskID:      "task1",
		RunID:       "run1",
		StepID:      "step1",
	}, connectorcontract.TaskStep{
		ID:            "step1",
		RunID:         "run1",
		StepIndex:     0,
		Name:          "Draft",
		Status:        shared.TaskStepStatusRunning,
		CapabilityKey: "mail_draft",
	})
	if err == nil {
		t.Fatalf("expected retry-budget exhausted connector error")
	}
	if requestCount != 2 {
		t.Fatalf("expected exactly two attempts for maxRetries=1, got %d", requestCount)
	}
}

func TestSupervisorConnectorStepDispatcherOpensCircuitAfterRetryableFailures(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestCount++
		writer.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(writer).Encode(map[string]any{"error": "connector overloaded"})
	}))
	defer server.Close()

	supervisor := &channelDispatchSupervisorStub{
		status: PluginWorkerStatus{
			PluginID: "mail.daemon",
			Kind:     shared.AdapterKindConnector,
			State:    PluginWorkerStateRunning,
			Metadata: shared.AdapterMetadata{
				ID:   "mail.daemon",
				Kind: shared.AdapterKindConnector,
				Capabilities: []shared.CapabilityDescriptor{
					{Key: "mail_draft"},
				},
				Runtime: map[string]string{
					connectorRuntimeExecAddressKey: strings.TrimPrefix(server.URL, "http://"),
				},
			},
			execAuthToken: "worker-token",
		},
	}

	dispatcher := NewSupervisorConnectorStepDispatcher(supervisor, nil)
	dispatcher.httpClient = server.Client()
	dispatcher.resilience = newWorkerDispatchResilience(workerDispatchResilienceOptions{
		OperationTimeout:     200 * time.Millisecond,
		MaxRetries:           0,
		RetryBackoffBase:     time.Millisecond,
		RetryBackoffMax:      time.Millisecond,
		RetryJitterFraction:  0,
		CircuitOpenThreshold: 2,
		CircuitOpenCooldown:  5 * time.Second,
	})

	execCtx := connectorcontract.ExecutionContext{
		WorkspaceID: "ws1",
		TaskID:      "task1",
		RunID:       "run1",
		StepID:      "step1",
	}
	step := connectorcontract.TaskStep{
		ID:            "step1",
		RunID:         "run1",
		StepIndex:     0,
		Name:          "Draft",
		Status:        shared.TaskStepStatusRunning,
		CapabilityKey: "mail_draft",
	}

	_, firstErr := dispatcher.ExecuteStep(context.Background(), "mail.daemon", execCtx, step)
	if firstErr == nil {
		t.Fatalf("expected first retryable connector error")
	}
	_, secondErr := dispatcher.ExecuteStep(context.Background(), "mail.daemon", execCtx, step)
	if secondErr == nil {
		t.Fatalf("expected second retryable connector error")
	}
	_, circuitErr := dispatcher.ExecuteStep(context.Background(), "mail.daemon", execCtx, step)
	if circuitErr == nil {
		t.Fatalf("expected circuit-open connector failure")
	}
	if !strings.Contains(strings.ToLower(circuitErr.Error()), "circuit open") {
		t.Fatalf("expected circuit-open error, got %v", circuitErr)
	}
	if requestCount != 2 {
		t.Fatalf("expected third call to fail before HTTP execute when circuit is open, got %d requests", requestCount)
	}
}

func TestSupervisorConnectorStepDispatcherEnforcesPerOperationTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		time.Sleep(150 * time.Millisecond)
		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(connectorcontract.StepExecutionResult{
			Status: shared.TaskStepStatusCompleted,
		})
	}))
	defer server.Close()

	supervisor := &channelDispatchSupervisorStub{
		status: PluginWorkerStatus{
			PluginID: "mail.daemon",
			Kind:     shared.AdapterKindConnector,
			State:    PluginWorkerStateRunning,
			Metadata: shared.AdapterMetadata{
				ID:   "mail.daemon",
				Kind: shared.AdapterKindConnector,
				Capabilities: []shared.CapabilityDescriptor{
					{Key: "mail_draft"},
				},
				Runtime: map[string]string{
					connectorRuntimeExecAddressKey: strings.TrimPrefix(server.URL, "http://"),
				},
			},
			execAuthToken: "worker-token",
		},
	}

	dispatcher := NewSupervisorConnectorStepDispatcher(supervisor, nil)
	dispatcher.httpClient = &http.Client{Timeout: time.Second}
	dispatcher.resilience = newWorkerDispatchResilience(workerDispatchResilienceOptions{
		OperationTimeout:     40 * time.Millisecond,
		MaxRetries:           0,
		RetryBackoffBase:     time.Millisecond,
		RetryBackoffMax:      time.Millisecond,
		RetryJitterFraction:  0,
		CircuitOpenThreshold: 10,
		CircuitOpenCooldown:  time.Second,
	})

	start := time.Now()
	_, err := dispatcher.ExecuteStep(context.Background(), "mail.daemon", connectorcontract.ExecutionContext{
		WorkspaceID: "ws1",
		TaskID:      "task1",
		RunID:       "run1",
		StepID:      "step1",
	}, connectorcontract.TaskStep{
		ID:            "step1",
		RunID:         "run1",
		StepIndex:     0,
		Name:          "Draft",
		Status:        shared.TaskStepStatusRunning,
		CapabilityKey: "mail_draft",
	})
	elapsed := time.Since(start)
	if err == nil {
		t.Fatalf("expected per-operation timeout error")
	}
	if elapsed > 120*time.Millisecond {
		t.Fatalf("expected timeout failure before server response, took %s", elapsed)
	}
}

func connectorDispatchWorkerSpec(t *testing.T, mode string, pluginID string, capabilities []string, maxRestarts int, extraEnv []string) PluginWorkerSpec {
	t.Helper()
	env := []string{"PA_CONNECTOR_DISPATCH_HELPER_PROCESS=1"}
	env = append(env, extraEnv...)
	return PluginWorkerSpec{
		PluginID: pluginID,
		Kind:     shared.AdapterKindConnector,
		Command:  os.Args[0],
		Args: []string{
			"-test.run=TestConnectorDispatchWorkerHelperProcess",
			"--",
			mode,
			pluginID,
			strings.Join(capabilities, ","),
		},
		Env:              env,
		HandshakeTimeout: 2 * time.Second,
		HealthInterval:   50 * time.Millisecond,
		HealthTimeout:    400 * time.Millisecond,
		RestartPolicy: PluginRestartPolicy{
			MaxRestarts: maxRestarts,
			Delay:       50 * time.Millisecond,
		},
	}
}

func TestConnectorDispatchWorkerHelperProcess(t *testing.T) {
	if os.Getenv("PA_CONNECTOR_DISPATCH_HELPER_PROCESS") != "1" {
		return
	}

	args := flag.Args()
	if len(args) < 3 {
		_, _ = fmt.Fprintf(os.Stderr, "invalid helper args: %v\n", args)
		os.Exit(2)
	}

	mode := strings.TrimSpace(args[0])
	pluginID := strings.TrimSpace(args[1])
	capabilityKeys := splitCapabilities(args[2])
	authToken := strings.TrimSpace(os.Getenv(WorkerExecAuthTokenEnvVar))
	if authToken == "" {
		_, _ = fmt.Fprintf(os.Stderr, "missing %s\n", WorkerExecAuthTokenEnvVar)
		os.Exit(2)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "listen: %v\n", err)
		os.Exit(2)
	}
	defer listener.Close()

	metadata := shared.AdapterMetadata{
		ID:          pluginID,
		Kind:        shared.AdapterKindConnector,
		DisplayName: pluginID,
		Version:     "test",
		Runtime: map[string]string{
			connectorRuntimeExecAddressKey: listener.Addr().String(),
		},
	}
	for _, capability := range capabilityKeys {
		metadata.Capabilities = append(metadata.Capabilities, shared.CapabilityDescriptor{Key: capability})
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/execute", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			writer.Header().Set("Allow", http.MethodPost)
			writer.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if strings.TrimSpace(request.Header.Get("Authorization")) != "Bearer "+authToken {
			writer.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(writer).Encode(map[string]any{"error": "unauthorized"})
			return
		}
		var payload workerExecuteRequest
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(writer).Encode(map[string]any{"error": err.Error()})
			return
		}

		if mode == "crash_once" {
			markerPath := strings.TrimSpace(os.Getenv("PA_CONNECTOR_DISPATCH_HELPER_CRASH_MARKER"))
			if markerPath != "" {
				if _, err := os.Stat(markerPath); os.IsNotExist(err) {
					_ = os.WriteFile(markerPath, []byte("1"), 0o644)
					os.Exit(1)
				}
			}
		}

		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(connectorcontract.StepExecutionResult{
			Status:  shared.TaskStepStatusCompleted,
			Summary: fmt.Sprintf("worker executed %s", payload.Step.CapabilityKey),
			Evidence: map[string]string{
				"worker": pluginID,
			},
		})
	})

	server := &http.Server{Handler: mux}
	go func() {
		_ = server.Serve(listener)
	}()

	if err := emitWorkerMessage(pluginWorkerMessage{
		Type:   "handshake",
		Plugin: &metadata,
	}); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "emit handshake: %v\n", err)
		os.Exit(2)
	}

	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		<-ticker.C
		healthy := true
		_ = emitWorkerMessage(pluginWorkerMessage{Type: "health", Healthy: &healthy})
	}
}

type sequencedConnectorStatusSupervisorStub struct {
	mu                  sync.Mutex
	statusBeforeRestart PluginWorkerStatus
	statusAfterRestart  PluginWorkerStatus
	restarts            int
}

func (s *sequencedConnectorStatusSupervisorStub) SetHooks(_ PluginLifecycleHooks)         {}
func (s *sequencedConnectorStatusSupervisorStub) RegisterWorker(_ PluginWorkerSpec) error { return nil }
func (s *sequencedConnectorStatusSupervisorStub) ListWorkers() []PluginWorkerStatus {
	status, ok := s.WorkerStatus(strings.TrimSpace(s.statusBeforeRestart.PluginID))
	if !ok {
		return nil
	}
	return []PluginWorkerStatus{status}
}

func (s *sequencedConnectorStatusSupervisorStub) WorkerStatus(pluginID string) (PluginWorkerStatus, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.TrimSpace(pluginID) != strings.TrimSpace(s.statusBeforeRestart.PluginID) {
		return PluginWorkerStatus{}, false
	}
	if s.restarts > 0 {
		return s.statusAfterRestart, true
	}
	return s.statusBeforeRestart, true
}

func (s *sequencedConnectorStatusSupervisorStub) RestartWorker(_ context.Context, pluginID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.TrimSpace(pluginID) == strings.TrimSpace(s.statusBeforeRestart.PluginID) {
		s.restarts++
	}
	return nil
}

func (s *sequencedConnectorStatusSupervisorStub) StopWorker(_ context.Context, _ string) error {
	return nil
}
func (s *sequencedConnectorStatusSupervisorStub) Start(_ context.Context) error { return nil }
func (s *sequencedConnectorStatusSupervisorStub) Stop(_ context.Context) error  { return nil }

func (s *sequencedConnectorStatusSupervisorStub) restartCalls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.restarts
}
