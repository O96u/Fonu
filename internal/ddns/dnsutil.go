package ddns

import "strings"

func fqdnForRecord(rootDomain, recordName string) string {
	recordName = strings.TrimSpace(recordName)
	if recordName == "" || recordName == "@" {
		return rootDomain
	}
	if strings.Contains(recordName, ".") {
		return recordName
	}
	if recordName == "*" {
		return "*." + rootDomain
	}
	return recordName + "." + rootDomain
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
