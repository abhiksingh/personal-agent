package browser

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"runtime"
	"strings"
	"time"

	adapterscaffold "personalagent/runtime/internal/connectors/adapters/scaffold"
)

type browserOperationResult struct {
	OperationID string
	Transport   string
	Summary     string
}

var runBrowserCommand adapterscaffold.CommandRunner = adapterscaffold.DefaultCommandRunner

func executeBrowserOperation(ctx context.Context, mode string, targetURL string, browserApp string) (browserOperationResult, error) {
	normalizedMode := strings.ToLower(strings.TrimSpace(mode))
	if normalizedMode == "" {
		return browserOperationResult{}, fmt.Errorf("browser operation mode is required")
	}
	targetURL = strings.TrimSpace(targetURL)
	if targetURL == "" {
		return browserOperationResult{}, fmt.Errorf("browser target URL is required")
	}
	browserApp = resolveBrowserAppNameFromValue(browserApp)

	operationID := newBrowserOperationID(normalizedMode)
	if isBrowserAutomationDryRunEnabled() {
		return browserOperationResult{
			OperationID: operationID,
			Transport:   transportBrowserDryRun,
			Summary:     browserDryRunSummary(normalizedMode, targetURL, browserApp),
		}, nil
	}
	if runtime.GOOS != "darwin" {
		return browserOperationResult{}, fmt.Errorf("browser automation requires macOS (set %s=1 for dry-run)", envBrowserAutomationDryRun)
	}

	var scriptLines []string
	switch normalizedMode {
	case "open":
		scriptLines = browserOpenScript(browserApp)
	case "extract":
		scriptLines = browserExtractScript(browserApp)
	case "close":
		scriptLines = browserCloseScript(browserApp)
	default:
		return browserOperationResult{}, fmt.Errorf("unsupported browser operation mode %q", normalizedMode)
	}

	args := make([]string, 0, len(scriptLines)*2+1)
	for _, line := range scriptLines {
		args = append(args, "-e", line)
	}
	args = append(args, targetURL)

	result := adapterscaffold.ExecuteCommand(ctx, runBrowserCommand, "osascript", args...)
	if result.Err != nil {
		if result.Output != "" {
			return browserOperationResult{}, fmt.Errorf("browser automation failed: %s", result.Output)
		}
		return browserOperationResult{}, fmt.Errorf("browser automation failed: %w", result.Err)
	}

	summary := strings.TrimSpace(result.Output)
	if summary == "" {
		summary = browserDryRunSummary(normalizedMode, targetURL, browserApp)
	}
	return browserOperationResult{
		OperationID: operationID,
		Transport:   transportBrowserSafari,
		Summary:     summary,
	}, nil
}

func browserOpenScript(browserApp string) []string {
	return []string{
		"on run argv",
		"set targetURL to item 1 of argv",
		fmt.Sprintf("tell application \"%s\"", browserApp),
		"activate",
		"open location targetURL",
		"return \"opened \" & targetURL",
		"end tell",
		"end run",
	}
}

func browserExtractScript(browserApp string) []string {
	return []string{
		"on run argv",
		"set targetURL to item 1 of argv",
		fmt.Sprintf("tell application \"%s\"", browserApp),
		"activate",
		"if (count of documents) is 0 then",
		"open location targetURL",
		"delay 0.2",
		"end if",
		"set encodedTitle to do JavaScript \"encodeURIComponent(document.title || '')\" in front document",
		"set encodedURL to do JavaScript \"encodeURIComponent(window.location.href || '')\" in front document",
		"set encodedContent to do JavaScript \"(() => { const body = document.body; const text = body ? body.innerText : ''; return encodeURIComponent(text.slice(0, 24000)); })();\" in front document",
		"if encodedTitle is missing value then set encodedTitle to \"\"",
		"if encodedURL is missing value then set encodedURL to \"\"",
		"if encodedContent is missing value then set encodedContent to \"\"",
		"return \"" + browserExtractPrefix + "\" & encodedTitle & \"::\" & encodedURL & \"::\" & encodedContent",
		"end tell",
		"end run",
	}
}

func browserCloseScript(browserApp string) []string {
	return []string{
		"on run argv",
		"set targetURL to item 1 of argv",
		fmt.Sprintf("tell application \"%s\"", browserApp),
		"if (count of windows) is 0 then",
		"return \"no_windows\"",
		"end if",
		"repeat with w in windows",
		"repeat with t in tabs of w",
		"if (URL of t as text) is targetURL then",
		"set current tab of w to t",
		"close current tab of w",
		"return \"closed \" & targetURL",
		"end if",
		"end repeat",
		"end repeat",
		"set targetWindow to front window",
		"close current tab of targetWindow",
		"return \"closed_front_tab\"",
		"end tell",
		"end run",
	}
}

func browserDryRunSummary(mode string, targetURL string, browserApp string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "open":
		return fmt.Sprintf("dry-run: opened %s via %s", targetURL, browserApp)
	case "extract":
		return buildBrowserExtractPayload("Dry-Run Browser Page", targetURL, "This is deterministic dry-run browser extract content for "+targetURL+".")
	case "close":
		return fmt.Sprintf("dry-run: closed %s via %s", targetURL, browserApp)
	default:
		return "dry-run: browser automation simulated"
	}
}

func buildBrowserExtractPayload(title string, targetURL string, content string) string {
	return browserExtractPrefix +
		url.QueryEscape(strings.TrimSpace(title)) + "::" +
		url.QueryEscape(strings.TrimSpace(targetURL)) + "::" +
		url.QueryEscape(strings.TrimSpace(content))
}

func newBrowserOperationID(prefix string) string {
	tokenBytes := make([]byte, 4)
	if _, err := rand.Read(tokenBytes); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UTC().UnixNano())
	}
	return fmt.Sprintf("%s-%d-%s", prefix, time.Now().UTC().UnixNano(), hex.EncodeToString(tokenBytes))
}
