package api
import("net/http/httptest";"testing";"testing/fstest")
func TestSPACachePolicy(t *testing.T){
	r:=&Router{staticFS:fstest.MapFS{"index.html":{Data:[]byte("<html>app</html>")},"assets/index-AbCd1234.js":{Data:[]byte("export {}")}}}
	for _,tc:=range []struct{path,cache string;status int}{{"/","no-cache",200},{"/dashboard","no-cache",200},{"/assets/index-AbCd1234.js","public, max-age=31536000, immutable",200},{"/assets/missing.js","",404}}{
		w:=httptest.NewRecorder();r.spaHandler().ServeHTTP(w,httptest.NewRequest("GET",tc.path,nil))
		if w.Code!=tc.status||w.Header().Get("Cache-Control")!=tc.cache{t.Errorf("%s: status=%d cache=%q",tc.path,w.Code,w.Header().Get("Cache-Control"))}
	}
}
