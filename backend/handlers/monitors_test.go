package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"tunnel-manager/models"
	"tunnel-manager/services"
	"tunnel-manager/store"
)

func TestListMonitorsHidesOtherUsersProjectsAfterReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	st := store.NewStore(path)
	if err := st.AddMonitor(models.Monitor{
		ID:     "admin-monitor",
		UserID: st.AdminUserID(),
		Name:   "Administrator project",
	}); err != nil {
		t.Fatal(err)
	}

	reloaded := store.NewStore(path)
	heartbeats := services.NewHeartbeatLog(filepath.Join(t.TempDir(), "heartbeats.json"))
	handler := NewMonitorsHandler(reloaded, heartbeats, nil, nil)
	req := withUser(httptest.NewRequest(http.MethodGet, "/api/monitors", nil), models.SessionUser{
		ID:   "other-user",
		Role: models.RoleUser,
	})
	resp := httptest.NewRecorder()

	handler.List(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("List() status = %d, want %d", resp.Code, http.StatusOK)
	}
	var monitors []monitorView
	decodeResponse(t, resp, &monitors)
	if len(monitors) != 0 {
		t.Fatalf("List() returned %d monitor(s) owned by another user", len(monitors))
	}
}

func TestCreateMonitorWithAPIKeyIdentityAssignsAdministratorOwner(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	st := store.NewStore(path)
	heartbeats := services.NewHeartbeatLog(filepath.Join(t.TempDir(), "heartbeats.json"))
	handler := NewMonitorsHandler(st, heartbeats, nil, nil)
	req := withUser(httptest.NewRequest(http.MethodPost, "/api/monitors", strings.NewReader(`{"name":"API monitor"}`)), models.SessionUser{
		Role: models.RoleAdmin,
	})
	resp := httptest.NewRecorder()

	handler.Create(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("Create() status = %d, want %d: %s", resp.Code, http.StatusOK, resp.Body.String())
	}
	monitors := st.GetConfig().Monitors
	if len(monitors) != 1 {
		t.Fatalf("monitor count = %d, want 1", len(monitors))
	}
	if monitors[0].UserID != st.AdminUserID() {
		t.Fatalf("monitor owner = %q, want administrator %q", monitors[0].UserID, st.AdminUserID())
	}
}

func TestUpdateMonitorPersistsDomainWhenProvisioningFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	st := store.NewStore(path)
	if err := st.SetPanelHost("panel.example.com"); err != nil {
		t.Fatal(err)
	}
	if err := st.AddMonitor(models.Monitor{
		ID:     "user-monitor",
		UserID: "owner-user",
		Name:   "User project",
	}); err != nil {
		t.Fatal(err)
	}
	provisioner := &fakeStatusDomainProvisioner{err: errors.New("zone unavailable")}
	heartbeats := services.NewHeartbeatLog(filepath.Join(t.TempDir(), "heartbeats.json"))
	handler := NewMonitorsHandler(st, heartbeats, nil, provisioner)
	req := httptest.NewRequest(http.MethodPut, "/api/monitors/user-monitor", strings.NewReader(`{"public_domain":"status.example.com"}`))
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("monitorID", "user-monitor")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeContext))
	req = withUser(req, models.SessionUser{ID: st.AdminUserID(), Role: models.RoleAdmin})
	resp := httptest.NewRecorder()

	handler.Update(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("Update() status = %d, want %d: %s", resp.Code, http.StatusOK, resp.Body.String())
	}
	var view monitorView
	decodeResponse(t, resp, &view)
	if view.PublicDomain != "status.example.com" || view.DomainWarning != "zone unavailable" {
		t.Fatalf("Update() response = %#v", view)
	}
	if provisioner.userID != "owner-user" || provisioner.panelHost != "panel.example.com" || provisioner.hostname != "status.example.com" {
		t.Fatalf("provision request = (%q, %q, %q)", provisioner.userID, provisioner.panelHost, provisioner.hostname)
	}
	stored := st.GetConfig().Monitors
	if len(stored) != 1 || stored[0].PublicDomain != "status.example.com" {
		t.Fatalf("stored monitors = %#v", stored)
	}
}

func TestUpdateMonitorRejectsCurrentRequestHostBeforePanelHostIsSeeded(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	st := store.NewStore(path)
	if err := st.AddMonitor(models.Monitor{
		ID:     "user-monitor",
		UserID: "owner-user",
		Name:   "User project",
	}); err != nil {
		t.Fatal(err)
	}
	heartbeats := services.NewHeartbeatLog(filepath.Join(t.TempDir(), "heartbeats.json"))
	handler := NewMonitorsHandler(st, heartbeats, nil, nil)
	req := httptest.NewRequest(http.MethodPut, "/api/monitors/user-monitor", strings.NewReader(`{"public_domain":"panel.example.com"}`))
	req.Host = "panel.example.com"
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("monitorID", "user-monitor")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeContext))
	req = withUser(req, models.SessionUser{ID: "owner-user", Role: models.RoleUser})
	resp := httptest.NewRecorder()

	handler.Update(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("Update() status = %d, want %d: %s", resp.Code, http.StatusBadRequest, resp.Body.String())
	}
	stored := st.GetConfig().Monitors
	if len(stored) != 1 || stored[0].PublicDomain != "" {
		t.Fatalf("stored monitors = %#v", stored)
	}
}

func TestStatusDomainRedirectRoutesOnlyCustomDomainRoot(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	st := store.NewStore(path)
	if err := st.AddMonitor(models.Monitor{
		ID:             "public-monitor",
		UserID:         st.AdminUserID(),
		Name:           "Public project",
		PublicDomain:   "status.example.com",
		PublicSlug:     "team",
		PublishEnabled: true,
	}); err != nil {
		t.Fatal(err)
	}
	fallbackCalls := 0
	handler := StatusDomainRedirect(st)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fallbackCalls++
		w.WriteHeader(http.StatusNoContent)
	}))

	rootReq := httptest.NewRequest(http.MethodGet, "http://status.example.com/?from=custom", nil)
	rootReq.Host = "status.example.com"
	rootResp := httptest.NewRecorder()
	handler.ServeHTTP(rootResp, rootReq)
	if rootResp.Code != http.StatusFound {
		t.Fatalf("root status = %d, want %d", rootResp.Code, http.StatusFound)
	}
	if location := rootResp.Header().Get("Location"); location != "/status/team?from=custom" {
		t.Fatalf("root location = %q", location)
	}
	if fallbackCalls != 0 {
		t.Fatalf("fallback calls after root = %d, want 0", fallbackCalls)
	}

	otherReq := httptest.NewRequest(http.MethodGet, "http://status.example.com/api/site", nil)
	otherReq.Host = "status.example.com"
	otherResp := httptest.NewRecorder()
	handler.ServeHTTP(otherResp, otherReq)
	if otherResp.Code != http.StatusNoContent {
		t.Fatalf("other path status = %d, want %d", otherResp.Code, http.StatusNoContent)
	}
	if fallbackCalls != 1 {
		t.Fatalf("fallback calls after other path = %d, want 1", fallbackCalls)
	}
}

type fakeStatusDomainProvisioner struct {
	userID    string
	panelHost string
	hostname  string
	err       error
}

func (f *fakeStatusDomainProvisioner) ProvisionStatusDomain(userID, panelHost, hostname string) error {
	f.userID = userID
	f.panelHost = panelHost
	f.hostname = hostname
	return f.err
}
