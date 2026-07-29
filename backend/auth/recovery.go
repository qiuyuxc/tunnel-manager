package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

const (
	recoveryCodeBytes  = 15
	recoveryCodeLength = 24
)

func GenerateRecoveryCodes(count int) ([]string, []string, error) {
	if count < 0 {
		return nil, nil, errors.New("recovery code count must not be negative")
	}
	displayCodes := make([]string, 0, count)
	hashes := make([]string, 0, count)
	seen := make(map[string]struct{}, count)
	for len(displayCodes) < count {
		random := make([]byte, recoveryCodeBytes)
		if _, err := rand.Read(random); err != nil {
			return nil, nil, fmt.Errorf("generate recovery code: %w", err)
		}
		normalized := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(random)
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		displayCodes = append(displayCodes, formatRecoveryCode(normalized))
		hashes = append(hashes, HashRecoveryCode(normalized))
	}
	return displayCodes, hashes, nil
}

func NormalizeRecoveryCode(code string) (string, error) {
	normalized := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(code), "-", ""))
	if len(normalized) != recoveryCodeLength {
		return "", errors.New("recovery code must contain 24 Base32 characters")
	}
	decoded, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(normalized)
	if err != nil || len(decoded) != recoveryCodeBytes {
		return "", errors.New("recovery code must contain only Base32 characters")
	}
	return normalized, nil
}

func HashRecoveryCode(normalized string) string {
	hash := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(hash[:])
}

func formatRecoveryCode(code string) string {
	groups := make([]string, 0, recoveryCodeLength/4)
	for i := 0; i < len(code); i += 4 {
		groups = append(groups, code[i:i+4])
	}
	return strings.Join(groups, "-")
}
