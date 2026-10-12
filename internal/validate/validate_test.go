package validate

import "testing"

func TestUpstreamIPWithDefaultPort(t *testing.T) {
	got, err := Upstream("http://192.168.2.1:80")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "http://192.168.2.1:80" {
		t.Fatalf("got %q", got)
	}

	_, err = Upstream("http://192.168.2.1")
	if err == nil {
		t.Fatal("expected error for IP without port")
	}

	got, err = Upstream("https://192.168.2.1:443")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "https://192.168.2.1:443" {
		t.Fatalf("got %q", got)
	}
}

func TestFrontendAddress(t *testing.T) {
	host, port, err := FrontendAddress("example.com:8011")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if host != "example.com" || port != 8011 {
		t.Fatalf("got %q %d", host, port)
	}

	host, port, err = FrontendAddress("nas.example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if host != "nas.example.com" || port != 0 {
		t.Fatalf("got %q %d", host, port)
	}
}
