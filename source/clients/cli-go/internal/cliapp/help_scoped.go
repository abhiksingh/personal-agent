package cliapp

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

func runHelpCommand(args []string, stdout io.Writer, stderr io.Writer) int {
	return renderCLIHelpForPath(parseHelpTopicPath(args), buildCLISchemaDocument(), stdout, stderr)
}

func resolveInlineHelpPath(command string, args []string) ([]string, bool) {
	helpIndex := -1
	for i, raw := range args {
		if isHelpToken(raw) {
			helpIndex = i
			break
		}
	}
	if helpIndex < 0 {
		return nil, false
	}

	path := []string{strings.TrimSpace(command)}
	for _, raw := range args[:helpIndex] {
		token := strings.TrimSpace(raw)
		if token == "" {
			continue
		}
		if strings.HasPrefix(token, "-") {
			break
		}
		path = append(path, token)
	}
	return normalizeHelpPath(path), true
}

func parseHelpTopicPath(args []string) []string {
	path := make([]string, 0, len(args))
	for _, raw := range args {
		token := strings.TrimSpace(raw)
		if token == "" || isHelpToken(token) || strings.HasPrefix(token, "-") {
			continue
		}
		path = append(path, token)
	}
	return normalizeHelpPath(path)
}

func renderCLIHelpForPath(path []string, schema cliSchemaDocument, stdout io.Writer, stderr io.Writer) int {
	normalizedPath := normalizeHelpPath(path)
	if len(normalizedPath) == 0 {
		printUsage(stdout)
		return 0
	}

	command, found := findSchemaCommand(schema.Commands, normalizedPath[0])
	if !found {
		writeUnknownCommandError(stderr, "command", normalizedPath[0], schemaCommandNames(schema.Commands))
		return 2
	}
	if len(normalizedPath) == 1 {
		printScopedCommandUsage(stdout, command)
		return 0
	}

	subcommand, subcommandFound := findSchemaSubcommand(command.Subcommands, normalizedPath[1])
	if !subcommandFound {
		// Keep help deterministic even when schema omits deeper command trees.
		if len(command.Subcommands) == 0 {
			printScopedCommandUsage(stdout, command)
			return 0
		}
		writeUnknownCommandError(stderr, fmt.Sprintf("%s subcommand", command.Name), normalizedPath[1], schemaSubcommandNames(command.Subcommands))
		printScopedCommandUsage(stdout, command)
		return 2
	}
	printScopedSubcommandUsage(stdout, command, subcommand)
	return 0
}

func printScopedCommandUsage(writer io.Writer, command cliCommandSchema) {
	fmt.Fprintf(writer, "Usage: personal-agent %s", command.Name)
	if len(command.Subcommands) > 0 {
		fmt.Fprint(writer, " <subcommand>")
	}
	fmt.Fprintln(writer, " [flags]")
	if strings.TrimSpace(command.Summary) != "" {
		fmt.Fprintf(writer, "Summary: %s\n", strings.TrimSpace(command.Summary))
	}
	if len(command.Subcommands) > 0 {
		fmt.Fprintln(writer, "Subcommands:")
		for _, subcommand := range sortSchemaSubcommands(command.Subcommands) {
			summary := strings.TrimSpace(subcommand.Summary)
			if summary == "" {
				fmt.Fprintf(writer, "  %s\n", subcommand.Name)
				continue
			}
			fmt.Fprintf(writer, "  %s\t%s\n", subcommand.Name, summary)
		}
		fmt.Fprintf(writer, "Tip: personal-agent help %s <subcommand>\n", command.Name)
	}
}

func printScopedSubcommandUsage(writer io.Writer, command cliCommandSchema, subcommand cliCommandSchema) {
	fmt.Fprintf(writer, "Usage: personal-agent %s %s [flags]\n", command.Name, subcommand.Name)
	if strings.TrimSpace(subcommand.Summary) != "" {
		fmt.Fprintf(writer, "Summary: %s\n", strings.TrimSpace(subcommand.Summary))
	}
	if len(subcommand.RequiredFlags) > 0 {
		fmt.Fprintln(writer, "Required flags:")
		for _, requiredFlag := range subcommand.RequiredFlags {
			trimmed := strings.TrimSpace(requiredFlag)
			if trimmed == "" {
				continue
			}
			fmt.Fprintf(writer, "  %s\n", trimmed)
		}
	}
	fmt.Fprintf(writer, "Tip: personal-agent help %s\n", command.Name)
}

func isHelpToken(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "-h", "--help", "help":
		return true
	default:
		return false
	}
}

func normalizeHelpPath(path []string) []string {
	normalized := make([]string, 0, len(path))
	for _, raw := range path {
		token := strings.TrimSpace(raw)
		if token == "" {
			continue
		}
		normalized = append(normalized, token)
	}
	return normalized
}

func findSchemaCommand(commands []cliCommandSchema, commandName string) (cliCommandSchema, bool) {
	target := strings.ToLower(strings.TrimSpace(commandName))
	for _, command := range commands {
		if strings.ToLower(strings.TrimSpace(command.Name)) == target {
			return command, true
		}
	}
	return cliCommandSchema{}, false
}

func findSchemaSubcommand(subcommands []cliCommandSchema, subcommandName string) (cliCommandSchema, bool) {
	target := strings.ToLower(strings.TrimSpace(subcommandName))
	for _, subcommand := range subcommands {
		if strings.ToLower(strings.TrimSpace(subcommand.Name)) == target {
			return subcommand, true
		}
	}
	return cliCommandSchema{}, false
}

func schemaCommandNames(commands []cliCommandSchema) []string {
	names := make([]string, 0, len(commands))
	for _, command := range commands {
		name := strings.TrimSpace(command.Name)
		if name == "" {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func schemaSubcommandNames(subcommands []cliCommandSchema) []string {
	names := make([]string, 0, len(subcommands))
	for _, subcommand := range subcommands {
		name := strings.TrimSpace(subcommand.Name)
		if name == "" {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func sortSchemaSubcommands(subcommands []cliCommandSchema) []cliCommandSchema {
	sorted := append([]cliCommandSchema(nil), subcommands...)
	sort.Slice(sorted, func(i int, j int) bool {
		return strings.ToLower(strings.TrimSpace(sorted[i].Name)) < strings.ToLower(strings.TrimSpace(sorted[j].Name))
	})
	return sorted
}
