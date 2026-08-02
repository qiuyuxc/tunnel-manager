package auth

import (
	"bytes"
	"testing"
)

func TestEncryptSecretRoundTripAndPurposeBinding(t *testing.T) {
	key := bytes.Repeat([]byte{0x42}, 32)
	encrypted, err := EncryptSecret(key, "oauth-access", []byte("sensitive-token"))
	if err != nil {
		t.Fatalf("EncryptSecret() error = %v", err)
	}
	if encrypted == "sensitive-token" {
		t.Fatal("EncryptSecret() returned plaintext")
	}
	plain, err := DecryptSecret(key, "oauth-access", encrypted)
	if err != nil {
		t.Fatalf("DecryptSecret() error = %v", err)
	}
	if string(plain) != "sensitive-token" {
		t.Fatalf("DecryptSecret() = %q", plain)
	}
	if _, err := DecryptSecret(key, "oauth-refresh", encrypted); err == nil {
		t.Fatal("DecryptSecret() accepted a credential with a different purpose")
	}
}
