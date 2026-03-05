package cliapp

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"personalagent/runtime/internal/channelconfig"
	"personalagent/runtime/internal/transport"
)

type twilioIngestSMSResponse struct {
	WorkspaceID              string `json:"workspace_id"`
	Accepted                 bool   `json:"accepted"`
	Replayed                 bool   `json:"replayed"`
	EventID                  string `json:"event_id,omitempty"`
	ThreadID                 string `json:"thread_id,omitempty"`
	MessageSID               string `json:"message_sid,omitempty"`
	AssistantReply           string `json:"assistant_reply,omitempty"`
	AssistantOperationID     string `json:"assistant_operation_id,omitempty"`
	AssistantDelivered       bool   `json:"assistant_delivered,omitempty"`
	AssistantProviderReceipt string `json:"assistant_provider_receipt,omitempty"`
	AssistantError           string `json:"assistant_error,omitempty"`
	Error                    string `json:"error,omitempty"`
}

type twilioIngestVoiceResponse struct {
	WorkspaceID              string `json:"workspace_id"`
	Accepted                 bool   `json:"accepted"`
	Replayed                 bool   `json:"replayed"`
	ProviderEventID          string `json:"provider_event_id"`
	CallSID                  string `json:"call_sid"`
	CallSessionID            string `json:"call_session_id,omitempty"`
	ThreadID                 string `json:"thread_id,omitempty"`
	CallStatus               string `json:"call_status,omitempty"`
	StatusEventID            string `json:"status_event_id,omitempty"`
	TranscriptEventID        string `json:"transcript_event_id,omitempty"`
	AssistantReply           string `json:"assistant_reply,omitempty"`
	AssistantReplySource     string `json:"assistant_reply_source,omitempty"`
	AssistantReplyTranscript string `json:"assistant_reply_transcript,omitempty"`
	AssistantError           string `json:"assistant_error,omitempty"`
	Error                    string `json:"error,omitempty"`
}

func runChannelCommand(ctx context.Context, client *transport.Client, args []string, dbPath string, correlationID string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "channel subcommand required: messages|mapping")
		return 2
	}

	switch args[0] {
	case "messages":
		return runChannelMessagesDaemonCommand(ctx, client, args[1:], correlationID, stdout, stderr)
	case "mapping":
		return runChannelMappingDaemonCommand(ctx, client, args[1:], correlationID, stdout, stderr)
	default:
		writeUnknownSubcommandError(stderr, "channel subcommand", args[0])
		return 2
	}
}

func runConnectorTwilioCommand(ctx context.Context, client *transport.Client, args []string, dbPath string, correlationID string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "connector twilio subcommand required: set|get|check|ingest-sms|start-call|ingest-voice|webhook|sms-chat|call-status|transcript")
		return 2
	}
	_ = dbPath

	switch args[0] {
	case "set":
		flags := flag.NewFlagSet("connector twilio set", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		accountSIDSecret := flags.String("account-sid-secret", "TWILIO_ACCOUNT_SID", "secret name for twilio account sid")
		authTokenSecret := flags.String("auth-token-secret", "TWILIO_AUTH_TOKEN", "secret name for twilio auth token")
		accountSIDValue := flags.String("account-sid", "", "twilio account sid value (stored in secure storage)")
		accountSIDFile := flags.String("account-sid-file", "", "path to file containing twilio account sid")
		authTokenValue := flags.String("auth-token", "", "twilio auth token value (stored in secure storage)")
		authTokenFile := flags.String("auth-token-file", "", "path to file containing twilio auth token")
		smsNumber := flags.String("sms-number", "", "twilio sms number (E.164)")
		voiceNumber := flags.String("voice-number", "", "twilio voice number (E.164)")
		endpoint := flags.String("endpoint", "", "twilio api endpoint override")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}

		workspace := normalizeWorkspace(*workspaceID)
		manager, err := newSecretManager()
		if err != nil {
			fmt.Fprintf(stderr, "request failed: secret manager setup failed: %v\n", err)
			return 1
		}

		accountSIDSecretName := strings.TrimSpace(*accountSIDSecret)
		authTokenSecretName := strings.TrimSpace(*authTokenSecret)
		if accountSIDSecretName == "" || authTokenSecretName == "" {
			fmt.Fprintln(stderr, "request failed: --account-sid-secret and --auth-token-secret are required")
			return 1
		}

		if strings.TrimSpace(*accountSIDValue) != "" || strings.TrimSpace(*accountSIDFile) != "" {
			resolvedValue, resolveErr := resolveSecretValue(*accountSIDValue, *accountSIDFile)
			if resolveErr != nil {
				fmt.Fprintf(stderr, "request failed: %v\n", resolveErr)
				return 1
			}
			if _, err := manager.Put(workspace, accountSIDSecretName, resolvedValue); err != nil {
				fmt.Fprintf(stderr, "request failed: store twilio account sid secret: %v\n", err)
				return 1
			}
		}
		if strings.TrimSpace(*authTokenValue) != "" || strings.TrimSpace(*authTokenFile) != "" {
			resolvedValue, resolveErr := resolveSecretValue(*authTokenValue, *authTokenFile)
			if resolveErr != nil {
				fmt.Fprintf(stderr, "request failed: %v\n", resolveErr)
				return 1
			}
			if _, err := manager.Put(workspace, authTokenSecretName, resolvedValue); err != nil {
				fmt.Fprintf(stderr, "request failed: store twilio auth token secret: %v\n", err)
				return 1
			}
		}
		if client == nil {
			fmt.Fprintln(stderr, "request failed: daemon client is not configured")
			return 1
		}
		config, err := client.TwilioSet(ctx, transport.TwilioSetRequest{
			WorkspaceID:          workspace,
			AccountSIDSecretName: accountSIDSecretName,
			AuthTokenSecretName:  authTokenSecretName,
			SMSNumber:            strings.TrimSpace(*smsNumber),
			VoiceNumber:          strings.TrimSpace(*voiceNumber),
			Endpoint:             strings.TrimSpace(*endpoint),
		}, correlationID)
		if err != nil {
			return writeError(stderr, err)
		}
		return writeJSON(stdout, config)
	case "get":
		flags := flag.NewFlagSet("connector twilio get", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}

		if client == nil {
			fmt.Fprintln(stderr, "request failed: daemon client is not configured")
			return 1
		}
		config, err := client.TwilioGet(ctx, transport.TwilioGetRequest{
			WorkspaceID: normalizeWorkspace(*workspaceID),
		}, correlationID)
		if err != nil {
			return writeError(stderr, err)
		}
		return writeJSON(stdout, config)
	case "check":
		flags := flag.NewFlagSet("connector twilio check", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}

		if client == nil {
			fmt.Fprintln(stderr, "request failed: daemon client is not configured")
			return 1
		}
		response, err := client.TwilioCheck(ctx, transport.TwilioCheckRequest{
			WorkspaceID: normalizeWorkspace(*workspaceID),
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
	case "sms-chat":
		if client == nil {
			fmt.Fprintln(stderr, "request failed: daemon client is not configured")
			return 1
		}
		return runConnectorTwilioSMSChatDaemonCommand(ctx, client, args[1:], correlationID, stdout, stderr)
	case "start-call":
		flags := flag.NewFlagSet("connector twilio start-call", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		toAddress := flags.String("to", "", "destination phone number (E.164)")
		fromAddress := flags.String("from", "", "originating Twilio number (defaults to configured voice number)")
		twimlURL := flags.String("twiml-url", "", "public TwiML URL for call instructions")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}

		if client == nil {
			fmt.Fprintln(stderr, "request failed: daemon client is not configured")
			return 1
		}
		response, err := client.TwilioStartCall(ctx, transport.TwilioStartCallRequest{
			WorkspaceID: normalizeWorkspace(*workspaceID),
			To:          strings.TrimSpace(*toAddress),
			From:        strings.TrimSpace(*fromAddress),
			TwimlURL:    strings.TrimSpace(*twimlURL),
		}, correlationID)
		if err != nil {
			return writeError(stderr, err)
		}
		return writeJSON(stdout, response)
	case "call-status":
		flags := flag.NewFlagSet("connector twilio call-status", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		callSID := flags.String("call-sid", "", "optional Twilio call sid filter")
		limit := flags.Int("limit", 20, "max records to return when listing")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}

		if client == nil {
			fmt.Fprintln(stderr, "request failed: daemon client is not configured")
			return 1
		}
		response, err := client.TwilioCallStatus(ctx, transport.TwilioCallStatusRequest{
			WorkspaceID: normalizeWorkspace(*workspaceID),
			CallSID:     strings.TrimSpace(*callSID),
			Limit:       *limit,
		}, correlationID)
		if err != nil {
			return writeError(stderr, err)
		}
		return writeJSON(stdout, response)
	case "transcript":
		flags := flag.NewFlagSet("connector twilio transcript", flag.ContinueOnError)
		flags.SetOutput(stderr)

		workspaceID := flags.String("workspace", "", "workspace id")
		threadID := flags.String("thread-id", "", "optional thread id filter")
		callSID := flags.String("call-sid", "", "optional call sid filter")
		limit := flags.Int("limit", 50, "max transcript events")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}

		if client == nil {
			fmt.Fprintln(stderr, "request failed: daemon client is not configured")
			return 1
		}
		response, err := client.TwilioTranscript(ctx, transport.TwilioTranscriptRequest{
			WorkspaceID: normalizeWorkspace(*workspaceID),
			ThreadID:    strings.TrimSpace(*threadID),
			CallSID:     strings.TrimSpace(*callSID),
			Limit:       *limit,
		}, correlationID)
		if err != nil {
			return writeError(stderr, err)
		}
		return writeJSON(stdout, response)
	case "ingest-sms":
		if client == nil {
			fmt.Fprintln(stderr, "request failed: daemon client is not configured")
			return 1
		}
		return runConnectorTwilioIngestSMSDaemonCommand(ctx, client, args[1:], correlationID, stdout, stderr)
	case "ingest-voice":
		if client == nil {
			fmt.Fprintln(stderr, "request failed: daemon client is not configured")
			return 1
		}
		return runConnectorTwilioIngestVoiceDaemonCommand(ctx, client, args[1:], correlationID, stdout, stderr)
	case "webhook":
		if client == nil {
			fmt.Fprintln(stderr, "request failed: daemon client is not configured")
			return 1
		}
		return runConnectorTwilioWebhookDaemonCommand(ctx, client, args[1:], correlationID, stdout, stderr)
	default:
		writeUnknownSubcommandError(stderr, "connector twilio subcommand", args[0])
		return 2
	}
}

type twilioWorkspaceCredentials struct {
	Config     channelconfig.TwilioConfig
	AccountSID string
	AuthToken  string
}

func resolveTwilioWorkspaceCredentials(workspace string, config channelconfig.TwilioConfig) (twilioWorkspaceCredentials, error) {
	manager, err := newSecretManager()
	if err != nil {
		return twilioWorkspaceCredentials{}, fmt.Errorf("secret manager setup failed: %w", err)
	}

	_, accountSID, err := manager.Get(workspace, config.AccountSIDSecretName)
	if err != nil {
		return twilioWorkspaceCredentials{}, fmt.Errorf("resolve twilio account sid secret %q: %w", config.AccountSIDSecretName, err)
	}
	_, authToken, err := manager.Get(workspace, config.AuthTokenSecretName)
	if err != nil {
		return twilioWorkspaceCredentials{}, fmt.Errorf("resolve twilio auth token secret %q: %w", config.AuthTokenSecretName, err)
	}
	return twilioWorkspaceCredentials{
		Config:     config,
		AccountSID: accountSID,
		AuthToken:  authToken,
	}, nil
}

func recordTwilioWebhookAudit(ctx context.Context, db *sql.DB, workspaceID string, eventType string, payload map[string]any) error {
	workspace := normalizeWorkspace(workspaceID)
	if strings.TrimSpace(eventType) == "" {
		return fmt.Errorf("event type is required")
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	payloadJSON := ""
	if len(payload) > 0 {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshal audit payload: %w", err)
		}
		payloadJSON = strings.TrimSpace(string(encoded))
	}
	auditID, err := commRandomID()
	if err != nil {
		return err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin audit tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO workspaces(id, name, status, created_at, updated_at)
		VALUES (?, ?, 'ACTIVE', ?, ?)
		ON CONFLICT(id) DO UPDATE SET updated_at = excluded.updated_at
	`, workspace, workspace, now, now); err != nil {
		return fmt.Errorf("ensure workspace: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO audit_log_entries(
			id, workspace_id, run_id, step_id, event_type, actor_id, acting_as_actor_id,
			correlation_id, payload_json, created_at
		) VALUES (?, ?, NULL, NULL, ?, NULL, NULL, NULL, ?, ?)
	`, auditID, workspace, strings.TrimSpace(eventType), nullableString(payloadJSON), now); err != nil {
		return fmt.Errorf("insert audit log entry: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit audit tx: %w", err)
	}
	return nil
}

func nullableString(value string) any {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}
