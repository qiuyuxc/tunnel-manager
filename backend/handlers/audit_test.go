package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"tunnel-manager/models"
	"tunnel-manager/store"
)

func newAuditTestMiddleware(t *testing.T) (*Middleware, *store.Store) {
	t.Helper()
	st := newTestStore(t)
	return &Middleware{Store: st}, st
}

func TestAuditMiddlewareRecordsOutcomeAndTarget(t *testing.T) {
	mw, st := newAuditTestMiddleware(t)
	handler := mw.Audit(models.AuditCategoryUser, models.AuditActionUserCreate, nil, func(w http.ResponseWriter, r *http.Request) {
		SetAuditTarget(r, "new-user")
		writeJSON(w, http.StatusCreated, map[string]string{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodPost, "/api/admin/users", nil)
	req = withUser(req, models.SessionUser{ID: "u1", Username: "alice"})
	req.Header.Set("X-Forwarded-For", "203.0.113.7, 10.0.0.1")
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("handler status = %d, want 201", rec.Code)
	}
	page, err := st.QueryAuditLogs(models.AuditQuery{})
	if err != nil {
		t.Fatalf("QueryAuditLogs() error = %v", err)
	}
	if page.Total != 1 {
		t.Fatalf("recorded %d entries, want 1", page.Total)
	}
	entry := page.Logs[0]
	if entry.Category != models.AuditCategoryUser || entry.Action != models.AuditActionUserCreate {
		t.Fatalf("entry = %s/%s, want user/user_create", entry.Category, entry.Action)
	}
	if entry.ActorID != "u1" || entry.ActorName != "alice" {
		t.Fatalf("actor = %q/%q, want u1/alice", entry.ActorID, entry.ActorName)
	}
	if entry.Target != "new-user" {
		t.Fatalf("target = %q, want the handler supplied name", entry.Target)
	}
	if entry.IP != "203.0.113.7" {
		t.Fatalf("ip = %q, want the forwarded client address", entry.IP)
	}
	if !entry.Success {
		t.Fatal("entry marked failed, want success")
	}
}

func TestAuditMiddlewareMarksRejectedRequests(t *testing.T) {
	mw, st := newAuditTestMiddleware(t)
	handler := mw.Audit(models.AuditCategoryTunnel, models.AuditActionTunnelDelete, nil, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "permission denied"})
	})

	req := httptest.NewRequest(http.MethodDelete, "/api/tunnels/t-1", nil)
	req = withUser(req, models.SessionUser{ID: "u1", Username: "alice"})
	rec := httptest.NewRecorder()
	handler(rec, req)

	page, err := st.QueryAuditLogs(models.AuditQuery{})
	if err != nil {
		t.Fatalf("QueryAuditLogs() error = %v", err)
	}
	if page.Total != 1 || page.Logs[0].Success {
		t.Fatalf("entries = %#v, want one failed entry", page.Logs)
	}
}

func TestAuditMiddlewareRecordsWrittenBodyWithoutStatus(t *testing.T) {
	mw, st := newAuditTestMiddleware(t)
	handler := mw.Audit(models.AuditCategoryMonitor, models.AuditActionMonitorCheck, nil, func(w http.ResponseWriter, r *http.Request) {
		// No explicit WriteHeader: the recorder must treat the write as a 200.
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	req := httptest.NewRequest(http.MethodPost, "/api/monitors/m-1/check", nil)
	req = withUser(req, models.SessionUser{ID: "u1", Username: "alice"})
	rec := httptest.NewRecorder()
	handler(rec, req)

	page, err := st.QueryAuditLogs(models.AuditQuery{})
	if err != nil {
		t.Fatalf("QueryAuditLogs() error = %v", err)
	}
	if page.Total != 1 || !page.Logs[0].Success {
		t.Fatalf("entries = %#v, want one successful entry", page.Logs)
	}
}

func TestAuditMiddlewareResolvesPathTargets(t *testing.T) {
	mw, st := newAuditTestMiddleware(t)
	router := chi.NewRouter()
	router.Put("/users/{id}/status", mw.Audit(models.AuditCategoryUser, models.AuditActionUserStatus, mw.UserTarget("id"), func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}))
	router.Delete("/zones/{zoneID}/dns-records/{recordID}", mw.Audit(models.AuditCategoryDNS, models.AuditActionDNSDelete, PathParam("recordID"), func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}))

	user := models.User{Username: "bob", PasswordHash: store.HashPassword("password"), Role: models.RoleUser, Status: models.UserActive}
	if err := st.CreateUser(user); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	created, ok := st.GetUserByUsername("bob")
	if !ok {
		t.Fatal("created user missing")
	}

	req := httptest.NewRequest(http.MethodPut, "/users/"+created.ID+"/status", nil)
	req = withUser(req, models.SessionUser{ID: "admin", Username: "admin", Role: models.RoleAdmin})
	router.ServeHTTP(httptest.NewRecorder(), req)

	req = httptest.NewRequest(http.MethodDelete, "/zones/zone-1/dns-records/rec-9", nil)
	req = withUser(req, models.SessionUser{ID: "admin", Username: "admin", Role: models.RoleAdmin})
	router.ServeHTTP(httptest.NewRecorder(), req)

	page, err := st.QueryAuditLogs(models.AuditQuery{})
	if err != nil {
		t.Fatalf("QueryAuditLogs() error = %v", err)
	}
	if page.Total != 2 {
		t.Fatalf("recorded %d entries, want 2", page.Total)
	}
	targets := map[string]string{}
	for _, entry := range page.Logs {
		targets[entry.Action] = entry.Target
	}
	if targets[models.AuditActionDNSDelete] != "rec-9" {
		t.Fatalf("dns target = %q, want the record id", targets[models.AuditActionDNSDelete])
	}
	if targets[models.AuditActionUserStatus] != "bob" {
		t.Fatalf("user target = %q, want the resolved username", targets[models.AuditActionUserStatus])
	}
}

func TestAuditHandlerListsAndSummarizes(t *testing.T) {
	st := newTestStore(t)
	h := NewAuditHandler(st)
	for _, entry := range []models.AuditLog{
		{ActorID: "u1", ActorName: "alice", Category: models.AuditCategoryUser, Action: models.AuditActionUserCreate, Target: "bob", IP: "10.0.0.1", Success: true},
		{ActorID: "u2", ActorName: "carol", Category: models.AuditCategoryAuth, Action: models.AuditActionLoginFailed, Target: "carol", IP: "10.0.0.2"},
	} {
		if err := st.RecordAudit(entry); err != nil {
			t.Fatalf("RecordAudit() error = %v", err)
		}
	}

	resp := performJSON(t, h.List, http.MethodGet, "/api/admin/audit-logs?actor=alice", "", "")
	if resp.Code != http.StatusOK {
		t.Fatalf("List() code = %d: %s", resp.Code, resp.Body.String())
	}
	var page models.AuditPage
	decodeResponse(t, resp, &page)
	if page.Total != 1 || len(page.Logs) != 1 || page.Logs[0].Target != "bob" {
		t.Fatalf("List() = %#v, want the single alice entry", page)
	}

	resp = performJSON(t, h.List, http.MethodGet, "/api/admin/audit-logs?category=auth&page_size=5", "", "")
	decodeResponse(t, resp, &page)
	if page.Total != 1 || page.PageSize != 5 || page.Logs[0].Action != models.AuditActionLoginFailed {
		t.Fatalf("List() filtered = %#v, want the failed login", page)
	}

	resp = performJSON(t, h.Stats, http.MethodGet, "/api/admin/audit-logs/stats?days=7", "", "")
	if resp.Code != http.StatusOK {
		t.Fatalf("Stats() code = %d: %s", resp.Code, resp.Body.String())
	}
	var stats models.AuditStats
	decodeResponse(t, resp, &stats)
	if stats.Total != 2 || stats.Failed != 1 || stats.Actors != 2 || stats.Today != 2 {
		t.Fatalf("Stats() = %#v, want 2 total, 1 failed, 2 actors, 2 today", stats)
	}
}

func TestAuditRetentionRoundTrip(t *testing.T) {
	st := newTestStore(t)
	h := NewManagementHandler(st, []byte("0123456789abcdef0123456789abcdef"))

	resp := performJSON(t, h.UpdateAppSettings, http.MethodPut, "/api/admin/settings",
		`{"registration_enabled":true,"invite_mode":"off","audit_retention_days":30}`, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("UpdateAppSettings() code = %d: %s", resp.Code, resp.Body.String())
	}
	var view models.AppSettingsView
	decodeResponse(t, resp, &view)
	if view.AuditRetentionDays != 30 {
		t.Fatalf("response retention = %d, want 30", view.AuditRetentionDays)
	}
	if got := st.GetAppSettings().EffectiveAuditRetentionDays(); got != 30 {
		t.Fatalf("stored retention = %d, want 30", got)
	}

	resp = performJSON(t, h.UpdateAppSettings, http.MethodPut, "/api/admin/settings",
		`{"registration_enabled":true,"invite_mode":"off","audit_retention_days":99999}`, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("UpdateAppSettings() code = %d: %s", resp.Code, resp.Body.String())
	}
	if got := st.GetAppSettings().AuditRetentionDays; got != maxAuditRetentionDays {
		t.Fatalf("stored retention = %d, want the %d day cap", got, maxAuditRetentionDays)
	}
}
