package analytics

import (
	"sort"

	"github.com/fonu/fonu/internal/iplocate"
)

func mergeCountryRows(rows []CountryRow) []CountryRow {
	type acc struct {
		requests  int
		intercept int
	}
	m := map[string]*acc{}
	for _, r := range rows {
		c := iplocate.NormalizeCountry(r.Country)
		if c == "" || c == "未知" {
			continue
		}
		a := m[c]
		if a == nil {
			a = &acc{}
			m[c] = a
		}
		a.requests += r.Requests
		a.intercept += r.Intercept
	}
	out := make([]CountryRow, 0, len(m))
	for c, a := range m {
		out = append(out, CountryRow{Country: c, Requests: a.requests, Intercept: a.intercept})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Requests == out[j].Requests {
			return out[i].Country < out[j].Country
		}
		return out[i].Requests > out[j].Requests
	})
	return out
}

func mergeRegionVisitors(rows []RegionVisitorRow, limit int) []RegionVisitorRow {
	type acc struct {
		visitors int
	}
	m := map[string]*acc{}
	for _, r := range rows {
		region := iplocate.NormalizeCountry(r.Region)
		if region == "" || region == "未知" {
			continue
		}
		a := m[region]
		if a == nil {
			a = &acc{}
			m[region] = a
		}
		a.visitors += r.Visitors
	}
	out := make([]RegionVisitorRow, 0, len(m))
	for region, a := range m {
		out = append(out, RegionVisitorRow{Region: region, Visitors: a.visitors})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Visitors == out[j].Visitors {
			return out[i].Region < out[j].Region
		}
		return out[i].Visitors > out[j].Visitors
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}
