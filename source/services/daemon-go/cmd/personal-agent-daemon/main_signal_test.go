package main

import (
	"os"
	"syscall"
	"testing"
)

func TestWaitForDaemonLifecycleActionReturnsSignalStop(t *testing.T) {
	queue := make(chan daemonLifecycleAction, 1)
	signals := make(chan os.Signal, 1)
	signals <- syscall.SIGTERM

	action, stopSignal := waitForDaemonLifecycleAction(queue, signals)
	if action != daemonLifecycleActionStop {
		t.Fatalf("expected stop action for signal-driven shutdown, got %q", action)
	}
	if stopSignal == nil {
		t.Fatalf("expected stop signal metadata")
	}
	if stopSignal.String() != syscall.SIGTERM.String() {
		t.Fatalf("expected SIGTERM stop signal, got %q", stopSignal.String())
	}
}

func TestWaitForDaemonLifecycleActionReturnsQueuedAction(t *testing.T) {
	queue := make(chan daemonLifecycleAction, 1)
	signals := make(chan os.Signal, 1)
	queue <- daemonLifecycleActionRestart

	action, stopSignal := waitForDaemonLifecycleAction(queue, signals)
	if action != daemonLifecycleActionRestart {
		t.Fatalf("expected queued restart action, got %q", action)
	}
	if stopSignal != nil {
		t.Fatalf("expected nil stop signal for queued lifecycle action, got %q", stopSignal.String())
	}
}
