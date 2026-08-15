package services

import (
	"testing"

	"tunnel-manager/models"
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
