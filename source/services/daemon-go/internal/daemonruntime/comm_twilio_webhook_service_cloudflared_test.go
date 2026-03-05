package daemonruntime

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestResolveTwilioCloudflaredBinaryPathWithLookupRejectsUntrustedLookupPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("trusted install directory checks are unix-specific")
	}
	t.Setenv(twilioCloudflaredBinaryOverrideEnv, "")

	binaryPath := writeTwilioExecutableFile(t, "cloudflared-untrusted.sh", "#!/bin/sh\nexit 0\n")
	_, err := resolveTwilioCloudflaredBinaryPathWithLookup(func(string) (string, error) {
		return binaryPath, nil
	})
	if err == nil {
		t.Fatalf("expected untrusted lookup path to be rejected")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "trusted install directories") {
		t.Fatalf("expected trusted-directory rejection, got %v", err)
	}
}

func TestResolveTwilioCloudflaredBinaryPathWithLookupAllowsValidatedOverride(t *testing.T) {
	overridePath := writeTwilioExecutableFile(t, "cloudflared-override.sh", "#!/bin/sh\nexit 0\n")
	t.Setenv(twilioCloudflaredBinaryOverrideEnv, overridePath)

	resolved, err := resolveTwilioCloudflaredBinaryPathWithLookup(func(string) (string, error) {
		return "", nil
	})
	if err != nil {
		t.Fatalf("expected explicit override to pass validation: %v", err)
	}
	if resolved != overridePath {
		t.Fatalf("expected override path %q, got %q", overridePath, resolved)
	}
}

func TestResolveTwilioCloudflaredBinaryPathWithLookupRejectsSymlinkOverride(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink semantics are permission-dependent on windows")
	}

	targetPath := writeTwilioExecutableFile(t, "cloudflared-target.sh", "#!/bin/sh\nexit 0\n")
	symlinkPath := filepath.Join(t.TempDir(), "cloudflared-link")
	if err := os.Symlink(targetPath, symlinkPath); err != nil {
		t.Fatalf("create cloudflared symlink override: %v", err)
	}
	t.Setenv(twilioCloudflaredBinaryOverrideEnv, symlinkPath)

	_, err := resolveTwilioCloudflaredBinaryPathWithLookup(func(string) (string, error) {
		return "", nil
	})
	if err == nil {
		t.Fatalf("expected symlink override to be rejected")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "must not be a symlink") {
		t.Fatalf("expected symlink rejection message, got %v", err)
	}
}

func TestRunTwilioCloudflaredVersionSanityCheckExecutesBinary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test helper script uses unix shell semantics")
	}

	script := "#!/bin/sh\nif [ \"$1\" = \"version\" ]; then\n  echo \"cloudflared version test\"\n  exit 0\nfi\nexit 1\n"
	binaryPath := writeTwilioExecutableFile(t, "cloudflared-version.sh", script)

	if err := runTwilioCloudflaredVersionSanityCheck(context.Background(), binaryPath); err != nil {
		t.Fatalf("expected version sanity check to execute validated binary: %v", err)
	}
}

func writeTwilioExecutableFile(t *testing.T, name string, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o700); err != nil {
		t.Fatalf("write executable test file: %v", err)
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(path, 0o700); err != nil {
			t.Fatalf("chmod executable test file: %v", err)
		}
	}
	return path
}
