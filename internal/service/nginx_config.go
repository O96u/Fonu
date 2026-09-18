package service

import (
	"context"
	"fmt"
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
	if err != nil {
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
	rule, err := s.Get(ctx, id)
	if err != nil {
		return RuleNginxView{}, err
	}
	mode := strings.TrimSpace(in.Mode)
	if mode == "" {
		mode = rule.NginxMode
	}
	if mode != "auto" && mode != "custom" {
		return RuleNginxView{}, fmt.Errorf("无效的 nginx 模式")
	}

	if mode == "auto" {
		if err := s.store.SetNginxMode(ctx, id, "auto"); err != nil {
			return RuleNginxView{}, err
		}
		if err := s.applyNginx(ctx); err != nil {
			return RuleNginxView{}, err
		}
		return s.GetRuleNginx(ctx, id)
	}

	if in.Content == nil {
		return RuleNginxView{}, fmt.Errorf("手动模式需要提供配置内容")
	}
	content := *in.Content
	if strings.TrimSpace(content) == "" {
		return RuleNginxView{}, fmt.Errorf("Nginx 配置不能为空")
	}

	if err := s.validateRuleNginxPending(ctx, rule, content); err != nil {
		return RuleNginxView{}, err
	}
	if _, err := nginx.SaveRuleCustomWithBackup(s.cfg, id, content); err != nil {
		return RuleNginxView{}, err
	}
	if err := s.store.SetNginxMode(ctx, id, "custom"); err != nil {
		return RuleNginxView{}, err
	}
	if err := s.applyNginx(ctx); err != nil {
		return RuleNginxView{}, err
	}
	return s.GetRuleNginx(ctx, id)
}

func (s *ProxyService) RollbackRuleNginx(ctx context.Context, id int64, backupName string) (RuleNginxView, error) {
	if _, err := s.Get(ctx, id); err != nil {
		return RuleNginxView{}, err
	}
	backups, err := nginx.ListBackups(s.cfg.NginxRuleBackupsDir(id))
	if err != nil {
		return RuleNginxView{}, err
	}
	if len(backups) == 0 {
		return RuleNginxView{}, fmt.Errorf("没有可回滚的备份")
	}
	if backupName == "" {
		backupName = backups[0].Name
	}
	if err := nginx.RestoreBackup(s.cfg.NginxRuleBackupsDir(id), backupName, s.cfg.NginxRuleCustomPath(id)); err != nil {
		return RuleNginxView{}, err
	}
	custom, err := nginx.ReadRuleCustom(s.cfg, id)
	if err != nil {
		return RuleNginxView{}, err
	}
	rule, err := s.Get(ctx, id)
	if err != nil {
		return RuleNginxView{}, err
	}
	if err := s.validateRuleNginxPending(ctx, rule, custom); err != nil {
		return RuleNginxView{}, err
	}
	if err := s.store.SetNginxMode(ctx, id, "custom"); err != nil {
		return RuleNginxView{}, err
	}
	if err := s.applyNginx(ctx); err != nil {
		return RuleNginxView{}, err
	}
	return s.GetRuleNginx(ctx, id)
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
	mode := strings.TrimSpace(in.Mode)
	if mode == "" {
		var err error
		mode, err = s.settings.Get(ctx, settings.KeyNginxGlobalMode)
		if err != nil {
			return GlobalNginxView{}, err
		}
	}
	if mode != "auto" && mode != "custom" {
		return GlobalNginxView{}, fmt.Errorf("无效的 nginx 模式")
	}

	if mode == "auto" {
		if err := s.settings.Set(ctx, settings.KeyNginxGlobalMode, "auto"); err != nil {
			return GlobalNginxView{}, err
		}
		if err := s.applyNginx(ctx); err != nil {
			return GlobalNginxView{}, err
		}
		return s.GetGlobalNginx(ctx)
	}

	if in.Content == nil {
		return GlobalNginxView{}, fmt.Errorf("手动模式需要提供配置内容")
	}
	content := *in.Content
	if err := s.validateGlobalNginxPending(ctx, content); err != nil {
		return GlobalNginxView{}, err
	}
	if _, err := nginx.SaveGlobalCustomWithBackup(s.cfg, content); err != nil {
		return GlobalNginxView{}, err
	}
	if err := s.settings.Set(ctx, settings.KeyNginxGlobalMode, "custom"); err != nil {
		return GlobalNginxView{}, err
	}
	if err := s.applyNginx(ctx); err != nil {
		return GlobalNginxView{}, err
	}
	return s.GetGlobalNginx(ctx)
}

func (s *ProxyService) RollbackGlobalNginx(ctx context.Context, backupName string) (GlobalNginxView, error) {
	backups, err := nginx.ListBackups(s.cfg.NginxGlobalBackupsDir())
	if err != nil {
		return GlobalNginxView{}, err
	}
	if len(backups) == 0 {
		return GlobalNginxView{}, fmt.Errorf("没有可回滚的备份")
	}
	if backupName == "" {
		backupName = backups[0].Name
	}
	if err := nginx.RestoreBackup(s.cfg.NginxGlobalBackupsDir(), backupName, s.cfg.NginxGlobalCustomPath()); err != nil {
		return GlobalNginxView{}, err
	}
	content, err := nginx.ReadGlobalCustom(s.cfg)
	if err != nil {
		return GlobalNginxView{}, err
	}
	if err := s.validateGlobalNginxPending(ctx, content); err != nil {
		return GlobalNginxView{}, err
	}
	if err := s.settings.Set(ctx, settings.KeyNginxGlobalMode, "custom"); err != nil {
		return GlobalNginxView{}, err
	}
	if err := s.applyNginx(ctx); err != nil {
		return GlobalNginxView{}, err
	}
	return s.GetGlobalNginx(ctx)
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
