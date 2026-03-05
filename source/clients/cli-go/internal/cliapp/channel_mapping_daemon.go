package cliapp

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"

	"personalagent/runtime/internal/transport"
)

func runChannelMappingDaemonCommand(ctx context.Context, client *transport.Client, args []string, correlationID string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "channel mapping subcommand required: list|enable|disable|prioritize")
		return 2
	}
	if client == nil {
		fmt.Fprintln(stderr, "request failed: daemon client is not configured")
		return 1
	}

	switch args[0] {
	case "list":
		flags := flag.NewFlagSet("channel mapping list", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		channelID := flags.String("channel", "", "logical channel filter (app|message|voice)")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}

		response, err := client.ChannelConnectorMappingsList(ctx, transport.ChannelConnectorMappingListRequest{
			WorkspaceID: normalizeWorkspace(*workspaceID),
			ChannelID:   strings.TrimSpace(*channelID),
		}, correlationID)
		if err != nil {
			return writeError(stderr, err)
		}
		return writeJSON(stdout, response)
	case "enable":
		flags := flag.NewFlagSet("channel mapping enable", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		channelID := flags.String("channel", "", "logical channel id (app|message|voice)")
		connectorID := flags.String("connector", "", "connector id (for example builtin.app|imessage|twilio)")
		priority := flags.Int("priority", 0, "priority override (1=primary); defaults to existing or append")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}
		return runChannelMappingUpsertCommand(
			ctx,
			client,
			channelMappingMutationInput{
				WorkspaceID: normalizeWorkspace(*workspaceID),
				ChannelID:   strings.TrimSpace(*channelID),
				ConnectorID: strings.TrimSpace(*connectorID),
				Enabled:     true,
				Priority:    *priority,
				RequireRow:  false,
			},
			correlationID,
			stdout,
			stderr,
		)
	case "disable":
		flags := flag.NewFlagSet("channel mapping disable", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		channelID := flags.String("channel", "", "logical channel id (app|message|voice)")
		connectorID := flags.String("connector", "", "connector id")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}
		return runChannelMappingUpsertCommand(
			ctx,
			client,
			channelMappingMutationInput{
				WorkspaceID: normalizeWorkspace(*workspaceID),
				ChannelID:   strings.TrimSpace(*channelID),
				ConnectorID: strings.TrimSpace(*connectorID),
				Enabled:     false,
				Priority:    0,
				RequireRow:  true,
			},
			correlationID,
			stdout,
			stderr,
		)
	case "prioritize":
		flags := flag.NewFlagSet("channel mapping prioritize", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		channelID := flags.String("channel", "", "logical channel id (app|message|voice)")
		connectorID := flags.String("connector", "", "connector id")
		priority := flags.Int("priority", 0, "priority order where 1 is highest")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}
		if *priority <= 0 {
			fmt.Fprintln(stderr, "request failed: --priority must be >= 1")
			return 1
		}
		return runChannelMappingUpsertCommand(
			ctx,
			client,
			channelMappingMutationInput{
				WorkspaceID: normalizeWorkspace(*workspaceID),
				ChannelID:   strings.TrimSpace(*channelID),
				ConnectorID: strings.TrimSpace(*connectorID),
				Priority:    *priority,
				OnlyReorder: true,
				RequireRow:  true,
			},
			correlationID,
			stdout,
			stderr,
		)
	default:
		writeUnknownSubcommandError(stderr, "channel mapping subcommand", args[0])
		return 2
	}
}

type channelMappingMutationInput struct {
	WorkspaceID string
	ChannelID   string
	ConnectorID string
	Enabled     bool
	Priority    int
	OnlyReorder bool
	RequireRow  bool
}

func runChannelMappingUpsertCommand(
	ctx context.Context,
	client *transport.Client,
	input channelMappingMutationInput,
	correlationID string,
	stdout io.Writer,
	stderr io.Writer,
) int {
	channelID := strings.TrimSpace(input.ChannelID)
	connectorID := normalizeCLIChannelMappingConnectorID(input.ConnectorID)
	if channelID == "" || connectorID == "" {
		fmt.Fprintln(stderr, "request failed: --channel and --connector are required")
		return 1
	}

	current, err := client.ChannelConnectorMappingsList(ctx, transport.ChannelConnectorMappingListRequest{
		WorkspaceID: input.WorkspaceID,
		ChannelID:   channelID,
	}, correlationID)
	if err != nil {
		return writeError(stderr, err)
	}

	binding, found := findChannelConnectorBinding(current.Bindings, connectorID)
	if input.RequireRow && !found {
		fmt.Fprintf(stderr, "request failed: connector %q is not currently mapped to channel %q\n", connectorID, strings.TrimSpace(channelID))
		return 1
	}

	resolvedPriority := input.Priority
	resolvedEnabled := input.Enabled
	switch {
	case input.OnlyReorder:
		if !found {
			fmt.Fprintf(stderr, "request failed: connector %q is not currently mapped to channel %q\n", connectorID, strings.TrimSpace(channelID))
			return 1
		}
		resolvedEnabled = binding.Enabled
	case input.Enabled:
		if resolvedPriority <= 0 {
			if found {
				resolvedPriority = binding.Priority
			} else {
				resolvedPriority = len(current.Bindings) + 1
			}
		}
	default:
		if found {
			resolvedPriority = binding.Priority
		}
	}
	if resolvedPriority <= 0 {
		resolvedPriority = len(current.Bindings) + 1
	}

	response, err := client.ChannelConnectorMappingUpsert(ctx, transport.ChannelConnectorMappingUpsertRequest{
		WorkspaceID: input.WorkspaceID,
		ChannelID:   channelID,
		ConnectorID: connectorID,
		Enabled:     resolvedEnabled,
		Priority:    resolvedPriority,
	}, correlationID)
	if err != nil {
		return writeError(stderr, err)
	}
	return writeJSON(stdout, response)
}

func normalizeCLIChannelMappingConnectorID(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

func findChannelConnectorBinding(bindings []transport.ChannelConnectorMappingRecord, connectorID string) (transport.ChannelConnectorMappingRecord, bool) {
	normalizedTarget := normalizeCLIChannelMappingConnectorID(connectorID)
	for _, binding := range bindings {
		if normalizeCLIChannelMappingConnectorID(binding.ConnectorID) == normalizedTarget {
			return binding, true
		}
	}
	return transport.ChannelConnectorMappingRecord{}, false
}
