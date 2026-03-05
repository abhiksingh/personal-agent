package cliapp

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"

	"personalagent/runtime/internal/transport"
)

func runCommDaemonCommand(ctx context.Context, client *transport.Client, args []string, correlationID string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "comm subcommand required: send|attempts|policy")
		return 2
	}

	switch args[0] {
	case "send":
		flags := flag.NewFlagSet("comm send", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		operationID := flags.String("operation-id", "", "idempotency operation id")
		sourceChannel := flags.String("source-channel", "message", "originating inbound channel")
		connectorID := flags.String("connector-id", "", "optional connector hint (builtin.app|imessage|twilio)")
		destination := flags.String("destination", "", "destination endpoint (phone/email/etc)")
		message := flags.String("message", "", "message body")
		stepID := flags.String("step-id", "", "optional task step id")
		eventID := flags.String("event-id", "", "optional comm event id")
		imessageFailures := flags.Int("imessage-failures", 0, "simulated iMessage failures before success")
		smsFailures := flags.Int("sms-failures", 0, "simulated SMS failures before success")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}

		response, err := client.CommSend(ctx, transport.CommSendRequest{
			WorkspaceID:      normalizeWorkspace(*workspaceID),
			OperationID:      strings.TrimSpace(*operationID),
			SourceChannel:    strings.TrimSpace(*sourceChannel),
			ConnectorID:      strings.TrimSpace(*connectorID),
			Destination:      strings.TrimSpace(*destination),
			Message:          strings.TrimSpace(*message),
			StepID:           strings.TrimSpace(*stepID),
			EventID:          strings.TrimSpace(*eventID),
			IMessagesFailure: *imessageFailures,
			SMSFailures:      *smsFailures,
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
	case "attempts":
		flags := flag.NewFlagSet("comm attempts", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		operationID := flags.String("operation-id", "", "operation id")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}

		response, err := client.CommAttempts(ctx, transport.CommAttemptsRequest{
			WorkspaceID: normalizeWorkspace(*workspaceID),
			OperationID: strings.TrimSpace(*operationID),
		}, correlationID)
		if err != nil {
			return writeError(stderr, err)
		}
		return writeJSON(stdout, response)
	case "policy":
		return runCommPolicyDaemonCommand(ctx, client, args[1:], correlationID, stdout, stderr)
	default:
		writeUnknownSubcommandError(stderr, "comm subcommand", args[0])
		return 2
	}
}

func runCommPolicyDaemonCommand(ctx context.Context, client *transport.Client, args []string, correlationID string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "comm policy subcommand required: set|list")
		return 2
	}

	switch args[0] {
	case "set":
		flags := flag.NewFlagSet("comm policy set", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		sourceChannel := flags.String("source-channel", "message", "source channel")
		endpointPattern := flags.String("endpoint-pattern", "", "optional SQL LIKE endpoint pattern")
		primaryChannel := flags.String("primary-channel", "imessage", "primary delivery channel")
		retryCount := flags.Int("retry-count", 1, "primary channel retry count")
		fallbackChannels := flags.String("fallback-channels", "sms", "comma-separated fallback channels")
		isDefault := flags.Bool("is-default", true, "mark as default policy for source channel")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}

		response, err := client.CommPolicySet(ctx, transport.CommPolicySetRequest{
			WorkspaceID:      normalizeWorkspace(*workspaceID),
			SourceChannel:    strings.TrimSpace(*sourceChannel),
			EndpointPattern:  strings.TrimSpace(*endpointPattern),
			PrimaryChannel:   strings.TrimSpace(*primaryChannel),
			RetryCount:       *retryCount,
			FallbackChannels: parseFallbackChannels(*fallbackChannels),
			IsDefault:        *isDefault,
		}, correlationID)
		if err != nil {
			return writeError(stderr, err)
		}
		return writeJSON(stdout, response)
	case "list":
		flags := flag.NewFlagSet("comm policy list", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		sourceChannel := flags.String("source-channel", "", "optional source channel filter")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}

		response, err := client.CommPolicyList(ctx, transport.CommPolicyListRequest{
			WorkspaceID:   normalizeWorkspace(*workspaceID),
			SourceChannel: strings.TrimSpace(*sourceChannel),
		}, correlationID)
		if err != nil {
			return writeError(stderr, err)
		}
		return writeJSON(stdout, response)
	default:
		writeUnknownSubcommandError(stderr, "comm policy subcommand", args[0])
		return 2
	}
}
