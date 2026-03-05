package cliapp

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"

	"personalagent/runtime/internal/transport"
)

func runAgentCommand(ctx context.Context, client *transport.Client, args []string, correlationID string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "agent subcommand required: run|approve")
		return 2
	}

	switch args[0] {
	case "run":
		flags := flag.NewFlagSet("agent run", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		requestText := flags.String("request", "", "user request text for intent interpretation")
		requestedBy := flags.String("requested-by", "actor.requester", "requesting actor id")
		subject := flags.String("subject", "", "subject principal actor id")
		actingAs := flags.String("acting-as", "", "acting-as actor id")
		origin := flags.String("origin", "cli", "execution origin: app|cli|voice")
		inAppApprovalConfirmed := flags.Bool("in-app-approval-confirmed", false, "voice-origin flow: set true when in-app approval handoff has already been confirmed")
		requestCorrelationID := flags.String("correlation-id", "", "optional correlation id")
		approvalPhrase := flags.String("approval-phrase", "", "approval phrase for destructive operations")
		preferredAdapterID := flags.String("preferred-adapter", "", "optional preferred connector adapter id")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}
		if strings.TrimSpace(*requestText) == "" {
			fmt.Fprintln(stderr, "request failed: --request is required")
			return 1
		}

		requestedByValue := strings.TrimSpace(*requestedBy)
		if requestedByValue == "" {
			requestedByValue = "actor.requester"
		}
		subjectValue := strings.TrimSpace(*subject)
		if subjectValue == "" {
			subjectValue = requestedByValue
		}
		actingAsValue := strings.TrimSpace(*actingAs)
		if actingAsValue == "" {
			actingAsValue = subjectValue
		}

		effectiveCorrelationID := strings.TrimSpace(*requestCorrelationID)
		if effectiveCorrelationID == "" {
			effectiveCorrelationID = strings.TrimSpace(correlationID)
		}

		response, err := client.AgentRun(ctx, transport.AgentRunRequest{
			WorkspaceID:            normalizeWorkspace(*workspaceID),
			RequestText:            strings.TrimSpace(*requestText),
			RequestedByActorID:     requestedByValue,
			SubjectActorID:         subjectValue,
			ActingAsActorID:        actingAsValue,
			Origin:                 strings.TrimSpace(*origin),
			InAppApprovalConfirmed: *inAppApprovalConfirmed,
			CorrelationID:          effectiveCorrelationID,
			ApprovalPhrase:         strings.TrimSpace(*approvalPhrase),
			PreferredAdapterID:     strings.TrimSpace(*preferredAdapterID),
		}, effectiveCorrelationID)
		if err != nil {
			return writeError(stderr, err)
		}
		return writeJSON(stdout, response)
	case "approve":
		flags := flag.NewFlagSet("agent approve", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		approvalID := flags.String("approval-id", "", "approval request id")
		phrase := flags.String("phrase", "", "approval phrase")
		actorID := flags.String("actor-id", "", "approver actor id")
		requestCorrelationID := flags.String("correlation-id", "", "optional correlation id")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}
		if strings.TrimSpace(*approvalID) == "" {
			fmt.Fprintln(stderr, "request failed: --approval-id is required")
			return 1
		}
		if strings.TrimSpace(*actorID) == "" {
			fmt.Fprintln(stderr, "request failed: --actor-id is required")
			return 1
		}

		effectiveCorrelationID := strings.TrimSpace(*requestCorrelationID)
		if effectiveCorrelationID == "" {
			effectiveCorrelationID = strings.TrimSpace(correlationID)
		}

		response, err := client.AgentApprove(ctx, transport.AgentApproveRequest{
			WorkspaceID:       normalizeWorkspace(*workspaceID),
			ApprovalRequestID: strings.TrimSpace(*approvalID),
			Phrase:            *phrase,
			DecisionByActorID: strings.TrimSpace(*actorID),
			CorrelationID:     effectiveCorrelationID,
		}, effectiveCorrelationID)
		if err != nil {
			return writeError(stderr, err)
		}
		return writeJSON(stdout, response)
	default:
		writeUnknownSubcommandError(stderr, "agent subcommand", args[0])
		return 2
	}
}
