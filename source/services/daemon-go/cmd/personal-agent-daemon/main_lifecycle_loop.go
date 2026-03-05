package main

import (
	"context"
	"fmt"
	"io"
	"os"
)

func runDaemonMainLoop(
	config daemonRunConfig,
	executable string,
	actionRequests chan daemonLifecycleAction,
	signals <-chan os.Signal,
	stdout io.Writer,
) error {
	for {
		restart, err := runDaemonInstance(config, executable, actionRequests, signals)
		if err != nil {
			return err
		}
		if !restart {
			return nil
		}
		fmt.Fprintln(stdout, "daemon restart requested; starting a new daemon runtime instance")
	}
}

func requestDaemonLifecycleAction(queue chan daemonLifecycleAction, action daemonLifecycleAction) func(ctx context.Context) error {
	return func(_ context.Context) error {
		select {
		case queue <- action:
		default:
		}
		return nil
	}
}

func drainPendingDaemonLifecycleActions(queue chan daemonLifecycleAction) {
	for {
		select {
		case <-queue:
		default:
			return
		}
	}
}

func waitForDaemonLifecycleAction(queue <-chan daemonLifecycleAction, signals <-chan os.Signal) (daemonLifecycleAction, os.Signal) {
	select {
	case stopSignal := <-signals:
		return daemonLifecycleActionStop, stopSignal
	case action := <-queue:
		return action, nil
	}
}
