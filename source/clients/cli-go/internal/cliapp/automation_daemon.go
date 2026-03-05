package cliapp

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"strings"

	"personalagent/runtime/internal/transport"
)

func runAutomationDaemonCommand(ctx context.Context, client *transport.Client, args []string, correlationID string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "automation subcommand required: create|list|run")
		return 2
	}

	switch args[0] {
	case "create":
		flags := flag.NewFlagSet("automation create", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		subjectActor := flags.String("subject", "actor.automation", "subject principal actor id")
		triggerType := flags.String("trigger-type", "", "trigger type: SCHEDULE|ON_COMM_EVENT")
		title := flags.String("title", "", "directive title")
		instruction := flags.String("instruction", "", "directive instruction")
		directiveID := flags.String("directive-id", "", "optional directive id")
		triggerID := flags.String("trigger-id", "", "optional trigger id")
		intervalSeconds := flags.Int("interval-seconds", 300, "schedule interval in seconds (SCHEDULE)")
		filterRaw := flags.String("filter", "", "optional ON_COMM_EVENT filter json object")
		cooldownSeconds := flags.Int("cooldown-seconds", 0, "optional cooldown in seconds")
		enabled := flags.Bool("enabled", true, "enable trigger")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}
		filter, err := parseAutomationCommTriggerFilterFlag(strings.TrimSpace(*filterRaw))
		if err != nil {
			fmt.Fprintf(stderr, "request failed: %v\n", err)
			return 1
		}

		response, err := client.AutomationCreate(ctx, transport.AutomationCreateRequest{
			WorkspaceID:     normalizeWorkspace(*workspaceID),
			SubjectActorID:  strings.TrimSpace(*subjectActor),
			TriggerType:     strings.TrimSpace(*triggerType),
			Title:           strings.TrimSpace(*title),
			Instruction:     strings.TrimSpace(*instruction),
			DirectiveID:     strings.TrimSpace(*directiveID),
			TriggerID:       strings.TrimSpace(*triggerID),
			IntervalSeconds: *intervalSeconds,
			Filter:          filter,
			CooldownSeconds: *cooldownSeconds,
			Enabled:         *enabled,
		}, correlationID)
		if err != nil {
			return writeError(stderr, err)
		}
		return writeJSON(stdout, response)
	case "list":
		flags := flag.NewFlagSet("automation list", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		triggerType := flags.String("trigger-type", "", "optional trigger type filter")
		includeDisabled := flags.Bool("include-disabled", true, "include disabled triggers")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}

		response, err := client.AutomationList(ctx, transport.AutomationListRequest{
			WorkspaceID:     normalizeWorkspace(*workspaceID),
			TriggerType:     strings.TrimSpace(*triggerType),
			IncludeDisabled: *includeDisabled,
		}, correlationID)
		if err != nil {
			return writeError(stderr, err)
		}
		return writeJSON(stdout, response)
	case "run":
		return runAutomationDaemonRunCommand(ctx, client, args[1:], correlationID, stdout, stderr)
	default:
		writeUnknownSubcommandError(stderr, "automation subcommand", args[0])
		return 2
	}
}

func parseAutomationCommTriggerFilterFlag(raw string) (*transport.AutomationCommTriggerFilter, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	var filter transport.AutomationCommTriggerFilter
	if err := json.Unmarshal([]byte(trimmed), &filter); err != nil {
		return nil, fmt.Errorf("--filter must be valid ON_COMM_EVENT filter json: %w", err)
	}
	return &filter, nil
}

func runAutomationDaemonRunCommand(ctx context.Context, client *transport.Client, args []string, correlationID string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "automation run subcommand required: schedule|comm-event")
		return 2
	}

	switch strings.ToLower(strings.TrimSpace(args[0])) {
	case "schedule":
		flags := flag.NewFlagSet("automation run schedule", flag.ContinueOnError)
		flags.SetOutput(stderr)

		at := flags.String("at", "", "optional RFC3339 timestamp")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}

		response, err := client.AutomationRunSchedule(ctx, transport.AutomationRunScheduleRequest{
			At: strings.TrimSpace(*at),
		}, correlationID)
		if err != nil {
			return writeError(stderr, err)
		}
		return writeJSON(stdout, response)
	case "comm-event", "on-comm-event":
		flags := flag.NewFlagSet("automation run comm-event", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id for seeded event")
		eventID := flags.String("event-id", "", "event id to evaluate")
		seedEvent := flags.Bool("seed-event", true, "seed/update a synthetic event before evaluation")
		threadID := flags.String("thread-id", "", "thread id for seeded event")
		channel := flags.String("channel", "imessage", "channel for seeded event")
		body := flags.String("body", "automation test event", "message body for seeded event")
		sender := flags.String("sender", "sender@example.com", "sender address for seeded event")
		eventType := flags.String("event-type", "MESSAGE", "event type for seeded event")
		direction := flags.String("direction", "INBOUND", "direction for seeded event")
		assistantEmitted := flags.Bool("assistant-emitted", false, "assistant emitted flag for seeded event")
		occurredAt := flags.String("occurred-at", "", "optional RFC3339 occurrence timestamp")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}

		response, err := client.AutomationRunCommEvent(ctx, transport.AutomationRunCommEventRequest{
			WorkspaceID:      normalizeWorkspace(*workspaceID),
			EventID:          strings.TrimSpace(*eventID),
			SeedEvent:        *seedEvent,
			ThreadID:         strings.TrimSpace(*threadID),
			Channel:          strings.TrimSpace(*channel),
			Body:             strings.TrimSpace(*body),
			Sender:           strings.TrimSpace(*sender),
			EventType:        strings.TrimSpace(*eventType),
			Direction:        strings.TrimSpace(*direction),
			AssistantEmitted: *assistantEmitted,
			OccurredAt:       strings.TrimSpace(*occurredAt),
		}, correlationID)
		if err != nil {
			return writeError(stderr, err)
		}
		return writeJSON(stdout, response)
	default:
		writeUnknownSubcommandError(stderr, "automation run subcommand", args[0])
		return 2
	}
}
