package frp

import "testing"

func TestNormalizeServerAddr(t *testing.T) {
	if got := normalizeServerAddr("http://127.0.0.1"); got != "127.0.0.1" {
		t.Fatalf("got %q", got)
	}
	if got := normalizeServerAddr("https://frp.example.com/path"); got != "frp.example.com" {
		t.Fatalf("got %q", got)
	}
}
