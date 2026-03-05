package cliapp

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"

	"personalagent/runtime/internal/modelpolicy"
	"personalagent/runtime/internal/transport"
)

func runModelDaemonCommand(ctx context.Context, client *transport.Client, args []string, correlationID string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "model subcommand required: list|discover|add|remove|enable|disable|select|policy|resolve")
		return 2
	}

	switch args[0] {
	case "list":
		flags := flag.NewFlagSet("model list", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		providerName := flags.String("provider", "", "optional provider filter: openai|anthropic|google|ollama")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}

		response, err := client.ListModels(ctx, transport.ModelListRequest{
			WorkspaceID: normalizeWorkspace(*workspaceID),
			Provider:    strings.TrimSpace(*providerName),
		}, correlationID)
		if err != nil {
			return writeError(stderr, err)
		}
		return writeModelListResponse(stdout, response)
	case "discover":
		flags := flag.NewFlagSet("model discover", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		providerName := flags.String("provider", "", "optional provider filter: openai|anthropic|google|ollama")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}

		response, err := client.DiscoverModels(ctx, transport.ModelDiscoverRequest{
			WorkspaceID: normalizeWorkspace(*workspaceID),
			Provider:    strings.TrimSpace(*providerName),
		}, correlationID)
		if err != nil {
			return writeError(stderr, err)
		}
		return writeJSON(stdout, response)
	case "add":
		flags := flag.NewFlagSet("model add", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		providerName := flags.String("provider", "", "provider id: openai|anthropic|google|ollama")
		modelKey := flags.String("model", "", "model key")
		enabled := flags.Bool("enabled", false, "set model enabled on add")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}

		response, err := client.AddModel(ctx, transport.ModelCatalogAddRequest{
			WorkspaceID: normalizeWorkspace(*workspaceID),
			Provider:    strings.TrimSpace(*providerName),
			ModelKey:    strings.TrimSpace(*modelKey),
			Enabled:     *enabled,
		}, correlationID)
		if err != nil {
			return writeError(stderr, err)
		}
		return writeJSON(stdout, response)
	case "remove":
		flags := flag.NewFlagSet("model remove", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		providerName := flags.String("provider", "", "provider id: openai|anthropic|google|ollama")
		modelKey := flags.String("model", "", "model key")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}

		response, err := client.RemoveModel(ctx, transport.ModelCatalogRemoveRequest{
			WorkspaceID: normalizeWorkspace(*workspaceID),
			Provider:    strings.TrimSpace(*providerName),
			ModelKey:    strings.TrimSpace(*modelKey),
		}, correlationID)
		if err != nil {
			return writeError(stderr, err)
		}
		return writeJSON(stdout, response)
	case "enable", "disable":
		flags := flag.NewFlagSet("model "+args[0], flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		providerName := flags.String("provider", "", "provider id: openai|anthropic|google|ollama")
		modelKey := flags.String("model", "", "model key")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}

		request := transport.ModelToggleRequest{
			WorkspaceID: normalizeWorkspace(*workspaceID),
			Provider:    strings.TrimSpace(*providerName),
			ModelKey:    strings.TrimSpace(*modelKey),
		}
		var (
			response transport.ModelCatalogEntryRecord
			err      error
		)
		if args[0] == "enable" {
			response, err = client.EnableModel(ctx, request, correlationID)
		} else {
			response, err = client.DisableModel(ctx, request, correlationID)
		}
		if err != nil {
			return writeError(stderr, err)
		}
		return writeJSON(stdout, response)
	case "select":
		flags := flag.NewFlagSet("model select", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		taskClass := flags.String("task-class", modelpolicy.TaskClassDefault, "task class")
		providerName := flags.String("provider", "", "provider id: openai|anthropic|google|ollama")
		modelKey := flags.String("model", "", "model key")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}

		response, err := client.SelectModelRoute(ctx, transport.ModelSelectRequest{
			WorkspaceID: normalizeWorkspace(*workspaceID),
			TaskClass:   normalizeTaskClass(*taskClass),
			Provider:    strings.TrimSpace(*providerName),
			ModelKey:    strings.TrimSpace(*modelKey),
		}, correlationID)
		if err != nil {
			return writeError(stderr, err)
		}
		return writeJSON(stdout, response)
	case "policy":
		flags := flag.NewFlagSet("model policy", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		taskClass := flags.String("task-class", "", "optional task class filter")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}

		response, err := client.ModelPolicy(ctx, transport.ModelPolicyRequest{
			WorkspaceID: normalizeWorkspace(*workspaceID),
			TaskClass:   strings.TrimSpace(*taskClass),
		}, correlationID)
		if err != nil {
			return writeError(stderr, err)
		}
		if strings.TrimSpace(*taskClass) != "" {
			if response.Policy == nil {
				fmt.Fprintln(stderr, "request failed: model policy was not returned")
				return 1
			}
			return writeJSON(stdout, response.Policy)
		}
		return writeJSON(stdout, map[string]any{
			"workspace_id": response.WorkspaceID,
			"policies":     response.Policies,
		})
	case "resolve":
		flags := flag.NewFlagSet("model resolve", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		taskClass := flags.String("task-class", modelpolicy.TaskClassDefault, "task class")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}

		response, err := client.ResolveModelRoute(ctx, transport.ModelResolveRequest{
			WorkspaceID: normalizeWorkspace(*workspaceID),
			TaskClass:   normalizeTaskClass(*taskClass),
		}, correlationID)
		if err != nil {
			return writeError(stderr, err)
		}
		return writeJSON(stdout, response)
	default:
		writeUnknownSubcommandError(stderr, "model subcommand", args[0])
		return 2
	}
}
