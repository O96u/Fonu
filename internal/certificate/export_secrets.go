package certificate

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// Keep ordinary deployment settings readable while encrypting the four
// credential fields as an authenticated envelope under the application's key.
type exportEnvelope struct {
	ExportConfig
	Encrypted string `json:"ssh_secrets_encrypted,omitempty"`
}

type deploymentSecrets struct {
	Password string `json:"password,omitempty"`
	PrivateKey string `json:"private_key,omitempty"`
	Passphrase string `json:"passphrase,omitempty"`
	ProxyPassword string `json:"proxy_password,omitempty"`
}

func extractDeploymentSecrets(cfg ExportConfig) deploymentSecrets {
	return deploymentSecrets{cfg.SSH.Password,cfg.SSH.PrivateKey,cfg.SSH.PrivateKeyPassphrase,cfg.SSH.ProxyPassword}
}

func (s *Store) encodeExportConfig(cfg ExportConfig) (string,error) {
	values:=extractDeploymentSecrets(cfg)
	if values==(deploymentSecrets{}) { return cfg.Marshal() }
	if s.box==nil { return "",fmt.Errorf("证书部署凭证加密服务未初始化") }
	raw,err:=json.Marshal(values); if err!=nil{return "",err}
	encrypted,err:=s.box.Encrypt(string(raw)); if err!=nil{return "",fmt.Errorf("证书部署凭证加密失败")}
	out,err:=json.Marshal(exportEnvelope{ExportConfig:cfg.RedactSecrets(),Encrypted:encrypted})
	return string(out),err
}

func (s *Store) decodeExportConfig(raw string) (ExportConfig,error) {
	if strings.TrimSpace(raw)=="" {return DefaultExportConfig(),nil}
	var env exportEnvelope
	if err:=json.Unmarshal([]byte(raw),&env);err!=nil{return ExportConfig{},fmt.Errorf("证书映射配置 JSON 无效")}
	if env.Encrypted==""{return env.ExportConfig,nil}
	if s.box==nil{return ExportConfig{},fmt.Errorf("证书部署凭证加密服务未初始化")}
	plain,err:=s.box.Decrypt(env.Encrypted)
	if err!=nil{return ExportConfig{},fmt.Errorf("证书部署凭证无法解密，请检查原加密密钥")}
	var values deploymentSecrets
	if err:=json.Unmarshal([]byte(plain),&values);err!=nil{return ExportConfig{},fmt.Errorf("证书部署凭证格式无效")}
	env.SSH.Password=values.Password
	env.SSH.PrivateKey=values.PrivateKey
	env.SSH.PrivateKeyPassphrase=values.Passphrase
	env.SSH.ProxyPassword=values.ProxyPassword
	return env.ExportConfig,nil
}

// Migrate legacy plaintext settings in one transaction before serving HTTP.
// Already encrypted records are unchanged, making subsequent starts idempotent.
func (s *Store) MigrateExportSecrets(ctx context.Context) error {
	if s.box==nil{return fmt.Errorf("证书部署凭证加密服务未初始化")}
	tx,err:=s.db.BeginTx(ctx,nil);if err!=nil{return err};defer tx.Rollback()
	rows,err:=tx.QueryContext(ctx,"SELECT id, export_json FROM certificates WHERE export_json IS NOT NULL AND export_json != ''")
	if err!=nil{return err}
	type item struct{id int64; raw string}
	var pending []item
	for rows.Next(){
		var id int64;var raw string
		if err:=rows.Scan(&id,&raw);err!=nil{rows.Close();return err}
		var env exportEnvelope
		if err:=json.Unmarshal([]byte(raw),&env);err!=nil{rows.Close();return fmt.Errorf("证书 %d 映射配置 JSON 无效",id)}
		if env.Encrypted!=""||extractDeploymentSecrets(env.ExportConfig)==(deploymentSecrets{}){continue}
		next,err:=s.encodeExportConfig(env.ExportConfig);if err!=nil{rows.Close();return err}
		pending=append(pending,item{id,next})
	}
	err=rows.Err();rows.Close();if err!=nil{return err}
	for _,v:=range pending{if _,err:=tx.ExecContext(ctx,"UPDATE certificates SET export_json = ? WHERE id = ?",v.raw,v.id);err!=nil{return err}}
	return tx.Commit()
}
