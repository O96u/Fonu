package ddns

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/fonu/fonu/internal/notify"
	"github.com/fonu/fonu/internal/publicip"
	"github.com/fonu/fonu/internal/secret"
	"github.com/fonu/fonu/internal/settings"
)

type Service struct {
	store     *Store
	settings  *settings.Store
	secretBox *secret.Box
	providers map[string]Provider
	logger    *slog.Logger
	notify    *notify.Service
}

func NewService(store *Store, settings *settings.Store, secretBox *secret.Box, logger *slog.Logger, notifySvc *notify.Service) *Service {
	return &Service{
		store:     store,
		settings:  settings,
		secretBox: secretBox,
		providers: map[string]Provider{
			"cloudflare": NewCloudflare(),
			"dnspod":     NewDNSPod(),
			"alidns":     NewAliDNS(),
		},
		logger: logger.With("module", "DDNS"),
		notify: notifySvc,
	}
}

func (s *Service) List(ctx context.Context) ([]Config, error) {
	return s.store.List(ctx)
}

func (s *Service) Get(ctx context.Context, id int64) (Config, error) {
	return s.store.GetByID(ctx, id)
}

func (s *Service) CredentialsForDomain(ctx context.Context, rootDomain string) (string, Credentials, error) {
	cfg, err := s.store.GetByRootDomain(ctx, rootDomain)
	if err != nil {
		return "", Credentials{}, err
	}
	cred, err := s.loadCredentialsByID(ctx, cfg.ID)
	if err != nil {
		return cfg.Provider, Credentials{}, err
	}
	cred.Provider = cfg.Provider
	return cfg.Provider, cred, nil
}

func (s *Service) Create(ctx context.Context, in SaveInput) (Config, error) {
	if err := validateSaveInput(in); err != nil {
		return Config{}, err
	}
	if !in.HasCredentialUpdate() {
		return Config{}, fmt.Errorf("请填写 DNS API 凭证")
	}
	tokenEnc, err := s.encryptCredentials(in)
	if err != nil {
		return Config{}, err
	}
	return s.store.Create(ctx, in, tokenEnc)
}

func (s *Service) Update(ctx context.Context, id int64, in SaveInput) (Config, error) {
	if err := validateSaveInput(in); err != nil {
		return Config{}, err
	}
	existing, err := s.store.GetByID(ctx, id)
	if err != nil {
		return Config{}, err
	}
	tokenEnc := ""
	updateToken := in.HasCredentialUpdate()
	if updateToken {
		tokenEnc, err = s.encryptCredentials(in)
		if err != nil {
			return Config{}, err
		}
	} else if !existing.HasToken {
		return Config{}, fmt.Errorf("请填写 DNS API 凭证")
	}
	return s.store.Update(ctx, id, in, tokenEnc, updateToken)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.store.Delete(ctx, id)
}

type TestInput struct {
	ConfigID   int64
	Provider   string
	APIToken   string
	APITokenID string
	APISecret  string
}

func (s *Service) Test(ctx context.Context, in TestInput) error {
	providerName := strings.TrimSpace(in.Provider)
	cred := CredentialsFromSave(SaveInput{
		Provider:   providerName,
		APIToken:   in.APIToken,
		APITokenID: in.APITokenID,
		APISecret:  in.APISecret,
	})

	if cred.HasValues() {
		if providerName == "" {
			providerName = "cloudflare"
		}
		cred.Provider = providerName
		return s.providerFor(providerName).Verify(ctx, cred)
	}

	if in.ConfigID > 0 {
		cfg, err := s.store.GetByID(ctx, in.ConfigID)
		if err != nil {
			return err
		}
		cred, err = s.loadCredentialsByID(ctx, in.ConfigID)
		if err != nil {
			return err
		}
		cred.Provider = cfg.Provider
		return s.providerFor(cfg.Provider).Verify(ctx, cred)
	}

	return fmt.Errorf("请填写 DNS API 凭证")
}

func (s *Service) UpdateNow(ctx context.Context, id int64) (Config, error) {
	cfg, err := s.store.GetByID(ctx, id)
	if err != nil {
		return Config{}, err
	}
	if !cfg.Enabled {
		return Config{}, fmt.Errorf("该 DDNS 配置未启用")
	}
	if err := s.runUpdate(ctx, cfg); err != nil {
		return Config{}, err
	}
	return s.store.GetByID(ctx, id)
}

func (s *Service) UpdateAll(ctx context.Context) ([]Config, error) {
	configs, err := s.store.List(ctx)
	if err != nil {
		return nil, err
	}
	var lastErr error
	for _, cfg := range configs {
		if !cfg.Enabled {
			continue
		}
		if err := s.runUpdate(ctx, cfg); err != nil {
			lastErr = err
			s.logger.Error("ddns update failed", "domain", cfg.RootDomain, "error", err.Error())
		}
	}
	list, listErr := s.store.List(ctx)
	if listErr != nil {
		return nil, listErr
	}
	if lastErr != nil {
		return list, lastErr
	}
	return list, nil
}

func (s *Service) Tick(ctx context.Context) {
	configs, err := s.store.List(ctx)
	if err != nil {
		return
	}
	for _, cfg := range configs {
		if !cfg.Enabled {
			continue
		}
		if err := s.runUpdate(ctx, cfg); err != nil {
			s.logger.Error("ddns update failed", "domain", cfg.RootDomain, "error", err.Error())
			if s.notify != nil {
				s.notify.Alert(ctx, notify.EventDDNSError, "DDNS 更新失败", cfg.RootDomain+": "+err.Error())
			}
		}
	}
}

func (s *Service) Summary(ctx context.Context) (status string, lastUpdated string, count int) {
	configs, err := s.store.List(ctx)
	if err != nil || len(configs) == 0 {
		return "disabled", "", 0
	}
	count = len(configs)
	status = "ok"
	hasEnabled := false
	var latest *Config
	for _, cfg := range configs {
		if cfg.Enabled {
			hasEnabled = true
		}
		if latest == nil || (cfg.LastUpdatedAt != nil && (latest.LastUpdatedAt == nil || cfg.LastUpdatedAt.After(*latest.LastUpdatedAt))) {
			latest = &cfg
		}
		if cfg.LastStatus == "error" {
			status = "error"
		}
	}
	if !hasEnabled {
		return "disabled", "", count
	}
	if latest != nil && latest.LastUpdatedAt != nil {
		lastUpdated = latest.LastUpdatedAt.UTC().Format(time.RFC3339)
	}
	if status != "error" && latest != nil && latest.LastStatus != "" {
		status = latest.LastStatus
	}
	return status, lastUpdated, count
}

func (s *Service) runUpdate(ctx context.Context, cfg Config) error {
	cred, err := s.loadCredentialsByID(ctx, cfg.ID)
	if err != nil {
		_ = s.store.UpdateStatus(ctx, cfg.ID, cfg.LastIPv4, cfg.LastIPv6, "error", err.Error())
		return err
	}
	cred.Provider = cfg.Provider
	provider := s.providerFor(cfg.Provider)

	ipv4, ipv6, err := publicip.Detect(ctx)
	if err != nil {
		_ = s.store.UpdateStatus(ctx, cfg.ID, cfg.LastIPv4, cfg.LastIPv6, "error", err.Error())
		return err
	}

	changed := false
	if cfg.IPv4Enabled && ipv4 != "" && ipv4 != cfg.LastIPv4 {
		current, _ := provider.GetRecordIP(ctx, cred, cfg.RootDomain, cfg.RecordName, "A")
		if current != ipv4 {
			if err := provider.UpdateRecord(ctx, cred, cfg.RootDomain, cfg.RecordName, "A", ipv4); err != nil {
				_ = s.store.UpdateStatus(ctx, cfg.ID, ipv4, ipv6, "error", err.Error())
				return err
			}
			changed = true
			s.logger.Info("ddns ipv4 updated", "domain", cfg.RootDomain, "ip", ipv4)
		}
	}
	if cfg.IPv6Enabled && ipv6 != "" && ipv6 != cfg.LastIPv6 {
		current, _ := provider.GetRecordIP(ctx, cred, cfg.RootDomain, cfg.RecordName, "AAAA")
		if current != ipv6 {
			if err := provider.UpdateRecord(ctx, cred, cfg.RootDomain, cfg.RecordName, "AAAA", ipv6); err != nil {
				_ = s.store.UpdateStatus(ctx, cfg.ID, ipv4, ipv6, "error", err.Error())
				return err
			}
			changed = true
			s.logger.Info("ddns ipv6 updated", "domain", cfg.RootDomain, "ip", ipv6)
		}
	}

	status := "ok"
	if !changed {
		status = "ok"
	}
	_ = s.store.UpdateStatus(ctx, cfg.ID, ipv4, ipv6, status, "")
	return nil
}

func (s *Service) encryptCredentials(in SaveInput) (string, error) {
	cred := CredentialsFromSave(in)
	if err := cred.Validate(in.Provider); err != nil {
		return "", err
	}
	raw, err := cred.Marshal()
	if err != nil {
		return "", err
	}
	return s.secretBox.Encrypt(string(raw))
}

func (s *Service) loadCredentialsByID(ctx context.Context, id int64) (Credentials, error) {
	enc, err := s.store.GetTokenByID(ctx, id)
	if err != nil || enc == "" {
		return Credentials{}, fmt.Errorf("DNS API 凭证未配置")
	}
	raw, err := s.secretBox.Decrypt(enc)
	if err != nil {
		return Credentials{}, err
	}
	return ParseCredentials(raw)
}

func (s *Service) providerFor(name string) Provider {
	if p, ok := s.providers[name]; ok {
		return p
	}
	return s.providers["cloudflare"]
}

func validateSaveInput(in SaveInput) error {
	if strings.TrimSpace(in.RootDomain) == "" {
		return fmt.Errorf("主域名不能为空")
	}
	switch strings.TrimSpace(in.Provider) {
	case "dnspod", "cloudflare", "alidns", "":
	default:
		return fmt.Errorf("不支持的 DNS Provider")
	}
	return nil
}
