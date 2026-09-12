package certificate

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"
)

func TestReadPEMFile(t *testing.T) {
	certPEM, keyPEM := generateTestCert("example.com")
	certPath := t.TempDir() + "/cert.pem"
	keyPath := t.TempDir() + "/key.pem"
	if err := os.WriteFile(certPath, certPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		t.Fatal(err)
	}

	readCert, err := ReadPEMFile(certPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(readCert) != string(certPEM) {
		t.Fatal("cert content mismatch")
	}

	got, err := ResolvePEMSource("", certPath)
	if err != nil || string(got) != string(certPEM) {
		t.Fatalf("ResolvePEMSource path: %v", err)
	}
}

func TestInspectImport(t *testing.T) {
	certPEM, keyPEM := generateTestCertWithSANs([]string{"example.com", "*.example.com", "nas.example.com"})
	info, err := InspectImport(certPEM, keyPEM)
	if err != nil {
		t.Fatal(err)
	}
	if info.Primary != "example.com" {
		t.Fatalf("primary: %s", info.Primary)
	}
	if !info.Wildcard {
		t.Fatal("expected wildcard")
	}
	if len(info.Domains) != 3 {
		t.Fatalf("domains: %v", info.Domains)
	}
}

func TestValidatePEM(t *testing.T) {
	certPEM, keyPEM := generateTestCert("example.com")

	expiresAt, err := ValidatePEM(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("ValidatePEM: %v", err)
	}
	if expiresAt.Before(time.Now()) {
		t.Fatalf("expected future expiry, got %v", expiresAt)
	}

	_, err = ValidatePEM(certPEM, []byte("not a key"))
	if err == nil {
		t.Fatal("expected error for invalid key")
	}

	_, wrongKeyPEM := generateTestCert("other.example.com")
	_, err = ValidatePEM(certPEM, wrongKeyPEM)
	if err == nil {
		t.Fatal("expected mismatch error")
	}
}

func generateTestCert(domain string) ([]byte, []byte) {
	return generateTestCertWithSANs([]string{domain})
}

func generateTestCertWithSANs(domains []string) ([]byte, []byte) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(err)
	}
	cn := domains[0]
	if strings.HasPrefix(cn, "*.") {
		cn = strings.TrimPrefix(cn, "*.")
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: cn},
		DNSNames:     domains,
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		panic(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	return certPEM, keyPEM
}
