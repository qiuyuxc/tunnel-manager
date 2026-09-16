package services

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"

	"tunnel-manager/models"
	"tunnel-manager/store"
)

// TestFinishLoginFollowsTheCeremonyThatStarted covers the sign-in handler, which
// calls FinishLogin without naming an account — the ceremony already knows. A
// ceremony started by BeginLogin carries the account, and validating it as
// usernameless makes the library refuse the assertion outright with "Session was
// not initiated as a client-side discoverable login". The Android app signs in
// this way, so it could not get in at all.
func TestFinishLoginFollowsTheCeremonyThatStarted(t *testing.T) {
	svc, st := newPasskeyTestService(t)
	if err := st.CreateUser(models.User{Username: "xik", Email: "xik@example.com", Role: models.RoleUser}); err != nil {
		t.Fatal(err)
	}
	account, ok := st.GetUserByUsername("xik")
	if !ok {
		t.Fatal("account not found")
	}
	encoded, err := EncodePasskeyCredential(&webauthn.Credential{ID: []byte{1, 2, 3, 4}, PublicKey: []byte{5, 6, 7}})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.AddPasskey(models.Passkey{ID: "pk-1", UserID: account.ID, Name: "test", Credential: encoded}); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "https://panel.example.com/api/auth/passkey/login/finish", nil)
	options, token, _, err := svc.BeginLogin(req, account.ID)
	if err != nil {
		t.Fatalf("BeginLogin() error = %v", err)
	}

	// Shaped so the assertion clears parsing and the relying-party and challenge
	// checks, which is what makes the validator underneath it observable. The
	// user handle is what the usernameless path uses to find the account, so it
	// is supplied too, for the same reason: an earlier failure would mask the
	// error being guarded against.
	rpIDHash := sha256.Sum256([]byte("panel.example.com"))
	authenticatorData := append(rpIDHash[:], 0x01, 0, 0, 0, 0) // flags=UP, signCount=0
	clientData := fmt.Sprintf(`{"type":"webauthn.get","challenge":"%s","origin":"https://panel.example.com"}`,
		options.Response.Challenge.String())
	assertion := fmt.Sprintf(
		`{"id":"AQIDBA","rawId":"AQIDBA","type":"public-key","response":{"clientDataJSON":"%s","authenticatorData":"%s","signature":"AAAA","userHandle":"%s"}}`,
		base64.RawURLEncoding.EncodeToString([]byte(clientData)),
		base64.RawURLEncoding.EncodeToString(authenticatorData),
		base64.RawURLEncoding.EncodeToString([]byte(account.ID)))

	_, _, _, err = svc.FinishLogin(req, token, "", []byte(assertion))
	if err == nil {
		t.Fatal("FinishLogin() accepted a fabricated assertion")
	}
	if strings.Contains(err.Error(), "discoverable") {
		t.Fatalf("a named-account ceremony was validated as usernameless: %v", err)
	}
}

// TestValidatePasskeySettingsCatchesMismatchedPair covers the shape an
// administrator lands on after renaming the panel: the relying party moves to
// the new domain, the stored origins keep naming the old one, and every
// ceremony then fails with a mismatch. The pair has to be refused up front.
func TestValidatePasskeySettingsCatchesMismatchedPair(t *testing.T) {
	cases := []struct {
		name    string
		rpID    string
		origins string
		wantErr string
	}{
		{"matching pair", "panel.example.com", "https://panel.example.com", ""},
		{"origin below the relying party", "example.com", "https://panel.example.com", ""},
		{"several origins", "example.com", "https://a.example.com, https://b.example.com", ""},
		{"empty relying party defers to the request", "", "https://panel.example.com", ""},
		{"empty origins defer to the request", "panel.example.com", "", ""},
		{"relying party renamed, origins left behind", "m.veits.bond", "https://cf.kukie.cn", "不匹配"},
		{"origin is a parent of the relying party", "panel.example.com", "https://example.com", "不匹配"},
		{"origin carries a path", "panel.example.com", "https://panel.example.com/login", "不能带路径"},
		{"full-width comma separates", "example.com", "https://a.example.com，https://b.example.com", ""},
		{"full-width space separates", "example.com", "https://a.example.com\u3000https://b.example.com", ""},
		// Two unrelated domains cannot share one relying party, however they are
		// separated; the list has to be refused rather than half-applied.
		{"full-width comma hides a foreign origin", "m.veits.bond", "https://cf.kukie.cn，https://m.veits.bond", "不匹配"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidatePasskeySettings(tc.rpID, tc.origins)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("ValidatePasskeySettings(%q, %q) = %v; want nil", tc.rpID, tc.origins, err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("ValidatePasskeySettings(%q, %q) = %v; want an error containing %q", tc.rpID, tc.origins, err, tc.wantErr)
			}
		})
	}
}

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
