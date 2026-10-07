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
	url, err = DirectoryURL(CABuypassTest)
	if err != nil || url == "" {
		t.Fatalf("buypass test: %v", err)
	}
	url, err = DirectoryURL(CAActalis)
	if err != nil || url == "" {
		t.Fatalf("actalis: %v", err)
	}
	if url, err := DirectoryURL(CACustom); err != nil || url != "" {
		t.Fatalf("custom should resolve at apply time: url=%q err=%v", url, err)
	}
	if err := ValidateCA("unknown"); err == nil {
		t.Fatal("expected error for unknown ca")
	}
	if err := ValidateCA(CACustom); err != nil {
		t.Fatalf("custom ca should be valid: %v", err)
	}
}

func TestValidateCADomainsActalis(t *testing.T) {
	if err := ValidateCADomains(CAActalis, []string{"*.example.com"}); err == nil {
		t.Fatal("expected wildcard error")
	}
	if err := ValidateCADomains(CAActalis, []string{"a.com", "b.com", "c.com", "d.com", "e.com", "f.com"}); err == nil {
		t.Fatal("expected domain limit error")
	}
	if err := ValidateCADomains(CAActalis, []string{"example.com"}); err != nil {
		t.Fatalf("actalis single domain should be allowed: %v", err)
	}
	if err := ValidateCADomains(CABuypassTest, []string{"*.example.com"}); err == nil {
		t.Fatal("expected wildcard error for buypass test")
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
