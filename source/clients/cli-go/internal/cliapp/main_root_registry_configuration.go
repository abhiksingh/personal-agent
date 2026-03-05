package cliapp

import (
	"context"
	"io"
	"strings"

	"personalagent/runtime/internal/transport"
)

func registerCLIConfigurationCommands(registry map[string]cliRootCommand) {
	registerRootCommand(registry, "meta", func(commandCtx cliRootCommandContext, args []string) int {
		if len(args) > 0 && strings.EqualFold(strings.TrimSpace(args[0]), "capabilities") {
			return commandCtx.withDaemonClient(func(client *transport.Client) int {
				return runMetaCapabilities(commandCtx.ctx, client, commandCtx.correlationID, commandCtx.stdout, commandCtx.stderr)
			})
		}
		return runMetaCommand(args, commandCtx.stdout, commandCtx.stderr)
	})
	registerRootCommand(registry, "connector", func(commandCtx cliRootCommandContext, args []string) int {
		if connectorSubcommandRequiresDaemon(args) {
			return commandCtx.withDaemonClient(func(client *transport.Client) int {
				return runConnectorCommand(commandCtx.ctx, client, args, commandCtx.correlationID, commandCtx.stdout, commandCtx.stderr)
			})
		}
		return runConnectorCommand(commandCtx.ctx, nil, args, commandCtx.correlationID, commandCtx.stdout, commandCtx.stderr)
	})
	registerDaemonRootCommand(registry, "secret", func(ctx context.Context, client *transport.Client, args []string, correlationID string, stdout io.Writer, stderr io.Writer) int {
		return runSecretCommand(ctx, client, args, correlationID, stdout, stderr)
	})
	registerRootCommand(registry, "channel", func(commandCtx cliRootCommandContext, args []string) int {
		return commandCtx.withDaemonClient(func(client *transport.Client) int {
			return runChannelCommand(commandCtx.ctx, client, args, commandCtx.dbPath, commandCtx.correlationID, commandCtx.stdout, commandCtx.stderr)
		})
	})
}
