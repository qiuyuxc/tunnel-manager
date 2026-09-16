package services

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"tunnel-manager/auth"
	"tunnel-manager/models"
	"tunnel-manager/store"
)

// TestStaticCredentialsOnlyFallBackForAdmin guards multi-user isolation: a
// registered user without their own Cloudflare OAuth connection must never
// resolve the administrator's static token or account id, otherwise they
// could list and manage the administrator's tunnels.
func TestStaticCredentialsOnlyFallBackForAdmin(t *testing.T) {
	st := store.NewStore(filepath.Join(t.TempDir(), "config.json"))
	adminID := st.AdminUserID()
	if err := st.CreateUser(models.User{Username: "alice", Email: "alice@example.com", Role: models.RoleUser}); err != nil {
		t.Fatal(err)
	}
	alice, ok := st.GetUserByUsername("alice")
	if !ok {
		t.Fatal("alice not found")
	}

	cf := NewCloudflareClient("admin-token", "admin-account")
	cf.SetSessionStore(st)

	// A registered user without an OAuth connection must not receive the
	// administrator's static credentials.
	userCF := cf.ForUser(alice.ID)
	if token, err := userCF.accessToken(); err == nil || !strings.Contains(err.Error(), "尚未授权") {
		t.Fatalf("unbound user accessToken() = %q, %v; want 尚未授权 error", token, err)
	}
	if accountID, err := userCF.currentAccountID(); err == nil || !strings.Contains(err.Error(), "尚未授权") {
		t.Fatalf("unbound user currentAccountID() = %q, %v; want 尚未授权 error", accountID, err)
	}

	// The administrator keeps the legacy static fallback.
	adminCF := cf.ForUser(adminID)
	if token, err := adminCF.accessToken(); err != nil || token != "admin-token" {
		t.Fatalf("admin accessToken() = %q, %v; want admin-token", token, err)
	}
	if accountID, err := adminCF.currentAccountID(); err != nil || accountID != "admin-account" {
		t.Fatalf("admin currentAccountID() = %q, %v; want admin-account", accountID, err)
	}

	// The API key identity (no user id) keeps full static reach.
	if token, err := cf.accessToken(); err != nil || token != "admin-token" {
		t.Fatalf("api key accessToken() = %q, %v; want admin-token", token, err)
	}
	if accountID, err := cf.currentAccountID(); err != nil || accountID != "admin-account" {
		t.Fatalf("api key currentAccountID() = %q, %v; want admin-account", accountID, err)
	}

	// A user with their own connection uses their own token and account.
	key := bytes.Repeat([]byte{9}, 32)
	accessToken, err := auth.EncryptSecret(key, "cloudflare-oauth-access-token", []byte("alice-token"))
	if err != nil {
		t.Fatal(err)
	}
	connID, err := st.CreateCFConnection(models.CFConnection{
		UserID:      alice.ID,
		Label:       "alice-conn",
		AccountID:   "alice-account",
		AccessToken: accessToken,
		ExpiresAt:   time.Now().Add(time.Hour).Unix(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.SetActiveCFConnection(alice.ID, connID); err != nil {
		t.Fatal(err)
	}
	oauth := NewCloudflareOAuth(st, key, CloudflareOAuthConfig{ClientID: "client-id", ClientSecret: "client-secret"})
	boundCF := cf.ForUser(alice.ID)
	boundCF.oauth = oauth
	token, err := boundCF.accessToken()
	if err != nil || token != "alice-token" {
		t.Fatalf("bound user accessToken() = %q, %v; want alice-token", token, err)
	}
	accountID, err := boundCF.currentAccountID()
	if err != nil || accountID != "alice-account" {
		t.Fatalf("bound user currentAccountID() = %q, %v; want alice-account", accountID, err)
	}
}

// TestDeadOAuthConnectionFallsBackToStaticCredentials covers a grant Cloudflare
// has revoked: the connection can no longer mint a token, but the
// administrator's environment credentials still work, and without the fallback
// every Cloudflare-backed page answers 500.
func TestDeadOAuthConnectionFallsBackToStaticCredentials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error":             "invalid_grant",
			"error_description": "The provided authorization grant or refresh token is invalid.",
		})
	}))
	defer server.Close()

	st := newOAuthTestStore(t)
	key := bytes.Repeat([]byte{3}, 32)
	oauth := NewCloudflareOAuth(st, key, CloudflareOAuthConfig{ClientID: "client-id", ClientSecret: "client-secret"})
	oauth.httpClient = server.Client()
	oauth.tokenEndpoint = server.URL + "/token"
	oauth.now = func() time.Time { return time.Unix(1_800_000_000, 0) }

	access, err := auth.EncryptSecret(key, cloudflareAccessTokenPurpose, []byte("expired-access"))
	if err != nil {
		t.Fatal(err)
	}
	refresh, err := auth.EncryptSecret(key, cloudflareRefreshTokenPurpose, []byte("revoked-refresh"))
	if err != nil {
		t.Fatal(err)
	}
	deadConn := func(userID, label string) models.CFConnection {
		return models.CFConnection{
			UserID:       userID,
			Label:        label,
			AccountID:    label + "-account",
			AccessToken:  access,
			RefreshToken: refresh,
			ExpiresAt:    1_799_990_000, // already past oauth.now()
		}
	}

	adminID := st.AdminUserID()
	adminConn, err := st.CreateCFConnection(deadConn(adminID, "admin"))
	if err != nil {
		t.Fatal(err)
	}
	if err := st.SetActiveCFConnection(adminID, adminConn); err != nil {
		t.Fatal(err)
	}

	cf := NewCloudflareClient("admin-token", "admin-account")
	cf.SetOAuth(oauth)
	cf.SetSessionStore(st)

	// The administrator keeps working on the environment credentials.
	token, err := cf.ForUser(adminID).accessToken()
	if err != nil {
		t.Fatalf("admin accessToken() error = %v; want the static credential fallback", err)
	}
	if token != "admin-token" {
		t.Fatalf("admin accessToken() = %q; want the static credential", token)
	}

	// A registered user with an equally dead connection gets the error: the
	// fallback belongs to the administrator, and handing it out would expose
	// their Cloudflare account to everyone.
	if err := st.CreateUser(models.User{Username: "bob", Email: "bob@example.com", Role: models.RoleUser}); err != nil {
		t.Fatal(err)
	}
	bob, ok := st.GetUserByUsername("bob")
	if !ok {
		t.Fatal("bob not found")
	}
	bobConn, err := st.CreateCFConnection(deadConn(bob.ID, "bob"))
	if err != nil {
		t.Fatal(err)
	}
	if err := st.SetActiveCFConnection(bob.ID, bobConn); err != nil {
		t.Fatal(err)
	}
	if token, err := cf.ForUser(bob.ID).accessToken(); err == nil {
		t.Fatalf("bob accessToken() = %q, nil; want an error rather than the administrator's credential", token)
	}
}
