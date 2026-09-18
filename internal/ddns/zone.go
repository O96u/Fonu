package ddns

import (
	"context"
	"fmt"
	"strings"
)

type zoneChecker func(zone string) (bool, error)

// ResolveZoneForFQDN finds the longest DNS zone suffix that exists at the provider.
// For delegated zones (e.g. sub.example.com hosted separately), it prefers sub.example.com over example.com.
func ResolveZoneForFQDN(ctx context.Context, check zoneChecker, fqdn string) (zone, host string, err error) {
	fqdn = strings.ToLower(strings.TrimSpace(strings.TrimSuffix(fqdn, ".")))
	if fqdn == "" {
		return "", "", fmt.Errorf("域名不能为空")
	}

	if strings.HasPrefix(fqdn, "*.") {
		zone, host, err := ResolveZoneForFQDN(ctx, check, strings.TrimPrefix(fqdn, "*."))
		if err != nil {
			return "", "", err
		}
		if host == "@" {
			return zone, "*", nil
		}
		return zone, "*." + host, nil
	}

	parts := strings.Split(fqdn, ".")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("域名格式无效: %s", fqdn)
	}

	for i := 0; i < len(parts)-1; i++ {
		candidate := strings.Join(parts[i:], ".")
		ok, err := check(candidate)
		if err != nil {
			return "", "", err
		}
		if !ok {
			continue
		}
		host := "@"
		if i > 0 {
			host = strings.Join(parts[:i], ".")
		}
		return candidate, host, nil
	}
	return "", "", fmt.Errorf("未在 DNS 服务商中找到域名 %s 对应的 Zone", fqdn)
}

func FQDNFromRecord(cfg Config, name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return strings.ToLower(strings.TrimSpace(cfg.RootDomain))
	}
	if strings.Contains(name, ".") || strings.HasPrefix(name, "*.") {
		root, record, err := parseDomainLine(name)
		if err == nil {
			return FormatDomainLine(root, record)
		}
		return strings.TrimSuffix(name, ".")
	}
	return FormatDomainLine(cfg.RootDomain, name)
}

func isZoneNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "未找到") ||
		strings.Contains(msg, "not found") ||
		strings.Contains(msg, "resourcenotfound") ||
		strings.Contains(msg, "invaliddomain")
}

func zoneLookupOK(err error) (bool, error) {
	if err == nil {
		return true, nil
	}
	if isZoneNotFound(err) {
		return false, nil
	}
	return false, err
}
