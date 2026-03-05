package cliapp

import (
	"context"
	"io"

	"personalagent/runtime/internal/transport"
)

func registerCLIAgentCommands(registry map[string]cliRootCommand) {
	registerDaemonRootCommand(registry, "provider", func(ctx context.Context, client *transport.Client, args []string, correlationID string, stdout io.Writer, stderr io.Writer) int {
		return runProviderDaemonCommand(ctx, client, args, correlationID, stdout, stderr)
	})
	registerDaemonRootCommand(registry, "model", func(ctx context.Context, client *transport.Client, args []string, correlationID string, stdout io.Writer, stderr io.Writer) int {
		return runModelDaemonCommand(ctx, client, args, correlationID, stdout, stderr)
	})
	registerDaemonRootCommandWithInput(registry, "chat", func(ctx context.Context, client *transport.Client, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer, correlationID string) int {
		return runChatDaemonCommand(ctx, client, args, stdin, stdout, stderr, correlationID)
	})
	registerDaemonRootCommandWithInput(registry, "assistant", func(ctx context.Context, client *transport.Client, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer, correlationID string) int {
		return runAssistantCommand(ctx, client, args, stdin, stdout, stderr, correlationID)
	})
	registerDaemonRootCommand(registry, "agent", func(ctx context.Context, client *transport.Client, args []string, correlationID string, stdout io.Writer, stderr io.Writer) int {
		return runAgentCommand(ctx, client, args, correlationID, stdout, stderr)
	})
	registerDaemonRootCommand(registry, "delegation", func(ctx context.Context, client *transport.Client, args []string, correlationID string, stdout io.Writer, stderr io.Writer) int {
		return runDelegationCommand(ctx, client, args, correlationID, stdout, stderr)
	})
	registerDaemonRootCommand(registry, "identity", func(ctx context.Context, client *transport.Client, args []string, correlationID string, stdout io.Writer, stderr io.Writer) int {
		return runIdentityCommand(ctx, client, args, correlationID, stdout, stderr)
	})
}
