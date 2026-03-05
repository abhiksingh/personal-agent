package cliapp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"personalagent/runtime/internal/transport"
)

type cliOutputMode string

const (
	cliOutputModeJSON        cliOutputMode = "json"
	cliOutputModeJSONCompact cliOutputMode = "json-compact"
	cliOutputModeText        cliOutputMode = "text"
)

type cliErrorOutputMode string

const (
	cliErrorOutputModeText cliErrorOutputMode = "text"
	cliErrorOutputModeJSON cliErrorOutputMode = "json"
)

type cliOutputConfig struct {
	outputMode      cliOutputMode
	errorOutputMode cliErrorOutputMode
}

var (
	cliOutputConfigMu      sync.RWMutex
	currentOutputConfigCLI = cliOutputConfig{
		outputMode:      cliOutputModeJSON,
		errorOutputMode: cliErrorOutputModeText,
	}
)

func resolveCLIOutputConfig(outputModeRaw string, errorOutputModeRaw string) (cliOutputConfig, error) {
	outputMode := cliOutputMode(strings.ToLower(strings.TrimSpace(outputModeRaw)))
	if outputMode == "" {
		outputMode = cliOutputModeJSON
	}
	switch outputMode {
	case cliOutputModeJSON, cliOutputModeJSONCompact, cliOutputModeText:
	default:
		return cliOutputConfig{}, fmt.Errorf("unsupported --output %q", outputModeRaw)
	}

	errorOutputMode := cliErrorOutputMode(strings.ToLower(strings.TrimSpace(errorOutputModeRaw)))
	if errorOutputMode == "" {
		errorOutputMode = cliErrorOutputModeText
	}
	switch errorOutputMode {
	case cliErrorOutputModeText, cliErrorOutputModeJSON:
	default:
		return cliOutputConfig{}, fmt.Errorf("unsupported --error-output %q", errorOutputModeRaw)
	}

	return cliOutputConfig{
		outputMode:      outputMode,
		errorOutputMode: errorOutputMode,
	}, nil
}

func setCLIOutputConfig(config cliOutputConfig) func() {
	cliOutputConfigMu.Lock()
	previous := currentOutputConfigCLI
	currentOutputConfigCLI = config
	cliOutputConfigMu.Unlock()
	return func() {
		cliOutputConfigMu.Lock()
		currentOutputConfigCLI = previous
		cliOutputConfigMu.Unlock()
	}
}

func getCLIOutputConfig() cliOutputConfig {
	cliOutputConfigMu.RLock()
	config := currentOutputConfigCLI
	cliOutputConfigMu.RUnlock()
	return config
}

func finalizeCLIErrorOutput(stderr io.Writer, captured *bytes.Buffer, exitCode int) int {
	if captured == nil {
		return exitCode
	}

	raw := captured.String()
	trimmed := strings.TrimSpace(raw)
	if exitCode == 0 {
		if raw != "" {
			_, _ = io.WriteString(stderr, raw)
		}
		return 0
	}

	if trimmed == "" {
		_ = writeJSON(stderr, map[string]any{
			"error": map[string]any{
				"code":      "cli.command_failed",
				"message":   "command failed",
				"exit_code": exitCode,
			},
		})
		return exitCode
	}

	if json.Valid([]byte(trimmed)) {
		_, _ = io.WriteString(stderr, trimmed)
		if !strings.HasSuffix(trimmed, "\n") {
			_, _ = io.WriteString(stderr, "\n")
		}
		return exitCode
	}

	_ = writeJSON(stderr, map[string]any{
		"error": map[string]any{
			"code":      "cli.command_failed",
			"message":   trimmed,
			"exit_code": exitCode,
		},
	})
	return exitCode
}

func writeJSON(writer io.Writer, payload any) int {
	config := getCLIOutputConfig()
	encoder := json.NewEncoder(writer)
	if config.outputMode != cliOutputModeJSONCompact {
		encoder.SetIndent("", "  ")
	}
	if err := encoder.Encode(payload); err != nil {
		return 1
	}
	return 0
}

func writeStructuredCLIError(
	stderr io.Writer,
	code string,
	message string,
	statusCode int,
	correlationID string,
	details any,
) int {
	errorPayload := map[string]any{
		"message": strings.TrimSpace(message),
	}
	if strings.TrimSpace(code) != "" {
		errorPayload["code"] = strings.TrimSpace(code)
	}
	if statusCode > 0 {
		errorPayload["status_code"] = statusCode
	}
	if strings.TrimSpace(correlationID) != "" {
		errorPayload["correlation_id"] = strings.TrimSpace(correlationID)
	}
	if details != nil {
		errorPayload["details"] = details
	}
	if writeJSON(stderr, map[string]any{"error": errorPayload}) != 0 {
		return 1
	}
	return 1
}

func writeError(stderr io.Writer, err error) int {
	config := getCLIOutputConfig()
	var httpErr transport.HTTPError
	if errors.As(err, &httpErr) {
		message := strings.TrimSpace(httpErr.Message)
		if message == "" {
			message = strings.TrimSpace(httpErr.Body)
		}
		code := strings.TrimSpace(httpErr.Code)
		correlationID := strings.TrimSpace(httpErr.CorrelationID)
		if config.errorOutputMode == cliErrorOutputModeJSON {
			if code == "" {
				code = "transport.http_error"
			}
			return writeStructuredCLIError(stderr, code, message, httpErr.StatusCode, correlationID, httpErr.DetailsPayload)
		}
		return writeHTTPErrorAdvice(stderr, httpErr)
	}
	if config.errorOutputMode == cliErrorOutputModeJSON {
		return writeStructuredCLIError(stderr, "cli.request_failed", err.Error(), 0, "", nil)
	}
	fmt.Fprintf(stderr, "request failed: %v\n", err)
	return 1
}
