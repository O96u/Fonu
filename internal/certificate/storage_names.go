package certificate

import "strings"

// StorageNames returns on-disk directory names under the certs root for the given SAN list.
func StorageNames(domains []string) []string {
	seen := map[string]bool{}
	var names []string
	for _, domain := range domains {
		name := domain
		if strings.HasPrefix(domain, "*.") {
			name = "wildcard." + strings.TrimPrefix(domain, "*.")
		}
		if seen[name] {
			continue
		}
		seen[name] = true
		names = append(names, name)
	}
	return names
}

// RecordStorageNames returns storage directory names for a certificate record.
func RecordStorageNames(rec Record) []string {
	domains := rec.Domains
	if len(domains) == 0 {
		domain := strings.ToLower(strings.TrimSpace(rec.Domain))
		if rec.Wildcard {
			domains = []string{domain, "*." + domain}
		} else {
			domains = []string{domain}
		}
	}
	return StorageNames(domains)
}
