package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	totpPeriod      = int64(30)
	totpSecretBytes = 20
	totpVersion     = "v1"
	totpIssuer      = "Tunnel Manager"
)

var totpAAD = []byte("tunnel-manager:totp-secret:v1")

func ParseEncryptionKey(encoded string) ([]byte, error) {
	key, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decode encryption key: %w", err)
	}
	if len(key) != 32 || base64.StdEncoding.EncodeToString(key) != encoded {
		return nil, errors.New("encryption key must be standard base64 encoding of exactly 32 bytes")
	}
	return key, nil
}

func GenerateTOTPSecret() ([]byte, string, error) {
	plain := make([]byte, totpSecretBytes)
	if _, err := rand.Read(plain); err != nil {
		return nil, "", fmt.Errorf("generate TOTP secret: %w", err)
	}
	return plain, base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(plain), nil
}

func EncryptTOTPSecret(key, plaintext []byte) (string, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}
	sealed := gcm.Seal(nil, nonce, plaintext, totpAAD)
	payload := append(nonce, sealed...)
	return totpVersion + ":" + base64.StdEncoding.EncodeToString(payload), nil
}

func DecryptTOTPSecret(key []byte, encoded string) ([]byte, error) {
	version, payloadText, ok := strings.Cut(encoded, ":")
	if !ok || version != totpVersion || payloadText == "" {
		return nil, errors.New("unsupported encrypted TOTP secret format")
	}
	payload, err := base64.StdEncoding.DecodeString(payloadText)
	if err != nil {
		return nil, fmt.Errorf("decode encrypted TOTP secret: %w", err)
	}

	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	if len(payload) < gcm.NonceSize()+gcm.Overhead() {
		return nil, errors.New("encrypted TOTP secret is too short")
	}
	nonce, ciphertext := payload[:gcm.NonceSize()], payload[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, totpAAD)
	if err != nil {
		return nil, errors.New("decrypt TOTP secret")
	}
	return plaintext, nil
}

func BuildOTPAuthURI(username, base32Secret string) (string, error) {
	if username == "" {
		return "", errors.New("username is required")
	}
	secret := strings.ToUpper(base32Secret)
	decoded, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil || len(decoded) != totpSecretBytes || len(secret) != 32 || strings.Contains(base32Secret, "=") {
		return "", errors.New("TOTP secret must be 20 bytes encoded as unpadded Base32")
	}

	u := &url.URL{
		Scheme:  "otpauth",
		Host:    "totp",
		Path:    "/" + totpIssuer + ":" + username,
		RawPath: "/" + url.PathEscape(totpIssuer) + ":" + url.PathEscape(username),
	}
	query := url.Values{}
	query.Set("secret", secret)
	query.Set("issuer", totpIssuer)
	query.Set("algorithm", "SHA1")
	query.Set("digits", "6")
	query.Set("period", strconv.FormatInt(totpPeriod, 10))
	u.RawQuery = query.Encode()
	return u.String(), nil
}

func GenerateTOTPCode(secret []byte, step int64) string {
	var counter [8]byte
	binary.BigEndian.PutUint64(counter[:], uint64(step))
	mac := hmac.New(sha1.New, secret)
	_, _ = mac.Write(counter[:])
	digest := mac.Sum(nil)
	offset := digest[len(digest)-1] & 0x0f
	value := binary.BigEndian.Uint32(digest[offset:offset+4]) & 0x7fffffff
	return fmt.Sprintf("%06d", value%1_000_000)
}

func MatchTOTPCode(secret []byte, code string, now time.Time) (int64, bool) {
	if len(code) != 6 {
		return 0, false
	}
	for i := range code {
		if code[i] < '0' || code[i] > '9' {
			return 0, false
		}
	}

	current := now.Unix() / totpPeriod
	var matched int64
	ok := false
	for step := current - 1; step <= current+1; step++ {
		expected := GenerateTOTPCode(secret, step)
		if subtle.ConstantTimeCompare([]byte(expected), []byte(code)) == 1 {
			matched = step
			ok = true
		}
	}
	return matched, ok
}

func newGCM(key []byte) (cipher.AEAD, error) {
	if len(key) != 32 {
		return nil, errors.New("AES-256 key must be exactly 32 bytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}
	return gcm, nil
}
