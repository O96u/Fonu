package acme

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fonu/fonu/internal/validate"
)

func NormalizeCertDomains(raw []string) ([]string, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("请填写至少一个域名")
	}
	if len(raw) > 100 {
		return nil, fmt.Errorf("域名数量不能超过 100 个")
	}

	seen := map[string]bool{}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		for _, part := range splitDomainInput(item) {
			domain, err := normalizeCertDomain(part)
			if err != nil {
				return nil, err
			}
			if seen[domain] {
				continue
			}
			seen[domain] = true
			out = append(out, domain)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("请填写至少一个域名")
	}
	return out, nil
}

func splitDomainInput(input string) []string {
	input = strings.ReplaceAll(input, "\r\n", "\n")
	input = strings.ReplaceAll(input, ",", "\n")
	parts := strings.Split(input, "\n")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func normalizeCertDomain(domain string) (string, error) {
	domain = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(domain)), ".")
	if domain == "" {
		return "", fmt.Errorf("域名不能为空")
	}
	wildcard := strings.HasPrefix(domain, "*.")
	base := strings.TrimPrefix(domain, "*.")
	if err := validate.Domain(base); err != nil {
		return "", err
	}
	if wildcard {
		return "*." + base, nil
	}
	return base, nil
}

func DomainsUnderZone(domains []string, zone string) error {
	return DomainsUnderZones(domains, []string{zone})
}

func DomainsUnderZones(domains []string, zones []string) error {
	normalized := normalizeManagedZones(zones)
	if len(normalized) == 0 {
		return fmt.Errorf("请选择 DNS 凭证")
	}
	for _, domain := range domains {
		base := strings.TrimPrefix(domain, "*.")
		if !domainUnderAnyZone(base, normalized) {
			return fmt.Errorf("域名 %s 不在所选 DNS 凭证的管理范围内（%s）", domain, strings.Join(normalized, "、"))
		}
	}
	return nil
}

func normalizeManagedZones(zones []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(zones))
	for _, zone := range zones {
		zone = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(zone, "*.")))
		if zone == "" || seen[zone] {
			continue
		}
		seen[zone] = true
		out = append(out, zone)
	}
	return out
}

func domainUnderAnyZone(domain string, zones []string) bool {
	for _, zone := range zones {
		if domain == zone || strings.HasSuffix(domain, "."+zone) {
			return true
		}
	}
	return false
}

func PrimaryCertDomain(domains []string) string {
	for _, domain := range domains {
		if !strings.HasPrefix(domain, "*.") {
			return domain
		}
	}
	if len(domains) > 0 {
		return strings.TrimPrefix(domains[0], "*.")
	}
	return ""
}

func HasWildcardDomain(domains []string) bool {
	for _, domain := range domains {
		if strings.HasPrefix(domain, "*.") {
			return true
		}
	}
	return false
}

func LegacyCertDomains(domain string, wildcard bool) []string {
	domain = strings.ToLower(strings.TrimSpace(domain))
	if domain == "" {
		return nil
	}
	if wildcard {
		return []string{domain, "*." + domain}
	}
	return []string{domain}
}

func EncodeCertDomains(domains []string) string {
	if len(domains) == 0 {
		return ""
	}
	b, err := json.Marshal(domains)
	if err != nil {
		return ""
	}
	return string(b)
}

func DecodeCertDomains(raw string, domain string, wildcard bool) []string {
	raw = strings.TrimSpace(raw)
	if raw != "" {
		var domains []string
		if err := json.Unmarshal([]byte(raw), &domains); err == nil && len(domains) > 0 {
			normalized, err := NormalizeCertDomains(domains)
			if err == nil {
				return normalized
			}
		}
	}
	return LegacyCertDomains(domain, wildcard)
}
