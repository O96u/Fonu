package logstore

import "testing"

func TestParseRealNginxAccessLine(t *testing.T) {
	line := "2026-09-12T19:55:18+08:00 127.0.0.1 GET / 200 0.000 127.0.0.1 [::1]:5173"
	entry, ok := parseAccess(line)
	if !ok {
		t.Fatal("parse failed")
	}
	if entry.Status != 200 {
		t.Fatalf("status %d", entry.Status)
	}
}

func TestParseAccessWithBytes(t *testing.T) {
	line := "2026-09-12T19:55:18+08:00 speedtest.example.com 6893 GET /api 200 0.010 1.2.3.4 192.168.1.1:8080 512 4096"
	entry, ok := parseAccess(line)
	if !ok {
		t.Fatal("parse failed")
	}
	if entry.ServerPort != 6893 {
		t.Fatalf("server_port=%d", entry.ServerPort)
	}
	if entry.RequestLength != 512 || entry.BytesSent != 4096 {
		t.Fatalf("bytes upload=%d download=%d", entry.RequestLength, entry.BytesSent)
	}
}

func TestAccessEndpointFilterMatchesPort(t *testing.T) {
	line := "2026-09-12T19:55:18+08:00 speedtest.example.com 6893 GET /api 200 0.010 1.2.3.4 192.168.1.1:8080"
	other := "2026-09-12T19:55:19+08:00 speedtest.example.com 8443 GET / 404 0.010 1.2.3.4 192.168.1.2:8080"
	filter := AccessEndpointFilter([]AccessEndpoint{{Host: "speedtest.example.com", Port: 6893}})
	if filter == nil || !filter(line) || filter(other) {
		t.Fatal("endpoint filter mismatch")
	}
}

func TestAccessHostFilter(t *testing.T) {
	line := "2026-09-12T19:55:18+08:00 speedtest.example.com GET /api 200 0.010 1.2.3.4 192.168.1.1:8080"
	other := "2026-09-12T19:55:19+08:00 other.example.com GET / 404 0.010 1.2.3.4 192.168.1.2:8080"
	filter := AccessHostFilter([]string{"speedtest.example.com"})
	if filter == nil || !filter(line) || filter(other) {
		t.Fatal("host filter mismatch")
	}
}
