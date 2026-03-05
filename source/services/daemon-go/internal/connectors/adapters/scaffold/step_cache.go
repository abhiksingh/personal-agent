package scaffold

import (
	"fmt"
	"path/filepath"
	"strings"

	localstate "personalagent/runtime/internal/connectors/adapters/localstate"
	connectorcontract "personalagent/runtime/internal/connectors/contract"
)

// ExecuteStepWithCache applies the standard adapter step-result cache lifecycle:
// resolve step-result path, load cached result, execute uncached path, then save.
func ExecuteStepWithCache(
	adapterDomain string,
	execCtx connectorcontract.ExecutionContext,
	step connectorcontract.TaskStep,
	executeUncached func(stepResultPath string) (connectorcontract.StepExecutionResult, error),
) (connectorcontract.StepExecutionResult, error) {
	if strings.TrimSpace(adapterDomain) == "" {
		return connectorcontract.StepExecutionResult{}, fmt.Errorf("adapter domain is required")
	}
	if executeUncached == nil {
		return connectorcontract.StepExecutionResult{}, fmt.Errorf("execute uncached callback is required")
	}

	stepResultPath := localstate.StepResultPath(adapterDomain, execCtx.WorkspaceID, execCtx, step)
	cachedResult, ok, err := localstate.LoadStepResult(stepResultPath)
	if err != nil {
		return connectorcontract.StepExecutionResult{}, err
	}
	if ok {
		return cachedResult, nil
	}

	result, execErr := executeUncached(stepResultPath)
	if execErr != nil {
		return result, execErr
	}
	if err := localstate.SaveStepResult(stepResultPath, result); err != nil {
		return connectorcontract.StepExecutionResult{}, err
	}
	return result, nil
}

func WorkspaceRootFromStepResultPath(stepResultPath string) string {
	return filepath.Dir(filepath.Dir(stepResultPath))
}
