package auth

import (
	"encoding/base32"
	"encoding/base64"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestParseEncryptionKey(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString(make([]byte, 32))
	key, err := ParseEncryptionKey(encoded)
	if err != nil {
		t.Fatalf("ParseEncryptionKey() error = %v", err)
	}
	if len(key) != 32 {
		t.Fatalf("ParseEncryptionKey() length = %d, want 32", len(key))
	}

	for _, invalid := range []string{
		"not base64",
		base64.StdEncoding.EncodeToString(make([]byte, 31)),
		base64.StdEncoding.EncodeToString(make([]byte, 33)),
		base64.RawStdEncoding.EncodeToString(make([]byte, 32)),
		base64.StdEncoding.EncodeToString(make([]byte, 32)) + "\n",
	} {
		if _, err := ParseEncryptionKey(invalid); err == nil {
			t.Errorf("ParseEncryptionKey(%q) succeeded", invalid)
		}
	}
}

func TestEncryptDecryptTOTPSecret(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	secret := []byte("12345678901234567890")

	encrypted, err := EncryptTOTPSecret(key, secret)
	if err != nil {
		t.Fatalf("EncryptTOTPSecret() error = %v", err)
	}
	if !strings.HasPrefix(encrypted, "v1:") {
		t.Fatalf("EncryptTOTPSecret() = %q, want v1 prefix", encrypted)
	}
	payload, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(encrypted, "v1:"))
	if err != nil {
		t.Fatalf("encrypted payload is not standard base64: %v", err)
	}
	if len(payload) != 12+len(secret)+16 {
		t.Fatalf("encrypted payload length = %d, want %d", len(payload), 12+len(secret)+16)
	}

	decrypted, err := DecryptTOTPSecret(key, encrypted)
	if err != nil {
		t.Fatalf("DecryptTOTPSecret() error = %v", err)
	}
	if string(decrypted) != string(secret) {
		t.Fatalf("DecryptTOTPSecret() = %q, want %q", decrypted, secret)
	}

	second, err := EncryptTOTPSecret(key, secret)
	if err != nil {
		t.Fatalf("second EncryptTOTPSecret() error = %v", err)
	}
	if second == encrypted {
		t.Fatal("EncryptTOTPSecret() reused a nonce")
	}

	payload[len(payload)-1] ^= 1
	tampered := "v1:" + base64.StdEncoding.EncodeToString(payload)
	if _, err := DecryptTOTPSecret(key, tampered); err == nil {
		t.Fatal("DecryptTOTPSecret() accepted tampered ciphertext")
	}
	if _, err := DecryptTOTPSecret(key, "v2:"+strings.TrimPrefix(encrypted, "v1:")); err == nil {
		t.Fatal("DecryptTOTPSecret() accepted unsupported version")
	}
	if _, err := DecryptTOTPSecret([]byte("short"), encrypted); err == nil {
		t.Fatal("DecryptTOTPSecret() accepted invalid key length")
	}
}

func TestGenerateTOTPSecret(t *testing.T) {
	plain, encoded, err := GenerateTOTPSecret()
	if err != nil {
		t.Fatalf("GenerateTOTPSecret() error = %v", err)
	}
	if len(plain) != 20 {
		t.Fatalf("secret length = %d, want 20", len(plain))
	}
	if len(encoded) != 32 || strings.Contains(encoded, "=") || encoded != strings.ToUpper(encoded) {
		t.Fatalf("encoded secret = %q, want 32 uppercase unpadded Base32 characters", encoded)
	}
	decoded, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(encoded)
	if err != nil {
		t.Fatalf("decode generated secret: %v", err)
	}
	if string(decoded) != string(plain) {
		t.Fatal("encoded secret does not represent plaintext secret")
	}
}

func TestGenerateTOTPCodeRFC6238(t *testing.T) {
	secret := []byte("12345678901234567890")
	tests := []struct {
		unix int64
		want string
	}{
		{59, "287082"},
		{1111111109, "081804"},
		{1111111111, "050471"},
		{1234567890, "005924"},
		{2000000000, "279037"},
		{20000000000, "353130"},
	}
	for _, tt := range tests {
		if got := GenerateTOTPCode(secret, tt.unix/30); got != tt.want {
			t.Errorf("GenerateTOTPCode(step %d) = %q, want %q", tt.unix/30, got, tt.want)
		}
	}
}

func TestMatchTOTPCodeWindowAndStep(t *testing.T) {
	secret := []byte("12345678901234567890")
	now := time.Unix(1_700_000_000, 0)
	current := now.Unix() / 30

	for _, step := range []int64{current - 1, current, current + 1} {
		matched, ok := MatchTOTPCode(secret, GenerateTOTPCode(secret, step), now)
		if !ok || matched != step {
			t.Errorf("MatchTOTPCode(step %d) = (%d, %v)", step, matched, ok)
		}
	}
	if _, ok := MatchTOTPCode(secret, GenerateTOTPCode(secret, current+2), now); ok {
		t.Fatal("MatchTOTPCode() accepted code outside window")
	}
	for _, invalid := range []string{"12345", "1234567", "12a456"} {
		if _, ok := MatchTOTPCode(secret, invalid, now); ok {
			t.Errorf("MatchTOTPCode() accepted %q", invalid)
		}
	}
}

func TestBuildOTPAuthURI(t *testing.T) {
	secret := "JBSWY3DPEHPK3PXPJBSWY3DPEHPK3PXP"
	raw, err := BuildOTPAuthURI("admin/user +?", secret)
	if err != nil {
		t.Fatalf("BuildOTPAuthURI() error = %v", err)
	}
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse URI: %v", err)
	}
	if u.Scheme != "otpauth" || u.Host != "totp" {
		t.Fatalf("URI scheme/host = %s/%s", u.Scheme, u.Host)
	}
	if u.Path != "/Tunnel Manager:admin/user +?" {
		t.Fatalf("URI path = %q", u.Path)
	}
	if strings.Contains(raw, "admin/user") || strings.Contains(raw, "Tunnel Manager") {
		t.Fatalf("URI path is not escaped: %q", raw)
	}
	query := u.Query()
	for key, want := range map[string]string{
		"secret": secret, "issuer": "Tunnel Manager", "algorithm": "SHA1", "digits": "6", "period": "30",
	} {
		if got := query.Get(key); got != want {
			t.Errorf("query %s = %q, want %q", key, got, want)
		}
	}
}
