package service

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/fonu/fonu/internal/proxy"

	"github.com/fonu/fonu/internal/chinacidr"
	"github.com/fonu/fonu/internal/nginx"
	"github.com/fonu/fonu/internal/settings"
)

func (s *ProxyService) loadGenerateOptions(ctx context.Context) nginx.GenerateOptions {
	opts := nginx.GenerateOptions{}
	if s.settings == nil {
		return opts
	}
	opts.GlobalCustomMode, _ = s.settings.Get(ctx, settings.KeyNginxGlobalMode)
	raw, err := s.settings.Get(ctx, settings.KeyTrustedProxy)
	if err == nil {
		opts.TrustedProxy = nginx.ParseTrustedProxyJSON(raw)
	}
	blRaw, err := s.settings.Get(ctx, settings.KeyGlobalIPBlacklist)
	if err == nil {
		opts.GlobalIPBlacklist = chinacidr.ParseGlobalIPList(blRaw)
	}
	wlRaw, err := s.settings.Get(ctx, settings.KeyGlobalIPWhitelist)
	if err == nil {
		opts.GlobalIPWhitelist = chinacidr.ParseGlobalIPList(wlRaw)
	}
	return opts
}

 // SaveSettingsWithPublication commits global policy changes together with
 // the candidate Nginx configuration, rather than storing a rejected policy.
func (s *ProxyService) SaveSettingsWithPublication(ctx context.Context, values map[string]string) error {
	s.applyMu.Lock()
	defer s.applyMu.Unlock()
	var trusted *nginx.TrustedProxyConfig
	if raw,ok:=values[settings.KeyTrustedProxy];ok {
		var cfg nginx.TrustedProxyConfig
		if raw!="" {
			if err:=json.Unmarshal([]byte(raw),&cfg);err!=nil{return err}
		}
		trusted=&cfg
	}
	_,err:=mutateProxyWithFiles(ctx,s,func(_ *proxy.Store,tx *sql.Tx)(struct{},error){
		for key,value:=range values{
			if _,err:=tx.ExecContext(ctx,`INSERT INTO settings(key,value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`,key,value);err!=nil{return struct{}{},err}
		}
		return struct{}{},nil
	},func(opts *nginx.GenerateOptions){
		if trusted!=nil{opts.TrustedProxy=*trusted}
		if raw,ok:=values[settings.KeyGlobalIPBlacklist];ok{opts.GlobalIPBlacklist=chinacidr.ParseGlobalIPList(raw)}
		if raw,ok:=values[settings.KeyGlobalIPWhitelist];ok{opts.GlobalIPWhitelist=chinacidr.ParseGlobalIPList(raw)}
	},nil,nil)
	return err
}
