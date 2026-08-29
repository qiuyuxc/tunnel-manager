package services

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"tunnel-manager/auth"
	"tunnel-manager/models"
	"tunnel-manager/store"
)

func newUserTelegramTestManager(t *testing.T) (*UserTelegramManager, *store.Store, []byte) {
	t.Helper()
	st := store.NewStore(filepath.Join(t.TempDir(), "config.json"))
	key := bytes.Repeat([]byte{7}, 32)
	cf := NewCloudflareClient("token", "acct")
	ds := NewDomainService(cf, st)
	return NewUserTelegramManager(st, cf, ds, key), st, key
}

// telegramAPIMock answers the Telegram Bot API endpoints used by Start and
// dispatch; onRequest observes outgoing request paths.
func telegramAPIMock(t *testing.T, onRequest func(path string)) *httptest.Server {
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

func TestMigrateLegacyAdminBotFirstBoot(t *testing.T) {
	m, st, key := newUserTelegramTestManager(t)
	st.SetTelegramSettings(true, "123:LEGACYTOKEN", "7330290970", "webhook", "https://panel.example.com", "")
	st.SetTelegramWebhookSecret("legacy-secret")

	m.MigrateLegacyAdminBot()

	prefs := st.GetUserPrefs(st.AdminUserID())
	if !prefs.TGRemoteEnabled {
		t.Fatal("enabled flag not migrated")
	}
	if prefs.TGOperatorIDs != "7330290970" {
		t.Fatalf("operator ids = %q", prefs.TGOperatorIDs)
	}
	if prefs.TGRemoteMode != "webhook" {
		t.Fatalf("mode = %q, want webhook", prefs.TGRemoteMode)
	}
	if prefs.TGRemoteWebhookURL != "https://panel.example.com" {
		t.Fatalf("webhook url = %q", prefs.TGRemoteWebhookURL)
	}
	if prefs.TGRemoteWebhookSecret != "legacy-secret" {
		t.Fatalf("webhook secret = %q", prefs.TGRemoteWebhookSecret)
	}
	plain, err := auth.DecryptSecret(key, notifyTGTokenPurpose, prefs.TGRemoteTokenEncrypted)
	if err != nil || string(plain) != "123:LEGACYTOKEN" {
		t.Fatalf("decrypt migrated token = %q, %v", plain, err)
	}
}

func TestMigrateLegacyAdminBotPreservesWebhookForMigratedToken(t *testing.T) {
	m, st, key := newUserTelegramTestManager(t)
	adminID := st.AdminUserID()
	enc, err := auth.EncryptSecret(key, notifyTGTokenPurpose, []byte("123:EXISTINGTOKEN"))
	if err != nil {
		t.Fatal(err)
	}
	// v2.2.0-test.2 state: token already migrated, mode defaults to polling.
	if err := st.SetUserRemoteSettings(adminID, true, "7330290970", enc, "polling", "", ""); err != nil {
		t.Fatal(err)
	}
	// Legacy global webhook configuration is still present.
	st.SetTelegramSettings(true, "123:EXISTINGTOKEN", "7330290970", "webhook", "https://panel.example.com", "")
	st.SetTelegramWebhookSecret("legacy-secret")

	m.MigrateLegacyAdminBot()

	prefs := st.GetUserPrefs(adminID)
	if prefs.TGRemoteMode != "webhook" {
		t.Fatalf("mode = %q, want webhook (upgrade must not silently downgrade)", prefs.TGRemoteMode)
	}
	if prefs.TGRemoteWebhookURL != "https://panel.example.com" || prefs.TGRemoteWebhookSecret != "legacy-secret" {
		t.Fatalf("webhook prefs = %+v", prefs)
	}
	if prefs.TGRemoteTokenEncrypted != enc {
		t.Fatal("existing migrated token was replaced")
	}
	if prefs.TGOperatorIDs != "7330290970" || !prefs.TGRemoteEnabled {
		t.Fatalf("existing per-user settings clobbered: %+v", prefs)
	}
}

func TestMigrateLegacyAdminBotDoesNotOverrideExplicitSettings(t *testing.T) {
	m, st, key := newUserTelegramTestManager(t)
	adminID := st.AdminUserID()
	enc, _ := auth.EncryptSecret(key, notifyTGTokenPurpose, []byte("123:EXISTINGTOKEN"))
	if err := st.SetUserRemoteSettings(adminID, true, "7330290970", enc, "polling", "https://own.example.com", ""); err != nil {
		t.Fatal(err)
	}
	st.SetTelegramSettings(true, "123:EXISTINGTOKEN", "7330290970", "webhook", "https://legacy.example.com", "")
	st.SetTelegramWebhookSecret("legacy-secret")

	m.MigrateLegacyAdminBot()

	prefs := st.GetUserPrefs(adminID)
	if prefs.TGRemoteMode != "polling" || prefs.TGRemoteWebhookURL != "https://own.example.com" {
		t.Fatalf("explicit per-user settings were overridden: %+v", prefs)
	}
}

func TestReconcileRestartsBotOnSettingsChange(t *testing.T) {
	var mu sync.Mutex
	webhookCalls, deleteCalls := 0, 0
	registered := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		switch {
		case strings.HasSuffix(r.URL.Path, "/setWebhook"):
			webhookCalls++
			var payload struct {
				URL string `json:"url"`
			}
			_ = json.NewDecoder(r.Body).Decode(&payload)
			registered = append(registered, payload.URL)
		case strings.HasSuffix(r.URL.Path, "/deleteWebhook"):
			deleteCalls++
		}
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/getMe"):
			_, _ = w.Write([]byte(`{"ok":true,"result":{"username":"test_bot"}}`))
		default:
			_, _ = w.Write([]byte(`{"ok":true,"result":[]}`))
		}
	}))
	defer server.Close()

	m, st, key := newUserTelegramTestManager(t)
	defer m.StopAll()
	adminID := st.AdminUserID()
	enc, err := auth.EncryptSecret(key, notifyTGTokenPurpose, []byte("123:TESTTOKEN"))
	if err != nil {
		t.Fatal(err)
	}
	st.SetTelegramAPIEndpoint(server.URL)
	if err := st.SetUserRemoteSettings(adminID, true, "7330290970", enc, "webhook", "https://panel.example.com", "secret-1"); err != nil {
		t.Fatal(err)
	}

	m.Reconcile()
	mu.Lock()
	first := webhookCalls
	mu.Unlock()
	if first == 0 {
		t.Fatal("webhook not registered on first reconcile")
	}
	mu.Lock()
	firstURL := ""
	if len(registered) > 0 {
		firstURL = registered[0]
	}
	mu.Unlock()
	wantURL := "https://panel.example.com/api/telegram/webhook/" + adminID
	if firstURL != wantURL {
		t.Fatalf("registered webhook URL = %q, want %q", firstURL, wantURL)
	}
	if got := m.Status(adminID); !got.Running || got.Mode != "webhook" {
		t.Fatalf("initial status = %+v", got)
	}

	// Secret change restarts the bot and re-registers the webhook.
	if err := st.SetUserRemoteSettings(adminID, true, "7330290970", enc, "webhook", "https://panel.example.com", "secret-2"); err != nil {
		t.Fatal(err)
	}
	m.Reconcile()
	mu.Lock()
	second := webhookCalls
	mu.Unlock()
	if second <= first {
		t.Fatalf("webhook not re-registered after secret change: %d -> %d", first, second)
	}
	if got := m.Status(adminID); !got.Running || got.Mode != "webhook" {
		t.Fatalf("status after secret change = %+v", got)
	}

	// URL change restarts the bot too.
	if err := st.SetUserRemoteSettings(adminID, true, "7330290970", enc, "webhook", "https://panel2.example.com", "secret-2"); err != nil {
		t.Fatal(err)
	}
	m.Reconcile()
	mu.Lock()
	third := webhookCalls
	mu.Unlock()
	if third <= second {
		t.Fatalf("webhook not re-registered after URL change: %d -> %d", second, third)
	}

	// Switching to polling restarts the bot and deletes the webhook.
	if err := st.SetUserRemoteSettings(adminID, true, "7330290970", enc, "polling", "", ""); err != nil {
		t.Fatal(err)
	}
	m.Reconcile()
	if got := m.Status(adminID); !got.Running || got.Mode != "polling" {
		t.Fatalf("polling status = %+v", got)
	}
	mu.Lock()
	dc := deleteCalls
	mu.Unlock()
	if dc == 0 {
		t.Fatal("webhook not deleted when switching to polling")
	}
}

func TestHandleWebhookDispatchesOnlyToOwner(t *testing.T) {
	var mu sync.Mutex
	sent := []string{}
	server := telegramAPIMock(t, func(path string) {
		if strings.Contains(path, "/sendMessage") {
			mu.Lock()
			sent = append(sent, path)
			mu.Unlock()
		}
	})
	defer server.Close()

	m, st, key := newUserTelegramTestManager(t)
	defer m.StopAll()
	st.SetTelegramAPIEndpoint(server.URL)

	adminID := st.AdminUserID()
	enc1, _ := auth.EncryptSecret(key, notifyTGTokenPurpose, []byte("111:TOKENA"))
	if err := st.SetUserRemoteSettings(adminID, true, "7330290970", enc1, "webhook", "https://panel.example.com", "secret-a"); err != nil {
		t.Fatal(err)
	}
	user2 := models.User{Username: "member", PasswordHash: store.HashPassword("password"), Role: models.RoleUser, Status: models.UserActive, EmailVerified: true}
	if err := st.CreateUser(user2); err != nil {
		t.Fatal(err)
	}
	member, ok := st.GetUserByUsername("member")
	if !ok {
		t.Fatal("member user not found")
	}
	u2ID := member.ID
	enc2, _ := auth.EncryptSecret(key, notifyTGTokenPurpose, []byte("222:TOKENB"))
	if err := st.SetUserRemoteSettings(u2ID, true, "7330290970", enc2, "webhook", "https://panel.example.com", "secret-b"); err != nil {
		t.Fatal(err)
	}

	m.Reconcile()
	if !m.Status(adminID).Running || !m.Status(u2ID).Running {
		t.Fatalf("bots not running: %+v / %+v", m.Status(adminID), m.Status(u2ID))
	}

	body, _ := json.Marshal(map[string]interface{}{
		"update_id": 1,
		"message": map[string]interface{}{
			"message_id": 1,
			"from":       map[string]interface{}{"id": 7330290970, "is_bot": false, "first_name": "T"},
			"chat":       map[string]interface{}{"id": 7330290970, "type": "private"},
			"date":       1700000000,
			"text":       "/help",
		},
	})

	// Wrong secret is rejected and dispatches nothing.
	if err := m.HandleWebhook(adminID, "wrong-secret", body); err == nil {
		t.Fatal("wrong secret accepted")
	}
	// Unknown bot is rejected.
	if err := m.HandleWebhook("no-such-user", "secret-a", body); err == nil {
		t.Fatal("unknown bot accepted")
	}
	mu.Lock()
	if len(sent) != 0 {
		t.Fatalf("rejected updates dispatched messages: %v", sent)
	}
	mu.Unlock()

	if err := m.HandleWebhook(adminID, "secret-a", body); err != nil {
		t.Fatalf("HandleWebhook(owner) = %v", err)
	}
	if err := m.HandleWebhook(u2ID, "secret-b", body); err != nil {
		t.Fatalf("HandleWebhook(other) = %v", err)
	}

	// Updates are handled in a goroutine; wait until both bots answered.
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
		t.Fatalf("dispatch not isolated per owner, sent = %v", sent)
	}

	// A valid webhook update refreshes last_update_at.
	statusA := m.Status(adminID)
	if statusA.LastUpdateAt == "" {
		t.Fatalf("last_update_at not refreshed for owner bot: %+v", statusA)
	}
}
