package controlauth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	// DefaultTokenBytes yields a 43-char URL-safe token when base64url encoded.
	DefaultTokenBytes = 32
	MinTokenBytes     = 16
	MaxTokenBytes     = 128
	maxTokenFileBytes = 4096
)

func ResolveToken(flagToken string, tokenFile string) (string, error) {
	path := strings.TrimSpace(tokenFile)
	if path != "" {
		return LoadTokenFile(path)
	}
	return strings.TrimSpace(flagToken), nil
}

func LoadTokenFile(path string) (string, error) {
	trimmedPath := strings.TrimSpace(path)
	if trimmedPath == "" {
		return "", fmt.Errorf("--auth-token-file is required")
	}

	info, err := os.Lstat(trimmedPath)
	if err != nil {
		return "", fmt.Errorf("stat --auth-token-file %q: %w", trimmedPath, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("--auth-token-file %q must not be a symlink", trimmedPath)
	}
	if info.IsDir() || !info.Mode().IsRegular() {
		return "", fmt.Errorf("--auth-token-file %q must reference a regular file", trimmedPath)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		return "", fmt.Errorf(
			"--auth-token-file %q has insecure permissions %o (expected no group/other bits)",
			trimmedPath,
			info.Mode().Perm(),
		)
	}
	if info.Size() > maxTokenFileBytes {
		return "", fmt.Errorf("--auth-token-file %q exceeds max size of %d bytes", trimmedPath, maxTokenFileBytes)
	}

	file, err := os.Open(trimmedPath)
	if err != nil {
		return "", fmt.Errorf("open --auth-token-file %q: %w", trimmedPath, err)
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxTokenFileBytes+1))
	if err != nil {
		return "", fmt.Errorf("read --auth-token-file %q: %w", trimmedPath, err)
	}
	if len(data) > maxTokenFileBytes {
		return "", fmt.Errorf("--auth-token-file %q exceeds max size of %d bytes", trimmedPath, maxTokenFileBytes)
	}

	token := strings.TrimSpace(string(data))
	if token == "" {
		return "", fmt.Errorf("--auth-token-file %q is empty", trimmedPath)
	}
	return token, nil
}

func GenerateToken(byteCount int) (string, error) {
	if byteCount < MinTokenBytes || byteCount > MaxTokenBytes {
		return "", fmt.Errorf("token bytes must be between %d and %d", MinTokenBytes, MaxTokenBytes)
	}

	secret := make([]byte, byteCount)
	if _, err := rand.Read(secret); err != nil {
		return "", fmt.Errorf("generate auth token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(secret), nil
}

func WriteTokenFile(path string, token string, overwrite bool) error {
	trimmedPath := strings.TrimSpace(path)
	trimmedToken := strings.TrimSpace(token)
	if trimmedPath == "" {
		return fmt.Errorf("--file is required")
	}
	if trimmedToken == "" {
		return fmt.Errorf("token value is required")
	}

	if !overwrite {
		if _, err := os.Stat(trimmedPath); err == nil {
			return fmt.Errorf("token file already exists: %s", trimmedPath)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("stat token file %q: %w", trimmedPath, err)
		}
	}

	dir := filepath.Dir(trimmedPath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create token file directory %q: %w", dir, err)
	}

	tmpFile, err := os.CreateTemp(dir, ".pa-auth-token-*")
	if err != nil {
		return fmt.Errorf("create token temp file: %w", err)
	}
	tmpName := tmpFile.Name()
	defer func() {
		_ = os.Remove(tmpName)
	}()

	if err := tmpFile.Chmod(0o600); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("set token temp file permissions: %w", err)
	}
	if _, err := tmpFile.WriteString(trimmedToken + "\n"); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("write token temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close token temp file: %w", err)
	}

	if err := os.Rename(tmpName, trimmedPath); err != nil {
		return fmt.Errorf("replace token file %q: %w", trimmedPath, err)
	}
	if err := os.Chmod(trimmedPath, 0o600); err != nil {
		return fmt.Errorf("set token file permissions: %w", err)
	}
	return nil
}

func TokenSHA256(token string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(sum[:])
}
