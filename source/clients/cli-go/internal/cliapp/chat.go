package cliapp

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"personalagent/runtime/internal/chatruntime"
	"personalagent/runtime/internal/modelpolicy"
	"personalagent/runtime/internal/providerconfig"
)

type chatRoute struct {
	Provider string
	ModelKey string
	Endpoint string
	APIKey   string
}

func runInteractiveChat(
	ctx context.Context,
	route chatRoute,
	messages *[]chatruntime.Message,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
) int {
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

		err := runChatTurn(ctx, route, messages, userInput, stdout)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				fmt.Fprintln(stderr, "chat cancelled")
				return 130
			}
			fmt.Fprintf(stderr, "request failed: %v\n", err)
			return 1
		}
	}
}

func runChatTurn(ctx context.Context, route chatRoute, messages *[]chatruntime.Message, userInput string, stdout io.Writer) error {
	*messages = append(*messages, chatruntime.Message{
		Role:    "user",
		Content: userInput,
	})

	streamCtx, stopSignal := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stopSignal()

	builder := &strings.Builder{}
	err := chatruntime.StreamAssistantResponse(streamCtx, newCLIHTTPClientFromContext(streamCtx), chatruntime.StreamRequest{
		Provider: route.Provider,
		Endpoint: route.Endpoint,
		ModelKey: route.ModelKey,
		APIKey:   route.APIKey,
		Messages: *messages,
	}, func(delta string) error {
		builder.WriteString(delta)
		_, writeErr := fmt.Fprint(stdout, delta)
		return writeErr
	})
	fmt.Fprintln(stdout)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return context.Canceled
		}
		return err
	}

	assistantText := strings.TrimSpace(builder.String())
	if assistantText != "" {
		*messages = append(*messages, chatruntime.Message{
			Role:    "assistant",
			Content: assistantText,
		})
	}
	return nil
}

func resolveChatRoute(
	ctx context.Context,
	workspaceID string,
	taskClass string,
	providerOverride string,
	modelOverride string,
	modelStore *modelpolicy.SQLiteStore,
	providerStore *providerconfig.SQLiteStore,
) (chatRoute, error) {
	var providerName string
	var modelKey string

	if providerOverride != "" || modelOverride != "" {
		if providerOverride == "" || modelOverride == "" {
			return chatRoute{}, fmt.Errorf("both --provider and --model are required when overriding route")
		}
		normalizedProvider, err := providerconfig.NormalizeProvider(providerOverride)
		if err != nil {
			return chatRoute{}, err
		}
		if !modelpolicy.IsSupportedModel(normalizedProvider, modelOverride) {
			return chatRoute{}, modelpolicy.ErrModelNotFound
		}
		providerName = normalizedProvider
		modelKey = modelOverride
	} else {
		resolved, err := resolveModelRoute(ctx, modelStore, providerStore, workspaceID, taskClass)
		if err != nil {
			return chatRoute{}, err
		}
		providerName = resolved.Provider
		modelKey = resolved.ModelKey
	}

	providerConfig, err := providerStore.Get(ctx, workspaceID, providerName)
	if err != nil {
		return chatRoute{}, fmt.Errorf("load provider config for %s: %w", providerName, err)
	}

	apiKey := ""
	secretName := strings.TrimSpace(providerConfig.APIKeySecretName)
	if secretName != "" {
		manager, err := newSecretManager()
		if err != nil {
			return chatRoute{}, fmt.Errorf("secret manager setup failed: %w", err)
		}
		_, value, err := manager.Get(workspaceID, secretName)
		if err != nil {
			return chatRoute{}, fmt.Errorf("resolve secret %q: %w", secretName, err)
		}
		apiKey = value
	}

	return chatRoute{
		Provider: providerName,
		ModelKey: modelKey,
		Endpoint: providerConfig.Endpoint,
		APIKey:   apiKey,
	}, nil
}
