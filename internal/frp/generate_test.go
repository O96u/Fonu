package frp

import (
	"strings"
	"testing"

	"github.com/fonu/fonu/internal/config"
)

func TestGenerateNginxGateway(t *testing.T) {
	cfg := config.Config{
		DataDir:               "/data",
		NginxDefaultHTTPPort:  18080,
		NginxDefaultHTTPSPort: 9443,
		FrpPIDFile:            "/data/frp/frpc.pid",
	}
	frpCfg := Config{
		ServerAddr:    "vps.example.com",
		ServerPort:    7000,
		TLSEnabled:    true,
		CustomDomains: []string{"nas.example.com", "*.example.com"},
	}
	out, err := Generate(cfg, frpCfg, "secret-token", 18080, 9443)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `serverAddr = "vps.example.com"`) {
		t.Fatalf("missing server addr: %s", out)
	}
	if !strings.Contains(out, `localPort = 18080`) {
		t.Fatalf("missing http local port: %s", out)
	}
	if !strings.Contains(out, `type = "https2https"`) {
		t.Fatalf("missing https plugin: %s", out)
	}
	if !strings.Contains(out, `localAddr = "127.0.0.1:9443"`) {
		t.Fatalf("missing https local addr: %s", out)
	}
	if !strings.Contains(out, `transport.tls.enable = true`) {
		t.Fatalf("missing tls enable: %s", out)
	}
}

func TestGenerateDisablesTLSByDefault(t *testing.T) {
	cfg := config.Config{DataDir: "/data", FrpPIDFile: "/data/frp/frpc.pid"}
	frpCfg := Config{
		ServerAddr:    "vps.example.com",
		ServerPort:    7000,
		CustomDomains: []string{"nas.example.com"},
	}
	out, err := Generate(cfg, frpCfg, "secret-token", 18080, 9443)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `transport.tls.enable = false`) {
		t.Fatalf("expected tls disabled: %s", out)
	}
}

func TestGenerateTCPOnly(t *testing.T) {
	cfg := config.Config{DataDir: "/data", FrpPIDFile: "/data/frp/frpc.pid"}
	frpCfg := Config{
		ServerAddr: "vps.example.com",
		ServerPort: 7000,
		TCPProxies: []TCPProxy{
			{ID: "1", Name: "ssh", LocalIP: "127.0.0.1", LocalPort: 22, RemotePort: 6000, Enabled: true},
		},
	}
	out, err := Generate(cfg, frpCfg, "secret-token", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, `fonu-nginx-http`) {
		t.Fatalf("expected no http gateway: %s", out)
	}
	for _, want := range []string{
		`name = "fonu-tcp-ssh"`,
		`type = "tcp"`,
		`localIP = "127.0.0.1"`,
		`localPort = 22`,
		`remotePort = 6000`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}

func TestGenerateMixedWebAndTCP(t *testing.T) {
	cfg := config.Config{DataDir: "/data", FrpPIDFile: "/data/frp/frpc.pid"}
	frpCfg := Config{
		ServerAddr:    "vps.example.com",
		ServerPort:    7000,
		CustomDomains: []string{"nas.example.com"},
		TCPProxies: []TCPProxy{
			{ID: "1", Name: "mysql", LocalIP: "127.0.0.1", LocalPort: 3306, RemotePort: 13306, Enabled: true},
		},
	}
	out, err := Generate(cfg, frpCfg, "secret-token", 18080, 9443)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `fonu-nginx-http`) {
		t.Fatalf("missing http gateway: %s", out)
	}
	if !strings.Contains(out, `name = "fonu-tcp-mysql"`) {
		t.Fatalf("missing tcp proxy: %s", out)
	}
}
