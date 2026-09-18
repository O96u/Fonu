package notify

const (
	EventDDNSIPChange       = "ddns_ip_change"
	EventDDNSFailure        = "ddns_failure"
	EventCertExpiry         = "cert_expiry"
	EventCertRenewSuccess   = "cert_renew_success"
	EventCertRenewFailure   = "cert_renew_failure"
	EventIPFrequentAccess   = "ip_frequent_access"
	EventLoginFailure       = "login_failure"
	EventNginxReloadFailure = "nginx_reload_failure"

	MaskedSecret = "********"

	DefaultBarkServer               = "https://api.day.app"
	DefaultIPFrequentThreshold      = 100
	DefaultIPFrequentWindowSec      = 60
	DefaultIPFrequentAlertCooldown  = 15 * 60 // seconds
	DefaultLoginFailureThreshold    = 5
	DefaultLoginFailureWindowSec    = 300
	DefaultLoginFailureAlertCooldown = 15 * 60 // seconds
)

type NotifyType string

const (
	NotifyTypeEmail    NotifyType = "email"
	NotifyTypeWebhook  NotifyType = "webhook"
	NotifyTypeTelegram NotifyType = "telegram"
)

type WebhookProvider string

const (
	WebhookBark   WebhookProvider = "bark"
	WebhookNtfy   WebhookProvider = "ntfy"
	WebhookGotify WebhookProvider = "gotify"
	WebhookCustom WebhookProvider = "custom"
)

type EmailConfig struct {
	Host        string   `json:"host"`
	Port        int      `json:"port"`
	Username    string   `json:"username"`
	From        string   `json:"from"`
	To          []string `json:"to"`
	TLS         bool     `json:"tls"`
	HasPassword bool     `json:"has_password"`
}

type WebhookConfig struct {
	Provider  WebhookProvider `json:"provider"`
	Server    string          `json:"server"`
	Key       string          `json:"key"`
	Topic     string          `json:"topic"`
	URL       string          `json:"url"`
	HasSecret bool            `json:"has_secret"`
}

type TelegramConfig struct {
	ChatID      string `json:"chat_id"`
	ProxyURL    string `json:"proxy_url"`
	HasBotToken bool   `json:"has_bot_token"`
}

type Config struct {
	Type                  NotifyType    `json:"type"`
	Email                 EmailConfig   `json:"email"`
	Webhook               WebhookConfig `json:"webhook"`
	Telegram              TelegramConfig `json:"telegram"`
	OnDDNSIPChange         bool `json:"on_ddns_ip_change"`
	OnDDNSFailure          bool `json:"on_ddns_failure"`
	OnCertExpiry           bool `json:"on_cert_expiry"`
	OnCertRenewSuccess     bool `json:"on_cert_renew_success"`
	OnCertRenewFailure     bool `json:"on_cert_renew_failure"`
	OnIPFrequentAccess     bool `json:"on_ip_frequent_access"`
	OnLoginFailure         bool `json:"on_login_failure"`
	OnNginxReloadFailure   bool `json:"on_nginx_reload_failure"`
	IPFrequentThreshold    int  `json:"ip_frequent_threshold"`
	IPFrequentWindowSec    int  `json:"ip_frequent_window_sec"`
	LoginFailureThreshold  int  `json:"login_failure_threshold"`
	LoginFailureWindowSec  int  `json:"login_failure_window_sec"`
}

type RuntimeConfig struct {
	Config
	SMTPPassword  string
	WebhookSecret string
	TelegramToken string
}

type SaveInput struct {
	Type                NotifyType
	Email               EmailConfig
	SMTPPassword        string
	Webhook             WebhookConfig
	WebhookSecret       string
	Telegram            TelegramConfig
	TelegramToken       string
	OnDDNSIPChange         bool
	OnDDNSFailure          bool
	OnCertExpiry           bool
	OnCertRenewSuccess     bool
	OnCertRenewFailure     bool
	OnIPFrequentAccess     bool
	OnLoginFailure         bool
	OnNginxReloadFailure   bool
	IPFrequentThreshold    int
	IPFrequentWindowSec    int
	LoginFailureThreshold  int
	LoginFailureWindowSec  int
}

type TestInput struct {
	Config
	SMTPPassword  string `json:"smtp_password"`
	WebhookSecret string `json:"webhook_secret"`
	TelegramToken string `json:"telegram_token"`
}

func (c Config) Runtime(password, webhookSecret, telegramToken string) RuntimeConfig {
	return RuntimeConfig{
		Config:        c,
		SMTPPassword:  password,
		WebhookSecret: webhookSecret,
		TelegramToken: telegramToken,
	}
}
