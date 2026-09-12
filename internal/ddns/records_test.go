package ddns

import "testing"

func TestParseDomainLinesMultiple(t *testing.T) {
	root, names, err := ParseDomainLines([]string{"s.example.com", "www.example.com", "example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if root != "example.com" || len(names) != 3 || names[0] != "s.example.com" || names[1] != "www.example.com" || names[2] != "example.com" {
		t.Fatalf("unexpected: root=%s names=%v", root, names)
	}
}

func TestParseDomainLinesAllowsMixedRoots(t *testing.T) {
	root, names, err := ParseDomainLines([]string{"s.example.com", "www.other.com"})
	if err != nil {
		t.Fatal(err)
	}
	if root != "example.com" || len(names) != 2 || names[1] != "www.other.com" {
		t.Fatalf("unexpected: root=%s names=%v", root, names)
	}
}

func TestParseDomainLinesWildcard(t *testing.T) {
	root, names, err := ParseDomainLines([]string{"*.example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if root != "example.com" || names[0] != "*.example.com" {
		t.Fatalf("unexpected: root=%s names=%v", root, names)
	}
}

func TestFormatDomainLine(t *testing.T) {
	if FormatDomainLine("example.com", "s") != "s.example.com" {
		t.Fatal("subdomain format failed")
	}
	if FormatDomainLine("example.com", "@") != "example.com" {
		t.Fatal("apex format failed")
	}
}

func TestParseRecordNamesRejectsDuplicate(t *testing.T) {
	_, err := ParseRecordNames([]string{"s", "s"}, "")
	if err == nil {
		t.Fatal("expected duplicate error")
	}
}
