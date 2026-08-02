package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

const secretVersion = "v1"

// EncryptSecret encrypts application credentials with purpose-bound AES-GCM.
func EncryptSecret(key []byte, purpose string, plaintext []byte) (string, error) {
	if purpose == "" {
		return "", errors.New("secret purpose is required")
	}
	gcm, err := newGCM(key)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}
	sealed := gcm.Seal(nil, nonce, plaintext, secretAAD(purpose))
	payload := append(nonce, sealed...)
	return secretVersion + ":" + base64.StdEncoding.EncodeToString(payload), nil
}

// DecryptSecret decrypts a credential encrypted by EncryptSecret.
func DecryptSecret(key []byte, purpose, encoded string) ([]byte, error) {
	if purpose == "" {
		return nil, errors.New("secret purpose is required")
	}
	version, payloadText, ok := strings.Cut(encoded, ":")
	if !ok || version != secretVersion || payloadText == "" {
		return nil, errors.New("unsupported encrypted secret format")
	}
	payload, err := base64.StdEncoding.DecodeString(payloadText)
	if err != nil {
		return nil, fmt.Errorf("decode encrypted secret: %w", err)
	}
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	if len(payload) < gcm.NonceSize()+gcm.Overhead() {
		return nil, errors.New("encrypted secret is too short")
	}
	nonce, ciphertext := payload[:gcm.NonceSize()], payload[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, secretAAD(purpose))
	if err != nil {
		return nil, errors.New("decrypt secret")
	}
	return plaintext, nil
}

func secretAAD(purpose string) []byte {
	return []byte("tunnel-manager:" + purpose + ":" + secretVersion)
}
