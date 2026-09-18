package notify

import (
	"strings"
	"testing"
	"time"
)

func TestFormatAlert(t *testing.T) {
	at := time.Date(2026, 9, 18, 15, 49, 0, 0, time.FixedZone("CST", 8*3600))
	content := FormatAlertAt(EventDDNSIPChange, "DDNS IP 已变更", "example.com\nIPv4: 1.2.3.4 → 5.6.7.8", at)

	if content.Subject != "[Fonu] 🌐 DDNS IP 已变更" {
		t.Fatalf("unexpected subject: %s", content.Subject)
	}
	if !strings.Contains(content.PlainBody, "example.com") {
		t.Fatalf("missing detail: %s", content.PlainBody)
	}
	if !strings.Contains(content.PlainBody, "Fonu · 2026-09-18 15:49:00") {
		t.Fatalf("missing footer: %s", content.PlainBody)
	}
	if !strings.Contains(content.HTMLBody, "<b>") || !strings.Contains(content.HTMLBody, "<pre>") {
		t.Fatalf("unexpected html: %s", content.HTMLBody)
	}
	if content.webhookPayload()["category"] != "DDNS" {
		t.Fatalf("unexpected webhook category: %v", content.webhookPayload())
	}
}
