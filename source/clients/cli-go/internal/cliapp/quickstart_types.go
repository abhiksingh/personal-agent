package cliapp

type quickstartStepStatus string

const (
	quickstartStepStatusPass    quickstartStepStatus = "pass"
	quickstartStepStatusWarn    quickstartStepStatus = "warn"
	quickstartStepStatusFail    quickstartStepStatus = "fail"
	quickstartStepStatusSkipped quickstartStepStatus = "skipped"
)

type quickstartStep struct {
	ID          string               `json:"id"`
	Title       string               `json:"title"`
	Status      quickstartStepStatus `json:"status"`
	Summary     string               `json:"summary"`
	Details     map[string]any       `json:"details,omitempty"`
	Remediation []string             `json:"remediation,omitempty"`
}

type quickstartSummary struct {
	Pass    int `json:"pass"`
	Warn    int `json:"warn"`
	Fail    int `json:"fail"`
	Skipped int `json:"skipped"`
}

type quickstartRemediation struct {
	HumanSummary string   `json:"human_summary"`
	NextSteps    []string `json:"next_steps,omitempty"`
}

type quickstartReport struct {
	SchemaVersion string                     `json:"schema_version"`
	GeneratedAt   string                     `json:"generated_at"`
	WorkspaceID   string                     `json:"workspace_id"`
	ProfileName   string                     `json:"profile_name"`
	Defaults      onboardingDefaultsMetadata `json:"defaults"`
	Provider      string                     `json:"provider,omitempty"`
	ModelKey      string                     `json:"model_key,omitempty"`
	TaskClass     string                     `json:"task_class,omitempty"`
	OverallStatus quickstartStepStatus       `json:"overall_status"`
	Summary       quickstartSummary          `json:"summary"`
	Success       bool                       `json:"success"`
	Steps         []quickstartStep           `json:"steps"`
	Remediation   quickstartRemediation      `json:"remediation"`
}

type quickstartCommandHints struct {
	WorkspaceID   string
	ProfileName   string
	ListenerMode  string
	Address       string
	TokenFilePath string
	ProfileActive bool
}

type quickstartProviderConfigInput struct {
	WorkspaceID       string
	Provider          string
	Endpoint          string
	APIKeySecretName  string
	APIKey            string
	APIKeyFile        string
	CommandHints      quickstartCommandHints
	CorrelationIDBase string
}

type quickstartModelRouteInput struct {
	WorkspaceID       string
	Provider          string
	ModelKey          string
	TaskClass         string
	CommandHints      quickstartCommandHints
	CorrelationIDBase string
}
