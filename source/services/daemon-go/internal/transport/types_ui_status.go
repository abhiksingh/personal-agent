package transport

type ChannelStatusRequest struct {
	WorkspaceID string `json:"workspace_id"`
}

type ChannelConnectorMappingListRequest struct {
	WorkspaceID string `json:"workspace_id"`
	ChannelID   string `json:"channel_id,omitempty"`
}

type ChannelConnectorMappingRecord struct {
	ChannelID    string   `json:"channel_id"`
	ConnectorID  string   `json:"connector_id"`
	Enabled      bool     `json:"enabled"`
	Priority     int      `json:"priority"`
	Capabilities []string `json:"capabilities,omitempty"`
	CreatedAt    string   `json:"created_at,omitempty"`
	UpdatedAt    string   `json:"updated_at,omitempty"`
}

type ChannelConnectorMappingListResponse struct {
	WorkspaceID    string                          `json:"workspace_id"`
	ChannelID      string                          `json:"channel_id,omitempty"`
	FallbackPolicy string                          `json:"fallback_policy"`
	Bindings       []ChannelConnectorMappingRecord `json:"bindings"`
}

type ChannelConnectorMappingUpsertRequest struct {
	WorkspaceID    string `json:"workspace_id"`
	ChannelID      string `json:"channel_id"`
	ConnectorID    string `json:"connector_id"`
	Enabled        bool   `json:"enabled"`
	Priority       int    `json:"priority,omitempty"`
	FallbackPolicy string `json:"fallback_policy,omitempty"`
}

type ChannelConnectorMappingUpsertResponse struct {
	WorkspaceID    string                          `json:"workspace_id"`
	ChannelID      string                          `json:"channel_id"`
	ConnectorID    string                          `json:"connector_id"`
	Enabled        bool                            `json:"enabled"`
	Priority       int                             `json:"priority"`
	FallbackPolicy string                          `json:"fallback_policy"`
	UpdatedAt      string                          `json:"updated_at"`
	Bindings       []ChannelConnectorMappingRecord `json:"bindings"`
}

type PluginWorkerStatusCard struct {
	PluginID           string `json:"plugin_id"`
	Kind               string `json:"kind"`
	State              string `json:"state"`
	ProcessID          int    `json:"process_id"`
	RestartCount       int    `json:"restart_count"`
	LastError          string `json:"last_error,omitempty"`
	LastErrorSource    string `json:"last_error_source,omitempty"`
	LastErrorOperation string `json:"last_error_operation,omitempty"`
	LastErrorStderr    string `json:"last_error_stderr,omitempty"`
	LastHeartbeat      string `json:"last_heartbeat,omitempty"`
	LastTransition     string `json:"last_transition,omitempty"`
}

type ConfigFieldDescriptor struct {
	Key         string   `json:"key"`
	Label       string   `json:"label"`
	Type        string   `json:"type"`
	Required    bool     `json:"required"`
	EnumOptions []string `json:"enum_options"`
	Editable    bool     `json:"editable"`
	Secret      bool     `json:"secret"`
	WriteOnly   bool     `json:"write_only"`
	HelpText    string   `json:"help_text"`
}

type ActionReadinessBlocker struct {
	Code              string `json:"code"`
	Message           string `json:"message"`
	RemediationAction string `json:"remediation_action,omitempty"`
}

type ChannelStatusCard struct {
	ChannelID              string                         `json:"channel_id"`
	DisplayName            string                         `json:"display_name"`
	Category               string                         `json:"category"`
	Enabled                bool                           `json:"enabled"`
	Configured             bool                           `json:"configured"`
	Status                 string                         `json:"status"`
	Summary                string                         `json:"summary,omitempty"`
	Configuration          map[string]any                 `json:"configuration,omitempty"`
	ConfigFieldDescriptors []ConfigFieldDescriptor        `json:"config_field_descriptors,omitempty"`
	Capabilities           []string                       `json:"capabilities,omitempty"`
	ActionReadiness        string                         `json:"action_readiness,omitempty"`
	ActionBlockers         []ActionReadinessBlocker       `json:"action_blockers,omitempty"`
	RemediationActions     []DiagnosticsRemediationAction `json:"remediation_actions"`
	Worker                 *PluginWorkerStatusCard        `json:"worker,omitempty"`
}

type ChannelStatusResponse struct {
	WorkspaceID string              `json:"workspace_id"`
	Channels    []ChannelStatusCard `json:"channels"`
}

type ChannelDiagnosticsRequest struct {
	WorkspaceID string `json:"workspace_id"`
	ChannelID   string `json:"channel_id,omitempty"`
}

type WorkerHealthSnapshot struct {
	Registered bool                    `json:"registered"`
	Worker     *PluginWorkerStatusCard `json:"worker,omitempty"`
}

type DiagnosticsRemediationAction struct {
	Identifier  string            `json:"identifier"`
	Label       string            `json:"label"`
	Intent      string            `json:"intent"`
	Destination string            `json:"destination,omitempty"`
	Parameters  map[string]string `json:"parameters,omitempty"`
	Enabled     bool              `json:"enabled"`
	Recommended bool              `json:"recommended"`
	Reason      string            `json:"reason,omitempty"`
}

type ChannelDiagnosticsSummary struct {
	ChannelID          string                         `json:"channel_id"`
	DisplayName        string                         `json:"display_name"`
	Category           string                         `json:"category"`
	Configured         bool                           `json:"configured"`
	Status             string                         `json:"status"`
	Summary            string                         `json:"summary,omitempty"`
	WorkerHealth       WorkerHealthSnapshot           `json:"worker_health"`
	RemediationActions []DiagnosticsRemediationAction `json:"remediation_actions"`
}

type ChannelDiagnosticsResponse struct {
	WorkspaceID string                      `json:"workspace_id"`
	Diagnostics []ChannelDiagnosticsSummary `json:"diagnostics"`
}

type ConnectorStatusRequest struct {
	WorkspaceID string `json:"workspace_id"`
}

type ConnectorStatusCard struct {
	ConnectorID            string                         `json:"connector_id"`
	PluginID               string                         `json:"plugin_id"`
	DisplayName            string                         `json:"display_name"`
	Enabled                bool                           `json:"enabled"`
	Configured             bool                           `json:"configured"`
	Status                 string                         `json:"status"`
	Summary                string                         `json:"summary,omitempty"`
	Configuration          map[string]any                 `json:"configuration,omitempty"`
	ConfigFieldDescriptors []ConfigFieldDescriptor        `json:"config_field_descriptors,omitempty"`
	Capabilities           []string                       `json:"capabilities,omitempty"`
	ActionReadiness        string                         `json:"action_readiness,omitempty"`
	ActionBlockers         []ActionReadinessBlocker       `json:"action_blockers,omitempty"`
	RemediationActions     []DiagnosticsRemediationAction `json:"remediation_actions"`
	Worker                 *PluginWorkerStatusCard        `json:"worker,omitempty"`
}

type ConnectorStatusResponse struct {
	WorkspaceID string                `json:"workspace_id"`
	Connectors  []ConnectorStatusCard `json:"connectors"`
}

type ConnectorDiagnosticsRequest struct {
	WorkspaceID string `json:"workspace_id"`
	ConnectorID string `json:"connector_id,omitempty"`
}

type ConnectorDiagnosticsSummary struct {
	ConnectorID        string                         `json:"connector_id"`
	PluginID           string                         `json:"plugin_id"`
	DisplayName        string                         `json:"display_name"`
	Configured         bool                           `json:"configured"`
	Status             string                         `json:"status"`
	Summary            string                         `json:"summary,omitempty"`
	WorkerHealth       WorkerHealthSnapshot           `json:"worker_health"`
	RemediationActions []DiagnosticsRemediationAction `json:"remediation_actions"`
}

type ConnectorDiagnosticsResponse struct {
	WorkspaceID string                        `json:"workspace_id"`
	Diagnostics []ConnectorDiagnosticsSummary `json:"diagnostics"`
}

type ConnectorPermissionRequest struct {
	WorkspaceID string `json:"workspace_id"`
	ConnectorID string `json:"connector_id"`
}

type ConnectorPermissionResponse struct {
	WorkspaceID     string `json:"workspace_id"`
	ConnectorID     string `json:"connector_id"`
	PermissionState string `json:"permission_state"`
	Message         string `json:"message,omitempty"`
}

type ChannelConfigUpsertRequest struct {
	WorkspaceID   string                `json:"workspace_id"`
	ChannelID     string                `json:"channel_id"`
	Configuration UIStatusConfiguration `json:"configuration"`
	Merge         bool                  `json:"merge,omitempty"`
}

type ChannelConfigUpsertResponse struct {
	WorkspaceID   string                `json:"workspace_id"`
	ChannelID     string                `json:"channel_id"`
	Configuration UIStatusConfiguration `json:"configuration"`
	UpdatedAt     string                `json:"updated_at"`
}

type ConnectorConfigUpsertRequest struct {
	WorkspaceID   string                `json:"workspace_id"`
	ConnectorID   string                `json:"connector_id"`
	Configuration UIStatusConfiguration `json:"configuration"`
	Merge         bool                  `json:"merge,omitempty"`
}

type ConnectorConfigUpsertResponse struct {
	WorkspaceID   string                `json:"workspace_id"`
	ConnectorID   string                `json:"connector_id"`
	Configuration UIStatusConfiguration `json:"configuration"`
	UpdatedAt     string                `json:"updated_at"`
}

type ChannelTestOperationRequest struct {
	WorkspaceID string `json:"workspace_id"`
	ChannelID   string `json:"channel_id"`
	Operation   string `json:"operation,omitempty"`
}

type ChannelTestOperationResponse struct {
	WorkspaceID string                       `json:"workspace_id"`
	ChannelID   string                       `json:"channel_id"`
	Operation   string                       `json:"operation"`
	Success     bool                         `json:"success"`
	Status      string                       `json:"status"`
	Summary     string                       `json:"summary"`
	CheckedAt   string                       `json:"checked_at"`
	Details     UIStatusTestOperationDetails `json:"details,omitempty"`
}

type ConnectorTestOperationRequest struct {
	WorkspaceID string `json:"workspace_id"`
	ConnectorID string `json:"connector_id"`
	Operation   string `json:"operation,omitempty"`
}

type ConnectorTestOperationResponse struct {
	WorkspaceID string                       `json:"workspace_id"`
	ConnectorID string                       `json:"connector_id"`
	Operation   string                       `json:"operation"`
	Success     bool                         `json:"success"`
	Status      string                       `json:"status"`
	Summary     string                       `json:"summary"`
	CheckedAt   string                       `json:"checked_at"`
	Details     UIStatusTestOperationDetails `json:"details,omitempty"`
}
