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

func TestNewDNSHEDNSProvider(t *testing.T) {
	_, err := newDNSHEDNSProvider(ddns.Credentials{Provider: "dnshe", Token: "key"})
	if err == nil {
		t.Fatal("expected validation error without secret")
	}
	p, err := newDNSHEDNSProvider(ddns.Credentials{Provider: "dnshe", Token: "key", Secret: "sec"})
	if err != nil || p == nil {
		t.Fatalf("provider: %v err=%v", p, err)
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
