package proxy

import (
	"testing"

	"github.com/fonu/fonu/internal/validate"
)

func TestMergeSecurityBasicAuthKeepsHash(t *testing.T) {
	hash, err := validate.BasicAuthPasswordHash("admin", "password123")
	if err != nil {
		t.Fatal(err)
	}
	current := SecurityConfig{
		BasicAuth: &BasicAuthConfig{
			Enabled:      true,
			Username:     "admin",
			PasswordHash: hash,
		},
	}
	enabled := true
	out, err := MergeSecurity(current, SecurityInput{
		BasicAuth: &BasicAuthInput{Enabled: &enabled, Username: strPtr("admin")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.BasicAuth.PasswordHash != hash {
		t.Fatalf("expected hash preserved")
	}
}

func TestValidateSecurityWhitelistRequiresEntries(t *testing.T) {
	err := ValidateSecurity(SecurityConfig{IPWhitelistMode: true})
	if err == nil {
		t.Fatal("expected error")
	}
}

func strPtr(s string) *string { return &s }
