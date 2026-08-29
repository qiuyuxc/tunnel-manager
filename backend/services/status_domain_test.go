package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"tunnel-manager/models"
	"tunnel-manager/store"
)

func TestProvisionStatusDomainCopiesPanelIngressAndUpsertsDNS(t *testing.T) {
	var updated []models.IngressRule
	var dns models.DNSRecord
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/zones":
			writeCFTestResult(w, []models.Zone{{ID: "zone-id", Name: "example.com"}})
		case r.Method == http.MethodGet && r.URL.Path == "/accounts/account-id/cfd_tunnel/tunnel-id/configurations":
			writeCFTestResult(w, map[string]interface{}{"config": map[string]interface{}{"ingress": []models.IngressRule{
				{Hostname: "panel.example.com", Service: "http://panel:8080", OriginRequest: map[string]interface{}{"httpHostHeader": "panel.internal"}},
				{Hostname: "status.example.com", Service: "http://old:8080"},
				{Service: "http_status:404"},
			}}})
		case r.Method == http.MethodPut && r.URL.Path == "/accounts/account-id/cfd_tunnel/tunnel-id/configurations":
			var payload struct {
				Config struct {
					Ingress []models.IngressRule `json:"ingress"`
				} `json:"config"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Errorf("decode tunnel update: %v", err)
			}
			updated = payload.Config.Ingress
			writeCFTestResult(w, map[string]interface{}{})
		case r.Method == http.MethodGet && r.URL.Path == "/zones/zone-id/dns_records":
			writeCFTestResult(w, []models.DNSRecord{})
		case r.Method == http.MethodPost && r.URL.Path == "/zones/zone-id/dns_records":
			if err := json.NewDecoder(r.Body).Decode(&dns); err != nil {
				t.Errorf("decode DNS record: %v", err)
			}
			writeCFTestResult(w, map[string]interface{}{})
		default:
			t.Errorf("unexpected Cloudflare request: %s %s", r.Method, r.URL.String())
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	svc, userID := newStatusDomainTestService(t, server)
	if err := svc.ProvisionStatusDomain(userID, "panel.example.com", "status.example.com"); err != nil {
		t.Fatalf("ProvisionStatusDomain() error = %v", err)
	}

	if len(updated) != 3 {
		t.Fatalf("updated ingress count = %d, want 3: %#v", len(updated), updated)
	}
	statusRule := updated[1]
	if statusRule.Hostname != "status.example.com" || statusRule.Service != "http://panel:8080" {
		t.Fatalf("status ingress = %#v", statusRule)
	}
	if statusRule.OriginRequest["httpHostHeader"] != "panel.internal" {
		t.Fatalf("status origin request = %#v", statusRule.OriginRequest)
	}
	if updated[2].Service != "http_status:404" {
		t.Fatalf("terminal fallback = %#v", updated[2])
	}
	if dns.Name != "status.example.com" || dns.Type != "CNAME" || dns.Content != "tunnel-id.cfargotunnel.com" || !dns.Proxied {
		t.Fatalf("DNS record = %#v", dns)
	}
}

func TestProvisionStatusDomainDoesNotWriteWithoutPanelIngress(t *testing.T) {
	mutations := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/zones":
			writeCFTestResult(w, []models.Zone{{ID: "zone-id", Name: "example.com"}})
		case r.Method == http.MethodGet && r.URL.Path == "/accounts/account-id/cfd_tunnel/tunnel-id/configurations":
			writeCFTestResult(w, map[string]interface{}{"config": map[string]interface{}{"ingress": []models.IngressRule{
				{Hostname: "other.example.com", Service: "http://other:8080"},
				{Service: "http_status:404"},
			}}})
		default:
			mutations++
			writeCFTestResult(w, map[string]interface{}{})
		}
	}))
	defer server.Close()

	svc, userID := newStatusDomainTestService(t, server)
	err := svc.ProvisionStatusDomain(userID, "panel.example.com", "status.example.com")
	if err == nil || !strings.Contains(err.Error(), "未找到面板域名") {
		t.Fatalf("ProvisionStatusDomain() error = %v", err)
	}
	if mutations != 0 {
		t.Fatalf("Cloudflare mutation count = %d, want 0", mutations)
	}
}

func newStatusDomainTestService(t *testing.T, server *httptest.Server) (*DomainService, string) {
	t.Helper()
	st := store.NewStore(filepath.Join(t.TempDir(), "config.json"))
	userID := st.AdminUserID()
	if err := st.SetUserTunnelSelection(userID, "tunnel-id", "Panel tunnel"); err != nil {
		t.Fatal(err)
	}
	cf := NewCloudflareClient("token", "account-id")
	cf.baseURL = server.URL
	cf.httpClient = server.Client()
	return NewDomainService(cf, st), userID
}

func writeCFTestResult(w http.ResponseWriter, result interface{}) {
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "result": result})
}
