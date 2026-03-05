package main

import (
	"context"
	"fmt"
	"strings"
)

type daemonLifecyclePhase struct {
	name  string
	start func(context.Context) error
	stop  func(context.Context) error
}

func startDaemonLifecyclePhases(ctx context.Context, phases []daemonLifecyclePhase) error {
	started := make([]daemonLifecyclePhase, 0, len(phases))
	for _, phase := range phases {
		if phase.start != nil {
			if err := phase.start(ctx); err != nil {
				rollbackErr := stopDaemonLifecyclePhases(ctx, reverseDaemonLifecyclePhases(started))
				if rollbackErr != nil {
					return fmt.Errorf("phase %s start: %w (rollback failed: %v)", phase.name, err, rollbackErr)
				}
				return fmt.Errorf("phase %s start: %w", phase.name, err)
			}
		}
		started = append(started, phase)
	}
	return nil
}

func stopDaemonLifecyclePhases(ctx context.Context, phases []daemonLifecyclePhase) error {
	if len(phases) == 0 {
		return nil
	}

	failures := make([]string, 0)
	for _, phase := range phases {
		if phase.stop == nil {
			continue
		}
		if err := phase.stop(ctx); err != nil {
			phaseName := strings.TrimSpace(phase.name)
			if phaseName == "" {
				phaseName = "unnamed"
			}
			failures = append(failures, fmt.Sprintf("%s: %v", phaseName, err))
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("lifecycle shutdown failures: %s", strings.Join(failures, "; "))
	}
	return nil
}

func reverseDaemonLifecyclePhases(phases []daemonLifecyclePhase) []daemonLifecyclePhase {
	reversed := make([]daemonLifecyclePhase, 0, len(phases))
	for index := len(phases) - 1; index >= 0; index-- {
		reversed = append(reversed, phases[index])
	}
	return reversed
}
