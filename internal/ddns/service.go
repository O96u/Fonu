package ddns

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

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

func (s *Service) Get(ctx context.Context) (Config, error) {
	return s.store.Get(ctx)
}

func (s *Service) Credentials(ctx context.Context) (string, Credentials, error) {
	cfg, err := s.store.Get(ctx)
	if err != nil {
		return "", Credentials{}, err
	}
	cred, err := s.loadCredentials(ctx)
	if err != nil {
		return cfg.Provider, Credentials{}, err
	}
	cred.Provider = cfg.Provider
	return cfg.Provider, cred, nil
}

func (s *Service) Save(ctx context.Context, in SaveInput) (Config, error) {
	if err := validateSaveInput(in); err != nil {
		return Config{}, err
	}
	tokenEnc := ""
	if in.HasCredentialUpdate() {
		cred := CredentialsFromSave(in)
		if err := cred.Validate(in.Provider); err != nil {
			return Config{}, err
		}
		raw, err := cred.Marshal()
		if err != nil {
			return Config{}, err
		}
		tokenEnc, err = s.secretBox.Encrypt(string(raw))
		if err != nil {
			return Config{}, err
		}
	}
	return s.store.Save(ctx, in, tokenEnc)
}

type TestInput struct {
	Provider   string
	APIToken   string
	APITokenID string
	APISecret  string
}

func (s *Service) Test(ctx context.Context, in TestInput) error {
	cfg, err := s.store.Get(ctx)
	providerName := strings.TrimSpace(in.Provider)
	if providerName == "" && err == nil {
		providerName = cfg.Provider
	}
	if providerName == "" {
		providerName = "cloudflare"
	}

	cred := CredentialsFromSave(SaveInput{
		Provider:   providerName,
		APIToken:   in.APIToken,
		APITokenID: in.APITokenID,
		APISecret:  in.APISecret,
	})
	if !cred.HasValues() {
		cred, err = s.loadCredentials(ctx)
		if err != nil {
			return err
		}
	}
	cred.Provider = providerName
	return s.providerFor(providerName).Verify(ctx, cred)
}

func (s *Service) UpdateNow(ctx context.Context) error {
	cfg, err := s.store.Get(ctx)
	if err != nil {
		return err
	}
	if !cfg.Enabled {
		return fmt.Errorf("DDNS 未启用")
	}
	return s.runUpdate(ctx, cfg)
}

func (s *Service) Tick(ctx context.Context) {
	cfg, err := s.store.Get(ctx)
	if err != nil || !cfg.Enabled {
		return
	}
	if err := s.runUpdate(ctx, cfg); err != nil {
		s.logger.Error("ddns update failed", "error", err.Error())
		if s.notify != nil {
			s.notify.Alert(ctx, notify.EventDDNSError, "DDNS 更新失败", err.Error())
		}
	}
}

func (s *Service) runUpdate(ctx context.Context, cfg Config) error {
	cred, err := s.loadCredentials(ctx)
	if err != nil {
		_ = s.store.UpdateStatus(ctx, cfg.LastIPv4, cfg.LastIPv6, "error", err.Error())
		return err
	}
	cred.Provider = cfg.Provider
	provider := s.providerFor(cfg.Provider)

	ipv4, ipv6, err := publicip.Detect(ctx)
	if err != nil {
		_ = s.store.UpdateStatus(ctx, cfg.LastIPv4, cfg.LastIPv6, "error", err.Error())
		return err
	}

	changed := false
	if cfg.IPv4Enabled && ipv4 != "" && ipv4 != cfg.LastIPv4 {
		current, _ := provider.GetRecordIP(ctx, cred, cfg.RootDomain, cfg.RecordName, "A")
		if current != ipv4 {
			if err := provider.UpdateRecord(ctx, cred, cfg.RootDomain, cfg.RecordName, "A", ipv4); err != nil {
				_ = s.store.UpdateStatus(ctx, ipv4, ipv6, "error", err.Error())
				return err
			}
			changed = true
			s.logger.Info("ddns ipv4 updated", "ip", ipv4)
		}
	}
	if cfg.IPv6Enabled && ipv6 != "" && ipv6 != cfg.LastIPv6 {
		current, _ := provider.GetRecordIP(ctx, cred, cfg.RootDomain, cfg.RecordName, "AAAA")
		if current != ipv6 {
			if err := provider.UpdateRecord(ctx, cred, cfg.RootDomain, cfg.RecordName, "AAAA", ipv6); err != nil {
				_ = s.store.UpdateStatus(ctx, ipv4, ipv6, "error", err.Error())
				return err
			}
			changed = true
			s.logger.Info("ddns ipv6 updated", "ip", ipv6)
		}
	}

	status := "ok"
	if !changed {
		status = "ok"
	}
	_ = s.store.UpdateStatus(ctx, ipv4, ipv6, status, "")
	return nil
}

func (s *Service) loadCredentials(ctx context.Context) (Credentials, error) {
	enc, err := s.store.GetToken(ctx)
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
