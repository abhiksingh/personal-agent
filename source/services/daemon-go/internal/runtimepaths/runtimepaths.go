package runtimepaths

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	// EnvRuntimeRootDir overrides all profile-based runtime path defaults.
	EnvRuntimeRootDir = "PA_RUNTIME_ROOT_DIR"
	// EnvRuntimeProfile selects a non-user runtime profile namespace.
	EnvRuntimeProfile = "PA_RUNTIME_PROFILE"

	defaultRuntimeProfile = "user"
	defaultRuntimeDirName = "personal-agent"
	profileParentDirName  = "personal-agent-profiles"
)

var invalidProfileTokenPattern = regexp.MustCompile(`[^a-z0-9._-]+`)

func NormalizeProfile(raw string) string {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	if trimmed == "" {
		return defaultRuntimeProfile
	}
	sanitized := invalidProfileTokenPattern.ReplaceAllString(trimmed, "-")
	sanitized = strings.Trim(sanitized, ".-_")
	if sanitized == "" {
		return defaultRuntimeProfile
	}
	return sanitized
}

func ResolveRootDir() (string, error) {
	return resolveRootDirWith(os.Getenv, os.UserConfigDir)
}

func DefaultDBPath() (string, error) {
	root, err := ResolveRootDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "runtime.db"), nil
}

func DefaultInboundDir() (string, error) {
	root, err := ResolveRootDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "inbound"), nil
}

func DefaultChannelsDir() (string, error) {
	root, err := ResolveRootDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "channels"), nil
}

func DefaultConnectorsDir() (string, error) {
	root, err := ResolveRootDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "connectors"), nil
}

func resolveRootDirWith(getenv func(string) string, userConfigDir func() (string, error)) (string, error) {
	if getenv == nil {
		getenv = os.Getenv
	}
	if userConfigDir == nil {
		userConfigDir = os.UserConfigDir
	}

	if explicitRoot := strings.TrimSpace(getenv(EnvRuntimeRootDir)); explicitRoot != "" {
		return absPathOrRaw(explicitRoot), nil
	}

	configDir, err := userConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	profile := NormalizeProfile(getenv(EnvRuntimeProfile))
	if profile == defaultRuntimeProfile {
		return filepath.Join(configDir, defaultRuntimeDirName), nil
	}
	return filepath.Join(configDir, profileParentDirName, profile), nil
}

func absPathOrRaw(raw string) string {
	if absolute, err := filepath.Abs(raw); err == nil {
		return absolute
	}
	return raw
}
