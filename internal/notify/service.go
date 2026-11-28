package notify

import (
	"context"
	"database/sql"
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

 securityMu sync.Mutex
 securityState attackState
 accessConfigMu sync.Mutex
 accessConfig RuntimeConfig
 accessConfigUntil time.Time
 alertSlots chan struct{}
 historyMu sync.Mutex
 history []Delivery
 historyDB *sql.DB

	loginMu      sync.Mutex
	loginHits    map[string][]time.Time
	loginAlerted map[string]time.Time
}

func New(settingsStore *settings.Store, secretBox *secret.Box, logger *slog.Logger, historyDB ...*sql.DB) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	svc := &Service{
		settings:      settingsStore,
		store:         NewStore(settingsStore, secretBox),
		logger:        logger,
		client:        &http.Client{Timeout: 10 * time.Second},
		accessHits:    map[string][]time.Time{},
		accessAlerted: map[string]time.Time{},
		loginHits:     map[string][]time.Time{},
		loginAlerted:  map[string]time.Time{},
 alertSlots:make(chan struct{},4),
	}
	if len(historyDB)>0 {svc.historyDB=historyDB[0]}
	return svc
}

// Delivery records contain only event metadata, never message text or credentials.
type Delivery struct{
 At string `json:"at"`
 Event string `json:"event"`
 Channel string `json:"channel"`
 State string `json:"state"`
 Detail string `json:"detail"`
}
func (s *Service) record(event, channel, state, detail string) {
	row := Delivery{At: time.Now().UTC().Format(time.RFC3339Nano), Event: event, Channel: channel, State: state, Detail: detail}
	s.logger.Info("notification delivery", "event", event, "channel", channel, "state", state, "detail", detail)
	if s.historyDB != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := insertDelivery(ctx, s.historyDB, row); err != nil {
			// Do not retry a delivered message just because the audit write failed.
			s.logger.Warn("notification history write failed", "error", err.Error())
		}
		return
	}
	// In-memory fallback for standalone services without a database.
	s.historyMu.Lock()
	defer s.historyMu.Unlock()
	if len(s.history) >= 100 {
		copy(s.history, s.history[1:])
		s.history = s.history[:99]
	}
	s.history = append(s.history, row)
}

func (s *Service) HistoryContext(ctx context.Context) ([]Delivery, error) {
	if s.historyDB != nil {
		return listDeliveries(ctx, s.historyDB)
	}
	s.historyMu.Lock()
	defer s.historyMu.Unlock()
	out := make([]Delivery, len(s.history))
	for i := range s.history {
		out[i] = s.history[len(s.history)-1-i]
	}
	return out, nil
}

func (s *Service) History() []Delivery {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	rows, err := s.HistoryContext(ctx)
	if err != nil {
		s.logger.Warn("notification history read failed", "error", err.Error())
	}
	return rows
}

func deliveryChannel(cfg RuntimeConfig) string {
	if cfg.Type == NotifyTypeWebhook {
		return string(cfg.Webhook.Provider)
	}
	return string(cfg.Type)
}

func(s *Service) InvalidateConfig(){
 s.accessConfigMu.Lock();s.accessConfigUntil=time.Time{};s.accessConfigMu.Unlock()
 // A new threshold or event toggle must not inherit old cooldown/count state.
 s.accessMu.Lock();clear(s.accessHits);clear(s.accessAlerted);s.accessMu.Unlock()
 s.loginMu.Lock();clear(s.loginHits);clear(s.loginAlerted);s.loginMu.Unlock()
 s.securityMu.Lock();s.securityState=attackState{};s.securityMu.Unlock()
}
func(s *Service) Alert(ctx context.Context,event,title,message string){
 if s==nil||s.store==nil{return}
 // An event emitted just before a scheduled/manual context expires still deserves
 // an independent bounded delivery attempt.
 sendCtx,cancel:=context.WithTimeout(context.WithoutCancel(ctx),20*time.Second);defer cancel()
 cfg,err:=s.store.LoadRuntime(sendCtx)
 if err!=nil{s.record(event,"","failed","无法读取通知配置或解密凭证，请检查已保存的设置");return}
 channel:=deliveryChannel(cfg)
 if cfg.Type==""{s.record(event,channel,"skipped","通知通道未启用，本次未发送");return}
 if !s.enabledFor(cfg,event){s.record(event,channel,"skipped","该事件未启用或无法识别，本次未发送");return}
 if err:=s.send(sendCtx,cfg,event,title,message);err!=nil{s.record(event,channel,"failed","通道未确认接收，请检查凭证、网络及通知服务状态");s.logger.Warn("notify send failed","event",event,"type",channel,"error",err.Error());return}
 s.record(event,channel,"sent","通知通道已确认接收")
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

	cfg, err := s.cachedAccessConfig(ctx)
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
 if len(hits)>=4096 {for key,values:=range hits{if len(values)==0||values[len(values)-1].Before(cutoff){delete(hits,key);delete(alerted,key)}};if _,ok:=hits[ip];!ok&&len(hits)>=4096{mu.Unlock();return}}
	prev := hits[ip]
	// Hit timestamps are ordered. Reuse the existing buffer instead of
	// allocating and copying the entire window on every access.
	first := 0
	for first < len(prev) && prev[first].Before(cutoff) {
		first++
	}
	if keep := len(prev)-first; keep >= threshold {
		first = len(prev)-threshold+1
	}
	pruned := prev[:copy(prev, prev[first:])]
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
	s.enqueueAlert(event,title,messageFn(ip, windowSec, count, threshold))
}

func (s *Service) SendTest(ctx context.Context, in TestInput) error {
	runtime, err := s.resolveTestRuntime(ctx, in)
	if err != nil {
		channel := string(in.Type)
		if in.Type == NotifyTypeWebhook {channel=string(in.Webhook.Provider)}
		s.record("test",channel,"failed","测试配置校验失败，请检查通道参数及凭证")
		return err
	}
	if runtime.Type == "" {
		return fmt.Errorf("请先选择通知方式")
	}
	if err := s.send(ctx, runtime, "test", "测试通知", "这是一条手动发送的测试消息。收到此消息说明当前通知通道已成功接收本次请求；正式事件仍以已保存的通知设置为准。"); err != nil {
 s.record("test",deliveryChannel(runtime),"failed","测试消息未获通道确认，请检查当前表单配置")
		s.logger.Warn("notify test send failed", "error", err.Error())
		return err
	}
 s.record("test",deliveryChannel(runtime),"sent","测试消息已获通道确认")
	return nil
}

func (s *Service) resolveTestRuntime(ctx context.Context, in TestInput) (RuntimeConfig, error) {
	smtpPassword := strings.TrimSpace(in.SMTPPassword)
	webhookSecret := strings.TrimSpace(in.WebhookSecret)
	telegramToken := strings.TrimSpace(in.TelegramToken)

	if key := strings.TrimSpace(in.Webhook.Key); key != "" && key != MaskedSecret {
		webhookSecret = key
	}

	needStored := in.Type==NotifyTypeWebhook || smtpPassword == "" || smtpPassword == MaskedSecret ||
		webhookSecret == "" || webhookSecret == MaskedSecret ||
		telegramToken == "" || telegramToken == MaskedSecret

	if needStored && s.store != nil {
		stored, err := s.store.loadRuntimeForEdit(ctx)
		if err != nil {
			return RuntimeConfig{}, err
		}
		if (smtpPassword == "" || smtpPassword == MaskedSecret) && in.Email.HasPassword {
			smtpPassword = stored.SMTPPassword
		}
		if in.Type==NotifyTypeWebhook {
            webhookSecret,err=resolveWebhookCredential(in.Webhook,webhookSecret,stored.Webhook,webhookSecretFromRuntime(stored))
            if err!=nil{return RuntimeConfig{},err}
        }
		if (telegramToken == "" || telegramToken == MaskedSecret) && in.Telegram.HasBotToken {
			telegramToken = stored.TelegramToken
		}
	}

	if in.Type==NotifyTypeWebhook {if err:=validateWebhookCredential(in.Webhook,webhookSecret);err!=nil{return RuntimeConfig{},err}}
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
		return sendEmailContext(ctx, cfg, content.Subject, content.PlainBody)
	case NotifyTypeWebhook:
		return SendWebhook(ctx, s.client, cfg, content)
	case NotifyTypeTelegram:
		return SendTelegram(ctx, s.client, cfg, content)
	default:
		return fmt.Errorf("未配置有效的通知方式")
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
 case EventScan:return cfg.OnScan
 case EventCC:return cfg.OnCC
	case EventLoginFailure:
		return cfg.OnLoginFailure
	case EventNginxReloadFailure:
		return cfg.OnNginxReloadFailure
	default:
		return false
	}
}
