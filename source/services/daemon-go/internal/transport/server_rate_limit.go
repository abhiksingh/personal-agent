package transport

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	controlRateLimitKeyChatTurn               = "chat_turn"
	controlRateLimitKeyChatPersonaSet         = "chat_persona_set"
	controlRateLimitKeyAgentRun               = "agent_run"
	controlRateLimitKeyAgentApprove           = "agent_approve"
	controlRateLimitKeyTaskSubmit             = "task_submit"
	controlRateLimitKeyTaskCancel             = "task_cancel"
	controlRateLimitKeyTaskRetry              = "task_retry"
	controlRateLimitKeyTaskRequeue            = "task_requeue"
	controlRateLimitKeyDaemonLifecycleControl = "daemon_lifecycle_control"
	controlRateLimitKeyAutomationCreate       = "automation_create"
	controlRateLimitKeyAutomationUpdate       = "automation_update"
	controlRateLimitKeyAutomationDelete       = "automation_delete"
	controlRateLimitKeyAutomationRunSchedule  = "automation_run_schedule"
	controlRateLimitKeyAutomationRunCommEvent = "automation_run_comm_event"

	controlRateLimitScopeTypeActor     = "actor"
	controlRateLimitScopeTypeWorkspace = "workspace"
	controlRateLimitScopeTypeToken     = "token"
	controlRateLimitScopeTypeGlobal    = "global"

	controlRateLimitDefaultScopeKey         = "default"
	controlRateLimitTrustedActorHeader      = "X-PersonalAgent-Trusted-Actor-ID"     // deprecated, reject if supplied
	controlRateLimitTrustedWorkspaceHeader  = "X-PersonalAgent-Trusted-Workspace-ID" // deprecated, reject if supplied
	controlRateLimitDefaultMaxBuckets       = 4096
	controlRateLimitBucketTTLWindowMultiple = 4
)

type controlEndpointRateLimiter struct {
	mu         sync.Mutex
	limit      int
	window     time.Duration
	now        func() time.Time
	maxBuckets int
	bucketTTL  time.Duration
	buckets    map[string]controlEndpointRateBucket
}

type controlEndpointRateBucket struct {
	windowStart time.Time
	count       int
	lastSeen    time.Time
}

type controlEndpointRateDecision struct {
	allowed    bool
	limit      int
	remaining  int
	window     time.Duration
	resetAt    time.Time
	retryAfter time.Duration

	endpointKey string
	scopeType   string
	scopeKey    string
	bucketKey   string
}

func newControlEndpointRateLimiter(limit int, window time.Duration, now func() time.Time) *controlEndpointRateLimiter {
	if limit <= 0 || window <= 0 {
		return nil
	}
	nowFn := now
	if nowFn == nil {
		nowFn = func() time.Time { return time.Now().UTC() }
	}
	bucketTTL := window * controlRateLimitBucketTTLWindowMultiple
	if bucketTTL < window {
		bucketTTL = window
	}
	return &controlEndpointRateLimiter{
		limit:      limit,
		window:     window,
		now:        nowFn,
		maxBuckets: controlRateLimitDefaultMaxBuckets,
		bucketTTL:  bucketTTL,
		buckets:    map[string]controlEndpointRateBucket{},
	}
}

func (l *controlEndpointRateLimiter) allow(endpointKey string, scopeType string, scopeKey string) controlEndpointRateDecision {
	trimmedEndpoint := strings.TrimSpace(endpointKey)
	if l == nil || trimmedEndpoint == "" {
		return controlEndpointRateDecision{allowed: true}
	}
	normalizedScopeType, normalizedScopeKey := normalizeControlRateLimitScope(scopeType, scopeKey)
	bucketKey := controlRateLimitBucketKey(trimmedEndpoint, normalizedScopeType, normalizedScopeKey)

	now := l.now().UTC()
	l.mu.Lock()
	defer l.mu.Unlock()

	l.evictExpiredBucketsLocked(now)
	bucket, exists := l.buckets[bucketKey]
	if !exists && l.maxBuckets > 0 && len(l.buckets) >= l.maxBuckets {
		l.evictOldestBucketsLocked(len(l.buckets) - l.maxBuckets + 1)
	}
	if !exists || now.Sub(bucket.windowStart) >= l.window {
		bucket = controlEndpointRateBucket{windowStart: now, count: 0, lastSeen: now}
	}
	bucket.lastSeen = now

	resetAt := bucket.windowStart.Add(l.window)
	if bucket.count < l.limit {
		bucket.count++
		l.buckets[bucketKey] = bucket
		return controlEndpointRateDecision{
			allowed:     true,
			limit:       l.limit,
			remaining:   l.limit - bucket.count,
			window:      l.window,
			resetAt:     resetAt,
			endpointKey: trimmedEndpoint,
			scopeType:   normalizedScopeType,
			scopeKey:    normalizedScopeKey,
			bucketKey:   bucketKey,
		}
	}

	retryAfter := time.Until(resetAt)
	if retryAfter < 0 {
		retryAfter = 0
	}
	l.buckets[bucketKey] = bucket
	return controlEndpointRateDecision{
		allowed:     false,
		limit:       l.limit,
		remaining:   0,
		window:      l.window,
		resetAt:     resetAt,
		retryAfter:  retryAfter,
		endpointKey: trimmedEndpoint,
		scopeType:   normalizedScopeType,
		scopeKey:    normalizedScopeKey,
		bucketKey:   bucketKey,
	}
}

type controlRateLimitBucketCandidate struct {
	key      string
	lastSeen time.Time
}

func (l *controlEndpointRateLimiter) evictExpiredBucketsLocked(now time.Time) {
	if l == nil || len(l.buckets) == 0 || l.bucketTTL <= 0 {
		return
	}
	for key, bucket := range l.buckets {
		if now.Sub(bucket.lastSeen) < l.bucketTTL {
			continue
		}
		delete(l.buckets, key)
	}
}

func (l *controlEndpointRateLimiter) evictOldestBucketsLocked(count int) {
	if l == nil || len(l.buckets) == 0 || count <= 0 {
		return
	}
	candidates := make([]controlRateLimitBucketCandidate, 0, len(l.buckets))
	for key, bucket := range l.buckets {
		candidates = append(candidates, controlRateLimitBucketCandidate{
			key:      key,
			lastSeen: bucket.lastSeen,
		})
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].lastSeen.Equal(candidates[j].lastSeen) {
			return candidates[i].key < candidates[j].key
		}
		return candidates[i].lastSeen.Before(candidates[j].lastSeen)
	})
	if count > len(candidates) {
		count = len(candidates)
	}
	for i := 0; i < count; i++ {
		delete(l.buckets, candidates[i].key)
	}
}

func (d controlEndpointRateDecision) retryAfterSeconds() int64 {
	seconds := ceilDurationSeconds(d.retryAfter)
	if seconds <= 0 {
		return 1
	}
	return seconds
}

func ceilDurationSeconds(duration time.Duration) int64 {
	if duration <= 0 {
		return 0
	}
	return int64((duration + time.Second - 1) / time.Second)
}

func (s *Server) enforceControlRateLimit(
	writer http.ResponseWriter,
	request *http.Request,
	correlationID string,
	endpointKey string,
	explicitScopeType string,
	explicitScopeKey string,
) bool {
	if s == nil || s.controlRateLimiter == nil {
		return true
	}
	_ = explicitScopeType
	_ = explicitScopeKey

	if headers := unsupportedTrustedRateLimitHeaders(request); len(headers) > 0 {
		writeJSONErrorWithDetails(writer, http.StatusBadRequest, "trusted rate-limit scope headers are not supported", correlationID, map[string]any{
			"category":               "rate_limit",
			"unsupported_headers":    headers,
			"supported_scope_source": "token_fingerprint",
		})
		return false
	}

	scopeType := controlRateLimitScopeTypeToken
	scopeKey := controlRateLimitTokenFingerprint(request)
	scopeSource := "token_fingerprint"
	if strings.TrimSpace(scopeKey) == "" {
		scopeType = controlRateLimitScopeTypeGlobal
		scopeKey = controlRateLimitDefaultScopeKey
		scopeSource = "global_fallback"
	}
	decision := s.controlRateLimiter.allow(endpointKey, scopeType, scopeKey)
	if decision.allowed {
		return true
	}

	retryAfterSeconds := decision.retryAfterSeconds()
	writer.Header().Set("Retry-After", fmt.Sprintf("%d", retryAfterSeconds))
	writeJSONErrorWithDetails(writer, http.StatusTooManyRequests, "control endpoint rate limit exceeded", correlationID, map[string]any{
		"category":            "rate_limit",
		"endpoint":            strings.TrimSpace(endpointKey),
		"scope_type":          decision.scopeType,
		"scope_key":           decision.scopeKey,
		"scope_source":        scopeSource,
		"bucket_key":          decision.bucketKey,
		"limit":               decision.limit,
		"remaining":           decision.remaining,
		"window_seconds":      ceilDurationSeconds(decision.window),
		"reset_at":            decision.resetAt.UTC().Format(time.RFC3339Nano),
		"retry_after_seconds": retryAfterSeconds,
	})
	return false
}

func unsupportedTrustedRateLimitHeaders(request *http.Request) []string {
	if request == nil {
		return nil
	}
	headers := make([]string, 0, 2)
	if strings.TrimSpace(request.Header.Get(controlRateLimitTrustedActorHeader)) != "" {
		headers = append(headers, controlRateLimitTrustedActorHeader)
	}
	if strings.TrimSpace(request.Header.Get(controlRateLimitTrustedWorkspaceHeader)) != "" {
		headers = append(headers, controlRateLimitTrustedWorkspaceHeader)
	}
	return headers
}

func normalizeControlRateLimitScope(scopeType string, scopeKey string) (string, string) {
	key := strings.ToLower(strings.TrimSpace(scopeKey))
	if key == "" {
		return controlRateLimitScopeTypeGlobal, controlRateLimitDefaultScopeKey
	}
	typeValue := strings.ToLower(strings.TrimSpace(scopeType))
	if typeValue == "" {
		typeValue = "principal"
	}
	return typeValue, key
}

func controlRateLimitBucketKey(endpoint string, scopeType string, scopeKey string) string {
	return strings.TrimSpace(endpoint) + "|" + strings.TrimSpace(scopeType) + ":" + strings.TrimSpace(scopeKey)
}

func controlRateLimitTokenFingerprint(request *http.Request) string {
	token := bearerTokenValueFromRequest(request)
	if token == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:8])
}

func bearerTokenValueFromRequest(request *http.Request) string {
	if request == nil {
		return ""
	}
	authorization := strings.TrimSpace(request.Header.Get("Authorization"))
	if authorization == "" {
		return ""
	}
	parts := strings.Fields(authorization)
	if len(parts) != 2 {
		return ""
	}
	if !strings.EqualFold(strings.TrimSpace(parts[0]), "bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func controlRateLimitScopeFromActorOrWorkspace(actorID string, workspaceID string) (string, string) {
	actor := strings.TrimSpace(actorID)
	if actor != "" {
		return controlRateLimitScopeTypeActor, actor
	}
	workspace := strings.TrimSpace(workspaceID)
	if workspace != "" {
		return controlRateLimitScopeTypeWorkspace, workspace
	}
	return "", ""
}

func firstNonEmptyTrimmed(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
