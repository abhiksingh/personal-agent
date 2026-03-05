package daemonruntime

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

var twilioCloudflaredTrustedBinaryDirs = []string{
	"/usr/local/bin",
	"/opt/homebrew/bin",
	"/usr/bin",
	"/bin",
}

func resolveTwilioCloudflaredBinaryPath() (string, error) {
	return resolveTwilioCloudflaredBinaryPathWithLookup(exec.LookPath)
}

func resolveTwilioCloudflaredBinaryPathWithLookup(lookup func(string) (string, error)) (string, error) {
	configured := strings.TrimSpace(os.Getenv(twilioCloudflaredBinaryOverrideEnv))
	if configured != "" {
		return validateTwilioCloudflaredBinaryPath(configured, true)
	}
	if lookup == nil {
		return "", fmt.Errorf("cloudflared binary resolver is required")
	}

	resolvedPath, err := lookup(twilioCloudflaredBinaryName)
	if err != nil {
		return "", fmt.Errorf("%w: %v", errTwilioCloudflaredNotInstalled, err)
	}
	return validateTwilioCloudflaredBinaryPath(resolvedPath, false)
}

func validateTwilioCloudflaredBinaryPath(rawPath string, explicitOverride bool) (string, error) {
	candidate := filepath.Clean(strings.TrimSpace(rawPath))
	if candidate == "" {
		if explicitOverride {
			return "", fmt.Errorf("cloudflared binary override is empty")
		}
		return "", fmt.Errorf("resolved cloudflared binary path is empty")
	}
	if !filepath.IsAbs(candidate) {
		if explicitOverride {
			return "", fmt.Errorf("cloudflared binary override must be an absolute path")
		}
		return "", fmt.Errorf("resolved cloudflared binary path must be absolute")
	}

	info, err := os.Lstat(candidate)
	if err != nil {
		return "", fmt.Errorf("cloudflared binary path %q is not accessible: %w", candidate, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("cloudflared binary path %q must not be a symlink", candidate)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("cloudflared binary path %q must point to a regular file", candidate)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o111 == 0 {
		return "", fmt.Errorf("cloudflared binary path %q must be executable", candidate)
	}
	if runtime.GOOS != "windows" && !explicitOverride && !isTwilioTrustedCloudflaredInstallDir(candidate) {
		return "", fmt.Errorf(
			"resolved cloudflared binary path %q is outside trusted install directories (set %s to an explicit absolute path to override)",
			candidate,
			twilioCloudflaredBinaryOverrideEnv,
		)
	}
	return candidate, nil
}

func isTwilioTrustedCloudflaredInstallDir(path string) bool {
	binaryDir := filepath.Clean(filepath.Dir(strings.TrimSpace(path)))
	if binaryDir == "" {
		return false
	}
	for _, trusted := range twilioCloudflaredTrustedBinaryDirs {
		if sameCloudflaredPath(binaryDir, trusted) {
			return true
		}
	}
	return false
}

func sameCloudflaredPath(left string, right string) bool {
	leftAbs, leftErr := filepath.Abs(strings.TrimSpace(left))
	rightAbs, rightErr := filepath.Abs(strings.TrimSpace(right))
	if leftErr == nil && rightErr == nil {
		return filepath.Clean(leftAbs) == filepath.Clean(rightAbs)
	}
	return filepath.Clean(left) == filepath.Clean(right)
}
