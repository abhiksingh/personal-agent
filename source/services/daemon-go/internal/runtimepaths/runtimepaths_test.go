package runtimepaths

import (
	"path/filepath"
	"testing"
)

func TestNormalizeProfile(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		raw  string
		want string
	}{
		{name: "empty", raw: "", want: "user"},
		{name: "whitespace", raw: "   ", want: "user"},
		{name: "default profile stays explicit", raw: "default", want: "default"},
		{name: "upper user", raw: "USER", want: "user"},
		{name: "custom", raw: "test", want: "test"},
		{name: "sanitized", raw: "Test Profile!", want: "test-profile"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := NormalizeProfile(tc.raw)
			if got != tc.want {
				t.Fatalf("NormalizeProfile(%q)=%q want %q", tc.raw, got, tc.want)
			}
		})
	}
}

func TestResolveRootDirWithExplicitRootOverride(t *testing.T) {
	t.Parallel()

	explicit := filepath.Join("tmp", "runtime-state")
	resolved, err := resolveRootDirWith(
		func(key string) string {
			if key == EnvRuntimeRootDir {
				return explicit
			}
			return ""
		},
		func() (string, error) {
			t.Fatalf("user config dir should not be called when %s is set", EnvRuntimeRootDir)
			return "", nil
		},
	)
	if err != nil {
		t.Fatalf("resolve root with explicit override: %v", err)
	}
	if !filepath.IsAbs(resolved) {
		t.Fatalf("expected absolute resolved root path, got %q", resolved)
	}
	if filepath.Base(resolved) != "runtime-state" {
		t.Fatalf("expected explicit root tail runtime-state, got %q", resolved)
	}
}

func TestResolveRootDirWithProfile(t *testing.T) {
	t.Parallel()

	configDir := filepath.Join(string(filepath.Separator), "tmp", "pa-config")
	resolve := func(profile string) (string, error) {
		return resolveRootDirWith(
			func(key string) string {
				if key == EnvRuntimeProfile {
					return profile
				}
				return ""
			},
			func() (string, error) {
				return configDir, nil
			},
		)
	}

	userRoot, err := resolve("")
	if err != nil {
		t.Fatalf("resolve user profile root: %v", err)
	}
	wantUser := filepath.Join(configDir, "personal-agent")
	if userRoot != wantUser {
		t.Fatalf("user profile root=%q want %q", userRoot, wantUser)
	}

	testRoot, err := resolve("test-suite")
	if err != nil {
		t.Fatalf("resolve non-user profile root: %v", err)
	}
	wantTest := filepath.Join(configDir, "personal-agent-profiles", "test-suite")
	if testRoot != wantTest {
		t.Fatalf("test profile root=%q want %q", testRoot, wantTest)
	}

	defaultRoot, err := resolve("default")
	if err != nil {
		t.Fatalf("resolve explicit default profile root: %v", err)
	}
	wantDefault := filepath.Join(configDir, "personal-agent-profiles", "default")
	if defaultRoot != wantDefault {
		t.Fatalf("default profile root=%q want %q", defaultRoot, wantDefault)
	}
}

func TestDefaultPathsUseRuntimeRootOverride(t *testing.T) {
	root := t.TempDir()
	t.Setenv(EnvRuntimeRootDir, root)

	dbPath, err := DefaultDBPath()
	if err != nil {
		t.Fatalf("default db path: %v", err)
	}
	if dbPath != filepath.Join(root, "runtime.db") {
		t.Fatalf("db path=%q", dbPath)
	}

	inbound, err := DefaultInboundDir()
	if err != nil {
		t.Fatalf("default inbound dir: %v", err)
	}
	if inbound != filepath.Join(root, "inbound") {
		t.Fatalf("inbound path=%q", inbound)
	}

	channels, err := DefaultChannelsDir()
	if err != nil {
		t.Fatalf("default channels dir: %v", err)
	}
	if channels != filepath.Join(root, "channels") {
		t.Fatalf("channels path=%q", channels)
	}

	connectors, err := DefaultConnectorsDir()
	if err != nil {
		t.Fatalf("default connectors dir: %v", err)
	}
	if connectors != filepath.Join(root, "connectors") {
		t.Fatalf("connectors path=%q", connectors)
	}
}
