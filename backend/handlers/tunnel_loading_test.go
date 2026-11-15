package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"tunnel-manager/services"
)

func TestTunnelDetailLoadsOnlySelectedTunnelInParallel(t *testing.T) {
	var requests atomic.Int32
	started := make(chan struct{}, 2)
	release := make(chan struct{})
	transport := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		requests.Add(1)
		started <- struct{}{}
		select {
		case <-release:
		case <-r.Context().Done():
			return nil, r.Context().Err()
		}
		var result any
		switch r.URL.Path {
		case "/client/v4/accounts/account/cfd_tunnel/tunnel":
			result = map[string]string{"id": "tunnel", "name": "Production", "status": "healthy"}
		case "/client/v4/accounts/account/cfd_tunnel/tunnel/configurations":
			result = map[string]any{"config": map[string]any{"ingress": []any{map[string]string{"hostname": "app.example.com", "service": "http://localhost:3000"}}}}
		default:
			t.Errorf("unnecessary upstream request: %s", r.URL.Path)
		}
		body, _ := json.Marshal(map[string]any{"success": true, "result": result})
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(string(body)))}, nil
	})
	t.Cleanup(func() { http.DefaultTransport = transport })
	cf := services.NewCloudflareClient("test-token", "account")
	h := NewTunnelHandler(cf, newTestStore(t))
	router := chi.NewRouter()
	router.Get("/tunnels/{tunnelID}", h.GetTunnelDetail)
	req := withCF(httptest.NewRequest(http.MethodGet, "/tunnels/tunnel", nil), cf)
	w := httptest.NewRecorder()
	done := make(chan struct{})
	go func() { router.ServeHTTP(w, req); close(done) }()
	for i := 0; i < 2; i++ {
		select {
		case <-started:
		case <-time.After(time.Second):
			close(release)
			<-done
			t.Fatal("detail and configuration did not start in parallel")
		}
	}
	close(release)
	<-done
	if w.Code != http.StatusOK || requests.Load() != 2 || !strings.Contains(w.Body.String(), "Production") {
		t.Fatalf("detail: %d %s; requests=%d", w.Code, w.Body.String(), requests.Load())
	}
}
