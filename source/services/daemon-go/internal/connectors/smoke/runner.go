package smoke

import (
	"context"
	"time"

	browseradapter "personalagent/runtime/internal/connectors/adapters/browser"
	calendaradapter "personalagent/runtime/internal/connectors/adapters/calendar"
	finderadapter "personalagent/runtime/internal/connectors/adapters/finder"
	mailadapter "personalagent/runtime/internal/connectors/adapters/mail"
	connectorregistry "personalagent/runtime/internal/connectors/registry"
	"personalagent/runtime/internal/core/service/connectorflow"
	"personalagent/runtime/internal/core/types"
)

type ScenarioResult struct {
	Name     string            `json:"name"`
	Success  bool              `json:"success"`
	Evidence map[string]string `json:"evidence,omitempty"`
	Error    string            `json:"error,omitempty"`
}

type Report struct {
	StartedAt  time.Time        `json:"started_at"`
	FinishedAt time.Time        `json:"finished_at"`
	Success    bool             `json:"success"`
	Scenarios  []ScenarioResult `json:"scenarios"`
}

type Runner struct{}

func NewRunner() *Runner {
	return &Runner{}
}

func (r *Runner) Run(ctx context.Context) Report {
	report := Report{StartedAt: time.Now().UTC(), Scenarios: []ScenarioResult{}}

	registry := connectorregistry.New()
	_ = registry.Register(mailadapter.NewAdapter("mail.smoke"))
	_ = registry.Register(calendaradapter.NewAdapter("calendar.smoke"))
	_ = registry.Register(browseradapter.NewAdapter("browser.smoke"))
	_ = registry.Register(finderadapter.NewAdapter("finder.smoke"))

	mailService := connectorflow.NewMailHappyPathService(registry)
	report.Scenarios = append(report.Scenarios, runScenario("mail", func() (map[string]string, error) {
		result, err := mailService.Execute(ctx, types.MailHappyPathRequest{
			WorkspaceID:      "ws_smoke",
			RunID:            "run_mail_smoke",
			RequestedByActor: "actor_requester",
			SubjectPrincipal: "actor_subject",
			ActingAsActor:    "actor_subject",
			CorrelationID:    "corr_mail_smoke",
		})
		if err != nil {
			return nil, err
		}
		return map[string]string{
			"draft_id":   result.DraftTrace.Evidence["draft_id"],
			"message_id": result.SendTrace.Evidence["message_id"],
			"reply_id":   result.ReplyTrace.Evidence["reply_id"],
		}, nil
	}))

	calendarService := connectorflow.NewCalendarHappyPathService(registry)
	report.Scenarios = append(report.Scenarios, runScenario("calendar", func() (map[string]string, error) {
		result, err := calendarService.Execute(ctx, types.CalendarHappyPathRequest{
			WorkspaceID:      "ws_smoke",
			RunID:            "run_calendar_smoke",
			RequestedByActor: "actor_requester",
			SubjectPrincipal: "actor_subject",
			ActingAsActor:    "actor_subject",
			CorrelationID:    "corr_calendar_smoke",
		})
		if err != nil {
			return nil, err
		}
		return map[string]string{
			"event_id":            result.CreateTrace.Evidence["event_id"],
			"updated_event_id":    result.UpdateTrace.Evidence["event_id"],
			"cancelled_event_id":  result.CancelTrace.Evidence["event_id"],
			"create_operation_id": result.CreateTrace.Evidence["operation_id"],
			"update_operation_id": result.UpdateTrace.Evidence["operation_id"],
			"cancel_operation_id": result.CancelTrace.Evidence["operation_id"],
		}, nil
	}))

	browserService := connectorflow.NewBrowserHappyPathService(registry)
	report.Scenarios = append(report.Scenarios, runScenario("browser", func() (map[string]string, error) {
		result, err := browserService.Execute(ctx, types.BrowserHappyPathRequest{
			WorkspaceID:      "ws_smoke",
			RunID:            "run_browser_smoke",
			RequestedByActor: "actor_requester",
			SubjectPrincipal: "actor_subject",
			ActingAsActor:    "actor_subject",
			CorrelationID:    "corr_browser_smoke",
			TargetURL:        "https://example.com",
		})
		if err != nil {
			return nil, err
		}
		return map[string]string{
			"url":           result.OpenTrace.Evidence["url"],
			"extraction_id": result.ExtractTrace.Evidence["extraction_id"],
			"close_id":      result.CloseTrace.Evidence["close_id"],
		}, nil
	}))

	finderService := connectorflow.NewFinderHappyPathService(registry, nil, nil)
	report.Scenarios = append(report.Scenarios, runScenario("finder", func() (map[string]string, error) {
		result, err := finderService.Execute(ctx, types.FinderHappyPathRequest{
			WorkspaceID:      "ws_smoke",
			RunID:            "run_finder_smoke",
			RequestedByActor: "actor_requester",
			SubjectPrincipal: "actor_subject",
			ActingAsActor:    "actor_subject",
			CorrelationID:    "corr_finder_smoke",
			TargetPath:       "/tmp/smoke.txt",
			ApprovalPhrase:   types.DestructiveApprovalPhrase,
		})
		if err != nil {
			return nil, err
		}
		return map[string]string{
			"path":         result.ListTrace.Evidence["path"],
			"preview_id":   result.PreviewTrace.Evidence["preview_id"],
			"deleted_path": result.DeleteTrace.Evidence["deleted_path"],
		}, nil
	}))

	report.Success = true
	for _, scenario := range report.Scenarios {
		if !scenario.Success {
			report.Success = false
			break
		}
	}
	report.FinishedAt = time.Now().UTC()
	return report
}

func runScenario(name string, run func() (map[string]string, error)) ScenarioResult {
	evidence, err := run()
	if err != nil {
		return ScenarioResult{Name: name, Success: false, Error: err.Error()}
	}
	return ScenarioResult{Name: name, Success: true, Evidence: evidence}
}
