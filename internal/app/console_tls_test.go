package app

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fonu/fonu/internal/config"
)

func TestConsoleTLS(t *testing.T) {
	if c,err:=consoleTLSConfig(config.Config{});err!=nil||c!=nil {t.Fatalf("default HTTP: %v",err)}
	if _,err:=consoleTLSConfig(config.Config{TLSCertFile:"missing"});err==nil {t.Fatal("accepted incomplete TLS configuration")}
	if _,err:=consoleTLSConfig(config.Config{TLSCertFile:"missing",TLSKeyFile:"missing"});err==nil {t.Fatal("accepted missing TLS files")}
	key,err:=ecdsa.GenerateKey(elliptic.P256(),rand.Reader);if err!=nil {t.Fatal(err)}
	now:=time.Now()
	template:=&x509.Certificate{SerialNumber:big.NewInt(1),NotBefore:now.Add(-time.Hour),NotAfter:now.Add(time.Hour),DNSNames:[]string{"localhost"},KeyUsage:x509.KeyUsageDigitalSignature,ExtKeyUsage:[]x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	der,err:=x509.CreateCertificate(rand.Reader,template,template,&key.PublicKey,key);if err!=nil {t.Fatal(err)}
	keyDER,err:=x509.MarshalECPrivateKey(key);if err!=nil {t.Fatal(err)}
	dir:=t.TempDir()
	certFile,keyFile:=filepath.Join(dir,"cert.pem"),filepath.Join(dir,"key.pem")
	if err:=os.WriteFile(certFile,pem.EncodeToMemory(&pem.Block{Type:"CERTIFICATE",Bytes:der}),0600);err!=nil {t.Fatal(err)}
	if err:=os.WriteFile(keyFile,pem.EncodeToMemory(&pem.Block{Type:"EC PRIVATE KEY",Bytes:keyDER}),0600);err!=nil {t.Fatal(err)}
	cfg,err:=consoleTLSConfig(config.Config{TLSCertFile:certFile,TLSKeyFile:keyFile});if err!=nil {t.Fatal(err)}
	if cfg.MinVersion!=tls.VersionTLS12 {t.Fatal("unexpected TLS minimum")}
	s:=httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){if r.TLS==nil {t.Error("missing TLS state")};w.WriteHeader(http.StatusOK)}))
	s.TLS=cfg
	s.StartTLS()
	defer s.Close()
	cert,err:=x509.ParseCertificate(der);if err!=nil {t.Fatal(err)}
	roots:=x509.NewCertPool();roots.AddCert(cert)
	client:=&http.Client{Transport:&http.Transport{TLSClientConfig:&tls.Config{RootCAs:roots,ServerName:"localhost",MinVersion:tls.VersionTLS12}},Timeout:5*time.Second}
	defer client.CloseIdleConnections()
	resp,err:=client.Get(s.URL);if err!=nil {t.Fatal(err)}
	resp.Body.Close()
	if resp.StatusCode!=http.StatusOK {t.Fatalf("status %d",resp.StatusCode)}
	if err:=os.WriteFile(keyFile,[]byte("invalid key"),0600);err!=nil {t.Fatal(err)}
	if _,err:=consoleTLSConfig(config.Config{TLSCertFile:certFile,TLSKeyFile:keyFile});err==nil {t.Fatal("accepted invalid key")}
}
