package api

import(
 "testing"
 "context"
 "path/filepath"
 "net/http"
 "net/http/httptest"
 "strings"
 "log/slog"
 "github.com/fonu/fonu/internal/db"
 "github.com/fonu/fonu/internal/secret"
 "github.com/fonu/fonu/internal/settings"
 "github.com/fonu/fonu/internal/notify"
)
func TestSettingsResponseMasksSecrets(t *testing.T){
 values:=map[string]string{settings.KeyZeroSSLEABHmac:"private",settings.KeyFRPAuthToken:"private",settings.KeyDNSHEAccounts:"private",settings.KeyNotifySMTPPassword:"private",settings.KeyTimezone:"Asia/Shanghai",settings.KeyNotifyWebhookJSON:`{"provider":"bark","key":"private"}`}
 sanitizeSettingsResponse(values)
 if values[settings.KeyZeroSSLEABHmac]!=notify.MaskedSecret||values[settings.KeyFRPAuthToken]!=notify.MaskedSecret{t.Fatal("unmasked credential")}
 if values[settings.KeyDNSHEAccounts]!=""||values[settings.KeyNotifySMTPPassword]!=""{t.Fatal("private setting returned")}
 if values[settings.KeyTimezone]!="Asia/Shanghai"{t.Fatal("ordinary setting changed")}
}
func TestSettingsValidationBounds(t *testing.T){
 h:=&SettingsHandler{}
 for key,value:=range map[string]string{settings.KeyDDNSCheckInterval:"0",settings.KeyCertRenewThreshold:"91",settings.KeyACMEDNSPropagationTimeout:"10",settings.KeyTimezone:"Not/A_Zone",settings.KeyCustomACMEDirectoryURL:"file:///etc/passwd",settings.KeyTheme:"invalid"}{
  if h.validateSetting(key,value)==nil{t.Fatalf("accepted invalid %s",key)}
 }
 for key,value:=range map[string]string{settings.KeyDDNSCheckInterval:"5",settings.KeyCertRenewThreshold:"30",settings.KeyCustomACMEDirectoryURL:"https://acme.example.com/directory",settings.KeyTheme:"system"}{
  if err:=h.validateSetting(key,value);err!=nil{t.Fatalf("valid %s rejected: %v",key,err)}
 }
}

func TestMixedSettingsRollbackNotification(t *testing.T){
 ctx:=context.Background()
 conn,err:=db.Open(filepath.Join(t.TempDir(),"test.db"));if err!=nil{t.Fatal(err)};defer conn.Close()
 if err:=db.Migrate(ctx,conn,filepath.Join("..","..","migrations"));err!=nil{t.Fatal(err)}
 store:=settings.NewStore(conn)
 box,err:=secret.NewBox("test-key");if err!=nil{t.Fatal(err)}
 h:=NewSettingsHandler(store,nil,notify.New(store,box,slog.Default()))
 if _,err:=conn.Exec("CREATE TRIGGER fail_theme BEFORE INSERT ON settings WHEN NEW.key='theme' BEGIN SELECT RAISE(ABORT,'test failure'); END");err!=nil{t.Fatal(err)}
 // Default theme may already be present; SQLite BEFORE INSERT still runs for upsert.
 req:=httptest.NewRequest(http.MethodPut,"/api/settings",strings.NewReader("{\"theme\":\"dark\",\"notify_type\":\"\",\"notify_on_scan\":\"true\"}"))
 w:=httptest.NewRecorder();h.Put(w,req)
 if w.Code==http.StatusOK{t.Fatal("expected transaction failure")}
 value,err:=store.Get(ctx,settings.KeyNotifyOnScan)
 if err!=nil||value=="true"{t.Fatalf("notification partially committed: %q %v",value,err)}
 if _,err:=conn.Exec("DROP TRIGGER fail_theme");err!=nil{t.Fatal(err)}
 w=httptest.NewRecorder();req=httptest.NewRequest(http.MethodPut,"/api/settings",strings.NewReader("{\"theme\":\"dark\",\"notify_type\":\"\",\"notify_on_scan\":\"true\"}"))
 h.Put(w,req)
 if w.Code!=http.StatusOK{t.Fatalf("valid save failed: %d %s",w.Code,w.Body.String())}
 value,err=store.Get(ctx,settings.KeyNotifyOnScan)
 if err!=nil||value!="true"{t.Fatalf("notification missing: %q %v",value,err)}
}
