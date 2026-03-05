package cliapp

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"

	"personalagent/runtime/internal/transport"
)

func runChannelMessagesDaemonCommand(ctx context.Context, client *transport.Client, args []string, correlationID string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "channel messages subcommand required: ingest")
		return 2
	}
	if client == nil {
		fmt.Fprintln(stderr, "request failed: daemon client is not configured")
		return 1
	}

	switch args[0] {
	case "ingest":
		flags := flag.NewFlagSet("channel messages ingest", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		sourceScope := flags.String("source-scope", "", "optional source scope identifier")
		sourceDBPath := flags.String("source-db-path", "", "optional Messages chat.db path override")
		limit := flags.Int("limit", 100, "max inbound message rows to poll")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}

		response, err := client.CommMessagesIngest(ctx, transport.MessagesIngestRequest{
			WorkspaceID:  normalizeWorkspace(*workspaceID),
			SourceScope:  strings.TrimSpace(*sourceScope),
			SourceDBPath: strings.TrimSpace(*sourceDBPath),
			Limit:        *limit,
		}, correlationID)
		if err != nil {
			return writeError(stderr, err)
		}
		return writeJSON(stdout, response)
	default:
		writeUnknownSubcommandError(stderr, "channel messages subcommand", args[0])
		return 2
	}
}
