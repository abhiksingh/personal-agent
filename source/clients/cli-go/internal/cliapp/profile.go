package cliapp

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"personalagent/runtime/internal/runtimepaths"
	"personalagent/runtime/internal/transport"
)

const cliProfilesPathEnvKey = "PA_CLI_PROFILES_PATH"

type cliProfileRecord struct {
	Name          string `json:"name"`
	ListenerMode  string `json:"listener_mode,omitempty"`
	Address       string `json:"address,omitempty"`
	WorkspaceID   string `json:"workspace_id,omitempty"`
	AuthTokenFile string `json:"auth_token_file,omitempty"`
}

type cliProfilesState struct {
	ActiveProfile string                      `json:"active_profile,omitempty"`
	Profiles      map[string]cliProfileRecord `json:"profiles"`
}

type cliProfileResponse struct {
	Profile       cliProfileRecord `json:"profile"`
	ActiveProfile string           `json:"active_profile,omitempty"`
	Active        bool             `json:"active"`
	Path          string           `json:"path"`
	PreviousName  string           `json:"previous_name,omitempty"`
}

type cliProfileListResponse struct {
	ActiveProfile string             `json:"active_profile,omitempty"`
	Profiles      []cliProfileRecord `json:"profiles"`
	Path          string             `json:"path"`
}

type cliProfileDeleteResponse struct {
	DeletedProfile       string `json:"deleted_profile"`
	ActiveProfile        string `json:"active_profile,omitempty"`
	ActiveProfileChanged bool   `json:"active_profile_changed"`
	Path                 string `json:"path"`
}

type cliProfileActiveResponse struct {
	ActiveProfile string            `json:"active_profile,omitempty"`
	Profile       *cliProfileRecord `json:"profile,omitempty"`
	ProfileExists bool              `json:"profile_exists"`
	Path          string            `json:"path"`
}

func runProfileCommand(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "profile subcommand required: set|use|get|list|rename|delete|active")
		return 2
	}

	switch args[0] {
	case "set":
		flags := flag.NewFlagSet("profile set", flag.ContinueOnError)
		flags.SetOutput(stderr)

		name := flags.String("name", "", "profile name")
		listenerMode := flags.String("mode", "", "transport mode override: tcp|unix|named_pipe")
		address := flags.String("address", "", "transport address override")
		workspaceID := flags.String("workspace", "", "default workspace id")
		authTokenFile := flags.String("auth-token-file", "", "default auth-token file reference path")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}

		normalizedName := strings.TrimSpace(*name)
		if normalizedName == "" {
			fmt.Fprintln(stderr, "request failed: --name is required")
			return 2
		}

		state, path, err := loadCLIProfilesState()
		if err != nil {
			fmt.Fprintf(stderr, "request failed: %v\n", err)
			return 1
		}
		record := state.Profiles[normalizedName]
		record.Name = normalizedName
		visited := visitedFlagNames(flags)

		if visited["mode"] {
			mode, err := normalizeCLIProfileListenerMode(*listenerMode)
			if err != nil {
				fmt.Fprintf(stderr, "request failed: %v\n", err)
				return 1
			}
			record.ListenerMode = mode
		}
		if visited["address"] {
			record.Address = strings.TrimSpace(*address)
		}
		if visited["workspace"] {
			record.WorkspaceID = strings.TrimSpace(*workspaceID)
		}
		if visited["auth-token-file"] {
			record.AuthTokenFile = normalizeCLIProfilePath(*authTokenFile)
		}

		state.Profiles[normalizedName] = record
		if strings.TrimSpace(state.ActiveProfile) == "" {
			state.ActiveProfile = normalizedName
		}
		if err := saveCLIProfilesState(path, state); err != nil {
			fmt.Fprintf(stderr, "request failed: %v\n", err)
			return 1
		}
		return writeJSON(stdout, cliProfileResponse{
			Profile:       record,
			ActiveProfile: state.ActiveProfile,
			Active:        state.ActiveProfile == normalizedName,
			Path:          path,
		})
	case "use":
		flags := flag.NewFlagSet("profile use", flag.ContinueOnError)
		flags.SetOutput(stderr)

		name := flags.String("name", "", "profile name")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}
		normalizedName := strings.TrimSpace(*name)
		if normalizedName == "" {
			fmt.Fprintln(stderr, "request failed: --name is required")
			return 2
		}

		state, path, err := loadCLIProfilesState()
		if err != nil {
			fmt.Fprintf(stderr, "request failed: %v\n", err)
			return 1
		}
		record, exists := state.Profiles[normalizedName]
		if !exists {
			fmt.Fprintf(stderr, "request failed: profile %q not found\n", normalizedName)
			return 1
		}
		state.ActiveProfile = normalizedName
		if err := saveCLIProfilesState(path, state); err != nil {
			fmt.Fprintf(stderr, "request failed: %v\n", err)
			return 1
		}
		return writeJSON(stdout, cliProfileResponse{
			Profile:       record,
			ActiveProfile: state.ActiveProfile,
			Active:        true,
			Path:          path,
		})
	case "get":
		flags := flag.NewFlagSet("profile get", flag.ContinueOnError)
		flags.SetOutput(stderr)

		name := flags.String("name", "", "profile name (defaults to active profile)")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}

		state, path, err := loadCLIProfilesState()
		if err != nil {
			fmt.Fprintf(stderr, "request failed: %v\n", err)
			return 1
		}
		targetName := strings.TrimSpace(*name)
		if targetName == "" {
			targetName = strings.TrimSpace(state.ActiveProfile)
		}
		if targetName == "" {
			fmt.Fprintln(stderr, "request failed: no active profile is configured")
			return 1
		}
		record, exists := state.Profiles[targetName]
		if !exists {
			fmt.Fprintf(stderr, "request failed: profile %q not found\n", targetName)
			return 1
		}
		return writeJSON(stdout, cliProfileResponse{
			Profile:       record,
			ActiveProfile: state.ActiveProfile,
			Active:        state.ActiveProfile == targetName,
			Path:          path,
		})
	case "list":
		state, path, err := loadCLIProfilesState()
		if err != nil {
			fmt.Fprintf(stderr, "request failed: %v\n", err)
			return 1
		}

		names := make([]string, 0, len(state.Profiles))
		for name := range state.Profiles {
			names = append(names, name)
		}
		sort.Strings(names)
		records := make([]cliProfileRecord, 0, len(names))
		for _, name := range names {
			records = append(records, state.Profiles[name])
		}
		return writeJSON(stdout, cliProfileListResponse{
			ActiveProfile: state.ActiveProfile,
			Profiles:      records,
			Path:          path,
		})
	case "rename":
		flags := flag.NewFlagSet("profile rename", flag.ContinueOnError)
		flags.SetOutput(stderr)

		name := flags.String("name", "", "existing profile name")
		target := flags.String("to", "", "new profile name")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}

		normalizedName := strings.TrimSpace(*name)
		if normalizedName == "" {
			fmt.Fprintln(stderr, "request failed: --name is required")
			return 2
		}
		normalizedTarget := strings.TrimSpace(*target)
		if normalizedTarget == "" {
			fmt.Fprintln(stderr, "request failed: --to is required")
			return 2
		}

		state, path, err := loadCLIProfilesState()
		if err != nil {
			fmt.Fprintf(stderr, "request failed: %v\n", err)
			return 1
		}
		record, exists := state.Profiles[normalizedName]
		if !exists {
			fmt.Fprintf(stderr, "request failed: profile %q not found\n", normalizedName)
			return 1
		}
		if normalizedTarget != normalizedName {
			if _, targetExists := state.Profiles[normalizedTarget]; targetExists {
				fmt.Fprintf(stderr, "request failed: profile %q already exists\n", normalizedTarget)
				return 1
			}
			delete(state.Profiles, normalizedName)
		}
		record.Name = normalizedTarget
		state.Profiles[normalizedTarget] = record
		if strings.TrimSpace(state.ActiveProfile) == normalizedName {
			state.ActiveProfile = normalizedTarget
		}
		if err := saveCLIProfilesState(path, state); err != nil {
			fmt.Fprintf(stderr, "request failed: %v\n", err)
			return 1
		}
		return writeJSON(stdout, cliProfileResponse{
			Profile:       state.Profiles[normalizedTarget],
			ActiveProfile: state.ActiveProfile,
			Active:        state.ActiveProfile == normalizedTarget,
			Path:          path,
			PreviousName:  normalizedName,
		})
	case "delete":
		flags := flag.NewFlagSet("profile delete", flag.ContinueOnError)
		flags.SetOutput(stderr)

		name := flags.String("name", "", "profile name")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}
		normalizedName := strings.TrimSpace(*name)
		if normalizedName == "" {
			fmt.Fprintln(stderr, "request failed: --name is required")
			return 2
		}

		state, path, err := loadCLIProfilesState()
		if err != nil {
			fmt.Fprintf(stderr, "request failed: %v\n", err)
			return 1
		}
		if _, exists := state.Profiles[normalizedName]; !exists {
			fmt.Fprintf(stderr, "request failed: profile %q not found\n", normalizedName)
			return 1
		}
		delete(state.Profiles, normalizedName)
		activeProfileChanged := false
		if strings.TrimSpace(state.ActiveProfile) == normalizedName {
			activeProfileChanged = true
			state.ActiveProfile = selectNextCLIProfileName(state.Profiles)
		}
		if err := saveCLIProfilesState(path, state); err != nil {
			fmt.Fprintf(stderr, "request failed: %v\n", err)
			return 1
		}
		return writeJSON(stdout, cliProfileDeleteResponse{
			DeletedProfile:       normalizedName,
			ActiveProfile:        state.ActiveProfile,
			ActiveProfileChanged: activeProfileChanged,
			Path:                 path,
		})
	case "active":
		state, path, err := loadCLIProfilesState()
		if err != nil {
			fmt.Fprintf(stderr, "request failed: %v\n", err)
			return 1
		}
		activeName := strings.TrimSpace(state.ActiveProfile)
		response := cliProfileActiveResponse{
			ActiveProfile: activeName,
			ProfileExists: false,
			Path:          path,
		}
		if activeName != "" {
			if record, exists := state.Profiles[activeName]; exists {
				profile := record
				response.Profile = &profile
				response.ProfileExists = true
			}
		}
		return writeJSON(stdout, response)
	default:
		writeUnknownSubcommandError(stderr, "profile subcommand", args[0])
		return 2
	}
}

func applyActiveCLIProfileDefaults(explicitFlags map[string]bool, listenerMode *string, address *string, authToken *string, authTokenFile *string) error {
	state, _, err := loadCLIProfilesState()
	if err != nil {
		return err
	}
	activeName := strings.TrimSpace(state.ActiveProfile)
	if activeName == "" {
		return nil
	}
	record, exists := state.Profiles[activeName]
	if !exists {
		return nil
	}

	if !isCLIFlagExplicit(explicitFlags, "mode") && strings.TrimSpace(record.ListenerMode) != "" && listenerMode != nil {
		*listenerMode = strings.TrimSpace(record.ListenerMode)
	}
	if !isCLIFlagExplicit(explicitFlags, "address") && strings.TrimSpace(record.Address) != "" && address != nil {
		*address = strings.TrimSpace(record.Address)
	}
	if !isCLIFlagExplicit(explicitFlags, "auth-token") && !isCLIFlagExplicit(explicitFlags, "auth-token-file") &&
		strings.TrimSpace(record.AuthTokenFile) != "" && authTokenFile != nil {
		*authTokenFile = strings.TrimSpace(record.AuthTokenFile)
	}
	if authToken != nil && isCLIFlagExplicit(explicitFlags, "auth-token-file") {
		*authToken = strings.TrimSpace(*authToken)
	}
	if strings.TrimSpace(os.Getenv(cliWorkspaceEnvKey)) == "" && strings.TrimSpace(record.WorkspaceID) != "" {
		_ = os.Setenv(cliWorkspaceEnvKey, strings.TrimSpace(record.WorkspaceID))
	}
	return nil
}

func loadCLIProfilesState() (cliProfilesState, string, error) {
	path, err := resolveCLIProfilesPath()
	if err != nil {
		return cliProfilesState{}, "", err
	}
	state := cliProfilesState{
		Profiles: map[string]cliProfileRecord{},
	}

	bytes, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return state, path, nil
		}
		return cliProfilesState{}, "", fmt.Errorf("read cli profile state %q: %w", path, err)
	}
	if err := json.Unmarshal(bytes, &state); err != nil {
		return cliProfilesState{}, "", fmt.Errorf("decode cli profile state %q: %w", path, err)
	}
	if state.Profiles == nil {
		state.Profiles = map[string]cliProfileRecord{}
	}
	for name, record := range state.Profiles {
		record.Name = strings.TrimSpace(record.Name)
		if record.Name == "" {
			record.Name = name
		}
		if mode := strings.TrimSpace(record.ListenerMode); mode != "" {
			normalizedMode, normalizeErr := normalizeCLIProfileListenerMode(mode)
			if normalizeErr != nil {
				return cliProfilesState{}, "", fmt.Errorf("invalid listener mode in profile %q: %w", name, normalizeErr)
			}
			record.ListenerMode = normalizedMode
		}
		record.WorkspaceID = strings.TrimSpace(record.WorkspaceID)
		record.Address = strings.TrimSpace(record.Address)
		record.AuthTokenFile = normalizeCLIProfilePath(record.AuthTokenFile)
		state.Profiles[name] = record
	}
	return state, path, nil
}

func saveCLIProfilesState(path string, state cliProfilesState) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("profile path is required")
	}
	if state.Profiles == nil {
		state.Profiles = map[string]cliProfileRecord{}
	}
	for name, record := range state.Profiles {
		record.Name = firstNonEmpty(strings.TrimSpace(record.Name), strings.TrimSpace(name))
		record.WorkspaceID = strings.TrimSpace(record.WorkspaceID)
		record.Address = strings.TrimSpace(record.Address)
		record.AuthTokenFile = normalizeCLIProfilePath(record.AuthTokenFile)
		if mode := strings.TrimSpace(record.ListenerMode); mode != "" {
			normalizedMode, err := normalizeCLIProfileListenerMode(mode)
			if err != nil {
				return fmt.Errorf("normalize profile %q listener mode: %w", name, err)
			}
			record.ListenerMode = normalizedMode
		} else {
			record.ListenerMode = ""
		}
		state.Profiles[name] = record
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create cli profile directory %q: %w", dir, err)
	}
	payload, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode cli profile state: %w", err)
	}
	if err := os.WriteFile(path, append(payload, '\n'), 0o600); err != nil {
		return fmt.Errorf("write cli profile state %q: %w", path, err)
	}
	return nil
}

func resolveCLIProfilesPath() (string, error) {
	if explicit := normalizeCLIProfilePath(os.Getenv(cliProfilesPathEnvKey)); explicit != "" {
		return explicit, nil
	}
	root, err := runtimepaths.ResolveRootDir()
	if err != nil {
		return "", fmt.Errorf("resolve runtime root for profiles: %w", err)
	}
	return filepath.Join(root, "cli", "profiles.json"), nil
}

func normalizeCLIProfilePath(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	absolute, err := filepath.Abs(trimmed)
	if err != nil {
		return trimmed
	}
	return absolute
}

func selectNextCLIProfileName(profiles map[string]cliProfileRecord) string {
	if len(profiles) == 0 {
		return ""
	}
	names := make([]string, 0, len(profiles))
	for name := range profiles {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			continue
		}
		names = append(names, trimmed)
	}
	if len(names) == 0 {
		return ""
	}
	sort.Strings(names)
	return names[0]
}

func normalizeCLIProfileListenerMode(raw string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if normalized == "" {
		return "", nil
	}
	switch normalized {
	case string(transport.ListenerModeTCP), string(transport.ListenerModeUnix), string(transport.ListenerModeNamedPipe):
		return normalized, nil
	default:
		return "", fmt.Errorf("unsupported listener mode %q", raw)
	}
}

func visitedFlagNames(flags *flag.FlagSet) map[string]bool {
	visited := map[string]bool{}
	if flags == nil {
		return visited
	}
	flags.Visit(func(item *flag.Flag) {
		visited[item.Name] = true
	})
	return visited
}

func isCLIFlagExplicit(explicitFlags map[string]bool, name string) bool {
	if len(explicitFlags) == 0 {
		return false
	}
	return explicitFlags[strings.TrimSpace(name)]
}
