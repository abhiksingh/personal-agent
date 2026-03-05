package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"personalagent/runtime/internal/daemonruntime"
	shared "personalagent/runtime/internal/shared/contracts"
)

const (
	envCloudflaredBinary  = "PA_CLOUDFLARED_BINARY"
	envCloudflaredDryRun  = "PA_CLOUDFLARED_DRY_RUN"
	defaultCloudflaredBin = "cloudflared"
	defaultExecTimeoutMS  = int64(30000)
)

type cloudflaredWorkerExecuteRequest struct {
	Operation string          `json:"operation"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

type cloudflaredWorkerExecuteResponse struct {
	Result any    `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

type cloudflaredVersionResult struct {
	Available  bool   `json:"available"`
	BinaryPath string `json:"binary_path"`
	Version    string `json:"version,omitempty"`
	Stdout     string `json:"stdout,omitempty"`
	Stderr     string `json:"stderr,omitempty"`
	ExitCode   int    `json:"exit_code"`
	DryRun     bool   `json:"dry_run"`
	Error      string `json:"error,omitempty"`
}

type cloudflaredExecRequest struct {
	Args      []string `json:"args"`
	TimeoutMS int64    `json:"timeout_ms,omitempty"`
}

type cloudflaredExecResult struct {
	Success    bool     `json:"success"`
	BinaryPath string   `json:"binary_path"`
	Args       []string `json:"args"`
	ExitCode   int      `json:"exit_code"`
	Stdout     string   `json:"stdout,omitempty"`
	Stderr     string   `json:"stderr,omitempty"`
	TimedOut   bool     `json:"timed_out"`
	DurationMS int64    `json:"duration_ms"`
	DryRun     bool     `json:"dry_run"`
	Error      string   `json:"error,omitempty"`
}

type cloudflaredWorkerState struct{}

type cloudflaredAllowlistedExecCommand string

const (
	cloudflaredAllowlistedExecVersion          cloudflaredAllowlistedExecCommand = "version"
	cloudflaredAllowlistedExecTunnelList       cloudflaredAllowlistedExecCommand = "tunnel_list"
	cloudflaredAllowlistedExecTunnelListOutput cloudflaredAllowlistedExecCommand = "tunnel_list_output_json"
)

func runCloudflaredConnectorWorker(pluginID string, healthInterval time.Duration) error {
	execAuthToken, err := loadWorkerExecAuthTokenFromEnv()
	if err != nil {
		return err
	}
	if healthInterval <= 0 {
		healthInterval = 250 * time.Millisecond
	}

	trimmedID := strings.TrimSpace(pluginID)
	if trimmedID == "" {
		trimmedID = daemonruntime.CloudflaredConnectorPluginID
	}
	state := &cloudflaredWorkerState{}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("listen cloudflared worker execute endpoint: %w", err)
	}
	defer listener.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/execute", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			writer.Header().Set("Allow", http.MethodPost)
			writer.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if !authorizeWorkerExecuteRequest(request, execAuthToken) {
			writeWorkerUnauthorized(writer)
			return
		}

		var payload cloudflaredWorkerExecuteRequest
		if statusCode, err := decodeWorkerExecuteJSONPayload(writer, request, &payload, "cloudflared execute"); err != nil {
			writeWorkerError(writer, statusCode, err)
			return
		}

		result, execErr := state.execute(request.Context(), strings.TrimSpace(payload.Operation), payload.Payload)
		if execErr != nil {
			writeWorkerError(writer, http.StatusBadRequest, execErr)
			return
		}

		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(cloudflaredWorkerExecuteResponse{Result: result})
	})

	server := newWorkerExecuteHTTPServer(mux)
	go func() {
		_ = server.Serve(listener)
	}()

	metadata := shared.AdapterMetadata{
		ID:          trimmedID,
		Kind:        shared.AdapterKindConnector,
		DisplayName: "Cloudflared Connector",
		Version:     "0.2.0",
		Capabilities: []shared.CapabilityDescriptor{
			{Key: daemonruntime.CloudflaredConnectorCapabilityVersion, Description: "Inspect cloudflared availability/version"},
			{Key: daemonruntime.CloudflaredConnectorCapabilityExec, Description: "Proxy cloudflared command execution"},
		},
		Runtime: map[string]string{
			connectorWorkerExecAddressKey: listener.Addr().String(),
		},
	}
	if err := emitWorkerMessage(workerMessage{Type: "handshake", Plugin: &metadata}); err != nil {
		return fmt.Errorf("emit cloudflared worker handshake: %w", err)
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)

	ticker := time.NewTicker(healthInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			healthy := true
			if err := emitWorkerMessage(workerMessage{Type: "health", Healthy: &healthy}); err != nil {
				return fmt.Errorf("emit cloudflared worker health: %w", err)
			}
		case <-signals:
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = server.Shutdown(shutdownCtx)
			return nil
		}
	}
}

func (s *cloudflaredWorkerState) execute(ctx context.Context, operation string, rawPayload json.RawMessage) (any, error) {
	switch strings.TrimSpace(operation) {
	case daemonruntime.CloudflaredConnectorOperationVersion:
		return s.executeVersion(ctx), nil
	case daemonruntime.CloudflaredConnectorOperationExec:
		var request cloudflaredExecRequest
		if err := decodeCloudflaredPayload(rawPayload, &request); err != nil {
			return nil, err
		}
		return s.executeExec(ctx, request), nil
	default:
		return nil, fmt.Errorf("unsupported cloudflared operation %q", operation)
	}
}

func (s *cloudflaredWorkerState) executeVersion(ctx context.Context) cloudflaredVersionResult {
	binaryPath, resolveErr := resolveCloudflaredBinaryPath()
	if resolveErr != nil {
		return cloudflaredVersionResult{
			Available:  false,
			BinaryPath: configuredCloudflaredBinaryPath(),
			ExitCode:   -1,
			DryRun:     isCloudflaredDryRunEnabled(),
			Error:      resolveErr.Error(),
		}
	}
	if isCloudflaredDryRunEnabled() {
		return cloudflaredVersionResult{
			Available:  true,
			BinaryPath: binaryPath,
			Version:    "cloudflared dry-run",
			Stdout:     "cloudflared dry-run",
			ExitCode:   0,
			DryRun:     true,
		}
	}

	stdout, stderr, exitCode, err := runCloudflaredVersionCommand(ctx, binaryPath)
	result := cloudflaredVersionResult{
		Available:  err == nil && exitCode == 0,
		BinaryPath: binaryPath,
		Version:    cloudflaredVersionFromOutput(stdout),
		Stdout:     stdout,
		Stderr:     stderr,
		ExitCode:   exitCode,
		DryRun:     false,
	}
	if err != nil {
		result.Error = err.Error()
	}
	return result
}

func (s *cloudflaredWorkerState) executeExec(ctx context.Context, request cloudflaredExecRequest) cloudflaredExecResult {
	args := trimStringSlice(request.Args)
	binaryPath := configuredCloudflaredBinaryPath()
	if len(args) == 0 {
		return cloudflaredExecResult{
			Success:    false,
			BinaryPath: binaryPath,
			Args:       []string{},
			ExitCode:   -1,
			DryRun:     isCloudflaredDryRunEnabled(),
			Error:      "args are required",
		}
	}
	resolvedBinaryPath, resolveErr := resolveCloudflaredBinaryPath()
	if resolveErr != nil {
		return cloudflaredExecResult{
			Success:    false,
			BinaryPath: binaryPath,
			Args:       append([]string{}, args...),
			ExitCode:   -1,
			DryRun:     isCloudflaredDryRunEnabled(),
			Error:      resolveErr.Error(),
		}
	}
	binaryPath = resolvedBinaryPath
	allowlistedCommand, allowlistedArgs, allowlistErr := allowlistedCloudflaredExecArgs(args)
	if allowlistErr != nil {
		return cloudflaredExecResult{
			Success:    false,
			BinaryPath: binaryPath,
			Args:       append([]string{}, args...),
			ExitCode:   -1,
			DryRun:     isCloudflaredDryRunEnabled(),
			Error:      allowlistErr.Error(),
		}
	}
	if isCloudflaredDryRunEnabled() {
		return cloudflaredExecResult{
			Success:    true,
			BinaryPath: binaryPath,
			Args:       append([]string{}, allowlistedArgs...),
			ExitCode:   0,
			Stdout:     "cloudflared dry-run exec: " + strings.Join(allowlistedArgs, " "),
			DryRun:     true,
		}
	}

	timeoutMS := request.TimeoutMS
	if timeoutMS <= 0 {
		timeoutMS = defaultExecTimeoutMS
	}
	startedAt := time.Now()
	execCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMS)*time.Millisecond)
	defer cancel()

	stdout, stderr, exitCode, err := runCloudflaredAllowlistedExec(execCtx, binaryPath, allowlistedCommand)
	timedOut := errors.Is(execCtx.Err(), context.DeadlineExceeded)
	result := cloudflaredExecResult{
		Success:    err == nil && exitCode == 0 && !timedOut,
		BinaryPath: binaryPath,
		Args:       append([]string{}, allowlistedArgs...),
		ExitCode:   exitCode,
		Stdout:     stdout,
		Stderr:     stderr,
		TimedOut:   timedOut,
		DurationMS: time.Since(startedAt).Milliseconds(),
		DryRun:     false,
	}
	if err != nil {
		result.Error = err.Error()
	}
	return result
}

func allowlistedCloudflaredExecArgs(args []string) (cloudflaredAllowlistedExecCommand, []string, error) {
	normalized := normalizeCloudflaredExecArgs(args)
	switch strings.Join(normalized, " ") {
	case "version":
		return cloudflaredAllowlistedExecVersion, []string{"version"}, nil
	case "tunnel list":
		return cloudflaredAllowlistedExecTunnelList, []string{"tunnel", "list"}, nil
	case "tunnel list --output json":
		return cloudflaredAllowlistedExecTunnelListOutput, []string{"tunnel", "list", "--output", "json"}, nil
	case "tunnel list --output=json":
		return cloudflaredAllowlistedExecTunnelListOutput, []string{"tunnel", "list", "--output=json"}, nil
	default:
		return "", nil, fmt.Errorf(
			"unsupported cloudflared exec args %q; allowed commands: version | tunnel list | tunnel list --output json",
			strings.Join(args, " "),
		)
	}
}

func normalizeCloudflaredExecArgs(args []string) []string {
	if len(args) == 0 {
		return nil
	}
	normalized := make([]string, 0, len(args))
	for _, arg := range args {
		candidate := strings.ToLower(strings.TrimSpace(arg))
		if candidate == "" {
			continue
		}
		normalized = append(normalized, candidate)
	}
	return normalized
}

func decodeCloudflaredPayload(raw json.RawMessage, target any) error {
	if target == nil {
		return nil
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return nil
	}
	if err := json.Unmarshal(trimmed, target); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}
	return nil
}

func runCloudflaredVersionCommand(ctx context.Context, binaryPath string) (string, string, int, error) {
	cmd := exec.CommandContext(ctx, binaryPath, "version")
	return runCloudflaredCommand(cmd)
}

func runCloudflaredAllowlistedExec(ctx context.Context, binaryPath string, command cloudflaredAllowlistedExecCommand) (string, string, int, error) {
	switch command {
	case cloudflaredAllowlistedExecVersion:
		cmd := exec.CommandContext(ctx, binaryPath, "version")
		return runCloudflaredCommand(cmd)
	case cloudflaredAllowlistedExecTunnelList:
		cmd := exec.CommandContext(ctx, binaryPath, "tunnel", "list")
		return runCloudflaredCommand(cmd)
	case cloudflaredAllowlistedExecTunnelListOutput:
		cmd := exec.CommandContext(ctx, binaryPath, "tunnel", "list", "--output", "json")
		return runCloudflaredCommand(cmd)
	default:
		return "", "", -1, fmt.Errorf("unsupported allowlisted cloudflared command %q", command)
	}
}

func runCloudflaredCommand(cmd *exec.Cmd) (string, string, int, error) {
	var stdoutBuffer bytes.Buffer
	var stderrBuffer bytes.Buffer
	cmd.Stdout = &stdoutBuffer
	cmd.Stderr = &stderrBuffer
	err := cmd.Run()

	stdout := strings.TrimSpace(stdoutBuffer.String())
	stderr := strings.TrimSpace(stderrBuffer.String())
	exitCode := 0
	if err != nil {
		exitCode = -1
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		}
	}
	return stdout, stderr, exitCode, err
}

func configuredCloudflaredBinaryPath() string {
	if configured := strings.TrimSpace(os.Getenv(envCloudflaredBinary)); configured != "" {
		return configured
	}
	return defaultCloudflaredBin
}

func resolveCloudflaredBinaryPath() (string, error) {
	configured := strings.TrimSpace(os.Getenv(envCloudflaredBinary))
	if configured == "" {
		return defaultCloudflaredBin, nil
	}
	return validateCloudflaredBinaryOverride(configured)
}

func validateCloudflaredBinaryOverride(path string) (string, error) {
	candidate := filepath.Clean(strings.TrimSpace(path))
	if candidate == "" {
		return "", fmt.Errorf("cloudflared binary override is empty")
	}
	if !filepath.IsAbs(candidate) {
		return "", fmt.Errorf("cloudflared binary override must be an absolute path")
	}
	info, err := os.Lstat(candidate)
	if err != nil {
		return "", fmt.Errorf("cloudflared binary override is not accessible: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("cloudflared binary override must not be a symlink")
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("cloudflared binary override must point to a regular file")
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o111 == 0 {
		return "", fmt.Errorf("cloudflared binary override must be executable")
	}
	return candidate, nil
}

func isCloudflaredDryRunEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(envCloudflaredDryRun))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func cloudflaredVersionFromOutput(output string) string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func trimStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	trimmed := make([]string, 0, len(values))
	for _, value := range values {
		candidate := strings.TrimSpace(value)
		if candidate == "" {
			continue
		}
		trimmed = append(trimmed, candidate)
	}
	return trimmed
}
