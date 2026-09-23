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

type runtimeSecrets struct {
	smtpPassword   string
	webhookSecret  string
	telegramToken  string
}

func (s *Store) load(ctx context.Context, decrypt bool) (Config, runtimeSecrets, error) {
	cfg := Config{
		Email:    EmailConfig{Port: 587, TLS: true},
		Webhook:  WebhookConfig{Provider: WebhookBark, Server: DefaultBarkServer},
		Telegram: TelegramConfig{},
	}
	secrets := runtimeSecrets{}

	typeRaw, _ := s.settings.Get(ctx, settings.KeyNotifyType)
	cfg.Type = NotifyType(strings.TrimSpace(typeRaw))

	emailRaw, _ := s.settings.Get(ctx, settings.KeyNotifyEmailJSON)
	if emailRaw != "" {
		_ = json.Unmarshal([]byte(emailRaw), &cfg.Email)
	}
	if cfg.Email.Port <= 0 {
		cfg.Email.Port = 587
	}

	webhookRaw, _ := s.settings.Get(ctx, settings.KeyNotifyWebhookJSON)
	if webhookRaw != "" {
		_ = json.Unmarshal([]byte(webhookRaw), &cfg.Webhook)
	}
	if cfg.Webhook.Server == "" && cfg.Webhook.Provider == WebhookBark {
		cfg.Webhook.Server = DefaultBarkServer
	}

	telegramRaw, _ := s.settings.Get(ctx, settings.KeyNotifyTelegramJSON)
	if telegramRaw != "" {
		_ = json.Unmarshal([]byte(telegramRaw), &cfg.Telegram)
	}

	cfg.IPFrequentThreshold = intSetting(s.settings, ctx, settings.KeyNotifyIPFrequentThreshold, DefaultIPFrequentThreshold)
	cfg.IPFrequentWindowSec = intSetting(s.settings, ctx, settings.KeyNotifyIPFrequentWindowSec, DefaultIPFrequentWindowSec)

	cfg.LoginFailureThreshold = intSetting(s.settings, ctx, settings.KeyNotifyLoginFailureThreshold, DefaultLoginFailureThreshold)
	cfg.LoginFailureWindowSec = intSetting(s.settings, ctx, settings.KeyNotifyLoginFailureWindowSec, DefaultLoginFailureWindowSec)

	if hasNotifyEventSettings(ctx, s.settings) {
		cfg.OnDDNSIPChange = flag(s.settings, ctx, settings.KeyNotifyOnDDNSIPChange, true)
		cfg.OnDDNSFailure = flag(s.settings, ctx, settings.KeyNotifyOnDDNSFailure, true)
		cfg.OnCertExpiry = flag(s.settings, ctx, settings.KeyNotifyOnCertExpiry, true)
		cfg.OnCertRenewSuccess = flag(s.settings, ctx, settings.KeyNotifyOnCertRenewSuccess, true)
		cfg.OnCertRenewFailure = flag(s.settings, ctx, settings.KeyNotifyOnCertRenewFailure, true)
		cfg.OnIPFrequentAccess = flag(s.settings, ctx, settings.KeyNotifyOnIPFrequentAccess, false)
		cfg.OnLoginFailure = flag(s.settings, ctx, settings.KeyNotifyOnLoginFailure, false)
		cfg.OnNginxReloadFailure = flag(s.settings, ctx, settings.KeyNotifyOnNginxReloadFailure, true)
	} else {
		legacyDDNS := flag(s.settings, ctx, settings.KeyNotifyOnDDNSError, true)
		legacyCert := flag(s.settings, ctx, settings.KeyNotifyOnCertError, true)
		cfg.OnDDNSFailure = legacyDDNS
		cfg.OnCertExpiry = legacyCert
		cfg.OnCertRenewFailure = legacyCert
		cfg.OnDDNSIPChange = true
		cfg.OnCertRenewSuccess = true
		cfg.OnNginxReloadFailure = true
		cfg.OnIPFrequentAccess = false
		cfg.OnLoginFailure = false
	}

	smtpEnc, _ := s.settings.Get(ctx, settings.KeyNotifySMTPPassword)
	if smtpEnc != "" {
		cfg.Email.HasPassword = true
		if decrypt && s.secret != nil {
			if plain, err := s.secret.Decrypt(smtpEnc); err == nil {
				secrets.smtpPassword = plain
			}
		}
	}

	webhookEnc, _ := s.settings.Get(ctx, settings.KeyNotifyWebhookSecret)
	if webhookEnc != "" {
		cfg.Webhook.HasSecret = true
		if decrypt && s.secret != nil {
			if plain, err := s.secret.Decrypt(webhookEnc); err == nil {
				secrets.webhookSecret = plain
			}
		}
	}

	telegramEnc, _ := s.settings.Get(ctx, settings.KeyNotifyTelegramToken)
	if telegramEnc != "" {
		cfg.Telegram.HasBotToken = true
		if decrypt && s.secret != nil {
			if plain, err := s.secret.Decrypt(telegramEnc); err == nil {
				secrets.telegramToken = plain
			}
		}
	}

	if cfg.Type == "" {
		legacyURL, _ := s.settings.Get(ctx, settings.KeyNotifyWebhookURL)
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

func (s *Store) Save(ctx context.Context, in SaveInput) error {
	if err := validateSaveInput(in); err != nil {
		return err
	}

	if err := s.settings.Set(ctx, settings.KeyNotifyType, string(in.Type)); err != nil {
		return err
	}
	if err := s.settings.SetBool(ctx, settings.KeyNotifyOnDDNSIPChange, in.OnDDNSIPChange); err != nil {
		return err
	}
	if err := s.settings.SetBool(ctx, settings.KeyNotifyOnDDNSFailure, in.OnDDNSFailure); err != nil {
		return err
	}
	if err := s.settings.SetBool(ctx, settings.KeyNotifyOnCertExpiry, in.OnCertExpiry); err != nil {
		return err
	}
	if err := s.settings.SetBool(ctx, settings.KeyNotifyOnCertRenewSuccess, in.OnCertRenewSuccess); err != nil {
		return err
	}
	if err := s.settings.SetBool(ctx, settings.KeyNotifyOnIPFrequentAccess, in.OnIPFrequentAccess); err != nil {
		return err
	}
	if err := s.settings.SetBool(ctx, settings.KeyNotifyOnCertRenewFailure, in.OnCertRenewFailure); err != nil {
		return err
	}
	if err := s.settings.SetBool(ctx, settings.KeyNotifyOnLoginFailure, in.OnLoginFailure); err != nil {
		return err
	}
	if err := s.settings.SetBool(ctx, settings.KeyNotifyOnNginxReloadFailure, in.OnNginxReloadFailure); err != nil {
		return err
	}
	threshold := in.IPFrequentThreshold
	if threshold <= 0 {
		threshold = DefaultIPFrequentThreshold
	}
	windowSec := in.IPFrequentWindowSec
	if windowSec <= 0 {
		windowSec = DefaultIPFrequentWindowSec
	}
	if err := s.settings.SetInt(ctx, settings.KeyNotifyIPFrequentThreshold, threshold); err != nil {
		return err
	}
	if err := s.settings.SetInt(ctx, settings.KeyNotifyIPFrequentWindowSec, windowSec); err != nil {
		return err
	}
	loginThreshold := in.LoginFailureThreshold
	if loginThreshold <= 0 {
		loginThreshold = DefaultLoginFailureThreshold
	}
	loginWindowSec := in.LoginFailureWindowSec
	if loginWindowSec <= 0 {
		loginWindowSec = DefaultLoginFailureWindowSec
	}
	if err := s.settings.SetInt(ctx, settings.KeyNotifyLoginFailureThreshold, loginThreshold); err != nil {
		return err
	}
	if err := s.settings.SetInt(ctx, settings.KeyNotifyLoginFailureWindowSec, loginWindowSec); err != nil {
		return err
	}

	email := in.Email
	email.HasPassword = email.HasPassword || strings.TrimSpace(in.SMTPPassword) != "" && in.SMTPPassword != MaskedSecret
	emailJSON, err := json.Marshal(email)
	if err != nil {
		return err
	}
	if err := s.settings.Set(ctx, settings.KeyNotifyEmailJSON, string(emailJSON)); err != nil {
		return err
	}

	webhook := in.Webhook
	secretValue := strings.TrimSpace(in.WebhookSecret)
	if secretValue != "" && secretValue != MaskedSecret {
		webhook.HasSecret = true
	} else if webhook.Provider == WebhookBark || webhook.Provider == WebhookGotify {
		if strings.TrimSpace(webhook.Key) != "" && webhook.Key != MaskedSecret {
			webhook.HasSecret = true
		}
	}
	webhookJSON, err := json.Marshal(webhook)
	if err != nil {
		return err
	}
	if err := s.settings.Set(ctx, settings.KeyNotifyWebhookJSON, string(webhookJSON)); err != nil {
		return err
	}

	telegram := in.Telegram
	normalizedProxyURL, err := normalizeProxyURL(telegram.ProxyURL)
	if err != nil {
		return err
	}
	telegram.ProxyURL = normalizedProxyURL
	if strings.TrimSpace(in.TelegramToken) != "" && in.TelegramToken != MaskedSecret {
		telegram.HasBotToken = true
	}
	telegramJSON, err := json.Marshal(telegram)
	if err != nil {
		return err
	}
	if err := s.settings.Set(ctx, settings.KeyNotifyTelegramJSON, string(telegramJSON)); err != nil {
		return err
	}

	if in.Type == NotifyTypeWebhook && webhook.Provider == WebhookCustom {
		if err := s.settings.Set(ctx, settings.KeyNotifyWebhookURL, strings.TrimSpace(webhook.URL)); err != nil {
			return err
		}
	}

	if err := s.saveSecret(ctx, settings.KeyNotifySMTPPassword, in.SMTPPassword, in.Email.HasPassword); err != nil {
		return err
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
	if err := s.saveSecret(ctx, settings.KeyNotifyWebhookSecret, webhookPlain, webhook.HasSecret); err != nil {
		return err
	}

	if err := s.saveSecret(ctx, settings.KeyNotifyTelegramToken, in.TelegramToken, telegram.HasBotToken); err != nil {
		return err
	}

	return nil
}

func (s *Store) saveSecret(ctx context.Context, key, plain string, hasExisting bool) error {
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
	return s.settings.Set(ctx, key, enc)
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
}

func ParseSaveInputFromMap(values map[string]string) (SaveInput, bool) {
	if !mapHasNotifySavePayload(values) {
		return SaveInput{}, false
	}

	in := SaveInput{
		Type:           NotifyType(values[settings.KeyNotifyType]),
		SMTPPassword:   values[settings.KeyNotifySMTPPassword],
		WebhookSecret:  values[settings.KeyNotifyWebhookSecret],
		TelegramToken:  values[settings.KeyNotifyTelegramToken],
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
	in.OnDDNSIPChange = boolFromMap(values, settings.KeyNotifyOnDDNSIPChange)
	in.OnDDNSFailure = boolFromMap(values, settings.KeyNotifyOnDDNSFailure)
	if !in.OnDDNSFailure {
		in.OnDDNSFailure = boolFromMap(values, settings.KeyNotifyOnDDNSError)
	}
	in.OnCertExpiry = boolFromMap(values, settings.KeyNotifyOnCertExpiry)
	if !in.OnCertExpiry {
		in.OnCertExpiry = boolFromMap(values, settings.KeyNotifyOnCertError)
	}
	in.OnCertRenewSuccess = boolFromMap(values, settings.KeyNotifyOnCertRenewSuccess)
	in.OnIPFrequentAccess = boolFromMap(values, settings.KeyNotifyOnIPFrequentAccess)
	in.OnCertRenewFailure = boolFromMap(values, settings.KeyNotifyOnCertRenewFailure)
	in.OnLoginFailure = boolFromMap(values, settings.KeyNotifyOnLoginFailure)
	in.OnNginxReloadFailure = boolFromMap(values, settings.KeyNotifyOnNginxReloadFailure)
	in.IPFrequentThreshold = intSettingFromMap(values, settings.KeyNotifyIPFrequentThreshold, DefaultIPFrequentThreshold)
	in.IPFrequentWindowSec = intSettingFromMap(values, settings.KeyNotifyIPFrequentWindowSec, DefaultIPFrequentWindowSec)
	in.LoginFailureThreshold = intSettingFromMap(values, settings.KeyNotifyLoginFailureThreshold, DefaultLoginFailureThreshold)
	in.LoginFailureWindowSec = intSettingFromMap(values, settings.KeyNotifyLoginFailureWindowSec, DefaultLoginFailureWindowSec)
	return in, true
}

func mapHasNotifySavePayload(values map[string]string) bool {
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
