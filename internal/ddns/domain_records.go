package ddns

import (
	"encoding/json"
	"strings"
)

type DomainRecord struct {
	Domain  string `json:"domain"`
	IPv4    string `json:"ipv4,omitempty"`
	IPv6    string `json:"ipv6,omitempty"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

func EncodeDomainRecords(records []DomainRecord) string {
	if len(records) == 0 {
		return "[]"
	}
	b, err := json.Marshal(records)
	if err != nil {
		return "[]"
	}
	return string(b)
}

func DecodeDomainRecords(raw string) []DomainRecord {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return nil
	}
	var records []DomainRecord
	if err := json.Unmarshal([]byte(raw), &records); err != nil {
		return nil
	}
	return records
}

func BuildDomainRecords(cfg Config) []DomainRecord {
	stored := cfg.DomainRecords
	byDomain := make(map[string]DomainRecord, len(stored))
	for _, rec := range stored {
		byDomain[strings.ToLower(rec.Domain)] = rec
	}

	lines := cfg.DomainLines()
	out := make([]DomainRecord, 0, len(lines))
	for _, domain := range lines {
		key := strings.ToLower(domain)
		if rec, ok := byDomain[key]; ok {
			rec.Domain = domain
			out = append(out, rec)
			continue
		}
		rec := DomainRecord{Domain: domain, Status: "unknown"}
		if cfg.LastStatus != "" && cfg.LastStatus != "unknown" {
			rec.Status = cfg.LastStatus
			rec.Message = cfg.LastError
		}
		out = append(out, rec)
	}
	return out
}
