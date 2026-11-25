package awareness

import (
	"testing"
	"time"
)

func TestNormalizeRange(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", RangeToday},
		{"today", RangeToday},
		{"7d", Range7d},
		{"30", Range30d},
		{"all", RangeAll},
		{"bogus", RangeToday},
	}
	for _, tc := range tests {
		if got := NormalizeRange(tc.in); got != tc.want {
			t.Fatalf("NormalizeRange(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestPeriodForRange7dDaily(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	now := time.Date(2026, 3, 15, 14, 30, 0, 0, loc)
	p := PeriodForRange(Range7d, now, loc)
	if p.BucketKind != "daily" || p.MaxBuckets != 7 {
		t.Fatalf("unexpected period: %+v", p)
	}
	if p.Start.Format("2006-01-02") != "2026-03-09" {
		t.Fatalf("start = %v", p.Start)
	}
}

func TestDayIndexAcrossDST(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil { t.Fatal(err) }
	for _, start := range []time.Time{time.Date(2026, 3, 7, 0, 0, 0, 0, loc), time.Date(2026, 10, 31, 0, 0, 0, 0, loc)} {
		if got := dayIndex(start, start.AddDate(0, 0, 2), loc); got != 2 { t.Fatalf("dayIndex = %d, want 2", got) }
	}
}
