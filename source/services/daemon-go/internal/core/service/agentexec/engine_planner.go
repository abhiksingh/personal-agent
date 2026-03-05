package agentexec

import (
	"fmt"
	"path/filepath"
	"strings"
)

func planSteps(intent Intent) ([]plannedStep, error) {
	switch intent.Workflow {
	case WorkflowMail:
		operation := strings.ToLower(strings.TrimSpace(intent.Action.Operation))
		if intent.Action.Mail != nil && strings.TrimSpace(intent.Action.Mail.Operation) != "" {
			operation = strings.ToLower(strings.TrimSpace(intent.Action.Mail.Operation))
		}
		recipient := ""
		subject := ""
		body := ""
		limit := 0
		if intent.Action.Mail != nil {
			recipient = strings.TrimSpace(intent.Action.Mail.Recipient)
			subject = strings.TrimSpace(intent.Action.Mail.Subject)
			body = strings.TrimSpace(intent.Action.Mail.Body)
			limit = intent.Action.Mail.Limit
		}
		if operation != mailOperationSummarizeUnread && body == "" {
			return nil, fmt.Errorf("mail workflow requires body")
		}
		switch operation {
		case mailOperationDraft:
			input := map[string]any{
				"body": body,
			}
			if recipient != "" {
				input["recipient"] = recipient
			}
			if subject != "" {
				input["subject"] = subject
			}
			return []plannedStep{
				{StepIndex: 0, Name: "Draft email", CapabilityKey: "mail_draft", Input: cloneAnyMap(input)},
			}, nil
		case mailOperationSend:
			if recipient == "" {
				return nil, fmt.Errorf("mail send workflow requires recipient")
			}
			input := map[string]any{
				"body":      body,
				"recipient": recipient,
			}
			if subject != "" {
				input["subject"] = subject
			}
			return []plannedStep{
				{StepIndex: 0, Name: "Send email", CapabilityKey: "mail_send", Input: cloneAnyMap(input)},
			}, nil
		case mailOperationReply:
			if recipient == "" {
				return nil, fmt.Errorf("mail reply workflow requires recipient")
			}
			input := map[string]any{
				"body":      body,
				"recipient": recipient,
			}
			if subject != "" {
				input["subject"] = subject
			}
			return []plannedStep{
				{StepIndex: 0, Name: "Reply to thread", CapabilityKey: "mail_reply", Input: cloneAnyMap(input)},
			}, nil
		case mailOperationSummarizeUnread:
			if limit <= 0 {
				limit = 5
			}
			if limit > 50 {
				limit = 50
			}
			return []plannedStep{
				{StepIndex: 0, Name: "Summarize unread inbox mail", CapabilityKey: "mail_unread_summary", Input: map[string]any{"limit": limit}},
			}, nil
		default:
			return nil, fmt.Errorf("mail workflow requires supported operation")
		}
	case WorkflowCalendar:
		operation := strings.ToLower(strings.TrimSpace(intent.Action.Operation))
		if intent.Action.Calendar != nil && strings.TrimSpace(intent.Action.Calendar.Operation) != "" {
			operation = strings.ToLower(strings.TrimSpace(intent.Action.Calendar.Operation))
		}
		eventID := ""
		title := ""
		notes := ""
		if intent.Action.Calendar != nil {
			eventID = strings.TrimSpace(intent.Action.Calendar.EventID)
			title = strings.TrimSpace(intent.Action.Calendar.Title)
			notes = strings.TrimSpace(intent.Action.Calendar.Notes)
		}
		switch operation {
		case calendarOperationCreate:
			if title == "" {
				return nil, fmt.Errorf("calendar create workflow requires title")
			}
			input := map[string]any{
				"title": title,
			}
			if notes != "" {
				input["notes"] = notes
			}
			return []plannedStep{
				{StepIndex: 0, Name: "Create event", CapabilityKey: "calendar_create", Input: cloneAnyMap(input)},
			}, nil
		case calendarOperationUpdate:
			if eventID == "" {
				return nil, fmt.Errorf("calendar update workflow requires event_id")
			}
			if title == "" && notes == "" {
				return nil, fmt.Errorf("calendar update workflow requires title or notes")
			}
			input := map[string]any{
				"event_id": eventID,
			}
			if title != "" {
				input["title"] = title
			}
			if notes != "" {
				input["notes"] = notes
			}
			return []plannedStep{
				{StepIndex: 0, Name: "Update event", CapabilityKey: "calendar_update", Input: cloneAnyMap(input)},
			}, nil
		case calendarOperationCancel:
			if eventID == "" {
				return nil, fmt.Errorf("calendar cancel workflow requires event_id")
			}
			input := map[string]any{
				"event_id": eventID,
			}
			return []plannedStep{
				{StepIndex: 0, Name: "Cancel event", CapabilityKey: "calendar_cancel", Input: cloneAnyMap(input)},
			}, nil
		default:
			return nil, fmt.Errorf("calendar workflow requires supported operation")
		}
	case WorkflowBrowser:
		targetURL := strings.TrimSpace(intent.TargetURL)
		query := ""
		if intent.Action.Browser != nil && strings.TrimSpace(intent.Action.Browser.TargetURL) != "" {
			targetURL = strings.TrimSpace(intent.Action.Browser.TargetURL)
		}
		if intent.Action.Browser != nil {
			query = strings.TrimSpace(intent.Action.Browser.Query)
		}
		if targetURL == "" {
			return nil, fmt.Errorf("browser workflow requires target URL")
		}
		operation := strings.ToLower(strings.TrimSpace(intent.Action.Operation))
		if intent.Action.Browser != nil && strings.TrimSpace(intent.Action.Browser.Operation) != "" {
			operation = strings.ToLower(strings.TrimSpace(intent.Action.Browser.Operation))
		}
		if operation == "" {
			operation = "open_extract_close"
		}
		openInput := map[string]any{"url": targetURL}
		extractInput := map[string]any{"url": targetURL}
		closeInput := map[string]any{"url": targetURL}
		if query != "" {
			extractInput["query"] = query
		}
		switch operation {
		case "open":
			return []plannedStep{
				{StepIndex: 0, Name: "Open browser URL", CapabilityKey: "browser_open", Input: cloneAnyMap(openInput)},
			}, nil
		case "extract":
			return []plannedStep{
				{StepIndex: 0, Name: "Extract browser URL", CapabilityKey: "browser_extract", Input: cloneAnyMap(extractInput)},
			}, nil
		case "close":
			return []plannedStep{
				{StepIndex: 0, Name: "Close browser URL", CapabilityKey: "browser_close", Input: cloneAnyMap(closeInput)},
			}, nil
		case "open_extract_close":
			return []plannedStep{
				{StepIndex: 0, Name: "Open browser URL", CapabilityKey: "browser_open", Input: cloneAnyMap(openInput)},
				{StepIndex: 1, Name: "Extract browser URL", CapabilityKey: "browser_extract", Input: cloneAnyMap(extractInput)},
				{StepIndex: 2, Name: "Close browser URL", CapabilityKey: "browser_close", Input: cloneAnyMap(closeInput)},
			}, nil
		default:
			return nil, fmt.Errorf("browser workflow requires supported operation")
		}
	case WorkflowFinder:
		targetPath := strings.TrimSpace(intent.TargetPath)
		query := ""
		rootPath := ""
		if intent.Action.Finder != nil && strings.TrimSpace(intent.Action.Finder.TargetPath) != "" {
			targetPath = strings.TrimSpace(intent.Action.Finder.TargetPath)
		}
		if intent.Action.Finder != nil {
			query = strings.TrimSpace(intent.Action.Finder.Query)
			rootPath = strings.TrimSpace(intent.Action.Finder.RootPath)
		}
		if targetPath != "" && !filepath.IsAbs(targetPath) {
			return nil, fmt.Errorf("finder workflow requires absolute target path")
		}
		operation := strings.ToLower(strings.TrimSpace(intent.Action.Operation))
		if intent.Action.Finder != nil && strings.TrimSpace(intent.Action.Finder.Operation) != "" {
			operation = strings.ToLower(strings.TrimSpace(intent.Action.Finder.Operation))
		}
		if operation == "" {
			if query != "" {
				operation = finderOperationFind
			} else {
				operation = finderOperationList
			}
		}
		if rootPath != "" && !filepath.IsAbs(rootPath) {
			return nil, fmt.Errorf("finder workflow requires absolute root_path")
		}
		input := map[string]any{}
		if targetPath != "" {
			input["path"] = targetPath
		}
		if query != "" {
			input["query"] = query
		}
		if rootPath != "" {
			input["root_path"] = rootPath
		}
		requirePathOrQuery := func() error {
			if targetPath == "" && query == "" {
				return fmt.Errorf("finder workflow requires target_path or query")
			}
			return nil
		}
		switch operation {
		case finderOperationFind:
			if query == "" {
				return nil, fmt.Errorf("finder find operation requires query")
			}
			return []plannedStep{
				{StepIndex: 0, Name: "Find path", CapabilityKey: "finder_find", Input: cloneAnyMap(input)},
			}, nil
		case finderOperationList:
			if err := requirePathOrQuery(); err != nil {
				return nil, err
			}
			return []plannedStep{
				{StepIndex: 0, Name: "List path", CapabilityKey: "finder_list", Input: cloneAnyMap(input)},
			}, nil
		case finderOperationPreview:
			if err := requirePathOrQuery(); err != nil {
				return nil, err
			}
			return []plannedStep{
				{StepIndex: 0, Name: "Preview path", CapabilityKey: "finder_preview", Input: cloneAnyMap(input)},
			}, nil
		case finderOperationDelete:
			if err := requirePathOrQuery(); err != nil {
				return nil, err
			}
			return []plannedStep{
				{StepIndex: 0, Name: "Delete path", CapabilityKey: "finder_delete", Input: cloneAnyMap(input)},
			}, nil
		default:
			return nil, fmt.Errorf("finder workflow requires supported operation")
		}
	case WorkflowMessages:
		if intent.Action.Messages == nil {
			return nil, fmt.Errorf("messages workflow requires action payload")
		}
		channel := normalizeMessageChannel(intent.Action.Messages.Channel)
		recipient := strings.TrimSpace(intent.Action.Messages.Recipient)
		body := strings.TrimSpace(intent.Action.Messages.Body)
		if channel == "" {
			return nil, fmt.Errorf("messages workflow requires message channel")
		}
		if recipient == "" {
			return nil, fmt.Errorf("messages workflow requires recipient")
		}
		if body == "" {
			return nil, fmt.Errorf("messages workflow requires message body")
		}

		capability := "messages_send_" + channel
		input := map[string]any{
			"channel":   channel,
			"recipient": recipient,
			"body":      body,
		}
		return []plannedStep{
			{
				StepIndex:        0,
				Name:             fmt.Sprintf("Send %s message", channel),
				CapabilityKey:    capability,
				Input:            cloneAnyMap(input),
				MessageChannel:   channel,
				MessageRecipient: recipient,
				MessageBody:      body,
			},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported workflow %q", intent.Workflow)
	}
}
