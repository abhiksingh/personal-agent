package daemonruntime

import (
	"context"
	"fmt"
	"strings"
	"time"

	"personalagent/runtime/internal/transport"
)

func (s *DaemonLifecycleService) requestLifecycleTransition(
	ctx context.Context,
	action string,
	targetState string,
	hook func(ctx context.Context) error,
) (transport.DaemonLifecycleControlResponse, error) {
	current := s.currentLifecycleState()
	now := s.now().UTC()
	if current == targetState {
		return transport.DaemonLifecycleControlResponse{
			Action:         action,
			Accepted:       true,
			Idempotent:     true,
			LifecycleState: current,
			Message:        fmt.Sprintf("daemon %s already requested", action),
			OperationState: lifecycleOperationStateSucceeded,
			RequestedAt:    formatRFC3339(now),
			CompletedAt:    formatRFC3339(now),
		}, nil
	}
	if targetState == lifecycleStateStopRequested && current == lifecycleStateRestartRequested {
		return transport.DaemonLifecycleControlResponse{
			Action:         action,
			Accepted:       true,
			Idempotent:     true,
			LifecycleState: current,
			Message:        "daemon restart is already requested",
			OperationState: lifecycleOperationStateSucceeded,
			RequestedAt:    formatRFC3339(now),
			CompletedAt:    formatRFC3339(now),
		}, nil
	}
	if targetState == lifecycleStateRestartRequested && current == lifecycleStateStopRequested {
		return transport.DaemonLifecycleControlResponse{
			Action:         action,
			Accepted:       true,
			Idempotent:     true,
			LifecycleState: current,
			Message:        "daemon stop is already requested",
			OperationState: lifecycleOperationStateSucceeded,
			RequestedAt:    formatRFC3339(now),
			CompletedAt:    formatRFC3339(now),
		}, nil
	}

	if hook == nil {
		return transport.DaemonLifecycleControlResponse{}, fmt.Errorf("%s control hook is not configured", action)
	}

	s.setLifecycleState(targetState)
	if err := hook(ctx); err != nil {
		s.setLifecycleState(current)
		return transport.DaemonLifecycleControlResponse{}, fmt.Errorf("request daemon %s: %w", action, err)
	}
	return transport.DaemonLifecycleControlResponse{
		Action:         action,
		Accepted:       true,
		Idempotent:     false,
		LifecycleState: targetState,
		Message:        fmt.Sprintf("daemon %s requested", action),
		OperationState: lifecycleOperationStateSucceeded,
		RequestedAt:    formatRFC3339(now),
		CompletedAt:    formatRFC3339(now),
	}, nil
}

func (s *DaemonLifecycleService) requestSetupOperation(
	action string,
	request transport.DaemonLifecycleControlRequest,
) (transport.DaemonLifecycleControlResponse, error) {
	hook := s.setupOperationHook(action)

	s.mu.Lock()
	currentLifecycleState := s.lifecycleState
	currentOperation := s.setupOperation
	if strings.TrimSpace(currentOperation.State) == "" {
		currentOperation.State = lifecycleOperationStateIdle
	}
	if currentOperation.State == lifecycleOperationStateInProgress {
		response := transport.DaemonLifecycleControlResponse{
			Action:         action,
			Accepted:       currentOperation.Action == action,
			Idempotent:     true,
			LifecycleState: currentLifecycleState,
			Message:        fmt.Sprintf("daemon %s operation is already in progress", currentOperation.Action),
			OperationState: lifecycleOperationStateInProgress,
			RequestedAt:    formatRFC3339(currentOperation.RequestedAt),
		}
		s.mu.Unlock()
		return response, nil
	}

	now := s.now().UTC()
	s.setupOperation = daemonLifecycleOperationState{
		Action:      action,
		State:       lifecycleOperationStateInProgress,
		Message:     fmt.Sprintf("daemon %s operation in progress", action),
		RequestedAt: now,
	}
	s.mu.Unlock()

	go s.executeSetupOperation(action, request, hook)

	return transport.DaemonLifecycleControlResponse{
		Action:         action,
		Accepted:       true,
		Idempotent:     false,
		LifecycleState: currentLifecycleState,
		Message:        fmt.Sprintf("daemon %s operation started", action),
		OperationState: lifecycleOperationStateInProgress,
		RequestedAt:    formatRFC3339(now),
	}, nil
}

func (s *DaemonLifecycleService) executeSetupOperation(
	action string,
	request transport.DaemonLifecycleControlRequest,
	hook daemonLifecycleSetupActionHook,
) {
	actionCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var err error
	if hook != nil {
		err = hook(actionCtx, request)
	}
	s.completeSetupOperation(action, err)
}

func (s *DaemonLifecycleService) completeSetupOperation(action string, operationErr error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.setupOperation.Action != action || s.setupOperation.State != lifecycleOperationStateInProgress {
		return
	}
	s.setupOperation.CompletedAt = s.now().UTC()
	if operationErr != nil {
		s.setupOperation.State = lifecycleOperationStateFailed
		s.setupOperation.Error = strings.TrimSpace(operationErr.Error())
		s.setupOperation.Message = fmt.Sprintf("daemon %s operation failed", action)
		return
	}
	s.setupOperation.State = lifecycleOperationStateSucceeded
	s.setupOperation.Error = ""
	s.setupOperation.Message = fmt.Sprintf("daemon %s operation completed", action)
}

func (s *DaemonLifecycleService) setupOperationHook(action string) daemonLifecycleSetupActionHook {
	switch action {
	case lifecycleActionInstall:
		if s.requestInstall != nil {
			return s.requestInstall
		}
		return s.defaultInstallOperation
	case lifecycleActionUninstall:
		if s.requestUninstall != nil {
			return s.requestUninstall
		}
		return s.defaultUninstallOperation
	case lifecycleActionRepair:
		if s.requestRepair != nil {
			return s.requestRepair
		}
		return s.defaultRepairOperation
	default:
		return nil
	}
}

func (s *DaemonLifecycleService) defaultInstallOperation(_ context.Context, _ transport.DaemonLifecycleControlRequest) error {
	_, needsInstall := resolveInstallState(s.executablePath)
	if needsInstall {
		return fmt.Errorf("daemon executable is missing; install cannot proceed from this runtime")
	}
	return nil
}

func (s *DaemonLifecycleService) defaultUninstallOperation(_ context.Context, _ transport.DaemonLifecycleControlRequest) error {
	// Uninstall is an external host concern; this API acknowledges the request deterministically.
	return nil
}

func (s *DaemonLifecycleService) defaultRepairOperation(ctx context.Context, _ transport.DaemonLifecycleControlRequest) error {
	databaseReady, databaseError := resolveDatabaseState(ctx, s.container)
	if !databaseReady {
		if strings.TrimSpace(databaseError) == "" {
			return fmt.Errorf("daemon database is not reachable")
		}
		return fmt.Errorf("daemon database is not reachable: %s", databaseError)
	}
	if s.container == nil || s.container.PluginSupervisor == nil {
		return nil
	}
	workers := s.container.PluginSupervisor.ListWorkers()
	for _, worker := range workers {
		if strings.ToLower(strings.TrimSpace(string(worker.State))) != string(PluginWorkerStateFailed) {
			continue
		}
		if err := s.container.PluginSupervisor.RestartWorker(ctx, worker.PluginID); err != nil {
			return fmt.Errorf("restart failed worker %s: %w", worker.PluginID, err)
		}
	}
	return nil
}

func transportLifecycleOperation(input daemonLifecycleOperationState) transport.DaemonLifecycleControlOperation {
	state := strings.TrimSpace(input.State)
	if state == "" {
		state = lifecycleOperationStateIdle
	}
	return transport.DaemonLifecycleControlOperation{
		Action:      strings.TrimSpace(input.Action),
		State:       state,
		Message:     strings.TrimSpace(input.Message),
		Error:       strings.TrimSpace(input.Error),
		RequestedAt: formatRFC3339(input.RequestedAt),
		CompletedAt: formatRFC3339(input.CompletedAt),
	}
}

func daemonControlAuthState(token string, source string) transport.DaemonControlAuthState {
	normalizedToken := strings.TrimSpace(token)
	normalizedSource := normalizeDaemonControlAuthSource(source)
	if normalizedToken == "" {
		return transport.DaemonControlAuthState{
			State:  daemonControlAuthStateMissing,
			Source: normalizedSource,
			RemediationHints: []string{
				"Start daemon with --auth-token-file <path> or --auth-token <token>.",
				"Generate a token via `personal-agent auth bootstrap --file <path>` for local development.",
			},
		}
	}
	return transport.DaemonControlAuthState{
		State:  daemonControlAuthStateConfigured,
		Source: normalizedSource,
	}
}

func normalizeDaemonControlAuthSource(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case daemonControlAuthSourceFlag:
		return daemonControlAuthSourceFlag
	case daemonControlAuthSourceFile:
		return daemonControlAuthSourceFile
	default:
		return daemonControlAuthSourceUnknown
	}
}

func (s *DaemonLifecycleService) currentLifecycleState() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lifecycleState
}

func (s *DaemonLifecycleService) setLifecycleState(state string) {
	s.mu.Lock()
	s.lifecycleState = state
	s.lastTransition = s.now().UTC()
	s.mu.Unlock()
}
