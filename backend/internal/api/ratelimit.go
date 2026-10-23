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
type attemptLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
	limit    int
	window   time.Duration
}

func newAttemptLimiter(limit int, window time.Duration) *attemptLimiter {
	return &attemptLimiter{attempts: map[string][]time.Time{}, limit: limit, window: window}
}

func (l *attemptLimiter) allow(key string) bool {
	now := time.Now()
	cutoff := now.Add(-l.window)

	l.mu.Lock()
	defer l.mu.Unlock()

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
