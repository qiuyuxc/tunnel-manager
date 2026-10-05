package services

import (
	"context"
	"errors"
	"testing"

	"tunnel-manager/models"
)

type labDNSStub struct {
	find   func(context.Context, string) (models.Zone, error)
	ensure func(context.Context, models.Zone, string, string) (bool, error)
}

func (provider labDNSStub) FindZone(ctx context.Context, domain string) (models.Zone, error) {
	return provider.find(ctx, domain)
}

func (provider labDNSStub) EnsureTXT(ctx context.Context, zone models.Zone, name, token string) (bool, error) {
	return provider.ensure(ctx, zone, name, token)
}

func TestLabDNSProvisionReusesChallengeAndWaitsForRealTXT(t *testing.T) {
	runner := newVerifiedLabRunner(t)
	owner := runner.store.AdminUserID()
	initialToken := runner.store.GetLabSecurity().Verification.Token
	settings := runner.store.GetLabSettings()
	settings.Schedule = true
	if err := runner.SaveSettings(settings, owner); err != nil {
		t.Fatal(err)
	}
	txtVisible := false
	runner.resolver = labResolverFunc(func(context.Context, string) ([]string, error) {
		if txtVisible {
			return []string{initialToken}, nil
		}
		return nil, nil
	})
	writes := 0
	provider := labDNSStub{
		find: func(ctx context.Context, domain string) (models.Zone, error) {
			if domain != "example.com" {
				t.Fatalf("unexpected authorization domain: %s", domain)
			}
			return models.Zone{ID: "zone", Name: domain}, nil
		},
		ensure: func(ctx context.Context, zone models.Zone, name, token string) (bool, error) {
			if token != initialToken || token != runner.store.GetLabSecurity().Verification.Token || name != "_tunnel-manager-verify.example.com" || runner.store.GetLabSettings().Schedule {
				t.Fatal("TXT must use persisted authorization with scheduling disabled")
			}
			if writes != 0 {
				return false, nil
			}
			writes++
			return true, nil
		},
	}
	first, err := runner.ProvisionDNS(context.Background(), "example.com", owner, provider)
	if err != nil || first.Verified || !first.RecordCreated || first.Verification.VerifiedAt != 0 || first.Message == "" {
		t.Fatalf("provider success fabricated TXT verification: %#v %v", first, err)
	}
	txtVisible = true
	second, err := runner.ProvisionDNS(context.Background(), "example.com", owner, provider)
	if err != nil || !second.Verified || second.RecordCreated || second.Verification.Token != initialToken || writes != 1 {
		t.Fatalf("retry did not reuse the TXT record: %#v %v", second, err)
	}
	if runner.Status().Running || runner.Status().BudgetUsed != 0 || runner.store.GetLabSettings().Schedule {
		t.Fatal("one-click verification started scanning or enabled scheduling")
	}
}

func TestLabDNSProvisionRejectsUnauthorizedAndFailedWrites(t *testing.T) {
	for _, mode := range []string{"unbound", "wrong-owner", "invalid-domain", "no-zone", "write-denied", "cancelled", "disable-during-discovery", "fresh-challenge"} {
		t.Run(mode, func(t *testing.T) {
			runner := newVerifiedLabRunner(t)
			owner := runner.store.AdminUserID()
			domain := "example.com"
			original := runner.store.GetLabSecurity().Verification
			if mode == "fresh-challenge" {
				if err := runner.store.UpdateLabSecurity(func(state *models.LabSecurityState) error {
					state.Verification = models.LabDomainVerification{}
					return nil
				}); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "wrong-owner" {
				owner = "other-owner"
			}
			if mode == "invalid-domain" {
				domain = "https://owner.example.org"
			}
			writes := 0
			var provider LabDNSProvider = labDNSStub{
				find: func(ctx context.Context, domain string) (models.Zone, error) {
					if mode == "no-zone" {
						return models.Zone{}, errors.New("no managed zone")
					}
					if mode == "disable-during-discovery" {
						app := runner.store.GetAppSettings()
						app.ExperimentalFeatures = false
						if err := runner.store.SetAppSettings(app); err != nil {
							t.Fatal(err)
						}
					}
					return models.Zone{ID: "zone", Name: domain}, nil
				},
				ensure: func(ctx context.Context, zone models.Zone, name, token string) (bool, error) {
					writes++
					if mode == "write-denied" {
						return false, errors.New("DNS write denied")
					}
					if token == "" || token == original.Token || token != runner.store.GetLabSecurity().Verification.Token {
						t.Fatal("new TXT was not generated and persisted before writing")
					}
					return true, nil
				},
			}
			if mode == "unbound" {
				provider = nil
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if mode == "cancelled" {
				cancel()
			}
			result, err := runner.ProvisionDNS(ctx, domain, owner, provider)
			if mode == "fresh-challenge" {
				if err != nil || !result.Verified || !result.RecordCreated || writes != 1 {
					t.Fatalf("fresh automatic authorization failed: %#v %v", result, err)
				}
			} else {
				if err == nil {
					t.Fatal("unauthorized automation succeeded")
				}
				if mode != "write-denied" && writes != 0 {
					t.Fatal("unauthorized automation reached a DNS write")
				}
			}
			if runner.Status().Running || runner.Status().BudgetUsed != 0 {
				t.Fatal("verification performed a scan")
			}
		})
	}
}
