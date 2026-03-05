package daemonruntime

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
)

const WorkerExecAuthTokenEnvVar = "PERSONAL_AGENT_WORKER_EXEC_AUTH_TOKEN"

func issueWorkerExecAuthToken() (string, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate worker auth token: %w", err)
	}
	return hex.EncodeToString(buffer), nil
}

func withWorkerExecAuthEnv(env []string, token string) []string {
	prefix := WorkerExecAuthTokenEnvVar + "="
	out := make([]string, 0, len(env)+1)
	for _, entry := range env {
		trimmed := strings.TrimSpace(entry)
		if strings.HasPrefix(trimmed, prefix) {
			continue
		}
		out = append(out, entry)
	}
	out = append(out, prefix+token)
	return out
}
