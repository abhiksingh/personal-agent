package transport

type DaemonWorkerStateSummary struct {
	Total      int `json:"total"`
	Registered int `json:"registered"`
	Starting   int `json:"starting"`
	Running    int `json:"running"`
	Restarting int `json:"restarting"`
	Stopped    int `json:"stopped"`
	Failed     int `json:"failed"`
}

type DaemonLifecycleControls struct {
	Start     bool `json:"start"`
	Stop      bool `json:"stop"`
	Restart   bool `json:"restart"`
	Install   bool `json:"install,omitempty"`
	Uninstall bool `json:"uninstall,omitempty"`
	Repair    bool `json:"repair,omitempty"`
}

type DaemonLifecycleControlOperation struct {
	Action      string `json:"action,omitempty"`
	State       string `json:"state"`
	Message     string `json:"message,omitempty"`
	Error       string `json:"error,omitempty"`
	RequestedAt string `json:"requested_at,omitempty"`
	CompletedAt string `json:"completed_at,omitempty"`
}

type DaemonLifecycleHealthClassification struct {
	OverallState       string `json:"overall_state"`
	CoreRuntimeState   string `json:"core_runtime_state"`
	PluginRuntimeState string `json:"plugin_runtime_state"`
	Blocking           bool   `json:"blocking"`
	CoreReason         string `json:"core_reason,omitempty"`
	PluginReason       string `json:"plugin_reason,omitempty"`
}

type DaemonControlAuthState struct {
	State            string   `json:"state"`
	Source           string   `json:"source"`
	RemediationHints []string `json:"remediation_hints,omitempty"`
}

type DaemonLifecycleStatusResponse struct {
	LifecycleState       string                              `json:"lifecycle_state"`
	ProcessID            int                                 `json:"process_id"`
	StartedAt            string                              `json:"started_at"`
	LastTransitionAt     string                              `json:"last_transition_at"`
	RuntimeMode          string                              `json:"runtime_mode,omitempty"`
	ConfiguredAddress    string                              `json:"configured_address,omitempty"`
	BoundAddress         string                              `json:"bound_address,omitempty"`
	SetupState           string                              `json:"setup_state"`
	InstallState         string                              `json:"install_state"`
	NeedsInstall         bool                                `json:"needs_install"`
	NeedsRepair          bool                                `json:"needs_repair"`
	RepairHint           string                              `json:"repair_hint,omitempty"`
	HealthClassification DaemonLifecycleHealthClassification `json:"health_classification"`
	ExecutablePath       string                              `json:"executable_path,omitempty"`
	DatabasePath         string                              `json:"database_path,omitempty"`
	DatabaseReady        bool                                `json:"database_ready"`
	DatabaseError        string                              `json:"database_error,omitempty"`
	ControlAuth          DaemonControlAuthState              `json:"control_auth"`
	WorkerSummary        DaemonWorkerStateSummary            `json:"worker_summary"`
	Controls             DaemonLifecycleControls             `json:"controls"`
	ControlOperation     DaemonLifecycleControlOperation     `json:"control_operation"`
}

type DaemonLifecycleControlRequest struct {
	Action string `json:"action"`
	Reason string `json:"reason,omitempty"`
}

type DaemonLifecycleControlResponse struct {
	Action         string `json:"action"`
	Accepted       bool   `json:"accepted"`
	Idempotent     bool   `json:"idempotent"`
	LifecycleState string `json:"lifecycle_state"`
	Message        string `json:"message,omitempty"`
	OperationState string `json:"operation_state,omitempty"`
	RequestedAt    string `json:"requested_at,omitempty"`
	CompletedAt    string `json:"completed_at,omitempty"`
	Error          string `json:"error,omitempty"`
}

type DaemonPluginLifecycleHistoryRequest struct {
	WorkspaceID     string `json:"workspace_id,omitempty"`
	PluginID        string `json:"plugin_id,omitempty"`
	Kind            string `json:"kind,omitempty"`
	State           string `json:"state,omitempty"`
	EventType       string `json:"event_type,omitempty"`
	CursorCreatedAt string `json:"cursor_created_at,omitempty"`
	CursorID        string `json:"cursor_id,omitempty"`
	Limit           int    `json:"limit,omitempty"`
}

type DaemonPluginLifecycleHistoryRecord struct {
	AuditID          string `json:"audit_id"`
	WorkspaceID      string `json:"workspace_id"`
	PluginID         string `json:"plugin_id"`
	Kind             string `json:"kind"`
	State            string `json:"state"`
	EventType        string `json:"event_type"`
	ProcessID        int    `json:"process_id"`
	RestartCount     int    `json:"restart_count"`
	Reason           string `json:"reason"`
	Error            string `json:"error,omitempty"`
	ErrorSource      string `json:"error_source,omitempty"`
	ErrorOperation   string `json:"error_operation,omitempty"`
	ErrorStderr      string `json:"error_stderr,omitempty"`
	RestartEvent     bool   `json:"restart_event"`
	FailureEvent     bool   `json:"failure_event"`
	RecoveryEvent    bool   `json:"recovery_event"`
	LastHeartbeatAt  string `json:"last_heartbeat_at,omitempty"`
	LastTransitionAt string `json:"last_transition_at,omitempty"`
	OccurredAt       string `json:"occurred_at"`
}

type DaemonPluginLifecycleHistoryResponse struct {
	WorkspaceID         string                               `json:"workspace_id"`
	Items               []DaemonPluginLifecycleHistoryRecord `json:"items"`
	HasMore             bool                                 `json:"has_more"`
	NextCursorCreatedAt string                               `json:"next_cursor_created_at,omitempty"`
	NextCursorID        string                               `json:"next_cursor_id,omitempty"`
}
