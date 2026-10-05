package models

import (
	"encoding/json"
	"testing"
)

func TestDNSRecordValidation(t *testing.T) {
	for _, test := range []struct {
		name, body string
		valid      bool
	}{
		{"A", `{"type":"A","content":"192.0.2.1"}`, true},
		{"AAAA", `{"type":"AAAA","content":"2001:db8::1"}`, true},
		{"CNAME", `{"type":"CNAME","content":"target.example.com."}`, true},
		{"TXT", `{"type":"TXT","content":"  exact TXT  "}`, true},
		{"MX", `{"type":"MX","content":"mail.example.com","priority":0}`, true},
		{"null MX", `{"type":"MX","content":".","priority":0}`, true},
		{"NS", `{"type":"NS","content":"ns.example.com"}`, true},
		{"PTR", `{"type":"PTR","content":"host.example.com"}`, true},
		{"SRV", `{"name":"_sip._tcp.example.com","type":"SRV","data":{"priority":0,"weight":65535,"port":0,"target":"."}}`, true},
		{"CAA", `{"type":"CAA","data":{"flags":128,"tag":"issuewild","value":";"}}`, true},
		{"empty CAA", `{"type":"CAA","data":{"flags":0,"tag":"issue","value":""}}`, true},
		{"unknown", `{"type":"HTTPS","content":"x"}`, false},
		{"A family", `{"type":"A","content":"::1"}`, false},
		{"AAAA family", `{"type":"AAAA","content":"1.2.3.4"}`, false},
		{"AAAA zone", `{"type":"AAAA","content":"fe80::1%eth0"}`, false},
		{"A leading zero", `{"type":"A","content":"01.2.3.4"}`, false},
		{"NS IP", `{"type":"NS","content":"1.2.3.4"}`, false},
		{"PTR URL", `{"type":"PTR","content":"https://example.com"}`, false},
		{"MX priority missing", `{"type":"MX","content":"mail.example.com"}`, false},
		{"MX priority range", `{"type":"MX","content":"mail.example.com","priority":65536}`, false},
		{"MX null priority", `{"type":"MX","content":".","priority":10}`, false},
		{"SRV text", `{"type":"SRV","content":"10 1 443 target.example.com"}`, false},
		{"SRV incomplete", `{"name":"_sip._tcp.example.com","type":"SRV","data":{"priority":0,"port":443,"target":"example.com"}}`, false},
		{"SRV name", `{"type":"SRV","data":{"priority":0,"weight":0,"port":443,"target":"example.com"}}`, false},
		{"SRV content", `{"name":"_sip._tcp.example.com","type":"SRV","content":"old","data":{"priority":0,"weight":0,"port":443,"target":"example.com"}}`, false},
		{"CAA flags", `{"type":"CAA","data":{"flags":256,"tag":"issue","value":"ca.example"}}`, false},
		{"CAA value missing", `{"type":"CAA","data":{"flags":0,"tag":"issue"}}`, false},
		{"CAA cross fields", `{"type":"CAA","data":{"flags":0,"tag":"issue","value":"ca.example","port":443}}`, false},
		{"data on TXT", `{"type":"TXT","content":"x","data":{"flags":0}}`, false},
		{"TTL zero", `{"type":"TXT","content":"x","ttl":0}`, false},
		{"TTL low", `{"type":"TXT","content":"x","ttl":59}`, false},
		{"TTL high", `{"type":"TXT","content":"x","ttl":86401}`, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			payload := DNSRecordRequest{Name: "example.com", TTL: 1, Proxied: true}
			if err := json.Unmarshal([]byte(test.body), &payload); err != nil {
				t.Fatal(err)
			}
			err := ValidateDNSRecordRequest(&payload)
			if (err == nil) != test.valid {
				t.Fatalf("valid=%v error=%v payload=%+v", test.valid, err, payload)
			}
			if test.valid && payload.Proxied != DNSProxyEligible(payload.Type) {
				t.Fatal("proxy mismatch")
			}
			if test.name == "TXT" && payload.Content != "  exact TXT  " {
				t.Fatal("TXT whitespace lost")
			}
		})
	}
}

func TestDNSStructuredRoundTrip(t *testing.T) {
	for _, raw := range []string{
		`{"type":"SRV","name":"_sip._tcp.example.com","ttl":60,"data":{"priority":0,"weight":5,"port":5060,"target":"sip.example.com","service":"_sip","proto":"_tcp","name":"example.com"}}`,
		`{"type":"CAA","name":"example.com","ttl":1,"data":{"flags":128,"tag":"iodef","value":"mailto:security@example.com"}}`,
	} {
		var record DNSRecord
		if err := json.Unmarshal([]byte(raw), &record); err != nil {
			t.Fatal(err)
		}
		var request DNSRecordRequest
		if err := json.Unmarshal([]byte(raw), &request); err != nil {
			t.Fatal(err)
		}
		if err := ValidateDNSRecordRequest(&request); err != nil {
			t.Fatal(err)
		}
		result, _ := json.Marshal(request.Data)
		var actual, expected map[string]any
		json.Unmarshal(result, &actual)
		json.Unmarshal(record.Data, &expected)
		if len(actual) != len(expected) {
			t.Fatalf("lost fields: %s", result)
		}
		for key, value := range expected {
			if actual[key] != value {
				t.Fatalf("lost %s: %s", key, result)
			}
		}
	}
}
