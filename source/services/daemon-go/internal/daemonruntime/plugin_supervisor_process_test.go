package daemonruntime

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	shared "personalagent/runtime/internal/shared/contracts"
)

func TestProcessPluginSupervisorStartHandshakeHealthAndStop(t *testing.T) {
	supervisor := NewProcessPluginSupervisor()
	events := &recordingPluginEvents{}
	supervisor.SetHooks(PluginLifecycleHooks{
		OnEvent: events.Record,
	})

	spec := helperWorkerSpec(t, "stable", "plugin.channel.stable", shared.AdapterKindChannel, []string{"channel.sms.send"}, 0)
	if err := supervisor.RegisterWorker(spec); err != nil {
		t.Fatalf("register worker: %v", err)
	}

	if err := supervisor.Start(context.Background()); err != nil {
		t.Fatalf("start supervisor: %v", err)
	}
	t.Cleanup(func() {
		_ = supervisor.Stop(context.Background())
	})

	status := waitForWorkerState(t, supervisor, spec.PluginID, 3*time.Second, func(status PluginWorkerStatus) bool {
		return status.State == PluginWorkerStateRunning && status.ProcessID > 0 && len(status.Metadata.Capabilities) > 0
	})
	if status.Metadata.ID != spec.PluginID {
		t.Fatalf("expected metadata id %s, got %s", spec.PluginID, status.Metadata.ID)
	}

	if err := supervisor.Stop(context.Background()); err != nil {
		t.Fatalf("stop supervisor: %v", err)
	}
	stopped := waitForWorkerState(t, supervisor, spec.PluginID, 2*time.Second, func(status PluginWorkerStatus) bool {
		return status.State == PluginWorkerStateStopped
	})
	if stopped.State != PluginWorkerStateStopped {
		t.Fatalf("expected stopped worker state, got %s", stopped.State)
	}

	if count := waitForPluginEventCount(t, events, spec.PluginID, pluginEventWorkerStarted, 1, 2*time.Second); count == 0 {
		t.Fatalf("expected %s event", pluginEventWorkerStarted)
	}
	if count := waitForPluginEventCount(t, events, spec.PluginID, pluginEventHandshakeAccepted, 1, 2*time.Second); count == 0 {
		t.Fatalf("expected %s event", pluginEventHandshakeAccepted)
	}
	if count := waitForPluginEventCount(t, events, spec.PluginID, pluginEventWorkerStopped, 1, 2*time.Second); count == 0 {
		t.Fatalf("expected %s event", pluginEventWorkerStopped)
	}
}

func TestProcessPluginSupervisorStopAndRestartWorker(t *testing.T) {
	supervisor := NewProcessPluginSupervisor()
	spec := helperWorkerSpec(t, "stable", "plugin.channel.restartable", shared.AdapterKindChannel, []string{"channel.sms.send"}, 1)
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
		return status.State == PluginWorkerStateRunning
	})

	if err := supervisor.StopWorker(context.Background(), spec.PluginID); err != nil {
		t.Fatalf("stop worker: %v", err)
	}
	_ = waitForWorkerState(t, supervisor, spec.PluginID, 3*time.Second, func(status PluginWorkerStatus) bool {
		return status.State == PluginWorkerStateStopped
	})

	if err := supervisor.RestartWorker(context.Background(), spec.PluginID); err != nil {
		t.Fatalf("restart worker: %v", err)
	}
	_ = waitForWorkerState(t, supervisor, spec.PluginID, 3*time.Second, func(status PluginWorkerStatus) bool {
		return status.State == PluginWorkerStateRunning
	})
}

func TestProcessPluginSupervisorManualRestartBypassesRestartLimit(t *testing.T) {
	supervisor := NewProcessPluginSupervisor()
	events := &recordingPluginEvents{}
	supervisor.SetHooks(PluginLifecycleHooks{
		OnEvent: events.Record,
	})

	spec := helperWorkerSpec(t, "stable", "plugin.channel.manual-restart", shared.AdapterKindChannel, []string{"channel.sms.send"}, 0)
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
		return status.State == PluginWorkerStateRunning && status.ProcessID > 0
	})

	if err := supervisor.RestartWorker(context.Background(), spec.PluginID); err != nil {
		t.Fatalf("restart worker: %v", err)
	}

	restarted := waitForWorkerState(t, supervisor, spec.PluginID, 3*time.Second, func(status PluginWorkerStatus) bool {
		return status.State == PluginWorkerStateRunning && status.ProcessID > 0 && status.LastTransition.After(initial.LastTransition)
	})
	if restarted.RestartCount != 0 {
		t.Fatalf("expected manual restart not to consume restart budget, got %d", restarted.RestartCount)
	}
	if strings.TrimSpace(restarted.LastError) != "" {
		t.Fatalf("expected manual restart not to persist last_error, got %q", restarted.LastError)
	}
	if events.Count(spec.PluginID, pluginEventWorkerRestartLimit) != 0 {
		t.Fatalf("manual restart should not trigger restart-limit event")
	}
}

func TestProcessPluginSupervisorRestartPolicyAndFailureIsolation(t *testing.T) {
	supervisor := NewProcessPluginSupervisor()
	events := &recordingPluginEvents{}
	supervisor.SetHooks(PluginLifecycleHooks{
		OnEvent: events.Record,
	})

	stable := helperWorkerSpec(t, "stable", "plugin.connector.stable", shared.AdapterKindConnector, []string{"connector.browser.open"}, 0)
	flaky := helperWorkerSpec(t, "crash_after_handshake", "plugin.channel.flaky", shared.AdapterKindChannel, []string{"channel.sms.send"}, 1)
	flaky.HealthTimeout = 500 * time.Millisecond
	flaky.HealthInterval = 50 * time.Millisecond

	if err := supervisor.RegisterWorker(stable); err != nil {
		t.Fatalf("register stable worker: %v", err)
	}
	if err := supervisor.RegisterWorker(flaky); err != nil {
		t.Fatalf("register flaky worker: %v", err)
	}

	if err := supervisor.Start(context.Background()); err != nil {
		t.Fatalf("start supervisor: %v", err)
	}
	t.Cleanup(func() {
		_ = supervisor.Stop(context.Background())
	})

	flakyStatus := waitForWorkerState(t, supervisor, flaky.PluginID, 5*time.Second, func(status PluginWorkerStatus) bool {
		return status.State == PluginWorkerStateFailed
	})
	if flakyStatus.RestartCount != 1 {
		t.Fatalf("expected flaky restart count 1, got %d", flakyStatus.RestartCount)
	}
	if count := waitForPluginEventCount(t, events, flaky.PluginID, pluginEventWorkerRestarting, 1, 2*time.Second); count == 0 {
		t.Fatalf("expected flaky worker restarting events")
	}

	stableStatus := waitForWorkerState(t, supervisor, stable.PluginID, 3*time.Second, func(status PluginWorkerStatus) bool {
		return status.State == PluginWorkerStateRunning
	})
	if stableStatus.State != PluginWorkerStateRunning {
		t.Fatalf("expected stable worker running, got %s", stableStatus.State)
	}
	if stableStatus.RestartCount != 0 {
		t.Fatalf("expected stable worker restart count 0, got %d", stableStatus.RestartCount)
	}
}

func TestProcessPluginSupervisorHealthTimeoutTriggersRestart(t *testing.T) {
	supervisor := NewProcessPluginSupervisor()
	events := &recordingPluginEvents{}
	supervisor.SetHooks(PluginLifecycleHooks{
		OnEvent: events.Record,
	})

	spec := helperWorkerSpec(t, "no_heartbeat", "plugin.channel.no-heartbeat", shared.AdapterKindChannel, []string{"channel.sms.send"}, 0)
	spec.HealthInterval = 40 * time.Millisecond
	spec.HealthTimeout = 120 * time.Millisecond

	if err := supervisor.RegisterWorker(spec); err != nil {
		t.Fatalf("register worker: %v", err)
	}
	if err := supervisor.Start(context.Background()); err != nil {
		t.Fatalf("start supervisor: %v", err)
	}
	t.Cleanup(func() {
		_ = supervisor.Stop(context.Background())
	})

	status := waitForWorkerState(t, supervisor, spec.PluginID, 5*time.Second, func(status PluginWorkerStatus) bool {
		return status.State == PluginWorkerStateFailed
	})
	if status.RestartCount != 0 {
		t.Fatalf("expected restart count 0 when restart policy is disabled, got %d", status.RestartCount)
	}
	healthTimeoutEvents := waitForPluginEventCount(t, events, spec.PluginID, pluginEventHealthTimeout, 1, 2*time.Second)
	if healthTimeoutEvents == 0 && !strings.Contains(strings.ToLower(status.LastError), "health timeout") {
		t.Fatalf("expected health-timeout signal in events or status, status=%+v", status)
	}
}

func TestProcessPluginSupervisorSlowLifecycleHooksDoNotBlockHealthLoop(t *testing.T) {
	supervisor := NewProcessPluginSupervisor()
	events := &recordingPluginEvents{}
	blockHooks := make(chan struct{})
	hookStarted := make(chan struct{}, 1)
	releaseHooks := sync.OnceFunc(func() {
		close(blockHooks)
	})
	supervisor.SetHooks(PluginLifecycleHooks{
		OnEvent: func(event PluginLifecycleEvent) {
			select {
			case hookStarted <- struct{}{}:
			default:
			}
			<-blockHooks
			events.Record(event)
		},
	})

	spec := helperWorkerSpec(t, "no_heartbeat", "plugin.channel.slow-hooks", shared.AdapterKindChannel, []string{"channel.sms.send"}, 0)
	spec.HealthInterval = 25 * time.Millisecond
	spec.HealthTimeout = 90 * time.Millisecond

	if err := supervisor.RegisterWorker(spec); err != nil {
		t.Fatalf("register worker: %v", err)
	}
	if err := supervisor.Start(context.Background()); err != nil {
		t.Fatalf("start supervisor: %v", err)
	}
	t.Cleanup(func() {
		_ = supervisor.Stop(context.Background())
	})
	t.Cleanup(func() {
		releaseHooks()
	})

	select {
	case <-hookStarted:
	case <-time.After(2 * time.Second):
		t.Fatalf("expected lifecycle hook to be invoked")
	}

	startedAt := time.Now()
	status := waitForWorkerState(t, supervisor, spec.PluginID, 2*time.Second, func(status PluginWorkerStatus) bool {
		return status.State == PluginWorkerStateFailed
	})
	if status.State != PluginWorkerStateFailed {
		t.Fatalf("expected failed state, got %+v", status)
	}
	if elapsed := time.Since(startedAt); elapsed > 600*time.Millisecond {
		t.Fatalf("worker loop stalled by slow hooks; expected failure transition in <600ms, got %s", elapsed)
	}

	releaseHooks()
	if count := waitForPluginEventCount(t, events, spec.PluginID, pluginEventWorkerStarted, 1, 2*time.Second); count == 0 {
		t.Fatalf("expected %s event", pluginEventWorkerStarted)
	}
	if count := waitForPluginEventCount(t, events, spec.PluginID, pluginEventHandshakeAccepted, 1, 2*time.Second); count == 0 {
		t.Fatalf("expected %s event", pluginEventHandshakeAccepted)
	}
	if count := waitForPluginEventCount(t, events, spec.PluginID, pluginEventHealthTimeout, 1, 2*time.Second); count == 0 {
		t.Fatalf("expected %s event", pluginEventHealthTimeout)
	}
	if count := waitForPluginEventCount(t, events, spec.PluginID, pluginEventWorkerRestartLimit, 1, 2*time.Second); count == 0 {
		t.Fatalf("expected %s event", pluginEventWorkerRestartLimit)
	}

	ordered := events.ForPlugin(spec.PluginID)
	if len(ordered) < 4 {
		t.Fatalf("expected at least 4 lifecycle events, got %d (%+v)", len(ordered), ordered)
	}
	expectedOrder := []string{
		pluginEventWorkerStarted,
		pluginEventHandshakeAccepted,
		pluginEventHealthTimeout,
		pluginEventWorkerRestartLimit,
	}
	for index, expectedType := range expectedOrder {
		if ordered[index].EventType != expectedType {
			t.Fatalf("expected event[%d]=%s, got %s", index, expectedType, ordered[index].EventType)
		}
		if ordered[index].PluginID != spec.PluginID {
			t.Fatalf("expected event[%d] plugin id %s, got %s", index, spec.PluginID, ordered[index].PluginID)
		}
		if ordered[index].OccurredAt.IsZero() {
			t.Fatalf("expected non-zero occurred_at for event[%d]", index)
		}
	}
	if strings.TrimSpace(ordered[2].Error) == "" {
		t.Fatalf("expected health timeout event to include error payload, got %+v", ordered[2])
	}
	if strings.TrimSpace(ordered[3].ErrorOperation) == "" || strings.TrimSpace(ordered[3].ErrorSource) == "" {
		t.Fatalf("expected restart-limit event to include error context, got %+v", ordered[3])
	}
}

func TestProcessPluginSupervisorHookDispatchQueueSaturationDoesNotBlockWorkerLoop(t *testing.T) {
	supervisor := NewProcessPluginSupervisor()
	blockHooks := make(chan struct{})
	releaseHooks := sync.OnceFunc(func() {
		close(blockHooks)
	})
	supervisor.SetHooks(PluginLifecycleHooks{
		OnEvent: func(_ PluginLifecycleEvent) {
			<-blockHooks
		},
	})

	spec := helperWorkerSpec(t, "crash_with_stderr", "plugin.channel.saturated-hooks", shared.AdapterKindChannel, []string{"channel.sms.send"}, 80)
	spec.RestartPolicy.Delay = 1 * time.Millisecond

	if err := supervisor.RegisterWorker(spec); err != nil {
		t.Fatalf("register worker: %v", err)
	}
	if err := supervisor.Start(context.Background()); err != nil {
		t.Fatalf("start supervisor: %v", err)
	}
	t.Cleanup(func() {
		_ = supervisor.Stop(context.Background())
	})
	t.Cleanup(func() {
		releaseHooks()
	})

	status := waitForWorkerState(t, supervisor, spec.PluginID, 7*time.Second, func(status PluginWorkerStatus) bool {
		return status.State == PluginWorkerStateFailed && status.RestartCount >= spec.RestartPolicy.MaxRestarts
	})
	if status.State != PluginWorkerStateFailed {
		t.Fatalf("expected failed state, got %+v", status)
	}
	if status.RestartCount < spec.RestartPolicy.MaxRestarts {
		t.Fatalf("expected restart_count >= %d after restart-limit failure, got %+v", spec.RestartPolicy.MaxRestarts, status)
	}
	if dropped := supervisor.HookDispatchDroppedCount(); dropped == 0 {
		t.Fatalf("expected hook dispatch drops under saturated queue")
	}
}

func TestProcessPluginSupervisorCapturesFailureContextFromStderr(t *testing.T) {
	supervisor := NewProcessPluginSupervisor()
	events := &recordingPluginEvents{}
	supervisor.SetHooks(PluginLifecycleHooks{
		OnEvent: events.Record,
	})

	spec := helperWorkerSpec(t, "crash_with_stderr", "plugin.channel.stderr-failure", shared.AdapterKindChannel, []string{"channel.sms.send"}, 0)
	if err := supervisor.RegisterWorker(spec); err != nil {
		t.Fatalf("register worker: %v", err)
	}
	if err := supervisor.Start(context.Background()); err != nil {
		t.Fatalf("start supervisor: %v", err)
	}
	t.Cleanup(func() {
		_ = supervisor.Stop(context.Background())
	})

	status := waitForWorkerState(t, supervisor, spec.PluginID, 5*time.Second, func(status PluginWorkerStatus) bool {
		return status.State == PluginWorkerStateFailed && strings.TrimSpace(status.LastError) != ""
	})
	if strings.TrimSpace(status.LastErrorOperation) == "" {
		t.Fatalf("expected failure operation context, got %+v", status)
	}
	if strings.TrimSpace(status.LastErrorSource) == "" {
		t.Fatalf("expected failure source context, got %+v", status)
	}
	if !strings.Contains(status.LastErrorStderr, "helper simulated stderr failure") {
		t.Fatalf("expected stderr context in worker status, got %+v", status)
	}

	lifecycleEvent, ok := waitForPluginEvent(t, events, spec.PluginID, pluginEventWorkerRestartLimit, 2*time.Second)
	if !ok {
		t.Fatalf("expected %s event for stderr failure worker", pluginEventWorkerRestartLimit)
	}
	if strings.TrimSpace(lifecycleEvent.ErrorOperation) == "" || strings.TrimSpace(lifecycleEvent.ErrorSource) == "" {
		t.Fatalf("expected lifecycle event error operation/source context, got %+v", lifecycleEvent)
	}
	if !strings.Contains(lifecycleEvent.ErrorStderr, "helper simulated stderr failure") {
		t.Fatalf("expected lifecycle event stderr context, got %+v", lifecycleEvent)
	}
}

func helperWorkerSpec(t *testing.T, mode string, pluginID string, kind shared.AdapterKind, capabilities []string, maxRestarts int) PluginWorkerSpec {
	t.Helper()
	return PluginWorkerSpec{
		PluginID: pluginID,
		Kind:     kind,
		Command:  os.Args[0],
		Args: []string{
			"-test.run=TestPluginWorkerHelperProcess",
			"--",
			mode,
			pluginID,
			string(kind),
			strings.Join(capabilities, ","),
		},
		Env: []string{
			"PA_PLUGIN_HELPER_PROCESS=1",
		},
		HandshakeTimeout: 2 * time.Second,
		HealthInterval:   50 * time.Millisecond,
		HealthTimeout:    300 * time.Millisecond,
		RestartPolicy: PluginRestartPolicy{
			MaxRestarts: maxRestarts,
			Delay:       50 * time.Millisecond,
		},
	}
}

func waitForWorkerState(t *testing.T, supervisor *ProcessPluginSupervisor, pluginID string, timeout time.Duration, predicate func(status PluginWorkerStatus) bool) PluginWorkerStatus {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		status, ok := supervisor.WorkerStatus(pluginID)
		if ok && predicate(status) {
			return status
		}
		time.Sleep(20 * time.Millisecond)
	}
	status, _ := supervisor.WorkerStatus(pluginID)
	t.Fatalf("timeout waiting for worker %s, last status: %+v", pluginID, status)
	return PluginWorkerStatus{}
}

func waitForPluginEventCount(t *testing.T, events *recordingPluginEvents, pluginID string, eventType string, minimum int, timeout time.Duration) int {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		count := events.Count(pluginID, eventType)
		if count >= minimum {
			return count
		}
		time.Sleep(20 * time.Millisecond)
	}
	return events.Count(pluginID, eventType)
}

func waitForPluginEvent(t *testing.T, events *recordingPluginEvents, pluginID string, eventType string, timeout time.Duration) (PluginLifecycleEvent, bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		event, ok := events.Last(pluginID, eventType)
		if ok {
			return event, true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return events.Last(pluginID, eventType)
}

type recordingPluginEvents struct {
	mu     sync.Mutex
	events []PluginLifecycleEvent
}

func (r *recordingPluginEvents) Record(event PluginLifecycleEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, event)
}

func (r *recordingPluginEvents) Count(pluginID string, eventType string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	count := 0
	for _, event := range r.events {
		if event.PluginID == pluginID && event.EventType == eventType {
			count++
		}
	}
	return count
}

func (r *recordingPluginEvents) Last(pluginID string, eventType string) (PluginLifecycleEvent, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for index := len(r.events) - 1; index >= 0; index-- {
		event := r.events[index]
		if event.PluginID == pluginID && event.EventType == eventType {
			return event, true
		}
	}
	return PluginLifecycleEvent{}, false
}

func (r *recordingPluginEvents) ForPlugin(pluginID string) []PluginLifecycleEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	matches := make([]PluginLifecycleEvent, 0, len(r.events))
	for _, event := range r.events {
		if event.PluginID == pluginID {
			matches = append(matches, event)
		}
	}
	return matches
}

func TestPluginWorkerHelperProcess(t *testing.T) {
	if os.Getenv("PA_PLUGIN_HELPER_PROCESS") != "1" {
		return
	}

	args := flag.Args()
	if len(args) < 4 {
		_, _ = fmt.Fprintf(os.Stderr, "invalid helper args: %v\n", args)
		os.Exit(2)
	}

	mode := strings.TrimSpace(args[0])
	pluginID := strings.TrimSpace(args[1])
	kind := shared.AdapterKind(strings.TrimSpace(args[2]))
	capabilityKeys := splitCapabilities(args[3])
	if strings.TrimSpace(os.Getenv(WorkerExecAuthTokenEnvVar)) == "" {
		_, _ = fmt.Fprintf(os.Stderr, "missing %s\n", WorkerExecAuthTokenEnvVar)
		os.Exit(2)
	}

	metadata := shared.AdapterMetadata{
		ID:          pluginID,
		Kind:        kind,
		DisplayName: pluginID,
		Version:     "test",
	}
	for _, capability := range capabilityKeys {
		metadata.Capabilities = append(metadata.Capabilities, shared.CapabilityDescriptor{Key: capability})
	}

	if err := emitWorkerMessage(pluginWorkerMessage{
		Type:   "handshake",
		Plugin: &metadata,
	}); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "emit handshake: %v\n", err)
		os.Exit(2)
	}

	switch mode {
	case "stable":
		ticker := time.NewTicker(25 * time.Millisecond)
		defer ticker.Stop()
		for {
			<-ticker.C
			_ = emitWorkerMessage(pluginWorkerMessage{
				Type:    "health",
				Healthy: boolPointer(true),
			})
		}
	case "crash_after_handshake":
		_ = emitWorkerMessage(pluginWorkerMessage{
			Type:    "health",
			Healthy: boolPointer(true),
		})
		time.Sleep(100 * time.Millisecond)
		os.Exit(1)
	case "crash_with_stderr":
		_, _ = fmt.Fprintln(os.Stderr, "helper simulated stderr failure")
		time.Sleep(50 * time.Millisecond)
		os.Exit(1)
	case "no_heartbeat":
		for {
			time.Sleep(1 * time.Second)
		}
	default:
		os.Exit(2)
	}
}

func splitCapabilities(raw string) []string {
	parts := strings.Split(raw, ",")
	capabilities := make([]string, 0, len(parts))
	for _, part := range parts {
		key := strings.TrimSpace(part)
		if key == "" {
			continue
		}
		capabilities = append(capabilities, key)
	}
	if len(capabilities) == 0 {
		return []string{"cap.default"}
	}
	return capabilities
}

func emitWorkerMessage(message pluginWorkerMessage) error {
	bytes, err := json.Marshal(message)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(os.Stdout, string(bytes))
	return err
}

func boolPointer(value bool) *bool {
	return &value
}
