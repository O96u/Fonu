package settings

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/fonu/fonu/internal/db"
)

func TestStoreNowUsesConfiguredTimezone(t *testing.T) {
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
	if err := store.Set(ctx, KeyTimezone, "UTC"); err != nil {
		t.Fatal(err)
	}

	now := store.Now(ctx)
	if now.Location().String() != "UTC" {
		t.Fatalf("now location=%s, want UTC", now.Location().String())
	}
	if time.Now().UTC().Sub(now) > time.Minute || now.Sub(time.Now().UTC()) > time.Minute {
		t.Fatalf("now=%s is too far from current UTC time", now.Format(time.RFC3339))
	}
}
