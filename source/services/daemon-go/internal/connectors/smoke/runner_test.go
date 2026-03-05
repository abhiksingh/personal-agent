package smoke

import (
	"context"
	"testing"
)

func TestRunExecutesAllConnectorHappyPaths(t *testing.T) {
	t.Setenv("PA_MAIL_AUTOMATION_DRY_RUN", "1")
	t.Setenv("PA_CALENDAR_AUTOMATION_DRY_RUN", "1")
	t.Setenv("PA_BROWSER_AUTOMATION_DRY_RUN", "1")

	runner := NewRunner()
	report := runner.Run(context.Background())

	if !report.Success {
		t.Fatalf("expected smoke report success, got failure: %+v", report)
	}
	if len(report.Scenarios) != 4 {
		t.Fatalf("expected 4 connector scenarios, got %d", len(report.Scenarios))
	}

	seen := map[string]bool{}
	for _, scenario := range report.Scenarios {
		if !scenario.Success {
			t.Fatalf("expected scenario %s to pass: %s", scenario.Name, scenario.Error)
		}
		seen[scenario.Name] = true
	}

	for _, name := range []string{"mail", "calendar", "browser", "finder"} {
		if !seen[name] {
			t.Fatalf("expected scenario %s to be present", name)
		}
	}
}
