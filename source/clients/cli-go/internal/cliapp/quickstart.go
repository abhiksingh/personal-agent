package cliapp

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"personalagent/runtime/internal/controlauth"
	"personalagent/runtime/internal/providerconfig"
	"personalagent/runtime/internal/transport"
)

func runQuickstartCommand(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("quickstart", flag.ContinueOnError)
	flags.SetOutput(stderr)

	workspaceID := flags.String("workspace", defaultLocalDevWorkspaceID, "workspace id")
	profileName := flags.String("profile", defaultLocalDevProfileName, "profile name to create/update")
	listenerModeRaw := flags.String("mode", string(transport.ListenerModeTCP), "transport mode for quickstart profile + daemon target: tcp|unix|named_pipe")
	address := flags.String("address", transport.DefaultTCPAddress, "transport address for quickstart profile + daemon target")
	tokenFile := flags.String("token-file", "", "control auth token file path (defaults under runtime root)")
	tokenBytes := flags.Int("bytes", controlauth.DefaultTokenBytes, "number of random bytes before base64url encoding when creating/rotating quickstart token")
	rotateToken := flags.Bool("rotate-token", false, "rotate token file when it already exists")
	activateProfile := flags.Bool("activate", true, "set quickstart profile as active")
	providerRaw := flags.String("provider", providerconfig.ProviderOpenAI, "provider to configure: openai|anthropic|google|ollama")
	endpoint := flags.String("endpoint", "", "provider endpoint override")
	apiKeySecret := flags.String("api-key-secret", "", "secret name used for provider API key references")
	apiKey := flags.String("api-key", "", "provider API key value (write-only)")
	apiKeyFile := flags.String("api-key-file", "", "path to file containing provider API key value")
	modelKey := flags.String("model", "", "model key for routing policy (defaults by provider)")
	taskClass := flags.String("task-class", "chat", "task class to bind to selected model route")
	skipProviderSetup := flags.Bool("skip-provider-setup", false, "skip provider/secret setup phase")
	skipModelRoute := flags.Bool("skip-model-route", false, "skip model routing policy phase")
	skipDoctor := flags.Bool("skip-doctor", false, "skip readiness doctor execution")
	includeOptional := flags.Bool("include-optional", false, "include optional doctor checks (tooling/local bridge)")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	visited := visitedFlagNames(flags)

	listenerModeNormalized, err := normalizeCLIProfileListenerMode(*listenerModeRaw)
	if err != nil {
		fmt.Fprintf(stderr, "request failed: %v\n", err)
		return 1
	}
	if strings.TrimSpace(listenerModeNormalized) == "" {
		listenerModeNormalized = string(transport.ListenerModeTCP)
	}
	workspace := normalizeWorkspace(*workspaceID)
	providerName := strings.ToLower(strings.TrimSpace(*providerRaw))
	selectedModel := strings.TrimSpace(*modelKey)
	if selectedModel == "" {
		selectedModel = quickstartDefaultModelKey(providerName)
	}
	normalizedTaskClass := normalizeTaskClass(*taskClass)

	report := quickstartReport{
		SchemaVersion: "1.0.0",
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339Nano),
		WorkspaceID:   workspace,
		ProfileName:   strings.TrimSpace(*profileName),
		Defaults: buildOnboardingDefaultsMetadata(
			workspace,
			onboardingSelectionSource(visited["workspace"]),
			strings.TrimSpace(*profileName),
			onboardingSelectionSource(visited["profile"]),
			strings.TrimSpace(*tokenFile),
			onboardingTokenFileSource(visited["token-file"], *tokenFile),
		),
		Provider:      providerName,
		ModelKey:      selectedModel,
		TaskClass:     normalizedTaskClass,
		OverallStatus: quickstartStepStatusPass,
		Summary:       quickstartSummary{},
		Success:       true,
		Steps:         []quickstartStep{},
	}

	resolvedTokenFile := strings.TrimSpace(*tokenFile)
	if resolvedTokenFile == "" {
		defaultPath, resolveErr := resolveDefaultLocalDevTokenFile()
		if resolveErr != nil {
			appendQuickstartStep(&report, quickstartStep{
				ID:      "auth.bootstrap",
				Title:   "Auth/Profile Bootstrap",
				Status:  quickstartStepStatusFail,
				Summary: "Unable to resolve quickstart token file path.",
				Details: map[string]any{
					"error": resolveErr.Error(),
				},
				Remediation: []string{
					"Set --token-file explicitly and rerun quickstart.",
				},
			})
			return finalizeQuickstartAndWrite(stdout, &report)
		}
		resolvedTokenFile = defaultPath
	}
	report.Defaults.TokenFile.Value = resolvedTokenFile

	tokenMaterial, tokenErr := ensureControlAuthTokenMaterial(resolvedTokenFile, *tokenBytes, *rotateToken)
	if tokenErr != nil {
		appendQuickstartStep(&report, quickstartStep{
			ID:      "auth.bootstrap",
			Title:   "Auth/Profile Bootstrap",
			Status:  quickstartStepStatusFail,
			Summary: "Failed to create/load local control auth token material.",
			Details: map[string]any{
				"token_file": resolvedTokenFile,
				"error":      tokenErr.Error(),
			},
			Remediation: []string{
				"Ensure the token-file parent directory is writable, then rerun quickstart.",
			},
		})
		return finalizeQuickstartAndWrite(stdout, &report)
	}

	profileRecord, activeProfile, profilePath, profileErr := upsertLocalDevCLIProfile(localDevProfileOptions{
		Name:          report.ProfileName,
		ListenerMode:  listenerModeNormalized,
		Address:       strings.TrimSpace(*address),
		WorkspaceID:   workspace,
		AuthTokenFile: tokenMaterial.FilePath,
		Activate:      *activateProfile,
	})
	if profileErr != nil {
		appendQuickstartStep(&report, quickstartStep{
			ID:      "auth.bootstrap",
			Title:   "Auth/Profile Bootstrap",
			Status:  quickstartStepStatusFail,
			Summary: "Failed to upsert quickstart CLI profile defaults.",
			Details: map[string]any{
				"profile": report.ProfileName,
				"error":   profileErr.Error(),
			},
			Remediation: []string{
				"Check CLI profile write permissions, then rerun quickstart.",
			},
		})
		return finalizeQuickstartAndWrite(stdout, &report)
	}

	appendQuickstartStep(&report, quickstartStep{
		ID:      "auth.bootstrap",
		Title:   "Auth/Profile Bootstrap",
		Status:  quickstartStepStatusPass,
		Summary: "Control auth token and CLI profile defaults are ready.",
		Details: map[string]any{
			"profile":        profileRecord,
			"active_profile": activeProfile,
			"profile_path":   profilePath,
			"token_file":     tokenMaterial.FilePath,
			"token_created":  tokenMaterial.Created,
			"token_rotated":  tokenMaterial.Rotated,
			"token_sha256":   controlauth.TokenSHA256(tokenMaterial.Token),
		},
	})

	commandHints := quickstartCommandHints{
		WorkspaceID:   workspace,
		ProfileName:   report.ProfileName,
		ListenerMode:  listenerModeNormalized,
		Address:       strings.TrimSpace(*address),
		TokenFilePath: tokenMaterial.FilePath,
		ProfileActive: strings.TrimSpace(activeProfile) == strings.TrimSpace(report.ProfileName),
	}

	client, clientErr := transport.NewClient(transport.ClientConfig{
		ListenerMode: transport.ListenerMode(listenerModeNormalized),
		Address:      strings.TrimSpace(*address),
		AuthToken:    tokenMaterial.Token,
		Timeout:      10 * time.Second,
	})
	if clientErr != nil {
		appendQuickstartStep(&report, quickstartStep{
			ID:      "daemon.connectivity",
			Title:   "Daemon Connectivity",
			Status:  quickstartStepStatusFail,
			Summary: "Failed to construct daemon client from quickstart profile settings.",
			Details: map[string]any{
				"mode":    listenerModeNormalized,
				"address": strings.TrimSpace(*address),
				"error":   clientErr.Error(),
			},
			Remediation: []string{
				"Verify --mode/--address flags and rerun quickstart.",
			},
		})
		return finalizeQuickstartAndWrite(stdout, &report)
	}

	capabilities, capabilitiesErr := client.DaemonCapabilities(ctx, "quickstart.capabilities")
	if capabilitiesErr != nil {
		remediation := []string{
			fmt.Sprintf("Start/restart Personal Agent Daemon with: %s", commandHints.daemonStartCommand()),
		}
		if profileUse := commandHints.profileUseCommand(); profileUse != "" {
			remediation = append(remediation, fmt.Sprintf("Activate quickstart profile with: %s", profileUse))
		}
		remediation = append(remediation, fmt.Sprintf("Confirm daemon health with `%s`.", commandHints.smokeCommand()))
		appendQuickstartStep(&report, quickstartStep{
			ID:      "daemon.connectivity",
			Title:   "Daemon Connectivity",
			Status:  quickstartStepStatusFail,
			Summary: "Daemon connectivity/auth check failed.",
			Details: map[string]any{
				"mode":       listenerModeNormalized,
				"address":    strings.TrimSpace(*address),
				"token_file": tokenMaterial.FilePath,
				"error":      doctorErrorDetails(capabilitiesErr),
			},
			Remediation: remediation,
		})
		return finalizeQuickstartAndWrite(stdout, &report)
	}

	appendQuickstartStep(&report, quickstartStep{
		ID:      "daemon.connectivity",
		Title:   "Daemon Connectivity",
		Status:  quickstartStepStatusPass,
		Summary: "Daemon connectivity/auth check passed.",
		Details: map[string]any{
			"api_version":        capabilities.APIVersion,
			"protocol_modes":     capabilities.ProtocolModes,
			"listener_modes":     capabilities.TransportListenerModes,
			"route_group_count":  len(capabilities.RouteGroups),
			"realtime_supported": len(capabilities.RealtimeEventTypes) > 0,
		},
	})

	if *skipProviderSetup {
		appendQuickstartStep(&report, quickstartStep{
			ID:      "provider.configure",
			Title:   "Provider Configuration",
			Status:  quickstartStepStatusSkipped,
			Summary: "Skipped provider setup by request.",
		})
	} else {
		step := quickstartConfigureProviderStep(ctx, client, quickstartProviderConfigInput{
			WorkspaceID:       workspace,
			Provider:          providerName,
			Endpoint:          strings.TrimSpace(*endpoint),
			APIKeySecretName:  strings.TrimSpace(*apiKeySecret),
			APIKey:            strings.TrimSpace(*apiKey),
			APIKeyFile:        strings.TrimSpace(*apiKeyFile),
			CommandHints:      commandHints,
			CorrelationIDBase: "quickstart.provider",
		})
		appendQuickstartStep(&report, step)
	}

	if *skipModelRoute {
		appendQuickstartStep(&report, quickstartStep{
			ID:      "model.route",
			Title:   "Model Route Selection",
			Status:  quickstartStepStatusSkipped,
			Summary: "Skipped model routing policy setup by request.",
		})
	} else {
		step := quickstartConfigureModelRouteStep(ctx, client, quickstartModelRouteInput{
			WorkspaceID:       workspace,
			Provider:          providerName,
			ModelKey:          selectedModel,
			TaskClass:         normalizedTaskClass,
			CommandHints:      commandHints,
			CorrelationIDBase: "quickstart.model",
		})
		appendQuickstartStep(&report, step)
	}

	if *skipDoctor {
		appendQuickstartStep(&report, quickstartStep{
			ID:      "readiness.doctor",
			Title:   "Readiness Diagnostics",
			Status:  quickstartStepStatusSkipped,
			Summary: "Skipped readiness diagnostics by request.",
		})
	} else {
		appendQuickstartStep(&report, quickstartRunDoctorStep(ctx, client, workspace, *includeOptional))
	}

	return finalizeQuickstartAndWrite(stdout, &report)
}
