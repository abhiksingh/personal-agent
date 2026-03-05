package main

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"personalagent/runtime/internal/daemonruntime"
)

func loadWorkerExecAuthTokenFromEnv() (string, error) {
	token := strings.TrimSpace(os.Getenv(daemonruntime.WorkerExecAuthTokenEnvVar))
	if token == "" {
		return "", fmt.Errorf("%s is required", daemonruntime.WorkerExecAuthTokenEnvVar)
	}
	return token, nil
}

func authorizeWorkerExecuteRequest(request *http.Request, expectedToken string) bool {
	if request == nil {
		return false
	}
	trimmedExpected := strings.TrimSpace(expectedToken)
	if trimmedExpected == "" {
		return false
	}
	authValue := strings.TrimSpace(request.Header.Get("Authorization"))
	expectedAuth := "Bearer " + trimmedExpected
	if len(authValue) != len(expectedAuth) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(authValue), []byte(expectedAuth)) == 1
}

func writeWorkerUnauthorized(writer http.ResponseWriter) {
	writer.Header().Set("Content-Type", "application/json")
	writer.Header().Set("WWW-Authenticate", "Bearer")
	writer.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(writer).Encode(map[string]any{
		"error": "unauthorized",
	})
}
