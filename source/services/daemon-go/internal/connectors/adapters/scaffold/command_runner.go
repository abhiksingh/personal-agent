package scaffold

import (
	"context"
	"os/exec"
	"strings"
)

type CommandResult struct {
	Output string
	Err    error
}

type CommandRunner func(ctx context.Context, name string, args ...string) CommandResult

var DefaultCommandRunner CommandRunner = func(ctx context.Context, name string, args ...string) CommandResult {
	output, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	return CommandResult{
		Output: strings.TrimSpace(string(output)),
		Err:    err,
	}
}

func ExecuteCommand(ctx context.Context, runner CommandRunner, name string, args ...string) CommandResult {
	if runner == nil {
		runner = DefaultCommandRunner
	}
	return runner(ctx, name, args...)
}
