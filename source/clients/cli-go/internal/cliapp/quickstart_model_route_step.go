package cliapp

import (
	"context"
	"strings"
	"time"

	"personalagent/runtime/internal/providerconfig"
	"personalagent/runtime/internal/transport"
)

func quickstartConfigureModelRouteStep(ctx context.Context, client *transport.Client, input quickstartModelRouteInput) quickstartStep {
	providerName, err := providerconfig.NormalizeProvider(input.Provider)
	if err != nil {
		return quickstartStep{
			ID:      "model.route",
			Title:   "Model Route Selection",
			Status:  quickstartStepStatusFail,
			Summary: "Provider value is invalid for model routing.",
			Details: map[string]any{
				"provider": input.Provider,
				"error":    err.Error(),
			},
			Remediation: []string{
				"Set --provider to one of: openai|anthropic|google|ollama.",
			},
		}
	}

	modelKey := strings.TrimSpace(input.ModelKey)
	if modelKey == "" {
		return quickstartStep{
			ID:      "model.route",
			Title:   "Model Route Selection",
			Status:  quickstartStepStatusFail,
			Summary: "Model key is required for routing policy.",
			Details: map[string]any{
				"provider": providerName,
			},
			Remediation: []string{
				"Set --model explicitly or choose a provider with a quickstart default model.",
			},
		}
	}

	record, selectErr := client.SelectModelRoute(ctx, transport.ModelSelectRequest{
		WorkspaceID: input.WorkspaceID,
		TaskClass:   normalizeTaskClass(input.TaskClass),
		Provider:    providerName,
		ModelKey:    modelKey,
	}, input.CorrelationIDBase+".select")
	if selectErr != nil {
		return quickstartStep{
			ID:      "model.route",
			Title:   "Model Route Selection",
			Status:  quickstartStepStatusFail,
			Summary: "Failed to apply model routing policy.",
			Details: map[string]any{
				"provider":   providerName,
				"model_key":  modelKey,
				"task_class": normalizeTaskClass(input.TaskClass),
				"error":      doctorErrorDetails(selectErr),
			},
			Remediation: quickstartModelRouteFailureRemediation(input.CommandHints, input.WorkspaceID, providerName, normalizeTaskClass(input.TaskClass), modelKey),
		}
	}

	return quickstartStep{
		ID:      "model.route",
		Title:   "Model Route Selection",
		Status:  quickstartStepStatusPass,
		Summary: "Model routing policy applied.",
		Details: map[string]any{
			"workspace_id": record.WorkspaceID,
			"task_class":   record.TaskClass,
			"provider":     record.Provider,
			"model_key":    record.ModelKey,
			"updated_at":   record.UpdatedAt.Format(time.RFC3339Nano),
		},
	}
}
