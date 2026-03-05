package cliapp

func registerCLICoreCommands(registry map[string]cliRootCommand) {
	registerRootCommand(registry, "auth", func(commandCtx cliRootCommandContext, args []string) int {
		return runAuthCommand(args, commandCtx.stdout, commandCtx.stderr)
	})
	registerRootCommand(registry, "profile", func(commandCtx cliRootCommandContext, args []string) int {
		return runProfileCommand(args, commandCtx.stdout, commandCtx.stderr)
	})
	registerRootCommand(registry, "help", func(commandCtx cliRootCommandContext, args []string) int {
		return runHelpCommand(args, commandCtx.stdout, commandCtx.stderr)
	})
	registerRootCommand(registry, "completion", func(commandCtx cliRootCommandContext, args []string) int {
		return runCompletionCommand(args, commandCtx.stdout, commandCtx.stderr)
	})
	registerRootCommand(registry, "quickstart", func(commandCtx cliRootCommandContext, args []string) int {
		return runQuickstartCommand(commandCtx.ctx, args, commandCtx.stdout, commandCtx.stderr)
	})
	registerRootCommand(registry, "version", func(commandCtx cliRootCommandContext, args []string) int {
		return runVersionCommand(args, commandCtx.stdout, commandCtx.stderr)
	})
}
