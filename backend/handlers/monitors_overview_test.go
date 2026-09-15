package handlers

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"tunnel-manager/models"
	"tunnel-manager/services"
	"tunnel-manager/store"
)

// The dashboard chart is a week of calendar days: seven buckets, gaps included,
// with today's bucket last. Before the window grew the API skipped empty hours,
// which is what made the chart draw one oversized bar.
func TestOverviewReturnsSevenDailyBuckets(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	st := store.NewStore(path)
	if err := st.AddMonitor(models.Monitor{
		ID:     "monitor-1",
		UserID: st.AdminUserID(),
		Name:   "Home lab",
		Targets: []models.MonitorTarget{
			{ID: "target-1", Name: "site", URL: "https://example.com"},
		},
	}); err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, time.Local)
	heartbeats := services.NewHeartbeatLog(filepath.Join(t.TempDir(), "heartbeats.json"))
	heartbeats.Append("monitor-1", "target-1", services.Heartbeat{
		T: today.AddDate(0, 0, -1).UnixMilli(), S: "ok", M: 120,
	})
	heartbeats.Append("monitor-1", "target-1", services.Heartbeat{
		T: today.UnixMilli(), S: "warn", M: 80,
	})

	handler := NewMonitorsHandler(st, heartbeats, nil, nil)
	req := withUser(httptest.NewRequest(http.MethodGet, "/api/monitors/overview", nil), models.SessionUser{
		ID:   st.AdminUserID(),
		Role: models.RoleAdmin,
	})
	resp := httptest.NewRecorder()

	handler.Overview(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("Overview() status = %d, want %d", resp.Code, http.StatusOK)
	}
	var body struct {
		Uptime    float64 `json:"uptime"`
		Uptime24h float64 `json:"uptime_24h"`
		BucketSec int     `json:"bucket_sec"`
		Buckets   []struct {
			Hour   int64   `json:"hour"`
			AvgMs  float64 `json:"avg_ms"`
			PeakMs int64   `json:"peak_ms"`
			Total  int     `json:"total"`
			Warn   int     `json:"warn"`
			Down   int     `json:"down"`
		} `json:"buckets"`
	}
	decodeResponse(t, resp, &body)

	if len(body.Buckets) != overviewDays {
		t.Fatalf("got %d buckets, want %d", len(body.Buckets), overviewDays)
	}
	if body.BucketSec != overviewBucketSec {
		t.Fatalf("bucket_sec = %d, want %d", body.BucketSec, overviewBucketSec)
	}
	// Only "ok" counts as good: one of the two checks was a warn.
	if body.Uptime != 50 || body.Uptime24h != body.Uptime {
		t.Fatalf("uptime = %v / uptime_24h = %v, want 50 and the same value", body.Uptime, body.Uptime24h)
	}

	midnight := func(offsetDays int) int64 {
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local).
			AddDate(0, 0, offsetDays).Unix()
	}
	if last := body.Buckets[len(body.Buckets)-1].Hour; last != midnight(0) {
		t.Fatalf("last bucket starts at %d, want today's midnight %d", last, midnight(0))
	}
	if first := body.Buckets[0].Hour; first != midnight(-(overviewDays - 1)) {
		t.Fatalf("first bucket starts at %d, want %d", first, midnight(-(overviewDays - 1)))
	}

	// Every day in between is present and empty rather than missing.
	for i, bucket := range body.Buckets[:len(body.Buckets)-2] {
		if bucket.Total != 0 {
			t.Fatalf("bucket %d (hour %d) has %d checks, want none", i, bucket.Hour, bucket.Total)
		}
	}
	yesterday := body.Buckets[len(body.Buckets)-2]
	if yesterday.Total != 1 || yesterday.PeakMs != 120 || yesterday.AvgMs != 120 {
		t.Fatalf("yesterday bucket = %+v, want one 120ms check", yesterday)
	}
	todayBucket := body.Buckets[len(body.Buckets)-1]
	if todayBucket.Total != 1 || todayBucket.Warn != 1 || todayBucket.PeakMs != 80 {
		t.Fatalf("today bucket = %+v, want one warn check at 80ms", todayBucket)
	}
}
