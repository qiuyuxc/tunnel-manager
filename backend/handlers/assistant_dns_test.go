package handlers

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	"github.com/go-chi/chi/v5"
	"tunnel-manager/models"
)

func TestAIExtendedDNSSchema(test *testing.T) {
	for _, tool := range aiTools(&models.SessionUser{Role: models.RoleAdmin}) {
		function := tool["function"].(map[string]any)
		if function["name"] != "create_dns_record" {
			continue
		}
		properties := function["parameters"].(map[string]any)["properties"].(map[string]any)
		expected := []string{"A", "AAAA", "CNAME", "TXT", "MX", "NS", "SRV", "CAA", "PTR"}
		if !reflect.DeepEqual(properties["type"].(map[string]any)["enum"], expected) {
			test.Fatal("assistant DNS types differ from the editor")
		}
		data := properties["data"].(map[string]any)
		if data["additionalProperties"] != false || len(data["properties"].(map[string]any)) != 7 {
			test.Fatal("structured DNS schema is incomplete")
		}
		return
	}
	test.Fatal("DNS tool missing")
}

func TestAIExtendedDNSTasks(test *testing.T) {
	for _, testCase := range []struct {
		name, arguments string
		valid           bool
	}{
		{"NS", `{"zone_id":"zone","name":"sub.example.com","type":"NS","content":"ns.example.net"}`, true},
		{"PTR", `{"zone_id":"zone","name":"1.2.0.192.in-addr.arpa","type":"PTR","content":"host.example.com"}`, true},
		{"SRV", `{"zone_id":"zone","name":"_sip._tcp.example.com","type":"SRV","data":{"priority":0,"weight":5,"port":5060,"target":"sip.example.com"}}`, true},
		{"CAA", `{"zone_id":"zone","name":"example.com","type":"CAA","data":{"flags":0,"tag":"issue","value":"ca.example"}}`, true},
		{"CAA empty value", `{"zone_id":"zone","name":"example.com","type":"CAA","data":{"flags":0,"tag":"issue","value":""}}`, true},
		{"missing content", `{"zone_id":"zone","name":"example.com","type":"NS"}`, false},
		{"invalid NS", `{"zone_id":"zone","name":"example.com","type":"NS","content":"192.0.2.1"}`, false},
		{"missing SRV weight", `{"zone_id":"zone","name":"_sip._tcp.example.com","type":"SRV","data":{"priority":0,"port":5060,"target":"sip.example.com"}}`, false},
		{"null SRV priority", `{"zone_id":"zone","name":"_sip._tcp.example.com","type":"SRV","data":{"priority":null,"weight":5,"port":5060,"target":"sip.example.com"}}`, false},
		{"missing CAA flags", `{"zone_id":"zone","name":"example.com","type":"CAA","data":{"tag":"issue","value":"ca.example"}}`, false},
		{"null CAA value", `{"zone_id":"zone","name":"example.com","type":"CAA","data":{"flags":0,"tag":"issue","value":null}}`, false},
		{"unexpected data", `{"zone_id":"zone","name":"example.com","type":"TXT","content":"test","data":{"flags":0}}`, false},
	} {
		test.Run(testCase.name, func(test *testing.T) {
			call := aiDefaultTestCall("create_dns_record", testCase.arguments)
			task, err := aiBuildTask(&models.SessionUser{Role: models.RoleAdmin}, call)
			if (err == nil) != testCase.valid {
				test.Fatalf("valid=%v error=%v", testCase.valid, err)
			}
			if !testCase.valid {
				return
			}
			var actual, original map[string]any
			if json.Unmarshal(task.Arguments, &actual) != nil || json.Unmarshal([]byte(testCase.arguments), &original) != nil {
				test.Fatal("invalid task arguments")
			}
			if actual["ttl"] != float64(1) || actual["proxied"] != false || task.Status != "pending" || !reflect.DeepEqual(actual["data"], original["data"]) {
				test.Fatalf("defaults or structured fields lost: %s", task.Arguments)
			}
			if _, err := aiBuildTask(&models.SessionUser{Role: models.RoleUser}, call); err == nil {
				test.Fatal("extended DNS bypassed permissions")
			}
		})
	}
}

func TestAIStructuredDNSExecutionKeepsConfirmedData(test *testing.T) {
	for _, arguments := range []string{
		`{"zone_id":"zone","name":"_sip._tcp.example.com","type":"SRV","data":{"priority":0,"weight":5,"port":5060,"target":"sip.example.com"}}`,
		`{"zone_id":"zone","name":"example.com","type":"CAA","data":{"flags":0,"tag":"issue","value":"ca.example"}}`,
	} {
		handler, state, admin, member := newAITestHandler(test)
		task, err := aiBuildTask(&admin, aiDefaultTestCall("create_dns_record", arguments))
		if err != nil {
			test.Fatal(err)
		}
		task.ConnectionID = handler.connectionMarker(admin.ID)
		response := aiTestRequest(handler.NewConversation, admin, "", "", "")
		var conversation models.AIConversation
		if json.Unmarshal(response.Body.Bytes(), &conversation) != nil {
			test.Fatal("invalid conversation")
		}
		if err := state.UpdateAI(admin.ID, func(_ *models.AIState, state *models.AIUserState) error {
			saved, err := aiConversation(state, conversation.ID)
			if err == nil {
				saved.Tasks = append(saved.Tasks, task)
			}
			return err
		}); err != nil {
			test.Fatal(err)
		}
		calls := 0
		handler.actions = map[string]http.HandlerFunc{"create_dns_record": func(response http.ResponseWriter, request *http.Request) {
			calls++
			zone := chi.URLParam(request, "zoneID")
			record, valid := readDNSRecordRequest(response, request, zone)
			var expected models.DNSRecordRequest
			json.Unmarshal(task.Arguments, &expected)
			if !valid || zone != "zone" || !reflect.DeepEqual(record, expected) {
				test.Fatalf("confirmed DNS data changed: %+v want %+v", record, expected)
			}
			writeJSON(response, http.StatusCreated, map[string]string{"id": "mock-dns"})
		}}
		for _, attempt := range []struct {
			user models.SessionUser
			body string
		}{{admin, `{"action":"execute"}`}, {member, `{"action":"execute","confirmed":true}`}} {
			if response := aiTestRequest(handler.TaskAction, attempt.user, attempt.body, conversation.ID, task.ID); response.Code != http.StatusConflict || calls != 0 {
				test.Fatal("DNS reached business handler without owner confirmation")
			}
		}
		response = aiTestRequest(handler.TaskAction, admin, `{"action":"execute","confirmed":true}`, conversation.ID, task.ID)
		var completed models.AITask
		if json.Unmarshal(response.Body.Bytes(), &completed) != nil || completed.Status != "succeeded" || calls != 1 {
			test.Fatalf("confirmed execution failed: %s", response.Body.String())
		}
		response = aiTestRequest(handler.TaskAction, admin, `{"action":"execute","confirmed":true}`, conversation.ID, task.ID)
		if response.Code != http.StatusConflict || calls != 1 {
			test.Fatal("confirmed task was executed twice")
		}
	}
}
