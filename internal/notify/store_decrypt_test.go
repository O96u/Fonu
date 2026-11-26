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

func TestCorruptActiveCredentialCanBeReplaced(t *testing.T){
 ctx:=context.Background()
 conn,err:=db.Open(filepath.Join(t.TempDir(),"test.db"));if err!=nil{t.Fatal(err)};defer conn.Close()
 if err:=db.Migrate(ctx,conn,filepath.Join("..","..","migrations"));err!=nil{t.Fatal(err)}
 st:=settings.NewStore(conn);box,err:=secret.NewBox("test-key");if err!=nil{t.Fatal(err)}
 if err:=st.SetMany(ctx,map[string]string{settings.KeyNotifyType:"telegram",settings.KeyNotifyTelegramToken:"corrupt",settings.KeyNotifyTelegramJSON:"{\"chat_id\":\"1\",\"has_bot_token\":true}"});err!=nil{t.Fatal(err)}
 s:=NewStore(st,box)
 if _,err:=s.LoadRuntime(ctx);err==nil{t.Fatal("corrupt active token ignored")}
 if err:=s.Save(ctx,SaveInput{Type:NotifyTypeTelegram,Telegram:TelegramConfig{ChatID:"1"},TelegramToken:"replacement"});err!=nil{t.Fatal(err)}
 runtime,err:=s.LoadRuntime(ctx);if err!=nil||runtime.TelegramToken!="replacement"{t.Fatalf("repair failed: %v",err)}
}
