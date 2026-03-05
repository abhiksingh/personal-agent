package cliapp

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"personalagent/runtime/internal/transport"
)

func runChatDaemonCommand(ctx context.Context, client *transport.Client, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer, correlationID string) int {
	flags := flag.NewFlagSet("chat", flag.ContinueOnError)
	flags.SetOutput(stderr)

	workspaceID := flags.String("workspace", "", "workspace id")
	taskClass := flags.String("task-class", "chat", "task class")
	message := flags.String("message", "", "single user message (one-shot mode)")
	providerName := flags.String("provider", "", "optional provider override: openai|anthropic|google|ollama")
	modelKey := flags.String("model", "", "optional model override")
	systemPrompt := flags.String("system", "", "optional system prompt")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	items := make([]transport.ChatTurnItem, 0)

	request := transport.ChatTurnRequest{
		WorkspaceID:      normalizeWorkspace(*workspaceID),
		TaskClass:        normalizeTaskClass(*taskClass),
		ProviderOverride: strings.TrimSpace(*providerName),
		ModelOverride:    strings.TrimSpace(*modelKey),
		SystemPrompt:     strings.TrimSpace(*systemPrompt),
	}

	oneShotMessage := strings.TrimSpace(*message)
	if oneShotMessage != "" {
		if err := runChatDaemonTurn(ctx, client, request, &items, oneShotMessage, stdout, correlationID); err != nil {
			if errors.Is(err, context.Canceled) {
				fmt.Fprintln(stderr, "chat cancelled")
				return 130
			}
			fmt.Fprintf(stderr, "request failed: %v\n", err)
			return 1
		}
		return 0
	}

	if stdin == nil {
		stdin = os.Stdin
	}
	return runInteractiveChatDaemon(ctx, client, request, &items, stdin, stdout, stderr, correlationID)
}

func runInteractiveChatDaemon(ctx context.Context, client *transport.Client, request transport.ChatTurnRequest, items *[]transport.ChatTurnItem, stdin io.Reader, stdout io.Writer, stderr io.Writer, correlationID string) int {
	scanner := bufio.NewScanner(stdin)
	scanner.Buffer(make([]byte, 1024), 1024*1024)

	fmt.Fprintln(stdout, "Chat started. Type /exit to quit.")
	for {
		fmt.Fprint(stdout, "> ")
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				fmt.Fprintf(stderr, "input error: %v\n", err)
				return 1
			}
			return 0
		}

		userInput := strings.TrimSpace(scanner.Text())
		if userInput == "" {
			continue
		}
		if userInput == "/exit" || userInput == "/quit" {
			return 0
		}

		if err := runChatDaemonTurn(ctx, client, request, items, userInput, stdout, correlationID); err != nil {
			if errors.Is(err, context.Canceled) {
				fmt.Fprintln(stderr, "chat cancelled")
				return 130
			}
			fmt.Fprintf(stderr, "request failed: %v\n", err)
			return 1
		}
	}
}

func runChatDaemonTurn(ctx context.Context, client *transport.Client, request transport.ChatTurnRequest, items *[]transport.ChatTurnItem, userInput string, stdout io.Writer, correlationID string) error {
	*items = append(*items, transport.ChatTurnItem{
		Type:    "user_message",
		Role:    "user",
		Status:  "completed",
		Content: userInput,
	})

	streamCtx, stopSignal := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stopSignal()

	turnCorrelation := newTurnCorrelationID(correlationID)
	stream, streamErr := client.ConnectRealtime(streamCtx, turnCorrelation)

	var streamedOutput atomic.Bool
	var streamedText strings.Builder
	var streamedTextMu sync.Mutex
	streamDone := make(chan struct{})
	if streamErr == nil {
		go func() {
			defer close(streamDone)
			for {
				event, err := stream.Receive()
				if err != nil {
					return
				}
				if !isAssistantRealtimeDeltaEvent(event, turnCorrelation) {
					continue
				}
				delta := event.Payload.Delta
				if delta == "" {
					continue
				}
				streamedOutput.Store(true)
				streamedTextMu.Lock()
				streamedText.WriteString(delta)
				streamedTextMu.Unlock()
				_, _ = fmt.Fprint(stdout, delta)
			}
		}()
	} else {
		close(streamDone)
	}

	request.Items = append([]transport.ChatTurnItem(nil), (*items)...)
	response, err := client.ChatTurn(streamCtx, request, turnCorrelation)

	if stream != nil {
		_ = stream.Close()
	}
	select {
	case <-streamDone:
	case <-time.After(250 * time.Millisecond):
	}

	if err != nil {
		if errors.Is(err, context.Canceled) {
			return context.Canceled
		}
		return err
	}

	assistantText := assistantMessageFromTurnItems(response.Items)
	if streamedOutput.Load() {
		streamedTextMu.Lock()
		streamed := streamedText.String()
		streamedTextMu.Unlock()
		if assistantText != "" && !strings.Contains(streamed, assistantText) {
			remainder := assistantText
			if strings.HasPrefix(assistantText, streamed) {
				remainder = assistantText[len(streamed):]
			}
			if strings.TrimSpace(remainder) != "" {
				_, _ = fmt.Fprint(stdout, remainder)
			}
		}
		fmt.Fprintln(stdout)
	} else if assistantText != "" {
		fmt.Fprintln(stdout, assistantText)
	}

	if assistantText != "" {
		*items = append(*items, response.Items...)
	}
	return nil
}

func assistantMessageFromTurnItems(items []transport.ChatTurnItem) string {
	for index := len(items) - 1; index >= 0; index-- {
		item := items[index]
		if strings.ToLower(strings.TrimSpace(item.Type)) != "assistant_message" {
			continue
		}
		content := item.Content
		if strings.TrimSpace(content) != "" {
			return content
		}
	}
	return ""
}

func isAssistantRealtimeDeltaEvent(event transport.RealtimeEventEnvelope, correlationID string) bool {
	if strings.TrimSpace(event.EventType) != "turn_item_delta" {
		return false
	}
	if strings.TrimSpace(event.CorrelationID) != strings.TrimSpace(correlationID) {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(event.Payload.ItemType), "assistant_message") {
		return false
	}
	return event.Payload.Delta != ""
}

func newTurnCorrelationID(base string) string {
	trimmed := strings.TrimSpace(base)
	if trimmed == "" {
		trimmed = "chat-turn"
	}
	return fmt.Sprintf("%s-%d", trimmed, time.Now().UTC().UnixNano())
}
