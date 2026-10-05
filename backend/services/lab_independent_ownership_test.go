package services

import (
	"context"
	"testing"

	"tunnel-manager/models"
)

func TestLabOwnershipDoesNotRequireProbeConfiguration(t *testing.T) {
	runner := newVerifiedLabRunner(t)
	owner := runner.store.AdminUserID()
	if err := runner.store.SetLabSettings(models.LabIPSelectorSettings{}); err != nil {
		t.Fatal(err)
	}
	verification, err := runner.NewChallenge("owner.example.org", owner)
	if err != nil {
		t.Fatal(err)
	}
	if verification.Domain != "owner.example.org" || verification.RecordName != "_tunnel-manager-verify.owner.example.org" {
		t.Fatalf("unexpected independent verification: %#v", verification)
	}
	if _, err := runner.Verify(context.Background(), owner); err != nil {
		t.Fatal(err)
	}
	if settings := runner.store.GetLabSettings(); settings.Host != "" || settings.SNI != "" {
		t.Fatalf("verification changed probe settings: %#v", settings)
	}
}

func TestLabProbeChangesPreserveIndependentOwnership(t *testing.T) {
	runner := newVerifiedLabRunner(t)
	owner := runner.store.AdminUserID()
	before := runner.store.GetLabSecurity().Verification
	settings := runner.store.GetLabSettings()
	settings.Host, settings.SNI = "probe.example.net", "tls.example.net"
	if err := runner.SaveSettings(settings, owner); err != nil {
		t.Fatal(err)
	}
	after := runner.store.GetLabSecurity().Verification
	if after.Token != before.Token || after.Domain != before.Domain || after.VerifiedAt == 0 {
		t.Fatalf("probe change invalidated domain verification: %#v", after)
	}
	if _, err := runner.prepare(context.Background(), owner); err != nil {
		t.Fatal(err)
	}
}

func TestLabAutomaticTXTUsesIndependentDomain(t *testing.T) {
	runner := newVerifiedLabRunner(t)
	owner := runner.store.AdminUserID()
	settings := runner.store.GetLabSettings()
	writes := 0
	provider := labDNSStub{
		find: func(ctx context.Context, domain string) (models.Zone, error) {
			if domain != "owner.example.org" {
				t.Fatalf("DNS lookup used probe domain: %s", domain)
			}
			return models.Zone{ID: "owner-zone", Name: "example.org"}, nil
		},
		ensure: func(ctx context.Context, zone models.Zone, name, token string) (bool, error) {
			writes++
			if name != "_tunnel-manager-verify.owner.example.org" || token == "" {
				t.Fatalf("unexpected verification record: %s", name)
			}
			return true, nil
		},
	}
	result, err := runner.ProvisionDNS(context.Background(), "owner.example.org", owner, provider)
	if err != nil || !result.Verified || writes != 1 {
		t.Fatalf("independent TXT provisioning failed: %#v, %v", result, err)
	}
	after := runner.store.GetLabSettings()
	if after.Host != settings.Host || after.SNI != settings.SNI || after.Path != settings.Path {
		t.Fatal("TXT provisioning overwrote probe settings")
	}
}
