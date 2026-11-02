package awareness

import (
	"strings"
	"time"
)

const (
	RangeToday = "today"
	Range7d    = "7d"
	Range30d   = "30d"
	RangeAll   = "all"
)

// Period describes a stats window in the app timezone.
type Period struct {
	Range      string
	Start      time.Time
	End        time.Time
	BucketKind string // two_hour | daily
	MaxBuckets int
}

// NormalizeRange returns a supported range key.
func NormalizeRange(raw string) string {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "", RangeToday:
		return RangeToday
	case Range7d, "7":
		return Range7d
	case Range30d, "30":
		return Range30d
	case RangeAll:
		return RangeAll
	default:
		return RangeToday
	}
}

// PeriodForRange builds inclusive [Start, End] bounds for aggregation.
func PeriodForRange(rangeKey string, now time.Time, loc *time.Location) Period {
	rangeKey = NormalizeRange(rangeKey)
	if loc == nil {
		loc = time.Local
	}
	now = now.In(loc)
	end := now

	switch rangeKey {
	case Range7d:
		start := startOfDay(now.AddDate(0, 0, -6), loc)
		return Period{
			Range:      Range7d,
			Start:      start,
			End:        end,
			BucketKind: "daily",
			MaxBuckets: 7,
		}
	case Range30d:
		start := startOfDay(now.AddDate(0, 0, -29), loc)
		return Period{
			Range:      Range30d,
			Start:      start,
			End:        end,
			BucketKind: "daily",
			MaxBuckets: 30,
		}
	case RangeAll:
		start := startOfDay(now.AddDate(-1, 0, 0), loc) // up to 365 daily buckets
		return Period{
			Range:      RangeAll,
			Start:      start,
			End:        end,
			BucketKind: "daily",
			MaxBuckets: 366,
		}
	default:
		start := startOfDay(now, loc)
		return Period{
			Range:      RangeToday,
			Start:      start,
			End:        end,
			BucketKind: "two_hour",
			MaxBuckets: 12,
		}
	}
}

func startOfDay(t time.Time, loc *time.Location) time.Time {
	t = t.In(loc)
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, loc)
}

func dayIndex(start, t time.Time, loc *time.Location) int {
	s := startOfDay(start.In(loc), loc)
	tt := startOfDay(t.In(loc), loc)
	return int(tt.Sub(s) / (24 * time.Hour))
}
