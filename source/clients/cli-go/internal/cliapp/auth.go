package cliapp

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"personalagent/runtime/internal/controlauth"
	"personalagent/runtime/internal/runtimepaths"
	"personalagent/runtime/internal/transport"
)

type controlAuthCommandResponse struct {
	Operation        string   `json:"operation"`
	TokenFile        string   `json:"token_file"`
	TokenLength      int      `json:"token_length"`
	TokenSHA256      string   `json:"token_sha256"`
	Created          bool     `json:"created"`
	Rotated          bool     `json:"rotated"`
	RestartRequired  bool     `json:"restart_required"`
	DaemonArgs       []string `json:"daemon_args"`
	CLIArgs          []string `json:"cli_args"`
	NextStepReminder string   `json:"next_step_reminder"`
}

type localDevAuthBootstrapResponse struct {
	Operation        string                     `json:"operation"`
	TokenFile        string                     `json:"token_file"`
	TokenLength      int                        `json:"token_length"`
	TokenSHA256      string                     `json:"token_sha256"`
	TokenCreated     bool                       `json:"token_created"`
	TokenRotated     bool                       `json:"token_rotated"`
	Profile          cliProfileRecord           `json:"profile"`
	ActiveProfile    string                     `json:"active_profile"`
	Defaults         onboardingDefaultsMetadata `json:"defaults"`
	ProfilePath      string                     `json:"profile_path"`
	DaemonArgs       []string                   `json:"daemon_args"`
	CLIArgs          []string                   `json:"cli_args"`
	NextStepReminder string                     `json:"next_step_reminder"`
}

type controlAuthTokenMaterial struct {
	FilePath string
	Token    string
	Created  bool
	Rotated  bool
}

const (
	defaultLocalDevProfileName = "local-daemon"
	defaultLocalDevWorkspaceID = "ws1"
	defaultLocalDevTokenName   = "local-dev.control.token"
)

func runAuthCommand(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "auth subcommand required: bootstrap|rotate|bootstrap-local-dev")
		return 2
	}

	switch args[0] {
	case "bootstrap":
		flags := flag.NewFlagSet("auth bootstrap", flag.ContinueOnError)
		flags.SetOutput(stderr)
		filePath := flags.String("file", "", "path to write daemon/cli control auth token")
		byteCount := flags.Int("bytes", controlauth.DefaultTokenBytes, "number of random bytes before base64url encoding")
		force := flags.Bool("force", false, "overwrite token file if it already exists")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}
		return writeControlAuthToken(*filePath, *byteCount, *force, false, stdout, stderr)
	case "rotate":
		flags := flag.NewFlagSet("auth rotate", flag.ContinueOnError)
		flags.SetOutput(stderr)
		filePath := flags.String("file", "", "path to existing daemon/cli control auth token file")
		byteCount := flags.Int("bytes", controlauth.DefaultTokenBytes, "number of random bytes before base64url encoding")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}
		trimmedPath := strings.TrimSpace(*filePath)
		if trimmedPath == "" {
			fmt.Fprintln(stderr, "request failed: --file is required")
			return 2
		}
		if _, err := os.Stat(trimmedPath); err != nil {
			fmt.Fprintf(stderr, "request failed: token file does not exist: %s\n", trimmedPath)
			return 1
		}
		return writeControlAuthToken(trimmedPath, *byteCount, true, true, stdout, stderr)
	case "bootstrap-local-dev":
		flags := flag.NewFlagSet("auth bootstrap-local-dev", flag.ContinueOnError)
		flags.SetOutput(stderr)
		profileName := flags.String("profile", defaultLocalDevProfileName, "cli profile name to create/update")
		listenerMode := flags.String("mode", string(transport.ListenerModeTCP), "transport mode for the local-dev profile: tcp|unix|named_pipe")
		address := flags.String("address", transport.DefaultTCPAddress, "transport address for the local-dev profile")
		workspaceID := flags.String("workspace", defaultLocalDevWorkspaceID, "workspace id for the local-dev profile")
		tokenFile := flags.String("token-file", "", "control auth token file path (defaults under runtime root)")
		byteCount := flags.Int("bytes", controlauth.DefaultTokenBytes, "number of random bytes before base64url encoding when generating/rotating token material")
		rotateToken := flags.Bool("rotate-token", false, "rotate token file when it already exists")
		activate := flags.Bool("activate", true, "set the profile as active after bootstrap")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}
		visited := visitedFlagNames(flags)
		profileSelectionSource := onboardingSelectionSource(visited["profile"])
		workspaceSelectionSource := onboardingSelectionSource(visited["workspace"])
		tokenFileSelectionSource := onboardingTokenFileSource(visited["token-file"], *tokenFile)

		resolvedTokenFile := strings.TrimSpace(*tokenFile)
		if resolvedTokenFile == "" {
			defaultPath, err := resolveDefaultLocalDevTokenFile()
			if err != nil {
				fmt.Fprintf(stderr, "request failed: %v\n", err)
				return 1
			}
			resolvedTokenFile = defaultPath
		}

		tokenMaterial, err := ensureControlAuthTokenMaterial(resolvedTokenFile, *byteCount, *rotateToken)
		if err != nil {
			fmt.Fprintf(stderr, "request failed: %v\n", err)
			return 1
		}

		profileRecord, activeProfile, profilePath, err := upsertLocalDevCLIProfile(localDevProfileOptions{
			Name:          *profileName,
			ListenerMode:  *listenerMode,
			Address:       *address,
			WorkspaceID:   *workspaceID,
			AuthTokenFile: tokenMaterial.FilePath,
			Activate:      *activate,
		})
		if err != nil {
			fmt.Fprintf(stderr, "request failed: %v\n", err)
			return 1
		}

		response := localDevAuthBootstrapResponse{
			Operation:     "bootstrap_local_dev",
			TokenFile:     tokenMaterial.FilePath,
			TokenLength:   len(tokenMaterial.Token),
			TokenSHA256:   controlauth.TokenSHA256(tokenMaterial.Token),
			TokenCreated:  tokenMaterial.Created,
			TokenRotated:  tokenMaterial.Rotated,
			Profile:       profileRecord,
			ActiveProfile: activeProfile,
			Defaults: buildOnboardingDefaultsMetadata(
				profileRecord.WorkspaceID,
				workspaceSelectionSource,
				profileRecord.Name,
				profileSelectionSource,
				tokenMaterial.FilePath,
				tokenFileSelectionSource,
			),
			ProfilePath:      profilePath,
			DaemonArgs:       []string{"--auth-token-file", tokenMaterial.FilePath},
			CLIArgs:          []string{"--auth-token-file", tokenMaterial.FilePath},
			NextStepReminder: "Start/restart daemon with --auth-token-file and use the configured CLI profile for daemon-backed commands.",
		}
		return writeJSON(stdout, response)
	default:
		writeUnknownSubcommandError(stderr, "auth subcommand", args[0])
		return 2
	}
}

func writeControlAuthToken(
	filePath string,
	byteCount int,
	overwrite bool,
	rotated bool,
	stdout io.Writer,
	stderr io.Writer,
) int {
	token, err := controlauth.GenerateToken(byteCount)
	if err != nil {
		fmt.Fprintf(stderr, "request failed: %v\n", err)
		return 1
	}

	if err := controlauth.WriteTokenFile(filePath, token, overwrite); err != nil {
		fmt.Fprintf(stderr, "request failed: %v\n", err)
		return 1
	}

	trimmedPath := strings.TrimSpace(filePath)
	response := controlAuthCommandResponse{
		Operation:        "bootstrap",
		TokenFile:        trimmedPath,
		TokenLength:      len(token),
		TokenSHA256:      controlauth.TokenSHA256(token),
		Created:          !rotated,
		Rotated:          rotated,
		RestartRequired:  true,
		DaemonArgs:       []string{"--auth-token-file", trimmedPath},
		CLIArgs:          []string{"--auth-token-file", trimmedPath},
		NextStepReminder: "Restart daemon and clients to pick up the updated auth token file.",
	}
	if rotated {
		response.Operation = "rotate"
	}
	return writeJSON(stdout, response)
}

func resolveDefaultLocalDevTokenFile() (string, error) {
	root, err := runtimepaths.ResolveRootDir()
	if err != nil {
		return "", fmt.Errorf("resolve runtime root for local-dev auth token: %w", err)
	}
	return filepath.Join(root, "control", defaultLocalDevTokenName), nil
}

func ensureControlAuthTokenMaterial(filePath string, byteCount int, rotate bool) (controlAuthTokenMaterial, error) {
	resolvedPath := normalizeCLIProfilePath(filePath)
	if strings.TrimSpace(resolvedPath) == "" {
		return controlAuthTokenMaterial{}, fmt.Errorf("--token-file is required")
	}

	if _, err := os.Stat(resolvedPath); err == nil {
		if !rotate {
			token, loadErr := controlauth.LoadTokenFile(resolvedPath)
			if loadErr != nil {
				return controlAuthTokenMaterial{}, loadErr
			}
			return controlAuthTokenMaterial{
				FilePath: resolvedPath,
				Token:    token,
			}, nil
		}
		token, err := controlauth.GenerateToken(byteCount)
		if err != nil {
			return controlAuthTokenMaterial{}, err
		}
		if err := controlauth.WriteTokenFile(resolvedPath, token, true); err != nil {
			return controlAuthTokenMaterial{}, err
		}
		return controlAuthTokenMaterial{
			FilePath: resolvedPath,
			Token:    token,
			Rotated:  true,
		}, nil
	} else if !os.IsNotExist(err) {
		return controlAuthTokenMaterial{}, fmt.Errorf("stat token file %q: %w", resolvedPath, err)
	}

	token, err := controlauth.GenerateToken(byteCount)
	if err != nil {
		return controlAuthTokenMaterial{}, err
	}
	if err := controlauth.WriteTokenFile(resolvedPath, token, false); err != nil {
		return controlAuthTokenMaterial{}, err
	}
	return controlAuthTokenMaterial{
		FilePath: resolvedPath,
		Token:    token,
		Created:  true,
	}, nil
}

type localDevProfileOptions struct {
	Name          string
	ListenerMode  string
	Address       string
	WorkspaceID   string
	AuthTokenFile string
	Activate      bool
}

func upsertLocalDevCLIProfile(options localDevProfileOptions) (cliProfileRecord, string, string, error) {
	name := strings.TrimSpace(options.Name)
	if name == "" {
		return cliProfileRecord{}, "", "", fmt.Errorf("--profile is required")
	}
	listenerMode, err := normalizeCLIProfileListenerMode(options.ListenerMode)
	if err != nil {
		return cliProfileRecord{}, "", "", err
	}

	state, path, err := loadCLIProfilesState()
	if err != nil {
		return cliProfileRecord{}, "", "", err
	}

	record := state.Profiles[name]
	record.Name = name
	record.ListenerMode = listenerMode
	record.Address = strings.TrimSpace(options.Address)
	record.WorkspaceID = strings.TrimSpace(options.WorkspaceID)
	record.AuthTokenFile = normalizeCLIProfilePath(options.AuthTokenFile)
	state.Profiles[name] = record

	if options.Activate || strings.TrimSpace(state.ActiveProfile) == "" {
		state.ActiveProfile = name
	}
	if err := saveCLIProfilesState(path, state); err != nil {
		return cliProfileRecord{}, "", "", err
	}

	return record, strings.TrimSpace(state.ActiveProfile), path, nil
}
