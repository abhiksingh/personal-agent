package daemonruntime

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestInspectInboundWatcherBridgeReturnsNotReadyWhenDirectoriesMissing(t *testing.T) {
	root := filepath.Join(t.TempDir(), "inbound")
	status := InspectInboundWatcherBridge(root)

	if status.InboxRoot != root {
		t.Fatalf("expected inbox root %q, got %q", root, status.InboxRoot)
	}
	if status.Ready {
		t.Fatalf("expected bridge status not ready when directories are missing")
	}
	if len(status.Sources) != 3 {
		t.Fatalf("expected 3 bridge sources, got %d", len(status.Sources))
	}
	for _, source := range status.Sources {
		if source.Ready {
			t.Fatalf("expected source %s not ready while directories are missing", source.Source)
		}
		if source.Pending.Exists || source.Processed.Exists || source.Failed.Exists {
			t.Fatalf("expected source %s directories to be absent", source.Source)
		}
	}
}

func TestEnsureInboundWatcherBridgeCreatesReadyLayout(t *testing.T) {
	root := filepath.Join(t.TempDir(), "inbound")
	status := EnsureInboundWatcherBridge(root)
	if !status.Ready {
		t.Fatalf("expected bridge status ready after ensure, got %+v", status)
	}

	for _, source := range status.Sources {
		if !source.Ready {
			t.Fatalf("expected source %s ready after ensure, got %+v", source.Source, source)
		}
		for _, dir := range []InboundWatcherBridgeDirectoryStatus{source.Pending, source.Processed, source.Failed} {
			if !dir.Exists {
				t.Fatalf("expected ensured directory to exist for source %s: %+v", source.Source, dir)
			}
			if !dir.Writable {
				t.Fatalf("expected ensured directory to be writable for source %s: %+v", source.Source, dir)
			}
			if _, err := os.Stat(dir.Path); err != nil {
				t.Fatalf("expected directory %s to exist on disk: %v", dir.Path, err)
			}
			if runtime.GOOS != "windows" {
				info, err := os.Stat(dir.Path)
				if err != nil {
					t.Fatalf("stat ensured directory %s: %v", dir.Path, err)
				}
				if got := info.Mode().Perm(); got != 0o700 {
					t.Fatalf("expected bridge directory permissions 0700 for %s, got %o", dir.Path, got)
				}
			}
		}
	}

	inspect := InspectInboundWatcherBridge(root)
	if !inspect.Ready {
		t.Fatalf("expected inspect to report ready after ensure, got %+v", inspect)
	}
}

func TestInboundWatcherBridgeSourceByID(t *testing.T) {
	status := InboundWatcherBridgeStatus{
		Sources: []InboundWatcherBridgeSourceStatus{
			{Source: "mail"},
			{Source: "calendar"},
		},
	}

	found, ok := InboundWatcherBridgeSourceByID(status, " calendar ")
	if !ok {
		t.Fatalf("expected calendar source to be found")
	}
	if found.Source != "calendar" {
		t.Fatalf("expected calendar source, got %+v", found)
	}

	if _, ok := InboundWatcherBridgeSourceByID(status, "browser"); ok {
		t.Fatalf("expected browser source to be missing")
	}
}
