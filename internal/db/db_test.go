package db

import (
	"context"
	"path/filepath"
	"testing"
)

func TestMigrate(t *testing.T) {
	dir := t.TempDir()
	conn, err := Open(filepath.Join(dir, "fonu.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer conn.Close()

	migrationsDir := filepath.Join("..", "..", "migrations")
	if err := Migrate(context.Background(), conn, migrationsDir); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	if err := Migrate(context.Background(), conn, migrationsDir); err != nil {
		t.Fatalf("second migrate failed: %v", err)
	}
}
