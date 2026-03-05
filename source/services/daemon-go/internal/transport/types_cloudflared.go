package transport

type CloudflaredVersionRequest struct {
	WorkspaceID string `json:"workspace_id"`
}

type CloudflaredVersionResponse struct {
	WorkspaceID string `json:"workspace_id"`
	Available   bool   `json:"available"`
	BinaryPath  string `json:"binary_path"`
	Version     string `json:"version,omitempty"`
	Stdout      string `json:"stdout,omitempty"`
	Stderr      string `json:"stderr,omitempty"`
	ExitCode    int    `json:"exit_code"`
	DryRun      bool   `json:"dry_run"`
	Error       string `json:"error,omitempty"`
}

type CloudflaredExecRequest struct {
	WorkspaceID string   `json:"workspace_id"`
	Args        []string `json:"args"`
	TimeoutMS   int64    `json:"timeout_ms,omitempty"`
}

type CloudflaredExecResponse struct {
	WorkspaceID string   `json:"workspace_id"`
	Success     bool     `json:"success"`
	BinaryPath  string   `json:"binary_path"`
	Args        []string `json:"args"`
	ExitCode    int      `json:"exit_code"`
	Stdout      string   `json:"stdout,omitempty"`
	Stderr      string   `json:"stderr,omitempty"`
	TimedOut    bool     `json:"timed_out"`
	DurationMS  int64    `json:"duration_ms"`
	DryRun      bool     `json:"dry_run"`
	Error       string   `json:"error,omitempty"`
}
