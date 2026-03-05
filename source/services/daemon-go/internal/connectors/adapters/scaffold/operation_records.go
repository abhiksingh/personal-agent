package scaffold

import (
	"path/filepath"
	"regexp"
	"strings"

	localstate "personalagent/runtime/internal/connectors/adapters/localstate"
)

var unsafeOperationTokenRegex = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

func OperationRecordPath(workspaceRoot string, capability string, operationID string) string {
	return filepath.Join(
		strings.TrimSpace(workspaceRoot),
		"operations",
		strings.TrimSpace(capability),
		OperationRecordToken(operationID)+".json",
	)
}

func OperationRecordToken(operationID string) string {
	trimmed := strings.TrimSpace(operationID)
	cleaned := unsafeOperationTokenRegex.ReplaceAllString(trimmed, "_")
	cleaned = strings.Trim(cleaned, "._-")
	if cleaned == "" {
		cleaned = "operation"
	}
	if len(cleaned) > 96 {
		return cleaned[:96]
	}
	return cleaned
}

func WriteOperationRecord(workspaceRoot string, capability string, operationID string, record any) (string, error) {
	path := OperationRecordPath(workspaceRoot, capability, operationID)
	if err := localstate.WriteJSONFile(path, record); err != nil {
		return "", err
	}
	return path, nil
}
