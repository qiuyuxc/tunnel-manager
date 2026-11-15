package handlers

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"tunnel-manager/models"
	"tunnel-manager/services"
)

func TestPublicHistoryKeepsRealWindowAndWorstState(t *testing.T) {
	end := int64(1800000000000)
	start := end - int64(24*time.Hour/time.Millisecond)
	step := int64(publicBucketSec * 1000)
	bars, observed := publicHistory([]services.Heartbeat{
		{T: start - 1, S: "down"},
		{T: start, S: "ok", M: 10},
		{T: start + 1, S: "down"},
		{T: start + 2, S: "ok", M: 30},
		{T: start + step, S: "warn", M: 50},
		{T: end, S: "ok", M: 70},
		{T: end + 1, S: "down"},
	}, start, end)
	if len(bars) != 48 || len(observed) != 5 {
		t.Fatalf("bars=%d observed=%d, want 48 and 5", len(bars), len(observed))
	}
	if bars[0].S != "down" || bars[0].Samples != 3 || bars[0].OK != 2 || bars[0].Down != 1 || bars[0].M != 20 {
		t.Fatalf("brief outage was lost or counts/latency incorrect: %#v", bars[0])
	}
	if bars[1].S != "warn" || bars[1].Warn != 1 || bars[47].S != "ok" {
		t.Fatalf("bucket boundaries incorrect: %#v %#v", bars[1], bars[47])
	}
	for i, bar := range bars {
		if bar.T != start+int64(i)*step {
			t.Fatalf("bar %d timestamp=%d does not follow 30-minute spacing", i, bar.T)
		}
		if i > 1 && i < 47 && (bar.S != "unknown" || bar.Samples != 0) {
			t.Fatalf("empty history fabricated results in bar %d: %#v", i, bar)
		}
	}
	if bars[47].T+step != end {
		t.Fatal("48 bars do not span exactly 24 hours")
	}
}

func TestPublicStatusShortHistoryStillSpans24Hours(t *testing.T) {
	st := newTestStore(t)
	hb := services.NewHeartbeatLog(filepath.Join(t.TempDir(), "heartbeats.json"))
	monitor := models.Monitor{ID: "short", UserID: st.AdminUserID(), Name: "public", PublishEnabled: true, PublicToken: "short-history", IntervalSec: 60, Targets: []models.MonitorTarget{{ID: "recorded", Name: "Recorded"}, {ID: "empty", Name: "Empty"}}}
	if err := st.AddMonitor(monitor); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UnixMilli()
	hb.Append(monitor.ID, "recorded", services.Heartbeat{T: now - int64(25*time.Hour/time.Millisecond), S: "down"})
	for i := 40; i >= 0; i-- {
		hb.Append(monitor.ID, "recorded", services.Heartbeat{T: now - int64(i)*60000, S: "ok", M: 25})
	}
	h := NewMonitorsHandler(st, hb, nil, nil)
	router := chi.NewRouter()
	router.Get("/api/public/status/{token}", h.PublicStatus)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/public/status/short-history", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d: %s", response.Code, response.Body.String())
	}
	var payload publicStatus
	decodeResponse(t, response, &payload)
	if payload.WindowEnd-payload.WindowStart != int64(24*time.Hour/time.Millisecond) || payload.BucketSec != 1800 {
		t.Fatalf("wrong advertised range: %#v", payload)
	}
	recorded, empty := payload.Targets[0], payload.Targets[1]
	if len(recorded.Bars) != 48 || recorded.ObservedCount != 41 || recorded.Uptime24h != 100 {
		t.Fatalf("short history semantics incorrect: %#v", recorded)
	}
	unknown, checks := 0, 0
	for _, bar := range recorded.Bars {
		checks += bar.Samples
		if bar.S == "unknown" {
			unknown++
		}
	}
	if unknown < 46 || checks != 41 {
		t.Fatalf("short history filled unavailable windows: unknown=%d checks=%d", unknown, checks)
	}
	if empty.State != "unknown" || empty.ObservedCount != 0 || len(empty.Bars) != 48 {
		t.Fatalf("unobserved service reported availability: %#v", empty)
	}
	for _, bar := range empty.Bars {
		if bar.S != "unknown" || bar.Samples != 0 {
			t.Fatalf("fabricated empty service result: %#v", bar)
		}
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("public status lacks no-store")
	}
}

func TestPublicStatusStaleChecksCannotReportHealthy(t *testing.T) {
	st := newTestStore(t)
	hb := services.NewHeartbeatLog(filepath.Join(t.TempDir(), "heartbeats.json"))
	monitor := models.Monitor{ID: "stale", UserID: st.AdminUserID(), Name: "stale", PublishEnabled: true, PublicToken: "stale-token", IntervalSec: 60, Targets: []models.MonitorTarget{{ID: "target", Name: "Target"}}}
	if err := st.AddMonitor(monitor); err != nil {
		t.Fatal(err)
	}
	timestamp := time.Now().Add(-10 * time.Minute).UnixMilli()
	hb.Append(monitor.ID, "target", services.Heartbeat{T: timestamp, S: "ok", M: 25})
	h := NewMonitorsHandler(st, hb, nil, nil)
	router := chi.NewRouter()
	router.Get("/api/public/status/{token}", h.PublicStatus)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/public/status/stale-token", nil))
	var payload publicStatus
	decodeResponse(t, response, &payload)
	if payload.Targets[0].State != "unknown" || payload.Targets[0].ObservedCount != 1 || payload.UpdatedAt != timestamp {
		t.Fatalf("stopped checks appear live: %#v", payload)
	}
}

func TestPublicLinkRejectsExecutableSchemesAndStripsCredentials(t *testing.T) {
	tests := []struct{ input, want string }{
		{"javascript://example.com/payload", ""},
		{"data://example.com/payload", ""},
		{"https://user:password@example.com/path", "https://example.com/path"},
		{"example.com/path", "https://example.com/path"},
	}
	for _, tc := range tests {
		got := linkURL(models.MonitorTarget{Type: "http", LinkEnabled: true, URL: tc.input})
		if got != tc.want {
			t.Errorf("linkURL(%q)=%q, want %q", tc.input, got, tc.want)
		}
	}
	if got := linkURL(models.MonitorTarget{Type: "http", LinkEnabled: false, URL: "https://example.com/?secret=hidden"}); got != "" {
		t.Fatal("private target exposed a public link")
	}
}

func TestMonitorSummaryListKeepsDetailHistoryCompatible(t *testing.T) {
	st := newTestStore(t)
	hb := services.NewHeartbeatLog(filepath.Join(t.TempDir(), "heartbeats.json"))
	monitor := models.Monitor{ID: "summary", UserID: st.AdminUserID(), Name: "Summary", Targets: []models.MonitorTarget{{ID: "target", Name: "Target"}}}
	if err := st.AddMonitor(monitor); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UnixMilli()
	hb.Append(monitor.ID, "target", services.Heartbeat{T: now - int64(25*time.Hour/time.Millisecond), S: "down"})
	hb.Append(monitor.ID, "target", services.Heartbeat{T: now, S: "ok", M: 33})
	h := NewMonitorsHandler(st, hb, nil, nil)
	for _, summary := range []bool{false, true} {
		path := "/api/monitors"
		if summary {
			path += "?summary=1"
		}
		request := withUser(httptest.NewRequest(http.MethodGet, path, nil), models.SessionUser{ID: st.AdminUserID(), Role: models.RoleAdmin})
		response := httptest.NewRecorder()
		h.List(response, request)
		var payload []monitorView
		decodeResponse(t, response, &payload)
		target := payload[0].Targets[0]
		if target.State != "ok" || target.Uptime24h != 100 || target.LatencyMs != 33 {
			t.Fatalf("summary=%v lost current stats: %#v", summary, target)
		}
		if summary && len(target.Bars) != 0 {
			t.Fatalf("summary includes unneeded history: %#v", target.Bars)
		}
		if !summary && len(target.Bars) != 2 {
			t.Fatalf("default list lost native preview history: %#v", target.Bars)
		}
	}
}
