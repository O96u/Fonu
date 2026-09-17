package api

import "testing"

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
