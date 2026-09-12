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
	"os"
	"path/filepath"
	"sort"
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
	}
}

func (s *Service) List(ctx context.Context) ([]certstore.Record, error) {
	return s.store.List(ctx)
}

func (s *Service) Apply(ctx context.Context, dnsZone string, domains []string, ca, email string) ([]certstore.Record, error) {
	domains, err := NormalizeCertDomains(domains)
	if err != nil {
		return nil, err
	}
	dnsZone = strings.ToLower(strings.TrimSpace(dnsZone))
	if err := DomainsUnderZone(domains, dnsZone); err != nil {
		return nil, err
	}
	primary := PrimaryCertDomain(domains)
	if primary == "" {
		return nil, fmt.Errorf("请填写至少一个域名")
	}
	email, err = s.resolveACMEEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	ca, err = s.resolveCA(ctx, ca)
	if err != nil {
		return nil, err
	}
	provider, cred, err := s.dnsCredentialsForDomain(ctx, dnsZone)
	if err != nil {
		return nil, err
	}

	if err := s.obtain(ctx, ca, email, provider, cred, primary, domains); err != nil {
		_ = s.store.UpdateStatus(ctx, primary, "error", err.Error())
		return nil, err
	}
	if err := s.settings.Set(ctx, settings.KeyACMECA, ca); err != nil {
		s.logger.Warn("save acme ca setting failed", "error", err.Error())
	}
	if err := s.proxySvc.ReloadAll(ctx); err != nil {
		s.logger.Error("nginx reload after cert apply failed", "error", err.Error())
	}
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
	if err := s.obtain(ctx, ca, email, provider, cred, existing.Domain, domains); err != nil {
		_ = s.store.UpdateStatus(ctx, rootDomain, "error", err.Error())
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

func (s *Service) obtain(ctx context.Context, ca, email, provider string, cred ddns.Credentials, rootDomain string, domains []string) error {
	caDir, err := DirectoryURL(ca)
	if err != nil {
		return err
	}
	user, err := newUser(email)
	if err != nil {
		return err
	}
	config := lego.NewConfig(user)
	config.CADirURL = caDir
	config.Certificate.KeyType = certcrypto.EC256

	client, err := lego.NewClient(config)
	if err != nil {
		return err
	}

	dnsProvider, err := newDNS01Provider(provider, cred)
	if err != nil {
		return fmt.Errorf("初始化 DNS Provider 失败：%w", err)
	}
	if err := client.Challenge.SetDNS01Provider(dnsProvider); err != nil {
		return err
	}

	reg, err := client.Registration.ResolveAccountByKey()
	if err != nil {
		reg, err = client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
		if err != nil {
			return fmt.Errorf("ACME 注册失败：%w", err)
		}
	}
	user.registration = reg

	request := legocert.ObtainRequest{
		Domains: domains,
		Bundle:  true,
	}
	cert, err := client.Certificate.Obtain(request)
	if err != nil {
		return fmt.Errorf("证书申请失败：%w", err)
	}

	certPath, keyPath, err := writeCertFiles(s.cfg.CertsDir(), domains, cert.Certificate, cert.PrivateKey)
	if err != nil {
		return err
	}

	expiresAt, err := parseCertExpiry(cert.Certificate)
	if err != nil {
		expiresAt = time.Now().Add(90 * 24 * time.Hour)
	}
	_, err = s.store.Upsert(ctx, rootDomain, HasWildcardDomain(domains), EncodeCertDomains(domains), ca, certPath, keyPath, expiresAt, "ok", "")
	if err != nil {
		return err
	}
	s.logger.Info("certificate obtained", "domain", rootDomain, "ca", ca, "expires_at", expiresAt.Format(time.RFC3339))
	return nil
}

func (s *Service) resolveACMEEmail(ctx context.Context, email string) (string, error) {
	email = strings.TrimSpace(email)
	if email == "" {
		email, err := s.settings.Get(ctx, settings.KeyACMEEmail)
		if err != nil || strings.TrimSpace(email) == "" {
			return "", fmt.Errorf("请填写 ACME 邮箱")
		}
		return email, nil
	}
	if err := s.settings.Set(ctx, settings.KeyACMEEmail, email); err != nil {
		s.logger.Warn("save acme email failed", "error", err.Error())
	}
	return email, nil
}

func (s *Service) resolveDNSZone(ctx context.Context, domains []string) (string, error) {
	seen := map[string]bool{}
	var candidates []string
	for _, domain := range domains {
		base := strings.TrimPrefix(domain, "*.")
		parts := strings.Split(base, ".")
		for i := 0; i < len(parts)-1; i++ {
			zone := strings.Join(parts[i:], ".")
			if seen[zone] {
				continue
			}
			seen[zone] = true
			candidates = append(candidates, zone)
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		return len(candidates[i]) > len(candidates[j])
	})
	for _, zone := range candidates {
		if _, _, err := s.ddnsSvc.CredentialsForDomain(ctx, zone); err == nil {
			return zone, nil
		}
	}
	return "", fmt.Errorf("请先在 DDNS 页面配置对应根域名的 DNS 凭证")
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
