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
