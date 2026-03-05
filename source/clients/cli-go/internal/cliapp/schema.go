package cliapp

import (
	"sort"
	"strings"

	"personalagent/runtime/internal/transport"
)

type cliFlagSchema struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Default     string `json:"default,omitempty"`
	Required    bool   `json:"required"`
}

type cliCommandSchema struct {
	Name              string             `json:"name"`
	Summary           string             `json:"summary"`
	RequiresDaemon    bool               `json:"requires_daemon"`
	RequiredFlags     []string           `json:"required_flags,omitempty"`
	Subcommands       []cliCommandSchema `json:"subcommands,omitempty"`
	SupportsStreaming bool               `json:"supports_streaming,omitempty"`
	MachineOutputSafe bool               `json:"machine_output_safe"`
}

type cliSchemaDocument struct {
	SchemaVersion    string             `json:"schema_version"`
	Program          string             `json:"program"`
	OutputModes      []string           `json:"output_modes"`
	ErrorOutputModes []string           `json:"error_output_modes"`
	GlobalFlags      []cliFlagSchema    `json:"global_flags"`
	Commands         []cliCommandSchema `json:"commands"`
}

func buildCLISchemaDocument() cliSchemaDocument {
	registry := cliRootCommandRegistry()
	catalog := cliSchemaCommandCatalog()
	return cliSchemaDocument{
		SchemaVersion:    "1.0.0",
		Program:          "personal-agent",
		OutputModes:      []string{string(cliOutputModeJSON), string(cliOutputModeJSONCompact), string(cliOutputModeText)},
		ErrorOutputModes: []string{string(cliErrorOutputModeText), string(cliErrorOutputModeJSON)},
		GlobalFlags: []cliFlagSchema{
			{Name: "--mode", Type: "string", Description: "transport mode: tcp|unix|named_pipe", Default: "tcp"},
			{Name: "--address", Type: "string", Description: "transport address", Default: transport.DefaultTCPAddress},
			{Name: "--runtime-profile", Type: "string", Description: "runtime profile: local|prod", Default: cliRuntimeProfileLocal},
			{Name: "--auth-token", Type: "string", Description: "bearer auth token", Default: defaultCLIControlAuthToken},
			{Name: "--auth-token-file", Type: "string", Description: "path to file containing bearer auth token"},
			{Name: "--db", Type: "string", Description: "SQLite path for local CLI state"},
			{Name: "--correlation-id", Type: "string", Description: "optional correlation id"},
			{Name: "--output", Type: "string", Description: "output mode: json|json-compact|text", Default: string(cliOutputModeJSON)},
			{Name: "--error-output", Type: "string", Description: "error output mode", Default: string(cliErrorOutputModeText)},
			{Name: "--timeout", Type: "duration", Description: "request timeout", Default: "10s"},
			{Name: "--tls-ca-file", Type: "string", Description: "CA bundle PEM for daemon TLS certificate verification"},
			{Name: "--tls-client-cert-file", Type: "string", Description: "client certificate PEM for daemon mTLS auth"},
			{Name: "--tls-client-key-file", Type: "string", Description: "client private key PEM for daemon mTLS auth"},
			{Name: "--tls-server-name", Type: "string", Description: "TLS server-name override for daemon certificate verification"},
			{Name: "--tls-insecure-skip-verify", Type: "bool", Description: "skip daemon TLS certificate verification (testing only)", Default: "false"},
		},
		Commands: buildSchemaCommandsFromRegistry(registry, catalog),
	}
}

func buildSchemaCommandsFromRegistry(registry map[string]cliRootCommand, catalog map[string]cliCommandSchema) []cliCommandSchema {
	names := sortedRootCommandNames(registry)
	commands := make([]cliCommandSchema, 0, len(names))
	for _, name := range names {
		command, found := catalog[name]
		if !found {
			command = cliCommandSchema{
				Name:              name,
				Summary:           "Command metadata unavailable.",
				RequiresDaemon:    false,
				MachineOutputSafe: true,
			}
		}
		command.Name = name
		command.Subcommands = normalizeSchemaSubcommands(command.Subcommands)
		commands = append(commands, command)
	}
	return commands
}

func normalizeSchemaSubcommands(subcommands []cliCommandSchema) []cliCommandSchema {
	normalized := append([]cliCommandSchema(nil), subcommands...)
	for index := range normalized {
		normalized[index].Subcommands = normalizeSchemaSubcommands(normalized[index].Subcommands)
	}
	sort.Slice(normalized, func(i int, j int) bool {
		return strings.ToLower(strings.TrimSpace(normalized[i].Name)) < strings.ToLower(strings.TrimSpace(normalized[j].Name))
	})
	return normalized
}

func cliSchemaCommandCatalog() map[string]cliCommandSchema {
	streamCommand := schemaCommand("stream", "Realtime event stream", true, nil)
	streamCommand.SupportsStreaming = true
	chatCommand := schemaCommand("chat", "Chat turn operations", true, nil)
	chatCommand.SupportsStreaming = true
	chatCommand.MachineOutputSafe = false

	return map[string]cliCommandSchema{
		"agent": schemaCommand("agent", "Agent run/approve orchestration", true, nil,
			schemaCommand("approve", "Submit approval decision for a pending run", true, []string{"--workspace", "--approval-id", "--phrase", "--actor-id"}),
			schemaCommand("run", "Run an agent instruction request", true, []string{"--workspace", "--request"}),
		),
		"assistant": schemaCommand("assistant", "Interactive assistant workflows (setup/task/approval/comm)", true, nil),
		"auth": schemaCommand("auth", "Control token bootstrap/rotation", false, nil,
			schemaCommand("bootstrap", "Bootstrap daemon control token file", false, []string{"--file"}),
			schemaCommand("bootstrap-local-dev", "Bootstrap local-dev token/profile defaults", false, nil),
			schemaCommand("rotate", "Rotate daemon control token file", false, []string{"--file"}),
		),
		"automation": schemaCommand("automation", "Automation create/list/run operations", true, nil,
			schemaCommand("create", "Create automation trigger", true, []string{"--workspace", "--subject", "--trigger-type"}),
			schemaCommand("list", "List automation triggers", true, nil),
			schemaCommand("run", "Trigger automation execution paths", true, nil,
				schemaCommand("comm-event", "Run ON_COMM_EVENT automation evaluation", true, []string{"--event-id"}),
				schemaCommand("schedule", "Run SCHEDULE automation evaluation", true, nil),
			),
		),
		"channel": schemaCommand("channel", "Channel operations and mappings", true, nil,
			schemaCommand("mapping", "Manage channel-to-connector mappings", true, nil,
				schemaCommand("disable", "Disable channel mapping entry", true, []string{"--workspace", "--channel", "--connector"}),
				schemaCommand("enable", "Enable channel mapping entry", true, []string{"--workspace", "--channel", "--connector"}),
				schemaCommand("list", "List channel mapping entries", true, nil),
				schemaCommand("prioritize", "Update channel mapping priority", true, []string{"--workspace", "--channel", "--connector", "--priority"}),
			),
			schemaCommand("messages", "Messages channel operations", true, nil,
				schemaCommand("ingest", "Poll/ingest iMessage channel events", true, nil),
			),
		),
		"chat": chatCommand,
		"comm": schemaCommand("comm", "Communication send/attempt/policy workflows", true, nil,
			schemaCommand("attempts", "List delivery attempts for operation", true, []string{"--operation-id"}),
			schemaCommand("policy", "Manage communication policy rules", true, nil,
				schemaCommand("list", "List policy rules", true, nil),
				schemaCommand("set", "Set policy rule", true, nil),
			),
			schemaCommand("send", "Send communication through channel routing", true, []string{"--destination"}),
		),
		"completion": schemaCommand("completion", "Generate shell completion scripts (bash|zsh|fish)", false, nil,
			schemaCommand("bash", "Emit bash completion script", false, nil),
			schemaCommand("fish", "Emit fish completion script", false, nil),
			schemaCommand("zsh", "Emit zsh completion script", false, nil),
		),
		"connector": schemaCommand("connector", "Connector operations (smoke/bridge/twilio/mail/calendar/browser/cloudflared)", true, nil,
			schemaCommand("bridge", "Local ingest bridge helper commands", false, nil,
				schemaCommand("setup", "Ensure local ingest bridge directories", false, nil),
				schemaCommand("status", "Inspect local ingest bridge readiness", false, nil),
			),
			schemaCommand("browser", "Browser connector commands", false, nil,
				schemaCommand("handoff", "Queue browser handoff event via local bridge", false, nil),
				schemaCommand("ingest", "Ingest browser event through daemon", true, nil),
			),
			schemaCommand("calendar", "Calendar connector commands", false, nil,
				schemaCommand("handoff", "Queue calendar handoff event via local bridge", false, nil),
				schemaCommand("ingest", "Ingest calendar event through daemon", true, nil),
			),
			schemaCommand("cloudflared", "Cloudflared connector daemon commands", true, nil,
				schemaCommand("exec", "Proxy cloudflared CLI exec command", true, []string{"--arg"}),
				schemaCommand("version", "Read cloudflared runtime version", true, nil),
			),
			schemaCommand("mail", "Mail connector commands", false, nil,
				schemaCommand("handoff", "Queue mail handoff event via local bridge", false, nil),
				schemaCommand("ingest", "Ingest mail event through daemon", true, nil),
			),
			schemaCommand("smoke", "Run local connector smoke checks", false, nil),
			schemaCommand("twilio", "Twilio connector/channel commands", true, nil,
				schemaCommand("call-status", "Read Twilio call status", true, nil),
				schemaCommand("check", "Check Twilio connectivity", true, nil),
				schemaCommand("get", "Get Twilio connector configuration metadata", true, nil),
				schemaCommand("ingest-sms", "Ingest Twilio SMS webhook payload", true, nil),
				schemaCommand("ingest-voice", "Ingest Twilio voice webhook payload", true, nil),
				schemaCommand("set", "Set Twilio connector configuration", true, nil),
				schemaCommand("sms-chat", "Send conversational Twilio SMS chat turn", true, nil),
				schemaCommand("start-call", "Start outbound Twilio voice call", true, nil),
				schemaCommand("transcript", "Read Twilio call transcript", true, nil),
				schemaCommand("webhook", "Manage Twilio webhook runtime", true, nil,
					schemaCommand("replay", "Replay persisted webhook event fixture(s)", true, nil),
					schemaCommand("serve", "Start local Twilio webhook listener runtime", true, nil),
				),
			),
		),
		"context": schemaCommand("context", "Context budget and memory-retrieval introspection", true, nil,
			schemaCommand("samples", "Read context telemetry samples", true, []string{"--workspace", "--task-class"}),
			schemaCommand("tune", "Tune context retrieval multiplier", true, []string{"--workspace", "--task-class"}),
		),
		"delegation": schemaCommand("delegation", "Delegation rule management", true, nil,
			schemaCommand("check", "Check delegation authorization decision", true, []string{"--workspace", "--requested-by", "--acting-as"}),
			schemaCommand("grant", "Create delegation rule", true, []string{"--workspace", "--from", "--to"}),
			schemaCommand("list", "List delegation rules", true, nil),
			schemaCommand("revoke", "Revoke delegation rule", true, []string{"--workspace", "--rule-id"}),
		),
		"doctor": schemaCommand("doctor", "Workspace readiness diagnostics report (machine-readable JSON)", true, nil),
		"help":   schemaCommand("help", "Print skim-first usage and command reference", false, nil),
		"identity": schemaCommand("identity", "Workspace/principal/session identity operations", true, nil,
			schemaCommand("bootstrap", "Bootstrap workspace/principal identity defaults", true, []string{"--workspace", "--principal"}),
			schemaCommand("context", "Get active identity context", true, nil),
			schemaCommand("devices", "List identity devices", true, nil),
			schemaCommand("principals", "List workspace principals", true, nil),
			schemaCommand("revoke-session", "Revoke identity session", true, []string{"--session-id"}),
			schemaCommand("select-workspace", "Select active workspace context", true, []string{"--workspace"}),
			schemaCommand("sessions", "List identity sessions", true, nil),
			schemaCommand("workspaces", "List available workspaces", true, nil),
		),
		"inspect": schemaCommand("inspect", "Execution inspection queries", true, nil,
			schemaCommand("memory", "Inspect retained memory entries", true, nil),
			schemaCommand("run", "Inspect specific run details", true, []string{"--run-id"}),
			schemaCommand("transcript", "Inspect communication transcript entries", true, nil),
		),
		"meta": schemaCommand("meta", "CLI metadata and schema outputs", false, nil,
			schemaCommand("capabilities", "Query daemon runtime discovery capabilities JSON", true, nil),
			schemaCommand("schema", "Emit deterministic command/flag schema JSON", false, nil),
		),
		"model": schemaCommand("model", "Model catalog and selection policy controls", true, nil,
			schemaCommand("add", "Add model catalog entry", true, []string{"--workspace", "--provider", "--model"}),
			schemaCommand("disable", "Disable model entry", true, []string{"--workspace", "--provider", "--model"}),
			schemaCommand("discover", "Discover provider-backed model catalog", true, nil),
			schemaCommand("enable", "Enable model entry", true, []string{"--workspace", "--provider", "--model"}),
			schemaCommand("list", "List model catalog entries", true, nil),
			schemaCommand("policy", "Read model routing policy", true, nil),
			schemaCommand("remove", "Remove model catalog entry", true, []string{"--workspace", "--provider", "--model"}),
			schemaCommand("resolve", "Resolve selected route for task class", true, nil),
			schemaCommand("select", "Select route for task class", true, []string{"--workspace", "--task-class", "--provider", "--model"}),
		),
		"profile": schemaCommand("profile", "Manage local CLI endpoint/auth/workspace profiles", false, nil,
			schemaCommand("active", "Inspect active profile resolution state", false, nil),
			schemaCommand("delete", "Delete named profile", false, []string{"--name"}),
			schemaCommand("get", "Get active or named profile", false, nil),
			schemaCommand("list", "List saved profiles", false, nil),
			schemaCommand("rename", "Rename a saved profile", false, []string{"--name", "--to"}),
			schemaCommand("set", "Create/update named profile", false, []string{"--name"}),
			schemaCommand("use", "Activate named profile", false, []string{"--name"}),
		),
		"provider": schemaCommand("provider", "Provider configuration and connectivity checks", true, nil,
			schemaCommand("check", "Check provider connectivity", true, nil),
			schemaCommand("list", "List provider config status", true, nil),
			schemaCommand("set", "Set provider configuration", true, []string{"--workspace", "--provider"}),
		),
		"quickstart": schemaCommand("quickstart", "Guided first-run setup with readiness diagnostics", false, nil),
		"retention": schemaCommand("retention", "Retention purge/compaction operations", true, nil,
			schemaCommand("compact-memory", "Compact stale memory records", true, []string{"--workspace", "--owner"}),
			schemaCommand("purge", "Purge trace/transcript/memory retention windows", true, nil),
		),
		"secret": schemaCommand("secret", "SecretRef metadata + secure-store registration commands", true, nil,
			schemaCommand("delete", "Delete secret value and registered reference", true, []string{"--workspace", "--name"}),
			schemaCommand("get", "Get registered secret reference metadata", true, []string{"--workspace", "--name"}),
			schemaCommand("set", "Set secret value and register reference", true, []string{"--workspace", "--name", "--value|--file"}),
		),
		"smoke":  schemaCommand("smoke", "Daemon capability smoke check", true, nil),
		"stream": streamCommand,
		"task": schemaCommand("task", "Task submit/status/cancel/retry/requeue operations", true, nil,
			schemaCommand("cancel", "Cancel a queued/running task run", true, []string{"--task-id|--run-id"}),
			schemaCommand("requeue", "Requeue a pending task run as a fresh queued run", true, []string{"--task-id|--run-id"}),
			schemaCommand("retry", "Retry a failed/cancelled task run", true, []string{"--task-id|--run-id"}),
			schemaCommand("status", "Get task status", true, []string{"--task-id"}),
			schemaCommand("submit", "Submit a task", true, []string{"--workspace", "--requested-by", "--subject", "--title"}),
		),
		"version": schemaCommand("version", "Emit CLI build/version metadata", false, nil),
	}
}

func schemaCommand(name string, summary string, requiresDaemon bool, requiredFlags []string, subcommands ...cliCommandSchema) cliCommandSchema {
	command := cliCommandSchema{
		Name:              name,
		Summary:           summary,
		RequiresDaemon:    requiresDaemon,
		MachineOutputSafe: true,
	}
	if len(requiredFlags) > 0 {
		command.RequiredFlags = append([]string(nil), requiredFlags...)
	}
	if len(subcommands) > 0 {
		command.Subcommands = append([]cliCommandSchema(nil), subcommands...)
	}
	return command
}
