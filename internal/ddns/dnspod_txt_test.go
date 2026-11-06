package ddns

import "testing"

func TestTxtValuePresentKeepsDistinctTokens(t *testing.T) {
	existing := []string{"token-apex"}
	if txtValuePresent(existing, "token-apex") != true {
		t.Fatal("expected existing token")
	}
	if txtValuePresent(existing, "token-wildcard") {
		t.Fatal("different token must be added, not treated as present")
	}
}

func TestDnspodTXTValue(t *testing.T) {
	if got := dnspodTXTValue(`"abc"`); got != "abc" {
		t.Fatalf("quoted: got %q", got)
	}
	if got := dnspodTXTValue(" abc "); got != "abc" {
		t.Fatalf("trim: got %q", got)
	}
}
