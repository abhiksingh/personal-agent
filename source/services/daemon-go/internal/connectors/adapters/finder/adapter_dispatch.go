package finder

import (
	"fmt"
	"strconv"
	"strings"

	adapterhelpers "personalagent/runtime/internal/connectors/adapters/helpers"
	connectorcontract "personalagent/runtime/internal/connectors/contract"
	shared "personalagent/runtime/internal/shared/contracts"
)

func (a *Adapter) executeUncached(execCtx connectorcontract.ExecutionContext, step connectorcontract.TaskStep) (connectorcontract.StepExecutionResult, error) {
	switch step.CapabilityKey {
	case CapabilityFind:
		finderInput, err := resolveFinderInput(step.Input)
		if err != nil {
			return connectorcontract.StepExecutionResult{
				Status:      shared.TaskStepStatusFailed,
				Summary:     "finder find input missing",
				Retryable:   false,
				ErrorReason: "invalid_query",
			}, err
		}
		if strings.TrimSpace(finderInput.Query) == "" {
			return connectorcontract.StepExecutionResult{
				Status:      shared.TaskStepStatusFailed,
				Summary:     "finder query is required",
				Retryable:   false,
				ErrorReason: "invalid_query",
			}, fmt.Errorf("finder find query is required")
		}
		candidates, resolvedRoot, searchErr := semanticFindCandidates(finderInput.Query, finderInput.RootPath)
		if searchErr != nil {
			return connectorcontract.StepExecutionResult{
				Status:      shared.TaskStepStatusFailed,
				Summary:     "finder search failed",
				Retryable:   false,
				ErrorReason: "search_failed",
				Evidence: map[string]string{
					"query": finderInput.Query,
				},
			}, searchErr
		}
		selectedPath := ""
		if len(candidates) > 0 {
			selectedPath = candidates[0].Path
		}
		summary := "finder found no matching paths"
		if len(candidates) > 0 {
			summary = "finder found matching paths"
		}
		return connectorcontract.StepExecutionResult{
			Status:    shared.TaskStepStatusCompleted,
			Summary:   summary,
			Retryable: false,
			Evidence: map[string]string{
				"query":         finderInput.Query,
				"root_path":     resolvedRoot,
				"match_count":   strconv.Itoa(len(candidates)),
				"selected_path": selectedPath,
			},
			Output: map[string]any{
				"query":         finderInput.Query,
				"root_path":     resolvedRoot,
				"match_count":   len(candidates),
				"selected_path": selectedPath,
				"matches":       finderMatchesOutput(candidates),
			},
		}, nil
	case CapabilityList:
		resolution, inputErr := resolveFinderTarget(step.CapabilityKey, step.Input)
		if inputErr != nil {
			return finderTargetErrorResult("finder list failed to resolve target", inputErr)
		}
		targetPath := resolution.ResolvedPath
		fileCount, exists, listErr := listPath(targetPath)
		if listErr != nil {
			return connectorcontract.StepExecutionResult{
				Status:      shared.TaskStepStatusFailed,
				Summary:     "finder list failed",
				Retryable:   true,
				ErrorReason: "io_error",
				Evidence: map[string]string{
					"path": targetPath,
				},
			}, listErr
		}
		output := map[string]any{"path": targetPath, "file_count": fileCount, "exists": exists}
		applyFinderResolutionOutput(output, resolution)
		return connectorcontract.StepExecutionResult{
			Status:    shared.TaskStepStatusCompleted,
			Summary:   "finder listed target path",
			Retryable: false,
			Evidence: applyFinderResolutionEvidence(map[string]string{
				"path":       targetPath,
				"file_count": strconv.Itoa(fileCount),
				"exists":     strconv.FormatBool(exists),
			}, resolution),
			Output: output,
		}, nil
	case CapabilityPreview:
		resolution, inputErr := resolveFinderTarget(step.CapabilityKey, step.Input)
		if inputErr != nil {
			return finderTargetErrorResult("finder preview failed to resolve target", inputErr)
		}
		targetPath := resolution.ResolvedPath
		info, exists, previewErr := previewPath(targetPath)
		if previewErr != nil {
			return connectorcontract.StepExecutionResult{
				Status:      shared.TaskStepStatusFailed,
				Summary:     "finder preview failed",
				Retryable:   true,
				ErrorReason: "io_error",
				Evidence: map[string]string{
					"path": targetPath,
				},
			}, previewErr
		}
		previewID := "finder-preview-" + adapterhelpers.StableStepToken(execCtx, step)
		output := map[string]any{
			"path":       targetPath,
			"exists":     exists,
			"is_dir":     info.IsDir,
			"size_bytes": info.SizeBytes,
		}
		applyFinderResolutionOutput(output, resolution)
		return connectorcontract.StepExecutionResult{
			Status:    shared.TaskStepStatusCompleted,
			Summary:   "finder previewed destructive delete",
			Retryable: false,
			Evidence: applyFinderResolutionEvidence(map[string]string{
				"preview_id": previewID,
				"path":       targetPath,
				"exists":     strconv.FormatBool(exists),
				"is_dir":     strconv.FormatBool(info.IsDir),
				"size_bytes": strconv.FormatInt(info.SizeBytes, 10),
			}, resolution),
			Output: output,
		}, nil
	case CapabilityDelete:
		resolution, inputErr := resolveFinderTarget(step.CapabilityKey, step.Input)
		if inputErr != nil {
			return finderTargetErrorResult("finder delete failed to resolve target", inputErr)
		}
		targetPath := resolution.ResolvedPath
		if err := guardDeletePath(targetPath); err != nil {
			return connectorcontract.StepExecutionResult{
				Status:      shared.TaskStepStatusFailed,
				Summary:     "finder delete blocked by guardrail",
				Retryable:   false,
				ErrorReason: "guardrail_denied",
				Evidence: map[string]string{
					"path": targetPath,
				},
			}, err
		}
		existed, err := deletePath(targetPath)
		if err != nil {
			return connectorcontract.StepExecutionResult{
				Status:      shared.TaskStepStatusFailed,
				Summary:     "finder delete failed",
				Retryable:   true,
				ErrorReason: "io_error",
				Evidence: map[string]string{
					"path": targetPath,
				},
			}, err
		}
		output := map[string]any{"deleted_path": targetPath, "existed": existed}
		applyFinderResolutionOutput(output, resolution)
		return connectorcontract.StepExecutionResult{
			Status:    shared.TaskStepStatusCompleted,
			Summary:   "finder deleted target path",
			Retryable: false,
			Evidence: applyFinderResolutionEvidence(map[string]string{
				"delete_id":    "finder-delete-" + adapterhelpers.StableStepToken(execCtx, step),
				"deleted_path": targetPath,
				"existed":      strconv.FormatBool(existed),
			}, resolution),
			Output: output,
		}, nil
	default:
		return connectorcontract.StepExecutionResult{
			Status:      shared.TaskStepStatusFailed,
			Summary:     "unsupported finder capability",
			Retryable:   false,
			ErrorReason: "unsupported_capability",
		}, fmt.Errorf("unsupported finder capability: %s", step.CapabilityKey)
	}
}
