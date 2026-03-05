package cliapp

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"personalagent/runtime/internal/transport"
)

const (
	cloudflaredModeAuto     = "auto"
	cloudflaredModeOff      = "off"
	cloudflaredModeRequired = "required"
)

const (
	defaultVoiceGreeting = "Personal Agent here. How can I help you today?"
	defaultVoiceFallback = "I did not catch that. Please repeat your request."
)

type twilioWebhookFixture struct {
	Kind       string            `json:"kind"`
	RequestURL string            `json:"request_url,omitempty"`
	Params     map[string]string `json:"params"`
}

func runConnectorTwilioWebhookDaemonCommand(
	ctx context.Context,
	client *transport.Client,
	args []string,
	correlationID string,
	stdout io.Writer,
	stderr io.Writer,
) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "connector twilio webhook subcommand required: serve|replay")
		return 2
	}

	switch args[0] {
	case "serve":
		return runConnectorTwilioWebhookServeDaemonCommand(ctx, client, args[1:], correlationID, stdout, stderr)
	case "replay":
		return runConnectorTwilioWebhookReplayDaemonCommand(ctx, client, args[1:], correlationID, stdout, stderr)
	default:
		writeUnknownSubcommandError(stderr, "connector twilio webhook subcommand", args[0])
		return 2
	}
}

func runConnectorTwilioWebhookServeDaemonCommand(
	ctx context.Context,
	client *transport.Client,
	args []string,
	correlationID string,
	stdout io.Writer,
	stderr io.Writer,
) int {
	flags := flag.NewFlagSet("connector twilio webhook serve", flag.ContinueOnError)
	flags.SetOutput(stderr)

	workspaceID := flags.String("workspace", "", "workspace id")
	listenAddress := flags.String("listen", "127.0.0.1:8088", "listen address for local webhook server")
	signatureMode := flags.String("signature-mode", "strict", "signature mode: strict|bypass")
	cloudflaredMode := flags.String("cloudflared-mode", cloudflaredModeAuto, "cloudflared mode: auto|off|required")
	cloudflaredStartupTimeout := flags.Duration("cloudflared-startup-timeout", 2*time.Second, "timeout waiting for cloudflared public URL discovery")
	assistantReplies := flags.Bool("assistant-replies", false, "enable assistant-generated replies for inbound webhook turns")
	assistantTaskClass := flags.String("assistant-task-class", "chat", "task class for conversational assistant responses")
	assistantSystemPrompt := flags.String("assistant-system", "", "optional system prompt for conversational assistant replies")
	assistantMaxHistory := flags.Int("assistant-max-history", 20, "max prior thread events to include when generating assistant replies")
	assistantReplyTimeout := flags.Duration("assistant-reply-timeout", 12*time.Second, "timeout for assistant reply generation + delivery")
	voiceResponseMode := flags.String("voice-response-mode", "json", "voice webhook response mode: json|twiml")
	voiceGreeting := flags.String("voice-greeting", defaultVoiceGreeting, "prompt used when voice turn has no transcript")
	voiceFallback := flags.String("voice-fallback", defaultVoiceFallback, "fallback prompt appended after voice gather")
	smsPath := flags.String("sms-path", "", "path for Twilio SMS webhook callbacks (default: daemon project-name path)")
	voicePath := flags.String("voice-path", "", "path for Twilio voice webhook callbacks (default: daemon project-name path)")
	runFor := flags.Duration("run-for", 0, "optional serve duration (0 means until command context ends)")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	response, err := client.TwilioWebhookServe(ctx, transport.TwilioWebhookServeRequest{
		WorkspaceID:                 normalizeWorkspace(*workspaceID),
		ListenAddress:               strings.TrimSpace(*listenAddress),
		SignatureMode:               strings.ToLower(strings.TrimSpace(*signatureMode)),
		CloudflaredMode:             strings.ToLower(strings.TrimSpace(*cloudflaredMode)),
		CloudflaredStartupTimeoutMS: durationToMillis(*cloudflaredStartupTimeout),
		AssistantReplies:            *assistantReplies,
		AssistantTaskClass:          normalizeTaskClass(*assistantTaskClass),
		AssistantSystemPrompt:       strings.TrimSpace(*assistantSystemPrompt),
		AssistantMaxHistory:         *assistantMaxHistory,
		AssistantReplyTimeoutMS:     durationToMillis(*assistantReplyTimeout),
		VoiceResponseMode:           strings.ToLower(strings.TrimSpace(*voiceResponseMode)),
		VoiceGreeting:               strings.TrimSpace(*voiceGreeting),
		VoiceFallback:               strings.TrimSpace(*voiceFallback),
		SMSPath:                     normalizeOptionalWebhookPath(*smsPath),
		VoicePath:                   normalizeOptionalWebhookPath(*voicePath),
		RunForMS:                    durationToMillis(*runFor),
	}, correlationID)
	if err != nil {
		return writeError(stderr, err)
	}
	return writeJSON(stdout, response)
}

func runConnectorTwilioWebhookReplayDaemonCommand(
	ctx context.Context,
	client *transport.Client,
	args []string,
	correlationID string,
	stdout io.Writer,
	stderr io.Writer,
) int {
	flags := flag.NewFlagSet("connector twilio webhook replay", flag.ContinueOnError)
	flags.SetOutput(stderr)

	workspaceID := flags.String("workspace", "", "workspace id")
	fixturePath := flags.String("fixture", "", "path to JSON fixture file")
	baseURL := flags.String("base-url", "http://127.0.0.1:8088", "base URL of local webhook harness")
	signatureMode := flags.String("signature-mode", "strict", "signature mode: strict|bypass")
	requestURL := flags.String("request-url", "", "optional request URL override for Twilio signature computation")
	smsPath := flags.String("sms-path", "", "path for Twilio SMS webhook callbacks (default: daemon project-name path)")
	voicePath := flags.String("voice-path", "", "path for Twilio voice webhook callbacks (default: daemon project-name path)")
	httpTimeout := flags.Duration("http-timeout", 10*time.Second, "timeout for replay HTTP request")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	if strings.TrimSpace(*fixturePath) == "" {
		fmt.Fprintln(stderr, "request failed: --fixture is required")
		return 1
	}

	fixtureBytes, err := os.ReadFile(strings.TrimSpace(*fixturePath))
	if err != nil {
		fmt.Fprintf(stderr, "request failed: read fixture file: %v\n", err)
		return 1
	}

	fixture := twilioWebhookFixture{}
	if err := json.Unmarshal(fixtureBytes, &fixture); err != nil {
		fmt.Fprintf(stderr, "request failed: decode fixture JSON: %v\n", err)
		return 1
	}
	if len(fixture.Params) == 0 {
		fmt.Fprintln(stderr, "request failed: fixture params are required")
		return 1
	}

	kind := strings.ToLower(strings.TrimSpace(fixture.Kind))
	if kind != "sms" && kind != "voice" {
		fmt.Fprintf(stderr, "request failed: fixture kind must be sms or voice (got %q)\n", fixture.Kind)
		return 1
	}

	resolvedBaseURL := strings.TrimRight(strings.TrimSpace(*baseURL), "/")
	targetPath := normalizeWebhookPath(*smsPath)
	if kind == "voice" {
		targetPath = normalizeWebhookPath(*voicePath)
	}
	targetURL := resolvedBaseURL + targetPath
	resolvedRequestURL := firstNonEmpty(strings.TrimSpace(*requestURL), strings.TrimSpace(fixture.RequestURL), targetURL)

	response, err := client.TwilioWebhookReplay(ctx, transport.TwilioWebhookReplayRequest{
		WorkspaceID:   normalizeWorkspace(*workspaceID),
		Kind:          kind,
		BaseURL:       strings.TrimSpace(*baseURL),
		RequestURL:    resolvedRequestURL,
		SignatureMode: strings.ToLower(strings.TrimSpace(*signatureMode)),
		SMSPath:       normalizeOptionalWebhookPath(*smsPath),
		VoicePath:     normalizeOptionalWebhookPath(*voicePath),
		HTTPTimeoutMS: durationToMillis(*httpTimeout),
		Params:        fixture.Params,
	}, correlationID)
	if err != nil {
		return writeError(stderr, err)
	}
	if code := writeJSON(stdout, response); code != 0 {
		return code
	}
	if response.StatusCode >= 400 {
		return 1
	}
	return 0
}

func durationToMillis(value time.Duration) int64 {
	if value == 0 {
		return 0
	}
	return int64(value / time.Millisecond)
}

func normalizeOptionalWebhookPath(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	return normalizeWebhookPath(value)
}
