package iplocate

import "strings"

// Region holds structured geo fields for analytics and maps.
type Region struct {
	Country  string
	Province string
	City     string
	Label    string
}

// ParseRegion resolves ip to structured fields (empty if unknown).
func (s *Service) ParseRegion(ip string) Region {
	ip = strings.TrimSpace(ip)
	if ip == "" || ip == "—" {
		return Region{}
	}
	if label, ok := LabelForPrivate(ip); ok {
		return Region{Country: "本地", Label: label}
	}
	raw := s.lookupRaw(ip)
	if raw == "" {
		return Region{}
	}
	return regionFromRaw(raw)
}

func (s *Service) lookupRaw(ip string) string {
	s.mu.RLock()
	searcher := s.searcher
	s.mu.RUnlock()
	if searcher == nil {
		s.kickEnsure()
		return ""
	}
	raw, err := searcher.Search(ip)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(raw)
}

func regionFromRaw(raw string) Region {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "0" {
		return Region{}
	}
	parts := strings.Split(raw, "|")
	for len(parts) < 5 {
		parts = append(parts, "")
	}
	country := translateCountry(trimField(parts[0]))
	region := trimField(parts[1])
	province := trimField(parts[2])
	city := trimField(parts[3])
	isp := trimField(parts[4])
	if province == "" && region != "" {
		province = region
	}
	if country == "中国" && isCNISPName(city) && (isp == "" || isCountryCode(isp)) {
		isp = city
		city = ""
	}
	if isCNISPName(province) && city == "" && isp == "" {
		isp = province
		province = ""
	}
	label := FormatRegion(raw)
	if country == "中国" {
		province = normalizeProvinceKey(province)
	}
	return Region{
		Country:  country,
		Province: province,
		City:     city,
		Label:    label,
	}
}

// normalizeProvinceKey strips suffix for map matching (广东省 -> 广东).
func normalizeProvinceKey(province string) string {
	province = strings.TrimSpace(province)
	if province == "" {
		return ""
	}
	for _, suf := range []string{"特别行政区", "维吾尔自治区", "壮族自治区", "回族自治区", "自治区", "省", "市"} {
		if strings.HasSuffix(province, suf) {
			return strings.TrimSuffix(province, suf)
		}
	}
	return province
}
