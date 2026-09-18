package acme

import (
	"testing"

	"github.com/fonu/fonu/internal/ddns"
)

func TestValidateDNSCredentialsVolcengine(t *testing.T) {
	if err := validateDNSCredentials("volcengine", ddns.Credentials{Token: "ak", Secret: "sk"}); err != nil {
		t.Fatalf("expected volcengine credentials to pass: %v", err)
	}
}

func TestNewVolcengineDNSProvider(t *testing.T) {
	p, err := newVolcengineDNSProvider(ddns.Credentials{Token: "ak", Secret: "sk"})
	if err != nil {
		t.Fatal(err)
	}
	if p == nil {
		t.Fatal("expected provider")
	}
}
