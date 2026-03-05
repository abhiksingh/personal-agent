package filesecurity

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	PrivateDirMode  os.FileMode = 0o700
	PrivateFileMode os.FileMode = 0o600
)

func EnsurePrivateDir(path string) error {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return errors.New("path is required")
	}
	if err := os.MkdirAll(trimmed, PrivateDirMode); err != nil {
		return fmt.Errorf("create directory %q: %w", trimmed, err)
	}
	if shouldSkipDirPermissionTightening(trimmed) {
		return nil
	}
	return enforcePathPermissions(trimmed, PrivateDirMode, true)
}

func EnsurePrivateFile(path string) error {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return errors.New("path is required")
	}
	info, err := os.Lstat(trimmed)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("stat file %q: %w", trimmed, err)
	}
	if info.IsDir() {
		return fmt.Errorf("path %q is a directory, expected file", trimmed)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("path %q is a symlink, expected file", trimmed)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("path %q is not a regular file", trimmed)
	}
	if runtime.GOOS == "windows" {
		return nil
	}
	if info.Mode().Perm() == PrivateFileMode {
		return nil
	}
	if err := os.Chmod(trimmed, PrivateFileMode); err != nil {
		return fmt.Errorf("set file permissions on %q: %w", trimmed, err)
	}
	return nil
}

func enforcePathPermissions(path string, mode os.FileMode, expectDir bool) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat path %q: %w", path, err)
	}
	if expectDir && !info.IsDir() {
		return fmt.Errorf("path %q is not a directory", path)
	}
	if !expectDir && info.IsDir() {
		return fmt.Errorf("path %q is a directory", path)
	}
	if runtime.GOOS == "windows" {
		return nil
	}
	if info.Mode().Perm() == mode {
		return nil
	}
	if err := os.Chmod(path, mode); err != nil {
		return fmt.Errorf("set permissions on %q: %w", path, err)
	}
	return nil
}

func shouldSkipDirPermissionTightening(path string) bool {
	if runtime.GOOS == "windows" {
		return true
	}
	cleaned := filepath.Clean(path)
	if cleaned == "." || cleaned == string(filepath.Separator) {
		return true
	}

	for _, sharedRoot := range sharedRootPaths() {
		if sharedRoot == "" {
			continue
		}
		if samePath(cleaned, sharedRoot) {
			return true
		}
	}
	return false
}

func sharedRootPaths() []string {
	values := []string{}

	if tmp := strings.TrimSpace(os.TempDir()); tmp != "" {
		values = append(values, tmp)
	}
	if home, err := os.UserHomeDir(); err == nil {
		if trimmed := strings.TrimSpace(home); trimmed != "" {
			values = append(values, trimmed)
		}
	}
	if config, err := os.UserConfigDir(); err == nil {
		if trimmed := strings.TrimSpace(config); trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}

func samePath(left string, right string) bool {
	leftAbs, leftErr := filepath.Abs(left)
	rightAbs, rightErr := filepath.Abs(right)
	if leftErr == nil && rightErr == nil {
		return filepath.Clean(leftAbs) == filepath.Clean(rightAbs)
	}
	return filepath.Clean(left) == filepath.Clean(right)
}
