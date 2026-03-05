package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"personalagent/runtime/internal/daemonruntime"
)

func TestCloudflaredWorkerStateExecuteUnsupportedOperation(t *testing.T) {
	state := &cloudflaredWorkerState{}
	_, err := state.execute(context.Background(), "unsupported", json.RawMessage(`{}`))
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "unsupported") {
		t.Fatalf("expected unsupported cloudflared operation error, got %v", err)
	}
}

func TestCloudflaredWorkerStateExecutePayloadDecodeFailure(t *testing.T) {
	state := &cloudflaredWorkerState{}
	_, err := state.execute(context.Background(), daemonruntime.CloudflaredConnectorOperationExec, json.RawMessage(`{`))
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "decode payload") {
		t.Fatalf("expected decode payload error, got %v", err)
	}
}

func TestCloudflaredWorkerStateExecuteVersionDryRun(t *testing.T) {
	t.Setenv(envCloudflaredDryRun, "1")
	overridePath := createCloudflaredTestBinary(t, filepath.Join(t.TempDir(), "cloudflared"), true)
	t.Setenv(envCloudflaredBinary, overridePath)

	state := &cloudflaredWorkerState{}
	resultRaw, err := state.execute(context.Background(), daemonruntime.CloudflaredConnectorOperationVersion, nil)
	if err != nil {
		t.Fatalf("execute version: %v", err)
	}

	result, ok := resultRaw.(cloudflaredVersionResult)
	if !ok {
		t.Fatalf("expected cloudflaredVersionResult, got %T", resultRaw)
	}
	if !result.Available || !result.DryRun || result.ExitCode != 0 {
		t.Fatalf("expected available dry-run version result, got %+v", result)
	}
	if result.BinaryPath != overridePath {
		t.Fatalf("expected configured binary path, got %s", result.BinaryPath)
	}
}

func TestCloudflaredWorkerStateExecuteVersionRejectsSymlinkOverride(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink behavior is environment-specific on windows")
	}

	t.Setenv(envCloudflaredDryRun, "1")
	tempDir := t.TempDir()
	targetPath := createCloudflaredTestBinary(t, filepath.Join(tempDir, "cloudflared"), true)
	symlinkPath := filepath.Join(tempDir, "cloudflared-link")
	if err := os.Symlink(targetPath, symlinkPath); err != nil {
		t.Fatalf("create symlink override: %v", err)
	}
	t.Setenv(envCloudflaredBinary, symlinkPath)

	state := &cloudflaredWorkerState{}
	resultRaw, err := state.execute(context.Background(), daemonruntime.CloudflaredConnectorOperationVersion, nil)
	if err != nil {
		t.Fatalf("execute version with symlink override: %v", err)
	}

	result, ok := resultRaw.(cloudflaredVersionResult)
	if !ok {
		t.Fatalf("expected cloudflaredVersionResult, got %T", resultRaw)
	}
	if result.Available {
		t.Fatalf("expected symlink override to be rejected, got %+v", result)
	}
	if !strings.Contains(strings.ToLower(result.Error), "must not be a symlink") {
		t.Fatalf("expected symlink rejection error, got %+v", result)
	}
}

func TestCloudflaredWorkerStateExecuteExecDryRun(t *testing.T) {
	t.Setenv(envCloudflaredDryRun, "1")
	state := &cloudflaredWorkerState{}

	payload := json.RawMessage(`{"args":["version"]}`)
	resultRaw, err := state.execute(context.Background(), daemonruntime.CloudflaredConnectorOperationExec, payload)
	if err != nil {
		t.Fatalf("execute dry-run exec: %v", err)
	}

	result, ok := resultRaw.(cloudflaredExecResult)
	if !ok {
		t.Fatalf("expected cloudflaredExecResult, got %T", resultRaw)
	}
	if !result.Success || !result.DryRun || result.ExitCode != 0 {
		t.Fatalf("expected successful dry-run exec result, got %+v", result)
	}
	if len(result.Args) != 1 || result.Args[0] != "version" {
		t.Fatalf("expected version arg echo, got %+v", result.Args)
	}
}

func TestCloudflaredWorkerStateExecuteExecDryRunUsesValidatedOverridePath(t *testing.T) {
	t.Setenv(envCloudflaredDryRun, "1")
	overridePath := createCloudflaredTestBinary(t, filepath.Join(t.TempDir(), "cloudflared"), true)
	t.Setenv(envCloudflaredBinary, " "+overridePath+" ")
	state := &cloudflaredWorkerState{}

	payload := json.RawMessage(`{"args":["version"]}`)
	resultRaw, err := state.execute(context.Background(), daemonruntime.CloudflaredConnectorOperationExec, payload)
	if err != nil {
		t.Fatalf("execute dry-run exec with override: %v", err)
	}

	result, ok := resultRaw.(cloudflaredExecResult)
	if !ok {
		t.Fatalf("expected cloudflaredExecResult, got %T", resultRaw)
	}
	if !result.Success || !result.DryRun || result.ExitCode != 0 {
		t.Fatalf("expected successful dry-run exec result, got %+v", result)
	}
	if result.BinaryPath != overridePath {
		t.Fatalf("expected cleaned validated override path %q, got %q", overridePath, result.BinaryPath)
	}
}

func TestCloudflaredWorkerStateExecuteExecRejectsRelativeBinaryOverride(t *testing.T) {
	t.Setenv(envCloudflaredDryRun, "1")
	t.Setenv(envCloudflaredBinary, "cloudflared-local")
	state := &cloudflaredWorkerState{}

	payload := json.RawMessage(`{"args":["version"]}`)
	resultRaw, err := state.execute(context.Background(), daemonruntime.CloudflaredConnectorOperationExec, payload)
	if err != nil {
		t.Fatalf("execute exec with relative override: %v", err)
	}

	result, ok := resultRaw.(cloudflaredExecResult)
	if !ok {
		t.Fatalf("expected cloudflaredExecResult, got %T", resultRaw)
	}
	if result.Success {
		t.Fatalf("expected relative override rejection, got %+v", result)
	}
	if !strings.Contains(strings.ToLower(result.Error), "absolute path") {
		t.Fatalf("expected absolute-path validation error, got %+v", result)
	}
}

func TestCloudflaredWorkerStateExecuteExecRejectsNonExecutableBinaryOverride(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("executable-bit checks are unix-only")
	}

	t.Setenv(envCloudflaredDryRun, "1")
	overridePath := createCloudflaredTestBinary(t, filepath.Join(t.TempDir(), "cloudflared"), false)
	t.Setenv(envCloudflaredBinary, overridePath)
	state := &cloudflaredWorkerState{}

	payload := json.RawMessage(`{"args":["version"]}`)
	resultRaw, err := state.execute(context.Background(), daemonruntime.CloudflaredConnectorOperationExec, payload)
	if err != nil {
		t.Fatalf("execute exec with non-executable override: %v", err)
	}

	result, ok := resultRaw.(cloudflaredExecResult)
	if !ok {
		t.Fatalf("expected cloudflaredExecResult, got %T", resultRaw)
	}
	if result.Success {
		t.Fatalf("expected non-executable override rejection, got %+v", result)
	}
	if !strings.Contains(strings.ToLower(result.Error), "must be executable") {
		t.Fatalf("expected executable validation error, got %+v", result)
	}
}

func TestCloudflaredWorkerStateExecuteExecMissingArgs(t *testing.T) {
	state := &cloudflaredWorkerState{}
	resultRaw, err := state.execute(context.Background(), daemonruntime.CloudflaredConnectorOperationExec, json.RawMessage(`{"args":[]}`))
	if err != nil {
		t.Fatalf("execute exec missing args: %v", err)
	}

	result, ok := resultRaw.(cloudflaredExecResult)
	if !ok {
		t.Fatalf("expected cloudflaredExecResult, got %T", resultRaw)
	}
	if result.Success || !strings.Contains(strings.ToLower(result.Error), "args are required") {
		t.Fatalf("expected args-required failure result, got %+v", result)
	}
}

func TestCloudflaredWorkerStateExecuteExecRejectsUnsupportedArgs(t *testing.T) {
	t.Setenv(envCloudflaredDryRun, "1")
	state := &cloudflaredWorkerState{}

	payload := json.RawMessage(`{"args":["tunnel","run","--token","secret"]}`)
	resultRaw, err := state.execute(context.Background(), daemonruntime.CloudflaredConnectorOperationExec, payload)
	if err != nil {
		t.Fatalf("execute unsupported exec args: %v", err)
	}

	result, ok := resultRaw.(cloudflaredExecResult)
	if !ok {
		t.Fatalf("expected cloudflaredExecResult, got %T", resultRaw)
	}
	if result.Success {
		t.Fatalf("expected unsupported args to fail, got %+v", result)
	}
	if !strings.Contains(strings.ToLower(result.Error), "unsupported cloudflared exec args") {
		t.Fatalf("expected unsupported-args error, got %+v", result)
	}
}

func TestCloudflaredWorkerStateExecuteExecAllowsTunnelListDryRun(t *testing.T) {
	t.Setenv(envCloudflaredDryRun, "1")
	state := &cloudflaredWorkerState{}

	payload := json.RawMessage(`{"args":["tunnel","list","--output","json"]}`)
	resultRaw, err := state.execute(context.Background(), daemonruntime.CloudflaredConnectorOperationExec, payload)
	if err != nil {
		t.Fatalf("execute tunnel list dry-run: %v", err)
	}

	result, ok := resultRaw.(cloudflaredExecResult)
	if !ok {
		t.Fatalf("expected cloudflaredExecResult, got %T", resultRaw)
	}
	if !result.Success || !result.DryRun || result.ExitCode != 0 {
		t.Fatalf("expected successful tunnel list dry-run result, got %+v", result)
	}
	if len(result.Args) != 4 || result.Args[0] != "tunnel" || result.Args[1] != "list" || result.Args[2] != "--output" || result.Args[3] != "json" {
		t.Fatalf("expected canonical tunnel list args, got %+v", result.Args)
	}
}

func TestAllowlistedCloudflaredExecArgs(t *testing.T) {
	testCases := []struct {
		name string
		args []string
		want []string
	}{
		{name: "version", args: []string{"version"}, want: []string{"version"}},
		{name: "tunnel-list", args: []string{"tunnel", "list"}, want: []string{"tunnel", "list"}},
		{name: "tunnel-list-output-json", args: []string{"tunnel", "list", "--output", "json"}, want: []string{"tunnel", "list", "--output", "json"}},
		{name: "tunnel-list-output-inline", args: []string{"tunnel", "list", "--output=json"}, want: []string{"tunnel", "list", "--output=json"}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, resolved, err := allowlistedCloudflaredExecArgs(testCase.args)
			if err != nil {
				t.Fatalf("allowlistedCloudflaredExecArgs returned error: %v", err)
			}
			if strings.Join(resolved, "|") != strings.Join(testCase.want, "|") {
				t.Fatalf("expected %v, got %v", testCase.want, resolved)
			}
		})
	}
}

func TestAllowlistedCloudflaredExecArgsRejectsUnsupported(t *testing.T) {
	_, _, err := allowlistedCloudflaredExecArgs([]string{"tunnel", "run", "--token", "secret"})
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "unsupported cloudflared exec args") {
		t.Fatalf("expected unsupported command error, got %v", err)
	}
}

func createCloudflaredTestBinary(t *testing.T, path string, executable bool) string {
	t.Helper()
	mode := os.FileMode(0o600)
	if executable && runtime.GOOS != "windows" {
		mode = 0o700
	}
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), mode); err != nil {
		t.Fatalf("write cloudflared test binary: %v", err)
	}
	return path
}
