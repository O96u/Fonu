package frp

import "testing"

func TestValidateTCPProxies(t *testing.T) {
	err := validateTCPProxies([]TCPProxy{
		{ID: "1", Name: "ssh", LocalIP: "127.0.0.1", LocalPort: 22, RemotePort: 6000, Enabled: true},
		{ID: "2", Name: "db", LocalIP: "127.0.0.1", LocalPort: 3306, RemotePort: 6000, Enabled: true},
	})
	if err == nil {
		t.Fatal("expected duplicate remote port error")
	}
}

func TestAssignTCPRemotePorts(t *testing.T) {
	proxies, err := prepareTCPProxies([]TCPProxy{
		{ID: "1", Name: "ssh", LocalIP: "127.0.0.1", LocalPort: 22, RemotePort: 0, Enabled: true},
		{ID: "2", Name: "db", LocalIP: "127.0.0.1", LocalPort: 3306, RemotePort: 0, Enabled: true},
		{ID: "3", Name: "web", LocalIP: "127.0.0.1", LocalPort: 8080, RemotePort: 7000, Enabled: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	ports := map[int]bool{}
	for _, p := range proxies {
		if p.RemotePort < tcpRemotePortMin || p.RemotePort > 65535 {
			t.Fatalf("invalid port %d", p.RemotePort)
		}
		if ports[p.RemotePort] {
			t.Fatalf("duplicate port %d", p.RemotePort)
		}
		ports[p.RemotePort] = true
	}
	if !ports[7000] {
		t.Fatal("expected fixed port 7000 to remain")
	}
}

func TestValidateSaveInputAllowsConnectionOnly(t *testing.T) {
	err := validateSaveInput(SaveInput{
		Enabled:    true,
		ServerAddr: "vps.example.com",
		ServerPort: 7000,
		AuthToken:  "token",
	}, false)
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidateTCPProxiesAllowsLowRemotePort(t *testing.T) {
	err := validateTCPProxies([]TCPProxy{
		{ID: "1", Name: "http", LocalIP: "127.0.0.1", LocalPort: 80, RemotePort: 81, Enabled: true},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidateSaveInputAllowsTCPOnly(t *testing.T) {
	err := validateSaveInput(SaveInput{
		Enabled:    true,
		ServerAddr: "vps.example.com",
		ServerPort: 7000,
		AuthToken:  "token",
		TCPProxies: []TCPProxy{
			{ID: "1", Name: "ssh", LocalIP: "127.0.0.1", LocalPort: 22, RemotePort: 6000, Enabled: true},
		},
	}, false)
	if err != nil {
		t.Fatal(err)
	}
}
