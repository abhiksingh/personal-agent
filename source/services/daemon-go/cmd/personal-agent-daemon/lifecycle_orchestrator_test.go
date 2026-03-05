package main

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestStartDaemonLifecyclePhasesRunsInOrder(t *testing.T) {
	events := []string{}
	phases := []daemonLifecyclePhase{
		{
			name: "phase-a",
			start: func(context.Context) error {
				events = append(events, "start:phase-a")
				return nil
			},
			stop: func(context.Context) error {
				events = append(events, "stop:phase-a")
				return nil
			},
		},
		{
			name: "phase-b",
			start: func(context.Context) error {
				events = append(events, "start:phase-b")
				return nil
			},
			stop: func(context.Context) error {
				events = append(events, "stop:phase-b")
				return nil
			},
		},
	}

	if err := startDaemonLifecyclePhases(context.Background(), phases); err != nil {
		t.Fatalf("start phases: %v", err)
	}

	expected := []string{"start:phase-a", "start:phase-b"}
	if !reflect.DeepEqual(events, expected) {
		t.Fatalf("events=%v want %v", events, expected)
	}
}

func TestStartDaemonLifecyclePhasesRollsBackOnFailure(t *testing.T) {
	events := []string{}
	startErr := errors.New("boom")
	phases := []daemonLifecyclePhase{
		{
			name: "phase-a",
			start: func(context.Context) error {
				events = append(events, "start:phase-a")
				return nil
			},
			stop: func(context.Context) error {
				events = append(events, "stop:phase-a")
				return nil
			},
		},
		{
			name: "phase-b",
			start: func(context.Context) error {
				events = append(events, "start:phase-b")
				return startErr
			},
			stop: func(context.Context) error {
				events = append(events, "stop:phase-b")
				return nil
			},
		},
	}

	err := startDaemonLifecyclePhases(context.Background(), phases)
	if err == nil {
		t.Fatalf("expected startup failure")
	}
	if !strings.Contains(err.Error(), "phase phase-b start") {
		t.Fatalf("expected phase-b startup failure in error, got %v", err)
	}

	expected := []string{"start:phase-a", "start:phase-b", "stop:phase-a"}
	if !reflect.DeepEqual(events, expected) {
		t.Fatalf("events=%v want %v", events, expected)
	}
}

func TestStopDaemonLifecyclePhasesAggregatesFailures(t *testing.T) {
	phases := []daemonLifecyclePhase{
		{
			name: "phase-a",
			stop: func(context.Context) error {
				return errors.New("failure-a")
			},
		},
		{
			name: "phase-b",
			stop: func(context.Context) error {
				return errors.New("failure-b")
			},
		},
	}

	err := stopDaemonLifecyclePhases(context.Background(), phases)
	if err == nil {
		t.Fatalf("expected shutdown failure")
	}
	if !strings.Contains(err.Error(), "phase-a: failure-a") {
		t.Fatalf("expected phase-a failure in error, got %v", err)
	}
	if !strings.Contains(err.Error(), "phase-b: failure-b") {
		t.Fatalf("expected phase-b failure in error, got %v", err)
	}
}
