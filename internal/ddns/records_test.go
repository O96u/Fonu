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
	root, names, err := ParseDomainLines([]string{"s.example.com", "api.example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if root != "example.com" || len(names) != 2 || names[1] != "api.example.com" {
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

func TestParseRecordNamesKeepsDelegatedSubdomainFQDN(t *testing.T) {
	names, err := ParseRecordNames([]string{"s", "www.sub.example.com"}, "")
	if err != nil {
		t.Fatal(err)
	}
	if names[0] != "s" || names[1] != "www.sub.example.com" {
		t.Fatalf("unexpected names: %v", names)
	}
}

func TestDomainLinesMixedRootDomains(t *testing.T) {
	cfg := Config{
		RootDomain:  "example.com",
		RecordNames: []string{"s", "www.sub.example.com"},
	}
	lines := cfg.DomainLines()
	if len(lines) != 2 || lines[0] != "s.example.com" || lines[1] != "www.sub.example.com" {
		t.Fatalf("unexpected lines: %v", lines)
	}
}

func TestFQDNFromRecordDelegatedSubdomainHost(t *testing.T) {
	cfg := Config{RootDomain: "example.com"}
	if got := FQDNFromRecord(cfg, "www.sub.example.com"); got != "www.sub.example.com" {
		t.Fatalf("expected www.sub.example.com, got %s", got)
	}
}

func TestFormatDomainLineMultiLabelHost(t *testing.T) {
	if got := FormatDomainLine("example.com", "api.nas"); got != "api.nas.example.com" {
		t.Fatalf("expected api.nas.example.com, got %s", got)
	}
}

func TestParseDomainLinesDeepSubdomain(t *testing.T) {
	_, names, err := ParseDomainLines([]string{"api.nas.example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if names[0] != "api.nas.example.com" {
		t.Fatalf("expected full fqdn, got %s", names[0])
	}
}

func TestDomainLinesPreservesDeepSubdomain(t *testing.T) {
	cfg := Config{
		RootDomain:  "example.com",
		RecordNames: []string{"s.example.com", "api.nas.example.com"},
	}
	lines := cfg.DomainLines()
	if len(lines) != 2 || lines[1] != "api.nas.example.com" {
		t.Fatalf("unexpected lines: %v", lines)
	}
}

func TestValidateDomainFormat(t *testing.T) {
	if err := ValidateDomainFormat("s.example.com"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateDomainFormat("www.sub.example.com"); err != nil {
		t.Fatal(err)
	}
	if ValidateDomainFormat("s") == nil {
		t.Fatal("expected error for short domain")
	}
	if ValidateDomainFormat("bad_label-.com") == nil {
		t.Fatal("expected error for invalid label")
	}
}

func TestFQDNsFromSaveInput(t *testing.T) {
	names, err := FQDNsFromSaveInput(SaveInput{
		Domains: []string{"s.example.com", "www.sub.example.com"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 2 || names[1] != "www.sub.example.com" {
		t.Fatalf("unexpected names: %v", names)
	}
}
