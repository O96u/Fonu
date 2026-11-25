package nginx

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFindCertificateForHostsFallsBackToDiskWhenDBPathsMissing(t *testing.T) {
	dir := t.TempDir()
	certDir := filepath.Join(dir, "certs", "wildcard.example.com")
	if err := os.MkdirAll(certDir, 0o755); err != nil {
		t.Fatal(err)
	}
	certPath := filepath.Join(certDir, "fullchain.pem")
	writeTestCertificate(t, certDir, []string{"*.example.com", "example.com"}, time.Now().Add(time.Hour))

	cert := findCertificateForHosts(filepath.Join(dir, "certs"), []string{"app.example.com"}, []CertSource{{
		Domains:  []string{"example.com", "*.example.com"},
		CertPath: "/data/certs/wildcard.example.com/missing.pem",
		KeyPath:  "/data/certs/wildcard.example.com/missing-key.pem",
	}})
	if cert == nil || !certUsable(cert) {
		t.Fatal("expected disk certificate fallback")
	}
	if filepath.Clean(cert.CertPath) != filepath.Clean(certPath) {
		t.Fatalf("unexpected cert path: %s", cert.CertPath)
	}
}
