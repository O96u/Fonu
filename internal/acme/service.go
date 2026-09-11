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

func (s *Service) Apply(ctx context.Context) ([]certstore.Record, error) {
	rootDomain, err := s.settings.Get(ctx, settings.KeyRootDomain)
	if err != nil || strings.TrimSpace(rootDomain) == "" {
		return nil, fmt.Errorf("请先在设置中配置主域名")
	}
	email, err := s.settings.Get(ctx, settings.KeyACMEEmail)
	if err != nil || strings.TrimSpace(email) == "" {
		return nil, fmt.Errorf("请先在设置中配置 ACME 邮箱")
	}
	provider, cred, err := s.dnsCredentials(ctx)
	if err != nil {
		return nil, err
	}

	domains := []string{rootDomain, "*." + rootDomain}
	if err := s.obtain(ctx, email, provider, cred, rootDomain, domains); err != nil {
		_ = s.store.UpdateStatus(ctx, rootDomain, "error", err.Error())
		return nil, err
	}
	if err := s.proxySvc.ReloadAll(ctx); err != nil {
		s.logger.Error("nginx reload after cert apply failed", "error", err.Error())
	}
	return s.store.List(ctx)
}

func (s *Service) Renew(ctx context.Context, domain string) (certstore.Record, error) {
	if domain == "" {
		rootDomain, err := s.settings.Get(ctx, settings.KeyRootDomain)
		if err != nil || rootDomain == "" {
			return certstore.Record{}, fmt.Errorf("未指定证书域名")
		}
		domain = rootDomain
	}
	email, err := s.settings.Get(ctx, settings.KeyACMEEmail)
	if err != nil || email == "" {
		return certstore.Record{}, fmt.Errorf("请先在设置中配置 ACME 邮箱")
	}
	provider, cred, err := s.dnsCredentials(ctx)
	if err != nil {
		return certstore.Record{}, err
	}
	rootDomain := strings.TrimPrefix(domain, "*.")
	domains := []string{rootDomain, "*." + rootDomain}
	if err := s.obtain(ctx, email, provider, cred, rootDomain, domains); err != nil {
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
			if _, err := s.Renew(ctx, rec.Domain); err != nil {
				s.logger.Error("certificate renew failed", "domain", rec.Domain, "error", err.Error())
				if s.notify != nil {
					s.notify.Alert(ctx, notify.EventCertError, "证书续签失败", err.Error())
				}
			}
		}
	}
}

func (s *Service) obtain(ctx context.Context, email, provider string, cred ddns.Credentials, rootDomain string, domains []string) error {
	user, err := newUser(email)
	if err != nil {
		return err
	}
	config := lego.NewConfig(user)
	config.CADirURL = lego.LEDirectoryProduction
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

	certPath, keyPath, err := writeCertFiles(s.cfg.CertsDir(), rootDomain, cert.Certificate, cert.PrivateKey)
	if err != nil {
		return err
	}

	expiresAt, err := parseCertExpiry(cert.Certificate)
	if err != nil {
		expiresAt = time.Now().Add(90 * 24 * time.Hour)
	}
	_, err = s.store.Upsert(ctx, rootDomain, true, certPath, keyPath, expiresAt, "ok", "")
	if err != nil {
		return err
	}
	s.logger.Info("certificate obtained", "domain", rootDomain, "expires_at", expiresAt.Format(time.RFC3339))
	return nil
}

func (s *Service) dnsCredentials(ctx context.Context) (string, ddns.Credentials, error) {
	provider, cred, err := s.ddnsSvc.Credentials(ctx)
	if err != nil {
		return "", ddns.Credentials{}, fmt.Errorf("请先在 DDNS 页面配置 DNS 凭证")
	}
	if err := validateDNSCredentials(provider, cred); err != nil {
		return "", ddns.Credentials{}, err
	}
	return provider, cred, nil
}

func writeCertFiles(certsDir, rootDomain string, certPEM, keyPEM []byte) (string, string, error) {
	dirs := []string{
		filepath.Join(certsDir, rootDomain),
		filepath.Join(certsDir, "wildcard."+rootDomain),
	}
	var certPath, keyPath string
	for _, dir := range dirs {
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
