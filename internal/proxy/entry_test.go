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
