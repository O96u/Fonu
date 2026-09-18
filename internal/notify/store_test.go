package notify

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/fonu/fonu/internal/db"
	"github.com/fonu/fonu/internal/secret"
	"github.com/fonu/fonu/internal/settings"
)

func TestStoreSaveTelegramPersistsType(t *testing.T) {
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
	store := NewStore(settingsStore, box)
	if err := store.Save(context.Background(), SaveInput{
		Type: NotifyTypeTelegram,
		Telegram: TelegramConfig{
			ChatID:      "743955235",
			ProxyURL:    "http://127.0.0.1:7890",
			HasBotToken: true,
		},
		TelegramToken: "123456:ABC-DEF",
	}); err != nil {
		t.Fatal(err)
	}

	rawType, err := settingsStore.Get(context.Background(), settings.KeyNotifyType)
	if err != nil {
		t.Fatal(err)
	}
	if rawType != "telegram" {
		t.Fatalf("notify_type=%q, want telegram", rawType)
	}
	cfg, err := store.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Type != NotifyTypeTelegram || cfg.Telegram.ChatID != "743955235" {
		t.Fatalf("unexpected loaded config: %+v", cfg)
	}
}

func TestStoreMigratesLegacyWebhookURL(t *testing.T) {
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
	if err := settingsStore.Set(context.Background(), settings.KeyNotifyWebhookURL, "https://example.com/hook"); err != nil {
		t.Fatal(err)
	}

	box, err := secret.NewBox("test-secret-key-32bytes-long!!!")
	if err != nil {
		t.Fatal(err)
	}
	store := NewStore(settingsStore, box)
	cfg, err := store.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Type != NotifyTypeWebhook || cfg.Webhook.Provider != WebhookCustom || cfg.Webhook.URL != "https://example.com/hook" {
		t.Fatalf("unexpected migrated config: %+v", cfg)
	}
}
