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

// ovIncident mirrors one outage window in the overview response.
type ovIncident struct {
	From  int64  `json:"from"`
	To    int64  `json:"to"`
	State string `json:"state"`
	Count int    `json:"count"`
	Code  int    `json:"code"`
	Error string `json:"error"`
}

// ovIssue mirrors one target's share of a bucket.
type ovIssue struct {
	MonitorID   string       `json:"monitor_id"`
	MonitorName string       `json:"monitor_name"`
	TargetID    string       `json:"target_id"`
	TargetName  string       `json:"target_name"`
	Total       int          `json:"total"`
	Warn        int          `json:"warn"`
	Down        int          `json:"down"`
	PeakMs      int64        `json:"peak_ms"`
	AvgMs       float64      `json:"avg_ms"`
	Incidents   []ovIncident `json:"incidents"`
	IncidentCnt int          `json:"incident_count"`
}

type ovIssuesBody struct {
	Buckets []struct {
		Hour   int64     `json:"hour"`
		Total  int       `json:"total"`
		Issues []ovIssue `json:"issues"`
	} `json:"buckets"`
}

// ovFixture builds a store holding one monitor with one target, plus an empty
// heartbeat log to append probes to.
func ovFixture(t *testing.T, monitorID, monitorName, targetID, targetName string) (*store.Store, *services.HeartbeatLog) {
	t.Helper()
	st := store.NewStore(filepath.Join(t.TempDir(), "config.json"))
	if err := st.AddMonitor(models.Monitor{
		ID:     monitorID,
		UserID: st.AdminUserID(),
		Name:   monitorName,
		Targets: []models.MonitorTarget{
			{ID: targetID, Name: targetName, URL: "https://example.com"},
		},
	}); err != nil {
		t.Fatal(err)
	}
	return st, services.NewHeartbeatLog(filepath.Join(t.TempDir(), "heartbeats.json"))
}

// ovAdminSession is the session the fixture's monitors belong to.
func ovAdminSession(st *store.Store) models.SessionUser {
	return models.SessionUser{ID: st.AdminUserID(), Role: models.RoleAdmin}
}

// ovToday is a clock time on the current calendar day, which is the bucket the
// overview always puts last.
func ovToday(hour, minute int) int64 {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, time.Local).UnixMilli()
}

// ovRunIssues calls the endpoint and returns today's issues.
func ovRunIssues(t *testing.T, st *store.Store, hb *services.HeartbeatLog, user models.SessionUser) []ovIssue {
	t.Helper()
	handler := NewMonitorsHandler(st, hb, nil, nil)
	req := withUser(httptest.NewRequest(http.MethodGet, "/api/monitors/overview", nil), user)
	resp := httptest.NewRecorder()
	handler.Overview(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("Overview() status = %d, want %d", resp.Code, http.StatusOK)
	}
	var body ovIssuesBody
	decodeResponse(t, resp, &body)
	if len(body.Buckets) != overviewDays {
		t.Fatalf("got %d buckets, want %d", len(body.Buckets), overviewDays)
	}
	return body.Buckets[len(body.Buckets)-1].Issues
}

// A day with nothing but ok probes says so by leaving the list empty: the bar
// already reads as healthy and there is nothing to drill into.
func TestOverviewIssuesOmittedWhenDayIsHealthy(t *testing.T) {
	st, hb := ovFixture(t, "monitor-1", "Home lab", "target-1", "site")
	hb.Append("monitor-1", "target-1", services.Heartbeat{T: ovToday(9, 0), S: "ok", M: 120})
	hb.Append("monitor-1", "target-1", services.Heartbeat{T: ovToday(9, 1), S: "ok", M: 130})

	if issues := ovRunIssues(t, st, hb, ovAdminSession(st)); len(issues) != 0 {
		t.Fatalf("healthy day returned %d issues, want none: %+v", len(issues), issues)
	}
}

// Consecutive failures are one outage window, and an ok probe between them
// starts a new one. This is the whole point of the field: the bar shows a bad
// day, this shows the two separate stretches that made it bad.
func TestOverviewIssuesGroupConsecutiveFailures(t *testing.T) {
	st, hb := ovFixture(t, "monitor-1", "Home lab", "target-1", "site")
	hb.Append("monitor-1", "target-1", services.Heartbeat{T: ovToday(9, 0), S: "ok", M: 100})
	hb.Append("monitor-1", "target-1", services.Heartbeat{T: ovToday(9, 1), S: "warn", M: 900, C: 200})
	hb.Append("monitor-1", "target-1", services.Heartbeat{T: ovToday(9, 2), S: "warn", M: 950, C: 200})
	hb.Append("monitor-1", "target-1", services.Heartbeat{T: ovToday(9, 3), S: "ok", M: 110})
	hb.Append("monitor-1", "target-1", services.Heartbeat{T: ovToday(9, 4), S: "down", C: 502, E: "HTTP 502"})
	hb.Append("monitor-1", "target-1", services.Heartbeat{T: ovToday(9, 5), S: "down", C: 502, E: "HTTP 502"})
	hb.Append("monitor-1", "target-1", services.Heartbeat{T: ovToday(9, 6), S: "down", C: 503, E: "HTTP 503"})
	hb.Append("monitor-1", "target-1", services.Heartbeat{T: ovToday(9, 7), S: "ok", M: 105})

	issues := ovRunIssues(t, st, hb, ovAdminSession(st))
	if len(issues) != 1 {
		t.Fatalf("got %d issues, want 1: %+v", len(issues), issues)
	}
	issue := issues[0]
	if issue.MonitorName != "Home lab" || issue.TargetName != "site" {
		t.Fatalf("issue names = %q / %q, want Home lab / site", issue.MonitorName, issue.TargetName)
	}
	if issue.MonitorID != "monitor-1" || issue.TargetID != "target-1" {
		t.Fatalf("issue ids = %q / %q, want monitor-1 / target-1", issue.MonitorID, issue.TargetID)
	}
	if issue.Total != 8 || issue.Warn != 2 || issue.Down != 3 {
		t.Fatalf("issue counts = total %d warn %d down %d, want 8 / 2 / 3", issue.Total, issue.Warn, issue.Down)
	}
	if issue.PeakMs != 950 {
		t.Fatalf("peak_ms = %d, want 950", issue.PeakMs)
	}
	if issue.IncidentCnt != 2 || len(issue.Incidents) != 2 {
		t.Fatalf("incidents = %d of %d, want 2 of 2", len(issue.Incidents), issue.IncidentCnt)
	}

	degraded := issue.Incidents[0]
	if degraded.State != "warn" || degraded.Count != 2 {
		t.Fatalf("first incident = %+v, want a 2-sample warn window", degraded)
	}
	if degraded.From != ovToday(9, 1)/1000 || degraded.To != ovToday(9, 2)/1000 {
		t.Fatalf("first incident spans %d..%d, want %d..%d",
			degraded.From, degraded.To, ovToday(9, 1)/1000, ovToday(9, 2)/1000)
	}

	outage := issue.Incidents[1]
	if outage.State != "down" || outage.Count != 3 {
		t.Fatalf("second incident = %+v, want a 3-sample down window", outage)
	}
	// The first failing probe supplies the code and message, so the window keeps
	// the cause it opened with rather than the last one seen.
	if outage.Code != 502 || outage.Error != "HTTP 502" {
		t.Fatalf("second incident code/error = %d / %q, want 502 / HTTP 502", outage.Code, outage.Error)
	}
}

// A window that degrades and then fails outright is reported as the worse of
// the two states, not as whatever it happened to open with.
func TestOverviewIncidentStateEscalatesToDown(t *testing.T) {
	st, hb := ovFixture(t, "monitor-1", "Home lab", "target-1", "site")
	hb.Append("monitor-1", "target-1", services.Heartbeat{T: ovToday(9, 0), S: "warn", M: 800})
	hb.Append("monitor-1", "target-1", services.Heartbeat{T: ovToday(9, 1), S: "down", C: 500})
	hb.Append("monitor-1", "target-1", services.Heartbeat{T: ovToday(9, 2), S: "down", C: 500})

	issues := ovRunIssues(t, st, hb, ovAdminSession(st))
	if len(issues) != 1 || len(issues[0].Incidents) != 1 {
		t.Fatalf("got %+v, want one issue with one incident", issues)
	}
	if got := issues[0].Incidents[0]; got.State != "down" || got.Count != 3 {
		t.Fatalf("incident = %+v, want a 3-sample down window", got)
	}
}

// A flapping target must not turn one bucket into an unbounded list: the count
// stays truthful while the windows themselves are capped.
func TestOverviewIncidentsCappedPerTarget(t *testing.T) {
	st, hb := ovFixture(t, "monitor-1", "Home lab", "target-1", "site")
	const windows = 20
	for i := 0; i < windows; i++ {
		hb.Append("monitor-1", "target-1", services.Heartbeat{T: ovToday(1, i*2), S: "down", C: 500})
		hb.Append("monitor-1", "target-1", services.Heartbeat{T: ovToday(1, i*2+1), S: "ok", M: 100})
	}

	issues := ovRunIssues(t, st, hb, ovAdminSession(st))
	if len(issues) != 1 {
		t.Fatalf("got %d issues, want 1", len(issues))
	}
	if issues[0].IncidentCnt != windows {
		t.Fatalf("incident_count = %d, want %d", issues[0].IncidentCnt, windows)
	}
	if len(issues[0].Incidents) != maxBucketIncidents {
		t.Fatalf("kept %d incidents, want the cap of %d", len(issues[0].Incidents), maxBucketIncidents)
	}
}

// Worst first, so the target that was actually down leads the list instead of
// whichever one the map happened to yield first.
func TestOverviewIssuesSortedWorstFirst(t *testing.T) {
	st := store.NewStore(filepath.Join(t.TempDir(), "config.json"))
	if err := st.AddMonitor(models.Monitor{
		ID:     "monitor-1",
		UserID: st.AdminUserID(),
		Name:   "Home lab",
		Targets: []models.MonitorTarget{
			{ID: "target-a", Name: "alpha", URL: "https://example.com/a"},
			{ID: "target-b", Name: "bravo", URL: "https://example.com/b"},
		},
	}); err != nil {
		t.Fatal(err)
	}
	hb := services.NewHeartbeatLog(filepath.Join(t.TempDir(), "heartbeats.json"))
	hb.Append("monitor-1", "target-a", services.Heartbeat{T: ovToday(9, 0), S: "warn", M: 700})
	for i := 0; i < 3; i++ {
		hb.Append("monitor-1", "target-b", services.Heartbeat{T: ovToday(9, i), S: "down", C: 500})
	}

	issues := ovRunIssues(t, st, hb, ovAdminSession(st))
	if len(issues) != 2 {
		t.Fatalf("got %d issues, want 2: %+v", len(issues), issues)
	}
	if issues[0].TargetName != "bravo" || issues[1].TargetName != "alpha" {
		t.Fatalf("order = %q then %q, want bravo then alpha", issues[0].TargetName, issues[1].TargetName)
	}
}

// The detail is as private as the bar it belongs to: a member who cannot see a
// monitor must not learn which of its targets failed.
func TestOverviewIssuesRespectVisibility(t *testing.T) {
	st, hb := ovFixture(t, "monitor-1", "Home lab", "target-1", "site")
	if err := st.CreateUser(models.User{
		ID: "member-1", Username: "member", PasswordHash: store.HashPassword("password"),
		Role: models.RoleUser, Status: models.UserActive, EmailVerified: true,
	}); err != nil {
		t.Fatal(err)
	}
	hb.Append("monitor-1", "target-1", services.Heartbeat{T: ovToday(9, 0), S: "down", C: 500})

	admin := ovRunIssues(t, st, hb, ovAdminSession(st))
	if len(admin) != 1 {
		t.Fatalf("administrator saw %d issues, want 1", len(admin))
	}
	member := ovRunIssues(t, st, hb, models.SessionUser{ID: "member-1", Role: models.RoleUser})
	if len(member) != 0 {
		t.Fatalf("member saw %d issues for someone else's monitor, want none: %+v", len(member), member)
	}
}
