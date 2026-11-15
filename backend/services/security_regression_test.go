package services

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"tunnel-manager/auth"
	"tunnel-manager/models"
)

func TestSecurityRegressionTelegramEnforcesGroupAndDisabledStatus(t *testing.T) {
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
	if reads.Load() != 0 {
		t.Fatalf("permission bypass, reads=%d", reads.Load())
	}
	group.Permissions = append(group.Permissions, models.PermTunnels)
	if err := st.UpdateGroup(group.ID, group.Name, group.Permissions); err != nil {
		t.Fatal(err)
	}
	bot.handleUpdate(update)
	if reads.Load() != 1 {
		t.Fatalf("authorized command didn't run: %d", reads.Load())
	}
	if err := st.SetUserStatus("review-user", models.UserDisabled); err != nil {
		t.Fatal(err)
	}
	bot.handleUpdate(update)
	if reads.Load() != 1 {
		t.Fatalf("disabled account executed command, reads=%d", reads.Load())
	}
}

func TestPasskeyCeremonyCapacityIsSharedFairly(t *testing.T) {
	svc, _ := newPasskeyTestService(t)
	request := httptest.NewRequest("POST", "https://panel.example.com/api/auth/passkey/login/begin", nil)
	request.RemoteAddr = "198.51.100.1:1234"
	for i := 0; i < 8; i++ {
		if _, _, _, err := svc.BeginDiscoverableLogin(request); err != nil {
			t.Fatal(err)
		}
	}
	request.RemoteAddr = "198.51.100.1:5678"
	request.Header.Set("X-Forwarded-For", "203.0.113.5")
	if _, _, _, err := svc.BeginDiscoverableLogin(request); err == nil {
		t.Fatal("source can exhaust shared ceremony capacity")
	}
	request.RemoteAddr = "198.51.100.2:5678"
	if _, _, _, err := svc.BeginDiscoverableLogin(request); err != nil {
		t.Fatalf("other source blocked: %v", err)
	}
}

func TestTelegramPermissionAliasesAndDelayedDelete(t *testing.T) {
	st := newTestStore(t)
	group, err := st.CreateGroup("no-permissions", []string{})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.CreateUser(models.User{ID: "owner", Username: "owner", GroupID: group.ID}); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{"ok":true}`)) }))
	defer server.Close()
	st.SetTelegramAPIEndpoint(server.URL)
	bot := NewUserTelegramBot(st, NewCloudflareClient("", ""), NewDomainService(NewCloudflareClient("", ""), st), "owner", "123:MOCK", "42", "webhook", "", "secret")
	cfg := bot.settings()
	for _, command := range []string{"/tunnels", "/列出隧道", "/dns_add", "/DNS添加", "/dns_delete", "/DNS删除", "/dns_confirm", "/确认删除", "/bind_direct", "/直连域名", "/bind_preferred", "/优选绑定", "/绑定域名", "/设置回退源"} {
		if bot.authorizeCommand(cfg, 42, command) {
			t.Errorf("unauthorized alias: %s", command)
		}
	}
	if err := st.UpdateGroup(group.ID, group.Name, []string{models.PermDNS}); err != nil {
		t.Fatal(err)
	}
	if !bot.authorizeCommand(cfg, 42, "/dns_confirm") {
		t.Fatal("authorized confirmation blocked")
	}
	if err := st.UpdateGroup(group.ID, group.Name, nil); err != nil {
		t.Fatal(err)
	}
	if bot.authorizeCommand(cfg, 42, "/dns_confirm") {
		t.Fatal("confirmation kept old permissions")
	}
}

func TestBeginBudgetIsAtomicAndBounded(t *testing.T) {
	limiter := NewRateLimiter()
	var accepted atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if ok, _ := limiter.Take("same-source", 3, time.Minute); ok {
				accepted.Add(1)
			}
		}()
	}
	wg.Wait()
	if accepted.Load() != 3 {
		t.Fatalf("concurrent admission: %d", accepted.Load())
	}
	for i := 0; i < 1000; i++ {
		if ok, _ := limiter.Take("large-budget", 1000, time.Minute); !ok {
			t.Fatalf("large budget stopped at %d", i)
		}
	}
	if ok, _ := limiter.Take("large-budget", 1000, time.Minute); ok {
		t.Fatal("large budget exceeded")
	}
	if len(limiter.entries["large-budget"].failures) != 0 {
		t.Fatal("request budget grows failure history")
	}
}
