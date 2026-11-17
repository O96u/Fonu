package fnos

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseIdentityAcceptsBothUIDHeaders(t *testing.T) {
	h := make(http.Header)
	h.Set("X-Trim-Userid", "1000")
	h.Set("X-Trim-Username", "admin")
	h.Set("X-Trim-Isadmin", "true")
	ident, ok := ParseIdentity(h)
	if !ok || ident.UID != "1000" || ident.Username != "admin" || !ident.IsAdmin {
		t.Fatalf("userid header: %+v ok=%v", ident, ok)
	}

	h = make(http.Header)
	h.Set("X-Trim-Uid", "1001")
	h.Set("X-Trim-Isadmin", "false")
	ident, ok = ParseIdentity(h)
	if !ok || ident.UID != "1001" || ident.IsAdmin {
		t.Fatalf("uid header: %+v ok=%v", ident, ok)
	}

	if _, ok := ParseIdentity(make(http.Header)); ok {
		t.Fatal("empty headers must not produce an identity")
	}
}

func TestHandlerStripsPrefixAndIgnoresMissingIdentity(t *testing.T) {
	var gotPath string
	var gotAdmin bool
	var gotPrefix string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotPrefix = PrefixFromContext(r.Context())
		if ident, ok := IdentityFromContext(r.Context()); ok {
			gotAdmin = ident.IsAdmin
		}
		w.WriteHeader(http.StatusNoContent)
	})
	h := Handler("/app/fonu", next)

	req := httptest.NewRequest(http.MethodGet, "/app/fonu/api/auth/fnos", nil)
	req.Header.Set("X-Trim-Userid", "1000")
	req.Header.Set("X-Trim-Username", "admin")
	req.Header.Set("X-Trim-Isadmin", "true")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status %d", rec.Code)
	}
	if gotPath != "/api/auth/fnos" || gotPrefix != "/app/fonu" || !gotAdmin {
		t.Fatalf("path=%s prefix=%s admin=%v", gotPath, gotPrefix, gotAdmin)
	}

	root := httptest.NewRequest(http.MethodGet, "/app/fonu", nil)
	rootRec := httptest.NewRecorder()
	h.ServeHTTP(rootRec, root)
	if rootRec.Code != http.StatusFound || rootRec.Header().Get("Location") != "/app/fonu/" {
		t.Fatalf("root redirect status=%d location=%s", rootRec.Code, rootRec.Header().Get("Location"))
	}

	outside := httptest.NewRequest(http.MethodGet, "/api/auth/fnos", nil)
	outside.Header.Set("X-Trim-Isadmin", "true")
	outside.Header.Set("X-Trim-Username", "admin")
	outRec := httptest.NewRecorder()
	h.ServeHTTP(outRec, outside)
	if outRec.Code != http.StatusNotFound {
		t.Fatalf("unprefixed status %d", outRec.Code)
	}
}

func TestStripPrefix(t *testing.T) {
	cases := []struct {
		path string
		want string
		ok   bool
	}{
		{"/app/fonu", "/", true},
		{"/app/fonu/", "/", true},
		{"/app/fonu/dashboard", "/dashboard", true},
		{"/app/other", "", false},
		{"/app/fonu-extra", "", false},
	}
	for _, tc := range cases {
		got, ok := stripPrefix(tc.path, "/app/fonu")
		if ok != tc.ok || got != tc.want {
			t.Fatalf("%s: got %q ok=%v want %q ok=%v", tc.path, got, ok, tc.want, tc.ok)
		}
	}
}
