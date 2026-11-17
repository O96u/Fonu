package fnos

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNormalizeGatewayURLRejectsPublicAndOpenRedirect(t *testing.T) {
	if _, err := NormalizeGatewayURL("http://1.2.3.4/app/fonu/", "10.147.20.5"); err == nil {
		t.Fatal("public address was accepted")
	}
	if _, err := NormalizeGatewayURL("http://evil.example/app/fonu/?next=https://phish", "10.147.20.5"); err == nil {
		t.Fatal("query redirect was accepted")
	}
	if _, err := NormalizeGatewayURL("http://10.147.20.5/secret", "10.147.20.5"); err == nil {
		t.Fatal("arbitrary path was accepted")
	}
	if _, err := NormalizeGatewayURL("http://user:pass@10.147.20.5/app/fonu/", "10.147.20.5"); err == nil {
		t.Fatal("credentials were accepted")
	}
	got, err := NormalizeGatewayURL("http://10.147.20.5:5666/app/fonu", "192.168.1.100")
	if err != nil {
		t.Fatal(err)
	}
	if got != "http://10.147.20.5:5666/app/fonu/" {
		t.Fatalf("got %s", got)
	}
	got, err = NormalizeGatewayURL("https://nas.example/app/fonu/", "nas.example")
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://nas.example/app/fonu/" {
		t.Fatalf("got %s", got)
	}
}

func TestVerifyGatewayRejectsCrossHostRedirectAndBadCert(t *testing.T) {
	okServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/app/fonu/" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer okServer.Close()
	if _, err := VerifyGateway(t.Context(), okServer.URL+"/app/fonu/", "127.0.0.1"); err != nil {
		t.Fatal(err)
	}

	hop := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://evil.example/phish", http.StatusFound)
	}))
	defer hop.Close()
	if _, err := VerifyGateway(t.Context(), hop.URL+"/app/fonu/", "127.0.0.1"); err == nil {
		t.Fatal("cross-host redirect was accepted")
	}

	tlsServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer tlsServer.Close()
	if _, err := VerifyGateway(t.Context(), tlsServer.URL+"/app/fonu/", "127.0.0.1"); err == nil {
		t.Fatal("untrusted certificate was accepted")
	}
}

func TestLoginRedirectKeepsZeroTierHost(t *testing.T) {
	t.Setenv("FONU_FNOS_URL", "")
	t.Setenv("TRIM_APPDEST", t.TempDir())
	t.Setenv("TRIM_APPNAME", "")
	req := httptestRequest("10.147.20.5:6893")
	saved := "http://192.168.1.100:5666/app/fonu/"
	got := LoginRedirect(req, saved)
	if got == saved {
		t.Fatal("redirect left the ZeroTier host for the LAN address")
	}
	if got != "" && got != "http://10.147.20.5:5666/app/fonu/" && got != "https://10.147.20.5:5667/app/fonu/" {
		t.Fatalf("got %s", got)
	}
}

func TestLoginRedirectUsesSavedURLOffNAS(t *testing.T) {
	t.Setenv("FONU_FNOS_URL", "")
	t.Setenv("TRIM_APPDEST", "")
	t.Setenv("TRIM_APPNAME", "")
	req := httptestRequest("127.0.0.1:6893")
	got := LoginRedirect(req, "http://10.147.20.5:5666/app/fonu/")
	if got != "http://10.147.20.5:5666/app/fonu/" {
		t.Fatalf("got %s", got)
	}
	if got := LoginRedirect(req, ""); got != "" {
		t.Fatalf("docker without a saved gateway returned %s", got)
	}
}
