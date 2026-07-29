package auth

import (
	"encoding/base32"
	"regexp"
	"testing"
)

func TestGenerateRecoveryCodes(t *testing.T) {
	const count = 10
	display, hashes, err := GenerateRecoveryCodes(count)
	if err != nil {
		t.Fatalf("GenerateRecoveryCodes() error = %v", err)
	}
	if len(display) != count || len(hashes) != count {
		t.Fatalf("counts = (%d, %d), want (%d, %d)", len(display), len(hashes), count, count)
	}

	pattern := regexp.MustCompile(`^[A-Z2-7]{4}(?:-[A-Z2-7]{4}){5}$`)
	seen := make(map[string]bool, count)
	for i, code := range display {
		if !pattern.MatchString(code) {
			t.Errorf("display code %q has invalid format", code)
		}
		normalized, err := NormalizeRecoveryCode(code)
		if err != nil {
			t.Errorf("NormalizeRecoveryCode(%q) error = %v", code, err)
			continue
		}
		if seen[normalized] {
			t.Errorf("duplicate recovery code %q", code)
		}
		seen[normalized] = true
		if hashes[i] != HashRecoveryCode(normalized) {
			t.Errorf("hash %d does not match code", i)
		}
		if len(hashes[i]) != 64 {
			t.Errorf("hash length = %d, want 64", len(hashes[i]))
		}
	}
}

func TestGenerateRecoveryCodesCount(t *testing.T) {
	display, hashes, err := GenerateRecoveryCodes(0)
	if err != nil {
		t.Fatalf("GenerateRecoveryCodes(0) error = %v", err)
	}
	if len(display) != 0 || len(hashes) != 0 {
		t.Fatalf("GenerateRecoveryCodes(0) returned %d codes and %d hashes", len(display), len(hashes))
	}
	if _, _, err := GenerateRecoveryCodes(-1); err == nil {
		t.Fatal("GenerateRecoveryCodes(-1) succeeded")
	}
}

func TestNormalizeRecoveryCode(t *testing.T) {
	raw := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString([]byte("123456789012345"))
	formatted := formatRecoveryCode(raw)
	got, err := NormalizeRecoveryCode("  " + formatted + "  ")
	if err != nil {
		t.Fatalf("NormalizeRecoveryCode() error = %v", err)
	}
	if got != raw {
		t.Fatalf("NormalizeRecoveryCode() = %q, want %q", got, raw)
	}
	got, err = NormalizeRecoveryCode("  " + stringLower(formatted) + "  ")
	if err != nil || got != raw {
		t.Fatalf("NormalizeRecoveryCode(lowercase) = (%q, %v), want (%q, nil)", got, err, raw)
	}

	for _, invalid := range []string{
		"SHORT",
		"AAAA-AAAA-AAAA-AAAA-AAAA-AAA1",
		"AAAA-AAAA-AAAA-AAAA-AAAA-AAA=",
	} {
		if _, err := NormalizeRecoveryCode(invalid); err == nil {
			t.Errorf("NormalizeRecoveryCode(%q) succeeded", invalid)
		}
	}
}

func TestHashRecoveryCode(t *testing.T) {
	const normalized = "GEZDGNBVGY3TQOJQGEZDGNBV"
	const want = "b189b47d3a904ea55fa51fa1a50de4d72ef8f342f1b767fc190996793e71e9ba"
	if got := HashRecoveryCode(normalized); got != want {
		t.Fatalf("HashRecoveryCode() = %q, want %q", got, want)
	}
}

func stringLower(value string) string {
	result := []byte(value)
	for i, char := range result {
		if char >= 'A' && char <= 'Z' {
			result[i] = char + ('a' - 'A')
		}
	}
	return string(result)
}
