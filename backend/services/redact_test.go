package services

import (
	"errors"
	"testing"
)

func TestRedactSecretsRemovesBotToken(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "token inside a transport error",
			input: `Get "https://api.telegram.org/bot123456789:AAF74aaAN7agi683qbFRT9IviDr4DagJnmk/getMe": dial tcp: no such host`,
			want:  `Get "https://api.telegram.org/bot<redacted>/getMe": dial tcp: no such host`,
		},
		{
			name:  "custom api endpoint",
			input: `Post "https://tele.example.com/bot9876543210:AAH-abc_123/sendMessage": EOF`,
			want:  `Post "https://tele.example.com/bot<redacted>/sendMessage": EOF`,
		},
		{
			name:  "no token",
			input: "getMe failed: Unauthorized",
			want:  "getMe failed: Unauthorized",
		},
		{
			name:  "short id is not a token",
			input: "bot123:abc",
			want:  "bot123:abc",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := RedactSecrets(tc.input); got != tc.want {
				t.Fatalf("RedactSecrets() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRedactErrorPassthroughNil(t *testing.T) {
	if RedactError(nil) != nil {
		t.Fatal("RedactError(nil) should stay nil")
	}
	if got := RedactError(errors.New("bot11111111:SECRETTOKEN/x")).Error(); got != "bot<redacted>/x" {
		t.Fatalf("RedactError() = %q", got)
	}
}
