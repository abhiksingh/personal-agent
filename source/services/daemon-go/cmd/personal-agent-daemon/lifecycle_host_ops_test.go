package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"personalagent/runtime/internal/transport"
)

type lifecycleHostCommandCall struct {
	name string
	args []string
}

func TestNormalizeDaemonLifecycleHostOpsMode(t *testing.T) {
	mode, err := normalizeDaemonLifecycleHostOpsMode("")
	if err != nil {
		t.Fatalf("normalize empty mode: %v", err)
	}
	if mode != daemonLifecycleHostOpsModeUnsupported {
		t.Fatalf("expected unsupported default mode, got %q", mode)
	}

	mode, err = normalizeDaemonLifecycleHostOpsMode("DrY-RuN")
	if err != nil {
		t.Fatalf("normalize dry-run mode: %v", err)
	}
	if mode != daemonLifecycleHostOpsModeDryRun {
		t.Fatalf("expected dry-run mode, got %q", mode)
	}

	mode, err = normalizeDaemonLifecycleHostOpsMode("apply")
	if err != nil {
		t.Fatalf("normalize apply mode: %v", err)
	}
	if mode != daemonLifecycleHostOpsModeApply {
		t.Fatalf("expected apply mode, got %q", mode)
	}

	if _, err := normalizeDaemonLifecycleHostOpsMode("invalid"); err == nil {
		t.Fatalf("expected invalid mode error")
	}
}

func TestLifecycleHostHooksUnsupportedMode(t *testing.T) {
	hooks, err := buildDaemonLifecycleSetupHooksWithPlatform(
		daemonRunConfig{
			lifecycleHostOpsMode: string(daemonLifecycleHostOpsModeUnsupported),
		},
		"/tmp/personal-agent-daemon",
		filepath.Join(t.TempDir(), "runtime.db"),
		"darwin",
		t.TempDir(),
		nil,
	)
	if err != nil {
		t.Fatalf("build hooks: %v", err)
	}

	if err := hooks.install(context.Background(), transport.DaemonLifecycleControlRequest{Action: "install"}); err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("expected unsupported install error, got %v", err)
	}
	if err := hooks.uninstall(context.Background(), transport.DaemonLifecycleControlRequest{Action: "uninstall"}); err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("expected unsupported uninstall error, got %v", err)
	}
	if err := hooks.repair(context.Background(), transport.DaemonLifecycleControlRequest{Action: "repair"}); err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("expected unsupported repair error, got %v", err)
	}
}

func TestLifecycleHostHooksDryRunNoHostExecution(t *testing.T) {
	calls := []lifecycleHostCommandCall{}
	execFn := func(_ context.Context, name string, args ...string) error {
		calls = append(calls, lifecycleHostCommandCall{name: name, args: append([]string{}, args...)})
		return nil
	}
	hooks, err := buildDaemonLifecycleSetupHooksWithPlatform(
		daemonRunConfig{
			lifecycleHostOpsMode: string(daemonLifecycleHostOpsModeDryRun),
			listenerMode:         "tcp",
			listenAddress:        "127.0.0.1:7071",
			authTokenFile:        "",
		},
		"/tmp/personal-agent-daemon",
		filepath.Join(t.TempDir(), "runtime.db"),
		"linux",
		t.TempDir(),
		execFn,
	)
	if err != nil {
		t.Fatalf("build hooks: %v", err)
	}

	if err := hooks.install(context.Background(), transport.DaemonLifecycleControlRequest{Action: "install"}); err != nil {
		t.Fatalf("dry-run install: %v", err)
	}
	if err := hooks.repair(context.Background(), transport.DaemonLifecycleControlRequest{Action: "repair"}); err != nil {
		t.Fatalf("dry-run repair: %v", err)
	}
	if err := hooks.uninstall(context.Background(), transport.DaemonLifecycleControlRequest{Action: "uninstall"}); err != nil {
		t.Fatalf("dry-run uninstall: %v", err)
	}
	if len(calls) != 0 {
		t.Fatalf("expected no host command execution in dry-run mode, got %+v", calls)
	}
}

func TestLifecycleHostHooksApplyRequiresAuthTokenFile(t *testing.T) {
	hooks, err := buildDaemonLifecycleSetupHooksWithPlatform(
		daemonRunConfig{
			lifecycleHostOpsMode: string(daemonLifecycleHostOpsModeApply),
			listenerMode:         "tcp",
			listenAddress:        "127.0.0.1:7071",
		},
		"/tmp/personal-agent-daemon",
		filepath.Join(t.TempDir(), "runtime.db"),
		"darwin",
		t.TempDir(),
		nil,
	)
	if err != nil {
		t.Fatalf("build hooks: %v", err)
	}

	if err := hooks.install(context.Background(), transport.DaemonLifecycleControlRequest{Action: "install"}); err == nil || !strings.Contains(err.Error(), "--auth-token-file") {
		t.Fatalf("expected install apply mode auth-token-file error, got %v", err)
	}
	if err := hooks.repair(context.Background(), transport.DaemonLifecycleControlRequest{Action: "repair"}); err == nil || !strings.Contains(err.Error(), "--auth-token-file") {
		t.Fatalf("expected repair apply mode auth-token-file error, got %v", err)
	}
}

func TestLifecycleHostHooksApplyLinuxInstallWritesUnitAndRunsCommands(t *testing.T) {
	homeDir := t.TempDir()
	dbPath := filepath.Join(t.TempDir(), "runtime.db")
	calls := []lifecycleHostCommandCall{}
	execFn := func(_ context.Context, name string, args ...string) error {
		calls = append(calls, lifecycleHostCommandCall{name: name, args: append([]string{}, args...)})
		return nil
	}

	hooks, err := buildDaemonLifecycleSetupHooksWithPlatform(
		daemonRunConfig{
			lifecycleHostOpsMode: string(daemonLifecycleHostOpsModeApply),
			listenerMode:         "tcp",
			listenAddress:        "127.0.0.1:7071",
			authTokenFile:        "/tmp/personal-agent.control.token",
		},
		"/tmp/personal-agent-daemon",
		dbPath,
		"linux",
		homeDir,
		execFn,
	)
	if err != nil {
		t.Fatalf("build hooks: %v", err)
	}

	if err := hooks.install(context.Background(), transport.DaemonLifecycleControlRequest{Action: "install"}); err != nil {
		t.Fatalf("apply install: %v", err)
	}

	unitPath := filepath.Join(homeDir, ".config", "systemd", "user", defaultLinuxServiceName+".service")
	unitBytes, err := os.ReadFile(unitPath)
	if err != nil {
		t.Fatalf("read generated unit: %v", err)
	}
	unit := string(unitBytes)
	if !strings.Contains(unit, "/tmp/personal-agent.control.token") {
		t.Fatalf("expected auth-token-file in generated unit, got:\n%s", unit)
	}
	if !strings.Contains(unit, dbPath) {
		t.Fatalf("expected db path in generated unit, got:\n%s", unit)
	}
	if len(calls) != 2 {
		t.Fatalf("expected 2 systemctl calls, got %+v", calls)
	}
	if calls[0].name != "systemctl" || strings.Join(calls[0].args, " ") != "--user daemon-reload" {
		t.Fatalf("unexpected first systemctl call: %+v", calls[0])
	}
	if calls[1].name != "systemctl" || strings.Join(calls[1].args, " ") != "--user enable --now personal-agent-daemon.service" {
		t.Fatalf("unexpected second systemctl call: %+v", calls[1])
	}
}

func TestLifecycleHostHooksApplyLinuxRepairWritesUnitAndRunsCommands(t *testing.T) {
	homeDir := t.TempDir()
	dbPath := filepath.Join(t.TempDir(), "runtime.db")
	calls := []lifecycleHostCommandCall{}
	execFn := func(_ context.Context, name string, args ...string) error {
		calls = append(calls, lifecycleHostCommandCall{name: name, args: append([]string{}, args...)})
		return nil
	}

	hooks, err := buildDaemonLifecycleSetupHooksWithPlatform(
		daemonRunConfig{
			lifecycleHostOpsMode: string(daemonLifecycleHostOpsModeApply),
			listenerMode:         "tcp",
			listenAddress:        "127.0.0.1:7071",
			authTokenFile:        "/tmp/personal-agent.control.token",
		},
		"/tmp/personal-agent-daemon",
		dbPath,
		"linux",
		homeDir,
		execFn,
	)
	if err != nil {
		t.Fatalf("build hooks: %v", err)
	}

	if err := hooks.repair(context.Background(), transport.DaemonLifecycleControlRequest{Action: "repair"}); err != nil {
		t.Fatalf("apply repair: %v", err)
	}

	unitPath := filepath.Join(homeDir, ".config", "systemd", "user", defaultLinuxServiceName+".service")
	unitBytes, err := os.ReadFile(unitPath)
	if err != nil {
		t.Fatalf("read generated unit: %v", err)
	}
	unit := string(unitBytes)
	if !strings.Contains(unit, "/tmp/personal-agent.control.token") {
		t.Fatalf("expected auth-token-file in generated unit, got:\n%s", unit)
	}
	if !strings.Contains(unit, dbPath) {
		t.Fatalf("expected db path in generated unit, got:\n%s", unit)
	}
	if len(calls) != 2 {
		t.Fatalf("expected 2 systemctl calls, got %+v", calls)
	}
	if calls[0].name != "systemctl" || strings.Join(calls[0].args, " ") != "--user daemon-reload" {
		t.Fatalf("unexpected first systemctl call: %+v", calls[0])
	}
	if calls[1].name != "systemctl" || strings.Join(calls[1].args, " ") != "--user enable --now personal-agent-daemon.service" {
		t.Fatalf("unexpected second systemctl call: %+v", calls[1])
	}
}

func TestLifecycleHostHooksApplyLinuxInstallIncludesAuthTokenScopes(t *testing.T) {
	homeDir := t.TempDir()
	dbPath := filepath.Join(t.TempDir(), "runtime.db")
	execFn := func(_ context.Context, _ string, _ ...string) error { return nil }

	hooks, err := buildDaemonLifecycleSetupHooksWithPlatform(
		daemonRunConfig{
			lifecycleHostOpsMode: string(daemonLifecycleHostOpsModeApply),
			listenerMode:         "tcp",
			listenAddress:        "127.0.0.1:7071",
			authTokenFile:        "/tmp/personal-agent.control.token",
			authTokenScopes:      []string{"chat:read", "daemon:write"},
		},
		"/tmp/personal-agent-daemon",
		dbPath,
		"linux",
		homeDir,
		execFn,
	)
	if err != nil {
		t.Fatalf("build hooks: %v", err)
	}

	if err := hooks.install(context.Background(), transport.DaemonLifecycleControlRequest{Action: "install"}); err != nil {
		t.Fatalf("apply install: %v", err)
	}

	unitPath := filepath.Join(homeDir, ".config", "systemd", "user", defaultLinuxServiceName+".service")
	unitBytes, err := os.ReadFile(unitPath)
	if err != nil {
		t.Fatalf("read generated unit: %v", err)
	}
	unit := string(unitBytes)
	if !strings.Contains(unit, "--auth-token-scopes chat:read,daemon:write") {
		t.Fatalf("expected auth-token-scopes in generated unit, got:\\n%s", unit)
	}
}

func TestLifecycleHostHooksApplyLinuxUninstallRemovesUnitAndRunsCommands(t *testing.T) {
	homeDir := t.TempDir()
	dbPath := filepath.Join(t.TempDir(), "runtime.db")
	calls := []lifecycleHostCommandCall{}
	execFn := func(_ context.Context, name string, args ...string) error {
		calls = append(calls, lifecycleHostCommandCall{name: name, args: append([]string{}, args...)})
		return nil
	}

	hooks, err := buildDaemonLifecycleSetupHooksWithPlatform(
		daemonRunConfig{
			lifecycleHostOpsMode: string(daemonLifecycleHostOpsModeApply),
			listenerMode:         "tcp",
			listenAddress:        "127.0.0.1:7071",
			authTokenFile:        "/tmp/personal-agent.control.token",
		},
		"/tmp/personal-agent-daemon",
		dbPath,
		"linux",
		homeDir,
		execFn,
	)
	if err != nil {
		t.Fatalf("build hooks: %v", err)
	}

	// Seed unit file to ensure uninstall path removes existing artifact.
	unitPath := filepath.Join(homeDir, ".config", "systemd", "user", defaultLinuxServiceName+".service")
	if err := os.MkdirAll(filepath.Dir(unitPath), 0o755); err != nil {
		t.Fatalf("mkdir unit dir: %v", err)
	}
	if err := os.WriteFile(unitPath, []byte("[Unit]\nDescription=Personal Agent Daemon\n"), 0o644); err != nil {
		t.Fatalf("write seed unit: %v", err)
	}

	if err := hooks.uninstall(context.Background(), transport.DaemonLifecycleControlRequest{Action: "uninstall"}); err != nil {
		t.Fatalf("apply uninstall: %v", err)
	}
	if _, err := os.Stat(unitPath); !os.IsNotExist(err) {
		t.Fatalf("expected unit file removed, stat err=%v", err)
	}
	if len(calls) != 2 {
		t.Fatalf("expected 2 systemctl calls for uninstall, got %+v", calls)
	}
	if calls[0].name != "systemctl" || strings.Join(calls[0].args, " ") != "--user disable --now personal-agent-daemon.service" {
		t.Fatalf("unexpected first systemctl call: %+v", calls[0])
	}
	if calls[1].name != "systemctl" || strings.Join(calls[1].args, " ") != "--user daemon-reload" {
		t.Fatalf("unexpected second systemctl call: %+v", calls[1])
	}
}

func TestLifecycleHostOperatorDryRunAcrossPlatformsNoHostExecution(t *testing.T) {
	cases := []struct {
		name     string
		platform string
	}{
		{name: "darwin", platform: "darwin"},
		{name: "linux", platform: "linux"},
		{name: "windows", platform: "windows"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			calls := []lifecycleHostCommandCall{}
			operator := daemonLifecycleHostOperator{
				mode:          daemonLifecycleHostOpsModeDryRun,
				platformOS:    tc.platform,
				homeDir:       t.TempDir(),
				executable:    "/tmp/personal-agent-daemon",
				listenerMode:  "tcp",
				listenAddress: "127.0.0.1:7071",
				authTokenFile: "/tmp/personal-agent.control.token",
				dbPath:        filepath.Join(t.TempDir(), "runtime.db"),
				exec: func(_ context.Context, name string, args ...string) error {
					calls = append(calls, lifecycleHostCommandCall{name: name, args: append([]string{}, args...)})
					return nil
				},
			}

			for _, action := range []string{
				daemonLifecycleSetupActionInstall,
				daemonLifecycleSetupActionRepair,
				daemonLifecycleSetupActionUninstall,
			} {
				if err := operator.execute(context.Background(), action); err != nil {
					t.Fatalf("dry-run %s on %s: %v", action, tc.platform, err)
				}
			}
			if len(calls) != 0 {
				t.Fatalf("expected no host command calls in dry-run mode, got %+v", calls)
			}
		})
	}
}

func TestLifecycleHostOperatorUnsupportedPlatformFallback(t *testing.T) {
	operator := daemonLifecycleHostOperator{
		mode:          daemonLifecycleHostOpsModeApply,
		platformOS:    "plan9",
		homeDir:       t.TempDir(),
		executable:    "/tmp/personal-agent-daemon",
		listenerMode:  "tcp",
		listenAddress: "127.0.0.1:7071",
		authTokenFile: "/tmp/personal-agent.control.token",
		dbPath:        filepath.Join(t.TempDir(), "runtime.db"),
		exec: func(_ context.Context, _ string, _ ...string) error {
			return nil
		},
	}

	for _, action := range []string{
		daemonLifecycleSetupActionInstall,
		daemonLifecycleSetupActionRepair,
		daemonLifecycleSetupActionUninstall,
	} {
		err := operator.execute(context.Background(), action)
		if err == nil || !strings.Contains(err.Error(), "unsupported on platform plan9") {
			t.Fatalf("expected unsupported-platform fallback error for action %s, got %v", action, err)
		}
	}
}

func TestLifecycleHostOperatorApplyDarwinRunsLaunchctlSequences(t *testing.T) {
	homeDir := t.TempDir()
	authTokenFile := filepath.Join(homeDir, "daemon.token")
	if err := os.WriteFile(authTokenFile, []byte("token"), 0o600); err != nil {
		t.Fatalf("write auth token file: %v", err)
	}

	calls := []lifecycleHostCommandCall{}
	operator := daemonLifecycleHostOperator{
		mode:          daemonLifecycleHostOpsModeApply,
		platformOS:    "darwin",
		homeDir:       homeDir,
		executable:    "/tmp/personal-agent-daemon",
		listenerMode:  "tcp",
		listenAddress: "127.0.0.1:7071",
		authTokenFile: authTokenFile,
		dbPath:        filepath.Join(homeDir, "runtime.db"),
		exec: func(_ context.Context, name string, args ...string) error {
			calls = append(calls, lifecycleHostCommandCall{name: name, args: append([]string{}, args...)})
			return nil
		},
	}

	for _, action := range []string{
		daemonLifecycleSetupActionInstall,
		daemonLifecycleSetupActionRepair,
		daemonLifecycleSetupActionUninstall,
	} {
		if err := operator.execute(context.Background(), action); err != nil {
			t.Fatalf("darwin %s: %v", action, err)
		}
	}

	expectedCommands := []string{
		"launchctl bootout",
		"launchctl bootstrap",
		"launchctl enable",
		"launchctl bootout",
		"launchctl bootstrap",
		"launchctl enable",
		"launchctl bootout",
	}
	if len(calls) != len(expectedCommands) {
		t.Fatalf("expected %d launchctl calls, got %+v", len(expectedCommands), calls)
	}
	for idx, call := range calls {
		actual := strings.TrimSpace(call.name + " " + strings.Join(call.args, " "))
		if !strings.HasPrefix(actual, expectedCommands[idx]) {
			t.Fatalf("unexpected call[%d]=%q expected prefix %q", idx, actual, expectedCommands[idx])
		}
	}

	plistPath := filepath.Join(homeDir, "Library", "LaunchAgents", defaultServiceLabel+".plist")
	if _, err := os.Stat(plistPath); !os.IsNotExist(err) {
		t.Fatalf("expected launchagent plist removed after uninstall, stat err=%v", err)
	}
}

func TestLifecycleHostOperatorApplyWindowsRunsScheduledTaskSequences(t *testing.T) {
	calls := []lifecycleHostCommandCall{}
	operator := daemonLifecycleHostOperator{
		mode:          daemonLifecycleHostOpsModeApply,
		platformOS:    "windows",
		homeDir:       "C:\\Users\\tester",
		executable:    "C:\\Apps\\personal-agent-daemon.exe",
		listenerMode:  "tcp",
		listenAddress: "127.0.0.1:7071",
		authTokenFile: "C:\\Users\\tester\\daemon.token",
		dbPath:        "C:\\Users\\tester\\runtime.db",
		exec: func(_ context.Context, name string, args ...string) error {
			calls = append(calls, lifecycleHostCommandCall{name: name, args: append([]string{}, args...)})
			return nil
		},
	}

	for _, action := range []string{
		daemonLifecycleSetupActionInstall,
		daemonLifecycleSetupActionRepair,
		daemonLifecycleSetupActionUninstall,
	} {
		if err := operator.execute(context.Background(), action); err != nil {
			t.Fatalf("windows %s: %v", action, err)
		}
	}

	if len(calls) != 4 {
		t.Fatalf("expected 4 schtasks calls, got %+v", calls)
	}
	expectedPrefixes := []string{
		"schtasks /Create",
		"schtasks /Delete",
		"schtasks /Create",
		"schtasks /Delete",
	}
	for idx, call := range calls {
		actual := strings.TrimSpace(call.name + " " + strings.Join(call.args, " "))
		if !strings.HasPrefix(actual, expectedPrefixes[idx]) {
			t.Fatalf("unexpected schtasks call[%d]=%q expected prefix %q", idx, actual, expectedPrefixes[idx])
		}
	}
	createCall := strings.TrimSpace(calls[0].name + " " + strings.Join(calls[0].args, " "))
	for _, needle := range []string{"--listen-mode tcp", "--listen-address 127.0.0.1:7071", "--auth-token-file", "--db"} {
		if !strings.Contains(createCall, needle) {
			t.Fatalf("expected windows create command to include %q, got %q", needle, createCall)
		}
	}
}

func TestLifecycleHostOperatorRenderCommandsIncludeRequiredDaemonArgs(t *testing.T) {
	operator := daemonLifecycleHostOperator{
		listenerMode:  "tcp",
		listenAddress: "127.0.0.1:7071",
		authTokenFile: "/tmp/daemon.token",
		dbPath:        "/tmp/runtime.db",
		executable:    "/tmp/personal-agent-daemon",
		homeDir:       "/tmp/home",
	}

	rendered := map[string]string{
		"darwin":  operator.renderDarwinLaunchAgentPlist(),
		"linux":   operator.renderLinuxSystemdUnit(),
		"windows": operator.windowsScheduledTaskCommand(),
	}
	for platform, command := range rendered {
		for _, needle := range []string{"--listen-mode", "--listen-address", "--auth-token-file", "--db"} {
			if !strings.Contains(command, needle) {
				t.Fatalf("expected %s rendered command to include %q, got %s", platform, needle, command)
			}
		}
	}
}

func TestLifecycleHostOperatorRenderCommandsIncludeAuthTokenScopesWhenConfigured(t *testing.T) {
	operator := daemonLifecycleHostOperator{
		listenerMode:    "tcp",
		listenAddress:   "127.0.0.1:7071",
		authTokenFile:   "/tmp/daemon.token",
		authTokenScopes: []string{"chat:read", "daemon:write"},
		dbPath:          "/tmp/runtime.db",
		executable:      "/tmp/personal-agent-daemon",
		homeDir:         "/tmp/home",
	}

	rendered := map[string]string{
		"darwin":  operator.renderDarwinLaunchAgentPlist(),
		"linux":   operator.renderLinuxSystemdUnit(),
		"windows": operator.windowsScheduledTaskCommand(),
	}
	for platform, command := range rendered {
		if !strings.Contains(command, "--auth-token-scopes") {
			t.Fatalf("expected %s rendered command to include --auth-token-scopes, got %s", platform, command)
		}
		if !strings.Contains(command, "chat:read,daemon:write") {
			t.Fatalf("expected %s rendered command to include joined scopes, got %s", platform, command)
		}
	}
}

func TestResolveDarwinServiceExecutablePathPrefersPackagedApp(t *testing.T) {
	t.Setenv(envDaemonServiceExecutable, "")
	t.Setenv(envDaemonServiceAppPath, "")

	homeDir := t.TempDir()
	packagedExecutable := filepath.Join(homeDir, "Applications", defaultDarwinDaemonAppName, "Contents", "MacOS", defaultDarwinDaemonExec)
	if err := os.MkdirAll(filepath.Dir(packagedExecutable), 0o755); err != nil {
		t.Fatalf("mkdir packaged daemon executable dir: %v", err)
	}
	if err := os.WriteFile(packagedExecutable, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write packaged daemon executable: %v", err)
	}

	resolved := resolveDarwinServiceExecutablePath("/tmp/go-build12345/b001/exe/personal-agent-daemon", homeDir)
	if resolved != packagedExecutable {
		t.Fatalf("expected packaged executable path %q, got %q", packagedExecutable, resolved)
	}
}

func TestResolveDarwinServiceExecutablePathUsesExplicitExecutableOverride(t *testing.T) {
	homeDir := t.TempDir()
	overridePath := filepath.Join(homeDir, "custom-bin", "personal-agent-daemon")
	if err := os.MkdirAll(filepath.Dir(overridePath), 0o755); err != nil {
		t.Fatalf("mkdir override dir: %v", err)
	}
	if err := os.WriteFile(overridePath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write override executable: %v", err)
	}
	t.Setenv(envDaemonServiceExecutable, overridePath)
	t.Setenv(envDaemonServiceAppPath, "")

	resolved := resolveDarwinServiceExecutablePath("/tmp/current-daemon", homeDir)
	if resolved != overridePath {
		t.Fatalf("expected executable override path %q, got %q", overridePath, resolved)
	}
}

func TestResolveDarwinServiceExecutablePathUsesAppOverride(t *testing.T) {
	t.Setenv(envDaemonServiceExecutable, "")

	homeDir := t.TempDir()
	overrideAppPath := filepath.Join(homeDir, "Override", defaultDarwinDaemonAppName)
	overrideExecutable := filepath.Join(overrideAppPath, "Contents", "MacOS", defaultDarwinDaemonExec)
	if err := os.MkdirAll(filepath.Dir(overrideExecutable), 0o755); err != nil {
		t.Fatalf("mkdir override app executable dir: %v", err)
	}
	if err := os.WriteFile(overrideExecutable, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write override app executable: %v", err)
	}
	t.Setenv(envDaemonServiceAppPath, overrideAppPath)

	resolved := resolveDarwinServiceExecutablePath("/tmp/current-daemon", homeDir)
	if resolved != overrideExecutable {
		t.Fatalf("expected app override executable path %q, got %q", overrideExecutable, resolved)
	}
}
