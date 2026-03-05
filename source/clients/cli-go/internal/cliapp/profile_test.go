package cliapp

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunProfileSetUseGetList(t *testing.T) {
	profilesPath := filepath.Join(t.TempDir(), "profiles.json")
	t.Setenv(cliProfilesPathEnvKey, profilesPath)
	t.Setenv(cliWorkspaceEnvKey, "")

	devTokenPath := filepath.Join(t.TempDir(), "dev.token")
	if err := os.WriteFile(devTokenPath, []byte("dev-token"), 0o600); err != nil {
		t.Fatalf("write dev token file: %v", err)
	}

	var setDevOut bytes.Buffer
	var setDevErr bytes.Buffer
	setDevCode := run([]string{
		"profile", "set",
		"--name", "dev",
		"--mode", "tcp",
		"--address", "127.0.0.1:17101",
		"--workspace", "ws-dev",
		"--auth-token-file", devTokenPath,
	}, &setDevOut, &setDevErr)
	if setDevCode != 0 {
		t.Fatalf("profile set dev failed: code=%d stderr=%s output=%s", setDevCode, setDevErr.String(), setDevOut.String())
	}

	var setDevResponse cliProfileResponse
	if err := json.Unmarshal(setDevOut.Bytes(), &setDevResponse); err != nil {
		t.Fatalf("decode profile set dev response: %v", err)
	}
	if setDevResponse.Profile.Name != "dev" {
		t.Fatalf("expected profile name dev, got %q", setDevResponse.Profile.Name)
	}
	if !setDevResponse.Active || setDevResponse.ActiveProfile != "dev" {
		t.Fatalf("expected dev profile to be active on first set, got active=%v active_profile=%q", setDevResponse.Active, setDevResponse.ActiveProfile)
	}
	if setDevResponse.Profile.WorkspaceID != "ws-dev" {
		t.Fatalf("expected workspace ws-dev, got %q", setDevResponse.Profile.WorkspaceID)
	}
	if setDevResponse.Path != profilesPath {
		t.Fatalf("expected profile path %q, got %q", profilesPath, setDevResponse.Path)
	}

	prodTokenPath := filepath.Join(t.TempDir(), "prod.token")
	if err := os.WriteFile(prodTokenPath, []byte("prod-token"), 0o600); err != nil {
		t.Fatalf("write prod token file: %v", err)
	}

	var setProdOut bytes.Buffer
	var setProdErr bytes.Buffer
	setProdCode := run([]string{
		"profile", "set",
		"--name", "prod",
		"--mode", "tcp",
		"--address", "127.0.0.1:17102",
		"--workspace", "ws-prod",
		"--auth-token-file", prodTokenPath,
	}, &setProdOut, &setProdErr)
	if setProdCode != 0 {
		t.Fatalf("profile set prod failed: code=%d stderr=%s output=%s", setProdCode, setProdErr.String(), setProdOut.String())
	}
	var setProdResponse cliProfileResponse
	if err := json.Unmarshal(setProdOut.Bytes(), &setProdResponse); err != nil {
		t.Fatalf("decode profile set prod response: %v", err)
	}
	if setProdResponse.ActiveProfile != "dev" {
		t.Fatalf("expected active profile to stay dev after adding prod, got %q", setProdResponse.ActiveProfile)
	}
	if setProdResponse.Active {
		t.Fatalf("expected prod profile to be inactive before use")
	}

	var useProdOut bytes.Buffer
	var useProdErr bytes.Buffer
	useProdCode := run([]string{"profile", "use", "--name", "prod"}, &useProdOut, &useProdErr)
	if useProdCode != 0 {
		t.Fatalf("profile use prod failed: code=%d stderr=%s output=%s", useProdCode, useProdErr.String(), useProdOut.String())
	}
	var useProdResponse cliProfileResponse
	if err := json.Unmarshal(useProdOut.Bytes(), &useProdResponse); err != nil {
		t.Fatalf("decode profile use prod response: %v", err)
	}
	if useProdResponse.ActiveProfile != "prod" || !useProdResponse.Active {
		t.Fatalf("expected prod to be active after use, got active=%v active_profile=%q", useProdResponse.Active, useProdResponse.ActiveProfile)
	}

	var getActiveOut bytes.Buffer
	var getActiveErr bytes.Buffer
	getActiveCode := run([]string{"profile", "get"}, &getActiveOut, &getActiveErr)
	if getActiveCode != 0 {
		t.Fatalf("profile get active failed: code=%d stderr=%s output=%s", getActiveCode, getActiveErr.String(), getActiveOut.String())
	}
	var getActiveResponse cliProfileResponse
	if err := json.Unmarshal(getActiveOut.Bytes(), &getActiveResponse); err != nil {
		t.Fatalf("decode profile get active response: %v", err)
	}
	if getActiveResponse.Profile.Name != "prod" || !getActiveResponse.Active {
		t.Fatalf("expected active get to return prod active profile, got name=%q active=%v", getActiveResponse.Profile.Name, getActiveResponse.Active)
	}

	var getDevOut bytes.Buffer
	var getDevErr bytes.Buffer
	getDevCode := run([]string{"profile", "get", "--name", "dev"}, &getDevOut, &getDevErr)
	if getDevCode != 0 {
		t.Fatalf("profile get --name dev failed: code=%d stderr=%s output=%s", getDevCode, getDevErr.String(), getDevOut.String())
	}
	var getDevResponse cliProfileResponse
	if err := json.Unmarshal(getDevOut.Bytes(), &getDevResponse); err != nil {
		t.Fatalf("decode profile get --name dev response: %v", err)
	}
	if getDevResponse.Profile.Name != "dev" || getDevResponse.Active {
		t.Fatalf("expected dev profile to be returned inactive, got name=%q active=%v", getDevResponse.Profile.Name, getDevResponse.Active)
	}

	var listOut bytes.Buffer
	var listErr bytes.Buffer
	listCode := run([]string{"profile", "list"}, &listOut, &listErr)
	if listCode != 0 {
		t.Fatalf("profile list failed: code=%d stderr=%s output=%s", listCode, listErr.String(), listOut.String())
	}
	var listResponse cliProfileListResponse
	if err := json.Unmarshal(listOut.Bytes(), &listResponse); err != nil {
		t.Fatalf("decode profile list response: %v", err)
	}
	if listResponse.ActiveProfile != "prod" {
		t.Fatalf("expected active profile prod, got %q", listResponse.ActiveProfile)
	}
	if len(listResponse.Profiles) != 2 {
		t.Fatalf("expected two profiles, got %d", len(listResponse.Profiles))
	}
	if listResponse.Profiles[0].Name != "dev" || listResponse.Profiles[1].Name != "prod" {
		t.Fatalf("expected deterministic sorted profiles [dev, prod], got [%s, %s]", listResponse.Profiles[0].Name, listResponse.Profiles[1].Name)
	}

	info, err := os.Stat(profilesPath)
	if err != nil {
		t.Fatalf("stat profiles file: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("expected profiles file permissions 0600, got %o", info.Mode().Perm())
	}
}

func TestRunProfileRenameDeleteAndActiveInspection(t *testing.T) {
	profilesPath := filepath.Join(t.TempDir(), "profiles.json")
	t.Setenv(cliProfilesPathEnvKey, profilesPath)

	runSet := func(name, address, workspace string) {
		t.Helper()
		var out bytes.Buffer
		var stderr bytes.Buffer
		exitCode := run([]string{
			"profile", "set",
			"--name", name,
			"--mode", "tcp",
			"--address", address,
			"--workspace", workspace,
		}, &out, &stderr)
		if exitCode != 0 {
			t.Fatalf("profile set %s failed: code=%d stderr=%s output=%s", name, exitCode, stderr.String(), out.String())
		}
	}
	runSet("dev", "127.0.0.1:17101", "ws-dev")
	runSet("prod", "127.0.0.1:17102", "ws-prod")

	var activeInitialOut bytes.Buffer
	var activeInitialErr bytes.Buffer
	activeInitialCode := run([]string{"profile", "active"}, &activeInitialOut, &activeInitialErr)
	if activeInitialCode != 0 {
		t.Fatalf("profile active initial failed: code=%d stderr=%s output=%s", activeInitialCode, activeInitialErr.String(), activeInitialOut.String())
	}
	var activeInitial cliProfileActiveResponse
	if err := json.Unmarshal(activeInitialOut.Bytes(), &activeInitial); err != nil {
		t.Fatalf("decode profile active initial response: %v", err)
	}
	if activeInitial.ActiveProfile != "dev" || !activeInitial.ProfileExists || activeInitial.Profile == nil || activeInitial.Profile.Name != "dev" {
		t.Fatalf("expected active profile dev, got %+v", activeInitial)
	}

	var renameOut bytes.Buffer
	var renameErr bytes.Buffer
	renameCode := run([]string{"profile", "rename", "--name", "prod", "--to", "prod-main"}, &renameOut, &renameErr)
	if renameCode != 0 {
		t.Fatalf("profile rename failed: code=%d stderr=%s output=%s", renameCode, renameErr.String(), renameOut.String())
	}
	var renameResponse cliProfileResponse
	if err := json.Unmarshal(renameOut.Bytes(), &renameResponse); err != nil {
		t.Fatalf("decode profile rename response: %v", err)
	}
	if renameResponse.PreviousName != "prod" || renameResponse.Profile.Name != "prod-main" {
		t.Fatalf("expected rename prod -> prod-main, got %+v", renameResponse)
	}
	if renameResponse.ActiveProfile != "dev" || renameResponse.Active {
		t.Fatalf("expected rename of inactive profile to keep active=dev, got %+v", renameResponse)
	}

	var useOut bytes.Buffer
	var useErr bytes.Buffer
	useCode := run([]string{"profile", "use", "--name", "prod-main"}, &useOut, &useErr)
	if useCode != 0 {
		t.Fatalf("profile use prod-main failed: code=%d stderr=%s output=%s", useCode, useErr.String(), useOut.String())
	}

	var activeRenamedOut bytes.Buffer
	var activeRenamedErr bytes.Buffer
	activeRenamedCode := run([]string{"profile", "active"}, &activeRenamedOut, &activeRenamedErr)
	if activeRenamedCode != 0 {
		t.Fatalf("profile active renamed failed: code=%d stderr=%s output=%s", activeRenamedCode, activeRenamedErr.String(), activeRenamedOut.String())
	}
	var activeRenamed cliProfileActiveResponse
	if err := json.Unmarshal(activeRenamedOut.Bytes(), &activeRenamed); err != nil {
		t.Fatalf("decode profile active renamed response: %v", err)
	}
	if activeRenamed.ActiveProfile != "prod-main" || !activeRenamed.ProfileExists || activeRenamed.Profile == nil || activeRenamed.Profile.Name != "prod-main" {
		t.Fatalf("expected active profile prod-main, got %+v", activeRenamed)
	}

	var deleteRenamedOut bytes.Buffer
	var deleteRenamedErr bytes.Buffer
	deleteRenamedCode := run([]string{"profile", "delete", "--name", "prod-main"}, &deleteRenamedOut, &deleteRenamedErr)
	if deleteRenamedCode != 0 {
		t.Fatalf("profile delete prod-main failed: code=%d stderr=%s output=%s", deleteRenamedCode, deleteRenamedErr.String(), deleteRenamedOut.String())
	}
	var deleteRenamed cliProfileDeleteResponse
	if err := json.Unmarshal(deleteRenamedOut.Bytes(), &deleteRenamed); err != nil {
		t.Fatalf("decode profile delete prod-main response: %v", err)
	}
	if deleteRenamed.DeletedProfile != "prod-main" || !deleteRenamed.ActiveProfileChanged || deleteRenamed.ActiveProfile != "dev" {
		t.Fatalf("expected delete active profile to promote dev, got %+v", deleteRenamed)
	}

	var deleteDevOut bytes.Buffer
	var deleteDevErr bytes.Buffer
	deleteDevCode := run([]string{"profile", "delete", "--name", "dev"}, &deleteDevOut, &deleteDevErr)
	if deleteDevCode != 0 {
		t.Fatalf("profile delete dev failed: code=%d stderr=%s output=%s", deleteDevCode, deleteDevErr.String(), deleteDevOut.String())
	}
	var deleteDev cliProfileDeleteResponse
	if err := json.Unmarshal(deleteDevOut.Bytes(), &deleteDev); err != nil {
		t.Fatalf("decode profile delete dev response: %v", err)
	}
	if deleteDev.DeletedProfile != "dev" || !deleteDev.ActiveProfileChanged || deleteDev.ActiveProfile != "" {
		t.Fatalf("expected delete final profile to clear active profile, got %+v", deleteDev)
	}

	var activeEmptyOut bytes.Buffer
	var activeEmptyErr bytes.Buffer
	activeEmptyCode := run([]string{"profile", "active"}, &activeEmptyOut, &activeEmptyErr)
	if activeEmptyCode != 0 {
		t.Fatalf("profile active empty failed: code=%d stderr=%s output=%s", activeEmptyCode, activeEmptyErr.String(), activeEmptyOut.String())
	}
	var activeEmpty cliProfileActiveResponse
	if err := json.Unmarshal(activeEmptyOut.Bytes(), &activeEmpty); err != nil {
		t.Fatalf("decode profile active empty response: %v", err)
	}
	if activeEmpty.ActiveProfile != "" || activeEmpty.ProfileExists || activeEmpty.Profile != nil {
		t.Fatalf("expected no active profile after deleting all profiles, got %+v", activeEmpty)
	}

	var listOut bytes.Buffer
	var listErr bytes.Buffer
	listCode := run([]string{"profile", "list"}, &listOut, &listErr)
	if listCode != 0 {
		t.Fatalf("profile list after deletes failed: code=%d stderr=%s output=%s", listCode, listErr.String(), listOut.String())
	}
	var listResponse cliProfileListResponse
	if err := json.Unmarshal(listOut.Bytes(), &listResponse); err != nil {
		t.Fatalf("decode profile list after deletes response: %v", err)
	}
	if listResponse.ActiveProfile != "" || len(listResponse.Profiles) != 0 {
		t.Fatalf("expected empty profile state after deletes, got %+v", listResponse)
	}
}

func TestRunProfileRenameDeleteValidationErrors(t *testing.T) {
	profilesPath := filepath.Join(t.TempDir(), "profiles.json")
	t.Setenv(cliProfilesPathEnvKey, profilesPath)

	var missingNameErr bytes.Buffer
	missingSetNameCode := run([]string{"profile", "set"}, &bytes.Buffer{}, &missingNameErr)
	if missingSetNameCode != 2 || !strings.Contains(missingNameErr.String(), "--name is required") {
		t.Fatalf("expected profile set missing-name usage error, code=%d stderr=%s", missingSetNameCode, missingNameErr.String())
	}
	missingNameErr.Reset()

	missingUseNameCode := run([]string{"profile", "use"}, &bytes.Buffer{}, &missingNameErr)
	if missingUseNameCode != 2 || !strings.Contains(missingNameErr.String(), "--name is required") {
		t.Fatalf("expected profile use missing-name usage error, code=%d stderr=%s", missingUseNameCode, missingNameErr.String())
	}
	missingNameErr.Reset()

	missingDeleteNameCode := run([]string{"profile", "delete"}, &bytes.Buffer{}, &missingNameErr)
	if missingDeleteNameCode != 2 || !strings.Contains(missingNameErr.String(), "--name is required") {
		t.Fatalf("expected profile delete missing-name usage error, code=%d stderr=%s", missingDeleteNameCode, missingNameErr.String())
	}

	var setOut bytes.Buffer
	var setErr bytes.Buffer
	setCode := run([]string{"profile", "set", "--name", "dev"}, &setOut, &setErr)
	if setCode != 0 {
		t.Fatalf("profile set dev failed: code=%d stderr=%s output=%s", setCode, setErr.String(), setOut.String())
	}

	var renameOut bytes.Buffer
	var renameErr bytes.Buffer
	renameCode := run([]string{"profile", "rename", "--name", "dev", "--to", "dev"}, &renameOut, &renameErr)
	if renameCode != 0 {
		t.Fatalf("profile rename same-name no-op failed: code=%d stderr=%s output=%s", renameCode, renameErr.String(), renameOut.String())
	}

	renameMissingTargetCode := run([]string{"profile", "rename", "--name", "dev"}, &bytes.Buffer{}, &renameErr)
	if renameMissingTargetCode != 2 || !strings.Contains(renameErr.String(), "--to is required") {
		t.Fatalf("expected missing --to validation error, code=%d stderr=%s", renameMissingTargetCode, renameErr.String())
	}
	renameErr.Reset()

	renameMissingSourceCode := run([]string{"profile", "rename", "--name", "missing", "--to", "new"}, &bytes.Buffer{}, &renameErr)
	if renameMissingSourceCode != 1 || !strings.Contains(renameErr.String(), `profile "missing" not found`) {
		t.Fatalf("expected missing source profile error, code=%d stderr=%s", renameMissingSourceCode, renameErr.String())
	}
	renameErr.Reset()

	if run([]string{"profile", "set", "--name", "prod"}, &bytes.Buffer{}, &bytes.Buffer{}) != 0 {
		t.Fatalf("profile set prod failed")
	}
	renameConflictCode := run([]string{"profile", "rename", "--name", "prod", "--to", "dev"}, &bytes.Buffer{}, &renameErr)
	if renameConflictCode != 1 || !strings.Contains(renameErr.String(), `profile "dev" already exists`) {
		t.Fatalf("expected rename conflict error, code=%d stderr=%s", renameConflictCode, renameErr.String())
	}

	var deleteErr bytes.Buffer
	deleteMissingCode := run([]string{"profile", "delete", "--name", "missing"}, &bytes.Buffer{}, &deleteErr)
	if deleteMissingCode != 1 || !strings.Contains(deleteErr.String(), `profile "missing" not found`) {
		t.Fatalf("expected missing delete profile error, code=%d stderr=%s", deleteMissingCode, deleteErr.String())
	}

	var activeOut bytes.Buffer
	var activeErr bytes.Buffer
	activeCode := run([]string{"profile", "active"}, &activeOut, &activeErr)
	if activeCode != 0 {
		t.Fatalf("profile active failed: code=%d stderr=%s output=%s", activeCode, activeErr.String(), activeOut.String())
	}
}

func TestRunAppliesActiveProfileDefaults(t *testing.T) {
	server := startCLITestServer(t)
	profilesPath := filepath.Join(t.TempDir(), "profiles.json")
	t.Setenv(cliProfilesPathEnvKey, profilesPath)
	t.Setenv(cliWorkspaceEnvKey, "")

	tokenFile := filepath.Join(t.TempDir(), "control.token")
	if err := os.WriteFile(tokenFile, []byte("cli-test-token\n"), 0o600); err != nil {
		t.Fatalf("write control token file: %v", err)
	}

	var setOut bytes.Buffer
	var setErr bytes.Buffer
	setCode := run([]string{
		"profile", "set",
		"--name", "local",
		"--mode", "tcp",
		"--address", server.Address(),
		"--workspace", "ws-profile",
		"--auth-token-file", tokenFile,
	}, &setOut, &setErr)
	if setCode != 0 {
		t.Fatalf("profile set failed: code=%d stderr=%s output=%s", setCode, setErr.String(), setOut.String())
	}

	var smokeOut bytes.Buffer
	var smokeErr bytes.Buffer
	smokeCode := run([]string{"smoke"}, &smokeOut, &smokeErr)
	if smokeCode != 0 {
		t.Fatalf("smoke with profile defaults failed: code=%d stderr=%s output=%s", smokeCode, smokeErr.String(), smokeOut.String())
	}
	if !strings.Contains(smokeOut.String(), `"healthy"`) {
		t.Fatalf("expected smoke output to contain healthy field, got: %s", smokeOut.String())
	}
	if got := strings.TrimSpace(os.Getenv(cliWorkspaceEnvKey)); got != "ws-profile" {
		t.Fatalf("expected profile workspace env ws-profile, got %q", got)
	}
}

func TestRunProfileDefaultsRespectExplicitFlagsAndWorkspaceEnv(t *testing.T) {
	server := startCLITestServer(t)
	profilesPath := filepath.Join(t.TempDir(), "profiles.json")
	t.Setenv(cliProfilesPathEnvKey, profilesPath)
	t.Setenv(cliWorkspaceEnvKey, "ws-explicit")

	badTokenFile := filepath.Join(t.TempDir(), "bad.token")
	if err := os.WriteFile(badTokenFile, []byte("bad-token"), 0o600); err != nil {
		t.Fatalf("write bad token file: %v", err)
	}

	var setOut bytes.Buffer
	var setErr bytes.Buffer
	setCode := run([]string{
		"profile", "set",
		"--name", "bad-default",
		"--mode", "tcp",
		"--address", "127.0.0.1:1",
		"--workspace", "ws-profile",
		"--auth-token-file", badTokenFile,
	}, &setOut, &setErr)
	if setCode != 0 {
		t.Fatalf("profile set bad-default failed: code=%d stderr=%s output=%s", setCode, setErr.String(), setOut.String())
	}

	var smokeOut bytes.Buffer
	var smokeErr bytes.Buffer
	smokeCode := run([]string{
		"--mode", "tcp",
		"--address", server.Address(),
		"--auth-token", "cli-test-token",
		"smoke",
	}, &smokeOut, &smokeErr)
	if smokeCode != 0 {
		t.Fatalf("smoke with explicit flags failed: code=%d stderr=%s output=%s", smokeCode, smokeErr.String(), smokeOut.String())
	}
	if got := strings.TrimSpace(os.Getenv(cliWorkspaceEnvKey)); got != "ws-explicit" {
		t.Fatalf("expected workspace env to remain explicit ws-explicit, got %q", got)
	}
}

func TestRunProfileCommandBypassesAuthValidation(t *testing.T) {
	profilesPath := filepath.Join(t.TempDir(), "profiles.json")
	t.Setenv(cliProfilesPathEnvKey, profilesPath)

	var out bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"--runtime-profile", "prod", "profile", "list"}, &out, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected profile command to bypass production auth validation: code=%d stderr=%s output=%s", exitCode, stderr.String(), out.String())
	}

	var payload cliProfileListResponse
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("decode profile list response: %v", err)
	}
	if len(payload.Profiles) != 0 {
		t.Fatalf("expected empty profile list for fresh state, got %d profiles", len(payload.Profiles))
	}
}

func TestRunAuthCommandBypassesProfileStateLoad(t *testing.T) {
	profilesPath := filepath.Join(t.TempDir(), "profiles.json")
	if err := os.WriteFile(profilesPath, []byte("{invalid-json"), 0o600); err != nil {
		t.Fatalf("write malformed profiles state: %v", err)
	}
	t.Setenv(cliProfilesPathEnvKey, profilesPath)

	authTokenFile := filepath.Join(t.TempDir(), "control.token")
	var out bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"auth", "bootstrap", "--file", authTokenFile}, &out, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected auth bootstrap to bypass profile-state decoding: code=%d stderr=%s output=%s", exitCode, stderr.String(), out.String())
	}
	if strings.Contains(stderr.String(), "decode cli profile state") {
		t.Fatalf("expected no profile decode errors for auth bootstrap, stderr=%s", stderr.String())
	}
}
