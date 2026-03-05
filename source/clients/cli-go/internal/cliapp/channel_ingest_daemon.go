package cliapp

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"

	"personalagent/runtime/internal/transport"
)

func runConnectorTwilioIngestSMSDaemonCommand(
	ctx context.Context,
	client *transport.Client,
	args []string,
	correlationID string,
	stdout io.Writer,
	stderr io.Writer,
) int {
	flags := flag.NewFlagSet("connector twilio ingest-sms", flag.ContinueOnError)
	flags.SetOutput(stderr)

	workspaceID := flags.String("workspace", "", "workspace id")
	requestURL := flags.String("request-url", "", "webhook request URL used for signature validation")
	signature := flags.String("signature", "", "X-Twilio-Signature header value")
	skipSignature := flags.Bool("skip-signature", false, "bypass signature validation (local development only)")
	fromAddress := flags.String("from", "", "sender phone number (E.164)")
	toAddress := flags.String("to", "", "destination Twilio number (E.164)")
	bodyText := flags.String("body", "", "sms body text")
	messageSID := flags.String("message-sid", "", "Twilio message SID")
	accountSID := flags.String("account-sid", "", "Twilio account SID")
	_ = flags.String("occurred-at", "", "optional occurred-at timestamp RFC3339")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	normalizedSID := strings.TrimSpace(*messageSID)
	fromValue := strings.TrimSpace(*fromAddress)
	toValue := strings.TrimSpace(*toAddress)
	if normalizedSID == "" {
		fmt.Fprintln(stderr, "request failed: --message-sid is required")
		return 1
	}
	if fromValue == "" || toValue == "" {
		fmt.Fprintln(stderr, "request failed: --from and --to are required")
		return 1
	}

	if !*skipSignature {
		if strings.TrimSpace(*requestURL) == "" {
			fmt.Fprintln(stderr, "request failed: --request-url is required when signature validation is enabled")
			return 1
		}
		if strings.TrimSpace(*signature) == "" {
			fmt.Fprintln(stderr, "request failed: --signature is required when signature validation is enabled")
			return 1
		}
	}

	response, err := client.TwilioIngestSMS(ctx, transport.TwilioIngestSMSRequest{
		WorkspaceID:   normalizeWorkspace(*workspaceID),
		RequestURL:    strings.TrimSpace(*requestURL),
		Signature:     strings.TrimSpace(*signature),
		SkipSignature: *skipSignature,
		FromAddress:   fromValue,
		ToAddress:     toValue,
		BodyText:      strings.TrimSpace(*bodyText),
		MessageSID:    normalizedSID,
		AccountSID:    strings.TrimSpace(*accountSID),
	}, correlationID)
	if err != nil {
		return writeError(stderr, err)
	}
	if code := writeJSON(stdout, response); code != 0 {
		return code
	}
	if !response.Accepted {
		return 1
	}
	return 0
}

func runConnectorTwilioIngestVoiceDaemonCommand(
	ctx context.Context,
	client *transport.Client,
	args []string,
	correlationID string,
	stdout io.Writer,
	stderr io.Writer,
) int {
	flags := flag.NewFlagSet("connector twilio ingest-voice", flag.ContinueOnError)
	flags.SetOutput(stderr)

	workspaceID := flags.String("workspace", "", "workspace id")
	requestURL := flags.String("request-url", "", "webhook request URL used for signature validation")
	signature := flags.String("signature", "", "X-Twilio-Signature header value")
	skipSignature := flags.Bool("skip-signature", false, "bypass signature validation (local development only)")
	providerEventID := flags.String("provider-event-id", "", "unique provider callback/event id for replay safety")
	callSID := flags.String("call-sid", "", "Twilio call SID")
	accountSID := flags.String("account-sid", "", "Twilio account SID")
	fromAddress := flags.String("from", "", "source phone number (E.164)")
	toAddress := flags.String("to", "", "destination phone number (E.164)")
	direction := flags.String("direction", "inbound", "call direction: inbound|outbound")
	callStatus := flags.String("call-status", "initiated", "call status (Twilio status value)")
	transcript := flags.String("transcript", "", "optional transcript text")
	transcriptDirection := flags.String("transcript-direction", "INBOUND", "transcript direction: INBOUND|OUTBOUND")
	transcriptAssistantEmitted := flags.Bool("transcript-assistant-emitted", false, "marks transcript event as assistant emitted")
	_ = flags.String("occurred-at", "", "optional occurred-at timestamp RFC3339")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	callSIDValue := strings.TrimSpace(*callSID)
	fromValue := strings.TrimSpace(*fromAddress)
	toValue := strings.TrimSpace(*toAddress)
	if callSIDValue == "" {
		fmt.Fprintln(stderr, "request failed: --call-sid is required")
		return 1
	}
	if fromValue == "" || toValue == "" {
		fmt.Fprintln(stderr, "request failed: --from and --to are required")
		return 1
	}

	if !*skipSignature {
		if strings.TrimSpace(*requestURL) == "" {
			fmt.Fprintln(stderr, "request failed: --request-url is required when signature validation is enabled")
			return 1
		}
		if strings.TrimSpace(*signature) == "" {
			fmt.Fprintln(stderr, "request failed: --signature is required when signature validation is enabled")
			return 1
		}
	}

	response, err := client.TwilioIngestVoice(ctx, transport.TwilioIngestVoiceRequest{
		WorkspaceID:                normalizeWorkspace(*workspaceID),
		RequestURL:                 strings.TrimSpace(*requestURL),
		Signature:                  strings.TrimSpace(*signature),
		SkipSignature:              *skipSignature,
		ProviderEventID:            strings.TrimSpace(*providerEventID),
		CallSID:                    callSIDValue,
		AccountSID:                 strings.TrimSpace(*accountSID),
		FromAddress:                fromValue,
		ToAddress:                  toValue,
		Direction:                  strings.TrimSpace(*direction),
		CallStatus:                 strings.TrimSpace(*callStatus),
		Transcript:                 strings.TrimSpace(*transcript),
		TranscriptDirection:        strings.TrimSpace(*transcriptDirection),
		TranscriptAssistantEmitted: *transcriptAssistantEmitted,
	}, correlationID)
	if err != nil {
		return writeError(stderr, err)
	}
	if code := writeJSON(stdout, response); code != 0 {
		return code
	}
	if !response.Accepted {
		return 1
	}
	return 0
}
