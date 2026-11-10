package ddns

import "testing"

func TestHuaweiTXTValue(t *testing.T) {
	if got := huaweiQuoteTXT(`"abc"`); got != `"abc"` {
		t.Fatalf("quote: %s", got)
	}
	if got := huaweiPlainValue(`"abc"`); got != "abc" {
		t.Fatalf("plain: %s", got)
	}
	if got := huaweiFQDN("Example.COM."); got != "example.com." {
		t.Fatalf("fqdn: %s", got)
	}
}
