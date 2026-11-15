package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"tunnel-manager/auth"
	"tunnel-manager/models"
)

func TestSlimAdminPersonalSettingsEditLegacyEffectiveConnection(t *testing.T) {
	handler, state, admin, _ := newAITestHandler(t)
	legacy := `{"scope":"shared","endpoint":"https://model.example.com/v1","model":"legacy-model","api_key":"legacy-secret","shared_enabled":true}`
	if response := aiTestRequest(handler.SaveSettings, admin, legacy, "", ""); response.Code != http.StatusOK {
		t.Fatal(response.Body.String())
	}
	response := aiTestRequest(handler.Settings, admin, "", "", "")
	var view struct {
		SharedEnabled bool `json:"shared_enabled"`
		Personal      struct {
			Model  string `json:"model"`
			KeySet bool   `json:"key_set"`
		} `json:"personal"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	if view.SharedEnabled || view.Personal.Model != "legacy-model" || !view.Personal.KeySet || strings.Contains(response.Body.String(), `"shared":`) {
		t.Fatalf("legacy effective connection not projected as personal: %s", response.Body.String())
	}
	update := `{"scope":"personal","endpoint":"https://model.example.com/v1","model":"updated-model","api_format":"responses","reasoning_summary":true}`
	if response := aiTestRequest(handler.SaveSettings, admin, update, "", ""); response.Code != http.StatusOK {
		t.Fatal(response.Body.String())
	}
	enabled, shared, personal := state.AIStateFor(admin.ID)
	secret, err := auth.DecryptSecret(handler.key, "ai-shared", shared.KeyEncrypted)
	if !enabled || shared.Model != "updated-model" || shared.APIFormat != "responses" || !shared.ReasoningSummary || err != nil || string(secret) != "legacy-secret" || personal.Connection.Endpoint != "" {
		t.Fatal("personal save failed to edit the active legacy connection")
	}
	var conversation models.AIConversation
	created := aiTestRequest(handler.NewConversation, admin, "", "", "")
	if err := json.Unmarshal(created.Body.Bytes(), &conversation); err != nil {
		t.Fatal(err)
	}
	calls := 0
	handler.client.Client.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		calls++
		if request.URL.Path != "/v1/responses" || request.Header.Get("Authorization") != "Bearer legacy-secret" {
			t.Fatal("chat did not use the configured legacy connection")
		}
		payload, err := io.ReadAll(request.Body)
		if err != nil || !strings.Contains(string(payload), `"model":"updated-model"`) {
			t.Fatal("chat ignored the updated model")
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"Ready"}]}]}`))}, nil
	})
	chat := aiTestRequest(handler.Chat, admin, `{"message":"hello","include_resources":false}`, conversation.ID, "")
	if chat.Code != http.StatusOK || calls != 1 {
		t.Fatalf("chat after upgrade: status=%d calls=%d body=%s", chat.Code, calls, chat.Body.String())
	}
}
