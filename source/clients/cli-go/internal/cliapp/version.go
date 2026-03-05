package cliapp

import (
	"flag"
	"fmt"
	"io"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
)

var (
	cliVersion   = "dev"
	cliCommit    = ""
	cliBuiltTime = ""
)

type cliVersionResponse struct {
	SchemaVersion string `json:"schema_version"`
	Program       string `json:"program"`
	Version       string `json:"version"`
	Commit        string `json:"commit,omitempty"`
	BuiltAt       string `json:"built_at,omitempty"`
	GoVersion     string `json:"go_version"`
	Platform      string `json:"platform"`
	VCSRevision   string `json:"vcs_revision,omitempty"`
	VCSTime       string `json:"vcs_time,omitempty"`
	VCSModified   *bool  `json:"vcs_modified,omitempty"`
}

func runVersionCommand(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("version", flag.ContinueOnError)
	flags.SetOutput(stderr)
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if len(flags.Args()) > 0 {
		fmt.Fprintln(stderr, "version does not accept positional arguments")
		return 2
	}

	return writeVersionResponse(stdout, buildCLIVersionResponse())
}

func buildCLIVersionResponse() cliVersionResponse {
	response := cliVersionResponse{
		SchemaVersion: "1.0.0",
		Program:       "personal-agent",
		Version:       strings.TrimSpace(cliVersion),
		Commit:        strings.TrimSpace(cliCommit),
		BuiltAt:       strings.TrimSpace(cliBuiltTime),
		GoVersion:     runtime.Version(),
		Platform:      runtime.GOOS + "/" + runtime.GOARCH,
	}
	if response.Version == "" {
		response.Version = "dev"
	}

	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return response
	}
	if response.Version == "dev" {
		candidate := strings.TrimSpace(buildInfo.Main.Version)
		if candidate != "" && candidate != "(devel)" {
			response.Version = candidate
		}
	}

	settings := make(map[string]string, len(buildInfo.Settings))
	for _, setting := range buildInfo.Settings {
		key := strings.TrimSpace(setting.Key)
		if key == "" {
			continue
		}
		settings[key] = strings.TrimSpace(setting.Value)
	}

	if response.VCSRevision = settings["vcs.revision"]; response.VCSRevision != "" && response.Commit == "" {
		response.Commit = response.VCSRevision
	}
	response.VCSTime = settings["vcs.time"]

	if modifiedRaw := settings["vcs.modified"]; modifiedRaw != "" {
		if modified, err := strconv.ParseBool(modifiedRaw); err == nil {
			modifiedValue := modified
			response.VCSModified = &modifiedValue
		}
	}
	return response
}
