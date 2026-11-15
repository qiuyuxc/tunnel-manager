package services

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"tunnel-manager/auth"
	"tunnel-manager/models"
)

func TestSecurityReviewTelegramBypassesGroupAndDisabledStatus(t *testing.T) {
	st := newTestStore(t)
	group, err := st.CreateGroup("oauth-only-review", []string{models.PermOAuthConnect})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.CreateUser(models.User{ID: "review-user", Username: "review-user", Role: models.RoleUser, GroupID: group.ID}); err != nil {
		t.Fatal(err)
	}
	var reads atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/cfd_tunnel") {
			reads.Add(1)
			json.NewEncoder(w).Encode(map[string]any{"success": true, "result": []models.Tunnel{{ID: "review-tunnel", Name: "review"}}})
			return
		}
		w.Write([]byte(`{"ok":true,"result":[]}`))
	}))
	defer server.Close()
	cf := NewCloudflareClient("mock-token", "mock-account")
	cf.baseURL = server.URL
	cf.httpClient = server.Client()
	key := bytes.Repeat([]byte{8}, 32)
	token, err := auth.EncryptSecret(key, "cloudflare-oauth-access-token", []byte("mock-token"))
	if err != nil {
		t.Fatal(err)
	}
	connection, err := st.CreateCFConnection(models.CFConnection{UserID: "review-user", AccountID: "mock-account", AccessToken: token, ExpiresAt: time.Now().Add(time.Hour).Unix()})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.SetActiveCFConnection("review-user", connection); err != nil {
		t.Fatal(err)
	}
	cf.SetSessionStore(st)
	cf.SetOAuth(NewCloudflareOAuth(st, key, CloudflareOAuthConfig{ClientID: "mock", ClientSecret: "mock"}))
	st.SetTelegramAPIEndpoint(server.URL)
	bot := NewUserTelegramBot(st, cf, NewDomainService(cf, st), "review-user", "123:MOCK", "42", "webhook", "https://panel.example.com", "secret")
	bot.botUsername = "review_bot"
	var update tgUpdate
	if err := json.Unmarshal([]byte(`{"message":{"chat":{"id":42},"from":{"id":42},"text":"/tunnels"}}`), &update); err != nil {
		t.Fatal(err)
	}
	bot.handleUpdate(update)
	if reads.Load() != 1 {
		t.Fatalf("expected permission bypass, reads=%d", reads.Load())
	}
	if err := st.SetUserStatus("review-user", models.UserDisabled); err != nil {
		t.Fatal(err)
	}
	bot.handleUpdate(update)
	if reads.Load() != 2 {
		t.Fatalf("expected disabled account to keep executing, reads=%d", reads.Load())
	}
	t.Log("user group lacks tunnels permission; /tunnels still invokes CF API; disabling account does not stop next command")
}
