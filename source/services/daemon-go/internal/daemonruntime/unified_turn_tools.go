package daemonruntime

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"personalagent/runtime/internal/transport"
)

func (s *UnifiedTurnService) resolveAvailableTools(ctx context.Context, workspaceID string) ([]modelToolDefinition, error) {
	availableCapabilities := map[string]struct{}{}
	capabilityInventoryAvailable := false
	if s.uiStatus != nil {
		connectors, err := s.uiStatus.ListConnectorStatus(ctx, transport.ConnectorStatusRequest{WorkspaceID: workspaceID})
		if err != nil {
			return nil, err
		}
		capabilityInventoryAvailable = true
		for _, connector := range connectors.Connectors {
			if !toolCapabilityInventoryReady(connector.ActionReadiness) {
				continue
			}
			for _, capability := range connector.Capabilities {
				normalized := normalizeToolCapabilityKey(capability)
				if normalized == "" {
					continue
				}
				availableCapabilities[normalized] = struct{}{}
			}
		}
	}

	if !capabilityInventoryAvailable {
		return allSupportedModelTools(), nil
	}
	return deriveModelToolsFromCapabilities(availableCapabilities), nil
}

func toolCapabilityInventoryReady(actionReadiness string) bool {
	normalized := strings.ToLower(strings.TrimSpace(actionReadiness))
	return normalized == "" || normalized == "ready"
}

func allSupportedModelTools() []modelToolDefinition {
	allCapabilities := map[string]struct{}{}
	for _, capability := range supportedModelToolCapabilities {
		normalized := normalizeToolCapabilityKey(capability)
		if normalized == "" {
			continue
		}
		allCapabilities[normalized] = struct{}{}
	}
	return deriveModelToolsFromCapabilities(allCapabilities)
}

func deriveModelToolsFromCapabilities(capabilities map[string]struct{}) []modelToolDefinition {
	if len(capabilities) == 0 {
		return []modelToolDefinition{}
	}
	capabilityList := make([]string, 0, len(capabilities))
	for capability := range capabilities {
		capabilityList = append(capabilityList, capability)
	}
	sort.Strings(capabilityList)

	toolByName := map[string]modelToolDefinition{}
	for _, capability := range capabilityList {
		tool, ok := modelToolDefinitionFromCapability(capability)
		if !ok {
			continue
		}
		existing, found := toolByName[tool.Name]
		if !found {
			toolByName[tool.Name] = tool
			continue
		}
		toolByName[tool.Name] = mergeModelToolDefinitions(existing, tool)
	}

	resolved := make([]modelToolDefinition, 0, len(toolByName))
	for _, tool := range toolByName {
		resolved = append(resolved, tool)
	}
	sort.Slice(resolved, func(i, j int) bool {
		return resolved[i].Name < resolved[j].Name
	})
	return resolved
}

func mergeModelToolDefinitions(existing modelToolDefinition, incoming modelToolDefinition) modelToolDefinition {
	merged := existing
	merged.CapabilityKeys = mergeToolCapabilityKeys(existing.CapabilityKeys, incoming.CapabilityKeys)
	if strings.TrimSpace(merged.Description) == "" {
		merged.Description = strings.TrimSpace(incoming.Description)
	}
	if len(merged.Arguments) == 0 && len(incoming.Arguments) > 0 {
		merged.Arguments = cloneToolArgumentSpecs(incoming.Arguments)
	}
	if merged.BuildNativeAction == nil {
		merged.BuildNativeAction = incoming.BuildNativeAction
	}
	return merged
}

func mergeToolCapabilityKeys(left []string, right []string) []string {
	keySet := map[string]struct{}{}
	for _, key := range append(append([]string{}, left...), right...) {
		normalized := normalizeToolCapabilityKey(key)
		if normalized == "" {
			continue
		}
		keySet[normalized] = struct{}{}
	}
	merged := make([]string, 0, len(keySet))
	for key := range keySet {
		merged = append(merged, key)
	}
	sort.Strings(merged)
	return merged
}

func normalizeToolCapabilityKey(capability string) string {
	normalized := strings.ToLower(strings.TrimSpace(capability))
	if normalized == "" {
		return ""
	}
	normalized = strings.ReplaceAll(normalized, ".", "_")
	normalized = strings.ReplaceAll(normalized, "-", "_")
	return normalized
}

func cloneToolArgumentSpecs(arguments map[string]toolArgumentSpec) map[string]toolArgumentSpec {
	if len(arguments) == 0 {
		return map[string]toolArgumentSpec{}
	}
	cloned := make(map[string]toolArgumentSpec, len(arguments))
	for key, spec := range arguments {
		cloned[key] = toolArgumentSpec{
			Type:        strings.TrimSpace(spec.Type),
			Required:    spec.Required,
			EnumOptions: append([]string{}, spec.EnumOptions...),
			Description: strings.TrimSpace(spec.Description),
		}
	}
	return cloned
}

func modelToolDefinitionFromCapability(capability string) (modelToolDefinition, bool) {
	normalizedCapability := normalizeToolCapabilityKey(capability)
	switch normalizedCapability {
	case "mail_draft":
		return modelToolDefinition{
			Name:           "mail_draft",
			CapabilityKeys: []string{normalizedCapability},
			Description:    "Draft an email through the mail connector workflow.",
			Arguments: map[string]toolArgumentSpec{
				"recipient": {Type: "string", Required: false, Description: "Recipient email address."},
				"subject":   {Type: "string", Required: false, Description: "Email subject line."},
				"body":      {Type: "string", Required: true, Description: "Email body content."},
			},
			BuildNativeAction: func(arguments map[string]any) (*transport.AgentNativeAction, error) {
				return buildMailNativeAction("draft", arguments)
			},
		}, true
	case "mail_send":
		return modelToolDefinition{
			Name:           "mail_send",
			CapabilityKeys: []string{normalizedCapability},
			Description:    "Send an email through the mail connector workflow.",
			Arguments: map[string]toolArgumentSpec{
				"recipient": {Type: "string", Required: true, Description: "Recipient email address."},
				"subject":   {Type: "string", Required: false, Description: "Email subject line."},
				"body":      {Type: "string", Required: true, Description: "Email body content."},
			},
			BuildNativeAction: func(arguments map[string]any) (*transport.AgentNativeAction, error) {
				return buildMailNativeAction("send", arguments)
			},
		}, true
	case "mail_reply":
		return modelToolDefinition{
			Name:           "mail_reply",
			CapabilityKeys: []string{normalizedCapability},
			Description:    "Reply to an email thread through the mail connector workflow.",
			Arguments: map[string]toolArgumentSpec{
				"recipient": {Type: "string", Required: true, Description: "Reply recipient email address."},
				"subject":   {Type: "string", Required: false, Description: "Optional subject override."},
				"body":      {Type: "string", Required: true, Description: "Reply body content."},
			},
			BuildNativeAction: func(arguments map[string]any) (*transport.AgentNativeAction, error) {
				return buildMailNativeAction("reply", arguments)
			},
		}, true
	case "mail_unread_summary":
		return modelToolDefinition{
			Name:           "mail_unread_summary",
			CapabilityKeys: []string{normalizedCapability},
			Description:    "Summarize unread inbox messages from persisted mail events.",
			Arguments: map[string]toolArgumentSpec{
				"limit": {Type: "integer", Required: false, Description: "Optional maximum unread messages to summarize (1-50)."},
			},
			BuildNativeAction: func(arguments map[string]any) (*transport.AgentNativeAction, error) {
				return buildMailNativeAction("summarize_unread", arguments)
			},
		}, true
	case "calendar_create", "calendar_update", "calendar_cancel":
		operation := strings.TrimPrefix(normalizedCapability, "calendar_")
		description := map[string]string{
			"create": "Create a calendar event.",
			"update": "Update a calendar event.",
			"cancel": "Cancel a calendar event.",
		}[operation]
		arguments := map[string]toolArgumentSpec{}
		switch operation {
		case "create":
			arguments["title"] = toolArgumentSpec{Type: "string", Required: true, Description: "Event title."}
			arguments["notes"] = toolArgumentSpec{Type: "string", Required: false, Description: "Optional event notes."}
		case "update":
			arguments["event_id"] = toolArgumentSpec{Type: "string", Required: true, Description: "Stable event identifier returned by calendar_create."}
			arguments["title"] = toolArgumentSpec{Type: "string", Required: false, Description: "Optional updated event title."}
			arguments["notes"] = toolArgumentSpec{Type: "string", Required: false, Description: "Optional updated event notes."}
		case "cancel":
			arguments["event_id"] = toolArgumentSpec{Type: "string", Required: true, Description: "Stable event identifier returned by calendar_create."}
		}
		return modelToolDefinition{
			Name:           normalizedCapability,
			CapabilityKeys: []string{normalizedCapability},
			Description:    description,
			Arguments:      arguments,
			BuildNativeAction: func(arguments map[string]any) (*transport.AgentNativeAction, error) {
				return buildCalendarNativeAction(operation, arguments)
			},
		}, true
	case "browser_open", "browser_extract", "browser_close":
		operation := strings.TrimPrefix(normalizedCapability, "browser_")
		description := map[string]string{
			"open":    "Open a web page.",
			"extract": "Extract structured content from a web page and optionally answer a query from that content.",
			"close":   "Close a web page session.",
		}[operation]
		arguments := map[string]toolArgumentSpec{
			"url": {Type: "string", Required: true, Description: "Target URL (http/https)."},
		}
		if operation == "extract" {
			arguments["query"] = toolArgumentSpec{Type: "string", Required: false, Description: "Optional question to answer using extracted page content."}
		}
		return modelToolDefinition{
			Name:           normalizedCapability,
			CapabilityKeys: []string{normalizedCapability},
			Description:    description,
			Arguments:      arguments,
			BuildNativeAction: func(arguments map[string]any) (*transport.AgentNativeAction, error) {
				return buildBrowserNativeAction(operation, arguments)
			},
		}, true
	case "finder_find", "finder_list", "finder_preview", "finder_delete":
		operation := strings.TrimPrefix(normalizedCapability, "finder_")
		description := map[string]string{
			"find":    "Find files or folders from a query.",
			"list":    "List files for an absolute path or query-resolved target.",
			"preview": "Preview file or folder details before deletion.",
			"delete":  "Delete a file or folder at an absolute path or query-resolved target.",
		}[operation]
		arguments := map[string]toolArgumentSpec{
			"path":      {Type: "string", Required: false, Description: "Absolute filesystem path."},
			"query":     {Type: "string", Required: false, Description: "Finder query used to resolve candidate paths."},
			"root_path": {Type: "string", Required: false, Description: "Optional absolute search root for query resolution."},
		}
		if operation == "find" {
			arguments["query"] = toolArgumentSpec{Type: "string", Required: true, Description: "Finder query used to resolve candidate paths."}
		}
		return modelToolDefinition{
			Name:           normalizedCapability,
			CapabilityKeys: []string{normalizedCapability},
			Description:    description,
			Arguments:      arguments,
			BuildNativeAction: func(arguments map[string]any) (*transport.AgentNativeAction, error) {
				return buildFinderNativeAction(operation, arguments)
			},
		}, true
	case "channel_messages_send", "channel_twilio_sms_send", "messages_send_imessage", "messages_send_sms":
		return modelToolDefinition{
			Name:           "message_send",
			CapabilityKeys: []string{normalizedCapability},
			Description:    "Send a message over iMessage or SMS.",
			Arguments: map[string]toolArgumentSpec{
				"channel":   {Type: "string", Required: true, EnumOptions: []string{"message", "imessage", "sms"}, Description: "Message channel (`message`, `imessage`, or `sms`)."},
				"recipient": {Type: "string", Required: true, Description: "Recipient handle."},
				"body":      {Type: "string", Required: true, Description: "Message body."},
			},
			BuildNativeAction: func(arguments map[string]any) (*transport.AgentNativeAction, error) {
				return buildMessageNativeAction(arguments)
			},
		}, true
	default:
		return modelToolDefinition{}, false
	}
}

func buildMailNativeAction(operation string, arguments map[string]any) (*transport.AgentNativeAction, error) {
	normalizedOperation := strings.ToLower(strings.TrimSpace(operation))
	if normalizedOperation == "" {
		return nil, fmt.Errorf("mail operation is required")
	}
	switch normalizedOperation {
	case "draft", "send", "reply":
		body, err := requiredStringArgument(arguments, "body")
		if err != nil {
			return nil, err
		}
		recipient := optionalStringArgument(arguments, "recipient")
		subject := optionalStringArgument(arguments, "subject")
		if (normalizedOperation == "send" || normalizedOperation == "reply") && recipient == "" {
			return nil, fmt.Errorf("missing required argument recipient")
		}
		return &transport.AgentNativeAction{
			Connector: "mail",
			Operation: normalizedOperation,
			Mail: &transport.AgentMailAction{
				Operation: normalizedOperation,
				Recipient: recipient,
				Subject:   subject,
				Body:      body,
			},
		}, nil
	case "summarize_unread":
		limit, err := optionalIntegerArgument(arguments, "limit")
		if err != nil {
			return nil, err
		}
		if limit < 0 {
			return nil, fmt.Errorf("argument limit must be non-negative")
		}
		if limit > 50 {
			limit = 50
		}
		return &transport.AgentNativeAction{
			Connector: "mail",
			Operation: normalizedOperation,
			Mail: &transport.AgentMailAction{
				Operation: normalizedOperation,
				Limit:     limit,
			},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported mail operation %s", operation)
	}
}

func buildCalendarNativeAction(operation string, arguments map[string]any) (*transport.AgentNativeAction, error) {
	normalizedOperation := strings.ToLower(strings.TrimSpace(operation))
	switch normalizedOperation {
	case "create", "update", "cancel":
	default:
		return nil, fmt.Errorf("unsupported calendar operation %s", operation)
	}
	eventID := optionalStringArgument(arguments, "event_id")
	title := optionalStringArgument(arguments, "title")
	notes := optionalStringArgument(arguments, "notes")
	switch normalizedOperation {
	case "create":
		if title == "" {
			return nil, fmt.Errorf("missing required argument title")
		}
	case "update":
		if eventID == "" {
			return nil, fmt.Errorf("missing required argument event_id")
		}
		if title == "" && notes == "" {
			return nil, fmt.Errorf("calendar update requires title or notes")
		}
	case "cancel":
		if eventID == "" {
			return nil, fmt.Errorf("missing required argument event_id")
		}
	}
	return &transport.AgentNativeAction{
		Connector: "calendar",
		Operation: normalizedOperation,
		Calendar: &transport.AgentCalendarAction{
			Operation: normalizedOperation,
			EventID:   eventID,
			Title:     title,
			Notes:     notes,
		},
	}, nil
}

func buildBrowserNativeAction(operation string, arguments map[string]any) (*transport.AgentNativeAction, error) {
	targetURL, err := requiredStringArgument(arguments, "url")
	if err != nil {
		return nil, err
	}
	query := optionalStringArgument(arguments, "query")
	normalizedOperation := strings.ToLower(strings.TrimSpace(operation))
	switch normalizedOperation {
	case "open", "extract", "close":
	default:
		return nil, fmt.Errorf("unsupported browser operation %s", operation)
	}
	if normalizedOperation != "extract" {
		query = ""
	}
	return &transport.AgentNativeAction{
		Connector: "browser",
		Operation: normalizedOperation,
		Browser: &transport.AgentBrowserAction{
			Operation: normalizedOperation,
			TargetURL: targetURL,
			Query:     query,
		},
	}, nil
}

func buildFinderNativeAction(operation string, arguments map[string]any) (*transport.AgentNativeAction, error) {
	targetPath := optionalStringArgument(arguments, "path")
	query := optionalStringArgument(arguments, "query")
	rootPath := optionalStringArgument(arguments, "root_path")
	normalizedOperation := strings.ToLower(strings.TrimSpace(operation))
	switch normalizedOperation {
	case "find", "list", "preview", "delete":
	default:
		return nil, fmt.Errorf("unsupported finder operation %s", operation)
	}
	if normalizedOperation == "find" {
		if query == "" {
			return nil, fmt.Errorf("missing required argument query")
		}
		targetPath = ""
	} else if targetPath == "" && query == "" {
		return nil, fmt.Errorf("finder %s requires path or query", normalizedOperation)
	}
	return &transport.AgentNativeAction{
		Connector: "finder",
		Operation: normalizedOperation,
		Finder: &transport.AgentFinderAction{
			Operation:  normalizedOperation,
			TargetPath: targetPath,
			Query:      query,
			RootPath:   rootPath,
		},
	}, nil
}

func buildMessageNativeAction(arguments map[string]any) (*transport.AgentNativeAction, error) {
	channel, err := requiredStringArgument(arguments, "channel")
	if err != nil {
		return nil, err
	}
	normalizedChannel, err := normalizeMessageToolChannel(channel)
	if err != nil {
		return nil, err
	}
	recipient, err := requiredStringArgument(arguments, "recipient")
	if err != nil {
		return nil, err
	}
	body, err := requiredStringArgument(arguments, "body")
	if err != nil {
		return nil, err
	}
	return &transport.AgentNativeAction{
		Connector: "messages",
		Operation: "send_message",
		Messages: &transport.AgentMessagesAction{
			Operation: "send_message",
			Channel:   normalizedChannel,
			Recipient: recipient,
			Body:      body,
		},
	}, nil
}

func normalizeMessageToolChannel(channel string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(channel)) {
	case "imessage":
		return "imessage", nil
	case "sms":
		return "sms", nil
	case "message":
		return "imessage", nil
	default:
		return "", fmt.Errorf("unsupported message channel %q (allowed: message|imessage|sms)", strings.TrimSpace(channel))
	}
}
