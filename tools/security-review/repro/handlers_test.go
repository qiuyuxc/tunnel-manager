package handlers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-webauthn/webauthn/webauthn"
	"tunnel-manager/auth"
	"tunnel-manager/models"
	"tunnel-manager/services"
	"tunnel-manager/store"
)

func TestSecurityReviewPasskeyFactorLocksChallenge(t *testing.T) {
	admin, secret, _ := newEnabledAdminHandler(t)
	encoded, err := services.EncodePasskeyCredential(&webauthn.Credential{ID: []byte{1, 2, 3, 4}, PublicKey: []byte{5, 6, 7}})
	if err != nil {
		t.Fatal(err)
	}
	if err := admin.store.AddPasskey(models.Passkey{ID: "AQIDBA", UserID: admin.store.AdminUserID(), Credential: encoded}); err != nil {
		t.Fatal(err)
	}
	challenge := beginChallenge(t, admin)
	h := NewPasskeyHandler(admin.store, services.NewPasskeyService(admin.store), admin)
	begin := performJSON(t, h.TwoFactorBegin, "POST", "https://panel.example.com/api/admin/login/2fa/passkey/begin", `{"challenge_token":"`+challenge+`"}`, "")
	if begin.Code != http.StatusOK {
		t.Fatalf("begin = %d %s", begin.Code, begin.Body.String())
	}
	finish := performJSON(t, h.TwoFactorFinish, "POST", "https://panel.example.com/api/admin/login/2fa/passkey/finish", `{"challenge_token":"`+challenge+`","ceremony_token":"unused","credential":{}}`, "")
	code := auth.GenerateTOTPCode(secret, admin.now().Unix()/30)
	fallback := performJSON(t, admin.LoginTwoFactor, "POST", "https://panel.example.com/api/admin/login/2fa", `{"challenge_token":"`+challenge+`","code":"`+code+`"}`, "")
	if finish.Code != 401 || fallback.Code != 401 || !admin.challenges[challenge].inUse {
		t.Fatal("expected challenge to remain locked")
	}
	t.Logf("passkey begin=%d; finish=%d before assertion validation; valid TOTP fallback=%d; inUse=true", begin.Code, finish.Code, fallback.Code)
}

func TestSecurityReviewPasswordResetLeavesOldChallenge(t *testing.T) {
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
	if finish.Code != 200 || !admin.ValidateToken(login.Token) {
		t.Fatalf("expected old challenge to issue a usable token: %d", finish.Code)
	}
	t.Log("password reset=200; pre-reset challenge + factor=200; issued session is valid")
}

func TestSecurityReviewForwardedIPChangesRateBudget(t *testing.T) {
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
	if before || !after {
		t.Fatalf("same remote peer: before=%v after=%v", before, after)
	}
	t.Logf("same RemoteAddr; exhausted IP allowed=%v; changed untrusted first XFF entry allowed=%v", before, after)
}

func TestSecurityReviewMonitorReachesLoopback(t *testing.T) {
	st := newTestStore(t)
	group, err := st.CreateGroup("review-monitor-only", []string{models.PermMonitors})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.CreateUser(models.User{ID: "review-user", Username: "review-user", Role: models.RoleUser, GroupID: group.ID}); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateSession(hashToken(strings.Repeat("a", 64)), "review-user", time.Now().Add(time.Hour).Unix()); err != nil {
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
	t.Log("ordinary monitor-only user triggered POST to 127.0.0.1 internal service through authenticated production handlers")
}

func TestSecurityReviewOAuthStatusDropsCFContext(t *testing.T) {
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
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: http.NoBody}, nil
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
	if response.Code != 200 || !view.Connected || len(view.Accounts) != 0 || calls.Load() != 0 {
		t.Fatalf("status=%d connected=%v accounts=%d APIcalls=%d", response.Code, view.Connected, len(view.Accounts), calls.Load())
	}
	t.Log("actual SessionOnly route: connected=true; accounts=[]; ListAccounts never called")
}

func TestSecurityReviewPublicFailureLeaksHiddenURL(t *testing.T) {
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
	hb.Append(monitor.ID, "t", services.Heartbeat{T: time.Now().UnixMilli(), S: "down", E: services.ProbeErrorText(probe.Err)})
	h := NewMonitorsHandler(st, hb, nil, nil)
	router := chi.NewRouter()
	router.Get("/api/public/status/{token}", h.PublicStatus)
	response := serveSession(t, router, "GET", "/api/public/status/review-public", "", "")
	if response.Code != 200 || !strings.Contains(response.Body.String(), "review-secret") {
		t.Fatalf("hidden URL leak not reproduced: %d %s", response.Code, response.Body.String())
	}
	t.Log("public unauthenticated API exposes target query secret in error despite link_enabled=false")
}

func TestSecurityReviewCustomStatusDomainBlocksFont(t *testing.T) {
	st := newTestStore(t)
	if err := st.AddMonitor(models.Monitor{ID: "review-font", UserID: st.AdminUserID(), Name: "font", PublicDomain: "status.example.com", PublicToken: "font-token", PublishEnabled: true}); err != nil {
		t.Fatal(err)
	}
	called := false
	wrapped := StatusDomainRedirect(st)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true; w.WriteHeader(200) }))
	response := httptest.NewRecorder()
	wrapped.ServeHTTP(response, httptest.NewRequest("GET", "https://status.example.com/fonts/fonts.css", nil))
	if response.Code != 404 || called {
		t.Fatal("expected asset allowlist to block fonts")
	}
	t.Log("status custom domain GET /fonts/fonts.css returns 404 before static handler")
}

func TestSecurityReviewEmailClaimPreHijacksOIDCAccount(t *testing.T) {
	admin := newTestAdminHandler(t)
	st := admin.store
	if err := st.CreateUser(models.User{ID: "local-attacker", Username: "local-attacker", Email: "attacker@example.com", PasswordHash: store.HashPassword("attacker-password"), Role: models.RoleUser, EmailVerified: true}); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateSession(hashToken(strings.Repeat("a", 64)), "local-attacker", time.Now().Add(time.Hour).Unix()); err != nil {
		t.Fatal(err)
	}
	claim := performJSON(t, admin.ChangeEmail, "PUT", "/api/admin/email", `{"current_password":"attacker-password","new_email":"victim@example.com"}`, strings.Repeat("a", 64))
	if claim.Code != 200 {
		t.Fatalf("email claim=%d %s", claim.Code, claim.Body.String())
	}
	registration := NewAuthHandler(st, admin.encryptionKey)
	h := NewOIDCHandler(st, admin, registration, admin.encryptionKey)
	provider, err := st.SaveOIDCProvider(models.OIDCProvider{Name: "test-provider", Issuer: "https://id.example.com", ClientID: "panel", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	h.pending["ticket"] = &oidcPending{flow: oidcFlow{provider: provider, action: "login", browserHash: hashToken("victim-browser"), expires: time.Now().Add(time.Minute)}, claims: services.OIDCClaims{Issuer: provider.Issuer, Subject: "verified-victim-subject", Email: "victim@example.com", EmailVerified: true}}
	r := httptest.NewRequest("POST", "https://panel.example.com/api/auth/oidc/complete", strings.NewReader(`{"ticket":"ticket"}`))
	r.AddCookie(&http.Cookie{Name: oidcCookieName, Value: "victim-browser"})
	response := httptest.NewRecorder()
	h.Complete(response, r)
	var result models.LoginResponse
	decodeResponse(t, response, &result)
	identity, valid := admin.ValidateSession(result.Token)
	local, _ := st.GetUserByID("local-attacker")
	if response.Code != 200 || !valid || identity.ID != "local-attacker" || !st.ValidateUserPassword(local.ID, "attacker-password", local.PasswordHash) {
		t.Fatalf("OIDC complete=%d; identity=%s; body=%s", response.Code, identity.ID, response.Body.String())
	}
	t.Log("local attacker claims victim mailbox without mail proof; verified victim OIDC signs into attacker account; attacker's password remains valid")
}

func TestSecurityReviewAnonymousPasskeyBeginExhaustsCapacity(t *testing.T) {
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
		if response.Code != 400 || !strings.Contains(response.Body.String(), "请求过多") || accepted != 256 {
			t.Fatalf("begins=%d, status=%d, body=%s", accepted, response.Code, response.Body.String())
		}
		t.Logf("one IP with per-IP budget=1 created %d live anonymous ceremonies; next begin fails global capacity instead of rate limiting", accepted)
		return
	}
	t.Fatal("capacity not reached")
}
