package certexport

import (
	"context"
	"net"
	"testing"
	"time"

	certstore "github.com/fonu/fonu/internal/certificate"
)

func TestSOCKSProxyHonorsContext(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil { t.Fatal(err) }
	defer ln.Close()
	done := make(chan struct{})
	go func() {
		defer close(done)
		c, err := ln.Accept()
		if err != nil { return }
		defer c.Close()
		_ = c.SetDeadline(time.Now().Add(2*time.Second))
		var b [128]byte
		for { if _, err := c.Read(b[:]); err != nil { return } }
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	cfg := certstore.SSHExportConfig{ProxyEnabled:true, ProxyType:certstore.SSHProxySOCKS5, ProxyHost:"127.0.0.1", ProxyPort:ln.Addr().(*net.TCPAddr).Port}
	if conn, err := dialTCP(ctx, cfg, "example.com:22"); err == nil { conn.Close(); t.Fatal("expected cancellation") }
	select {
	case <-done:
	case <-time.After(time.Second): t.Fatal("proxy connection not closed")
	}
}
