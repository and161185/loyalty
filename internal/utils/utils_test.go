package utils

import "testing"

func TestIsValidLuhn(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"79927398713", true}, // валидный номер
		{"1234567812345670", true},
		{"1234567812345678", false},
		{"", false},
		{"abcdef", false},
		{"49927398716", true},
		{"49927398717", false},
		{"79927398710", false},
	}

	for _, tt := range tests {
		if got := IsValidLuhn(tt.input); got != tt.valid {
			t.Errorf("IsValidLuhn(%q) = %v; want %v", tt.input, got, tt.valid)
		}
	}
}
