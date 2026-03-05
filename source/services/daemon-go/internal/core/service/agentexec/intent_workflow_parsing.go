package agentexec

import (
	"fmt"
	"strings"
)

func InterpretIntent(request string) (Intent, error) {
	trimmed := strings.TrimSpace(request)
	if trimmed == "" {
		return Intent{}, fmt.Errorf("request is required")
	}

	targetURL := extractURL(trimmed)
	if targetURL != "" {
		return newBrowserIntent(trimmed, targetURL), nil
	}

	lower := strings.ToLower(trimmed)
	if looksLikeFinderRequest(lower) {
		return newFinderIntent(trimmed, extractPath(trimmed), extractFinderQuery(trimmed)), nil
	}

	if containsAny(lower, []string{"calendar", "meeting", "schedule event", "reschedule"}) {
		return newCalendarIntent(trimmed), nil
	}

	if containsAny(lower, []string{"email", "mail", "inbox", "reply"}) {
		return newMailIntent(trimmed), nil
	}

	if looksLikeMessagesRequest(lower) {
		return newMessagesIntent(
			trimmed,
			extractMessageChannel(trimmed),
			extractMessageRecipient(trimmed),
			extractMessageBody(trimmed),
		), nil
	}

	if looksLikeBrowserRequest(lower) {
		return newBrowserIntent(trimmed, ""), nil
	}

	return Intent{}, fmt.Errorf("unable to determine intent from request")
}

func newMailIntent(rawRequest string) Intent {
	operation := detectMailOperation(rawRequest)
	recipient := strings.TrimSpace(firstMatch(emailAddressPattern, rawRequest))
	subject := extractQuotedText(rawRequest)
	body := strings.TrimSpace(rawRequest)
	limit := 0
	if operation == mailOperationSummarizeUnread {
		body = ""
		limit = extractMailSummaryLimit(rawRequest)
	}
	intent := Intent{
		Workflow:   WorkflowMail,
		RawRequest: strings.TrimSpace(rawRequest),
		Action: NativeAction{
			Connector: WorkflowMail,
			Operation: operation,
			Mail: &MailAction{
				Operation: operation,
				Recipient: recipient,
				Subject:   subject,
				Body:      body,
				Limit:     limit,
			},
		},
	}
	return intent
}

func newCalendarIntent(rawRequest string) Intent {
	operation := detectCalendarOperation(rawRequest)
	eventID := extractCalendarEventID(rawRequest)
	title := extractQuotedText(rawRequest)
	if operation == calendarOperationCreate && title == "" {
		title = strings.TrimSpace(rawRequest)
	}
	intent := Intent{
		Workflow:   WorkflowCalendar,
		RawRequest: strings.TrimSpace(rawRequest),
		Action: NativeAction{
			Connector: WorkflowCalendar,
			Operation: operation,
			Calendar: &CalendarAction{
				Operation: operation,
				EventID:   eventID,
				Title:     title,
			},
		},
	}
	missingSlots := make([]string, 0, 2)
	switch operation {
	case calendarOperationCreate:
		if strings.TrimSpace(title) == "" {
			missingSlots = append(missingSlots, slotCalendarTitle)
		}
	case calendarOperationUpdate:
		if strings.TrimSpace(eventID) == "" {
			missingSlots = append(missingSlots, slotCalendarEventID)
		}
		if strings.TrimSpace(title) == "" {
			missingSlots = append(missingSlots, slotCalendarTitle)
		}
	case calendarOperationCancel:
		if strings.TrimSpace(eventID) == "" {
			missingSlots = append(missingSlots, slotCalendarEventID)
		}
	}
	return withClarification(intent, missingSlots)
}

func newMessagesIntent(rawRequest string, channel string, recipient string, body string) Intent {
	resolvedChannel := normalizeMessageChannel(channel)
	intent := Intent{
		Workflow:   WorkflowMessages,
		RawRequest: strings.TrimSpace(rawRequest),
		Action: NativeAction{
			Connector: WorkflowMessages,
			Operation: "send_message",
			Messages: &MessagesAction{
				Operation: "send_message",
				Recipient: strings.TrimSpace(recipient),
				Body:      strings.TrimSpace(body),
				Channel:   resolvedChannel,
			},
		},
	}

	missingSlots := make([]string, 0, 3)
	if resolvedChannel == "" {
		missingSlots = append(missingSlots, slotMessageChannel)
	}
	if strings.TrimSpace(recipient) == "" {
		missingSlots = append(missingSlots, slotMessageRecipient)
	}
	if strings.TrimSpace(body) == "" {
		missingSlots = append(missingSlots, slotMessageBody)
	}
	return withClarification(intent, missingSlots)
}

func newBrowserIntent(rawRequest string, targetURL string) Intent {
	resolvedURL := strings.TrimSpace(targetURL)
	query := strings.TrimSpace(rawRequest)
	intent := Intent{
		Workflow:   WorkflowBrowser,
		TargetURL:  resolvedURL,
		RawRequest: strings.TrimSpace(rawRequest),
		Action: NativeAction{
			Connector: WorkflowBrowser,
			Operation: "open_extract_close",
			Browser: &BrowserAction{
				Operation: "open_extract_close",
				TargetURL: resolvedURL,
				Query:     query,
			},
		},
	}

	missingSlots := make([]string, 0, 1)
	if resolvedURL == "" {
		missingSlots = append(missingSlots, slotTargetURL)
	}
	return withClarification(intent, missingSlots)
}

func newFinderIntent(rawRequest string, targetPath string, targetQuery string) Intent {
	operation := detectFinderOperation(rawRequest)
	resolvedPath := strings.TrimSpace(targetPath)
	resolvedQuery := strings.TrimSpace(targetQuery)
	intent := Intent{
		Workflow:   WorkflowFinder,
		TargetPath: resolvedPath,
		RawRequest: strings.TrimSpace(rawRequest),
		Action: NativeAction{
			Connector: WorkflowFinder,
			Operation: operation,
			Finder: &FinderAction{
				Operation:  operation,
				TargetPath: resolvedPath,
				Query:      resolvedQuery,
			},
		},
	}

	missingSlots := make([]string, 0, 2)
	if resolvedPath != "" && !strings.HasPrefix(resolvedPath, "/") {
		missingSlots = append(missingSlots, slotTargetPath)
	}
	switch operation {
	case finderOperationFind:
		if resolvedQuery == "" {
			missingSlots = append(missingSlots, slotFinderQuery)
		}
	default:
		if resolvedPath == "" && resolvedQuery == "" {
			missingSlots = append(missingSlots, slotFinderQuery)
		}
	}
	return withClarification(intent, missingSlots)
}

func withClarification(intent Intent, missingSlots []string) Intent {
	if len(missingSlots) == 0 {
		intent.MissingSlots = nil
		intent.ClarificationPrompt = ""
		return intent
	}

	intent.MissingSlots = append([]string(nil), missingSlots...)
	intent.ClarificationPrompt = buildClarificationPrompt(intent.Workflow, intent.MissingSlots)
	return intent
}

func buildClarificationPrompt(workflow string, missingSlots []string) string {
	switch workflow {
	case WorkflowBrowser:
		return "I can run the browser workflow, but I need a target URL (http/https). Share the URL to continue."
	case WorkflowFinder:
		if containsSlot(missingSlots, slotTargetPath) && containsSlot(missingSlots, slotFinderQuery) {
			return "I can run the finder workflow, but I need an absolute path or a specific file query."
		}
		if containsSlot(missingSlots, slotTargetPath) {
			return "I can run the finder workflow, but I need an absolute target path (for example /Users/name/file.txt)."
		}
		if containsSlot(missingSlots, slotFinderQuery) {
			return "I can run the finder workflow, but I need a specific file query (for example \"budget report\" or a full path)."
		}
		return "I can run the finder workflow, but I need a target path or file query."
	case WorkflowMessages:
		return "I can prepare a messages action, but I need the delivery channel (imessage or sms), recipient, and message body."
	case WorkflowCalendar:
		if containsSlot(missingSlots, slotCalendarEventID) && containsSlot(missingSlots, slotCalendarTitle) {
			return "I can run the calendar workflow, but I need the target event_id and updated title."
		}
		if containsSlot(missingSlots, slotCalendarEventID) {
			return "I can run the calendar workflow, but I need the target event_id."
		}
		if containsSlot(missingSlots, slotCalendarTitle) {
			return "I can run the calendar workflow, but I need an event title."
		}
		return "I can run the calendar workflow, but I need additional event details."
	default:
		return fmt.Sprintf("I need additional details before executing this %s action: %s.", workflow, strings.Join(missingSlots, ", "))
	}
}

func detectMailOperation(rawRequest string) string {
	lower := strings.ToLower(strings.TrimSpace(rawRequest))
	switch {
	case containsAny(lower, []string{"unread", "inbox summary", "summarize inbox", "summarise inbox", "summarize unread", "summarise unread", "list unread"}):
		return mailOperationSummarizeUnread
	case containsAny(lower, []string{"reply", "respond"}):
		return mailOperationReply
	case containsAny(lower, []string{"draft", "compose"}):
		return mailOperationDraft
	default:
		return mailOperationSend
	}
}

func detectCalendarOperation(rawRequest string) string {
	lower := strings.ToLower(strings.TrimSpace(rawRequest))
	switch {
	case containsAny(lower, []string{"cancel", "delete event", "remove event", "drop event"}):
		return calendarOperationCancel
	case containsAny(lower, []string{"update", "reschedule", "move", "change"}) && containsAny(lower, []string{"calendar", "event", "meeting", "schedule"}):
		return calendarOperationUpdate
	default:
		return calendarOperationCreate
	}
}

func detectFinderOperation(rawRequest string) string {
	lower := strings.ToLower(strings.TrimSpace(rawRequest))
	switch {
	case containsAny(lower, []string{"find", "search", "locate"}):
		return finderOperationFind
	case containsAny(lower, []string{"delete", "remove", "trash"}):
		return finderOperationDelete
	case containsAny(lower, []string{"preview", "show", "inspect", "open", "read"}):
		return finderOperationPreview
	case containsAny(lower, []string{"list"}):
		return finderOperationList
	default:
		return finderOperationFind
	}
}
