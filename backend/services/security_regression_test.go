package services

import (
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestPasskeyCeremonyCapacityIsSharedFairly(t *testing.T) {
	svc, _ := newPasskeyTestService(t)
	request := httptest.NewRequest("POST", "https://panel.example.com/api/auth/passkey/login/begin", nil)
	request.RemoteAddr = "198.51.100.1:1234"
	for i := 0; i < 8; i++ {
		if _, _, _, err := svc.BeginDiscoverableLogin(request); err != nil {
			t.Fatal(err)
		}
	}
	request.RemoteAddr = "198.51.100.1:5678"
	request.Header.Set("X-Forwarded-For", "203.0.113.5")
	if _, _, _, err := svc.BeginDiscoverableLogin(request); err == nil {
		t.Fatal("source can exhaust shared ceremony capacity")
	}
	request.RemoteAddr = "198.51.100.2:5678"
	if _, _, _, err := svc.BeginDiscoverableLogin(request); err != nil {
		t.Fatalf("other source blocked: %v", err)
	}
}

func TestBeginBudgetIsAtomicAndBounded(t *testing.T) {
	limiter := NewRateLimiter()
	var accepted atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if ok, _ := limiter.Take("same-source", 3, time.Minute); ok {
				accepted.Add(1)
			}
		}()
	}
	wg.Wait()
	if accepted.Load() != 3 {
		t.Fatalf("concurrent admission: %d", accepted.Load())
	}
	for i := 0; i < 1000; i++ {
		if ok, _ := limiter.Take("large-budget", 1000, time.Minute); !ok {
			t.Fatalf("large budget stopped at %d", i)
		}
	}
	if ok, _ := limiter.Take("large-budget", 1000, time.Minute); ok {
		t.Fatal("large budget exceeded")
	}
	if len(limiter.entries["large-budget"].failures) != 0 {
		t.Fatal("request budget grows failure history")
	}
}
