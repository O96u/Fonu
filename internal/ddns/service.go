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
			"cloudflare":   NewCloudflare(),
			"dnspod":       NewDNSPod(),
			"alidns":       NewAliDNS(),
			"tencentcloud": NewTencentCloud(),
		},
		logger: logger.With("module", "DDNS"),
		notify: notifySvc,
	}
}

func (s *Service) List(ctx context.Context) ([]Config, error) {
	configs, err := s.store.List(ctx)
	if err != nil {
		return nil, err
	}
	for i := range configs {
		s.enrichDomainRecords(ctx, &configs[i])
		sanitizeConfig(&configs[i])
	}
	return configs, nil
}

func (s *Service) ListLite(ctx context.Context) ([]Config, error) {
	configs, err := s.store.List(ctx)
	if err != nil {
		return nil, err
	}
	for i := range configs {
		if len(configs[i].DomainRecords) == 0 {
			configs[i].DomainRecords = BuildDomainRecords(configs[i])
		}
		sanitizeConfig(&configs[i])
	}
	return configs, nil
}

func (s *Service) Get(ctx context.Context, id int64) (Config, error) {
	cfg, err := s.store.GetByID(ctx, id)
	if err != nil {
		return Config{}, err
	}
	s.enrichDomainRecords(ctx, &cfg)
	sanitizeConfig(&cfg)
	return cfg, nil
}

func (s *Service) CredentialsForDomain(ctx context.Context, rootDomain string) (string, Credentials, error) {
	cfg, err := s.store.GetByRootDomain(ctx, rootDomain)
	if err != nil {
		return "", Credentials{}, err
	}
	return s.credentialsForConfig(ctx, cfg)
}

func (s *Service) CredentialsForConfigID(ctx context.Context, id int64) (Config, string, Credentials, error) {
	cfg, err := s.Get(ctx, id)
	if err != nil {
		return Config{}, "", Credentials{}, fmt.Errorf("未找到所选 DNS 配置")
	}
	provider, cred, err := s.credentialsForConfig(ctx, cfg)
	if err != nil {
		return Config{}, "", Credentials{}, err
	}
	return cfg, provider, cred, nil
}

func (s *Service) ConfigForDNSZone(ctx context.Context, dnsZone string) (Config, error) {
	cfg, err := s.store.GetByRootDomain(ctx, dnsZone)
	if err != nil {
		return Config{}, err
	}
	if err := requireEnabled(cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (s *Service) ConfigForAnyDomain(ctx context.Context, domain string) (Config, error) {
	domain = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(domain, "*.")))
	configs, err := s.store.List(ctx)
	if err != nil {
		return Config{}, err
	}
	for _, cfg := range configs {
		if !cfg.Enabled {
			continue
		}
		if cfg.CoversDomain(domain) {
			return cfg, nil
		}
	}
	return Config{}, fmt.Errorf("未找到域名 %s 的 DDNS 配置", domain)
}

func (s *Service) CredentialsForAnyDomain(ctx context.Context, domain string) (string, Credentials, error) {
	cfg, err := s.ConfigForAnyDomain(ctx, domain)
	if err != nil {
		return "", Credentials{}, fmt.Errorf("请先在 DDNS 页面配置域名 %s 的 DNS 凭证", strings.TrimPrefix(domain, "*."))
	}
	return s.credentialsForConfig(ctx, cfg)
}

func requireEnabled(cfg Config) error {
	if !cfg.Enabled {
		return fmt.Errorf("DNS 任务 %s 未启用，请先在 DDNS 页面开启", cfg.RootDomain)
	}
	return nil
}

func (s *Service) credentialsForConfig(ctx context.Context, cfg Config) (string, Credentials, error) {
	if err := requireEnabled(cfg); err != nil {
		return cfg.Provider, Credentials{}, err
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
	return s.Get(ctx, id)
}

func (s *Service) UpdateAll(ctx context.Context) ([]Config, error) {
	configs, err := s.store.List(ctx)
	if err != nil {
		return nil, err
	}
	var lastErr error
	var failCount int
	var runCount int
	for _, cfg := range configs {
		if !cfg.Enabled {
			continue
		}
		runCount++
		if err := s.runUpdate(ctx, cfg); err != nil {
			lastErr = err
			failCount++
			s.logger.Error("ddns update failed", "domain", cfg.RootDomain, "error", err.Error())
		}
	}
	list, listErr := s.List(ctx)
	if listErr != nil {
		return nil, listErr
	}
	if failCount > 0 && failCount == runCount {
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

// PublicIPs returns the most recently synced public IPs from enabled DDNS configs.
func (s *Service) PublicIPs(ctx context.Context) (ipv4, ipv6 string, ok bool) {
	return s.latestStoredPublicIPs(ctx, true)
}

// LastKnownPublicIPs returns stored public IPs from the most recently updated config,
// including paused tasks. Used to avoid live IP detection when DDNS is temporarily disabled.
func (s *Service) LastKnownPublicIPs(ctx context.Context) (ipv4, ipv6 string, ok bool) {
	return s.latestStoredPublicIPs(ctx, false)
}

func (s *Service) latestStoredPublicIPs(ctx context.Context, enabledOnly bool) (ipv4, ipv6 string, ok bool) {
	configs, err := s.store.List(ctx)
	if err != nil || len(configs) == 0 {
		return "", "", false
	}
	var latest *Config
	for _, cfg := range configs {
		if enabledOnly && !cfg.Enabled {
			continue
		}
		if latest == nil || (cfg.LastUpdatedAt != nil && (latest.LastUpdatedAt == nil || cfg.LastUpdatedAt.After(*latest.LastUpdatedAt))) {
			latest = &cfg
		}
	}
	if latest == nil {
		return "", "", false
	}
	if latest.LastIPv4 == "" && latest.LastIPv6 == "" {
		return "", "", false
	}
	return latest.LastIPv4, latest.LastIPv6, true
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

func sanitizeConfig(cfg *Config) {
	if !publicip.IsPublicIPv4(cfg.LastIPv4) {
		cfg.LastIPv4 = ""
	}
	if !publicip.IsPublicIPv6(cfg.LastIPv6) {
		cfg.LastIPv6 = ""
	}
}

func (s *Service) enrichDomainRecords(ctx context.Context, cfg *Config) {
	records := BuildDomainRecords(*cfg)
	if !cfg.HasToken || len(records) == 0 {
		cfg.DomainRecords = records
		return
	}
	cred, err := s.loadCredentialsByID(ctx, cfg.ID)
	if err != nil {
		cfg.DomainRecords = records
		return
	}
	cred.Provider = cfg.Provider
	provider := s.providerFor(cfg.Provider)

	for i := range records {
		prevIPv4 := records[i].IPv4
		prevIPv6 := records[i].IPv6
		prevStatus := records[i].Status
		prevMessage := records[i].Message

		root, record := ResolveDomainTarget(cfg.RootDomain, records[i].Domain)
		records[i].Domain = FormatDomainLine(root, record)

		if cfg.IPv4Enabled {
			ip, err := provider.GetRecordIP(ctx, cred, root, record, "A")
			switch {
			case err != nil:
				records[i].IPv4 = prevIPv4
				if prevStatus == "" || prevStatus == "unknown" {
					records[i].Status = "error"
					records[i].Message = err.Error()
				}
			case ip != "":
				records[i].IPv4 = ip
				if prevStatus == "" || prevStatus == "unknown" {
					records[i].Status = "unchanged"
					records[i].Message = "记录未变化"
				} else {
					records[i].Status = prevStatus
					records[i].Message = prevMessage
				}
			default:
				records[i].IPv4 = prevIPv4
			}
		}
		if cfg.IPv6Enabled {
			ip, err := provider.GetRecordIP(ctx, cred, root, record, "AAAA")
			switch {
			case err != nil:
				records[i].IPv6 = prevIPv6
				if records[i].Status == "" || records[i].Status == "unknown" {
					records[i].Status = "error"
					records[i].Message = err.Error()
				}
			case ip != "":
				records[i].IPv6 = ip
			default:
				records[i].IPv6 = prevIPv6
			}
		}
		if records[i].Status == "" {
			records[i].Status = "unknown"
		}
	}
	cfg.DomainRecords = records
}

func (s *Service) runUpdate(ctx context.Context, cfg Config) error {
	cred, err := s.loadCredentialsByID(ctx, cfg.ID)
	if err != nil {
		_ = s.store.UpdateStatus(ctx, cfg.ID, cfg.LastIPv4, cfg.LastIPv6, "error", err.Error())
		return err
	}
	cred.Provider = cfg.Provider
	provider := s.providerFor(cfg.Provider)

	detectedIPv4, detectedIPv6, _ := publicip.Detect(ctx)
	if detectedIPv4 != "" && !publicip.IsPublicIPv4(detectedIPv4) {
		detectedIPv4 = ""
	}
	if detectedIPv6 != "" && !publicip.IsPublicIPv6(detectedIPv6) {
		detectedIPv6 = ""
	}
	storeIPv4 := cfg.LastIPv4
	storeIPv6 := cfg.LastIPv6
	if !publicip.IsPublicIPv4(storeIPv4) {
		storeIPv4 = ""
	}
	if !publicip.IsPublicIPv6(storeIPv6) {
		storeIPv6 = ""
	}
	if cfg.IPv4Enabled && detectedIPv4 != "" {
		storeIPv4 = detectedIPv4
	}
	if cfg.IPv6Enabled && detectedIPv6 != "" {
		storeIPv6 = detectedIPv6
	}

	if cfg.IPv4Enabled && detectedIPv4 == "" {
		msg := "无法获取公网 IPv4，已跳过更新（不会写入内网 IP）"
		_ = s.store.UpdateStatus(ctx, cfg.ID, storeIPv4, storeIPv6, "error", msg)
		return fmt.Errorf(msg)
	}
	if cfg.IPv6Enabled && detectedIPv6 == "" {
		msg := "无法获取公网 IPv6，已跳过更新"
		_ = s.store.UpdateStatus(ctx, cfg.ID, storeIPv4, storeIPv6, "error", msg)
		return fmt.Errorf(msg)
	}

	recordNames := cfg.RecordNamesList()
	domainRecords := make([]DomainRecord, 0, len(recordNames))
	var lastErr error
	var errCount, okCount int
	changed := false

	for _, name := range recordNames {
		root, record := ResolveDomainTarget(cfg.RootDomain, name)
		fqdn := FormatDomainLine(root, record)
		dr := DomainRecord{Domain: fqdn}

		if cfg.IPv4Enabled && detectedIPv4 != "" {
			current, getErr := provider.GetRecordIP(ctx, cred, root, record, "A")
			if getErr != nil {
				dr.Status = "error"
				dr.Message = getErr.Error()
				lastErr = getErr
				domainRecords = append(domainRecords, dr)
				continue
			}
			dr.IPv4 = current
			if current == detectedIPv4 {
				dr.Status = "unchanged"
				dr.Message = "记录未变化"
			} else if !publicip.IsPublicIPv4(detectedIPv4) {
				dr.Status = "error"
				dr.Message = "检测到内网 IPv4，拒绝更新"
				lastErr = fmt.Errorf("检测到内网 IPv4，拒绝更新")
			} else if err := provider.UpdateRecord(ctx, cred, root, record, "A", detectedIPv4); err != nil {
				dr.Status = "error"
				dr.Message = err.Error()
				lastErr = err
			} else {
				dr.IPv4 = detectedIPv4
				dr.Status = "ok"
				dr.Message = "已更新"
				changed = true
				s.logger.Info("ddns ipv4 updated", "domain", fqdn, "ip", detectedIPv4)
			}
		}

		if cfg.IPv6Enabled && detectedIPv6 != "" && dr.Status != "error" {
			current, getErr := provider.GetRecordIP(ctx, cred, root, record, "AAAA")
			if getErr != nil {
				dr.Status = "error"
				dr.Message = getErr.Error()
				lastErr = getErr
			} else {
				dr.IPv6 = current
				if current == detectedIPv6 {
					if dr.Status == "" {
						dr.Status = "unchanged"
						dr.Message = "记录未变化"
					}
				} else if !publicip.IsPublicIPv6(detectedIPv6) {
					dr.Status = "error"
					dr.Message = "检测到内网 IPv6，拒绝更新"
					lastErr = fmt.Errorf("检测到内网 IPv6，拒绝更新")
				} else if err := provider.UpdateRecord(ctx, cred, root, record, "AAAA", detectedIPv6); err != nil {
					dr.Status = "error"
					dr.Message = err.Error()
					lastErr = err
				} else {
					dr.IPv6 = detectedIPv6
					dr.Status = "ok"
					dr.Message = "已更新"
					changed = true
					s.logger.Info("ddns ipv6 updated", "domain", fqdn, "ip", detectedIPv6)
				}
			}
		}

		if dr.Status == "" {
			dr.Status = "unchanged"
			dr.Message = "记录未变化"
		}
		if dr.Status == "error" {
			errCount++
		} else {
			okCount++
		}
		domainRecords = append(domainRecords, dr)
	}

	overallStatus := "ok"
	overallError := ""
	switch {
	case errCount > 0 && okCount == 0:
		overallStatus = "error"
		if lastErr != nil {
			overallError = lastErr.Error()
		}
	case errCount > 0:
		overallStatus = "warning"
		overallError = fmt.Sprintf("%d 个域名同步失败，其余正常", errCount)
		lastErr = nil
	}
	if changed {
		s.logger.Info("ddns records updated", "provider", cfg.Provider)
	}
	_ = s.store.UpdateSyncResult(ctx, cfg.ID, storeIPv4, storeIPv6, overallStatus, overallError, domainRecords)
	return lastErr
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
		return Credentials{}, secret.DecryptHint(err)
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
	if len(in.Domains) > 0 {
		if _, _, err := ParseDomainLines(in.Domains); err != nil {
			return err
		}
	} else {
		if strings.TrimSpace(in.RootDomain) == "" {
			return fmt.Errorf("至少需要一个域名")
		}
		if _, err := ParseRecordNames(in.RecordNames, in.RecordName); err != nil {
			return err
		}
	}
	switch strings.TrimSpace(in.Provider) {
	case "dnspod", "cloudflare", "alidns", "tencentcloud", "":
	default:
		return fmt.Errorf("不支持的 DNS Provider")
	}
	return nil
}
