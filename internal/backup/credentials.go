package backup

import (
 "context"
 "database/sql"
 "encoding/json"
 "fmt"
 "net/url"
 "os"
 "path/filepath"
 "strings"

 "github.com/fonu/fonu/internal/secret"
)

// Verify that the staged database and effective encryption key agree.
func validateRestoreCredentials(ctx context.Context,stage,live,explicit string)error{
 u:=url.URL{Scheme:"file",Path:filepath.ToSlash(filepath.Join(stage,"fonu.db")),RawQuery:"mode=ro"}
 conn,err:=sql.Open("sqlite",u.String());if err!=nil{return err};defer conn.Close()
 conn.SetMaxOpenConns(1)
 var ciphertexts []string
 hasColumn:=func(table,column string)(bool,error){
  var n int
  err:=conn.QueryRowContext(ctx,"SELECT count(*) FROM pragma_table_info(?) WHERE name = ?",table,column).Scan(&n)
  return n>0,err
 }
 read:=func(query string,consume func(string)error)error{
  rows,err:=conn.QueryContext(ctx,query);if err!=nil{return err};defer rows.Close()
  for rows.Next(){var raw string;if err:=rows.Scan(&raw);err!=nil{return err};if err:=consume(raw);err!=nil{return err}}
  return rows.Err()
 }
 add:=func(raw string)error{if raw!=""{ciphertexts=append(ciphertexts,raw)};return nil}
 if ok,err:=hasColumn("ddns_configs","api_token_enc");err!=nil{return err}else if ok{
  if err:=read("SELECT api_token_enc FROM ddns_configs WHERE api_token_enc IS NOT NULL AND api_token_enc != ''",add);err!=nil{return err}
 }
 if ok,err:=hasColumn("settings","value");err!=nil{return err}else if ok{
  if err:=read("SELECT value FROM settings WHERE key IN ('frp_auth_token','notify_smtp_password','notify_webhook_secret','notify_telegram_token','dnshe_api_key','dnshe_api_secret') AND value != ''",add);err!=nil{return err}
  if err:=read("SELECT value FROM settings WHERE key = 'dnshe_accounts' AND value != ''",func(raw string)error{
   var accounts []map[string]json.RawMessage
   if err:=json.Unmarshal([]byte(raw),&accounts);err!=nil{return fmt.Errorf("备份 DNSHE 凭证格式无效")}
   for _,account:=range accounts{for _,field:=range []string{"api_key_enc","api_secret_enc"}{if value,ok:=account[field];ok{var enc string;if err:=json.Unmarshal(value,&enc);err!=nil{return err};_ = add(enc)}}};return nil
  });err!=nil{return err}
 }
 if ok,err:=hasColumn("certificates","export_json");err!=nil{return err}else if ok{
  if err:=read("SELECT export_json FROM certificates WHERE export_json IS NOT NULL AND export_json != ''",func(raw string)error{
   var config map[string]json.RawMessage
   if err:=json.Unmarshal([]byte(raw),&config);err!=nil{return fmt.Errorf("备份证书部署配置格式无效")}
   if value,ok:=config["ssh_secrets_encrypted"];ok{var enc string;if err:=json.Unmarshal(value,&enc);err!=nil{return err};return add(enc)}
   return nil
  });err!=nil{return err}
 }
 if len(ciphertexts)==0{return nil}
 matches:=func(key string)bool{
  box,err:=secret.NewBox(key);if err!=nil{return false}
  for _,enc:=range ciphertexts{if _,err:=box.Decrypt(enc);err!=nil{return false}}
  return true
 }
 mismatch:=fmt.Errorf("备份加密凭证与密钥不匹配，已拒绝恢复；请使用原始备份及对应密钥")
 if explicit!=""&&explicit!=secret.LegacyDefaultKey{
  if !matches(explicit){return mismatch};return nil
 }
 raw,err:=os.ReadFile(filepath.Join(stage,secret.KeyFileName))
 if err==nil{
  if !matches(strings.TrimSpace(string(raw))){return mismatch};return nil
 }
 if !os.IsNotExist(err){return err}
 raw,err=os.ReadFile(filepath.Join(live,secret.KeyFileName))
 if err==nil&&matches(strings.TrimSpace(string(raw))){return nil}
 if err!=nil&&!os.IsNotExist(err){return err}
 if os.IsNotExist(err){
  if legacyFile:=os.Getenv("FONU_LEGACY_FPK_KEY_FILE");legacyFile!=""{
   if raw,e:=os.ReadFile(legacyFile);e==nil&&matches(strings.TrimSpace(string(raw))){return nil}
  }
 }
 // Old exports without a key can be accepted only after successful decryption.
 if matches(secret.LegacyDefaultKey){return secret.SaveStoredKey(stage,secret.LegacyDefaultKey)}
 return mismatch
}
