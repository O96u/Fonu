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
	if err := ValidateCA("unknown"); err == nil {
		t.Fatal("expected error for unknown ca")
	}
}
