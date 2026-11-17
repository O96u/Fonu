package api

import (
	"testing"

	certstore "github.com/fonu/fonu/internal/certificate"
)

func TestRequestTrendPercent(t *testing.T) {
	if got := requestTrendPercent(0, 0); got != nil {
		t.Fatalf("expected nil trend, got %v", *got)
	}
	if got := requestTrendPercent(10, 0); got == nil || *got != 100 {
		t.Fatalf("expected 100 for new traffic, got %v", got)
	}
	if got := requestTrendPercent(150, 100); got == nil || *got != 50 {
		t.Fatalf("expected 50, got %v", got)
	}
	if got := requestTrendPercent(50, 100); got == nil || *got != -50 {
		t.Fatalf("expected -50, got %v", got)
	}
}

func TestSummarizeCertificates(t *testing.T) {
	status, days, count, summary := summarizeCertificates(nil)
	if status != "none" || days != 0 || count != 0 || summary != "" {
		t.Fatalf("empty: got %s %d %d %q", status, days, count, summary)
	}

	ok := certstore.Record{Domain: "a.example.com", DaysLeft: 60, Status: "ok"}
	expiring := certstore.Record{Domain: "b.example.com", DaysLeft: 30, Status: "ok"}
	expired := certstore.Record{Domain: "c.example.com", DaysLeft: 0, Status: "ok"}
	failed := certstore.Record{Domain: "d.example.com", DaysLeft: 80, Status: "error"}

	status, days, count, _ = summarizeCertificates([]certstore.Record{ok})
	if status != "ok" || days != 60 || count != 1 {
		t.Fatalf("healthy: got %s %d %d", status, days, count)
	}

	status, days, count, _ = summarizeCertificates([]certstore.Record{expiring})
	if status != "expiring" || days != 30 || count != 1 {
		t.Fatalf("expiring: got %s %d %d", status, days, count)
	}

	status, days, count, _ = summarizeCertificates([]certstore.Record{ok, expiring})
	if status != "expiring" || days != 30 || count != 2 {
		t.Fatalf("mixed expiring: got %s %d %d", status, days, count)
	}

	status, _, _, _ = summarizeCertificates([]certstore.Record{expiring, expired})
	if status != "error" {
		t.Fatalf("expired should stay error, got %s", status)
	}

	status, _, _, _ = summarizeCertificates([]certstore.Record{failed, expiring})
	if status != "error" {
		t.Fatalf("apply failure should stay error, got %s", status)
	}
}
