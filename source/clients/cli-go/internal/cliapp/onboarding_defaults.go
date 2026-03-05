package cliapp

import (
	"fmt"
	"strings"
)

type onboardingSelectionMetadata struct {
	Value        string `json:"value"`
	Source       string `json:"source"`
	OverrideFlag string `json:"override_flag"`
}

type onboardingDefaultsMetadata struct {
	Workspace     onboardingSelectionMetadata `json:"workspace"`
	Profile       onboardingSelectionMetadata `json:"profile"`
	TokenFile     onboardingSelectionMetadata `json:"token_file"`
	OverrideHints []string                    `json:"override_hints"`
}

func onboardingSelectionSource(explicit bool) string {
	if explicit {
		return "explicit"
	}
	return "default"
}

func onboardingTokenFileSource(explicitTokenFlag bool, rawTokenPath string) string {
	if explicitTokenFlag && strings.TrimSpace(rawTokenPath) != "" {
		return "explicit"
	}
	return "default"
}

func buildOnboardingDefaultsMetadata(workspaceValue, workspaceSource, profileValue, profileSource, tokenFileValue, tokenFileSource string) onboardingDefaultsMetadata {
	trimmedWorkspace := strings.TrimSpace(workspaceValue)
	trimmedProfile := strings.TrimSpace(profileValue)
	trimmedTokenFile := strings.TrimSpace(tokenFileValue)
	return onboardingDefaultsMetadata{
		Workspace: onboardingSelectionMetadata{
			Value:        trimmedWorkspace,
			Source:       strings.TrimSpace(workspaceSource),
			OverrideFlag: "--workspace",
		},
		Profile: onboardingSelectionMetadata{
			Value:        trimmedProfile,
			Source:       strings.TrimSpace(profileSource),
			OverrideFlag: "--profile",
		},
		TokenFile: onboardingSelectionMetadata{
			Value:        trimmedTokenFile,
			Source:       strings.TrimSpace(tokenFileSource),
			OverrideFlag: "--token-file",
		},
		OverrideHints: []string{
			fmt.Sprintf("Override workspace selection with `--workspace <workspace-id>` (current: %s).", onboardingHintValue(trimmedWorkspace)),
			fmt.Sprintf("Override profile selection with `--profile <profile-name>` (current: %s).", onboardingHintValue(trimmedProfile)),
			fmt.Sprintf("Override token file path with `--token-file <path>` (current: %s).", onboardingHintValue(trimmedTokenFile)),
		},
	}
}

func onboardingHintValue(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "<unset>"
	}
	return trimmed
}
