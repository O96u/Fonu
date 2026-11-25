package acme

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

func TestStageCertFilesRollback(t *testing.T) {
 key,err:=ecdsa.GenerateKey(elliptic.P256(),rand.Reader);if err!=nil{t.Fatal(err)}
 template:= &x509.Certificate{SerialNumber:big.NewInt(1),NotBefore:time.Now().Add(-time.Hour),NotAfter:time.Now().Add(time.Hour),DNSNames:[]string{"example.com"}}
 der,err:=x509.CreateCertificate(rand.Reader,template,template,&key.PublicKey,key);if err!=nil{t.Fatal(err)}
 kd,err:=x509.MarshalECPrivateKey(key);if err!=nil{t.Fatal(err)}
 cert:=pem.EncodeToMemory(&pem.Block{Type:"CERTIFICATE",Bytes:der})
 priv:=pem.EncodeToMemory(&pem.Block{Type:"EC PRIVATE KEY",Bytes:kd})
 root:=t.TempDir();dir:=filepath.Join(root,"example.com")
 if err=os.Mkdir(dir,0700);err!=nil{t.Fatal(err)}
 for _,name:=range []string{"fullchain.pem","privatekey.pem"}{if err=os.WriteFile(filepath.Join(dir,name),[]byte("old"),0600);err!=nil{t.Fatal(err)}}
 _,_,finish,err:=stageCertFiles(root,[]string{"example.com"},cert,priv);if err!=nil{t.Fatal(err)}
 if err=finish(false);err!=nil{t.Fatal(err)}
 for _,name:=range []string{"fullchain.pem","privatekey.pem"}{data,err:=os.ReadFile(filepath.Join(dir,name));if err!=nil||string(data)!="old"{t.Fatalf("rollback failed: %s %v",name,err)}}
 _,_,finish,err=stageCertFiles(root,[]string{"example.com"},cert,priv);if err!=nil{t.Fatal(err)}
 if err=finish(true);err!=nil{t.Fatal(err)}
 data,err:=os.ReadFile(filepath.Join(dir,"fullchain.pem"));if err!=nil||string(data)!=string(cert){t.Fatal("commit failed")}
}
