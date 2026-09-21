package notify

import "testing"

func TestNormalizeProxyURL(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"127.0.0.1:7890", "http://127.0.0.1:7890"},
		{"http://127.0.0.1:7890", "http://127.0.0.1:7890"},
		{"socks5://127.0.0.1:7890", "socks5://127.0.0.1:7890"},
		{"", ""},
		{"  http://127.0.0.1:7890  ", "http://127.0.0.1:7890"},
	}
	for _, tc := range tests {
		got, err := normalizeProxyURL(tc.in)
		if err != nil {
			t.Fatalf("%q: %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("%q => %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestNormalizeProxyURLInvalid(t *testing.T) {
	_, err := normalizeProxyURL("http://127.0.0.1")
	if err == nil {
		t.Fatal("expected error for missing port")
	}
}
