package browser

import (
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"

	adapterhelpers "personalagent/runtime/internal/connectors/adapters/helpers"
	connectorcontract "personalagent/runtime/internal/connectors/contract"
)

var (
	safeAppNamePattern = regexp.MustCompile(`^[A-Za-z0-9 _-]{1,32}$`)
)

func resolveBrowserAppName() string {
	return resolveBrowserAppNameFromValue(os.Getenv(envBrowserAppName))
}

func resolveBrowserAppNameFromValue(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "Safari"
	}
	if !safeAppNamePattern.MatchString(trimmed) {
		return "Safari"
	}
	return trimmed
}

func isBrowserAutomationDryRunEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(envBrowserAutomationDryRun))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func resolveBrowserStepInput(step connectorcontract.TaskStep) (browserStepInput, error) {
	if len(step.Input) == 0 {
		return browserStepInput{}, fmt.Errorf("browser step input is required")
	}
	targetURL, err := adapterhelpers.RequiredStringInput(step.Input, "url")
	if err != nil {
		return browserStepInput{}, err
	}
	query, err := adapterhelpers.OptionalStringInput(step.Input, "query")
	if err != nil {
		return browserStepInput{}, err
	}
	return browserStepInput{
		URL:   targetURL,
		Query: query,
	}, nil
}

func enforceURLGuardrails(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if parsed.Scheme == "" || parsed.Hostname() == "" {
		return fmt.Errorf("URL must include scheme and host")
	}

	scheme := strings.ToLower(parsed.Scheme)
	host := strings.ToLower(parsed.Hostname())
	loopback := host == "localhost" || host == "127.0.0.1" || host == "::1"

	if scheme == "http" && !loopback {
		return fmt.Errorf("only https URLs are allowed for non-loopback hosts")
	}
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("only http/https URLs are supported")
	}
	if strings.HasSuffix(host, ".internal") {
		return fmt.Errorf("target host is blocked by guardrail")
	}
	return nil
}

func removeIfExists(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
