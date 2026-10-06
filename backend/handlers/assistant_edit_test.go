package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tunnel-manager/models"
)

func TestAIEditRequiresFinalRevisionAndExecutesEditedValues(t *testing.T) {
	handler, _, _, member := newAITestHandler(t)
	task := aiChangeTask(t, handler, member, httptest.NewRequest("POST", "/", nil), "create_monitor", `{"name":"AI proposal","interval_sec":60}`)
	writes := 0
	handler.actions = map[string]http.HandlerFunc{"create_monitor": func(w http.ResponseWriter, r *http.Request) {
		writes++
		var args struct {
			Name     string `json:"name"`
			Interval int    `json:"interval_sec"`
		}
		if json.NewDecoder(r.Body).Decode(&args) != nil || args.Name != "My final name" || args.Interval != 300 {
			t.Fatalf("executed original rather than final values: %+v", args)
		}
		writeJSON(w, 201, map[string]string{"id": "created"})
	}}
	for _, body := range []string{
		`{"action":"edit","arguments":{"name":"changed"}}`,
		`{"action":"edit","expected_revision":1,"arguments":{"name":"changed"}}`,
		`{"action":"edit","expected_revision":0,"arguments":{"name":"changed","interval_sec":0}}`,
		`{"action":"edit","expected_revision":0,"arguments":{"name":"changed","command":"rm"}}`,
	} {
		if response := aiTestRequest(handler.TaskAction, member, body, "changes", task.ID); response.Code < 400 {
			t.Fatal("invalid or stale edit saved")
		}
	}
	response := aiTestRequest(handler.TaskAction, member, `{"action":"edit","expected_revision":0,"arguments":{"name":"My final name","interval_sec":300}}`, "changes", task.ID)
	var edited models.AITask
	if json.Unmarshal(response.Body.Bytes(), &edited) != nil || edited.Status != "pending" || edited.ID != task.ID || edited.Revision != 1 || writes != 0 {
		t.Fatalf("edit executed or lost task identity: %s", response.Body.String())
	}
	for _, body := range []string{
		`{"action":"execute","confirmed":true,"expected_revision":0}`,
		`{"action":"execute","confirmed":true}`,
		`{"action":"execute","expected_revision":1}`,
		`{"action":"execute","confirmed":true,"expected_revision":1,"arguments":{"name":"unreviewed"}}`,
	} {
		if response := aiTestRequest(handler.TaskAction, member, body, "changes", task.ID); response.Code < 400 || writes != 0 {
			t.Fatal("unreviewed revision executed")
		}
	}
	response = aiTestRequest(handler.TaskAction, member, `{"action":"execute","confirmed":true,"expected_revision":1}`, "changes", task.ID)
	if writes != 1 || !strings.Contains(response.Body.String(), `"status":"succeeded"`) {
		t.Fatal(response.Body.String())
	}
	response = aiTestRequest(handler.TaskAction, member, `{"action":"edit","expected_revision":1,"arguments":{"name":"late change"}}`, "changes", task.ID)
	if response.Code != 409 || writes != 1 {
		t.Fatal("completed task was editable")
	}
}

func TestAIEditDNSKeepsTargetSnapshotAndValidatesManualValues(t *testing.T) {
	handler, _, admin, member := newAITestHandler(t)
	record := models.DNSRecord{ID: "record", Name: "a.example.com", Type: "A", Content: "192.0.2.1", TTL: 300, Proxied: true}
	client := aiMockCloudflare(t, func(r *http.Request) any {
		if r.Method != http.MethodGet {
			t.Fatal("editing wrote DNS")
		}
		return record
	})
	task := aiChangeTask(t, handler, admin, withCF(httptest.NewRequest("POST", "/", nil), client), "update_dns_record", `{"zone_id":"zone","record_id":"record","record_name":"a.example.com","record_type":"A","changes":{"content":"192.0.2.2"}}`)
	action := func(w http.ResponseWriter, r *http.Request) { handler.TaskAction(w, withCF(r, client)) }
	args := `{"zone_id":"zone","record_id":"record","name":"a.example.com","type":"A","content":"192.0.2.3","ttl":600,"proxied":false}`
	edit := `{"action":"edit","expected_revision":0,"arguments":` + args + `}`
	if response := aiTestRequest(action, member, edit, "changes", task.ID); response.Code < 400 {
		t.Fatal("another user edited task")
	}
	if response := aiTestRequest(action, admin, strings.Replace(edit, `"record_id":"record"`, `"record_id":"different"`, 1), "changes", task.ID); response.Code != 400 {
		t.Fatal("record target silently changed")
	}
	if response := aiTestRequest(action, admin, strings.Replace(edit, `"ttl":600`, `"ttl":0`, 1), "changes", task.ID); response.Code != 400 {
		t.Fatal("invalid manual TTL accepted")
	}
	response := aiTestRequest(action, admin, edit, "changes", task.ID)
	var edited models.AITask
	if json.Unmarshal(response.Body.Bytes(), &edited) != nil || edited.Revision != 1 || string(edited.Before) != string(task.Before) || !strings.Contains(string(edited.Arguments), "192.0.2.3") || !strings.Contains(string(edited.Arguments), `"proxied":false`) {
		t.Fatal(response.Body.String())
	}
	record.Content = "192.0.2.99"
	response = aiTestRequest(action, admin, strings.Replace(edit, `"expected_revision":0`, `"expected_revision":1`, 1), "changes", task.ID)
	if response.Code != 409 {
		t.Fatal("stale resource accepted during edit")
	}
}
