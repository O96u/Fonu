package notify

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/fonu/fonu/internal/db"
	"github.com/fonu/fonu/internal/settings"
)

func TestDeliveryHistoryPersistsAndIsBounded(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "fonu.db")
	conn, err := db.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Migrate(ctx, conn, filepath.Join("..", "..", "migrations")); err != nil {
		conn.Close()
		t.Fatal(err)
	}
	s := New(settings.NewStore(conn), nil, nil, conn)
	s.record("test", "telegram", "sent", "通道已确认接收")
	for i := 0; i < deliveryHistoryLimit; i++ {
		if err := insertDelivery(ctx, conn, Delivery{At: "2026-11-28T00:00:00Z", Event: "test", Channel: "custom", State: "failed", Detail: fmt.Sprint(i)}); err != nil {
			conn.Close()
			t.Fatal(err)
		}
	}
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}
	conn, err = db.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := db.Migrate(ctx, conn, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}
	restarted := New(settings.NewStore(conn), nil, nil, conn)
	rows, err := restarted.HistoryContext(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != deliveryHistoryLimit || rows[0].Detail != "999" || rows[len(rows)-1].Detail != "0" {
		t.Fatalf("history after restart: len=%d", len(rows))
	}
	if _, err := conn.Exec("DROP TABLE notify_delivery_history"); err != nil {
		t.Fatal(err)
	}
	if _, err := restarted.HistoryContext(ctx); err == nil {
		t.Fatal("read failure must not be reported as an empty history")
	}
}
