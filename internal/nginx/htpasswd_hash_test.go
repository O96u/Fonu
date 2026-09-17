package nginx

import "testing"

func TestNginxHtpasswdHashNormalizesBcryptPrefix(t *testing.T) {
	in := "$2a$10$abcdefghijklmnopqrstuv"
	got := nginxHtpasswdHash(in)
	want := "$2y$10$abcdefghijklmnopqrstuv"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestNginxHtpasswdHashLeavesOtherFormats(t *testing.T) {
	in := "$apr1$abc123"
	if got := nginxHtpasswdHash(in); got != in {
		t.Fatalf("got %q want %q", got, in)
	}
}
