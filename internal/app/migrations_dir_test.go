package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveMigrationsDirFromEnv(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("FONU_MIGRATIONS_DIR", dir)
	got, err := ResolveMigrationsDir()
	if err != nil {
		t.Fatalf("ResolveMigrationsDir: %v", err)
	}
	if got != dir {
		t.Fatalf("got %q want %q", got, dir)
	}
}

func TestResolveMigrationsDirFromEnvMissing(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "missing")
	t.Setenv("FONU_MIGRATIONS_DIR", dir)
	_, err := ResolveMigrationsDir()
	if err == nil {
		t.Fatal("expected error for missing FONU_MIGRATIONS_DIR")
	}
}

func TestResolveMigrationsDirRelative(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.Mkdir("migrations", 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FONU_MIGRATIONS_DIR", "")
	got, err := ResolveMigrationsDir()
	if err != nil {
		t.Fatalf("ResolveMigrationsDir: %v", err)
	}
	if got != "migrations" {
		t.Fatalf("got %q", got)
	}
}
