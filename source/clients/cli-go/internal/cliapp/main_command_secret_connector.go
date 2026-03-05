package cliapp

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	connectorsmoke "personalagent/runtime/internal/connectors/smoke"
	"personalagent/runtime/internal/transport"
)

type secretCommandResponse struct {
	WorkspaceID string `json:"workspace_id"`
	Name        string `json:"name"`
	Backend     string `json:"backend"`
	Service     string `json:"service"`
	Account     string `json:"account"`
	Registered  bool   `json:"registered,omitempty"`
	Deleted     bool   `json:"deleted,omitempty"`
}

func runSecretCommand(ctx context.Context, client *transport.Client, args []string, correlationID string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "secret subcommand required: set|get|delete")
		return 2
	}

	switch args[0] {
	case "set":
		manager, err := newSecretManager()
		if err != nil {
			fmt.Fprintf(stderr, "secret manager setup failed: %v\n", err)
			return 1
		}

		flags := flag.NewFlagSet("secret set", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		name := flags.String("name", "", "secret name")
		value := flags.String("value", "", "secret value")
		filePath := flags.String("file", "", "path to file containing secret value")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}
		workspace := normalizeWorkspace(*workspaceID)

		resolvedValue, resolveErr := resolveSecretValue(*value, *filePath)
		if resolveErr != nil {
			fmt.Fprintf(stderr, "request failed: %v\n", resolveErr)
			return 1
		}

		ref, putErr := manager.Put(workspace, *name, resolvedValue)
		if putErr != nil {
			fmt.Fprintf(stderr, "request failed: %v\n", putErr)
			return 1
		}

		registerResponse, registerErr := client.UpsertSecretReference(ctx, transport.SecretReferenceUpsertRequest{
			WorkspaceID: ref.WorkspaceID,
			Name:        ref.Name,
			Backend:     ref.Backend,
			Service:     ref.Service,
			Account:     ref.Account,
		}, correlationID)
		if registerErr != nil {
			return writeError(stderr, registerErr)
		}

		record := registerResponse.Reference
		if strings.TrimSpace(record.Backend) == "" {
			record.Backend = ref.Backend
		}
		return writeJSON(stdout, secretCommandResponse{
			WorkspaceID: record.WorkspaceID,
			Name:        record.Name,
			Backend:     record.Backend,
			Service:     record.Service,
			Account:     record.Account,
			Registered:  true,
		})
	case "get":
		flags := flag.NewFlagSet("secret get", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		name := flags.String("name", "", "secret name")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}
		workspace := normalizeWorkspace(*workspaceID)

		response, getErr := client.GetSecretReference(ctx, workspace, *name, correlationID)
		if getErr != nil {
			return writeError(stderr, getErr)
		}
		record := response.Reference
		return writeJSON(stdout, secretCommandResponse{
			WorkspaceID: record.WorkspaceID,
			Name:        record.Name,
			Backend:     record.Backend,
			Service:     record.Service,
			Account:     record.Account,
		})
	case "delete":
		manager, err := newSecretManager()
		if err != nil {
			fmt.Fprintf(stderr, "secret manager setup failed: %v\n", err)
			return 1
		}

		flags := flag.NewFlagSet("secret delete", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		name := flags.String("name", "", "secret name")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}
		workspace := normalizeWorkspace(*workspaceID)

		ref, deleteErr := manager.Delete(workspace, *name)
		if deleteErr != nil {
			fmt.Fprintf(stderr, "request failed: %v\n", deleteErr)
			return 1
		}

		deleteResponse, daemonDeleteErr := client.DeleteSecretReference(ctx, ref.WorkspaceID, ref.Name, correlationID)
		if daemonDeleteErr != nil {
			var httpErr transport.HTTPError
			if !errors.As(daemonDeleteErr, &httpErr) || httpErr.StatusCode != 404 {
				return writeError(stderr, daemonDeleteErr)
			}
		}

		record := deleteResponse.Reference
		if strings.TrimSpace(record.WorkspaceID) == "" {
			record.WorkspaceID = ref.WorkspaceID
			record.Name = ref.Name
			record.Backend = ref.Backend
			record.Service = ref.Service
			record.Account = ref.Account
		}
		return writeJSON(stdout, secretCommandResponse{
			WorkspaceID: record.WorkspaceID,
			Name:        record.Name,
			Backend:     record.Backend,
			Service:     record.Service,
			Account:     record.Account,
			Deleted:     true,
		})
	default:
		writeUnknownCommandError(stderr, "secret subcommand", args[0], []string{"set", "get", "delete"})
		return 2
	}
}

func runConnectorCommand(ctx context.Context, client *transport.Client, args []string, correlationID string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "connector subcommand required: smoke|bridge|twilio|mail|calendar|browser|cloudflared")
		return 2
	}

	switch args[0] {
	case "smoke":
		runner := connectorsmoke.NewRunner()
		report := runner.Run(ctx)
		exitCode := writeJSON(stdout, report)
		if exitCode != 0 {
			return exitCode
		}
		if !report.Success {
			return 1
		}
		return 0
	case "bridge":
		return runConnectorBridgeLocalCommand(args[1:], stdout, stderr)
	case "mail":
		return runConnectorMailDaemonCommand(ctx, client, args[1:], correlationID, stdout, stderr)
	case "calendar":
		return runConnectorCalendarDaemonCommand(ctx, client, args[1:], correlationID, stdout, stderr)
	case "browser":
		return runConnectorBrowserDaemonCommand(ctx, client, args[1:], correlationID, stdout, stderr)
	case "cloudflared":
		if client == nil {
			fmt.Fprintln(stderr, "request failed: daemon client is not configured")
			return 1
		}
		return runConnectorCloudflaredDaemonCommand(ctx, client, args[1:], correlationID, stdout, stderr)
	case "twilio":
		if client == nil {
			fmt.Fprintln(stderr, "request failed: daemon client is not configured")
			return 1
		}
		return runConnectorTwilioCommand(ctx, client, args[1:], "", correlationID, stdout, stderr)
	default:
		writeUnknownCommandError(stderr, "connector subcommand", args[0], []string{"smoke", "bridge", "twilio", "mail", "calendar", "browser", "cloudflared"})
		return 2
	}
}

func resolveSecretValue(value string, filePath string) (string, error) {
	trimmedValue := strings.TrimSpace(value)
	trimmedPath := strings.TrimSpace(filePath)
	if trimmedValue == "" && trimmedPath == "" {
		return "", fmt.Errorf("either --value or --file is required")
	}
	if trimmedValue != "" && trimmedPath != "" {
		return "", fmt.Errorf("provide only one of --value or --file")
	}
	if trimmedPath != "" {
		bytes, err := os.ReadFile(trimmedPath)
		if err != nil {
			return "", fmt.Errorf("read secret file: %w", err)
		}
		trimmed := strings.TrimSpace(string(bytes))
		if trimmed == "" {
			return "", fmt.Errorf("secret file is empty")
		}
		return trimmed, nil
	}
	return trimmedValue, nil
}
