package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tunnel-manager/models"
	"tunnel-manager/store"
)

func TestNormalizeBindingMode(t *testing.T) {
	tests := []struct {
		name, input, want string
		wantErr           bool
	}{
		{"legacy default", "", BindingModePreferred, false},
		{"simple", " SIMPLE ", BindingModeSimple, false},
		{"preferred", "preferred", BindingModePreferred, false},
		{"invalid", "direct", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeBindingMode(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error=%v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
			}
		})
	}
}

func TestMergeIngressRulesPreservesPathOnlyRules(t *testing.T) {
	originRequest := map[string]interface{}{"noTLSVerify": true}
	existing := []models.IngressRule{
		{Path: "/health", Service: "http://health:8080", OriginRequest: originRequest},
		{Hostname: "old.example.com", Service: "http://old:8080"},
		{Service: "http_status:404"},
	}
	managed := []models.IngressRule{{Hostname: "app.example.com", Service: "http://app:8080"}}

	got := mergeIngressRules(existing, managed, map[string]bool{"app.example.com": true})
	if len(got) != 4 {
		t.Fatalf("got %d rules, want 4: %+v", len(got), got)
	}
	if got[0].Path != "/health" || got[0].Service != "http://health:8080" {
		t.Fatalf("path-only rule was not preserved: %+v", got[0])
	}
	if verify, ok := got[0].OriginRequest["noTLSVerify"].(bool); !ok || !verify {
		t.Fatalf("origin request was not preserved: %+v", got[0].OriginRequest)
	}
	if got[2].Hostname != managed[0].Hostname || got[2].Service != managed[0].Service {
		t.Fatalf("managed rule inserted at wrong position: %+v", got)
	}
	if got[3].Service != "http_status:404" || got[3].Hostname != "" || got[3].Path != "" {
		t.Fatalf("terminal catch-all changed: %+v", got[3])
	}
}

func TestMergeIngressRulesReplacesManagedHostnameOnly(t *testing.T) {
	existing := []models.IngressRule{
		{Hostname: "app.example.com", Path: "/api", Service: "http://old:8080"},
		{Path: "/public", Service: "http://public:8080"},
		{Service: "http_status:404"},
	}
	managed := []models.IngressRule{{Hostname: "app.example.com", Service: "http://new:8080"}}

	got := mergeIngressRules(existing, managed, map[string]bool{"app.example.com": true})
	if len(got) != 3 {
		t.Fatalf("got %d rules, want 3: %+v", len(got), got)
	}
	if got[0].Path != "/public" {
		t.Fatalf("unrelated path-only rule removed: %+v", got)
	}
	if got[1].Hostname != managed[0].Hostname || got[1].Service != managed[0].Service || got[2].Service != "http_status:404" {
		t.Fatalf("unexpected rule order: %+v", got)
	}
}

// TestBindDomainUsesUserSelections guards against binding always falling back
// to the global tunnel/service URL instead of the requesting user's own
// selections saved through the UI.
func TestBindDomainUsesUserSelections(t *testing.T) {
	dir := t.TempDir()
	legacy := filepath.Join(dir, "config.json")
	if err := os.WriteFile(legacy, []byte(`{
		"tunnel_id": "global-tunnel",
		"service_url": "http://localhost:18791",
		"preferred_cname": "cf.edge.example"
	}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var updatedIngress []models.IngressRule
	var createdCNAME models.DNSRecord
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/zones":
			writeCFTestResult(w, []models.Zone{{ID: "zone-1", Name: "example.com"}})
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/configurations"):
			writeCFTestResult(w, map[string]interface{}{"config": map[string]interface{}{"ingress": []models.IngressRule{
				{Service: "http_status:404"},
			}}})
		case r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/configurations"):
			var payload struct {
				Config struct {
					Ingress []models.IngressRule `json:"ingress"`
				} `json:"config"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Errorf("decode tunnel update: %v", err)
			}
			updatedIngress = payload.Config.Ingress
			writeCFTestResult(w, map[string]interface{}{})
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/zones/zone-1/dns_records"):
			writeCFTestResult(w, []models.DNSRecord{})
		case r.Method == http.MethodPost && r.URL.Path == "/zones/zone-1/dns_records":
			if err := json.NewDecoder(r.Body).Decode(&createdCNAME); err != nil {
				t.Errorf("decode DNS record: %v", err)
			}
			writeCFTestResult(w, models.DNSRecord{ID: "dns-1"})
		default:
			t.Errorf("unexpected Cloudflare request: %s %s", r.Method, r.URL.String())
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	st := store.NewStore(legacy)
	userID := st.AdminUserID()
	if err := st.SetUserTunnelSelection(userID, "user-tunnel", "User tunnel"); err != nil {
		t.Fatal(err)
	}
	if err := st.SetUserServiceURL(userID, "http://localhost:3000"); err != nil {
		t.Fatal(err)
	}

	cf := NewCloudflareClient("token", "account-id")
	cf.baseURL = server.URL
	cf.httpClient = server.Client()
	svc := NewDomainService(cf, st).ForUser(userID)

	if _, _, err := svc.BindDomainWithConfiguredService(BindingModeSimple, "app.example.com", "", ""); err != nil {
		t.Fatalf("BindDomainWithConfiguredService() error = %v", err)
	}

	if len(updatedIngress) != 2 {
		t.Fatalf("updated ingress = %#v, want hostname rule plus catch-all", updatedIngress)
	}
	rule := updatedIngress[0]
	if rule.Hostname != "app.example.com" || rule.Service != "http://localhost:3000" {
		t.Fatalf("ingress rule = %#v, want user's service URL", rule)
	}
	if createdCNAME.Content != "user-tunnel.cfargotunnel.com" {
		t.Fatalf("CNAME content = %q, want user's tunnel", createdCNAME.Content)
	}
}
