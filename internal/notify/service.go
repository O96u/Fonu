package notify

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/fonu/fonu/internal/secret"
	"github.com/fonu/fonu/internal/settings"
)

type Service struct {
	settings *settings.Store
	store    *Store
	logger   *slog.Logger
	client   *http.Client

	accessMu      sync.Mutex
	accessHits    map[string][]time.Time
	accessAlerted map[string]time.Time

	loginMu      sync.Mutex
	loginHits    map[string][]time.Time
	loginAlerted map[string]time.Time
}

func New(settingsStore *settings.Store, secretBox *secret.Box, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		settings:      settingsStore,
		store:         NewStore(settingsStore, secretBox),
		logger:        logger,
		client:        &http.Client{Timeout: 10 * time.Second},
		accessHits:    map[string][]time.Time{},
		accessAlerted: map[string]time.Time{},
		loginHits:     map[string][]time.Time{},
		loginAlerted:  map[string]time.Time{},
	}
}

func (s *Service) Alert(ctx context.Context, event, title, message string) {
	if s == nil || s.store == nil {
		return
	}
	cfg, err := s.store.LoadRuntime(ctx)
	if err != nil {
		s.logger.Warn("notify load config failed", "error", err.Error())
		return
	}
	if !s.enabledFor(cfg, event) {
		return
	}
	if err := s.send(ctx, cfg, event, title, message); err != nil {
		s.logger.Warn("notify send failed", "event", event, "error", err.Error())
	}
}

func (s *Service) RecordAccessHit(ctx context.Context, ip string) {
	s.recordThresholdHit(ctx, ip, EventIPFrequentAccess, func(cfg RuntimeConfig) (threshold, windowSec, cooldownSec int, enabled bool) {
		enabled = s.enabledFor(cfg, EventIPFrequentAccess)
		threshold = cfg.IPFrequentThreshold
		if threshold <= 0 {
			threshold = DefaultIPFrequentThreshold
		}
		windowSec = cfg.IPFrequentWindowSec
		if windowSec <= 0 {
			windowSec = DefaultIPFrequentWindowSec
		}
		return threshold, windowSec, DefaultIPFrequentAlertCooldown, enabled
	}, &s.accessMu, s.accessHits, s.accessAlerted, func(ip string, windowSec, count, threshold int) string {
		return fmt.Sprintf("IP %s 在 %d 秒内访问 %d 次（阈值 %d）", ip, windowSec, count, threshold)
	}, "IP 频繁访问")
}

func (s *Service) RecordLoginFailure(ctx context.Context, ip string) {
	s.recordThresholdHit(ctx, ip, EventLoginFailure, func(cfg RuntimeConfig) (threshold, windowSec, cooldownSec int, enabled bool) {
		enabled = s.enabledFor(cfg, EventLoginFailure)
		threshold = cfg.LoginFailureThreshold
		if threshold <= 0 {
			threshold = DefaultLoginFailureThreshold
		}
		windowSec = cfg.LoginFailureWindowSec
		if windowSec <= 0 {
			windowSec = DefaultLoginFailureWindowSec
		}
		return threshold, windowSec, DefaultLoginFailureAlertCooldown, enabled
	}, &s.loginMu, s.loginHits, s.loginAlerted, func(ip string, windowSec, count, threshold int) string {
		return fmt.Sprintf("IP %s 在 %d 秒内登录失败 %d 次（阈值 %d）", ip, windowSec, count, threshold)
	}, "登录异常")
}

func (s *Service) recordThresholdHit(
	ctx context.Context,
	ip string,
	event string,
	cfgFn func(RuntimeConfig) (threshold, windowSec, cooldownSec int, enabled bool),
	mu *sync.Mutex,
	hits map[string][]time.Time,
	alerted map[string]time.Time,
	messageFn func(ip string, windowSec, count, threshold int) string,
	title string,
) {
	if s == nil || s.store == nil {
		return
	}
	ip = strings.TrimSpace(ip)
	if ip == "" || ip == "-" {
		return
	}

	cfg, err := s.store.LoadRuntime(ctx)
	if err != nil {
		return
	}
	threshold, windowSec, cooldownSec, enabled := cfgFn(cfg)
	if !enabled {
		return
	}

	window := time.Duration(windowSec) * time.Second
	now := time.Now()
	cutoff := now.Add(-window)

	mu.Lock()
	prev := hits[ip]
	pruned := make([]time.Time, 0, len(prev)+1)
	for _, at := range prev {
		if !at.Before(cutoff) {
			pruned = append(pruned, at)
		}
	}
	pruned = append(pruned, now)
	hits[ip] = pruned

	count := len(pruned)
	lastAlert := alerted[ip]
	cooldown := time.Duration(cooldownSec) * time.Second
	shouldAlert := count >= threshold && now.Sub(lastAlert) >= cooldown
	if shouldAlert {
		alerted[ip] = now
	}
	mu.Unlock()

	if !shouldAlert {
		return
	}
	s.Alert(ctx, event, title, messageFn(ip, windowSec, count, threshold))
}

func (s *Service) SendTest(ctx context.Context, in TestInput) error {
	runtime, err := s.resolveTestRuntime(ctx, in)
	if err != nil {
		return err
	}
	if runtime.Type == "" {
		return fmt.Errorf("请先选择通知方式")
	}
	if err := s.send(ctx, runtime, "test", "测试通知", "配置验证通过，当前通知通道可正常使用。"); err != nil {
		s.logger.Warn("notify test send failed", "error", err.Error())
		return err
	}
	return nil
}

func (s *Service) resolveTestRuntime(ctx context.Context, in TestInput) (RuntimeConfig, error) {
	smtpPassword := strings.TrimSpace(in.SMTPPassword)
	webhookSecret := strings.TrimSpace(in.WebhookSecret)
	telegramToken := strings.TrimSpace(in.TelegramToken)

	if key := strings.TrimSpace(in.Webhook.Key); key != "" && key != MaskedSecret {
		webhookSecret = key
	}

	needStored := smtpPassword == "" || smtpPassword == MaskedSecret ||
		webhookSecret == "" || webhookSecret == MaskedSecret ||
		telegramToken == "" || telegramToken == MaskedSecret

	if needStored && s.store != nil {
		stored, err := s.store.LoadRuntime(ctx)
		if err != nil {
			return RuntimeConfig{}, err
		}
		if (smtpPassword == "" || smtpPassword == MaskedSecret) && in.Email.HasPassword {
			smtpPassword = stored.SMTPPassword
		}
		if (webhookSecret == "" || webhookSecret == MaskedSecret) && in.Webhook.HasSecret {
			webhookSecret = stored.WebhookSecret
		}
		if (telegramToken == "" || telegramToken == MaskedSecret) && in.Telegram.HasBotToken {
			telegramToken = stored.TelegramToken
		}
	}

	return in.Config.Runtime(smtpPassword, webhookSecret, telegramToken), nil
}

func (s *Service) Store() *Store {
	if s == nil {
		return nil
	}
	return s.store
}

func (s *Service) alertTime(ctx context.Context) time.Time {
	if s == nil || s.settings == nil {
		return time.Now()
	}
	return s.settings.Now(ctx)
}

func (s *Service) send(ctx context.Context, cfg RuntimeConfig, event, title, message string) error {
	content := FormatAlertAt(event, title, message, s.alertTime(ctx))
	switch cfg.Type {
	case NotifyTypeEmail:
		return SendEmail(cfg, content.Subject, content.PlainBody)
	case NotifyTypeWebhook:
		return SendWebhook(ctx, s.client, cfg, content)
	case NotifyTypeTelegram:
		return SendTelegram(ctx, s.client, cfg, content)
	default:
		return nil
	}
}

func (s *Service) enabledFor(cfg RuntimeConfig, event string) bool {
	if cfg.Type == "" {
		return false
	}
	switch event {
	case EventDDNSIPChange:
		return cfg.OnDDNSIPChange
	case EventDDNSFailure:
		return cfg.OnDDNSFailure
	case EventCertExpiry:
		return cfg.OnCertExpiry
	case EventCertRenewSuccess:
		return cfg.OnCertRenewSuccess
	case EventCertRenewFailure:
		return cfg.OnCertRenewFailure
	case EventIPFrequentAccess:
		return cfg.OnIPFrequentAccess
	case EventLoginFailure:
		return cfg.OnLoginFailure
	case EventNginxReloadFailure:
		return cfg.OnNginxReloadFailure
	default:
		return true
	}
}
