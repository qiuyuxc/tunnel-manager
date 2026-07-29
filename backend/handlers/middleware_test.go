package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSessionOnlyRejectsAPIKeyHeaderAndQuery(t *testing.T) {
	h := newTestAdminHandler(t)
	mw := &Middleware{APIKey: "service-key", AdminHandler: h}
	called := false
	protected := mw.SessionOnly(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})

	for _, target := range []string{"/admin/2fa/status", "/admin/2fa/status?api_key=service-key"} {
		req := httptest.NewRequest(http.MethodGet, target, nil)
		req.Header.Set("X-API-Key", "service-key")
		resp := httptest.NewRecorder()
		protected(resp, req)
		if resp.Code != http.StatusUnauthorized || called {
			t.Fatalf("SessionOnly(%q) = %d, called %v", target, resp.Code, called)
		}
	}
}

func TestSessionOnlyRequiresExactSessionTokenFormat(t *testing.T) {
	h := newTestAdminHandler(t)
	mw := &Middleware{APIKey: "service-key", AdminHandler: h}
	protected := mw.SessionOnly(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
	token := login(t, h, "admin", "password")

	for _, candidate := range []string{token, strings.ToUpper(token), token[:63], token + "0"} {
		req := httptest.NewRequest(http.MethodGet, "/admin/2fa/status", nil)
		req.Header.Set("X-Auth-Token", candidate)
		resp := httptest.NewRecorder()
		protected(resp, req)
		want := http.StatusUnauthorized
		if candidate == token {
			want = http.StatusNoContent
		}
		if resp.Code != want {
			t.Fatalf("token %q status = %d, want %d", candidate, resp.Code, want)
		}
	}
}
