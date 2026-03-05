package agentexec

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

func containsAny(value string, keys []string) bool {
	for _, key := range keys {
		if strings.Contains(value, key) {
			return true
		}
	}
	return false
}

func extractURL(text string) string {
	for _, token := range strings.Fields(text) {
		candidate := strings.TrimSpace(strings.Trim(token, ".,;:()[]{}<>\"'"))
		if candidate == "" {
			continue
		}
		parsed, err := url.Parse(candidate)
		if err != nil {
			continue
		}
		if parsed.Scheme == "" || parsed.Host == "" {
			continue
		}
		if strings.EqualFold(parsed.Scheme, "http") || strings.EqualFold(parsed.Scheme, "https") {
			return candidate
		}
	}
	return ""
}

func normalizeURLCandidate(raw string) string {
	candidate := strings.TrimSpace(strings.Trim(raw, ".,;:()[]{}<>\"'"))
	if candidate == "" {
		return ""
	}
	parsed, err := url.Parse(candidate)
	if err != nil {
		return ""
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	if strings.EqualFold(parsed.Scheme, "http") || strings.EqualFold(parsed.Scheme, "https") {
		return candidate
	}
	return ""
}

func extractPath(text string) string {
	for _, token := range strings.Fields(text) {
		candidate := strings.TrimSpace(strings.Trim(token, ".,;:()[]{}<>\"'"))
		if strings.HasPrefix(candidate, "/") {
			return candidate
		}
	}
	return ""
}

func looksLikeFinderRequest(lowerRequest string) bool {
	if containsAny(lowerRequest, []string{
		"finder",
		"delete file", "remove file", "delete folder", "remove folder",
		"list file", "list files", "list folder", "list directory",
		"preview file", "show file", "inspect file",
		"find file", "search file", "locate file",
	}) {
		return true
	}
	if containsAny(lowerRequest, []string{"find ", "search ", "locate "}) &&
		!containsAny(lowerRequest, []string{
			"http://", "https://", "url", "website", "web page",
			"email", "mail", "inbox", "reply",
			"calendar", "meeting", "schedule event", "reschedule",
			"sms", "imessage", "text ",
		}) {
		return true
	}
	return containsAny(lowerRequest, []string{"file", "folder", "directory"}) &&
		containsAny(lowerRequest, []string{"find", "search", "locate", "list", "preview", "inspect", "delete", "remove", "trash"})
}

func extractFinderQuery(text string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return ""
	}
	pathToken := extractPath(trimmed)
	normalized := trimmed
	if pathToken != "" {
		normalized = strings.ReplaceAll(normalized, pathToken, " ")
	}
	normalized = strings.ToLower(normalized)
	tokens := finderQueryWordPattern.FindAllString(normalized, -1)
	if len(tokens) == 0 {
		return ""
	}
	queryTokens := make([]string, 0, len(tokens))
	for _, token := range tokens {
		cleaned := strings.TrimSpace(strings.ToLower(token))
		if cleaned == "" {
			continue
		}
		if _, blocked := finderStopWords[cleaned]; blocked {
			continue
		}
		queryTokens = append(queryTokens, cleaned)
	}
	return strings.TrimSpace(strings.Join(queryTokens, " "))
}

func looksLikeBrowserRequest(lowerRequest string) bool {
	if containsAny(lowerRequest, []string{"browser", "website", "web site", "webpage", "web page", "url", "link"}) {
		return true
	}
	return containsAny(lowerRequest, []string{"open", "visit", "browse"}) &&
		containsAny(lowerRequest, []string{"web", "site", "page", "url", "link"})
}

func looksLikeMessagesRequest(lowerRequest string) bool {
	return containsAny(lowerRequest, []string{"imessage", "i message", "sms", "send message", "message to", "text "})
}

func extractMessageRecipient(text string) string {
	email := strings.TrimSpace(firstMatch(emailAddressPattern, text))
	if email != "" {
		return email
	}
	phone := strings.TrimSpace(firstMatch(phoneNumberPattern, text))
	if phone != "" {
		return strings.ReplaceAll(phone, " ", "")
	}
	hint := recipientHintRegex.FindStringSubmatch(text)
	if len(hint) > 1 {
		return strings.TrimSpace(strings.Trim(hint[1], ".,;:()[]{}<>\"'"))
	}
	return ""
}

func extractMessageChannel(text string) string {
	lower := strings.ToLower(strings.TrimSpace(text))
	if lower == "" {
		return ""
	}
	if containsAny(lower, []string{"imessage", "i message"}) {
		return "imessage"
	}
	if strings.Contains(lower, "sms") {
		return "sms"
	}
	return ""
}

func normalizeMessageChannel(channel string) string {
	switch strings.ToLower(strings.TrimSpace(channel)) {
	case "imessage":
		return "imessage"
	case "sms":
		return "sms"
	case "message":
		return "imessage"
	default:
		return ""
	}
}

func extractMessageBody(text string) string {
	if quoted := extractQuotedText(text); quoted != "" {
		return quoted
	}
	colon := strings.Index(text, ":")
	if colon >= 0 && colon < len(text)-1 {
		return strings.TrimSpace(strings.Trim(text[colon+1:], " \"'"))
	}
	return ""
}

func extractCalendarEventID(text string) string {
	if matches := calendarEventIDHintPattern.FindStringSubmatch(text); len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return strings.TrimSpace(firstMatch(calendarEventIDTokenPattern, text))
}

func containsSlot(slots []string, key string) bool {
	target := strings.TrimSpace(key)
	for _, slot := range slots {
		if strings.TrimSpace(slot) == target {
			return true
		}
	}
	return false
}

func extractMailSummaryLimit(text string) int {
	fields := strings.Fields(strings.TrimSpace(text))
	for _, field := range fields {
		candidate := strings.Trim(strings.TrimSpace(field), ".,;:()[]{}<>\"'")
		if candidate == "" {
			continue
		}
		value, err := strconv.Atoi(candidate)
		if err != nil || value <= 0 {
			continue
		}
		if value > 50 {
			return 50
		}
		return value
	}
	return 0
}

func extractQuotedText(text string) string {
	start := strings.Index(text, "\"")
	if start >= 0 {
		end := strings.Index(text[start+1:], "\"")
		if end >= 0 {
			return strings.TrimSpace(text[start+1 : start+1+end])
		}
	}
	start = strings.Index(text, "'")
	if start >= 0 {
		end := strings.Index(text[start+1:], "'")
		if end >= 0 {
			return strings.TrimSpace(text[start+1 : start+1+end])
		}
	}
	return ""
}

func firstMatch(pattern *regexp.Regexp, value string) string {
	if pattern == nil {
		return ""
	}
	return pattern.FindString(strings.TrimSpace(value))
}

func parseConfidence(raw string) float64 {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0
	}
	value, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return 0
	}
	return value
}
