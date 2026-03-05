package cliapp

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"

	"personalagent/runtime/internal/transport"
)

func runConnectorBrowserDaemonCommand(
	ctx context.Context,
	client *transport.Client,
	args []string,
	correlationID string,
	stdout io.Writer,
	stderr io.Writer,
) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "connector browser subcommand required: ingest|handoff")
		return 2
	}
	if client == nil && args[0] == "ingest" {
		fmt.Fprintln(stderr, "request failed: daemon client is not configured")
		return 1
	}

	switch args[0] {
	case "ingest":
		flags := flag.NewFlagSet("connector browser ingest", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		sourceScope := flags.String("source-scope", "", "browser source scope identifier (for example Safari window id)")
		sourceEventID := flags.String("source-event-id", "", "idempotency event id for the browser extension event")
		sourceCursor := flags.String("source-cursor", "", "optional monotonic source cursor")
		windowID := flags.String("window-id", "", "browser window identifier")
		tabID := flags.String("tab-id", "", "browser tab identifier")
		pageURL := flags.String("page-url", "", "browser page URL")
		pageTitle := flags.String("page-title", "", "browser page title")
		eventType := flags.String("event-type", "update", "event type: navigation|message|tab_opened|tab_closed|update")
		payloadText := flags.String("payload", "", "optional browser extension payload text")
		occurredAt := flags.String("occurred-at", "", "event timestamp RFC3339")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}

		if strings.TrimSpace(*sourceEventID) == "" {
			fmt.Fprintln(stderr, "request failed: --source-event-id is required")
			return 1
		}

		response, err := client.CommBrowserIngest(ctx, transport.BrowserEventIngestRequest{
			WorkspaceID:   normalizeWorkspace(*workspaceID),
			SourceScope:   strings.TrimSpace(*sourceScope),
			SourceEventID: strings.TrimSpace(*sourceEventID),
			SourceCursor:  strings.TrimSpace(*sourceCursor),
			WindowID:      strings.TrimSpace(*windowID),
			TabID:         strings.TrimSpace(*tabID),
			PageURL:       strings.TrimSpace(*pageURL),
			PageTitle:     strings.TrimSpace(*pageTitle),
			EventType:     strings.TrimSpace(*eventType),
			PayloadText:   strings.TrimSpace(*payloadText),
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
		flags := flag.NewFlagSet("connector browser handoff", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		inboxDir := flags.String("inbox-dir", "", "optional local ingress bridge inbox root override")
		sourceScope := flags.String("source-scope", "", "browser source scope identifier (for example Safari window id)")
		sourceEventID := flags.String("source-event-id", "", "idempotency event id for the browser extension event")
		sourceCursor := flags.String("source-cursor", "", "optional monotonic source cursor")
		windowID := flags.String("window-id", "", "browser window identifier")
		tabID := flags.String("tab-id", "", "browser tab identifier")
		pageURL := flags.String("page-url", "", "browser page URL")
		pageTitle := flags.String("page-title", "", "browser page title")
		eventType := flags.String("event-type", "update", "event type: navigation|message|tab_opened|tab_closed|update")
		payloadText := flags.String("payload", "", "optional browser extension payload text")
		occurredAt := flags.String("occurred-at", "", "event timestamp RFC3339")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}

		if strings.TrimSpace(*sourceEventID) == "" {
			fmt.Fprintln(stderr, "request failed: --source-event-id is required")
			return 1
		}

		request := transport.BrowserEventIngestRequest{
			WorkspaceID:   normalizeWorkspace(*workspaceID),
			SourceScope:   strings.TrimSpace(*sourceScope),
			SourceEventID: strings.TrimSpace(*sourceEventID),
			SourceCursor:  strings.TrimSpace(*sourceCursor),
			WindowID:      strings.TrimSpace(*windowID),
			TabID:         strings.TrimSpace(*tabID),
			PageURL:       strings.TrimSpace(*pageURL),
			PageTitle:     strings.TrimSpace(*pageTitle),
			EventType:     strings.TrimSpace(*eventType),
			PayloadText:   strings.TrimSpace(*payloadText),
			OccurredAt:    strings.TrimSpace(*occurredAt),
		}

		response, err := enqueueLocalIngressBridgePayload(
			ctx,
			request.WorkspaceID,
			"browser",
			request.SourceEventID,
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
		writeUnknownSubcommandError(stderr, "connector browser subcommand", args[0])
		return 2
	}
}
