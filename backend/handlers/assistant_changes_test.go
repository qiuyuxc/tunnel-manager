package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"tunnel-manager/models"
	"tunnel-manager/services"
)

func aiChangeTask(t *testing.T, handler *AssistantHandler, user models.SessionUser, request *http.Request, tool, arguments string) models.AITask {
	t.Helper()
	task, err := aiBuildTask(&user, aiDefaultTestCall(tool, arguments))
	if err != nil {
		t.Fatal(err)
	}
	if err := handler.prepareChange(request, &user, &task); err != nil {
		t.Fatal(err)
	}
	task.ConnectionID = handler.connectionMarker(user.ID)
	if err := handler.store.UpdateAI(user.ID, func(_ *models.AIState, state *models.AIUserState) error {
		state.Conversations = []models.AIConversation{{ID: "changes", Tasks: []models.AITask{task}}}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return task
}

func aiMockCloudflare(t *testing.T, respond func(*http.Request) any) *services.CloudflareClient {
	t.Helper()
	original := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Host != "api.cloudflare.com" {
			t.Fatalf("unexpected network destination: %s", request.URL.Host)
		}
		body, _ := json.Marshal(map[string]any{"success": true, "result": respond(request)})
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(string(body)))}, nil
	})
	t.Cleanup(func() { http.DefaultTransport = original })
	return services.NewCloudflareClient("mock-token", "account")
}

func TestAIChangePermissionsAndValidation(t *testing.T) {
	admin := &models.SessionUser{Role: models.RoleAdmin}
	member := &models.SessionUser{Role: models.RoleUser, Permissions: []string{models.PermDNS, "assistant_admin"}}
	for _, tool := range aiTools(member) {
		name := tool["function"].(map[string]any)["name"].(string)
		if aiPermission(name) != models.PermDNS {
			t.Fatalf("unprivileged tool exposed: %s", name)
		}
	}
	for _, args := range []string{`{}`, `{"registration_enabled":null}`, `{"registration_enabled":"true"}`, `{"password_login_disabled":true}`, `{"turnstile_secret":"secret"}`, `{"default_group_id":"admins"}`} {
		if _, err := aiBuildTask(admin, aiDefaultTestCall("update_app_settings", args)); err == nil {
			t.Fatalf("accepted forbidden settings: %s", args)
		}
	}
	if _, err := aiBuildTask(member, aiDefaultTestCall("update_app_settings", `{"registration_enabled":false}`)); err == nil {
		t.Fatal("group permission granted administrator settings")
	}
	for _, args := range []string{
		`{"zone_id":"../zone","record_name":"a.example.com","record_type":"A","changes":{"ttl":60}}`,
		`{"zone_id":"zone","record_name":"a.example.com","record_type":"A","changes":{}}`,
		`{"zone_id":"zone","record_name":"a.example.com","record_type":"A","changes":{"ttl":null}}`,
		`{"zone_id":"zone","record_name":"a.example.com","record_type":"A","changes":{"proxied":"false"}}`,
		`{"zone_id":"zone","record_name":"a.example.com","record_type":"A","changes":{"data":{"flags":null}}}`,
	} {
		if _, err := aiBuildTask(admin, aiDefaultTestCall("update_dns_record", args)); err == nil {
			t.Fatalf("accepted invalid DNS change: %s", args)
		}
	}
}

func TestAIChatPreparesChangesWithoutSendingResolvedValues(t *testing.T) {
	handler, _, admin, _ := newAITestHandler(t)
	cf := aiMockCloudflare(t, func(r *http.Request) any {
		if r.Method != http.MethodGet {
			t.Fatal("chat wrote Cloudflare")
		}
		return models.DNSRecord{ID: "record", Name: "a.example.com", Type: "A", Content: "192.0.2.77", TTL: 300, Proxied: true}
	})
	if response := aiTestRequest(handler.SaveSettings, admin, `{"scope":"personal","endpoint":"https://model.example.com/v1","model":"mock","api_key":"mock-key"}`, "", ""); response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	response := aiTestRequest(handler.NewConversation, admin, "", "", "")
	var conversation models.AIConversation
	_ = json.Unmarshal(response.Body.Bytes(), &conversation)
	calls := 0
	handler.client.Client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		body, _ := io.ReadAll(r.Body)
		if strings.Contains(string(body), "192.0.2.77") || strings.Contains(string(body), "禁止删除、修改既有资源") {
			t.Fatal("resolved DNS content leaked or old tool ban remains")
		}
		var reply any = map[string]any{"content": "已准备修改，需确认", "tool_calls": []any{map[string]any{"id": "call", "type": "function", "function": map[string]string{"name": "update_dns_record", "arguments": `{"zone_id":"zone","record_id":"record","record_name":"a.example.com","record_type":"A","changes":{"ttl":600}}`}}}}
		if calls > 1 {
			reply = map[string]any{"content": "任务仍待确认"}
		}
		raw, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"message": reply}}})
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(string(raw)))}, nil
	})
	chat := func(w http.ResponseWriter, r *http.Request) { handler.Chat(w, withCF(r, cf)) }
	for i := 0; i < 2; i++ {
		response = aiTestRequest(chat, admin, `{"message":"修改记录 TTL 到 600","include_resources":false}`, conversation.ID, "")
		if response.Code != 200 {
			t.Fatal(response.Body.String())
		}
		if json.Unmarshal(response.Body.Bytes(), &conversation) != nil || len(conversation.Tasks) != 1 || conversation.Tasks[0].Status != "pending" || !strings.Contains(string(conversation.Tasks[0].Before), "192.0.2.77") {
			t.Fatal("chat did not persist a reviewable plan")
		}
	}
}

func TestAITunnelDeleteAndStaleSettings(t *testing.T) {
	handler, state, admin, _ := newAITestHandler(t)
	writes := 0
	client := aiMockCloudflare(t, func(r *http.Request) any {
		if r.Method == http.MethodGet {
			return []models.Tunnel{{ID: "tunnel", Name: "Existing tunnel"}}
		}
		if r.Method != http.MethodDelete || r.URL.Path != "/client/v4/accounts/account/cfd_tunnel/tunnel" {
			t.Fatal("unexpected tunnel operation")
		}
		writes++
		return map[string]bool{"ok": true}
	})
	task := aiChangeTask(t, handler, admin, withCF(httptest.NewRequest("POST", "/", nil), client), "delete_tunnel", `{"tunnel_id":"tunnel"}`)
	handler.actions = map[string]http.HandlerFunc{"delete_tunnel": NewTunnelHandler(client, state).DeleteTunnel}
	response := aiTestRequest(func(w http.ResponseWriter, r *http.Request) { handler.TaskAction(w, withCF(r, client)) }, admin, `{"action":"execute","confirmed":true}`, "changes", task.ID)
	if writes != 1 || !strings.Contains(response.Body.String(), `"status":"succeeded"`) {
		t.Fatal(response.Body.String())
	}
	task = aiChangeTask(t, handler, admin, httptest.NewRequest("POST", "/", nil), "set_preferred_cname", `{"value":"new.example.com"}`)
	state.SetPreferredCNAME("changed-elsewhere.example.com")
	handler.actions["set_preferred_cname"] = func(http.ResponseWriter, *http.Request) { t.Fatal("stale setting executed") }
	response = aiTestRequest(handler.TaskAction, admin, `{"action":"execute","confirmed":true}`, "changes", task.ID)
	if !strings.Contains(response.Body.String(), "已变化") {
		t.Fatal(response.Body.String())
	}
}

func TestAIDNSChangesConfirmExactRecord(t *testing.T) {
	for _, operation := range []string{"update_dns_record", "delete_dns_record"} {
		t.Run(operation, func(t *testing.T) {
			handler, _, admin, _ := newAITestHandler(t)
			record := models.DNSRecord{ID: "record", Name: "a.example.com", Type: "A", Content: "192.0.2.1", TTL: 300, Proxied: true}
			writes := 0
			client := aiMockCloudflare(t, func(request *http.Request) any {
				if request.Method == http.MethodGet {
					if strings.HasSuffix(request.URL.Path, "/dns_records") {
						if request.URL.Query().Get("name") != record.Name || request.URL.Query().Get("type") != "A" {
							t.Fatal("DNS lookup was not exact")
						}
						return []models.DNSRecord{record}
					}
					return record
				}
				writes++
				if request.URL.Path != "/client/v4/zones/zone/dns_records/record" {
					t.Fatalf("wrong write path: %s", request.URL.Path)
				}
				if operation == "delete_dns_record" {
					if request.Method != http.MethodDelete {
						t.Fatal("not a delete")
					}
				} else {
					var payload models.DNSRecordRequest
					if request.Method != http.MethodPut || json.NewDecoder(request.Body).Decode(&payload) != nil || payload.Content != "192.0.2.2" || payload.TTL != 300 || !payload.Proxied {
						t.Fatalf("omitted DNS fields were reset: %+v", payload)
					}
				}
				return record
			})
			args := `{"zone_id":"zone","record_name":"a.example.com","record_type":"A"}`
			if operation == "update_dns_record" {
				args = strings.TrimSuffix(args, "}") + `,"changes":{"content":"192.0.2.2"}}`
			}
			task := aiChangeTask(t, handler, admin, withCF(httptest.NewRequest("POST", "/", nil), client), operation, args)
			if !strings.Contains(string(task.Before), "192.0.2.1") || writes != 0 {
				t.Fatal("preparation must show old value without writing")
			}
			dns := NewDNSHandler(client)
			handler.actions = map[string]http.HandlerFunc{"update_dns_record": dns.Update, "delete_dns_record": dns.Delete}
			execute := func(w http.ResponseWriter, r *http.Request) { handler.TaskAction(w, withCF(r, client)) }
			if response := aiTestRequest(execute, admin, `{"action":"execute"}`, "changes", task.ID); response.Code != 409 || writes != 0 {
				t.Fatal("unconfirmed change executed")
			}
			response := aiTestRequest(execute, admin, `{"action":"execute","confirmed":true}`, "changes", task.ID)
			var completed models.AITask
			if json.Unmarshal(response.Body.Bytes(), &completed) != nil || completed.Status != "succeeded" || writes != 1 {
				t.Fatalf("change failed: %s", response.Body.String())
			}
			if response := aiTestRequest(execute, admin, `{"action":"execute","confirmed":true}`, "changes", task.ID); response.Code != 409 || writes != 1 {
				t.Fatal("change repeated")
			}
		})
	}
}

func TestAIDNSChangeRefusesAmbiguityAndStaleValues(t *testing.T) {
	handler, _, admin, _ := newAITestHandler(t)
	record := models.DNSRecord{ID: "record", Name: "a.example.com", Type: "A", Content: "192.0.2.1", TTL: 1}
	duplicates := true
	client := aiMockCloudflare(t, func(r *http.Request) any {
		if r.Method != http.MethodGet {
			t.Fatal("unexpected write")
		}
		if strings.HasSuffix(r.URL.Path, "/dns_records") {
			if duplicates {
				return []models.DNSRecord{record, record}
			}
			return []models.DNSRecord{record}
		}
		return record
	})
	request := withCF(httptest.NewRequest("POST", "/", nil), client)
	args := `{"zone_id":"zone","record_name":"a.example.com","record_type":"A"}`
	task, _ := aiBuildTask(&admin, aiDefaultTestCall("delete_dns_record", args))
	if err := handler.prepareChange(request, &admin, &task); err == nil {
		t.Fatal("ambiguous DNS record selected")
	}
	duplicates = false
	task = aiChangeTask(t, handler, admin, request, "delete_dns_record", args)
	record.Content = "192.0.2.9"
	handler.actions = map[string]http.HandlerFunc{"delete_dns_record": func(http.ResponseWriter, *http.Request) { t.Fatal("stale snapshot executed") }}
	response := aiTestRequest(func(w http.ResponseWriter, r *http.Request) { handler.TaskAction(w, withCF(r, client)) }, admin, `{"action":"execute","confirmed":true}`, "changes", task.ID)
	if !strings.Contains(response.Body.String(), "已变化") {
		t.Fatalf("missing stale result: %s", response.Body.String())
	}
	wrong, _ := aiBuildTask(&admin, aiDefaultTestCall("delete_dns_record", `{"zone_id":"zone","record_id":"record","record_name":"wrong.example.com","record_type":"A"}`))
	if handler.prepareChange(request, &admin, &wrong) == nil {
		t.Fatal("record ID/name mismatch accepted")
	}
}

func TestAIDNSStructuredPartialChanges(t *testing.T) {
	for _, record := range []models.DNSRecord{
		{ID: "record", Name: "_sip._tcp.example.com", Type: "SRV", TTL: 600, Data: json.RawMessage(`{"priority":0,"weight":5,"port":5060,"target":"old.example.com"}`)},
		{ID: "record", Name: "example.com", Type: "CAA", TTL: 600, Data: json.RawMessage(`{"flags":0,"tag":"issue","value":"old.example.com"}`)},
		{ID: "record", Name: "example.com", Type: "MX", TTL: 600, Content: "mail.example.com", Priority: 0},
	} {
		t.Run(record.Type, func(t *testing.T) {
			handler, _, admin, _ := newAITestHandler(t)
			client := aiMockCloudflare(t, func(*http.Request) any { return record })
			changes := `{"ttl":300}`
			if record.Type == "SRV" {
				changes = `{"data":{"target":"new.example.com"}}`
			}
			if record.Type == "CAA" {
				changes = `{"data":{"value":""}}`
			}
			args, _ := json.Marshal(map[string]any{"zone_id": "zone", "record_id": "record", "record_name": record.Name, "record_type": record.Type, "changes": json.RawMessage(changes)})
			task := aiChangeTask(t, handler, admin, withCF(httptest.NewRequest("POST", "/", nil), client), "update_dns_record", string(args))
			var after models.DNSRecordRequest
			if json.Unmarshal(task.Arguments, &after) != nil {
				t.Fatal("bad normalized payload")
			}
			if record.Type == "SRV" && (after.Data == nil || *after.Data.Priority != 0 || *after.Data.Weight != 5 || *after.Data.Port != 5060 || after.Data.Target != "new.example.com" || after.TTL != 600) {
				t.Fatalf("SRV fields lost: %+v", after)
			}
			if record.Type == "CAA" && (after.Data == nil || *after.Data.Flags != 0 || after.Data.Tag != "issue" || *after.Data.Value != "") {
				t.Fatalf("CAA fields lost: %+v", after)
			}
			if record.Type == "MX" && (after.Priority == nil || *after.Priority != 0 || after.Content != record.Content || after.TTL != 300) {
				t.Fatalf("MX priority lost: %+v", after)
			}
		})
	}
}

func TestAIAdminChangesPreserveUnrelatedSettings(t *testing.T) {
	handler, state, admin, member := newAITestHandler(t)
	settings := state.GetAppSettings()
	settings.RegistrationEnabled = true
	settings.TurnstileSecret, settings.TurnstileSiteKey, settings.TurnstileEnabled = "encrypted-secret", "site-key", true
	settings.PasskeyRPID, settings.PasskeyOrigins = "example.com", "https://example.com"
	settings.RateLimitPerIP, settings.RateLimitNotify = 41, true
	if err := state.SetAppSettings(settings); err != nil {
		t.Fatal(err)
	}
	settings = state.GetAppSettings()
	task := aiChangeTask(t, handler, admin, httptest.NewRequest("POST", "/", nil), "update_app_settings", `{"registration_enabled":false}`)
	if strings.Contains(string(task.Before), "secret") || strings.Contains(string(task.Before), "passkey") {
		t.Fatal("unrelated authentication config in confirmation")
	}
	handler.actions = map[string]http.HandlerFunc{"update_app_settings": handler.UpdateAppSettings}
	if response := aiTestRequest(handler.TaskAction, member, `{"action":"execute","confirmed":true}`, "changes", task.ID); response.Code != 409 {
		t.Fatal("cross-user settings execution")
	}
	if response := aiTestRequest(handler.TaskAction, admin, `{"action":"execute"}`, "changes", task.ID); response.Code != 409 {
		t.Fatal("settings executed without confirmation")
	}
	// An unavailable Cloudflare client must not block local settings.
	response := aiTestRequest(func(w http.ResponseWriter, r *http.Request) {
		handler.TaskAction(w, withCF(r, services.NewCloudflareClient("", "")))
	}, admin, `{"action":"execute","confirmed":true}`, "changes", task.ID)
	if !strings.Contains(response.Body.String(), `"status":"succeeded"`) {
		t.Fatal(response.Body.String())
	}
	expected := settings
	expected.RegistrationEnabled = false
	if actual := state.GetAppSettings(); !reflect.DeepEqual(actual, expected) {
		t.Fatalf("unrelated settings changed:\ngot  %+v\nwant %+v", actual, expected)
	}
	if err := state.SetSiteSettings("Original", "Keep description", "/icon.png", true); err != nil {
		t.Fatal(err)
	}
	if err := state.SetPanelHost("panel.example.com"); err != nil {
		t.Fatal(err)
	}
	task = aiChangeTask(t, handler, admin, httptest.NewRequest("POST", "/", nil), "update_site_settings", `{"name":"Changed"}`)
	handler.actions["update_site_settings"] = NewConfigHandler(state).SetSiteSettings
	response = aiTestRequest(handler.TaskAction, admin, `{"action":"execute","confirmed":true}`, "changes", task.ID)
	cfg := state.GetConfig()
	if !strings.Contains(response.Body.String(), `"status":"succeeded"`) || cfg.SiteName != "Changed" || cfg.SiteDescription != "Keep description" || cfg.SiteIcon != "/icon.png" || !cfg.LandingEnabled || cfg.PanelHost != "panel.example.com" {
		t.Fatal("partial branding update reset unrelated fields")
	}
}

func TestAIMonitorDeleteOwnershipAndDispatch(t *testing.T) {
	for _, tool := range []string{"delete_monitor", "delete_monitor_target"} {
		t.Run(tool, func(t *testing.T) {
			handler, state, admin, member := newAITestHandler(t)
			if err := state.AddMonitor(models.Monitor{ID: "monitor", UserID: member.ID, Name: "My monitor", PublicToken: "private-token", Targets: []models.MonitorTarget{{ID: "target", Name: "Target", URL: "https://example.com"}}}); err != nil {
				t.Fatal(err)
			}
			args := `{"monitor_id":"monitor"}`
			if tool == "delete_monitor_target" {
				args = `{"monitor_id":"monitor","target_id":"target"}`
			}
			foreign, _ := aiBuildTask(&admin, aiDefaultTestCall(tool, args))
			if handler.prepareChange(httptest.NewRequest("POST", "/", nil), &admin, &foreign) == nil {
				t.Fatal("assistant deleted another user's monitor")
			}
			task := aiChangeTask(t, handler, member, httptest.NewRequest("POST", "/", nil), tool, args)
			if strings.Contains(string(task.Before), "private-token") {
				t.Fatal("monitor token leaked")
			}
			calls := 0
			handler.actions = map[string]http.HandlerFunc{tool: func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != http.MethodDelete || chi.URLParam(r, "monitorID") != "monitor" || (tool == "delete_monitor_target" && chi.URLParam(r, "targetID") != "target") {
					t.Fatal("wrong delete dispatch")
				}
				writeJSON(w, 200, map[string]bool{"ok": true})
			}}
			revoked := member
			revoked.Permissions = nil
			response := aiTestRequest(handler.TaskAction, revoked, `{"action":"execute","confirmed":true}`, "changes", task.ID)
			if calls != 0 || !strings.Contains(response.Body.String(), "已阻止执行") {
				t.Fatal("revoked permission executed")
			}
			task = aiChangeTask(t, handler, member, httptest.NewRequest("POST", "/", nil), tool, args)
			response = aiTestRequest(handler.TaskAction, member, `{"action":"execute","confirmed":true}`, "changes", task.ID)
			if calls != 1 || !strings.Contains(response.Body.String(), `"status":"succeeded"`) {
				t.Fatal(response.Body.String())
			}
		})
	}
}
