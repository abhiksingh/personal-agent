package cliapp

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"

	"personalagent/runtime/internal/transport"
)

func runConnectorCalendarDaemonCommand(
	ctx context.Context,
	client *transport.Client,
	args []string,
	correlationID string,
	stdout io.Writer,
	stderr io.Writer,
) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "connector calendar subcommand required: ingest|handoff")
		return 2
	}
	if client == nil && args[0] == "ingest" {
		fmt.Fprintln(stderr, "request failed: daemon client is not configured")
		return 1
	}

	switch args[0] {
	case "ingest":
		flags := flag.NewFlagSet("connector calendar ingest", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		sourceScope := flags.String("source-scope", "", "calendar source scope identifier (for example calendar id)")
		sourceEventID := flags.String("source-event-id", "", "idempotency event id for the calendar change")
		sourceCursor := flags.String("source-cursor", "", "optional monotonic source cursor")
		calendarID := flags.String("calendar-id", "", "calendar identifier")
		calendarName := flags.String("calendar-name", "", "calendar display name")
		eventUID := flags.String("event-uid", "", "calendar event UID")
		changeType := flags.String("change-type", "updated", "change type: created|updated|deleted")
		title := flags.String("title", "", "calendar event title")
		notes := flags.String("notes", "", "calendar event notes")
		location := flags.String("location", "", "calendar event location")
		startsAt := flags.String("starts-at", "", "event start timestamp RFC3339")
		endsAt := flags.String("ends-at", "", "event end timestamp RFC3339")
		occurredAt := flags.String("occurred-at", "", "change timestamp RFC3339")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}

		if strings.TrimSpace(*sourceEventID) == "" && strings.TrimSpace(*eventUID) == "" {
			fmt.Fprintln(stderr, "request failed: --source-event-id or --event-uid is required")
			return 1
		}

		response, err := client.CommCalendarIngest(ctx, transport.CalendarChangeIngestRequest{
			WorkspaceID:   normalizeWorkspace(*workspaceID),
			SourceScope:   strings.TrimSpace(*sourceScope),
			SourceEventID: strings.TrimSpace(*sourceEventID),
			SourceCursor:  strings.TrimSpace(*sourceCursor),
			CalendarID:    strings.TrimSpace(*calendarID),
			CalendarName:  strings.TrimSpace(*calendarName),
			EventUID:      strings.TrimSpace(*eventUID),
			ChangeType:    strings.TrimSpace(*changeType),
			Title:         strings.TrimSpace(*title),
			Notes:         strings.TrimSpace(*notes),
			Location:      strings.TrimSpace(*location),
			StartsAt:      strings.TrimSpace(*startsAt),
			EndsAt:        strings.TrimSpace(*endsAt),
			OccurredAt:    strings.TrimSpace(*occurredAt),
		}, correlationID)
		if err != nil {
			return writeError(stderr, err)
		}
		if code := writeJSON(stdout, response); code != 0 {
			return code
		}
		if !response.Accepted {
			return 1
		}
		return 0
	case "handoff":
		flags := flag.NewFlagSet("connector calendar handoff", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		inboxDir := flags.String("inbox-dir", "", "optional local ingress bridge inbox root override")
		sourceScope := flags.String("source-scope", "", "calendar source scope identifier (for example calendar id)")
		sourceEventID := flags.String("source-event-id", "", "idempotency event id for the calendar change")
		sourceCursor := flags.String("source-cursor", "", "optional monotonic source cursor")
		calendarID := flags.String("calendar-id", "", "calendar identifier")
		calendarName := flags.String("calendar-name", "", "calendar display name")
		eventUID := flags.String("event-uid", "", "calendar event UID")
		changeType := flags.String("change-type", "updated", "change type: created|updated|deleted")
		title := flags.String("title", "", "calendar event title")
		notes := flags.String("notes", "", "calendar event notes")
		location := flags.String("location", "", "calendar event location")
		startsAt := flags.String("starts-at", "", "event start timestamp RFC3339")
		endsAt := flags.String("ends-at", "", "event end timestamp RFC3339")
		occurredAt := flags.String("occurred-at", "", "change timestamp RFC3339")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}

		if strings.TrimSpace(*sourceEventID) == "" && strings.TrimSpace(*eventUID) == "" {
			fmt.Fprintln(stderr, "request failed: --source-event-id or --event-uid is required")
			return 1
		}

		request := transport.CalendarChangeIngestRequest{
			WorkspaceID:   normalizeWorkspace(*workspaceID),
			SourceScope:   strings.TrimSpace(*sourceScope),
			SourceEventID: strings.TrimSpace(*sourceEventID),
			SourceCursor:  strings.TrimSpace(*sourceCursor),
			CalendarID:    strings.TrimSpace(*calendarID),
			CalendarName:  strings.TrimSpace(*calendarName),
			EventUID:      strings.TrimSpace(*eventUID),
			ChangeType:    strings.TrimSpace(*changeType),
			Title:         strings.TrimSpace(*title),
			Notes:         strings.TrimSpace(*notes),
			Location:      strings.TrimSpace(*location),
			StartsAt:      strings.TrimSpace(*startsAt),
			EndsAt:        strings.TrimSpace(*endsAt),
			OccurredAt:    strings.TrimSpace(*occurredAt),
		}
		eventID := request.SourceEventID
		if strings.TrimSpace(eventID) == "" {
			eventID = request.EventUID
		}

		response, err := enqueueLocalIngressBridgePayload(
			ctx,
			request.WorkspaceID,
			"calendar",
			eventID,
			request,
			strings.TrimSpace(*inboxDir),
		)
		if err != nil {
			fmt.Fprintf(stderr, "request failed: %v\n", err)
			return 1
		}
		if code := writeJSON(stdout, response); code != 0 {
			return code
		}
		if !response.Queued {
			return 1
		}
		return 0
	default:
		writeUnknownSubcommandError(stderr, "connector calendar subcommand", args[0])
		return 2
	}
}
