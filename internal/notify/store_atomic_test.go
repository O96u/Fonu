package notify

import (
 "context"
 "path/filepath"
 "testing"
 "github.com/fonu/fonu/internal/db"
 "github.com/fonu/fonu/internal/secret"
 "github.com/fonu/fonu/internal/settings"
)
func TestNotifyDisableAndAtomicSecretFailure(t *testing.T){
 ctx:=context.Background()
 conn,err:=db.Open(filepath.Join(t.TempDir(),"fonu.db"));if err!=nil{t.Fatal(err)};defer conn.Close()
 if err:=db.Migrate(ctx,conn,filepath.Join("..","..","migrations"));err!=nil{t.Fatal(err)}
 st:=settings.NewStore(conn);box,err:=secret.NewBox("test-secret-key-32bytes-long!!!");if err!=nil{t.Fatal(err)}
 svc:=NewStore(st,box)
 if err:=svc.Save(ctx,SaveInput{Type:NotifyTypeWebhook,Webhook:WebhookConfig{Provider:WebhookCustom,URL:"https://example.com/hook"}});err!=nil{t.Fatal(err)}
 if err:=svc.Save(ctx,SaveInput{Type:""});err!=nil{t.Fatal(err)}
 cfg,err:=svc.Load(ctx);if err!=nil||cfg.Type!=""{t.Fatalf("notification not disabled: %v %q",err,cfg.Type)}
 if url,_:=st.Get(ctx,settings.KeyNotifyWebhookURL);url!=""{t.Fatal("legacy URL still enables notification")}
 before,err:=st.List(ctx);if err!=nil{t.Fatal(err)}
 noBox:=NewStore(st,nil)
 err=noBox.Save(ctx,SaveInput{Type:NotifyTypeTelegram,Telegram:TelegramConfig{ChatID:"42"},TelegramToken:"new-token"})
 if err==nil{t.Fatal("expected encryption failure")}
 after,err:=st.List(ctx);if err!=nil{t.Fatal(err)}
 if len(before)!=len(after){t.Fatal("partial settings write")}
 for key,v:=range before{if after[key]!=v{t.Fatalf("partial settings write: %s",key)}}
}
