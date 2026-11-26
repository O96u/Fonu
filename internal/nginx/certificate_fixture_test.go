package nginx

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeTestCertificate(t *testing.T, dir string, hosts []string, expires time.Time) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil { t.Fatal(err) }
	leaf := &x509.Certificate{SerialNumber: big.NewInt(1), DNSNames: hosts,
		NotBefore: time.Now().Add(-24*time.Hour), NotAfter: expires,
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	der, err := x509.CreateCertificate(rand.Reader, leaf, leaf, &key.PublicKey, key)
	if err != nil { t.Fatal(err) }
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil { t.Fatal(err) }
	if err := os.MkdirAll(dir, 0755); err != nil { t.Fatal(err) }
	if err := os.WriteFile(filepath.Join(dir, "fullchain.pem"), pem.EncodeToMemory(&pem.Block{Type:"CERTIFICATE", Bytes:der}), 0644); err != nil { t.Fatal(err) }
	if err := os.WriteFile(filepath.Join(dir, "privatekey.pem"), pem.EncodeToMemory(&pem.Block{Type:"EC PRIVATE KEY", Bytes:keyDER}), 0600); err != nil { t.Fatal(err) }
}

func TestCertificateUsesActualSANAndValidity(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "wildcard.example.com")
	writeTestCertificate(t, first, []string{"*.example.com"}, time.Now().Add(time.Hour))
	if !HasCertificateForHosts(dir, []string{"app.example.com"}, nil) { t.Fatal("valid wildcard not found") }
	for _, host := range []string{"example.com","a.b.example.com","other.test"} {
		if HasCertificateForHosts(dir, []string{host}, nil) { t.Fatalf("unexpected coverage for %s", host) }
	}
	writeTestCertificate(t, first, []string{"*.example.com"}, time.Now().Add(-time.Hour))
	if HasCertificateForHosts(dir, []string{"app.example.com"}, nil) { t.Fatal("expired certificate accepted") }
	writeTestCertificate(t, first, []string{"different.test"}, time.Now().Add(time.Hour))
	if HasCertificateForHosts(dir, []string{"app.example.com"}, nil) { t.Fatal("directory name incorrectly trusted") }
}

func TestCertificateRejectsMismatchedPrivateKey(t *testing.T) {
	dir := t.TempDir()
	first, second := filepath.Join(dir,"first"), filepath.Join(dir,"second")
	writeTestCertificate(t, first, []string{"app.example.com"}, time.Now().Add(time.Hour))
	writeTestCertificate(t, second, []string{"other.example.com"}, time.Now().Add(time.Hour))
	key, err := os.ReadFile(filepath.Join(second,"privatekey.pem"))
	if err != nil { t.Fatal(err) }
	if err := os.WriteFile(filepath.Join(first,"privatekey.pem"),key,0600); err != nil { t.Fatal(err) }
	if HasCertificateForHosts(dir, []string{"app.example.com"},nil) { t.Fatal("mismatched private key accepted") }
}
