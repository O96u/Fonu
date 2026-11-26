package secret

import (
	"context"
	"path/filepath"
	"testing"
 "os"
 "encoding/json"

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

func TestLegacyRotationResumeAndDistinctInstallationKeys(t *testing.T){
 ctx:=context.Background()
 migrate:=func()string{
  dir:=t.TempDir();conn,err:=db.Open(filepath.Join(dir,"fonu.db"));if err!=nil{t.Fatal(err)};defer conn.Close()
  if err:=db.Migrate(ctx,conn,filepath.Join("..","..","migrations"));err!=nil{t.Fatal(err)}
  old,err:=NewBox(LegacyDefaultKey);if err!=nil{t.Fatal(err)}
  enc,err:=old.Encrypt("notify-secret");if err!=nil{t.Fatal(err)}
  if _,err:=conn.Exec("INSERT INTO settings(key,value) VALUES('notify_telegram_token',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value",enc);err!=nil{t.Fatal(err)}
  accounts,_:=json.Marshal([]map[string]string{{"api_key_enc":enc,"api_secret_enc":enc}})
  if _,err:=conn.Exec("INSERT INTO settings(key,value) VALUES('dnshe_accounts',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value",string(accounts));err!=nil{t.Fatal(err)}
  key,err:=RotateLegacyKey(ctx,conn,dir,LegacyDefaultKey);if err!=nil{t.Fatal(err)}
  if err:=writeKeyFile(dir,rotationFile,key);err!=nil{t.Fatal(err)}
  resumed,err:=RotateLegacyKey(ctx,conn,dir,LegacyDefaultKey);if err!=nil||resumed!=key{t.Fatalf("resume: %v",err)}
  box,err:=NewBox(key);if err!=nil{t.Fatal(err)}
  var saved string
  if err:=conn.QueryRow("SELECT value FROM settings WHERE key='notify_telegram_token'").Scan(&saved);err!=nil{t.Fatal(err)}
  if plain,err:=box.Decrypt(saved);err!=nil||plain!="notify-secret"{t.Fatal("credential lost")}
  if err:=conn.QueryRow("SELECT value FROM settings WHERE key='dnshe_accounts'").Scan(&saved);err!=nil{t.Fatal(err)}
  var rows []map[string]string;if err:=json.Unmarshal([]byte(saved),&rows);err!=nil{t.Fatal(err)}
  if plain,err:=box.Decrypt(rows[0]["api_key_enc"]);err!=nil||plain!="notify-secret"{t.Fatal("account credential lost")}
  info,err:=os.Stat(filepath.Join(dir,KeyFileName));if err!=nil||info.Mode().Perm()!=0600{t.Fatal("key file permissions")}
  return key
 }
 if a,b:=migrate(),migrate();a==b||a==LegacyDefaultKey||b==LegacyDefaultKey{t.Fatal("keys not unique")}
}
