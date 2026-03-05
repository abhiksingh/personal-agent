package cliapp

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

func sortedRootCommandNames(registry map[string]cliRootCommand) []string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			continue
		}
		names = append(names, trimmed)
	}
	sort.Strings(names)
	return names
}

func writeUnknownCommandError(writer io.Writer, scope string, provided string, candidates []string) {
	fmt.Fprintln(writer, formatUnknownCommandError(scope, provided, candidates))
}

func writeUnknownSubcommandError(writer io.Writer, scope string, provided string) {
	fmt.Fprintln(writer, formatUnknownCommandError(scope, provided, inferUnknownCommandCandidates(scope)))
}

func formatUnknownCommandError(scope string, provided string, candidates []string) string {
	message := fmt.Sprintf("unknown %s %q", strings.TrimSpace(scope), strings.TrimSpace(provided))
	suggestions := suggestClosestTokens(provided, candidates, 3)
	if len(suggestions) == 1 {
		message += fmt.Sprintf("; did you mean %q?", suggestions[0])
	} else if len(suggestions) > 1 {
		quoted := make([]string, 0, len(suggestions))
		for _, suggestion := range suggestions {
			quoted = append(quoted, fmt.Sprintf("%q", suggestion))
		}
		message += fmt.Sprintf("; did you mean one of: %s?", strings.Join(quoted, ", "))
	}
	message += fmt.Sprintf("; %s", unknownCommandNextStep(scope))
	return message
}

func inferUnknownCommandCandidates(scope string) []string {
	normalizedScope := strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(scope))), " ")
	if normalizedScope == "" {
		return nil
	}
	if normalizedScope == "command" {
		schema := buildCLISchemaDocument()
		return schemaCommandNames(schema.Commands)
	}
	if !strings.HasSuffix(normalizedScope, " subcommand") {
		return nil
	}

	path := strings.TrimSpace(strings.TrimSuffix(normalizedScope, " subcommand"))
	if path == "" {
		return nil
	}
	segments := strings.Fields(path)
	if len(segments) == 0 {
		return nil
	}

	schema := buildCLISchemaDocument()
	command, found := findSchemaCommand(schema.Commands, segments[0])
	if !found {
		return nil
	}
	currentSubcommands := command.Subcommands
	for _, segment := range segments[1:] {
		subcommand, subcommandFound := findSchemaSubcommand(currentSubcommands, segment)
		if !subcommandFound {
			return schemaSubcommandNames(currentSubcommands)
		}
		currentSubcommands = subcommand.Subcommands
	}
	return schemaSubcommandNames(currentSubcommands)
}

func unknownCommandNextStep(scope string) string {
	normalizedScope := strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(scope))), " ")
	if normalizedScope == "" || normalizedScope == "command" {
		return "run `personal-agent help` to view available commands"
	}
	if strings.HasSuffix(normalizedScope, " subcommand") {
		path := strings.TrimSpace(strings.TrimSuffix(normalizedScope, " subcommand"))
		if path != "" {
			return fmt.Sprintf("run `personal-agent help %s` to view available subcommands", path)
		}
	}
	return "run `personal-agent help` to view available commands"
}

func suggestClosestTokens(input string, candidates []string, limit int) []string {
	normalizedInput := strings.ToLower(strings.TrimSpace(input))
	if normalizedInput == "" || len(candidates) == 0 || limit <= 0 {
		return nil
	}

	maxDistance := 2
	if len(normalizedInput) >= 6 {
		maxDistance = 3
	}
	if len(normalizedInput) >= 10 {
		maxDistance = 4
	}

	type suggestion struct {
		candidate string
		distance  int
		prefix    bool
	}
	suggestions := make([]suggestion, 0, len(candidates))
	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		trimmedCandidate := strings.TrimSpace(candidate)
		if trimmedCandidate == "" {
			continue
		}
		if _, exists := seen[trimmedCandidate]; exists {
			continue
		}
		seen[trimmedCandidate] = struct{}{}

		normalizedCandidate := strings.ToLower(trimmedCandidate)
		distance := levenshteinDistance(normalizedInput, normalizedCandidate)
		prefix := strings.HasPrefix(normalizedCandidate, normalizedInput) || strings.HasPrefix(normalizedInput, normalizedCandidate)
		if !prefix && distance > maxDistance {
			continue
		}
		suggestions = append(suggestions, suggestion{
			candidate: trimmedCandidate,
			distance:  distance,
			prefix:    prefix,
		})
	}

	sort.SliceStable(suggestions, func(i int, j int) bool {
		if suggestions[i].prefix != suggestions[j].prefix {
			return suggestions[i].prefix
		}
		if suggestions[i].distance != suggestions[j].distance {
			return suggestions[i].distance < suggestions[j].distance
		}
		return suggestions[i].candidate < suggestions[j].candidate
	})

	if len(suggestions) > limit {
		suggestions = suggestions[:limit]
	}
	result := make([]string, 0, len(suggestions))
	for _, suggestion := range suggestions {
		result = append(result, suggestion.candidate)
	}
	return result
}

func levenshteinDistance(a string, b string) int {
	if a == b {
		return 0
	}
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)
	for j := 0; j <= len(b); j++ {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			insert := curr[j-1] + 1
			delete := prev[j] + 1
			replace := prev[j-1] + cost
			curr[j] = minInt(insert, minInt(delete, replace))
		}
		copy(prev, curr)
	}
	return prev[len(b)]
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
