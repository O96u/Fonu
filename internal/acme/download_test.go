package acme

import (
	"path/filepath"
	"testing"
)

func TestValidateCertFilePath(t *testing.T) {
	certsDir := filepath.Clean("/data/certs")
	if err := validateCertFilePath(certsDir, "/data/certs/example.com/fullchain.pem"); err != nil {
		t.Fatalf("expected valid path: %v", err)
	}
	if err := validateCertFilePath(certsDir, "/etc/passwd"); err == nil {
		t.Fatal("expected invalid path")
	}
}

func TestZipCertBundle(t *testing.T) {
	data, err := zipCertBundle("example.com", []byte("cert"), []byte("key"))
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Fatal("empty zip")
	}
}
