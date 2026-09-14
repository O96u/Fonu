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

func TestChinaBypassCIDRsIncludesDefaults(t *testing.T) {
	got := ChinaBypassCIDRs(nil)
	for _, want := range DefaultPrivateCIDRs {
		if !contains(got, want) {
			t.Fatalf("missing default %s in %v", want, got)
		}
	}
}

func TestChinaBypassCIDRsMergesExtra(t *testing.T) {
	got := ChinaBypassCIDRs([]string{"203.0.113.0/24", "192.168.0.0/16"})
	if !contains(got, "203.0.113.0/24") {
		t.Fatalf("missing extra CIDR: %v", got)
	}
	if count(got, "192.168.0.0/16") != 1 {
		t.Fatalf("expected single 192.168.0.0/16 entry: %v", got)
	}
}

func contains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

func count(ss []string, s string) int {
	n := 0
	for _, v := range ss {
		if v == s {
			n++
		}
	}
	return n
}

func strPtr(s string) *string { return &s }
