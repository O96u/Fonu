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
