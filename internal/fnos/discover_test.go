package fnos

import (
	"net"
	"testing"
)

func TestHostsOnStaysInsideLocal24(t *testing.T) {
	ip := net.ParseIP("10.147.20.8")
	hosts := hostsOn(ip, net.CIDRMask(16, 32))
	if len(hosts) < 200 || len(hosts) > 254 {
		t.Fatalf("got %d hosts", len(hosts))
	}
	for _, host := range hosts {
		if host.Equal(ip) {
			t.Fatal("included self")
		}
		if host.To4()[2] != 20 {
			t.Fatalf("left the local /24: %s", host)
		}
	}
}

func TestLooksLikeFNOS(t *testing.T) {
	if !looksLikeFNOS([]byte("<title>fnOS</title>")) {
		t.Fatal("missed fnOS")
	}
	if !looksLikeFNOS([]byte("飞牛")) {
		t.Fatal("missed 飞牛")
	}
	if looksLikeFNOS([]byte("<title>router</title>")) {
		t.Fatal("ordinary page matched")
	}
}
