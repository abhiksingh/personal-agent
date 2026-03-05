package daemonruntime

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"personalagent/runtime/internal/securestore"
	shared "personalagent/runtime/internal/shared/contracts"
	"personalagent/runtime/internal/transport"
)

type lifecycleTestPluginSupervisor struct {
	workers      []PluginWorkerStatus
	restartCalls []string
	restartErr   error
}

func (s *lifecycleTestPluginSupervisor) SetHooks(PluginLifecycleHooks) {}
func (s *lifecycleTestPluginSupervisor) RegisterWorker(PluginWorkerSpec) error {
	return nil
}
func (s *lifecycleTestPluginSupervisor) ListWorkers() []PluginWorkerStatus {
	out := make([]PluginWorkerStatus, len(s.workers))
	copy(out, s.workers)
	return out
}
func (s *lifecycleTestPluginSupervisor) WorkerStatus(pluginID string) (PluginWorkerStatus, bool) {
	for _, worker := range s.workers {
		if worker.PluginID == pluginID {
			return worker, true
		}
	}
	return PluginWorkerStatus{}, false
}
func (s *lifecycleTestPluginSupervisor) RestartWorker(_ context.Context, pluginID string) error {
	s.restartCalls = append(s.restartCalls, pluginID)
	return s.restartErr
}
func (s *lifecycleTestPluginSupervisor) StopWorker(context.Context, string) error { return nil }
func (s *lifecycleTestPluginSupervisor) Start(context.Context) error              { return nil }
func (s *lifecycleTestPluginSupervisor) Stop(context.Context) error               { return nil }

func newLifecycleTestContainer(t *testing.T, workers []PluginWorkerStatus) *ServiceContainer {
	t.Helper()
	container, _ := newLifecycleTestContainerWithSupervisor(t, workers)
	return container
}

func newLifecycleTestContainerWithSupervisor(t *testing.T, workers []PluginWorkerStatus) (*ServiceContainer, *lifecycleTestPluginSupervisor) {
	t.Helper()
	manager, err := securestore.NewManager("personal-agent", "memory", securestore.NewMemoryBackend())
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	supervisor := &lifecycleTestPluginSupervisor{workers: workers}
	container, err := NewServiceContainer(context.Background(), ServiceContainerConfig{
		DBPath: filepath.Join(t.TempDir(), "runtime.db"),
		SecretManagerFactory: func() (*securestore.Manager, error) {
			return manager, nil
		},
		PluginSupervisor: supervisor,
	})
	if err != nil {
		t.Fatalf("new service container: %v", err)
	}
	t.Cleanup(func() {
		_ = container.Close(context.Background())
	})
	return container, supervisor
}

func TestDaemonLifecycleStatusReady(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	executablePath := filepath.Join(t.TempDir(), "personal-agent-daemon")
	if err := osWriteFileExecutable(executablePath); err != nil {
		t.Fatalf("create executable: %v", err)
	}

	stopCalls := 0
	restartCalls := 0
	service, err := NewDaemonLifecycleService(DaemonLifecycleServiceConfig{
		Container:         container,
		RuntimeMode:       "tcp",
		ConfiguredAddress: "127.0.0.1:7071",
		ExecutablePath:    executablePath,
		AuthToken:         "daemon-auth-token",
		AuthTokenSource:   daemonControlAuthSourceFile,
		RequestStop: func(context.Context) error {
			stopCalls++
			return nil
		},
		RequestRestart: func(context.Context) error {
			restartCalls++
			return nil
		},
	})
	if err != nil {
		t.Fatalf("new lifecycle service: %v", err)
	}
	service.SetBoundAddress("127.0.0.1:18080")

	status, err := service.DaemonLifecycleStatus(context.Background())
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.SetupState != setupStateReady {
		t.Fatalf("expected setup_state=%s, got %s", setupStateReady, status.SetupState)
	}
	if status.InstallState != installStateInstalled || status.NeedsInstall {
		t.Fatalf("expected installed executable, got state=%s needs_install=%v", status.InstallState, status.NeedsInstall)
	}
	if !status.DatabaseReady {
		t.Fatalf("expected database ready")
	}
	if status.BoundAddress != "127.0.0.1:18080" {
		t.Fatalf("expected bound address, got %s", status.BoundAddress)
	}
	if status.WorkerSummary.Total != 0 {
		t.Fatalf("expected worker total 0, got %d", status.WorkerSummary.Total)
	}
	if !status.Controls.Start || !status.Controls.Stop || !status.Controls.Restart {
		t.Fatalf("expected start/stop/restart controls enabled, got %+v", status.Controls)
	}
	if !status.Controls.Install || !status.Controls.Uninstall || !status.Controls.Repair {
		t.Fatalf("expected install/uninstall/repair controls enabled, got %+v", status.Controls)
	}
	if status.ControlOperation.State != lifecycleOperationStateIdle {
		t.Fatalf("expected idle control operation state, got %+v", status.ControlOperation)
	}
	if status.HealthClassification.OverallState != lifecycleHealthOverallReady {
		t.Fatalf("expected lifecycle overall health=%s, got %+v", lifecycleHealthOverallReady, status.HealthClassification)
	}
	if status.HealthClassification.CoreRuntimeState != lifecycleHealthCoreReady {
		t.Fatalf("expected core runtime health=%s, got %+v", lifecycleHealthCoreReady, status.HealthClassification)
	}
	if status.HealthClassification.PluginRuntimeState != lifecycleHealthPluginsHealthy {
		t.Fatalf("expected plugin runtime health=%s, got %+v", lifecycleHealthPluginsHealthy, status.HealthClassification)
	}
	if status.HealthClassification.Blocking {
		t.Fatalf("expected non-blocking lifecycle health, got %+v", status.HealthClassification)
	}
	if status.ControlAuth.State != daemonControlAuthStateConfigured {
		t.Fatalf("expected control_auth.state=%s, got %+v", daemonControlAuthStateConfigured, status.ControlAuth)
	}
	if status.ControlAuth.Source != daemonControlAuthSourceFile {
		t.Fatalf("expected control_auth.source=%s, got %+v", daemonControlAuthSourceFile, status.ControlAuth)
	}
	if len(status.ControlAuth.RemediationHints) != 0 {
		t.Fatalf("expected no remediation hints for configured auth, got %+v", status.ControlAuth.RemediationHints)
	}

	control, err := service.DaemonLifecycleControl(context.Background(), transport.DaemonLifecycleControlRequest{Action: "start"})
	if err != nil {
		t.Fatalf("control start: %v", err)
	}
	if !control.Accepted || !control.Idempotent {
		t.Fatalf("expected idempotent accepted start, got %+v", control)
	}
	if control.OperationState != lifecycleOperationStateSucceeded {
		t.Fatalf("expected start operation_state=%s, got %+v", lifecycleOperationStateSucceeded, control)
	}
	if stopCalls != 0 || restartCalls != 0 {
		t.Fatalf("expected no stop/restart calls for start no-op, got stop=%d restart=%d", stopCalls, restartCalls)
	}
}

func TestDaemonControlAuthStateClassification(t *testing.T) {
	tests := []struct {
		name             string
		token            string
		source           string
		expectedState    string
		expectedSource   string
		expectHintsCount int
	}{
		{
			name:             "configured via flag",
			token:            "custom-token",
			source:           daemonControlAuthSourceFlag,
			expectedState:    daemonControlAuthStateConfigured,
			expectedSource:   daemonControlAuthSourceFlag,
			expectHintsCount: 0,
		},
		{
			name:             "missing token",
			token:            "",
			source:           daemonControlAuthSourceFile,
			expectedState:    daemonControlAuthStateMissing,
			expectedSource:   daemonControlAuthSourceFile,
			expectHintsCount: 1,
		},
		{
			name:             "unknown source fallback",
			token:            "custom-token",
			source:           "unsupported",
			expectedState:    daemonControlAuthStateConfigured,
			expectedSource:   daemonControlAuthSourceUnknown,
			expectHintsCount: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := daemonControlAuthState(test.token, test.source)
			if state.State != test.expectedState {
				t.Fatalf("expected auth state %q, got %+v", test.expectedState, state)
			}
			if state.Source != test.expectedSource {
				t.Fatalf("expected auth source %q, got %+v", test.expectedSource, state)
			}
			if len(state.RemediationHints) < test.expectHintsCount {
				t.Fatalf(
					"expected remediation hint count >= %d, got %+v",
					test.expectHintsCount,
					state.RemediationHints,
				)
			}
		})
	}
}

func TestDaemonLifecycleHealthClassificationMatrix(t *testing.T) {
	executablePath := filepath.Join(t.TempDir(), "personal-agent-daemon")
	if err := osWriteFileExecutable(executablePath); err != nil {
		t.Fatalf("create executable: %v", err)
	}

	tests := []struct {
		name                     string
		executablePath           string
		workers                  []PluginWorkerStatus
		requestStop              func(context.Context) error
		setStoppedLifecycleState bool
		breakDatabase            bool
		expectedOverall          string
		expectedCore             string
		expectedPlugin           string
		expectedBlocking         bool
	}{
		{
			name:             "ready core and healthy plugins",
			executablePath:   executablePath,
			expectedOverall:  lifecycleHealthOverallReady,
			expectedCore:     lifecycleHealthCoreReady,
			expectedPlugin:   lifecycleHealthPluginsHealthy,
			expectedBlocking: false,
		},
		{
			name:             "missing install blocks core runtime",
			executablePath:   filepath.Join(t.TempDir(), "missing-daemon"),
			expectedOverall:  lifecycleHealthOverallBlocked,
			expectedCore:     lifecycleHealthCoreInstallRequired,
			expectedPlugin:   lifecycleHealthPluginsHealthy,
			expectedBlocking: true,
		},
		{
			name:             "database unavailable blocks core runtime",
			executablePath:   executablePath,
			breakDatabase:    true,
			expectedOverall:  lifecycleHealthOverallBlocked,
			expectedCore:     lifecycleHealthCoreDatabaseUnavailable,
			expectedPlugin:   lifecycleHealthPluginsHealthy,
			expectedBlocking: true,
		},
		{
			name:           "plugin worker failures are degraded not core-blocking",
			executablePath: executablePath,
			workers: []PluginWorkerStatus{
				{
					PluginID: "messages.daemon",
					Kind:     shared.AdapterKindChannel,
					State:    PluginWorkerStateFailed,
				},
			},
			expectedOverall:  lifecycleHealthOverallDegraded,
			expectedCore:     lifecycleHealthCoreReady,
			expectedPlugin:   lifecycleHealthPluginsDegraded,
			expectedBlocking: false,
		},
		{
			name:                     "stop requested lifecycle blocks control plane readiness",
			executablePath:           executablePath,
			requestStop:              func(context.Context) error { return nil },
			setStoppedLifecycleState: true,
			expectedOverall:          lifecycleHealthOverallBlocked,
			expectedCore:             lifecycleHealthCoreControlPlaneDown,
			expectedPlugin:           lifecycleHealthPluginsHealthy,
			expectedBlocking:         true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			container := newLifecycleTestContainer(t, tc.workers)
			if tc.breakDatabase && container.DB != nil {
				if err := container.DB.Close(); err != nil {
					t.Fatalf("close database handle: %v", err)
				}
			}

			service, err := NewDaemonLifecycleService(DaemonLifecycleServiceConfig{
				Container:      container,
				ExecutablePath: tc.executablePath,
				RequestStop:    tc.requestStop,
			})
			if err != nil {
				t.Fatalf("new lifecycle service: %v", err)
			}
			if tc.setStoppedLifecycleState {
				_, err := service.DaemonLifecycleControl(context.Background(), transport.DaemonLifecycleControlRequest{
					Action: "stop",
					Reason: "test stop-requested classification",
				})
				if err != nil {
					t.Fatalf("request stop transition: %v", err)
				}
			}

			status, err := service.DaemonLifecycleStatus(context.Background())
			if err != nil {
				t.Fatalf("status: %v", err)
			}
			if status.HealthClassification.OverallState != tc.expectedOverall {
				t.Fatalf("expected overall health=%s, got %+v", tc.expectedOverall, status.HealthClassification)
			}
			if status.HealthClassification.CoreRuntimeState != tc.expectedCore {
				t.Fatalf("expected core health=%s, got %+v", tc.expectedCore, status.HealthClassification)
			}
			if status.HealthClassification.PluginRuntimeState != tc.expectedPlugin {
				t.Fatalf("expected plugin health=%s, got %+v", tc.expectedPlugin, status.HealthClassification)
			}
			if status.HealthClassification.Blocking != tc.expectedBlocking {
				t.Fatalf("expected blocking=%v, got %+v", tc.expectedBlocking, status.HealthClassification)
			}
		})
	}
}

func TestDaemonLifecycleStatusTreatsBusySingleWriterConnectionAsReady(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	executablePath := filepath.Join(t.TempDir(), "personal-agent-daemon")
	if err := osWriteFileExecutable(executablePath); err != nil {
		t.Fatalf("create executable: %v", err)
	}

	now := time.Date(2026, time.March, 4, 12, 0, 0, 0, time.UTC)
	service, err := NewDaemonLifecycleService(DaemonLifecycleServiceConfig{
		Container:      container,
		ExecutablePath: executablePath,
		Now: func() time.Time {
			return now
		},
	})
	if err != nil {
		t.Fatalf("new lifecycle service: %v", err)
	}

	tx, err := container.DB.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	withinBusyWindow, err := service.DaemonLifecycleStatus(context.Background())
	if err != nil {
		t.Fatalf("status with busy single-writer connection: %v", err)
	}
	if !withinBusyWindow.DatabaseReady {
		t.Fatalf("expected database_ready=true while single-writer connection is busy, got %+v", withinBusyWindow)
	}
	if withinBusyWindow.NeedsRepair {
		t.Fatalf("expected needs_repair=false while single-writer connection is busy, got %+v", withinBusyWindow)
	}
	if withinBusyWindow.SetupState != setupStateReady {
		t.Fatalf("expected setup_state=%s while single-writer connection is busy, got %+v", setupStateReady, withinBusyWindow)
	}

	now = now.Add(daemonDatabaseProbeGraceWindow + time.Second)
	afterLongBusyDuration, err := service.DaemonLifecycleStatus(context.Background())
	if err != nil {
		t.Fatalf("status after long busy duration: %v", err)
	}
	if !afterLongBusyDuration.DatabaseReady {
		t.Fatalf("expected database_ready=true after long busy duration on single-writer connection, got %+v", afterLongBusyDuration)
	}
	if afterLongBusyDuration.NeedsRepair {
		t.Fatalf("expected needs_repair=false after long busy duration on single-writer connection, got %+v", afterLongBusyDuration)
	}
	if afterLongBusyDuration.SetupState != setupStateReady {
		t.Fatalf("expected setup_state=%s after long busy duration on single-writer connection, got %+v", setupStateReady, afterLongBusyDuration)
	}
	if afterLongBusyDuration.HealthClassification.CoreRuntimeState != lifecycleHealthCoreReady {
		t.Fatalf("expected core runtime health=%s after long busy duration on single-writer connection, got %+v", lifecycleHealthCoreReady, afterLongBusyDuration.HealthClassification)
	}
}

func TestDaemonLifecycleControlRestartAndIdempotency(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	executablePath := filepath.Join(t.TempDir(), "personal-agent-daemon")
	if err := osWriteFileExecutable(executablePath); err != nil {
		t.Fatalf("create executable: %v", err)
	}

	restartCalls := 0
	service, err := NewDaemonLifecycleService(DaemonLifecycleServiceConfig{
		Container:      container,
		ExecutablePath: executablePath,
		RequestRestart: func(context.Context) error {
			restartCalls++
			return nil
		},
		RequestStop: func(context.Context) error {
			return nil
		},
	})
	if err != nil {
		t.Fatalf("new lifecycle service: %v", err)
	}

	first, err := service.DaemonLifecycleControl(context.Background(), transport.DaemonLifecycleControlRequest{Action: "restart"})
	if err != nil {
		t.Fatalf("first restart: %v", err)
	}
	if !first.Accepted || first.Idempotent || first.LifecycleState != lifecycleStateRestartRequested {
		t.Fatalf("unexpected first restart response: %+v", first)
	}
	if restartCalls != 1 {
		t.Fatalf("expected restart hook called once, got %d", restartCalls)
	}

	second, err := service.DaemonLifecycleControl(context.Background(), transport.DaemonLifecycleControlRequest{Action: "restart"})
	if err != nil {
		t.Fatalf("second restart: %v", err)
	}
	if !second.Accepted || !second.Idempotent || second.LifecycleState != lifecycleStateRestartRequested {
		t.Fatalf("unexpected second restart response: %+v", second)
	}
	if restartCalls != 1 {
		t.Fatalf("expected restart hook not called again, got %d", restartCalls)
	}
}

func TestDaemonLifecycleControlStopAndIdempotency(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	executablePath := filepath.Join(t.TempDir(), "personal-agent-daemon")
	if err := osWriteFileExecutable(executablePath); err != nil {
		t.Fatalf("create executable: %v", err)
	}

	stopCalls := 0
	service, err := NewDaemonLifecycleService(DaemonLifecycleServiceConfig{
		Container:      container,
		ExecutablePath: executablePath,
		RequestStop: func(context.Context) error {
			stopCalls++
			return nil
		},
		RequestRestart: func(context.Context) error {
			return nil
		},
	})
	if err != nil {
		t.Fatalf("new lifecycle service: %v", err)
	}

	first, err := service.DaemonLifecycleControl(context.Background(), transport.DaemonLifecycleControlRequest{Action: "stop"})
	if err != nil {
		t.Fatalf("first stop: %v", err)
	}
	if !first.Accepted || first.Idempotent || first.LifecycleState != lifecycleStateStopRequested {
		t.Fatalf("unexpected first stop response: %+v", first)
	}
	if stopCalls != 1 {
		t.Fatalf("expected stop hook called once, got %d", stopCalls)
	}

	second, err := service.DaemonLifecycleControl(context.Background(), transport.DaemonLifecycleControlRequest{Action: "stop"})
	if err != nil {
		t.Fatalf("second stop: %v", err)
	}
	if !second.Accepted || !second.Idempotent || second.LifecycleState != lifecycleStateStopRequested {
		t.Fatalf("unexpected second stop response: %+v", second)
	}
	if stopCalls != 1 {
		t.Fatalf("expected stop hook not called again, got %d", stopCalls)
	}
}

func TestDaemonLifecycleControlInstallTracksInProgressAndSuccess(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	executablePath := filepath.Join(t.TempDir(), "personal-agent-daemon")
	if err := osWriteFileExecutable(executablePath); err != nil {
		t.Fatalf("create executable: %v", err)
	}

	started := make(chan struct{}, 1)
	release := make(chan struct{})
	service, err := NewDaemonLifecycleService(DaemonLifecycleServiceConfig{
		Container:      container,
		ExecutablePath: executablePath,
		RequestInstall: func(ctx context.Context, _ transport.DaemonLifecycleControlRequest) error {
			select {
			case started <- struct{}{}:
			default:
			}
			select {
			case <-release:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	})
	if err != nil {
		t.Fatalf("new lifecycle service: %v", err)
	}

	first, err := service.DaemonLifecycleControl(context.Background(), transport.DaemonLifecycleControlRequest{
		Action: "install",
		Reason: "test install operation",
	})
	if err != nil {
		t.Fatalf("first install: %v", err)
	}
	if !first.Accepted || first.Idempotent || first.OperationState != lifecycleOperationStateInProgress {
		t.Fatalf("unexpected first install response: %+v", first)
	}

	second, err := service.DaemonLifecycleControl(context.Background(), transport.DaemonLifecycleControlRequest{
		Action: "install",
		Reason: "test install operation repeat",
	})
	if err != nil {
		t.Fatalf("second install: %v", err)
	}
	if !second.Accepted || !second.Idempotent || second.OperationState != lifecycleOperationStateInProgress {
		t.Fatalf("unexpected second install response: %+v", second)
	}

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for install hook to start")
	}

	status := waitForLifecycleControlOperationState(t, service, lifecycleOperationStateInProgress)
	if status.ControlOperation.Action != "install" {
		t.Fatalf("expected in-progress install action, got %+v", status.ControlOperation)
	}

	close(release)

	status = waitForLifecycleControlOperationState(t, service, lifecycleOperationStateSucceeded)
	if status.ControlOperation.Action != "install" || strings.TrimSpace(status.ControlOperation.CompletedAt) == "" {
		t.Fatalf("expected succeeded install operation with completion timestamp, got %+v", status.ControlOperation)
	}
}

func TestDaemonLifecycleControlUninstallTracksInProgressAndSuccess(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	executablePath := filepath.Join(t.TempDir(), "personal-agent-daemon")
	if err := osWriteFileExecutable(executablePath); err != nil {
		t.Fatalf("create executable: %v", err)
	}

	started := make(chan struct{}, 1)
	release := make(chan struct{})
	service, err := NewDaemonLifecycleService(DaemonLifecycleServiceConfig{
		Container:      container,
		ExecutablePath: executablePath,
		RequestUninstall: func(ctx context.Context, _ transport.DaemonLifecycleControlRequest) error {
			select {
			case started <- struct{}{}:
			default:
			}
			select {
			case <-release:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	})
	if err != nil {
		t.Fatalf("new lifecycle service: %v", err)
	}

	first, err := service.DaemonLifecycleControl(context.Background(), transport.DaemonLifecycleControlRequest{
		Action: "uninstall",
		Reason: "test uninstall operation",
	})
	if err != nil {
		t.Fatalf("first uninstall: %v", err)
	}
	if !first.Accepted || first.Idempotent || first.OperationState != lifecycleOperationStateInProgress {
		t.Fatalf("unexpected first uninstall response: %+v", first)
	}

	second, err := service.DaemonLifecycleControl(context.Background(), transport.DaemonLifecycleControlRequest{
		Action: "uninstall",
		Reason: "test uninstall operation repeat",
	})
	if err != nil {
		t.Fatalf("second uninstall: %v", err)
	}
	if !second.Accepted || !second.Idempotent || second.OperationState != lifecycleOperationStateInProgress {
		t.Fatalf("unexpected second uninstall response: %+v", second)
	}

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for uninstall hook to start")
	}

	status := waitForLifecycleControlOperationState(t, service, lifecycleOperationStateInProgress)
	if status.ControlOperation.Action != "uninstall" {
		t.Fatalf("expected in-progress uninstall action, got %+v", status.ControlOperation)
	}

	close(release)

	status = waitForLifecycleControlOperationState(t, service, lifecycleOperationStateSucceeded)
	if status.ControlOperation.Action != "uninstall" || strings.TrimSpace(status.ControlOperation.CompletedAt) == "" {
		t.Fatalf("expected succeeded uninstall operation with completion timestamp, got %+v", status.ControlOperation)
	}
}

func TestDaemonLifecycleControlRepairReflectsFailure(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	service, err := NewDaemonLifecycleService(DaemonLifecycleServiceConfig{
		Container: container,
		RequestRepair: func(context.Context, transport.DaemonLifecycleControlRequest) error {
			return fmt.Errorf("simulated repair failure")
		},
	})
	if err != nil {
		t.Fatalf("new lifecycle service: %v", err)
	}

	response, err := service.DaemonLifecycleControl(context.Background(), transport.DaemonLifecycleControlRequest{
		Action: "repair",
		Reason: "test repair failure",
	})
	if err != nil {
		t.Fatalf("repair control: %v", err)
	}
	if !response.Accepted || response.OperationState != lifecycleOperationStateInProgress {
		t.Fatalf("expected accepted in-progress repair response, got %+v", response)
	}

	status := waitForLifecycleControlOperationState(t, service, lifecycleOperationStateFailed)
	if status.ControlOperation.Action != "repair" {
		t.Fatalf("expected failed repair action, got %+v", status.ControlOperation)
	}
	if !strings.Contains(status.ControlOperation.Error, "simulated repair failure") {
		t.Fatalf("expected repair error detail in status, got %+v", status.ControlOperation)
	}
}

func TestDaemonLifecycleControlRepairRestartsFailedWorkers(t *testing.T) {
	container, supervisor := newLifecycleTestContainerWithSupervisor(t, []PluginWorkerStatus{
		{
			PluginID: "connector.mail",
			Kind:     shared.AdapterKindConnector,
			State:    PluginWorkerStateFailed,
		},
	})
	service, err := NewDaemonLifecycleService(DaemonLifecycleServiceConfig{
		Container: container,
	})
	if err != nil {
		t.Fatalf("new lifecycle service: %v", err)
	}

	response, err := service.DaemonLifecycleControl(context.Background(), transport.DaemonLifecycleControlRequest{
		Action: "repair",
		Reason: "restart failed workers",
	})
	if err != nil {
		t.Fatalf("repair control: %v", err)
	}
	if !response.Accepted || response.OperationState != lifecycleOperationStateInProgress {
		t.Fatalf("expected accepted in-progress repair response, got %+v", response)
	}

	_ = waitForLifecycleControlOperationState(t, service, lifecycleOperationStateSucceeded)
	if len(supervisor.restartCalls) != 1 || supervisor.restartCalls[0] != "connector.mail" {
		t.Fatalf("expected failed worker restart call for connector.mail, got %+v", supervisor.restartCalls)
	}
}

func TestDaemonLifecycleStatusReportsInstallAndRepairNeeds(t *testing.T) {
	container := newLifecycleTestContainer(t, []PluginWorkerStatus{
		{
			PluginID: "connector.mail",
			Kind:     shared.AdapterKindConnector,
			State:    PluginWorkerStateFailed,
		},
	})

	service, err := NewDaemonLifecycleService(DaemonLifecycleServiceConfig{
		Container:      container,
		ExecutablePath: filepath.Join(t.TempDir(), "missing-daemon"),
	})
	if err != nil {
		t.Fatalf("new lifecycle service: %v", err)
	}

	status, err := service.DaemonLifecycleStatus(context.Background())
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.SetupState != setupStateInstallRequired {
		t.Fatalf("expected install_required setup state, got %s", status.SetupState)
	}
	if status.InstallState != installStateMissing || !status.NeedsInstall {
		t.Fatalf("expected missing install state, got state=%s needs_install=%v", status.InstallState, status.NeedsInstall)
	}
	if !status.NeedsRepair {
		t.Fatalf("expected needs_repair due failed worker")
	}
	if status.WorkerSummary.Failed != 1 || status.WorkerSummary.Total != 1 {
		t.Fatalf("expected failed worker summary, got %+v", status.WorkerSummary)
	}
}

func TestDaemonLifecycleStatusSurfacesPluginAuditPersistenceFailures(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	if _, err := container.DB.ExecContext(context.Background(), `DROP TABLE runtime_plugins`); err != nil {
		t.Fatalf("drop runtime_plugins table: %v", err)
	}

	container.enqueuePluginLifecycleAuditEvent(PluginLifecycleEvent{
		PluginID:  "plugin.channel.audit.failures",
		Kind:      shared.AdapterKindChannel,
		State:     PluginWorkerStateRunning,
		EventType: pluginEventHandshakeAccepted,
		Metadata: shared.AdapterMetadata{
			ID:          "plugin.channel.audit.failures",
			Kind:        shared.AdapterKindChannel,
			DisplayName: "Audit Failure Plugin",
			Version:     "test",
			Capabilities: []shared.CapabilityDescriptor{
				{Key: "channel.sms.send"},
			},
		},
	})

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if diagnostics := container.pluginLifecycleAuditDiagnostics(); diagnostics.PersistFailures > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	diagnostics := container.pluginLifecycleAuditDiagnostics()
	if diagnostics.PersistFailures == 0 {
		t.Fatalf("expected plugin audit persistence failure diagnostics, got %+v", diagnostics)
	}

	executablePath := filepath.Join(t.TempDir(), "personal-agent-daemon")
	if err := osWriteFileExecutable(executablePath); err != nil {
		t.Fatalf("create executable: %v", err)
	}

	service, err := NewDaemonLifecycleService(DaemonLifecycleServiceConfig{
		Container:      container,
		ExecutablePath: executablePath,
	})
	if err != nil {
		t.Fatalf("new lifecycle service: %v", err)
	}

	status, err := service.DaemonLifecycleStatus(context.Background())
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.HealthClassification.PluginRuntimeState != lifecycleHealthPluginsDegraded {
		t.Fatalf("expected degraded plugin runtime from audit persistence failures, got %+v", status.HealthClassification)
	}
	if status.HealthClassification.OverallState != lifecycleHealthOverallDegraded {
		t.Fatalf("expected degraded overall health from audit persistence failures, got %+v", status.HealthClassification)
	}
	if !strings.Contains(status.HealthClassification.PluginReason, "plugin lifecycle audit persistence failure") {
		t.Fatalf("expected plugin reason to include audit persistence diagnostics, got %+v", status.HealthClassification)
	}
	if !status.NeedsRepair {
		t.Fatalf("expected needs_repair=true when plugin lifecycle audit persistence is degraded")
	}
	if status.SetupState != setupStateRepairRequired {
		t.Fatalf("expected setup_state=%s when plugin lifecycle audit persistence is degraded, got %s", setupStateRepairRequired, status.SetupState)
	}
}

func TestDaemonLifecycleControlRejectsUnknownActions(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	service, err := NewDaemonLifecycleService(DaemonLifecycleServiceConfig{
		Container: container,
	})
	if err != nil {
		t.Fatalf("new lifecycle service: %v", err)
	}

	_, err = service.DaemonLifecycleControl(context.Background(), transport.DaemonLifecycleControlRequest{Action: "explode"})
	if err == nil {
		t.Fatalf("expected error for unknown action")
	}
	if !strings.Contains(err.Error(), "unsupported lifecycle action") {
		t.Fatalf("expected unsupported action error, got %v", err)
	}
}

func TestDaemonLifecycleControlRestoresStateOnHookError(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	service, err := NewDaemonLifecycleService(DaemonLifecycleServiceConfig{
		Container: container,
		RequestRestart: func(context.Context) error {
			return fmt.Errorf("restart queue unavailable")
		},
	})
	if err != nil {
		t.Fatalf("new lifecycle service: %v", err)
	}

	if _, err := service.DaemonLifecycleControl(context.Background(), transport.DaemonLifecycleControlRequest{Action: "restart"}); err == nil {
		t.Fatalf("expected restart hook error")
	}

	status, err := service.DaemonLifecycleStatus(context.Background())
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.LifecycleState != lifecycleStateRunning {
		t.Fatalf("expected lifecycle state to revert to running, got %s", status.LifecycleState)
	}
}

func TestDaemonPluginLifecycleHistoryFiltersPaginationAndClassification(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	service, err := NewDaemonLifecycleService(DaemonLifecycleServiceConfig{
		Container: container,
	})
	if err != nil {
		t.Fatalf("new lifecycle service: %v", err)
	}

	base := time.Date(2026, 2, 25, 0, 0, 0, 0, time.UTC)
	seedLifecyclePluginEvent(t, container, PluginLifecycleEvent{
		PluginID:       "messages.daemon",
		Kind:           shared.AdapterKindChannel,
		State:          PluginWorkerStateStarting,
		EventType:      pluginEventWorkerStarted,
		ProcessID:      1101,
		RestartCount:   0,
		OccurredAt:     base.Add(1 * time.Second),
		LastTransition: base.Add(1 * time.Second),
	})
	seedLifecyclePluginEvent(t, container, PluginLifecycleEvent{
		PluginID:       "messages.daemon",
		Kind:           shared.AdapterKindChannel,
		State:          PluginWorkerStateRunning,
		EventType:      pluginEventHandshakeAccepted,
		ProcessID:      1101,
		RestartCount:   0,
		OccurredAt:     base.Add(2 * time.Second),
		LastTransition: base.Add(2 * time.Second),
		LastHeartbeat:  base.Add(2 * time.Second),
	})
	seedLifecyclePluginEvent(t, container, PluginLifecycleEvent{
		PluginID:       "messages.daemon",
		Kind:           shared.AdapterKindChannel,
		State:          PluginWorkerStateFailed,
		EventType:      pluginEventHealthTimeout,
		ProcessID:      1101,
		RestartCount:   0,
		Error:          "health timeout",
		ErrorSource:    "health_timeout",
		ErrorOperation: "health",
		ErrorStderr:    "worker heartbeat stopped",
		OccurredAt:     base.Add(3 * time.Second),
		LastTransition: base.Add(3 * time.Second),
	})
	seedLifecyclePluginEvent(t, container, PluginLifecycleEvent{
		PluginID:       "messages.daemon",
		Kind:           shared.AdapterKindChannel,
		State:          PluginWorkerStateRestarting,
		EventType:      pluginEventWorkerRestarting,
		ProcessID:      1101,
		RestartCount:   1,
		Error:          "health timeout",
		ErrorSource:    "health_timeout",
		ErrorOperation: "health",
		ErrorStderr:    "worker heartbeat stopped",
		OccurredAt:     base.Add(4 * time.Second),
		LastTransition: base.Add(4 * time.Second),
	})
	seedLifecyclePluginEvent(t, container, PluginLifecycleEvent{
		PluginID:       "messages.daemon",
		Kind:           shared.AdapterKindChannel,
		State:          PluginWorkerStateRunning,
		EventType:      pluginEventHandshakeAccepted,
		ProcessID:      1102,
		RestartCount:   1,
		OccurredAt:     base.Add(5 * time.Second),
		LastTransition: base.Add(5 * time.Second),
		LastHeartbeat:  base.Add(5 * time.Second),
	})
	seedLifecyclePluginEvent(t, container, PluginLifecycleEvent{
		PluginID:       "connector.mail",
		Kind:           shared.AdapterKindConnector,
		State:          PluginWorkerStateRunning,
		EventType:      pluginEventHandshakeAccepted,
		ProcessID:      2201,
		RestartCount:   0,
		OccurredAt:     base.Add(6 * time.Second),
		LastTransition: base.Add(6 * time.Second),
		LastHeartbeat:  base.Add(6 * time.Second),
	})

	firstPage, err := service.DaemonPluginLifecycleHistory(context.Background(), transport.DaemonPluginLifecycleHistoryRequest{
		WorkspaceID: daemonPluginAuditWorkspaceID,
		PluginID:    "messages.daemon",
		Limit:       3,
	})
	if err != nil {
		t.Fatalf("daemon plugin lifecycle history first page: %v", err)
	}
	if firstPage.WorkspaceID != daemonPluginAuditWorkspaceID {
		t.Fatalf("unexpected workspace in lifecycle history response: %+v", firstPage)
	}
	if len(firstPage.Items) != 3 || !firstPage.HasMore {
		t.Fatalf("expected first page to contain 3 items with has_more=true, got %+v", firstPage)
	}
	if firstPage.Items[0].EventType != pluginEventHandshakeAccepted || !firstPage.Items[0].RecoveryEvent || firstPage.Items[0].Reason != "worker_recovered" {
		t.Fatalf("expected recovered handshake record first, got %+v", firstPage.Items[0])
	}
	if firstPage.Items[1].EventType != pluginEventWorkerRestarting || !firstPage.Items[1].RestartEvent || firstPage.Items[1].Reason != "restart_after_error" {
		t.Fatalf("expected restart record second, got %+v", firstPage.Items[1])
	}
	if firstPage.Items[1].ErrorSource != "health_timeout" || firstPage.Items[1].ErrorOperation != "health" || strings.TrimSpace(firstPage.Items[1].ErrorStderr) == "" {
		t.Fatalf("expected restart record to include error context fields, got %+v", firstPage.Items[1])
	}
	if firstPage.Items[2].EventType != pluginEventHealthTimeout || !firstPage.Items[2].FailureEvent || firstPage.Items[2].Reason != "health_timeout" {
		t.Fatalf("expected failure record third, got %+v", firstPage.Items[2])
	}
	if firstPage.Items[0].OccurredAt <= firstPage.Items[1].OccurredAt || firstPage.Items[1].OccurredAt <= firstPage.Items[2].OccurredAt {
		t.Fatalf("expected strict descending occurred_at ordering, got %+v", firstPage.Items)
	}

	secondPage, err := service.DaemonPluginLifecycleHistory(context.Background(), transport.DaemonPluginLifecycleHistoryRequest{
		WorkspaceID:     daemonPluginAuditWorkspaceID,
		PluginID:        "messages.daemon",
		CursorCreatedAt: firstPage.NextCursorCreatedAt,
		CursorID:        firstPage.NextCursorID,
		Limit:           3,
	})
	if err != nil {
		t.Fatalf("daemon plugin lifecycle history second page: %v", err)
	}
	if len(secondPage.Items) != 2 || secondPage.HasMore {
		t.Fatalf("expected second page to contain 2 items with has_more=false, got %+v", secondPage)
	}
	for _, item := range secondPage.Items {
		if item.PluginID != "messages.daemon" {
			t.Fatalf("expected plugin filter to retain only messages.daemon, got %+v", item)
		}
	}

	connectorOnly, err := service.DaemonPluginLifecycleHistory(context.Background(), transport.DaemonPluginLifecycleHistoryRequest{
		WorkspaceID: daemonPluginAuditWorkspaceID,
		Kind:        "connector",
		Limit:       10,
	})
	if err != nil {
		t.Fatalf("daemon plugin lifecycle connector filter: %v", err)
	}
	if len(connectorOnly.Items) != 1 || connectorOnly.Items[0].PluginID != "connector.mail" || connectorOnly.Items[0].Kind != "connector" {
		t.Fatalf("unexpected connector lifecycle history filter result: %+v", connectorOnly.Items)
	}

	runningOnly, err := service.DaemonPluginLifecycleHistory(context.Background(), transport.DaemonPluginLifecycleHistoryRequest{
		WorkspaceID: daemonPluginAuditWorkspaceID,
		PluginID:    "messages.daemon",
		State:       "running",
		Limit:       10,
	})
	if err != nil {
		t.Fatalf("daemon plugin lifecycle running state filter: %v", err)
	}
	if len(runningOnly.Items) != 2 {
		t.Fatalf("expected two running records for messages.daemon, got %+v", runningOnly.Items)
	}
	for _, item := range runningOnly.Items {
		if item.State != "running" {
			t.Fatalf("expected running state rows, got %+v", item)
		}
	}
}

func TestDaemonPluginLifecycleHistoryValidationErrors(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	service, err := NewDaemonLifecycleService(DaemonLifecycleServiceConfig{
		Container: container,
	})
	if err != nil {
		t.Fatalf("new lifecycle service: %v", err)
	}

	if _, err := service.DaemonPluginLifecycleHistory(context.Background(), transport.DaemonPluginLifecycleHistoryRequest{
		Kind: "not-a-kind",
	}); err == nil {
		t.Fatalf("expected invalid kind filter error")
	}
	if _, err := service.DaemonPluginLifecycleHistory(context.Background(), transport.DaemonPluginLifecycleHistoryRequest{
		State: "not-a-state",
	}); err == nil {
		t.Fatalf("expected invalid state filter error")
	}
	if _, err := service.DaemonPluginLifecycleHistory(context.Background(), transport.DaemonPluginLifecycleHistoryRequest{
		EventType: "NOT_A_REAL_EVENT",
	}); err == nil {
		t.Fatalf("expected invalid event_type filter error")
	}
	if _, err := service.DaemonPluginLifecycleHistory(context.Background(), transport.DaemonPluginLifecycleHistoryRequest{
		CursorCreatedAt: "not-a-timestamp",
	}); err == nil {
		t.Fatalf("expected invalid cursor_created_at error")
	}
	if _, err := service.DaemonPluginLifecycleHistory(context.Background(), transport.DaemonPluginLifecycleHistoryRequest{
		CursorID: "audit-only-without-created-at",
	}); err == nil {
		t.Fatalf("expected cursor_id without cursor_created_at to fail")
	}
}

func seedLifecyclePluginEvent(t *testing.T, container *ServiceContainer, event PluginLifecycleEvent) {
	t.Helper()
	if strings.TrimSpace(event.Metadata.ID) == "" {
		event.Metadata = shared.AdapterMetadata{
			ID:          event.PluginID,
			Kind:        event.Kind,
			DisplayName: event.PluginID,
			Version:     "test",
			Capabilities: []shared.CapabilityDescriptor{
				{Key: "health"},
			},
		}
	}
	container.recordPluginLifecycleEvent(event)
}

func waitForLifecycleControlOperationState(t *testing.T, service *DaemonLifecycleService, expected string) transport.DaemonLifecycleStatusResponse {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		status, err := service.DaemonLifecycleStatus(context.Background())
		if err != nil {
			t.Fatalf("read lifecycle status: %v", err)
		}
		if status.ControlOperation.State == expected {
			return status
		}
		if time.Now().After(deadline) {
			t.Fatalf("timeout waiting for lifecycle control operation state %q, last=%+v", expected, status.ControlOperation)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func osWriteFileExecutable(path string) error {
	content := []byte("#!/bin/sh\nexit 0\n")
	return os.WriteFile(path, content, 0o755)
}
