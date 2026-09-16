package models

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strings"
)

// DefaultAndroidPackage is the application id of the native app in this
// repository.
const DefaultAndroidPackage = "com.tunnelmanager.app"

// DefaultAndroidFingerprint is the SHA-256 certificate hash of the shared debug
// keystore android/build.sh signs with. Release builds sign with their own key
// and have to add that fingerprint in the panel.
const DefaultAndroidFingerprint = "FD:6C:C9:AA:64:FF:59:AA:3E:B9:B0:B6:06:3E:A3:78:D8:9D:8D:A5:2E:90:AA:3C:89:02:D5:E4:CF:2A:4A:81"

// AndroidPackage is the asset-links package name, falling back to this
// repository's app id.
func (s AppSettings) AndroidPackage() string {
	if pkg := strings.TrimSpace(s.PasskeyAndroidPackage); pkg != "" {
		return pkg
	}
	return DefaultAndroidPackage
}

// AndroidFingerprints returns the SHA-256 certificate fingerprints served in
// /.well-known/assetlinks.json. A blank configuration falls back to the shared
// debug keystore; unparseable input yields no fingerprints, which makes the
// endpoint report the misconfiguration instead of serving a wrong hash.
func (s AppSettings) AndroidFingerprints() []string {
	if strings.TrimSpace(s.PasskeyAndroidFingerprints) == "" {
		return []string{DefaultAndroidFingerprint}
	}
	return NormalizeAndroidFingerprints(s.PasskeyAndroidFingerprints)
}

// NormalizeAndroidFingerprints accepts whatever keytool or apksigner printed —
// colon separated or not, one per line or comma separated — and returns the
// form Digital Asset Links expects. Entries that are not 32 byte hashes are
// dropped.
func NormalizeAndroidFingerprints(raw string) []string {
	out := []string{}
	for _, candidate := range strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r' || r == ' ' || r == '\t' || r == ';'
	}) {
		if hash := normalizeFingerprint(candidate); hash != "" {
			out = append(out, hash)
		}
	}
	return out
}

// AndroidOriginPrefix starts the origin Android's Credential Manager reports
// when an app, rather than a browser, runs a ceremony against a web relying
// party.
const AndroidOriginPrefix = "android:apk-key-hash:"

// AndroidOrigins returns the origins the native app presents for this panel: the
// signing certificate's SHA-256, base64url encoded, one per configured
// fingerprint.
//
// Credential Manager does not report the site's origin — it reports the app's
// own key hash, and the library compares that as a plain string. Leaving these
// out of the relying party's origins is why the app cannot register or use a
// passkey while the browser manages both.
func (s AppSettings) AndroidOrigins() []string {
	fingerprints := s.AndroidFingerprints()
	out := make([]string, 0, len(fingerprints))
	for _, fingerprint := range fingerprints {
		raw, err := hex.DecodeString(strings.ReplaceAll(fingerprint, ":", ""))
		if err != nil || len(raw) != 32 {
			continue
		}
		out = append(out, AndroidOriginPrefix+base64.RawURLEncoding.EncodeToString(raw))
	}
	return out
}

// normalizeFingerprint uppercases a SHA-256 hash and inserts the colons.
func normalizeFingerprint(raw string) string {
	hex := strings.ToUpper(strings.NewReplacer(":", "", " ", "", "-", "").Replace(strings.TrimSpace(raw)))
	if len(hex) != 64 {
		return ""
	}
	for _, r := range hex {
		if !strings.ContainsRune("0123456789ABCDEF", r) {
			return ""
		}
	}
	parts := make([]string, 0, 32)
	for i := 0; i < len(hex); i += 2 {
		parts = append(parts, hex[i:i+2])
	}
	return strings.Join(parts, ":")
}

// Passkey is one bound WebAuthn credential. Credential holds the serialized
// go-webauthn credential: public key, signature counter, transports and the
// backup flags of the authenticator that created it.
type Passkey struct {
	ID         string
	UserID     string
	Name       string
	Credential string
	CreatedAt  int64
	LastUsedAt int64
}

// PasskeyView is the API projection of a bound passkey.
type PasskeyView struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	CreatedAt  int64  `json:"created_at"`
	LastUsedAt int64  `json:"last_used_at"`
	// BackupEligible and BackupState mark a synced passkey (iCloud Keychain,
	// Google Password Manager); the account page shows it so a user can tell a
	// device-bound credential from one that follows them.
	BackupEligible bool `json:"backup_eligible"`
	BackupState    bool `json:"backup_state"`
}

// PasskeySettingsView describes one account's passwordless state.
type PasskeySettingsView struct {
	Passkeys []PasskeyView `json:"passkeys"`
	// PasswordLoginDisabled is this account's switch; the global switch is
	// reported separately because the account page explains both.
	PasswordLoginDisabled         bool `json:"password_login_disabled"`
	PasswordLoginDisabledGlobally bool `json:"password_login_disabled_globally"`
	// RPID and Origin describe the relying party resolved for this request, so
	// the page can say which domain a passkey covers.
	RPID string `json:"rp_id"`
	// Origin is the browser origin the panel will accept assertions from.
	Origin string `json:"origin"`
	// Available reports whether the relying party could be resolved at all;
	// when false the panel cannot run a WebAuthn ceremony (for example an
	// install reached through a bare IP address).
	Available bool `json:"available"`
}

// PasskeyBeginRequest starts a registration ceremony and re-authenticates the
// account with its current password.
type PasskeyBeginRequest struct {
	Password string `json:"password"`
}

// PasskeyFinishRequest completes a registration ceremony. Credential is the
// raw PublicKeyCredential JSON produced by the browser.
type PasskeyFinishRequest struct {
	CeremonyToken string          `json:"ceremony_token"`
	Name          string          `json:"name"`
	Credential    json.RawMessage `json:"credential"`
}

// PasskeyRenameRequest renames a bound passkey.
type PasskeyRenameRequest struct {
	Name string `json:"name"`
}

// PasskeyDeleteRequest removes a bound passkey after re-authentication.
type PasskeyDeleteRequest struct {
	Password string `json:"password"`
}

// PasskeyLoginBeginRequest starts a passwordless login. Account is optional and
// only narrows the credentials the browser offers; leaving it empty asks for a
// discoverable (usernameless) credential.
type PasskeyLoginBeginRequest struct {
	Account string `json:"account"`
}

// PasskeyLoginFinishRequest completes a passwordless or second-factor login.
type PasskeyLoginFinishRequest struct {
	CeremonyToken string          `json:"ceremony_token"`
	Credential    json.RawMessage `json:"credential"`
}

// PasskeyTwoFactorBeginRequest starts the passkey second step of a pending
// password login.
type PasskeyTwoFactorBeginRequest struct {
	ChallengeToken string `json:"challenge_token"`
}

// PasskeyTwoFactorFinishRequest completes the second step of a password login
// with a passkey instead of a TOTP code.
type PasskeyTwoFactorFinishRequest struct {
	ChallengeToken string          `json:"challenge_token"`
	CeremonyToken  string          `json:"ceremony_token"`
	Credential     json.RawMessage `json:"credential"`
}

// PasswordLoginRequest toggles password sign-in for the requesting account.
type PasswordLoginRequest struct {
	Disabled bool   `json:"disabled"`
	Password string `json:"password"`
}

// AdminPasswordLoginRequest forces the switch for another account, so an
// administrator can restore password sign-in for a locked-out user.
type AdminPasswordLoginRequest struct {
	Disabled bool `json:"disabled"`
}
