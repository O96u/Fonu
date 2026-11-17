package fnos

import (
	"net/http"
	"testing"
)

func TestBrowserEntryUsesConfiguredURL(t *testing.T) {
	t.Setenv("FONU_FNOS_URL", "http://10.147.20.8:5666/ignored")
	req := httptestRequest("203.0.113.5:6893")
	if got := BrowserEntry(req); got != "http://10.147.20.8:5666/app/fonu/" {
		t.Fatalf("got %s", got)
	}
}

func TestBrowserEntryKeepsRequestHostOnNAS(t *testing.T) {
	t.Setenv("FONU_FNOS_URL", "")
	t.Setenv("TRIM_APPDEST", t.TempDir())
	req := httptestRequest("10.147.20.8:6893")
	got := BrowserEntry(req)
	if got != "http://10.147.20.8:5666/app/fonu/" && got != "https://10.147.20.8:5667/app/fonu/" {
		t.Fatalf("got %s", got)
	}
}

func TestBrowserEntryEmptyOffNAS(t *testing.T) {
	t.Setenv("FONU_FNOS_URL", "")
	t.Setenv("TRIM_APPDEST", "")
	t.Setenv("TRIM_APPNAME", "")
	req := httptestRequest("10.147.20.8:6893")
	if got := BrowserEntry(req); got != "" {
		t.Fatalf("got %s", got)
	}
}

func httptestRequest(host string) *http.Request {
	req, err := http.NewRequest(http.MethodGet, "http://"+host+"/api/auth/fnos", nil)
	if err != nil {
		panic(err)
	}
	req.Host = host
	return req
}
