package secret_test

import (
 "context"
 "os"
 "path/filepath"
 "strings"
 "testing"

 "github.com/fonu/fonu/internal/certificate"
 "github.com/fonu/fonu/internal/db"
 "github.com/fonu/fonu/internal/notify"
 "github.com/fonu/fonu/internal/secret"
 "github.com/fonu/fonu/internal/settings"
)

// main e10f5be has migrations 001–023; their Git blob hashes match v2.
func TestV1DatabaseUpgradePreservesRecordsAndCredentials(t *testing.T){
 ctx:=context.Background()
 dir:=t.TempDir();legacy:=filepath.Join(dir,"legacy-migrations")
 if err:=os.Mkdir(legacy,0700);err!=nil{t.Fatal(err)}
 current:=filepath.Join("..","..","migrations")
 entries,err:=os.ReadDir(current);if err!=nil{t.Fatal(err)}
 for _,e:=range entries{if e.IsDir()||!strings.HasSuffix(e.Name(),".sql")||e.Name()[:3]>"023"{continue};raw,err:=os.ReadFile(filepath.Join(current,e.Name()));if err!=nil{t.Fatal(err)};if err:=os.WriteFile(filepath.Join(legacy,e.Name()),raw,0600);err!=nil{t.Fatal(err)}}
 path:=filepath.Join(dir,"fonu.db");conn,err:=db.Open(path);if err!=nil{t.Fatal(err)};defer conn.Close()
 if err:=db.Migrate(ctx,conn,legacy);err!=nil{t.Fatal(err)}
 old,err:=secret.NewBox(secret.LegacyDefaultKey);if err!=nil{t.Fatal(err)}
 enc,err:=old.Encrypt("fixture-secret");if err!=nil{t.Fatal(err)}
 for _,q:=range []struct{sql string;args []any}{
  {"INSERT INTO admins(username,password_hash) VALUES('fixture-admin','fixture-hash')",nil},
  {"INSERT INTO proxy_rules(upstream,listen_port,https_enabled,http_redirect) VALUES('http://127.0.0.1:8080',8080,0,0)",nil},
  {"INSERT INTO proxy_hosts(rule_id,hostname,listen_port) SELECT id,'app.example.com',listen_port FROM proxy_rules",nil},
  {"INSERT INTO ddns_configs(provider,root_domain,record_name,api_token_enc) VALUES('cloudflare','example.com','www',?)",[]any{enc}},
  {"INSERT INTO certificates(domain,acme_ca,export_json) VALUES('example.com','imported',?)",[]any{`{"ssh":{"host":"example.com","password":"fixture-ssh"}}`}},
  {"INSERT INTO settings(key,value) VALUES('notify_type','telegram') ON CONFLICT(key) DO UPDATE SET value=excluded.value",nil},
  {"INSERT INTO settings(key,value) VALUES('notify_telegram_token',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value",[]any{enc}},
  {"INSERT INTO settings(key,value) VALUES('frp_auth_token',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value",[]any{enc}},
  {"INSERT INTO access_facts(ts,client_ip,bytes_sent) VALUES('2026-01-01T00:00:00Z','8.8.8.8',123)",nil},
 }{if _,err:=conn.ExecContext(ctx,q.sql,q.args...);err!=nil{t.Fatal(err)}}
 key,err:=secret.ResolveKey(dir,"");if err!=nil||key!=secret.LegacyDefaultKey{t.Fatalf("legacy resolve: %v",err)}
 if err:=db.Migrate(ctx,conn,current);err!=nil{t.Fatal(err)}
 key,err=secret.RotateLegacyKey(ctx,conn,dir,key);if err!=nil||key==secret.LegacyDefaultKey{t.Fatalf("rotation: %v",err)}
 box,err:=secret.NewBox(key);if err!=nil{t.Fatal(err)}
 certs:=certificate.NewStore(conn,box);if err:=certs.MigrateExportSecrets(ctx);err!=nil{t.Fatal(err)}
 if err:=db.Migrate(ctx,conn,current);err!=nil{t.Fatal(err)}
 for _,table:=range []string{"admins","proxy_rules","proxy_hosts","ddns_configs","certificates","access_facts","access_minute_ip"}{
  var count int;if err:=conn.QueryRow("SELECT count(*) FROM "+table).Scan(&count);err!=nil||count!=1{t.Fatalf("%s count=%d err=%v",table,count,err)}
 }
 for _,query:=range []string{"SELECT api_token_enc FROM ddns_configs","SELECT value FROM settings WHERE key='frp_auth_token'"}{
  var v string;if err:=conn.QueryRow(query).Scan(&v);err!=nil{t.Fatal(err)}
  if plain,err:=box.Decrypt(v);err!=nil||plain!="fixture-secret"{t.Fatal("encrypted credential lost")}
 }
 runtime,err:=notify.NewStore(settings.NewStore(conn),box).LoadRuntime(ctx);if err!=nil||runtime.TelegramToken!="fixture-secret"{t.Fatal("notify credential lost")}
 rec,err:=certs.GetByDomain(ctx,"example.com");if err!=nil||rec.AutoRenew||rec.Export.SSH.Password!="fixture-ssh"{t.Fatalf("certificate migration: %v",err)}
 var raw string;if err:=conn.QueryRow("SELECT export_json FROM certificates").Scan(&raw);err!=nil||strings.Contains(raw,"fixture-ssh"){t.Fatal("SSH credential not encrypted")}
}
