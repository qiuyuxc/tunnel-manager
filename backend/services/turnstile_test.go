package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestVerifyTurnstile(t *testing.T) {
	tests := []struct {
		name      string
		secret    string
		token     string
		action    string
		respond   func(w http.ResponseWriter, r *http.Request)
		wantOK    bool
		wantError bool
	}{
		{
			name:   "success with matching action",
			secret: "s3cr3t", token: "token-1", action: "login",
			respond: func(w http.ResponseWriter, r *http.Request) {
				if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/x-www-form-urlencoded") {
					t.Errorf("content type = %q", ct)
				}
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "action": "login"})
			},
			wantOK: true,
		},
		{
			name:   "action mismatch rejected",
			secret: "s3cr3t", token: "token-1", action: "login",
			respond: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "action": "register"})
			},
			wantOK: false,
		},
		{
			name:   "verifier rejection",
			secret: "s3cr3t", token: "token-1", action: "login",
			respond: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(map[string]any{"success": false, "error-codes": []string{"invalid-input-response"}})
			},
			wantOK: false,
		},
		{
			name:   "transport failure fails closed",
			secret: "s3cr3t", token: "token-1", action: "login",
			respond: func(w http.ResponseWriter, r *http.Request) {
				hj := w.(http.Hijacker)
				conn, _, _ := hj.Hijack()
				conn.Close()
			},
			wantOK:    false,
			wantError: true,
		},
		{
			name:   "empty token rejected without request",
			secret: "s3cr3t", token: "", action: "login",
			wantOK: false,
		},
		{
			name:   "empty secret rejected without request",
			secret: "", token: "token-1", action: "login",
			wantOK: false,
		},
	}

	previous := TurnstileSiteVerifyURL
	defer func() { TurnstileSiteVerifyURL = previous }()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var server *httptest.Server
			if tt.respond != nil {
				server = httptest.NewServer(http.HandlerFunc(tt.respond))
				defer server.Close()
				TurnstileSiteVerifyURL = server.URL
			}
			ok, err := VerifyTurnstile(tt.secret, tt.token, "203.0.113.7", tt.action)
			if ok != tt.wantOK {
				t.Fatalf("VerifyTurnstile() ok = %v, want %v", ok, tt.wantOK)
			}
			if (err != nil) != tt.wantError {
				t.Fatalf("VerifyTurnstile() err = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}
