package cliapp

const (
	completionShellBash = "bash"
	completionShellFish = "fish"
	completionShellZsh  = "zsh"
)

type completionCommandIndex struct {
	rootCommands      []string
	globalFlags       []string
	knownPaths        []string
	subcommandsByPath map[string][]string
	flagsByPath       map[string][]string
}
