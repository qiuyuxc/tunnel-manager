package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"tunnel-manager/models"
	"tunnel-manager/services"
	"tunnel-manager/store"
)

type alertFeed struct {
	Cursor int64           `json:"cursor"`
	Now    int64           `json:"now"`
	Alerts []alertFeedItem `json:"alerts"`
}

func newAlertsFeedFixture(t *testing.T) (*store.Store, *MonitorsHandler) {
	t.Helper()
	st := newTestStore(t)
	heartbeats := services.NewHeartbeatLog(filepath.Join(t.TempDir(), "heartbeats.json"))
	return st, NewMonitorsHandler(st, heartbeats, nil, nil)
}

func fetchAlerts(t *testing.T, h *MonitorsHandler, user models.SessionUser, since int64) alertFeed {
	t.Helper()
	req := withUser(httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/alerts?since=%d", since), nil), user)
	resp := httptest.NewRecorder()
	h.AlertsFeed(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("AlertsFeed() status = %d, want %d", resp.Code, http.StatusOK)
	}
	var feed alertFeed
	decodeResponse(t, resp, &feed)
	return feed
}

func TestAlertsFeedFirstPollSeedsCursorWithoutReplaying(t *testing.T) {
	st, h := newAlertsFeedFixture(t)
	if err := st.AddMonitor(models.Monitor{ID: "mon-a", UserID: st.AdminUserID(), Name: "Alpha"}); err != nil {
		t.Fatal(err)
	}
	st.AddAlertLog(models.AlertLog{MonitorID: "mon-a", TargetID: "t1", TargetName: "Site", State: "down"})

	feed := fetchAlerts(t, h, models.SessionUser{ID: st.AdminUserID(), Role: models.RoleAdmin}, 0)

	if feed.Cursor == 0 {
		t.Fatal("AlertsFeed() cursor = 0, want the server clock so the client starts from now")
	}
	if len(feed.Alerts) != 0 {
		t.Fatalf("AlertsFeed() replayed %d historical alert(s) on a seeding poll", len(feed.Alerts))
	}
}

func TestAlertsFeedReturnsAlertsWithMonitorName(t *testing.T) {
	st, h := newAlertsFeedFixture(t)
	if err := st.AddMonitor(models.Monitor{ID: "mon-a", UserID: st.AdminUserID(), Name: "Alpha"}); err != nil {
		t.Fatal(err)
	}
	admin := models.SessionUser{ID: st.AdminUserID(), Role: models.RoleAdmin}
	seed := fetchAlerts(t, h, admin, 0)

	st.AddAlertLog(models.AlertLog{
		MonitorID: "mon-a", TargetID: "t1", TargetName: "Site",
		State: "down", HTTPCode: 502, Error: "bad gateway", Notified: false,
	})

	feed := fetchAlerts(t, h, admin, seed.Cursor-1)

	if len(feed.Alerts) != 1 {
		t.Fatalf("AlertsFeed() returned %d alert(s), want 1", len(feed.Alerts))
	}
	got := feed.Alerts[0]
	if got.MonitorName != "Alpha" {
		t.Errorf("monitor_name = %q, want %q", got.MonitorName, "Alpha")
	}
	if got.State != "down" || got.HTTPCode != 502 || got.TargetName != "Site" {
		t.Errorf("alert = %+v, want the recorded down transition", got)
	}
	if feed.Cursor != got.CreatedAt {
		t.Errorf("cursor = %d, want the newest alert's timestamp %d", feed.Cursor, got.CreatedAt)
	}
}

func TestAlertsFeedScopesToMonitorsTheSessionCanSee(t *testing.T) {
	st, h := newAlertsFeedFixture(t)
	if err := st.AddMonitor(models.Monitor{ID: "mon-a", UserID: "someone-else", Name: "Not mine"}); err != nil {
		t.Fatal(err)
	}
	admin := models.SessionUser{ID: st.AdminUserID(), Role: models.RoleAdmin}
	seed := fetchAlerts(t, h, admin, 0)
	st.AddAlertLog(models.AlertLog{MonitorID: "mon-a", TargetID: "t1", TargetName: "Site", State: "down"})

	feed := fetchAlerts(t, h, models.SessionUser{ID: "other-user", Role: models.RoleUser}, seed.Cursor-1)

	if len(feed.Alerts) != 0 {
		t.Fatalf("AlertsFeed() leaked %d alert(s) from another user's monitor", len(feed.Alerts))
	}
}
