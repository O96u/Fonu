package validate

import "testing"

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
