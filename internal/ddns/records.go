package ddns

import (
	"encoding/json"
	"fmt"
	"strings"
)

func ValidateDomainFormat(domain string) error {
	domain = strings.TrimSpace(strings.ToLower(strings.TrimSuffix(domain, ".")))
	if domain == "" {
		return fmt.Errorf("域名不能为空")
	}
	if strings.HasPrefix(domain, "*.") {
		domain = strings.TrimPrefix(domain, "*.")
	}
	parts := strings.Split(domain, ".")
	if len(parts) < 2 {
		return fmt.Errorf("请填写完整域名，例如 s.example.com")
	}
	for _, part := range parts {
		if part == "" {
			return fmt.Errorf("域名格式无效")
		}
		if len(part) > 63 {
			return fmt.Errorf("域名标签过长")
		}
		for i := 0; i < len(part); i++ {
			c := part[i]
			if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '-' {
				return fmt.Errorf("域名包含非法字符")
			}
		}
		if part[0] == '-' || part[len(part)-1] == '-' {
			return fmt.Errorf("域名标签不能以连字符开头或结尾")
		}
	}
	return nil
}

func FQDNsFromSaveInput(in SaveInput) ([]string, error) {
	var names []string
	var err error
	if len(in.Domains) > 0 {
		_, names, err = ParseDomainLines(in.Domains)
		if err != nil {
			return nil, err
		}
	} else {
		root := strings.ToLower(strings.TrimSpace(in.RootDomain))
		if root == "" {
			return nil, fmt.Errorf("至少需要一个域名")
		}
		recordNames, err := ParseRecordNames(in.RecordNames, in.RecordName)
		if err != nil {
			return nil, err
		}
		names = FormatDomainLines(root, recordNames)
	}
	for _, name := range names {
		if err := ValidateDomainFormat(name); err != nil {
			return nil, fmt.Errorf("%s：%s", name, err.Error())
		}
	}
	return names, nil
}

func ParseDomainLines(lines []string) (rootDomain string, recordNames []string, err error) {
	seen := make(map[string]struct{})
	names := make([]string, 0, len(lines))
	for _, line := range lines {
		root, record, lineErr := parseDomainLine(line)
		if lineErr != nil {
			return "", nil, lineErr
		}
		if root == "" {
			continue
		}
		fqdn := FormatDomainLine(root, record)
		if _, ok := seen[fqdn]; ok {
			return "", nil, fmt.Errorf("域名 %s 重复", fqdn)
		}
		seen[fqdn] = struct{}{}
		if rootDomain == "" {
			rootDomain = root
		}
		names = append(names, fqdn)
	}
	if len(names) == 0 {
		return "", nil, fmt.Errorf("至少需要一个域名")
	}
	return rootDomain, names, nil
}

func ResolveDomainTarget(configRoot, name string) (rootDomain, recordName string) {
	name = strings.TrimSpace(strings.ToLower(name))
	if strings.Contains(name, ".") {
		if root, record, err := parseDomainLine(name); err == nil {
			return root, record
		}
	}
	return strings.ToLower(strings.TrimSpace(configRoot)), name
}

func parseDomainLine(line string) (rootDomain, recordName string, err error) {
	line = strings.TrimSpace(strings.ToLower(line))
	if line == "" {
		return "", "", nil
	}
	if idx := strings.Index(line, ":"); idx > 0 {
		host := line[:idx]
		if strings.Count(host, ":") <= 1 {
			line = host
		}
	}
	line = strings.TrimSuffix(line, ".")
	if strings.HasPrefix(line, "*.") {
		root := strings.TrimPrefix(line, "*.")
		if root == "" || !strings.Contains(root, ".") {
			return "", "", fmt.Errorf("域名格式无效: %s", line)
		}
		return root, "*", nil
	}
	parts := strings.Split(line, ".")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("域名格式无效: %s", line)
	}
	rootDomain = parts[len(parts)-2] + "." + parts[len(parts)-1]
	if len(parts) == 2 {
		return rootDomain, "@", nil
	}
	recordName = strings.Join(parts[:len(parts)-2], ".")
	return rootDomain, recordName, nil
}

func FormatDomainLine(rootDomain, recordName string) string {
	recordName = strings.TrimSpace(strings.ToLower(recordName))
	rootDomain = strings.TrimSpace(strings.ToLower(rootDomain))
	if strings.Contains(recordName, ".") {
		parts := strings.Split(recordName, ".")
		if len(parts) >= 3 {
			if root, record, err := parseDomainLine(recordName); err == nil {
				return joinDomainLine(root, record)
			}
		}
		if len(parts) == 2 {
			if root, record, err := parseDomainLine(recordName); err == nil && record == "@" && recordName == root {
				if rootDomain == "" || rootDomain == root {
					return recordName
				}
			}
		}
	}
	return joinDomainLine(rootDomain, recordName)
}

// canonicalFQDN returns name when it is already a complete FQDN (e.g. api.nas.example.com).
func canonicalFQDN(name string) string {
	name = strings.TrimSpace(strings.ToLower(strings.TrimSuffix(name, ".")))
	if name == "" || name == "@" || !strings.Contains(name, ".") {
		return ""
	}
	root, record, err := parseDomainLine(name)
	if err != nil {
		return name
	}
	fqdn := joinDomainLine(root, record)
	if fqdn == name {
		return name
	}
	return ""
}

func joinDomainLine(rootDomain, recordName string) string {
	if recordName == "" || recordName == "@" {
		return rootDomain
	}
	if recordName == "*" {
		return "*." + rootDomain
	}
	return recordName + "." + rootDomain
}

func FormatDomainLines(rootDomain string, recordNames []string) []string {
	out := make([]string, 0, len(recordNames))
	for _, name := range recordNames {
		if fqdn := canonicalFQDN(name); fqdn != "" {
			out = append(out, fqdn)
			continue
		}
		root, record := ResolveDomainTarget(rootDomain, name)
		out = append(out, FormatDomainLine(root, record))
	}
	return out
}

func ParseRecordNames(recordNames []string, legacy string) ([]string, error) {
	raw := recordNames
	if len(raw) == 0 && strings.TrimSpace(legacy) != "" {
		raw = strings.Split(legacy, "\n")
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("至少需要一个子域名")
	}

	seen := make(map[string]struct{})
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		name := normalizeRecordName(item)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			return nil, fmt.Errorf("子域名 %s 重复", name)
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("至少需要一个子域名")
	}
	return out, nil
}

func normalizeRecordName(raw string) string {
	name := strings.TrimSpace(strings.ToLower(raw))
	name = strings.TrimSuffix(name, ".")
	if name == "" || name == "@" {
		return "@"
	}
	if strings.HasPrefix(name, "*.") {
		root := strings.TrimPrefix(name, "*.")
		if root != "" {
			return FormatDomainLine(root, "*")
		}
		return "*"
	}
	if strings.Contains(name, ".") {
		root, record, err := parseDomainLine(name)
		if err == nil {
			return FormatDomainLine(root, record)
		}
	}
	return name
}

func EncodeRecordNames(names []string) string {
	if len(names) == 0 {
		return "[]"
	}
	b, err := json.Marshal(names)
	if err != nil {
		return "[]"
	}
	return string(b)
}

func DecodeRecordNames(raw string, fallback string) []string {
	raw = strings.TrimSpace(raw)
	if raw != "" && raw != "[]" {
		var names []string
		if err := json.Unmarshal([]byte(raw), &names); err == nil && len(names) > 0 {
			return names
		}
	}
	if strings.TrimSpace(fallback) != "" {
		return []string{normalizeRecordName(fallback)}
	}
	return nil
}

func (c Config) RecordNamesList() []string {
	if len(c.RecordNames) > 0 {
		return c.RecordNames
	}
	if names := DecodeRecordNames("", c.RecordName); len(names) > 0 {
		return names
	}
	return []string{"@"}
}

func (c Config) DomainLines() []string {
	return FormatDomainLines(c.RootDomain, c.RecordNamesList())
}

// ManagedDNSZones returns apex zones this DDNS credential can manage (for ACME DNS-01).
func (c Config) ManagedDNSZones() []string {
	seen := map[string]bool{}
	var zones []string
	add := func(zone string) {
		zone = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(zone, "*.")))
		if zone == "" || zone == "@" || seen[zone] {
			return
		}
		seen[zone] = true
		zones = append(zones, zone)
	}
	add(c.RootDomain)
	for _, name := range c.RecordNamesList() {
		fqdn := FQDNFromRecord(c, name)
		add(fqdn)
		root, record := ResolveDomainTarget(c.RootDomain, name)
		add(root)
		if record != "@" && record != "*" {
			add(FormatDomainLine(root, record))
		}
	}
	return zones
}

func (c Config) CoversDomain(domain string) bool {
	domain = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(domain, "*.")))
	for _, zone := range c.ManagedDNSZones() {
		if domain == zone || strings.HasSuffix(domain, "."+zone) {
			return true
		}
	}
	return false
}
