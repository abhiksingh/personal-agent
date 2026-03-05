package cliapp

import (
	"flag"
	"fmt"
	"io"
	"strings"
)

func runCompletionCommand(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("completion", flag.ContinueOnError)
	flags.SetOutput(stderr)

	shellName := flags.String("shell", "", "shell name: bash|zsh|fish")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	selectedShell := strings.ToLower(strings.TrimSpace(*shellName))
	remaining := flags.Args()
	if selectedShell == "" && len(remaining) > 0 {
		selectedShell = strings.ToLower(strings.TrimSpace(remaining[0]))
		remaining = remaining[1:]
	}
	if len(remaining) > 0 {
		fmt.Fprintln(stderr, "completion accepts at most one positional shell argument")
		return 2
	}
	if selectedShell == "" {
		fmt.Fprintln(stderr, "completion shell required: bash|zsh|fish")
		return 2
	}

	script, err := renderCompletionScript(selectedShell, buildCLISchemaDocument())
	if err != nil {
		fmt.Fprintf(stderr, "completion generation failed: %v\n", err)
		return 2
	}
	if _, err := io.WriteString(stdout, script); err != nil {
		fmt.Fprintf(stderr, "completion write failed: %v\n", err)
		return 1
	}
	return 0
}

func renderCompletionScript(shellName string, schema cliSchemaDocument) (string, error) {
	switch strings.ToLower(strings.TrimSpace(shellName)) {
	case completionShellBash:
		return renderBashCompletionScript(schema), nil
	case completionShellFish:
		return renderFishCompletionScript(schema), nil
	case completionShellZsh:
		return renderZshCompletionScript(schema), nil
	default:
		return "", fmt.Errorf("unsupported shell %q (supported: bash|zsh|fish)", shellName)
	}
}

func renderBashCompletionScript(schema cliSchemaDocument) string {
	index := buildCompletionCommandIndex(schema)
	rootCompletions := sanitizeCompletionTokens(append(append([]string{}, index.rootCommands...), index.globalFlags...))
	globalFlagTokens := sanitizeCompletionTokens(append([]string{}, index.globalFlags...))
	pathKeysWithSubcommands := sortedCompletionMapKeys(index.subcommandsByPath)
	pathKeysWithFlags := sortedCompletionMapKeys(index.flagsByPath)

	var builder strings.Builder
	builder.WriteString("# shellcheck shell=bash\n")
	writeBashPathExistsFunction(&builder, index.knownPaths)
	writeBashLookupFunction(&builder, "_pa_subcommands_for_path", index.subcommandsByPath, pathKeysWithSubcommands)
	writeBashLookupFunction(&builder, "_pa_flags_for_path", index.flagsByPath, pathKeysWithFlags)
	builder.WriteString("_pa_resolve_path() {\n")
	builder.WriteString("  local path token candidate i\n")
	builder.WriteString("  path=\"\"\n")
	builder.WriteString("  for ((i=1; i<COMP_CWORD; i++)); do\n")
	builder.WriteString("    token=\"${COMP_WORDS[i]}\"\n")
	builder.WriteString("    if [[ -z \"$token\" || \"$token\" == -* ]]; then\n")
	builder.WriteString("      continue\n")
	builder.WriteString("    fi\n")
	builder.WriteString("    candidate=\"$token\"\n")
	builder.WriteString("    if [[ -n \"$path\" ]]; then\n")
	builder.WriteString("      candidate=\"$path $token\"\n")
	builder.WriteString("    fi\n")
	builder.WriteString("    if _pa_path_exists \"$candidate\"; then\n")
	builder.WriteString("      path=\"$candidate\"\n")
	builder.WriteString("    fi\n")
	builder.WriteString("  done\n")
	builder.WriteString("  printf '%s' \"$path\"\n")
	builder.WriteString("}\n")
	builder.WriteString("_personal_agent_completion() {\n")
	builder.WriteString("  local cur path path_subcommands path_flags\n")
	builder.WriteString("  COMPREPLY=()\n")
	builder.WriteString("  cur=\"${COMP_WORDS[COMP_CWORD]}\"\n")
	builder.WriteString(fmt.Sprintf("  local global_flags=%q\n", strings.Join(globalFlagTokens, " ")))
	builder.WriteString(fmt.Sprintf("  local root_completions=%q\n", strings.Join(rootCompletions, " ")))
	builder.WriteString("  if [[ ${COMP_CWORD} -eq 1 ]]; then\n")
	builder.WriteString("    COMPREPLY=( $(compgen -W \"$root_completions\" -- \"$cur\") )\n")
	builder.WriteString("    return 0\n")
	builder.WriteString("  fi\n")
	builder.WriteString("  path=\"$(_pa_resolve_path)\"\n")
	builder.WriteString("  if [[ -z \"$path\" ]]; then\n")
	builder.WriteString("    COMPREPLY=( $(compgen -W \"$root_completions\" -- \"$cur\") )\n")
	builder.WriteString("    return 0\n")
	builder.WriteString("  fi\n")
	builder.WriteString("  path_subcommands=\"$(_pa_subcommands_for_path \"$path\" || true)\"\n")
	builder.WriteString("  path_flags=\"$(_pa_flags_for_path \"$path\" || true)\"\n")
	builder.WriteString("  if [[ \"$cur\" == -* ]]; then\n")
	builder.WriteString("    COMPREPLY=( $(compgen -W \"$global_flags $path_flags --help -h\" -- \"$cur\") )\n")
	builder.WriteString("    return 0\n")
	builder.WriteString("  fi\n")
	builder.WriteString("  COMPREPLY=( $(compgen -W \"$path_subcommands $path_flags\" -- \"$cur\") )\n")
	builder.WriteString("  return 0\n")
	builder.WriteString("}\n")
	builder.WriteString("complete -F _personal_agent_completion personal-agent\n")
	return builder.String()
}

func renderFishCompletionScript(schema cliSchemaDocument) string {
	index := buildCompletionCommandIndex(schema)
	rootCompletions := sanitizeCompletionTokens(append(append([]string{}, index.rootCommands...), index.globalFlags...))
	globalFlagTokens := sanitizeCompletionTokens(append([]string{}, index.globalFlags...))
	pathKeysWithSubcommands := sortedCompletionMapKeys(index.subcommandsByPath)
	pathKeysWithFlags := sortedCompletionMapKeys(index.flagsByPath)

	var builder strings.Builder
	builder.WriteString("function __pa_path_exists\n")
	builder.WriteString("  switch \"$argv[1]\"\n")
	for _, pathKey := range index.knownPaths {
		builder.WriteString(fmt.Sprintf("    case %q\n", pathKey))
		builder.WriteString("      return 0\n")
	}
	builder.WriteString("  end\n")
	builder.WriteString("  return 1\n")
	builder.WriteString("end\n")

	writeFishLookupFunction(&builder, "__pa_subcommands_for_path", index.subcommandsByPath, pathKeysWithSubcommands)
	writeFishLookupFunction(&builder, "__pa_flags_for_path", index.flagsByPath, pathKeysWithFlags)

	builder.WriteString("function __pa_resolve_path\n")
	builder.WriteString("  set -l tokens (commandline -opc)\n")
	builder.WriteString("  set -l path \"\"\n")
	builder.WriteString("  for token in $tokens[2..-1]\n")
	builder.WriteString("    if test -z \"$token\"\n")
	builder.WriteString("      continue\n")
	builder.WriteString("    end\n")
	builder.WriteString("    if string match -rq '^-' -- \"$token\"\n")
	builder.WriteString("      continue\n")
	builder.WriteString("    end\n")
	builder.WriteString("    set -l candidate \"$token\"\n")
	builder.WriteString("    if test -n \"$path\"\n")
	builder.WriteString("      set candidate \"$path $token\"\n")
	builder.WriteString("    end\n")
	builder.WriteString("    if __pa_path_exists \"$candidate\"\n")
	builder.WriteString("      set path \"$candidate\"\n")
	builder.WriteString("    end\n")
	builder.WriteString("  end\n")
	builder.WriteString("  echo \"$path\"\n")
	builder.WriteString("end\n")

	builder.WriteString("function __pa_dynamic_completions\n")
	builder.WriteString(fmt.Sprintf("  set -l root_completions %s\n", strings.Join(rootCompletions, " ")))
	builder.WriteString(fmt.Sprintf("  set -l global_flags %s\n", strings.Join(globalFlagTokens, " ")))
	builder.WriteString("  set -l token (commandline -ct)\n")
	builder.WriteString("  set -l tokens (commandline -opc)\n")
	builder.WriteString("  if test (count $tokens) -le 1\n")
	builder.WriteString("    for value in $root_completions\n")
	builder.WriteString("      echo $value\n")
	builder.WriteString("    end\n")
	builder.WriteString("    return\n")
	builder.WriteString("  end\n")
	builder.WriteString("  set -l path (__pa_resolve_path)\n")
	builder.WriteString("  if string match -rq '^-' -- \"$token\"\n")
	builder.WriteString("    for value in $global_flags\n")
	builder.WriteString("      echo $value\n")
	builder.WriteString("    end\n")
	builder.WriteString("    for value in (__pa_flags_for_path \"$path\")\n")
	builder.WriteString("      echo $value\n")
	builder.WriteString("    end\n")
	builder.WriteString("    echo --help\n")
	builder.WriteString("    echo -h\n")
	builder.WriteString("    return\n")
	builder.WriteString("  end\n")
	builder.WriteString("  if test -z \"$path\"\n")
	builder.WriteString("    for value in $root_completions\n")
	builder.WriteString("      echo $value\n")
	builder.WriteString("    end\n")
	builder.WriteString("    return\n")
	builder.WriteString("  end\n")
	builder.WriteString("  for value in (__pa_subcommands_for_path \"$path\")\n")
	builder.WriteString("    echo $value\n")
	builder.WriteString("  end\n")
	builder.WriteString("  for value in (__pa_flags_for_path \"$path\")\n")
	builder.WriteString("    echo $value\n")
	builder.WriteString("  end\n")
	builder.WriteString("end\n")

	builder.WriteString("complete -c personal-agent -f -a '(__pa_dynamic_completions)'\n")
	return builder.String()
}

func renderZshCompletionScript(schema cliSchemaDocument) string {
	index := buildCompletionCommandIndex(schema)
	rootCompletions := sanitizeCompletionTokens(append(append([]string{}, index.rootCommands...), index.globalFlags...))
	globalFlagTokens := sanitizeCompletionTokens(append([]string{}, index.globalFlags...))
	pathKeysWithSubcommands := sortedCompletionMapKeys(index.subcommandsByPath)
	pathKeysWithFlags := sortedCompletionMapKeys(index.flagsByPath)

	var builder strings.Builder
	builder.WriteString("#compdef personal-agent\n")
	writeZshPathExistsFunction(&builder, index.knownPaths)
	writeZshLookupFunction(&builder, "_pa_subcommands_for_path", index.subcommandsByPath, pathKeysWithSubcommands)
	writeZshLookupFunction(&builder, "_pa_flags_for_path", index.flagsByPath, pathKeysWithFlags)
	builder.WriteString("_pa_resolve_path_zsh() {\n")
	builder.WriteString("  local path token candidate i\n")
	builder.WriteString("  path=\"\"\n")
	builder.WriteString("  for ((i=2; i<CURRENT; i++)); do\n")
	builder.WriteString("    token=\"${words[i]}\"\n")
	builder.WriteString("    if [[ -z \"$token\" || \"$token\" == -* ]]; then\n")
	builder.WriteString("      continue\n")
	builder.WriteString("    fi\n")
	builder.WriteString("    candidate=\"$token\"\n")
	builder.WriteString("    if [[ -n \"$path\" ]]; then\n")
	builder.WriteString("      candidate=\"$path $token\"\n")
	builder.WriteString("    fi\n")
	builder.WriteString("    if _pa_path_exists \"$candidate\"; then\n")
	builder.WriteString("      path=\"$candidate\"\n")
	builder.WriteString("    fi\n")
	builder.WriteString("  done\n")
	builder.WriteString("  print -r -- \"$path\"\n")
	builder.WriteString("}\n")
	builder.WriteString("_personal_agent() {\n")
	builder.WriteString("  local cur path path_subcommands path_flags\n")
	builder.WriteString("  local root_completions global_flags\n")
	builder.WriteString("  local -a suggestions\n")
	builder.WriteString("  cur=\"${words[CURRENT]}\"\n")
	builder.WriteString(fmt.Sprintf("  root_completions=%q\n", strings.Join(rootCompletions, " ")))
	builder.WriteString(fmt.Sprintf("  global_flags=%q\n", strings.Join(globalFlagTokens, " ")))
	builder.WriteString("  if (( CURRENT == 2 )); then\n")
	builder.WriteString("    suggestions=(${=root_completions})\n")
	builder.WriteString("    compadd -- \"${suggestions[@]}\"\n")
	builder.WriteString("    return\n")
	builder.WriteString("  fi\n")
	builder.WriteString("  path=\"$(_pa_resolve_path_zsh)\"\n")
	builder.WriteString("  if [[ -z \"$path\" ]]; then\n")
	builder.WriteString("    suggestions=(${=root_completions})\n")
	builder.WriteString("    compadd -- \"${suggestions[@]}\"\n")
	builder.WriteString("    return\n")
	builder.WriteString("  fi\n")
	builder.WriteString("  path_subcommands=\"$(_pa_subcommands_for_path \"$path\" 2>/dev/null || true)\"\n")
	builder.WriteString("  path_flags=\"$(_pa_flags_for_path \"$path\" 2>/dev/null || true)\"\n")
	builder.WriteString("  if [[ \"$cur\" == -* ]]; then\n")
	builder.WriteString("    suggestions=(${=global_flags} ${=path_flags} --help -h)\n")
	builder.WriteString("    compadd -- \"${suggestions[@]}\"\n")
	builder.WriteString("    return\n")
	builder.WriteString("  fi\n")
	builder.WriteString("  suggestions=(${=path_subcommands} ${=path_flags})\n")
	builder.WriteString("  compadd -- \"${suggestions[@]}\"\n")
	builder.WriteString("}\n")
	builder.WriteString("compdef _personal_agent personal-agent\n")
	return builder.String()
}

func writeBashPathExistsFunction(builder *strings.Builder, knownPaths []string) {
	builder.WriteString("_pa_path_exists() {\n")
	builder.WriteString("  case \"$1\" in\n")
	for _, pathKey := range knownPaths {
		builder.WriteString(fmt.Sprintf("    %q)\n", pathKey))
		builder.WriteString("      return 0\n")
		builder.WriteString("      ;;\n")
	}
	builder.WriteString("  esac\n")
	builder.WriteString("  return 1\n")
	builder.WriteString("}\n")
}

func writeBashLookupFunction(builder *strings.Builder, functionName string, valuesByPath map[string][]string, pathKeys []string) {
	builder.WriteString(functionName + "() {\n")
	builder.WriteString("  case \"$1\" in\n")
	for _, pathKey := range pathKeys {
		values := valuesByPath[pathKey]
		if len(values) == 0 {
			continue
		}
		builder.WriteString(fmt.Sprintf("    %q)\n", pathKey))
		builder.WriteString(fmt.Sprintf("      printf '%%s' %q\n", strings.Join(values, " ")))
		builder.WriteString("      return 0\n")
		builder.WriteString("      ;;\n")
	}
	builder.WriteString("  esac\n")
	builder.WriteString("  return 1\n")
	builder.WriteString("}\n")
}

func writeZshPathExistsFunction(builder *strings.Builder, knownPaths []string) {
	builder.WriteString("_pa_path_exists() {\n")
	builder.WriteString("  case \"$1\" in\n")
	for _, pathKey := range knownPaths {
		builder.WriteString(fmt.Sprintf("    %q)\n", pathKey))
		builder.WriteString("      return 0\n")
		builder.WriteString("      ;;\n")
	}
	builder.WriteString("  esac\n")
	builder.WriteString("  return 1\n")
	builder.WriteString("}\n")
}

func writeZshLookupFunction(builder *strings.Builder, functionName string, valuesByPath map[string][]string, pathKeys []string) {
	builder.WriteString(functionName + "() {\n")
	builder.WriteString("  case \"$1\" in\n")
	for _, pathKey := range pathKeys {
		values := valuesByPath[pathKey]
		if len(values) == 0 {
			continue
		}
		builder.WriteString(fmt.Sprintf("    %q)\n", pathKey))
		builder.WriteString(fmt.Sprintf("      print -r -- %q\n", strings.Join(values, " ")))
		builder.WriteString("      return 0\n")
		builder.WriteString("      ;;\n")
	}
	builder.WriteString("  esac\n")
	builder.WriteString("  return 1\n")
	builder.WriteString("}\n")
}

func writeFishLookupFunction(builder *strings.Builder, functionName string, valuesByPath map[string][]string, pathKeys []string) {
	builder.WriteString("function " + functionName + "\n")
	builder.WriteString("  switch \"$argv[1]\"\n")
	for _, pathKey := range pathKeys {
		values := valuesByPath[pathKey]
		if len(values) == 0 {
			continue
		}
		builder.WriteString(fmt.Sprintf("    case %q\n", pathKey))
		for _, value := range values {
			builder.WriteString(fmt.Sprintf("      echo %q\n", value))
		}
	}
	builder.WriteString("  end\n")
	builder.WriteString("end\n")
}
