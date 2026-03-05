package cliapp

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

type rootHelpWorkflowExample struct {
	Command string
}

type rootHelpSkimGroup struct {
	Title    string
	Commands []string
}

func printUsage(writer io.Writer) {
	schema := buildCLISchemaDocument()
	fmt.Fprintln(writer, "Usage: personal-agent [--mode tcp|unix|named_pipe] [--address <value>] [--runtime-profile local|prod] [--auth-token <token>] [--auth-token-file <path>] [--tls-ca-file <path>] [--tls-client-cert-file <path>] [--tls-client-key-file <path>] [--tls-server-name <name>] [--tls-insecure-skip-verify=<true|false>] [--db <path>] [--correlation-id <id>] [--output json|json-compact|text] [--error-output text|json] [--timeout <duration>] <command>")

	fmt.Fprintln(writer, "Quickstart workflows (copy/paste):")
	for _, workflow := range rootHelpWorkflowExamples() {
		fmt.Fprintf(writer, "  personal-agent %s\n", workflow.Command)
	}

	fmt.Fprintln(writer, "Skim command groups:")
	for _, group := range rootHelpSkimGroups() {
		if len(group.Commands) == 0 {
			continue
		}
		fmt.Fprintf(writer, "  %s: %s\n", group.Title, strings.Join(group.Commands, ", "))
	}

	fmt.Fprintln(writer, "Full command reference (generated from schema):")
	for _, command := range sortedSchemaCommands(schema.Commands) {
		summary := strings.TrimSpace(command.Summary)
		if summary == "" {
			fmt.Fprintf(writer, "  %s\n", command.Name)
			continue
		}
		fmt.Fprintf(writer, "  %-12s %s\n", command.Name, summary)
	}

	fmt.Fprintln(writer, "Help tips:")
	fmt.Fprintln(writer, "  personal-agent help <command>")
	fmt.Fprintln(writer, "  personal-agent <command> --help")
	fmt.Fprintln(writer, "  personal-agent <command> <subcommand> --help")
}

func rootHelpWorkflowExamples() []rootHelpWorkflowExample {
	return []rootHelpWorkflowExample{
		{Command: "quickstart --workspace ws1 --provider openai --api-key-file ~/.secrets/openai.key"},
		{Command: "doctor --workspace ws1 --include-optional=false"},
		{Command: "chat --workspace ws1 --task-class chat --message \"Summarize my inbox\""},
		{Command: "agent run --workspace ws1 --request \"send an update to +15550001111\""},
		{Command: "task retry --task-id <task-id>"},
	}
}

func rootHelpSkimGroups() []rootHelpSkimGroup {
	return []rootHelpSkimGroup{
		{
			Title:    "setup/local",
			Commands: []string{"help", "completion", "quickstart", "version", "auth", "profile", "meta schema"},
		},
		{
			Title:    "agent/runtime",
			Commands: []string{"doctor", "assistant", "chat", "agent", "task", "delegation", "identity"},
		},
		{
			Title:    "communications",
			Commands: []string{"comm", "channel", "connector"},
		},
		{
			Title:    "inspect/ops",
			Commands: []string{"automation", "inspect", "retention", "context", "meta capabilities", "stream", "smoke"},
		},
	}
}

func sortedSchemaCommands(commands []cliCommandSchema) []cliCommandSchema {
	sorted := append([]cliCommandSchema(nil), commands...)
	sort.Slice(sorted, func(i int, j int) bool {
		return strings.ToLower(strings.TrimSpace(sorted[i].Name)) < strings.ToLower(strings.TrimSpace(sorted[j].Name))
	})
	return sorted
}

func rootCommandToken(commandLine string) string {
	fields := strings.Fields(strings.TrimSpace(commandLine))
	if len(fields) == 0 {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(fields[0]))
}
