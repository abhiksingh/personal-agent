package localstate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"personalagent/runtime/internal/filesecurity"
	"personalagent/runtime/internal/runtimepaths"
	shared "personalagent/runtime/internal/shared/contracts"
)

const envDataDir = "PA_CONNECTOR_DATA_DIR"

var unsafePathTokenPattern = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

type storedStepResult struct {
	SavedAt string                     `json:"saved_at"`
	Result  shared.StepExecutionResult `json:"result"`
}

func baseDir() string {
	if raw := strings.TrimSpace(os.Getenv(envDataDir)); raw != "" {
		return filepath.Join(raw, "connectors")
	}
	if defaultConnectorsDir, err := runtimepaths.DefaultConnectorsDir(); err == nil && strings.TrimSpace(defaultConnectorsDir) != "" {
		return defaultConnectorsDir
	}
	return filepath.Join(os.TempDir(), "personal-agent", "connectors")
}

func sanitizeToken(value string, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	cleaned := unsafePathTokenPattern.ReplaceAllString(trimmed, "_")
	cleaned = strings.Trim(cleaned, "._-")
	if cleaned == "" {
		return fallback
	}
	return cleaned
}

func resolveStepKey(execCtx shared.ExecutionContext, step shared.TaskStep) string {
	if trimmed := strings.TrimSpace(execCtx.StepID); trimmed != "" {
		return trimmed
	}
	if trimmed := strings.TrimSpace(step.ID); trimmed != "" {
		return trimmed
	}
	if trimmed := strings.TrimSpace(step.CapabilityKey); trimmed != "" {
		return trimmed
	}
	return "step"
}

func StepResultPath(connector string, workspaceID string, execCtx shared.ExecutionContext, step shared.TaskStep) string {
	connectorToken := sanitizeToken(connector, "connector")
	workspaceToken := sanitizeToken(workspaceID, "workspace")
	stepToken := sanitizeToken(resolveStepKey(execCtx, step), "step")
	return filepath.Join(baseDir(), connectorToken, workspaceToken, "steps", stepToken+".json")
}

func LoadStepResult(path string) (shared.StepExecutionResult, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return shared.StepExecutionResult{}, false, nil
		}
		return shared.StepExecutionResult{}, false, fmt.Errorf("read cached step result: %w", err)
	}

	var stored storedStepResult
	if err := json.Unmarshal(data, &stored); err != nil {
		return shared.StepExecutionResult{}, false, fmt.Errorf("decode cached step result: %w", err)
	}
	return stored.Result, true, nil
}

func SaveStepResult(path string, result shared.StepExecutionResult) error {
	if err := filesecurity.EnsurePrivateDir(filepath.Dir(path)); err != nil {
		return fmt.Errorf("create step result directory: %w", err)
	}

	stored := storedStepResult{
		SavedAt: time.Now().UTC().Format(time.RFC3339Nano),
		Result:  result,
	}
	payload, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		return fmt.Errorf("encode step result payload: %w", err)
	}

	tempFile, err := os.CreateTemp(filepath.Dir(path), ".step-result-*.json")
	if err != nil {
		return fmt.Errorf("create temp step result file: %w", err)
	}
	tempName := tempFile.Name()
	cleanup := true
	defer func() {
		_ = tempFile.Close()
		if cleanup {
			_ = os.Remove(tempName)
		}
	}()

	if err := tempFile.Chmod(filesecurity.PrivateFileMode); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("set temp step result permissions: %w", err)
	}
	if _, err := tempFile.Write(payload); err != nil {
		return fmt.Errorf("write temp step result: %w", err)
	}
	if err := tempFile.Sync(); err != nil {
		return fmt.Errorf("sync temp step result: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("close temp step result: %w", err)
	}
	if err := os.Rename(tempName, path); err != nil {
		return fmt.Errorf("replace cached step result: %w", err)
	}
	if err := filesecurity.EnsurePrivateFile(path); err != nil {
		return fmt.Errorf("harden cached step result permissions: %w", err)
	}
	cleanup = false
	return nil
}

func ReadJSONFile(path string, target any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, target)
}

func WriteJSONFile(path string, value any) error {
	if err := filesecurity.EnsurePrivateDir(filepath.Dir(path)); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, payload, filesecurity.PrivateFileMode); err != nil {
		return err
	}
	return filesecurity.EnsurePrivateFile(path)
}
