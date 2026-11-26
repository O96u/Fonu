package notify

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/fonu/fonu/internal/secret"
	"github.com/fonu/fonu/internal/settings"
)

type Store struct {
	settings *settings.Store
	secret   *secret.Box
}

func NewStore(settingsStore *settings.Store, secretBox *secret.Box) *Store {
	return &Store{settings: settingsStore, secret: secretBox}
}

func (s *Store) Load(ctx context.Context) (Config, error) {
	cfg, _, err := s.load(ctx, false)
	return cfg, err
}

func (s *Store) LoadRuntime(ctx context.Context) (RuntimeConfig, error) {
	cfg, runtime, err := s.load(ctx, true)
	if err != nil {
		return RuntimeConfig{}, err
	}
	return cfg.Runtime(runtime.smtpPassword, runtime.webhookSecret, runtime.telegramToken), nil
}

// Editing must remain possible when a previously stored credential is corrupt.
func(s *Store) loadRuntimeForEdit(ctx context.Context)(RuntimeConfig,error){
 cfg,r,err:=s.load(ctx,true,false);if err!=nil{return RuntimeConfig{},err};return cfg.Runtime(r.smtpPassword,r.webhookSecret,r.telegramToken),nil
}

type runtimeSecrets struct {
	smtpPassword   string
	webhookSecret  string
	telegramToken  string
}

func (s *Store) load(ctx context.Context, decrypt bool, strict ...bool) (Config, runtimeSecrets, error) {
 strictActive:=decrypt&&(len(strict)==0||strict[0])
	cfg := Config{
		Email:    EmailConfig{Port: 587, TLS: true},
		Webhook:  WebhookConfig{Provider: WebhookBark, Server: DefaultBarkServer},
		Telegram: TelegramConfig{},
	}
	secrets := runtimeSecrets{}
 values,err:=s.settings.List(ctx);if err!=nil{return cfg,secrets,err}
 get:=func(key string)(string,error){return values[key],nil}
 flagValue:=func(key string,fallback bool)bool{v:=strings.TrimSpace(values[key]);if v==""{return fallback};return v=="1"||strings.EqualFold(v,"true")}
 intValue:=func(key string,fallback int)int{n,e:=strconv.Atoi(strings.TrimSpace(values[key]));if e!=nil||n<=0{return fallback};return n}

	typeRaw, _ := get(settings.KeyNotifyType)
	cfg.Type = NotifyType(strings.TrimSpace(typeRaw))

	emailRaw, _ := get(settings.KeyNotifyEmailJSON)
	if emailRaw != "" {
		if err:=json.Unmarshal([]byte(emailRaw), &cfg.Email);err!=nil&&strictActive&&cfg.Type==NotifyTypeEmail{return cfg,secrets,fmt.Errorf("当前通知配置格式无效")}
	}
	if cfg.Email.Port <= 0 {
		cfg.Email.Port = 587
	}

	webhookRaw, _ := get(settings.KeyNotifyWebhookJSON)
	if webhookRaw != "" {
		if err:=json.Unmarshal([]byte(webhookRaw), &cfg.Webhook);err!=nil&&strictActive&&cfg.Type==NotifyTypeWebhook{return cfg,secrets,fmt.Errorf("当前通知配置格式无效")}
	}
	if cfg.Webhook.Server == "" && cfg.Webhook.Provider == WebhookBark {
		cfg.Webhook.Server = DefaultBarkServer
	}

	telegramRaw, _ := get(settings.KeyNotifyTelegramJSON)
	if telegramRaw != "" {
		if err:=json.Unmarshal([]byte(telegramRaw), &cfg.Telegram);err!=nil&&strictActive&&cfg.Type==NotifyTypeTelegram{return cfg,secrets,fmt.Errorf("当前通知配置格式无效")}
	}

	cfg.IPFrequentThreshold = intValue(settings.KeyNotifyIPFrequentThreshold, DefaultIPFrequentThreshold)
	cfg.IPFrequentWindowSec = intValue(settings.KeyNotifyIPFrequentWindowSec, DefaultIPFrequentWindowSec)

	cfg.LoginFailureThreshold = intValue(settings.KeyNotifyLoginFailureThreshold, DefaultLoginFailureThreshold)
	cfg.LoginFailureWindowSec = intValue(settings.KeyNotifyLoginFailureWindowSec, DefaultLoginFailureWindowSec)

 cfg.OnDDNSIPChange=flagValue(settings.KeyNotifyOnDDNSIPChange,true)
 cfg.OnDDNSFailure=flagValue(settings.KeyNotifyOnDDNSFailure,flagValue(settings.KeyNotifyOnDDNSError,true))
 cfg.OnCertExpiry=flagValue(settings.KeyNotifyOnCertExpiry,flagValue(settings.KeyNotifyOnCertError,true))
 cfg.OnCertRenewSuccess=flagValue(settings.KeyNotifyOnCertRenewSuccess,true)
 cfg.OnCertRenewFailure=flagValue(settings.KeyNotifyOnCertRenewFailure,flagValue(settings.KeyNotifyOnCertError,true))
 cfg.OnIPFrequentAccess=flagValue(settings.KeyNotifyOnIPFrequentAccess,false)
 cfg.OnScan=flagValue(settings.KeyNotifyOnScan,false)
 cfg.OnCC=flagValue(settings.KeyNotifyOnCC,false)
 cfg.OnLoginFailure=flagValue(settings.KeyNotifyOnLoginFailure,false)
 cfg.OnNginxReloadFailure=flagValue(settings.KeyNotifyOnNginxReloadFailure,true)

	smtpEnc, _ := get(settings.KeyNotifySMTPPassword)
	if smtpEnc != "" {
		cfg.Email.HasPassword = true
		if decrypt {
    if s.secret==nil{if strictActive&&cfg.Type==NotifyTypeEmail{return cfg,secrets,fmt.Errorf("通知加密模块不可用")}}else if plain,err:=s.secret.Decrypt(smtpEnc);err==nil{secrets.smtpPassword=plain}else if strictActive&&cfg.Type==NotifyTypeEmail{return cfg,secrets,fmt.Errorf("当前通知凭证无法解密，请重新保存凭证")}
  }
	}

	webhookEnc, _ := get(settings.KeyNotifyWebhookSecret)
	if webhookEnc != "" {
		cfg.Webhook.HasSecret = true
		if decrypt {
    if s.secret==nil{if strictActive&&cfg.Type==NotifyTypeWebhook{return cfg,secrets,fmt.Errorf("通知加密模块不可用")}}else if plain,err:=s.secret.Decrypt(webhookEnc);err==nil{secrets.webhookSecret=plain}else if strictActive&&cfg.Type==NotifyTypeWebhook{return cfg,secrets,fmt.Errorf("当前通知凭证无法解密，请重新保存凭证")}
  }
	}

	telegramEnc, _ := get(settings.KeyNotifyTelegramToken)
	if telegramEnc != "" {
		cfg.Telegram.HasBotToken = true
		if decrypt {
    if s.secret==nil{if strictActive&&cfg.Type==NotifyTypeTelegram{return cfg,secrets,fmt.Errorf("通知加密模块不可用")}}else if plain,err:=s.secret.Decrypt(telegramEnc);err==nil{secrets.telegramToken=plain}else if strictActive&&cfg.Type==NotifyTypeTelegram{return cfg,secrets,fmt.Errorf("当前通知凭证无法解密，请重新保存凭证")}
  }
	}

	if _,explicit:=values[settings.KeyNotifyType];!explicit && cfg.Type == "" {
		legacyURL, _ := get(settings.KeyNotifyWebhookURL)
		if strings.TrimSpace(legacyURL) != "" {
			cfg.Type = NotifyTypeWebhook
			cfg.Webhook.Provider = WebhookCustom
			cfg.Webhook.URL = strings.TrimSpace(legacyURL)
		}
	}

	return cfg, secrets, nil
}

func flag(store *settings.Store, ctx context.Context, key string, fallback bool) bool {
	raw, err := store.Get(ctx, key)
	if err != nil || raw == "" {
		return fallback
	}
	return raw == "1" || strings.EqualFold(raw, "true")
}

func intSetting(store *settings.Store, ctx context.Context, key string, fallback int) int {
	raw, err := store.Get(ctx, key)
	if err != nil || strings.TrimSpace(raw) == "" {
		return fallback
	}
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func hasNotifyEventSettings(ctx context.Context, store *settings.Store) bool {
	keys := []string{
		settings.KeyNotifyOnDDNSIPChange,
		settings.KeyNotifyOnDDNSFailure,
		settings.KeyNotifyOnCertExpiry,
		settings.KeyNotifyOnCertRenewSuccess,
		settings.KeyNotifyOnIPFrequentAccess,
 settings.KeyNotifyOnScan,
 settings.KeyNotifyOnCC,
		settings.KeyNotifyOnCertRenewFailure,
		settings.KeyNotifyOnLoginFailure,
		settings.KeyNotifyOnNginxReloadFailure,
	}
	for _, key := range keys {
		raw, err := store.Get(ctx, key)
		if err == nil && raw != "" {
			return true
		}
	}
	return false
}

func (s *Store) Save(ctx context.Context,in SaveInput)error{
 values,err:=s.PrepareSave(ctx,in)
 if err!=nil{return err}
 return s.settings.SetMany(ctx,values)
}

// PrepareSave validates and encrypts notification values without persisting them.
func (s *Store) PrepareSave(ctx context.Context, in SaveInput) (map[string]string,error) {
	existing, err := s.Load(ctx)
 if err != nil{return nil,err}
 in.Email.HasPassword=existing.Email.HasPassword || (strings.TrimSpace(in.SMTPPassword)!="" && in.SMTPPassword!=MaskedSecret)
 if strings.TrimSpace(in.WebhookSecret)==""||in.WebhookSecret==MaskedSecret {if in.Webhook.Provider==existing.Webhook.Provider&&strings.TrimSpace(in.Webhook.Key)==""&&existing.Webhook.Key!=""&&existing.Webhook.Key!=MaskedSecret {in.Webhook.Key=existing.Webhook.Key}}
 in.Webhook.HasSecret=(in.Webhook.Provider==existing.Webhook.Provider&&existing.Webhook.HasSecret) || (strings.TrimSpace(in.WebhookSecret)!="" && in.WebhookSecret!=MaskedSecret) || (strings.TrimSpace(in.Webhook.Key)!="" && in.Webhook.Key!=MaskedSecret)
 in.Telegram.HasBotToken=existing.Telegram.HasBotToken || (strings.TrimSpace(in.TelegramToken)!="" && in.TelegramToken!=MaskedSecret)
 same:=sameWebhookIdentity(in.Webhook,existing.Webhook)
 runtime,err:=s.loadRuntimeForEdit(ctx);if err!=nil{return nil,err}
 if in.Type==NotifyTypeEmail&&runtime.SMTPPassword==""{in.Email.HasPassword=strings.TrimSpace(in.SMTPPassword)!=""&&in.SMTPPassword!=MaskedSecret}
 if in.Type==NotifyTypeTelegram&&runtime.TelegramToken==""{in.Telegram.HasBotToken=strings.TrimSpace(in.TelegramToken)!=""&&in.TelegramToken!=MaskedSecret}
 input:=in.WebhookSecret
 if (input==""||input==MaskedSecret)&&in.Webhook.Key!=""&&in.Webhook.Key!=MaskedSecret{input=in.Webhook.Key}
 oldSecret:=webhookSecret(runtime)
 resolved,err:=resolveWebhookCredential(in.Webhook,input,existing.Webhook,oldSecret)
 if err!=nil{return nil,err}
 in.Webhook.HasSecret=resolved!=""
 in.WebhookSecret=resolved
 in.Webhook.Key=""
 if in.Type==NotifyTypeWebhook {if err:=validateWebhookCredential(in.Webhook,resolved);err!=nil{return nil,err}}
 values:=map[string]string{}
 if !same || resolved==""{values[settings.KeyNotifyWebhookSecret]=""}

	if err := validateSaveInput(in); err != nil {
		return nil,err
	}

	values[settings.KeyNotifyType] = string(in.Type)
	values[settings.KeyNotifyOnDDNSIPChange] = strconv.FormatBool(in.OnDDNSIPChange)
	values[settings.KeyNotifyOnDDNSFailure] = strconv.FormatBool(in.OnDDNSFailure)
	values[settings.KeyNotifyOnCertExpiry] = strconv.FormatBool(in.OnCertExpiry)
	values[settings.KeyNotifyOnCertRenewSuccess] = strconv.FormatBool(in.OnCertRenewSuccess)
	values[settings.KeyNotifyOnIPFrequentAccess] = strconv.FormatBool(in.OnIPFrequentAccess)
 values[settings.KeyNotifyOnScan]=strconv.FormatBool(in.OnScan)
 values[settings.KeyNotifyOnCC]=strconv.FormatBool(in.OnCC)
	values[settings.KeyNotifyOnCertRenewFailure] = strconv.FormatBool(in.OnCertRenewFailure)
	values[settings.KeyNotifyOnLoginFailure] = strconv.FormatBool(in.OnLoginFailure)
	values[settings.KeyNotifyOnNginxReloadFailure] = strconv.FormatBool(in.OnNginxReloadFailure)
	threshold := in.IPFrequentThreshold
	if threshold <= 0 {
		threshold = DefaultIPFrequentThreshold
	}
	windowSec := in.IPFrequentWindowSec
	if windowSec <= 0 {
		windowSec = DefaultIPFrequentWindowSec
	}
	values[settings.KeyNotifyIPFrequentThreshold] = strconv.Itoa(threshold)
	values[settings.KeyNotifyIPFrequentWindowSec] = strconv.Itoa(windowSec)
	loginThreshold := in.LoginFailureThreshold
	if loginThreshold <= 0 {
		loginThreshold = DefaultLoginFailureThreshold
	}
	loginWindowSec := in.LoginFailureWindowSec
	if loginWindowSec <= 0 {
		loginWindowSec = DefaultLoginFailureWindowSec
	}
	values[settings.KeyNotifyLoginFailureThreshold] = strconv.Itoa(loginThreshold)
	values[settings.KeyNotifyLoginFailureWindowSec] = strconv.Itoa(loginWindowSec)

	email := in.Email
	email.HasPassword = email.HasPassword || strings.TrimSpace(in.SMTPPassword) != "" && in.SMTPPassword != MaskedSecret
	emailJSON, err := json.Marshal(email)
	if err != nil {
		return nil,err
	}
	values[settings.KeyNotifyEmailJSON] = string(emailJSON)

	webhook := in.Webhook
	secretValue := strings.TrimSpace(in.WebhookSecret)
	if secretValue != "" && secretValue != MaskedSecret {
		webhook.HasSecret = true
	} else if webhook.Provider == WebhookBark || webhook.Provider == WebhookGotify {
		if strings.TrimSpace(webhook.Key) != "" && webhook.Key != MaskedSecret {
			webhook.HasSecret = true
		}
	}
	storedWebhook:=webhook
 storedWebhook.Key=""
 webhookJSON, err := json.Marshal(storedWebhook)
	if err != nil {
		return nil,err
	}
	values[settings.KeyNotifyWebhookJSON] = string(webhookJSON)

	telegram := in.Telegram
	normalizedProxyURL, err := normalizeProxyURL(telegram.ProxyURL)
	if err != nil {
		return nil,err
	}
	telegram.ProxyURL = normalizedProxyURL
	if strings.TrimSpace(in.TelegramToken) != "" && in.TelegramToken != MaskedSecret {
		telegram.HasBotToken = true
	}
	telegramJSON, err := json.Marshal(telegram)
	if err != nil {
		return nil,err
	}
	values[settings.KeyNotifyTelegramJSON] = string(telegramJSON)

 values[settings.KeyNotifyWebhookURL]=""
	if in.Type == NotifyTypeWebhook && webhook.Provider == WebhookCustom {
		values[settings.KeyNotifyWebhookURL] = strings.TrimSpace(webhook.URL)
	}

	if err := s.prepareSecret(values, settings.KeyNotifySMTPPassword, in.SMTPPassword, in.Email.HasPassword); err != nil {
		return nil,err
	}

	webhookPlain := secretValue
	if webhookPlain == "" || webhookPlain == MaskedSecret {
		if webhook.Provider == WebhookBark || webhook.Provider == WebhookGotify {
			if strings.TrimSpace(webhook.Key) != "" && webhook.Key != MaskedSecret {
				webhookPlain = strings.TrimSpace(webhook.Key)
			}
		} else if webhook.Provider == WebhookNtfy {
			if strings.TrimSpace(webhook.Key) != "" && webhook.Key != MaskedSecret {
				webhookPlain = strings.TrimSpace(webhook.Key)
			}
		}
	}
	if err := s.prepareSecret(values, settings.KeyNotifyWebhookSecret, webhookPlain, webhook.HasSecret); err != nil {
		return nil,err
	}

	if err := s.prepareSecret(values, settings.KeyNotifyTelegramToken, in.TelegramToken, telegram.HasBotToken); err != nil {
		return nil,err
	}

 return values,nil
}

func (s *Store) prepareSecret(values map[string]string, key, plain string, hasExisting bool) error {
	plain = strings.TrimSpace(plain)
	if plain == "" || plain == MaskedSecret {
		if !hasExisting {
			return nil
		}
		return nil
	}
	if s.secret == nil {
		return fmt.Errorf("加密模块未初始化")
	}
	enc, err := s.secret.Encrypt(plain)
	if err != nil {
		return err
	}
	values[key]=enc
 return nil
}

func validateSaveInput(in SaveInput) error {
	switch in.Type {
	case "":
		return nil
	case NotifyTypeEmail:
		if strings.TrimSpace(in.Email.Host) == "" {
			return fmt.Errorf("请填写 SMTP 主机")
		}
		if len(in.Email.To) == 0 {
			return fmt.Errorf("请填写收件人邮箱")
		}
		password := strings.TrimSpace(in.SMTPPassword)
		if password == "" || password == MaskedSecret {
			if !in.Email.HasPassword {
				return fmt.Errorf("请填写 SMTP 密码")
			}
		}
	case NotifyTypeWebhook:
		switch in.Webhook.Provider {
		case WebhookBark:
			if strings.TrimSpace(in.Webhook.Key) == "" && !in.Webhook.HasSecret {
				return fmt.Errorf("请填写 Bark Device Key")
			}
		case WebhookNtfy:
			if strings.TrimSpace(in.Webhook.Topic) == "" {
				return fmt.Errorf("请填写 ntfy Topic")
			}
		case WebhookGotify:
			if strings.TrimSpace(in.Webhook.Key) == "" && !in.Webhook.HasSecret {
				return fmt.Errorf("请填写 Gotify App Token")
			}
        case WebhookWeCom,WebhookDingTalk,WebhookFeishu,WebhookWeComApp,WebhookFeishuApp:
          return validateWebhookCredential(in.Webhook,in.WebhookSecret)
		case WebhookCustom:
			if strings.TrimSpace(in.Webhook.URL) == "" {
				return fmt.Errorf("请填写 Webhook URL")
			}
		default:
			return fmt.Errorf("不支持的 Webhook 预设")
		}
	case NotifyTypeTelegram:
		token := strings.TrimSpace(in.TelegramToken)
		if token == "" && !in.Telegram.HasBotToken {
			return fmt.Errorf("请填写 Telegram Bot Token")
		}
		if strings.TrimSpace(in.Telegram.ChatID) == "" {
			return fmt.Errorf("请填写 Telegram Chat ID")
		}
		if _, err := normalizeProxyURL(in.Telegram.ProxyURL); err != nil {
			return err
		}
	default:
		return fmt.Errorf("不支持的通知类型")
	}
	return nil
}

func SanitizeSettings(values map[string]string) {
	delete(values, settings.KeyNotifySMTPPassword)
	delete(values, settings.KeyNotifyWebhookSecret)
	delete(values, settings.KeyNotifyTelegramToken)
 if raw:=values[settings.KeyNotifyWebhookJSON];raw!=""{var cfg WebhookConfig;if json.Unmarshal([]byte(raw),&cfg)==nil{if cfg.Key!=""{cfg.HasSecret=true};cfg.Key="";if encoded,err:=json.Marshal(cfg);err==nil{values[settings.KeyNotifyWebhookJSON]=string(encoded)}}}
}

func ParseSaveInputFromMap(values map[string]string, existing Config) (SaveInput, bool) {
	if !mapHasNotifySavePayload(values) {
		return SaveInput{}, false
	}

	in := SaveInput{
		Type:          existing.Type,
		Email:         existing.Email,
		Webhook:       existing.Webhook,
		Telegram:      existing.Telegram,
		SMTPPassword:  values[settings.KeyNotifySMTPPassword],
		WebhookSecret: values[settings.KeyNotifyWebhookSecret],
		TelegramToken: values[settings.KeyNotifyTelegramToken],
	}
	if raw, ok := values[settings.KeyNotifyType]; ok {
		in.Type = NotifyType(strings.TrimSpace(raw))
	}
	if raw := values[settings.KeyNotifyEmailJSON]; raw != "" {
		_ = json.Unmarshal([]byte(raw), &in.Email)
	}
	if raw := values[settings.KeyNotifyWebhookJSON]; raw != "" {
		_ = json.Unmarshal([]byte(raw), &in.Webhook)
	}
	if raw := values[settings.KeyNotifyTelegramJSON]; raw != "" {
		_ = json.Unmarshal([]byte(raw), &in.Telegram)
	}
	in.OnDDNSIPChange = boolFromMapOr(values, settings.KeyNotifyOnDDNSIPChange, existing.OnDDNSIPChange)
	in.OnDDNSFailure = boolFromMapOr(values, settings.KeyNotifyOnDDNSFailure, existing.OnDDNSFailure)
	if _, ok := values[settings.KeyNotifyOnDDNSFailure]; !ok {
		if _, legacyOk := values[settings.KeyNotifyOnDDNSError]; legacyOk {
			in.OnDDNSFailure = boolFromMap(values, settings.KeyNotifyOnDDNSError)
		}
	}
	in.OnCertExpiry = boolFromMapOr(values, settings.KeyNotifyOnCertExpiry, existing.OnCertExpiry)
	if _, ok := values[settings.KeyNotifyOnCertExpiry]; !ok {
		if _, legacyOk := values[settings.KeyNotifyOnCertError]; legacyOk {
			in.OnCertExpiry = boolFromMap(values, settings.KeyNotifyOnCertError)
		}
	}
	in.OnCertRenewSuccess = boolFromMapOr(values, settings.KeyNotifyOnCertRenewSuccess, existing.OnCertRenewSuccess)
	in.OnIPFrequentAccess = boolFromMapOr(values, settings.KeyNotifyOnIPFrequentAccess, existing.OnIPFrequentAccess)
 in.OnScan=boolFromMapOr(values,settings.KeyNotifyOnScan,existing.OnScan)
 in.OnCC=boolFromMapOr(values,settings.KeyNotifyOnCC,existing.OnCC)
	in.OnCertRenewFailure = boolFromMapOr(values, settings.KeyNotifyOnCertRenewFailure, existing.OnCertRenewFailure)
	in.OnLoginFailure = boolFromMapOr(values, settings.KeyNotifyOnLoginFailure, existing.OnLoginFailure)
	in.OnNginxReloadFailure = boolFromMapOr(values, settings.KeyNotifyOnNginxReloadFailure, existing.OnNginxReloadFailure)
	in.IPFrequentThreshold = intSettingFromMapOr(values, settings.KeyNotifyIPFrequentThreshold, existing.IPFrequentThreshold, DefaultIPFrequentThreshold)
	in.IPFrequentWindowSec = intSettingFromMapOr(values, settings.KeyNotifyIPFrequentWindowSec, existing.IPFrequentWindowSec, DefaultIPFrequentWindowSec)
	in.LoginFailureThreshold = intSettingFromMapOr(values, settings.KeyNotifyLoginFailureThreshold, existing.LoginFailureThreshold, DefaultLoginFailureThreshold)
	in.LoginFailureWindowSec = intSettingFromMapOr(values, settings.KeyNotifyLoginFailureWindowSec, existing.LoginFailureWindowSec, DefaultLoginFailureWindowSec)
	return in, true
}

func mapHasNotifySavePayload(values map[string]string) bool {
 if _,present:=values[settings.KeyNotifyType];present{return true}
	if values[settings.KeyNotifyType] != "" ||
		values[settings.KeyNotifyEmailJSON] != "" ||
		values[settings.KeyNotifyWebhookJSON] != "" ||
		values[settings.KeyNotifyTelegramJSON] != "" ||
		values[settings.KeyNotifySMTPPassword] != "" ||
		values[settings.KeyNotifyWebhookSecret] != "" ||
		values[settings.KeyNotifyTelegramToken] != "" {
		return true
	}
	keys := []string{
		settings.KeyNotifyOnDDNSIPChange,
		settings.KeyNotifyOnDDNSFailure,
		settings.KeyNotifyOnCertExpiry,
		settings.KeyNotifyOnCertRenewSuccess,
		settings.KeyNotifyOnIPFrequentAccess,
 settings.KeyNotifyOnScan,
 settings.KeyNotifyOnCC,
		settings.KeyNotifyOnCertRenewFailure,
		settings.KeyNotifyOnLoginFailure,
		settings.KeyNotifyOnNginxReloadFailure,
		settings.KeyNotifyIPFrequentThreshold,
		settings.KeyNotifyIPFrequentWindowSec,
		settings.KeyNotifyLoginFailureThreshold,
		settings.KeyNotifyLoginFailureWindowSec,
		settings.KeyNotifyWebhookURL,
	}
	for _, key := range keys {
		if _, ok := values[key]; ok {
			return true
		}
	}
	return false
}

func boolFromMap(values map[string]string, key string) bool {
	raw, ok := values[key]
	if !ok {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func boolFromMapOr(values map[string]string, key string, fallback bool) bool {
	if _, ok := values[key]; !ok {
		return fallback
	}
	return boolFromMap(values, key)
}

func intSettingFromMap(values map[string]string, key string, fallback int) int {
	raw := strings.TrimSpace(values[key])
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func intSettingFromMapOr(values map[string]string, key string, existing, fallback int) int {
	if _, ok := values[key]; !ok {
		if existing > 0 {
			return existing
		}
		return fallback
	}
	return intSettingFromMap(values, key, fallback)
}
