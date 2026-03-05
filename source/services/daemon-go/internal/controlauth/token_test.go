package controlauth

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestResolveTokenFromFile(t *testing.T) {
	tokenFile := filepath.Join(t.TempDir(), "control.token")
	if err := os.WriteFile(tokenFile, []byte("file-token\n"), 0o600); err != nil {
		t.Fatalf("write token file: %v", err)
	}

	token, err := ResolveToken("flag-token", tokenFile)
	if err != nil {
		t.Fatalf("resolve token: %v", err)
	}
	if token != "file-token" {
		t.Fatalf("expected file token, got %q", token)
	}
}

func TestGenerateTokenBounds(t *testing.T) {
	if _, err := GenerateToken(MinTokenBytes - 1); err == nil {
		t.Fatalf("expected low-byte-count error")
	}
	if _, err := GenerateToken(MaxTokenBytes + 1); err == nil {
		t.Fatalf("expected high-byte-count error")
	}

	token, err := GenerateToken(DefaultTokenBytes)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	if strings.TrimSpace(token) == "" {
		t.Fatalf("expected non-empty token")
	}
}

func TestWriteTokenFileCreateAndOverwrite(t *testing.T) {
	tokenFile := filepath.Join(t.TempDir(), "control.token")
	if err := WriteTokenFile(tokenFile, "token-one", false); err != nil {
		t.Fatalf("write token file: %v", err)
	}

	data, err := os.ReadFile(tokenFile)
	if err != nil {
		t.Fatalf("read token file: %v", err)
	}
	if strings.TrimSpace(string(data)) != "token-one" {
		t.Fatalf("expected token-one, got %q", string(data))
	}

	if err := WriteTokenFile(tokenFile, "token-two", false); err == nil {
		t.Fatalf("expected duplicate file error")
	}
	if err := WriteTokenFile(tokenFile, "token-two", true); err != nil {
		t.Fatalf("overwrite token file: %v", err)
	}

	data, err = os.ReadFile(tokenFile)
	if err != nil {
		t.Fatalf("read overwritten token file: %v", err)
	}
	if strings.TrimSpace(string(data)) != "token-two" {
		t.Fatalf("expected token-two, got %q", string(data))
	}
}

func TestLoadTokenFileRejectsSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink semantics are permission-dependent on windows")
	}

	root := t.TempDir()
	target := filepath.Join(root, "token-target")
	if err := os.WriteFile(target, []byte("file-token\n"), 0o600); err != nil {
		t.Fatalf("write target token: %v", err)
	}
	linkPath := filepath.Join(root, "token-link")
	if err := os.Symlink(target, linkPath); err != nil {
		t.Fatalf("create token symlink: %v", err)
	}

	_, err := LoadTokenFile(linkPath)
	if err == nil {
		t.Fatalf("expected symlink token file to be rejected")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "must not be a symlink") {
		t.Fatalf("expected symlink rejection message, got %v", err)
	}
}

func TestLoadTokenFileRejectsNonRegularPath(t *testing.T) {
	_, err := LoadTokenFile(t.TempDir())
	if err == nil {
		t.Fatalf("expected directory token path to be rejected")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "regular file") {
		t.Fatalf("expected regular-file rejection message, got %v", err)
	}
}

func TestLoadTokenFileRejectsInsecurePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission mode checks are unix-specific")
	}

	tokenFile := filepath.Join(t.TempDir(), "insecure.token")
	if err := os.WriteFile(tokenFile, []byte("file-token\n"), 0o644); err != nil {
		t.Fatalf("write insecure token file: %v", err)
	}

	_, err := LoadTokenFile(tokenFile)
	if err == nil {
		t.Fatalf("expected insecure-permissions token file to be rejected")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "insecure permissions") {
		t.Fatalf("expected insecure-permissions rejection message, got %v", err)
	}
}

func TestLoadTokenFileRejectsOversizedContent(t *testing.T) {
	tokenFile := filepath.Join(t.TempDir(), "oversized.token")
	oversized := strings.Repeat("a", maxTokenFileBytes+1)
	if err := os.WriteFile(tokenFile, []byte(oversized), 0o600); err != nil {
		t.Fatalf("write oversized token file: %v", err)
	}

	_, err := LoadTokenFile(tokenFile)
	if err == nil {
		t.Fatalf("expected oversized token file to be rejected")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "exceeds max size") {
		t.Fatalf("expected oversized rejection message, got %v", err)
	}
}
