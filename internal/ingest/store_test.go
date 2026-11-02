package ingest

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/fonu/fonu/internal/db"
)

func TestStoreInsertAndState(t *testing.T) {
	dir := t.TempDir()
	conn, err := db.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := db.Migrate(context.Background(), conn, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}
	store := NewStore(conn)
	ctx := context.Background()
	now := time.Now().UTC()
	rows := []FactRow{{
		TS: now, ClientIP: "1.2.3.4", Host: "a.example.com", Service: "web",
		Method: "GET", Path: "/", Status: 200, Kind: "access", GeoPending: true,
	}}
	if err := store.InsertBatch(ctx, rows); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveState(ctx, "/tmp/access.log", IngestState{Offset: 123, FileSize: 456}); err != nil {
		t.Fatal(err)
	}
	st, err := store.LoadState(ctx, "/tmp/access.log")
	if err != nil || st.Offset != 123 {
		t.Fatalf("state=%+v err=%v", st, err)
	}
}
