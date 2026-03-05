package cliapp

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"

	"personalagent/runtime/internal/transport"
)

func runConnectorMailDaemonCommand(
	ctx context.Context,
	client *transport.Client,
	args []string,
	correlationID string,
	stdout io.Writer,
	stderr io.Writer,
) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "connector mail subcommand required: ingest|handoff")
		return 2
	}
	if client == nil && args[0] == "ingest" {
		fmt.Fprintln(stderr, "request failed: daemon client is not configured")
		return 1
	}

	switch args[0] {
	case "ingest":
		flags := flag.NewFlagSet("connector mail ingest", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		sourceScope := flags.String("source-scope", "", "mail source scope identifier (for example mailbox/rule id)")
		sourceEventID := flags.String("source-event-id", "", "idempotency event id for the mail-rule event")
		sourceCursor := flags.String("source-cursor", "", "optional monotonic source cursor")
		messageID := flags.String("message-id", "", "email message-id header value")
		threadRef := flags.String("thread-ref", "", "external thread reference")
		inReplyTo := flags.String("in-reply-to", "", "in-reply-to header value")
		referencesHeader := flags.String("references-header", "", "references header value")
		fromAddress := flags.String("from", "", "sender email address")
		toAddress := flags.String("to", "", "recipient email address")
		subject := flags.String("subject", "", "email subject")
		body := flags.String("body", "", "email body text")
		occurredAt := flags.String("occurred-at", "", "event timestamp RFC3339")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}

		if strings.TrimSpace(*fromAddress) == "" {
			fmt.Fprintln(stderr, "request failed: --from is required")
			return 1
		}

		response, err := client.CommMailRuleIngest(ctx, transport.MailRuleIngestRequest{
			WorkspaceID:      normalizeWorkspace(*workspaceID),
			SourceScope:      strings.TrimSpace(*sourceScope),
			SourceEventID:    strings.TrimSpace(*sourceEventID),
			SourceCursor:     strings.TrimSpace(*sourceCursor),
			MessageID:        strings.TrimSpace(*messageID),
			ThreadRef:        strings.TrimSpace(*threadRef),
			InReplyTo:        strings.TrimSpace(*inReplyTo),
			ReferencesHeader: strings.TrimSpace(*referencesHeader),
			FromAddress:      strings.TrimSpace(*fromAddress),
			ToAddress:        strings.TrimSpace(*toAddress),
			Subject:          strings.TrimSpace(*subject),
			BodyText:         strings.TrimSpace(*body),
			OccurredAt:       strings.TrimSpace(*occurredAt),
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
		flags := flag.NewFlagSet("connector mail handoff", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		inboxDir := flags.String("inbox-dir", "", "optional local ingress bridge inbox root override")
		sourceScope := flags.String("source-scope", "", "mail source scope identifier (for example mailbox/rule id)")
		sourceEventID := flags.String("source-event-id", "", "idempotency event id for the mail-rule event")
		sourceCursor := flags.String("source-cursor", "", "optional monotonic source cursor")
		messageID := flags.String("message-id", "", "email message-id header value")
		threadRef := flags.String("thread-ref", "", "external thread reference")
		inReplyTo := flags.String("in-reply-to", "", "in-reply-to header value")
		referencesHeader := flags.String("references-header", "", "references header value")
		fromAddress := flags.String("from", "", "sender email address")
		toAddress := flags.String("to", "", "recipient email address")
		subject := flags.String("subject", "", "email subject")
		body := flags.String("body", "", "email body text")
		occurredAt := flags.String("occurred-at", "", "event timestamp RFC3339")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}

		if strings.TrimSpace(*fromAddress) == "" {
			fmt.Fprintln(stderr, "request failed: --from is required")
			return 1
		}
		if strings.TrimSpace(*sourceEventID) == "" {
			fmt.Fprintln(stderr, "request failed: --source-event-id is required")
			return 1
		}

		request := transport.MailRuleIngestRequest{
			WorkspaceID:      normalizeWorkspace(*workspaceID),
			SourceScope:      strings.TrimSpace(*sourceScope),
			SourceEventID:    strings.TrimSpace(*sourceEventID),
			SourceCursor:     strings.TrimSpace(*sourceCursor),
			MessageID:        strings.TrimSpace(*messageID),
			ThreadRef:        strings.TrimSpace(*threadRef),
			InReplyTo:        strings.TrimSpace(*inReplyTo),
			ReferencesHeader: strings.TrimSpace(*referencesHeader),
			FromAddress:      strings.TrimSpace(*fromAddress),
			ToAddress:        strings.TrimSpace(*toAddress),
			Subject:          strings.TrimSpace(*subject),
			BodyText:         strings.TrimSpace(*body),
			OccurredAt:       strings.TrimSpace(*occurredAt),
		}

		response, err := enqueueLocalIngressBridgePayload(
			ctx,
			request.WorkspaceID,
			"mail",
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
		writeUnknownSubcommandError(stderr, "connector mail subcommand", args[0])
		return 2
	}
}
