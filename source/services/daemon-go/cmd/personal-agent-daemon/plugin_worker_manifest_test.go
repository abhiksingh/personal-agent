package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	shared "personalagent/runtime/internal/shared/contracts"
)

func TestResolveDaemonPluginWorkersManifestPathPrefersFlagValue(t *testing.T) {
	t.Setenv(envDaemonPluginWorkersManifest, "/tmp/from-env.json")
	resolved := resolveDaemonPluginWorkersManifestPath("/tmp/from-flag.json")
	if resolved != "/tmp/from-flag.json" {
		t.Fatalf("expected flag path precedence, got %q", resolved)
	}
}

func TestResolveDaemonPluginWorkersManifestPathUsesEnvFallback(t *testing.T) {
	t.Setenv(envDaemonPluginWorkersManifest, "/tmp/from-env.json")
	resolved := resolveDaemonPluginWorkersManifestPath("")
	if resolved != "/tmp/from-env.json" {
		t.Fatalf("expected env path fallback, got %q", resolved)
	}
}

func TestLoadDaemonPluginWorkersFromCustomManifestFile(t *testing.T) {
	manifestPath := filepath.Join(t.TempDir(), "workers.json")
	if err := os.WriteFile(manifestPath, []byte(`{
  "workers": [
    {
      "plugin_id": "custom.connector",
      "kind": "connector",
      "worker_type": "mail",
      "worker_health_interval_ms": 300,
      "restart_max_restarts": 1
    },
    {
      "plugin_id": "custom.channel",
      "kind": "channel",
      "worker_type": "app_chat"
    }
  ]
}`), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	workers, err := loadDaemonPluginWorkers("/tmp/personal-agent-daemon", "/tmp/runtime.db", manifestPath)
	if err != nil {
		t.Fatalf("load daemon plugin workers: %v", err)
	}
	if len(workers) != 2 {
		t.Fatalf("expected two workers from custom manifest, got %d", len(workers))
	}

	connector := workers[0]
	if connector.PluginID != "custom.connector" || connector.Kind != shared.AdapterKindConnector {
		t.Fatalf("unexpected connector worker spec: %+v", connector)
	}
	foundDB := false
	for idx := 0; idx < len(connector.Args)-1; idx++ {
		if connector.Args[idx] == "--db" && connector.Args[idx+1] == "/tmp/runtime.db" {
			foundDB = true
			break
		}
	}
	if !foundDB {
		t.Fatalf("expected connector worker args to include --db /tmp/runtime.db, got %v", connector.Args)
	}

	channel := workers[1]
	if channel.PluginID != "custom.channel" || channel.Kind != shared.AdapterKindChannel {
		t.Fatalf("unexpected channel worker spec: %+v", channel)
	}
	for idx := 0; idx < len(channel.Args)-1; idx++ {
		if channel.Args[idx] == "--db" {
			t.Fatalf("expected channel worker args to omit --db, got %v", channel.Args)
		}
	}
}

func TestLoadDaemonPluginWorkersRejectsInvalidKind(t *testing.T) {
	manifestPath := filepath.Join(t.TempDir(), "workers-invalid.json")
	if err := os.WriteFile(manifestPath, []byte(`{
  "workers": [
    {
      "plugin_id": "bad.worker",
      "kind": "invalid",
      "worker_type": "mail"
    }
  ]
}`), 0o600); err != nil {
		t.Fatalf("write invalid manifest: %v", err)
	}

	_, err := loadDaemonPluginWorkers("/tmp/personal-agent-daemon", "/tmp/runtime.db", manifestPath)
	if err == nil {
		t.Fatalf("expected invalid worker kind error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "unsupported worker kind") {
		t.Fatalf("expected unsupported kind error, got %v", err)
	}
}

func TestLoadDaemonPluginWorkersRejectsDuplicatePluginID(t *testing.T) {
	manifestPath := filepath.Join(t.TempDir(), "workers-duplicate.json")
	if err := os.WriteFile(manifestPath, []byte(`{
  "workers": [
    {
      "plugin_id": "dup.worker",
      "kind": "connector",
      "worker_type": "mail"
    },
    {
      "plugin_id": "dup.worker",
      "kind": "channel",
      "worker_type": "app_chat"
    }
  ]
}`), 0o600); err != nil {
		t.Fatalf("write duplicate manifest: %v", err)
	}

	_, err := loadDaemonPluginWorkers("/tmp/personal-agent-daemon", "/tmp/runtime.db", manifestPath)
	if err == nil {
		t.Fatalf("expected duplicate plugin id error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "duplicate plugin_id") {
		t.Fatalf("expected duplicate plugin id error, got %v", err)
	}
}
