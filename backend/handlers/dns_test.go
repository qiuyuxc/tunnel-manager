package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tunnel-manager/models"
)

func TestReadDNSRecordRequestNormalizesRecords(t *testing.T) {
	tests := []struct {
		name, body string
		ok         bool
		check      func(*testing.T, models.DNSRecordRequest)
	}{
		{"cname", `{"name":" app.example.com. ","type":"cname","content":" target.example.com ","proxied":true,"ttl":0}`, true, func(t *testing.T, p models.DNSRecordRequest) {
			if p.Name != "app.example.com" || p.Type != "CNAME" || p.TTL != 1 || !p.Proxied {
				t.Fatalf("unexpected: %+v", p)
			}
		}},
		{"txt disables proxy", `{"name":"example.com","type":"TXT","content":"value","proxied":true,"ttl":300}`, true, func(t *testing.T, p models.DNSRecordRequest) {
			if p.Proxied {
				t.Fatal("TXT remained proxied")
			}
		}},
		{"mx priority required", `{"name":"example.com","type":"MX","content":"mail.example.com","ttl":300}`, false, nil},
		{"unsupported", `{"name":"example.com","type":"SRV","content":"x","ttl":300}`, false, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			w := httptest.NewRecorder()
			payload, ok := readDNSRecordRequest(w, r, "zone")
			if ok != tt.ok {
				t.Fatalf("ok=%v body=%s", ok, w.Body.String())
			}
			if ok && tt.check != nil {
				tt.check(t, payload)
			}
		})
	}
}
