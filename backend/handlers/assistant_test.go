package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/go-chi/chi/v5"
	"tunnel-manager/auth"
	"tunnel-manager/models"
	"tunnel-manager/services"
	"tunnel-manager/store"
)

func aiTestRequest(handler http.HandlerFunc, user models.SessionUser, body, conversationID, taskID string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, "/api/assistant", strings.NewReader(body))
	route := chi.NewRouteContext()
	route.URLParams.Add("conversationID", conversationID)
	route.URLParams.Add("taskID", taskID)
	request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, route))
	response := httptest.NewRecorder()
	handler(response, withUser(request, user))
	return response
}

func newAITestHandler(t *testing.T) (*AssistantHandler, *store.Store, models.SessionUser, models.SessionUser) {
	t.Helper()
	state := newTestStore(t)
	if err := state.CreateUser(models.User{ID: "member", Username: "member", Role: models.RoleUser}); err != nil {
		t.Fatal(err)
	}
	admin := models.SessionUser{ID: state.AdminUserID(), Role: models.RoleAdmin}
	member := models.SessionUser{ID: "member", Role: models.RoleUser, Permissions: []string{models.PermTunnels, models.PermMonitors}}
	return NewAssistantHandler(state, bytes.Repeat([]byte{7}, 32), nil), state, admin, member
}

func TestAISettingsSelfHostedEndpoints(t *testing.T) {
	for _, scope := range []string{"personal", "shared"} {
		t.Run(scope, func(t *testing.T) {
			handler, state, admin, member := newAITestHandler(t)
			user := member
			if scope == "shared" {
				user = admin
			}
			for _, endpoint := range []string{"http://127.0.0.1:8787/v1/", "http://localhost:11434/v1", "https://192.168.1.10:8443/v1", "http://[::1]:8787/v1"} {
				payload, err := json.Marshal(map[string]string{"scope": scope, "endpoint": endpoint, "model": "local-model", "api_key": "local-secret"})
				if err != nil {
					t.Fatal(err)
				}
				response := aiTestRequest(handler.SaveSettings, user, string(payload), "", "")
				if response.Code != http.StatusOK || strings.Contains(response.Body.String(), "local-secret") {
					t.Fatalf("self-hosted settings: %d %s", response.Code, response.Body.String())
				}
				_, shared, personal := state.AIStateFor(user.ID)
				connection := personal.Connection
				if scope == "shared" {
					connection = shared
				}
				if connection.Endpoint != strings.TrimRight(endpoint, "/") || connection.KeyEncrypted == "" || strings.Contains(connection.KeyEncrypted, "local-secret") {
					t.Fatal("self-hosted connection was not normalized and encrypted")
				}
			}
		})
	}
}

func TestAISettingsIsolationSharedAndEncryption(t *testing.T) {
	handler, state, admin, member := newAITestHandler(t)
	personal := `{"scope":"personal","endpoint":"https://personal.example.com/v1","model":"personal-model","api_key":"personal-secret"}`
	response := aiTestRequest(handler.SaveSettings, member, personal, "", "")
	if response.Code != 200 || strings.Contains(response.Body.String(), "personal-secret") {
		t.Fatalf("save: %s", response.Body.String())
	}
	_, _, memberState := state.AIStateFor(member.ID)
	plaintext, err := auth.DecryptSecret(handler.key, "ai-personal:"+member.ID, memberState.Connection.KeyEncrypted)
	if err != nil || string(plaintext) != "personal-secret" {
		t.Fatal("key not encrypted with owner-bound purpose")
	}
	if _, err := auth.DecryptSecret(handler.key, "ai-personal:"+admin.ID, memberState.Connection.KeyEncrypted); err == nil {
		t.Fatal("key transferable across users")
	}
	response = aiTestRequest(handler.Settings, admin, "", "", "")
	if strings.Contains(response.Body.String(), "personal.example") {
		t.Fatal("personal endpoint leaked")
	}
	shared := `{"scope":"shared","endpoint":"https://shared.example.com/v1","model":"shared-model","api_key":"shared-secret","shared_enabled":true}`
	if response := aiTestRequest(handler.SaveSettings, member, shared, "", ""); response.Code != 403 {
		t.Fatal("member changed shared settings")
	}
	if response := aiTestRequest(handler.SaveSettings, admin, shared, "", ""); response.Code != 200 {
		t.Fatalf("shared save: %s", response.Body.String())
	}
	response = aiTestRequest(handler.Settings, member, "", "", "")
	if strings.Contains(response.Body.String(), "endpoint") || strings.Contains(response.Body.String(), "secret") || !strings.Contains(response.Body.String(), `"shared_enabled":true`) {
		t.Fatalf("shared view: %s", response.Body.String())
	}
	if response := aiTestRequest(handler.SaveSettings, member, personal, "", ""); response.Code == 200 {
		t.Fatal("member changed settings under shared mode")
	}
	blank := `{"scope":"shared","endpoint":"https://shared.example.com/v1","model":"shared-model","api_key":"","shared_enabled":false}`
	if response := aiTestRequest(handler.SaveSettings, admin, blank, "", ""); response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	_, storedShared, restored := state.AIStateFor(member.ID)
	if restored.Connection != memberState.Connection || storedShared.KeyEncrypted == "" {
		t.Fatal("mode switch lost stored configurations")
	}
	changedEndpoint := strings.Replace(strings.Replace(personal, "personal-secret", "", 1), "personal.example", "attacker.example", 1)
	if response := aiTestRequest(handler.SaveSettings, member, changedEndpoint, "", ""); response.Code != 400 {
		t.Fatal("old secret forwarded to a changed endpoint")
	}
	if response := aiTestRequest(handler.Settings, models.SessionUser{Role: models.RoleAdmin}, "", "", ""); response.Code != 401 {
		t.Fatal("static identity should not own conversations")
	}
}

func TestAISharedNeverFallsBackAndChatOnlyProposes(t *testing.T) {
	handler, state, admin, member := newAITestHandler(t)
	shared := `{"scope":"shared","endpoint":"https://shared.example.com/v1","model":"shared-model","api_key":"shared-secret","shared_enabled":true}`
	aiTestRequest(handler.SaveSettings, admin, shared, "", "")
	response := aiTestRequest(handler.NewConversation, member, "", "", "")
	var conversation models.AIConversation
	if json.Unmarshal(response.Body.Bytes(), &conversation) != nil {
		t.Fatal("invalid conversation")
	}
	var calls atomic.Int32
	handler.client.Client.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		calls.Add(1)
		if request.Header.Get("Authorization") != "Bearer shared-secret" {
			t.Fatal("wrong effective key")
		}
		data, _ := io.ReadAll(request.Body)
		if strings.Contains(string(data), "shared-secret") {
			t.Fatal("key in model prompt")
		}
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"choices":[{"message":{"content":"请确认","tool_calls":[{"id":"call","type":"function","function":{"name":"create_tunnel","arguments":"{\"name\":\"test tunnel\"}"}}]}}]}`))}, nil
	})
	handler.actions = map[string]http.HandlerFunc{"create_tunnel": func(http.ResponseWriter, *http.Request) { t.Fatal("chat executed a write without confirmation") }}
	response = aiTestRequest(handler.Chat, member, `{"message":"创建测试隧道","include_resources":false}`, conversation.ID, "")
	if response.Code != 200 {
		t.Fatalf("chat: %s", response.Body.String())
	}
	_ = json.Unmarshal(response.Body.Bytes(), &conversation)
	if len(conversation.Tasks) != 1 || conversation.Tasks[0].Status != "pending" || calls.Load() != 1 {
		t.Fatal("missing pending task")
	}
	if response := aiTestRequest(handler.Chat, admin, `{"message":"read"}`, conversation.ID, ""); response.Code != 404 {
		t.Fatal("cross-user chat access")
	}
	if response := aiTestRequest(handler.Conversations, admin, "", "", ""); strings.Contains(response.Body.String(), conversation.ID) {
		t.Fatal("conversation leaked to admin")
	}
	if err := state.UpdateAI(admin.ID, func(global *models.AIState, _ *models.AIUserState) error {
		global.Shared.KeyEncrypted = "corrupt"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	response = aiTestRequest(handler.Chat, member, `{"message":"retry"}`, conversation.ID, "")
	if response.Code != 503 || calls.Load() != 1 {
		t.Fatal("shared failure did not fail closed")
	}
}

func TestAITasksConfirmationOwnershipAndIdempotency(t *testing.T) {
	handler, state, admin, member := newAITestHandler(t)
	var executions atomic.Int32
	started, finish := make(chan struct{}), make(chan struct{})
	handler.actions = map[string]http.HandlerFunc{"create_tunnel": func(response http.ResponseWriter, request *http.Request) {
		executions.Add(1)
		close(started)
		<-finish
		writeJSON(response, 201, map[string]string{"id": "new-resource", "token": "connector-secret", "run_command": "secret command"})
	}}
	err := state.UpdateAI(member.ID, func(_ *models.AIState, user *models.AIUserState) error {
		user.Conversations = []models.AIConversation{{ID: "chat", Tasks: []models.AITask{{ID: "task", Tool: "create_tunnel", Title: "test", Arguments: json.RawMessage(`{"name":"approved-name"}`), Status: "pending"}}}}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, user := range []models.SessionUser{admin, member} {
		body := `{"action":"execute","confirmed":false}`
		if user.ID == admin.ID {
			body = `{"action":"execute","confirmed":true}`
		}
		if response := aiTestRequest(handler.TaskAction, user, body, "chat", "task"); response.Code != 409 {
			t.Fatal("unconfirmed/cross-user execution allowed")
		}
	}
	if response := aiTestRequest(handler.TaskAction, member, `{"action":"execute","confirmed":true,"arguments":{"name":"tampered"}}`, "chat", "task"); response.Code != 400 {
		t.Fatal("client rewrote approved arguments")
	}
	for _, action := range []string{"pause", "resume"} {
		if response := aiTestRequest(handler.TaskAction, member, `{"action":"`+action+`"}`, "chat", "task"); response.Code != 200 {
			t.Fatal("pause/resume failed")
		}
	}
	completed := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		completed <- aiTestRequest(handler.TaskAction, member, `{"action":"execute","confirmed":true}`, "chat", "task")
	}()
	<-started
	if response := aiTestRequest(handler.TaskAction, member, `{"action":"execute","confirmed":true}`, "chat", "task"); response.Code != 409 {
		t.Fatal("concurrent duplicate execution allowed")
	}
	if response := aiTestRequest(handler.DeleteConversation, member, "", "chat", ""); response.Code != 409 {
		t.Fatal("running conversation deleted")
	}
	close(finish)
	response := <-completed
	if response.Code != 200 || !strings.Contains(response.Body.String(), "succeeded") || strings.Contains(response.Body.String(), "connector-secret") {
		t.Fatalf("execution result: %s", response.Body.String())
	}
	if response := aiTestRequest(handler.TaskAction, member, `{"action":"execute","confirmed":true}`, "chat", "task"); response.Code != 409 || executions.Load() != 1 {
		t.Fatal("completed task executed again")
	}
}

func TestAIToolValidation(t *testing.T) {
	user := &models.SessionUser{Role: models.RoleAdmin}
	for _, testCase := range []struct {
		tool, arguments string
		valid           bool
	}{
		{"create_tunnel", `{"name":"site"}`, true},
		{"bind_domain", `{"tunnel_id":"tunnel","zone_id":"zone","hostname":"nas.example.com","service_url":"http://192.168.1.8:5000"}`, true},
		{"bind_domain", `{"tunnel_id":"../tunnel","zone_id":"zone","hostname":"nas.example.com","service_url":"http://192.168.1.8:5000"}`, false},
		{"bind_domain", `{"tunnel_id":"tunnel","zone_id":"zone","hostname":"*.example.com","service_url":"https://user:pass@example.com"}`, false},
		{"create_tunnel", `{"name":"site","command":"rm"}`, false},
		{"create_tunnel", `null`, false},
		{"delete_tunnel", `{"name":"site"}`, false},
		{"create_dns_record", `{"zone_id":"zone","name":"site.example.com","type":"A","content":"1.1.1.1","ttl":1,"proxied":true}`, true},
		{"create_dns_record", `{"zone_id":"../../admin","name":"site","type":"A","content":"1.1.1.1"}`, false},
		{"create_monitor", `{"name":"site","interval_sec":60}`, true},
		{"create_monitor", `{"name":"site","interval_sec":1}`, false},
		{"add_monitor_target", `{"monitor_id":"id","name":"site","url":"https://user:pass@example.com"}`, false},
		{"add_monitor_target", `{"monitor_id":"id","name":"site","url":"https://example.com"}`, true},
	} {
		call := services.AIToolCall{Type: "function"}
		call.Function.Name, call.Function.Arguments = testCase.tool, testCase.arguments
		_, err := aiBuildTask(user, call)
		if (err == nil) != testCase.valid {
			t.Errorf("%s %s: %v", testCase.tool, testCase.arguments, err)
		}
	}
	if len(aiTools(&models.SessionUser{Role: models.RoleUser})) != 0 {
		t.Fatal("tools ignore permissions")
	}
}

func TestAITaskFailureNeedsVerifiedRetry(t *testing.T) {
	handler, state, _, member := newAITestHandler(t)
	if err := state.UpdateAI(member.ID, func(_ *models.AIState, user *models.AIUserState) error {
		user.Conversations = []models.AIConversation{{ID: "chat", Tasks: []models.AITask{{ID: "task", Tool: "create_tunnel", Arguments: json.RawMessage(`{"name":"test"}`), Status: "pending"}}}}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	calls := 0
	handler.actions = map[string]http.HandlerFunc{"create_tunnel": func(response http.ResponseWriter, _ *http.Request) {
		calls++
		if calls == 1 {
			panic("upstream-secret-not-for-client")
		}
		writeJSON(response, 201, map[string]string{"id": "created"})
	}}
	response := aiTestRequest(handler.TaskAction, member, `{"action":"execute","confirmed":true}`, "chat", "task")
	if response.Code != 200 || !strings.Contains(response.Body.String(), `"status":"unknown"`) || strings.Contains(response.Body.String(), "upstream-secret") {
		t.Fatalf("panic recovery: %s", response.Body.String())
	}
	for _, body := range []string{`{"action":"execute","confirmed":true}`, `{"action":"retry","confirmed":true}`, `{"action":"retry","checked_resources":true}`} {
		if response := aiTestRequest(handler.TaskAction, member, body, "chat", "task"); response.Code != 409 {
			t.Fatal("unverified retry allowed")
		}
	}
	response = aiTestRequest(handler.TaskAction, member, `{"action":"retry","confirmed":true,"checked_resources":true}`, "chat", "task")
	if response.Code != 200 || calls != 2 || !strings.Contains(response.Body.String(), `"status":"succeeded"`) {
		t.Fatalf("verified retry: %s", response.Body.String())
	}
}

func TestAITaskExecutionRechecksPermission(t *testing.T) {
	handler, state, _, member := newAITestHandler(t)
	if err := state.UpdateAI(member.ID, func(_ *models.AIState, user *models.AIUserState) error {
		user.Conversations = []models.AIConversation{{ID: "chat", Tasks: []models.AITask{{ID: "task", Tool: "create_tunnel", Arguments: json.RawMessage(`{"name":"test"}`), Status: "pending"}}}}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	handler.actions = map[string]http.HandlerFunc{"create_tunnel": func(http.ResponseWriter, *http.Request) { t.Fatal("revoked permission reached business handler") }}
	member.Permissions = nil
	response := aiTestRequest(handler.TaskAction, member, `{"action":"execute","confirmed":true}`, "chat", "task")
	if response.Code != 200 || !strings.Contains(response.Body.String(), "已阻止执行") {
		t.Fatalf("permission not rechecked: %s", response.Body.String())
	}
}
