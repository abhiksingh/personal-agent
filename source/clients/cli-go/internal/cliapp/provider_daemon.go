package cliapp

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"

	"personalagent/runtime/internal/transport"
)

func runProviderDaemonCommand(ctx context.Context, client *transport.Client, args []string, correlationID string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "provider subcommand required: set|list|check")
		return 2
	}

	switch args[0] {
	case "set":
		flags := flag.NewFlagSet("provider set", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		providerName := flags.String("provider", "", "provider id: openai|anthropic|google|ollama")
		endpoint := flags.String("endpoint", "", "provider base endpoint")
		apiKeySecret := flags.String("api-key-secret", "", "secret name stored via `personal-agent secret set`")
		clearAPIKey := flags.Bool("clear-api-key", false, "clear configured api key secret reference")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}

		response, err := client.SetProvider(ctx, transport.ProviderSetRequest{
			WorkspaceID:      normalizeWorkspace(*workspaceID),
			Provider:         strings.TrimSpace(*providerName),
			Endpoint:         strings.TrimSpace(*endpoint),
			APIKeySecretName: strings.TrimSpace(*apiKeySecret),
			ClearAPIKey:      *clearAPIKey,
		}, correlationID)
		if err != nil {
			return writeError(stderr, err)
		}
		return writeJSON(stdout, response)
	case "list":
		flags := flag.NewFlagSet("provider list", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}

		response, err := client.ListProviders(ctx, transport.ProviderListRequest{
			WorkspaceID: normalizeWorkspace(*workspaceID),
		}, correlationID)
		if err != nil {
			return writeError(stderr, err)
		}
		return writeProviderListResponse(stdout, response)
	case "check":
		flags := flag.NewFlagSet("provider check", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		providerName := flags.String("provider", "", "optional provider filter: openai|anthropic|google|ollama")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}

		response, err := client.CheckProviders(ctx, transport.ProviderCheckRequest{
			WorkspaceID: normalizeWorkspace(*workspaceID),
			Provider:    strings.TrimSpace(*providerName),
		}, correlationID)
		if err != nil {
			return writeError(stderr, err)
		}
		exitCode := writeJSON(stdout, response)
		if exitCode != 0 {
			return exitCode
		}
		if !response.Success {
			return 1
		}
		return 0
	default:
		writeUnknownSubcommandError(stderr, "provider subcommand", args[0])
		return 2
	}
}
