package proxy

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/fonu/fonu/internal/db"
)

func TestCreateEntryRejectsDuplicateListenPort(t *testing.T) {
	dir := t.TempDir()
	conn, err := db.Open(filepath.Join(dir, "fonu.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := db.Migrate(context.Background(), conn, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}

	store := NewStore(conn)
	ctx := context.Background()
	if _, err := store.CreateEntry(ctx, EntryCreateInput{
		Name:         "a",
		ListenPort:   8007,
		ListenIPv4:   true,
		HTTPSEnabled: true,
	}); err != nil {
		t.Fatal(err)
	}
	_, err = store.CreateEntry(ctx, EntryCreateInput{
		Name:         "b",
		ListenPort:   8007,
		ListenIPv4:   true,
		HTTPSEnabled: true,
	})
	if err == nil {
		t.Fatal("expected duplicate port error")
	}
}

func TestUpdateEntrySyncsRuleAndHostListenPort(t *testing.T) {
	dir := t.TempDir()
	conn, err := db.Open(filepath.Join(dir, "fonu.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := db.Migrate(context.Background(), conn, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}

	store := NewStore(conn)
	ctx := context.Background()
	entry, err := store.CreateEntry(ctx, EntryCreateInput{
		ListenPort:   4243,
		ListenIPv4:   true,
		HTTPSEnabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	entryID := entry.ID
	rule, err := store.Create(ctx, CreateInput{
		EntryID:      &entryID,
		Upstream:     "http://127.0.0.1:5173",
		Hosts:        []string{"a.example.com"},
		Enabled:      true,
		ListenPort:   4243,
		ListenIPv4:   true,
		HTTPSEnabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	newPort := 4433
	if _, err := store.UpdateEntry(ctx, entry.ID, EntryUpdateInput{ListenPort: &newPort}); err != nil {
		t.Fatal(err)
	}

	updated, err := store.Get(ctx, rule.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.ListenPort != newPort {
		t.Fatalf("rule listen_port = %d, want %d", updated.ListenPort, newPort)
	}
	if len(updated.Hosts) != 1 {
		t.Fatalf("hosts len = %d", len(updated.Hosts))
	}
	if updated.Hosts[0].ListenPort == nil || *updated.Hosts[0].ListenPort != newPort {
		got := 0
		if updated.Hosts[0].ListenPort != nil {
			got = *updated.Hosts[0].ListenPort
		}
		t.Fatalf("host listen_port = %d, want %d", got, newPort)
	}
}

func TestUpdateEntryRepairsStaleHostListenPort(t *testing.T) {
	dir := t.TempDir()
	conn, err := db.Open(filepath.Join(dir, "fonu.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := db.Migrate(context.Background(), conn, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}

	store := NewStore(conn)
	ctx := context.Background()
	entry, err := store.CreateEntry(ctx, EntryCreateInput{
		ListenPort:   4435,
		ListenIPv4:   true,
		HTTPSEnabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	entryID := entry.ID
	rule, err := store.Create(ctx, CreateInput{
		EntryID:      &entryID,
		Upstream:     "http://127.0.0.1:5173",
		Hosts:        []string{"stale.example.com"},
		Enabled:      true,
		ListenPort:   4435,
		ListenIPv4:   true,
		HTTPSEnabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.ExecContext(ctx, `UPDATE proxy_hosts SET listen_port = 4243 WHERE rule_id = ?`, rule.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.ExecContext(ctx, `UPDATE proxy_rules SET listen_port = 4435 WHERE id = ?`, rule.ID); err != nil {
		t.Fatal(err)
	}

	newPort := 8018
	if _, err := store.UpdateEntry(ctx, entry.ID, EntryUpdateInput{ListenPort: &newPort}); err != nil {
		t.Fatal(err)
	}

	updated, err := store.Get(ctx, rule.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.ListenPort != newPort {
		t.Fatalf("rule listen_port = %d, want %d", updated.ListenPort, newPort)
	}
	if len(updated.Hosts) != 1 || updated.Hosts[0].ListenPort == nil || *updated.Hosts[0].ListenPort != newPort {
		t.Fatalf("host listen_port not repaired: %+v", updated.Hosts)
	}
}
