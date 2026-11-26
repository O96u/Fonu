package api

import(
 "testing"
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
