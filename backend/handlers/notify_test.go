package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"tunnel-manager/models"
	"tunnel-manager/store"
)

func newNotifyTestHandler(t *testing.T) (*NotifyHandler, *store.Store, string) {
	t.Helper()
	st := store.NewStore(filepath.Join(t.TempDir(), "config.json"))
	if err := st.SetAdminCredentials("admin", store.HashPassword("password")); err != nil {
		t.Fatal(err)
	}
	key := bytes.Repeat([]byte{9}, 32)
	return NewNotifyHandler(st, key, nil), st, st.AdminUserID()
}

func TestNotifySettingsRoundTrip(t *testing.T) {
	h, st, userID := newNotifyTestHandler(t)
	body := `{"channels":["email","telegram"],"events":{"login":true},"emails":"a@example.com\nb@example.com","tg_bot_token":"123456:TESTTOKEN","tg_notify_chat_id":"987654321"}`
	resp := performWithUser(t, h.SaveSettings, http.MethodPut, body, userID)
	if resp.Code != http.StatusOK {
		t.Fatalf("SaveSettings() = %d: %s", resp.Code, resp.Body.String())
	}
	if strings.Contains(resp.Body.String(), "TESTTOKEN") {
		t.Fatal("response leaked the Telegram token")
	}
	var view models.NotifySettingsView
	decodeResponse(t, resp, &view)
	if !view.TGBotTokenSet || len(view.Channels) != 2 || !view.Events[models.NotifyEventLogin] || view.TGNotifyChatID != "987654321" {
		t.Fatalf("view = %#v", view)
	}
	if !strings.Contains(view.Emails, "a@example.com") || !strings.Contains(view.Emails, "b@example.com") {
		t.Fatalf("emails = %q", view.Emails)
	}

	// The token is stored encrypted, not in plaintext.
	stored := st.GetUserPrefs(userID)
	if stored.TGBotTokenEncrypted == "" || stored.TGBotTokenEncrypted == "123456:TESTTOKEN" {
		t.Fatalf("token storage = %q", stored.TGBotTokenEncrypted)
	}

	// A blank token keeps the stored one; chat id updates.
	body2 := `{"channels":["email"],"events":{"login":false},"emails":"a@example.com","tg_bot_token":"","tg_notify_chat_id":"111"}`
	resp2 := performWithUser(t, h.SaveSettings, http.MethodPut, body2, userID)
	if resp2.Code != http.StatusOK {
		t.Fatalf("SaveSettings(2) = %d: %s", resp2.Code, resp2.Body.String())
	}
	if st.GetUserPrefs(userID).TGBotTokenEncrypted != stored.TGBotTokenEncrypted {
		t.Fatal("blank token did not preserve the stored token")
	}
}

func TestNotifySettingsValidation(t *testing.T) {
	h, _, userID := newNotifyTestHandler(t)
	cases := []struct {
		name string
		body string
	}{
		{"invalid channel", `{"channels":["sms"],"events":{},"emails":"","tg_notify_chat_id":""}`},
		{"invalid email", `{"channels":["email"],"events":{},"emails":"not-an-email","tg_notify_chat_id":""}`},
		{"invalid chat id", `{"channels":["telegram"],"events":{},"emails":"","tg_bot_token":"123:ABC","tg_notify_chat_id":"abc"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := performWithUser(t, h.SaveSettings, http.MethodPut, tc.body, userID)
			if resp.Code != http.StatusBadRequest {
				t.Fatalf("SaveSettings() = %d: %s", resp.Code, resp.Body.String())
			}
		})
	}
}

func TestNotifyGetSettingsDefaults(t *testing.T) {
	h, _, userID := newNotifyTestHandler(t)
	resp := performWithUser(t, h.GetSettings, http.MethodGet, "", userID)
	if resp.Code != http.StatusOK {
		t.Fatalf("GetSettings() = %d: %s", resp.Code, resp.Body.String())
	}
	var view models.NotifySettingsView
	decodeResponse(t, resp, &view)
	if len(view.Channels) != 0 || !view.Events[models.NotifyEventLogin] || view.TGBotTokenSet {
		t.Fatalf("default view = %#v", view)
	}
}

// performWithUser runs a handler with an authenticated session identity.
func performWithUser(t *testing.T, handler http.HandlerFunc, method, body, userID string) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body == "" {
		reader = bytes.NewReader(nil)
	} else {
		reader = bytes.NewReader([]byte(body))
	}
	req := httptest.NewRequest(method, "/", reader)
	req = withUser(req, models.SessionUser{ID: userID, Role: models.RoleAdmin})
	resp := httptest.NewRecorder()
	handler(resp, req)
	return resp
}
