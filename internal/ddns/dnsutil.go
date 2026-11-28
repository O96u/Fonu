package ddns

import "strings"

func fqdnForRecord(rootDomain, recordName string) string {
	recordName = strings.TrimSpace(recordName)
	if recordName == "" || recordName == "@" {
		return rootDomain
	}
	// Provider calls receive a relative host from ResolveZoneForFQDN.
	// Dots in that host do not make it an absolute DNS name.
	return joinDomainLine(rootDomain, recordName)
}

func subDomainForRecord(recordName string) string {
	recordName = strings.TrimSpace(recordName)
	if recordName == "" || recordName == "@" {
		return "@"
	}
	return recordName
}

// txtValuePresent reports whether an ACME DNS-01 token is already published.
// Apex and wildcard share one TXT name and must keep both values.
func txtValuePresent(existing []string, want string) bool {
	want = strings.TrimSpace(want)
	if want == "" {
		return false
	}
	for _, value := range existing {
		if strings.TrimSpace(value) == want {
			return true
		}
	}
	return false
}
