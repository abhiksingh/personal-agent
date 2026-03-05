package cliapp

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"

	"personalagent/runtime/internal/transport"
)

func runDelegationCommand(ctx context.Context, client *transport.Client, args []string, correlationID string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "delegation subcommand required: grant|list|revoke|check")
		return 2
	}

	switch args[0] {
	case "grant":
		flags := flag.NewFlagSet("delegation grant", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		fromActor := flags.String("from", "", "requester actor id")
		toActor := flags.String("to", "", "acting-as actor id")
		scopeType := flags.String("scope-type", "EXECUTION", "scope type: EXECUTION|APPROVAL|ALL")
		scopeKey := flags.String("scope-key", "", "optional scope key")
		expiresAt := flags.String("expires-at", "", "optional RFC3339 expiry timestamp")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}

		if strings.TrimSpace(*fromActor) == "" || strings.TrimSpace(*toActor) == "" {
			fmt.Fprintln(stderr, "request failed: --from and --to are required")
			return 1
		}

		response, err := client.DelegationGrant(ctx, transport.DelegationGrantRequest{
			WorkspaceID: normalizeWorkspace(*workspaceID),
			FromActorID: strings.TrimSpace(*fromActor),
			ToActorID:   strings.TrimSpace(*toActor),
			ScopeType:   strings.TrimSpace(*scopeType),
			ScopeKey:    strings.TrimSpace(*scopeKey),
			ExpiresAt:   strings.TrimSpace(*expiresAt),
		}, correlationID)
		if err != nil {
			return writeError(stderr, err)
		}
		return writeJSON(stdout, response)
	case "list":
		flags := flag.NewFlagSet("delegation list", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		fromActor := flags.String("from", "", "optional requester actor filter")
		toActor := flags.String("to", "", "optional acting-as actor filter")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}

		response, err := client.DelegationList(ctx, transport.DelegationListRequest{
			WorkspaceID: normalizeWorkspace(*workspaceID),
			FromActorID: strings.TrimSpace(*fromActor),
			ToActorID:   strings.TrimSpace(*toActor),
		}, correlationID)
		if err != nil {
			return writeError(stderr, err)
		}
		return writeJSON(stdout, response)
	case "revoke":
		flags := flag.NewFlagSet("delegation revoke", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		ruleID := flags.String("rule-id", "", "delegation rule id")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}
		if strings.TrimSpace(*ruleID) == "" {
			fmt.Fprintln(stderr, "request failed: --rule-id is required")
			return 1
		}

		response, err := client.DelegationRevoke(ctx, transport.DelegationRevokeRequest{
			WorkspaceID: normalizeWorkspace(*workspaceID),
			RuleID:      strings.TrimSpace(*ruleID),
		}, correlationID)
		if err != nil {
			return writeError(stderr, err)
		}
		return writeJSON(stdout, response)
	case "check":
		flags := flag.NewFlagSet("delegation check", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		requestedBy := flags.String("requested-by", "", "requester actor id")
		actingAs := flags.String("acting-as", "", "acting-as actor id")
		scopeType := flags.String("scope-type", "EXECUTION", "scope type: EXECUTION|APPROVAL|ALL")
		scopeKey := flags.String("scope-key", "", "scope key")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}
		if strings.TrimSpace(*requestedBy) == "" || strings.TrimSpace(*actingAs) == "" {
			fmt.Fprintln(stderr, "request failed: --requested-by and --acting-as are required")
			return 1
		}

		response, err := client.DelegationCheck(ctx, transport.DelegationCheckRequest{
			WorkspaceID:        normalizeWorkspace(*workspaceID),
			RequestedByActorID: strings.TrimSpace(*requestedBy),
			ActingAsActorID:    strings.TrimSpace(*actingAs),
			ScopeType:          strings.TrimSpace(*scopeType),
			ScopeKey:           strings.TrimSpace(*scopeKey),
		}, correlationID)
		if err != nil {
			return writeError(stderr, err)
		}
		return writeJSON(stdout, response)
	default:
		writeUnknownSubcommandError(stderr, "delegation subcommand", args[0])
		return 2
	}
}
