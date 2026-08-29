package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"tunnel-manager/auth"
	"tunnel-manager/models"
	"tunnel-manager/services"
	"tunnel-manager/store"
)

// encryptTGToken encrypts a bot token the same way SaveSettings does.
func encryptTGToken(key []byte, token string) (string, error) {
	return auth.EncryptSecret(key, notifyTGTokenPurpose, []byte(token))
}

// newTelegramTestHandler builds a TelegramHandler with an isolated store and
// the per-user bot manager; no bot is started until the test reconciles.
func newTelegramTestHandler(t *testing.T) (*TelegramHandler, *store.Store, *services.UserTelegramManager, *services.TelegramBot, []byte) {
	t.Helper()
	st := store.NewStore(filepath.Join(t.TempDir(), "config.json"))
	key := bytes.Repeat([]byte{9}, 32)
	cf := services.NewCloudflareClient("token", "acct")
	ds := services.NewDomainService(cf, st)
	legacyBot := services.NewTelegramBot(st, cf, ds)
	manager := services.NewUserTelegramManager(st, cf, ds, key)
	return NewTelegramHandler(st, legacyBot, manager, key), st, manager, legacyBot, key
}

// telegramAPITestServer answers the Telegram Bot API endpoints used by bot
// start/dispatch and records sendMessage paths.
func telegramAPITestServer(t *testing.T, onRequest func(path string)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if onRequest != nil {
			onRequest(r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/getMe"):
			_, _ = w.Write([]byte(`{"ok":true,"result":{"username":"test_bot"}}`))
		default:
			_, _ = w.Write([]byte(`{"ok":true,"result":[]}`))
		}
	}))
}

// telegramWebhookRouter wires the per-user webhook route so chi populates the
// {userID} URL parameter.
func telegramWebhookRouter(h *TelegramHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Post("/api/telegram/webhook/{userID}", h.UserWebhook)
	return r
}

func tgUpdateBody(text string) string {
	return `{"update_id":1,"message":{"message_id":1,"from":{"id":7330290970,"is_bot":false,"first_name":"T"},"chat":{"id":7330290970,"type":"private"},"date":1700000000,"text":"` + text + `"}}`
}

func TestUserWebhookRejectionPaths(t *testing.T) {
	h, st, _, _, key := newTelegramTestHandler(t)
	router := telegramWebhookRouter(h)

	adminID := st.AdminUserID()
	enc, err := encryptTGToken(key, "111:TOKENA")
	if err != nil {
		t.Fatal(err)
	}

	// Unknown userID.
	req := httptest.NewRequest(http.MethodPost, "/api/telegram/webhook/no-such-user", strings.NewReader(tgUpdateBody("/help")))
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "secret-a")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusForbidden {
		t.Fatalf("unknown user = %d: %s", resp.Code, resp.Body.String())
	}

	// Polling user cannot receive webhooks.
	if err := st.SetUserRemoteSettings(adminID, true, "7330290970", enc, "polling", "", ""); err != nil {
		t.Fatal(err)
	}
	req = httptest.NewRequest(http.MethodPost, "/api/telegram/webhook/"+adminID, strings.NewReader(tgUpdateBody("/help")))
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "whatever")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusForbidden {
		t.Fatalf("polling user = %d: %s", resp.Code, resp.Body.String())
	}

	// Disabled user.
	if err := st.SetUserRemoteSettings(adminID, false, "7330290970", enc, "webhook", "https://panel.example.com", "secret-a"); err != nil {
		t.Fatal(err)
	}
	req = httptest.NewRequest(http.MethodPost, "/api/telegram/webhook/"+adminID, strings.NewReader(tgUpdateBody("/help")))
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "secret-a")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusForbidden {
		t.Fatalf("disabled user = %d: %s", resp.Code, resp.Body.String())
	}

	// Enabled webhook user whose bot never started.
	if err := st.SetUserRemoteSettings(adminID, true, "7330290970", enc, "webhook", "https://panel.example.com", "secret-a"); err != nil {
		t.Fatal(err)
	}
	req = httptest.NewRequest(http.MethodPost, "/api/telegram/webhook/"+adminID, strings.NewReader(tgUpdateBody("/help")))
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "secret-a")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusForbidden {
		t.Fatalf("not-running bot = %d: %s", resp.Code, resp.Body.String())
	}
}

func TestUserWebhookDispatchAndIsolation(t *testing.T) {
	h, st, manager, _, key := newTelegramTestHandler(t)
	router := telegramWebhookRouter(h)

	var mu sync.Mutex
	sent := []string{}
	server := telegramAPITestServer(t, func(path string) {
		if strings.Contains(path, "/sendMessage") {
			mu.Lock()
			sent = append(sent, path)
			mu.Unlock()
		}
	})
	defer server.Close()
	st.SetTelegramAPIEndpoint(server.URL)

	adminID := st.AdminUserID()
	enc1, err := encryptTGToken(key, "111:TOKENA")
	if err != nil {
		t.Fatal(err)
	}
	if err := st.SetUserRemoteSettings(adminID, true, "7330290970", enc1, "webhook", "https://panel.example.com", "secret-a"); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateUser(models.User{Username: "member", PasswordHash: store.HashPassword("password"), Role: models.RoleUser, Status: models.UserActive, EmailVerified: true}); err != nil {
		t.Fatal(err)
	}
	member, ok := st.GetUserByUsername("member")
	if !ok {
		t.Fatal("member not found")
	}
	enc2, err := encryptTGToken(key, "222:TOKENB")
	if err != nil {
		t.Fatal(err)
	}
	if err := st.SetUserRemoteSettings(member.ID, true, "7330290970", enc2, "webhook", "https://panel.example.com", "secret-b"); err != nil {
		t.Fatal(err)
	}

	manager.Reconcile()
	defer manager.StopAll()

	// Wrong secret for the owner is rejected.
	req := httptest.NewRequest(http.MethodPost, "/api/telegram/webhook/"+adminID, strings.NewReader(tgUpdateBody("/help")))
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "wrong-secret")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusForbidden {
		t.Fatalf("wrong secret = %d: %s", resp.Code, resp.Body.String())
	}

	// Correct secret reaches the owner's bot only.
	req = httptest.NewRequest(http.MethodPost, "/api/telegram/webhook/"+adminID, strings.NewReader(tgUpdateBody("/help")))
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "secret-a")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("owner webhook = %d: %s", resp.Code, resp.Body.String())
	}

	// The other account's update goes only to that account's bot.
	req = httptest.NewRequest(http.MethodPost, "/api/telegram/webhook/"+member.ID, strings.NewReader(tgUpdateBody("/help")))
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "secret-b")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("member webhook = %d: %s", resp.Code, resp.Body.String())
	}

	var sawA, sawB bool
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		for _, path := range sent {
			if strings.Contains(path, "/bot111:TOKENA/sendMessage") {
				sawA = true
			}
			if strings.Contains(path, "/bot222:TOKENB/sendMessage") {
				sawB = true
			}
		}
		mu.Unlock()
		if sawA && sawB {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !sawA || !sawB {
		t.Fatalf("updates not isolated per owner, sent = %v", sent)
	}
}

func TestLegacyGlobalWebhookRoute(t *testing.T) {
	h, st, _, legacyBot, _ := newTelegramTestHandler(t)

	// Global bot in webhook mode must keep working through the legacy route.
	var mu sync.Mutex
	sent := []string{}
	server := telegramAPITestServer(t, func(path string) {
		if strings.Contains(path, "/sendMessage") {
			mu.Lock()
			sent = append(sent, path)
			mu.Unlock()
		}
	})
	defer server.Close()
	st.SetTelegramAPIEndpoint(server.URL)
	st.SetTelegramSettings(true, "123:LEGACY", "7330290970", "webhook", "https://panel.example.com", "")
	st.SetTelegramWebhookSecret("legacy-secret")
	if err := legacyBot.Start(); err != nil {
		t.Fatalf("legacy bot start: %v", err)
	}
	defer legacyBot.Stop()

	req := httptest.NewRequest(http.MethodPost, "/api/telegram/webhook", strings.NewReader(tgUpdateBody("/help")))
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "legacy-secret")
	resp := httptest.NewRecorder()
	h.Webhook(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("legacy webhook = %d: %s", resp.Code, resp.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/telegram/webhook", strings.NewReader(tgUpdateBody("/help")))
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "wrong-secret")
	resp = httptest.NewRecorder()
	h.Webhook(resp, req)
	if resp.Code != http.StatusForbidden {
		t.Fatalf("legacy wrong secret = %d: %s", resp.Code, resp.Body.String())
	}

	var sawLegacy bool
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		for _, path := range sent {
			if strings.Contains(path, "/bot123:LEGACY/sendMessage") {
				sawLegacy = true
			}
		}
		mu.Unlock()
		if sawLegacy {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !sawLegacy {
		t.Fatalf("legacy bot did not process webhook update, sent = %v", sent)
	}

	// Global bot in polling mode rejects webhooks.
	legacyBot.Stop()
	st.SetTelegramSettings(true, "123:LEGACY", "7330290970", "polling", "", "")
	if err := legacyBot.Start(); err != nil {
		t.Fatalf("legacy polling start: %v", err)
	}
	req = httptest.NewRequest(http.MethodPost, "/api/telegram/webhook", strings.NewReader(tgUpdateBody("/help")))
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "legacy-secret")
	resp = httptest.NewRecorder()
	h.Webhook(resp, req)
	if resp.Code != http.StatusForbidden {
		t.Fatalf("legacy polling route = %d: %s", resp.Code, resp.Body.String())
	}
}

func TestTelegramSettingsModeValidationAndSecretHiding(t *testing.T) {
	h, st, _, _, _ := newTelegramTestHandler(t)
	adminID := st.AdminUserID()

	// Bot start fails fast through a local endpoint so the save path reports
	// the failure without real network access.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":false,"description":"unauthorized"}`))
	}))
	defer server.Close()
	st.SetTelegramAPIEndpoint(server.URL)

	// GET returns mode/webhook_url and never the secret.
	resp := performWithUser(t, h.GetSettings, http.MethodGet, "", adminID)
	if resp.Code != http.StatusOK {
		t.Fatalf("GetSettings = %d: %s", resp.Code, resp.Body.String())
	}
	body := resp.Body.String()
	if !strings.Contains(body, `"mode":"polling"`) || !strings.Contains(body, `"webhook_url":""`) {
		t.Fatalf("GetSettings missing mode/webhook_url: %s", body)
	}
	if strings.Contains(body, "secret") {
		t.Fatalf("GetSettings leaked a secret: %s", body)
	}

	// Invalid mode is rejected.
	resp = performWithUser(t, h.SaveSettings, http.MethodPut, `{"enabled":true,"bot_token":"111:TOKENA","admin_tg_ids":"7330290970","mode":"bogus"}`, adminID)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("invalid mode = %d: %s", resp.Code, resp.Body.String())
	}

	// Webhook mode requires a valid public HTTPS base URL.
	resp = performWithUser(t, h.SaveSettings, http.MethodPut, `{"enabled":true,"bot_token":"111:TOKENA","admin_tg_ids":"7330290970","mode":"webhook","webhook_url":"http://insecure.example.com"}`, adminID)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("insecure webhook url = %d: %s", resp.Code, resp.Body.String())
	}

	// Valid webhook settings are saved; bot start failure is reported back.
	resp = performWithUser(t, h.SaveSettings, http.MethodPut, `{"enabled":true,"bot_token":"111:TOKENA","admin_tg_ids":"7330290970","mode":"webhook","webhook_url":"https://panel.example.com"}`, adminID)
	if resp.Code != http.StatusOK {
		t.Fatalf("valid webhook save = %d: %s", resp.Code, resp.Body.String())
	}
	if !strings.Contains(resp.Body.String(), `"running":false`) || !strings.Contains(resp.Body.String(), `"error"`) {
		t.Fatalf("start failure not reported: %s", resp.Body.String())
	}

	prefs := st.GetUserPrefs(adminID)
	if prefs.TGRemoteMode != "webhook" || prefs.TGRemoteWebhookURL != "https://panel.example.com" {
		t.Fatalf("saved prefs = %+v", prefs)
	}
	// The secret stays backend-only.
	if prefs.TGRemoteWebhookSecret != "" {
		t.Fatalf("save generated a webhook secret without a running bot: %+v", prefs)
	}
}
