package services

import (
	"sync"
	"time"
)

// maxRateLimitKeys bounds how many keys the limiter tracks. An attempt against
// an invented username mints a key of its own, so without a cap an attacker
// could grow the map without bound.
const maxRateLimitKeys = 4096

// maxFailuresPerKey bounds one key's history. Past the budget the exact count
// stops mattering, and a sliding window only ever consults the oldest entries.
const maxFailuresPerKey = 512

// RateLimiter counts recent authentication failures per key and reports when a
// key has spent its budget.
//
// Counters live in memory on purpose. A restart clearing them is a smaller
// problem than a lockout table that outlives the process, and the window is
// minutes rather than days.
type RateLimiter struct {
	mu      sync.Mutex
	entries map[string]*rateEntry
	now     func() time.Time
}

type rateEntry struct {
	// failures holds unix seconds, oldest first.
	failures []int64
	// lastSeen orders eviction once the map is full.
	lastSeen int64
}

// NewRateLimiter creates an empty limiter.
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{entries: map[string]*rateEntry{}, now: time.Now}
}

// Allow reports whether key still has budget and, when it does not, how long
// until the oldest failure ages out of the window.
//
// A limit of zero or less disables the check for that key space, which is how
// the per-account and per-address budgets are turned off independently.
func (l *RateLimiter) Allow(key string, limit int, window time.Duration) (bool, time.Duration) {
	if l == nil || limit <= 0 || window <= 0 {
		return true, 0
	}
	seconds := int64(window / time.Second)
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now().Unix()
	entry := l.entries[key]
	if entry == nil {
		return true, 0
	}
	entry.lastSeen = now
	entry.failures = pruneFailures(entry.failures, now-seconds)
	if len(entry.failures) < limit {
		return true, 0
	}
	// The budget refills when the oldest failure in the window falls out of it.
	wait := time.Duration(entry.failures[0]+seconds-now) * time.Second
	if wait <= 0 {
		wait = time.Second
	}
	return false, wait
}

// Fail records one failed attempt against key.
func (l *RateLimiter) Fail(key string, window time.Duration) {
	if l == nil || window <= 0 {
		return
	}
	seconds := int64(window / time.Second)
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now().Unix()
	entry := l.entries[key]
	if entry == nil {
		l.evictOldestLocked()
		entry = &rateEntry{}
		l.entries[key] = entry
	}
	entry.lastSeen = now
	entry.failures = append(pruneFailures(entry.failures, now-seconds), now)
	if len(entry.failures) > maxFailuresPerKey {
		entry.failures = entry.failures[len(entry.failures)-maxFailuresPerKey:]
	}
}

// Reset forgets a key, which is what a successful sign-in earns.
func (l *RateLimiter) Reset(key string) {
	if l == nil {
		return
	}
	l.mu.Lock()
	delete(l.entries, key)
	l.mu.Unlock()
}

// Len reports how many keys are tracked, for tests and diagnostics.
func (l *RateLimiter) Len() int {
	if l == nil {
		return 0
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.entries)
}

// evictOldestLocked keeps the map at its cap by dropping the least recently
// touched key. Called with the lock held, and only on insert at the cap, so the
// scan cost is paid once per new key rather than per attempt.
func (l *RateLimiter) evictOldestLocked() {
	if len(l.entries) < maxRateLimitKeys {
		return
	}
	var oldestKey string
	var oldestAt int64
	for key, entry := range l.entries {
		if oldestKey == "" || entry.lastSeen < oldestAt {
			oldestKey, oldestAt = key, entry.lastSeen
		}
	}
	delete(l.entries, oldestKey)
}

// pruneFailures drops timestamps at or before cutoff, keeping the slice ordered
// so the caller can read the oldest surviving failure off the front.
func pruneFailures(failures []int64, cutoff int64) []int64 {
	first := 0
	for first < len(failures) && failures[first] <= cutoff {
		first++
	}
	if first == 0 {
		return failures
	}
	return failures[first:]
}
