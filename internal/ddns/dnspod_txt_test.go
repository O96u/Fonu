package ddns

import "testing"

func TestDnspodTXTValue(t *testing.T) {
	if got := dnspodTXTValue(`"abc"`); got != "abc" {
		t.Fatalf("quoted: got %q", got)
	}
	if got := dnspodTXTValue(" abc "); got != "abc" {
		t.Fatalf("trim: got %q", got)
	}
}
