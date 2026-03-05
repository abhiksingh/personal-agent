package daemonruntime

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	shared "personalagent/runtime/internal/shared/contracts"
)

func (s *ProcessPluginSupervisor) runWorkerProcess(runCtx context.Context, worker *managedPluginWorker) (bool, bool, error) {
	spec := worker.spec
	execAuthToken, err := issueWorkerExecAuthToken()
	if err != nil {
		return false, false, newPluginWorkerRuntimeError("bootstrap", "issue_exec_auth_token", nil, fmt.Errorf("issue worker auth token: %w", err))
	}

	command := exec.CommandContext(runCtx, spec.Command, spec.Args...)
	command.Env = append(os.Environ(), withWorkerExecAuthEnv(spec.Env, execAuthToken)...)
	if strings.TrimSpace(spec.WorkingDirectory) != "" {
		command.Dir = spec.WorkingDirectory
	}

	stdout, err := command.StdoutPipe()
	if err != nil {
		return false, false, newPluginWorkerRuntimeError("start", "stdout_pipe", nil, fmt.Errorf("stdout pipe: %w", err))
	}
	stderr, err := command.StderrPipe()
	if err != nil {
		return false, false, newPluginWorkerRuntimeError("start", "stderr_pipe", nil, fmt.Errorf("stderr pipe: %w", err))
	}

	if err := command.Start(); err != nil {
		return false, false, newPluginWorkerRuntimeError("start", "process_start", nil, fmt.Errorf("start plugin worker: %w", err))
	}

	worker.updateStatus(func(status *PluginWorkerStatus) {
		status.State = PluginWorkerStateStarting
		status.ProcessID = command.Process.Pid
		status.execAuthToken = ""
		status.LastTransition = time.Now().UTC()
		clearPluginWorkerErrorContext(status)
	})
	s.emitEvent(worker.snapshot(), pluginEventWorkerStarted, nil)

	waitCh := make(chan error, 1)
	go func() {
		waitCh <- command.Wait()
	}()

	lines := make(chan string, 32)
	scanErr := make(chan error, 1)
	go scanProcessLines(stdout, lines, scanErr)
	stderrLines := make(chan string, 32)
	stderrScanErr := make(chan error, 1)
	go scanProcessLines(stderr, stderrLines, stderrScanErr)

	handshakeTimer := time.NewTimer(spec.HandshakeTimeout)
	defer handshakeTimer.Stop()

	ticker := time.NewTicker(spec.HealthInterval)
	defer ticker.Stop()

	handshakeReceived := false
	lastHeartbeat := time.Time{}
	stderrTail := make([]string, 0, pluginWorkerErrorStderrTailMaxLines)

	for {
		select {
		case <-runCtx.Done():
			_ = command.Process.Kill()
			return false, true, nil
		case <-worker.stopSignal:
			_ = command.Process.Kill()
			return false, true, nil
		case <-worker.restartSignal:
			_ = command.Process.Kill()
			return true, false, errManualRestartRequested
		case waitErr := <-waitCh:
			if waitErr == nil {
				return true, false, newPluginWorkerRuntimeError("wait", "process_exit", stderrTail, fmt.Errorf("plugin worker exited"))
			}
			return true, false, newPluginWorkerRuntimeError("wait", "process_exit", stderrTail, waitErr)
		case line, ok := <-lines:
			if !ok {
				lines = nil
				continue
			}
			message, ok := parsePluginWorkerMessage(line)
			if !ok {
				continue
			}
			switch strings.ToLower(strings.TrimSpace(message.Type)) {
			case "handshake":
				if message.Plugin == nil {
					continue
				}
				metadata := *message.Plugin
				if err := validateHandshakeMetadata(spec, metadata); err != nil {
					_ = command.Process.Kill()
					return true, false, newPluginWorkerRuntimeError("handshake", "handshake_validation", stderrTail, err)
				}
				now := time.Now().UTC()
				handshakeReceived = true
				lastHeartbeat = now
				worker.updateStatus(func(status *PluginWorkerStatus) {
					status.Metadata = metadata
					status.State = PluginWorkerStateRunning
					status.execAuthToken = execAuthToken
					status.LastHeartbeat = now
					status.LastTransition = now
				})
				s.emitEvent(worker.snapshot(), pluginEventHandshakeAccepted, nil)
			case "health":
				if !handshakeReceived {
					continue
				}
				now := time.Now().UTC()
				lastHeartbeat = now
				worker.updateStatus(func(status *PluginWorkerStatus) {
					status.LastHeartbeat = now
					status.State = PluginWorkerStateRunning
				})
			}
		case line, ok := <-stderrLines:
			if !ok {
				stderrLines = nil
				continue
			}
			stderrTail = appendPluginWorkerStderrTail(stderrTail, line)
		case scanError := <-scanErr:
			if scanError != nil {
				return true, false, newPluginWorkerRuntimeError("scan", "stdout_scan", stderrTail, scanError)
			}
			scanErr = nil
		case stderrError := <-stderrScanErr:
			if stderrError != nil {
				return true, false, newPluginWorkerRuntimeError("scan", "stderr_scan", stderrTail, stderrError)
			}
			stderrScanErr = nil
		case <-handshakeTimer.C:
			if !handshakeReceived {
				_ = command.Process.Kill()
				return true, false, newPluginWorkerRuntimeError(
					"handshake",
					"handshake_timeout",
					stderrTail,
					fmt.Errorf("plugin handshake timeout: %s", spec.PluginID),
				)
			}
		case <-ticker.C:
			if !handshakeReceived {
				continue
			}
			if time.Since(lastHeartbeat) > spec.HealthTimeout {
				healthErr := newPluginWorkerRuntimeError(
					"health",
					"health_timeout",
					stderrTail,
					fmt.Errorf("plugin health timeout: %s", spec.PluginID),
				)
				worker.updateStatus(func(status *PluginWorkerStatus) {
					status.State = PluginWorkerStateFailed
					status.LastTransition = time.Now().UTC()
					applyPluginWorkerErrorContext(status, healthErr)
				})
				s.emitEvent(worker.snapshot(), pluginEventHealthTimeout, healthErr)
				_ = command.Process.Kill()
				return true, false, healthErr
			}
		}
	}
}

func normalizePluginWorkerSpec(spec PluginWorkerSpec) (PluginWorkerSpec, error) {
	spec.PluginID = strings.TrimSpace(spec.PluginID)
	spec.Command = strings.TrimSpace(spec.Command)
	if spec.PluginID == "" {
		return PluginWorkerSpec{}, fmt.Errorf("plugin id is required")
	}
	if spec.Kind != shared.AdapterKindChannel && spec.Kind != shared.AdapterKindConnector {
		return PluginWorkerSpec{}, fmt.Errorf("plugin %s has invalid kind %q", spec.PluginID, spec.Kind)
	}
	if spec.Command == "" {
		return PluginWorkerSpec{}, fmt.Errorf("plugin %s command is required", spec.PluginID)
	}
	if spec.HandshakeTimeout <= 0 {
		spec.HandshakeTimeout = 3 * time.Second
	}
	if spec.HealthInterval <= 0 {
		spec.HealthInterval = 500 * time.Millisecond
	}
	if spec.HealthTimeout <= 0 {
		spec.HealthTimeout = 3 * time.Second
	}
	if spec.HealthTimeout < spec.HealthInterval {
		spec.HealthTimeout = spec.HealthInterval
	}
	if spec.RestartPolicy.Delay <= 0 {
		spec.RestartPolicy.Delay = 200 * time.Millisecond
	}
	if spec.RestartPolicy.MaxRestarts < 0 {
		spec.RestartPolicy.MaxRestarts = 0
	}
	return spec, nil
}

func validateHandshakeMetadata(spec PluginWorkerSpec, metadata shared.AdapterMetadata) error {
	if strings.TrimSpace(metadata.ID) != spec.PluginID {
		return fmt.Errorf("plugin handshake id mismatch: expected %s got %s", spec.PluginID, metadata.ID)
	}
	if metadata.Kind != spec.Kind {
		return fmt.Errorf("plugin handshake kind mismatch for %s: expected %s got %s", spec.PluginID, spec.Kind, metadata.Kind)
	}
	if len(metadata.Capabilities) == 0 {
		return fmt.Errorf("plugin %s handshake missing capabilities", spec.PluginID)
	}
	seen := map[string]struct{}{}
	for _, capability := range metadata.Capabilities {
		key := strings.TrimSpace(capability.Key)
		if key == "" {
			return fmt.Errorf("plugin %s handshake includes empty capability key", spec.PluginID)
		}
		if _, exists := seen[key]; exists {
			return fmt.Errorf("plugin %s handshake duplicate capability key: %s", spec.PluginID, key)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func parsePluginWorkerMessage(line string) (pluginWorkerMessage, bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return pluginWorkerMessage{}, false
	}
	message := pluginWorkerMessage{}
	if err := json.Unmarshal([]byte(trimmed), &message); err != nil {
		return pluginWorkerMessage{}, false
	}
	if strings.TrimSpace(message.Type) == "" {
		return pluginWorkerMessage{}, false
	}
	return message, true
}

func scanProcessLines(reader io.Reader, lines chan<- string, scanErr chan<- error) {
	defer close(lines)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		lines <- scanner.Text()
	}
	scanErr <- scanner.Err()
}
