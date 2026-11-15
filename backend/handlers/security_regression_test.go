package handlers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"tunnel-manager/auth"
	"tunnel-manager/models"
	"tunnel-manager/services"
	"tunnel-manager/store"
)

func TestSecurityRegressionPasswordResetRevokesOldChallenge(t *testing.T) {
	admin, secret, _ := newEnabledAdminHandler(t)
	if err := admin.store.SetUserEmail(admin.store.AdminUserID(), "owner@example.com", true); err != nil {
		t.Fatal(err)
	}
	challenge := beginChallenge(t, admin)
	code := "123456"
	sum := sha256.Sum256([]byte(code))
	admin.store.PutVerifyCode("owner@example.com", "reset", hex.EncodeToString(sum[:]), time.Minute)
	registration := NewAuthHandler(admin.store, admin.encryptionKey)
	reset := performJSON(t, registration.ResetPassword, "POST", "/api/auth/reset-password", `{"email":"owner@example.com","code":"123456","new_password":"new-password"}`, "")
	if reset.Code != 200 {
		t.Fatalf("reset = %d %s", reset.Code, reset.Body.String())
	}
	factor := auth.GenerateTOTPCode(secret, admin.now().Unix()/30)
	finish := performJSON(t, admin.LoginTwoFactor, "POST", "/api/admin/login/2fa", `{"challenge_token":"`+challenge+`","code":"`+factor+`"}`, "")
	var login models.LoginResponse
	decodeResponse(t, finish, &login)
	if finish.Code != 401 || admin.ValidateToken(login.Token) {
		t.Fatalf("old challenge issued a token: %d", finish.Code)
	}
}

func TestSecurityRegressionForwardedIPSpoofCannotChangeRateBudget(t *testing.T) {
	st := newTestStore(t)
	settings := st.GetAppSettings()
	settings.RateLimitPerIP = 1
	if err := st.SetAppSettings(settings); err != nil {
		t.Fatal(err)
	}
	throttle := NewThrottle(st)
	request := httptest.NewRequest("POST", "/api/admin/login", nil)
	request.RemoteAddr = "198.51.100.9:40000"
	request.Header.Set("X-Forwarded-For", "203.0.113.1, 198.51.100.9")
	throttle.Fail(request, "", "")
	before, _ := throttle.Guard(request, "", "")
	request.Header.Set("X-Forwarded-For", "203.0.113.2, 198.51.100.9")
	after, _ := throttle.Guard(request, "", "")
	if before || after {
		t.Fatalf("same remote peer: before=%v after=%v", before, after)
	}
}

func TestSlimAdminCanMonitorLoopback(t *testing.T) {
	st := newTestStore(t)
	// The slim edition exposes monitoring only to its single administrator.
	uid := st.AdminUserID()
	if err := st.CreateSession(hashToken(strings.Repeat("a", 64)), uid, time.Now().Add(time.Hour).Unix()); err != nil {
		t.Fatal(err)
	}
	var hits atomic.Int32
	internal := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/internal-action" {
			t.Errorf("unexpected internal request %s %s", r.Method, r.URL.Path)
		}
		hits.Add(1)
		w.WriteHeader(204)
	}))
	defer internal.Close()
	hb := services.NewHeartbeatLog(filepath.Join(t.TempDir(), "heartbeats.json"))
	h := NewMonitorsHandler(st, hb, services.NewRunner(st, hb), nil)
	admin := NewAdminHandler(st)
	mw := &Middleware{AdminHandler: admin}
	r := chi.NewRouter()
	r.Post("/api/monitors", mw.Auth(mw.RequirePerm(models.PermMonitors, h.Create)))
	r.Post("/api/monitors/{monitorID}/targets", mw.Auth(mw.RequirePerm(models.PermMonitors, h.AddTarget)))
	r.Post("/api/monitors/{monitorID}/check", mw.Auth(mw.RequirePerm(models.PermMonitors, h.CheckNow)))
	created := serveSession(t, r, "POST", "/api/monitors", `{"name":"internal"}`, strings.Repeat("a", 64))
	var monitor monitorView
	decodeResponse(t, created, &monitor)
	added := serveSession(t, r, "POST", "/api/monitors/"+monitor.ID+"/targets", `{"name":"action","url":"`+internal.URL+`/internal-action","type":"http","method":"POST"}`, strings.Repeat("a", 64))
	checked := serveSession(t, r, "POST", "/api/monitors/"+monitor.ID+"/check", `{}`, strings.Repeat("a", 64))
	if created.Code != 200 || added.Code != 200 || checked.Code != 200 || hits.Load() != 1 {
		t.Fatalf("create/add/check/hits=%d/%d/%d/%d", created.Code, added.Code, checked.Code, hits.Load())
	}
}

func TestSecurityRegressionOAuthStatusLoadsAccountsWithoutCFContext(t *testing.T) {
	st := newTestStore(t)
	key := bytes.Repeat([]byte{4}, 32)
	uid := st.AdminUserID()
	encrypted, err := auth.EncryptSecret(key, "cloudflare-oauth-access-token", []byte("mock-access"))
	if err != nil {
		t.Fatal(err)
	}
	id, err := st.CreateCFConnection(models.CFConnection{UserID: uid, AccountID: "account-a", AccessToken: encrypted, ExpiresAt: time.Now().Add(time.Hour).Unix()})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.SetActiveCFConnection(uid, id); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateSession(hashToken(strings.Repeat("a", 64)), uid, time.Now().Add(time.Hour).Unix()); err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	transport := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"success":true,"result":[{"id":"account-a","name":"Account A"}]}`))}, nil
	})
	t.Cleanup(func() { http.DefaultTransport = transport })
	oauth := services.NewCloudflareOAuth(st, key, services.CloudflareOAuthConfig{ClientID: "mock", ClientSecret: "mock"})
	cf := services.NewCloudflareClient("", "")
	cf.SetOAuth(oauth)
	cf.SetSessionStore(st)
	admin := NewAdminHandler(st)
	h := NewCloudflareOAuthHandler(st, oauth, cf, admin)
	mw := &Middleware{AdminHandler: admin, CF: cf}
	response := performJSON(t, mw.SessionOnly(h.Status), "GET", "https://panel.example.com/api/cloudflare/oauth/status", ``, strings.Repeat("a", 64))
	var view models.CloudflareOAuthStatus
	decodeResponse(t, response, &view)
	if response.Code != 200 || !view.Connected || len(view.Accounts) != 1 || view.Accounts[0].ID != "account-a" || calls.Load() != 1 {
		t.Fatalf("status=%d connected=%v accounts=%d APIcalls=%d", response.Code, view.Connected, len(view.Accounts), calls.Load())
	}
}

func TestSecurityRegressionPublicFailureHidesTargetURL(t *testing.T) {
	st := newTestStore(t)
	hb := services.NewHeartbeatLog(filepath.Join(t.TempDir(), "hb.json"))
	closed := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	closed.Close()
	url := closed.URL + "/?key=review-secret"
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	probe := services.ProbeURL(ctx, url)
	if probe.Err == nil {
		t.Fatal("expected canceled request")
	}
	monitor := models.Monitor{ID: "review-monitor", UserID: st.AdminUserID(), Name: "public", PublicToken: "review-public", PublishEnabled: true, Targets: []models.MonitorTarget{{ID: "t", Name: "hidden", Type: "http", URL: url, LinkEnabled: false}}}
	if err := st.AddMonitor(monitor); err != nil {
		t.Fatal(err)
	}
	hb.Append(monitor.ID, "t", services.Heartbeat{T: time.Now().UnixMilli(), S: "down", E: `Post "` + url + `": context canceled`})
	h := NewMonitorsHandler(st, hb, nil, nil)
	router := chi.NewRouter()
	router.Get("/api/public/status/{token}", h.PublicStatus)
	response := serveSession(t, router, "GET", "/api/public/status/review-public", "", "")
	if response.Code != 200 || strings.Contains(response.Body.String(), "review-secret") || strings.Contains(response.Body.String(), closed.URL) {
		t.Fatalf("hidden URL exposed: %d %s", response.Code, response.Body.String())
	}
}

func TestSecurityRegressionCustomStatusDomainServesFont(t *testing.T) {
	st := newTestStore(t)
	if err := st.AddMonitor(models.Monitor{ID: "review-font", UserID: st.AdminUserID(), Name: "font", PublicDomain: "status.example.com", PublicToken: "font-token", PublishEnabled: true}); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/fonts/fonts.css", "/fonts/misans-400-001.woff2"} {
		called := false
		wrapped := StatusDomainRedirect(st)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true; w.WriteHeader(200) }))
		response := httptest.NewRecorder()
		wrapped.ServeHTTP(response, httptest.NewRequest("GET", "https://status.example.com"+path, nil))
		if response.Code != 200 || !called {
			t.Fatalf("font asset blocked: %s", path)
		}
	}
}

func TestSecurityRegressionAnonymousPasskeyBeginChargesBudget(t *testing.T) {
	h, admin, st := newTestPasskeyHandler(t)
	settings := st.GetAppSettings()
	settings.RateLimitPerIP = 1
	if err := st.SetAppSettings(settings); err != nil {
		t.Fatal(err)
	}
	admin.SetThrottle(NewThrottle(st))
	accepted := 0
	for i := 0; i < 300; i++ {
		response := performJSON(t, h.LoginBegin, "POST", "https://panel.example.com/api/auth/passkey/login/begin", `{}`, "")
		if response.Code == 200 {
			accepted++
			continue
		}
		if response.Code != 429 || accepted != 1 {
			t.Fatalf("begins=%d, status=%d, body=%s", accepted, response.Code, response.Body.String())
		}
		return
	}
	t.Fatal("capacity not reached")
}

func TestSecurityRegressionEmailUpdateRequiresMailboxProof(t *testing.T) {
	admin := newTestAdminHandler(t)
	session := login(t, admin, "admin", "password")
	user, _ := admin.store.GetUserByID(admin.store.AdminUserID())
	original := user.Email
	request := `{"current_password":"password","new_email":"new@example.com","code":"123456"}`
	denied := performJSON(t, admin.ChangeEmail, "PUT", "/api/admin/email", request, session)
	if denied.Code != 400 {
		t.Fatalf("unissued code accepted: %d", denied.Code)
	}
	unchanged, _ := admin.store.GetUserByID(user.ID)
	if unchanged.Email != original {
		t.Fatal("changed email before proof")
	}
	sum := sha256.Sum256([]byte("123456"))
	admin.store.PutVerifyCode("new@example.com", emailChangePurpose(user), hex.EncodeToString(sum[:]), time.Minute)
	accepted := performJSON(t, admin.ChangeEmail, "PUT", "/api/admin/email", request, session)
	updated, _ := admin.store.GetUserByID(user.ID)
	if accepted.Code != 200 || updated.Email != "new@example.com" || !updated.EmailVerified {
		t.Fatalf("mail proof failed: %d %+v", accepted.Code, updated)
	}
}

func TestSecurityRegressionCLIResetRejectsClaimedChallenge(t *testing.T) {
	admin, _, _ := newEnabledAdminHandler(t)
	token := beginChallenge(t, admin)
	claimed, ok := admin.claimChallenge(token)
	if !ok {
		t.Fatal("cannot claim challenge")
	}
	if err := admin.store.SetAdminCredentials("admin", store.HashPassword("new-password")); err != nil {
		t.Fatalf("CLI credential reset: %v", err)
	}
	sessionToken, _ := generateToken()
	valid, err := admin.finishChallenge(token, claimed, sessionToken, nil)
	if valid || err != nil || admin.ValidateToken(sessionToken) {
		t.Fatalf("claimed old challenge survived reset: %v %v", valid, err)
	}
}

func TestManualMonitoringChecksAreLimitedPerAccount(t *testing.T) {
	st := newTestStore(t)
	hb := services.NewHeartbeatLog(filepath.Join(t.TempDir(), "hb.json"))
	h := NewMonitorsHandler(st, hb, services.NewRunner(st, hb), nil)
	monitor := models.Monitor{ID: "limited-monitor", UserID: st.AdminUserID(), Name: "limited"}
	if err := st.AddMonitor(monitor); err != nil {
		t.Fatal(err)
	}
	admin := NewAdminHandler(st)
	session, err := admin.createSession(st.AdminUserID(), admin.currentAuthEpoch())
	if err != nil {
		t.Fatal(err)
	}
	mw := &Middleware{AdminHandler: admin}
	router := chi.NewRouter()
	router.Post("/monitors/{monitorID}/check", mw.SessionOnly(h.CheckNow))
	for i := 0; i < 6; i++ {
		response := serveSession(t, router, "POST", "/monitors/limited-monitor/check", "{}", session)
		want := 200
		if i == 5 {
			want = 429
		}
		if response.Code != want {
			t.Fatalf("check %d: %d %s", i, response.Code, response.Body.String())
		}
	}
}
