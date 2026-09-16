package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"tunnel-manager/models"
	"tunnel-manager/services"
	"tunnel-manager/store"
)

// Throttle is the sign-in rate limiter, shared by every handler that can be
// used to guess a password.
//
// Two key spaces are charged per attempt: the account and the client address.
// The account budget is the one that actually stops a guessing attack, because
// it cannot be shed by changing address; the address budget stops a single
// machine working through a list of accounts. Whichever runs out first ends the
// attempt.
type Throttle struct {
	limiter *services.RateLimiter
	store   *store.Store
	// notify tells the operators that an account just spent its budget. Wired
	// by main so this package stays out of the notification plumbing.
	notify func(userID, username, ip string, retryAfter time.Duration)
}

// NewThrottle creates the limiter. Counters start empty, so a restart clears
// every lockout — a deliberate trade in favour of never leaving the panel
// locked out by state nobody can inspect.
func NewThrottle(st *store.Store) *Throttle {
	return &Throttle{limiter: services.NewRateLimiter(), store: st}
}

// SetNotifier wires the lockout notification.
func (t *Throttle) SetNotifier(fn func(userID, username, ip string, retryAfter time.Duration)) {
	if t != nil {
		t.notify = fn
	}
}

// Guard reports whether an attempt may proceed and, when it may not, how long
// the client should wait.
//
// userID may be empty: the account budget is still charged for names that do
// not exist, and only the familiar-network relaxation needs a real account.
func (t *Throttle) Guard(r *http.Request, account, userID string) (bool, time.Duration) {
	if t == nil || t.limiter == nil {
		return true, 0
	}
	limits := t.store.GetAppSettings().RateLimits()
	if !limits.Enabled {
		return true, 0
	}
	window := time.Duration(limits.WindowMinutes) * time.Minute
	ip := clientIP(r)

	budgets := []rateBudget{{ipKey(ip), limits.PerIP}}
	// An empty account means the caller has not named one yet — the passkey
	// ceremony does not know who it is talking to until it finishes. Charging
	// a shared "" key would throttle every anonymous caller as one.
	if account != "" {
		budgets = append(budgets, rateBudget{accountKey(account), t.accountLimit(limits, userID, ip)})
	}

	var wait time.Duration
	for _, budget := range budgets {
		if ok, w := t.limiter.Allow(budget.key, budget.limit, window); !ok && w > wait {
			wait = w
		}
	}
	return wait == 0, wait
}

// Fail charges one failed attempt against both budgets.
//
// It reports true only on the attempt that actually spent the account budget,
// so a run of blocked retries notifies the operator once rather than on every
// request.
func (t *Throttle) Fail(r *http.Request, account, userID string) (bool, time.Duration) {
	if t == nil || t.limiter == nil {
		return false, 0
	}
	limits := t.store.GetAppSettings().RateLimits()
	if !limits.Enabled {
		return false, 0
	}
	window := time.Duration(limits.WindowMinutes) * time.Minute
	ip := clientIP(r)
	key := accountKey(account)
	limit := t.accountLimit(limits, userID, ip)

	t.limiter.Fail(ipKey(ip), window)
	if account == "" {
		return false, 0
	}

	wasOpen, _ := t.limiter.Allow(key, limit, window)
	t.limiter.Fail(key, window)
	nowBlocked, wait := t.limiter.Allow(key, limit, window)
	return wasOpen && nowBlocked, wait
}

// Clear forgets the account budget after a successful sign-in, and records the
// network so a later attempt from the same place is recognised.
//
// The address budget is deliberately left alone. Resetting it here would let
// anyone holding one valid account wipe the shared budget between rounds of
// guessing at somebody else's; it ages out on its own instead.
func (t *Throttle) Clear(r *http.Request, account, userID string) {
	if t == nil || t.limiter == nil {
		return
	}
	t.limiter.Reset(accountKey(account))
	if userID == "" {
		return
	}
	if err := t.store.RememberFamiliarIP(userID, clientIP(r)); err != nil {
		log.Printf("remember familiar network: %v", err)
	}
}

// accountLimit is the account budget for this attempt, widened when the address
// belongs to a network the account has signed in from before. The relaxation is
// a multiplier rather than a bypass: someone who reaches a familiar network
// still runs out, just later.
func (t *Throttle) accountLimit(limits models.RateLimits, userID, ip string) int {
	limit := limits.PerAccount
	if limit <= 0 || userID == "" || limits.FamiliarMultiplier <= 1 {
		return limit
	}
	if t.store.IsFamiliarIP(userID, ip) {
		return limit * limits.FamiliarMultiplier
	}
	return limit
}

// rateBudget is one key space and the budget it is allowed.
type rateBudget struct {
	key   string
	limit int
}

// accountKey is case-insensitive, matching how accounts are looked up.
func accountKey(account string) string {
	return "acct:" + strings.ToLower(strings.TrimSpace(account))
}

func ipKey(ip string) string {
	return "ip:" + ip
}

// writeTooManyAttempts ends a request with the standard 429 shape: a
// Retry-After header for anything that reads headers, and the same number in
// the body for the login form.
func writeTooManyAttempts(w http.ResponseWriter, retryAfter time.Duration) {
	seconds := int(retryAfter.Seconds() + 0.5)
	if seconds < 1 {
		seconds = 1
	}
	w.Header().Set("Retry-After", strconv.Itoa(seconds))
	writeJSON(w, http.StatusTooManyRequests, map[string]interface{}{
		"error":       fmt.Sprintf("尝试过于频繁，请在 %s 后重试", humanWait(seconds)),
		"retry_after": seconds,
	})
}

// humanWait phrases a wait the way a person would say it.
func humanWait(seconds int) string {
	if seconds < 60 {
		return strconv.Itoa(seconds) + " 秒"
	}
	return strconv.Itoa((seconds+59)/60) + " 分钟"
}
