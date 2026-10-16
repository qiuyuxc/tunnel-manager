package handlers

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"tunnel-manager/auth"
	"tunnel-manager/models"
	"tunnel-manager/services"
	"tunnel-manager/store"
)

func testOIDCHandler(t *testing.T) (*OIDCHandler, models.OIDCProvider) {
	t.Helper()
	s := newTestStore(t)
	key := bytes.Repeat([]byte{3}, 32)
	h := NewOIDCHandler(s, NewAdminHandler(s, key), NewAuthHandler(s, key), key)
	p, err := s.SaveOIDCProvider(models.OIDCProvider{Name: "Identity", Issuer: "https://id.example.com", ClientID: "panel", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	return h, p
}

func oidcTestRequest(t *testing.T, h *OIDCHandler, ticket, cookie, token string, body map[string]string, handler http.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	if body == nil {
		body = map[string]string{}
	}
	body["ticket"] = ticket
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodPost, "/api/auth/oidc/complete", bytes.NewReader(raw))
	if cookie != "" {
		r.AddCookie(&http.Cookie{Name: oidcCookieName, Value: cookie})
	}
	if token != "" {
		r.Header.Set("X-Auth-Token", token)
	}
	w := httptest.NewRecorder()
	handler(w, r)
	return w
}

func addOIDCPending(h *OIDCHandler, p models.OIDCProvider, ticket, action, owner, session string) {
	h.pending[ticket] = &oidcPending{flow: oidcFlow{provider: p, browserHash: hashToken("browser"), action: action, ownerID: owner, ownerSession: session, expires: time.Now().Add(time.Minute)}, claims: services.OIDCClaims{Issuer: p.Issuer, Subject: "subject", Email: "member@example.com", EmailVerified: true}}
	if action == "bind" {
		user, _ := h.store.GetUserByID(owner)
		h.pending[ticket].flow.ownerPasswordHash = user.PasswordHash
	}
}

func TestOIDCHandoffIsBrowserBoundAndSingleUse(t *testing.T) {
	h, p := testOIDCHandler(t)
	addOIDCPending(h, p, "ticket", "login", "", "")
	w := oidcTestRequest(t, h, "ticket", "wrong-browser", "", nil, h.Complete)
	if w.Code != 400 {
		t.Fatalf("wrong browser: %d", w.Code)
	}
	w = oidcTestRequest(t, h, "ticket", "browser", "", nil, h.Complete)
	if w.Code != 200 || !bytes.Contains(w.Body.Bytes(), []byte(`"token"`)) {
		t.Fatalf("login: %d %s", w.Code, w.Body.String())
	}
	w = oidcTestRequest(t, h, "ticket", "browser", "", nil, h.Complete)
	if w.Code != 400 {
		t.Fatalf("replay: %d", w.Code)
	}
}

func TestOIDCAccountCompletionRestoresAndChecksSession(t *testing.T) {
	h, p := testOIDCHandler(t)
	uid := h.store.AdminUserID()
	token, err := h.admin.createSession(uid, h.admin.currentAuthEpoch())
	if err != nil {
		t.Fatal(err)
	}
	otherToken, err := h.admin.createSession(uid, h.admin.currentAuthEpoch())
	if err != nil {
		t.Fatal(err)
	}
	addOIDCPending(h, p, "ticket", "bind", uid, token)
	for _, invalid := range []string{"", otherToken} {
		w := oidcTestRequest(t, h, "ticket", "browser", invalid, nil, h.Complete)
		if w.Code != 401 {
			t.Fatalf("wrong session: %d %s", w.Code, w.Body.String())
		}
	}
	// Call the same public handler that main.go mounts, with no middleware context.
	w := oidcTestRequest(t, h, "ticket", "browser", token, nil, h.Complete)
	if w.Code != 200 || len(h.store.ListOIDCIdentities(uid)) != 1 {
		t.Fatalf("bind: %d %s", w.Code, w.Body.String())
	}
	addOIDCPending(h, p, "password", "password", uid, token)
	w = oidcTestRequest(t, h, "password", "browser", token, nil, h.Complete)
	if w.Code != 200 || !bytes.Contains(w.Body.Bytes(), []byte(`"password_setup_required":true`)) {
		t.Fatalf("password handoff: %d %s", w.Code, w.Body.String())
	}
}

func TestOIDCRequiresSiteEmailVerificationForNewAccounts(t *testing.T) {
	h, p := testOIDCHandler(t)
	if err := h.store.SetSMTPSettings(models.SMTPSettings{Host: "localhost", Port: 2525, From: "panel@example.com"}); err != nil {
		t.Fatal(err)
	}
	addOIDCPending(h, p, "ticket", "login", "", "")
	w := oidcTestRequest(t, h, "ticket", "browser", "", nil, h.Complete)
	if w.Code != 202 || !bytes.Contains(w.Body.Bytes(), []byte(`"email_verification_required":true`)) {
		t.Fatalf("email step: %d %s", w.Code, w.Body.String())
	}
	if _, exists, _ := h.store.OIDCAccount(p.ID, p.Issuer, "subject", "member@example.com"); exists {
		t.Fatal("account created before site email verification")
	}
	code := "123456"
	sum := sha256.Sum256([]byte(code))
	h.store.PutVerifyCode("member@example.com", "oidc:ticket", hex.EncodeToString(sum[:]), time.Minute)
	w = oidcTestRequest(t, h, "ticket", "browser", "", map[string]string{"code": "000000"}, h.Complete)
	if w.Code != 400 {
		t.Fatalf("wrong code: %d", w.Code)
	}
	// Verification codes are consumed on an incorrect attempt as well; a
	// fresh email is required before retrying, matching password registration.
	h.store.PutVerifyCode("member@example.com", "oidc:ticket", hex.EncodeToString(sum[:]), time.Minute)
	w = oidcTestRequest(t, h, "ticket", "browser", "", map[string]string{"code": code}, h.Complete)
	if w.Code != 200 {
		t.Fatalf("verified login: %d %s", w.Code, w.Body.String())
	}
	u, exists, _ := h.store.OIDCAccount(p.ID, p.Issuer, "subject", "")
	if !exists || !u.EmailVerified {
		t.Fatal("email proof not persisted")
	}
}

func TestOIDCKeepsLocalTwoFactor(t *testing.T) {
	h, p := testOIDCHandler(t)
	uid := h.store.AdminUserID()
	if err := h.store.SetUserEmail(uid, "member@example.com", true); err != nil {
		t.Fatal(err)
	}
	encrypted, err := auth.EncryptTOTPSecret(h.key, []byte("12345678901234567890"))
	if err != nil {
		t.Fatal(err)
	}
	if err := h.store.EnableTOTP(uid, encrypted, nil, 0); err != nil {
		t.Fatal(err)
	}
	addOIDCPending(h, p, "ticket", "login", "", "")
	w := oidcTestRequest(t, h, "ticket", "browser", "", nil, h.Complete)
	if w.Code != 202 || !bytes.Contains(w.Body.Bytes(), []byte(`"two_factor_required":true`)) || bytes.Contains(w.Body.Bytes(), []byte(`"token":`)) {
		t.Fatalf("two factor: %d %s", w.Code, w.Body.String())
	}
}

func TestOIDCInvalidatesExpiredOrChangedConfiguration(t *testing.T) {
	for _, reason := range []string{"expired", "changed"} {
		t.Run(reason, func(t *testing.T) {
			h, p := testOIDCHandler(t)
			addOIDCPending(h, p, "ticket", "login", "", "")
			if reason == "expired" {
				h.pending["ticket"].flow.expires = time.Now().Add(-time.Second)
			} else {
				p.Enabled = false
				if _, err := h.store.SaveOIDCProvider(p); err != nil {
					t.Fatal(err)
				}
			}
			w := oidcTestRequest(t, h, "ticket", "browser", "", nil, h.Complete)
			if w.Code != 400 {
				t.Fatalf("invalidated: %d %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestOIDCClientSecretSurvivesUnboundClientEdit(t *testing.T) {
	h, p := testOIDCHandler(t)
	encrypted, err := auth.EncryptSecret(h.key, oidcSecretPurpose(p.Issuer+"|"+p.ClientID), []byte("private-secret"))
	if err != nil {
		t.Fatal(err)
	}
	p.ClientSecret = encrypted
	if _, err := h.store.SaveOIDCProvider(p); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(map[string]any{"name": p.Name, "issuer": p.Issuer, "client_id": "new-panel", "redirect_uri": "https://panel.example.com/api/auth/oidc/callback", "scopes": "openid email", "enabled": true})
	router := chi.NewRouter()
	router.Put("/providers/{id}", h.SaveProvider)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/providers/"+p.ID, bytes.NewReader(raw)))
	if w.Code != 200 || bytes.Contains(w.Body.Bytes(), []byte("private-secret")) || bytes.Contains(w.Body.Bytes(), []byte(encrypted)) {
		t.Fatalf("secret projection: %d %s", w.Code, w.Body.String())
	}
	saved, _ := h.store.GetOIDCProvider(p.ID)
	plain, err := auth.DecryptSecret(h.key, oidcSecretPurpose(saved.Issuer+"|"+saved.ClientID), saved.ClientSecret)
	if err != nil || string(plain) != "private-secret" {
		t.Fatalf("edited secret: %v", err)
	}
}

func TestOIDCBindingRequiresCurrentPassword(t *testing.T) {
	for name, password := range map[string]string{"missing": "", "wrong": "wrong-password", "oversized": strings.Repeat("x", maxPasswordLength+1)} {
		t.Run(name, func(t *testing.T) {
			h, p := testOIDCHandler(t)
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { requests++; http.Error(w, "unexpected discovery", 500) }))
			defer server.Close()
			p.Issuer, p.RedirectURI = server.URL, server.URL+"/callback"
			p, err := h.store.SaveOIDCProvider(p)
			if err != nil {
				t.Fatal(err)
			}
			uid := h.store.AdminUserID()
			token, err := h.admin.createSession(uid, h.admin.currentAuthEpoch())
			if err != nil {
				t.Fatal(err)
			}
			user, _ := h.admin.ValidateSession(token)
			body := map[string]string{"provider_id": p.ID, "action": "bind"}
			if password != "" {
				body["password"] = password
			}
			raw, _ := json.Marshal(body)
			r := httptest.NewRequest(http.MethodPost, "/api/account/oidc/start", bytes.NewReader(raw))
			r.Header.Set("X-Auth-Token", token)
			w := httptest.NewRecorder()
			h.AccountStart(w, withUser(r, user))
			if w.Code != http.StatusForbidden || requests != 0 || len(h.flows) != 0 {
				t.Fatalf("binding without current password: %d %s, discovery=%d", w.Code, w.Body.String(), requests)
			}
		})
	}
}

func TestOIDCBindingRejectsPasswordChangeDuringAuthorization(t *testing.T) {
	h, p := testOIDCHandler(t)
	uid := h.store.AdminUserID()
	token, err := h.admin.createSession(uid, h.admin.currentAuthEpoch())
	if err != nil {
		t.Fatal(err)
	}
	addOIDCPending(h, p, "ticket", "bind", uid, token)
	if err := h.store.SetUserPasswordHash(uid, store.HashPassword("changed-password")); err != nil {
		t.Fatal(err)
	}
	w := oidcTestRequest(t, h, "ticket", "browser", token, nil, h.Complete)
	if w.Code != 400 || len(h.store.ListOIDCIdentities(uid)) != 0 {
		t.Fatalf("binding survived password reset: %d %s", w.Code, w.Body.String())
	}
}
