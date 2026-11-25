package secret

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/fonu/fonu/internal/db"
)

func TestRotateLegacyKey(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	conn, err := db.Open(filepath.Join(dir, "fonu.db"))
	if err != nil { t.Fatal(err) }
	defer conn.Close()
	if err := db.Migrate(ctx, conn, filepath.Join("..", "..", "migrations")); err != nil { t.Fatal(err) }
	old, err := NewBox(LegacyDefaultKey)
	if err != nil { t.Fatal(err) }
	enc, err := old.Encrypt("credential")
	if err != nil { t.Fatal(err) }
	if _, err := conn.Exec("INSERT INTO settings(key,value) VALUES('frp_auth_token',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", enc); err != nil { t.Fatal(err) }
	key, err := RotateLegacyKey(ctx, conn, dir, LegacyDefaultKey)
	if err != nil || key == LegacyDefaultKey { t.Fatalf("rotation: %v", err) }
	var value string
	if err := conn.QueryRow("SELECT value FROM settings WHERE key='frp_auth_token'").Scan(&value); err != nil { t.Fatal(err) }
	box, err := NewBox(key); if err != nil { t.Fatal(err) }
	plain, err := box.Decrypt(value)
	if err != nil || plain != "credential" { t.Fatalf("decrypt: %v", err) }
	if _, err := old.Decrypt(value); err == nil { t.Fatal("legacy key still decrypts") }
	if err := writeKeyFile(dir, rotationFile, key); err != nil { t.Fatal(err) }
	if _, err := RotateLegacyKey(ctx, conn, dir, key); err != nil { t.Fatal(err) }
}
