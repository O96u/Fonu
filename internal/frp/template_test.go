package frp

import (
	"strings"
	"testing"
)

func TestBuildFRPSConfig(t *testing.T) {
	out := BuildFRPSConfig(7000, 18080, 9443, "fonu-dev-token", []int{6000, 13306})
	if out == "" {
		t.Fatal("empty config")
	}
	for _, want := range []string{
		"bindPort = 7000",
		"vhostHTTPPort = 18080",
		"vhostHTTPSPort = 9443",
		`token = "fonu-dev-token"`,
		`allowPorts = ["6000", "13306"]`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}
