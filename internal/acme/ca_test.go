package acme

import "testing"

func TestDirectoryURL(t *testing.T) {
	url, err := DirectoryURL(CALetsEncrypt)
	if err != nil || url == "" {
		t.Fatalf("production: %v", err)
	}
	url, err = DirectoryURL(CALetsEncryptStaging)
	if err != nil || url == "" {
		t.Fatalf("staging: %v", err)
	}
	url, err = DirectoryURL(CAZeroSSL)
	if err != nil || url == "" {
		t.Fatalf("zerossl: %v", err)
	}
	url, err = DirectoryURL(CABuypass)
	if err != nil || url == "" {
		t.Fatalf("buypass: %v", err)
	}
	if err := ValidateCA("unknown"); err == nil {
		t.Fatal("expected error for unknown ca")
	}
}

func TestValidateCADomainsBuypass(t *testing.T) {
	if err := ValidateCADomains(CABuypass, []string{"*.example.com"}); err == nil {
		t.Fatal("expected wildcard error")
	}
	if err := ValidateCADomains(CABuypass, []string{"a.com", "b.com", "c.com", "d.com", "e.com", "f.com"}); err == nil {
		t.Fatal("expected domain limit error")
	}
	if err := ValidateCADomains(CALetsEncrypt, []string{"*.example.com"}); err != nil {
		t.Fatalf("letsencrypt wildcard should be allowed: %v", err)
	}
}
