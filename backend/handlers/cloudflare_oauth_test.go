package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"tunnel-manager/auth"
	"tunnel-manager/models"
	"tunnel-manager/services"
	"tunnel-manager/store"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestCloudflareOAuthRedirectURIUsesForwardedHeaders(t *testing.T) {
	request := httptest.NewRequest("POST", "http://backend:8080/api/cloudflare/oauth/start", nil)
	request.Header.Set("X-Forwarded-Proto", "https, http")
	request.Header.Set("X-Forwarded-Host", "panel.example.com, proxy.internal")
	if got := cloudflareOAuthRedirectURI(request); got != "https://panel.example.com/api/cloudflare/oauth/callback" {
		t.Fatalf("cloudflareOAuthRedirectURI() = %q", got)
	}
}

func TestCloudflareOAuthRedirectResultUsesAccountPage(t *testing.T) {
	for _, test := range []struct {
		name     string
		result   string
		message  string
		location string
	}{
		{name: "success", result: "success", location: "/account?cloudflare_oauth=success"},
		{name: "error", result: "error", message: "授权失败", location: "/account?cloudflare_oauth=error&message=%E6%8E%88%E6%9D%83%E5%A4%B1%E8%B4%A5"},
	} {
		t.Run(test.name, func(t *testing.T) {
			handler := &CloudflareOAuthHandler{}
			request := httptest.NewRequest("GET", "http://backend/api/cloudflare/oauth/callback", nil)
			response := httptest.NewRecorder()

			handler.redirectResult(response, request, test.result, test.message)

			if got := response.Header().Get("Location"); got != test.location {
				t.Fatalf("Location = %q, want %q", got, test.location)
			}
		})
	}
}

func TestCloudflareOAuthStatusUsesRefreshedExpiry(t *testing.T) {
	initialExpiry := time.Now().Add(30 * time.Second)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/oauth2/token":
			if err := r.ParseForm(); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if r.Form.Get("grant_type") != "refresh_token" || r.Form.Get("refresh_token") != "refresh-token" {
				http.Error(w, "invalid refresh request", http.StatusBadRequest)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token":  "refreshed-access-token",
				"refresh_token": "rotated-refresh-token",
				"expires_in":    3600,
			})
		case "/client/v4/accounts":
			if r.Header.Get("Authorization") != "Bearer refreshed-access-token" {
				http.Error(w, "unexpected access token", http.StatusUnauthorized)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"result":  []models.Account{{ID: "account-id", Name: "Account"}},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	target, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}
	originalTransport := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		request = request.Clone(request.Context())
		request.URL.Scheme = target.Scheme
		request.URL.Host = target.Host
		return originalTransport.RoundTrip(request)
	})
	t.Cleanup(func() { http.DefaultTransport = originalTransport })

	t.Setenv("ADMIN_PASSWORD", "test-password")
	st := store.NewStore(t.TempDir() + "/config.json")
	key := bytes.Repeat([]byte{4}, 32)
	accessToken, err := auth.EncryptSecret(key, "cloudflare-oauth-access-token", []byte("initial-access-token"))
	if err != nil {
		t.Fatalf("EncryptSecret(access token) error = %v", err)
	}
	refreshToken, err := auth.EncryptSecret(key, "cloudflare-oauth-refresh-token", []byte("refresh-token"))
	if err != nil {
		t.Fatalf("EncryptSecret(refresh token) error = %v", err)
	}
	if err := st.SetCloudflareOAuth(accessToken, refreshToken, initialExpiry, "account.read"); err != nil {
		t.Fatalf("SetCloudflareOAuth() error = %v", err)
	}
	if err := st.SetCloudflareAccount("account-id", "Account"); err != nil {
		t.Fatalf("SetCloudflareAccount() error = %v", err)
	}

	oauth := services.NewCloudflareOAuth(st, key, services.CloudflareOAuthConfig{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
	})
	cf := services.NewCloudflareClient("", "")
	cf.SetOAuth(oauth)
	handler := NewCloudflareOAuthHandler(st, oauth, cf, nil)
	request := httptest.NewRequest(http.MethodGet, "http://backend/api/cloudflare/oauth/status", nil)
	response := httptest.NewRecorder()

	handler.Status(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("Status() code = %d, body = %s", response.Code, response.Body.String())
	}
	var status models.CloudflareOAuthStatus
	if err := json.NewDecoder(response.Body).Decode(&status); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	storedConfig := st.GetConfig()
	wantExpiry := time.Unix(storedConfig.CFOAuthExpiresAt, 0).Format(time.RFC3339)
	if status.ExpiresAt != wantExpiry {
		t.Fatalf("expires_at = %q, want %q", status.ExpiresAt, wantExpiry)
	}
	if storedConfig.CFOAuthExpiresAt <= initialExpiry.Unix() {
		t.Fatalf("stored expiry = %d, want later than %d", storedConfig.CFOAuthExpiresAt, initialExpiry.Unix())
	}
	if status.Source != "oauth" || len(status.Accounts) != 1 || status.Accounts[0].ID != "account-id" {
		t.Fatalf("unexpected status: %+v", status)
	}
}

func TestSelectCloudflareAccountUsesPreferredThenFirst(t *testing.T) {
	accounts := []models.Account{{ID: "first", Name: "First"}, {ID: "preferred", Name: "Preferred"}}
	if got := selectCloudflareAccount(accounts, "missing", "preferred"); got.ID != "preferred" {
		t.Fatalf("selectCloudflareAccount() = %q", got.ID)
	}
	if got := selectCloudflareAccount(accounts, "missing"); got.ID != "first" {
		t.Fatalf("selectCloudflareAccount() fallback = %q", got.ID)
	}
}
