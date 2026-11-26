package service

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fonu/fonu/internal/nginx"
	"github.com/fonu/fonu/internal/proxy"
	"github.com/fonu/fonu/internal/settings"
)

type NginxBackupInfo struct {
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

type RuleNginxView struct {
	Mode      string            `json:"mode"`
	Enabled   bool              `json:"enabled"`
	Active    bool              `json:"active"`
	Generated string            `json:"generated"`
	Content   string            `json:"content"`
	Backups   []NginxBackupInfo `json:"backups"`
}

type GlobalNginxView struct {
	Mode                string            `json:"mode"`
	GeneratedFramework  string            `json:"generated_framework"`
	GeneratedSnippet    string            `json:"generated_snippet"`
	Content             string            `json:"content"`
	Backups             []NginxBackupInfo `json:"backups"`
}

type SaveRuleNginxInput struct {
	Mode    string
	Content *string
}

type SaveGlobalNginxInput struct {
	Mode    string
	Content *string
}

func (s *ProxyService) GetRuleNginx(ctx context.Context, id int64) (RuleNginxView, error) {
	rule, err := s.Get(ctx, id)
	if err != nil {
		return RuleNginxView{}, err
	}
	certs, err := s.loadCertSources(ctx)
	if err != nil {
		return RuleNginxView{}, err
	}
	opts := s.loadGenerateOptions(ctx)
	generated, err := nginx.GenerateRuleBlocks(s.cfg, rule, certs, opts)
	if err != nil && rule.NginxMode != "custom" {
		return RuleNginxView{}, err
	}
	mode := rule.NginxMode
	if mode == "" {
		mode = "auto"
	}
	content := generated
	if mode == "custom" {
		custom, err := nginx.ReadRuleCustom(s.cfg, id)
		if err != nil {
			return RuleNginxView{}, err
		}
		if strings.TrimSpace(custom) != "" {
			content = custom
		}
	}
	backups, err := nginx.ListBackups(s.cfg.NginxRuleBackupsDir(id))
	if err != nil {
		return RuleNginxView{}, err
	}
	return RuleNginxView{
		Mode:      mode,
		Enabled:   rule.Enabled,
		Active:    rule.Enabled,
		Generated: generated,
		Content:   content,
		Backups:   mapBackups(backups),
	}, nil
}

func (s *ProxyService) SaveRuleNginx(ctx context.Context, id int64, in SaveRuleNginxInput) (RuleNginxView, error) {
	s.applyMu.Lock()
	defer s.applyMu.Unlock()
	return s.saveRuleNginxLocked(ctx,id,in)
}

func (s *ProxyService) saveRuleNginxLocked(ctx context.Context, id int64, in SaveRuleNginxInput) (RuleNginxView, error) {
	rule, err := s.Get(ctx,id)
	if err != nil { return RuleNginxView{}, err }
	mode := strings.TrimSpace(in.Mode)
	if mode == "" { mode = rule.NginxMode }
	if mode != "auto" && mode != "custom" { return RuleNginxView{}, fmt.Errorf("无效的 nginx 模式") }
	var override func(*nginx.GenerateOptions)
	var persist, restore func() error
	if mode == "custom" {
		if in.Content == nil || strings.TrimSpace(*in.Content) == "" { return RuleNginxView{}, fmt.Errorf("Nginx 配置不能为空") }
		content := *in.Content
		if err := s.validateRuleNginxPending(ctx,rule,content); err != nil { return RuleNginxView{}, err }
		override = func(opts *nginx.GenerateOptions) { opts.RuleCustomOverrides = map[int64]string{id:content} }
		restore, err = snapshotCustom(s.cfg.NginxRuleCustomPath(id))
		if err != nil { return RuleNginxView{}, err }
		persist = func() error { _, err := nginx.SaveRuleCustomWithBackup(s.cfg,id,content); return err }
	}
	_, err = mutateProxyWithFiles(ctx,s,func(store *proxy.Store,_ *sql.Tx)(struct{},error){
		return struct{}{},store.SetNginxMode(ctx,id,mode)
	},override,persist,restore)
	if err != nil { return RuleNginxView{},err }
	return s.GetRuleNginx(ctx,id)
}

func (s *ProxyService) RollbackRuleNginx(ctx context.Context, id int64, backupName string) (RuleNginxView, error) {
	s.applyMu.Lock()
	defer s.applyMu.Unlock()
	content, err := readNginxBackup(s.cfg.NginxRuleBackupsDir(id),backupName)
	if err != nil { return RuleNginxView{},err }
	return s.saveRuleNginxLocked(ctx,id,SaveRuleNginxInput{Mode:"custom",Content:&content})
}

func (s *ProxyService) GetGlobalNginx(ctx context.Context) (GlobalNginxView, error) {
	mode, err := s.settings.Get(ctx, settings.KeyNginxGlobalMode)
	if err != nil {
		return GlobalNginxView{}, err
	}
	if mode == "" {
		mode = "auto"
	}
	content, err := nginx.ReadGlobalCustom(s.cfg)
	if err != nil {
		return GlobalNginxView{}, err
	}
	backups, err := nginx.ListBackups(s.cfg.NginxGlobalBackupsDir())
	if err != nil {
		return GlobalNginxView{}, err
	}
	return GlobalNginxView{
		Mode:               mode,
		GeneratedFramework: nginx.GenerateGlobalFramework(s.cfg),
		GeneratedSnippet:   nginx.DefaultGlobalHTTPSnippet(),
		Content:            content,
		Backups:            mapBackups(backups),
	}, nil
}

func (s *ProxyService) SaveGlobalNginx(ctx context.Context, in SaveGlobalNginxInput) (GlobalNginxView, error) {
	s.applyMu.Lock()
	defer s.applyMu.Unlock()
	return s.saveGlobalNginxLocked(ctx,in)
}

func (s *ProxyService) saveGlobalNginxLocked(ctx context.Context, in SaveGlobalNginxInput) (GlobalNginxView, error) {
	mode := strings.TrimSpace(in.Mode)
	if mode == "" {
		var err error
		mode, err = s.settings.Get(ctx,settings.KeyNginxGlobalMode)
		if err != nil { return GlobalNginxView{},err }
	}
	if mode != "auto" && mode != "custom" { return GlobalNginxView{},fmt.Errorf("无效的 nginx 模式") }
	var content string
	var persist, restore func() error
	if mode == "custom" {
		if in.Content == nil { return GlobalNginxView{},fmt.Errorf("手动模式需要提供配置内容") }
		content = *in.Content
		var err error
		restore, err = snapshotCustom(s.cfg.NginxGlobalCustomPath())
		if err != nil { return GlobalNginxView{},err }
		persist = func() error { _,err := nginx.SaveGlobalCustomWithBackup(s.cfg,content); return err }
	}
	override := func(opts *nginx.GenerateOptions) {
		opts.GlobalCustomMode = mode
		opts.GlobalCustomOverride = content
		opts.GlobalCustomOverrideSet = mode == "custom"
	}
	_, err := mutateProxyWithFiles(ctx,s,func(_ *proxy.Store,tx *sql.Tx)(struct{},error){
		_,err := tx.ExecContext(ctx,`INSERT INTO settings(key,value) VALUES (?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`,settings.KeyNginxGlobalMode,mode)
		return struct{}{},err
	},override,persist,restore)
	if err != nil { return GlobalNginxView{},err }
	return s.GetGlobalNginx(ctx)
}

func (s *ProxyService) RollbackGlobalNginx(ctx context.Context, backupName string) (GlobalNginxView, error) {
	s.applyMu.Lock()
	defer s.applyMu.Unlock()
	content,err := readNginxBackup(s.cfg.NginxGlobalBackupsDir(),backupName)
	if err != nil { return GlobalNginxView{},err }
	return s.saveGlobalNginxLocked(ctx,SaveGlobalNginxInput{Mode:"custom",Content:&content})
}

// Resolve a backup by membership in the known list, not by user-provided paths.
func readNginxBackup(dir,name string)(string,error){
	backups,err := nginx.ListBackups(dir)
	if err != nil { return "",err }
	if name == "" && len(backups)>0 { name=backups[0].Name }
	for _,backup := range backups {
		if backup.Name == name {
			data,err := os.ReadFile(filepath.Join(dir,name))
			return string(data),err
		}
	}
	return "",fmt.Errorf("备份不存在")
}

func snapshotCustom(path string)(func() error,error){
	data,err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return func()error{
			err:=os.Remove(path)
			if os.IsNotExist(err){return nil}
			return err
		},nil
	}
	if err != nil {return nil,err}
	return func()error{return nginx.WriteCustomAtomic(path,string(data))},nil
}

func (s *ProxyService) validateRuleNginxPending(ctx context.Context, rule proxy.Rule, content string) error {
	rules, err := s.store.ListEnabled(ctx)
	if err != nil {
		return err
	}
	merged := make([]proxy.Rule, 0, len(rules))
	found := false
	for _, r := range rules {
		if r.ID == rule.ID {
			ruleCopy := rule
			ruleCopy.NginxMode = "custom"
			ruleCopy.Enabled = true
			merged = append(merged, ruleCopy)
			found = true
			continue
		}
		merged = append(merged, r)
	}
	if !found && rule.Enabled {
		ruleCopy := rule
		ruleCopy.NginxMode = "custom"
		merged = append(merged, ruleCopy)
	}
	return s.validateWithOverrides(ctx, merged, nginx.GenerateOptions{
		RuleCustomOverrides: map[int64]string{rule.ID: content},
	})
}

func (s *ProxyService) validateGlobalNginxPending(ctx context.Context, content string) error {
	rules, err := s.store.ListEnabled(ctx)
	if err != nil {
		return err
	}
	return s.validateWithOverrides(ctx, rules, nginx.GenerateOptions{
		GlobalCustomOverride: content,
	})
}

func (s *ProxyService) validateWithOverrides(ctx context.Context, rules []proxy.Rule, extra nginx.GenerateOptions) error {
	certs, err := s.loadCertSources(ctx)
	if err != nil {
		return err
	}
	opts := s.loadGenerateOptions(ctx)
	opts.GlobalCustomOverride = extra.GlobalCustomOverride
	opts.GlobalCustomPathOverride = extra.GlobalCustomPathOverride
	opts.RuleCustomOverrides = extra.RuleCustomOverrides
	opts.ChinaCIDRAvailable = nginx.ChinaCIDRExists(s.cfg)
	content, err := nginx.Generate(s.cfg, rules, certs, opts)
	if err != nil {
		return err
	}
	return s.nginx.ValidateContent(ctx, content)
}

func mapBackups(entries []nginx.BackupEntry) []NginxBackupInfo {
	out := make([]NginxBackupInfo, 0, len(entries))
	for _, entry := range entries {
		out = append(out, NginxBackupInfo{
			Name:      entry.Name,
			CreatedAt: entry.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}
	return out
}
