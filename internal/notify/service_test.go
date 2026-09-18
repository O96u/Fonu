package notify

import (
	"context"
	"encoding/json"
	"testing"
)

func TestTestInputDecodesSecretFields(t *testing.T) {
	raw := `{
		"type": "telegram",
		"telegram": {"chat_id": "743955235", "proxy_url": "http://127.0.0.1:7890", "has_bot_token": false},
		"telegram_token": "123456:ABC-DEF"
	}`
	var in TestInput
	if err := json.Unmarshal([]byte(raw), &in); err != nil {
		t.Fatal(err)
	}
	if in.Type != NotifyTypeTelegram {
		t.Fatalf("unexpected type: %s", in.Type)
	}
	if in.TelegramToken != "123456:ABC-DEF" {
		t.Fatalf("telegram token not decoded: %q", in.TelegramToken)
	}
	if in.Telegram.ChatID != "743955235" {
		t.Fatalf("chat id not decoded: %q", in.Telegram.ChatID)
	}
}

func TestResolveTestRuntimeUsesPlainToken(t *testing.T) {
	svc := &Service{}
	runtime, err := svc.resolveTestRuntime(context.Background(), TestInput{
		Config: Config{
			Type: NotifyTypeTelegram,
			Telegram: TelegramConfig{
				ChatID: "1",
			},
		},
		TelegramToken: "plain-token",
	})
	if err != nil {
		t.Fatal(err)
	}
	if runtime.TelegramToken != "plain-token" {
		t.Fatalf("expected plain-token, got %q", runtime.TelegramToken)
	}
}
