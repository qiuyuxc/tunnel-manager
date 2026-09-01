package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"tunnel-manager/models"
	"tunnel-manager/services"
	"tunnel-manager/store"
)

// TestServiceStatusUsesUserTunnel guards against the services health widget
// always probing the global tunnel instead of the requesting user's own.
func TestServiceStatusUsesUserTunnel(t *testing.T) {
	dir := t.TempDir()
	legacy := filepath.Join(dir, "config.json")
	if err := os.WriteFile(legacy, []byte(`{
		"tunnel_id": "global-tunnel",
		"tunnel_name": "Global Tunnel",
		"service_url": "http://localhost:18791"
	}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var probedTunnel string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet && r.URL.Path == "/client/v4/accounts/account-id/cfd_tunnel/user-tunnel/configurations" {
			probedTunnel = "user-tunnel"
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "result": map[string]interface{}{
				"config": map[string]interface{}{"ingress": []models.IngressRule{
					{Hostname: "app.example.com", Service: "http://localhost:3000"},
					{Service: "http_status:404"},
				}},
			}})
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/" {
			// Ingress probe from the services health check.
			w.WriteHeader(http.StatusOK)
			return
		}
		t.Errorf("unexpected Cloudflare request: %s %s", r.Method, r.URL.String())
		http.NotFound(w, r)
	}))
	defer server.Close()

	target, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}
	originalTransport := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		request = request.Clone(request.Context())
		request.URL.Scheme = target.Scheme
		request.URL.Host = target.Host
		return originalTransport.RoundTrip(request)
	})
	t.Cleanup(func() { http.DefaultTransport = originalTransport })

	st := store.NewStore(legacy)
	userID := st.AdminUserID()
	if err := st.SetUserTunnelSelection(userID, "user-tunnel", "User Tunnel"); err != nil {
		t.Fatal(err)
	}

	cf := services.NewCloudflareClient("token", "account-id")
	h := NewMonitorHandler(cf, st)

	req := httptest.NewRequest(http.MethodGet, "/monitor/services", nil)
	req = withUser(req, models.SessionUser{ID: userID, Role: models.RoleAdmin})
	req = withCF(req, cf)
	resp := httptest.NewRecorder()
	h.ServiceStatus(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("ServiceStatus() code = %d: %s", resp.Code, resp.Body.String())
	}
	var body ServicesHealthResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if probedTunnel != "user-tunnel" {
		t.Fatalf("probed tunnel = %q, want user's tunnel", probedTunnel)
	}
	if body.TunnelID != "user-tunnel" || body.TunnelName != "User Tunnel" {
		t.Fatalf("response tunnel = %#v, want user's selection", body)
	}
	if len(body.Services) != 1 || body.Services[0].Hostname != "app.example.com" {
		t.Fatalf("services = %#v, want routed hostname", body.Services)
	}
}
