package finder

import (
	"errors"
	"strconv"
	"strings"

	connectorcontract "personalagent/runtime/internal/connectors/contract"
	shared "personalagent/runtime/internal/shared/contracts"
)

func finderTargetErrorResult(defaultSummary string, err error) (connectorcontract.StepExecutionResult, error) {
	targetErr := finderTargetError{}
	if errors.As(err, &targetErr) {
		summary := strings.TrimSpace(targetErr.Summary)
		if summary == "" {
			summary = strings.TrimSpace(defaultSummary)
		}
		if summary == "" {
			summary = "finder operation failed to resolve target"
		}
		return connectorcontract.StepExecutionResult{
			Status:      shared.TaskStepStatusFailed,
			Summary:     summary,
			Retryable:   false,
			ErrorReason: targetErr.Reason,
		}, err
	}
	summary := strings.TrimSpace(defaultSummary)
	if summary == "" {
		summary = "finder operation failed to resolve target"
	}
	return connectorcontract.StepExecutionResult{
		Status:      shared.TaskStepStatusFailed,
		Summary:     summary,
		Retryable:   false,
		ErrorReason: "invalid_path",
	}, err
}

func applyFinderResolutionEvidence(base map[string]string, resolution finderTargetResolution) map[string]string {
	evidence := cloneStringMap(base)
	if strings.TrimSpace(resolution.ResolvedBy) == "query" {
		evidence["resolved_via"] = "query"
		evidence["query"] = resolution.Query
		evidence["root_path"] = resolution.RootPath
		evidence["match_count"] = strconv.Itoa(len(resolution.Candidates))
		if strings.TrimSpace(resolution.ResolvedPath) != "" {
			evidence["selected_path"] = resolution.ResolvedPath
		}
	} else {
		evidence["resolved_via"] = "path"
	}
	return evidence
}

func applyFinderResolutionOutput(output map[string]any, resolution finderTargetResolution) {
	if output == nil {
		return
	}
	if strings.TrimSpace(resolution.ResolvedBy) == "query" {
		output["resolved_via"] = "query"
		output["query"] = resolution.Query
		output["root_path"] = resolution.RootPath
		output["match_count"] = len(resolution.Candidates)
		output["selected_path"] = resolution.ResolvedPath
		output["matches"] = finderMatchesOutput(resolution.Candidates)
		return
	}
	output["resolved_via"] = "path"
}

func finderMatchesOutput(candidates []finderSearchCandidate) []map[string]any {
	if len(candidates) == 0 {
		return []map[string]any{}
	}
	matches := make([]map[string]any, 0, len(candidates))
	for _, candidate := range candidates {
		matches = append(matches, map[string]any{
			"path":   candidate.Path,
			"score":  candidate.Score,
			"is_dir": candidate.IsDir,
		})
	}
	return matches
}

func cloneStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return map[string]string{}
	}
	result := make(map[string]string, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}
