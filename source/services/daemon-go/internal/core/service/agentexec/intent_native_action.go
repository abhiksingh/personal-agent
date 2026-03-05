package agentexec

import (
	"fmt"
	"path/filepath"
	"strings"
)

func IntentFromNativeAction(action *NativeAction) (Intent, error) {
	normalized, err := normalizeNativeAction(action)
	if err != nil {
		return Intent{}, err
	}
	intent := Intent{
		Workflow:   strings.TrimSpace(normalized.Connector),
		RawRequest: fmt.Sprintf("native:%s.%s", strings.TrimSpace(normalized.Connector), strings.TrimSpace(normalized.Operation)),
		Action:     normalized,
	}
	if normalized.Browser != nil {
		intent.TargetURL = strings.TrimSpace(normalized.Browser.TargetURL)
	}
	if normalized.Finder != nil {
		intent.TargetPath = strings.TrimSpace(normalized.Finder.TargetPath)
	}
	return intent, nil
}

func normalizeNativeAction(action *NativeAction) (NativeAction, error) {
	if action == nil {
		return NativeAction{}, fmt.Errorf("native action is required")
	}

	normalized := NativeAction{
		Connector: strings.ToLower(strings.TrimSpace(action.Connector)),
		Operation: strings.ToLower(strings.TrimSpace(action.Operation)),
	}
	if action.Mail != nil {
		normalized.Mail = &MailAction{
			Operation: strings.ToLower(strings.TrimSpace(action.Mail.Operation)),
			Recipient: strings.TrimSpace(action.Mail.Recipient),
			Subject:   strings.TrimSpace(action.Mail.Subject),
			Body:      strings.TrimSpace(action.Mail.Body),
			Limit:     action.Mail.Limit,
		}
	}
	if action.Calendar != nil {
		normalized.Calendar = &CalendarAction{
			Operation: strings.ToLower(strings.TrimSpace(action.Calendar.Operation)),
			EventID:   strings.TrimSpace(action.Calendar.EventID),
			Title:     strings.TrimSpace(action.Calendar.Title),
			Notes:     strings.TrimSpace(action.Calendar.Notes),
		}
	}
	if action.Messages != nil {
		normalized.Messages = &MessagesAction{
			Operation: strings.ToLower(strings.TrimSpace(action.Messages.Operation)),
			Channel:   normalizeMessageChannel(action.Messages.Channel),
			Recipient: strings.TrimSpace(action.Messages.Recipient),
			Body:      strings.TrimSpace(action.Messages.Body),
		}
	}
	if action.Browser != nil {
		normalized.Browser = &BrowserAction{
			Operation: strings.ToLower(strings.TrimSpace(action.Browser.Operation)),
			TargetURL: normalizeURLCandidate(action.Browser.TargetURL),
			Query:     strings.TrimSpace(action.Browser.Query),
		}
	}
	if action.Finder != nil {
		normalized.Finder = &FinderAction{
			Operation:  strings.ToLower(strings.TrimSpace(action.Finder.Operation)),
			TargetPath: strings.TrimSpace(action.Finder.TargetPath),
			Query:      strings.TrimSpace(action.Finder.Query),
			RootPath:   strings.TrimSpace(action.Finder.RootPath),
		}
	}

	inferredConnector, inferErr := inferNativeActionConnector(normalized)
	if inferErr != nil {
		return NativeAction{}, inferErr
	}
	if normalized.Connector == "" {
		normalized.Connector = inferredConnector
	} else {
		normalized.Connector = normalizeWorkflow(normalized.Connector)
		if normalized.Connector == "" {
			return NativeAction{}, fmt.Errorf("native action connector is not supported")
		}
		if inferredConnector != "" && normalized.Connector != inferredConnector {
			return NativeAction{}, fmt.Errorf("native action connector/payload mismatch")
		}
	}
	if normalized.Connector == "" {
		return NativeAction{}, fmt.Errorf("native action connector is required")
	}

	switch normalized.Connector {
	case WorkflowMail:
		if normalized.Mail == nil {
			return NativeAction{}, fmt.Errorf("mail native action payload is required")
		}
		operation := firstNonEmptyLower(normalized.Mail.Operation, normalized.Operation)
		switch operation {
		case mailOperationDraft, mailOperationSend, mailOperationReply, mailOperationSummarizeUnread:
		default:
			return NativeAction{}, fmt.Errorf("mail native action operation is invalid")
		}
		if operation != mailOperationSummarizeUnread && strings.TrimSpace(normalized.Mail.Body) == "" {
			return NativeAction{}, fmt.Errorf("mail native action body is required")
		}
		if (operation == mailOperationSend || operation == mailOperationReply) && strings.TrimSpace(normalized.Mail.Recipient) == "" {
			return NativeAction{}, fmt.Errorf("mail native action recipient is required for %s", operation)
		}
		if normalized.Mail.Limit < 0 {
			return NativeAction{}, fmt.Errorf("mail native action limit must be non-negative")
		}
		normalized.Operation = operation
		normalized.Mail.Operation = operation
	case WorkflowCalendar:
		if normalized.Calendar == nil {
			return NativeAction{}, fmt.Errorf("calendar native action payload is required")
		}
		operation := firstNonEmptyLower(normalized.Calendar.Operation, normalized.Operation)
		switch operation {
		case calendarOperationCreate, calendarOperationUpdate, calendarOperationCancel:
		default:
			return NativeAction{}, fmt.Errorf("calendar native action operation is invalid")
		}
		eventID := strings.TrimSpace(normalized.Calendar.EventID)
		title := strings.TrimSpace(normalized.Calendar.Title)
		notes := strings.TrimSpace(normalized.Calendar.Notes)
		switch operation {
		case calendarOperationCreate:
			if title == "" {
				return NativeAction{}, fmt.Errorf("calendar native action title is required")
			}
		case calendarOperationUpdate:
			if eventID == "" {
				return NativeAction{}, fmt.Errorf("calendar native action event_id is required for update")
			}
			if title == "" && notes == "" {
				return NativeAction{}, fmt.Errorf("calendar native action update requires title or notes")
			}
		case calendarOperationCancel:
			if eventID == "" {
				return NativeAction{}, fmt.Errorf("calendar native action event_id is required for cancel")
			}
		}
		normalized.Operation = operation
		normalized.Calendar.Operation = operation
		normalized.Calendar.EventID = eventID
		normalized.Calendar.Title = title
		normalized.Calendar.Notes = notes
	case WorkflowMessages:
		if normalized.Messages == nil {
			return NativeAction{}, fmt.Errorf("messages native action payload is required")
		}
		operation := firstNonEmptyLower(normalized.Messages.Operation, normalized.Operation)
		if operation == "" {
			operation = "send_message"
		}
		if normalized.Messages.Channel == "" {
			return NativeAction{}, fmt.Errorf("messages native action channel is required")
		}
		if strings.TrimSpace(normalized.Messages.Recipient) == "" {
			return NativeAction{}, fmt.Errorf("messages native action recipient is required")
		}
		if strings.TrimSpace(normalized.Messages.Body) == "" {
			return NativeAction{}, fmt.Errorf("messages native action body is required")
		}
		normalized.Operation = operation
		normalized.Messages.Operation = operation
	case WorkflowBrowser:
		if normalized.Browser == nil {
			return NativeAction{}, fmt.Errorf("browser native action payload is required")
		}
		operation := firstNonEmptyLower(normalized.Browser.Operation, normalized.Operation)
		if operation == "" {
			operation = "open_extract_close"
		}
		switch operation {
		case "open", "extract", "close", "open_extract_close":
		default:
			return NativeAction{}, fmt.Errorf("browser native action operation is invalid")
		}
		if strings.TrimSpace(normalized.Browser.TargetURL) == "" {
			return NativeAction{}, fmt.Errorf("browser native action target_url is required")
		}
		normalized.Operation = operation
		normalized.Browser.Operation = operation
		normalized.Browser.Query = strings.TrimSpace(normalized.Browser.Query)
	case WorkflowFinder:
		if normalized.Finder == nil {
			return NativeAction{}, fmt.Errorf("finder native action payload is required")
		}
		operation := firstNonEmptyLower(normalized.Finder.Operation, normalized.Operation)
		switch operation {
		case finderOperationFind, finderOperationList, finderOperationPreview, finderOperationDelete:
		default:
			return NativeAction{}, fmt.Errorf("finder native action operation is invalid")
		}
		if normalized.Finder.TargetPath != "" && !filepath.IsAbs(strings.TrimSpace(normalized.Finder.TargetPath)) {
			return NativeAction{}, fmt.Errorf("finder native action target_path must be absolute")
		}
		if normalized.Finder.RootPath != "" && !filepath.IsAbs(strings.TrimSpace(normalized.Finder.RootPath)) {
			return NativeAction{}, fmt.Errorf("finder native action root_path must be absolute")
		}
		if operation == finderOperationFind {
			if strings.TrimSpace(normalized.Finder.Query) == "" {
				return NativeAction{}, fmt.Errorf("finder native action query is required for find")
			}
		} else if strings.TrimSpace(normalized.Finder.TargetPath) == "" && strings.TrimSpace(normalized.Finder.Query) == "" {
			return NativeAction{}, fmt.Errorf("finder native action requires target_path or query")
		}
		normalized.Operation = operation
		normalized.Finder.Operation = operation
		normalized.Finder.Query = strings.TrimSpace(normalized.Finder.Query)
		normalized.Finder.RootPath = strings.TrimSpace(normalized.Finder.RootPath)
		normalized.Finder.TargetPath = strings.TrimSpace(normalized.Finder.TargetPath)
	default:
		return NativeAction{}, fmt.Errorf("unsupported native action connector %q", normalized.Connector)
	}

	return normalized, nil
}

func inferNativeActionConnector(action NativeAction) (string, error) {
	seen := make([]string, 0, 5)
	if action.Mail != nil {
		seen = append(seen, WorkflowMail)
	}
	if action.Calendar != nil {
		seen = append(seen, WorkflowCalendar)
	}
	if action.Messages != nil {
		seen = append(seen, WorkflowMessages)
	}
	if action.Browser != nil {
		seen = append(seen, WorkflowBrowser)
	}
	if action.Finder != nil {
		seen = append(seen, WorkflowFinder)
	}
	if len(seen) == 0 {
		return "", nil
	}
	if len(seen) > 1 {
		return "", fmt.Errorf("native action payload must target exactly one connector")
	}
	return seen[0], nil
}

func firstNonEmptyLower(values ...string) string {
	for _, value := range values {
		trimmed := strings.ToLower(strings.TrimSpace(value))
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}
