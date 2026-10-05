package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"tunnel-manager/models"
	"tunnel-manager/services"
)

func aiDefaultTestCall(tool, arguments string) services.AIToolCall {
	call := services.AIToolCall{Type: "function"}
	call.Function.Name, call.Function.Arguments = tool, arguments
	return call
}

func TestAIToolOptionalDefaultsSchema(test *testing.T) {
	expectedRequired := map[string][]string{
		"bind_domain":        {"tunnel_id", "zone_id", "hostname", "service_url"},
		"create_tunnel":      {"name"},
		"create_dns_record":  {"zone_id", "name", "type"},
		"create_monitor":     {"name"},
		"add_monitor_target": {"monitor_id", "name", "url"},
	}
	expectedDefaults := map[string]map[string]any{
		"create_monitor":    {"interval_sec": services.DefaultMonitorInterval},
		"create_dns_record": {"ttl": 1, "proxied": false},
	}
	tools := aiTools(&models.SessionUser{Role: models.RoleAdmin})
	if len(tools) != len(expectedRequired) {
		test.Fatalf("unexpected tool count: %d", len(tools))
	}
	for _, tool := range tools {
		function := tool["function"].(map[string]any)
		name := function["name"].(string)
		parameters := function["parameters"].(map[string]any)
		if !reflect.DeepEqual(parameters["required"], expectedRequired[name]) {
			test.Errorf("%s required = %v, want %v", name, parameters["required"], expectedRequired[name])
		}
		properties := parameters["properties"].(map[string]any)
		for field, expected := range expectedDefaults[name] {
			property := properties[field].(map[string]any)
			if property["default"] != expected {
				test.Errorf("%s.%s default = %v, want %v", name, field, property["default"], expected)
			}
		}
		if parameters["additionalProperties"] != false {
			test.Errorf("%s accepts unknown fields", name)
		}
	}
}

func TestAITaskOptionalDefaults(test *testing.T) {
	for _, testCase := range []struct {
		name, tool, arguments, expected string
	}{
		{"monitor omitted", "create_monitor", `{"name":" site "}`, `{"name":"site","interval_sec":60}`},
		{"monitor explicit", "create_monitor", `{"name":"site","interval_sec":300}`, `{"name":"site","interval_sec":300}`},
		{"monitor maximum", "create_monitor", `{"name":"site","interval_sec":3600}`, `{"name":"site","interval_sec":3600}`},
		{"DNS omitted", "create_dns_record", `{"zone_id":"zone","name":"site.example.com.","type":"a","content":" 1.1.1.1 "}`, `{"zone_id":"zone","name":"site.example.com","type":"A","content":"1.1.1.1","ttl":1,"proxied":false}`},
		{"DNS explicit", "create_dns_record", `{"zone_id":"zone","name":"site.example.com","type":"A","content":"1.1.1.1","ttl":300,"proxied":true}`, `{"zone_id":"zone","name":"site.example.com","type":"A","content":"1.1.1.1","ttl":300,"proxied":true}`},
		{"DNS explicit false", "create_dns_record", `{"zone_id":"zone","name":"site.example.com","type":"A","content":"1.1.1.1","proxied":false}`, `{"zone_id":"zone","name":"site.example.com","type":"A","content":"1.1.1.1","ttl":1,"proxied":false}`},
		{"DNS maximum", "create_dns_record", `{"zone_id":"zone","name":"site.example.com","type":"A","content":"1.1.1.1","ttl":86400}`, `{"zone_id":"zone","name":"site.example.com","type":"A","content":"1.1.1.1","ttl":86400,"proxied":false}`},
		{"DNS minimum custom TTL", "create_dns_record", `{"zone_id":"zone","name":"site.example.com","type":"A","content":"1.1.1.1","ttl":60}`, `{"zone_id":"zone","name":"site.example.com","type":"A","content":"1.1.1.1","ttl":60,"proxied":false}`},
		{"MX zero priority", "create_dns_record", `{"zone_id":"zone","name":"example.com","type":"MX","content":"mail.example.com","priority":0}`, `{"zone_id":"zone","name":"example.com","type":"MX","content":"mail.example.com","priority":0,"ttl":1,"proxied":false}`},
		{"TXT omitted", "create_dns_record", `{"zone_id":"zone","name":"example.com","type":"TXT","content":"verification"}`, `{"zone_id":"zone","name":"example.com","type":"TXT","content":"verification","ttl":1,"proxied":false}`},
		{"AAAA omitted", "create_dns_record", `{"zone_id":"zone","name":"example.com","type":"AAAA","content":"2001:db8::1"}`, `{"zone_id":"zone","name":"example.com","type":"AAAA","content":"2001:db8::1","ttl":1,"proxied":false}`},
		{"CNAME omitted", "create_dns_record", `{"zone_id":"zone","name":"www.example.com","type":"CNAME","content":"example.com"}`, `{"zone_id":"zone","name":"www.example.com","type":"CNAME","content":"example.com","ttl":1,"proxied":false}`},
	} {
		test.Run(testCase.name, func(test *testing.T) {
			task, err := aiBuildTask(&models.SessionUser{Role: models.RoleAdmin}, aiDefaultTestCall(testCase.tool, testCase.arguments))
			if err != nil {
				test.Fatal(err)
			}
			var actual, expected map[string]any
			if json.Unmarshal(task.Arguments, &actual) != nil || json.Unmarshal([]byte(testCase.expected), &expected) != nil {
				test.Fatal("invalid normalized JSON")
			}
			if !reflect.DeepEqual(actual, expected) || task.Status != "pending" || task.ID == "" {
				test.Fatalf("task = %+v, arguments = %s, want %s", task, task.Arguments, testCase.expected)
			}
		})
	}
}

func TestAITaskOptionalDefaultsRejectExplicitInvalid(test *testing.T) {
	for _, testCase := range []struct {
		tool, base, field string
		invalid           []string
	}{
		{"create_monitor", `{"name":"site"}`, "interval_sec", []string{"0", "null", `""`, `"60"`, "false", "59", "-1", "3601", "60.5", "[]", "{}"}},
		{"create_dns_record", `{"zone_id":"zone","name":"example.com","type":"A","content":"1.1.1.1"}`, "ttl", []string{"0", "null", `""`, `"60"`, "false", "2", "59", "-1", "86401", "60.5", "[]", "{}"}},
		{"create_dns_record", `{"zone_id":"zone","name":"example.com","type":"A","content":"1.1.1.1"}`, "proxied", []string{"0", "null", `""`, `"false"`, "1", "[]", "{}"}},
		{"create_dns_record", `{"zone_id":"zone","name":"example.com","type":"MX","content":"mail.example.com"}`, "priority", []string{"null", `""`, "-1", "65536"}},
	} {
		for _, invalid := range testCase.invalid {
			test.Run(testCase.tool+"/"+testCase.field+"/"+invalid, func(test *testing.T) {
				arguments := strings.TrimSuffix(testCase.base, "}") + `,"` + testCase.field + `":` + invalid + `}`
				if task, err := aiBuildTask(&models.SessionUser{Role: models.RoleAdmin}, aiDefaultTestCall(testCase.tool, arguments)); err == nil {
					test.Fatalf("accepted explicit invalid argument %s as %s", arguments, task.Arguments)
				}
			})
		}
	}
}

func TestAITaskDefaultsKeepRequiredInformation(test *testing.T) {
	for _, testCase := range []struct{ tool, arguments string }{
		{"create_monitor", `{}`},
		{"create_monitor", `{"name":null}`},
		{"create_monitor", `{"name":" "}`},
		{"create_monitor", `{"name":"site","publish_enabled":true}`},
		{"create_monitor", `{"name":"site","alert_enabled":true}`},
		{"create_monitor", `{"name":"site"} {}`},
		{"create_monitor", `null`},
		{"create_dns_record", `{"name":"example.com","type":"A","content":"1.1.1.1"}`},
		{"create_dns_record", `{"zone_id":"zone","type":"A","content":"1.1.1.1"}`},
		{"create_dns_record", `{"zone_id":"zone","name":"example.com","content":"1.1.1.1"}`},
		{"create_dns_record", `{"zone_id":"zone","name":"example.com","type":"A"}`},
		{"create_dns_record", `{"zone_id":"zone","name":"example.com","type":"MX","content":"mail.example.com"}`},
		{"create_dns_record", `{"zone_id":"../zone","name":"example.com","type":"A","content":"1.1.1.1"}`},
		{"create_tunnel", `{}`},
		{"bind_domain", `{"zone_id":"zone","hostname":"example.com","service_url":"https://example.com"}`},
		{"bind_domain", `{"tunnel_id":"tunnel","hostname":"example.com","service_url":"https://example.com"}`},
		{"bind_domain", `{"tunnel_id":"tunnel","zone_id":"zone","service_url":"https://example.com"}`},
		{"bind_domain", `{"tunnel_id":"tunnel","zone_id":"zone","hostname":"example.com"}`},
		{"add_monitor_target", `{"name":"site","url":"https://example.com"}`},
		{"add_monitor_target", `{"monitor_id":"monitor","url":"https://example.com"}`},
		{"add_monitor_target", `{"monitor_id":"monitor","name":"site"}`},
	} {
		if _, err := aiBuildTask(&models.SessionUser{Role: models.RoleAdmin}, aiDefaultTestCall(testCase.tool, testCase.arguments)); err == nil {
			test.Errorf("accepted %s %s", testCase.tool, testCase.arguments)
		}
	}
	for _, tool := range []string{"create_monitor", "create_dns_record"} {
		if _, err := aiBuildTask(&models.SessionUser{Role: models.RoleUser}, aiDefaultTestCall(tool, `{"name":"site"}`)); err == nil {
			test.Errorf("%s bypassed permissions", tool)
		}
	}
}

func TestAIChatOptionalDefaultsStayPendingUntilConfirmed(test *testing.T) {
	handler, state, admin, member := newAITestHandler(test)
	settings := `{"scope":"personal","endpoint":"https://model.example.com/v1","model":"test-model","api_key":"test-secret"}`
	if response := aiTestRequest(handler.SaveSettings, admin, settings, "", ""); response.Code != http.StatusOK {
		test.Fatal(response.Body.String())
	}
	response := aiTestRequest(handler.NewConversation, admin, "", "", "")
	var conversation models.AIConversation
	if json.Unmarshal(response.Body.Bytes(), &conversation) != nil {
		test.Fatal("invalid conversation")
	}
	handler.client.Client.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		var payload struct {
			Messages []models.AIMessage `json:"messages"`
		}
		if json.NewDecoder(request.Body).Decode(&payload) != nil || len(payload.Messages) == 0 || payload.Messages[0].Role != "system" {
			test.Fatal("missing system prompt")
		}
		for _, rule := range []string{"可选参数", "默认值", "不要追问", "interval_sec", "ttl", "proxied", "必要信息", "不得杜撰资源 ID", "待确认"} {
			if !strings.Contains(payload.Messages[0].Content, rule) {
				test.Errorf("system prompt lacks default/clarification rule %q", rule)
			}
		}
		body := `{"choices":[{"message":{"tool_calls":[{"id":"monitor","type":"function","function":{"name":"create_monitor","arguments":"{\"name\":\"site\"}"}},{"id":"dns","type":"function","function":{"name":"create_dns_record","arguments":"{\"zone_id\":\"zone\",\"name\":\"example.com\",\"type\":\"A\",\"content\":\"1.1.1.1\"}"}}]}}]}`
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	monitors := NewMonitorsHandler(state, nil, nil, nil)
	handler.actions = map[string]http.HandlerFunc{
		"create_monitor": monitors.Create,
		"create_dns_record": func(http.ResponseWriter, *http.Request) {
			test.Fatal("DNS task must not execute")
		},
	}
	response = aiTestRequest(handler.Chat, admin, `{"message":"创建监控和 DNS，其他用默认值","include_resources":false}`, conversation.ID, "")
	if response.Code != http.StatusOK {
		test.Fatalf("chat: %d %s", response.Code, response.Body.String())
	}
	if json.Unmarshal(response.Body.Bytes(), &conversation) != nil || len(conversation.Tasks) != 2 {
		test.Fatalf("missing tasks: %s", response.Body.String())
	}
	for _, task := range conversation.Tasks {
		if task.Status != "pending" {
			test.Fatalf("task not pending: %+v", task)
		}
	}
	if !strings.Contains(string(conversation.Tasks[0].Arguments), `"interval_sec":60`) || !strings.Contains(string(conversation.Tasks[1].Arguments), `"ttl":1`) || !strings.Contains(string(conversation.Tasks[1].Arguments), `"proxied":false`) {
		test.Fatalf("confirmation cards lack effective defaults: %s", response.Body.String())
	}
	_, _, saved := state.AIStateFor(admin.ID)
	if !reflect.DeepEqual(saved.Conversations[0].Tasks, conversation.Tasks) {
		test.Fatal("stored confirmation arguments differ from response")
	}
	monitorTask := conversation.Tasks[0]
	for _, attempt := range []struct {
		user      models.SessionUser
		confirmed bool
	}{{admin, false}, {member, true}} {
		body, _ := json.Marshal(map[string]any{"action": "execute", "confirmed": attempt.confirmed})
		if response := aiTestRequest(handler.TaskAction, attempt.user, string(body), conversation.ID, monitorTask.ID); response.Code != http.StatusConflict {
			test.Fatalf("unconfirmed/cross-user execution allowed: %d", response.Code)
		}
	}
	if len(state.GetConfig().Monitors) != 0 {
		test.Fatal("monitor created before owner confirmation")
	}
	response = aiTestRequest(handler.TaskAction, admin, `{"action":"execute","confirmed":true}`, conversation.ID, monitorTask.ID)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "succeeded") {
		test.Fatalf("confirmed creation failed: %d %s", response.Code, response.Body.String())
	}
	created := state.GetConfig().Monitors
	if len(created) != 1 || created[0].IntervalSec != services.DefaultMonitorInterval || created[0].PublishEnabled || created[0].AlertEnabled || created[0].UserID != admin.ID {
		test.Fatalf("unexpected created monitor: %+v", created)
	}
	response = aiTestRequest(monitors.Create, admin, `{"name":"API default"}`, "", "")
	if response.Code != http.StatusOK {
		test.Fatal(response.Body.String())
	}
	created = state.GetConfig().Monitors
	if len(created) != 2 || created[1].IntervalSec != created[0].IntervalSec || created[1].PublishEnabled || created[1].AlertEnabled {
		test.Fatalf("assistant defaults differ from product creation: %+v", created)
	}
}
