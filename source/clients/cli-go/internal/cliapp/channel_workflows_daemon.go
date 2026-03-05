package cliapp

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"personalagent/runtime/internal/transport"
)

func runConnectorTwilioSMSChatDaemonCommand(ctx context.Context, client *transport.Client, args []string, correlationID string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("connector twilio sms-chat", flag.ContinueOnError)
	flags.SetOutput(stderr)

	workspaceID := flags.String("workspace", "", "workspace id")
	destination := flags.String("to", "", "destination phone number (E.164)")
	message := flags.String("message", "", "message text for one-shot mode")
	interactive := flags.Bool("interactive", false, "run interactive sms chat loop from stdin")
	operationID := flags.String("operation-id", "", "optional operation id (or operation id prefix in interactive mode)")
	taskClass := flags.String("task-class", "chat", "assistant task class")
	systemPrompt := flags.String("system-prompt", "", "assistant system prompt override")
	maxHistory := flags.Int("max-history", 20, "max prior thread events for assistant reply generation")
	replyTimeout := flags.Duration("reply-timeout", 12*time.Second, "timeout for assistant reply generation + delivery")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	workspace := normalizeWorkspace(*workspaceID)
	to := strings.TrimSpace(*destination)
	if to == "" {
		fmt.Fprintln(stderr, "request failed: --to is required")
		return 1
	}

	turns := make([]twilioSMSChatTurn, 0)
	sendTurn := func(opID string, body string) {
		replyTimeoutMS := int((*replyTimeout) / time.Millisecond)
		if replyTimeoutMS < 0 {
			replyTimeoutMS = 0
		}
		response, err := client.TwilioSMSChatTurn(ctx, transport.TwilioSMSChatTurnRequest{
			WorkspaceID:    workspace,
			To:             to,
			Message:        strings.TrimSpace(body),
			OperationID:    strings.TrimSpace(opID),
			TaskClass:      strings.TrimSpace(*taskClass),
			SystemPrompt:   strings.TrimSpace(*systemPrompt),
			MaxHistory:     *maxHistory,
			ReplyTimeoutMS: replyTimeoutMS,
		}, correlationID)
		turn := twilioSMSChatTurn{
			OperationID:          response.OperationID,
			Message:              response.Message,
			Success:              response.Success,
			Delivered:            response.Delivered,
			Channel:              response.Channel,
			ProviderReceipt:      response.ProviderReceipt,
			IdempotentReplay:     response.IdempotentReplay,
			ThreadID:             response.ThreadID,
			AssistantReply:       response.AssistantReply,
			AssistantOperationID: response.AssistantOperationID,
			AssistantError:       response.AssistantError,
			Error:                response.Error,
		}
		if err != nil {
			turn.OperationID = opID
			turn.Message = body
			turn.Success = false
			turn.Error = err.Error()
		}
		turns = append(turns, turn)
	}

	exitCode := 0
	trimmedMessage := strings.TrimSpace(*message)
	if !*interactive {
		if trimmedMessage == "" {
			fmt.Fprintln(stderr, "request failed: --message is required when --interactive is false")
			return 1
		}
		opID := strings.TrimSpace(*operationID)
		if opID == "" {
			resolved, err := commRandomID()
			if err != nil {
				fmt.Fprintf(stderr, "request failed: %v\n", err)
				return 1
			}
			opID = resolved
		}
		sendTurn(opID, trimmedMessage)
		if len(turns) > 0 && !turns[len(turns)-1].Success {
			exitCode = 1
		}
	} else {
		baseOperationID := strings.TrimSpace(*operationID)
		if baseOperationID == "" {
			resolved, err := commRandomID()
			if err != nil {
				fmt.Fprintf(stderr, "request failed: %v\n", err)
				return 1
			}
			baseOperationID = "sms-chat-" + resolved
		}

		turnIndex := 0
		if trimmedMessage != "" {
			opID := fmt.Sprintf("%s-%03d", baseOperationID, turnIndex)
			sendTurn(opID, trimmedMessage)
			if len(turns) > 0 && !turns[len(turns)-1].Success {
				exitCode = 1
			}
			turnIndex++
		}

		scanner := bufio.NewScanner(chatInput)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			if strings.EqualFold(line, "/exit") || strings.EqualFold(line, "exit") {
				break
			}
			opID := fmt.Sprintf("%s-%03d", baseOperationID, turnIndex)
			sendTurn(opID, line)
			if len(turns) > 0 && !turns[len(turns)-1].Success {
				exitCode = 1
			}
			turnIndex++
		}
		if err := scanner.Err(); err != nil {
			fmt.Fprintf(stderr, "request failed: read interactive input: %v\n", err)
			return 1
		}
	}

	if code := writeJSON(stdout, map[string]any{
		"workspace_id": workspace,
		"to":           to,
		"interactive":  *interactive,
		"turns":        turns,
	}); code != 0 {
		return code
	}
	return exitCode
}
