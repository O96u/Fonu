package certificate

import "testing"

func TestStorageNamesWildcard(t *testing.T) {
	names := StorageNames([]string{"example.com", "*.example.com"})
	if len(names) != 2 || names[1] != "wildcard.example.com" {
		t.Fatalf("got %v", names)
	}
}
