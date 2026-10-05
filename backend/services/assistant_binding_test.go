package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tunnel-manager/models"
)

func TestAIBindingPreservesConfigurationAndRefusesConflicts(t *testing.T) {
	for _, scenario := range []string{"success", "existing-dns", "existing-route", "changed-config"} {
		t.Run(scenario, func(t *testing.T) {
			writes, reads := 0, 0
			server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				response.Header().Set("Content-Type", "application/json")
				switch {
				case request.URL.Path == "/zones/zone":
					fmt.Fprint(response, `{"success":true,"result":{"id":"zone","name":"example.com"}}`)
				case request.Method == http.MethodGet && strings.HasSuffix(request.URL.Path, "/dns_records"):
					if request.URL.Query().Get("name") != "nas.example.com" {
						t.Error("missing exact DNS name filter")
					}
					if scenario == "existing-dns" {
						fmt.Fprint(response, `{"success":true,"result":[{"id":"existing"}]}`)
					} else {
						fmt.Fprint(response, `{"success":true,"result":[]}`)
					}
				case request.Method == http.MethodGet && strings.HasSuffix(request.URL.Path, "/configurations"):
					reads++
					hostname := "keep.example.com"
					if scenario == "existing-route" {
						hostname = "nas.example.com"
					}
					if scenario == "changed-config" && reads > 1 {
						hostname = "changed.example.com"
					}
					fmt.Fprintf(response, `{"success":true,"result":{"config":{"warp-routing":{"enabled":true},"originRequest":{"connectTimeout":12},"ingress":[{"hostname":%q,"service":"http://keep:80","customField":"preserve"},{"service":"http_status:404"}]}}}`, hostname)
				case request.Method == http.MethodPost:
					writes++
					fmt.Fprint(response, `{"success":true,"result":{"id":"dns-created"}}`)
				case request.Method == http.MethodPut:
					writes++
					var payload struct {
						Config map[string]json.RawMessage `json:"config"`
					}
					if json.NewDecoder(request.Body).Decode(&payload) != nil {
						t.Error("bad payload")
					}
					if !strings.Contains(string(payload.Config["warp-routing"]), "true") || !strings.Contains(string(payload.Config["ingress"]), `"customField":"preserve"`) || !strings.Contains(string(payload.Config["originRequest"]), "12") {
						t.Error("unrelated configuration lost")
					}
					var rules []models.IngressRule
					_ = json.Unmarshal(payload.Config["ingress"], &rules)
					if len(rules) != 3 || rules[1].Hostname != "nas.example.com" || rules[2].Service != "http_status:404" {
						t.Error("incorrect ingress ordering")
					}
					fmt.Fprint(response, `{"success":true,"result":{}}`)
				default:
					t.Error("unexpected Cloudflare request")
					response.WriteHeader(500)
				}
			}))
			defer server.Close()
			client := NewCloudflareClient("test-token", "account")
			client.baseURL = server.URL
			err := client.CreateAssistantBinding(context.Background(), models.AIBindingRequest{TunnelID: "tunnel", ZoneID: "zone", Hostname: "nas.example.com", ServiceURL: "http://192.168.1.8:5000"})
			if (err == nil) != (scenario == "success") {
				t.Fatalf("scenario=%s err=%v", scenario, err)
			}
			if scenario == "success" && writes != 2 {
				t.Fatal("missing writes")
			}
			if strings.HasPrefix(scenario, "existing-") && writes != 0 {
				t.Fatal("existing resources overwritten")
			}
			if scenario == "changed-config" && writes != 1 {
				t.Fatal("concurrent config overwritten")
			}
		})
	}
}
