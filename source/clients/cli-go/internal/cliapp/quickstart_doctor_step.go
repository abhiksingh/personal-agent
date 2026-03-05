package cliapp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"personalagent/runtime/internal/transport"
)

func quickstartRunDoctorStep(ctx context.Context, client *transport.Client, workspaceID string, includeOptional bool) quickstartStep {
	doctorStdout := &bytes.Buffer{}
	doctorStderr := &bytes.Buffer{}
	doctorArgs := []string{
		"--workspace", workspaceID,
		fmt.Sprintf("--include-optional=%t", includeOptional),
		"--strict=false",
	}
	exitCode := runDoctorCommand(ctx, client, doctorArgs, "quickstart.doctor", doctorStdout, doctorStderr)
	if exitCode == 2 {
		return quickstartStep{
			ID:      "readiness.doctor",
			Title:   "Readiness Diagnostics",
			Status:  quickstartStepStatusFail,
			Summary: "Doctor command input validation failed during quickstart.",
			Details: map[string]any{
				"stderr": strings.TrimSpace(doctorStderr.String()),
			},
			Remediation: []string{
				"Rerun quickstart with valid doctor flags.",
			},
		}
	}

	var doctorPayload doctorReport
	if err := json.Unmarshal(doctorStdout.Bytes(), &doctorPayload); err != nil {
		return quickstartStep{
			ID:      "readiness.doctor",
			Title:   "Readiness Diagnostics",
			Status:  quickstartStepStatusFail,
			Summary: "Failed to decode doctor diagnostics payload.",
			Details: map[string]any{
				"error":   err.Error(),
				"payload": strings.TrimSpace(doctorStdout.String()),
			},
			Remediation: []string{
				"Run `personal-agent doctor --workspace <id>` directly and inspect the output.",
			},
		}
	}

	stepStatus := quickstartStepStatusPass
	switch doctorPayload.OverallStatus {
	case doctorCheckStatusFail:
		stepStatus = quickstartStepStatusFail
	case doctorCheckStatusWarn:
		stepStatus = quickstartStepStatusWarn
	default:
		stepStatus = quickstartStepStatusPass
	}

	remediation := []string{}
	seen := map[string]struct{}{}
	for _, check := range doctorPayload.Checks {
		if check.Status != doctorCheckStatusWarn && check.Status != doctorCheckStatusFail {
			continue
		}
		for _, item := range check.Remediation {
			trimmed := strings.TrimSpace(item)
			if trimmed == "" {
				continue
			}
			if _, exists := seen[trimmed]; exists {
				continue
			}
			seen[trimmed] = struct{}{}
			remediation = append(remediation, trimmed)
		}
	}

	summary := "Readiness diagnostics passed."
	if stepStatus == quickstartStepStatusWarn {
		summary = "Readiness diagnostics completed with warnings."
	}
	if stepStatus == quickstartStepStatusFail {
		summary = "Readiness diagnostics found blocking failures."
	}

	return quickstartStep{
		ID:      "readiness.doctor",
		Title:   "Readiness Diagnostics",
		Status:  stepStatus,
		Summary: summary,
		Details: map[string]any{
			"overall_status": doctorPayload.OverallStatus,
			"summary":        doctorPayload.Summary,
			"checks":         doctorPayload.Checks,
		},
		Remediation: remediation,
	}
}
