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
	"testing"
	"time"

	"personalagent/runtime/internal/channelcheck"
	messagesadapter "personalagent/runtime/internal/channels/adapters/messages"
	twilioadapter "personalagent/runtime/internal/channels/adapters/twilio"
	shared "personalagent/runtime/internal/shared/contracts"
)

func TestSupervisorChannelWorkerDispatcherExecutesViaWorker(t *testing.T) {
	supervisor := NewProcessPluginSupervisor()
	spec := channelDispatchWorkerSpec(t, "stable", "twilio.daemon", []string{
		twilioChannelWorkerCapabilityCheck,
		twilioChannelWorkerCapabilitySendSMS,
		twilioChannelWorkerCapabilityStartVoice,
	}, 0, nil)
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
		return status.State == PluginWorkerStateRunning && channelWorkerExecAddress(status.Metadata) != ""
	})

	dispatcher := NewSupervisorChannelWorkerDispatcher(supervisor)
	result, err := dispatcher.CheckTwilio(context.Background(), channelcheck.TwilioRequest{
		Endpoint:   "https://api.twilio.test",
		AccountSID: "AC123",
		AuthToken:  "token",
	})
	if err != nil {
		t.Fatalf("check twilio via worker: %v", err)
	}
	if result.Endpoint != "worker://twilio/check" {
		t.Fatalf("expected worker endpoint marker, got %s", result.Endpoint)
	}
	if result.StatusCode != http.StatusOK {
		t.Fatalf("expected status_code=200, got %d", result.StatusCode)
	}
}

func TestSupervisorChannelWorkerDispatcherFailsWithoutWorker(t *testing.T) {
	dispatcher := NewSupervisorChannelWorkerDispatcher(nil)
	_, err := dispatcher.CheckTwilio(context.Background(), channelcheck.TwilioRequest{
		Endpoint:   "https://api.twilio.test",
		AccountSID: "AC123",
		AuthToken:  "token",
	})
	if err == nil {
		t.Fatalf("expected error when no channel worker is available")
	}
	if !strings.Contains(err.Error(), "worker is not available") {
		t.Fatalf("expected worker availability error, got %v", err)
	}
}

func TestSupervisorChannelWorkerDispatcherExecuteWorkerOperationRequiresAuthToken(t *testing.T) {
	dispatcher := NewSupervisorChannelWorkerDispatcher(nil)
	status := PluginWorkerStatus{
		PluginID: "twilio.daemon",
		Metadata: shared.AdapterMetadata{
			Runtime: map[string]string{
				channelRuntimeExecAddressKey: "127.0.0.1:65535",
			},
		},
	}

	var result map[string]any
	err := dispatcher.executeWorkerOperation(context.Background(), status, "twilio_check", map[string]any{}, &result)
	if err == nil {
		t.Fatalf("expected missing worker auth token error")
	}
	if !strings.Contains(err.Error(), "daemon-issued auth token") {
		t.Fatalf("expected auth-token error, got %v", err)
	}
}

func TestSupervisorChannelWorkerDispatcherExecuteWorkerOperationPropagatesUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if strings.TrimSpace(request.Header.Get("Authorization")) != "Bearer expected-token" {
			writer.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(writer).Encode(map[string]any{"error": "unauthorized"})
			return
		}
		_ = json.NewEncoder(writer).Encode(map[string]any{
			"result": map[string]any{"ok": true},
		})
	}))
	defer server.Close()

	dispatcher := &SupervisorChannelWorkerDispatcher{
		httpClient: server.Client(),
	}
	status := PluginWorkerStatus{
		PluginID: "twilio.daemon",
		Metadata: shared.AdapterMetadata{
			Runtime: map[string]string{
				channelRuntimeExecAddressKey: strings.TrimPrefix(server.URL, "http://"),
			},
		},
		execAuthToken: "wrong-token",
	}

	var result map[string]any
	err := dispatcher.executeWorkerOperation(context.Background(), status, "twilio_check", map[string]any{}, &result)
	if err == nil {
		t.Fatalf("expected unauthorized worker execute error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "unauthorized") {
		t.Fatalf("expected unauthorized error, got %v", err)
	}
}

func TestSupervisorChannelWorkerDispatcherExecuteWorkerOperationRejectsOversizedResponse(t *testing.T) {
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

	dispatcher := &SupervisorChannelWorkerDispatcher{
		httpClient: server.Client(),
	}
	status := PluginWorkerStatus{
		PluginID: "twilio.daemon",
		Metadata: shared.AdapterMetadata{
			Runtime: map[string]string{
				channelRuntimeExecAddressKey: strings.TrimPrefix(server.URL, "http://"),
			},
		},
		execAuthToken: "expected-token",
	}

	var result map[string]any
	err := dispatcher.executeWorkerOperation(context.Background(), status, "twilio_check", map[string]any{}, &result)
	if err == nil {
		t.Fatalf("expected oversized worker response to be rejected")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "exceeded max size") {
		t.Fatalf("expected oversized response error, got %v", err)
	}
}

func TestSupervisorChannelWorkerDispatcherDoesNotRestartOnOperationError(t *testing.T) {
	supervisor := NewProcessPluginSupervisor()
	events := &recordingPluginEvents{}
	supervisor.SetHooks(PluginLifecycleHooks{
		OnEvent: events.Record,
	})

	spec := channelDispatchWorkerSpec(t, "operation_error", "messages.daemon", []string{
		messagesWorkerCapabilitySend,
	}, 3, nil)
	if err := supervisor.RegisterWorker(spec); err != nil {
		t.Fatalf("register worker: %v", err)
	}
	if err := supervisor.Start(context.Background()); err != nil {
		t.Fatalf("start supervisor: %v", err)
	}
	t.Cleanup(func() {
		_ = supervisor.Stop(context.Background())
	})

	initial := waitForWorkerState(t, supervisor, spec.PluginID, 3*time.Second, func(status PluginWorkerStatus) bool {
		return status.State == PluginWorkerStateRunning && channelWorkerExecAddress(status.Metadata) != ""
	})

	dispatcher := NewSupervisorChannelWorkerDispatcher(supervisor)
	_, err := dispatcher.SendMessages(context.Background(), messagesadapter.SendRequest{
		WorkspaceID: "ws1",
		Destination: "+15555550999",
		Message:     "operation error test",
	})
	if err == nil {
		t.Fatalf("expected operation error from worker")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "simulated operation failure") {
		t.Fatalf("expected simulated operation failure, got %v", err)
	}

	time.Sleep(300 * time.Millisecond)
	after, ok := supervisor.WorkerStatus(spec.PluginID)
	if !ok {
		t.Fatalf("expected worker status to remain available")
	}
	if after.State != PluginWorkerStateRunning {
		t.Fatalf("expected worker to remain running after operation error, got %+v", after)
	}
	if after.RestartCount != initial.RestartCount {
		t.Fatalf("expected no restart-count change after operation error, before=%d after=%d", initial.RestartCount, after.RestartCount)
	}
	if after.ProcessID != initial.ProcessID {
		t.Fatalf("expected no process restart after operation error, before=%d after=%d", initial.ProcessID, after.ProcessID)
	}
	if events.Count(spec.PluginID, pluginEventWorkerRestarting) != 0 {
		t.Fatalf("expected no worker restarting lifecycle event after operation error")
	}
}

func TestSupervisorChannelWorkerDispatcherRetriesAfterWorkerError(t *testing.T) {
	supervisor := NewProcessPluginSupervisor()
	failMarker := filepathForWorkerMarker(t, "channel-dispatch-fail-once.marker")
	spec := channelDispatchWorkerSpec(t, "fail_once", "twilio.daemon", []string{
		twilioChannelWorkerCapabilitySendSMS,
	}, 0, []string{
		"PA_CHANNEL_DISPATCH_HELPER_FAIL_MARKER=" + failMarker,
	})
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
		return status.State == PluginWorkerStateRunning && channelWorkerExecAddress(status.Metadata) != ""
	})

	dispatcher := NewSupervisorChannelWorkerDispatcher(supervisor)
	response, err := dispatcher.SendTwilioSMS(context.Background(), twilioadapter.SMSAPIRequest{
		Endpoint:   "https://api.twilio.test",
		AccountSID: "AC123",
		AuthToken:  "token",
		From:       "+15555550001",
		To:         "+15555550999",
		Body:       "worker retry",
	})
	if err != nil {
		t.Fatalf("send twilio sms after transient worker failure: %v", err)
	}
	if response.MessageSID == "" {
		t.Fatalf("expected message sid after retry")
	}
}

func TestSupervisorChannelWorkerDispatcherRecoversAfterWorkerRegistration(t *testing.T) {
	supervisor := NewProcessPluginSupervisor()
	if err := supervisor.Start(context.Background()); err != nil {
		t.Fatalf("start supervisor: %v", err)
	}
	t.Cleanup(func() {
		_ = supervisor.Stop(context.Background())
	})

	dispatcher := NewSupervisorChannelWorkerDispatcher(supervisor)
	_, preErr := dispatcher.SendTwilioSMS(context.Background(), twilioadapter.SMSAPIRequest{
		Endpoint:   "https://api.twilio.test",
		AccountSID: "AC123",
		AuthToken:  "token",
		From:       "+15555550001",
		To:         "+15555550999",
		Body:       "before worker",
	})
	if preErr == nil {
		t.Fatalf("expected unavailable worker error before registration")
	}

	spec := channelDispatchWorkerSpec(t, "stable", "twilio.recovery", []string{
		twilioChannelWorkerCapabilitySendSMS,
	}, 0, nil)
	if err := supervisor.RegisterWorker(spec); err != nil {
		t.Fatalf("register worker: %v", err)
	}
	_ = waitForWorkerState(t, supervisor, spec.PluginID, 3*time.Second, func(status PluginWorkerStatus) bool {
		return status.State == PluginWorkerStateRunning && channelWorkerExecAddress(status.Metadata) != ""
	})

	response, err := dispatcher.SendTwilioSMS(context.Background(), twilioadapter.SMSAPIRequest{
		Endpoint:   "https://api.twilio.test",
		AccountSID: "AC123",
		AuthToken:  "token",
		From:       "+15555550001",
		To:         "+15555550999",
		Body:       "after worker",
	})
	if err != nil {
		t.Fatalf("send twilio sms after worker registration: %v", err)
	}
	if response.MessageSID == "" {
		t.Fatalf("expected message sid from worker after recovery")
	}
}

func TestSupervisorChannelWorkerDispatcherMessagesSendViaWorker(t *testing.T) {
	supervisor := NewProcessPluginSupervisor()
	spec := channelDispatchWorkerSpec(t, "stable", "messages.daemon", []string{
		messagesWorkerCapabilitySend,
	}, 0, nil)
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
		return status.State == PluginWorkerStateRunning && channelWorkerExecAddress(status.Metadata) != ""
	})

	dispatcher := NewSupervisorChannelWorkerDispatcher(supervisor)
	response, err := dispatcher.SendMessages(context.Background(), messagesadapter.SendRequest{
		WorkspaceID: "ws1",
		Destination: "+15555550999",
		Message:     "hello",
	})
	if err != nil {
		t.Fatalf("send messages via worker: %v", err)
	}
	if response.Channel != "imessage" {
		t.Fatalf("expected imessage channel response, got %s", response.Channel)
	}
	if response.Status != "sent" {
		t.Fatalf("expected sent status, got %s", response.Status)
	}
	if response.MessageID == "" {
		t.Fatalf("expected message id in worker response")
	}
}

func TestSupervisorChannelWorkerDispatcherMessagesPollInboundViaWorker(t *testing.T) {
	supervisor := NewProcessPluginSupervisor()
	spec := channelDispatchWorkerSpec(t, "stable", "messages.daemon", []string{
		messagesWorkerCapabilityPollInbound,
	}, 0, nil)
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
		return status.State == PluginWorkerStateRunning && channelWorkerExecAddress(status.Metadata) != ""
	})

	dispatcher := NewSupervisorChannelWorkerDispatcher(supervisor)
	response, err := dispatcher.PollMessagesInbound(context.Background(), messagesadapter.InboundPollRequest{
		WorkspaceID: "ws1",
		SinceCursor: "0",
		Limit:       10,
	})
	if err != nil {
		t.Fatalf("poll messages inbound via worker: %v", err)
	}
	if response.Polled != 1 {
		t.Fatalf("expected one polled event, got %d", response.Polled)
	}
	if len(response.Events) != 1 {
		t.Fatalf("expected one polled event payload")
	}
	if response.Events[0].SourceEventID != "messages-worker-event-1" {
		t.Fatalf("unexpected polled source event id: %s", response.Events[0].SourceEventID)
	}
}

func TestSupervisorChannelWorkerDispatcherPollInboundDoesNotManualRestartOnRetryableError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		time.Sleep(90 * time.Millisecond)
		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(map[string]any{
			"result": map[string]any{
				"workspace_id": "ws1",
				"source":       messagesadapter.SourceName,
				"source_scope": "worker-scope",
				"polled":       0,
				"events":       []any{},
			},
		})
	}))
	defer server.Close()

	supervisor := &channelDispatchSupervisorStub{
		status: PluginWorkerStatus{
			PluginID: "messages.daemon",
			Kind:     shared.AdapterKindChannel,
			State:    PluginWorkerStateRunning,
			Metadata: shared.AdapterMetadata{
				ID:   "messages.daemon",
				Kind: shared.AdapterKindChannel,
				Capabilities: []shared.CapabilityDescriptor{
					{Key: messagesWorkerCapabilityPollInbound},
				},
				Runtime: map[string]string{
					channelRuntimeExecAddressKey: strings.TrimPrefix(server.URL, "http://"),
				},
			},
			execAuthToken: "worker-token",
		},
	}

	dispatcher := NewSupervisorChannelWorkerDispatcher(supervisor)
	dispatcher.httpClient = &http.Client{Timeout: 20 * time.Millisecond}
	dispatcher.restartBackoff = 5 * time.Millisecond
	dispatcher.restartDeadline = 60 * time.Millisecond

	_, err := dispatcher.PollMessagesInbound(context.Background(), messagesadapter.InboundPollRequest{
		WorkspaceID: "ws1",
		SinceCursor: "0",
		Limit:       1,
	})
	if err == nil {
		t.Fatalf("expected retryable poll error from timed-out worker request")
	}
	if supervisor.restartCalls != 0 {
		t.Fatalf("expected no manual restart calls for poll-inbound retryable errors, got %d", supervisor.restartCalls)
	}
}

func TestSupervisorChannelWorkerDispatcherPollInboundHonorsRetryBudget(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestCount++
		writer.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(writer).Encode(map[string]any{"error": "temporary backend failure"})
	}))
	defer server.Close()

	supervisor := &channelDispatchSupervisorStub{
		status: PluginWorkerStatus{
			PluginID: "messages.daemon",
			Kind:     shared.AdapterKindChannel,
			State:    PluginWorkerStateRunning,
			Metadata: shared.AdapterMetadata{
				ID:   "messages.daemon",
				Kind: shared.AdapterKindChannel,
				Capabilities: []shared.CapabilityDescriptor{
					{Key: messagesWorkerCapabilityPollInbound},
				},
				Runtime: map[string]string{
					channelRuntimeExecAddressKey: strings.TrimPrefix(server.URL, "http://"),
				},
			},
			execAuthToken: "worker-token",
		},
	}

	dispatcher := NewSupervisorChannelWorkerDispatcher(supervisor)
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

	_, err := dispatcher.PollMessagesInbound(context.Background(), messagesadapter.InboundPollRequest{
		WorkspaceID: "ws1",
		SinceCursor: "0",
		Limit:       1,
	})
	if err == nil {
		t.Fatalf("expected retry-budget exhausted error")
	}
	if requestCount != 2 {
		t.Fatalf("expected exactly two attempts for maxRetries=1, got %d", requestCount)
	}
	if supervisor.restartCalls != 0 {
		t.Fatalf("expected no manual restart for poll-inbound retries, got %d", supervisor.restartCalls)
	}
}

func TestSupervisorChannelWorkerDispatcherOpensCircuitAfterRetryableFailures(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestCount++
		writer.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(writer).Encode(map[string]any{"error": "worker overloaded"})
	}))
	defer server.Close()

	supervisor := &channelDispatchSupervisorStub{
		status: PluginWorkerStatus{
			PluginID: "twilio.daemon",
			Kind:     shared.AdapterKindChannel,
			State:    PluginWorkerStateRunning,
			Metadata: shared.AdapterMetadata{
				ID:   "twilio.daemon",
				Kind: shared.AdapterKindChannel,
				Capabilities: []shared.CapabilityDescriptor{
					{Key: twilioChannelWorkerCapabilitySendSMS},
				},
				Runtime: map[string]string{
					channelRuntimeExecAddressKey: strings.TrimPrefix(server.URL, "http://"),
				},
			},
			execAuthToken: "worker-token",
		},
	}

	dispatcher := NewSupervisorChannelWorkerDispatcher(supervisor)
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

	request := twilioadapter.SMSAPIRequest{
		Endpoint:   "https://api.twilio.test",
		AccountSID: "AC123",
		AuthToken:  "token",
		From:       "+15555550001",
		To:         "+15555550999",
		Body:       "hello",
	}
	_, firstErr := dispatcher.SendTwilioSMS(context.Background(), request)
	if firstErr == nil {
		t.Fatalf("expected first retryable worker error")
	}
	_, secondErr := dispatcher.SendTwilioSMS(context.Background(), request)
	if secondErr == nil {
		t.Fatalf("expected second retryable worker error")
	}
	_, circuitErr := dispatcher.SendTwilioSMS(context.Background(), request)
	if circuitErr == nil {
		t.Fatalf("expected circuit-open failure")
	}
	if !strings.Contains(strings.ToLower(circuitErr.Error()), "circuit open") {
		t.Fatalf("expected circuit-open error, got %v", circuitErr)
	}
	if requestCount != 2 {
		t.Fatalf("expected circuit-open third call to fail fast without HTTP request, got %d requests", requestCount)
	}
}

func TestSupervisorChannelWorkerDispatcherEnforcesPerOperationTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		time.Sleep(150 * time.Millisecond)
		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(map[string]any{
			"result": map[string]any{
				"workspace_id": "ws1",
				"source":       messagesadapter.SourceName,
				"source_scope": "worker-scope",
				"polled":       0,
				"events":       []any{},
			},
		})
	}))
	defer server.Close()

	supervisor := &channelDispatchSupervisorStub{
		status: PluginWorkerStatus{
			PluginID: "messages.daemon",
			Kind:     shared.AdapterKindChannel,
			State:    PluginWorkerStateRunning,
			Metadata: shared.AdapterMetadata{
				ID:   "messages.daemon",
				Kind: shared.AdapterKindChannel,
				Capabilities: []shared.CapabilityDescriptor{
					{Key: messagesWorkerCapabilityPollInbound},
				},
				Runtime: map[string]string{
					channelRuntimeExecAddressKey: strings.TrimPrefix(server.URL, "http://"),
				},
			},
			execAuthToken: "worker-token",
		},
	}

	dispatcher := NewSupervisorChannelWorkerDispatcher(supervisor)
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
	_, err := dispatcher.PollMessagesInbound(context.Background(), messagesadapter.InboundPollRequest{
		WorkspaceID: "ws1",
		SinceCursor: "0",
		Limit:       1,
	})
	elapsed := time.Since(start)
	if err == nil {
		t.Fatalf("expected timeout error from per-operation deadline")
	}
	if elapsed > 120*time.Millisecond {
		t.Fatalf("expected operation timeout to fail quickly, took %s", elapsed)
	}
}

func channelDispatchWorkerSpec(t *testing.T, mode string, pluginID string, capabilities []string, maxRestarts int, extraEnv []string) PluginWorkerSpec {
	t.Helper()
	env := []string{"PA_CHANNEL_DISPATCH_HELPER_PROCESS=1"}
	env = append(env, extraEnv...)
	return PluginWorkerSpec{
		PluginID: pluginID,
		Kind:     shared.AdapterKindChannel,
		Command:  os.Args[0],
		Args: []string{
			"-test.run=TestChannelDispatchWorkerHelperProcess",
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

func TestChannelDispatchWorkerHelperProcess(t *testing.T) {
	if os.Getenv("PA_CHANNEL_DISPATCH_HELPER_PROCESS") != "1" {
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
		Kind:        shared.AdapterKindChannel,
		DisplayName: pluginID,
		Version:     "test",
		Runtime: map[string]string{
			channelRuntimeExecAddressKey: listener.Addr().String(),
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
		var payload channelWorkerExecuteRequest
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(writer).Encode(map[string]any{"error": err.Error()})
			return
		}

		if mode == "fail_once" {
			markerPath := strings.TrimSpace(os.Getenv("PA_CHANNEL_DISPATCH_HELPER_FAIL_MARKER"))
			if markerPath != "" {
				if _, err := os.Stat(markerPath); os.IsNotExist(err) {
					_ = os.WriteFile(markerPath, []byte("1"), 0o644)
					writer.WriteHeader(http.StatusInternalServerError)
					_ = json.NewEncoder(writer).Encode(map[string]any{"error": "temporary failure"})
					return
				}
			}
		}
		if mode == "operation_error" {
			writer.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(writer).Encode(map[string]any{"error": "simulated operation failure"})
			return
		}

		response := map[string]any{}
		switch strings.TrimSpace(payload.Operation) {
		case "twilio_check":
			response["result"] = channelcheck.TwilioResult{
				Endpoint:   "worker://twilio/check",
				StatusCode: http.StatusOK,
				LatencyMS:  1,
				Message:    "ok",
			}
		case "twilio_sms_send":
			response["result"] = twilioadapter.SMSAPIResponse{
				Endpoint:   "worker://twilio/sms",
				StatusCode: http.StatusCreated,
				MessageSID: "SMWORKER1",
				AccountSID: "AC123",
				Status:     "queued",
				From:       "+15555550001",
				To:         "+15555550999",
			}
		case "twilio_voice_start_call":
			response["result"] = twilioadapter.VoiceCallResponse{
				Endpoint:   "worker://twilio/voice",
				StatusCode: http.StatusCreated,
				CallSID:    "CAWORKER1",
				AccountSID: "AC123",
				Status:     "queued",
				Direction:  "outbound-api",
				From:       "+15555550002",
				To:         "+15555550999",
			}
		case "messages_send":
			response["result"] = messagesadapter.SendResponse{
				WorkspaceID: "ws1",
				Destination: "+15555550999",
				MessageID:   "imessage-worker-1",
				Channel:     "imessage",
				Status:      "sent",
				Transport:   "messages_dry_run",
			}
		case "messages_poll_inbound":
			response["result"] = messagesadapter.InboundPollResponse{
				WorkspaceID:  "ws1",
				Source:       messagesadapter.SourceName,
				SourceScope:  "worker-scope",
				SourceDBPath: "/tmp/worker-chat.db",
				CursorStart:  "0",
				CursorEnd:    "10",
				Polled:       1,
				Events: []messagesadapter.InboundMessageEvent{
					{
						SourceEventID: "messages-worker-event-1",
						SourceCursor:  "10",
						SenderAddress: "+15555550100",
						BodyText:      "hello from worker",
						OccurredAt:    "2026-02-24T12:00:00Z",
					},
				},
			}
		default:
			response["error"] = "unsupported operation"
		}
		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(response)
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

func filepathForWorkerMarker(t *testing.T, name string) string {
	t.Helper()
	return strings.TrimSpace(t.TempDir() + string(os.PathSeparator) + name)
}

type channelDispatchSupervisorStub struct {
	status       PluginWorkerStatus
	restartCalls int
}

func (s *channelDispatchSupervisorStub) SetHooks(_ PluginLifecycleHooks)         {}
func (s *channelDispatchSupervisorStub) RegisterWorker(_ PluginWorkerSpec) error { return nil }
func (s *channelDispatchSupervisorStub) ListWorkers() []PluginWorkerStatus {
	return []PluginWorkerStatus{s.status}
}
func (s *channelDispatchSupervisorStub) WorkerStatus(pluginID string) (PluginWorkerStatus, bool) {
	if strings.TrimSpace(pluginID) != strings.TrimSpace(s.status.PluginID) {
		return PluginWorkerStatus{}, false
	}
	return s.status, true
}
func (s *channelDispatchSupervisorStub) RestartWorker(_ context.Context, pluginID string) error {
	if strings.TrimSpace(pluginID) == strings.TrimSpace(s.status.PluginID) {
		s.restartCalls++
	}
	return nil
}
func (s *channelDispatchSupervisorStub) StopWorker(_ context.Context, _ string) error { return nil }
func (s *channelDispatchSupervisorStub) Start(_ context.Context) error                { return nil }
func (s *channelDispatchSupervisorStub) Stop(_ context.Context) error                 { return nil }
