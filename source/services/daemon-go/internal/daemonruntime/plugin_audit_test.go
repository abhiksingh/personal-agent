package daemonruntime

import (
	"testing"
	"time"

	shared "personalagent/runtime/internal/shared/contracts"
)

func TestPluginLifecycleAuditRuntimeQueueSaturationDoesNotBlockEnqueue(t *testing.T) {
	blockPersist := make(chan struct{})
	persistStarted := make(chan struct{}, 1)
	auditRuntime := newPluginLifecycleAuditRuntime(1, func(PluginLifecycleEvent) error {
		select {
		case persistStarted <- struct{}{}:
		default:
		}
		<-blockPersist
		return nil
	})
	auditRuntime.start()
	defer auditRuntime.stop()

	event := PluginLifecycleEvent{
		PluginID:  "plugin.channel.audit.saturation",
		Kind:      shared.AdapterKindChannel,
		State:     PluginWorkerStateRunning,
		EventType: pluginEventHandshakeAccepted,
	}
	auditRuntime.enqueue(event)

	select {
	case <-persistStarted:
	case <-time.After(1 * time.Second):
		t.Fatalf("timed out waiting for audit persistence to start")
	}

	done := make(chan struct{})
	go func() {
		for i := 0; i < 512; i++ {
			auditRuntime.enqueue(event)
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(250 * time.Millisecond):
		t.Fatalf("enqueue blocked while audit persistence worker was backpressured")
	}

	if diagnostics := auditRuntime.diagnostics(); diagnostics.DroppedEvents == 0 {
		t.Fatalf("expected dropped audit events under saturation, got %+v", diagnostics)
	}

	close(blockPersist)
}
