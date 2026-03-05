package cliapp

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"personalagent/runtime/internal/transport"
)

type cliHTTPErrorAdvice struct {
	WhatFailed string
	Why        string
	DoNext     []string
}

func writeHTTPErrorAdvice(stderr io.Writer, httpErr transport.HTTPError) int {
	advice := buildCLIHTTPErrorAdvice(httpErr)
	fmt.Fprintln(stderr, "request failed")
	fmt.Fprintf(stderr, "what failed: %s\n", advice.WhatFailed)
	fmt.Fprintf(stderr, "why: %s\n", advice.Why)
	fmt.Fprintln(stderr, "do next:")
	for _, step := range advice.DoNext {
		fmt.Fprintf(stderr, "- %s\n", step)
	}
	return 1
}

func buildCLIHTTPErrorAdvice(httpErr transport.HTTPError) cliHTTPErrorAdvice {
	code := strings.TrimSpace(httpErr.Code)
	message := firstNonEmpty(httpErr.Message, httpErr.Body, "Daemon request failed.")
	details := decodeHTTPErrorDetails(httpErr.DetailsPayload)

	advice := cliHTTPErrorAdvice{
		WhatFailed: "Daemon request could not be completed.",
		Why:        message,
		DoNext: []string{
			"Retry the command and confirm required flags/IDs are correct.",
		},
	}

	switch code {
	case "service_not_configured":
		advice.WhatFailed = serviceNotConfiguredWhatFailed(details)
		advice.Why = firstNonEmpty(message, "Required daemon service dependency is not configured.")
		advice.DoNext = []string{}
		configField := lookupNestedString(details, "service", "config_field")
		domain := lookupNestedString(details, "domain")
		if configField != "" {
			advice.DoNext = appendCLIAdviceStep(advice.DoNext, fmt.Sprintf("Configure daemon dependency `ServerConfig.%s` and restart the daemon.", configField))
		} else if domain != "" {
			advice.DoNext = appendCLIAdviceStep(advice.DoNext, fmt.Sprintf("Configure daemon dependencies for `%s` and restart the daemon.", domain))
		} else {
			advice.DoNext = appendCLIAdviceStep(advice.DoNext, "Configure the missing daemon service dependency and restart the daemon.")
		}
	case "auth_unauthorized":
		advice.WhatFailed = "Daemon authentication was rejected."
		advice.Why = firstNonEmpty(message, "Control auth token is missing or invalid.")
		advice.DoNext = []string{
			"Verify the active CLI auth token/profile matches the daemon control auth token.",
			"Run `personal-agent auth bootstrap-local-dev --profile <name>` then `personal-agent profile use --name <name>` if you need local-dev auth setup.",
		}
	case "auth_forbidden":
		advice.WhatFailed = "Daemon authorization denied this operation."
		advice.Why = firstNonEmpty(message, "Authenticated identity does not have permission for this request.")
		advice.DoNext = []string{
			"Confirm workspace and principal context for this command.",
			"Update delegation/capability grants and retry.",
		}
	case "resource_not_found":
		advice.WhatFailed = "Requested resource could not be found."
		advice.Why = firstNonEmpty(message, "Provided identifier does not exist in the selected workspace.")
		advice.DoNext = []string{
			"Verify resource IDs (task/run/thread/etc.) and workspace context, then retry.",
		}
	case "missing_required_field", "invalid_request_payload", "invalid_request":
		advice.WhatFailed = "Daemon rejected request parameters."
		advice.Why = firstNonEmpty(message, "Request payload is invalid or missing required fields.")
		advice.DoNext = []string{
			"Review command usage with `personal-agent help` and provide required flags/values.",
		}
	case "resource_conflict":
		advice.WhatFailed = "Requested operation conflicts with current resource state."
		advice.Why = firstNonEmpty(message, "Resource state is incompatible with this operation right now.")
		advice.DoNext = []string{
			"Refresh resource status and retry once state transitions complete.",
		}
	case "not_implemented":
		advice.WhatFailed = "Requested daemon endpoint is not implemented."
		advice.Why = firstNonEmpty(message, "This daemon build does not support the requested operation yet.")
		advice.DoNext = []string{
			"Use a daemon build/version that includes this endpoint or avoid this command path for now.",
		}
	case "internal_error":
		advice.WhatFailed = "Daemon encountered an internal runtime error."
		advice.Why = firstNonEmpty(message, "The daemon failed while processing the request.")
		advice.DoNext = []string{
			"Retry once; if it persists, inspect daemon logs before retrying again.",
		}
	}

	advice.DoNext = appendCLIAdviceStep(advice.DoNext, remediationHintSteps(details)...)
	advice.DoNext = appendCLIAdviceStep(advice.DoNext, "Re-run with `--error-output json` to inspect full structured error details.")
	if correlationID := strings.TrimSpace(httpErr.CorrelationID); correlationID != "" {
		advice.DoNext = appendCLIAdviceStep(advice.DoNext, fmt.Sprintf("If this persists, inspect daemon logs with correlation id `%s`.", correlationID))
	}

	if strings.TrimSpace(advice.WhatFailed) == "" {
		advice.WhatFailed = "Daemon request could not be completed."
	}
	if strings.TrimSpace(advice.Why) == "" {
		advice.Why = "Daemon returned an error response."
	}
	if len(advice.DoNext) == 0 {
		advice.DoNext = []string{
			"Re-run with `--error-output json` to inspect full structured error details.",
		}
	}
	return advice
}

func serviceNotConfiguredWhatFailed(details map[string]any) string {
	serviceLabel := lookupNestedString(details, "service", "label")
	if serviceLabel != "" {
		return fmt.Sprintf("%s is not configured on the daemon.", serviceLabel)
	}
	serviceID := lookupNestedString(details, "service", "id")
	if serviceID != "" {
		return fmt.Sprintf("Daemon service `%s` is not configured.", serviceID)
	}
	return "Requested daemon capability is not configured."
}

func decodeHTTPErrorDetails(raw json.RawMessage) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil
	}
	return payload
}

func remediationHintSteps(details map[string]any) []string {
	if len(details) == 0 {
		return nil
	}
	remediationRaw, ok := details["remediation"]
	if !ok || remediationRaw == nil {
		return nil
	}
	remediation, ok := remediationRaw.(map[string]any)
	if !ok {
		return nil
	}

	steps := make([]string, 0, 4)
	steps = appendCLIAdviceStep(steps, stringFromAny(remediation["summary"]))
	steps = appendCLIAdviceStep(steps, stringFromAny(remediation["hint"]))

	if action := stringFromAny(remediation["action"]); action != "" {
		steps = appendCLIAdviceStep(steps, fmt.Sprintf("Remediation action: `%s`.", action))
	}
	if label := stringFromAny(remediation["label"]); label != "" {
		steps = appendCLIAdviceStep(steps, label)
	}

	switch rawSteps := remediation["steps"].(type) {
	case []any:
		for _, rawStep := range rawSteps {
			steps = appendCLIAdviceStep(steps, stringFromAny(rawStep))
		}
	case []string:
		for _, rawStep := range rawSteps {
			steps = appendCLIAdviceStep(steps, stringFromAny(rawStep))
		}
	}
	return steps
}

func lookupNestedString(root map[string]any, path ...string) string {
	if len(root) == 0 || len(path) == 0 {
		return ""
	}
	var current any = root
	for _, key := range path {
		object, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		next, ok := object[key]
		if !ok {
			return ""
		}
		current = next
	}
	return stringFromAny(current)
}

func stringFromAny(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(typed)
	default:
		trimmed := strings.TrimSpace(fmt.Sprint(typed))
		if trimmed == "<nil>" {
			return ""
		}
		return trimmed
	}
}

func appendCLIAdviceStep(existing []string, candidates ...string) []string {
	result := append([]string(nil), existing...)
	for _, candidate := range candidates {
		trimmed := strings.TrimSpace(candidate)
		if trimmed == "" {
			continue
		}
		duplicate := false
		for _, prior := range result {
			if strings.EqualFold(strings.TrimSpace(prior), trimmed) {
				duplicate = true
				break
			}
		}
		if duplicate {
			continue
		}
		result = append(result, trimmed)
	}
	return result
}
