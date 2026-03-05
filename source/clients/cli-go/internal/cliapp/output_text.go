package cliapp

import (
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	"personalagent/runtime/internal/transport"
)

func isTextOutputMode() bool {
	return getCLIOutputConfig().outputMode == cliOutputModeText
}

func writeTaskStatusResponse(writer io.Writer, response transport.TaskStatusResponse) int {
	if !isTextOutputMode() {
		return writeJSON(writer, response)
	}

	var builder strings.Builder
	builder.WriteString("task status\n")
	builder.WriteString("-----------\n")
	fmt.Fprintf(&builder, "task_id: %s\n", formatTextValue(response.TaskID, "<unset>"))
	fmt.Fprintf(&builder, "run_id: %s\n", formatTextValue(response.RunID, "<unset>"))
	fmt.Fprintf(&builder, "state: %s\n", formatTextValue(response.State, "<unset>"))
	fmt.Fprintf(&builder, "run_state: %s\n", formatTextValue(response.RunState, "<unset>"))
	if strings.TrimSpace(response.LastError) != "" {
		fmt.Fprintf(&builder, "last_error: %s\n", strings.TrimSpace(response.LastError))
	}
	fmt.Fprintf(
		&builder,
		"actions: cancel=%t retry=%t requeue=%t\n",
		response.Actions.CanCancel,
		response.Actions.CanRetry,
		response.Actions.CanRequeue,
	)
	if !response.UpdatedAt.IsZero() {
		fmt.Fprintf(&builder, "updated_at: %s\n", response.UpdatedAt.UTC().Format(time.RFC3339Nano))
	}
	if strings.TrimSpace(response.CorrelationID) != "" {
		fmt.Fprintf(&builder, "correlation_id: %s\n", strings.TrimSpace(response.CorrelationID))
	}
	return writeTextOutput(writer, builder.String())
}

func writeProviderListResponse(writer io.Writer, response transport.ProviderListResponse) int {
	if !isTextOutputMode() {
		return writeJSON(writer, response)
	}

	providers := append([]transport.ProviderConfigRecord(nil), response.Providers...)
	slices.SortFunc(providers, func(a transport.ProviderConfigRecord, b transport.ProviderConfigRecord) int {
		aProvider := strings.ToLower(strings.TrimSpace(a.Provider))
		bProvider := strings.ToLower(strings.TrimSpace(b.Provider))
		if aProvider < bProvider {
			return -1
		}
		if aProvider > bProvider {
			return 1
		}
		return 0
	})

	var builder strings.Builder
	builder.WriteString("provider list\n")
	builder.WriteString("-------------\n")
	fmt.Fprintf(&builder, "workspace: %s\n", formatTextValue(response.WorkspaceID, "<unset>"))
	fmt.Fprintf(&builder, "providers: %d\n", len(providers))
	for _, provider := range providers {
		fmt.Fprintf(
			&builder,
			"- provider=%s endpoint=%s api_key_secret=%s api_key_configured=%t\n",
			formatTextValue(provider.Provider, "<unset>"),
			formatTextValue(provider.Endpoint, "<unset>"),
			formatTextValue(provider.APIKeySecretName, "<none>"),
			provider.APIKeyConfigured,
		)
	}
	return writeTextOutput(writer, builder.String())
}

func writeModelListResponse(writer io.Writer, response transport.ModelListResponse) int {
	if !isTextOutputMode() {
		return writeJSON(writer, response)
	}

	models := append([]transport.ModelListItem(nil), response.Models...)
	slices.SortFunc(models, func(a transport.ModelListItem, b transport.ModelListItem) int {
		aProvider := strings.ToLower(strings.TrimSpace(a.Provider))
		bProvider := strings.ToLower(strings.TrimSpace(b.Provider))
		if aProvider < bProvider {
			return -1
		}
		if aProvider > bProvider {
			return 1
		}
		aModel := strings.ToLower(strings.TrimSpace(a.ModelKey))
		bModel := strings.ToLower(strings.TrimSpace(b.ModelKey))
		if aModel < bModel {
			return -1
		}
		if aModel > bModel {
			return 1
		}
		return 0
	})

	var builder strings.Builder
	builder.WriteString("model list\n")
	builder.WriteString("----------\n")
	fmt.Fprintf(&builder, "workspace: %s\n", formatTextValue(response.WorkspaceID, "<unset>"))
	fmt.Fprintf(&builder, "models: %d\n", len(models))
	for _, model := range models {
		fmt.Fprintf(
			&builder,
			"- provider=%s model=%s enabled=%t provider_ready=%t endpoint=%s\n",
			formatTextValue(model.Provider, "<unset>"),
			formatTextValue(model.ModelKey, "<unset>"),
			model.Enabled,
			model.ProviderReady,
			formatTextValue(model.ProviderEndpoint, "<unset>"),
		)
	}
	return writeTextOutput(writer, builder.String())
}

func writeDoctorReportResponse(writer io.Writer, report doctorReport) int {
	if !isTextOutputMode() {
		return writeJSON(writer, report)
	}

	var builder strings.Builder
	builder.WriteString("doctor report\n")
	builder.WriteString("-------------\n")
	fmt.Fprintf(&builder, "workspace: %s\n", formatTextValue(report.WorkspaceID, "<unset>"))
	fmt.Fprintf(&builder, "generated_at: %s\n", formatTextValue(report.GeneratedAt, "<unset>"))
	fmt.Fprintf(
		&builder,
		"overall_status: %s (pass=%d warn=%d fail=%d skipped=%d)\n",
		formatTextValue(string(report.OverallStatus), "<unset>"),
		report.Summary.Pass,
		report.Summary.Warn,
		report.Summary.Fail,
		report.Summary.Skipped,
	)
	for _, check := range report.Checks {
		fmt.Fprintf(
			&builder,
			"- [%s] %s: %s\n",
			strings.ToUpper(formatTextValue(string(check.Status), "unknown")),
			formatTextValue(check.ID, "<unset>"),
			formatTextValue(check.Summary, "<none>"),
		)
	}
	return writeTextOutput(writer, builder.String())
}

func writeVersionResponse(writer io.Writer, response cliVersionResponse) int {
	if !isTextOutputMode() {
		return writeJSON(writer, response)
	}

	var builder strings.Builder
	builder.WriteString("personal-agent version\n")
	builder.WriteString("----------------------\n")
	fmt.Fprintf(&builder, "version: %s\n", formatTextValue(response.Version, "<unset>"))
	fmt.Fprintf(&builder, "program: %s\n", formatTextValue(response.Program, "<unset>"))
	fmt.Fprintf(&builder, "go_version: %s\n", formatTextValue(response.GoVersion, "<unset>"))
	fmt.Fprintf(&builder, "platform: %s\n", formatTextValue(response.Platform, "<unset>"))
	if strings.TrimSpace(response.Commit) != "" {
		fmt.Fprintf(&builder, "commit: %s\n", strings.TrimSpace(response.Commit))
	}
	if strings.TrimSpace(response.BuiltAt) != "" {
		fmt.Fprintf(&builder, "built_at: %s\n", strings.TrimSpace(response.BuiltAt))
	}
	if strings.TrimSpace(response.VCSRevision) != "" {
		fmt.Fprintf(&builder, "vcs_revision: %s\n", strings.TrimSpace(response.VCSRevision))
	}
	if strings.TrimSpace(response.VCSTime) != "" {
		fmt.Fprintf(&builder, "vcs_time: %s\n", strings.TrimSpace(response.VCSTime))
	}
	if response.VCSModified != nil {
		fmt.Fprintf(&builder, "vcs_modified: %t\n", *response.VCSModified)
	}
	return writeTextOutput(writer, builder.String())
}

func writeTextOutput(writer io.Writer, value string) int {
	trimmed := strings.TrimRight(value, "\n")
	if trimmed == "" {
		trimmed = "<empty>"
	}
	if _, err := io.WriteString(writer, trimmed+"\n"); err != nil {
		return 1
	}
	return 0
}

func formatTextValue(value string, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}
