package notify

import (
	"context"
	"encoding/json"
	"testing"
 "net/http"
 "net/http/httptest"
 "path/filepath"
 "sync/atomic"
 "github.com/fonu/fonu/internal/db"
 "github.com/fonu/fonu/internal/settings"
 "github.com/fonu/fonu/internal/secret"
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

func TestEveryEventRespectsSavedSwitchAndRecordsDelivery(t *testing.T){
 ctx:=context.Background()
 conn,err:=db.Open(filepath.Join(t.TempDir(),"test.db"));if err!=nil{t.Fatal(err)};defer conn.Close()
 if err:=db.Migrate(ctx,conn,filepath.Join("..","..","migrations"));err!=nil{t.Fatal(err)}
 st:=settings.NewStore(conn);box,err:=secret.NewBox("test-key");if err!=nil{t.Fatal(err)}
 var requests atomic.Int32
 endpoint:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){requests.Add(1);w.WriteHeader(200)}));defer endpoint.Close()
 s:=New(st,box,nil)
 all:=map[string]string{EventDDNSIPChange:settings.KeyNotifyOnDDNSIPChange,EventDDNSFailure:settings.KeyNotifyOnDDNSFailure,EventCertExpiry:settings.KeyNotifyOnCertExpiry,EventCertRenewSuccess:settings.KeyNotifyOnCertRenewSuccess,EventCertRenewFailure:settings.KeyNotifyOnCertRenewFailure,EventIPFrequentAccess:settings.KeyNotifyOnIPFrequentAccess,EventScan:settings.KeyNotifyOnScan,EventCC:settings.KeyNotifyOnCC,EventLoginFailure:settings.KeyNotifyOnLoginFailure,EventNginxReloadFailure:settings.KeyNotifyOnNginxReloadFailure}
 raw,_:=json.Marshal(WebhookConfig{Provider:WebhookCustom,URL:endpoint.URL})
 if err:=st.SetMany(ctx,map[string]string{settings.KeyNotifyType:"webhook",settings.KeyNotifyWebhookJSON:string(raw)});err!=nil{t.Fatal(err)}
 for event,key:=range all{
  if err:=st.Set(ctx,key,"false");err!=nil{t.Fatal(err)}
  before:=requests.Load();s.Alert(ctx,event,"test","test")
  if requests.Load()!=before||s.History()[0].State!="skipped"{t.Fatalf("disabled event %s sent",event)}
  if err:=st.Set(ctx,key,"true");err!=nil{t.Fatal(err)}
  s.Alert(ctx,event,"test","test")
  if requests.Load()!=before+1||s.History()[0].State!="sent"{t.Fatalf("enabled event %s not sent: %+v",event,s.History()[0])}
 }
 if err:=st.Set(ctx,settings.KeyNotifyWebhookURL,endpoint.URL);err!=nil{t.Fatal(err)}
 if err:=st.Set(ctx,settings.KeyNotifyType,"");err!=nil{t.Fatal(err)}
 before:=requests.Load();s.Alert(ctx,EventCC,"test","test")
 if requests.Load()!=before||s.History()[0].State!="skipped"{t.Fatal("legacy URL re-enabled disabled channel")}
}
