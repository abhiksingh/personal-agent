package agentexec

import (
	"fmt"
	"strings"
)

func normalizeModelIntentCandidate(candidate ModelIntentCandidate, rawRequest string, minConfidence float64) (Intent, error) {
	workflow := normalizeWorkflow(candidate.Workflow)
	if workflow == "" {
		return Intent{}, fmt.Errorf("model intent workflow is required")
	}
	if candidate.Confidence < minConfidence {
		return Intent{}, fmt.Errorf("model intent confidence %.3f below threshold %.3f", candidate.Confidence, minConfidence)
	}

	trimmedRaw := strings.TrimSpace(rawRequest)
	switch workflow {
	case WorkflowBrowser:
		targetURL := normalizeURLCandidate(candidate.TargetURL)
		if targetURL == "" {
			targetURL = extractURL(rawRequest)
		}
		return newBrowserIntent(trimmedRaw, targetURL), nil
	case WorkflowFinder:
		targetPath := strings.TrimSpace(candidate.TargetPath)
		if targetPath == "" {
			targetPath = extractPath(rawRequest)
		}
		targetQuery := strings.TrimSpace(candidate.TargetQuery)
		if targetQuery == "" && looksLikeFinderRequest(strings.ToLower(strings.TrimSpace(rawRequest))) {
			targetQuery = extractFinderQuery(rawRequest)
		}
		return newFinderIntent(trimmedRaw, targetPath, targetQuery), nil
	case WorkflowMail:
		return newMailIntent(trimmedRaw), nil
	case WorkflowCalendar:
		return newCalendarIntent(trimmedRaw), nil
	case WorkflowMessages:
		channel := normalizeMessageChannel(candidate.MessageChannel)
		if channel == "" {
			channel = extractMessageChannel(rawRequest)
		}
		recipient := strings.TrimSpace(candidate.MessageRecipient)
		if recipient == "" {
			recipient = extractMessageRecipient(rawRequest)
		}
		body := strings.TrimSpace(candidate.MessageBody)
		if body == "" {
			body = extractMessageBody(rawRequest)
		}
		return newMessagesIntent(trimmedRaw, channel, recipient, body), nil
	default:
		return Intent{}, fmt.Errorf("unsupported model intent workflow %q", candidate.Workflow)
	}
}

func normalizeWorkflow(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case WorkflowMail:
		return WorkflowMail
	case WorkflowCalendar:
		return WorkflowCalendar
	case WorkflowMessages:
		return WorkflowMessages
	case WorkflowBrowser:
		return WorkflowBrowser
	case WorkflowFinder:
		return WorkflowFinder
	default:
		return ""
	}
}
