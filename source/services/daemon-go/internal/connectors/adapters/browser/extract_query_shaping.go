package browser

import (
	"net/url"
	"strings"
)

type browserExtractSnapshot struct {
	Title   string
	URL     string
	Content string
}

func parseBrowserExtractSnapshot(targetURL string, rawSummary string) browserExtractSnapshot {
	summary := strings.TrimSpace(rawSummary)
	if summary == "" {
		parsed, err := url.Parse(strings.TrimSpace(targetURL))
		if err != nil {
			return browserExtractSnapshot{
				Title:   "Browser Page",
				URL:     strings.TrimSpace(targetURL),
				Content: "",
			}
		}
		return browserExtractSnapshot{
			Title:   parsed.Hostname(),
			URL:     strings.TrimSpace(targetURL),
			Content: "",
		}
	}

	if strings.HasPrefix(summary, browserExtractPrefix) {
		encoded := strings.TrimPrefix(summary, browserExtractPrefix)
		parts := strings.SplitN(encoded, "::", 3)
		if len(parts) == 3 {
			title, _ := url.QueryUnescape(parts[0])
			resolvedURL, _ := url.QueryUnescape(parts[1])
			content, _ := url.QueryUnescape(parts[2])
			title = strings.TrimSpace(title)
			resolvedURL = strings.TrimSpace(resolvedURL)
			content = normalizedBrowserContent(content)
			if title == "" {
				title = "Browser Page"
			}
			if resolvedURL == "" {
				resolvedURL = strings.TrimSpace(targetURL)
			}
			return browserExtractSnapshot{
				Title:   title,
				URL:     resolvedURL,
				Content: content,
			}
		}
	}

	parts := strings.SplitN(summary, "|", 2)
	if len(parts) == 2 {
		title := strings.TrimSpace(parts[0])
		resolvedURL := strings.TrimSpace(parts[1])
		if title == "" {
			title = "Browser Page"
		}
		if resolvedURL == "" {
			resolvedURL = strings.TrimSpace(targetURL)
		}
		return browserExtractSnapshot{
			Title:   title,
			URL:     resolvedURL,
			Content: "",
		}
	}

	return browserExtractSnapshot{
		Title:   "Browser Page",
		URL:     strings.TrimSpace(targetURL),
		Content: normalizedBrowserContent(summary),
	}
}

func normalizedBrowserContent(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	normalized := strings.Join(strings.Fields(trimmed), " ")
	if len(normalized) > maxBrowserExtractChars {
		return normalized[:maxBrowserExtractChars]
	}
	return normalized
}

func browserContentPreview(content string) string {
	normalized := normalizedBrowserContent(content)
	if len(normalized) <= 280 {
		return normalized
	}
	return normalized[:280]
}

func resolveBrowserQueryAnswer(query string, content string) string {
	trimmedQuery := strings.TrimSpace(query)
	trimmedContent := strings.TrimSpace(content)
	if trimmedQuery == "" || trimmedContent == "" {
		return ""
	}
	segments := splitBrowserContentSegments(trimmedContent)
	if len(segments) == 0 {
		return browserContentPreview(trimmedContent)
	}
	queryTerms := browserQueryTerms(trimmedQuery)
	bestScore := 0
	bestSegments := make([]string, 0, 3)
	for _, segment := range segments {
		score := browserQueryScore(segment, queryTerms)
		if score <= 0 {
			continue
		}
		if score > bestScore {
			bestScore = score
			bestSegments = []string{segment}
			continue
		}
		if score == bestScore && len(bestSegments) < 3 {
			bestSegments = append(bestSegments, segment)
		}
	}
	if len(bestSegments) == 0 {
		preview := browserContentPreview(trimmedContent)
		if preview == "" {
			return ""
		}
		return preview
	}
	answer := strings.Join(bestSegments, " ")
	return browserContentPreview(answer)
}

func splitBrowserContentSegments(content string) []string {
	parts := strings.FieldsFunc(content, func(r rune) bool {
		switch r {
		case '\n', '\r', '.', '!', '?':
			return true
		default:
			return false
		}
	})
	segments := make([]string, 0, len(parts))
	for _, part := range parts {
		normalized := strings.TrimSpace(strings.Join(strings.Fields(part), " "))
		if len(normalized) < 8 {
			continue
		}
		segments = append(segments, normalized)
		if len(segments) >= 120 {
			break
		}
	}
	return segments
}

func browserQueryTerms(query string) map[string]struct{} {
	terms := make(map[string]struct{})
	for _, token := range strings.Fields(strings.ToLower(strings.TrimSpace(query))) {
		candidate := strings.Trim(token, ".,;:()[]{}<>\"'")
		if len(candidate) < 3 {
			continue
		}
		terms[candidate] = struct{}{}
	}
	return terms
}

func browserQueryScore(segment string, terms map[string]struct{}) int {
	if len(terms) == 0 {
		return 0
	}
	score := 0
	for _, token := range strings.Fields(strings.ToLower(segment)) {
		candidate := strings.Trim(token, ".,;:()[]{}<>\"'")
		if _, ok := terms[candidate]; ok {
			score++
		}
	}
	return score
}
