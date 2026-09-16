package services

import (
	"fmt"
	"testing"
	"time"
)

// newTestLimiter returns a limiter on a clock the test drives by hand, so the
// window arithmetic is checked rather than waited out.
func newTestLimiter() (*RateLimiter, func(time.Duration)) {
	base := time.Unix(1_800_000_000, 0)
	l := NewRateLimiter()
	l.now = func() time.Time { return base }
	return l, func(d time.Duration) { base = base.Add(d) }
}

func TestRateLimiterAllowsUntilTheBudgetIsSpent(t *testing.T) {
	l, _ := newTestLimiter()
	window := 15 * time.Minute

	if ok, _ := l.Allow("acct:a", 5, window); !ok {
		t.Fatal("a key with no failures was refused")
	}
	for i := 0; i < 4; i++ {
		l.Fail("acct:a", window)
	}
	if ok, _ := l.Allow("acct:a", 5, window); !ok {
		t.Fatal("four failures out of five were refused")
	}
	l.Fail("acct:a", window)
	if ok, _ := l.Allow("acct:a", 5, window); ok {
		t.Fatal("the fifth failure left the budget open")
	}
}

func TestRateLimiterReportsWhenTheBudgetRefills(t *testing.T) {
	l, advance := newTestLimiter()
	window := 15 * time.Minute
	for i := 0; i < 5; i++ {
		l.Fail("acct:a", window)
	}

	advance(14 * time.Minute)
	ok, wait := l.Allow("acct:a", 5, window)
	if ok {
		t.Fatal("the budget reopened before the window elapsed")
	}
	// All five failures share the same instant, so the block lifts one minute
	// after the last check.
	if wait != time.Minute {
		t.Fatalf("wait = %v, want 1m", wait)
	}

	advance(2 * time.Minute)
	if ok, _ := l.Allow("acct:a", 5, window); !ok {
		t.Fatal("the budget did not reopen once the window elapsed")
	}
}

// A fixed window would clear the whole count on the boundary; a sliding one
// drops failures individually, which is what stops an attacker from parking on
// the boundary and getting a full fresh budget.
func TestRateLimiterSlidesRatherThanResetting(t *testing.T) {
	l, advance := newTestLimiter()
	window := 15 * time.Minute
	for i := 0; i < 5; i++ {
		l.Fail("acct:a", window)
		advance(time.Minute)
	}
	if ok, _ := l.Allow("acct:a", 5, window); ok {
		t.Fatal("five failures inside the window were allowed")
	}

	// One second past the oldest failure: only that one has aged out.
	advance(10*time.Minute + time.Second)
	if ok, _ := l.Allow("acct:a", 5, window); !ok {
		t.Fatal("the oldest failure should have aged out of the window")
	}
}

func TestRateLimiterResetClearsTheKey(t *testing.T) {
	l, _ := newTestLimiter()
	window := 15 * time.Minute
	for i := 0; i < 5; i++ {
		l.Fail("acct:a", window)
	}
	l.Reset("acct:a")
	if ok, _ := l.Allow("acct:a", 5, window); !ok {
		t.Fatal("a successful sign-in did not clear the budget")
	}
	if l.Len() != 0 {
		t.Fatalf("tracked %d keys after a reset, want none", l.Len())
	}
}

// Each key space is budgeted on its own, which is what lets the account and
// address limits be tuned separately.
func TestRateLimiterKeepsKeysIndependent(t *testing.T) {
	l, _ := newTestLimiter()
	window := 15 * time.Minute
	for i := 0; i < 5; i++ {
		l.Fail("acct:a", window)
	}
	if ok, _ := l.Allow("acct:b", 5, window); !ok {
		t.Fatal("one account's failures leaked into another's budget")
	}
	if ok, _ := l.Allow("ip:1.2.3.4", 5, window); !ok {
		t.Fatal("an account's failures leaked into the address budget")
	}
}

// Zero disables a key space outright, which is how the panel offers "no limit"
// for one dimension without turning the whole feature off.
func TestRateLimiterZeroLimitDisablesTheCheck(t *testing.T) {
	l, _ := newTestLimiter()
	window := 15 * time.Minute
	for i := 0; i < 50; i++ {
		l.Fail("acct:a", window)
	}
	if ok, _ := l.Allow("acct:a", 0, window); !ok {
		t.Fatal("a zero limit still refused an attempt")
	}
	var nilLimiter *RateLimiter
	if ok, _ := nilLimiter.Allow("acct:a", 1, window); !ok {
		t.Fatal("a nil limiter refused an attempt")
	}
}

// Invented usernames each mint a key, so the map has to stay bounded or a
// spray of random names would grow it without limit.
func TestRateLimiterBoundsTrackedKeys(t *testing.T) {
	l, _ := newTestLimiter()
	window := 15 * time.Minute
	for i := 0; i < maxRateLimitKeys+200; i++ {
		l.Fail(fmt.Sprintf("acct:user-%d", i), window)
	}
	if l.Len() > maxRateLimitKeys {
		t.Fatalf("tracked %d keys, want at most %d", l.Len(), maxRateLimitKeys)
	}
}
