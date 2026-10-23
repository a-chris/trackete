package api

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// attemptLimiter is a tiny in-memory throttle for auth endpoints. It is not
// meant to replace a reverse-proxy rate limiter; it just makes password
// guessing boring on a small self-hosted instance.
// sweepEvery is how many allow() calls pass between full map sweeps. Pruning
// only on the used key leaves stale keys (one per attacker IP) behind forever.
const sweepEvery = 512

type attemptLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
	limit    int
	window   time.Duration
	calls    int
}

func newAttemptLimiter(limit int, window time.Duration) *attemptLimiter {
	return &attemptLimiter{attempts: map[string][]time.Time{}, limit: limit, window: window}
}

func (l *attemptLimiter) allow(key string) bool {
	now := time.Now()
	cutoff := now.Add(-l.window)

	l.mu.Lock()
	defer l.mu.Unlock()

	l.calls++
	if l.calls%sweepEvery == 0 {
		l.evictLocked(cutoff)
	}

	hits := l.attempts[key]
	kept := hits[:0]
	for _, t := range hits {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= l.limit {
		l.attempts[key] = kept
		return false
	}
	l.attempts[key] = append(kept, now)
	return true
}

// evictLocked drops every key whose attempts have all expired. The caller must
// hold l.mu.
func (l *attemptLimiter) evictLocked(cutoff time.Time) {
	for key, hits := range l.attempts {
		live := hits[:0]
		for _, t := range hits {
			if t.After(cutoff) {
				live = append(live, t)
			}
		}
		if len(live) == 0 {
			delete(l.attempts, key)
			continue
		}
		l.attempts[key] = live
	}
}

func (l *attemptLimiter) reset(key string) {
	l.mu.Lock()
	delete(l.attempts, key)
	l.mu.Unlock()
}

// guard wraps a handler so failed attempts are rate limited per client IP.
func (l *attemptLimiter) guard(handler gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.ClientIP()
		if !l.allow(key) {
			c.Header("Retry-After", "300")
			fail(c, http.StatusTooManyRequests, "too many attempts, please wait a few minutes")
			return
		}
		handler(c)
	}
}
