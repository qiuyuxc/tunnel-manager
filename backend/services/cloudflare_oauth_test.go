package services

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"tunnel-manager/auth"
	"tunnel-manager/store"
)

func TestCloudflareOAuthAuthorizationURLUsesPKCE(t *testing.T) {
	oauth := NewCloudflareOAuth(newOAuthTestStore(t), bytes.Repeat([]byte{1}, 32), CloudflareOAuthConfig{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		Scopes:       "tunnel.write zone.read",
	})
	authorizationURL, err := oauth.AuthorizationURL("https://example.com/callback", "state-value", "challenge-value")
	if err != nil {
		t.Fatalf("AuthorizationURL() error = %v", err)
	}
	parsed, err := url.Parse(authorizationURL)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}
	query := parsed.Query()
	for key, want := range map[string]string{
		"response_type":         "code",
		"client_id":             "client-id",
		"redirect_uri":          "https://example.com/callback",
		"state":                 "state-value",
		"scope":                 "tunnel.write zone.read",
		"code_challenge":        "challenge-value",
		"code_challenge_method": "S256",
	} {
		if got := query.Get(key); got != want {
			t.Fatalf("query %s = %q, want %q", key, got, want)
		}
	}
}

func TestCloudflareOAuthExchangeRefreshAndRevoke(t *testing.T) {
	var tokenRequests atomic.Int32
	var revokeRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok || username != "client-id" || password != "client-secret" {
			http.Error(w, "invalid client authentication", http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/token":
			if err := r.ParseForm(); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			requestNumber := tokenRequests.Add(1)
			w.Header().Set("Content-Type", "application/json")
			if requestNumber == 1 {
				if r.Form.Get("grant_type") != "authorization_code" || r.Form.Get("code") != "code" || r.Form.Get("code_verifier") != "verifier" {
					http.Error(w, "invalid authorization code exchange", http.StatusBadRequest)
					return
				}
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"access_token":  "initial-access",
					"refresh_token": "refresh-token",
					"expires_in":    30,
					"scope":         "tunnel.write",
				})
				return
			}
			if r.Form.Get("grant_type") != "refresh_token" || r.Form.Get("refresh_token") != "refresh-token" {
				http.Error(w, "invalid refresh", http.StatusBadRequest)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token": "refreshed-access",
				"expires_in":   3600,
			})
		case "/revoke":
			revokeRequests.Add(1)
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	st := newOAuthTestStore(t)
	oauth := NewCloudflareOAuth(st, bytes.Repeat([]byte{2}, 32), CloudflareOAuthConfig{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
	})
	oauth.httpClient = server.Client()
	oauth.tokenEndpoint = server.URL + "/token"
	oauth.revokeEndpoint = server.URL + "/revoke"
	oauth.now = func() time.Time { return time.Unix(1_800_000_000, 0) }

	if err := oauth.ExchangeCode("code", "https://example.com/callback", "verifier"); err != nil {
		t.Fatalf("ExchangeCode() error = %v", err)
	}
	config := st.GetConfig()
	if config.CFOAuthAccessToken == "" || strings.Contains(config.CFOAuthAccessToken, "initial-access") {
		t.Fatalf("stored access token is not encrypted: %q", config.CFOAuthAccessToken)
	}
	if config.CFOAuthRefreshToken == "" || strings.Contains(config.CFOAuthRefreshToken, "refresh-token") {
		t.Fatalf("stored refresh token is not encrypted: %q", config.CFOAuthRefreshToken)
	}

	accessToken, err := oauth.AccessToken()
	if err != nil {
		t.Fatalf("AccessToken() error = %v", err)
	}
	if accessToken != "refreshed-access" {
		t.Fatalf("AccessToken() = %q", accessToken)
	}
	if tokenRequests.Load() != 2 {
		t.Fatalf("token requests = %d, want 2", tokenRequests.Load())
	}
	if err := st.SetCloudflareAccount("account-id", "Account"); err != nil {
		t.Fatalf("SetCloudflareAccount() error = %v", err)
	}
	if err := st.SetTunnelSelection("tunnel-id", "Tunnel"); err != nil {
		t.Fatalf("SetTunnelSelection() error = %v", err)
	}

	if err := oauth.RevokeAndClear(); err != nil {
		t.Fatalf("RevokeAndClear() error = %v", err)
	}
	if revokeRequests.Load() != 1 {
		t.Fatalf("revoke requests = %d, want 1", revokeRequests.Load())
	}
	config = st.GetConfig()
	if config.CFOAuthAccessToken != "" || config.CFOAuthRefreshToken != "" || config.CFAccountID != "" || config.TunnelID != "" {
		t.Fatal("RevokeAndClear() did not clear OAuth connection state")
	}
}

func TestCloudflareOAuthAccessTokenWithoutRefreshToken(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	for _, test := range []struct {
		name      string
		expiresAt time.Time
		wantToken string
		wantError string
	}{
		{name: "inside refresh window", expiresAt: now.Add(30 * time.Second), wantToken: "access-token"},
		{name: "expired", expiresAt: now, wantError: "expired and no refresh token"},
	} {
		t.Run(test.name, func(t *testing.T) {
			st := newOAuthTestStore(t)
			key := bytes.Repeat([]byte{3}, 32)
			encrypted, err := auth.EncryptSecret(key, cloudflareAccessTokenPurpose, []byte("access-token"))
			if err != nil {
				t.Fatalf("EncryptSecret() error = %v", err)
			}
			if err := st.SetCloudflareOAuth(encrypted, "", test.expiresAt, "zone.read"); err != nil {
				t.Fatalf("SetCloudflareOAuth() error = %v", err)
			}
			oauth := NewCloudflareOAuth(st, key, CloudflareOAuthConfig{})
			oauth.now = func() time.Time { return now }

			token, err := oauth.AccessToken()
			if test.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantError) {
					t.Fatalf("AccessToken() error = %v, want containing %q", err, test.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("AccessToken() error = %v", err)
			}
			if token != test.wantToken {
				t.Fatalf("AccessToken() = %q, want %q", token, test.wantToken)
			}
		})
	}
}

func newOAuthTestStore(t *testing.T) *store.Store {
	t.Helper()
	t.Setenv("ADMIN_PASSWORD", "test-password")
	return store.NewStore(t.TempDir() + "/config.json")
}
