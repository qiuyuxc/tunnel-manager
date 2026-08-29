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

func TestProvisionStatusDomainUsesAuxOriginAndCustomPreferredCNAME(t *testing.T) {
	var updated []models.IngressRule
	var statusDNS, auxDNS models.DNSRecord
	legacyHostnameDeleted := false
	var customHostname struct {
		Hostname           string `json:"hostname"`
		CustomOriginServer string `json:"custom_origin_server"`
		SSL                struct {
			Method string `json:"method"`
			Type   string `json:"type"`
		} `json:"ssl"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/zones":
			writeCFTestResult(w, []models.Zone{
				{ID: "panel-zone", Name: "provider.com"},
				{ID: "status-zone", Name: "customer.com"},
				{ID: "aux-zone", Name: "origin.net"},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/accounts/account-id/cfd_tunnel/tunnel-id/configurations":
			writeCFTestResult(w, map[string]interface{}{"config": map[string]interface{}{"ingress": []models.IngressRule{
				{
					Hostname: "panel.provider.com",
					Service:  "http://panel:8080",
					OriginRequest: map[string]interface{}{
						"httpHostHeader": "panel.internal",
						"noTLSVerify":    true,
					},
				},
				{Hostname: "status.customer.com", Service: "http://old:8080"},
				{Service: "http_status:404"},
			}}})
		case r.Method == http.MethodGet && r.URL.Path == "/zones/panel-zone/custom_hostnames":
			writeCFTestResult(w, []models.CustomHostname{{ID: "legacy-id", Hostname: "status.customer.com"}})
		case r.Method == http.MethodDelete && r.URL.Path == "/zones/panel-zone/custom_hostnames/legacy-id":
			legacyHostnameDeleted = true
			writeCFTestResult(w, map[string]interface{}{})
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
		case r.Method == http.MethodGet && r.URL.Path == "/zones/aux-zone/dns_records":
			writeCFTestResult(w, []models.DNSRecord{})
		case r.Method == http.MethodPost && r.URL.Path == "/zones/aux-zone/dns_records":
			if err := json.NewDecoder(r.Body).Decode(&auxDNS); err != nil {
				t.Errorf("decode auxiliary DNS record: %v", err)
			}
			writeCFTestResult(w, map[string]interface{}{})
		case r.Method == http.MethodGet && r.URL.Path == "/zones/status-zone/dns_records":
			writeCFTestResult(w, []models.DNSRecord{})
		case r.Method == http.MethodPost && r.URL.Path == "/zones/status-zone/dns_records":
			if err := json.NewDecoder(r.Body).Decode(&statusDNS); err != nil {
				t.Errorf("decode status DNS record: %v", err)
			}
			writeCFTestResult(w, map[string]interface{}{})
		case r.Method == http.MethodGet && r.URL.Path == "/zones/aux-zone/custom_hostnames":
			writeCFTestResult(w, []models.CustomHostname{})
		case r.Method == http.MethodPost && r.URL.Path == "/zones/aux-zone/custom_hostnames":
			if err := json.NewDecoder(r.Body).Decode(&customHostname); err != nil {
				t.Errorf("decode custom hostname: %v", err)
			}
			writeCFTestResult(w, models.CustomHostname{ID: "custom-hostname-id", Hostname: "status.customer.com"})
		default:
			t.Errorf("unexpected Cloudflare request: %s %s", r.Method, r.URL.String())
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	svc, userID := newStatusDomainTestService(t, server)
	if err := svc.ProvisionStatusDomain(
		userID,
		"panel.provider.com",
		"status.customer.com",
		BindingModePreferred,
		"origin.origin.net",
		"custom.edge.example",
	); err != nil {
		t.Fatalf("ProvisionStatusDomain() error = %v", err)
	}

	if len(updated) != 4 {
		t.Fatalf("updated ingress count = %d, want 4: %#v", len(updated), updated)
	}
	for _, index := range []int{1, 2} {
		rule := updated[index]
		if rule.Service != "http://panel:8080" {
			t.Fatalf("managed ingress %d = %#v", index, rule)
		}
		if _, exists := rule.OriginRequest["httpHostHeader"]; exists {
			t.Fatalf("managed ingress overrides Host header: %#v", rule.OriginRequest)
		}
		if verify, ok := rule.OriginRequest["noTLSVerify"].(bool); !ok || !verify {
			t.Fatalf("managed ingress lost safe origin settings: %#v", rule.OriginRequest)
		}
	}
	if updated[1].Hostname != "status.customer.com" || updated[2].Hostname != "origin.origin.net" {
		t.Fatalf("managed ingress hostnames = (%q, %q)", updated[1].Hostname, updated[2].Hostname)
	}
	if updated[3].Service != "http_status:404" {
		t.Fatalf("terminal fallback = %#v", updated[3])
	}
	if !legacyHostnameDeleted {
		t.Fatal("legacy custom hostname in panel zone was not deleted")
	}
	if customHostname.Hostname != "status.customer.com" ||
		customHostname.CustomOriginServer != "origin.origin.net" ||
		customHostname.SSL.Method != "http" ||
		customHostname.SSL.Type != "dv" {
		t.Fatalf("custom hostname = %#v", customHostname)
	}
	if statusDNS.Name != "status.customer.com" ||
		statusDNS.Content != "custom.edge.example" ||
		statusDNS.Proxied {
		t.Fatalf("status DNS record = %#v", statusDNS)
	}
	if auxDNS.Name != "origin.origin.net" ||
		auxDNS.Content != "tunnel-id.cfargotunnel.com" ||
		!auxDNS.Proxied {
		t.Fatalf("auxiliary DNS record = %#v", auxDNS)
	}
}

func TestProvisionStatusDomainDirectUsesProxiedTunnelCNAMEAndRemovesSaaSHostname(t *testing.T) {
	var updated []models.IngressRule
	var dns models.DNSRecord
	customHostnameDeleted := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/zones":
			writeCFTestResult(w, []models.Zone{
				{ID: "provider-zone", Name: "provider.com"},
				{ID: "status-zone", Name: "customer.com"},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/accounts/account-id/cfd_tunnel/tunnel-id/configurations":
			writeCFTestResult(w, map[string]interface{}{"config": map[string]interface{}{"ingress": []models.IngressRule{
				{Hostname: "panel.provider.com", Service: "http://panel:8080", OriginRequest: map[string]interface{}{"httpHostHeader": "panel.internal"}},
				{Hostname: "status.customer.com", Service: "http://old:8080"},
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
		case r.Method == http.MethodGet && r.URL.Path == "/zones/status-zone/dns_records":
			writeCFTestResult(w, []models.DNSRecord{})
		case r.Method == http.MethodPost && r.URL.Path == "/zones/status-zone/dns_records":
			if err := json.NewDecoder(r.Body).Decode(&dns); err != nil {
				t.Errorf("decode DNS record: %v", err)
			}
			writeCFTestResult(w, map[string]interface{}{})
		case r.Method == http.MethodGet && r.URL.Path == "/zones/provider-zone/custom_hostnames":
			writeCFTestResult(w, []models.CustomHostname{{ID: "custom-hostname-id", Hostname: "status.customer.com"}})
		case r.Method == http.MethodDelete && r.URL.Path == "/zones/provider-zone/custom_hostnames/custom-hostname-id":
			customHostnameDeleted = true
			writeCFTestResult(w, map[string]interface{}{})
		default:
			t.Errorf("unexpected Cloudflare request: %s %s", r.Method, r.URL.String())
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	svc, userID := newStatusDomainTestService(t, server)
	svc.store.SetPreferredCNAME("")
	if err := svc.ProvisionStatusDomain(userID, "panel.provider.com", "status.customer.com", BindingModeSimple, "origin.provider.com", ""); err != nil {
		t.Fatalf("ProvisionStatusDomain() error = %v", err)
	}

	if len(updated) != 3 {
		t.Fatalf("updated ingress count = %d, want 3: %#v", len(updated), updated)
	}
	statusRule := updated[1]
	if statusRule.Hostname != "status.customer.com" || statusRule.Service != "http://panel:8080" {
		t.Fatalf("status ingress = %#v", statusRule)
	}
	if statusRule.OriginRequest != nil {
		t.Fatalf("status ingress overrides origin request: %#v", statusRule.OriginRequest)
	}
	if dns.Name != "status.customer.com" ||
		dns.Type != "CNAME" ||
		dns.Content != "tunnel-id.cfargotunnel.com" ||
		!dns.Proxied {
		t.Fatalf("DNS record = %#v", dns)
	}
	if !customHostnameDeleted {
		t.Fatal("existing SaaS custom hostname was not deleted")
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
	err := svc.ProvisionStatusDomain(userID, "panel.example.com", "status.example.com", BindingModePreferred, "origin.example.com", "custom.edge.example")
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
	st.SetPreferredCNAME("preferred.edge.example")
	cf := NewCloudflareClient("token", "account-id")
	cf.baseURL = server.URL
	cf.httpClient = server.Client()
	return NewDomainService(cf, st), userID
}

func writeCFTestResult(w http.ResponseWriter, result interface{}) {
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "result": result})
}
