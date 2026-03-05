package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"personalagent/runtime/internal/daemonruntime"
	shared "personalagent/runtime/internal/shared/contracts"
)

const (
	envDaemonPluginWorkersManifest = "PA_DAEMON_PLUGIN_WORKERS_MANIFEST"
)

const (
	defaultWorkerHandshakeTimeout = 4 * time.Second
	defaultWorkerHealthInterval   = 500 * time.Millisecond
	defaultWorkerHealthTimeout    = 2 * time.Second
	defaultWorkerRestartDelay     = 200 * time.Millisecond
	defaultWorkerRestartMax       = 3
	defaultWorkerHeartbeat        = 250 * time.Millisecond
)

//go:embed plugin_workers_manifest.json
var embeddedPluginWorkersManifest []byte

type pluginWorkerManifestDocument struct {
	Workers []pluginWorkerManifestEntry `json:"workers"`
}

type pluginWorkerManifestEntry struct {
	PluginID               string   `json:"plugin_id"`
	Kind                   string   `json:"kind"`
	WorkerType             string   `json:"worker_type"`
	Enabled                *bool    `json:"enabled,omitempty"`
	WorkerHealthIntervalMS *int     `json:"worker_health_interval_ms,omitempty"`
	HandshakeTimeoutMS     *int     `json:"handshake_timeout_ms,omitempty"`
	HealthIntervalMS       *int     `json:"health_interval_ms,omitempty"`
	HealthTimeoutMS        *int     `json:"health_timeout_ms,omitempty"`
	RestartMaxRestarts     *int     `json:"restart_max_restarts,omitempty"`
	RestartDelayMS         *int     `json:"restart_delay_ms,omitempty"`
	WorkingDirectory       string   `json:"working_directory,omitempty"`
	Env                    []string `json:"env,omitempty"`
	Args                   []string `json:"args,omitempty"`
}

func resolveDaemonPluginWorkersManifestPath(flagValue string) string {
	if trimmed := strings.TrimSpace(flagValue); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(os.Getenv(envDaemonPluginWorkersManifest))
}

func loadDaemonPluginWorkers(executable string, dbPath string, manifestPath string) ([]daemonruntime.PluginWorkerSpec, error) {
	trimmedExecutable := strings.TrimSpace(executable)
	if trimmedExecutable == "" {
		return nil, fmt.Errorf("daemon executable path is required for plugin worker bootstrap")
	}
	trimmedDBPath := strings.TrimSpace(dbPath)

	rawManifest, source, err := readDaemonPluginWorkerManifestBytes(manifestPath)
	if err != nil {
		return nil, err
	}

	decoder := json.NewDecoder(bytes.NewReader(rawManifest))
	decoder.DisallowUnknownFields()
	var document pluginWorkerManifestDocument
	if err := decoder.Decode(&document); err != nil {
		return nil, fmt.Errorf("decode plugin worker manifest (%s): %w", source, err)
	}
	if len(document.Workers) == 0 {
		return nil, fmt.Errorf("plugin worker manifest (%s) does not declare any workers", source)
	}

	seenPluginIDs := map[string]struct{}{}
	specs := make([]daemonruntime.PluginWorkerSpec, 0, len(document.Workers))
	for idx, entry := range document.Workers {
		enabled := true
		if entry.Enabled != nil {
			enabled = *entry.Enabled
		}
		if !enabled {
			continue
		}

		spec, err := manifestEntryToWorkerSpec(entry, trimmedExecutable, trimmedDBPath)
		if err != nil {
			return nil, fmt.Errorf("plugin worker manifest (%s) entry %d: %w", source, idx, err)
		}
		if _, exists := seenPluginIDs[spec.PluginID]; exists {
			return nil, fmt.Errorf("plugin worker manifest (%s) declares duplicate plugin_id %q", source, spec.PluginID)
		}
		seenPluginIDs[spec.PluginID] = struct{}{}
		specs = append(specs, spec)
	}

	if len(specs) == 0 {
		return nil, fmt.Errorf("plugin worker manifest (%s) has no enabled workers", source)
	}
	return specs, nil
}

func readDaemonPluginWorkerManifestBytes(manifestPath string) ([]byte, string, error) {
	trimmed := strings.TrimSpace(manifestPath)
	if trimmed == "" {
		if len(embeddedPluginWorkersManifest) == 0 {
			return nil, "embedded", fmt.Errorf("embedded plugin worker manifest is empty")
		}
		return embeddedPluginWorkersManifest, "embedded", nil
	}

	resolvedPath := trimmed
	if !filepath.IsAbs(resolvedPath) {
		if absolute, err := filepath.Abs(resolvedPath); err == nil {
			resolvedPath = absolute
		}
	}

	data, err := os.ReadFile(resolvedPath)
	if err != nil {
		return nil, resolvedPath, fmt.Errorf("read plugin worker manifest %s: %w", resolvedPath, err)
	}
	return data, resolvedPath, nil
}

func manifestEntryToWorkerSpec(entry pluginWorkerManifestEntry, executable string, dbPath string) (daemonruntime.PluginWorkerSpec, error) {
	pluginID := strings.TrimSpace(entry.PluginID)
	if pluginID == "" {
		return daemonruntime.PluginWorkerSpec{}, fmt.Errorf("plugin_id is required")
	}
	workerType := strings.ToLower(strings.TrimSpace(entry.WorkerType))
	if workerType == "" {
		return daemonruntime.PluginWorkerSpec{}, fmt.Errorf("worker_type is required")
	}

	kind, err := normalizeManifestWorkerKind(entry.Kind)
	if err != nil {
		return daemonruntime.PluginWorkerSpec{}, err
	}

	workerHealthInterval, err := durationFromOptionalMS(entry.WorkerHealthIntervalMS, defaultWorkerHeartbeat, "worker_health_interval_ms")
	if err != nil {
		return daemonruntime.PluginWorkerSpec{}, err
	}
	handshakeTimeout, err := durationFromOptionalMS(entry.HandshakeTimeoutMS, defaultWorkerHandshakeTimeout, "handshake_timeout_ms")
	if err != nil {
		return daemonruntime.PluginWorkerSpec{}, err
	}
	healthInterval, err := durationFromOptionalMS(entry.HealthIntervalMS, defaultWorkerHealthInterval, "health_interval_ms")
	if err != nil {
		return daemonruntime.PluginWorkerSpec{}, err
	}
	healthTimeout, err := durationFromOptionalMS(entry.HealthTimeoutMS, defaultWorkerHealthTimeout, "health_timeout_ms")
	if err != nil {
		return daemonruntime.PluginWorkerSpec{}, err
	}
	restartDelay, err := durationFromOptionalMS(entry.RestartDelayMS, defaultWorkerRestartDelay, "restart_delay_ms")
	if err != nil {
		return daemonruntime.PluginWorkerSpec{}, err
	}
	restartMax, err := intFromOptional(entry.RestartMaxRestarts, defaultWorkerRestartMax, "restart_max_restarts")
	if err != nil {
		return daemonruntime.PluginWorkerSpec{}, err
	}

	args := make([]string, 0, 12)
	if kind == shared.AdapterKindConnector {
		args = append(args, "--connector-worker", workerType)
	} else {
		args = append(args, "--channel-worker", workerType)
	}
	args = append(
		args,
		"--plugin-id", pluginID,
		"--worker-health-interval", workerHealthInterval.String(),
	)
	if kind == shared.AdapterKindConnector && strings.TrimSpace(dbPath) != "" {
		args = append(args, "--db", strings.TrimSpace(dbPath))
	}
	args = append(args, entry.Args...)

	spec := daemonruntime.PluginWorkerSpec{
		PluginID:         pluginID,
		Kind:             kind,
		Command:          executable,
		Args:             args,
		Env:              append([]string(nil), entry.Env...),
		WorkingDirectory: strings.TrimSpace(entry.WorkingDirectory),
		HandshakeTimeout: handshakeTimeout,
		HealthInterval:   healthInterval,
		HealthTimeout:    healthTimeout,
		RestartPolicy: daemonruntime.PluginRestartPolicy{
			MaxRestarts: restartMax,
			Delay:       restartDelay,
		},
	}
	return spec, nil
}

func normalizeManifestWorkerKind(raw string) (shared.AdapterKind, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(shared.AdapterKindConnector):
		return shared.AdapterKindConnector, nil
	case string(shared.AdapterKindChannel):
		return shared.AdapterKindChannel, nil
	default:
		return "", fmt.Errorf("unsupported worker kind %q", raw)
	}
}

func durationFromOptionalMS(value *int, fallback time.Duration, field string) (time.Duration, error) {
	if value == nil {
		return fallback, nil
	}
	if *value < 0 {
		return 0, fmt.Errorf("%s must be >= 0", field)
	}
	return time.Duration(*value) * time.Millisecond, nil
}

func intFromOptional(value *int, fallback int, field string) (int, error) {
	if value == nil {
		return fallback, nil
	}
	if *value < 0 {
		return 0, fmt.Errorf("%s must be >= 0", field)
	}
	return *value, nil
}
