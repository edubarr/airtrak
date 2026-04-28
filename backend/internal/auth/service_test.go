package auth

import (
	"testing"
)

func TestNormalizeEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
		want  string
	}{
		{name: "lowercase", email: "USER@example.COM", want: "user@example.com"},
		{name: "trims", email: " user@example.com ", want: "user@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeEmail(tt.email)
			if err != nil {
				t.Fatalf("normalizeEmail() error = %v", err)
			}

			if got != tt.want {
				t.Fatalf("normalizeEmail() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizeEmailRejectsInvalidEmail(t *testing.T) {
	invalid := []string{"", "not-email", "Name <user@example.com>", "user @example.com"}
	for _, email := range invalid {
		t.Run(email, func(t *testing.T) {
			if _, err := normalizeEmail(email); err == nil {
				t.Fatalf("normalizeEmail(%q) error = nil, want error", email)
			}
		})
	}
}

func TestSupportedLocales(t *testing.T) {
	if !isSupportedLocale("pt-BR") {
		t.Fatal("pt-BR should be supported")
	}

	if !isSupportedLocale("en") {
		t.Fatal("en should be supported")
	}

	if isSupportedLocale("es") {
		t.Fatal("es should not be supported")
	}
}
