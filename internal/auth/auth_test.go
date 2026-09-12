package auth

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/fonu/fonu/internal/db"
)

func TestEnsureDefaultAdminCreatesOnce(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "fonu.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer conn.Close()

	if err := db.Migrate(context.Background(), conn, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	svc := New(conn)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	if err := svc.EnsureDefaultAdmin(context.Background(), logger); err != nil {
		t.Fatalf("first bootstrap: %v", err)
	}
	initialized, err := svc.IsInitialized(context.Background())
	if err != nil || !initialized {
		t.Fatalf("expected initialized admin, got initialized=%v err=%v", initialized, err)
	}

	if err := svc.EnsureDefaultAdmin(context.Background(), logger); err != nil {
		t.Fatalf("second bootstrap: %v", err)
	}
	if _, err := svc.Login(context.Background(), "admin", "wrong-password"); err != ErrUnauthorized {
		t.Fatalf("expected unauthorized for wrong password, got %v", err)
	}
}
