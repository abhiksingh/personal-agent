package daemonruntime

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"personalagent/runtime/internal/transport"
)

func resolveInstallState(executablePath string) (string, bool) {
	trimmedPath := strings.TrimSpace(executablePath)
	if trimmedPath == "" {
		return installStateMissing, true
	}
	info, err := os.Stat(trimmedPath)
	if err != nil || info.IsDir() {
		return installStateMissing, true
	}
	if info.Mode()&0o111 == 0 {
		return installStateMissing, true
	}
	return installStateInstalled, false
}

func resolveDatabaseState(ctx context.Context, container *ServiceContainer) (bool, string) {
	if container == nil || container.DB == nil {
		return false, "database handle is not configured"
	}
	if isSingleWriterPoolSaturated(container.DB) {
		// Single-writer pool saturation means work is in-flight and the connection
		// is healthy but busy. Treat as ready to avoid false repair states.
		return true, ""
	}
	checkCtx, cancel := context.WithTimeout(ctx, 750*time.Millisecond)
	defer cancel()
	if err := container.DB.PingContext(checkCtx); err != nil {
		if isDatabaseProbeTimeout(err) && isSingleWriterPoolSaturated(container.DB) {
			return true, ""
		}
		return false, err.Error()
	}
	return true, ""
}

func classifyLifecycleHealth(
	lifecycleState string,
	needsInstall bool,
	databaseReady bool,
	databaseError string,
	workerSummary transport.DaemonWorkerStateSummary,
	auditDiagnostics pluginLifecycleAuditDiagnostics,
) transport.DaemonLifecycleHealthClassification {
	classification := transport.DaemonLifecycleHealthClassification{
		OverallState:       lifecycleHealthOverallReady,
		CoreRuntimeState:   lifecycleHealthCoreReady,
		PluginRuntimeState: lifecycleHealthPluginsHealthy,
		Blocking:           false,
	}

	if needsInstall {
		classification.OverallState = lifecycleHealthOverallBlocked
		classification.CoreRuntimeState = lifecycleHealthCoreInstallRequired
		classification.Blocking = true
		classification.CoreReason = "daemon executable is missing; install is required"
	} else if !databaseReady {
		classification.OverallState = lifecycleHealthOverallBlocked
		classification.CoreRuntimeState = lifecycleHealthCoreDatabaseUnavailable
		classification.Blocking = true
		if strings.TrimSpace(databaseError) == "" {
			classification.CoreReason = "daemon database is not reachable; run repair and restart"
		} else {
			classification.CoreReason = fmt.Sprintf("daemon database is not reachable: %s", strings.TrimSpace(databaseError))
		}
	} else if !isLifecycleControlPlaneReady(lifecycleState) {
		classification.OverallState = lifecycleHealthOverallBlocked
		classification.CoreRuntimeState = lifecycleHealthCoreControlPlaneDown
		classification.Blocking = true
		classification.CoreReason = fmt.Sprintf(
			"daemon lifecycle state %q is not ready for control-plane operations",
			strings.TrimSpace(lifecycleState),
		)
	}

	if workerSummary.Failed > 0 {
		classification.PluginRuntimeState = lifecycleHealthPluginsDegraded
		classification.PluginReason = fmt.Sprintf(
			"%d plugin worker(s) are failed; inspect Channels/Connectors diagnostics",
			workerSummary.Failed,
		)
		if classification.OverallState == lifecycleHealthOverallReady {
			classification.OverallState = lifecycleHealthOverallDegraded
		}
	}
	if auditDiagnostics.degraded() {
		classification.PluginRuntimeState = lifecycleHealthPluginsDegraded
		pluginReason := pluginLifecycleAuditDiagnosticsReason(auditDiagnostics)
		if classification.PluginReason == "" {
			classification.PluginReason = pluginReason
		} else {
			classification.PluginReason = fmt.Sprintf("%s; %s", classification.PluginReason, pluginReason)
		}
		if classification.OverallState == lifecycleHealthOverallReady {
			classification.OverallState = lifecycleHealthOverallDegraded
		}
	}

	return classification
}

func pluginLifecycleAuditDiagnosticsReason(diagnostics pluginLifecycleAuditDiagnostics) string {
	parts := make([]string, 0, 2)
	if diagnostics.PersistFailures > 0 {
		parts = append(parts, fmt.Sprintf("%d plugin lifecycle audit persistence failure(s)", diagnostics.PersistFailures))
	}
	if diagnostics.DroppedEvents > 0 {
		parts = append(parts, fmt.Sprintf("%d plugin lifecycle audit event(s) dropped under pressure", diagnostics.DroppedEvents))
	}
	if len(parts) == 0 {
		return ""
	}
	reason := strings.Join(parts, "; ")
	if message := strings.TrimSpace(diagnostics.LastFailure); message != "" {
		reason = fmt.Sprintf("%s (last_error=%s)", reason, message)
	}
	return reason
}

func isLifecycleControlPlaneReady(state string) bool {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case lifecycleStateRunning, lifecycleStateRestartRequested:
		return true
	default:
		return false
	}
}

func lifecycleFirstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func isTransientDatabaseProbeTimeout(databaseError string) bool {
	normalized := strings.ToLower(strings.TrimSpace(databaseError))
	if normalized == "" {
		return false
	}
	return strings.Contains(normalized, "context deadline exceeded")
}

func isDatabaseProbeTimeout(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(strings.TrimSpace(err.Error())), "context deadline exceeded")
}

func isSingleWriterPoolSaturated(db *sql.DB) bool {
	if db == nil {
		return false
	}
	stats := db.Stats()
	if stats.MaxOpenConnections <= 0 {
		return false
	}
	return stats.MaxOpenConnections == daemonSQLiteMaxOpenConns &&
		stats.OpenConnections >= daemonSQLiteMaxOpenConns &&
		stats.InUse >= daemonSQLiteMaxOpenConns
}

func summarizeWorkerStates(container *ServiceContainer) transport.DaemonWorkerStateSummary {
	summary := transport.DaemonWorkerStateSummary{}
	if container == nil || container.PluginSupervisor == nil {
		return summary
	}
	workers := container.PluginSupervisor.ListWorkers()
	summary.Total = len(workers)
	for _, worker := range workers {
		switch strings.ToLower(strings.TrimSpace(string(worker.State))) {
		case string(PluginWorkerStateRegistered):
			summary.Registered++
		case string(PluginWorkerStateStarting):
			summary.Starting++
		case string(PluginWorkerStateRunning):
			summary.Running++
		case string(PluginWorkerStateRestarting):
			summary.Restarting++
		case string(PluginWorkerStateFailed):
			summary.Failed++
		case string(PluginWorkerStateStopped):
			summary.Stopped++
		default:
			summary.Stopped++
		}
	}
	return summary
}

func controlsForLifecycleState(state string, setupState string, operationState string) transport.DaemonLifecycleControls {
	controls := transport.DaemonLifecycleControls{}
	switch strings.TrimSpace(state) {
	case lifecycleStateRunning:
		controls.Start = true
		controls.Stop = true
		controls.Restart = true
	case lifecycleStateStopRequested:
		controls.Start = false
		controls.Stop = true
		controls.Restart = false
	case lifecycleStateRestartRequested:
		controls.Start = false
		controls.Stop = false
		controls.Restart = true
	default:
		controls.Start = true
		controls.Stop = false
		controls.Restart = false
	}

	setupControlsEnabled := strings.TrimSpace(state) == lifecycleStateRunning
	if setupControlsEnabled {
		controls.Install = true
		controls.Uninstall = true
		controls.Repair = true
	}

	switch strings.TrimSpace(setupState) {
	case setupStateInstallRequired:
		controls.Uninstall = false
	case setupStateRepairRequired:
		// keep all setup controls available while repair is needed.
	}

	if strings.TrimSpace(operationState) == lifecycleOperationStateInProgress {
		controls.Install = false
		controls.Uninstall = false
		controls.Repair = false
	}

	return controls
}
