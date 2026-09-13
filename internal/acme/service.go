package acme

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log/slog"
	"net/http"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-acme/lego/v4/certcrypto"
	legocert "github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"

	certstore "github.com/fonu/fonu/internal/certificate"
	"github.com/fonu/fonu/internal/config"
	"github.com/fonu/fonu/internal/ddns"
	"github.com/fonu/fonu/internal/notify"
	"github.com/fonu/fonu/internal/service"
	"github.com/fonu/fonu/internal/settings"
)

type Service struct {
	cfg      config.Config
	store    *certstore.Store
	ddnsSvc  *ddns.Service
	settings *settings.Store
	proxySvc *service.ProxyService
	logger   *slog.Logger
	notify   *notify.Service
	jobs     *JobManager
}

func NewService(cfg config.Config, store *certstore.Store, ddnsSvc *ddns.Service, settings *settings.Store, proxySvc *service.ProxyService, logger *slog.Logger, notifySvc *notify.Service) *Service {
	return &Service{
		cfg:      cfg,
		store:    store,
		ddnsSvc:  ddnsSvc,
		settings: settings,
		proxySvc: proxySvc,
		logger:   logger.With("module", "ACME"),
		notify:   notifySvc,
		jobs:     NewJobManager(),
	}
}

func (s *Service) List(ctx context.Context) ([]certstore.Record, error) {
	return s.store.List(ctx)
}

func (s *Service) StartApply(ctx context.Context, domains []string, ca, email string, ddnsConfigID int64) (string, error) {
	if ddnsConfigID <= 0 {
		return "", fmt.Errorf("请选择 DNS 任务")
	}
	domains, ca, email, primary, dnsZone, provider, cred, err := s.prepareApplyWithConfig(ctx, domains, ca, email, ddnsConfigID)
	if err != nil {
		return "", err
	}
	job := s.jobs.Create()
	go s.runApplyJob(job, dnsZone, domains, ca, email, primary, provider, cred)
	return job.ID(), nil
}

func (s *Service) StreamJob(ctx context.Context, w http.ResponseWriter, jobID string, flush func() error) error {
	return s.jobs.Stream(ctx, w, jobID, flush)
}

func (s *Service) prepareApplyWithConfig(ctx context.Context, domains []string, ca, email string, ddnsConfigID int64) ([]string, string, string, string, string, string, ddns.Credentials, error) {
	domains, err := NormalizeCertDomains(domains)
	if err != nil {
		return nil, "", "", "", "", "", ddns.Credentials{}, err
	}
	primary := PrimaryCertDomain(domains)
	if primary == "" {
		return nil, "", "", "", "", "", ddns.Credentials{}, fmt.Errorf("请填写至少一个域名")
	}
	email, err = s.resolveACMEEmail(ctx, email)
	if err != nil {
		return nil, "", "", "", "", "", ddns.Credentials{}, err
	}
	ca, err = s.resolveCA(ctx, ca)
	if err != nil {
		return nil, "", "", "", "", "", ddns.Credentials{}, err
	}
	if err := ValidateCADomains(ca, domains); err != nil {
		return nil, "", "", "", "", "", ddns.Credentials{}, err
	}

	cfg, provider, cred, err := s.ddnsSvc.CredentialsForConfigID(ctx, ddnsConfigID)
	if err != nil {
		return nil, "", "", "", "", "", ddns.Credentials{}, err
	}
	if err := validateDNSCredentials(provider, cred); err != nil {
		return nil, "", "", "", "", "", ddns.Credentials{}, err
	}
	dnsZone := cfg.RootDomain
	zones := cfg.ManagedDNSZones()
	if len(zones) > 0 {
		dnsZone = zones[0]
	}
	return domains, ca, email, primary, dnsZone, provider, cred, nil
}

func (s *Service) prepareApplyByZone(ctx context.Context, dnsZone string, domains []string, ca, email string) ([]string, string, string, string, string, string, ddns.Credentials, error) {
	domains, err := NormalizeCertDomains(domains)
	if err != nil {
		return nil, "", "", "", "", "", ddns.Credentials{}, err
	}
	dnsZone = strings.ToLower(strings.TrimSpace(dnsZone))
	ddnsCfg, err := s.ddnsSvc.ConfigForDNSZone(ctx, dnsZone)
	if err != nil {
		return nil, "", "", "", "", "", ddns.Credentials{}, err
	}
	if err := DomainsUnderZones(domains, ddnsCfg.ManagedDNSZones()); err != nil {
		return nil, "", "", "", "", "", ddns.Credentials{}, err
	}
	primary := PrimaryCertDomain(domains)
	if primary == "" {
		return nil, "", "", "", "", "", ddns.Credentials{}, fmt.Errorf("请填写至少一个域名")
	}
	email, err = s.resolveACMEEmail(ctx, email)
	if err != nil {
		return nil, "", "", "", "", "", ddns.Credentials{}, err
	}
	ca, err = s.resolveCA(ctx, ca)
	if err != nil {
		return nil, "", "", "", "", "", ddns.Credentials{}, err
	}
	if err := ValidateCADomains(ca, domains); err != nil {
		return nil, "", "", "", "", "", ddns.Credentials{}, err
	}
	provider, cred, err := s.dnsCredentialsForDomain(ctx, dnsZone)
	if err != nil {
		return nil, "", "", "", "", "", ddns.Credentials{}, err
	}
	return domains, ca, email, primary, dnsZone, provider, cred, nil
}

func (s *Service) runApplyJob(job *Job, dnsZone string, domains []string, ca, email, primary, provider string, cred ddns.Credentials) {
	ctx := context.Background()
	job.Info("开始申请证书…")
	job.Info(fmt.Sprintf("主域名: %s", primary))
	job.Info(fmt.Sprintf("覆盖域名: %s", strings.Join(domains, ", ")))
	job.Info(fmt.Sprintf("DNS 凭证: %s (%s)", dnsZone, provider))

	rec, err := s.obtain(ctx, job, ca, email, provider, cred, primary, domains)
	if err != nil {
		s.markCertError(ctx, primary, domains, ca, err)
		job.Error(err.Error())
		job.Finish(JobDonePayload{OK: false, Error: err.Error(), Domain: primary})
		return
	}
	if err := s.settings.Set(ctx, settings.KeyACMECA, ca); err != nil {
		job.Warn("保存 ACME 颁发机构设置失败: " + err.Error())
	}
	if err := s.proxySvc.ReloadAll(ctx); err != nil {
		job.Warn("Nginx 重载失败: " + err.Error())
	} else {
		job.Info("Nginx 已重载")
	}
	job.Info("证书申请完成")
	job.Finish(jobDoneFromRecord(rec, nil))
}

func (s *Service) Apply(ctx context.Context, dnsZone string, domains []string, ca, email string) ([]certstore.Record, error) {
	domains, ca, email, primary, _, provider, cred, err := s.prepareApplyByZone(ctx, dnsZone, domains, ca, email)
	if err != nil {
		return nil, err
	}
	rec, err := s.obtain(ctx, nil, ca, email, provider, cred, primary, domains)
	if err != nil {
		s.markCertError(ctx, primary, domains, ca, err)
		return nil, err
	}
	if err := s.settings.Set(ctx, settings.KeyACMECA, ca); err != nil {
		s.logger.Warn("save acme ca setting failed", "error", err.Error())
	}
	if err := s.proxySvc.ReloadAll(ctx); err != nil {
		s.logger.Error("nginx reload after cert apply failed", "error", err.Error())
	}
	_ = rec
	return s.store.List(ctx)
}

func (s *Service) Import(ctx context.Context, certPEM, keyPEM, certFile, keyFile string) (certstore.Record, error) {
	certBytes, err := certstore.ResolvePEMSource(certPEM, certFile)
	if err != nil {
		return certstore.Record{}, fmt.Errorf("证书：%w", err)
	}
	keyBytes, err := certstore.ResolvePEMSource(keyPEM, keyFile)
	if err != nil {
		return certstore.Record{}, fmt.Errorf("私钥：%w", err)
	}

	info, err := certstore.InspectImport(certBytes, keyBytes)
	if err != nil {
		return certstore.Record{}, err
	}

	certPath, keyPath, err := writeCertFiles(s.cfg.CertsDir(), info.Domains, certBytes, keyBytes)
	if err != nil {
		return certstore.Record{}, err
	}

	status := "ok"
	if time.Now().After(info.ExpiresAt) {
		status = "expired"
	}

	domainsJSON := EncodeCertDomains(info.Domains)
	record, err := s.store.Upsert(ctx, info.Primary, info.Wildcard, domainsJSON, "imported", certPath, keyPath, info.ExpiresAt, status, "")
	if err != nil {
		return certstore.Record{}, err
	}

	if err := s.proxySvc.ReloadAll(ctx); err != nil {
		s.logger.Warn("nginx reload after cert import failed", "error", err.Error())
	}
	s.logger.Info("certificate imported", "domain", info.Primary, "domains", info.Domains, "expires_at", info.ExpiresAt.Format(time.RFC3339))
	return record, nil
}

func (s *Service) Delete(ctx context.Context, domain string) error {
	domain = strings.ToLower(strings.TrimSpace(domain))
	if domain == "" {
		return fmt.Errorf("域名不能为空")
	}
	existing, err := s.store.GetByDomain(ctx, domain)
	if err != nil {
		return err
	}
	if err := removeCertFiles(s.cfg.CertsDir(), existing); err != nil {
		s.logger.Warn("remove cert files failed", "domain", domain, "error", err.Error())
	}
	if err := s.store.Delete(ctx, domain); err != nil {
		return err
	}
	if err := s.proxySvc.ReloadAll(ctx); err != nil {
		s.logger.Warn("nginx reload after cert delete failed", "error", err.Error())
	}
	s.logger.Info("certificate deleted", "domain", domain)
	return nil
}

func (s *Service) Renew(ctx context.Context, domain string, ca string) (certstore.Record, error) {
	rootDomain := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(domain, "*.")))
	if rootDomain == "" {
		return certstore.Record{}, fmt.Errorf("请选择要续签的证书域名")
	}
	existing, err := s.store.GetByDomain(ctx, rootDomain)
	if err != nil {
		return certstore.Record{}, err
	}
	if existing.ACMECA == "imported" {
		return certstore.Record{}, fmt.Errorf("手动导入的证书请重新导入，无法自动续签")
	}
	if ca == "" {
		ca = existing.ACMECA
	}
	ca, err = s.resolveCA(ctx, ca)
	if err != nil {
		return certstore.Record{}, err
	}
	email, err := s.resolveACMEEmail(ctx, "")
	if err != nil {
		return certstore.Record{}, err
	}
	domains := existing.Domains
	if len(domains) == 0 {
		domains = LegacyCertDomains(existing.Domain, existing.Wildcard)
	}
	dnsZone, err := s.resolveDNSZone(ctx, domains)
	if err != nil {
		return certstore.Record{}, err
	}
	provider, cred, err := s.dnsCredentialsForDomain(ctx, dnsZone)
	if err != nil {
		return certstore.Record{}, err
	}
	if _, err := s.obtain(ctx, nil, ca, email, provider, cred, existing.Domain, domains); err != nil {
		s.markCertError(ctx, rootDomain, domains, ca, err)
		return certstore.Record{}, err
	}
	if err := s.proxySvc.ReloadAll(ctx); err != nil {
		s.logger.Error("nginx reload after cert renew failed", "error", err.Error())
	}
	return s.store.GetByDomain(ctx, rootDomain)
}

func (s *Service) Tick(ctx context.Context) {
	records, err := s.store.List(ctx)
	if err != nil {
		return
	}
	threshold, err := s.settings.GetInt(ctx, settings.KeyCertRenewThreshold)
	if err != nil || threshold <= 0 {
		threshold = 30
	}
	for _, rec := range records {
		if rec.ExpiresAt == nil {
			continue
		}
		if rec.DaysLeft <= threshold {
			if rec.ACMECA == "imported" {
				continue
			}
			if _, err := s.Renew(ctx, rec.Domain, rec.ACMECA); err != nil {
				s.logger.Error("certificate renew failed", "domain", rec.Domain, "error", err.Error())
				if s.notify != nil {
					s.notify.Alert(ctx, notify.EventCertError, "证书续签失败", err.Error())
				}
			}
		}
	}
}

func (s *Service) markCertError(ctx context.Context, primary string, domains []string, ca string, err error) {
	if err == nil {
		return
	}
	errMsg := err.Error()
	s.logger.Error("certificate operation failed", "domain", primary, "error", errMsg)
	_, upsertErr := s.store.Upsert(ctx, primary, HasWildcardDomain(domains), EncodeCertDomains(domains), ca, "", "", time.Time{}, "error", errMsg)
	if upsertErr != nil {
		s.logger.Error("save cert error status failed", "domain", primary, "error", upsertErr.Error())
	}
}

func (s *Service) resolveCA(ctx context.Context, ca string) (string, error) {
	if strings.TrimSpace(ca) == "" {
		ca, err := s.settings.Get(ctx, settings.KeyACMECA)
		if err != nil || strings.TrimSpace(ca) == "" {
			ca = CALetsEncrypt
		}
	}
	ca = NormalizeCA(ca)
	if err := ValidateCA(ca); err != nil {
		return "", err
	}
	return ca, nil
}

func (s *Service) registerACMEAccount(ctx context.Context, client *lego.Client, ca string, logStep func(string)) (*registration.Resource, error) {
	reg, err := client.Registration.ResolveAccountByKey()
	if err == nil {
		return reg, nil
	}
	logStep("注册 ACME 账户…")
	switch NormalizeCA(ca) {
	case CAZeroSSL:
		apiKey, err := s.settings.Get(ctx, settings.KeyZeroSSLAPIKey)
		if err != nil {
			return nil, err
		}
		kid, hmac, err := zerosslEABCredentials(ctx, apiKey)
		if err != nil {
			return nil, err
		}
		return client.Registration.RegisterWithExternalAccountBinding(registration.RegisterEABOptions{
			TermsOfServiceAgreed: true,
			Kid:                  kid,
			HmacEncoded:          hmac,
		})
	default:
		return client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
	}
}

func (s *Service) obtain(ctx context.Context, job *Job, ca, email, provider string, cred ddns.Credentials, rootDomain string, domains []string) (certstore.Record, error) {
	logStep := func(msg string) {
		if job != nil {
			job.Info(msg)
		}
	}
	caDir, err := DirectoryURL(ca)
	if err != nil {
		return certstore.Record{}, err
	}
	logStep(fmt.Sprintf("连接 ACME 服务器 (%s)", CALabel(ca)))
	user, err := newUser(email)
	if err != nil {
		return certstore.Record{}, err
	}
	config := lego.NewConfig(user)
	config.CADirURL = caDir
	config.Certificate.KeyType = certcrypto.EC256

	client, err := lego.NewClient(config)
	if err != nil {
		return certstore.Record{}, err
	}

	dnsProvider, err := newDNS01Provider(provider, cred)
	if err != nil {
		return certstore.Record{}, fmt.Errorf("初始化 DNS Provider 失败：%w", err)
	}
	dnsProvider = wrapDNSProvider(dnsProvider, job)
	if err := client.Challenge.SetDNS01Provider(dnsProvider); err != nil {
		return certstore.Record{}, err
	}

	logStep("检查 ACME 账户…")
	reg, err := s.registerACMEAccount(ctx, client, ca, logStep)
	if err != nil {
		return certstore.Record{}, fmt.Errorf("ACME 注册失败：%w", err)
	}
	user.registration = reg

	logStep("开始 DNS-01 验证并申请证书（可能需要 1～3 分钟）…")
	request := legocert.ObtainRequest{
		Domains: domains,
		Bundle:  true,
	}
	cert, err := client.Certificate.Obtain(request)
	if err != nil {
		return certstore.Record{}, fmt.Errorf("证书申请失败：%w", err)
	}

	certPath, keyPath, err := writeCertFiles(s.cfg.CertsDir(), domains, cert.Certificate, cert.PrivateKey)
	if err != nil {
		return certstore.Record{}, err
	}
	logStep(fmt.Sprintf("证书已保存: %s", certPath))
	logStep(fmt.Sprintf("私钥已保存: %s", keyPath))

	expiresAt, err := parseCertExpiry(cert.Certificate)
	if err != nil {
		expiresAt = time.Now().Add(90 * 24 * time.Hour)
	}
	logStep(fmt.Sprintf("到期时间: %s", expiresAt.Local().Format("2006-01-02 15:04:ss")))
	rec, err := s.store.Upsert(ctx, rootDomain, HasWildcardDomain(domains), EncodeCertDomains(domains), ca, certPath, keyPath, expiresAt, "ok", "")
	if err != nil {
		return certstore.Record{}, err
	}
	s.logger.Info("certificate obtained", "domain", rootDomain, "ca", ca, "expires_at", expiresAt.Format(time.RFC3339))
	return rec, nil
}

func (s *Service) resolveACMEEmail(ctx context.Context, email string) (string, error) {
	email = strings.TrimSpace(email)
	if email == "" {
		stored, err := s.settings.Get(ctx, settings.KeyACMEEmail)
		if err != nil || strings.TrimSpace(stored) == "" {
			return "", fmt.Errorf("请先填写 ACME 邮箱，用于注册 ACME 账户（首次申请必填）")
		}
		email = strings.TrimSpace(stored)
	}
	if err := validateACMEEmail(email); err != nil {
		return "", err
	}
	if err := s.settings.Set(ctx, settings.KeyACMEEmail, email); err != nil {
		s.logger.Warn("save acme email failed", "error", err.Error())
	}
	return email, nil
}

func validateACMEEmail(email string) error {
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return fmt.Errorf("ACME 邮箱格式不正确，请填写类似 admin@example.com 的地址")
	}
	return nil
}

func (s *Service) resolveDNSZone(ctx context.Context, domains []string) (string, error) {
	for _, domain := range domains {
		cfg, err := s.ddnsSvc.ConfigForAnyDomain(ctx, domain)
		if err == nil {
			return cfg.RootDomain, nil
		}
	}
	return "", fmt.Errorf("请先在 DDNS 页面配置对应域名的 DNS 凭证")
}

func (s *Service) dnsCredentialsForDomain(ctx context.Context, rootDomain string) (string, ddns.Credentials, error) {
	provider, cred, err := s.ddnsSvc.CredentialsForDomain(ctx, rootDomain)
	if err != nil {
		return "", ddns.Credentials{}, fmt.Errorf("请先在 DDNS 页面为 %s 配置 DNS 凭证", rootDomain)
	}
	if err := validateDNSCredentials(provider, cred); err != nil {
		return "", ddns.Credentials{}, err
	}
	return provider, cred, nil
}

func writeCertFiles(certsDir string, domains []string, certPEM, keyPEM []byte) (string, string, error) {
	names := certStorageNames(domains)
	if len(names) == 0 {
		return "", "", fmt.Errorf("证书域名不能为空")
	}
	var certPath, keyPath string
	for _, name := range names {
		dir := filepath.Join(certsDir, name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", "", err
		}
		cp := filepath.Join(dir, "fullchain.pem")
		kp := filepath.Join(dir, "privatekey.pem")
		if err := os.WriteFile(cp, certPEM, 0o600); err != nil {
			return "", "", err
		}
		if err := os.WriteFile(kp, keyPEM, 0o600); err != nil {
			return "", "", err
		}
		certPath, keyPath = cp, kp
	}
	return certPath, keyPath, nil
}

func removeCertFiles(certsDir string, record certstore.Record) error {
	domains := record.Domains
	if len(domains) == 0 {
		domains = LegacyCertDomains(record.Domain, record.Wildcard)
	}
	var firstErr error
	for _, name := range certStorageNames(domains) {
		dir := filepath.Join(certsDir, name)
		if err := os.RemoveAll(dir); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func certStorageNames(domains []string) []string {
	seen := map[string]bool{}
	var names []string
	for _, domain := range domains {
		name := domain
		if strings.HasPrefix(domain, "*.") {
			name = "wildcard." + strings.TrimPrefix(domain, "*.")
		}
		if seen[name] {
			continue
		}
		seen[name] = true
		names = append(names, name)
	}
	return names
}

func parseCertExpiry(pemBytes []byte) (time.Time, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return time.Time{}, fmt.Errorf("invalid pem")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return time.Time{}, err
	}
	return cert.NotAfter, nil
}

type acmeUser struct {
	email        string
	registration *registration.Resource
	key          crypto.PrivateKey
}

func (u *acmeUser) GetEmail() string                        { return u.email }
func (u *acmeUser) GetRegistration() *registration.Resource { return u.registration }
func (u *acmeUser) GetPrivateKey() crypto.PrivateKey        { return u.key }

func newUser(email string) (*acmeUser, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	return &acmeUser{email: email, key: privateKey}, nil
}
