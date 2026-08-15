package services

import "testing"

func TestNormalizeBindingMode(t *testing.T) {
	tests := []struct {
		name, input, want string
		wantErr           bool
	}{
		{"legacy default", "", BindingModePreferred, false},
		{"simple", " SIMPLE ", BindingModeSimple, false},
		{"preferred", "preferred", BindingModePreferred, false},
		{"invalid", "direct", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeBindingMode(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error=%v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
			}
		})
	}
}
