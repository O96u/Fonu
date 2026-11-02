package awareness

import (
	"testing"

	"github.com/fonu/fonu/internal/logstore"
)

func TestMatchKindFilterIntercept(t *testing.T) {
	if !MatchKindFilter(KindBlocked, "intercept") {
		t.Fatal("403 should match intercept")
	}
	if !MatchKindFilter(KindRateLimited, "blocked") {
		t.Fatal("429 should match blocked alias")
	}
	if MatchKindFilter(KindAccess, "intercept") {
		t.Fatal("normal access should not match")
	}
}

func TestClassifyAccess403(t *testing.T) {
	kind, summary := ClassifyAccess(logstore.AccessEntry{Status: 403, Path: "/api"})
	if kind != KindBlocked || summary == "" {
		t.Fatalf("got kind=%q summary=%q", kind, summary)
	}
}
