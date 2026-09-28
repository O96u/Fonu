package frp

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/fonu/fonu/internal/db"
	"github.com/fonu/fonu/internal/secret"
	"github.com/fonu/fonu/internal/settings"
)

func TestSaveInputFromConfigPreservesTCPProxies(t *testing.T) {
	cfg := Config{
		Enabled:    true,
		ServerAddr: "frp.example.com",
		ServerPort: 7000,
		TCPProxies: []TCPProxy{
			{ID: "t1", Name: "ssh", LocalIP: "127.0.0.1", LocalPort: 22, RemotePort: 60022, Enabled: true},
		},
	}
	in := saveInputFromConfig(cfg)
	if len(in.TCPProxies) != 1 || in.TCPProxies[0].Name != "ssh" {
		t.Fatalf("expected tcp proxies preserved, got %+v", in.TCPProxies)
	}
}

func TestStoreSaveWithoutTCPProxiesClearsPersisted(t *testing.T) {
	dir := t.TempDir()
	conn, err := db.Open(filepath.Join(dir, "fonu.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := db.Migrate(context.Background(), conn, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}

	settingsStore := settings.NewStore(conn)
	box, err := secret.NewBox("test-secret-key-32bytes-long!!!")
	if err != nil {
		t.Fatal(err)
	}
	store := NewStore(settingsStore, box)

	proxies := []TCPProxy{
		{ID: "t1", Name: "ssh", LocalIP: "127.0.0.1", LocalPort: 22, RemotePort: 60022, Enabled: true},
	}
	if err := store.Save(context.Background(), SaveInput{
		Enabled:    true,
		ServerAddr: "frp.example.com",
		ServerPort: 7000,
		AuthToken:  "secret",
		TCPProxies: proxies,
	}, false); err != nil {
		t.Fatal(err)
	}

	// Simulates the old Bootstrap bug: Apply/Save without tcp_proxies wipes storage.
	if err := store.Save(context.Background(), SaveInput{
		Enabled:       true,
		ServerAddr:    "frp.example.com",
		ServerPort:    7000,
		AuthToken:     maskedToken,
		CustomDomains: nil,
	}, true); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.TCPProxies) != 0 {
		t.Fatalf("expected tcp proxies cleared when omitted from save, got %+v", loaded.TCPProxies)
	}

	if err := store.Save(context.Background(), SaveInput{
		Enabled:       true,
		ServerAddr:    "frp.example.com",
		ServerPort:    7000,
		AuthToken:     maskedToken,
		CustomDomains: nil,
		TCPProxies:    proxies,
	}, true); err != nil {
		t.Fatal(err)
	}
	loaded, err = store.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.TCPProxies) != 1 || loaded.TCPProxies[0].Name != "ssh" {
		t.Fatalf("expected tcp proxy restored, got %+v", loaded.TCPProxies)
	}
}
