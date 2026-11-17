package acme

import "testing"

func TestResolveFreeSSLDirectoryURL(t *testing.T) {
	t.Parallel()
	const want = FreeSSLDirectoryURLDefault

	cases := []struct {
		custom string
		want   string
	}{
		{"", want},
		{"  ", want},
		{"https://acmepro.freessl.cn/v2/DV", want},
		{"https://acme.freessl.cn/v2/DV90/directory", want},
		{"https://acme.freessl.cn/v2/DV90/directory/my-token", want},
		{"https://acme.example.com/directory", "https://acme.example.com/directory"},
	}
	for _, tc := range cases {
		got := resolveFreeSSLDirectoryURL(tc.custom, "")
		if got != tc.want {
			t.Errorf("resolveFreeSSLDirectoryURL(%q) = %q, want %q", tc.custom, got, tc.want)
		}
	}
}

func TestDirectoryURLFreeSSL(t *testing.T) {
	t.Parallel()
	got, err := DirectoryURL(CAFreeSSL)
	if err != nil {
		t.Fatal(err)
	}
	if got != FreeSSLDirectoryURLDefault {
		t.Fatalf("DirectoryURL(freessl) = %q, want %q", got, FreeSSLDirectoryURLDefault)
	}
}
