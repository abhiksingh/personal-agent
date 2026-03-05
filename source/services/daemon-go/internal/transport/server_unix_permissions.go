package transport

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	unixSocketParentDirMode = os.FileMode(0o700)
	unixSocketFileMode      = os.FileMode(0o600)
)

func ensurePrivateUnixSocketParentDir(address string) error {
	parentDir := filepath.Dir(address)
	if err := os.MkdirAll(parentDir, unixSocketParentDirMode); err != nil {
		return err
	}
	if runtime.GOOS == "windows" || shouldSkipUnixSocketPermissionTightening(parentDir) {
		return nil
	}
	if err := os.Chmod(parentDir, unixSocketParentDirMode); err != nil {
		return err
	}
	return nil
}

func enforceUnixSocketFileMode(path string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	return os.Chmod(path, unixSocketFileMode)
}

func shouldSkipUnixSocketPermissionTightening(path string) bool {
	cleaned := filepath.Clean(path)
	if cleaned == "." || cleaned == string(filepath.Separator) {
		return true
	}
	for _, sharedRoot := range unixSocketSharedRoots() {
		if sharedRoot == "" {
			continue
		}
		if sameFilePath(cleaned, sharedRoot) {
			return true
		}
	}
	return false
}

func unixSocketSharedRoots() []string {
	values := []string{}
	appendRoot := func(candidate string) {
		trimmed := strings.TrimSpace(candidate)
		if trimmed != "" {
			values = append(values, trimmed)
		}
	}
	if tmp := strings.TrimSpace(os.TempDir()); tmp != "" {
		appendRoot(tmp)
	}
	if home, err := os.UserHomeDir(); err == nil {
		appendRoot(home)
	}
	if configDir, err := os.UserConfigDir(); err == nil {
		appendRoot(configDir)
	}
	appendRoot("/tmp")
	appendRoot("/private/tmp")
	appendRoot("/var/tmp")
	appendRoot("/private/var/tmp")
	return dedupeCleanPaths(values)
}

func sameFilePath(left string, right string) bool {
	leftResolved := resolveComparablePath(left)
	rightResolved := resolveComparablePath(right)
	if leftResolved != "" && rightResolved != "" {
		return leftResolved == rightResolved
	}
	return filepath.Clean(left) == filepath.Clean(right)
}

func resolveComparablePath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}
	absolute, err := filepath.Abs(trimmed)
	if err == nil {
		trimmed = absolute
	}
	cleaned := filepath.Clean(trimmed)
	if resolved, resolveErr := filepath.EvalSymlinks(cleaned); resolveErr == nil {
		return filepath.Clean(resolved)
	}
	return cleaned
}

func dedupeCleanPaths(paths []string) []string {
	unique := make([]string, 0, len(paths))
	seen := map[string]struct{}{}
	for _, raw := range paths {
		resolved := resolveComparablePath(raw)
		if resolved == "" {
			continue
		}
		if _, exists := seen[resolved]; exists {
			continue
		}
		seen[resolved] = struct{}{}
		unique = append(unique, resolved)
	}
	return unique
}
