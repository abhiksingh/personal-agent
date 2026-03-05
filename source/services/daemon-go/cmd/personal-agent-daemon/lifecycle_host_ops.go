package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"personalagent/runtime/internal/transport"
)

type daemonLifecycleHostOpsMode string

const (
	daemonLifecycleHostOpsModeUnsupported daemonLifecycleHostOpsMode = "unsupported"
	daemonLifecycleHostOpsModeDryRun      daemonLifecycleHostOpsMode = "dry-run"
	daemonLifecycleHostOpsModeApply       daemonLifecycleHostOpsMode = "apply"

	daemonLifecycleSetupActionInstall   = "install"
	daemonLifecycleSetupActionUninstall = "uninstall"
	daemonLifecycleSetupActionRepair    = "repair"

	defaultServiceLabel        = "com.personalagent.daemon"
	defaultDarwinDaemonAppName = "Personal Agent Daemon.app"
	defaultDarwinDaemonExec    = "personal-agent-daemon"
	defaultLinuxServiceName    = "personal-agent-daemon"
	defaultWindowsTaskName     = "PersonalAgentDaemon"
	defaultWindowsExecName     = "personal-agent-daemon.exe"
	lifecycleOpsUnsupportedMsg = "daemon lifecycle %s host operation is unsupported; restart daemon with --lifecycle-host-ops-mode=dry-run or --lifecycle-host-ops-mode=apply"

	envDaemonServiceExecutable = "PA_DAEMON_SERVICE_EXECUTABLE"
	envDaemonServiceAppPath    = "PA_DAEMON_SERVICE_APP_PATH"
)

type daemonLifecycleSetupHooks struct {
	install   func(context.Context, transport.DaemonLifecycleControlRequest) error
	uninstall func(context.Context, transport.DaemonLifecycleControlRequest) error
	repair    func(context.Context, transport.DaemonLifecycleControlRequest) error
}

type daemonHostOperationExecutor func(ctx context.Context, name string, args ...string) error

type daemonLifecycleHostOperator struct {
	mode            daemonLifecycleHostOpsMode
	platformOS      string
	homeDir         string
	executable      string
	listenerMode    string
	listenAddress   string
	authTokenFile   string
	authTokenScopes []string
	dbPath          string
	exec            daemonHostOperationExecutor
}

func buildDaemonLifecycleSetupHooks(
	config daemonRunConfig,
	executablePath string,
	resolvedDBPath string,
) (daemonLifecycleSetupHooks, error) {
	return buildDaemonLifecycleSetupHooksWithPlatform(
		config,
		executablePath,
		resolvedDBPath,
		runtime.GOOS,
		"",
		nil,
	)
}

func buildDaemonLifecycleSetupHooksWithPlatform(
	config daemonRunConfig,
	executablePath string,
	resolvedDBPath string,
	platformOS string,
	homeDir string,
	executor daemonHostOperationExecutor,
) (daemonLifecycleSetupHooks, error) {
	mode, err := normalizeDaemonLifecycleHostOpsMode(config.lifecycleHostOpsMode)
	if err != nil {
		return daemonLifecycleSetupHooks{}, err
	}
	trimmedExecutable := strings.TrimSpace(executablePath)
	if trimmedExecutable == "" {
		return daemonLifecycleSetupHooks{}, fmt.Errorf("daemon executable path is required for lifecycle host operations")
	}
	trimmedPlatform := strings.ToLower(strings.TrimSpace(platformOS))
	if trimmedPlatform == "" {
		trimmedPlatform = runtime.GOOS
	}

	resolvedHome := strings.TrimSpace(homeDir)
	if resolvedHome == "" {
		resolvedHome, err = os.UserHomeDir()
		if err != nil {
			return daemonLifecycleSetupHooks{}, fmt.Errorf("resolve user home directory: %w", err)
		}
	}

	dbPath := strings.TrimSpace(resolvedDBPath)
	if dbPath == "" {
		dbPath = strings.TrimSpace(config.dbPath)
	}
	if dbPath == "" {
		dbPath = filepath.Join(resolvedHome, ".config", "personal-agent", "runtime.db")
	}

	operator := daemonLifecycleHostOperator{
		mode:            mode,
		platformOS:      trimmedPlatform,
		homeDir:         resolvedHome,
		executable:      trimmedExecutable,
		listenerMode:    strings.TrimSpace(config.listenerMode),
		listenAddress:   strings.TrimSpace(config.listenAddress),
		authTokenFile:   strings.TrimSpace(config.authTokenFile),
		authTokenScopes: append([]string(nil), config.authTokenScopes...),
		dbPath:          strings.TrimSpace(dbPath),
		exec:            executor,
	}
	if operator.exec == nil {
		operator.exec = executeHostCommand
	}
	if operator.platformOS == "darwin" {
		operator.executable = resolveDarwinServiceExecutablePath(operator.executable, operator.homeDir)
	}

	return daemonLifecycleSetupHooks{
		install: func(ctx context.Context, _ transport.DaemonLifecycleControlRequest) error {
			return operator.execute(ctx, daemonLifecycleSetupActionInstall)
		},
		uninstall: func(ctx context.Context, _ transport.DaemonLifecycleControlRequest) error {
			return operator.execute(ctx, daemonLifecycleSetupActionUninstall)
		},
		repair: func(ctx context.Context, _ transport.DaemonLifecycleControlRequest) error {
			return operator.execute(ctx, daemonLifecycleSetupActionRepair)
		},
	}, nil
}

func normalizeDaemonLifecycleHostOpsMode(raw string) (daemonLifecycleHostOpsMode, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return daemonLifecycleHostOpsModeUnsupported, nil
	}
	switch daemonLifecycleHostOpsMode(value) {
	case daemonLifecycleHostOpsModeUnsupported, daemonLifecycleHostOpsModeDryRun, daemonLifecycleHostOpsModeApply:
		return daemonLifecycleHostOpsMode(value), nil
	default:
		return "", fmt.Errorf("unsupported --lifecycle-host-ops-mode %q", raw)
	}
}

func (o daemonLifecycleHostOperator) execute(ctx context.Context, action string) error {
	normalizedAction := strings.ToLower(strings.TrimSpace(action))
	switch normalizedAction {
	case daemonLifecycleSetupActionInstall, daemonLifecycleSetupActionUninstall, daemonLifecycleSetupActionRepair:
	default:
		return fmt.Errorf("unsupported lifecycle setup action %q", action)
	}

	if o.mode == daemonLifecycleHostOpsModeUnsupported {
		return fmt.Errorf(lifecycleOpsUnsupportedMsg, normalizedAction)
	}

	if o.mode == daemonLifecycleHostOpsModeApply &&
		(normalizedAction == daemonLifecycleSetupActionInstall || normalizedAction == daemonLifecycleSetupActionRepair) &&
		strings.TrimSpace(o.authTokenFile) == "" {
		return fmt.Errorf(
			"daemon lifecycle %s in apply mode requires daemon startup with --auth-token-file",
			normalizedAction,
		)
	}

	switch o.platformOS {
	case "darwin":
		return o.executeDarwin(ctx, normalizedAction)
	case "linux":
		return o.executeLinux(ctx, normalizedAction)
	case "windows":
		return o.executeWindows(ctx, normalizedAction)
	default:
		return fmt.Errorf("daemon lifecycle %s host operation is unsupported on platform %s", normalizedAction, o.platformOS)
	}
}

func (o daemonLifecycleHostOperator) executeDarwin(ctx context.Context, action string) error {
	plistPath := filepath.Join(o.homeDir, "Library", "LaunchAgents", defaultServiceLabel+".plist")
	if o.mode == daemonLifecycleHostOpsModeDryRun {
		return nil
	}
	switch action {
	case daemonLifecycleSetupActionInstall, daemonLifecycleSetupActionRepair:
		plistContent := o.renderDarwinLaunchAgentPlist()
		if err := os.MkdirAll(filepath.Dir(plistPath), 0o755); err != nil {
			return fmt.Errorf("prepare launchagent directory: %w", err)
		}
		if err := os.MkdirAll(filepath.Join(o.homeDir, "Library", "Logs", "personal-agent"), 0o755); err != nil {
			return fmt.Errorf("prepare launchagent log directory: %w", err)
		}
		if err := os.WriteFile(plistPath, []byte(plistContent), 0o644); err != nil {
			return fmt.Errorf("write launchagent plist: %w", err)
		}
		domain := "gui/" + strconv.Itoa(os.Getuid())
		_ = o.exec(ctx, "launchctl", "bootout", domain, plistPath)
		if err := o.exec(ctx, "launchctl", "bootstrap", domain, plistPath); err != nil {
			return fmt.Errorf("launchctl bootstrap: %w", err)
		}
		if err := o.exec(ctx, "launchctl", "enable", domain+"/"+defaultServiceLabel); err != nil {
			return fmt.Errorf("launchctl enable: %w", err)
		}
		return nil
	case daemonLifecycleSetupActionUninstall:
		domain := "gui/" + strconv.Itoa(os.Getuid())
		_ = o.exec(ctx, "launchctl", "bootout", domain, plistPath)
		if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove launchagent plist: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("unsupported darwin lifecycle action %q", action)
	}
}

func (o daemonLifecycleHostOperator) executeLinux(ctx context.Context, action string) error {
	unitPath := filepath.Join(o.homeDir, ".config", "systemd", "user", defaultLinuxServiceName+".service")
	if o.mode == daemonLifecycleHostOpsModeDryRun {
		return nil
	}
	switch action {
	case daemonLifecycleSetupActionInstall, daemonLifecycleSetupActionRepair:
		unitContent := o.renderLinuxSystemdUnit()
		if err := os.MkdirAll(filepath.Dir(unitPath), 0o755); err != nil {
			return fmt.Errorf("prepare systemd unit directory: %w", err)
		}
		if err := os.WriteFile(unitPath, []byte(unitContent), 0o644); err != nil {
			return fmt.Errorf("write systemd unit: %w", err)
		}
		if err := o.exec(ctx, "systemctl", "--user", "daemon-reload"); err != nil {
			return fmt.Errorf("systemctl daemon-reload: %w", err)
		}
		if err := o.exec(ctx, "systemctl", "--user", "enable", "--now", defaultLinuxServiceName+".service"); err != nil {
			return fmt.Errorf("systemctl enable --now: %w", err)
		}
		return nil
	case daemonLifecycleSetupActionUninstall:
		_ = o.exec(ctx, "systemctl", "--user", "disable", "--now", defaultLinuxServiceName+".service")
		if err := os.Remove(unitPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove systemd unit: %w", err)
		}
		if err := o.exec(ctx, "systemctl", "--user", "daemon-reload"); err != nil {
			return fmt.Errorf("systemctl daemon-reload: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("unsupported linux lifecycle action %q", action)
	}
}

func (o daemonLifecycleHostOperator) executeWindows(ctx context.Context, action string) error {
	if o.mode == daemonLifecycleHostOpsModeDryRun {
		return nil
	}
	switch action {
	case daemonLifecycleSetupActionInstall:
		return o.createWindowsScheduledTask(ctx)
	case daemonLifecycleSetupActionRepair:
		_ = o.exec(ctx, "schtasks", "/Delete", "/TN", defaultWindowsTaskName, "/F")
		return o.createWindowsScheduledTask(ctx)
	case daemonLifecycleSetupActionUninstall:
		_ = o.exec(ctx, "schtasks", "/Delete", "/TN", defaultWindowsTaskName, "/F")
		return nil
	default:
		return fmt.Errorf("unsupported windows lifecycle action %q", action)
	}
}

func (o daemonLifecycleHostOperator) createWindowsScheduledTask(ctx context.Context) error {
	if strings.TrimSpace(o.authTokenFile) == "" {
		return fmt.Errorf("windows scheduled-task install requires --auth-token-file")
	}
	command := o.windowsScheduledTaskCommand()
	if err := o.exec(
		ctx,
		"schtasks",
		"/Create",
		"/F",
		"/SC",
		"ONLOGON",
		"/RL",
		"LIMITED",
		"/TN",
		defaultWindowsTaskName,
		"/TR",
		command,
	); err != nil {
		return fmt.Errorf("create windows scheduled task: %w", err)
	}
	return nil
}

func (o daemonLifecycleHostOperator) renderDarwinLaunchAgentPlist() string {
	authTokenScopesArgs := ""
	if joinedScopes := o.joinedAuthTokenScopes(); joinedScopes != "" {
		authTokenScopesArgs = fmt.Sprintf("    <string>--auth-token-scopes</string>\n    <string>%s</string>\n", joinedScopes)
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>%s</string>
  <key>ProgramArguments</key>
  <array>
    <string>%s</string>
    <string>--listen-mode</string>
    <string>%s</string>
    <string>--listen-address</string>
    <string>%s</string>
    <string>--auth-token-file</string>
    <string>%s</string>
%s
    <string>--db</string>
    <string>%s</string>
  </array>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <true/>
  <key>StandardOutPath</key>
  <string>%s</string>
  <key>StandardErrorPath</key>
  <string>%s</string>
</dict>
</plist>
`,
		defaultServiceLabel,
		o.executable,
		o.listenerMode,
		o.listenAddress,
		o.authTokenFile,
		authTokenScopesArgs,
		o.dbPath,
		filepath.Join(o.homeDir, "Library", "Logs", "personal-agent", "daemon-service-macos.out.log"),
		filepath.Join(o.homeDir, "Library", "Logs", "personal-agent", "daemon-service-macos.err.log"),
	)
}

func (o daemonLifecycleHostOperator) renderLinuxSystemdUnit() string {
	authTokenScopesArg := ""
	if joinedScopes := o.joinedAuthTokenScopes(); joinedScopes != "" {
		authTokenScopesArg = " --auth-token-scopes " + joinedScopes
	}
	return fmt.Sprintf(`[Unit]
Description=Personal Agent Daemon
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=%s --listen-mode %s --listen-address %s --auth-token-file %s%s --db %s
Restart=always
RestartSec=2
Environment=HOME=%s
WorkingDirectory=%s

[Install]
WantedBy=default.target
`,
		o.executable,
		o.listenerMode,
		o.listenAddress,
		o.authTokenFile,
		authTokenScopesArg,
		o.dbPath,
		o.homeDir,
		o.homeDir,
	)
}

func (o daemonLifecycleHostOperator) windowsScheduledTaskCommand() string {
	executable := strings.TrimSpace(o.executable)
	if executable == "" {
		executable = defaultWindowsExecName
	}
	authTokenScopesArg := ""
	if joinedScopes := o.joinedAuthTokenScopes(); joinedScopes != "" {
		authTokenScopesArg = fmt.Sprintf(` --auth-token-scopes "%s"`, joinedScopes)
	}
	return fmt.Sprintf(
		`"%s" --listen-mode %s --listen-address %s --auth-token-file "%s"%s --db "%s"`,
		executable,
		o.listenerMode,
		o.listenAddress,
		o.authTokenFile,
		authTokenScopesArg,
		o.dbPath,
	)
}

func (o daemonLifecycleHostOperator) joinedAuthTokenScopes() string {
	scopes := make([]string, 0, len(o.authTokenScopes))
	for _, scope := range o.authTokenScopes {
		trimmed := strings.TrimSpace(scope)
		if trimmed == "" {
			continue
		}
		scopes = append(scopes, trimmed)
	}
	return strings.Join(scopes, ",")
}

func executeHostCommand(ctx context.Context, name string, args ...string) error {
	command := exec.CommandContext(ctx, name, args...)
	output, err := command.CombinedOutput()
	if err != nil {
		trimmed := strings.TrimSpace(string(output))
		if trimmed != "" {
			return fmt.Errorf("%s: %w: %s", name, err, trimmed)
		}
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

func resolveDarwinServiceExecutablePath(currentExecutable string, homeDir string) string {
	if override := strings.TrimSpace(os.Getenv(envDaemonServiceExecutable)); override != "" {
		return override
	}
	if overrideApp := strings.TrimSpace(os.Getenv(envDaemonServiceAppPath)); overrideApp != "" {
		candidate := darwinAppExecutablePath(overrideApp)
		if isExecutableFile(candidate) {
			return candidate
		}
	}

	trimmedCurrent := strings.TrimSpace(currentExecutable)
	if pathLooksLikeDarwinAppExecutable(trimmedCurrent) && isExecutableFile(trimmedCurrent) {
		return trimmedCurrent
	}

	candidates := make([]string, 0, 2)
	if strings.TrimSpace(homeDir) != "" {
		candidates = append(candidates, filepath.Join(homeDir, "Applications", defaultDarwinDaemonAppName))
	}
	candidates = append(candidates, filepath.Join("/Applications", defaultDarwinDaemonAppName))
	for _, appPath := range candidates {
		candidate := darwinAppExecutablePath(appPath)
		if isExecutableFile(candidate) {
			return candidate
		}
	}
	return trimmedCurrent
}

func darwinAppExecutablePath(appPath string) string {
	trimmed := strings.TrimSpace(appPath)
	if trimmed == "" {
		return ""
	}
	return filepath.Join(trimmed, "Contents", "MacOS", defaultDarwinDaemonExec)
}

func pathLooksLikeDarwinAppExecutable(path string) bool {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return false
	}
	cleaned := filepath.Clean(trimmed)
	marker := ".app" + string(os.PathSeparator) + "Contents" + string(os.PathSeparator) + "MacOS" + string(os.PathSeparator)
	return strings.Contains(cleaned, marker)
}

func isExecutableFile(path string) bool {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return false
	}
	info, err := os.Stat(trimmed)
	if err != nil || info.IsDir() {
		return false
	}
	return info.Mode()&0o111 != 0
}
