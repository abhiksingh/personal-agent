package cliapp

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"

	"personalagent/runtime/internal/transport"
)

func runIdentityCommand(ctx context.Context, client *transport.Client, args []string, correlationID string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "identity subcommand required: workspaces|principals|context|select-workspace|bootstrap|devices|sessions|revoke-session")
		return 2
	}

	switch args[0] {
	case "workspaces":
		flags := flag.NewFlagSet("identity workspaces", flag.ContinueOnError)
		flags.SetOutput(stderr)

		includeInactive := flags.Bool("include-inactive", false, "include inactive workspaces")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}

		response, err := client.IdentityWorkspaces(ctx, transport.IdentityWorkspacesRequest{
			IncludeInactive: *includeInactive,
		}, correlationID)
		if err != nil {
			return writeError(stderr, err)
		}
		return writeJSON(stdout, response)
	case "principals":
		flags := flag.NewFlagSet("identity principals", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		includeInactive := flags.Bool("include-inactive", false, "include inactive principals")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}

		resolvedWorkspaceID, err := resolveWorkspaceForIdentityPrincipals(ctx, client, strings.TrimSpace(*workspaceID), correlationID)
		if err != nil {
			return writeError(stderr, err)
		}

		response, err := client.IdentityPrincipals(ctx, transport.IdentityPrincipalsRequest{
			WorkspaceID:     resolvedWorkspaceID,
			IncludeInactive: *includeInactive,
		}, correlationID)
		if err != nil {
			return writeError(stderr, err)
		}
		return writeJSON(stdout, response)
	case "context":
		flags := flag.NewFlagSet("identity context", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "optional preferred workspace id")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}

		request := transport.IdentityActiveContextRequest{}
		if trimmedWorkspaceID := strings.TrimSpace(*workspaceID); trimmedWorkspaceID != "" {
			request.WorkspaceID = normalizeWorkspace(trimmedWorkspaceID)
		}
		response, err := client.IdentityActiveContext(ctx, request, correlationID)
		if err != nil {
			return writeError(stderr, err)
		}
		return writeJSON(stdout, response)
	case "select-workspace":
		flags := flag.NewFlagSet("identity select-workspace", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		principalActorID := flags.String("principal", "", "optional principal actor id")
		source := flags.String("source", "cli", "selection source")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}

		if strings.TrimSpace(*workspaceID) == "" {
			fmt.Fprintln(stderr, "request failed: --workspace is required")
			return 1
		}

		response, err := client.IdentitySelectWorkspace(ctx, transport.IdentityWorkspaceSelectRequest{
			WorkspaceID:      normalizeWorkspace(*workspaceID),
			PrincipalActorID: strings.TrimSpace(*principalActorID),
			Source:           strings.TrimSpace(*source),
		}, correlationID)
		if err != nil {
			return writeError(stderr, err)
		}
		return writeJSON(stdout, response)
	case "bootstrap":
		flags := flag.NewFlagSet("identity bootstrap", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		workspaceName := flags.String("workspace-name", "", "optional workspace display name")
		workspaceStatus := flags.String("workspace-status", "ACTIVE", "workspace status: ACTIVE|INACTIVE")
		principalActorID := flags.String("principal", "", "principal actor id")
		principalDisplayName := flags.String("display-name", "", "optional principal display name")
		principalActorType := flags.String("actor-type", "human", "principal actor type")
		principalStatus := flags.String("principal-status", "ACTIVE", "principal status: ACTIVE|INACTIVE")
		handleChannel := flags.String("handle-channel", "", "optional handle channel")
		handleValue := flags.String("handle-value", "", "optional handle value")
		handlePrimary := flags.Bool("handle-primary", false, "whether the handle is primary for actor/channel")
		source := flags.String("source", "cli", "bootstrap source")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}

		if strings.TrimSpace(*workspaceID) == "" {
			fmt.Fprintln(stderr, "request failed: --workspace is required")
			return 1
		}
		if strings.TrimSpace(*principalActorID) == "" {
			fmt.Fprintln(stderr, "request failed: --principal is required")
			return 1
		}
		if (strings.TrimSpace(*handleChannel) == "") != (strings.TrimSpace(*handleValue) == "") {
			fmt.Fprintln(stderr, "request failed: --handle-channel and --handle-value must be provided together")
			return 1
		}

		request := transport.IdentityBootstrapRequest{
			WorkspaceID:          normalizeWorkspace(*workspaceID),
			WorkspaceName:        strings.TrimSpace(*workspaceName),
			WorkspaceStatus:      strings.TrimSpace(*workspaceStatus),
			PrincipalActorID:     strings.TrimSpace(*principalActorID),
			PrincipalDisplayName: strings.TrimSpace(*principalDisplayName),
			PrincipalActorType:   strings.TrimSpace(*principalActorType),
			PrincipalStatus:      strings.TrimSpace(*principalStatus),
			Source:               strings.TrimSpace(*source),
		}
		if strings.TrimSpace(*handleChannel) != "" {
			request.Handle = &transport.IdentityBootstrapHandle{
				Channel:     strings.TrimSpace(*handleChannel),
				HandleValue: strings.TrimSpace(*handleValue),
				IsPrimary:   *handlePrimary,
			}
		}

		response, err := client.IdentityBootstrap(ctx, request, correlationID)
		if err != nil {
			return writeError(stderr, err)
		}
		return writeJSON(stdout, response)
	case "devices":
		flags := flag.NewFlagSet("identity devices", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		userID := flags.String("user-id", "", "optional user id filter")
		deviceType := flags.String("device-type", "", "optional device type filter")
		platform := flags.String("platform", "", "optional platform filter")
		cursorCreatedAt := flags.String("cursor-created-at", "", "optional pagination cursor timestamp")
		cursorID := flags.String("cursor-id", "", "optional pagination cursor id")
		limit := flags.Int("limit", 0, "optional page size")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}

		resolvedWorkspaceID, err := resolveWorkspaceForIdentityPrincipals(ctx, client, strings.TrimSpace(*workspaceID), correlationID)
		if err != nil {
			return writeError(stderr, err)
		}

		response, err := client.IdentityDevices(ctx, transport.IdentityDeviceListRequest{
			WorkspaceID:     resolvedWorkspaceID,
			UserID:          strings.TrimSpace(*userID),
			DeviceType:      strings.TrimSpace(*deviceType),
			Platform:        strings.TrimSpace(*platform),
			CursorCreatedAt: strings.TrimSpace(*cursorCreatedAt),
			CursorID:        strings.TrimSpace(*cursorID),
			Limit:           *limit,
		}, correlationID)
		if err != nil {
			return writeError(stderr, err)
		}
		return writeJSON(stdout, response)
	case "sessions":
		flags := flag.NewFlagSet("identity sessions", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		deviceID := flags.String("device-id", "", "optional device id filter")
		userID := flags.String("user-id", "", "optional user id filter")
		sessionHealth := flags.String("session-health", "", "optional session health filter: active|expired|revoked")
		cursorStartedAt := flags.String("cursor-started-at", "", "optional pagination cursor timestamp")
		cursorID := flags.String("cursor-id", "", "optional pagination cursor id")
		limit := flags.Int("limit", 0, "optional page size")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}

		resolvedWorkspaceID, err := resolveWorkspaceForIdentityPrincipals(ctx, client, strings.TrimSpace(*workspaceID), correlationID)
		if err != nil {
			return writeError(stderr, err)
		}

		response, err := client.IdentitySessions(ctx, transport.IdentitySessionListRequest{
			WorkspaceID:     resolvedWorkspaceID,
			DeviceID:        strings.TrimSpace(*deviceID),
			UserID:          strings.TrimSpace(*userID),
			SessionHealth:   strings.TrimSpace(*sessionHealth),
			CursorStartedAt: strings.TrimSpace(*cursorStartedAt),
			CursorID:        strings.TrimSpace(*cursorID),
			Limit:           *limit,
		}, correlationID)
		if err != nil {
			return writeError(stderr, err)
		}
		return writeJSON(stdout, response)
	case "revoke-session":
		flags := flag.NewFlagSet("identity revoke-session", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		sessionID := flags.String("session-id", "", "session id")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}
		if strings.TrimSpace(*sessionID) == "" {
			fmt.Fprintln(stderr, "request failed: --session-id is required")
			return 1
		}

		resolvedWorkspaceID, err := resolveWorkspaceForIdentityPrincipals(ctx, client, strings.TrimSpace(*workspaceID), correlationID)
		if err != nil {
			return writeError(stderr, err)
		}
		response, err := client.IdentitySessionRevoke(ctx, transport.IdentitySessionRevokeRequest{
			WorkspaceID: resolvedWorkspaceID,
			SessionID:   strings.TrimSpace(*sessionID),
		}, correlationID)
		if err != nil {
			return writeError(stderr, err)
		}
		return writeJSON(stdout, response)
	default:
		writeUnknownSubcommandError(stderr, "identity subcommand", args[0])
		return 2
	}
}

func resolveWorkspaceForIdentityPrincipals(ctx context.Context, client *transport.Client, requestedWorkspaceID string, correlationID string) (string, error) {
	if requestedWorkspaceID != "" {
		return normalizeWorkspace(requestedWorkspaceID), nil
	}

	contextResponse, err := client.IdentityActiveContext(ctx, transport.IdentityActiveContextRequest{}, correlationID)
	if err != nil {
		return "", err
	}
	if resolved := strings.TrimSpace(contextResponse.ActiveContext.WorkspaceID); resolved != "" {
		return resolved, nil
	}
	return normalizeWorkspace(""), nil
}
