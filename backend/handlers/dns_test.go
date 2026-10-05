package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"tunnel-manager/models"
	"tunnel-manager/services"
)

func TestReadDNSRecordRequestNormalizesRecords(t *testing.T) {
	tests := []struct {
		name, body string
		ok         bool
		check      func(*testing.T, models.DNSRecordRequest)
	}{
		{"cname", `{"name":" app.example.com. ","type":"cname","content":" target.example.com ","proxied":true}`, true, func(t *testing.T, p models.DNSRecordRequest) {
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
		{"zero TTL", `{"name":"example.com","type":"TXT","content":"x","ttl":0}`, false, nil},
		{"null TTL", `{"name":"example.com","type":"TXT","content":"x","ttl":null}`, false, nil},
		{"null proxy", `{"name":"example.com","type":"TXT","content":"x","proxied":null}`, false, nil},
		{"trailing JSON", `{"name":"example.com","type":"TXT","content":"x"} {}`, false, nil},
		{"unknown data", `{"name":"example.com","type":"CAA","data":{"flags":0,"tag":"issue","value":"ca.example","future":true}}`, false, nil},
		{"default TXT", `{"name":"example.com","type":"TXT","content":" exact "}`, true, func(t *testing.T, p models.DNSRecordRequest) {
			if p.TTL != 1 || p.Proxied || p.Content != " exact " {
				t.Fatalf("unexpected defaults: %+v", p)
			}
		}},
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

type dnsMockTransport func(*http.Request) (*http.Response, error)

func (transport dnsMockTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

func TestDNSHandlersScopedStructuredRoundTrip(t *testing.T) {
	originalTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = originalTransport })
	for _, body := range []string{
		`{"type":"SRV","name":"_sip._tcp.example.com","ttl":300,"data":{"priority":0,"weight":5,"port":5060,"target":"sip.example.com","service":"_sip","proto":"_tcp","name":"example.com"}}`,
		`{"type":"CAA","name":"example.com","data":{"flags":128,"tag":"iodef","value":"mailto:security@example.com"}}`,
		`{"type":"NS","name":"sub.example.com","content":"ns.example.net","proxied":true}`,
		`{"type":"MX","name":"example.com","content":"mail.example.net","priority":0}`,
	} {
		var stored map[string]any
		writes := 0
		http.DefaultTransport = dnsMockTransport(func(request *http.Request) (*http.Response, error) {
			if request.Header.Get("Authorization") != "Bearer current-user-token" || !strings.HasPrefix(request.URL.Path, "/client/v4/zones/current-zone/dns_records") {
				t.Fatalf("wrong identity or resource: %s %s", request.Header.Get("Authorization"), request.URL)
			}
			if request.Method == http.MethodPost || request.Method == http.MethodPut {
				writes++
				if err := json.NewDecoder(request.Body).Decode(&stored); err != nil {
					t.Fatal(err)
				}
				if stored["proxied"] != false {
					t.Fatal("unexpected proxy")
				}
				if stored["type"] == "SRV" || stored["type"] == "CAA" {
					if _, exists := stored["content"]; exists {
						t.Fatal("structured content leaked")
					}
				}
				stored["id"] = "record"
			}
			encoded, _ := json.Marshal(map[string]any{"success": true, "result": stored})
			return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(string(encoded)))}, nil
		})
		handler := NewDNSHandler(services.NewCloudflareClient("wrong-user-token", "wrong-account"))
		invoke := func(method, payload string, action http.HandlerFunc) *httptest.ResponseRecorder {
			request := httptest.NewRequest(method, "/", strings.NewReader(payload))
			route := chi.NewRouteContext()
			route.URLParams.Add("zoneID", "current-zone")
			route.URLParams.Add("recordID", "record")
			ctx := context.WithValue(request.Context(), chi.RouteCtxKey, route)
			ctx = context.WithValue(ctx, cfContextKey{}, services.NewCloudflareClient("current-user-token", "current-account"))
			holder := &auditTargetHolder{}
			ctx = context.WithValue(ctx, auditTargetKey{}, holder)
			response := httptest.NewRecorder()
			action(response, request.WithContext(ctx))
			if response.Code >= 400 {
				t.Fatalf("%s: %s", method, response.Body.String())
			}
			if method != http.MethodGet && holder.value == "" {
				t.Fatal("missing audit target")
			}
			return response
		}
		created := invoke(http.MethodPost, body, handler.Create)
		var record models.DNSRecord
		json.Unmarshal(created.Body.Bytes(), &record)
		var update models.DNSRecordRequest
		json.Unmarshal([]byte(body), &update)
		update.TTL = 600
		encoded, _ := json.Marshal(update)
		updated := invoke(http.MethodPut, string(encoded), handler.Update)
		var roundtrip models.DNSRecord
		json.Unmarshal(updated.Body.Bytes(), &roundtrip)
		if string(record.Data) != string(roundtrip.Data) || record.Content != roundtrip.Content || roundtrip.TTL != 600 || writes != 2 {
			t.Fatalf("roundtrip mismatch: %s", updated.Body.String())
		}
	}
}

func TestDNSHandlersProtectUnknownRecords(t *testing.T) {
	originalTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = originalTransport })
	http.DefaultTransport = dnsMockTransport(func(request *http.Request) (*http.Response, error) {
		if request.Method != http.MethodGet {
			t.Fatal("unknown record mutated")
		}
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"success":true,"result":{"id":"unknown","type":"HTTPS","name":"example.com","data":{"future":"preserved"}}}`))}, nil
	})
	handler := NewDNSHandler(nil)
	for _, method := range []string{http.MethodPut, http.MethodDelete} {
		request := httptest.NewRequest(method, "/", strings.NewReader(`{"name":"example.com","type":"TXT","content":"replace"}`))
		route := chi.NewRouteContext()
		route.URLParams.Add("zoneID", "zone")
		route.URLParams.Add("recordID", "unknown")
		ctx := context.WithValue(request.Context(), chi.RouteCtxKey, route)
		ctx = context.WithValue(ctx, cfContextKey{}, services.NewCloudflareClient("mock", "mock"))
		response := httptest.NewRecorder()
		if method == http.MethodPut {
			handler.Update(response, request.WithContext(ctx))
		} else {
			handler.Delete(response, request.WithContext(ctx))
		}
		if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "read-only") {
			t.Fatalf("unprotected: %s", response.Body.String())
		}
	}
}
