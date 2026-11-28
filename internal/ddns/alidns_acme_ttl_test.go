package ddns

import (
	"strconv"
	"testing"
)

func TestAliDNSACMETTLWithinProviderRange(t *testing.T) {
	ttl, err := strconv.Atoi(alidnsACMETTL)
	if err != nil {
		t.Fatalf("ACME TTL must be numeric: %v", err)
	}
	if ttl < 600 || ttl > 86400 {
		t.Fatalf("Aliyun rejects ACME TXT TTL outside 600-86400, got %d", ttl)
	}
}
