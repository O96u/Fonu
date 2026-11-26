package service

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"time"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/fonu/fonu/internal/certificate"
	"github.com/fonu/fonu/internal/config"
	"github.com/fonu/fonu/internal/nginx"
	"github.com/fonu/fonu/internal/notify"
	"github.com/fonu/fonu/internal/proxy"
	"github.com/fonu/fonu/internal/settings"
	"github.com/fonu/fonu/internal/validate"
)

type ProxyService struct {
	applyMu sync.Mutex
	db        *sql.DB
	cfg       config.Config
	store     *proxy.Store
	certStore *certificate.Store
	settings  *settings.Store
	nginx     *nginx.Manager
	notify    *notify.Service
}

func NewProxyService(cfg config.Config, db *sql.DB, store *proxy.Store, certStore *certificate.Store, settingsStore *settings.Store, nginxMgr *nginx.Manager) *ProxyService {
	return &ProxyService{db: db, cfg: cfg, store: store, certStore: certStore, settings: settingsStore, nginx: nginxMgr}
}

func (s *ProxyService) SetNotify(notifySvc *notify.Service) {
	s.notify = notifySvc
}

func (s *ProxyService) ListEntries(ctx context.Context) ([]proxy.Entry, error) {
	return s.store.ListEntries(ctx)
}

func (s *ProxyService) ReorderEntries(ctx context.Context, ids []int64) error {
	s.applyMu.Lock()
	defer s.applyMu.Unlock()
	return s.store.ReorderEntries(ctx, ids)
}

func (s *ProxyService) CreateEntry(ctx context.Context, in proxy.EntryCreateInput) (proxy.Entry, error) {
	s.applyMu.Lock()
	defer s.applyMu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil { return proxy.Entry{}, err }
	defer tx.Rollback()
	entry, err := proxy.NewStoreWithTx(tx).CreateEntry(ctx, in)
	if err != nil { return proxy.Entry{}, err }
	if err := tx.Commit(); err != nil { return proxy.Entry{}, err }
	return entry, nil
}

func (s *ProxyService) UpdateEntry(ctx context.Context, id int64, in proxy.EntryUpdateInput) (proxy.Entry, error) {
	s.applyMu.Lock()
	defer s.applyMu.Unlock()
	return mutateProxy(ctx, s, func(store *proxy.Store) (proxy.Entry, error) {
		return store.UpdateEntry(ctx, id, in)
	})
}

func (s *ProxyService) DeleteEntry(ctx context.Context, id int64) error {
	s.applyMu.Lock()
	defer s.applyMu.Unlock()
	ids, err := mutateProxy(ctx, s, func(store *proxy.Store) ([]int64, error) {
		ids, err := store.RuleIDsByEntry(ctx, id)
		if err != nil { return nil, err }
		return ids, store.DeleteEntry(ctx, id)
	})
	if err != nil { return err }
	for _, id := range ids { _ = nginx.RemoveRuleCustom(s.cfg, id) }
	return nil
}

func (s *ProxyService) CloneEntry(ctx context.Context, sourceID int64, in proxy.EntryCloneInput) (proxy.Entry, int, error) {
	s.applyMu.Lock()
	defer s.applyMu.Unlock()
	var copied []int64
	count := 0
	entry, err := mutateProxy(ctx, s, func(store *proxy.Store) (proxy.Entry, error) {
		entry, idMap, err := store.CloneEntry(ctx, sourceID, in)
		if err != nil { return proxy.Entry{}, err }
		count = len(idMap)
		for oldID, newID := range idMap {
			content, err := nginx.ReadRuleCustom(s.cfg, oldID)
			if err != nil { return proxy.Entry{}, err }
			if strings.TrimSpace(content) == "" { continue }
			// A custom server block may contain the source port, host and auth paths.
			// Require explicit conversion/edit instead of copying an active conflicting block.
			old, err := store.Get(ctx, oldID)
			if err != nil { return proxy.Entry{}, err }
			if old.NginxMode == "custom" && old.Enabled {
				return proxy.Entry{}, fmt.Errorf("入口包含启用的手动 Nginx 规则，请先切换为自动模式后复制")
			}
			copied = append(copied, newID)
			if err := nginx.WriteRuleCustom(s.cfg, newID, content); err != nil { return proxy.Entry{}, err }
		}
		return entry, nil
	})
	if err != nil {
		for _, id := range copied { _ = nginx.RemoveRuleCustom(s.cfg, id) }
		return proxy.Entry{}, 0, err
	}
	return entry, count, nil
}

func (s *ProxyService) List(ctx context.Context) ([]proxy.Rule, error) {
	return s.store.List(ctx)
}

func (s *ProxyService) Get(ctx context.Context, id int64) (proxy.Rule, error) {
	return s.store.Get(ctx, id)
}

func (s *ProxyService) Create(ctx context.Context, in proxy.CreateInput) (proxy.Rule, error) {
	s.applyMu.Lock()
	defer s.applyMu.Unlock()
	return mutateProxy(ctx, s, func(store *proxy.Store) (proxy.Rule, error) {
		return store.Create(ctx, in)
	})
}

func (s *ProxyService) Update(ctx context.Context, id int64, in proxy.UpdateInput) (proxy.Rule, error) {
	s.applyMu.Lock()
	defer s.applyMu.Unlock()
	return mutateProxy(ctx, s, func(store *proxy.Store) (proxy.Rule, error) {
		return store.Update(ctx, id, in)
	})
}

func (s *ProxyService) Reorder(ctx context.Context, ids []int64) error {
	s.applyMu.Lock()
	defer s.applyMu.Unlock()
	return s.store.Reorder(ctx, ids)
}

func (s *ProxyService) Delete(ctx context.Context, id int64) error {
	s.applyMu.Lock()
	defer s.applyMu.Unlock()
	_, err := mutateProxy(ctx, s, func(store *proxy.Store) (struct{}, error) {
		return struct{}{}, store.Delete(ctx, id)
	})
	if err != nil { return err }
	_ = nginx.RemoveRuleCustom(s.cfg, id)
	return nil
}

func (s *ProxyService) ReloadAll(ctx context.Context) error {
	return s.applyNginx(ctx)
}

func (s *ProxyService) applyNginx(ctx context.Context) error {
	s.applyMu.Lock()
	defer s.applyMu.Unlock()
	if err := s.recoverPendingLocked(ctx); err != nil { return err }
	return s.applyNginxLocked(ctx)
}

func (s *ProxyService) applyNginxLocked(ctx context.Context) error {
	rules, err := s.store.ListEnabled(ctx)
	if err != nil {
		return err
	}
	certs, err := s.loadCertSources(ctx)
	if err != nil {
		return err
	}
	opts := s.loadGenerateOptions(ctx)
	s.nginx.SetGenerateOptions(opts)
	_, err = s.nginx.ApplyWithOptions(ctx, rules, certs, opts)
	if err != nil && s.notify != nil {
		s.notify.Alert(ctx, notify.EventNginxReloadFailure, "Nginx 重载失败", err.Error())
	}
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
	if err := s.nginx.ValidateOnlyWithOptions(ctx, rules, certs, s.loadGenerateOptions(ctx)); err != nil {
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
	for _, hostname := range parsedHosts {
		if !nginx.HasCertificateForHosts(s.cfg.CertsDir(), []string{hostname}, certs) {
			return nginx.HTTPSCoverageError(hostname, certs)
		}
	}
	return nil
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

func (s *ProxyService) NginxApplyStatus() nginx.ApplyStatus {
	if s.nginx == nil { return nginx.ApplyStatus{State: "unknown"} }
	return s.nginx.ApplyStatus()
}

 // ApplyChinaCIDR serializes CIDR publication with the regular full configuration flow.
func (s *ProxyService) ApplyChinaCIDR(ctx context.Context, body string) error {
	s.applyMu.Lock()
	defer s.applyMu.Unlock()
	if err := s.recoverPendingLocked(ctx); err != nil { return err }
	rules, err := s.store.ListEnabled(ctx)
	if err != nil { return err }
	certs, err := s.loadCertSources(ctx)
	if err != nil { return err }
	final := s.cfg.ChinaCIDRPath()
	old, err := os.ReadFile(final)
	hadOld := err == nil
	if err != nil && !os.IsNotExist(err) { return err }
	tmp, err := writeCIDRTemp(filepath.Dir(final), []byte(body))
	if err != nil { return err }
	defer os.Remove(tmp)
	opts := s.loadGenerateOptions(ctx)
	opts.ChinaCIDRPathOverride = tmp
	opts.ChinaCIDRAvailable = true
	if err := s.nginx.ValidateOnlyWithOptions(ctx, rules, certs, opts); err != nil { return err }
	if err := os.Rename(tmp, final); err != nil { return err }
	if err := s.applyNginxLocked(ctx); err != nil {
		var rollbackErr error
		if hadOld {
			var staged string
			staged, rollbackErr = writeCIDRTemp(filepath.Dir(final), old)
			if rollbackErr == nil { defer os.Remove(staged); rollbackErr = os.Rename(staged, final) }
		} else { rollbackErr = os.Remove(final) }
		if rollbackErr != nil { return errors.Join(err, fmt.Errorf("CIDR rollback: %w", rollbackErr)) }
		rollbackCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if _, reloadErr := s.nginx.ApplyWithOptions(rollbackCtx, rules, certs, s.loadGenerateOptions(rollbackCtx)); reloadErr != nil { return errors.Join(err, reloadErr) }
		return err
	}
	return nil
}

func writeCIDRTemp(dir string, body []byte) (string, error) {
	f, err := os.CreateTemp(dir, "cidr-*.conf")
	if err != nil { return "", err }
	if err = f.Chmod(0644); err == nil { _, err = f.Write(body) }
	closeErr := f.Close()
	if err == nil { err = closeErr }
	if err != nil { os.Remove(f.Name()); return "", err }
	return f.Name(), nil
}

// SaveBatch applies the entire edit/create set in one SQLite transaction
// and one Nginx publication. No partial success is returned.
func (s *ProxyService) SaveBatch(ctx context.Context, id int64, update *proxy.UpdateInput, creates []proxy.CreateInput)([]proxy.Rule,error){
	count:=len(creates)
	if update!=nil{count++}
	if count<1||count>100{return nil,fmt.Errorf("单次批量操作支持 1–100 条规则")}
	s.applyMu.Lock()
	defer s.applyMu.Unlock()
	return mutateProxy(ctx,s,func(store *proxy.Store)([]proxy.Rule,error){
		out:=make([]proxy.Rule,0,count)
		if update!=nil{
			r,err:=store.Update(ctx,id,*update)
			if err!=nil{return nil,err}
			out=append(out,r)
		}
		for _,in:=range creates{
			r,err:=store.Create(ctx,in)
			if err!=nil{return nil,err}
			out=append(out,r)
		}
		return out,nil
	})
}

func (s *ProxyService) ListTrafficRules(ctx context.Context)([]proxy.Rule,error){
	return s.store.ListTrafficRules(ctx)
}
