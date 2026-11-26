package nginx

import (
	"path/filepath"
	"testing"
	"time"
)
func TestResolverDoesNotCacheAcrossPublications(t *testing.T){
	dir:=t.TempDir()
	path:=filepath.Join(dir,"wildcard.example.com")
	writeTestCertificate(t,path,[]string{"*.example.com"},time.Now().Add(time.Hour))
	r:=newCertificateResolver(dir,nil)
	for _,host:=range []string{"a.example.com","b.example.com"}{
		if r.find([]string{host})==nil{t.Fatal("missing certificate")}
	}
	if len(r.leaves)!=1{t.Fatalf("cached %d pairs",len(r.leaves))}
	writeTestCertificate(t,path,[]string{"other.example.com"},time.Now().Add(time.Hour))
	if newCertificateResolver(dir,nil).find([]string{"a.example.com"})!=nil{t.Fatal("renewed SAN not re-read")}
}
