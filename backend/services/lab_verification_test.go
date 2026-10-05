package services

import (
	"context"
	"errors"
	"strings"
	"testing"

	"tunnel-manager/models"
)

type labResolverFunc func(context.Context, string) ([]string, error)

func (resolve labResolverFunc) LookupTXT(ctx context.Context, name string) ([]string, error) {
	return resolve(ctx, name)
}

func TestLabHostnameAndOwnershipBoundaries(t *testing.T) {
	for _, host := range []string{"localhost", "1.1.1.1", "https://example.com", "example.com:443", "a..com", "co.uk", "pages.dev", "x\r\n.example.com", "-bad.example.com"} {
		if _, err := NormalizeLabHostname(host); err == nil {
			t.Errorf("accepted invalid host %q", host)
		}
	}
	if host, err := NormalizeLabHostname(" Probe.Example.COM. "); err != nil || host != "probe.example.com" {
		t.Fatalf("normalization: %q %v", host, err)
	}
	for _, domain := range []string{"", "com", "co.uk", "https://example.com", "127.0.0.1"} {
		if _, err := LabVerificationDomain(domain); err == nil {
			t.Errorf("accepted authorization for %q", domain)
		}
	}
	if domain, err := LabVerificationDomain(" Owner.Example.CO.UK. "); err != nil || domain != "owner.example.co.uk" {
		t.Fatalf("public suffix boundary: %q %v", domain, err)
	}
	for _, path := range []string{"//evil.com", "https://evil.com/", "/ HTTP/1.1\r\nHost: evil.com", "/a\tb"} {
		settings := models.LabIPSelectorSettings{Host: "probe.example.com", Path: path}
		if err := NormalizeLabProbe(&settings); err == nil {
			t.Errorf("accepted unsafe path %q", path)
		}
	}
}

func TestLabPublicTargets(t *testing.T) {
	for _, address := range []string{"127.0.0.1", "10.1.1.1", "172.16.0.1", "192.168.0.1", "169.254.169.254", "100.64.0.1", "192.0.2.1", "0.0.0.0", "224.0.0.1", "255.255.255.255", "::1", "::ffff:127.0.0.1", "fc00::1", "fe80::1", "2001:db8::1", "2002:7f00:1::", "64:ff9b::7f00:1"} {
		if err := ValidateLabPublicTargets([]string{address}); err == nil {
			t.Errorf("accepted forbidden address %s", address)
		}
	}
	if err := ValidateLabPublicTargets([]string{"1.1.1.1", "2606:4700:4700::1111"}); err != nil {
		t.Fatal(err)
	}
}

func TestLabVerificationScopeAndOwner(t *testing.T) {
	for _, scenario := range []struct {
		name   string
		domain string
		host   string
		sni    string
		owner  string
		valid  bool
	}{
		{"root", "example.com", "probe.example.com", "tls.example.com", "admin", true},
		{"exact-host", "probe.example.com", "probe.example.com", "probe.example.com", "admin", true},
		{"nested", "probe.example.com", "check.probe.example.com", "tls.probe.example.com", "admin", true},
		{"independent-sni", "probe.example.com", "probe.example.com", "tls.example.net", "admin", true},
		{"independent-host", "example.com", "probe.example.net", "probe.example.net", "admin", true},
		{"invalid-domain", "com", "probe.example.com", "probe.example.com", "admin", false},
		{"other-owner", "example.com", "probe.example.com", "probe.example.com", "other-admin", false},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			verification := models.LabDomainVerification{Domain: scenario.domain, Host: scenario.host, SNI: scenario.sni,
				OwnerID: "admin", Token: "test-token", RecordName: "_tunnel-manager-verify." + scenario.domain, VerifiedAt: 123}
			if err := validateLabVerification(verification, scenario.owner); (err == nil) != scenario.valid {
				t.Fatalf("unexpected scope result: %v", err)
			}
		})
	}
}

func TestLabTXTLookup(t *testing.T) {
	verification := models.LabDomainVerification{RecordName: "_tunnel-manager-verify.example.com", Token: "random-token"}
	for _, mode := range []string{"valid", "missing", "error", "cancelled"} {
		t.Run(mode, func(t *testing.T) {
			resolver := labResolverFunc(func(ctx context.Context, name string) ([]string, error) {
				if !strings.HasSuffix(name, ".") {
					t.Error("lookup must use absolute DNS name")
				}
				if mode == "error" {
					return nil, errors.New("DNS unavailable")
				}
				if mode == "cancelled" {
					<-ctx.Done()
					return nil, ctx.Err()
				}
				if mode == "valid" {
					return []string{"unrelated", verification.Token}, nil
				}
				return []string{"wrong-token"}, nil
			})
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if mode == "cancelled" {
				cancel()
			}
			if err := lookupLabVerification(ctx, resolver, verification); (err == nil) != (mode == "valid") {
				t.Fatalf("lookup %s: %v", mode, err)
			}
		})
	}
}
