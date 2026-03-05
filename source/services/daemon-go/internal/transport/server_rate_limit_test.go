package transport

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestUnsupportedTrustedRateLimitHeaders(t *testing.T) {
	request := httptest.NewRequest("POST", "http://localhost/v1/chat/turn", nil)
	request.Header.Set(controlRateLimitTrustedActorHeader, "actor.alpha")
	request.Header.Set(controlRateLimitTrustedWorkspaceHeader, "ws1")

	headers := unsupportedTrustedRateLimitHeaders(request)
	if len(headers) != 2 {
		t.Fatalf("expected both trusted headers to be flagged unsupported, got %v", headers)
	}

	requestWithoutHeaders := httptest.NewRequest("POST", "http://localhost/v1/chat/turn", nil)
	headers = unsupportedTrustedRateLimitHeaders(requestWithoutHeaders)
	if len(headers) != 0 {
		t.Fatalf("expected no unsupported trusted headers, got %v", headers)
	}
}

func TestControlEndpointRateLimiterBoundsBucketGrowthWithEvictionAndTTL(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	limiter := newControlEndpointRateLimiter(1, 100*time.Millisecond, func() time.Time {
		return now
	})
	if limiter == nil {
		t.Fatalf("expected limiter")
	}
	limiter.maxBuckets = 2
	limiter.bucketTTL = 250 * time.Millisecond

	allowAndAssert := func(scopeKey string) {
		t.Helper()
		decision := limiter.allow(controlRateLimitKeyChatTurn, controlRateLimitScopeTypeActor, scopeKey)
		if !decision.allowed {
			t.Fatalf("expected first request for scope %q to be allowed", scopeKey)
		}
	}

	allowAndAssert("actor.a")
	now = now.Add(1 * time.Millisecond)
	allowAndAssert("actor.b")
	now = now.Add(1 * time.Millisecond)
	allowAndAssert("actor.c")

	if got := len(limiter.buckets); got != 2 {
		t.Fatalf("expected limiter buckets bounded to 2, got %d", got)
	}
	if _, exists := limiter.buckets[controlRateLimitBucketKey(controlRateLimitKeyChatTurn, controlRateLimitScopeTypeActor, "actor.a")]; exists {
		t.Fatalf("expected oldest bucket actor.a to be evicted when max bucket bound is reached")
	}

	now = now.Add(500 * time.Millisecond)
	allowAndAssert("actor.d")
	if got := len(limiter.buckets); got != 1 {
		t.Fatalf("expected expired buckets to be evicted by TTL, got %d buckets", got)
	}
	if _, exists := limiter.buckets[controlRateLimitBucketKey(controlRateLimitKeyChatTurn, controlRateLimitScopeTypeActor, "actor.d")]; !exists {
		t.Fatalf("expected active actor.d bucket to remain after TTL eviction")
	}
}
