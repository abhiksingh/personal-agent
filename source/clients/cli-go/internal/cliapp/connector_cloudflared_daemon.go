package cliapp

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"personalagent/runtime/internal/transport"
)

type repeatableStringFlag []string

func (f *repeatableStringFlag) String() string {
	if f == nil || len(*f) == 0 {
		return ""
	}
	return strings.Join(*f, ",")
}

func (f *repeatableStringFlag) Set(value string) error {
	candidate := strings.TrimSpace(value)
	if candidate == "" {
		return nil
	}
	*f = append(*f, candidate)
	return nil
}

func runConnectorCloudflaredDaemonCommand(
	ctx context.Context,
	client *transport.Client,
	args []string,
	correlationID string,
	stdout io.Writer,
	stderr io.Writer,
) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "connector cloudflared subcommand required: version|exec")
		return 2
	}
	if client == nil {
		fmt.Fprintln(stderr, "request failed: daemon client is not configured")
		return 1
	}

	switch args[0] {
	case "version":
		flags := flag.NewFlagSet("connector cloudflared version", flag.ContinueOnError)
		flags.SetOutput(stderr)
		workspaceID := flags.String("workspace", "", "workspace id")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}

		response, err := client.CloudflaredVersion(ctx, transport.CloudflaredVersionRequest{
			WorkspaceID: normalizeWorkspace(*workspaceID),
		}, correlationID)
		if err != nil {
			return writeError(stderr, err)
		}
		if code := writeJSON(stdout, response); code != 0 {
			return code
		}
		if !response.Available {
			return 1
		}
		return 0
	case "exec":
		flags := flag.NewFlagSet("connector cloudflared exec", flag.ContinueOnError)
		flags.SetOutput(stderr)
		workspaceID := flags.String("workspace", "", "workspace id")
		timeout := flags.Duration("timeout", 30*time.Second, "command timeout")
		var opArgs repeatableStringFlag
		flags.Var(&opArgs, "arg", "cloudflared argument (repeatable)")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}
		execArgs := append([]string{}, opArgs...)
		execArgs = append(execArgs, trimStringSlice(flags.Args())...)
		if len(execArgs) == 0 {
			fmt.Fprintln(stderr, "request failed: at least one --arg is required")
			return 1
		}

		response, err := client.CloudflaredExec(ctx, transport.CloudflaredExecRequest{
			WorkspaceID: normalizeWorkspace(*workspaceID),
			Args:        execArgs,
			TimeoutMS:   durationToMillis(*timeout),
		}, correlationID)
		if err != nil {
			return writeError(stderr, err)
		}
		if code := writeJSON(stdout, response); code != 0 {
			return code
		}
		if !response.Success {
			return 1
		}
		return 0
	default:
		writeUnknownSubcommandError(stderr, "connector cloudflared subcommand", args[0])
		return 2
	}
}

func trimStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	trimmed := make([]string, 0, len(values))
	for _, value := range values {
		candidate := strings.TrimSpace(value)
		if candidate == "" {
			continue
		}
		trimmed = append(trimmed, candidate)
	}
	return trimmed
}
