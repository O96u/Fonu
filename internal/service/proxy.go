package service

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"github.com/fonu/fonu/internal/certificate"
	"github.com/fonu/fonu/internal/config"
	"github.com/fonu/fonu/internal/nginx"
	"github.com/fonu/fonu/internal/proxy"
	"github.com/fonu/fonu/internal/settings"
	"github.com/fonu/fonu/internal/validate"
)

type ProxyService struct {
	db        *sql.DB
	cfg       config.Config
	store     *proxy.Store
	certStore *certificate.Store
	settings  *settings.Store
	nginx     *nginx.Manager
}

func NewProxyService(cfg config.Config, db *sql.DB, store *proxy.Store, certStore *certificate.Store, settingsStore *settings.Store, nginxMgr *nginx.Manager) *ProxyService {
	return &ProxyService{db: db, cfg: cfg, store: store, certStore: certStore, settings: settingsStore, nginx: nginxMgr}
}

func (s *ProxyService) List(ctx context.Context) ([]proxy.Rule, error) {
	return s.store.List(ctx)
}

func (s *ProxyService) Get(ctx context.Context, id int64) (proxy.Rule, error) {
	return s.store.Get(ctx, id)
}

func (s *ProxyService) Create(ctx context.Context, in proxy.CreateInput) (proxy.Rule, error) {
	if err := s.validateHTTPSInput(ctx, in.Hosts, in.HTTPSEnabled); err != nil {
		return proxy.Rule{}, err
	}

	rule, err := s.store.Create(ctx, in)
	if err != nil {
		return proxy.Rule{}, err
	}
	if err := s.applyNginx(ctx); err != nil {
		return rule, fmt.Errorf("规则已保存，但 Nginx 重载失败：%w", err)
	}
	return rule, nil
}

func (s *ProxyService) Update(ctx context.Context, id int64, in proxy.UpdateInput) (proxy.Rule, error) {
	current, err := s.Get(ctx, id)
	if err != nil {
		return proxy.Rule{}, err
	}
	httpsEnabled := current.HTTPSEnabled
	if in.HTTPSEnabled != nil {
		httpsEnabled = *in.HTTPSEnabled
	}
	hosts := hostsFromRule(current)
	if in.Hosts != nil {
		hosts = *in.Hosts
	}
	if err := s.validateHTTPSInput(ctx, hosts, httpsEnabled); err != nil {
		return proxy.Rule{}, err
	}

	rule, err := s.store.Update(ctx, id, in)
	if err != nil {
		return proxy.Rule{}, err
	}
	if err := s.applyNginx(ctx); err != nil {
		return rule, fmt.Errorf("规则已保存，但 Nginx 重载失败：%w", err)
	}
	return rule, nil
}

func (s *ProxyService) Reorder(ctx context.Context, ids []int64) error {
	return s.store.Reorder(ctx, ids)
}

func (s *ProxyService) Delete(ctx context.Context, id int64) error {
	if err := s.store.Delete(ctx, id); err != nil {
		return err
	}
	if err := s.applyNginx(ctx); err != nil {
		return fmt.Errorf("规则已删除，但 Nginx 重载失败：%w", err)
	}
	return nil
}

func (s *ProxyService) ReloadAll(ctx context.Context) error {
	return s.applyNginx(ctx)
}

func (s *ProxyService) applyNginx(ctx context.Context) error {
	rules, err := s.store.ListEnabled(ctx)
	if err != nil {
		return err
	}
	certs, err := s.loadCertSources(ctx)
	if err != nil {
		return err
	}
	s.nginx.SetGenerateOptions(s.loadGenerateOptions(ctx))
	_, err = s.nginx.Apply(ctx, rules, certs)
	return err
}

func (s *ProxyService) ValidateRule(ctx context.Context, candidate proxy.Rule, existing []proxy.Rule) error {
	rules := make([]proxy.Rule, 0, len(existing)+1)
	replaced := false
	for _, r := range existing {
		if r.ID == candidate.ID {
			if candidate.Enabled {
				rules = append(rules, candidate)
			}
			replaced = true
			continue
		}
		if r.Enabled {
			rules = append(rules, r)
		}
	}
	if !replaced && candidate.Enabled {
		rules = append(rules, candidate)
	}
	certs, err := s.loadCertSources(ctx)
	if err != nil {
		return err
	}
	s.nginx.SetGenerateOptions(s.loadGenerateOptions(ctx))
	if err := s.nginx.ValidateOnly(ctx, rules, certs); err != nil {
		return err
	}
	return nil
}

func hostsFromRule(rule proxy.Rule) []string {
	hosts := make([]string, 0, len(rule.Hosts))
	for _, host := range rule.Hosts {
		if host.ListenPort != nil && *host.ListenPort > 0 && *host.ListenPort != rule.ListenPort {
			hosts = append(hosts, host.Hostname+":"+strconv.Itoa(*host.ListenPort))
			continue
		}
		hosts = append(hosts, host.Hostname)
	}
	return hosts
}

func (s *ProxyService) validateHTTPSInput(ctx context.Context, hosts []string, httpsEnabled bool) error {
	if !httpsEnabled || len(hosts) == 0 {
		return nil
	}
	certs, err := s.loadCertSources(ctx)
	if err != nil {
		return err
	}
	parsedHosts := make([]string, 0, len(hosts))
	for _, host := range hosts {
		hostname, _, err := validate.FrontendAddress(host)
		if err != nil {
			return err
		}
		parsedHosts = append(parsedHosts, hostname)
	}
	if nginx.HasCertificateForHosts(s.cfg.CertsDir(), parsedHosts, certs) {
		return nil
	}
	return nginx.HTTPSCoverageError(parsedHosts[0], certs)
}

func (s *ProxyService) loadCertSources(ctx context.Context) ([]nginx.CertSource, error) {
	records, err := s.certStore.List(ctx)
	if err != nil {
		return nil, err
	}
	sources := make([]nginx.CertSource, 0, len(records))
	for _, record := range records {
		sources = append(sources, nginx.CertSource{
			Domains:  record.Domains,
			CertPath: record.CertPath,
			KeyPath:  record.KeyPath,
		})
	}
	return sources, nil
}
