package daemonruntime

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"personalagent/runtime/internal/transport"
)

const (
	lifecycleStateRunning          = "running"
	lifecycleStateStopRequested    = "stop_requested"
	lifecycleStateRestartRequested = "restart_requested"

	lifecycleActionStart     = "start"
	lifecycleActionStop      = "stop"
	lifecycleActionRestart   = "restart"
	lifecycleActionInstall   = "install"
	lifecycleActionUninstall = "uninstall"
	lifecycleActionRepair    = "repair"

	setupStateReady           = "ready"
	setupStateRepairRequired  = "repair_required"
	setupStateInstallRequired = "install_required"

	installStateInstalled = "installed"
	installStateMissing   = "missing"

	lifecycleOperationStateIdle       = "idle"
	lifecycleOperationStateInProgress = "in_progress"
	lifecycleOperationStateSucceeded  = "succeeded"
	lifecycleOperationStateFailed     = "failed"

	lifecycleHealthOverallReady    = "ready"
	lifecycleHealthOverallDegraded = "degraded"
	lifecycleHealthOverallBlocked  = "blocked"

	lifecycleHealthCoreReady               = "ready"
	lifecycleHealthCoreInstallRequired     = "install_required"
	lifecycleHealthCoreDatabaseUnavailable = "database_unavailable"
	lifecycleHealthCoreControlPlaneDown    = "control_plane_unavailable"

	lifecycleHealthPluginsHealthy  = "healthy"
	lifecycleHealthPluginsDegraded = "degraded"

	daemonControlAuthStateConfigured = "configured"
	daemonControlAuthStateMissing    = "missing"

	daemonControlAuthSourceFlag    = "auth_token_flag"
	daemonControlAuthSourceFile    = "auth_token_file"
	daemonControlAuthSourceUnknown = "unknown"

	daemonPluginLifecycleHistoryDefaultLimit = 50
	daemonPluginLifecycleHistoryMaxLimit     = 200
	daemonPluginLifecycleHistoryScanLimit    = 500

	daemonDatabaseProbeGraceWindow = 20 * time.Second
)

type daemonLifecycleSetupActionHook func(ctx context.Context, request transport.DaemonLifecycleControlRequest) error

type daemonLifecycleOperationState struct {
	Action      string
	State       string
	Message     string
	Error       string
	RequestedAt time.Time
	CompletedAt time.Time
}

type daemonPluginLifecycleAuditRow struct {
	AuditID       string
	WorkspaceID   string
	EventType     string
	CorrelationID string
	PayloadJSON   string
	CreatedAt     string
}

type daemonPluginLifecycleAuditPayload struct {
	PluginID         string `json:"plugin_id"`
	Kind             string `json:"kind"`
	State            string `json:"state"`
	ProcessID        int    `json:"process_id"`
	RestartCount     int    `json:"restart_count"`
	Error            string `json:"error"`
	ErrorSource      string `json:"error_source"`
	ErrorOperation   string `json:"error_operation"`
	ErrorStderr      string `json:"error_stderr"`
	LastHeartbeatAt  string `json:"last_heartbeat_at"`
	LastTransitionAt string `json:"last_transition_at"`
	OccurredAt       string `json:"occurred_at"`
}

type DaemonLifecycleServiceConfig struct {
	Container         *ServiceContainer
	RuntimeMode       string
	ConfiguredAddress string
	ExecutablePath    string
	AuthToken         string
	AuthTokenSource   string
	RequestStop       func(ctx context.Context) error
	RequestRestart    func(ctx context.Context) error
	RequestInstall    daemonLifecycleSetupActionHook
	RequestUninstall  daemonLifecycleSetupActionHook
	RequestRepair     daemonLifecycleSetupActionHook
	Now               func() time.Time
}

type DaemonLifecycleService struct {
	container         *ServiceContainer
	runtimeMode       string
	configuredAddress string
	executablePath    string
	authToken         string
	authTokenSource   string
	requestStop       func(ctx context.Context) error
	requestRestart    func(ctx context.Context) error
	requestInstall    daemonLifecycleSetupActionHook
	requestUninstall  daemonLifecycleSetupActionHook
	requestRepair     daemonLifecycleSetupActionHook
	now               func() time.Time

	mu                  sync.RWMutex
	lifecycleState      string
	startedAt           time.Time
	lastTransition      time.Time
	boundAddress        string
	setupOperation      daemonLifecycleOperationState
	lastDatabaseReadyAt time.Time
}

func NewDaemonLifecycleService(config DaemonLifecycleServiceConfig) (*DaemonLifecycleService, error) {
	if config.Container == nil {
		return nil, fmt.Errorf("service container is required")
	}
	if config.Now == nil {
		config.Now = func() time.Time { return time.Now().UTC() }
	}

	now := config.Now().UTC()
	return &DaemonLifecycleService{
		container:           config.Container,
		runtimeMode:         strings.TrimSpace(config.RuntimeMode),
		configuredAddress:   strings.TrimSpace(config.ConfiguredAddress),
		executablePath:      strings.TrimSpace(config.ExecutablePath),
		authToken:           strings.TrimSpace(config.AuthToken),
		authTokenSource:     normalizeDaemonControlAuthSource(config.AuthTokenSource),
		requestStop:         config.RequestStop,
		requestRestart:      config.RequestRestart,
		requestInstall:      config.RequestInstall,
		requestUninstall:    config.RequestUninstall,
		requestRepair:       config.RequestRepair,
		now:                 config.Now,
		lifecycleState:      lifecycleStateRunning,
		startedAt:           now,
		lastTransition:      now,
		lastDatabaseReadyAt: now,
		setupOperation: daemonLifecycleOperationState{
			State: lifecycleOperationStateIdle,
		},
	}, nil
}

func (s *DaemonLifecycleService) SetBoundAddress(address string) {
	s.mu.Lock()
	s.boundAddress = strings.TrimSpace(address)
	s.mu.Unlock()
}

func (s *DaemonLifecycleService) DaemonLifecycleStatus(ctx context.Context) (transport.DaemonLifecycleStatusResponse, error) {
	s.mu.RLock()
	state := s.lifecycleState
	startedAt := s.startedAt
	lastTransition := s.lastTransition
	boundAddress := s.boundAddress
	setupOperation := s.setupOperation
	s.mu.RUnlock()

	installState, needsInstall := resolveInstallState(s.executablePath)
	databaseReady, databaseError := resolveDatabaseState(ctx, s.container)
	databaseReady, databaseError = s.reconcileDatabaseProbeState(s.now().UTC(), databaseReady, databaseError)
	workerSummary := summarizeWorkerStates(s.container)
	auditDiagnostics := s.container.pluginLifecycleAuditDiagnostics()
	healthClassification := classifyLifecycleHealth(
		state,
		needsInstall,
		databaseReady,
		databaseError,
		workerSummary,
		auditDiagnostics,
	)
	needsRepair := !databaseReady || workerSummary.Failed > 0 || auditDiagnostics.degraded()
	setupState := setupStateReady
	repairHint := ""
	switch {
	case needsInstall:
		setupState = setupStateInstallRequired
		repairHint = lifecycleFirstNonEmpty(
			healthClassification.CoreReason,
			"daemon executable is missing; install is required",
		)
	case !databaseReady:
		setupState = setupStateRepairRequired
		repairHint = lifecycleFirstNonEmpty(
			healthClassification.CoreReason,
			"daemon database is not reachable; run repair and restart",
		)
	case workerSummary.Failed > 0:
		setupState = setupStateRepairRequired
		repairHint = lifecycleFirstNonEmpty(
			healthClassification.PluginReason,
			"one or more plugin workers are failed; run repair or restart daemon",
		)
	case auditDiagnostics.degraded():
		setupState = setupStateRepairRequired
		repairHint = lifecycleFirstNonEmpty(
			healthClassification.PluginReason,
			"plugin lifecycle audit persistence is degraded; inspect daemon lifecycle diagnostics and restart",
		)
	}
	if repairHint == "" {
		repairHint = lifecycleFirstNonEmpty(healthClassification.CoreReason, healthClassification.PluginReason)
	}

	return transport.DaemonLifecycleStatusResponse{
		LifecycleState:       state,
		ProcessID:            os.Getpid(),
		StartedAt:            formatRFC3339(startedAt),
		LastTransitionAt:     formatRFC3339(lastTransition),
		RuntimeMode:          s.runtimeMode,
		ConfiguredAddress:    s.configuredAddress,
		BoundAddress:         boundAddress,
		SetupState:           setupState,
		InstallState:         installState,
		NeedsInstall:         needsInstall,
		NeedsRepair:          needsRepair,
		RepairHint:           repairHint,
		HealthClassification: healthClassification,
		ExecutablePath:       s.executablePath,
		DatabasePath:         strings.TrimSpace(s.container.DBPath),
		DatabaseReady:        databaseReady,
		DatabaseError:        databaseError,
		ControlAuth:          daemonControlAuthState(s.authToken, s.authTokenSource),
		WorkerSummary:        workerSummary,
		Controls:             controlsForLifecycleState(state, setupState, setupOperation.State),
		ControlOperation:     transportLifecycleOperation(setupOperation),
	}, nil
}

func (s *DaemonLifecycleService) reconcileDatabaseProbeState(now time.Time, databaseReady bool, databaseError string) (bool, string) {
	if databaseReady {
		s.mu.Lock()
		s.lastDatabaseReadyAt = now.UTC()
		s.mu.Unlock()
		return true, ""
	}
	if !isTransientDatabaseProbeTimeout(databaseError) {
		return false, databaseError
	}

	s.mu.RLock()
	lastReadyAt := s.lastDatabaseReadyAt
	s.mu.RUnlock()
	if lastReadyAt.IsZero() {
		return false, databaseError
	}
	if now.UTC().Sub(lastReadyAt.UTC()) <= daemonDatabaseProbeGraceWindow {
		return true, ""
	}
	return false, databaseError
}

func (s *DaemonLifecycleService) DaemonLifecycleControl(ctx context.Context, request transport.DaemonLifecycleControlRequest) (transport.DaemonLifecycleControlResponse, error) {
	action := strings.ToLower(strings.TrimSpace(request.Action))
	if action == "" {
		return transport.DaemonLifecycleControlResponse{}, fmt.Errorf("action is required")
	}
	request.Action = action
	switch action {
	case lifecycleActionStart:
		state := s.currentLifecycleState()
		now := s.now().UTC()
		return transport.DaemonLifecycleControlResponse{
			Action:         action,
			Accepted:       true,
			Idempotent:     true,
			LifecycleState: state,
			Message:        "daemon is already running",
			OperationState: lifecycleOperationStateSucceeded,
			RequestedAt:    formatRFC3339(now),
			CompletedAt:    formatRFC3339(now),
		}, nil
	case lifecycleActionStop:
		return s.requestLifecycleTransition(ctx, action, lifecycleStateStopRequested, s.requestStop)
	case lifecycleActionRestart:
		return s.requestLifecycleTransition(ctx, action, lifecycleStateRestartRequested, s.requestRestart)
	case lifecycleActionInstall, lifecycleActionUninstall, lifecycleActionRepair:
		return s.requestSetupOperation(action, request)
	default:
		return transport.DaemonLifecycleControlResponse{}, fmt.Errorf("unsupported lifecycle action %q", request.Action)
	}
}

func (s *DaemonLifecycleService) DaemonPluginLifecycleHistory(
	ctx context.Context,
	request transport.DaemonPluginLifecycleHistoryRequest,
) (transport.DaemonPluginLifecycleHistoryResponse, error) {
	if s.container == nil || s.container.DB == nil {
		return transport.DaemonPluginLifecycleHistoryResponse{}, fmt.Errorf("daemon service container database is not configured")
	}

	workspaceID := strings.TrimSpace(request.WorkspaceID)
	if workspaceID == "" {
		workspaceID = daemonPluginAuditWorkspaceID
	}

	pluginID := strings.TrimSpace(request.PluginID)
	kind, err := normalizeDaemonPluginLifecycleKindFilter(request.Kind)
	if err != nil {
		return transport.DaemonPluginLifecycleHistoryResponse{}, err
	}
	state, err := normalizeDaemonPluginLifecycleStateFilter(request.State)
	if err != nil {
		return transport.DaemonPluginLifecycleHistoryResponse{}, err
	}
	eventType, err := normalizeDaemonPluginLifecycleEventTypeFilter(request.EventType)
	if err != nil {
		return transport.DaemonPluginLifecycleHistoryResponse{}, err
	}
	cursorCreatedAt, cursorID, err := normalizeDaemonPluginLifecycleCursor(request.CursorCreatedAt, request.CursorID)
	if err != nil {
		return transport.DaemonPluginLifecycleHistoryResponse{}, err
	}
	limit := clampDaemonPluginLifecycleHistoryLimit(request.Limit)

	items := make([]transport.DaemonPluginLifecycleHistoryRecord, 0, limit+1)
	scanCursorCreatedAt := cursorCreatedAt
	scanCursorID := cursorID
	scanLimit := daemonPluginLifecycleHistoryScanLimit
	if scanLimit < limit+1 {
		scanLimit = limit + 1
	}

	for len(items) < limit+1 {
		rows, queryErr := s.queryDaemonPluginLifecycleAuditRows(
			ctx,
			workspaceID,
			pluginID,
			eventType,
			scanCursorCreatedAt,
			scanCursorID,
			scanLimit,
		)
		if queryErr != nil {
			return transport.DaemonPluginLifecycleHistoryResponse{}, queryErr
		}
		if len(rows) == 0 {
			break
		}

		for _, row := range rows {
			record := daemonPluginLifecycleRecordFromAuditRow(row)
			if kind != "" && !strings.EqualFold(record.Kind, kind) {
				continue
			}
			if state != "" && !strings.EqualFold(record.State, state) {
				continue
			}
			items = append(items, record)
			if len(items) >= limit+1 {
				break
			}
		}

		last := rows[len(rows)-1]
		scanCursorCreatedAt = strings.TrimSpace(last.CreatedAt)
		scanCursorID = strings.TrimSpace(last.AuditID)
		if len(rows) < scanLimit {
			break
		}
	}

	response := transport.DaemonPluginLifecycleHistoryResponse{
		WorkspaceID: workspaceID,
		Items:       []transport.DaemonPluginLifecycleHistoryRecord{},
		HasMore:     false,
	}
	if len(items) == 0 {
		return response, nil
	}

	if len(items) > limit {
		response.HasMore = true
		items = items[:limit]
	}
	response.Items = items
	if response.HasMore {
		last := items[len(items)-1]
		response.NextCursorCreatedAt = last.OccurredAt
		response.NextCursorID = last.AuditID
	}
	return response, nil
}
