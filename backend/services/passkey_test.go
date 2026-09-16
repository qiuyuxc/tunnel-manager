package services

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"

	"tunnel-manager/models"
	"tunnel-manager/store"
)

func newPasskeyTestService(t *testing.T) (*PasskeyService, *store.Store) {
	t.Helper()
	st := store.NewStore(filepath.Join(t.TempDir(), "config.json"))
	return NewPasskeyService(st), st
}

func TestResolveRelyingPartyFromRequestHost(t *testing.T) {
	svc, _ := newPasskeyTestService(t)
	req := httptest.NewRequest(http.MethodPost, "https://panel.example.com/api/auth/passkey/login/begin", nil)

	rp, err := svc.ResolveRelyingParty(req)
	if err != nil {
		t.Fatalf("ResolveRelyingParty() error = %v", err)
	}
	if rp.ID != "panel.example.com" {
		t.Fatalf("relying party id = %q, want the request host", rp.ID)
	}
	if len(rp.Origins) != 1 || rp.Origins[0] != "https://panel.example.com" {
		t.Fatalf("origins = %#v, want the request origin", rp.Origins)
	}
}

func TestResolveRelyingPartyHonoursAdministratorOverrides(t *testing.T) {
	svc, st := newPasskeyTestService(t)
	settings := st.GetAppSettings()
	settings.PasskeyRPID = "example.com"
	settings.PasskeyOrigins = "https://panel.example.com,https://alt.example.com/"
	if err := st.SetAppSettings(settings); err != nil {
		t.Fatalf("SetAppSettings() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "https://panel.example.com/api/auth/passkey/login/begin", nil)
	rp, err := svc.ResolveRelyingParty(req)
	if err != nil {
		t.Fatalf("ResolveRelyingParty() error = %v", err)
	}
	if rp.ID != "example.com" {
		t.Fatalf("relying party id = %q, want the configured override", rp.ID)
	}
	if len(rp.Origins) != 2 || rp.Origins[0] != "https://panel.example.com" || rp.Origins[1] != "https://alt.example.com" {
		t.Fatalf("origins = %#v, want the configured list without the trailing slash", rp.Origins)
	}
}

func TestResolveRelyingPartyRejectsForeignOrigin(t *testing.T) {
	svc, st := newPasskeyTestService(t)
	settings := st.GetAppSettings()
	settings.PasskeyRPID = "example.com"
	settings.PasskeyOrigins = "https://evil.test"
	if err := st.SetAppSettings(settings); err != nil {
		t.Fatalf("SetAppSettings() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "https://panel.example.com/api/auth/passkey/login/begin", nil)
	if _, err := svc.ResolveRelyingParty(req); err == nil {
		t.Fatal("ResolveRelyingParty() accepted an origin outside the relying party")
	}
}

func TestResolveRelyingPartyLocalhostKeepsPort(t *testing.T) {
	svc, _ := newPasskeyTestService(t)
	req := httptest.NewRequest(http.MethodPost, "http://localhost:8080/api/auth/passkey/login/begin", nil)

	rp, err := svc.ResolveRelyingParty(req)
	if err != nil {
		t.Fatalf("ResolveRelyingParty() error = %v", err)
	}
	if rp.ID != "localhost" {
		t.Fatalf("relying party id = %q, want localhost without the port", rp.ID)
	}
	if len(rp.Origins) != 1 || rp.Origins[0] != "http://localhost:8080" {
		t.Fatalf("origins = %#v, want the plain http origin with its port", rp.Origins)
	}
}

func TestResolveRelyingPartyAcceptsHTTPSBehindProxy(t *testing.T) {
	svc, _ := newPasskeyTestService(t)
	// The proxy terminated TLS but did not forward the scheme, so the panel
	// sees plain http and has to accept both.
	req := httptest.NewRequest(http.MethodPost, "http://panel.example.com/api/auth/passkey/login/begin", nil)

	rp, err := svc.ResolveRelyingParty(req)
	if err != nil {
		t.Fatalf("ResolveRelyingParty() error = %v", err)
	}
	if len(rp.Origins) != 2 || rp.Origins[0] != "http://panel.example.com" || rp.Origins[1] != "https://panel.example.com" {
		t.Fatalf("origins = %#v, want both schemes for a real domain", rp.Origins)
	}
}

func TestCeremonyTokensAreSingleUseAndExpire(t *testing.T) {
	svc, _ := newPasskeyTestService(t)
	svc.now = func() time.Time { return time.Unix(1_000, 0) }

	token, err := svc.storeCeremony(&webauthn.SessionData{Challenge: "abc"}, "user-1")
	if err != nil {
		t.Fatalf("storeCeremony() error = %v", err)
	}
	ceremony, ok := svc.claimCeremony(token)
	if !ok || ceremony.userID != "user-1" {
		t.Fatalf("claimCeremony() = %#v, %v, want the stored ceremony", ceremony, ok)
	}
	if _, ok := svc.claimCeremony(token); ok {
		t.Fatal("claimCeremony() accepted a token twice")
	}

	expiring, err := svc.storeCeremony(&webauthn.SessionData{Challenge: "def"}, "user-2")
	if err != nil {
		t.Fatalf("storeCeremony() error = %v", err)
	}
	svc.now = func() time.Time { return time.Unix(1_000, 0).Add(passkeyCeremonyTTL + time.Second) }
	if _, ok := svc.claimCeremony(expiring); ok {
		t.Fatal("claimCeremony() accepted an expired ceremony")
	}
}

func TestPasskeyCredentialRoundTrip(t *testing.T) {
	credential := &webauthn.Credential{
		ID:        []byte{1, 2, 3, 4},
		PublicKey: []byte{5, 6, 7},
		Flags:     webauthn.CredentialFlags{BackupEligible: true, BackupState: true},
	}
	encoded, err := EncodePasskeyCredential(credential)
	if err != nil {
		t.Fatalf("EncodePasskeyCredential() error = %v", err)
	}
	decoded, err := DecodePasskeyCredential(encoded)
	if err != nil {
		t.Fatalf("DecodePasskeyCredential() error = %v", err)
	}
	if !decoded.Flags.BackupEligible || !decoded.Flags.BackupState || len(decoded.PublicKey) != 3 {
		t.Fatalf("decoded credential = %#v, want the stored flags and key", decoded)
	}
	if got := PasskeyCredentialID(decoded); got != "AQIDBA" {
		t.Fatalf("PasskeyCredentialID() = %q, want the base64url credential id", got)
	}
	if _, err := DecodePasskeyCredential(`not json`); err == nil {
		t.Fatal("DecodePasskeyCredential() accepted garbage")
	}
}

func TestPasskeyUserAdapterUsesAccountIdentity(t *testing.T) {
	user := &passkeyUser{user: models.User{ID: "user-1", Username: "alice", Nickname: "Alice"}}
	if string(user.WebAuthnID()) != "user-1" {
		t.Fatalf("WebAuthnID() = %q, want the account id", user.WebAuthnID())
	}
	if user.WebAuthnName() != "alice" || user.WebAuthnDisplayName() != "Alice" {
		t.Fatalf("names = %q/%q, want alice/Alice", user.WebAuthnName(), user.WebAuthnDisplayName())
	}
	if len(user.WebAuthnCredentials()) != 0 {
		t.Fatal("WebAuthnCredentials() returned credentials for an account without any")
	}
}
