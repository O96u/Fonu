package awareness

import "github.com/fonu/fonu/internal/iplocate"

// EnrichEventsGeo fills Event.Geo using lookup (deduped per IP).
func EnrichEventsGeo(events []Event, geo *iplocate.Service) {
	if geo == nil || len(events) == 0 {
		return
	}
	cache := map[string]string{}
	for i := range events {
		events[i].Geo = geo.LookupCached(events[i].ClientIP, cache)
	}
}
