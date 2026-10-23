package api

import (
	"testing"
	"time"
)

// Stale keys used to stay in the map forever unless the same key was hit again,
// so a spray of one-shot client IPs grew memory without bound.
func TestAttemptLimiterEvictsExpiredKeys(t *testing.T) {
	l := newAttemptLimiter(3, time.Millisecond)

	if !l.allow("attacker-1") || !l.allow("attacker-2") {
		t.Fatal("fresh keys should be allowed")
	}
	time.Sleep(5 * time.Millisecond)

	// Force the next call to trigger the periodic sweep without 512 calls.
	l.calls = sweepEvery - 1
	if !l.allow("attacker-3") {
		t.Fatal("third key should be allowed")
	}

	l.mu.Lock()
	_, keptA := l.attempts["attacker-1"]
	_, keptB := l.attempts["attacker-2"]
	_, keptC := l.attempts["attacker-3"]
	l.mu.Unlock()

	if keptA || keptB {
		t.Fatalf("expired keys were not evicted: %v", l.attempts)
	}
	if !keptC {
		t.Fatal("live key was evicted")
	}
}

func TestAttemptLimiterBlocksWithinWindow(t *testing.T) {
	l := newAttemptLimiter(2, time.Minute)
	if !l.allow("ip") || !l.allow("ip") {
		t.Fatal("first two attempts should be allowed")
	}
	if l.allow("ip") {
		t.Fatal("third attempt within the window should be blocked")
	}
	l.reset("ip")
	if !l.allow("ip") {
		t.Fatal("reset should clear the key")
	}
}
