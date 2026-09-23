package notify

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/fonu/fonu/internal/db"
	"github.com/fonu/fonu/internal/secret"
	"github.com/fonu/fonu/internal/settings"
)

func TestLoadRuntimeIgnoresCorruptSMTPDecryptsTelegram(t *testing.T) {
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
	if err := settingsStore.Set(context.Background(), settings.KeyNotifyType, "telegram"); err != nil {
		t.Fatal(err)
	}
	if err := settingsStore.Set(context.Background(), settings.KeyNotifySMTPPassword, "not-valid-ciphertext"); err != nil {
		t.Fatal(err)
	}

	tokEnc, err := box.Encrypt("123456:ABC-DEF")
	if err != nil {
		t.Fatal(err)
	}
	if err := settingsStore.Set(context.Background(), settings.KeyNotifyTelegramToken, tokEnc); err != nil {
		t.Fatal(err)
	}
	telegramJSON := `{"chat_id":"1","proxy_url":"","has_bot_token":true}`
	if err := settingsStore.Set(context.Background(), settings.KeyNotifyTelegramJSON, telegramJSON); err != nil {
		t.Fatal(err)
	}

	store := NewStore(settingsStore, box)
	runtime, err := store.LoadRuntime(context.Background())
	if err != nil {
		t.Fatalf("LoadRuntime: %v", err)
	}
	if runtime.Type != NotifyTypeTelegram {
		t.Fatalf("type=%q want telegram", runtime.Type)
	}
	if runtime.TelegramToken != "123456:ABC-DEF" {
		t.Fatalf("telegram token=%q", runtime.TelegramToken)
	}
}
