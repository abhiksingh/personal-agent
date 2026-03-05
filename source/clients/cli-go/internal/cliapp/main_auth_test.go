package cliapp

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"personalagent/runtime/internal/runtimepaths"
)

func TestNormalizeCLIRuntimeProfile(t *testing.T) {
	value, err := normalizeCLIRuntimeProfile("")
	if err != nil {
		t.Fatalf("normalize empty runtime profile: %v", err)
	}
	if value != cliRuntimeProfileLocal {
		t.Fatalf("expected local default, got %q", value)
	}

	value, err = normalizeCLIRuntimeProfile("PrOd")
	if err != nil {
		t.Fatalf("normalize prod runtime profile: %v", err)
	}
	if value != cliRuntimeProfileProd {
		t.Fatalf("expected prod profile, got %q", value)
	}

	if _, err := normalizeCLIRuntimeProfile("PrOdUcTiOn"); err == nil {
		t.Fatalf("expected unsupported runtime profile for legacy production alias")
	}

	if _, err := normalizeCLIRuntimeProfile("staging"); err == nil {
		t.Fatalf("expected invalid runtime profile error")
	}
}

func TestValidateCLIAuthTokenByRuntimeProfile(t *testing.T) {
	if err := validateCLIAuthTokenByRuntimeProfile(cliRuntimeProfileProd, "", ""); err == nil {
		t.Fatalf("expected prod empty token rejection")
	}

	if err := validateCLIAuthTokenByRuntimeProfile(cliRuntimeProfileLocal, "", ""); err == nil {
		t.Fatalf("expected local empty token rejection")
	}

	if err := validateCLIAuthTokenByRuntimeProfile(cliRuntimeProfileProd, "prod-token-value", ""); err == nil {
		t.Fatalf("expected prod token-file requirement rejection")
	}

	if err := validateCLIAuthTokenByRuntimeProfile(cliRuntimeProfileProd, "short-token", "/tmp/token"); err == nil {
		t.Fatalf("expected prod token length rejection")
	}

	if err := validateCLIAuthTokenByRuntimeProfile(cliRuntimeProfileProd, "prod-token-value-0123456789012345", "/tmp/token"); err != nil {
		t.Fatalf("expected prod token with file to pass: %v", err)
	}

	if err := validateCLIAuthTokenByRuntimeProfile(cliRuntimeProfileLocal, "local-token", ""); err != nil {
		t.Fatalf("expected local token to pass: %v", err)
	}
}

func TestRunRejectsMissingAuthToken(t *testing.T) {
	t.Setenv(cliProfilesPathEnvKey, filepath.Join(t.TempDir(), "profiles.json"))

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := run([]string{"unknown"}, stdout, stderr)
	if exitCode != 2 {
		t.Fatalf("expected exit code 2, got %d", exitCode)
	}

	output := stderr.String()
	if !strings.Contains(output, "invalid CLI configuration") || !strings.Contains(output, "--auth-token is required") {
		t.Fatalf("expected production auth-token validation error, got: %s", output)
	}
	if strings.Contains(output, "unknown command") {
		t.Fatalf("expected auth validation before command dispatch, got: %s", output)
	}
}

func TestRunRejectsProdProfileWithoutAuthTokenFile(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := run([]string{"--runtime-profile", "prod", "--auth-token", "example-token", "unknown"}, stdout, stderr)
	if exitCode != 2 {
		t.Fatalf("expected exit code 2, got %d", exitCode)
	}

	output := stderr.String()
	if !strings.Contains(output, "invalid CLI configuration") || !strings.Contains(output, "--auth-token-file is required") {
		t.Fatalf("expected prod auth-token-file validation failure, got: %s", output)
	}
}

func TestRunRejectsUnsupportedOutputModes(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := run([]string{"--output", "yaml", "smoke"}, stdout, stderr)
	if exitCode != 2 {
		t.Fatalf("expected exit code 2 for unsupported output mode, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "unsupported --output") {
		t.Fatalf("expected unsupported --output error, got %s", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	exitCode = run([]string{"--error-output", "yaml", "smoke"}, stdout, stderr)
	if exitCode != 2 {
		t.Fatalf("expected exit code 2 for unsupported error-output mode, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "unsupported --error-output") {
		t.Fatalf("expected unsupported --error-output error, got %s", stderr.String())
	}
}

func TestRunAcceptsTextOutputMode(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := run([]string{"--output", "text", "--auth-token", "cli-test-token", "unknown"}, stdout, stderr)
	if exitCode != 2 {
		t.Fatalf("expected unknown-command exit code 2, got %d", exitCode)
	}
	if strings.Contains(stderr.String(), "unsupported --output") {
		t.Fatalf("expected text output mode to be accepted, got %s", stderr.String())
	}
}

func TestRunRejectsProdProfileWhenTLSVerifyIsDisabled(t *testing.T) {
	tokenFile := filepath.Join(t.TempDir(), "control.token")
	if err := os.WriteFile(tokenFile, []byte("prod-token-value-0123456789012345\n"), 0o600); err != nil {
		t.Fatalf("write token file: %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := run([]string{"--runtime-profile", "prod", "--auth-token-file", tokenFile, "--tls-insecure-skip-verify=true", "unknown"}, stdout, stderr)
	if exitCode != 2 {
		t.Fatalf("expected exit code 2, got %d", exitCode)
	}

	output := stderr.String()
	if !strings.Contains(output, "invalid CLI configuration") || !strings.Contains(output, "does not allow --tls-insecure-skip-verify") {
		t.Fatalf("expected prod TLS skip-verify validation failure, got: %s", output)
	}
}

func TestRunAllowsAuthTokenFileInProduction(t *testing.T) {
	tokenFile := filepath.Join(t.TempDir(), "control.token")
	if err := os.WriteFile(tokenFile, []byte("prod-token-from-file-0123456789012345\n"), 0o600); err != nil {
		t.Fatalf("write token file: %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run([]string{"--runtime-profile", "prod", "--auth-token-file", tokenFile, "unknown"}, stdout, stderr)
	if exitCode != 2 {
		t.Fatalf("expected exit code 2, got %d", exitCode)
	}

	output := stderr.String()
	if strings.Contains(output, "invalid CLI configuration") {
		t.Fatalf("expected no auth validation failure, got: %s", output)
	}
	if !strings.Contains(output, "unknown command") {
		t.Fatalf("expected unknown command after valid config, got: %s", output)
	}
}

func TestRunAuthBootstrapAndRotate(t *testing.T) {
	tokenFile := filepath.Join(t.TempDir(), "control.token")

	bootstrapOut := &bytes.Buffer{}
	bootstrapErr := &bytes.Buffer{}
	exitCode := run([]string{"auth", "bootstrap", "--file", tokenFile}, bootstrapOut, bootstrapErr)
	if exitCode != 0 {
		t.Fatalf("bootstrap exit code=%d stderr=%s", exitCode, bootstrapErr.String())
	}
	var bootstrapPayload map[string]any
	if err := json.Unmarshal(bootstrapOut.Bytes(), &bootstrapPayload); err != nil {
		t.Fatalf("unmarshal bootstrap output: %v", err)
	}
	if got := bootstrapPayload["operation"]; got != "bootstrap" {
		t.Fatalf("expected bootstrap operation, got %v", got)
	}
	if got := bootstrapPayload["token_file"]; got != tokenFile {
		t.Fatalf("expected token file %q, got %v", tokenFile, got)
	}

	originalTokenRaw, err := os.ReadFile(tokenFile)
	if err != nil {
		t.Fatalf("read token file: %v", err)
	}
	originalToken := strings.TrimSpace(string(originalTokenRaw))
	if originalToken == "" {
		t.Fatalf("expected token file to be populated")
	}
	if strings.Contains(bootstrapOut.String(), originalToken) {
		t.Fatalf("bootstrap output must not include raw token value")
	}

	rotateOut := &bytes.Buffer{}
	rotateErr := &bytes.Buffer{}
	exitCode = run([]string{"auth", "rotate", "--file", tokenFile}, rotateOut, rotateErr)
	if exitCode != 0 {
		t.Fatalf("rotate exit code=%d stderr=%s", exitCode, rotateErr.String())
	}
	var rotatePayload map[string]any
	if err := json.Unmarshal(rotateOut.Bytes(), &rotatePayload); err != nil {
		t.Fatalf("unmarshal rotate output: %v", err)
	}
	if got := rotatePayload["operation"]; got != "rotate" {
		t.Fatalf("expected rotate operation, got %v", got)
	}
	if got := rotatePayload["rotated"]; got != true {
		t.Fatalf("expected rotated=true, got %v", got)
	}

	rotatedTokenRaw, err := os.ReadFile(tokenFile)
	if err != nil {
		t.Fatalf("read rotated token file: %v", err)
	}
	rotatedToken := strings.TrimSpace(string(rotatedTokenRaw))
	if rotatedToken == "" {
		t.Fatalf("expected rotated token file to be populated")
	}
	if rotatedToken == originalToken {
		t.Fatalf("expected rotated token to differ from original")
	}
	if strings.Contains(rotateOut.String(), rotatedToken) {
		t.Fatalf("rotate output must not include raw token value")
	}
}

func TestRunAuthRotateMissingFileReturnsUsageExitCode(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := run([]string{"auth", "rotate"}, stdout, stderr)
	if exitCode != 2 {
		t.Fatalf("expected usage exit code 2 for auth rotate missing --file, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "--file is required") {
		t.Fatalf("expected missing --file message, got %q", stderr.String())
	}
}

func TestRunAuthBootstrapLocalDev(t *testing.T) {
	profilesPath := filepath.Join(t.TempDir(), "profiles.json")
	runtimeRoot := filepath.Join(t.TempDir(), "runtime-root")
	t.Setenv(cliProfilesPathEnvKey, profilesPath)
	t.Setenv(runtimepaths.EnvRuntimeRootDir, runtimeRoot)

	bootstrapOut := &bytes.Buffer{}
	bootstrapErr := &bytes.Buffer{}
	exitCode := run([]string{
		"auth", "bootstrap-local-dev",
		"--profile", "local-dev-test",
		"--mode", "tcp",
		"--address", "127.0.0.1:17071",
		"--workspace", "ws-local-dev",
	}, bootstrapOut, bootstrapErr)
	if exitCode != 0 {
		t.Fatalf("local-dev bootstrap exit code=%d stderr=%s", exitCode, bootstrapErr.String())
	}

	var bootstrapPayload localDevAuthBootstrapResponse
	if err := json.Unmarshal(bootstrapOut.Bytes(), &bootstrapPayload); err != nil {
		t.Fatalf("unmarshal local-dev bootstrap output: %v", err)
	}
	expectedTokenFile := filepath.Join(runtimeRoot, "control", defaultLocalDevTokenName)
	if bootstrapPayload.Operation != "bootstrap_local_dev" {
		t.Fatalf("expected bootstrap_local_dev operation, got %q", bootstrapPayload.Operation)
	}
	if bootstrapPayload.TokenFile != expectedTokenFile {
		t.Fatalf("expected token file %q, got %q", expectedTokenFile, bootstrapPayload.TokenFile)
	}
	if !bootstrapPayload.TokenCreated || bootstrapPayload.TokenRotated {
		t.Fatalf("expected token_created=true and token_rotated=false, got created=%v rotated=%v", bootstrapPayload.TokenCreated, bootstrapPayload.TokenRotated)
	}
	if strings.TrimSpace(bootstrapPayload.TokenSHA256) == "" {
		t.Fatalf("expected token sha256 metadata")
	}
	if bootstrapPayload.Profile.Name != "local-dev-test" {
		t.Fatalf("expected profile local-dev-test, got %q", bootstrapPayload.Profile.Name)
	}
	if bootstrapPayload.Profile.AuthTokenFile != expectedTokenFile {
		t.Fatalf("expected profile auth-token-file %q, got %q", expectedTokenFile, bootstrapPayload.Profile.AuthTokenFile)
	}
	if bootstrapPayload.ActiveProfile != "local-dev-test" {
		t.Fatalf("expected active profile local-dev-test, got %q", bootstrapPayload.ActiveProfile)
	}
	if bootstrapPayload.Defaults.Profile.Value != "local-dev-test" || bootstrapPayload.Defaults.Profile.Source != "explicit" || bootstrapPayload.Defaults.Profile.OverrideFlag != "--profile" {
		t.Fatalf("expected explicit profile defaults metadata, got %+v", bootstrapPayload.Defaults.Profile)
	}
	if bootstrapPayload.Defaults.Workspace.Value != "ws-local-dev" || bootstrapPayload.Defaults.Workspace.Source != "explicit" || bootstrapPayload.Defaults.Workspace.OverrideFlag != "--workspace" {
		t.Fatalf("expected explicit workspace defaults metadata, got %+v", bootstrapPayload.Defaults.Workspace)
	}
	if bootstrapPayload.Defaults.TokenFile.Value != expectedTokenFile || bootstrapPayload.Defaults.TokenFile.Source != "default" || bootstrapPayload.Defaults.TokenFile.OverrideFlag != "--token-file" {
		t.Fatalf("expected token-file defaults metadata, got %+v", bootstrapPayload.Defaults.TokenFile)
	}
	if len(bootstrapPayload.Defaults.OverrideHints) < 3 {
		t.Fatalf("expected override hints in defaults metadata, got %+v", bootstrapPayload.Defaults.OverrideHints)
	}
	if !strings.Contains(strings.Join(bootstrapPayload.Defaults.OverrideHints, " "), "--workspace") ||
		!strings.Contains(strings.Join(bootstrapPayload.Defaults.OverrideHints, " "), "--profile") ||
		!strings.Contains(strings.Join(bootstrapPayload.Defaults.OverrideHints, " "), "--token-file") {
		t.Fatalf("expected override hints to include workspace/profile/token-file flags, got %+v", bootstrapPayload.Defaults.OverrideHints)
	}
	if strings.Contains(bootstrapOut.String(), "token_value") {
		t.Fatalf("local-dev bootstrap output must not include raw token values")
	}

	tokenInfo, err := os.Stat(expectedTokenFile)
	if err != nil {
		t.Fatalf("stat local-dev token file: %v", err)
	}
	if tokenInfo.Mode().Perm() != 0o600 {
		t.Fatalf("expected local-dev token permissions 0600, got %o", tokenInfo.Mode().Perm())
	}

	replayOut := &bytes.Buffer{}
	replayErr := &bytes.Buffer{}
	exitCode = run([]string{
		"auth", "bootstrap-local-dev",
		"--profile", "local-dev-test",
		"--mode", "tcp",
		"--address", "127.0.0.1:17071",
		"--workspace", "ws-local-dev",
	}, replayOut, replayErr)
	if exitCode != 0 {
		t.Fatalf("local-dev bootstrap replay exit code=%d stderr=%s", exitCode, replayErr.String())
	}
	var replayPayload localDevAuthBootstrapResponse
	if err := json.Unmarshal(replayOut.Bytes(), &replayPayload); err != nil {
		t.Fatalf("unmarshal local-dev bootstrap replay output: %v", err)
	}
	if replayPayload.TokenCreated || replayPayload.TokenRotated {
		t.Fatalf("expected replay to reuse token material without rotate")
	}
	if replayPayload.TokenSHA256 != bootstrapPayload.TokenSHA256 {
		t.Fatalf("expected replay token sha256 to remain stable")
	}

	rotateOut := &bytes.Buffer{}
	rotateErr := &bytes.Buffer{}
	exitCode = run([]string{
		"auth", "bootstrap-local-dev",
		"--profile", "local-dev-test",
		"--mode", "tcp",
		"--address", "127.0.0.1:17071",
		"--workspace", "ws-local-dev",
		"--rotate-token",
	}, rotateOut, rotateErr)
	if exitCode != 0 {
		t.Fatalf("local-dev bootstrap rotate exit code=%d stderr=%s", exitCode, rotateErr.String())
	}
	var rotatePayload localDevAuthBootstrapResponse
	if err := json.Unmarshal(rotateOut.Bytes(), &rotatePayload); err != nil {
		t.Fatalf("unmarshal local-dev bootstrap rotate output: %v", err)
	}
	if rotatePayload.TokenCreated || !rotatePayload.TokenRotated {
		t.Fatalf("expected rotate call to set token_rotated=true")
	}
	if rotatePayload.TokenSHA256 == replayPayload.TokenSHA256 {
		t.Fatalf("expected rotated token sha256 to change")
	}
}

func TestRunAuthBootstrapLocalDevDefaultsMetadataSources(t *testing.T) {
	profilesPath := filepath.Join(t.TempDir(), "profiles.json")
	runtimeRoot := filepath.Join(t.TempDir(), "runtime-root")
	t.Setenv(cliProfilesPathEnvKey, profilesPath)
	t.Setenv(runtimepaths.EnvRuntimeRootDir, runtimeRoot)

	bootstrapOut := &bytes.Buffer{}
	bootstrapErr := &bytes.Buffer{}
	exitCode := run([]string{"auth", "bootstrap-local-dev"}, bootstrapOut, bootstrapErr)
	if exitCode != 0 {
		t.Fatalf("local-dev bootstrap defaults exit code=%d stderr=%s", exitCode, bootstrapErr.String())
	}

	var payload localDevAuthBootstrapResponse
	if err := json.Unmarshal(bootstrapOut.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal local-dev bootstrap defaults output: %v", err)
	}
	expectedTokenFile := filepath.Join(runtimeRoot, "control", defaultLocalDevTokenName)
	if payload.Defaults.Profile.Value != defaultLocalDevProfileName || payload.Defaults.Profile.Source != "default" {
		t.Fatalf("expected default profile metadata, got %+v", payload.Defaults.Profile)
	}
	if payload.Defaults.Workspace.Value != defaultLocalDevWorkspaceID || payload.Defaults.Workspace.Source != "default" {
		t.Fatalf("expected default workspace metadata, got %+v", payload.Defaults.Workspace)
	}
	if payload.Defaults.TokenFile.Value != expectedTokenFile || payload.Defaults.TokenFile.Source != "default" {
		t.Fatalf("expected default token-file metadata, got %+v", payload.Defaults.TokenFile)
	}
}

func TestBuildCLITransportTLSConfig(t *testing.T) {
	if cfg, err := buildCLITransportTLSConfig(cliTransportTLSOptions{ListenerMode: "tcp"}); err != nil || cfg != nil {
		t.Fatalf("expected nil tls config when no tls options set, cfg=%v err=%v", cfg, err)
	}

	if _, err := buildCLITransportTLSConfig(cliTransportTLSOptions{
		ListenerMode:        "unix",
		TLSInsecureSkipCert: true,
	}); err == nil {
		t.Fatalf("expected non-tcp tls usage to fail")
	}

	if _, err := buildCLITransportTLSConfig(cliTransportTLSOptions{
		ListenerMode:      "tcp",
		TLSClientCertFile: "/tmp/client.crt",
	}); err == nil {
		t.Fatalf("expected client cert/key pairing error")
	}

	cfg, err := buildCLITransportTLSConfig(cliTransportTLSOptions{
		ListenerMode:        "tcp",
		TLSInsecureSkipCert: true,
	})
	if err != nil {
		t.Fatalf("expected insecure tls config to pass: %v", err)
	}
	if cfg == nil {
		t.Fatalf("expected tls config")
	}
	if !cfg.InsecureSkipVerify {
		t.Fatalf("expected insecure skip verify to be enabled")
	}

	if _, err := buildCLITransportTLSConfig(cliTransportTLSOptions{
		ListenerMode: "tcp",
		TLSCAFile:    filepath.Join(t.TempDir(), "missing-ca.pem"),
	}); err == nil {
		t.Fatalf("expected missing CA file to fail")
	}
}
